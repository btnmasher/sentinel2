package server

import (
	"context"
	"sentinel2/internal/config"
	"sentinel2/internal/jobs"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func registerCrons(app *pocketbase.PocketBase, cfg *config.Config, deps *dependencies) {
	registerCleanupCron(app, deps)
	if cfg.AuthBackend == "eve" {
		registerCharacterRefreshCron(app, deps)
	}
	registerUploaderReleaseBootstrap(app, deps)
	registerUploaderReleaseCron(app, deps)
	registerSDEBootstrap(app)
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

func registerSDEBootstrap(app *pocketbase.PocketBase) {
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		runMapDataBootstrap(app)
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
