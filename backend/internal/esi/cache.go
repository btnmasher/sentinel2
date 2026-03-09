package esi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type esiNameCache struct {
	value   string
	etag    string
	expires time.Time
}

func (c *ESIPublicClient) getCached(key string) (string, bool) {
	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}

	if time.Now().After(entry.expires) {
		c.mu.Lock()
		delete(c.cache, key)
		c.mu.Unlock()
		return "", false
	}
	return entry.value, true
}

func (c *ESIPublicClient) getAny(key string) (esiNameCache, bool) {
	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()
	return entry, ok
}

func (c *ESIPublicClient) refreshExpiry(key string, resp *http.Response) {
	expires := parseESIExpires(resp)
	if expires.IsZero() {
		return
	}
	c.mu.Lock()
	entry, ok := c.cache[key]
	if ok {
		entry.expires = expires
		c.cache[key] = entry
	}
	c.mu.Unlock()
}

func (c *ESIPublicClient) setCached(key, value string, resp *http.Response) {
	c.mu.Lock()
	entry := esiNameCache{
		value:   value,
		etag:    "",
		expires: time.Now().Add(c.cacheTTL),
	}

	if resp != nil {
		entry.etag = resp.Header.Get("ETag")
		entry.expires = coalesceExpiry(parseESIExpires(resp), c.cacheTTL)
	}
	c.cache[key] = entry
	c.mu.Unlock()
}

func (c *ESIPublicClient) cachedNameFromNotModified(key string, httpResp *http.Response) (string, bool) {
	if httpResp == nil || httpResp.StatusCode != http.StatusNotModified {
		return "", false
	}
	entry, ok := c.getAny(key)
	if !ok {
		return "", false
	}
	c.refreshExpiry(key, httpResp)
	return entry.value, true
}

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

func coalesceExpiry(expires time.Time, fallback time.Duration) time.Time {
	if expires.IsZero() {
		return time.Now().Add(fallback)
	}
	return expires
}

func allianceCacheKey(allianceID int) string {
	return fmt.Sprintf("alliance:%d", allianceID)
}

func corporationCacheKey(corpID int) string {
	return fmt.Sprintf("corporation:%d", corpID)
}
