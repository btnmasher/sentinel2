package devconsole

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type helpItem struct {
	keys string
	desc string
}

func renderHelp(items []helpItem) string {
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, keyStyle.Render(it.keys)+" "+descStyle.Render(it.desc))
	}
	return strings.Join(parts, descStyle.Render(" | "))
}

func renderHelpRows(items []helpItem, width int) []string {
	if width <= 0 {
		return []string{renderHelp(items)}
	}
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sep := descStyle.Render(" | ")
	rows := make([]string, 0, 2)
	cur := ""
	curW := 0
	for _, it := range items {
		seg := keyStyle.Render(it.keys) + " " + descStyle.Render(it.desc)
		segW := ansi.StringWidth(seg)
		if cur == "" {
			cur = seg
			curW = segW
			continue
		}
		addW := ansi.StringWidth(sep) + segW
		if curW+addW <= width {
			cur += sep + seg
			curW += addW
			continue
		}
		rows = append(rows, cur)
		cur = seg
		curW = segW
	}
	if cur != "" {
		rows = append(rows, cur)
	}
	if len(rows) == 0 {
		return []string{""}
	}
	return rows
}

func (m viewState) headerLines(width int) []string {
	nextLayout := "horizontal"
	if !m.verticalSplit {
		nextLayout = "vertical"
	}
	titlePlain := fmt.Sprintf("%s Developer Console", m.appName())
	title := rainbowTitle(ansi.Cut(titlePlain, 0, max(width, 1)))
	statusValue := m.status
	if statusValue == "" {
		statusValue = "running"
	}
	status := colorizeStatusLine(statusValue)
	lines := []string{title}
	if m.showHelp {
		lines = append(lines, renderHelpRows([]helpItem{
			{keys: "q", desc: "quit"},
			{keys: "tab", desc: "switch pane"},
			{keys: "up/down/pgup/pgdn", desc: "scroll"},
			{keys: "end", desc: "follow bottom"},
			{keys: "home", desc: "top + pause follow"},
			{keys: "1/2", desc: "focus"},
			{keys: "s", desc: "layout -> " + nextLayout},
			{keys: "x", desc: "swap panes"},
			{keys: "z", desc: "fullscreen"},
			{keys: "m", desc: "mouse capture on/off"},
			{keys: "v", desc: "line-select mode"},
			{keys: "c", desc: "clear focused logs"},
			{keys: "h", desc: "hide shortcuts"},
		}, width)...)
		lines = append(lines, renderHelpRows([]helpItem{
			{keys: "r", desc: "restart focused"},
			{keys: "f/b", desc: "restart fe/be"},
			{keys: "ctrl+f/ctrl+r", desc: "rebuild+restart"},
			{keys: "ctrl+g", desc: "migrate+restart backend"},
			{keys: "up/down + enter/y", desc: "copy selected line (line mode)"},
		}, width)...)
	} else {
		hint := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("press h to show shortcut help")
		lines = append(lines, clampWidth(hint, width))
	}
	lines = append(lines, status)
	return lines
}

func (m viewState) appName() string {
	name := strings.TrimSpace(m.pm.cfg.AppName)
	if name != "" && !strings.EqualFold(name, "app") {
		return displayName(name)
	}
	candidate := strings.TrimSpace(m.pm.cfg.BackendBinName)
	if candidate == "" {
		candidate = strings.TrimSpace(filepath.Base(m.pm.cfg.BackendBinary()))
	}
	candidate = strings.TrimSuffix(candidate, ".exe")
	candidate = strings.TrimSuffix(candidate, "-server")
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "App"
	}
	return displayName(candidate)
}

func displayName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "App"
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func colorizeStatusLine(value string) string {
	prefix := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("status: ")
	return prefix + colorizeStatusValue(value)
}

func colorizeStatusValue(value string) string {
	plain := strings.ToLower(value)
	switch {
	case strings.Contains(plain, "failed"), strings.Contains(plain, "error:"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(value)
	case strings.Contains(plain, "in progress"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(value)
	case strings.Contains(plain, "succeeded"), strings.Contains(plain, "started"), strings.Contains(plain, "running"), strings.Contains(plain, "exit=0"), strings.Contains(plain, "done"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(value)
	case strings.Contains(plain, "canceled"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(value)
	case strings.Contains(plain, "exit="):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render(value)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(value)
	}
}

func rainbowTitle(text string) string {
	stops := []string{
		"#ff1f5a", "#ff8f1f", "#ffe44d", "#4ce06b", "#39d3ff", "#4f6bff", "#c45bff",
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}
	parts := make([]string, 0, len(runes))
	span := float64(max(len(stops)-1, 1))
	for i := range runes {
		t := float64(i) / float64(max(len(runes)-1, 1))
		x := t * span
		left := int(x)
		if left >= len(stops)-1 {
			left = len(stops) - 1
		}
		right := min(left+1, len(stops)-1)
		mix := x - float64(left)
		color := interpolateHex(stops[left], stops[right], mix)
		parts = append(parts, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render(string(runes[i])))
	}
	return strings.Join(parts, "")
}

func interpolateHex(a, b string, t float64) string {
	ar, ag, ab := parseHexRGB(a)
	br, bg, bb := parseHexRGB(b)
	lerp := func(x, y int) int {
		return int(float64(x) + (float64(y)-float64(x))*t)
	}
	return fmt.Sprintf("#%02x%02x%02x", lerp(ar, br), lerp(ag, bg), lerp(ab, bb))
}

func parseHexRGB(s string) (int, int, int) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return 255, 255, 255
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 255, 255, 255
	}
	return int((v >> 16) & 0xff), int((v >> 8) & 0xff), int(v & 0xff)
}
