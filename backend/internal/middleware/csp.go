package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

const cspNonceKey = "csp_nonce"
const cspNonceSize = 16

// ContentSecurityPolicy returns middleware that sets a restrictive CSP header.
//
// For the SPA frontend, we allow:
//   - scripts/styles from self (bundled SPA, no external JS/CSS CDNs)
//   - fonts from Google Fonts
//   - images from self, data URIs, and https:// (EVE image APIs, dotlan overlays, etc.)
//   - connect to self and ESI API endpoints
//   - no frame ancestors to prevent clickjacking
func ContentSecurityPolicy(devFrontendProxyEnabled bool) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		csp := baseCSP()
		if devFrontendProxyEnabled && shouldRelaxDevCSP(c) {
			nonce, err := newCSPNonce()
			if err != nil {
				return fmt.Errorf("create CSP nonce: %w", err)
			}
			c.Set(cspNonceKey, nonce)
			csp = devCSP(nonce)
		}

		c.Response.Header().Set("Content-Security-Policy", csp)
		return c.Next()
	}
}

func baseCSP() string {
	return "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: https:; connect-src 'self' https://esi.evetech.net https://login.eveonline.com; frame-ancestors 'none'; base-uri 'self'; form-action 'self';"
}

func devCSP(nonce string) string {
	return "default-src 'self'; script-src 'self' 'nonce-" + nonce + "'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: https:; connect-src 'self' https://esi.evetech.net https://login.eveonline.com; frame-ancestors 'none'; base-uri 'self'; form-action 'self';"
}

func shouldRelaxDevCSP(c *core.RequestEvent) bool {
	if c == nil || c.Request == nil {
		return false
	}
	if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
		return false
	}
	if strings.HasPrefix(c.Request.URL.Path, "/api") {
		return false
	}
	accept := strings.ToLower(c.Request.Header.Get("Accept"))
	return strings.Contains(accept, "text/html")
}

func newCSPNonce() (string, error) {
	b := make([]byte, cspNonceSize)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
