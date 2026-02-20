package esi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"
	"github.com/fnt-eve/goesi-openapi/esi"
	retry "github.com/sethvargo/go-retry"
)

const (
	defaultPublicCacheTTL      = 6 * time.Hour
	publicBackoffBase          = 200 * time.Millisecond
	publicBackoffJitter        = 100 * time.Millisecond
	publicBackoffCap           = 2 * time.Second
	publicBackoffMaxRetryCount = 3
)

type ESIPublicClient struct {
	client   *esi.APIClient
	cacheTTL time.Duration
	mu       sync.RWMutex
	cache    map[string]esiNameCache
	throttle *esiThrottle
	limiter  *esiRateLimiter
}

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

func (c *ESIPublicClient) CorporationDetails(ctx context.Context, corporationID int) (name, ticker string, allianceID int, err error) {
	if err := c.ensureConfigured(); err != nil {
		return "", "", 0, err
	}

	var responseName string
	var responseTicker string
	var responseAllianceID int
	//nolint:bodyclose // Body is closed in the callback immediately after Execute().
	httpResp, fetchErr := executeWithPublicRetry(
		c,
		ctx,
		publicRequestOptions{
			operation:   fmt.Sprintf("esi corporation details fetch failed (corp_id=%d)", corporationID),
			notFoundErr: fmt.Errorf("%w: corporation_id=%d", ErrOrganizationInactive, corporationID),
		},
		func(ctx context.Context) (*http.Response, error) {
			response, httpResp, respErr := c.client.CorporationAPI.GetCorporationsCorporationId(ctx, int64(corporationID)).Execute()
			defer closeResponseBody(httpResp)
			if respErr != nil {
				return httpResp, respErr
			}
			if response == nil {
				return httpResp, newNonRetryPublicError(fmt.Sprintf("esi corporation details response empty (corp_id=%d)", corporationID))
			}
			responseName = strings.TrimSpace(response.GetName())
			responseTicker = strings.TrimSpace(response.GetTicker())
			if responseAllianceIDPtr, ok := response.GetAllianceIdOk(); ok {
				responseAllianceID = int(*responseAllianceIDPtr)
			}
			return httpResp, nil
		},
	)
	if fetchErr != nil {
		return "", "", 0, fetchErr
	}
	c.setCached(corporationCacheKey(corporationID), responseName, httpResp)
	return responseName, responseTicker, responseAllianceID, nil
}

func (c *ESIPublicClient) AllianceDetails(ctx context.Context, allianceID int) (name, ticker string, err error) {
	if err := c.ensureConfigured(); err != nil {
		return "", "", err
	}

	var responseName string
	var responseTicker string
	//nolint:bodyclose // Body is closed in the callback immediately after Execute().
	httpResp, fetchErr := executeWithPublicRetry(
		c,
		ctx,
		publicRequestOptions{
			operation:   fmt.Sprintf("esi alliance details fetch failed (alliance_id=%d)", allianceID),
			notFoundErr: fmt.Errorf("%w: alliance_id=%d", ErrOrganizationInactive, allianceID),
		},
		func(ctx context.Context) (*http.Response, error) {
			response, httpResp, respErr := c.client.AllianceAPI.GetAlliancesAllianceId(ctx, int64(allianceID)).Execute()
			defer closeResponseBody(httpResp)
			if respErr != nil {
				return httpResp, respErr
			}
			if response == nil {
				return httpResp, newNonRetryPublicError(fmt.Sprintf("esi alliance details response empty (alliance_id=%d)", allianceID))
			}
			responseName = strings.TrimSpace(response.GetName())
			responseTicker = strings.TrimSpace(response.GetTicker())
			return httpResp, nil
		},
	)
	if fetchErr != nil {
		return "", "", fetchErr
	}
	c.setCached(allianceCacheKey(allianceID), responseName, httpResp)
	return responseName, responseTicker, nil
}

func (c *ESIPublicClient) ResolveOrganizationNames(ctx context.Context, names []string) (*esi.UniverseIdsPost, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("esi public client not configured")
	}
	if len(names) == 0 {
		return esi.NewUniverseIdsPost(), nil
	}
	var result *esi.UniverseIdsPost
	retryErr := retry.Do(ctx, publicRetryBackoff(), func(ctx context.Context) error {
		c.limiter.wait(ctx)
		c.throttle.wait(ctx)
		response, httpResp, respErr := c.client.UniverseAPI.PostUniverseIds(ctx).RequestBody(names).Execute()
		defer closeResponseBody(httpResp)
		if httpResp != nil {
			c.throttle.update(httpResp)
		}
		if respErr != nil {
			wrapped := fmt.Errorf("esi universe ids lookup failed: %w", respErr)
			if shouldRetryPublicESI(httpResp, respErr) {
				return retry.RetryableError(wrapped)
			}
			return wrapped
		}
		result = response
		return nil
	})
	if retryErr != nil {
		return nil, retryErr
	}
	if result == nil {
		return esi.NewUniverseIdsPost(), nil
	}
	return result, nil
}

func (c *ESIPublicClient) ThrottleDelay() time.Duration {
	if c == nil || c.throttle == nil {
		return 0
	}
	return c.throttle.delay()
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
	retryErr := retry.Do(ctx, publicRetryBackoff(), func(ctx context.Context) error {
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

func wrapPublicNameError(idLabel string, idValue int, httpResp *http.Response, respErr error) error {
	err := fmt.Errorf("esi public name fetch failed (%s=%d): %w", idLabel, idValue, respErr)
	if shouldRetryPublicESI(httpResp, respErr) {
		return retry.RetryableError(err)
	}
	return err
}
