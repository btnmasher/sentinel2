package esi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"
	goesiesi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/pocketbase/pocketbase/core"
	retry "github.com/sethvargo/go-retry"

	"sentinel2/internal/logging"
)

type testAuthESIClient struct {
	BaseURL  string
	Logger   *logging.Logger
	limiter  *esiRateLimiter
	throttle *esiThrottle
	cache    *affiliationCache
	public   *goesiesi.APIClient
}

type testAuthProfileResponse struct {
	Characters []testAuthProfileCharacter `json:"characters,omitempty"`
}

type testAuthProfileCharacter struct {
	CharacterID   string `json:"characterId"`
	CharacterName string `json:"characterName"`
	IsPrimary     bool   `json:"isPrimary"`
	HasValidToken bool   `json:"hasValidToken"`
}

type testAuthNotification struct {
	ID        int64     `json:"notification_id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Text      string    `json:"text"`
}

// NewTestAuthESIClient creates an ESI client that proxies requests through TestAuth.
func NewTestAuthESIClient(baseURL, userAgent string, logger *logging.Logger) ESIClient {
	return &testAuthESIClient{
		BaseURL:  baseURL,
		Logger:   logger,
		limiter:  globalESILimiter,
		throttle: newESIThrottle(logger),
		cache:    newAffiliationCache(),
		public:   goesi.NewPublicESIClient(userAgent),
	}
}

func (e *testAuthESIClient) Characters(ctx context.Context, user *core.Record, accessToken string) ([]int, error) {
	_ = user
	start := time.Now()
	info, err := e.fetchProfile(ctx, accessToken)
	if err != nil {
		e.logRequest("oauth.api.me", "GET", "", 0, start, err)
		return nil, err
	}

	ids := make([]int, 0, len(info.Characters))
	for _, character := range info.Characters {
		id, parseErr := strconv.Atoi(strings.TrimSpace(character.CharacterID))
		if parseErr != nil {
			e.logRequest("oauth.api.me", "GET", "", 0, start, parseErr)
			return nil, parseErr
		}
		ids = append(ids, id)
	}

	e.logRequest("oauth.api.me", "GET", "", http.StatusOK, start, nil)
	return ids, nil
}

func (e *testAuthESIClient) CharacterLocation(ctx context.Context, characterID, accessToken string) (CharacterLocation, error) {
	start := time.Now()
	var resp CharacterLocation
	if err := e.proxyJSON(ctx, accessToken, characterID, "/latest/characters/"+characterID+"/location/", &resp); err != nil {
		e.logRequest("characters.location", "GET", characterID, 0, start, err)
		return CharacterLocation{}, err
	}
	e.logRequest("characters.location", "GET", characterID, http.StatusOK, start, nil)
	return resp, nil
}

func (e *testAuthESIClient) CharacterAffiliation(ctx context.Context, characterID int) (corporationID, allianceID int, err error) {
	if cached, ok := e.cache.get(characterID); ok {
		return cached.CorporationID, cached.AllianceID, nil
	}

	start := time.Now()
	operation := "characters.affiliation"
	charID := strconv.Itoa(characterID)
	corpID := 0
	allianceID = 0

	retryErr := retry.Do(ctx, publicRetryBackoff(), func(ctx context.Context) error {
		nextCorpID, nextAllianceID, retryable, attemptErr := e.characterAffiliationAttempt(ctx, characterID, operation, charID, start)
		if attemptErr != nil {
			if retryable {
				return retry.RetryableError(attemptErr)
			}
			return attemptErr
		}
		corpID = nextCorpID
		allianceID = nextAllianceID
		return nil
	})
	if retryErr != nil {
		return 0, 0, retryErr
	}
	return corpID, allianceID, nil
}

func (e *testAuthESIClient) CharacterNotifications(ctx context.Context, characterID int, accessToken, ifNoneMatch string) (notifications []CharacterNotification, etag string, notModified bool, err error) {
	start := time.Now()
	var headers http.Header
	if strings.TrimSpace(ifNoneMatch) != "" {
		headers = http.Header{
			"If-None-Match": []string{ifNoneMatch},
		}
	}
	resp, respErr := e.proxyResponse(ctx, accessToken, strconv.Itoa(characterID), http.MethodGet, "/latest/characters/"+strconv.Itoa(characterID)+"/notifications/", nil, headers)
	if respErr != nil {
		e.logRequest("characters.notifications", "GET", strconv.Itoa(characterID), 0, start, respErr)
		return []CharacterNotification{}, "", false, respErr
	}
	defer func() { _ = resp.Body.Close() }()

	etag = strings.TrimSpace(resp.Header.Get("ETag"))
	if resp.StatusCode == http.StatusNotModified {
		e.logRequest("characters.notifications", "GET", strconv.Itoa(characterID), http.StatusNotModified, start, nil)
		return []CharacterNotification{}, etag, true, nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		statusErr := errors.New(resp.Status)
		if len(body) > 0 {
			statusErr = fmt.Errorf("%s: %s", resp.Status, string(body))
		}
		e.logRequest("characters.notifications", "GET", strconv.Itoa(characterID), resp.StatusCode, start, statusErr)
		return []CharacterNotification{}, etag, false, statusErr
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		e.logRequest("characters.notifications", "GET", strconv.Itoa(characterID), resp.StatusCode, start, readErr)
		return []CharacterNotification{}, etag, false, readErr
	}

	if len(body) == 0 {
		e.logRequest("characters.notifications", "GET", strconv.Itoa(characterID), resp.StatusCode, start, nil)
		return []CharacterNotification{}, etag, false, nil
	}

	var payload []testAuthNotification
	if decodeErr := json.Unmarshal(body, &payload); decodeErr != nil {
		e.logRequest("characters.notifications", "GET", strconv.Itoa(characterID), resp.StatusCode, start, decodeErr)
		return []CharacterNotification{}, etag, false, decodeErr
	}

	notifications = make([]CharacterNotification, 0, len(payload))
	for _, entry := range payload {
		notifications = append(notifications, CharacterNotification(entry))
	}
	e.logRequest("characters.notifications", "GET", strconv.Itoa(characterID), resp.StatusCode, start, nil)
	return notifications, etag, false, nil
}

func (e *testAuthESIClient) SearchOrganizations(ctx context.Context, characterID int, accessToken, query string, strict bool, categories []string) (corporationIDs, allianceIDs []int, err error) {
	if len(categories) == 0 {
		categories = []string{"corporation", "alliance"}
	}
	values := url.Values{}
	for _, category := range categories {
		values.Add("categories", category)
	}
	values.Set("search", query)
	values.Set("strict", strconv.FormatBool(strict))
	requestPath := "/latest/characters/" + strconv.Itoa(characterID) + "/search/?" + values.Encode()

	var payload struct {
		Corporation []int `json:"corporation"`
		Alliance    []int `json:"alliance"`
	}
	start := time.Now()
	if err := e.proxyJSON(ctx, accessToken, strconv.Itoa(characterID), requestPath, &payload); err != nil {
		e.logRequest("characters.search", "GET", strconv.Itoa(characterID), 0, start, err)
		return []int{}, []int{}, err
	}
	e.logRequest("characters.search", "GET", strconv.Itoa(characterID), http.StatusOK, start, nil)
	return payload.Corporation, payload.Alliance, nil
}

func (e *testAuthESIClient) SearchStructures(ctx context.Context, characterID int, accessToken, query string, strict bool) ([]int64, error) {
	requestPath := "/latest/characters/" + strconv.Itoa(characterID) + "/search/?" + url.Values{
		"categories": []string{"structure"},
		"search":     []string{query},
		"strict":     []string{strconv.FormatBool(strict)},
	}.Encode()

	var payload struct {
		Structure []int64 `json:"structure"`
	}
	start := time.Now()
	if err := e.proxyJSON(ctx, accessToken, strconv.Itoa(characterID), requestPath, &payload); err != nil {
		e.logRequest("characters.search", "GET", strconv.Itoa(characterID), 0, start, err)
		return []int64{}, err
	}
	e.logRequest("characters.search", "GET", strconv.Itoa(characterID), http.StatusOK, start, nil)
	return payload.Structure, nil
}

func (e *testAuthESIClient) UniverseStructure(ctx context.Context, characterID int, accessToken string, structureID int64) (UniverseStructure, error) {
	var payload struct {
		Name          string `json:"name"`
		OwnerID       int    `json:"owner_id"`
		SolarSystemID int    `json:"solar_system_id"`
		TypeID        int    `json:"type_id"`
	}
	start := time.Now()
	if err := e.proxyJSON(ctx, accessToken, strconv.Itoa(characterID), "/latest/universe/structures/"+strconv.FormatInt(structureID, 10)+"/", &payload); err != nil {
		e.logRequest("universe.structures", "GET", strconv.Itoa(characterID), 0, start, err)
		return UniverseStructure{}, err
	}
	e.logRequest("universe.structures", "GET", strconv.Itoa(characterID), http.StatusOK, start, nil)
	return UniverseStructure{
		ID:       structureID,
		Name:     payload.Name,
		OwnerID:  payload.OwnerID,
		SystemID: payload.SolarSystemID,
		TypeID:   payload.TypeID,
	}, nil
}

func (e *testAuthESIClient) SetAutopilotWaypoint(ctx context.Context, req AutopilotRequest, accessToken string) error {
	start := time.Now()
	path := "/latest/ui/autopilot/waypoint/?" + url.Values{
		"add_to_beginning":      []string{strconv.FormatBool(req.AddToBeginning)},
		"clear_other_waypoints": []string{strconv.FormatBool(req.ClearOtherWaypoints)},
		"destination_id":        []string{strconv.Itoa(req.DestinationID)},
	}.Encode()

	resp, respErr := e.proxyResponse(ctx, accessToken, req.CharacterID, http.MethodPost, path, http.NoBody, nil)
	if respErr != nil {
		e.logRequest("ui.autopilot_waypoint", "POST", req.CharacterID, 0, start, respErr)
		return respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		statusErr := errors.New(resp.Status)
		if len(body) > 0 {
			statusErr = fmt.Errorf("%s: %s", resp.Status, string(body))
		}
		e.logRequest("ui.autopilot_waypoint", "POST", req.CharacterID, resp.StatusCode, start, statusErr)
		return statusErr
	}

	e.logRequest("ui.autopilot_waypoint", "POST", req.CharacterID, resp.StatusCode, start, nil)
	return nil
}

func (e *testAuthESIClient) fetchProfile(ctx context.Context, accessToken string) (*testAuthProfileResponse, error) {
	start := time.Now()
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(e.BaseURL, "/")+"/oauth/api/me", http.NoBody)
	if reqErr != nil {
		e.logRequest("oauth.api.me", "GET", "", 0, start, reqErr)
		return nil, reqErr
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: defaultESIProxyTimeout}
	resp, respErr := client.Do(req)
	if respErr != nil {
		e.logRequest("oauth.api.me", "GET", "", 0, start, respErr)
		return nil, respErr
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		e.logRequest("oauth.api.me", "GET", "", resp.StatusCode, start, readErr)
		return nil, readErr
	}
	if resp.StatusCode >= http.StatusBadRequest {
		statusErr := errors.New(resp.Status)
		if len(body) > 0 {
			statusErr = fmt.Errorf("%s: %s", resp.Status, string(body))
		}
		e.logRequest("oauth.api.me", "GET", "", resp.StatusCode, start, statusErr)
		return nil, statusErr
	}

	var payload testAuthProfileResponse
	if decodeErr := json.Unmarshal(body, &payload); decodeErr != nil {
		e.logRequest("oauth.api.me", "GET", "", resp.StatusCode, start, decodeErr)
		return nil, decodeErr
	}

	return &payload, nil
}

func (e *testAuthESIClient) proxyJSON(ctx context.Context, accessToken, characterID, path string, out any) error {
	resp, respErr := e.proxyResponse(ctx, accessToken, characterID, http.MethodGet, path, http.NoBody, nil)
	if respErr != nil {
		return respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(resp.Body)
		if len(bodyBytes) == 0 {
			return errors.New(resp.Status)
		}
		return fmt.Errorf("%s: %s", resp.Status, string(bodyBytes))
	}

	payload, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return readErr
	}
	if len(payload) == 0 || out == nil {
		return nil
	}
	return json.Unmarshal(payload, out)
}

func (e *testAuthESIClient) proxyResponse(ctx context.Context, accessToken, characterID, method, path string, body io.Reader, headers http.Header) (*http.Response, error) {
	return e.doRequest(ctx, accessToken, characterID, method, path, body, headers)
}

func (e *testAuthESIClient) characterAffiliationAttempt(
	ctx context.Context,
	characterID int,
	operation, charID string,
	start time.Time,
) (corpID, allianceID int, retryable bool, err error) {
	if e == nil || e.public == nil {
		err = fmt.Errorf("testauth esi client not configured")
		return 0, 0, false, err
	}
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

	if httpResp != nil && httpResp.StatusCode == http.StatusNotModified {
		cached, ok := e.cache.getAny(characterID)
		if !ok {
			err = fmt.Errorf("%w: cache miss on 304 response", ErrNotModified)
			e.logRequest(operation, "GET", charID, httpResp.StatusCode, start, err)
			return 0, 0, true, err
		}
		e.cache.refreshExpiry(characterID, httpResp)
		e.logRequest(operation, "GET", charID, httpResp.StatusCode, start, nil)
		return cached.CorporationID, cached.AllianceID, false, nil
	}

	if respErr != nil {
		e.logRequest(operation, "GET", charID, status, start, respErr)
		err = fmt.Errorf("character fetch failed (character_id=%d): %w", characterID, respErr)
		return 0, 0, shouldRetryPublicESI(httpResp, respErr), err
	}

	if httpResp == nil {
		err = fmt.Errorf("character fetch failed: missing response")
		e.logRequest(operation, "GET", charID, 0, start, err)
		return 0, 0, true, err
	}

	if httpResp.StatusCode >= http.StatusBadRequest {
		err = fmt.Errorf("character fetch failed: %s", httpResp.Status)
		e.logRequest(operation, "GET", charID, httpResp.StatusCode, start, err)
		retryable = httpResp.StatusCode == http.StatusTooManyRequests || httpResp.StatusCode >= http.StatusInternalServerError
		return 0, 0, retryable, err
	}

	if resp == nil {
		err = fmt.Errorf("character fetch failed: empty response")
		e.logRequest(operation, "GET", charID, httpResp.StatusCode, start, err)
		return 0, 0, true, err
	}

	corpID = int(resp.GetCorporationId())
	allianceID = int(resp.GetAllianceId())
	e.cache.set(characterID, corpID, allianceID, httpResp)
	e.logRequest(operation, "GET", charID, httpResp.StatusCode, start, nil)
	return corpID, allianceID, false, nil
}

func (e *testAuthESIClient) doRequest(ctx context.Context, accessToken, characterID, method, path string, body io.Reader, headers http.Header) (*http.Response, error) {
	if e == nil {
		return nil, fmt.Errorf("testauth esi client not configured")
	}
	if e != nil && e.limiter != nil {
		e.limiter.wait(ctx)
	}
	return DoTestAuthProxyRequest(ctx, e.BaseURL, accessToken, characterID, method, path, body, headers)
}

func (e *testAuthESIClient) logRequest(endpoint, method, characterID string, status int, start time.Time, err error) {
	if e == nil || e.Logger == nil {
		return
	}
	fields := logging.Fields{
		"backend":     "testauth",
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
		return
	}
	log.Info("esi request completed")
}
