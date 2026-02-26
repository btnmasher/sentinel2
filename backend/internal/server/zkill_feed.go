package server

import (
	"context"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/config"
	"sentinel2/internal/zkill"
)

func registerZKillFeedWorker(app *pocketbase.PocketBase, cfg *config.Config, deps *dependencies, lifecycleCtx context.Context) {
	if cfg == nil || deps == nil {
		return
	}
	if !cfg.ZKillFeedEnabled {
		return
	}
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		go zkill.NewFeedIngestor(app, cfg, deps.intelService, deps.realtimePublisher).Run(lifecycleCtx)
		return nil
	})
}
