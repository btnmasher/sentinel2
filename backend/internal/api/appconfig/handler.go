package appconfig

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/config"
)

type appConfigResponse struct {
	AuthBackend    string   `json:"auth_backend"`
	Standalone     bool     `json:"standalone_auth"`
	DefaultRegions []string `json:"default_regions"`
	TimersEnabled  bool     `json:"timers_enabled"`
	TimerSource    string   `json:"timer_source"`
	TimersReadOnly bool     `json:"timers_read_only"`
	Version        string   `json:"version"`
}

func AppConfig(cfg *config.Config) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		if cfg == nil {
			return c.JSON(http.StatusOK, appConfigResponse{})
		}
		c.Response.Header().Set("Cache-Control", "no-store")
		standalone := cfg.AuthBackend == "eve"
		return c.JSON(http.StatusOK, appConfigResponse{
			AuthBackend:    cfg.AuthBackend,
			Standalone:     standalone,
			DefaultRegions: cfg.DefaultRegions(),
			TimersEnabled:  cfg.TimersEnabled,
			TimerSource:    cfg.TimerSource,
			TimersReadOnly: cfg.TimersReadOnly(),
			Version:        cfg.SentinelVersion,
		})
	}
}
