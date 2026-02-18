package mapdata

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
)

const sdeMaxAge = 24 * time.Hour

func RunMapDataUpdate(app *pocketbase.PocketBase, runner *jobs.Runner, trigger string, force bool) {
	RunMapDataUpdateWithContext(context.Background(), app, runner, trigger, force)
}

func RunMapDataUpdateWithContext(ctx context.Context, app *pocketbase.PocketBase, runner *jobs.Runner, trigger string, force bool) {
	service := NewMapDataService(app, nil)
	if force {
		runner.WithFields(logging.Fields{"forced": true})
	}

	//nolint:contextcheck // runner manages lifecycle context and exposes it to each step callback.
	err := runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
		importer := NewSDEImporter(app)
		needsUpdate, etag, checkErr := runSDECheckStep(stepper, importer, force)
		if checkErr != nil {
			return checkErr
		}
		if !force && !needsUpdate {
			stepper.WithMessage(jobs.MessageUpdateNotNeeded)
			return stepper.SkipParent(jobs.MessageUpdateNotNeeded)
		}
		return runMapDataPipeline(stepper, importer, service, etag)
	})

	_ = err
}

func runSDECheckStep(stepper jobs.Stepper, importer *SDEImporter, force bool) (needsUpdate bool, etag string, err error) {
	if force {
		err = stepper.SkipStep("sde_check", "skipped on manual run")
		return false, "", err
	}
	err = stepper.Run("sde_check", true, func(ctx context.Context) error {
		checked, newTag, sdeErr := ShouldUpdateSDE(ctx, importer, sdeMaxAge)
		if sdeErr != nil {
			return sdeErr
		}
		needsUpdate = checked
		etag = newTag
		return nil
	})
	return needsUpdate, etag, err
}

func runMapDataPipeline(stepper jobs.Stepper, importer *SDEImporter, service *MapDataService, etag string) error {
	if err := stepper.Run(StepSDEImport, true, func(ctx context.Context) error {
		return importer.DownloadAndImport(ctx, etag)
	}); err != nil {
		return err
	}

	if err := stepper.Run(StepRealPositions, true, func(ctx context.Context) error {
		return service.RunStep(ctx, StepRealPositions)
	}); err != nil {
		return err
	}

	if err := stepper.Run(StepEve2DPositions, true, func(ctx context.Context) error {
		return service.RunStep(ctx, StepEve2DPositions)
	}); err != nil {
		return err
	}

	if err := stepper.Run(StepDotlanImport, true, func(ctx context.Context) error {
		return service.RunStep(ctx, StepDotlanImport)
	}); err != nil {
		return err
	}

	if err := stepper.Run(StepMetroPositions, true, func(ctx context.Context) error {
		return service.RunStep(ctx, StepMetroPositions)
	}); err != nil {
		return err
	}
	return nil
}
