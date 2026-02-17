package pb_migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/store"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionAuditLogs)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}

		if collection.Fields.GetByName("target_type") == nil {
			collection.Fields.Add(&core.TextField{Name: "target_type"})
		}
		if collection.Fields.GetByName("target_id") == nil {
			collection.Fields.Add(&core.TextField{Name: "target_id"})
		}
		if collection.Fields.GetByName("target_label") == nil {
			collection.Fields.Add(&core.TextField{Name: "target_label"})
		}
		if collection.Fields.GetByName("target_meta") == nil {
			collection.Fields.Add(&core.JSONField{Name: "target_meta"})
		}
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionAuditLogs)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		for _, name := range []string{"target_meta", "target_label", "target_id", "target_type"} {
			field := collection.Fields.GetByName(name)
			if field != nil {
				collection.Fields.RemoveById(field.GetId())
			}
		}
		return app.Save(collection)
	})
}
