package logging

import (
	"fmt"
	"sort"
	"strings"
)

const clipLimit = 240

func Truncate(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	if value == "" {
		return "<empty>"
	}
	if len(value) > clipLimit {
		return value[:clipLimit] + "..."
	}
	return value
}

func AppendWithLimit(current string, next string, limit int) string {
	combined := current + next
	if len(combined) > limit {
		return combined[len(combined)-limit:]
	}
	return combined
}

func FormatEventLine(event Event) string {
	ts := event.Time.Format("15:04:05")
	level := strings.ToUpper(event.Level.String())
	fields := ""
	if len(event.Fields) > 0 {
		keys := make([]string, 0, len(event.Fields))
		for key := range event.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%v", key, event.Fields[key]))
		}
		fields = " " + strings.Join(parts, " ")
	}
	return fmt.Sprintf("%s [%s] %s%s\n", ts, level, event.Message, fields)
}
