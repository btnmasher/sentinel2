package esi

import (
	"net/http"
	"sync"
	"time"
)

type affiliationCacheEntry struct {
	CorporationID int
	AllianceID    int
	ETag          string
	ExpiresAt     time.Time
}

type affiliationCache struct {
	mu      sync.Mutex
	entries map[int]affiliationCacheEntry
}

func newAffiliationCache() *affiliationCache {
	return &affiliationCache{entries: make(map[int]affiliationCacheEntry)}
}

func (c *affiliationCache) get(id int) (affiliationCacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[id]
	if !ok {
		return affiliationCacheEntry{}, false
	}

	if entry.ExpiresAt.IsZero() || time.Now().After(entry.ExpiresAt) {
		return affiliationCacheEntry{}, false
	}
	return entry, true
}

func (c *affiliationCache) getAny(id int) (affiliationCacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[id]
	return entry, ok
}

func (c *affiliationCache) etag(id int) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[id]
	if !ok || entry.ETag == "" {
		return "", false
	}
	return entry.ETag, true
}

func (c *affiliationCache) set(id, corpID, allianceID int, resp *http.Response) {
	entry := affiliationCacheEntry{
		CorporationID: corpID,
		AllianceID:    allianceID,
		ETag:          "",
		ExpiresAt:     time.Time{},
	}

	if resp != nil {
		entry.ETag = resp.Header.Get("ETag")
		entry.ExpiresAt = parseESIExpires(resp)
	}
	c.mu.Lock()
	c.entries[id] = entry
	c.mu.Unlock()
}

func (c *affiliationCache) refreshExpiry(id int, resp *http.Response) {
	if resp == nil {
		return
	}
	expiresAt := parseESIExpires(resp)
	if expiresAt.IsZero() {
		return
	}
	c.mu.Lock()
	entry, ok := c.entries[id]
	if ok {
		entry.ExpiresAt = expiresAt
		c.entries[id] = entry
	}
	c.mu.Unlock()
}
