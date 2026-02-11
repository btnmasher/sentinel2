package mapdata

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
)

type StepTriggerOptions struct {
	Step    string
	Trigger string
	ActorID string
	JobName string
	Logger  *logging.Logger
}

func TriggerMapDataStep(app *pocketbase.PocketBase, opts StepTriggerOptions) string {
	jobName := opts.JobName
	if jobName == "" {
		jobName = JobMapDataStep
	}
	logger := opts.Logger
	if logger == nil {
		logger = logging.New(app)
	}

	runner := jobs.NewRunner(app, jobs.RunOptions{
		JobName: jobName,
		JobOptions: jobs.JobOptions{
			Kind:    "map_data_step",
			Step:    opts.Step,
			Trigger: opts.Trigger,
			ActorID: opts.ActorID,
		},
		Timeout: jobs.NoTimeout,
	})
	jobID := runner.JobID()
	logFields := logging.Fields{
		"job_id":        jobID,
		"trigger":       opts.Trigger,
		"map_data_step": opts.Step,
	}
	logger.WithFields(logFields).Info("map data step requested")

	go func(jobID string, step string, trigger string, actorID string) {
		start := time.Now()
		log := logging.New(app).WithFields(logFields)
		baseCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		localRunner := jobs.NewRunner(app, jobs.RunOptions{
			JobID: jobID,
			JobOptions: jobs.JobOptions{
				Kind:    "map_data_step",
				Step:    step,
				Trigger: trigger,
				ActorID: actorID,
			},
			Parent:  baseCtx,
			Timeout: jobs.NoTimeout,
		})
		err := localRunner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
			switch step {
			case StepSDEImport:
				importer := NewSDEImporter(app)
				return importer.DownloadAndImport(ctx, "")
			case StepRealPositions:
				return CalculateRealPositions(ctx, app)
			case StepEve2DPositions:
				return UpdateRegionPositionsFromSystems(app)
			case StepDotlanImport:
				return DownloadDotlan(ctx, app)
			case StepMetroPositions:
				if err := CalculateSystemGraphs(ctx, app); err != nil {
					return err
				}
				return CalculateRegionLayouts(ctx, app)
			case StepBuildGraph:
				return CalculateSystemGraphs(ctx, app)
			case StepRegionLayout:
				return CalculateRegionLayouts(ctx, app)
			default:
				log.Warn("unknown map data step")
				return nil
			}
		})

		if err != nil {
			log.WithFields(logging.Fields{
				"duration_ms": time.Since(start).Milliseconds(),
			}).WithErr(err).Error("map data step failed")
			return
		}
		log.WithFields(logging.Fields{
			"duration_ms": time.Since(start).Milliseconds(),
		}).Info("map data step completed")
	}(jobID, opts.Step, opts.Trigger, opts.ActorID)

	return jobID
}
