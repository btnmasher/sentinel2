package middleware

import (
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/intel"
	"sentinel2/internal/store"
)

func RequireUploaderRealtimeSession(service *intel.IntelService) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		authRecord := c.Auth
		if authRecord == nil || authRecord.Collection() == nil || authRecord.Collection().Name != store.CollectionUploaderSessions {
			return ErrUnauthorized
		}
		if authRecord.GetString("scope") != intel.UploaderSessionScopeConfig {
			return ErrForbidden
		}
		expiresAt := authRecord.GetDateTime("expires_at")
		if expiresAt.IsZero() || expiresAt.Time().Before(time.Now()) {
			return ErrUnauthorized
		}

		userID := strings.TrimSpace(authRecord.GetString("user"))
		if userID == "" {
			return ErrUnauthorized
		}
		uploaderTokenID := strings.TrimSpace(authRecord.GetString("uploader_token"))
		if uploaderTokenID == "" {
			return ErrUnauthorized
		}
		if service == nil {
			return ErrUnauthorized
		}
		if _, tokenErr := service.ValidateUploaderTokenID(uploaderTokenID); tokenErr != nil {
			return ErrUnauthorized
		}

		c.Set("uploader_user_id", userID)
		c.Set("uploader_token_id", uploaderTokenID)
		return c.Next()
	}
}
