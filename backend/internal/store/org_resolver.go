package store

import (
	"context"
	"strings"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/esi"
)

func GetOrFetchAlliance(ctx context.Context, app *pocketbase.PocketBase, publicESI *esi.ESIPublicClient, allianceID int) (name, ticker string, ok bool, err error) {
	if allianceID <= 0 {
		return "", "", false, nil
	}
	if app != nil {
		if name, ticker, ok = GetOrg(app, CollectionAlliances, allianceID); ok {
			return name, ticker, true, nil
		}
	}
	if publicESI == nil {
		return "", "", false, nil
	}
	name, ticker, err = publicESI.AllianceDetails(ctx, allianceID)
	if err != nil || strings.TrimSpace(name) == "" {
		return "", "", false, err
	}
	if app != nil {
		_ = UpsertOrg(app, CollectionAlliances, allianceID, name, ticker)
	}
	return name, ticker, true, nil
}

func GetOrFetchCorporation(ctx context.Context, app *pocketbase.PocketBase, publicESI *esi.ESIPublicClient, corporationID int) (name, ticker string, allianceID int, ok bool, err error) {
	if corporationID <= 0 {
		return "", "", 0, false, nil
	}
	if app != nil {
		if name, ticker, ok = GetOrg(app, CollectionCorporations, corporationID); ok {
			return name, ticker, 0, true, nil
		}
	}
	if publicESI == nil {
		return "", "", 0, false, nil
	}
	name, ticker, allianceID, err = publicESI.CorporationDetails(ctx, corporationID)
	if err != nil || strings.TrimSpace(name) == "" {
		return "", "", 0, false, err
	}
	if app != nil {
		_ = UpsertOrg(app, CollectionCorporations, corporationID, name, ticker)
	}
	return name, ticker, allianceID, true, nil
}

// WarmAllianceCache ensures the alliance is present in the local cache (best effort).
func WarmAllianceCache(ctx context.Context, app *pocketbase.PocketBase, publicESI *esi.ESIPublicClient, allianceID int) error {
	_, _, _, err := GetOrFetchAlliance(ctx, app, publicESI, allianceID)
	return err
}

// WarmCorporationCache ensures the corporation is present in the local cache (best effort).
func WarmCorporationCache(ctx context.Context, app *pocketbase.PocketBase, publicESI *esi.ESIPublicClient, corporationID int) error {
	_, _, _, _, err := GetOrFetchCorporation(ctx, app, publicESI, corporationID)
	return err
}
