package pb_migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/pbutils"
)

func init() {
	m.Register(func(app core.App) error {
		users, usersErr := app.FindCollectionByNameOrId("users")
		if usersErr != nil {
			if !errors.Is(usersErr, sql.ErrNoRows) {
				return usersErr
			}
			users = core.NewAuthCollection("users")
		}
		users.Fields.Add(
			&core.TextField{Name: "sub"},
			&core.DateField{Name: "created_at", Required: true},
			&core.TextField{Name: "auth_provider"},
			&core.TextField{Name: "auth_provider_sub"},
			&core.SelectField{
				Name:      "access_level",
				Values:    []string{"user", "staff", "admin"},
				MaxSelect: 1,
			},
			&core.TextField{Name: "oauth_access_token"},
			&core.TextField{Name: "oauth_refresh_token"},
			&core.DateField{Name: "oauth_access_expires_at"},
			&core.DateField{Name: "oauth_refresh_expires_at"},
			&core.NumberField{Name: "eve_character_id"},
			&core.TextField{Name: "eve_character_name"},
			&core.NumberField{Name: "eve_corporation_id"},
			&core.NumberField{Name: "eve_alliance_id"},
			&core.DateField{Name: "session_revoked_at"},
		)
		users.AddIndex("idx_users_auth_provider_sub", true, "auth_provider_sub", "auth_provider_sub != ''")
		if err := app.Save(users); err != nil {
			return err
		}

		uploaderTokens := core.NewBaseCollection("uploader_tokens")
		uploaderTokens.Fields.Add(
			&core.RelationField{
				Name:         "user",
				Required:     true,
				CollectionId: users.Id,
				MaxSelect:    1,
			},
			&core.BoolField{Name: "revoked"},
			&core.DateField{Name: "created_date", Required: true},
		)
		if err := app.Save(uploaderTokens); err != nil {
			return err
		}

		intelChannels := core.NewBaseCollection("intel_channels")
		intelChannels.Fields.Add(
			&core.TextField{Name: "channel_name", Required: true},
		)
		intelChannels.AddIndex("idx_intel_channels_channel_name", true, "channel_name", "")
		if err := app.Save(intelChannels); err != nil {
			return err
		}

		regions := core.NewBaseCollection("regions")
		regions.Fields.Add(
			&core.NumberField{Name: "eve_id", Required: true},
			&core.TextField{Name: "name", Required: true},
			&core.NumberField{Name: "raw_x"},
			&core.NumberField{Name: "raw_y"},
			&core.NumberField{Name: "metro_x"},
			&core.NumberField{Name: "metro_y"},
			&core.NumberField{Name: "real_x"},
			&core.NumberField{Name: "real_y"},
			&core.NumberField{Name: "eve2d_x"},
			&core.NumberField{Name: "eve2d_y"},
		)
		regions.AddIndex("idx_regions_eve_id", true, "eve_id", "")
		if err := app.Save(regions); err != nil {
			return err
		}

		constellations := core.NewBaseCollection("constellations")
		constellations.Fields.Add(
			&core.NumberField{Name: "eve_id", Required: true},
			&core.TextField{Name: "name", Required: true},
			&core.NumberField{Name: "region_id", Required: true},
		)
		constellations.AddIndex("idx_constellations_eve_id", true, "eve_id", "")
		if err := app.Save(constellations); err != nil {
			return err
		}

		solarSystems := core.NewBaseCollection("solar_systems")
		solarSystems.Fields.Add(
			&core.NumberField{Name: "eve_id", Required: true},
			&core.TextField{Name: "name", Required: true},
			&core.NumberField{Name: "security_status"},
			&core.NumberField{Name: "constellation", Required: true},
			&core.NumberField{Name: "region_id", Required: true},
			&core.TextField{Name: "region_name"},
			&core.NumberField{Name: "raw_x"},
			&core.NumberField{Name: "raw_y"},
			&core.NumberField{Name: "raw_z"},
			&core.NumberField{Name: "dotlan_x"},
			&core.NumberField{Name: "dotlan_y"},
			&core.NumberField{Name: "metro_x"},
			&core.NumberField{Name: "metro_y"},
			&core.NumberField{Name: "real_x"},
			&core.NumberField{Name: "real_y"},
			&core.NumberField{Name: "eve2d_x"},
			&core.NumberField{Name: "eve2d_y"},
		)
		solarSystems.AddIndex("idx_solar_systems_eve_id", true, "eve_id", "")
		if err := app.Save(solarSystems); err != nil {
			return err
		}

		gates := core.NewBaseCollection("gates")
		gates.Fields.Add(
			&core.NumberField{Name: "from_solarsystem", Required: true},
			&core.NumberField{Name: "to_solarsystem", Required: true},
			&core.NumberField{Name: "from_region", Required: true},
			&core.NumberField{Name: "to_region", Required: true},
			&core.NumberField{Name: "from_constellation", Required: true},
			&core.NumberField{Name: "to_constellation", Required: true},
			&core.NumberField{Name: "from_dotlan_x"},
			&core.NumberField{Name: "from_dotlan_y"},
			&core.NumberField{Name: "to_dotlan_x"},
			&core.NumberField{Name: "to_dotlan_y"},
			&core.NumberField{Name: "from_metro_x"},
			&core.NumberField{Name: "from_metro_y"},
			&core.NumberField{Name: "to_metro_x"},
			&core.NumberField{Name: "to_metro_y"},
		)
		if err := app.Save(gates); err != nil {
			return err
		}

		jumpbridges := core.NewBaseCollection("jumpbridges")
		jumpbridges.Fields.Add(
			&core.NumberField{Name: "structure_id", Required: true},
			&core.NumberField{Name: "from_solarsystem", Required: true},
			&core.NumberField{Name: "to_solarsystem", Required: true},
			&core.NumberField{Name: "from_region", Required: true},
			&core.NumberField{Name: "to_region", Required: true},
			&core.BoolField{Name: "is_friendly"},
			&core.DateField{Name: "created_date", Required: true},
		)
		jumpbridges.AddIndex("idx_jumpbridges_structure_id", true, "structure_id", "")
		if err := app.Save(jumpbridges); err != nil {
			return err
		}

		intelReports := core.NewBaseCollection("intel_reports")
		intelReports.Fields.Add(
			&core.NumberField{Name: "report_id", Required: true},
			&core.NumberField{Name: "report_time", Required: true},
			&core.TextField{Name: "author", Required: true},
			&core.TextField{Name: "text", Required: true},
			&core.JSONField{Name: "systems", Required: true},
			&core.JSONField{Name: "regions", Required: true},
			&core.RelationField{
				Name:         "uploader_user",
				CollectionId: users.Id,
				MaxSelect:    1,
			},
			&core.RelationField{
				Name:         "channel",
				CollectionId: intelChannels.Id,
				MaxSelect:    1,
			},
		)
		if err := app.Save(intelReports); err != nil {
			return err
		}

		intelHashes := core.NewBaseCollection("intel_report_hashes")
		intelHashes.Fields.Add(
			&core.TextField{Name: "hash", Required: true},
			&core.NumberField{Name: "hash_index", Required: true},
			&core.NumberField{Name: "report_time", Required: true},
			&core.DateField{Name: "expires_at", Required: true},
		)
		if err := app.Save(intelHashes); err != nil {
			return err
		}

		uploaders := core.NewBaseCollection("intel_uploaders")
		uploaders.Fields.Add(
			&core.RelationField{
				Name:         "user",
				Required:     true,
				CollectionId: users.Id,
				MaxSelect:    1,
			},
			&core.DateField{Name: "expires_at", Required: true},
		)
		if err := app.Save(uploaders); err != nil {
			return err
		}

		topRoutes := core.NewBaseCollection("top_routes")
		topRoutes.Fields.Add(
			&core.TextField{Name: "bucket", Required: true},
			&core.NumberField{Name: "system_id", Required: true},
			&core.NumberField{Name: "count", Required: true},
			&core.DateField{Name: "updated_at", Required: true},
		)
		if err := app.Save(topRoutes); err != nil {
			return err
		}

		sdeMeta := core.NewBaseCollection("sde_meta")
		sdeMeta.Fields.Add(
			&core.TextField{Name: "key", Required: true},
			&core.TextField{Name: "value", Required: true},
			&core.DateField{Name: "updated_at", Required: true},
		)
		sdeMeta.AddIndex("idx_sde_meta_key", true, "key", "")
		if err := app.Save(sdeMeta); err != nil {
			return err
		}

		jobRuns := core.NewBaseCollection("job_runs")
		jobRuns.Fields.Add(
			&core.TextField{Name: "job_id", Required: true},
			&core.TextField{Name: "kind", Required: true},
			&core.TextField{Name: "step"},
			&core.TextField{Name: "trigger"},
			&core.BoolField{Name: "hidden"},
			&core.SelectField{
				Name:      "status",
				Values:    []string{"running", "success", "failed", "partial", "skipped", "canceled"},
				MaxSelect: 1,
			},
			&core.TextField{Name: "actor_id"},
			&core.TextField{Name: "actor_display_name"},
			&core.DateField{Name: "started_at"},
			&core.DateField{Name: "completed_at"},
			&core.NumberField{Name: "duration_ms"},
			&core.TextField{Name: "error"},
		)
		adminRule := `@request.auth.id != "" && @request.auth.access_level = "admin"`
		jobRuns.ListRule = pbutils.RulePtr(adminRule)
		jobRuns.ViewRule = pbutils.RulePtr(adminRule)
		if err := app.Save(jobRuns); err != nil {
			return err
		}

		alliances := core.NewBaseCollection("alliances")
		alliances.Fields.Add(
			&core.NumberField{Name: "eve_id", Required: true},
			&core.TextField{Name: "name"},
			&core.DateField{Name: "updated_at"},
		)
		alliances.AddIndex("idx_alliances_eve_id", true, "eve_id", "")
		if err := app.Save(alliances); err != nil {
			return err
		}

		corporations := core.NewBaseCollection("corporations")
		corporations.Fields.Add(
			&core.NumberField{Name: "eve_id", Required: true},
			&core.TextField{Name: "name"},
			&core.DateField{Name: "updated_at"},
		)
		corporations.AddIndex("idx_corporations_eve_id", true, "eve_id", "")
		if err := app.Save(corporations); err != nil {
			return err
		}

		allowedAlliances := core.NewBaseCollection("allowed_alliances")
		allowedAlliances.Fields.Add(
			&core.NumberField{Name: "eve_id", Required: true},
			&core.TextField{Name: "name"},
		)
		allowedAlliances.AddIndex("idx_allowed_alliances_eve_id", true, "eve_id", "")
		if err := app.Save(allowedAlliances); err != nil {
			return err
		}

		allowedCorporations := core.NewBaseCollection("allowed_corporations")
		allowedCorporations.Fields.Add(
			&core.NumberField{Name: "eve_id", Required: true},
			&core.TextField{Name: "name"},
		)
		allowedCorporations.AddIndex("idx_allowed_corporations_eve_id", true, "eve_id", "")
		if err := app.Save(allowedCorporations); err != nil {
			return err
		}

		characters := core.NewBaseCollection("characters")
		characters.Fields.Add(
			&core.RelationField{
				Name:         "user",
				Required:     true,
				CollectionId: users.Id,
				MaxSelect:    1,
			},
			&core.NumberField{Name: "eve_character_id", Required: true},
			&core.TextField{Name: "eve_character_name"},
			&core.NumberField{Name: "eve_corporation_id"},
			&core.NumberField{Name: "eve_alliance_id"},
			&core.BoolField{Name: "is_main"},
			&core.TextField{Name: "oauth_access_token"},
			&core.TextField{Name: "oauth_refresh_token"},
			&core.DateField{Name: "oauth_access_expires_at"},
			&core.DateField{Name: "oauth_refresh_expires_at"},
			&core.TextField{Name: "oauth_scopes"},
			&core.DateField{Name: "esi_last_refresh_at"},
			&core.TextField{Name: "esi_last_error"},
			&core.BoolField{Name: "esi_token_valid"},
		)
		characters.AddIndex("idx_characters_eve_character_id", true, "eve_character_id", "")
		if err := app.Save(characters); err != nil {
			return err
		}

		adminAuditLogs := core.NewBaseCollection("admin_audit_logs")
		adminAuditLogs.Fields.Add(
			&core.TextField{Name: "action", Required: true},
			&core.TextField{Name: "summary", Required: true},
			&core.TextField{Name: "actor_id"},
			&core.TextField{Name: "actor_email"},
			&core.TextField{Name: "actor_display_name"},
			&core.TextField{Name: "target_user_id"},
			&core.TextField{Name: "target_user_name"},
			&core.NumberField{Name: "target_character_id"},
			&core.TextField{Name: "target_character_name"},
			&core.DateField{Name: "created"},
		)
		if err := app.Save(adminAuditLogs); err != nil {
			return err
		}

		staffRule := `@request.auth.id != "" && (@request.auth.access_level = "staff" || @request.auth.access_level = "admin")`
		authRule := `@request.auth.id != ""`
		userRule := `@request.auth.id != "" && user = @request.auth.id`

		if err := pbutils.SetRules(app, "intel_channels", staffRule, staffRule, staffRule, staffRule, staffRule); err != nil {
			return err
		}
		if err := pbutils.SetRules(app, "intel_reports", authRule, authRule, "", "", ""); err != nil {
			return err
		}
		if err := pbutils.SetRules(app, "intel_uploaders", "", "", "", "", ""); err != nil {
			return err
		}
		if err := pbutils.SetRules(app, "jumpbridges", staffRule, staffRule, "", "", ""); err != nil {
			return err
		}
		if err := pbutils.SetRules(app, "characters", userRule, userRule, "", "", ""); err != nil {
			return err
		}
		if err := pbutils.SetRules(app, "solar_systems", staffRule, staffRule, "", "", ""); err != nil {
			return err
		}
		if err := pbutils.SetRules(app, "alliances", adminRule, adminRule, "", "", ""); err != nil {
			return err
		}
		if err := pbutils.SetRules(app, "corporations", adminRule, adminRule, "", "", ""); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		collectionNames := []string{
			"admin_audit_logs",
			"characters",
			"corporations",
			"alliances",
			"allowed_corporations",
			"allowed_alliances",
			"job_runs",
			"sde_meta",
			"top_routes",
			"intel_uploaders",
			"intel_report_hashes",
			"intel_reports",
			"jumpbridges",
			"gates",
			"solar_systems",
			"constellations",
			"regions",
			"intel_channels",
			"uploader_tokens",
			"users",
		}

		for _, name := range collectionNames {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}
				return err
			}
			if err := app.Delete(collection); err != nil {
				return err
			}
		}

		return nil
	})
}
