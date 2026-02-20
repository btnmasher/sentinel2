package esi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
)

const defaultESIProxyTimeout = 30 * time.Second

type ESIProxyClient struct {
	BaseURL string
	Client  *http.Client
	Logger  *logging.Logger
	limiter *esiRateLimiter
}

type CharactersResponse struct {
	Characters []int `json:"characters"`
}

func NewESIProxyClient(baseURL string, logger *logging.Logger) *ESIProxyClient {
	return &ESIProxyClient{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: defaultESIProxyTimeout},
		Logger:  logger,
		limiter: globalESILimiter,
	}
}

func (e *ESIProxyClient) Characters(ctx context.Context, user *core.Record, token string) ([]int, error) {
	sub := user.GetString("auth_provider_sub")
	if sub == "" {
		return nil, ErrMissingUserSub
	}
	var resp CharactersResponse
	if fetchErr := e.getJSON(ctx, "_/characters/"+sub+"/", token, &resp, sub); fetchErr != nil {
		return nil, fetchErr
	}
	return resp.Characters, nil
}

func (e *ESIProxyClient) CharacterLocation(ctx context.Context, character, token string) (CharacterLocation, error) {
	var resp CharacterLocation
	if fetchErr := e.getJSON(ctx, "v2/characters/"+character+"/location/", token, &resp, character); fetchErr != nil {
		return CharacterLocation{}, fetchErr
	}
	return resp, nil
}

func (e *ESIProxyClient) CharacterAffiliation(ctx context.Context, characterID int) (corporationID, allianceID int, err error) {
	_ = ctx
	_ = characterID
	return 0, 0, ErrAffiliationUnsupported
}

func (e *ESIProxyClient) SearchOrganizations(ctx context.Context, characterID int, accessToken, query string, strict bool) (corporationIDs, allianceIDs []int, err error) {
	_ = ctx
	_ = characterID
	_ = accessToken
	_ = query
	_ = strict
	return []int{}, []int{}, ErrAffiliationUnsupported
}

func (e *ESIProxyClient) SetAutopilotWaypoint(ctx context.Context, req AutopilotRequest, token string) error {
	start := time.Now()
	params := url.Values{}
	params.Set("add_to_beginning", strconv.FormatBool(req.AddToBeginning))
	params.Set("clear_other_waypoints", strconv.FormatBool(req.ClearOtherWaypoints))
	params.Set("destination_id", strconv.Itoa(req.DestinationID))
	requestURL := e.BaseURL + "v2/ui/autopilot/waypoint/?" + params.Encode()

	httpReq, reqErr := http.NewRequestWithContext(ctx, "POST", requestURL, http.NoBody)
	if reqErr != nil {
		e.logRequest("v2/ui/autopilot/waypoint", "POST", req.CharacterID, 0, start, reqErr)
		return reqErr
	}
	httpReq.Header.Set("Authorization", "JWT "+token)
	httpReq.Header.Set("X-Character-ID", req.CharacterID)

	e.limiter.wait(ctx)
	resp, respErr := e.Client.Do(httpReq)
	if respErr != nil {
		e.logRequest("v2/ui/autopilot/waypoint", "POST", req.CharacterID, 0, start, respErr)
		return respErr
	}
	defer func() { _ = resp.Body.Close() }()

	status := resp.StatusCode
	if resp.StatusCode >= http.StatusInternalServerError {
		payload, _ := io.ReadAll(resp.Body)
		if string(payload) == "Character token does not have required scopes" {
			e.logRequest("v2/ui/autopilot/waypoint", "POST", req.CharacterID, status, start, ErrScopeRequired)
			return ErrScopeRequired
		}
	}

	if resp.StatusCode >= http.StatusBadRequest {
		err := errors.New(resp.Status)
		e.logRequest("v2/ui/autopilot/waypoint", "POST", req.CharacterID, status, start, err)
		return err
	}
	e.logRequest("v2/ui/autopilot/waypoint", "POST", req.CharacterID, status, start, nil)
	return nil
}

func (e *ESIProxyClient) getJSON(ctx context.Context, path, token string, out any, characterID string) error {
	start := time.Now()
	req, reqErr := http.NewRequestWithContext(ctx, "GET", e.BaseURL+path, http.NoBody)
	if reqErr != nil {
		e.logRequest(path, "GET", characterID, 0, start, reqErr)
		return reqErr
	}
	req.Header.Set("Authorization", "JWT "+token)

	e.limiter.wait(ctx)
	resp, respErr := e.Client.Do(req)
	if respErr != nil {
		e.logRequest(path, "GET", characterID, 0, start, respErr)
		return respErr
	}
	defer func() { _ = resp.Body.Close() }()

	status := resp.StatusCode
	if resp.StatusCode >= http.StatusBadRequest {
		err := errors.New(resp.Status)
		e.logRequest(path, "GET", characterID, status, start, err)
		return err
	}

	payload, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		e.logRequest(path, "GET", characterID, status, start, readErr)
		return readErr
	}
	decodeErr := json.Unmarshal(payload, out)
	e.logRequest(path, "GET", characterID, status, start, decodeErr)
	return decodeErr
}

func (e *ESIProxyClient) logRequest(endpoint, method, characterID string, status int, start time.Time, err error) {
	if e == nil || e.Logger == nil {
		return
	}
	fields := logging.Fields{
		"backend":     "proxy",
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
