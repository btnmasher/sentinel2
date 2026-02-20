package pb_migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/store"
)

//nolint:gocognit // Migration registration blocks are intentionally schema-focused.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionPlanets)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			collection = core.NewBaseCollection(store.CollectionPlanets)
		}

		if collection.Fields.GetByName("eve_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "eve_id", Required: true})
		}
		if collection.Fields.GetByName("name") == nil {
			collection.Fields.Add(&core.TextField{Name: "name", Required: true})
		}
		if collection.Fields.GetByName("system_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "system_id", Required: true})
		}
		if collection.Fields.GetByName("system_name") == nil {
			collection.Fields.Add(&core.TextField{Name: "system_name"})
		}
		if collection.Fields.GetByName("celestial_index") == nil {
			collection.Fields.Add(&core.NumberField{Name: "celestial_index"})
		}

		collection.AddIndex("idx_planets_eve_id", true, "eve_id", "")
		collection.AddIndex("idx_planets_system_name", false, "system_id,name", "")
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionPlanets)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return app.Delete(collection)
	})
}
