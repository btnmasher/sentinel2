package queryhelpers

import (
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func AppendAnd(filter, clause string) string {
	filter = strings.TrimSpace(filter)
	clause = strings.TrimSpace(clause)
	if filter == "" {
		return clause
	}
	if clause == "" {
		return filter
	}
	return filter + " && " + clause
}

// BuildOrEqualsFilter builds "field = {:id0} || field = {:id1} ..." with params.
func BuildOrEqualsFilter[T any](field string, values []T) (string, dbx.Params) {
	if len(values) == 0 {
		return "", dbx.Params{}
	}
	var filter strings.Builder
	filter.WriteString(field)
	filter.WriteString(" = {:id0}")
	params := dbx.Params{"id0": values[0]}
	for i := 1; i < len(values); i++ {
		key := "id" + strconv.Itoa(i)
		filter.WriteString(" || ")
		filter.WriteString(field)
		filter.WriteString(" = {:")
		filter.WriteString(key)
		filter.WriteString("}")
		params[key] = values[i]
	}
	return filter.String(), params
}

// BuildOrEqualsFilterWithPrefix builds
// "field = {:<prefix>0} || field = {:<prefix>1} ..." with prefixed params.
func BuildOrEqualsFilterWithPrefix[T any](field, prefix string, values []T) (string, dbx.Params) {
	if prefix == "" {
		return BuildOrEqualsFilter(field, values)
	}
	if len(values) == 0 {
		return "", dbx.Params{}
	}
	var filter strings.Builder
	firstKey := prefix + "0"
	filter.WriteString(field)
	filter.WriteString(" = {:")
	filter.WriteString(firstKey)
	filter.WriteString("}")
	params := dbx.Params{firstKey: values[0]}
	for i := 1; i < len(values); i++ {
		key := prefix + strconv.Itoa(i)
		filter.WriteString(" || ")
		filter.WriteString(field)
		filter.WriteString(" = {:")
		filter.WriteString(key)
		filter.WriteString("}")
		params[key] = values[i]
	}
	return filter.String(), params
}

func AppendDateTimeClauseUTC(filter string, params dbx.Params, value *time.Time, paramKey, clause string) string {
	if value == nil {
		return filter
	}
	params[paramKey] = value.UTC().Format(time.RFC3339)
	return AppendAnd(filter, clause)
}

func SetOptional[T any](record *core.Record, field string, value *T, normalize func(T) any) {
	if record == nil || value == nil {
		return
	}
	if normalize == nil {
		record.Set(field, *value)
		return
	}
	record.Set(field, normalize(*value))
}

func ValueOrTrim(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
