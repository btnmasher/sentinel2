package pb_migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/pbutils"
	"sentinel2/internal/store"
)

func init() {
	m.Register(func(app core.App) error {
		_, err := app.FindCollectionByNameOrId(store.CollectionItemTypes)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			collection := core.NewBaseCollection(store.CollectionItemTypes)
			collection.Fields.Add(
				&core.NumberField{Name: "eve_id", Required: true},
				&core.TextField{Name: "name", Required: true},
			)
			collection.AddIndex("idx_item_types_eve_id", true, "eve_id", "")
			if err := app.Save(collection); err != nil {
				return err
			}
		}

		staffRule := `@request.auth.id != "" && (@request.auth.access_level = "staff" || @request.auth.access_level = "admin")`
		if err := pbutils.SetRules(app, store.CollectionItemTypes, staffRule, staffRule, "", "", ""); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionItemTypes)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return app.Delete(collection)
	})
}
