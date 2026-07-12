package admin

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/audit"
	"sentinel2/internal/auth"
	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

func (h *Handler) RefreshCharacter(c *core.RequestEvent) error {
	if h.Refresher == nil {
		return router.NewInternalServerError("Character refresh is unavailable.", nil)
	}

	id := c.Request.PathValue("id")
	record, recordErr := h.App.FindRecordById("characters", id)
	if recordErr != nil {
		return router.NewNotFoundError("Character not found.", logging.Fields{
			"character_record_id": id,
		})
	}

	userID := record.GetString("user")
	actorID := actorIDFromRequest(c)
	step := ""
	if userID != "" {
		pending, pendingErr := h.App.FindRecordsByFilter(
			"job_runs",
			"kind = {:kind} && step = {:step} && status = {:status}",
			"",
			1,
			0,
			dbx.Params{
				"kind":   jobs.JobCharacterRefresh,
				"step":   "character:" + record.Id,
				"status": jobs.StatusRunning,
			},
		)
		if pendingErr == nil && len(pending) > 0 {
			return router.NewBadRequestError("Character refresh already running.", logging.Fields{
				"character_record_id": id,
				"user_id":             userID,
			})
		}
		step = "user:" + userID
	}

	runner := h.newCharacterRefreshRunner(actorID, step, adminSingleRefreshTimeout)
	jobID := runner.JobID()

	go func(character *core.Record, jobID string) {
		_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
			return stepper.Run("character:"+character.Id, true, func(ctx context.Context) error {
				return h.Refresher.RefreshCharacter(ctx, character)
			})
		})
		logging.New(h.App).WithFields(logging.Fields{
			"job_id":              jobID,
			"character_record_id": character.Id,
			"character_id":        character.GetInt("eve_character_id"),
			"user_id":             character.GetString("user"),
		}).Info("single character refresh completed")
	}(record, jobID)

	h.logAction(
		c,
		&audit.Event{
			Action:          audit.ActionCharacterRefresh,
			Summary:         "Queued character refresh for " + record.GetString("eve_character_name") + " (job " + jobID + ")",
			TargetUserID:    record.GetString("user"),
			TargetCharacter: record,
		},
	)

	return c.JSON(http.StatusAccepted, map[string]any{
		"job_id":       jobID,
		"character_id": record.GetInt("eve_character_id"),
		"user_id":      record.GetString("user"),
	})
}

func (h *Handler) RefreshAllCharacters(c *core.RequestEvent) error {
	if h.Refresher == nil {
		return router.NewInternalServerError("Character refresh is unavailable.", nil)
	}

	userID, payloadErr := parseRefreshAllUserID(c)
	if payloadErr != nil {
		return payloadErr
	}
	user, resolveErr := h.resolveRefreshAllTarget(userID)
	if resolveErr != nil {
		return resolveErr
	}

	if pendingErr := h.ensureNoRunningUserRefresh(userID); pendingErr != nil {
		return pendingErr
	}
	filter, params, scope, step := refreshAllScope(userID)
	records, recordsErr := h.loadRefreshAllCharacters(filter, params, userID)
	if recordsErr != nil {
		return recordsErr
	}

	if pendingErr := h.ensureNoRunningCharacterRefreshes(userID, records); pendingErr != nil {
		return pendingErr
	}
	actorID := actorIDFromRequest(c)
	runner := h.newCharacterRefreshRunner(actorID, step, adminRefreshAllTimeout)
	jobID := runner.JobID()
	targetName := ""
	if user != nil {
		targetName = user.GetString("eve_character_name")
	}
	h.logRefreshAllQueued(c, userID, targetName, len(records), jobID)
	h.runRefreshAllAsync(records, userID, scope, runner, jobID)

	return c.JSON(http.StatusAccepted, map[string]any{
		"job_id": jobID,
		"scope":  scope,
	})
}

func (h *Handler) ResyncAccount(c *core.RequestEvent) error {
	if h.Provider == nil {
		return router.NewInternalServerError("Account resync is unavailable.", nil)
	}
	if h.currentAuthProvider(c) != auth.AuthProviderTestAuth {
		return router.NewNotFoundError("Not found.", nil)
	}

	userID := c.Request.PathValue("id")
	user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}
	if strings.ToLower(strings.TrimSpace(user.GetString("auth_provider"))) != auth.AuthProviderTestAuth {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}

	if _, refreshErr := h.Provider.Refresh(c.Request.Context(), user); refreshErr != nil {
		return router.NewInternalServerError("Failed to resync account.", logging.Fields{
			"user_id": userID,
			"error":   refreshErr.Error(),
		})
	}

	h.logAction(
		c,
		&audit.Event{
			Action:         audit.ActionUserResync,
			Summary:        "Resynced TestAuth account",
			TargetUserID:   userID,
			TargetUserName: user.GetString("eve_character_name"),
		},
	)

	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func parseRefreshAllUserID(c *core.RequestEvent) (string, error) {
	payload := struct {
		UserID string `json:"user_id"`
	}{}

	if c.Request.ContentLength > 0 {
		if bindErr := c.BindBody(&payload); bindErr != nil {
			return "", router.NewBadRequestError("Invalid payload.", logging.Fields{"error": bindErr})
		}
	}
	return strings.TrimSpace(payload.UserID), nil
}

func (h *Handler) resolveRefreshAllTarget(userID string) (*core.Record, error) {
	if userID == "" {
		return nil, nil
	}
	userRecord, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return nil, router.NewNotFoundError("User not found.", logging.Fields{"user_id": userID})
	}
	return userRecord, nil
}

func (h *Handler) ensureNoRunningUserRefresh(userID string) error {
	if userID == "" {
		return nil
	}
	pending, pendingErr := h.App.FindRecordsByFilter(
		"job_runs",
		"kind = {:kind} && step = {:step} && status = {:status}",
		"",
		1,
		0,
		dbx.Params{
			"kind":   jobs.JobCharacterRefresh,
			"step":   "user:" + userID,
			"status": jobs.StatusRunning,
		},
	)
	if pendingErr == nil && len(pending) > 0 {
		return router.NewBadRequestError("Character refresh already running for this user.", logging.Fields{"user_id": userID})
	}
	return nil
}

func refreshAllScope(userID string) (filter string, params dbx.Params, scope, step string) {
	if userID == "" {
		return "", dbx.Params{}, "all", ""
	}
	return "user = {:user}", dbx.Params{"user": userID}, "user", "user:" + userID
}

func (h *Handler) loadRefreshAllCharacters(filter string, params dbx.Params, userID string) ([]*core.Record, error) {
	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionCharacters, filter, "", 0, 0, params)
	if recordsErr != nil {
		return nil, router.NewInternalServerError("Failed to load characters.", logging.Fields{"user_id": userID})
	}
	return records, nil
}

func (h *Handler) ensureNoRunningCharacterRefreshes(userID string, records []*core.Record) error {
	if userID == "" || len(records) == 0 {
		return nil
	}
	const chunkSize = 20
	for start := 0; start < len(records); start += chunkSize {
		end := start + chunkSize
		end = min(end, len(records))
		clauses := make([]string, 0, end-start)
		params := dbx.Params{"kind": jobs.JobCharacterRefresh, "status": jobs.StatusRunning}
		for i, rec := range records[start:end] {
			key := fmt.Sprintf("step_%d", i)
			clauses = append(clauses, fmt.Sprintf("step = {:%s}", key))
			params[key] = "character:" + rec.Id
		}
		filter := fmt.Sprintf("kind = {:kind} && status = {:status} && (%s)", strings.Join(clauses, " || "))
		pending, pendingErr := h.App.FindRecordsByFilter("job_runs", filter, "", 1, 0, params)
		if pendingErr == nil && len(pending) > 0 {
			return router.NewBadRequestError("Character refresh already running for this user.", logging.Fields{"user_id": userID})
		}
	}
	return nil
}

func (h *Handler) newCharacterRefreshRunner(actorID, step string, timeout time.Duration) *jobs.Runner {
	return jobs.NewRunner(h.App, &jobs.RunOptions{
		JobName: "admin.character_refresh",
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobCharacterRefresh,
			Step:    step,
			Trigger: "admin.character_refresh",
			ActorID: actorID,
		},
		Timeout: timeout,
		JobFunc: func(ctx context.Context) context.Context {
			return auth.WithRefreshJobMeta(ctx, "admin.character_refresh", actorID)
		},
	})
}

func buildRefreshAllSummary(userID, targetName string, count int, jobID string) string {
	suffix := ""
	if count != 1 {
		suffix = "s"
	}
	summary := fmt.Sprintf("Queued refresh for %d character%s (job %s)", count, suffix, jobID)
	if userID != "" && targetName != "" {
		summary = fmt.Sprintf("Queued refresh for %s (%d character%s, job %s)", targetName, count, suffix, jobID)
	}
	return summary
}

func (h *Handler) logRefreshAllQueued(c *core.RequestEvent, userID, targetName string, count int, jobID string) {
	summary := buildRefreshAllSummary(userID, targetName, count, jobID)
	if userID != "" {
		event := audit.Event{
			Action:         audit.ActionCharacterRefreshAll,
			Summary:        summary,
			TargetUserID:   userID,
			TargetUserName: targetName,
		}
		h.applyAccountTarget(&event, userID, targetName)
		h.logAction(c, &event)
		return
	}
	h.logAction(c, &audit.Event{Action: audit.ActionCharacterRefreshAll, Summary: summary})
}

func runRefreshAllSteps(ctx context.Context, stepper jobs.Stepper, records []*core.Record, refresher *auth.CharacterRefresher) (success, failed int, err error) {
	runErr := jobs.RunBatched(ctx, records, refreshAllBatchSize, refreshAllBatchPause, func(_ context.Context, _ int, character *core.Record) error {
		if character == nil {
			return nil
		}
		stepSuccess, stepFailed := runSingleRefreshStep(stepper, character, refresher)
		success += stepSuccess
		failed += stepFailed
		return nil
	})
	if runErr != nil {
		return success, failed, runErr
	}
	return success, failed, nil
}

func runSingleRefreshStep(stepper jobs.Stepper, character *core.Record, refresher *auth.CharacterRefresher) (success, failed int) {
	stepName := "character:" + character.Id
	stepErr := stepper.Run(stepName, false, func(ctx context.Context) error {
		err := refresher.RefreshCharacter(ctx, character)
		if err != nil {
			failed++
		} else {
			success++
		}
		return err
	})
	if stepErr != nil {
		failed++
	}
	return success, failed
}

func (h *Handler) runRefreshAllAsync(records []*core.Record, userID, scope string, runner *jobs.Runner, jobID string) {
	go func() {
		start := time.Now()
		var success int
		var failed int
		_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
			stepSuccess, stepFailed, runErr := runRefreshAllSteps(ctx, stepper, records, h.Refresher)
			success, failed = stepSuccess, stepFailed
			if runErr != nil {
				return runErr
			}
			if failed > 0 {
				stepper.Partial(fmt.Errorf("refresh completed with %d failures", failed))
			}
			return nil
		})
		logging.New(h.App).WithFields(logging.Fields{
			"job_id":      jobID,
			"scope":       scope,
			"user_id":     userID,
			"success":     success,
			"failed":      failed,
			"duration_ms": time.Since(start).Milliseconds(),
		}).Info("character refresh completed")
	}()
}
