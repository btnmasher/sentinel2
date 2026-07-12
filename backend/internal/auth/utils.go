package auth

import (
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

func absoluteURL(c *core.RequestEvent) string {
	return RequestBaseURL(c) + "/api/auth/callback"
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

func resolveRedirectBaseURL(c *core.RequestEvent, devMode bool) string {
	if c == nil || c.Request == nil {
		return ""
	}

	if devMode {
		if origin := requestHeaderOrigin(c.Request.Header.Get("Origin")); origin != "" {
			return origin
		}
		if origin := requestHeaderOrigin(c.Request.Header.Get("Referer")); origin != "" {
			return origin
		}
		return RequestBaseURL(c)
	}

	return forceHTTPS(RequestBaseURL(c))
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

func forceHTTPS(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	if parsed.Scheme != "" {
		parsed.Scheme = "https"
	}
	return parsed.String()
}
