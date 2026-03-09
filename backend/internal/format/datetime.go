package format

import (
	"errors"
	"strings"
	"time"
)

const (
	layoutDateTimeMinute    = "2006-01-02T15:04"
	layoutDateTimeDotSecond = "2006.01.02 15:04:05"
	layoutDateTimePB        = "2006-01-02 15:04:05.000Z"
)

var (
	ErrMissingDateTime = errors.New("missing datetime")
	ErrInvalidDateTime = errors.New("invalid datetime")
)

// ParseDateTimeFlexibleUTC accepts RFC3339, PocketBase datetime format, or YYYY-MM-DDTHH:MM and returns UTC.
func ParseDateTimeFlexibleUTC(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, ErrMissingDateTime
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}

	if parsed, err := time.Parse(layoutDateTimeMinute, value); err == nil {
		return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), 0, 0, time.UTC), nil
	}

	if parsed, err := time.Parse(layoutDateTimeDotSecond, value); err == nil {
		return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, time.UTC), nil
	}

	if parsed, err := time.Parse(layoutDateTimePB, value); err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, ErrInvalidDateTime
}
