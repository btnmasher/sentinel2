package mapdata

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/store"
)

const defaultCelestialListLimit = 200

type CelestialSearchItem struct {
	ID       int
	Name     string
	SystemID int
}

func ListMoonsBySystem(app core.App, systemID, limit int) ([]CelestialSearchItem, error) {
	return listCelestialsBySystem(app, store.CollectionMoons, systemID, limit)
}

func ListPlanetsBySystem(app core.App, systemID, limit int) ([]CelestialSearchItem, error) {
	return listCelestialsBySystem(app, store.CollectionPlanets, systemID, limit)
}

func listCelestialsBySystem(app core.App, collectionName string, systemID, limit int) ([]CelestialSearchItem, error) {
	if app == nil || systemID <= 0 {
		return []CelestialSearchItem{}, nil
	}

	if limit <= 0 {
		limit = defaultCelestialListLimit
	}

	records, err := app.FindRecordsByFilter(
		collectionName,
		"system_id = {:system}",
		"name",
		limit,
		0,
		dbx.Params{"system": systemID},
	)
	if err != nil {
		return nil, err
	}

	out := make([]CelestialSearchItem, 0, len(records))
	for _, record := range records {
		out = append(out, CelestialSearchItem{
			ID:       record.GetInt("eve_id"),
			Name:     record.GetString("name"),
			SystemID: record.GetInt("system_id"),
		})
	}
	return out, nil
}
