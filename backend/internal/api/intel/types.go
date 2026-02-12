package intel

import (
	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/config"
	"sentinel2/internal/intel"
)

type IntelHandler struct {
	App     *pocketbase.PocketBase
	Config  config.Config
	Service *intel.IntelService
}

type intelRetrieveResponse struct {
	Intel     []intel.IntelReport `json:"intel"`
	Uploaders int                 `json:"uploaders"`
	Version   string              `json:"version"`
}

type uploaderTokenResponse struct {
	Token string `json:"token"`
}

type uploaderRealtimeTokenResponse struct {
	Token               string `json:"token"`
	Topic               string `json:"topic"`
	ExpiresAt           int64  `json:"expires_at"`
	RefreshAfterSeconds int64  `json:"refresh_after_seconds"`
}

type submitPayload struct {
	Text      string `json:"text"`
	ChannelID string `json:"channel_id"`
}
