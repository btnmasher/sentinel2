package middleware

import (
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/intel"
)

// RequireUploaderToken enforces that a valid uploader token is provided.
// Tokens must be provided via Authorization: Bearer <token>.
func RequireUploaderToken(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		authHeader := strings.TrimSpace(c.Request.Header.Get("Authorization"))
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

		if token == "" || authHeader == token {
			return ErrInvalidUploaderToken
		}

		service := intel.NewIntelService(app)
		record, recordErr := service.ValidateUploaderToken(token)
		if recordErr != nil {
			return ErrInvalidUploaderToken
		}

		c.Set("uploader_token", token)
		c.Set("uploader_user_id", record.GetString("user"))

		return c.Next()
	}
}
