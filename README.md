# incident.io Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/incident-io/sdk-go.svg)](https://pkg.go.dev/github.com/incident-io/sdk-go)

The official Go SDK for the [incident.io](https://incident.io) [public API](https://api-docs.incident.io/).

It is generated automatically from our published OpenAPI schema, so it always
tracks the live API — there is a method for every endpoint, and a Go type for
every request and response.

## Install

```bash
go get github.com/incident-io/sdk-go
```

Requires Go 1.24 or later.

## Quickstart

Create an API key in your incident.io dashboard under **Settings → API keys**,
then:

```go
package main

import (
	"context"
	"fmt"
	"log"

	incident "github.com/incident-io/sdk-go"
)

func main() {
	c, err := incident.New("my-api-key")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := c.IncidentsV2ListWithResponse(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	if resp.JSON200 == nil {
		log.Fatalf("unexpected status %d: %s", resp.StatusCode(), resp.Body)
	}

	for _, inc := range resp.JSON200.Incidents {
		fmt.Printf("%s %s\n", inc.Reference, inc.Name)
	}
}
```

Every endpoint has a `...WithResponse` method that returns a typed response.
Inspect `resp.StatusCode()` and the `resp.JSONxxx` fields (e.g. `JSON200`,
`JSON404`) to handle results — a `nil` `JSON200` means the API returned a
non-2xx status, and `resp.Body` holds the raw payload.

## Configuration

`New` takes functional options:

```go
c, err := incident.New("my-api-key",
	incident.WithEndpoint("https://api.incident.io"), // override the base URL
	incident.WithUserAgent("my-app/1.0.0"),           // identify your integration
	incident.WithRetries(),                           // opt in to automatic retries
	incident.WithHTTPClient(myHTTPClient),            // bring your own *http.Client
)
```

### Retries

By default the client makes a **single attempt** per request and does not
retry. Pass `WithRetries()` to enable exponential backoff on transient failures
(network errors, `429`s and `5xx`s); it honours the `Retry-After` header. Pass
`WithRetries(n)` to cap the number of retries (default 4).

If you supply your own client with `WithHTTPClient`, `WithRetries` is ignored —
configure retrying on the client you pass in.

## Versioning

Releases are cut automatically whenever the API schema changes. We use
[SemVer](https://semver.org/): additive API changes bump the minor version and
backwards-compatible fixes bump the patch version. Changes that would break Go
consumers are never released automatically — they require a deliberate major
version.

## Support

Found a bug or missing something? Please
[open an issue](https://github.com/incident-io/sdk-go/issues). For questions
about the API itself, see the [API docs](https://api-docs.incident.io/).

Note that `client/client.gen.go` is generated — please don't send PRs editing it
directly; changes there come from the upstream schema.

## License

MIT — see [LICENSE](./LICENSE).

This SDK's generated code is produced by
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen), which is licensed
under Apache-2.0.
