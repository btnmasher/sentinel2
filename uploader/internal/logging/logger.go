package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Logger struct {
	debugEnabled atomic.Bool
	inner        *slog.Logger
	pretty       bool
	mu           sync.RWMutex
	nextID       int
	subscribers  map[int]func(Event)
}

type Event struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Fields  map[string]any
}

func New(debug bool) *Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	logger := &Logger{
		inner:       slog.New(handler),
		pretty:      shouldPrettyPrint(),
		subscribers: map[int]func(Event){},
	}
	logger.debugEnabled.Store(debug)
	return logger
}

func Field(key string, value any) slog.Attr {
	return slog.Any(key, value)
}

func (l *Logger) Debugf(format string, args ...any) {
	l.Debug(fmt.Sprintf(format, args...))
}

func (l *Logger) Debug(msg string, fields ...slog.Attr) {
	if l == nil || l.inner == nil || !l.debugEnabled.Load() {
		return
	}
	if l.pretty {
		prettyPrint(slog.LevelDebug, msg, fields)
		l.publish(slog.LevelDebug, msg, fields)
		return
	}
	l.inner.LogAttrs(context.Background(), slog.LevelDebug, msg, fields...)
	l.publish(slog.LevelDebug, msg, fields)
}

func (l *Logger) SetDebugEnabled(enabled bool) {
	if l == nil {
		return
	}
	l.debugEnabled.Store(enabled)
}

func (l *Logger) Info(msg string, fields ...slog.Attr) {
	if l == nil || l.inner == nil {
		return
	}
	if l.pretty {
		prettyPrint(slog.LevelInfo, msg, fields)
		l.publish(slog.LevelInfo, msg, fields)
		return
	}
	l.inner.LogAttrs(context.Background(), slog.LevelInfo, msg, fields...)
	l.publish(slog.LevelInfo, msg, fields)
}

func (l *Logger) Warn(msg string, fields ...slog.Attr) {
	if l == nil || l.inner == nil {
		return
	}
	if l.pretty {
		prettyPrint(slog.LevelWarn, msg, fields)
		l.publish(slog.LevelWarn, msg, fields)
		return
	}
	l.inner.LogAttrs(context.Background(), slog.LevelWarn, msg, fields...)
	l.publish(slog.LevelWarn, msg, fields)
}

func (l *Logger) Error(msg string, fields ...slog.Attr) {
	if l == nil || l.inner == nil {
		return
	}
	if l.pretty {
		prettyPrint(slog.LevelError, msg, fields)
		l.publish(slog.LevelError, msg, fields)
		return
	}
	l.inner.LogAttrs(context.Background(), slog.LevelError, msg, fields...)
	l.publish(slog.LevelError, msg, fields)
}

func (l *Logger) Subscribe(fn func(Event)) func() {
	if l == nil || fn == nil {
		return func() {}
	}
	l.mu.Lock()
	id := l.nextID
	l.nextID++
	l.subscribers[id] = fn
	l.mu.Unlock()
	return func() {
		l.mu.Lock()
		delete(l.subscribers, id)
		l.mu.Unlock()
	}
}

func (l *Logger) publish(level slog.Level, msg string, attrs []slog.Attr) {
	if l == nil {
		return
	}
	l.mu.RLock()
	if len(l.subscribers) == 0 {
		l.mu.RUnlock()
		return
	}
	callbacks := make([]func(Event), 0, len(l.subscribers))
	for _, cb := range l.subscribers {
		callbacks = append(callbacks, cb)
	}
	l.mu.RUnlock()

	event := Event{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
		Fields:  attrsToMap(attrs),
	}
	for _, cb := range callbacks {
		cb(event)
	}
}
