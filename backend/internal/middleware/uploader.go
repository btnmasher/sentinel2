package middleware

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/intel"
)

// RequireUploaderToken enforces that a valid uploader token is provided.
// Tokens must be provided via Authorization: Bearer <token>.
func RequireUploaderToken(service *intel.IntelService) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		authHeader := strings.TrimSpace(c.Request.Header.Get("Authorization"))
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

		if token == "" || authHeader == token {
			return ErrInvalidUploaderToken
		}

		if service == nil {
			return ErrInvalidUploaderToken
		}
		record, recordErr := service.ValidateUploaderToken(token)
		if recordErr != nil {
			return ErrInvalidUploaderToken
		}

		c.Set("uploader_token", token)
		c.Set("uploader_token_id", record.Id)
		c.Set("uploader_user_id", record.GetString("user"))

		return c.Next()
	}
}
