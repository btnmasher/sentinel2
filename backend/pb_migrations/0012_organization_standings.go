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
		_, err := app.FindCollectionByNameOrId(store.CollectionOrganizationStandings)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		collection := core.NewBaseCollection(store.CollectionOrganizationStandings)
		collection.Fields.Add(
			&core.SelectField{
				Name:      "owner_type",
				Required:  true,
				Values:    []string{"alliance", "corporation"},
				MaxSelect: 1,
			},
			&core.SelectField{
				Name:      "hostility",
				Required:  true,
				Values:    []string{"ours", "friendly", "neutral", "complicated", "hostile"},
				MaxSelect: 1,
			},
			&core.BoolField{Name: "include_in_sov_sync"},
			&core.NumberField{Name: "corporation_id"},
			&core.TextField{Name: "corporation_name"},
			&core.TextField{Name: "corporation_ticker"},
			&core.NumberField{Name: "alliance_id"},
			&core.TextField{Name: "alliance_name"},
			&core.TextField{Name: "alliance_ticker"},
		)
		collection.AddIndex("idx_organization_standings_owner_type", false, "owner_type", "")
		collection.AddIndex("idx_organization_standings_alliance_id", false, "alliance_id", "")
		collection.AddIndex("idx_organization_standings_corporation_id", false, "corporation_id", "")

		staffRule := `@request.auth.id != "" && (@request.auth.access_level = "staff" || @request.auth.access_level = "admin")`
		collection.ListRule = pbutils.RulePtr(staffRule)
		collection.ViewRule = pbutils.RulePtr(staffRule)
		collection.CreateRule = pbutils.RulePtr(staffRule)
		collection.UpdateRule = pbutils.RulePtr(staffRule)
		collection.DeleteRule = pbutils.RulePtr(staffRule)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionOrganizationStandings)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return app.Delete(collection)
	})
}
