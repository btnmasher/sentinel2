package auth

import (
	"testing"
)

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
		"CHARACTER:TRANQUILITY:123",
		"OTHER:EVE:123",
		"CHARACTER:EVE:123:456",
		"CHARACTER:EVE:not-a-number",
	}

	for _, tt := range tests {
		if _, err := parseCharacterID(tt); err == nil {
			t.Fatalf("parseCharacterID(%q) expected error", tt)
		}
	}
}
