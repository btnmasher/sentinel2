package server

import (
	"context"
	"fmt"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/audit"
	"sentinel2/internal/auth"
	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
)

func runCharacterRefreshJob(app *pocketbase.PocketBase, deps dependencies) {
	runner := jobs.NewRunner(app, jobs.RunOptions{
		JobName: jobs.JobCharacterRefresh,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobCharacterRefresh,
			Trigger: jobs.TriggerCronSchedule,
		},
		Unique: true,
		JobFunc: func(ctx context.Context) context.Context {
			return auth.WithRefreshJobMeta(ctx, jobs.TriggerCronSchedule, "")
		},
	})
	var success int
	var failed int
	_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
		success, failed = deps.characterRefresher.RefreshAll(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if failed > 0 {
			stepper.Partial(fmt.Errorf("refresh completed with %d failures", failed))
		}
		return nil
	})
	runner.WithFields(logging.Fields{
		"success": success,
		"failed":  failed,
	})
	if deps.audit != nil {
		deps.audit.LogEvent(audit.Event{
			Action:  audit.ActionCharacterRefreshAll,
			Summary: fmt.Sprintf("Scheduled refresh (%d ok, %d failed)", success, failed),
		})
	}
}
