package auth

import (
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

func TestResolveCallbackURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		host          string
		publicBaseURL string
		devMode       bool
		want          string
		wantErr       bool
	}{
		{name: "configured production origin", host: "attacker.example", publicBaseURL: "https://app.example.com", want: "https://app.example.com/api/auth/callback"},
		{name: "local development origin", host: "127.0.0.1:8090", devMode: true, want: "http://127.0.0.1:8090/api/auth/callback"},
		{name: "non-loopback development host", host: "example.test", devMode: true, wantErr: true},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "http://"+tt.host+"/api/auth/callback", http.NoBody)
		req.Host = tt.host
		c := &core.RequestEvent{Event: router.Event{Request: req}}
		got, err := resolveCallbackURL(c, tt.publicBaseURL, tt.devMode)
		if (err != nil) != tt.wantErr {
			t.Fatalf("%s: resolveCallbackURL() error = %v, wantErr %t", tt.name, err, tt.wantErr)
		}
		if !tt.wantErr && got != tt.want {
			t.Fatalf("%s: resolveCallbackURL() = %q, want %q", tt.name, got, tt.want)
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
		{
			name:            "external origin falls back to request host",
			host:            "127.0.0.1:8090",
			origin:          "https://example.com",
			expectedBaseURL: "http://127.0.0.1:8090",
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
		got, err := resolveRedirectBaseURL(c, "", true)
		if err != nil {
			t.Fatalf("%s: resolveRedirectBaseURL() error = %v", tt.name, err)
		}
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
	got, err := resolveRedirectBaseURL(c, "https://app.example.com", false)
	if err != nil {
		t.Fatalf("resolveRedirectBaseURL() error = %v", err)
	}
	want := "https://app.example.com"
	if got != want {
		t.Fatalf("resolveRedirectBaseURL() = %q, want %q", got, want)
	}
}

func TestValidatePublicBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawURL  string
		devMode bool
		wantErr bool
	}{
		{name: "production https", rawURL: "https://app.example.com", wantErr: false},
		{name: "production missing", wantErr: true},
		{name: "production http", rawURL: "http://app.example.com", wantErr: true},
		{name: "development http", rawURL: "http://localhost:5173", devMode: true, wantErr: false},
		{name: "path rejected", rawURL: "https://app.example.com/app", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePublicBaseURL(tt.rawURL, tt.devMode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePublicBaseURL() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func TestResolveRedirectBaseURLRejectsNonLoopbackDevelopmentHost(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://example.test/api/auth/login", http.NoBody)
	req.Host = "example.test"
	c := &core.RequestEvent{Event: router.Event{Request: req}}

	if _, err := resolveRedirectBaseURL(c, "", true); err == nil {
		t.Fatal("resolveRedirectBaseURL() error = nil, want non-loopback host rejection")
	}
}
