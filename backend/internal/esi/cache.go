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
	if expires == "" {
		cacheControl := resp.Header.Get("Cache-Control")
		if cacheControl != "" {
			parts := strings.Split(cacheControl, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "max-age=") {
					value := strings.TrimPrefix(part, "max-age=")
					if seconds, parseErr := strconv.Atoi(value); parseErr == nil && seconds > 0 {
						return time.Now().Add(time.Duration(seconds) * time.Second)
					}
				}
			}
		}
		return time.Time{}
	}
	parsed, parseErr := http.ParseTime(expires)
	if parseErr != nil {
		return time.Time{}
	}
	return parsed
}
