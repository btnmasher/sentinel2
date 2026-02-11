package maps

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/auth"
	"sentinel2/internal/config"
	"sentinel2/internal/esi"
	"sentinel2/internal/format"
	"sentinel2/internal/intel"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"

	"github.com/pocketbase/pocketbase/tools/router"
)

type MapHandler struct {
	App          *pocketbase.PocketBase
	Config       config.Config
	ESI          esi.ESIClient
	Provider     auth.Provider
	EVE          *auth.EVEProvider
	Routes       *intel.RoutePlanner
	TopRoutesSvc *intel.TopRoutesService
}

type RegionDTO struct {
	Region   int    `json:"region"`
	Name     string `json:"name"`
	Position struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"position"`
}

type SystemDTO struct {
	Name           string  `json:"name"`
	SecurityStatus float64 `json:"security_status"`
	Region         int     `json:"region"`
	Constellation  int     `json:"constellation"`
	System         int     `json:"system"`
	Position       struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"position"`
	Absolute struct {
		X int `json:"x"`
		Y int `json:"y"`
		Z int `json:"z"`
	} `json:"absolute"`
}

type GateDTO struct {
	To          int    `json:"to"`
	From        int    `json:"from"`
	Type        string `json:"type"`
	ToRegion    int    `json:"to_region"`
	FromRegion  int    `json:"from_region"`
	ToDotlanX   int    `json:"to_dotlan_x"`
	ToDotlanY   int    `json:"to_dotlan_y"`
	ToMetroX    int    `json:"to_metro_x"`
	ToMetroY    int    `json:"to_metro_y"`
	FromDotlanX int    `json:"from_dotlan_x"`
	FromDotlanY int    `json:"from_dotlan_y"`
	FromMetroX  int    `json:"from_metro_x"`
	FromMetroY  int    `json:"from_metro_y"`
}

type JumpbridgeDTO struct {
	From       int  `json:"from"`
	To         int  `json:"to"`
	FromRegion int  `json:"from_region"`
	ToRegion   int  `json:"to_region"`
	Friendly   bool `json:"friendly"`
}

type MapResponse struct {
	Regions     map[int]RegionDTO `json:"regions"`
	Systems     map[int]SystemDTO `json:"systems"`
	Gates       []GateDTO         `json:"gates"`
	Jumpbridges []JumpbridgeDTO   `json:"jumpbridges"`
}

type CharactersResponse struct {
	Characters []int `json:"characters"`
}

type RouteResponse struct {
	Route []int `json:"route"`
}

type SearchItem struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	RegionID int    `json:"region_id"`
	Region   string `json:"region"`
}

type SearchResponse struct {
	Systems []SearchItem `json:"systems"`
}

type TopRoutesResponse struct {
	Routes []SearchItem `json:"routes"`
}

type LocationEntry struct {
	CharacterID int64  `json:"character_id"`
	Location    int64  `json:"location"`
	SystemName  string `json:"system_name"`
	InSpace     bool   `json:"in_space"`
}

type LocationsRequest struct {
	Characters []int64 `json:"characters"`
}

type LocationsResponse struct {
	Locations []LocationEntry `json:"locations"`
}

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
	log := logging.WithRequest(h.App, c)
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		log.
			WithErr(userErr).
			Warn("map characters unauthorized")
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
			log.
				WithErr(recordsErr).
				Error("map characters query failed")
			return router.NewInternalServerError("Failed to fetch characters.", logging.Fields{
				"user_id": user.Id,
			})
		}
		for _, rec := range records {
			ids = append(ids, rec.GetInt("eve_character_id"))
		}
	} else {
		accessToken, tokenErr := h.userAccessToken(c.Request.Context(), user)
		if tokenErr != nil {
			log.
				WithErr(tokenErr).
				Warn("map characters access token failed")
			return tokenErr
		}
		var charsErr error
		ids, charsErr = h.ESI.Characters(c.Request.Context(), user, accessToken)
		if charsErr != nil {
			log.
				WithErr(charsErr).
				Error("map characters ESI failed")
			return router.NewInternalServerError("Failed to fetch characters.", logging.Fields{
				"user_id": user.Id,
			})
		}
	}

	return c.JSON(http.StatusOK, CharactersResponse{Characters: ids})
}

func (h *MapHandler) CharacterLocations(c *core.RequestEvent) error {
	log := logging.WithRequest(h.App, c)
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		log.WithErr(userErr).Warn("map locations unauthorized")
		return userErr
	}

	var payload LocationsRequest
	if err := c.BindBody(&payload); err != nil {
		return router.NewBadRequestError("Invalid locations payload.", logging.Fields{
			"error": err,
		})
	}
	if len(payload.Characters) == 0 {
		return router.NewBadRequestError("Missing characters.", nil)
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
	log := logging.WithRequest(h.App, c)
	character := c.Request.PathValue("character")
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		log.
			WithErr(userErr).
			Warn("map route unauthorized")
		return userErr
	}

	accessToken, tokenErr := h.resolveCharacterToken(c, user, character)
	if tokenErr != nil {
		log.
			WithErr(tokenErr).
			Warn("map route token resolve failed")
		return tokenErr
	}

	payload := struct {
		Waypoints []int `json:"waypoints"`
		Avoid     []int `json:"avoid"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Missing required data.", nil)
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
		log.WithFields(logging.Fields{
			"character_id": character,
		}).WithErr(locationErr).
			Error("map route location failed")
		return router.NewInternalServerError("Failed to fetch location.", logging.Fields{
			"character_id": character,
		})
	}

	source := location.SolarSystemID
	fullRoute := []int{}
	fullJBRoute := []int{}
	for _, destination := range payload.Waypoints {
		route, jbRoute, routeErr := h.Routes.GenerateRoute(source, destination, payload.Avoid)
		if routeErr != nil {
			log.WithFields(logging.Fields{
				"source":      source,
				"destination": destination,
			}).WithErr(routeErr).
				Warn("map route generation failed")
			return router.NewBadRequestError("Cannot find route to system.", logging.Fields{
				"source":      source,
				"destination": destination,
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
		log.WithFields(logging.Fields{
			"character_id": character,
		}).Warn("map route missing scopes")
		return router.NewForbiddenError("New scopes required.", logging.Fields{
			"character_id": character,
		})
	}
	if setErr != nil {
		log.WithFields(logging.Fields{
			"character_id": character,
		}).WithErr(setErr).
			Error("map route set waypoint failed")
		return router.NewInternalServerError("Failed to set route.", logging.Fields{
			"character_id": character,
			"destination":  fullJBRoute[0],
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
	log := logging.WithRequest(h.App, c)
	character := c.Request.PathValue("character")
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		log.
			WithErr(userErr).
			Warn("map clear route unauthorized")
		return userErr
	}

	accessToken, tokenErr := h.resolveCharacterToken(c, user, character)
	if tokenErr != nil {
		log.
			WithErr(tokenErr).
			Warn("map clear route token resolve failed")
		return tokenErr
	}

	location, locationErr := h.ESI.CharacterLocation(c.Request.Context(), character, accessToken)
	if locationErr != nil {
		log.WithFields(logging.Fields{
			"character_id": character,
		}).WithErr(locationErr).
			Error("map clear route location failed")
		return router.NewInternalServerError("Failed to fetch location.", logging.Fields{
			"character_id": character,
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
		log.WithFields(logging.Fields{
			"character_id": character,
		}).Warn("map clear route missing scopes")
		return router.NewForbiddenError("New scopes required.", logging.Fields{
			"character_id": character,
		})
	}
	if setErr != nil {
		log.WithFields(logging.Fields{
			"character_id": character,
		}).WithErr(setErr).
			Error("map clear route failed")
		return router.NewInternalServerError("Failed to clear route.", logging.Fields{
			"character_id": character,
			"destination":  destination,
		})
	}

	return c.NoContent(http.StatusOK)
}

func (h *MapHandler) resolveCharacterToken(c *core.RequestEvent, user *core.Record, character string) (string, error) {
	log := logging.WithRequest(h.App, c)
	if user == nil {
		return "", router.NewUnauthorizedError("Unauthorized", nil)
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
		if recordsErr != nil {
			log.
				WithErr(recordsErr).
				Warn("map resolve character token query failed")
		}
		return "", router.NewForbiddenError("Character not linked.", logging.Fields{
			"user_id":      user.Id,
			"character_id": charID,
		})
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
			log.WithFields(logging.Fields{
				"character_id": character,
			}).Warn("map character refresh access denied")
			return "", router.NewForbiddenError("Access revoked.", logging.Fields{
				"character_id": character,
			})
		}
		if refreshErr != nil {
			log.WithFields(logging.Fields{
				"character_id": character,
			}).WithErr(refreshErr).
				Error("map character refresh failed")
			return "", router.NewInternalServerError("Failed to refresh character token.", logging.Fields{
				"character_id": character,
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
		return "", router.NewUnauthorizedError("Unauthorized", nil)
	}
	accessToken := user.GetString("oauth_access_token")
	exp := user.GetDateTime("oauth_access_expires_at")
	if accessToken == "" {
		return "", router.NewUnauthorizedError("Unauthorized", nil)
	}
	if exp.IsZero() || exp.Time().Before(time.Now().Add(15*time.Second)) {
		if h.Provider == nil {
			return "", router.NewUnauthorizedError("Unauthorized", nil)
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
		return "", router.NewUnauthorizedError("Unauthorized", nil)
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
		filter, params := buildIntFilter("eve_id", ids)
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
		return router.NewInternalServerError("Failed to load routes.", nil)
	}

	if len(ids) == 0 {
		return c.JSON(http.StatusOK, TopRoutesResponse{Routes: []SearchItem{}})
	}

	filter, params := buildIntFilter("eve_id", ids)

	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionSolarSystems, filter, "name", 0, 0, params)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to load routes.", nil)
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

func (h *MapHandler) fetchRegions(regionIDs []int, mode string) (map[int]RegionDTO, error) {
	filter, params := buildIntFilter("eve_id", regionIDs)
	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionRegions, filter, "name", 0, 0, params)
	if recordsErr != nil {
		return nil, recordsErr
	}

	out := map[int]RegionDTO{}
	for _, rec := range records {
		id := int(rec.GetInt("eve_id"))
		dto := RegionDTO{
			Region: id,
			Name:   rec.GetString("name"),
		}
		switch mode {
		case "eve2d":
			dto.Position.X = rec.GetInt("eve2d_x")
			dto.Position.Y = rec.GetInt("eve2d_y")
			if dto.Position.X == 0 && dto.Position.Y == 0 {
				dto.Position.X = rec.GetInt("metro_x")
				dto.Position.Y = rec.GetInt("metro_y")
			}
		case "real":
			dto.Position.X = rec.GetInt("real_x")
			dto.Position.Y = rec.GetInt("real_y")
		case "dotlan":
			dto.Position.X = rec.GetInt("eve2d_x")
			dto.Position.Y = rec.GetInt("eve2d_y")
			if dto.Position.X == 0 && dto.Position.Y == 0 {
				dto.Position.X = rec.GetInt("metro_x")
				dto.Position.Y = rec.GetInt("metro_y")
			}
		case "metro":
			dto.Position.X = rec.GetInt("metro_x")
			dto.Position.Y = rec.GetInt("metro_y")
		default:
			dto.Position.X = rec.GetInt("metro_x")
			dto.Position.Y = rec.GetInt("metro_y")
		}
		out[id] = dto
	}
	return out, nil
}

func normalizeSystemsByRegion(systems map[int]SystemDTO, regionIDs []int, tx int, ty int) {
	regionSet := map[int]struct{}{}
	for _, id := range regionIDs {
		regionSet[id] = struct{}{}
	}

	type bounds struct {
		minX int
		minY int
		maxX int
		maxY int
	}
	regionBounds := map[int]*bounds{}

	for _, system := range systems {
		if _, ok := regionSet[int(system.Region)]; !ok {
			continue
		}
		b, exists := regionBounds[int(system.Region)]
		if !exists {
			regionBounds[int(system.Region)] = &bounds{
				minX: system.Position.X,
				minY: system.Position.Y,
				maxX: system.Position.X,
				maxY: system.Position.Y,
			}
			continue
		}
		if system.Position.X < b.minX {
			b.minX = system.Position.X
		}
		if system.Position.Y < b.minY {
			b.minY = system.Position.Y
		}
		if system.Position.X > b.maxX {
			b.maxX = system.Position.X
		}
		if system.Position.Y > b.maxY {
			b.maxY = system.Position.Y
		}
	}

	for id, b := range regionBounds {
		dx := b.maxX - b.minX
		dy := b.maxY - b.minY
		if dx == 0 || dy == 0 {
			continue
		}
		scale := float64(tx) / float64(dx)
		if yScale := float64(ty) / float64(dy); yScale < scale {
			scale = yScale
		}
		for systemID, system := range systems {
			if int(system.Region) != id {
				continue
			}
			system.Position.X = int(scale * float64(system.Position.X-b.minX))
			system.Position.Y = int(scale * float64(system.Position.Y-b.minY))
			systems[systemID] = system
		}
	}
}

func normalizeRegions(regions map[int]RegionDTO) {
	if len(regions) == 0 {
		return
	}
	minX := 0
	minY := 0
	first := true
	for _, region := range regions {
		if first {
			minX = region.Position.X
			minY = region.Position.Y
			first = false
			continue
		}
		if region.Position.X < minX {
			minX = region.Position.X
		}
		if region.Position.Y < minY {
			minY = region.Position.Y
		}
	}
	for id, region := range regions {
		region.Position.X -= minX
		region.Position.Y -= minY
		regions[id] = region
	}
}

func (h *MapHandler) fetchSystems(regionIDs []int, mode string) (map[int]SystemDTO, error) {
	filter, params := buildIntFilter("region_id", regionIDs)
	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionSolarSystems, filter, "", 0, 0, params)
	if recordsErr != nil {
		return nil, recordsErr
	}

	out := map[int]SystemDTO{}
	for _, rec := range records {
		id := int(rec.GetInt("eve_id"))
		dto := SystemDTO{
			Name:           rec.GetString("name"),
			SecurityStatus: rec.GetFloat("security_status"),
			Region:         rec.GetInt("region_id"),
			Constellation:  rec.GetInt("constellation"),
			System:         id,
		}
		if mode == "real" {
			dto.Position.X = rec.GetInt("real_x")
			dto.Position.Y = rec.GetInt("real_y")
			if dto.Position.X == 0 && dto.Position.Y == 0 {
				dto.Position.X = rec.GetInt("raw_x")
				dto.Position.Y = -rec.GetInt("raw_z")
			}
		} else if mode == "eve2d" {
			dto.Position.X = int(math.Round(rec.GetFloat("eve2d_x")))
			dto.Position.Y = -int(math.Round(rec.GetFloat("eve2d_y")))
		} else if mode == "metro" {
			dto.Position.X = rec.GetInt("metro_x")
			dto.Position.Y = rec.GetInt("metro_y")
		} else {
			dto.Position.X = rec.GetInt("dotlan_x")
			dto.Position.Y = rec.GetInt("dotlan_y")
		}
		dto.Absolute.X = rec.GetInt("raw_x")
		dto.Absolute.Y = rec.GetInt("raw_y")
		dto.Absolute.Z = rec.GetInt("raw_z")
		out[id] = dto
	}
	return out, nil
}

func (h *MapHandler) fetchGates(regionIDs []int) ([]GateDTO, error) {
	filter, params := buildIntFilter("from_region", regionIDs)
	filterTo, paramsTo := buildIntFilter("to_region", regionIDs)
	for k, v := range paramsTo {
		params[k] = v
	}

	combinedFilter := "(" + filter + ") || (" + filterTo + ")"

	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionGates, combinedFilter, "", 0, 0, params)
	if recordsErr != nil {
		return nil, recordsErr
	}

	out := []GateDTO{}
	for _, rec := range records {
		fromRegion := rec.GetInt("from_region")
		toRegion := rec.GetInt("to_region")
		gateType := "solarsystem"
		if fromRegion != toRegion {
			gateType = "region"
		} else if rec.GetInt("from_constellation") != rec.GetInt("to_constellation") {
			gateType = "constellation"
		}

		out = append(out, GateDTO{
			To:          rec.GetInt("to_solarsystem"),
			From:        rec.GetInt("from_solarsystem"),
			Type:        gateType,
			ToRegion:    toRegion,
			FromRegion:  fromRegion,
			ToDotlanX:   rec.GetInt("to_dotlan_x"),
			ToDotlanY:   rec.GetInt("to_dotlan_y"),
			ToMetroX:    rec.GetInt("to_metro_x"),
			ToMetroY:    rec.GetInt("to_metro_y"),
			FromDotlanX: rec.GetInt("from_dotlan_x"),
			FromDotlanY: rec.GetInt("from_dotlan_y"),
			FromMetroX:  rec.GetInt("from_metro_x"),
			FromMetroY:  rec.GetInt("from_metro_y"),
		})
	}

	return out, nil
}

func (h *MapHandler) fetchJumpbridges(regionIDs []int) ([]JumpbridgeDTO, error) {
	filter, params := buildIntFilter("from_region", regionIDs)
	filterTo, paramsTo := buildIntFilter("to_region", regionIDs)
	for k, v := range paramsTo {
		params[k] = v
	}

	combinedFilter := "(" + filter + ") || (" + filterTo + ")"

	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionJumpbridges, combinedFilter, "", 0, 0, params)
	if recordsErr != nil {
		return nil, recordsErr
	}

	out := []JumpbridgeDTO{}
	seen := map[string]struct{}{}
	for _, rec := range records {
		fromRegion := rec.GetInt("from_region")
		toRegion := rec.GetInt("to_region")
		from := rec.GetInt("from_solarsystem")
		to := rec.GetInt("to_solarsystem")
		if from > to {
			from, to = to, from
			fromRegion, toRegion = toRegion, fromRegion
		}
		key := strconv.Itoa(from) + "-" + strconv.Itoa(to)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		out = append(out, JumpbridgeDTO{
			From:       from,
			To:         to,
			FromRegion: fromRegion,
			ToRegion:   toRegion,
			Friendly:   rec.GetBool("is_friendly"),
		})
	}
	return out, nil
}

func buildIntFilter(field string, ids []int) (string, dbx.Params) {
	if len(ids) == 0 {
		return "", dbx.Params{}
	}
	filter := field + " = {:id0}"
	params := dbx.Params{"id0": ids[0]}
	for i := 1; i < len(ids); i++ {
		key := "id" + strconv.Itoa(i)
		filter += " || " + field + " = {:" + key + "}"
		params[key] = ids[i]
	}
	return filter, params
}

func buildStringFilter(field string, ids []string) (string, dbx.Params) {
	if len(ids) == 0 {
		return "", dbx.Params{}
	}
	filter := field + " = {:id0}"
	params := dbx.Params{"id0": ids[0]}
	for i := 1; i < len(ids); i++ {
		key := "id" + strconv.Itoa(i)
		filter += " || " + field + " = {:" + key + "}"
		params[key] = ids[i]
	}
	return filter, params
}

func (h *MapHandler) parseRegionIDs(value string) ([]int, error) {
	return h.parseIDsByName(value, store.CollectionRegions)
}

func (h *MapHandler) parseSystemIDs(value string) ([]int, error) {
	return h.parseIDsByName(value, store.CollectionSolarSystems)
}

func (h *MapHandler) parseIDsByName(value string, collection string) ([]int, error) {
	parts := format.SplitTokens(value)
	out := []int{}
	for _, part := range parts {
		if part == "" {
			continue
		}
		num, parseErr := strconv.Atoi(part)
		if parseErr == nil {
			out = append(out, num)
			continue
		}
		name := normalizeRegionToken(part)
		record, recordErr := h.findRecordByName(collection, name)
		if recordErr != nil {
			return nil, recordErr
		}
		out = append(out, int(record.GetInt("eve_id")))
	}
	return out, nil
}

func normalizeRegionToken(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
}

func (h *MapHandler) findRecordByName(collection string, name string) (*core.Record, error) {
	records, recordsErr := h.App.FindRecordsByFilter(
		collection,
		"name = {:name}",
		"",
		1,
		0, dbx.Params{"name": name},
	)
	if recordsErr == nil && len(records) > 0 {
		return records[0], nil
	}
	records, recordsErr = h.App.FindRecordsByFilter(
		collection,
		"name ~ {:name}",
		"",
		50,
		0, dbx.Params{"name": "%" + name + "%"},
	)
	if recordsErr != nil {
		return nil, recordsErr
	}
	for _, record := range records {
		if strings.EqualFold(record.GetString("name"), name) {
			return record, nil
		}
	}
	return nil, errors.New("not found")
}

func overlap(a []int, b []int) bool {
	set := map[int]struct{}{}
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}
