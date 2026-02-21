package esi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"
	esi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/pocketbase/pocketbase/core"
	retry "github.com/sethvargo/go-retry"

	"sentinel2/internal/logging"
	sharedcollections "sentinel2/internal/shared/collections"
)

const defaultESIDirectTimeout = 30 * time.Second

type ESIDirectClient struct {
	UserAgent string
	Timeout   time.Duration
	throttle  *esiThrottle
	limiter   *esiRateLimiter
	cache     *affiliationCache
	public    *esi.APIClient
	Logger    *logging.Logger
}

type authTransport struct {
	Base  http.RoundTripper
	Token string
}

func NewESIDirectClient(userAgent string, logger *logging.Logger) *ESIDirectClient {
	return &ESIDirectClient{
		UserAgent: userAgent,
		Timeout:   defaultESIDirectTimeout,
		throttle:  newESIThrottle(logger),
		limiter:   globalESILimiter,
		cache:     newAffiliationCache(),
		public:    goesi.NewPublicESIClient(userAgent),
		Logger:    logger,
	}
}

func (e *ESIDirectClient) Characters(ctx context.Context, user *core.Record, token string) ([]int, error) {
	if user == nil {
		return nil, ErrMissingUser
	}
	id := user.GetInt("eve_character_id")
	if id == 0 {
		return nil, ErrMissingCharacter
	}
	return []int{id}, nil
}

func (e *ESIDirectClient) CharacterLocation(ctx context.Context, characterID, token string) (CharacterLocation, error) {
	start := time.Now()
	client := e.authenticatedClient(token)
	charID, charIDErr := strconv.Atoi(characterID)
	if charIDErr != nil {
		e.logRequest("characters.location", "GET", characterID, 0, start, charIDErr)
		return CharacterLocation{}, charIDErr
	}

	e.limiter.wait(ctx)
	e.throttle.wait(ctx)
	resp, httpResp, respErr := client.LocationAPI.GetCharactersCharacterIdLocation(ctx, int64(charID)).Execute()
	if httpResp != nil && httpResp.Body != nil {
		defer func() { _ = httpResp.Body.Close() }()
	}
	if respErr != nil {
		status := httpStatus(httpResp)
		e.logRequest("characters.location", "GET", characterID, status, start, respErr)
		return CharacterLocation{}, respErr
	}
	if httpResp != nil {
		e.throttle.update(httpResp)
	}

	e.logRequest("characters.location", "GET", characterID, httpStatus(httpResp), start, nil)
	return CharacterLocation{
		SolarSystemID: int(resp.GetSolarSystemId()),
		StationID:     int(resp.GetStationId()),
		StructureID:   resp.GetStructureId(),
	}, nil
}

func (e *ESIDirectClient) CharacterAffiliation(ctx context.Context, characterID int) (corporationID, allianceID int, err error) {
	if cached, ok := e.cache.get(characterID); ok {
		return cached.CorporationID, cached.AllianceID, nil
	}

	start := time.Now()
	operation := "characters.affiliation"
	charID := strconv.Itoa(characterID)
	corpID := 0
	allianceID = 0

	retryErr := retry.Do(ctx, publicRetryBackoff(), func(ctx context.Context) error {
		e.limiter.wait(ctx)
		e.throttle.wait(ctx)

		request := e.public.CharacterAPI.GetCharactersCharacterId(ctx, int64(characterID))
		if etag, ok := e.cache.etag(characterID); ok {
			request = request.IfNoneMatch(etag)
		}
		resp, httpResp, respErr := request.Execute()
		defer closeResponseBody(httpResp)

		status := httpStatus(httpResp)
		if httpResp != nil {
			e.throttle.update(httpResp)
		}

		if respErr != nil {
			e.logRequest(operation, "GET", charID, status, start, respErr)
			err := fmt.Errorf("character fetch failed (character_id=%d): %w", characterID, respErr)
			if shouldRetryPublicESI(httpResp, respErr) {
				return retry.RetryableError(err)
			}
			return err
		}
		if httpResp == nil {
			err := fmt.Errorf("character fetch failed: missing response")
			e.logRequest(operation, "GET", charID, 0, start, err)
			return retry.RetryableError(err)
		}

		if httpResp.StatusCode == http.StatusNotModified {
			if cached, ok := e.cache.getAny(characterID); ok {
				e.cache.refreshExpiry(characterID, httpResp)
				corpID = cached.CorporationID
				allianceID = cached.AllianceID
				e.logRequest(operation, "GET", charID, httpResp.StatusCode, start, nil)
				return nil
			}
			err := fmt.Errorf("character fetch failed: cache miss on 304 response")
			e.logRequest(operation, "GET", charID, httpResp.StatusCode, start, err)
			return retry.RetryableError(err)
		}
		if httpResp.StatusCode >= http.StatusBadRequest {
			err := fmt.Errorf("character fetch failed: %s", httpResp.Status)
			e.logRequest(operation, "GET", charID, httpResp.StatusCode, start, err)
			if httpResp.StatusCode == http.StatusTooManyRequests || httpResp.StatusCode >= http.StatusInternalServerError {
				return retry.RetryableError(err)
			}
			return err
		}
		if resp == nil {
			err := fmt.Errorf("character fetch failed: empty response")
			e.logRequest(operation, "GET", charID, httpResp.StatusCode, start, err)
			return retry.RetryableError(err)
		}

		corpID = int(resp.GetCorporationId())
		allianceID = int(resp.GetAllianceId())
		e.cache.set(characterID, corpID, allianceID, httpResp)
		e.logRequest(operation, "GET", charID, httpResp.StatusCode, start, nil)
		return nil
	})
	if retryErr != nil {
		return 0, 0, retryErr
	}
	return corpID, allianceID, nil
}

func (e *ESIDirectClient) SearchOrganizations(ctx context.Context, characterID int, accessToken, query string, strict bool) (corporationIDs, allianceIDs []int, err error) {
	if e == nil {
		return []int{}, []int{}, fmt.Errorf("esi direct client not configured")
	}
	q, token, ok := normalizeCharacterSearchInput(characterID, accessToken, query)
	if !ok {
		return []int{}, []int{}, nil
	}

	client := e.authenticatedClient(token)
	var payload *esi.CharactersCharacterIdSearchGet
	retryErr := retry.Do(ctx, publicRetryBackoff(), func(ctx context.Context) error {
		e.limiter.wait(ctx)
		e.throttle.wait(ctx)
		response, httpResp, respErr := client.SearchAPI.
			GetCharactersCharacterIdSearch(ctx, int64(characterID)).
			Categories([]string{"corporation", "alliance"}).
			Search(q).
			Strict(strict).
			Execute()
		defer closeResponseBody(httpResp)
		if httpResp != nil {
			e.throttle.update(httpResp)
		}
		if respErr != nil {
			wrapped := fmt.Errorf("esi character search failed (character_id=%d): %w", characterID, respErr)
			if shouldRetryPublicESI(httpResp, respErr) {
				return retry.RetryableError(wrapped)
			}
			return wrapped
		}
		if response == nil {
			return fmt.Errorf("esi character search response empty (character_id=%d)", characterID)
		}
		payload = response
		return nil
	})
	if retryErr != nil {
		return []int{}, []int{}, retryErr
	}
	return sharedcollections.ToIntSlice(payload.GetCorporation()), sharedcollections.ToIntSlice(payload.GetAlliance()), nil
}

func (e *ESIDirectClient) SetAutopilotWaypoint(ctx context.Context, req AutopilotRequest, token string) error {
	start := time.Now()
	client := e.authenticatedClient(token)

	request := client.UserInterfaceAPI.PostUiAutopilotWaypoint(ctx)
	request = request.DestinationId(int64(req.DestinationID))
	request = request.ClearOtherWaypoints(req.ClearOtherWaypoints)
	request = request.AddToBeginning(req.AddToBeginning)

	e.limiter.wait(ctx)
	e.throttle.wait(ctx)
	httpResp, execErr := request.Execute()
	if httpResp != nil && httpResp.Body != nil {
		defer func() { _ = httpResp.Body.Close() }()
	}
	if httpResp != nil {
		e.throttle.update(httpResp)
	}
	e.logRequest("ui.autopilot_waypoint", "POST", req.CharacterID, httpStatus(httpResp), start, execErr)
	return execErr
}

func (e *ESIDirectClient) ThrottleDelay() time.Duration {
	if e == nil || e.throttle == nil {
		return 0
	}
	return e.throttle.delay()
}

func (e *ESIDirectClient) authenticatedClient(accessToken string) *esi.APIClient {
	httpClient := &http.Client{
		Timeout: e.Timeout,
		Transport: &authTransport{
			Base:  http.DefaultTransport,
			Token: accessToken,
		},
	}
	options := goesi.ClientOptions{
		UserAgent: e.UserAgent,
	}
	return goesi.NewESIClientWithOptions(httpClient, options)
}

func (e *ESIDirectClient) logRequest(endpoint, method, characterID string, status int, start time.Time, err error) {
	if e == nil || e.Logger == nil {
		return
	}
	fields := logging.Fields{
		"backend":     "direct",
		"endpoint":    endpoint,
		"method":      method,
		"status":      status,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	if characterID != "" {
		fields["character_id"] = characterID
	}
	log := e.Logger.WithFields(fields)
	if err != nil {
		log.WithErr(err).Error("esi request failed")
	} else {
		log.Info("esi request completed")
	}
}

func (a *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	if clone.Header.Get("Authorization") == "" {
		clone.Header.Set("Authorization", "Bearer "+a.Token)
	}
	return a.base().RoundTrip(clone)
}

func (a *authTransport) base() http.RoundTripper {
	if a.Base == nil {
		return http.DefaultTransport
	}
	return a.Base
}

func httpStatus(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}
