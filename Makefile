.PHONY: fetch generate test tidy

generate: incident.gen.go

# The generator version is pinned here rather than via a tools.go, so that the
# generator's own dependency tree stays out of this module's go.mod.
OAPI_CODEGEN_VERSION := v2.6.0

# Generate the client into the root package, then post-process it (see
# internal/postgen) to unexport the low-level request/response helpers and mark
# deprecated endpoints.
incident.gen.go: openapi3.json internal/postgen/main.go
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION) \
		--generate types,client \
		--package incident \
		-o $@ \
		openapi3.json
	go run ./internal/postgen $@ openapi3.json
	gofmt -w $@

test:
	go test ./...

tidy:
	go mod tidy

SCHEMA_URL := https://api.incident.io/v1/openapiV3.json

# OUT lets the release workflow keep the old schema to diff against.
OUT ?= openapi3.json

fetch:
	curl -sfSL $(SCHEMA_URL) -o $(OUT)
