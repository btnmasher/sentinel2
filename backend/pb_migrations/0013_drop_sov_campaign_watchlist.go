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
		collection, err := app.FindCollectionByNameOrId(store.CollectionSovereigntyCampaignWatchlist)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return app.Delete(collection)
	}, func(app core.App) error {
		_, err := app.FindCollectionByNameOrId(store.CollectionSovereigntyCampaignWatchlist)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		collection := core.NewBaseCollection(store.CollectionSovereigntyCampaignWatchlist)
		collection.Fields.Add(
			&core.SelectField{
				Name:      "hostility",
				Required:  true,
				Values:    []string{"ours", "friendly", "neutral", "complicated", "hostile"},
				MaxSelect: 1,
			},
			&core.NumberField{Name: "alliance_id", Required: true},
			&core.TextField{Name: "alliance_name", Required: true},
			&core.TextField{Name: "alliance_ticker"},
		)
		collection.AddIndex("idx_sov_campaign_watchlist_entity", true, "alliance_id", "")
		staffRule := `@request.auth.id != "" && (@request.auth.access_level = "staff" || @request.auth.access_level = "admin")`
		collection.ListRule = pbutils.RulePtr(staffRule)
		collection.ViewRule = pbutils.RulePtr(staffRule)
		collection.CreateRule = pbutils.RulePtr(staffRule)
		collection.UpdateRule = pbutils.RulePtr(staffRule)
		collection.DeleteRule = pbutils.RulePtr(staffRule)
		return app.Save(collection)
	})
}
