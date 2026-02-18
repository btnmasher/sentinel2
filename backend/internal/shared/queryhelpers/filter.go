package queryhelpers

import (
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
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
