package pb_migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const jobRunsCollection = "job_runs"

//nolint:gocognit // migration setup/teardown and data backfill are intentionally verbose.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(jobRunsCollection)
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("message") == nil {
			collection.Fields.Add(&core.TextField{Name: "message"})
		}
		if err := app.Save(collection); err != nil {
			return err
		}

		records, err := app.FindRecordsByFilter(jobRunsCollection, "", "", 0, 0)
		if err != nil {
			return err
		}
		for _, record := range records {
			if record == nil {
				continue
			}
			message := strings.TrimSpace(record.GetString("message"))
			if message == "" {
				message = strings.TrimSpace(record.GetString("error"))
			}
			if message == "" {
				step := strings.TrimSpace(record.GetString("step"))
				// Legacy job completion summaries were temporarily stored in step.
				if strings.Contains(step, " ") {
					message = step
				}
			}
			if message == record.GetString("message") {
				continue
			}
			record.Set("message", message)
			if err := app.Save(record); err != nil {
				return err
			}
		}

		collection, err = app.FindCollectionByNameOrId(jobRunsCollection)
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("error") != nil {
			collection.Fields.RemoveByName("error")
		}
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(jobRunsCollection)
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("error") == nil {
			collection.Fields.Add(&core.TextField{Name: "error"})
		}
		if err := app.Save(collection); err != nil {
			return err
		}

		records, err := app.FindRecordsByFilter(jobRunsCollection, "", "", 0, 0)
		if err != nil {
			return err
		}
		for _, record := range records {
			if record == nil {
				continue
			}
			legacyError := strings.TrimSpace(record.GetString("error"))
			if legacyError != "" {
				continue
			}
			message := strings.TrimSpace(record.GetString("message"))
			if message == "" {
				continue
			}
			record.Set("error", message)
			if err := app.Save(record); err != nil {
				return err
			}
		}

		collection, err = app.FindCollectionByNameOrId(jobRunsCollection)
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("message") != nil {
			collection.Fields.RemoveByName("message")
		}
		return app.Save(collection)
	})
}
