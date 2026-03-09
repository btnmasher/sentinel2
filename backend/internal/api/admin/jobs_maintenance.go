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

func (h *Handler) RunUpdateJumpbridgesJob(c *core.RequestEvent) error {
	if h.Jumpbridges == nil {
		return router.NewInternalServerError("Jumpbridge service unavailable.", nil)
	}
	actorID := actorIDFromRequest(c)
	runner := jobs.NewRunner(h.App, &jobs.RunOptions{
		JobName: jobs.JobUpdateJumpbridges,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobUpdateJumpbridges,
			Trigger: jobs.TriggerAdminManual,
			ActorID: actorID,
		},
		Timeout: jobs.NoTimeout,
		Unique:  true,
	})
	jobID := runner.JobID()

	h.runUpdateJumpbridgesAsync(runner)

	h.logAction(
		c,
		&audit.Event{
			Action:      audit.ActionJobJumpbridgeUpdateRun,
			Summary:     fmt.Sprintf("Triggered jumpbridge update job %s", jobID),
			TargetType:  audit.TargetTypeJob,
			TargetID:    jobID,
			TargetLabel: jobs.JobUpdateJumpbridges,
			TargetMeta: map[string]any{
				"job_id": jobID,
				"kind":   jobs.JobUpdateJumpbridges,
			},
		},
	)

	return c.JSON(http.StatusAccepted, map[string]any{"job_id": jobID})
}

func (h *Handler) runUpdateJumpbridgesAsync(runner *jobs.Runner) {
	go func() {
		_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
			validationSummary, err := h.runUpdateJumpbridgesValidationStep(stepper, runner)
			if err != nil {
				return err
			}
			return h.runUpdateJumpbridgesImportStep(stepper, runner, validationSummary)
		})
	}()
}

func (h *Handler) runUpdateJumpbridgesValidationStep(stepper jobs.Stepper, runner *jobs.Runner) (string, error) {
	validationSummary := ""
	err := stepper.Run("validate_pairs", false, func(ctx context.Context) error {
		summary, err := h.Jumpbridges.ValidateExistingPairsWithAllowedCharacters(ctx)
		if err != nil {
			return err
		}
		runner.WithFields(logging.Fields{
			"validation_character_ids": summary.CharacterIDs,
			"total_pairs":              summary.TotalPairs,
			"valid_pairs":              summary.ValidPairs,
			"invalid_pairs":            summary.InvalidPairs,
			"skipped_pairs":            summary.SkippedPairs,
			"skipped_organizations":    summary.SkippedOrganizations,
			"removed_pairs":            summary.RemovedPairs,
			"removed_keys":             summary.RemovedKeys,
			"removed_names":            summary.RemovedNames,
		})
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
	})
	return validationSummary, err
}

func (h *Handler) runUpdateJumpbridgesImportStep(stepper jobs.Stepper, runner *jobs.Runner, validationSummary string) error {
	return stepper.Run("import_pairs", false, func(ctx context.Context) error {
		summary, err := h.Jumpbridges.DiscoverAndImportAllowedSovPairs(ctx)
		if err != nil {
			return err
		}
		runner.WithFields(logging.Fields{
			"discovery_character_ids":   summary.CharacterIDs,
			"allowed_sov_systems":       summary.AllowedSovereigntySystems,
			"discovery_candidate_pairs": summary.CandidatePairs,
			"discovery_added_pairs":     summary.AddedPairs,
			"discovery_upgraded_pairs":  summary.UpgradedPairs,
			"discovery_skipped_orgs":    summary.SkippedOrganizations,
			"discovery_added_keys":      summary.AddedKeys,
			"discovery_added_names":     summary.AddedNames,
			"discovery_upgraded_keys":   summary.UpgradedKeys,
			"discovery_upgraded_names":  summary.UpgradedNames,
			"discovery_skipped_pairs":   summary.SkippedPairs,
		})
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
