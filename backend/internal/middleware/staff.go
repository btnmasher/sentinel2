package middleware

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/store"
)

func RequireStaff(app *pocketbase.PocketBase) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		if c.Auth == nil {
			return ErrUnauthorized
		}
		record, recordErr := app.FindRecordById(store.CollectionUsers, c.Auth.Id)
		if recordErr != nil {
			return ErrUnauthorized
		}
		accessLevel := record.GetString("access_level")
		if accessLevel != "staff" && accessLevel != "admin" {
			return ErrForbidden
		}

		return c.Next()
	}
}
