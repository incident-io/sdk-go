// Command postgen post-processes the oapi-codegen output so the generated file
// presents a clean public surface:
//
//   - The low-level request builders (NewXxxRequest / NewXxxRequestWithBody) and
//     response parsers (ParseXxxResponse) are unexported. They remain fully
//     functional (all callers are in-package) but disappear from the docs, so
//     the package leads with ClientWithResponses and its endpoint methods.
//   - Methods for operations marked `deprecated: true` in the OpenAPI schema get
//     a `// Deprecated:` doc comment, which pkg.go.dev renders prominently and
//     staticcheck (SA1019) flags at call sites.
//
// It edits the source as text (using AST positions) so it needs no third-party
// dependencies and can't perturb comment placement.
//
// Usage: postgen <generated.go> <schema.json>
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: postgen <generated.go> <schema.json>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "postgen:", err)
		os.Exit(1)
	}
}

const deprecatedNote = "// Deprecated: this endpoint is deprecated in the incident.io API. See\n// https://api-docs.incident.io/ for the recommended replacement.\n"

func run(genPath, schemaPath string) error {
	deprecated, err := deprecatedMethodNames(schemaPath)
	if err != nil {
		return err
	}

	src, err := os.ReadFile(genPath)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, genPath, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return err
	}

	// Names of package-level functions to unexport.
	unexport := map[string]bool{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		if shouldUnexport(fn.Name.Name) {
			unexport[fn.Name.Name] = true
		}
	}

	// Identifiers that are the selector of a selector expression (x.Sel): never
	// rename these — they're field/method accesses, not references to our funcs.
	selSkip := map[*ast.Ident]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if se, ok := n.(*ast.SelectorExpr); ok {
			selSkip[se.Sel] = true
		}
		return true
	})

	type edit struct {
		offset int
		del    int
		insert string
	}
	var edits []edit

	// Rename edits: lowercase the first byte of every non-selector identifier
	// that refers to a function we're unexporting.
	ast.Inspect(f, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok || selSkip[id] || !unexport[id.Name] {
			return true
		}
		off := fset.Position(id.Pos()).Offset
		edits = append(edits, edit{offset: off, del: 1, insert: strings.ToLower(id.Name[:1])})
		return true
	})

	// Deprecation edits: prepend a Deprecated paragraph to methods whose
	// operation is deprecated in the schema.
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || !deprecated[fn.Name.Name] {
			continue
		}
		if recv := receiverTypeName(fn); recv != "Client" && recv != "ClientWithResponses" {
			continue
		}
		insert := deprecatedNote
		if fn.Doc != nil {
			// Separate from the existing doc with a blank comment line so
			// "Deprecated:" begins its own paragraph.
			insert = "//\n" + deprecatedNote
			// Insert just before the func keyword, i.e. after the existing doc.
			edits = append(edits, edit{offset: fset.Position(fn.Pos()).Offset, del: 0, insert: insert})
			continue
		}
		edits = append(edits, edit{offset: fset.Position(fn.Pos()).Offset, del: 0, insert: insert})
	}

	// Apply edits from the end of the file backwards so offsets stay valid.
	sort.Slice(edits, func(i, j int) bool { return edits[i].offset > edits[j].offset })
	out := src
	for _, e := range edits {
		var b bytes.Buffer
		b.Write(out[:e.offset])
		b.WriteString(e.insert)
		b.Write(out[e.offset+e.del:])
		out = b.Bytes()
	}

	return os.WriteFile(genPath, out, 0o644)
}

func shouldUnexport(name string) bool {
	switch {
	case strings.HasPrefix(name, "New") && strings.HasSuffix(name, "Request"):
		return true
	case strings.HasPrefix(name, "New") && strings.HasSuffix(name, "RequestWithBody"):
		return true
	case strings.HasPrefix(name, "Parse") && strings.HasSuffix(name, "Response"):
		return true
	}
	return false
}

func receiverTypeName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	t := fn.Recv.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

// deprecatedMethodNames returns the set of generated method names (across the
// WithResponse / WithBody variants) for operations marked deprecated.
func deprecatedMethodNames(schemaPath string) (map[string]bool, error) {
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
			Deprecated  bool   `json:"deprecated"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	names := map[string]bool{}
	for _, methods := range doc.Paths {
		for _, op := range methods {
			if !op.Deprecated || op.OperationID == "" {
				continue
			}
			base := normalizeOperationID(op.OperationID)
			for _, suffix := range []string{"", "WithResponse", "WithBody", "WithBodyWithResponse"} {
				names[base+suffix] = true
			}
		}
	}
	return names, nil
}

// normalizeOperationID mirrors how oapi-codegen turns an operationId such as
// "Actions V1#List" into a Go method name like "ActionsV1List".
func normalizeOperationID(id string) string {
	var b strings.Builder
	for _, part := range nonAlnum.Split(id, -1) {
		if part == "" {
			continue
		}
		r := []rune(part)
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	return b.String()
}
