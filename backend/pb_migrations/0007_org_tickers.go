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
		for _, collectionName := range []string{
			store.CollectionCorporations,
			store.CollectionAlliances,
		} {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				return err
			}
			if collection.Fields.GetByName("ticker") == nil {
				collection.Fields.Add(&core.TextField{Name: "ticker"})
			}
			if err := app.Save(collection); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, collectionName := range []string{
			store.CollectionCorporations,
			store.CollectionAlliances,
		} {
			collection, err := app.FindCollectionByNameOrId(collectionName)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}
				return err
			}
			collection.Fields.RemoveByName("ticker")
			if err := app.Save(collection); err != nil {
				return err
			}
		}
		return nil
	})
}
