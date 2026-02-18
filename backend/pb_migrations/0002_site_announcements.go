package pb_migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/pbutils"
	"sentinel2/internal/store"
)

//nolint:gocognit // Migration registration blocks are intentionally long and schema-focused.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionSiteAnnouncements)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			collection = core.NewBaseCollection(store.CollectionSiteAnnouncements)
		}

		if collection.Fields.GetByName("variant") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "variant",
				Required:  true,
				Values:    []string{"banner", "modal"},
				MaxSelect: 1,
			})
		}
		if collection.Fields.GetByName("archived") == nil {
			collection.Fields.Add(&core.BoolField{Name: "archived"})
		}
		if collection.Fields.GetByName("message") == nil {
			collection.Fields.Add(&core.EditorField{
				Name:        "message",
				Required:    true,
				ConvertURLs: false,
			})
		}
		if collection.Fields.GetByName("published_at") == nil {
			collection.Fields.Add(&core.DateField{Name: "published_at"})
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		authRule := `@request.auth.id != ""`
		adminRule := `@request.auth.id != "" && @request.auth.access_level = "admin"`
		return pbutils.SetRules(
			app,
			store.CollectionSiteAnnouncements,
			authRule,
			authRule,
			adminRule,
			adminRule,
			adminRule,
		)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionSiteAnnouncements)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return app.Delete(collection)
	})
}
