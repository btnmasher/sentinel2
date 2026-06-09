package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const solarSystemsCollection = "solar_systems"

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(solarSystemsCollection)
		if err != nil {
			return err
		}

		for _, field := range []string{"name_ja", "name_ko", "name_zh"} {
			if collection.Fields.GetByName(field) == nil {
				collection.Fields.Add(&core.TextField{Name: field})
			}
		}

		collection.AddIndex("idx_solar_systems_name", false, "name", "")
		collection.AddIndex("idx_solar_systems_name_ja", false, "name_ja", "")
		collection.AddIndex("idx_solar_systems_name_ko", false, "name_ko", "")
		collection.AddIndex("idx_solar_systems_name_zh", false, "name_zh", "")
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(solarSystemsCollection)
		if err != nil {
			return err
		}

		collection.RemoveIndex("idx_solar_systems_name")
		collection.RemoveIndex("idx_solar_systems_name_ja")
		collection.RemoveIndex("idx_solar_systems_name_ko")
		collection.RemoveIndex("idx_solar_systems_name_zh")

		for _, field := range []string{"name_ja", "name_ko", "name_zh"} {
			if collection.Fields.GetByName(field) != nil {
				collection.Fields.RemoveByName(field)
			}
		}

		return app.Save(collection)
	})
}
