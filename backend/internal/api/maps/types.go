package maps

import (
	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/auth"
	"sentinel2/internal/config"
	"sentinel2/internal/esi"
	"sentinel2/internal/intel"
)

type MapHandler struct {
	App          *pocketbase.PocketBase
	Config       *config.Config
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
