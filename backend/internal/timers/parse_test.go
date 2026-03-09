package timers

import "testing"

func TestParseText_Reinforced(t *testing.T) {
	t.Parallel()
	input := "Orbital Skyhook (3T7-M I) [Sappo's Shuttle Gas] 1,234 m Reinforced Until 2069.04.20 13:37:00"
	got, err := parseText(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.System != "3T7-M" {
		t.Fatalf("expected system 3T7-M, got %q", got.System)
	}

	if got.TimerKind != "reinforcement" {
		t.Fatalf("expected reinforcement, got %q", got.TimerKind)
	}

	if got.ExpiresAt.Format("2006-01-02T15:04:05Z") != "2069-04-20T13:37:00Z" {
		t.Fatalf("unexpected parsed time %s", got.ExpiresAt.Format("2006-01-02T15:04:05Z"))
	}
}

func TestParseText_Anchoring(t *testing.T) {
	t.Parallel()
	input := "3T7-M - Catboy Milking Farm 42,069 km Anchoring until 2069.04.20 13:37:00"
	got, err := parseText(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.System != "3T7-M" {
		t.Fatalf("expected system 3T7-M, got %q", got.System)
	}

	if got.TimerKind != "anchoring" {
		t.Fatalf("expected anchoring, got %q", got.TimerKind)
	}

	if got.Title == "" {
		t.Fatal("expected title to be extracted")
	}
}

func TestParseText_CustomDateOnly(t *testing.T) {
	t.Parallel()
	input := "Enemy skyhook vulnerable 2069.04.20 13:37:00"
	got, err := parseText(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.TimerKind != "skyhook" {
		t.Fatalf("expected skyhook kind, got %q", got.TimerKind)
	}
}

func TestParseText_NoDate(t *testing.T) {
	t.Parallel()
	if _, err := parseText("No timer value here"); err == nil {
		t.Fatal("expected parse error")
	}
}
