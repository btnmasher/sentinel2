package logging

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

var jsonMu sync.Mutex

const jsonFilePerm = 0o600

type jsonEntry struct {
	Time   string         `json:"time"`
	Level  string         `json:"level"`
	Msg    string         `json:"msg"`
	Fields map[string]any `json:"fields,omitempty"`
}

func writeJSONLog(level slog.Level, msg string, attrs []slog.Attr) {
	if !jsonEnabled || jsonPath == "" {
		return
	}
	entry := jsonEntry{
		Time:  time.Now().UTC().Format(time.RFC3339Nano),
		Level: level.String(),
		Msg:   msg,
	}
	if fields := attrsToMap(attrs); len(fields) > 0 {
		entry.Fields = fields
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return
	}

	jsonMu.Lock()
	defer jsonMu.Unlock()

	file, err := os.OpenFile(jsonPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, jsonFilePerm)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	_, _ = fmt.Fprintln(file, string(payload))
}

func attrsToMap(attrs []slog.Attr) map[string]any {
	if len(attrs) == 0 {
		return nil
	}
	out := map[string]any{}
	for _, attr := range attrs {
		key, value := resolveAttr(attr)
		if key == "" {
			continue
		}
		out[key] = value
	}
	return out
}
