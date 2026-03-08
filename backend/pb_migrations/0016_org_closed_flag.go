package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/store"
)

func init() {
	m.Register(func(app core.App) error {
		for _, collectionName := range []string{store.CollectionCorporations, store.CollectionAlliances} {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				return err
			}
			if collection.Fields.GetByName("closed") == nil {
				collection.Fields.Add(&core.BoolField{Name: "closed"})
			}
			if collectionName == store.CollectionCorporations {
				if collection.Fields.GetByName("alliance_id") == nil {
					collection.Fields.Add(&core.NumberField{Name: "alliance_id"})
				}
				if collection.Fields.GetByName("member_count") == nil {
					collection.Fields.Add(&core.NumberField{Name: "member_count"})
				}
			}
			if err := app.Save(collection); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, collectionName := range []string{store.CollectionCorporations, store.CollectionAlliances} {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				return err
			}
			if collection.Fields.GetByName("closed") != nil {
				collection.Fields.RemoveByName("closed")
			}
			if collectionName == store.CollectionCorporations {
				if collection.Fields.GetByName("alliance_id") != nil {
					collection.Fields.RemoveByName("alliance_id")
				}
				if collection.Fields.GetByName("member_count") != nil {
					collection.Fields.RemoveByName("member_count")
				}
			}
			if err := app.Save(collection); err != nil {
				return err
			}
		}
		return nil
	})
}
