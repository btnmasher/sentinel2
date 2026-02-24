package server

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
)

const sovCampaignSyncTimeout = 45 * time.Second

func runSovCampaignSync(app *pocketbase.PocketBase, deps *dependencies) {
	runner := jobs.NewRunner(app, &jobs.RunOptions{
		JobName: jobs.JobSovCampaignSync,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobSovCampaignSync,
			Trigger: jobs.TriggerCronSchedule,
			Hidden:  true,
		},
		Timeout: sovCampaignSyncTimeout,
		Unique:  true,
	})

	_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
		result, err := deps.timerService.SyncSovereigntyCampaignTimers(ctx)
		if err != nil {
			return err
		}
		runner.WithFields(logging.Fields{
			"fetched":    result.Fetched,
			"considered": result.Considered,
			"created":    result.Created,
			"updated":    result.Updated,
			"canceled":   result.Canceled,
			"skipped":    result.Skipped,
		})
		return nil
	})
}
