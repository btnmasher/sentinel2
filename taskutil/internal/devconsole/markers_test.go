package devconsole

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestMarkerToken_RoundTrip(t *testing.T) {
	token := markerToken("12:34:56", "10", "backend started | pid=42")
	ts, color, msg, ok := parseMarkerToken(token)
	if !ok {
		t.Fatalf("parseMarkerToken() ok = false")
	}
	if ts != "12:34:56" || color != "10" {
		t.Fatalf("unexpected marker fields ts=%q color=%q", ts, color)
	}
	if msg != "backend started / pid=42" {
		t.Fatalf("message = %q, want %q", msg, "backend started / pid=42")
	}
}

func TestRenderDisplayLineForWidth_Reflows(t *testing.T) {
	token := markerToken("12:34:56", "10", "backend started")

	wide := renderDisplayLineForWidth(token, 60)
	narrow := renderDisplayLineForWidth(token, 30)

	wideWidth := ansi.StringWidth(ansi.Strip(wide))
	narrowWidth := ansi.StringWidth(ansi.Strip(narrow))
	if wideWidth < 60 || wideWidth > 61 {
		t.Fatalf("wide marker width = %d, want ~60", wideWidth)
	}
	if narrowWidth < 30 || narrowWidth > 31 {
		t.Fatalf("narrow marker width = %d, want ~30", narrowWidth)
	}
	if wide == narrow {
		t.Fatalf("expected marker rendering to differ by width")
	}
	if !strings.Contains(ansi.Strip(wide), "12:34:56  backend started") {
		t.Fatalf("wide marker missing expected body: %q", ansi.Strip(wide))
	}
}

func TestRenderDisplayLineForWidth_EnforcesMinimum(t *testing.T) {
	token := markerToken("12:34:56", "10", "x")
	got := renderDisplayLineForWidth(token, 1)
	gotWidth := ansi.StringWidth(ansi.Strip(got))
	if gotWidth < minMarkerWidth || gotWidth > minMarkerWidth+1 {
		t.Fatalf("marker width = %d, want ~= minMarkerWidth=%d", gotWidth, minMarkerWidth)
	}
}

func TestPlainLineForCopy_MarkerAndANSI(t *testing.T) {
	token := markerToken("12:34:56", "10", "backend started")
	if got := plainLineForCopy(token); got != "12:34:56 backend started" {
		t.Fatalf("plainLineForCopy(marker) = %q", got)
	}

	ansiLine := "\x1b[31mhello\x1b[0m"
	if got := plainLineForCopy(ansiLine); got != "hello" {
		t.Fatalf("plainLineForCopy(ansi) = %q", got)
	}
}
