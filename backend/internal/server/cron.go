package server

import (
	"context"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/audit"
	"sentinel2/internal/auth"
	"sentinel2/internal/config"
	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
	"sentinel2/internal/mapdata"
	"sentinel2/internal/store"
)

func registerCrons(app *pocketbase.PocketBase, cfg config.Config, deps dependencies) {
	registerCleanupCron(app, deps)
	if cfg.AuthBackend == "eve" {
		registerCharacterRefreshCron(app, deps)
	}
	registerSDEBootstrap(app)
	registerSDECron(app)
}

func registerCleanupCron(app *pocketbase.PocketBase, deps dependencies) {
	app.Cron().MustAdd("cleanup", "30 * * * *", func() {
		runner := jobs.NewRunner(app, jobs.RunOptions{
			JobName: jobs.JobCleanup,
			JobOptions: jobs.JobOptions{
				Kind:    jobs.JobCleanup,
				Trigger: jobs.TriggerCronSchedule,
			},
			Timeout: jobs.NoTimeout,
		})
		_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
			var reportHashCount int
			var intelUploaderCount int
			var uploaderSessionCount int
			var uploaderTokenCount int
			var intelReportCount int

			if err := stepper.Run("cleanup_report_hashes", false, func(ctx context.Context) error {
				count, err := deps.cleanup.RemoveExpired(store.CollectionIntelReportHash)
				if err == nil {
					reportHashCount = count
				}
				return err
			}); err != nil {
				return err
			}

			if err := stepper.Run("cleanup_intel_uploaders", false, func(ctx context.Context) error {
				count, err := deps.cleanup.RemoveExpired(store.CollectionIntelUploaders)
				intelUploaderCount = count
				return err
			}); err != nil {
				return err
			}

			if err := stepper.Run("cleanup_uploader_sessions", false, func(ctx context.Context) error {
				count, err := deps.cleanup.RemoveExpired(store.CollectionUploaderSessions)
				uploaderSessionCount = count
				return err
			}); err != nil {
				return err
			}

			if err := stepper.Run("cleanup_uploader_tokens", false, func(ctx context.Context) error {
				count, err := deps.cleanup.RemoveRevokedUploaderTokens()
				uploaderTokenCount = count
				return err
			}); err != nil {
				return err
			}

			if err := stepper.Run("cleanup_intel_reports", false, func(ctx context.Context) error {
				count, err := deps.cleanup.RemoveOldIntelReports(15 * time.Minute)
				intelReportCount = count
				return err
			}); err != nil {
				return err
			}

			runner.WithFields(logging.Fields{
				"intel_report_hash_count": reportHashCount,
				"intel_uploaders_count":   intelUploaderCount,
				"uploader_sessions_count": uploaderSessionCount,
				"uploader_tokens_count":   uploaderTokenCount,
				"intel_reports_count":     intelReportCount,
			})

			return nil
		})
	})
}

func registerCharacterRefreshCron(app *pocketbase.PocketBase, deps dependencies) {
	app.Cron().MustAdd("character_refresh", "0 * * * *", func() {
		runner := jobs.NewRunner(app, jobs.RunOptions{
			JobName: jobs.JobCharacterRefresh,
			JobOptions: jobs.JobOptions{
				Kind:    jobs.JobCharacterRefresh,
				Trigger: jobs.TriggerCronSchedule,
			},
			Unique: true,
			JobFunc: func(ctx context.Context) context.Context {
				return auth.WithRefreshJobMeta(ctx, jobs.TriggerCronSchedule, "")
			},
		})
		var success int
		var failed int
		_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
			success, failed = deps.characterRefresher.RefreshAll(ctx)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if failed > 0 {
				stepper.Partial(fmt.Errorf("refresh completed with %d failures", failed))
			}
			return nil
		})
		runner.WithFields(logging.Fields{
			"success": success,
			"failed":  failed,
		})
		audit.New(app).Log(
			"character.refresh_all",
			fmt.Sprintf("Scheduled refresh (%d ok, %d failed)", success, failed),
			"",
			"",
			nil,
		)
	})
}

func registerSDEBootstrap(app *pocketbase.PocketBase) {
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		runner := jobs.NewRunner(app, jobs.RunOptions{
			JobName: mapdata.JobMapDataUpdate,
			JobOptions: jobs.JobOptions{
				Kind:    "map_data_update",
				Trigger: jobs.TriggerCronSchedule,
			},
			Timeout: jobs.NoTimeout,
		})
		mapdata.RunMapDataUpdate(app, runner, jobs.TriggerCronSchedule, false)
		return nil
	})
}

func registerSDECron(app *pocketbase.PocketBase) {
	// Check SDE freshness every 6 hours.
	app.Cron().MustAdd("map_data_update", "0 */6 * * *", func() {
		ctx, cancel := context.WithCancel(context.Background())
		runner := jobs.NewRunner(app, jobs.RunOptions{
			JobName: mapdata.JobMapDataUpdate,
			JobOptions: jobs.JobOptions{
				Kind:    "map_data_update",
				Trigger: jobs.TriggerCronSchedule,
			},
			Timeout: jobs.NoTimeout,
			Parent:  ctx,
		})
		mapdata.RunMapDataUpdateWithContext(ctx, app, runner, jobs.TriggerCronSchedule, false)
		cancel()
	})
}
