package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/store"
)

//nolint:gocognit // migration setup/teardown is intentionally linear and verbose.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionJumpbridges)
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("from_structure_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "from_structure_id"})
		}
		if collection.Fields.GetByName("to_structure_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "to_structure_id"})
		}
		collection.AddIndex("idx_jumpbridges_from_structure_id", false, "from_structure_id", "")
		collection.AddIndex("idx_jumpbridges_to_structure_id", false, "to_structure_id", "")
		collection.RemoveIndex("idx_jumpbridges_structure_id")
		if collection.Fields.GetByName("structure_id") != nil {
			collection.Fields.RemoveByName("structure_id")
		}
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionJumpbridges)
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("structure_id") == nil {
			collection.Fields.Add(&core.NumberField{Name: "structure_id"})
		}
		collection.AddIndex("idx_jumpbridges_structure_id", true, "structure_id", "")
		collection.RemoveIndex("idx_jumpbridges_from_structure_id")
		collection.RemoveIndex("idx_jumpbridges_to_structure_id")
		if collection.Fields.GetByName("from_structure_id") != nil {
			collection.Fields.RemoveByName("from_structure_id")
		}
		if collection.Fields.GetByName("to_structure_id") != nil {
			collection.Fields.RemoveByName("to_structure_id")
		}
		return app.Save(collection)
	})
}
