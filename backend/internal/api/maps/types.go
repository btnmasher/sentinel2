package maps

import (
	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/auth"
	"sentinel2/internal/config"
	"sentinel2/internal/esi"
	"sentinel2/internal/intel"
	timerssvc "sentinel2/internal/timers"
)

type MapHandler struct {
	App          *pocketbase.PocketBase
	Config       *config.Config
	ESI          esi.ESIClient
	Provider     auth.Provider
	EVE          *auth.EVEProvider
	Routes       *intel.RoutePlanner
	TopRoutesSvc *intel.TopRoutesService
	Timers       *timerssvc.Service
}

type Region struct {
	Region   int    `json:"region"`
	Name     string `json:"name"`
	Position struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"position"`
}

type System struct {
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

type Gate struct {
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

type Jumpbridge struct {
	From       int  `json:"from"`
	To         int  `json:"to"`
	FromRegion int  `json:"from_region"`
	ToRegion   int  `json:"to_region"`
	Friendly   bool `json:"friendly"`
	Disabled   bool `json:"disabled"`
}

type TimerSignal struct {
	SystemID           int            `json:"system_id"`
	Count              int            `json:"count"`
	RemainingCount     int            `json:"remaining_count"`
	NextExpiresAt      string         `json:"next_expires_at"`
	Severity           string         `json:"severity"`
	StandingType       string         `json:"standing_type"`
	TimerKind          string         `json:"timer_kind"`
	Title              string         `json:"title"`
	StructureType      string         `json:"structure_type"`
	StageLabel         string         `json:"stage_label"`
	PlanetName         string         `json:"planet_name"`
	MoonName           string         `json:"moon_name"`
	SkyhookFullnessPct *int           `json:"skyhook_fullness_pct"`
	Timers             []TimerPreview `json:"timers"`
}

type TimerPreview struct {
	Title              string `json:"title"`
	NextExpiresAt      string `json:"next_expires_at"`
	Severity           string `json:"severity"`
	StandingType       string `json:"standing_type"`
	TimerKind          string `json:"timer_kind"`
	StructureType      string `json:"structure_type"`
	StageLabel         string `json:"stage_label"`
	PlanetName         string `json:"planet_name"`
	MoonName           string `json:"moon_name"`
	SkyhookFullnessPct *int   `json:"skyhook_fullness_pct"`
}

type MapResponse struct {
	Regions      map[int]Region      `json:"regions"`
	Systems      map[int]System      `json:"systems"`
	Gates        []Gate              `json:"gates"`
	Jumpbridges  []Jumpbridge        `json:"jumpbridges"`
	TimerSignals map[int]TimerSignal `json:"timer_signals"`
}

type MapOverlaysResponse struct {
	Jumpbridges  []Jumpbridge        `json:"jumpbridges"`
	TimerSignals map[int]TimerSignal `json:"timer_signals"`
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
	RegionID    int    `json:"region_id"`
	InSpace     bool   `json:"in_space"`
}

type LocationsRequest struct {
	Characters []int64 `json:"characters"`
}

type LocationsResponse struct {
	Locations []LocationEntry `json:"locations"`
}
