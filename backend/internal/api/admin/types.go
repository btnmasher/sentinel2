package admin

import (
	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/audit"
	"sentinel2/internal/auth"
	"sentinel2/internal/cleanup"
	"sentinel2/internal/intel"
	"sentinel2/internal/jumpbridges"
	timerssvc "sentinel2/internal/timers"
)

type Handler struct {
	App         *pocketbase.PocketBase
	Refresher   *auth.CharacterRefresher
	Provider    auth.Provider
	Cleanup     *cleanup.Service
	Intel       *intel.IntelService
	Timers      *timerssvc.Service
	Jumpbridges *jumpbridges.JumpbridgeService
	Audit       *audit.Service
}

type searchItem struct {
	CharacterRecordID string `json:"character_record_id"`
	CharacterID       int    `json:"character_id"`
	Name              string `json:"name"`
	UserID            string `json:"user_id"`
	AuthProvider      string `json:"auth_provider"`
	IsMain            bool   `json:"is_main"`
	MainName          string `json:"main_name"`
}

type characterResponse struct {
	ID               string `json:"id"`
	CharacterID      int    `json:"character_id"`
	Name             string `json:"name"`
	CorpID           int    `json:"corp_id"`
	CorpName         string `json:"corp_name"`
	AllianceID       int    `json:"alliance_id"`
	AllianceName     string `json:"alliance_name"`
	IsMain           bool   `json:"is_main"`
	Scopes           string `json:"scopes"`
	ESILastRefreshAt string `json:"esi_last_refresh_at"`
	ESILastError     string `json:"esi_last_error"`
	ESITokenValid    bool   `json:"esi_token_valid"`
}

type auditLogEntry struct {
	ID                  string `json:"id"`
	Action              string `json:"action"`
	Summary             string `json:"summary"`
	ActorID             string `json:"actor_id"`
	ActorDisplayName    string `json:"actor_display_name"`
	TargetUserID        string `json:"target_user_id"`
	TargetUserName      string `json:"target_user_name"`
	TargetCharacterID   int    `json:"target_character_id"`
	TargetCharacterName string `json:"target_character_name"`
	TargetType          string `json:"target_type"`
	TargetID            string `json:"target_id"`
	TargetLabel         string `json:"target_label"`
	TargetMeta          any    `json:"target_meta"`
	Created             string `json:"created"`
}

type jobRunEntry struct {
	ID               string `json:"id"`
	JobID            string `json:"job_id"`
	Kind             string `json:"kind"`
	Step             string `json:"step"`
	Message          string `json:"message"`
	Trigger          string `json:"trigger"`
	Status           string `json:"status"`
	ActorID          string `json:"actor_id"`
	ActorDisplayName string `json:"actor_display_name"`
	StartedAt        string `json:"started_at"`
	CompletedAt      string `json:"completed_at"`
	DurationMs       int64  `json:"duration_ms"`
}

type jobRunGroup struct {
	Parent jobRunEntry   `json:"parent"`
	Steps  []jobRunEntry `json:"steps"`
}

type userResponse struct {
	UserID             string              `json:"user_id"`
	AccessLevel        string              `json:"access_level"`
	AuthProvider       string              `json:"auth_provider"`
	SessionRevokedAt   string              `json:"session_revoked_at"`
	UploaderTokenValid bool                `json:"uploader_token_valid"`
	Characters         []characterResponse `json:"characters"`
}

type mapDataResponse struct {
	JobID string `json:"job_id"`
	Step  string `json:"step"`
}

type siteAnnouncementPayload struct {
	Variant string `json:"variant"`
	Message string `json:"message"`
}

type MapUpdateHandler struct {
	App   *pocketbase.PocketBase
	Audit *audit.Service
}
