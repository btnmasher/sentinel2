package esi

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parseESIExpires(resp *http.Response) time.Time {
	if resp == nil {
		return time.Time{}
	}
	expires := resp.Header.Get("Expires")
	if expires != "" {
		parsed, parseErr := http.ParseTime(expires)
		if parseErr == nil {
			return parsed
		}
		return time.Time{}
	}
	seconds, ok := maxAgeFromCacheControl(resp.Header.Get("Cache-Control"))
	if !ok {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(seconds) * time.Second)
}

func maxAgeFromCacheControl(cacheControl string) (int, bool) {
	if cacheControl == "" {
		return 0, false
	}
	for part := range strings.SplitSeq(cacheControl, ",") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "max-age=") {
			continue
		}
		value := strings.TrimPrefix(part, "max-age=")
		seconds, parseErr := strconv.Atoi(value)
		if parseErr == nil && seconds > 0 {
			return seconds, true
		}
	}
	return 0, false
}
