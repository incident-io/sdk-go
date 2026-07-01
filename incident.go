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
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/hashicorp/go-retryablehttp"
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
