package devconsole

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const markerPrefix = "__TASKUTIL_MARKER__|"

func markerToken(ts, color, message string) string {
	safeMsg := strings.ReplaceAll(strings.TrimSpace(message), "|", "/")
	return markerPrefix + ts + "|" + color + "|" + safeMsg
}

func parseMarkerToken(line string) (ts string, color string, message string, ok bool) {
	if !strings.HasPrefix(line, markerPrefix) {
		return "", "", "", false
	}
	parts := strings.SplitN(strings.TrimPrefix(line, markerPrefix), "|", 3)
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func renderDisplayLineForWidth(line string, width int) string {
	ts, color, message, ok := parseMarkerToken(line)
	if !ok {
		return line
	}
	return renderMarkerLine(ts, color, message, width)
}

func plainLineForCopy(line string) string {
	ts, _, message, ok := parseMarkerToken(line)
	if !ok {
		return ansi.Strip(line)
	}
	return fmt.Sprintf("%s %s", ts, message)
}

func renderMarkerLine(ts, color, message string, width int) string {
	if width < minMarkerWidth {
		width = minMarkerWidth
	}
	body := fmt.Sprintf(" %s  %s ", ts, message)
	bodyWidth := ansi.StringWidth(body)
	side := max((width-bodyWidth)/2, 2)
	total := side*2 + bodyWidth
	right := side
	if total < width {
		right += width - total
	}
	divider := strings.Repeat("─", side) + body + strings.Repeat("─", right)
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render(divider)
}
