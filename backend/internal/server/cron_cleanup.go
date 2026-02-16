package server

import (
	"context"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/cleanup"
	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

func runCleanupJob(app *pocketbase.PocketBase, deps dependencies) {
	runner := jobs.NewRunner(app, jobs.RunOptions{
		JobName: jobs.JobCleanup,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobCleanup,
			Trigger: jobs.TriggerCronSchedule,
		},
		Timeout: jobs.NoTimeout,
	})
	_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
		var reportHashCount int
		var intelUploaderCount int
		var uploaderSessionCount int
		var uploaderTokenCount int
		var intelReportCount int

		if err := stepper.Run("cleanup_report_hashes", false, func(ctx context.Context) error {
			count, err := deps.cleanup.RemoveExpired(store.CollectionIntelReportHash)
			if err == nil {
				reportHashCount = count
			}
			return err
		}); err != nil {
			return err
		}

		if err := stepper.Run("cleanup_intel_uploaders", false, func(ctx context.Context) error {
			count, err := deps.cleanup.RemoveExpired(store.CollectionIntelUploaders)
			intelUploaderCount = count
			return err
		}); err != nil {
			return err
		}

		if err := stepper.Run("cleanup_uploader_sessions", false, func(ctx context.Context) error {
			count, err := deps.cleanup.RemoveExpired(store.CollectionUploaderSessions)
			uploaderSessionCount = count
			return err
		}); err != nil {
			return err
		}

		if err := stepper.Run("cleanup_uploader_tokens", false, func(ctx context.Context) error {
			count, err := deps.cleanup.RemoveRevokedUploaderTokens()
			uploaderTokenCount = count
			return err
		}); err != nil {
			return err
		}

		if err := stepper.Run("cleanup_intel_reports", false, func(ctx context.Context) error {
			count, err := deps.cleanup.RemoveOldIntelReports(cleanup.IntelReportRetention)
			intelReportCount = count
			return err
		}); err != nil {
			return err
		}

		runner.WithFields(logging.Fields{
			"intel_report_hash_count": reportHashCount,
			"intel_uploaders_count":   intelUploaderCount,
			"uploader_sessions_count": uploaderSessionCount,
			"uploader_tokens_count":   uploaderTokenCount,
			"intel_reports_count":     intelReportCount,
		})

		return nil
	})
}
