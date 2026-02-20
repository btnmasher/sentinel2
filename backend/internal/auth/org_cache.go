package auth

import (
	"context"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/esi"
	"sentinel2/internal/store"
)

func ensureOrgName(ctx context.Context, app *pocketbase.PocketBase, publicESI *esi.ESIPublicClient, collection string, eveID int) {
	if app == nil || eveID == 0 {
		return
	}
	if name, _, ok := store.GetOrg(app, collection, eveID); ok && name != "" {
		return
	}
	if publicESI == nil {
		return
	}
	var name string
	var ticker string
	var err error
	switch collection {
	case store.CollectionAlliances:
		name, ticker, err = publicESI.AllianceDetails(ctx, eveID)
	case store.CollectionCorporations:
		name, ticker, _, err = publicESI.CorporationDetails(ctx, eveID)
	default:
		return
	}
	if err != nil || name == "" {
		return
	}
	_ = store.UpsertOrg(app, collection, eveID, name, ticker)
}
