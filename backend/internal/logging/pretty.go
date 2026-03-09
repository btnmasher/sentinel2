package logging

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	pblogger "github.com/pocketbase/pocketbase/tools/logger"
	"golang.org/x/term"
)

var (
	minLevel      = slog.LevelInfo
	prettyEnabled = false
	prettyViaPB   = false
	jsonEnabled   = false
	jsonPath      = ""
	jsonViaPB     = false
)

const prettyFieldIndent = 2

type Options struct {
	MinLevel             slog.Level
	PrettyEnabled        bool
	UsePocketBasePrinter bool
	JSONEnabled          bool
	JSONPath             string
	UsePocketBaseJSON    bool
}

func Configure(opts Options) {
	minLevel = opts.MinLevel
	prettyEnabled = opts.PrettyEnabled
	jsonEnabled = opts.JSONEnabled
	jsonPath = strings.TrimSpace(opts.JSONPath)
	prettyViaPB = false
	jsonViaPB = false
	if opts.PrettyEnabled && opts.UsePocketBasePrinter {
		installPocketBasePrettyPrinter()
		prettyViaPB = true
	}

	if opts.JSONEnabled && opts.UsePocketBaseJSON {
		installPocketBaseJSONWriter()
		jsonViaPB = true
	}
}

func ParseLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

func shouldPrettyPrint() bool {
	if !prettyEnabled {
		return false
	}
	return term.IsTerminal(int(os.Stderr.Fd()))
}

func prettyPrint(level slog.Level, msg string, attrs []slog.Attr) {
	if !shouldPrettyPrint() {
		return
	}

	ts := time.Now().Format("15:04:05.000")
	levelLabel, levelStyle := levelBadge(level)
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(ts),
		" ",
		levelStyle.Render(levelLabel),
		" ",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render(msg),
	)

	fields := renderAttrs(attrs)
	if fields != "" {
		header = header + "\n" + fields
	}

	fmt.Fprintln(os.Stderr, header)
}

func prettyPrintFromPB(log *pblogger.Log) {
	if !shouldPrettyPrint() || log == nil {
		return
	}
	level := log.Level
	attrs := attrsFromMap(log.Data)
	prettyPrint(level, log.Message, attrs)
}

func writeJSONFromPB(log *pblogger.Log) {
	if !jsonEnabled || log == nil {
		return
	}
	level := log.Level
	attrs := attrsFromMap(log.Data)
	writeJSONLog(level, log.Message, attrs)
}

func levelBadge(level slog.Level) (string, lipgloss.Style) {
	base := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	switch {
	case level <= slog.LevelDebug:
		return "DEBUG", base.Foreground(lipgloss.Color("236")).Background(lipgloss.Color("239"))
	case level <= slog.LevelInfo:
		return "INFO", base.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("31"))
	case level <= slog.LevelWarn:
		return "WARN", base.Foreground(lipgloss.Color("234")).Background(lipgloss.Color("214"))
	default:
		return "ERROR", base.Foreground(lipgloss.Color("231")).Background(lipgloss.Color("160"))
	}
}

func renderAttrs(attrs []slog.Attr) string {
	if len(attrs) == 0 {
		return ""
	}

	values := map[string]any{}
	for _, attr := range attrs {
		key, value := resolveAttr(attr)
		if key == "" {
			continue
		}
		values[key] = value
	}

	if len(values) == 0 {
		return ""
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, keyStyle.Render(k)+sepStyle.Render("=")+valStyle.Render(fmt.Sprintf("%v", values[k])))
	}

	return lipgloss.NewStyle().MarginLeft(prettyFieldIndent).Render(strings.Join(parts, " "))
}

func resolveAttr(attr slog.Attr) (key string, value any) {
	if attr.Key == "" {
		return "", nil
	}
	resolvedValue := attr.Value.Resolve()
	switch resolvedValue.Kind() {
	case slog.KindGroup:
		inner := map[string]any{}
		for _, groupAttr := range resolvedValue.Group() {
			key, val := resolveAttr(groupAttr)
			if key != "" {
				inner[key] = val
			}
		}
		return attr.Key, inner
	default:
		return attr.Key, resolvedValue.Any()
	}
}

func attrsFromMap(data map[string]any) []slog.Attr {
	if len(data) == 0 {
		return nil
	}
	attrs := make([]slog.Attr, 0, len(data))
	for key, value := range data {
		attrs = append(attrs, slog.Any(key, value))
	}
	return attrs
}
