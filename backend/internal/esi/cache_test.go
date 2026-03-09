package esi

import (
	"net/http"
	"testing"
	"time"
)

func TestESIPublicCacheSetAndGet(t *testing.T) {
	client := &ESIPublicClient{
		cacheTTL: time.Hour,
		cache:    map[string]esiNameCache{},
	}

	key := corporationCacheKey(123)
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Set("ETag", "etag-1")
	resp.Header.Set("Expires", time.Now().Add(30*time.Minute).UTC().Format(http.TimeFormat))
	client.setCached(key, "Acme Corp", resp)

	name, ok := client.getCached(key)
	if !ok {
		t.Fatalf("expected cached entry")
	}

	if name != "Acme Corp" {
		t.Fatalf("unexpected name: %q", name)
	}

	entry, ok := client.getAny(key)
	if !ok {
		t.Fatalf("expected cache entry")
	}

	if entry.etag != "etag-1" {
		t.Fatalf("unexpected etag: %q", entry.etag)
	}
}

func TestESIPublicCacheGetDeletesExpiredEntry(t *testing.T) {
	client := &ESIPublicClient{
		cacheTTL: time.Hour,
		cache: map[string]esiNameCache{
			"k": {value: "v", expires: time.Now().Add(-time.Minute)},
		},
	}

	if _, ok := client.getCached("k"); ok {
		t.Fatalf("expected cache miss for expired value")
	}

	if _, ok := client.getAny("k"); ok {
		t.Fatalf("expected expired entry to be removed")
	}
}

func TestESIPublicCacheNotModifiedRefreshesExpiry(t *testing.T) {
	client := &ESIPublicClient{
		cacheTTL: time.Hour,
		cache: map[string]esiNameCache{
			"k": {
				value:   "Persisted",
				etag:    "etag-2",
				expires: time.Now().Add(-time.Minute),
			},
		},
	}
	newExpiry := time.Now().Add(10 * time.Minute).UTC()
	httpResp := &http.Response{
		StatusCode: http.StatusNotModified,
		Header: http.Header{
			"Expires": []string{newExpiry.Format(http.TimeFormat)},
		},
	}

	name, handled := client.cachedNameFromNotModified("k", httpResp)
	if !handled {
		t.Fatalf("expected 304 to be handled from cache")
	}

	if name != "Persisted" {
		t.Fatalf("unexpected cached name: %q", name)
	}

	entry, ok := client.getAny("k")
	if !ok {
		t.Fatalf("expected cache entry to remain")
	}

	if entry.expires.Before(newExpiry.Add(-1*time.Second)) || entry.expires.After(newExpiry.Add(time.Second)) {
		t.Fatalf("expected refreshed expiry near %s, got %s", newExpiry, entry.expires)
	}
}
