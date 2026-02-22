package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/esi"
)

type stubESIClient struct {
	affiliation func(ctx context.Context, characterID int) (int, int, error)
}

func (s stubESIClient) Characters(ctx context.Context, user *core.Record, accessToken string) ([]int, error) {
	_ = ctx
	_ = user
	_ = accessToken
	return nil, nil
}

func (s stubESIClient) CharacterLocation(ctx context.Context, characterID string, accessToken string) (esi.CharacterLocation, error) {
	_ = ctx
	_ = characterID
	_ = accessToken
	return esi.CharacterLocation{}, nil
}

func (s stubESIClient) CharacterAffiliation(ctx context.Context, characterID int) (int, int, error) {
	if s.affiliation == nil {
		return 0, 0, nil
	}
	return s.affiliation(ctx, characterID)
}

func (s stubESIClient) SearchOrganizations(ctx context.Context, characterID int, accessToken, query string, strict bool) (corporationIDs, allianceIDs []int, err error) {
	_ = ctx
	_ = characterID
	_ = accessToken
	_ = query
	_ = strict
	return nil, nil, nil
}

func (s stubESIClient) SetAutopilotWaypoint(ctx context.Context, req esi.AutopilotRequest, accessToken string) error {
	_ = ctx
	_ = req
	_ = accessToken
	return nil
}

func TestResolveCharacterAffiliationForCallbackFallsBackToExistingCharacter(t *testing.T) {
	t.Parallel()

	provider := &EVEProvider{
		ESI: stubESIClient{
			affiliation: func(ctx context.Context, characterID int) (int, int, error) {
				_ = ctx
				_ = characterID
				return 0, 0, errors.New("304 Not Modified")
			},
		},
	}
	existing := core.NewRecord(core.NewBaseCollection("characters"))
	existing.Set("eve_corporation_id", 9001)
	existing.Set("eve_alliance_id", 8002)

	corpID, allianceID, err := provider.resolveCharacterAffiliationForCallback(context.Background(), 42, existing)
	if err != nil {
		t.Fatalf("resolveCharacterAffiliationForCallback() error = %v", err)
	}
	if corpID != 9001 || allianceID != 8002 {
		t.Fatalf("resolveCharacterAffiliationForCallback() = (%d,%d), want (9001,8002)", corpID, allianceID)
	}
}

func TestResolveCharacterAffiliationForCallbackFailsWithoutExistingCharacter(t *testing.T) {
	t.Parallel()

	provider := &EVEProvider{
		ESI: stubESIClient{
			affiliation: func(ctx context.Context, characterID int) (int, int, error) {
				_ = ctx
				_ = characterID
				return 0, 0, errors.New("esi down")
			},
		},
	}

	_, _, err := provider.resolveCharacterAffiliationForCallback(context.Background(), 42, nil)
	if err != ErrFailedFetchCharacter {
		t.Fatalf("resolveCharacterAffiliationForCallback() error = %v, want %v", err, ErrFailedFetchCharacter)
	}
}
