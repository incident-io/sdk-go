// Package incident is the Go SDK for the incident.io public API.
//
// Every request and response type, and a method for every API endpoint, is
// generated from incident.io's published OpenAPI schema. This file adds a small,
// hand-written constructor that wires up authentication and sensible defaults.
//
//	c, err := incident.New("my-api-key")
//	if err != nil {
//	    return err
//	}
//
//	resp, err := c.IncidentsV2ListWithResponse(ctx, nil)
//	if err != nil {
//	    return err
//	}
//	if resp.JSON200 == nil {
//	    return fmt.Errorf("unexpected status %d: %s", resp.StatusCode(), resp.Body)
//	}
//	for _, inc := range resp.JSON200.Incidents {
//	    fmt.Println(inc.Reference, inc.Name)
//	}
//
// The returned *ClientWithResponses has a FooWithResponse method for every
// endpoint. Configure the client by passing options to New: WithUserAgent and
// WithRetries below, plus the generated WithBaseURL and WithHTTPClient.
package incident

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/oapi-codegen/runtime"
)

// DefaultEndpoint is the base URL of the incident.io public API.
const DefaultEndpoint = "https://api.incident.io"

// New returns a client for the incident.io public API, authenticated with the
// given API key.
//
// By default the client makes a single attempt per request and does not retry;
// pass WithRetries to opt in. Override the base URL with WithBaseURL and supply
// a custom HTTP client with WithHTTPClient.
func New(apiKey string, opts ...ClientOption) (*ClientWithResponses, error) {
	base := []ClientOption{
		WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			return nil
		}),
		WithUserAgent(fmt.Sprintf("incident-io-sdk-go/%s", sdkVersion())),
	}
	return NewClientWithResponses(DefaultEndpoint, append(base, opts...)...)
}

// WithUserAgent sets the User-Agent header sent with each request. New sets a
// default identifying this SDK; pass this after it to override.
func WithUserAgent(userAgent string) ClientOption {
	return WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("User-Agent", userAgent)
		return nil
	})
}

// WithRetries enables automatic retrying of transient failures (network errors,
// 429s and 5xxs) with exponential backoff that honours the Retry-After header.
// Retrying is off by default. The optional maxRetries argument defaults to 4.
//
// It works by installing a retrying HTTP client, so passing it alongside
// WithHTTPClient is redundant — the later option wins.
func WithRetries(maxRetries ...int) ClientOption {
	max := 4
	if len(maxRetries) > 0 {
		max = maxRetries[0]
	}
	return WithHTTPClient(newRetryingClient(max))
}

// newRetryingClient builds an *http.Client backed by go-retryablehttp with a
// backoff that respects Retry-After on 429/503 responses.
func newRetryingClient(maxRetries int) *http.Client {
	rc := retryablehttp.NewClient()
	rc.Logger = nil
	rc.RetryMax = maxRetries
	rc.Backoff = retryablehttp.DefaultBackoff
	rc.RetryWaitMin = 1 * time.Second
	rc.RetryWaitMax = 30 * time.Second
	return rc.StandardClient()
}

// sdkVersion reports the module version the caller built against, for the
// User-Agent header. It reads the version stamped into the binary by the Go
// toolchain, so it always matches the released tag without a hand-maintained
// constant.
func sdkVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/incident-io/sdk-go" && dep.Version != "" {
				return dep.Version
			}
		}
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return "dev"
}

// styleParamDeepObject serialises a deepObject query parameter, such as a list
// filter, in the form the incident.io API parses. internal/postgen points every
// generated deepObject call here instead of at the runtime.
//
// The runtime writes each array element with an index, so a filter with two
// values becomes status[one_of][0]=a&status[one_of][1]=b. The API reads only
// the first bracket pair of each key, so those two keys overwrite each other
// and it filters on just one of the values. Dropping the index repeats the
// key instead, status[one_of]=a&status[one_of]=b, which the API reads as a
// list.
func styleParamDeepObject(explode bool, paramName string, value any, opts runtime.StyleParamOptions) (string, error) {
	frag, err := runtime.StyleParamWithOptions("deepObject", explode, paramName, value, opts)
	if err != nil {
		return "", err
	}
	parsed, err := url.ParseQuery(frag)
	if err != nil {
		return "", err
	}

	type entry struct {
		key    string
		index  int
		values []string
	}
	entries := make([]entry, 0, len(parsed))
	for key, values := range parsed {
		key, index := splitArrayIndex(key)
		entries = append(entries, entry{key, index, values})
	}

	// Encode sorts the keys, so only the order of each key's values matters.
	// Sorting by index keeps the order the caller gave them.
	slices.SortFunc(entries, func(a, b entry) int { return cmp.Compare(a.index, b.index) })

	out := url.Values{}
	for _, e := range entries {
		out[e.key] = append(out[e.key], e.values...)
	}
	return out.Encode(), nil
}

// splitArrayIndex splits status[one_of][1] into status[one_of] and 1. A key
// with no trailing index is returned as-is, with index 0.
func splitArrayIndex(key string) (string, int) {
	if i := strings.LastIndexByte(key, '['); i >= 0 && strings.HasSuffix(key, "]") {
		if index, err := strconv.Atoi(key[i+1 : len(key)-1]); err == nil {
			return key[:i], index
		}
	}
	return key, 0
}
