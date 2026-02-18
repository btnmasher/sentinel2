package format

import (
	"fmt"
	"time"
)

// ParseDateTimeFlexibleUTC accepts RFC3339 or YYYY-MM-DDTHH:MM and returns UTC.
func ParseDateTimeFlexibleUTC(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse("2006-01-02T15:04", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid datetime")
	}
	return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), 0, 0, time.UTC), nil
}
