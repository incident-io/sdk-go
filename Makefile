.PHONY: generate test tidy

# Regenerate the client from the committed OpenAPI schema. The schema itself is
# refreshed from https://api.incident.io/v1/openapiV3.json by the sync workflow.
generate: client/client.gen.go

# The generator version is pinned here rather than via a tools.go, so that the
# generator's own dependency tree stays out of this module's go.mod.
OAPI_CODEGEN_VERSION := v2.6.0

client/client.gen.go: client/openapi3.json
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION) \
		--generate types,client \
		--package client \
		-o $@ \
		client/openapi3.json

test:
	go test ./...

tidy:
	go mod tidy
