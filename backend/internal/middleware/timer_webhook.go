package middleware

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/config"
)

func RequireTimersWebhookToken(cfg *config.Config) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		if cfg == nil || cfg.TimerSource != config.TimerSourceWebhook {
			return ErrForbidden
		}

		authHeader := strings.TrimSpace(c.Request.Header.Get("Authorization"))
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token == "" || authHeader == token || cfg.TimersWebhookToken == "" || token != cfg.TimersWebhookToken {
			return ErrInvalidTimersWebhook
		}

		return c.Next()
	}
}
