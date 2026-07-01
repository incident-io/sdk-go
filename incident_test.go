package incident_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	incident "github.com/incident-io/sdk-go"
)

// TestNew_authAndHeaders verifies that the constructed client attaches the
// bearer token, a User-Agent, and honours a custom endpoint.
func TestNew_authAndHeaders(t *testing.T) {
	var gotAuth, gotUA, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"identity":{"name":"test"}}`))
	}))
	defer srv.Close()

	c, err := incident.New("secret-key", incident.WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp, err := c.UtilitiesV1IdentityWithResponse(context.Background())
	if err != nil {
		t.Fatalf("UtilitiesV1IdentityWithResponse: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode())
	}

	if gotAuth != "Bearer secret-key" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer secret-key")
	}
	if !strings.HasPrefix(gotUA, "incident-io-sdk-go/") {
		t.Errorf("User-Agent = %q, want prefix incident-io-sdk-go/", gotUA)
	}
	if gotPath != "/v1/identity" {
		t.Errorf("path = %q, want /v1/identity", gotPath)
	}
}

// TestNew_customUserAgent verifies WithUserAgent overrides the default.
func TestNew_customUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := incident.New("k",
		incident.WithEndpoint(srv.URL),
		incident.WithUserAgent("my-app/1.2.3"),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.UtilitiesV1IdentityWithResponse(context.Background()); err != nil {
		t.Fatalf("request: %v", err)
	}
	if gotUA != "my-app/1.2.3" {
		t.Errorf("User-Agent = %q, want my-app/1.2.3", gotUA)
	}
}
