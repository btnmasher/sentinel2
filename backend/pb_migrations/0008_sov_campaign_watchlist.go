package pb_migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/pbutils"
	"sentinel2/internal/store"
)

const timerScorePercentCap = 100

func init() {
	m.Register(func(app core.App) error {
		isNewWatchlistCollection := false
		watchlistCollection, err := app.FindCollectionByNameOrId(store.CollectionSovereigntyCampaignWatchlist)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			watchlistCollection = core.NewBaseCollection(store.CollectionSovereigntyCampaignWatchlist)
			isNewWatchlistCollection = true
		}
		if watchlistCollection.Fields.GetByName("key") == nil {
			watchlistCollection.Fields.Add(&core.TextField{Name: "key", Required: true})
		}
		if watchlistCollection.Fields.GetByName("hostility") == nil {
			watchlistCollection.Fields.Add(&core.SelectField{
				Name:      "hostility",
				Required:  true,
				Values:    []string{"ours", "friendly", "neutral", "complicated", "hostile"},
				MaxSelect: 1,
			})
		} else if field, ok := watchlistCollection.Fields.GetByName("hostility").(*core.SelectField); ok {
			field.Required = true
			field.Values = []string{"ours", "friendly", "neutral", "complicated", "hostile"}
			field.MaxSelect = 1
		}
		if watchlistCollection.Fields.GetByName("alliance_id") == nil {
			watchlistCollection.Fields.Add(&core.NumberField{Name: "alliance_id", Required: true})
		} else if field, ok := watchlistCollection.Fields.GetByName("alliance_id").(*core.NumberField); ok {
			field.Required = true
		}
		if watchlistCollection.Fields.GetByName("alliance_name") == nil {
			watchlistCollection.Fields.Add(&core.TextField{Name: "alliance_name", Required: true})
		} else if field, ok := watchlistCollection.Fields.GetByName("alliance_name").(*core.TextField); ok {
			field.Required = true
		}
		if watchlistCollection.Fields.GetByName("alliance_ticker") == nil {
			watchlistCollection.Fields.Add(&core.TextField{Name: "alliance_ticker"})
		}
		if isNewWatchlistCollection {
			watchlistCollection.AddIndex("idx_sov_campaign_watchlist_key", true, "key", "")
			watchlistCollection.AddIndex("idx_sov_campaign_watchlist_entity", true, "alliance_id", "")
		}

		staffRule := `@request.auth.id != "" && (@request.auth.access_level = "staff" || @request.auth.access_level = "admin")`
		watchlistCollection.ListRule = pbutils.RulePtr(staffRule)
		watchlistCollection.ViewRule = pbutils.RulePtr(staffRule)
		watchlistCollection.CreateRule = pbutils.RulePtr(staffRule)
		watchlistCollection.UpdateRule = pbutils.RulePtr(staffRule)
		watchlistCollection.DeleteRule = pbutils.RulePtr(staffRule)

		if err := app.Save(watchlistCollection); err != nil {
			return err
		}

		timersCollection, err := app.FindCollectionByNameOrId(store.CollectionTimers)
		if err != nil {
			return err
		}
		if timersCollection.Fields.GetByName("created_by_name") == nil {
			timersCollection.Fields.Add(&core.TextField{Name: "created_by_name"})
		}
		if timersCollection.Fields.GetByName("attackers_score_pct") == nil {
			minValue := float64(0)
			maxValue := float64(timerScorePercentCap)
			timersCollection.Fields.Add(&core.NumberField{Name: "attackers_score_pct", Min: &minValue, Max: &maxValue, OnlyInt: true})
		}
		if timersCollection.Fields.GetByName("defender_score_pct") == nil {
			minValue := float64(0)
			maxValue := float64(timerScorePercentCap)
			timersCollection.Fields.Add(&core.NumberField{Name: "defender_score_pct", Min: &minValue, Max: &maxValue, OnlyInt: true})
		}
		return app.Save(timersCollection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionSovereigntyCampaignWatchlist)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return app.Delete(collection)
	})
}
