package server

import (
	"context"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/jobs"
	"sentinel2/internal/mapdata"
)

func runMapDataBootstrap(app *pocketbase.PocketBase) {
	runner := jobs.NewRunner(app, jobs.RunOptions{
		JobName: mapdata.JobMapDataUpdate,
		JobOptions: jobs.JobOptions{
			Kind:    "map_data_update",
			Trigger: jobs.TriggerCronSchedule,
		},
		Timeout: jobs.NoTimeout,
	})
	mapdata.RunMapDataUpdate(app, runner, jobs.TriggerCronSchedule, false)
}

func runMapDataCron(app *pocketbase.PocketBase, ctx context.Context) {
	runner := jobs.NewRunner(app, jobs.RunOptions{
		JobName: mapdata.JobMapDataUpdate,
		JobOptions: jobs.JobOptions{
			Kind:    "map_data_update",
			Trigger: jobs.TriggerCronSchedule,
		},
		Timeout: jobs.NoTimeout,
		Parent:  ctx,
	})
	mapdata.RunMapDataUpdateWithContext(ctx, app, runner, jobs.TriggerCronSchedule, false)
}
