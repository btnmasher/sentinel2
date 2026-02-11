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

func NewESIPublicClient(userAgent string) *ESIPublicClient {
	return &ESIPublicClient{
		client:   goesi.NewPublicESIClient(userAgent),
		cacheTTL: 6 * time.Hour,
		cache:    make(map[string]esiNameCache),
		throttle: newESIThrottle(nil),
		limiter:  globalESILimiter,
	}
}

func (c *ESIPublicClient) AllianceName(ctx context.Context, allianceID int) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("esi public client not configured")
	}
	if name, ok := c.getCached(allianceCacheKey(allianceID)); ok {
		return name, nil
	}

	var name string
	retryErr := retry.Do(ctx, c.retryBackoff(), func(ctx context.Context) error {
		c.limiter.wait(ctx)
		c.throttle.wait(ctx)
		request := c.client.AllianceAPI.GetAlliancesAllianceId(ctx, int64(allianceID))
		if entry, ok := c.getAny(allianceCacheKey(allianceID)); ok && entry.etag != "" {
			request = request.IfNoneMatch(entry.etag)
		}
		resp, httpResp, respErr := request.Execute()
		if httpResp != nil {
			c.throttle.update(httpResp)
		}
		if httpResp != nil && httpResp.StatusCode == http.StatusNotModified {
			if entry, ok := c.getAny(allianceCacheKey(allianceID)); ok {
				c.refreshExpiry(allianceCacheKey(allianceID), httpResp)
				name = entry.value
				return nil
			}
		}
		if respErr != nil {
			if shouldRetryPublicESI(httpResp, respErr) {
				return retry.RetryableError(fmt.Errorf("esi public alliance name fetch failed (alliance_id=%d): %w", allianceID, respErr))
			}
			return fmt.Errorf("esi public alliance name fetch failed (alliance_id=%d): %w", allianceID, respErr)
		}
		name = resp.Name
		c.setCached(allianceCacheKey(allianceID), name, httpResp)
		return nil
	})
	if retryErr != nil {
		return "", retryErr
	}
	return name, nil
}

func (c *ESIPublicClient) CorporationName(ctx context.Context, corpID int) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("esi public client not configured")
	}
	if name, ok := c.getCached(corporationCacheKey(corpID)); ok {
		return name, nil
	}

	var name string
	retryErr := retry.Do(ctx, c.retryBackoff(), func(ctx context.Context) error {
		c.limiter.wait(ctx)
		c.throttle.wait(ctx)
		request := c.client.CorporationAPI.GetCorporationsCorporationId(ctx, int64(corpID))
		if entry, ok := c.getAny(corporationCacheKey(corpID)); ok && entry.etag != "" {
			request = request.IfNoneMatch(entry.etag)
		}
		resp, httpResp, respErr := request.Execute()
		if httpResp != nil {
			c.throttle.update(httpResp)
		}
		if httpResp != nil && httpResp.StatusCode == http.StatusNotModified {
			if entry, ok := c.getAny(corporationCacheKey(corpID)); ok {
				c.refreshExpiry(corporationCacheKey(corpID), httpResp)
				name = entry.value
				return nil
			}
		}
		if respErr != nil {
			if shouldRetryPublicESI(httpResp, respErr) {
				return retry.RetryableError(fmt.Errorf("esi public corporation name fetch failed (corp_id=%d): %w", corpID, respErr))
			}
			return fmt.Errorf("esi public corporation name fetch failed (corp_id=%d): %w", corpID, respErr)
		}
		name = resp.Name
		c.setCached(corporationCacheKey(corpID), name, httpResp)
		return nil
	})
	if retryErr != nil {
		return "", retryErr
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
	backoff := retry.NewExponential(200 * time.Millisecond)
	backoff = retry.WithJitter(100*time.Millisecond, backoff)
	backoff = retry.WithCappedDuration(2*time.Second, backoff)
	backoff = retry.WithMaxRetries(3, backoff)
	return backoff
}

func shouldRetryPublicESI(resp *http.Response, err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
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

func (c *ESIPublicClient) setCached(key string, value string, resp *http.Response) {
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
