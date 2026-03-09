package esi

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type AutopilotRequest struct {
	CharacterID         string
	DestinationID       int
	ClearOtherWaypoints bool
	AddToBeginning      bool
}

type CharacterLocation struct {
	SolarSystemID int   `json:"solar_system_id"`
	StationID     int   `json:"station_id"`
	StructureID   int64 `json:"structure_id"`
}

type CharacterNotification struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Text      string    `json:"text"`
}

type StructureSummary struct {
	StructureIDs []int64 `json:"structure_ids"`
}

type UniverseStructure struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	OwnerID  int    `json:"owner_id"`
	SystemID int    `json:"system_id"`
	TypeID   int    `json:"type_id"`
}

type ESIClient interface {
	Characters(ctx context.Context, user *core.Record, accessToken string) ([]int, error)
	CharacterLocation(ctx context.Context, characterID string, accessToken string) (CharacterLocation, error)
	CharacterAffiliation(ctx context.Context, characterID int) (int, int, error)
	CharacterNotifications(ctx context.Context, characterID int, accessToken, ifNoneMatch string) (notifications []CharacterNotification, etag string, notModified bool, err error)
	SearchOrganizations(ctx context.Context, characterID int, accessToken, query string, strict bool, categories []string) (corporationIDs, allianceIDs []int, err error)
	SearchStructures(ctx context.Context, characterID int, accessToken, query string, strict bool) ([]int64, error)
	UniverseStructure(ctx context.Context, characterID int, accessToken string, structureID int64) (UniverseStructure, error)
	SetAutopilotWaypoint(ctx context.Context, req AutopilotRequest, accessToken string) error
}
