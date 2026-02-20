package pb_migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

//nolint:gocognit // Migration registration blocks are intentionally schema-focused.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("moons")
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			collection = core.NewBaseCollection("moons")
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
		if collection.Fields.GetByName("planet_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "planet_id"})
		}
		if collection.Fields.GetByName("planet_name") == nil {
			collection.Fields.Add(&core.TextField{Name: "planet_name"})
		}

		collection.AddIndex("idx_moons_eve_id", true, "eve_id", "")
		collection.AddIndex("idx_moons_system_id_name", false, "system_id,name", "")
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("moons")
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return app.Delete(collection)
	})
}
