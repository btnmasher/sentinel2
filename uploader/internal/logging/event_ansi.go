package logging

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FormatEventANSI renders a single log event using the same terminal styling
// conventions as pretty console output, preserving ANSI color sequences.
func FormatEventANSI(event Event) string {
	ts := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(event.Time.Format("15:04:05.000"))
	levelLabel, levelStyle := levelBadge(event.Level)
	msg := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render(event.Message)

	line := lipgloss.JoinHorizontal(lipgloss.Center, ts, " ", levelStyle.Render(levelLabel), " ", msg)
	if len(event.Fields) == 0 {
		return line + "\n"
	}

	keys := make([]string, 0, len(event.Fields))
	for key := range event.Fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, keyStyle.Render(key)+sepStyle.Render("=")+valStyle.Render(fmt.Sprintf("%v", event.Fields[key])))
	}

	return line + "  " + strings.Join(parts, " ") + "\n"
}
