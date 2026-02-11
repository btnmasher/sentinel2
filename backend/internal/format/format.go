package format

import (
	"fmt"
	"strconv"
)

func CoerceInt64(value any) int64 {
	if value == nil {
		return 0
	}
	parsed, err := strconv.ParseFloat(fmt.Sprint(value), 64)
	if err != nil {
		return 0
	}
	return int64(parsed)
}
