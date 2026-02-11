package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	RequestIDKey = "request_id"
)

func EnsureRequestID(c *core.RequestEvent) string {
	if c == nil {
		return ""
	}
	if existing, ok := c.Get(RequestIDKey).(string); ok && existing != "" {
		return existing
	}
	requestID := newRequestID()
	c.Set(RequestIDKey, requestID)
	return requestID
}

func RequestID(c *core.RequestEvent) string {
	if c == nil {
		return ""
	}
	if existing, ok := c.Get(RequestIDKey).(string); ok {
		return existing
	}
	return ""
}

func New(app *pocketbase.PocketBase) *Logger {
	if app == nil {
		return &Logger{logger: slog.Default()}
	}
	return &Logger{logger: app.Logger(), ctx: context.Background()}
}

func WithRequest(app *pocketbase.PocketBase, c *core.RequestEvent) *Logger {
	if app == nil {
		return &Logger{logger: slog.Default()}
	}
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	return &Logger{
		logger: app.Logger(),
		ctx:    ctx,
		attrs:  requestAttrs(c),
	}
}

type Logger struct {
	logger *slog.Logger
	ctx    context.Context
	attrs  []slog.Attr
}

type Fields map[string]any

func (l *Logger) With(attrs ...slog.Attr) *Logger {
	if l == nil {
		return l
	}
	out := &Logger{
		logger: l.logger,
		ctx:    l.ctx,
		attrs:  append([]slog.Attr{}, l.attrs...),
	}
	out.attrs = append(out.attrs, attrs...)
	return out
}

func (l *Logger) WithFields(fields Fields) *Logger {
	if l == nil || len(fields) == 0 {
		return l
	}
	attrs := make([]slog.Attr, 0, len(fields))
	for key, value := range fields {
		attrs = append(attrs, slog.Any(key, value))
	}
	return l.With(attrs...)
}

func (l *Logger) WithErr(err error) *Logger {
	if err == nil {
		return l
	}
	return l.With(slog.Any("error", err))
}

func (l *Logger) Debug(msg string, attrs ...slog.Attr) {
	l.log(slog.LevelDebug, msg, attrs...)
}

func (l *Logger) Info(msg string, attrs ...slog.Attr) {
	l.log(slog.LevelInfo, msg, attrs...)
}

func (l *Logger) Warn(msg string, attrs ...slog.Attr) {
	l.log(slog.LevelWarn, msg, attrs...)
}

func (l *Logger) Error(msg string, attrs ...slog.Attr) {
	l.log(slog.LevelError, msg, attrs...)
}

func (l *Logger) log(level slog.Level, msg string, attrs ...slog.Attr) {
	if l == nil || l.logger == nil {
		return
	}
	if level < minLevel {
		return
	}
	callSite := callerAttrs()
	all := append([]slog.Attr{}, l.attrs...)
	all = append(all, callSite...)
	all = append(all, attrs...)
	if !prettyViaPB {
		prettyPrint(level, msg, all)
	}
	if !jsonViaPB {
		writeJSONLog(level, msg, all)
	}
	l.logger.LogAttrs(l.ctx, level, msg, all...)
}

func callerAttrs() []slog.Attr {
	pcs := make([]uintptr, 16)
	n := runtime.Callers(3, pcs) // skip runtime.Callers, callerAttrs, and Logger.log
	frames := runtime.CallersFrames(pcs[:n])

	var file string
	var line int
	var fullFunc string

	for {
		frame, more := frames.Next()
		if frame.Function != "" && !strings.Contains(frame.Function, "/internal/logging.") && !strings.HasPrefix(frame.Function, "sentinel2/internal/logging.") {
			file = frame.File
			line = frame.Line
			fullFunc = frame.Function
			break
		}
		if !more {
			break
		}
	}

	funcName := ""
	pkgPath := ""
	if fullFunc != "" {
		if dot := strings.LastIndex(fullFunc, "."); dot >= 0 {
			pkgPath = fullFunc[:dot]
			funcName = fullFunc[dot+1:]
		} else {
			funcName = fullFunc
		}
	}
	return []slog.Attr{
		slog.String("source_pkg", pkgPath),
		slog.String("source_file", filepath.Base(file)),
		slog.String("source_func", funcName),
		slog.Int("source_line", line),
	}
}

func requestAttrs(c *core.RequestEvent) []slog.Attr {
	if c == nil || c.Request == nil {
		return nil
	}
	attrs := []slog.Attr{
		slog.String("request_id", EnsureRequestID(c)),
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
		slog.String("ip", clientIP(c.Request, c)),
	}
	if c.Request.Pattern != "" {
		attrs = append(attrs, slog.String("pattern", c.Request.Pattern))
	}
	if c.Auth != nil {
		attrs = append(attrs,
			slog.String("auth_provider", c.Auth.GetString("auth_provider")),
			slog.Int("character_id", c.Auth.GetInt("eve_character_id")),
			slog.String("character_name", c.Auth.GetString("eve_character_name")),
			slog.String("access_level", c.Auth.GetString("access_level")),
		)
	}
	return attrs
}

func clientIP(req *http.Request, c *core.RequestEvent) string {
	if c != nil {
		if realIP := c.RealIP(); realIP != "" {
			return realIP
		}
	}
	if req == nil {
		return ""
	}
	return req.RemoteAddr
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}
