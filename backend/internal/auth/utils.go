package auth

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/oauth2"
)

func tokenExpiry(token *oauth2.Token) int64 {
	if !token.Expiry.IsZero() {
		return token.Expiry.Unix()
	}
	return time.Now().Add(time.Hour).Unix()
}

// ValidatePublicBaseURL validates the configured public origin used for OAuth.
// Production requires a configured HTTPS origin; local development may derive
// the origin from a loopback request when the value is empty.
func ValidatePublicBaseURL(rawURL string, devMode bool) error {
	if strings.TrimSpace(rawURL) == "" {
		if devMode {
			return nil
		}
		return ErrPublicBaseURLRequired
	}

	_, err := parsePublicBaseURL(rawURL, devMode)
	return err
}

func resolveCallbackURL(c *core.RequestEvent, publicBaseURL string, devMode bool) (string, error) {
	baseURL, err := resolvePublicBaseURL(c, publicBaseURL, devMode)
	if err != nil {
		return "", err
	}
	return baseURL + "/api/auth/callback", nil
}

// RequestBaseURL returns the request origin without a path.
func RequestBaseURL(c *core.RequestEvent) string {
	req := c.Request
	scheme := "http"
	if rawProto := req.Header.Get("X-Forwarded-Proto"); rawProto != "" {
		// Proxy chains may append values like "https,http"; trust the first hop value.
		proto := strings.ToLower(strings.TrimSpace(strings.Split(rawProto, ",")[0]))
		if proto == "https" || proto == "http" {
			scheme = proto
		}
	}

	if req.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + req.Host
}

func resolveRedirectBaseURL(c *core.RequestEvent, publicBaseURL string, devMode bool) (string, error) {
	if c == nil || c.Request == nil {
		return "", fmt.Errorf("missing request")
	}

	if strings.TrimSpace(publicBaseURL) != "" {
		return parsePublicBaseURL(publicBaseURL, devMode)
	}
	if !devMode {
		return "", ErrPublicBaseURLRequired
	}

	if origin := requestHeaderOrigin(c.Request.Header.Get("Origin")); origin != "" {
		return origin, nil
	}
	if origin := requestHeaderOrigin(c.Request.Header.Get("Referer")); origin != "" {
		return origin, nil
	}

	return resolveLocalRequestBaseURL(c)
}

func resolvePublicBaseURL(c *core.RequestEvent, publicBaseURL string, devMode bool) (string, error) {
	if strings.TrimSpace(publicBaseURL) != "" {
		return parsePublicBaseURL(publicBaseURL, devMode)
	}
	if !devMode {
		return "", ErrPublicBaseURLRequired
	}

	return resolveLocalRequestBaseURL(c)
}

func parsePublicBaseURL(rawURL string, devMode bool) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("PUBLIC_BASE_URL must be an absolute HTTP(S) origin")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("PUBLIC_BASE_URL must use http or https")
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("PUBLIC_BASE_URL must not include a path, query, or fragment")
	}
	if parsed.Scheme == "http" && !isLoopbackHost(parsed.Host) {
		return "", fmt.Errorf("HTTP PUBLIC_BASE_URL must use a loopback host")
	}
	if !devMode && parsed.Scheme != "https" {
		return "", fmt.Errorf("PUBLIC_BASE_URL must use https outside local development")
	}

	return parsed.Scheme + "://" + parsed.Host, nil
}

func resolveLocalRequestBaseURL(c *core.RequestEvent) (string, error) {
	baseURL := RequestBaseURL(c)
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || !isLoopbackHost(parsed.Host) {
		return "", fmt.Errorf("local OAuth URLs require a loopback request host")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("local OAuth URLs must use http or https")
	}

	return parsed.Scheme + "://" + parsed.Host, nil
}

func requestHeaderOrigin(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if parsed.Host == "" {
		return ""
	}
	if !isLoopbackHost(parsed.Host) {
		return ""
	}

	return parsed.Scheme + "://" + parsed.Host
}

func isLoopbackHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}

	name := host
	if strings.Contains(host, ":") {
		if parsedHost, _, err := net.SplitHostPort(host); err == nil {
			name = parsedHost
		}
	}
	name = strings.Trim(name, "[]")

	switch name {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}
