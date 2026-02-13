package middleware

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/intel"
)

// RequireUploaderToken enforces that a valid uploader token is provided.
// Preferred header is X-Uploader-Token to avoid collision with PocketBase auth token parsing.
// Authorization: Bearer <token> is accepted as a compatibility fallback.
func RequireUploaderToken(service *intel.IntelService) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		token := strings.TrimSpace(c.Request.Header.Get("X-Uploader-Token"))
		if token == "" {
			authHeader := strings.TrimSpace(c.Request.Header.Get("Authorization"))
			token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

			if token == "" || authHeader == token {
				return ErrInvalidUploaderToken
			}
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
