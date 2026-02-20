package esi

import (
	"context"

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

type ESIClient interface {
	Characters(ctx context.Context, user *core.Record, accessToken string) ([]int, error)
	CharacterLocation(ctx context.Context, characterID string, accessToken string) (CharacterLocation, error)
	CharacterAffiliation(ctx context.Context, characterID int) (int, int, error)
	SearchOrganizations(ctx context.Context, characterID int, accessToken, query string, strict bool) (corporationIDs, allianceIDs []int, err error)
	SetAutopilotWaypoint(ctx context.Context, req AutopilotRequest, accessToken string) error
}
