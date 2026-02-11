package mapdata

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
)

func RunMapDataUpdate(app *pocketbase.PocketBase, runner *jobs.Runner, trigger string, force bool) {
	RunMapDataUpdateWithContext(context.Background(), app, runner, trigger, force)
}

func RunMapDataUpdateWithContext(ctx context.Context, app *pocketbase.PocketBase, runner *jobs.Runner, trigger string, force bool) {
	if force {
		runner.WithFields(logging.Fields{"forced": true})
	}

	err := runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
		importer := NewSDEImporter(app)
		var needsUpdate bool
		var etag string
		if force {
			if err := stepper.SkipStep("sde_check", "skipped on manual run"); err != nil {
				return err
			}
		} else {
			if err := stepper.Run("sde_check", true, func(ctx context.Context) error {
				checked, newTag, sdeErr := ShouldUpdateSDE(importer, 24*time.Hour)
				if sdeErr != nil {
					return sdeErr
				}
				needsUpdate = checked
				etag = newTag
				return nil
			}); err != nil {
				return err
			}
		}

		if !force && !needsUpdate {
			stepper.WithMessage(jobs.MessageUpdateNotNeeded)
			return stepper.SkipParent(jobs.MessageUpdateNotNeeded)
		}

		if err := stepper.Run(StepSDEImport, true, func(ctx context.Context) error {
			return importer.DownloadAndImport(ctx, etag)
		}); err != nil {
			return err
		}

		if err := stepper.Run(StepRealPositions, true, func(ctx context.Context) error {
			return CalculateRealPositions(ctx, app)
		}); err != nil {
			return err
		}

		if err := stepper.Run(StepEve2DPositions, true, func(ctx context.Context) error {
			return UpdateRegionPositionsFromSystems(app)
		}); err != nil {
			return err
		}

		if err := stepper.Run(StepDotlanImport, true, func(ctx context.Context) error {
			return DownloadDotlan(ctx, app)
		}); err != nil {
			return err
		}

		if err := stepper.Run(StepMetroPositions, true, func(ctx context.Context) error {
			if err := CalculateSystemGraphs(ctx, app); err != nil {
				return err
			}
			return CalculateRegionLayouts(ctx, app)
		}); err != nil {
			return err
		}
		return nil
	})

	_ = err
}
