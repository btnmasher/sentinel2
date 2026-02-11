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

	"sentinel2/internal/logging"
)

type ESIDirectClient struct {
	UserAgent string
	Timeout   time.Duration
	throttle  *esiThrottle
	limiter   *esiRateLimiter
	cache     *affiliationCache
	public    *esi.APIClient
	Logger    *logging.Logger
}

func NewESIDirectClient(userAgent string, logger *logging.Logger) *ESIDirectClient {
	return &ESIDirectClient{
		UserAgent: userAgent,
		Timeout:   30 * time.Second,
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

func (e *ESIDirectClient) CharacterLocation(ctx context.Context, characterID string, token string) (CharacterLocation, error) {
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
		StructureID:   int64(resp.GetStructureId()),
	}, nil
}

func (e *ESIDirectClient) CharacterAffiliation(ctx context.Context, characterID int) (int, int, error) {
	if cached, ok := e.cache.get(characterID); ok {
		return cached.CorporationID, cached.AllianceID, nil
	}

	start := time.Now()
	e.limiter.wait(ctx)
	e.throttle.wait(ctx)
	request := e.public.CharacterAPI.GetCharactersCharacterId(ctx, int64(characterID))
	if etag, ok := e.cache.etag(characterID); ok {
		request = request.IfNoneMatch(etag)
	}
	resp, httpResp, respErr := request.Execute()
	if respErr != nil {
		e.logRequest("characters.affiliation", "GET", strconv.Itoa(characterID), httpStatus(httpResp), start, respErr)
		return 0, 0, respErr
	}

	if httpResp != nil {
		e.throttle.update(httpResp)
	} else {
		e.logRequest("characters.affiliation", "GET", strconv.Itoa(characterID), 0, start, fmt.Errorf("character fetch failed: missing response"))
		return 0, 0, fmt.Errorf("character fetch failed: missing response")
	}

	if httpResp.StatusCode == http.StatusNotModified {
		if cached, ok := e.cache.getAny(characterID); ok {
			e.cache.refreshExpiry(characterID, httpResp)
			e.logRequest("characters.affiliation", "GET", strconv.Itoa(characterID), httpResp.StatusCode, start, nil)
			return cached.CorporationID, cached.AllianceID, nil
		}
	}
	if httpResp.StatusCode >= 400 {
		err := fmt.Errorf("character fetch failed: %s", httpResp.Status)
		e.logRequest("characters.affiliation", "GET", strconv.Itoa(characterID), httpResp.StatusCode, start, err)
		return 0, 0, err
	}

	corpID := int(resp.GetCorporationId())
	allianceID := int(resp.GetAllianceId())
	e.cache.set(characterID, corpID, allianceID, httpResp)
	e.logRequest("characters.affiliation", "GET", strconv.Itoa(characterID), httpResp.StatusCode, start, nil)
	return corpID, allianceID, nil
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

func (e *ESIDirectClient) logRequest(endpoint string, method string, characterID string, status int, start time.Time, err error) {
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

func httpStatus(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

type authTransport struct {
	Base  http.RoundTripper
	Token string
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
