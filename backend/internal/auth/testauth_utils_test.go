package auth

import (
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestRefreshExpiry(t *testing.T) {
	t.Parallel()

	tokenWithExtra := (&oauth2.Token{}).WithExtra(map[string]any{
		"refresh_expires_in": float64(120),
	})

	lower := time.Now().Add(110 * time.Second).Unix()
	upper := time.Now().Add(130 * time.Second).Unix()
	got := refreshExpiry(tokenWithExtra)
	if got < lower || got > upper {
		t.Fatalf("refreshExpiry(extra) = %d, want within [%d,%d]", got, lower, upper)
	}

	fallback := refreshExpiry((&oauth2.Token{}).WithExtra(map[string]any{}))
	fallbackLower := time.Now().Add(29 * 24 * time.Hour).Unix()
	fallbackUpper := time.Now().Add(31 * 24 * time.Hour).Unix()
	if fallback < fallbackLower || fallback > fallbackUpper {
		t.Fatalf("refreshExpiry(fallback) = %d, want within [%d,%d]", fallback, fallbackLower, fallbackUpper)
	}
}
