// Package incident is the Go SDK for the incident.io public API.
//
// The bulk of this module — every request and response type, and a method for
// every API endpoint — is generated from incident.io's published OpenAPI
// schema and lives in the client subpackage. This root package adds a small,
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
// The returned *client.ClientWithResponses exposes a FooWithResponse method for
// every endpoint. Request and response types live in the client package.
package incident

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/incident-io/sdk-go/client"
)

// DefaultEndpoint is the base URL of the incident.io public API.
const DefaultEndpoint = "https://api.incident.io"

// New returns a client for the incident.io public API, authenticated with the
// given API key.
//
// By default the client makes a single attempt per request and does not retry.
// Pass WithRetries to opt in to automatic retrying of transient failures.
func New(apiKey string, opts ...Option) (*client.ClientWithResponses, error) {
	cfg := config{
		endpoint:  DefaultEndpoint,
		userAgent: fmt.Sprintf("incident-io-sdk-go/%s", sdkVersion()),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		if cfg.retries {
			httpClient = newRetryingClient(cfg.retryMax)
		} else {
			httpClient = http.DefaultClient
		}
	}

	editors := []client.RequestEditorFn{
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			return nil
		},
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("User-Agent", cfg.userAgent)
			return nil
		},
	}

	return client.NewClientWithResponses(cfg.endpoint,
		client.WithHTTPClient(httpClient),
		client.WithRequestEditorFn(chainEditors(editors...)),
	)
}

type config struct {
	endpoint   string
	userAgent  string
	httpClient client.HttpRequestDoer
	retries    bool
	retryMax   int
}

// Option configures the client returned by New.
type Option func(*config)

// WithEndpoint overrides the API base URL. Defaults to DefaultEndpoint.
func WithEndpoint(endpoint string) Option {
	return func(c *config) { c.endpoint = endpoint }
}

// WithUserAgent overrides the User-Agent header sent with each request.
func WithUserAgent(userAgent string) Option {
	return func(c *config) { c.userAgent = userAgent }
}

// WithHTTPClient supplies a custom HTTP client (any client.HttpRequestDoer).
// When set, WithRetries is ignored — wire your own retrying transport into the
// client you pass here.
func WithHTTPClient(doer client.HttpRequestDoer) Option {
	return func(c *config) { c.httpClient = doer }
}

// WithRetries enables automatic retrying of transient failures (network
// errors, 429s and 5xxs) with exponential backoff that honours the Retry-After
// header. Retrying is off by default. The optional maxRetries argument defaults
// to 4.
func WithRetries(maxRetries ...int) Option {
	return func(c *config) {
		c.retries = true
		c.retryMax = 4
		if len(maxRetries) > 0 {
			c.retryMax = maxRetries[0]
		}
	}
}

// newRetryingClient builds an *http.Client backed by go-retryablehttp with a
// backoff that respects Retry-After on 429 responses.
func newRetryingClient(maxRetries int) *http.Client {
	rc := retryablehttp.NewClient()
	rc.Logger = nil
	rc.RetryMax = maxRetries
	// DefaultBackoff honours the Retry-After header on 429/503 responses, and
	// otherwise backs off exponentially between the wait bounds below.
	rc.Backoff = retryablehttp.DefaultBackoff
	rc.RetryWaitMin = 1 * time.Second
	rc.RetryWaitMax = 30 * time.Second
	return rc.StandardClient()
}

// chainEditors runs each RequestEditorFn in order.
func chainEditors(editors ...client.RequestEditorFn) client.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		for _, e := range editors {
			if err := e(ctx, req); err != nil {
				return err
			}
		}
		return nil
	}
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
