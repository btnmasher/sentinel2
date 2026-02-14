package devconsole

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const ansiReset = "\x1b[0m"

func normalizeLineForViewport(line string) string {
	line = segmentAfterCarriageReturn(line)
	if line == "" {
		return ""
	}

	var b strings.Builder
	containsSGR := false

	parser := ansi.GetParser()
	defer ansi.PutParser(parser)
	state := byte(0)

	for len(line) > 0 {
		seq, width, n, newState := ansi.DecodeSequence(line, state, parser)
		if n <= 0 {
			break
		}
		if width > 0 {
			b.WriteString(seq)
		} else if isSGRSequence(seq) {
			containsSGR = true
			b.WriteString(seq)
		}
		line = line[n:]
		state = newState
	}

	out := b.String()
	if containsSGR && !strings.Contains(out, ansiReset) {
		// Prevent style bleed into borders and adjacent UI regions.
		out += ansiReset
	}
	return out
}

func segmentAfterCarriageReturn(s string) string {
	if idx := strings.LastIndexByte(s, '\r'); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

func isSGRSequence(seq string) bool {
	return strings.HasPrefix(seq, "\x1b[") && strings.HasSuffix(seq, "m")
}
