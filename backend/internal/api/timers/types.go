package timers

import (
	"sentinel2/internal/audit"
	"sentinel2/internal/auth"
	timerssvc "sentinel2/internal/timers"
)

type Handler struct {
	Service  *timerssvc.Service
	Audit    *audit.Service
	Provider auth.Provider
}

type createPayload struct {
	Title                  string `json:"title"`
	SystemID               int    `json:"system_id"`
	System                 string `json:"system"`
	Standing               string `json:"standing_type"`
	TimerKind              string `json:"timer_kind"`
	StructureType          string `json:"structure_type"`
	StageLabel             string `json:"stage_label"`
	PlanetID               int    `json:"planet_id"`
	PlanetName             string `json:"planet_name"`
	MoonID                 int    `json:"moon_id"`
	MoonName               string `json:"moon_name"`
	OwnerCorporationID     int    `json:"owner_corporation_id"`
	OwnerCorporationName   string `json:"owner_corporation_name"`
	OwnerCorporationTicker string `json:"owner_corporation_ticker"`
	OwnerAllianceID        int    `json:"owner_alliance_id"`
	OwnerAllianceName      string `json:"owner_alliance_name"`
	OwnerAllianceTicker    string `json:"owner_alliance_ticker"`
	SkyhookFullnessPct     *int   `json:"skyhook_fullness_pct"`
	Stage                  int    `json:"stage"`
	TotalStages            int    `json:"total_stages"`
	Severity               string `json:"severity"`
	Status                 string `json:"status"`
	ExpiresAt              string `json:"expires_at"`
	Source                 string `json:"source"`
	SourceRef              string `json:"source_ref"`
	Notes                  string `json:"notes"`
	RawText                string `json:"raw_text"`
	ReplacementAction      string `json:"replacement_action"`
}

type updatePayload struct {
	Title                  *string `json:"title"`
	Standing               *string `json:"standing_type"`
	TimerKind              *string `json:"timer_kind"`
	StructureType          *string `json:"structure_type"`
	StageLabel             *string `json:"stage_label"`
	PlanetID               *int    `json:"planet_id"`
	PlanetName             *string `json:"planet_name"`
	MoonID                 *int    `json:"moon_id"`
	MoonName               *string `json:"moon_name"`
	OwnerCorporationID     *int    `json:"owner_corporation_id"`
	OwnerCorporationName   *string `json:"owner_corporation_name"`
	OwnerCorporationTicker *string `json:"owner_corporation_ticker"`
	OwnerAllianceID        *int    `json:"owner_alliance_id"`
	OwnerAllianceName      *string `json:"owner_alliance_name"`
	OwnerAllianceTicker    *string `json:"owner_alliance_ticker"`
	SkyhookFullnessPct     *int    `json:"skyhook_fullness_pct"`
	Stage                  *int    `json:"stage"`
	TotalStages            *int    `json:"total_stages"`
	Severity               *string `json:"severity"`
	Status                 *string `json:"status"`
	ExpiresAt              *string `json:"expires_at"`
	SourceRef              *string `json:"source_ref"`
	Notes                  *string `json:"notes"`
	RawText                *string `json:"raw_text"`
	ReplacementAction      *string `json:"replacement_action"`
}

type timerDTO struct {
	ID                     string `json:"id"`
	Title                  string `json:"title"`
	SystemID               int    `json:"system_id"`
	SystemName             string `json:"system_name"`
	RegionID               int    `json:"region_id"`
	RegionName             string `json:"region_name"`
	Standing               string `json:"standing_type"`
	TimerKind              string `json:"timer_kind"`
	StructureType          string `json:"structure_type"`
	StageLabel             string `json:"stage_label"`
	PlanetID               int    `json:"planet_id"`
	PlanetName             string `json:"planet_name"`
	MoonID                 int    `json:"moon_id"`
	MoonName               string `json:"moon_name"`
	OwnerCorporationID     int    `json:"owner_corporation_id"`
	OwnerCorporationName   string `json:"owner_corporation_name"`
	OwnerCorporationTicker string `json:"owner_corporation_ticker"`
	OwnerAllianceID        int    `json:"owner_alliance_id"`
	OwnerAllianceName      string `json:"owner_alliance_name"`
	OwnerAllianceTicker    string `json:"owner_alliance_ticker"`
	SkyhookFullnessPct     int    `json:"skyhook_fullness_pct"`
	AttackersScorePct      int    `json:"attackers_score_pct"`
	DefenderScorePct       int    `json:"defender_score_pct"`
	Stage                  int    `json:"stage"`
	TotalStages            int    `json:"total_stages"`
	Severity               string `json:"severity"`
	Status                 string `json:"status"`
	ExpiresAt              string `json:"expires_at"`
	Source                 string `json:"source"`
	SourceRef              string `json:"source_ref"`
	Notes                  string `json:"notes"`
	RawText                string `json:"raw_text"`
	ReplacementAction      string `json:"replacement_action"`
	CreatedBy              string `json:"created_by"`
	CreatedByName          string `json:"created_by_name"`
	CanceledBy             string `json:"canceled_by"`
	CanceledAt             string `json:"canceled_at"`
	Created                string `json:"created"`
	Updated                string `json:"updated"`
}

type listResponse struct {
	Timers []timerDTO `json:"timers"`
}

type parseResponse struct {
	Title      string `json:"title"`
	System     string `json:"system"`
	SystemID   int    `json:"system_id"`
	TimerKind  string `json:"timer_kind"`
	Standing   string `json:"standing_type"`
	ExpiresAt  string `json:"expires_at"`
	RawExtract string `json:"raw_extract"`
}

type systemSearchResponse struct {
	Systems []timerssvc.SystemSearchItem `json:"systems"`
}

type entitySearchResponse struct {
	Entities []timerssvc.EntitySearchItem `json:"entities"`
}

type moonSearchResponse struct {
	Moons []timerssvc.MoonSearchItem `json:"moons"`
}

type planetSearchResponse struct {
	Planets []timerssvc.PlanetSearchItem `json:"planets"`
}
