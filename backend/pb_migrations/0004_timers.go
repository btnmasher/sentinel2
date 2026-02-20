package pb_migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/pbutils"
	"sentinel2/internal/store"
)

const skyhookFullnessPercentMax = 100

var timerStandingValues = []string{
	"ours",
	"friendly",
	"complicated",
	"neutral",
	"hostile",
}

var timerStructureValues = []string{
	"upwell_citadel_astrahus",
	"upwell_citadel_fortizar",
	"upwell_citadel_keepstar",
	"upwell_engineering_raitaru",
	"upwell_engineering_azbel",
	"upwell_engineering_sotiyo",
	"upwell_refinery_athanor",
	"upwell_refinery_tatara",
	"ansiblex_jump_bridge",
	"pharolux_cyno_beacon",
	"tenebrex_cyno_jammer",
	"orbital_skyhook",
	"metenox_moon_drill",
	"sovereignty_hub",
	"mercenary_den",
	"customs_office_poco",
	"player_owned_starbase",
	"custom",
}

var timerReplacementActionValues = []string{
	"not_replaceable",
	"logi_replacement",
	"corp_replacement",
	"alliance_replacement",
}

//nolint:gocognit // Migration registration blocks are intentionally schema-focused.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionTimers)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			collection = core.NewBaseCollection(store.CollectionTimers)
		}

		if collection.Fields.GetByName("title") == nil {
			collection.Fields.Add(&core.TextField{Name: "title", Required: true})
		}
		if collection.Fields.GetByName("system_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "system_id", Required: true})
		}
		if collection.Fields.GetByName("system_name") == nil {
			collection.Fields.Add(&core.TextField{Name: "system_name", Required: true})
		}
		if collection.Fields.GetByName("region_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "region_id", Required: true})
		}
		if collection.Fields.GetByName("region_name") == nil {
			collection.Fields.Add(&core.TextField{Name: "region_name", Required: true})
		}
		if collection.Fields.GetByName("standing_type") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "standing_type",
				Required:  true,
				Values:    timerStandingValues,
				MaxSelect: 1,
			})
		}
		if collection.Fields.GetByName("timer_kind") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "timer_kind",
				Required:  true,
				Values:    []string{"reinforcement", "anchoring", "unanchoring", "extraction", "custom"},
				MaxSelect: 1,
			})
		}
		if collection.Fields.GetByName("structure_type") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "structure_type",
				Required:  true,
				Values:    timerStructureValues,
				MaxSelect: 1,
			})
		}
		if collection.Fields.GetByName("stage_label") == nil {
			collection.Fields.Add(&core.TextField{Name: "stage_label"})
		}
		if collection.Fields.GetByName("moon_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "moon_id"})
		}
		if collection.Fields.GetByName("moon_name") == nil {
			collection.Fields.Add(&core.TextField{Name: "moon_name"})
		}
		if collection.Fields.GetByName("planet_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "planet_id"})
		}
		if collection.Fields.GetByName("planet_name") == nil {
			collection.Fields.Add(&core.TextField{Name: "planet_name"})
		}
		if collection.Fields.GetByName("owner_corporation_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "owner_corporation_id"})
		}
		if collection.Fields.GetByName("owner_corporation_name") == nil {
			collection.Fields.Add(&core.TextField{Name: "owner_corporation_name"})
		}
		if collection.Fields.GetByName("owner_corporation_ticker") == nil {
			collection.Fields.Add(&core.TextField{Name: "owner_corporation_ticker"})
		}
		if collection.Fields.GetByName("owner_alliance_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "owner_alliance_id"})
		}
		if collection.Fields.GetByName("owner_alliance_name") == nil {
			collection.Fields.Add(&core.TextField{Name: "owner_alliance_name"})
		}
		if collection.Fields.GetByName("owner_alliance_ticker") == nil {
			collection.Fields.Add(&core.TextField{Name: "owner_alliance_ticker"})
		}
		if collection.Fields.GetByName("skyhook_fullness_pct") == nil {
			minValue := float64(0)
			maxValue := float64(skyhookFullnessPercentMax)
			collection.Fields.Add(&core.NumberField{Name: "skyhook_fullness_pct", Min: &minValue, Max: &maxValue, OnlyInt: true})
		}
		if collection.Fields.GetByName("stage") == nil {
			collection.Fields.Add(&core.NumberField{Name: "stage"})
		}
		if collection.Fields.GetByName("total_stages") == nil {
			collection.Fields.Add(&core.NumberField{Name: "total_stages"})
		}
		if collection.Fields.GetByName("severity") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "severity",
				Required:  true,
				Values:    []string{"low", "medium", "high", "critical"},
				MaxSelect: 1,
			})
		}
		if collection.Fields.GetByName("status") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "status",
				Required:  true,
				Values:    []string{"active", "canceled"},
				MaxSelect: 1,
			})
		}
		if collection.Fields.GetByName("expires_at") == nil {
			collection.Fields.Add(&core.DateField{Name: "expires_at", Required: true})
		}
		if collection.Fields.GetByName("source") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "source",
				Required:  true,
				Values:    []string{"manual", "esi"},
				MaxSelect: 1,
			})
		}
		if collection.Fields.GetByName("source_ref") == nil {
			collection.Fields.Add(&core.TextField{Name: "source_ref"})
		}
		if collection.Fields.GetByName("notes") == nil {
			collection.Fields.Add(&core.TextField{Name: "notes"})
		}
		if collection.Fields.GetByName("raw_text") == nil {
			collection.Fields.Add(&core.TextField{Name: "raw_text"})
		}
		if collection.Fields.GetByName("replacement_action") == nil {
			collection.Fields.Add(&core.SelectField{
				Name:      "replacement_action",
				Required:  true,
				Values:    timerReplacementActionValues,
				MaxSelect: 1,
			})
		}
		if collection.Fields.GetByName("created_by") == nil {
			users, usersErr := app.FindCollectionByNameOrId(store.CollectionUsers)
			if usersErr == nil {
				collection.Fields.Add(&core.RelationField{
					Name:         "created_by",
					CollectionId: users.Id,
					MaxSelect:    1,
				})
			}
		}
		if collection.Fields.GetByName("canceled_at") == nil {
			collection.Fields.Add(&core.DateField{Name: "canceled_at"})
		}
		if collection.Fields.GetByName("canceled_by") == nil {
			users, usersErr := app.FindCollectionByNameOrId(store.CollectionUsers)
			if usersErr == nil {
				collection.Fields.Add(&core.RelationField{
					Name:         "canceled_by",
					CollectionId: users.Id,
					MaxSelect:    1,
				})
			}
		}

		collection.AddIndex("idx_timers_system_id", false, "system_id", "")
		collection.AddIndex("idx_timers_region_id", false, "region_id", "")
		collection.AddIndex("idx_timers_status_expires", false, "status,expires_at", "")

		authRule := `@request.auth.id != ""`
		staffRule := `@request.auth.id != "" && (@request.auth.access_level = "staff" || @request.auth.access_level = "admin")`
		adminRule := `@request.auth.id != "" && @request.auth.access_level = "admin"`
		collection.ListRule = pbutils.RulePtr(authRule)
		collection.ViewRule = pbutils.RulePtr(authRule)
		collection.CreateRule = pbutils.RulePtr(staffRule)
		collection.UpdateRule = pbutils.RulePtr(staffRule)
		collection.DeleteRule = pbutils.RulePtr(adminRule)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionTimers)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return app.Delete(collection)
	})
}
