package auth

import (
	"encoding/base64"
	"testing"
)

func TestParseEVEToken(t *testing.T) {
	t.Parallel()

	payload := `{"sub":"CHARACTER:EVE:123","name":"Tester","exp":1234567890,"scp":["a","b"]}`
	token := "hdr." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".sig"

	claims, err := parseEVEToken(token)
	if err != nil {
		t.Fatalf("parseEVEToken() error = %v", err)
	}
	if claims.Sub != "CHARACTER:EVE:123" {
		t.Fatalf("claims.Sub = %q", claims.Sub)
	}
	if claims.Name != "Tester" {
		t.Fatalf("claims.Name = %q", claims.Name)
	}
	if len(claims.Scp) != 2 || claims.Scp[0] != "a" || claims.Scp[1] != "b" {
		t.Fatalf("claims.Scp = %#v", claims.Scp)
	}
}

func TestParseEVETokenInvalid(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"not-a-jwt",
		"a.invalid*b64.c",
		"a." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":`)) + ".c",
	}

	for _, tt := range tests {
		if _, err := parseEVEToken(tt); err == nil {
			t.Fatalf("parseEVEToken(%q) expected error", tt)
		}
	}
}

func TestParseCharacterID(t *testing.T) {
	t.Parallel()

	id, err := parseCharacterID("CHARACTER:EVE:987654")
	if err != nil {
		t.Fatalf("parseCharacterID() error = %v", err)
	}
	if id != 987654 {
		t.Fatalf("parseCharacterID() = %d, want %d", id, 987654)
	}
}

func TestParseCharacterIDInvalid(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"CHARACTER:EVE",
		"CHARACTER:EVE:not-a-number",
	}

	for _, tt := range tests {
		if _, err := parseCharacterID(tt); err == nil {
			t.Fatalf("parseCharacterID(%q) expected error", tt)
		}
	}
}
