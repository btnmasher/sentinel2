package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// 0011 intentionally no-ops. The old sovereignty watchlist schema is fully
// replaced by organization_standings in later migrations.
func init() {
	m.Register(func(app core.App) error {
		return nil
	}, func(app core.App) error {
		return nil
	})
}
