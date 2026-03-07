package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/store"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionTimers)
		if err != nil {
			return err
		}

		sourceField, ok := collection.Fields.GetByName("source").(*core.SelectField)
		if ok {
			hasWebhook := false
			for _, value := range sourceField.Values {
				if value == "webhook" {
					hasWebhook = true
					break
				}
			}
			if !hasWebhook {
				sourceField.Values = append(sourceField.Values, "webhook")
			}
		}

		if collection.Fields.GetByName("webhook_id") == nil {
			collection.Fields.Add(&core.TextField{Name: "webhook_id"})
		}

		collection.AddIndex("idx_timers_webhook_id", true, "webhook_id", "webhook_id != ''")
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(store.CollectionTimers)
		if err != nil {
			return err
		}

		if field := collection.Fields.GetByName("webhook_id"); field != nil {
			collection.Fields.RemoveByName("webhook_id")
		}

		if sourceField, ok := collection.Fields.GetByName("source").(*core.SelectField); ok {
			next := make([]string, 0, len(sourceField.Values))
			for _, value := range sourceField.Values {
				if value != "webhook" {
					next = append(next, value)
				}
			}
			sourceField.Values = next
		}

		collection.RemoveIndex("idx_timers_webhook_id")
		return app.Save(collection)
	})
}
