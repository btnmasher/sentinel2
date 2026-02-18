package auth

import (
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
	return scheme + "://" + req.Host + "/api/auth/callback"
}
