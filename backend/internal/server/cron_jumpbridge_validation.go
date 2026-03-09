package server

import (
	"context"
	"fmt"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
)

func registerJumpbridgeUpdateCron(app *pocketbase.PocketBase, deps *dependencies) {
	app.Cron().MustAdd("jumpbridge_update", "0 * * * *", func() {
		runJumpbridgeUpdate(app, deps)
	})
}

func runJumpbridgeUpdate(app *pocketbase.PocketBase, deps *dependencies) {
	if app == nil || deps == nil || deps.staffJumpbridges == nil || deps.staffJumpbridges.Service == nil {
		return
	}
	runner := jobs.NewRunner(app, &jobs.RunOptions{
		JobName: jobs.JobUpdateJumpbridges,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobUpdateJumpbridges,
			Trigger: jobs.TriggerCronSchedule,
			Hidden:  true,
		},
		Timeout: jobs.NoTimeout,
		Unique:  true,
	})
	log := logging.New(app)
	_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
		var validationSummary string
		if err := stepper.Run("validate_pairs", false, func(ctx context.Context) error {
			summary, err := deps.staffJumpbridges.Service.ValidateExistingPairsWithAllowedCharacters(ctx)
			if err != nil {
				return err
			}
			log.WithFields(logging.Fields{
				"character_ids": summary.CharacterIDs,
				"total_pairs":   summary.TotalPairs,
				"valid_pairs":   summary.ValidPairs,
				"invalid_pairs": summary.InvalidPairs,
				"skipped_pairs": summary.SkippedPairs,
				"skipped_orgs":  summary.SkippedOrganizations,
				"removed_pairs": summary.RemovedPairs,
				"removed_keys":  summary.RemovedKeys,
				"removed_names": summary.RemovedNames,
			}).Info("jumpbridge validation completed")
			validationSummary = fmt.Sprintf(
				"validation: removed %d invalid (of %d total; skipped orgs %d)",
				summary.RemovedPairs,
				summary.TotalPairs,
				summary.SkippedOrganizations,
			)
			runner.WithMessage(validationSummary)
			if summary.InvalidPairs > 0 {
				stepper.Partial(fmt.Errorf("jumpbridge validation found %d invalid pairs", summary.InvalidPairs))
			}
			return nil
		}); err != nil {
			return err
		}
		return stepper.Run("import_pairs", false, func(ctx context.Context) error {
			summary, err := deps.staffJumpbridges.Service.DiscoverAndImportAllowedSovPairs(ctx)
			if err != nil {
				return err
			}
			log.WithFields(logging.Fields{
				"character_ids":               summary.CharacterIDs,
				"allowed_sovereignty_systems": summary.AllowedSovereigntySystems,
				"candidate_pairs":             summary.CandidatePairs,
				"added_pairs":                 summary.AddedPairs,
				"upgraded_pairs":              summary.UpgradedPairs,
				"skipped_orgs":                summary.SkippedOrganizations,
				"added_keys":                  summary.AddedKeys,
				"added_names":                 summary.AddedNames,
				"upgraded_keys":               summary.UpgradedKeys,
				"upgraded_names":              summary.UpgradedNames,
				"skipped_pairs":               summary.SkippedPairs,
			}).Info("jumpbridge discovery import completed")
			runner.WithMessage(fmt.Sprintf(
				"%s; discovery: added %d, upgraded %d, skipped %d (candidates %d; skipped orgs %d)",
				validationSummary,
				summary.AddedPairs,
				summary.UpgradedPairs,
				summary.SkippedPairs,
				summary.CandidatePairs,
				summary.SkippedOrganizations,
			))
			return nil
		})
	})
}
