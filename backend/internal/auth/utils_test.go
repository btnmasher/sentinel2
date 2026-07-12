package auth

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"golang.org/x/oauth2"
)

func TestTokenExpiry(t *testing.T) {
	t.Parallel()

	now := time.Now()
	exp := now.Add(5 * time.Minute).UTC().Truncate(time.Second)
	got := tokenExpiry(&oauth2.Token{Expiry: exp})
	if got != exp.Unix() {
		t.Fatalf("tokenExpiry() = %d, want %d", got, exp.Unix())
	}

	lower := time.Now().Add(59 * time.Minute).Unix()
	upper := time.Now().Add(61 * time.Minute).Unix()
	got = tokenExpiry(&oauth2.Token{})
	if got < lower || got > upper {
		t.Fatalf("tokenExpiry() default = %d, want within [%d,%d]", got, lower, upper)
	}
}

func TestAbsoluteURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		host            string
		forwardedProto  string
		withTLS         bool
		expectedURLHead string
	}{
		{name: "default http", host: "example.test", expectedURLHead: "http://example.test"},
		{name: "forwarded https first value", host: "example.test", forwardedProto: "https,http", expectedURLHead: "https://example.test"},
		{name: "invalid forwarded proto", host: "example.test", forwardedProto: "ftp", expectedURLHead: "http://example.test"},
		{name: "tls overrides forwarded http", host: "example.test", forwardedProto: "http", withTLS: true, expectedURLHead: "https://example.test"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "http://"+tt.host+"/api/auth/callback", http.NoBody)
		req.Host = tt.host
		if tt.forwardedProto != "" {
			req.Header.Set("X-Forwarded-Proto", tt.forwardedProto)
		}
		if tt.withTLS {
			req.TLS = &tls.ConnectionState{}
		}

		c := &core.RequestEvent{Event: router.Event{Request: req}}
		got := absoluteURL(c)
		want := tt.expectedURLHead + "/api/auth/callback"
		if got != want {
			t.Fatalf("%s: absoluteURL() = %q, want %q", tt.name, got, want)
		}
	}
}

func TestResolveRedirectBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		host            string
		origin          string
		referer         string
		expectedBaseURL string
	}{
		{
			name:            "loopback referer",
			host:            "127.0.0.1:8090",
			referer:         "http://127.0.0.1:5173/profile",
			expectedBaseURL: "http://127.0.0.1:5173",
		},
		{
			name:            "external referer falls back to request host",
			host:            "127.0.0.1:8090",
			referer:         "https://example.com/profile",
			expectedBaseURL: "http://127.0.0.1:8090",
		},
		{
			name:            "origin header",
			host:            "127.0.0.1:8090",
			origin:          "http://localhost:5173",
			expectedBaseURL: "http://localhost:5173",
		},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "http://"+tt.host+"/api/auth/login", http.NoBody)
		req.Host = tt.host
		if tt.origin != "" {
			req.Header.Set("Origin", tt.origin)
		}
		if tt.referer != "" {
			req.Header.Set("Referer", tt.referer)
		}

		c := &core.RequestEvent{Event: router.Event{Request: req}}
		got := resolveRedirectBaseURL(c, true)
		if got != tt.expectedBaseURL {
			t.Fatalf("%s: resolveRedirectBaseURL() = %q, want %q", tt.name, got, tt.expectedBaseURL)
		}
	}
}

func TestResolveRedirectBaseURLProduction(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://example.test/api/auth/login", http.NoBody)
	req.Host = "example.test"
	req.Header.Set("Origin", "http://localhost:5173")

	c := &core.RequestEvent{Event: router.Event{Request: req}}
	got := resolveRedirectBaseURL(c, false)
	want := "https://example.test"
	if got != want {
		t.Fatalf("resolveRedirectBaseURL() = %q, want %q", got, want)
	}
}
