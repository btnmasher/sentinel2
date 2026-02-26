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
		return setOrganizationSovSyncBoolRequired(app, false)
	}, func(app core.App) error {
		return setOrganizationSovSyncBoolRequired(app, true)
	})
}

func setOrganizationSovSyncBoolRequired(app core.App, required bool) error {
	collection, err := app.FindCollectionByNameOrId(store.CollectionOrganizationStandings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	field := collection.Fields.GetByName("include_in_sov_sync")
	boolField, ok := field.(*core.BoolField)
	if !ok || boolField == nil {
		return nil
	}
	if boolField.Required == required {
		return nil
	}

	boolField.Required = required
	return app.Save(collection)
}
