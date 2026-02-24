package staff

import (
	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/audit"
	"sentinel2/internal/jumpbridges"
)

type channelDTO struct {
	ID          string `json:"id"`
	ChannelName string `json:"channel_name"`
}

type channelListResponse struct {
	Channels []channelDTO `json:"channels"`
}

type channelCreateResponse struct {
	ID string `json:"id"`
}

type ChannelsHandler struct {
	App   *pocketbase.PocketBase
	Audit *audit.Service
}

type jumpbridgeImportResponse struct {
	Count int `json:"count"`
}

type jumpbridgeMutationResponse struct {
	Changed bool `json:"changed"`
	Count   int  `json:"count"`
}

type JumpbridgeHandler struct {
	App     *pocketbase.PocketBase
	Service *jumpbridges.JumpbridgeService
	Audit   *audit.Service
}

type SovereigntyCampaignWatchlistEntityDTO struct {
	ID             string `json:"id"`
	Hostility      string `json:"hostility"`
	AllianceID     int    `json:"alliance_id"`
	AllianceName   string `json:"alliance_name"`
	AllianceTicker string `json:"alliance_ticker"`
}

type SovereigntyCampaignWatchlistResponse struct {
	Entities []SovereigntyCampaignWatchlistEntityDTO `json:"entities"`
}

type SovereigntyCampaignWatchlistCreateResponse struct {
	ID string `json:"id"`
}

type SovereigntyCampaignWatchlistHandler struct {
	App   *pocketbase.PocketBase
	Audit *audit.Service
}
