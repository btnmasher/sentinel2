package server

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
)

const (
	skyhookSyncTimeout  = 45 * time.Second
	skyhookSyncInterval = 2 * time.Minute
)

func runSkyhookSync(app *pocketbase.PocketBase, deps *dependencies) {
	runner := jobs.NewRunner(app, &jobs.RunOptions{
		JobName: jobs.JobSkyhookSync,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobSkyhookSync,
			Trigger: jobs.TriggerCronSchedule,
			Hidden:  true,
		},
		Timeout: skyhookSyncTimeout,
		Unique:  true,
	})

	_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
		sourceCount, err := deps.timerService.NotificationSourceCount()
		if err != nil {
			return err
		}
		if sourceCount == 0 {
			return stepper.SkipParent("no eligible notification sources")
		}
		result, err := deps.timerService.SyncSkyhookNotifications(ctx, skyhookSyncInterval)
		if err != nil {
			return err
		}
		runner.WithFields(logging.Fields{
			"watched_characters": result.WatchedCharacters,
			"notifications_seen": result.NotificationsSeen,
			"intel_created":      result.IntelCreated,
			"timers_created":     result.TimersCreated,
			"timers_updated":     result.TimersUpdated,
			"skipped":            result.Skipped,
		})
		return nil
	})
}
