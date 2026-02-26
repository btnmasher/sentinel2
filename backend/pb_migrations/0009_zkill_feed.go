package pb_migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/store"
)

//nolint:gocognit // migration registration block is intentionally schema-focused.
func init() {
	m.Register(func(app core.App) error {
		intelReports, err := app.FindCollectionByNameOrId(store.CollectionIntelReports)
		if err != nil {
			return err
		}
		if intelReports.Fields.GetByName("meta") == nil {
			intelReports.Fields.Add(&core.JSONField{Name: "meta"})
		}
		if err := app.Save(intelReports); err != nil {
			return err
		}

		_, err = app.FindCollectionByNameOrId(store.CollectionZKillFeedState)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			state := core.NewBaseCollection(store.CollectionZKillFeedState)
			state.Fields.Add(
				&core.TextField{Name: "key", Required: true},
				&core.NumberField{Name: "sequence_id", Required: true},
				&core.DateField{Name: "updated_at", Required: true},
			)
			state.AddIndex("idx_zkill_feed_state_key", true, "key", "")
			if err := app.Save(state); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		intelReports, err := app.FindCollectionByNameOrId(store.CollectionIntelReports)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			return nil
		}
		if intelReports.Fields.GetByName("meta") != nil {
			intelReports.Fields.RemoveByName("meta")
			if err := app.Save(intelReports); err != nil {
				return err
			}
		}

		if collection, err := app.FindCollectionByNameOrId(store.CollectionZKillFeedState); err == nil {
			if deleteErr := app.Delete(collection); deleteErr != nil {
				return deleteErr
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		return nil
	})
}
