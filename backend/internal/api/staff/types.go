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
