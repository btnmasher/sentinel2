package maps

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/auth"
	"sentinel2/internal/config"
	"sentinel2/internal/esi"
	"sentinel2/internal/intel"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"

	"github.com/pocketbase/pocketbase/tools/router"
)

func NewMapHandler(app *pocketbase.PocketBase, cfg config.Config, esi esi.ESIClient, provider auth.Provider, eveProvider *auth.EVEProvider, planner *intel.RoutePlanner, topRoutes *intel.TopRoutesService) *MapHandler {
	return &MapHandler{App: app, Config: cfg, ESI: esi, Provider: provider, EVE: eveProvider, Routes: planner, TopRoutesSvc: topRoutes}
}

func (h *MapHandler) RegionsDotlan(c *core.RequestEvent) error { return h.regions(c, "dotlan") }
func (h *MapHandler) RegionsMetro(c *core.RequestEvent) error  { return h.regions(c, "metro") }
func (h *MapHandler) RegionsReal(c *core.RequestEvent) error   { return h.regions(c, "real") }
func (h *MapHandler) RegionsEve2D(c *core.RequestEvent) error  { return h.regions(c, "eve2d") }

func (h *MapHandler) regions(c *core.RequestEvent, mode string) error {
	regionIDs, parseErr := h.parseRegionIDs(c.Request.PathValue("regions"))
	if parseErr != nil {
		return router.NewBadRequestError("Invalid region list.", logging.Fields{
			"regions": c.Request.PathValue("regions"),
			"mode":    mode,
		})
	}

	systems, systemsErr := h.fetchSystems(regionIDs, mode)
	if systemsErr != nil {
		return router.NewInternalServerError("Failed to load systems.", logging.Fields{
			"region_ids": regionIDs,
			"mode":       mode,
		})
	}

	gates, gatesErr := h.fetchGates(regionIDs)
	if gatesErr != nil {
		return router.NewInternalServerError("Failed to load stargates.", logging.Fields{
			"region_ids": regionIDs,
		})
	}

	regions, regionsErr := h.fetchRegions(regionIDs, mode)
	if regionsErr != nil {
		return router.NewInternalServerError("Failed to load regions.", logging.Fields{
			"region_ids": regionIDs,
		})
	}
	normalizeSystemsByRegion(systems, regionIDs, 1000, 1000)
	normalizeRegions(regions)

	jumpbridges, jumpbridgesErr := h.fetchJumpbridges(regionIDs)
	if jumpbridgesErr != nil {
		return router.NewInternalServerError("Failed to load jumpbridges.", logging.Fields{
			"region_ids": regionIDs,
		})
	}

	return c.JSON(http.StatusOK, MapResponse{
		Regions:     regions,
		Systems:     systems,
		Gates:       gates,
		Jumpbridges: jumpbridges,
	})
}

func (h *MapHandler) Characters(c *core.RequestEvent) error {
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		return userErr
	}

	ids := []int{}
	if h.Config.AuthBackend == "eve" {
		records, recordsErr := h.App.FindRecordsByFilter(
			store.CollectionCharacters,
			"user = {:user}",
			"-is_main",
			0,
			0, dbx.Params{"user": user.Id},
		)
		if recordsErr != nil {
			return router.NewInternalServerError("Failed to fetch characters.", logging.Fields{
				"user_id": user.Id,
				"error":   recordsErr.Error(),
			})
		}
		for _, rec := range records {
			ids = append(ids, rec.GetInt("eve_character_id"))
		}
	} else {
		accessToken, tokenErr := h.userAccessToken(c.Request.Context(), user)
		if tokenErr != nil {
			return tokenErr
		}
		var charsErr error
		ids, charsErr = h.ESI.Characters(c.Request.Context(), user, accessToken)
		if charsErr != nil {
			return router.NewInternalServerError("Failed to fetch characters.", logging.Fields{
				"user_id": user.Id,
				"error":   charsErr.Error(),
			})
		}
	}

	return c.JSON(http.StatusOK, CharactersResponse{Characters: ids})
}

func (h *MapHandler) CharacterLocations(c *core.RequestEvent) error {
	log := logging.WithRequest(h.App, c)
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		return userErr
	}

	var payload LocationsRequest
	if err := c.BindBody(&payload); err != nil {
		return router.NewBadRequestError("Invalid locations payload.", logging.Fields{
			"error": err,
		})
	}
	if len(payload.Characters) == 0 {
		return router.NewBadRequestError("Missing characters.", logging.Fields{
			"user_id": user.Id,
		})
	}

	results := make([]LocationEntry, 0, len(payload.Characters))
	for _, characterID := range payload.Characters {
		character := fmt.Sprint(characterID)
		accessToken, tokenErr := h.resolveCharacterToken(c, user, character)
		if tokenErr != nil {
			log.WithErr(tokenErr).Warn("map locations token resolve failed")
			continue
		}
		location, locationErr := h.ESI.CharacterLocation(c.Request.Context(), character, accessToken)
		if locationErr != nil {
			log.WithFields(logging.Fields{
				"character_id": character,
			}).WithErr(locationErr).
				Error("map locations ESI failed")
			continue
		}
		systemID := int64(location.SolarSystemID)
		inSpace := location.StationID == 0 && location.StructureID == 0
		results = append(results, LocationEntry{
			CharacterID: characterID,
			Location:    systemID,
			SystemName:  "",
			InSpace:     inSpace,
		})
	}

	return c.JSON(http.StatusOK, LocationsResponse{Locations: results})
}

func (h *MapHandler) Route(c *core.RequestEvent) error {
	character := c.Request.PathValue("character")
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		return userErr
	}

	accessToken, tokenErr := h.resolveCharacterToken(c, user, character)
	if tokenErr != nil {
		return tokenErr
	}

	payload := struct {
		Waypoints []int `json:"waypoints"`
		Avoid     []int `json:"avoid"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Missing required data.", logging.Fields{
			"character_id": character,
			"error":        bindErr.Error(),
		})
	}

	if len(payload.Waypoints) == 0 {
		return router.NewBadRequestError("Missing required data.", logging.Fields{
			"waypoints": payload.Waypoints,
		})
	}

	if overlap(payload.Waypoints, payload.Avoid) {
		return router.NewBadRequestError("Route contains avoided system.", logging.Fields{
			"waypoints": payload.Waypoints,
			"avoid":     payload.Avoid,
		})
	}

	_ = h.TopRoutesSvc.Add(payload.Waypoints)

	location, locationErr := h.ESI.CharacterLocation(c.Request.Context(), character, accessToken)
	if locationErr != nil {
		return router.NewInternalServerError("Failed to fetch location.", logging.Fields{
			"character_id": character,
			"error":        locationErr.Error(),
		})
	}

	source := location.SolarSystemID
	fullRoute := []int{}
	fullJBRoute := []int{}
	for _, destination := range payload.Waypoints {
		route, jbRoute, routeErr := h.Routes.GenerateRoute(source, destination, payload.Avoid)
		if routeErr != nil {
			return router.NewBadRequestError("Cannot find route to system.", logging.Fields{
				"source":      source,
				"destination": destination,
				"error":       routeErr.Error(),
			})
		}
		fullRoute = append(fullRoute, route...)
		fullJBRoute = append(fullJBRoute, jbRoute...)
		source = destination
	}

	if len(fullJBRoute) == 0 {
		return c.JSON(http.StatusOK, RouteResponse{Route: fullRoute})
	}

	setErr := h.ESI.SetAutopilotWaypoint(c.Request.Context(), esi.AutopilotRequest{
		CharacterID:         character,
		DestinationID:       fullJBRoute[0],
		ClearOtherWaypoints: true,
		AddToBeginning:      false,
	}, accessToken)
	if errors.Is(setErr, esi.ErrScopeRequired) {
		return router.NewForbiddenError("Missing required ESI scopes.", logging.Fields{
			"character_id": character,
		})
	}
	if setErr != nil {
		return router.NewInternalServerError("Failed to set route.", logging.Fields{
			"character_id": character,
			"destination":  fullJBRoute[0],
			"error":        setErr.Error(),
		})
	}

	if len(fullJBRoute) > 1 {
		go func(points []int) {
			for _, point := range points {
				_ = h.ESI.SetAutopilotWaypoint(context.Background(), esi.AutopilotRequest{
					CharacterID:         character,
					DestinationID:       point,
					ClearOtherWaypoints: false,
					AddToBeginning:      false,
				}, accessToken)
			}
		}(fullJBRoute[1:])
	}

	return c.JSON(http.StatusOK, RouteResponse{Route: fullRoute})
}

func (h *MapHandler) ClearRoute(c *core.RequestEvent) error {
	character := c.Request.PathValue("character")
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		return userErr
	}

	accessToken, tokenErr := h.resolveCharacterToken(c, user, character)
	if tokenErr != nil {
		return tokenErr
	}

	location, locationErr := h.ESI.CharacterLocation(c.Request.Context(), character, accessToken)
	if locationErr != nil {
		return router.NewInternalServerError("Failed to fetch location.", logging.Fields{
			"character_id": character,
			"error":        locationErr.Error(),
		})
	}

	destination := location.SolarSystemID
	setErr := h.ESI.SetAutopilotWaypoint(c.Request.Context(), esi.AutopilotRequest{
		CharacterID:         character,
		DestinationID:       destination,
		ClearOtherWaypoints: true,
		AddToBeginning:      false,
	}, accessToken)
	if errors.Is(setErr, esi.ErrScopeRequired) {
		return router.NewForbiddenError("Missing required ESI scopes.", logging.Fields{
			"character_id": character,
		})
	}
	if setErr != nil {
		return router.NewInternalServerError("Failed to clear route.", logging.Fields{
			"character_id": character,
			"destination":  destination,
			"error":        setErr.Error(),
		})
	}

	return c.NoContent(http.StatusOK)
}

func (h *MapHandler) resolveCharacterToken(c *core.RequestEvent, user *core.Record, character string) (string, error) {
	if user == nil {
		return "", router.NewUnauthorizedError("Unauthorized", logging.Fields{
			"reason": "missing user context",
		})
	}
	if h.Config.AuthBackend != "eve" || h.EVE == nil {
		return h.userAccessToken(c.Request.Context(), user)
	}

	charID, parseErr := strconv.Atoi(character)
	if parseErr != nil {
		return "", router.NewBadRequestError("Invalid character.", logging.Fields{
			"character": character,
		})
	}

	records, recordsErr := h.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user} && eve_character_id = {:id}",
		"",
		1,
		0, dbx.Params{"user": user.Id, "id": charID},
	)
	if recordsErr != nil || len(records) == 0 {
		rawData := logging.Fields{
			"user_id":      user.Id,
			"character_id": charID,
		}
		if recordsErr != nil {
			rawData["error"] = recordsErr.Error()
		}
		return "", router.NewForbiddenError("Character not linked.", rawData)
	}
	charRecord := records[0]

	accessToken := charRecord.GetString("oauth_access_token")
	exp := charRecord.GetDateTime("oauth_access_expires_at")
	if accessToken == "" {
		return "", router.NewForbiddenError("Missing character token.", logging.Fields{
			"character_id": character,
		})
	}

	if exp.IsZero() || exp.Time().Before(time.Now().Add(15*time.Second)) {
		_, refreshErr := h.EVE.RefreshCharacter(c.Request.Context(), user, charRecord)
		if errors.Is(refreshErr, auth.ErrAccessDenied) {
			return "", router.NewForbiddenError("Access revoked.", logging.Fields{
				"character_id": character,
			})
		}
		if refreshErr != nil {
			return "", router.NewInternalServerError("Failed to refresh character token.", logging.Fields{
				"character_id": character,
				"error":        refreshErr.Error(),
			})
		}

		accessToken = charRecord.GetString("oauth_access_token")
	}

	if accessToken == "" {
		return "", router.NewForbiddenError("Missing character token.", logging.Fields{
			"character_id": character,
		})
	}

	return accessToken, nil
}

func (h *MapHandler) userAccessToken(ctx context.Context, user *core.Record) (string, error) {
	if user == nil {
		return "", router.NewUnauthorizedError("Unauthorized", logging.Fields{
			"reason": "missing user record",
		})
	}
	accessToken := user.GetString("oauth_access_token")
	exp := user.GetDateTime("oauth_access_expires_at")
	if accessToken == "" {
		return "", router.NewUnauthorizedError("Unauthorized", logging.Fields{
			"user_id": user.Id,
			"reason":  "missing oauth_access_token",
		})
	}
	if exp.IsZero() || exp.Time().Before(time.Now().Add(15*time.Second)) {
		if h.Provider == nil {
			return "", router.NewUnauthorizedError("Unauthorized", logging.Fields{
				"user_id": user.Id,
				"reason":  "provider unavailable",
			})
		}

		_, refreshErr := h.Provider.Refresh(ctx, user)
		if errors.Is(refreshErr, auth.ErrAccessDenied) {
			return "", router.NewForbiddenError("Access revoked.", logging.Fields{
				"user_id": user.Id,
			})
		}
		if refreshErr != nil {
			return "", router.NewInternalServerError("Failed to refresh token.", logging.Fields{
				"user_id": user.Id,
			})
		}

		accessToken = user.GetString("oauth_access_token")
	}
	if accessToken == "" {
		return "", router.NewUnauthorizedError("Unauthorized", logging.Fields{
			"user_id": user.Id,
			"reason":  "missing oauth_access_token after refresh",
		})
	}
	return accessToken, nil
}

func (h *MapHandler) Search(c *core.RequestEvent) error {
	query := c.Request.URL.Query().Get("q")
	systems := c.Request.URL.Query().Get("systems")
	if len(query) < 2 && systems == "" {
		return router.NewBadRequestError("Missing query.", logging.Fields{
			"query":   query,
			"systems": systems,
		})
	}

	var records []*core.Record
	var recordsErr error

	if systems != "" {
		ids, parseErr := h.parseSystemIDs(systems)
		if parseErr != nil {
			return router.NewBadRequestError("Invalid system list.", logging.Fields{
				"systems": systems,
			})
		}
		filter, params := buildFilter("eve_id", ids)
		records, recordsErr = h.App.FindRecordsByFilter(store.CollectionSolarSystems, filter, "name", 50, 0, params)
	} else {
		records, recordsErr = h.App.FindRecordsByFilter(store.CollectionSolarSystems, "name ~ {:q}", "name", 50, 0, dbx.Params{"q": "%" + query + "%"})
	}
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to search systems.", logging.Fields{
			"query":   query,
			"systems": systems,
		})
	}

	data := []SearchItem{}
	for _, rec := range records {
		data = append(data, SearchItem{
			ID:       rec.GetInt("eve_id"),
			Name:     rec.GetString("name"),
			RegionID: rec.GetInt("region_id"),
			Region:   rec.GetString("region_name"),
		})
	}

	return c.JSON(http.StatusOK, SearchResponse{Systems: data})
}

func (h *MapHandler) TopRoutes(c *core.RequestEvent) error {
	ids, idsErr := h.TopRoutesSvc.Top()
	if idsErr != nil {
		return router.NewInternalServerError("Failed to load routes.", logging.Fields{
			"error": idsErr.Error(),
			"step":  "top_routes_ids",
		})
	}

	if len(ids) == 0 {
		return c.JSON(http.StatusOK, TopRoutesResponse{Routes: []SearchItem{}})
	}

	filter, params := buildFilter("eve_id", ids)

	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionSolarSystems, filter, "name", 0, 0, params)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to load routes.", logging.Fields{
			"error": recordsErr.Error(),
			"step":  "top_routes_records",
		})
	}

	routes := []SearchItem{}
	for _, rec := range records {
		routes = append(routes, SearchItem{
			ID:   rec.GetInt("eve_id"),
			Name: rec.GetString("name"),
		})
	}

	return c.JSON(http.StatusOK, TopRoutesResponse{Routes: routes})
}
