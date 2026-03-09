package format

import "testing"

func TestParseDateTimeFlexibleUTC_PocketBaseLayout(t *testing.T) {
	got, err := ParseDateTimeFlexibleUTC("2026-02-20 00:00:00.000Z")
	if err != nil {
		t.Fatalf("expected parse success, got error: %v", err)
	}

	if got.Year() != 2026 || got.Month() != 2 || got.Day() != 20 {
		t.Fatalf("unexpected date parsed: %s", got)
	}

	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
		t.Fatalf("unexpected time parsed: %s", got)
	}
}
