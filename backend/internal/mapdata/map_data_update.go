package mapdata

import (
	"context"
	"strings"
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
		return runMapDataPipeline(ctx, stepper, importer, service, etag)
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

func runMapDataPipeline(ctx context.Context, stepper jobs.Stepper, importer *SDEImporter, service *MapDataService, etag string) error {
	topologyUnchanged, err := runSDEImportStep(stepper, importer, etag)
	if err != nil {
		return err
	}

	if topologyUnchanged {
		skipTopologySteps(stepper)
	} else if err := runTopologySteps(stepper, service); err != nil {
		return err
	}

	if err := runOptionalStepFromLatestJSONL(ctx, stepper, importer, service, "mapPlanets.jsonl", StepPlanetsImport); err != nil {
		return err
	}
	if err := runOptionalStepFromLatestJSONL(ctx, stepper, importer, service, "mapMoons.jsonl", StepMoonsImport); err != nil {
		return err
	}

	if topologyUnchanged {
		_ = stepper.SkipStep(StepMetroPositions, "skipped (topology unchanged)")
		return nil
	}
	return stepper.Run(StepMetroPositions, true, func(ctx context.Context) error {
		return service.RunStep(ctx, StepMetroPositions)
	})
}

func runSDEImportStep(stepper jobs.Stepper, importer *SDEImporter, etag string) (bool, error) {
	topologyUnchanged := false
	err := stepper.Run(StepSDEImport, true, func(ctx context.Context) error {
		report, importErr := importer.DownloadAndImportWithReport(ctx, etag)
		if importErr != nil {
			return importErr
		}
		topologyUnchanged = allSDEFilesSkipped(report)
		for _, file := range report.Files {
			if !file.Skipped {
				continue
			}
			stepName := StepSDEImport + "." + strings.TrimSuffix(strings.TrimSpace(file.Name), ".jsonl")
			_ = stepper.SkipStep(stepName, file.Reason)
		}
		return nil
	})
	return topologyUnchanged, err
}

func skipTopologySteps(stepper jobs.Stepper) {
	_ = stepper.SkipStep(StepRealPositions, "skipped (topology unchanged)")
	_ = stepper.SkipStep(StepEve2DPositions, "skipped (topology unchanged)")
	_ = stepper.SkipStep(StepDotlanImport, "skipped (topology unchanged)")
}

func runOptionalStepFromLatestJSONL(
	ctx context.Context,
	stepper jobs.Stepper,
	importer *SDEImporter,
	service *MapDataService,
	filename, stepName string,
) error {
	shouldImport, skipReason, checkErr := importer.ShouldImportJSONLFromLatest(ctx, filename)
	if checkErr != nil {
		return checkErr
	}
	if !shouldImport {
		_ = stepper.SkipStep(stepName, skipReason)
		return nil
	}
	return stepper.Run(stepName, true, func(ctx context.Context) error {
		return service.RunStep(ctx, stepName)
	})
}

func runTopologySteps(stepper jobs.Stepper, service *MapDataService) error {
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
	return nil
}

func allSDEFilesSkipped(report SDEImportReport) bool {
	if len(report.Files) == 0 {
		return false
	}
	for _, file := range report.Files {
		if !file.Skipped {
			return false
		}
	}
	return true
}
