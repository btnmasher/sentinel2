package queryhelpers

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"sentinel2/internal/format"
)

type DateRangeParseError struct {
	Field string
	Value string
}

func (e *DateRangeParseError) Error() string {
	if e == nil {
		return "invalid date range"
	}
	return fmt.Sprintf("invalid %s", e.Field)
}

// ParseFlexibleDateRangeUTC parses optional start/end datetime and date keys.
// Datetime supports RFC3339 and YYYY-MM-DDTHH:MM; date uses YYYY-MM-DD.
func ParseFlexibleDateRangeUTC(values url.Values, startAtKey, endAtKey, startDateKey, endDateKey string) (startAt, endAt *time.Time, err error) {
	if values == nil {
		return nil, nil, nil
	}
	startAtRaw := strings.TrimSpace(values.Get(startAtKey))
	endAtRaw := strings.TrimSpace(values.Get(endAtKey))
	startDateRaw := strings.TrimSpace(values.Get(startDateKey))
	endDateRaw := strings.TrimSpace(values.Get(endDateKey))
	if startAtRaw != "" {
		parsed, err := format.ParseDateTimeFlexibleUTC(startAtRaw)
		if err != nil {
			return nil, nil, &DateRangeParseError{Field: startAtKey, Value: startAtRaw}
		}
		startAt = &parsed
	} else if startDateRaw != "" {
		parsed, err := time.Parse("2006-01-02", startDateRaw)
		if err != nil {
			return nil, nil, &DateRangeParseError{Field: startDateKey, Value: startDateRaw}
		}
		startAt = &parsed
	}

	if endAtRaw != "" {
		parsed, err := format.ParseDateTimeFlexibleUTC(endAtRaw)
		if err != nil {
			return nil, nil, &DateRangeParseError{Field: endAtKey, Value: endAtRaw}
		}
		endAt = &parsed
	} else if endDateRaw != "" {
		parsed, err := time.Parse("2006-01-02", endDateRaw)
		if err != nil {
			return nil, nil, &DateRangeParseError{Field: endDateKey, Value: endDateRaw}
		}
		endOfDay := time.Date(
			parsed.Year(),
			parsed.Month(),
			parsed.Day(),
			23,
			59,
			59,
			int(time.Second-time.Nanosecond),
			time.UTC,
		)
		endAt = &endOfDay
	}

	return startAt, endAt, nil
}
