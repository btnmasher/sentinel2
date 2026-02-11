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
	if store.GetOrgName(app, collection, eveID) != "" {
		return
	}
	if publicESI == nil {
		return
	}
	var name string
	var err error
	switch collection {
	case store.CollectionAlliances:
		name, err = publicESI.AllianceName(ctx, eveID)
	case store.CollectionCorporations:
		name, err = publicESI.CorporationName(ctx, eveID)
	default:
		return
	}
	if err != nil || name == "" {
		return
	}
	_ = store.UpsertOrgName(app, collection, eveID, name)
}
