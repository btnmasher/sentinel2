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
	Count     int                       `json:"count"`
	LineCount int                       `json:"line_count"`
	PairCount int                       `json:"pair_count"`
	Failed    int                       `json:"failed"`
	Failures  []jumpbridgeImportFailure `json:"failures,omitempty"`
	Applied   bool                      `json:"applied"`
}

type jumpbridgeImportFailure struct {
	FromID   int    `json:"from_id"`
	ToID     int    `json:"to_id"`
	FromName string `json:"from_name"`
	ToName   string `json:"to_name"`
	Reason   string `json:"reason"`
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

type OrganizationStandingDTO struct {
	ID                string `json:"id"`
	OwnerType         string `json:"owner_type"`
	Hostility         string `json:"hostility"`
	IncludeInSovSync  bool   `json:"include_in_sov_sync"`
	CorporationID     int    `json:"corporation_id"`
	CorporationName   string `json:"corporation_name"`
	CorporationTicker string `json:"corporation_ticker"`
	AllianceID        int    `json:"alliance_id"`
	AllianceName      string `json:"alliance_name"`
	AllianceTicker    string `json:"alliance_ticker"`
}

type OrganizationStandingsResponse struct {
	Entities []OrganizationStandingDTO `json:"entities"`
}

type OrganizationStandingCreateResponse struct {
	ID string `json:"id"`
}

type OrganizationStandingsHandler struct {
	App   *pocketbase.PocketBase
	Audit *audit.Service
}
