package esi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"
	"github.com/fnt-eve/goesi-openapi/esi"
	retry "github.com/sethvargo/go-retry"
)

type ESIPublicClient struct {
	client   *esi.APIClient
	cacheTTL time.Duration
	mu       sync.RWMutex
	cache    map[string]esiNameCache
	throttle *esiThrottle
	limiter  *esiRateLimiter
}

type esiNameCache struct {
	value   string
	etag    string
	expires time.Time
}

const (
	defaultPublicCacheTTL      = 6 * time.Hour
	publicBackoffBase          = 200 * time.Millisecond
	publicBackoffJitter        = 100 * time.Millisecond
	publicBackoffCap           = 2 * time.Second
	publicBackoffMaxRetryCount = 3
)

func NewESIPublicClient(userAgent string) *ESIPublicClient {
	return &ESIPublicClient{
		client:   goesi.NewPublicESIClient(userAgent),
		cacheTTL: defaultPublicCacheTTL,
		cache:    make(map[string]esiNameCache),
		throttle: newESIThrottle(nil),
		limiter:  globalESILimiter,
	}
}

func (c *ESIPublicClient) AllianceName(ctx context.Context, allianceID int) (string, error) {
	key := allianceCacheKey(allianceID)
	name, err := c.fetchName(ctx, key, "alliance_id", allianceID, func(ctx context.Context, etag string) (string, *http.Response, error) {
		request := c.client.AllianceAPI.GetAlliancesAllianceId(ctx, int64(allianceID))
		if etag != "" {
			request = request.IfNoneMatch(etag)
		}
		resp, httpResp, respErr := request.Execute()
		if resp == nil {
			return "", httpResp, respErr
		}
		return resp.Name, httpResp, respErr
	})
	if err != nil {
		return "", err
	}
	return name, nil
}

func (c *ESIPublicClient) CorporationName(ctx context.Context, corpID int) (string, error) {
	key := corporationCacheKey(corpID)
	name, err := c.fetchName(ctx, key, "corp_id", corpID, func(ctx context.Context, etag string) (string, *http.Response, error) {
		request := c.client.CorporationAPI.GetCorporationsCorporationId(ctx, int64(corpID))
		if etag != "" {
			request = request.IfNoneMatch(etag)
		}
		resp, httpResp, respErr := request.Execute()
		if resp == nil {
			return "", httpResp, respErr
		}
		return resp.Name, httpResp, respErr
	})
	if err != nil {
		return "", err
	}
	return name, nil
}

func (c *ESIPublicClient) ThrottleDelay() time.Duration {
	if c == nil || c.throttle == nil {
		return 0
	}
	return c.throttle.delay()
}

func (c *ESIPublicClient) retryBackoff() retry.Backoff {
	backoff := retry.NewExponential(publicBackoffBase)
	backoff = retry.WithJitter(publicBackoffJitter, backoff)
	backoff = retry.WithCappedDuration(publicBackoffCap, backoff)
	backoff = retry.WithMaxRetries(publicBackoffMaxRetryCount, backoff)
	return backoff
}

func shouldRetryPublicESI(resp *http.Response, err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	if resp == nil {
		return true
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return resp.StatusCode >= http.StatusInternalServerError
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

func (c *ESIPublicClient) fetchName(
	ctx context.Context,
	key string,
	idLabel string,
	idValue int,
	call func(context.Context, string) (string, *http.Response, error),
) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("esi public client not configured")
	}
	if name, ok := c.getCached(key); ok {
		return name, nil
	}

	var name string
	retryErr := retry.Do(ctx, c.retryBackoff(), func(ctx context.Context) error {
		c.limiter.wait(ctx)
		c.throttle.wait(ctx)
		nextName, httpResp, respErr := c.fetchNameResponse(ctx, key, call)
		if httpResp != nil && httpResp.Body != nil {
			defer func() { _ = httpResp.Body.Close() }()
		}
		if httpResp != nil {
			c.throttle.update(httpResp)
		}
		cachedName, handled := c.cachedNameFromNotModified(key, httpResp)
		if handled {
			name = cachedName
			return nil
		}
		if respErr != nil {
			return wrapPublicNameError(idLabel, idValue, httpResp, respErr)
		}
		name = nextName
		c.setCached(key, name, httpResp)
		return nil
	})
	if retryErr != nil {
		return "", retryErr
	}
	return name, nil
}

func (c *ESIPublicClient) fetchNameResponse(
	ctx context.Context,
	key string,
	call func(context.Context, string) (string, *http.Response, error),
) (string, *http.Response, error) {
	etag := ""
	if entry, ok := c.getAny(key); ok && entry.etag != "" {
		etag = entry.etag
	}
	return call(ctx, etag)
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

func wrapPublicNameError(idLabel string, idValue int, httpResp *http.Response, respErr error) error {
	err := fmt.Errorf("esi public name fetch failed (%s=%d): %w", idLabel, idValue, respErr)
	if shouldRetryPublicESI(httpResp, respErr) {
		return retry.RetryableError(err)
	}
	return err
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
