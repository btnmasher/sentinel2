package middleware

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/auth"
	"sentinel2/internal/store"
)

// RequireMainCharacter enforces that EVE-authenticated users have a main character.
func RequireMainCharacter(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		if c.Auth == nil {
			return ErrUnauthorized
		}
		if c.Auth.GetString("auth_provider") != auth.AuthProviderEVE {
			return c.Next()
		}
		records, recordsErr := app.FindRecordsByFilter(
			store.CollectionCharacters,
			"user = {:user} && is_main = true",
			"",
			1,
			0,
			map[string]any{"user": c.Auth.Id},
		)
		if recordsErr != nil || len(records) == 0 {
			return ErrMainCharacterRequired
		}
		return c.Next()
	}
}
