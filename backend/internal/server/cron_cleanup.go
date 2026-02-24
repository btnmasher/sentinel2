package server

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/cleanup"
	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

type cleanupCounters struct {
	reportHashes     int
	intelUploaders   int
	uploaderSessions int
	uploaderTokens   int
	intelReports     int
	timers           int
	staleJobRuns     int
}

func runCleanupJob(app *pocketbase.PocketBase, deps *dependencies) {
	const staleJobTimeout = 30 * time.Minute

	runner := jobs.NewRunner(app, &jobs.RunOptions{
		JobName: jobs.JobCleanup,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobCleanup,
			Trigger: jobs.TriggerCronSchedule,
		},
		Timeout: jobs.NoTimeout,
	})
	_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
		counters, err := runCleanupSteps(app, deps, stepper, staleJobTimeout)
		if err != nil {
			return err
		}

		runner.WithFields(logging.Fields{
			"intel_report_hash_count": counters.reportHashes,
			"intel_uploaders_count":   counters.intelUploaders,
			"uploader_sessions_count": counters.uploaderSessions,
			"uploader_tokens_count":   counters.uploaderTokens,
			"intel_reports_count":     counters.intelReports,
			"timers_count":            counters.timers,
			"stale_job_runs_count":    counters.staleJobRuns,
		})

		return nil
	})
}

func runCleanupSteps(app *pocketbase.PocketBase, deps *dependencies, stepper jobs.Stepper, staleJobTimeout time.Duration) (cleanupCounters, error) {
	counters := cleanupCounters{}
	steps := []struct {
		name string
		run  func() (int, error)
		set  func(*cleanupCounters, int)
	}{
		{
			name: "cleanup_report_hashes",
			run:  func() (int, error) { return deps.cleanup.RemoveExpired(store.CollectionIntelReportHash) },
			set:  func(c *cleanupCounters, count int) { c.reportHashes = count },
		},
		{
			name: "cleanup_intel_uploaders",
			run:  func() (int, error) { return deps.cleanup.RemoveExpired(store.CollectionIntelUploaders) },
			set:  func(c *cleanupCounters, count int) { c.intelUploaders = count },
		},
		{
			name: "cleanup_uploader_sessions",
			run:  func() (int, error) { return deps.cleanup.RemoveExpired(store.CollectionUploaderSessions) },
			set:  func(c *cleanupCounters, count int) { c.uploaderSessions = count },
		},
		{
			name: "cleanup_uploader_tokens",
			run:  func() (int, error) { return deps.cleanup.RemoveRevokedUploaderTokens() },
			set:  func(c *cleanupCounters, count int) { c.uploaderTokens = count },
		},
		{
			name: "cleanup_intel_reports",
			run:  func() (int, error) { return deps.cleanup.RemoveOldIntelReports(cleanup.IntelReportRetention) },
			set:  func(c *cleanupCounters, count int) { c.intelReports = count },
		},
		{
			name: "cleanup_timers",
			run:  func() (int, error) { return deps.cleanup.RemoveOldTimers(cleanup.TimerInactiveRetention) },
			set:  func(c *cleanupCounters, count int) { c.timers = count },
		},
		{
			name: "cleanup_stale_job_runs",
			run:  func() (int, error) { return jobs.NewJobTracker(app).MarkStaleRunningAsTimeout(staleJobTimeout) },
			set:  func(c *cleanupCounters, count int) { c.staleJobRuns = count },
		},
	}
	for _, step := range steps {
		if err := stepper.Run(step.name, false, func(ctx context.Context) error {
			count, err := step.run()
			if err != nil {
				return err
			}
			step.set(&counters, count)
			return nil
		}); err != nil {
			return counters, err
		}
	}
	return counters, nil
}
