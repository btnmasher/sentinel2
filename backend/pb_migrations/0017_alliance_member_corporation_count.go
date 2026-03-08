package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/store"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionAlliances)
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("member_corporation_count") == nil {
			collection.Fields.Add(&core.NumberField{Name: "member_corporation_count"})
		}
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionAlliances)
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("member_corporation_count") != nil {
			collection.Fields.RemoveByName("member_corporation_count")
		}
		return app.Save(collection)
	})
}
