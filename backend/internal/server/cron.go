package server

import (
	"context"
	"sentinel2/internal/config"
	"sentinel2/internal/jobs"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func registerCrons(app *pocketbase.PocketBase, cfg *config.Config, deps *dependencies) {
	lifecycleCtx, cancelLifecycle := context.WithCancel(context.Background())
	app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		cancelLifecycle()
		return e.Next()
	})

	registerCleanupCron(app, deps)
	if cfg.AuthBackend == "eve" {
		registerCharacterRefreshCron(app, deps)
		if cfg.TimersEnabled && cfg.TimerSource == config.TimerSourceStandalone {
			registerSkyhookSyncCron(app, deps)
		}
	}
	registerUploaderReleaseBootstrap(app, deps)
	registerUploaderReleaseCron(app, deps)
	if cfg.TimersEnabled && cfg.TimerSource == config.TimerSourceStandalone {
		registerSovCampaignSyncCron(app, deps)
	}
	registerZKillFeedWorker(app, cfg, deps, lifecycleCtx)
	registerSDEBootstrap(app, lifecycleCtx)
	registerSDECron(app)
}

func registerCleanupCron(app *pocketbase.PocketBase, deps *dependencies) {
	app.Cron().MustAdd("cleanup", "30 * * * *", func() {
		runCleanupJob(app, deps)
	})
}

func registerCharacterRefreshCron(app *pocketbase.PocketBase, deps *dependencies) {
	app.Cron().MustAdd("character_refresh", "0 * * * *", func() {
		runCharacterRefreshJob(app, deps)
	})
}

func registerSDEBootstrap(app *pocketbase.PocketBase, lifecycleCtx context.Context) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		go runMapDataBootstrap(app, lifecycleCtx)
		return nil
	})
}

func registerSDECron(app *pocketbase.PocketBase) {
	// Check SDE freshness every 6 hours.
	app.Cron().MustAdd("map_data_update", "0 */6 * * *", func() {
		ctx, cancel := context.WithCancel(context.Background())
		runMapDataCron(app, ctx)
		cancel()
	})
}

func registerUploaderReleaseBootstrap(app *pocketbase.PocketBase, deps *dependencies) {
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		runUploaderReleaseRefresh(app, deps, jobs.TriggerServerStartup)
		return nil
	})
}

func registerUploaderReleaseCron(app *pocketbase.PocketBase, deps *dependencies) {
	app.Cron().MustAdd("uploader_releases", "*/15 * * * *", func() {
		runUploaderReleaseRefresh(app, deps, jobs.TriggerCronSchedule)
	})
}

func registerSovCampaignSyncCron(app *pocketbase.PocketBase, deps *dependencies) {
	app.Cron().MustAdd("sov_campaign_sync", "*/1 * * * *", func() {
		runSovCampaignSync(app, deps)
	})
}

func registerSkyhookSyncCron(app *pocketbase.PocketBase, deps *dependencies) {
	app.Cron().MustAdd("skyhook_notification_sync", "*/1 * * * *", func() {
		runSkyhookSync(app, deps)
	})
}
