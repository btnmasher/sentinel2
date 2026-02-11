package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type uploaderLogger struct {
	debugEnabled bool
	inner        *slog.Logger
	pretty       bool
}

func newUploaderLogger(debug bool) *uploaderLogger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	return &uploaderLogger{
		debugEnabled: debug,
		inner:        slog.New(handler),
		pretty:       shouldPrettyPrint(),
	}
}

func field(key string, value any) slog.Attr {
	return slog.Any(key, value)
}

func (l *uploaderLogger) Debug(msg string, fields ...slog.Attr) {
	if l == nil || l.inner == nil || !l.debugEnabled {
		return
	}
	if l.pretty {
		prettyPrint(slog.LevelDebug, msg, fields)
		return
	}
	l.inner.LogAttrs(context.Background(), slog.LevelDebug, msg, fields...)
}

func (l *uploaderLogger) Info(msg string, fields ...slog.Attr) {
	if l == nil || l.inner == nil {
		return
	}
	if l.pretty {
		prettyPrint(slog.LevelInfo, msg, fields)
		return
	}
	l.inner.LogAttrs(context.Background(), slog.LevelInfo, msg, fields...)
}

func (l *uploaderLogger) Warn(msg string, fields ...slog.Attr) {
	if l == nil || l.inner == nil {
		return
	}
	if l.pretty {
		prettyPrint(slog.LevelWarn, msg, fields)
		return
	}
	l.inner.LogAttrs(context.Background(), slog.LevelWarn, msg, fields...)
}

func (l *uploaderLogger) Error(msg string, fields ...slog.Attr) {
	if l == nil || l.inner == nil {
		return
	}
	if l.pretty {
		prettyPrint(slog.LevelError, msg, fields)
		return
	}
	l.inner.LogAttrs(context.Background(), slog.LevelError, msg, fields...)
}

func shouldPrettyPrint() bool {
	term := strings.TrimSpace(os.Getenv("TERM"))
	if term == "" || term == "dumb" {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return true
}

func prettyPrint(level slog.Level, msg string, attrs []slog.Attr) {
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

	return lipgloss.NewStyle().MarginLeft(2).Render(strings.Join(parts, " "))
}

func resolveAttr(attr slog.Attr) (string, any) {
	if attr.Key == "" {
		return "", nil
	}
	value := attr.Value.Resolve()
	switch value.Kind() {
	case slog.KindGroup:
		inner := map[string]any{}
		for _, groupAttr := range value.Group() {
			key, val := resolveAttr(groupAttr)
			if key != "" {
				inner[key] = val
			}
		}
		return attr.Key, inner
	default:
		return attr.Key, value.Any()
	}
}
