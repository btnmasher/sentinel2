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
	OIDCPortalURL  string   `json:"oidc_portal_url"`
}

func AppConfig(cfg config.Config) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		standalone := cfg.AuthBackend == "eve"
		return c.JSON(http.StatusOK, appConfigResponse{
			AuthBackend:    cfg.AuthBackend,
			Standalone:     standalone,
			DefaultRegions: cfg.DefaultRegions(),
			OIDCPortalURL:  cfg.OIDCPortalURL,
		})
	}
}
