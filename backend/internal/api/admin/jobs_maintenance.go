package admin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/audit"
	"sentinel2/internal/cleanup"
	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

func (h *Handler) RunCleanupJob(c *core.RequestEvent) error {
	actorID := actorIDFromRequest(c)

	runner := jobs.NewRunner(h.App, &jobs.RunOptions{
		JobName: jobs.JobCleanup,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobCleanup,
			Trigger: jobs.TriggerAdminManual,
			ActorID: actorID,
		},
		Timeout: jobs.NoTimeout,
	})

	jobID := runner.JobID()
	h.runCleanupAsync(runner)
	targetUserName := ""
	if c.Auth != nil {
		targetUserName = c.Auth.GetString("eve_character_name")
	}
	h.logAction(
		c,
		&audit.Event{
			Action:         audit.ActionJobCleanupRun,
			Summary:        fmt.Sprintf("Triggered cleanup job %s", jobID),
			TargetUserID:   actorID,
			TargetUserName: targetUserName,
			TargetType:     audit.TargetTypeJob,
			TargetID:       jobID,
			TargetLabel:    "cleanup",
			TargetMeta: map[string]any{
				"job_id": jobID,
			},
		},
	)

	return c.JSON(http.StatusAccepted, map[string]any{
		"job_id": jobID,
	})
}

func (h *Handler) RunSovereigntyCampaignSyncJob(c *core.RequestEvent) error {
	if h.Timers == nil {
		return router.NewInternalServerError("Timer service unavailable.", nil)
	}
	actorID := actorIDFromRequest(c)
	runner := jobs.NewRunner(h.App, &jobs.RunOptions{
		JobName: jobs.JobSovCampaignSync,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobSovCampaignSync,
			Trigger: jobs.TriggerAdminManual,
			ActorID: actorID,
		},
		Timeout: adminSovSyncTimeout,
		Unique:  true,
	})
	jobID := runner.JobID()

	go func() {
		_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
			result, err := h.Timers.SyncSovereigntyCampaignTimers(ctx)
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
	}()

	h.logAction(
		c,
		&audit.Event{
			Action:      audit.ActionJobSovCampaignSyncRun,
			Summary:     fmt.Sprintf("Triggered sovereignty campaign sync job %s", jobID),
			TargetType:  audit.TargetTypeJob,
			TargetID:    jobID,
			TargetLabel: jobs.JobSovCampaignSync,
			TargetMeta: map[string]any{
				"job_id": jobID,
				"kind":   jobs.JobSovCampaignSync,
			},
		},
	)

	return c.JSON(http.StatusAccepted, map[string]any{"job_id": jobID})
}

func (h *Handler) RunStructureNotificationsSyncJob(c *core.RequestEvent) error {
	if h.Timers == nil {
		return router.NewInternalServerError("Timer service unavailable.", nil)
	}
	sourceCount, sourceCountErr := h.Timers.NotificationSourceCount()
	if sourceCountErr != nil {
		return router.NewInternalServerError("Failed to validate notification source eligibility.", logging.Fields{"error": sourceCountErr.Error()})
	}
	actorID := actorIDFromRequest(c)
	runner := jobs.NewRunner(h.App, &jobs.RunOptions{
		JobName: jobs.JobSkyhookSync,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobSkyhookSync,
			Trigger: jobs.TriggerAdminManual,
			ActorID: actorID,
		},
		Timeout: adminSkyhookSyncTimeout,
		Unique:  true,
	})
	jobID := runner.JobID()

	go func() {
		_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
			if sourceCount == 0 {
				return stepper.SkipParent("no eligible notification sources")
			}
			result, err := h.Timers.SyncSkyhookNotifications(ctx, adminSkyhookSyncWindow)
			if err != nil {
				return err
			}
			runner.WithFields(logging.Fields{
				"watched_characters": result.WatchedCharacters,
				"notifications_seen": result.NotificationsSeen,
				"intel_created":      result.IntelCreated,
				"timers_created":     result.TimersCreated,
				"timers_updated":     result.TimersUpdated,
				"skipped":            result.Skipped,
			})
			return nil
		})
	}()

	h.logAction(
		c,
		&audit.Event{
			Action:      audit.ActionJobSkyhookSyncRun,
			Summary:     fmt.Sprintf("Triggered structure notifications sync job %s", jobID),
			TargetType:  audit.TargetTypeJob,
			TargetID:    jobID,
			TargetLabel: jobs.JobSkyhookSync,
			TargetMeta: map[string]any{
				"job_id": jobID,
				"kind":   jobs.JobSkyhookSync,
			},
		},
	)

	return c.JSON(http.StatusAccepted, map[string]any{"job_id": jobID})
}

func actorIDFromRequest(c *core.RequestEvent) string {
	if c != nil && c.Auth != nil {
		return c.Auth.Id
	}
	return ""
}

func (h *Handler) runCleanupAsync(runner *jobs.Runner) {
	go func() {
		_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
			counts, runErr := h.runCleanupSteps(stepper)
			if runErr != nil {
				return runErr
			}
			runner.WithFields(logging.Fields{
				"intel_report_hash_count": counts.reportHashCount,
				"intel_uploaders_count":   counts.intelUploaderCount,
				"uploader_tokens_count":   counts.uploaderTokenCount,
				"intel_reports_count":     counts.intelReportCount,
				"timers_count":            counts.timerCount,
				"stale_job_runs_count":    counts.staleJobRunCount,
			})
			return nil
		})
	}()
}

func (h *Handler) runCleanupSteps(stepper jobs.Stepper) (cleanupCounts, error) {
	counts := cleanupCounts{}
	if err := stepper.Run("cleanup_report_hashes", false, func(ctx context.Context) error {
		count, err := h.Cleanup.RemoveExpired(store.CollectionIntelReportHash)
		if err == nil {
			counts.reportHashCount = count
		}
		return err
	}); err != nil {
		return counts, err
	}
	if err := stepper.Run("cleanup_intel_uploaders", false, func(ctx context.Context) error {
		count, err := h.Cleanup.RemoveExpired(store.CollectionIntelUploaders)
		counts.intelUploaderCount = count
		return err
	}); err != nil {
		return counts, err
	}
	if err := stepper.Run("cleanup_uploader_tokens", false, func(ctx context.Context) error {
		count, err := h.Cleanup.RemoveRevokedUploaderTokens()
		counts.uploaderTokenCount = count
		return err
	}); err != nil {
		return counts, err
	}
	if err := stepper.Run("cleanup_intel_reports", false, func(ctx context.Context) error {
		count, err := h.Cleanup.RemoveOldIntelReports(cleanup.IntelReportRetention)
		counts.intelReportCount = count
		return err
	}); err != nil {
		return counts, err
	}
	if err := stepper.Run("cleanup_timers", false, func(ctx context.Context) error {
		count, err := h.Cleanup.RemoveOldTimers(cleanup.TimerInactiveRetention)
		counts.timerCount = count
		return err
	}); err != nil {
		return counts, err
	}
	if err := stepper.Run("cleanup_stale_job_runs", false, func(ctx context.Context) error {
		count, err := jobs.NewJobTracker(h.App).MarkStaleRunningAsTimeout(staleJobRunTimeout)
		counts.staleJobRunCount = count
		return err
	}); err != nil {
		return counts, err
	}
	return counts, nil
}
