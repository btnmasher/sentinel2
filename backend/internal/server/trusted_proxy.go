package server

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/config"
)

var preferredTrustedProxyHeaders = []string{
	"CF-Connecting-IP",
	"True-Client-IP",
}

func registerTrustedProxyDefaults(app *pocketbase.PocketBase, cfg *config.Config) {
	if app == nil {
		return
	}

	headers := append([]string(nil), preferredTrustedProxyHeaders...)
	if cfg != nil {
		if configured := cfg.TrustedProxyHeaders; len(configured) > 0 {
			headers = configured
		}
	}

	app.OnSettingsReload().BindFunc(func(e *core.SettingsReloadEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		settings := e.App.Settings()
		settings.TrustedProxy.Headers = append([]string(nil), headers...)
		return nil
	})
}
