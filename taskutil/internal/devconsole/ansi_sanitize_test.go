package devconsole

import "testing"

func TestNormalizeLineForViewport_PreservesSGRStripsCursorControl(t *testing.T) {
	in := "\x1b[31mred\x1b[0m \x1b[2K\x1b[1Gdone"
	got := normalizeLineForViewport(in)
	if got == "" {
		t.Fatalf("normalizeLineForViewport() returned empty output")
	}

	if got != "\x1b[31mred\x1b[0m done" {
		t.Fatalf("unexpected sanitized output: %q", got)
	}
}

func TestNormalizeLineForViewport_UsesLatestCarriageReturnSegment(t *testing.T) {
	in := "building 10%\rbuilding 20%\rbuilding 30%"
	got := normalizeLineForViewport(in)
	if got != "building 30%" {
		t.Fatalf("normalizeLineForViewport() = %q, want %q", got, "building 30%")
	}
}

func TestNormalizeLineForViewport_AppendsResetWhenMissing(t *testing.T) {
	in := "\x1b[32mok"
	got := normalizeLineForViewport(in)
	want := "\x1b[32mok\x1b[0m"
	if got != want {
		t.Fatalf("normalizeLineForViewport() = %q, want %q", got, want)
	}
}
