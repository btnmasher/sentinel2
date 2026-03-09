package jobs

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/store"
)

const collectionJobRuns = "job_runs"

type JobTracker struct {
	app *pocketbase.PocketBase
}

type JobOptions struct {
	Kind    string
	Step    string
	Trigger string
	ActorID string
	Hidden  bool
}

func NewJobTracker(app *pocketbase.PocketBase) *JobTracker {
	return &JobTracker{app: app}
}

func (t *JobTracker) IsRunning(kind, step string) (bool, error) {
	if t == nil || t.app == nil {
		return false, nil
	}
	filter := "kind = {:kind} && status = {:status}"
	params := map[string]any{
		"kind":   kind,
		"status": StatusRunning,
	}

	if step == "" {
		filter += " && (step = \"\" || step = null)"
	} else {
		filter += " && step = {:step}"
		params["step"] = step
	}
	records, err := t.app.FindRecordsByFilter(collectionJobRuns, filter, "", 1, 0, params)
	if err != nil {
		return false, err
	}
	return len(records) > 0, nil
}

func (t *JobTracker) Start(jobID string, opts JobOptions) (*core.Record, error) {
	collection, err := t.app.FindCollectionByNameOrId(collectionJobRuns)
	if err != nil {
		return nil, err
	}
	actorID := opts.ActorID
	if actorID == "" {
		actorID = "SYSTEM"
	}
	displayName := ""
	if actorID == "SYSTEM" {
		displayName = "System"
	} else {
		user, userErr := t.app.FindRecordById(store.CollectionUsers, actorID)
		if userErr == nil {
			displayName = user.GetString("eve_character_name")
		}
		if displayName == "" {
			displayName = actorID
		}
	}
	record := core.NewRecord(collection)
	record.Set("job_id", jobID)
	record.Set("kind", opts.Kind)
	record.Set("step", opts.Step)
	record.Set("trigger", opts.Trigger)
	record.Set("status", StatusRunning)
	record.Set("started_at", time.Now().UTC())
	record.Set("actor_id", actorID)
	record.Set("actor_display_name", displayName)
	record.Set("hidden", opts.Hidden)
	if err := t.app.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (t *JobTracker) Finish(record *core.Record, err error) {
	if record == nil {
		return
	}
	now := time.Now().UTC()
	record.Set("completed_at", now)
	started, ok := record.Get("started_at").(time.Time)
	if ok {
		record.Set("duration_ms", now.Sub(started).Milliseconds())
	}

	if err != nil {
		record.Set("status", StatusFailed)
		record.Set("message", err.Error())
	} else {
		record.Set("status", StatusSuccess)
	}
	_ = t.app.Save(record)
}

func (t *JobTracker) FinishPartial(record *core.Record, err error) {
	if record == nil {
		return
	}
	now := time.Now().UTC()
	record.Set("completed_at", now)
	started, ok := record.Get("started_at").(time.Time)
	if ok {
		record.Set("duration_ms", now.Sub(started).Milliseconds())
	}
	record.Set("status", StatusPartial)
	if err != nil {
		record.Set("message", err.Error())
	}
	_ = t.app.Save(record)
}

func (t *JobTracker) FinishSkipped(record *core.Record, reason string) {
	if record == nil {
		return
	}
	now := time.Now().UTC()
	record.Set("completed_at", now)
	started, ok := record.Get("started_at").(time.Time)
	if ok {
		record.Set("duration_ms", now.Sub(started).Milliseconds())
	}
	record.Set("status", StatusSkipped)
	record.Set("message", reason)
	_ = t.app.Save(record)
}

func (t *JobTracker) FinishCanceled(record *core.Record, reason string) {
	if record == nil {
		return
	}
	now := time.Now().UTC()
	record.Set("completed_at", now)
	started, ok := record.Get("started_at").(time.Time)
	if ok {
		record.Set("duration_ms", now.Sub(started).Milliseconds())
	}
	record.Set("status", StatusCanceled)
	record.Set("message", reason)
	_ = t.app.Save(record)
}

func (t *JobTracker) MarkStaleRunningAsTimeout(maxAge time.Duration) (int, error) {
	if t == nil || t.app == nil {
		return 0, nil
	}

	if maxAge <= 0 {
		return 0, nil
	}

	records, err := t.app.FindRecordsByFilter(
		collectionJobRuns,
		"status = {:status}",
		"",
		0,
		0,
		map[string]any{"status": StatusRunning},
	)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	cutoff := now.Add(-maxAge)
	updated := 0
	for _, record := range records {
		if record == nil {
			continue
		}

		jobID := record.GetString("job_id")
		started := recordTime(record.Get("started_at"))
		if started.After(cutoff) {
			continue
		}
		// If this process still has a live cancel handle for the stale job,
		// request graceful cancellation first and let the runner finalize state.
		if Cancel(jobID) {
			continue
		}

		record.Set("completed_at", now)
		record.Set("duration_ms", now.Sub(started).Milliseconds())
		record.Set("status", StatusTimeout)
		record.Set("message", fmt.Sprintf("timed out by cleanup after running longer than %s", maxAge))
		if saveErr := t.app.Save(record); saveErr != nil {
			return updated, saveErr
		}
		updated++
	}
	return updated, nil
}
