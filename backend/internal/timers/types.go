package timers

import "time"

type ParseResult struct {
	Title     string
	System    string
	TimerKind string
	ExpiresAt time.Time
	Raw       string
}

type ListInput struct {
	Statuses  []string
	RegionIDs []int
	From      *time.Time
	To        *time.Time
	Limit     int
}

type CreateInput struct {
	WebhookID              string
	Title                  string
	SystemID               int
	System                 string
	Standing               string
	TimerKind              string
	StructureType          string
	StageLabel             string
	PlanetID               int
	PlanetName             string
	MoonID                 int
	MoonName               string
	OwnerCorporationID     int
	OwnerCorporationName   string
	OwnerCorporationTicker string
	OwnerAllianceID        int
	OwnerAllianceName      string
	OwnerAllianceTicker    string
	SkyhookFullnessPct     *int
	AttackersScorePct      *int
	DefenderScorePct       *int
	Stage                  int
	TotalStages            int
	Severity               string
	Status                 string
	ExpiresAt              time.Time
	Source                 string
	SourceRef              string
	Notes                  string
	RawText                string
	ReplacementAction      string
}

type UpdateInput struct {
	Title                  *string
	Standing               *string
	TimerKind              *string
	StructureType          *string
	StageLabel             *string
	PlanetID               *int
	PlanetName             *string
	MoonID                 *int
	MoonName               *string
	OwnerCorporationID     *int
	OwnerCorporationName   *string
	OwnerCorporationTicker *string
	OwnerAllianceID        *int
	OwnerAllianceName      *string
	OwnerAllianceTicker    *string
	SkyhookFullnessPct     *int
	AttackersScorePct      *int
	DefenderScorePct       *int
	Stage                  *int
	TotalStages            *int
	Severity               *string
	Status                 *string
	ExpiresAt              *time.Time
	SourceRef              *string
	Notes                  *string
	RawText                *string
	ReplacementAction      *string
}

type Signal struct {
	SystemID           int
	Count              int
	RemainingCount     int
	NextExpiresAt      time.Time
	Severity           string
	StandingType       string
	TimerKind          string
	Title              string
	StructureType      string
	StageLabel         string
	PlanetName         string
	MoonName           string
	SkyhookFullnessPct *int
	Timers             []SignalTimerPreview
}

type SignalTimerPreview struct {
	Title              string
	ExpiresAt          time.Time
	Severity           string
	StandingType       string
	TimerKind          string
	StructureType      string
	StageLabel         string
	PlanetName         string
	MoonName           string
	SkyhookFullnessPct *int
}

type SystemSearchItem struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	RegionID int    `json:"region_id"`
	Region   string `json:"region"`
}

type EntitySearchItem struct {
	Type           string                `json:"type"`
	ID             int                   `json:"id"`
	Name           string                `json:"name"`
	Ticker         string                `json:"ticker"`
	ParentAlliance *EntitySearchAlliance `json:"parent_alliance,omitempty"`
}

type EntitySearchAlliance struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Ticker string `json:"ticker"`
}

type EntitySearchRequester struct {
	CharacterID int
	AccessToken string
}

type EntitySearchScope string

const (
	EntitySearchScopeBoth        EntitySearchScope = "both"
	EntitySearchScopeAlliance    EntitySearchScope = "alliance"
	EntitySearchScopeCorporation EntitySearchScope = "corporation"
)

type MoonSearchItem struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	SystemID int    `json:"system_id"`
}

type PlanetSearchItem struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	SystemID int    `json:"system_id"`
}
