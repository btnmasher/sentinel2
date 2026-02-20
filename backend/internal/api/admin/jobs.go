package admin

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/audit"
	"sentinel2/internal/auth"
	"sentinel2/internal/cleanup"
	"sentinel2/internal/format"
	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
	"sentinel2/internal/shared/collections"
	"sentinel2/internal/shared/pagination"
	"sentinel2/internal/shared/queryhelpers"
	"sentinel2/internal/store"
)

type jobRunsOptions struct {
	page         int
	limit        int
	excludeKinds []string
	startAt      *time.Time
	endAt        *time.Time
}

type cleanupCounts struct {
	reportHashCount    int
	intelUploaderCount int
	uploaderTokenCount int
	intelReportCount   int
	timerCount         int
}

const (
	defaultJobRunsLimit       = 20
	maxJobRunsLimit           = 100
	adminSingleRefreshTimeout = 2 * time.Minute
	adminRefreshAllTimeout    = 5 * time.Minute
	refreshAllBatchSize       = 25
	refreshAllBatchPause      = 350 * time.Millisecond
)

func parseJobRunsOptions(c *core.RequestEvent) (jobRunsOptions, error) {
	values := c.Request.URL.Query()
	startAt, endAt, err := parseJobRunRange(c)
	if err != nil {
		return jobRunsOptions{}, err
	}
	return jobRunsOptions{
		page:         format.GetPositiveInt(values, "page", 1, 0),
		limit:        format.GetPositiveInt(values, "limit", defaultJobRunsLimit, maxJobRunsLimit),
		excludeKinds: format.GetQueryList(values, "kinds"),
		startAt:      startAt,
		endAt:        endAt,
	}, nil
}

func validateJobRunsRange(startAt, endAt *time.Time) error {
	if startAt != nil && endAt != nil && startAt.After(*endAt) {
		return router.NewBadRequestError("Start date must be before end date.", logging.Fields{
			"startAt": startAt.Format(time.RFC3339),
			"endAt":   endAt.Format(time.RFC3339),
		})
	}
	return nil
}

func buildJobRunsScanFilter(startAt, endAt *time.Time, excludeKinds []string) (string, dbx.Params) {
	scanParams := dbx.Params{}
	dateFilter := ""
	if startAt != nil {
		dateFilter = queryhelpers.AppendAnd(dateFilter, fmt.Sprintf("started_at >= %q", startAt.UTC().Format(time.RFC3339)))
	}
	if endAt != nil {
		dateFilter = queryhelpers.AppendAnd(dateFilter, fmt.Sprintf("started_at <= %q", endAt.UTC().Format(time.RFC3339)))
	}
	kindFilter := ""
	if len(excludeKinds) > 0 {
		clauses := make([]string, 0, len(excludeKinds))
		for i, kind := range excludeKinds {
			if kind == "" {
				continue
			}
			key := fmt.Sprintf("kind_%d", i)
			clauses = append(clauses, fmt.Sprintf("kind != {:%s}", key))
			scanParams[key] = kind
		}
		if len(clauses) > 0 {
			kindFilter = strings.Join(clauses, " && ")
		}
	}

	hiddenFilter := `(hidden = false || hidden = null)`
	parentFilter := `(step = "" || step = null || kind = "map_data_step" || (kind = "character_refresh" && step ~ "user:%"))`
	scanFilter := parentFilter
	if dateFilter != "" {
		scanFilter = queryhelpers.AppendAnd(scanFilter, dateFilter)
	}
	if kindFilter != "" {
		scanFilter = queryhelpers.AppendAnd(scanFilter, kindFilter)
	}
	scanFilter = queryhelpers.AppendAnd(scanFilter, hiddenFilter)
	return scanFilter, scanParams
}

func (h *Handler) collectJobIDOrder(scanFilter string, scanParams dbx.Params, targetCount int) ([]string, error) {
	const scanBatch = 200
	jobIDOrder := []string{}
	seen := map[string]struct{}{}
	recordOffset := 0
	for {
		records, err := h.App.FindRecordsByFilter("job_runs", scanFilter, "-started_at", scanBatch, recordOffset, scanParams)
		if err != nil {
			return nil, router.NewInternalServerError("Failed to load jobs.", logging.Fields{"error": err})
		}
		if len(records) == 0 {
			break
		}
		if appendJobIDs(records, seen, &jobIDOrder, targetCount) {
			break
		}
		recordOffset += len(records)
	}
	return jobIDOrder, nil
}

func appendJobIDs(records []*core.Record, seen map[string]struct{}, jobIDOrder *[]string, targetCount int) bool {
	for _, record := range records {
		jobID := record.GetString("job_id")
		if jobID == "" {
			continue
		}
		if collections.AppendUnique(jobIDOrder, seen, jobID) && len(*jobIDOrder) >= targetCount {
			return true
		}
	}
	return len(*jobIDOrder) >= targetCount
}

func paginateJobIDs(jobIDOrder []string, offset, limit int) ([]string, bool) {
	return pagination.SliceByOffsetLimit(jobIDOrder, offset, limit)
}

func buildJobIDFilter(jobIDs []string) (string, dbx.Params) {
	filter, params := queryhelpers.BuildOrEqualsFilter("job_id", jobIDs)
	return queryhelpers.AppendAnd(filter, `(hidden = false || hidden = null)`), params
}

func groupJobRunRecords(records []*core.Record, jobIDs []string) (map[string]*jobRunGroup, []jobRunGroup, time.Time) {
	groups := initJobRunGroups(jobIDs)
	var latest time.Time
	for _, record := range records {
		if entry, ok := mapRecordToJobRunEntry(record, groups); ok {
			updateLatestFromEntryTimes(&latest, entry.StartedAt, entry.CompletedAt)
		}
	}
	ordered := orderJobRunGroups(groups, jobIDs)
	return groups, ordered, latest
}

func initJobRunGroups(jobIDs []string) map[string]*jobRunGroup {
	groups := map[string]*jobRunGroup{}
	for _, jobID := range jobIDs {
		groups[jobID] = &jobRunGroup{Steps: []jobRunEntry{}}
	}
	return groups
}

func mapRecordToJobRunEntry(record *core.Record, groups map[string]*jobRunGroup) (jobRunEntry, bool) {
	if record.GetBool("hidden") {
		return jobRunEntry{}, false
	}
	group := groups[record.GetString("job_id")]
	if group == nil {
		return jobRunEntry{}, false
	}
	entry := toJobRunEntry(record)
	if entry.Step == "" {
		group.Parent = entry
	} else {
		group.Steps = append(group.Steps, entry)
	}
	return entry, true
}

func updateLatestFromEntryTimes(latest *time.Time, startedAt, completedAt string) {
	if latest == nil {
		return
	}
	apply := func(raw string) {
		if raw == "" {
			return
		}
		parsed, err := time.Parse(time.RFC3339, raw)
		if err == nil && parsed.After(*latest) {
			*latest = parsed
		}
	}
	apply(startedAt)
	apply(completedAt)
}

func orderJobRunGroups(groups map[string]*jobRunGroup, jobIDs []string) []jobRunGroup {
	ordered := make([]jobRunGroup, 0, len(jobIDs))
	for _, jobID := range jobIDs {
		group := groups[jobID]
		if group == nil {
			continue
		}
		sort.Slice(group.Steps, func(i, j int) bool {
			return group.Steps[i].StartedAt > group.Steps[j].StartedAt
		})
		if group.Parent.JobID == "" && len(group.Steps) > 0 {
			group.Parent = group.Steps[0]
			group.Steps = group.Steps[1:]
		}
		ordered = append(ordered, *group)
	}
	return ordered
}

func computeJobRunsETag(excludeKinds, jobIDs []string, groups map[string]*jobRunGroup) string {
	etagHasher := fnv.New64a()
	_, _ = etagHasher.Write([]byte(strings.Join(excludeKinds, ",")))
	_, _ = etagHasher.Write([]byte{0})
	for _, jobID := range jobIDs {
		group := groups[jobID]
		if group == nil {
			continue
		}
		parent := group.Parent
		_, _ = etagHasher.Write([]byte(parent.JobID))
		_, _ = etagHasher.Write([]byte{0})
		_, _ = etagHasher.Write([]byte(parent.Kind))
		_, _ = etagHasher.Write([]byte{0})
		_, _ = etagHasher.Write([]byte(parent.Step))
		_, _ = etagHasher.Write([]byte{0})
		_, _ = etagHasher.Write([]byte(parent.Status))
		_, _ = etagHasher.Write([]byte{0})
		_, _ = etagHasher.Write([]byte(parent.StartedAt))
		_, _ = etagHasher.Write([]byte{0})
		_, _ = etagHasher.Write([]byte(parent.CompletedAt))
		_, _ = etagHasher.Write([]byte{0})
		_, _ = etagHasher.Write([]byte(parent.Error))
		_, _ = etagHasher.Write([]byte{0})
		for stepIndex := range group.Steps {
			step := &group.Steps[stepIndex]
			_, _ = etagHasher.Write([]byte(step.JobID))
			_, _ = etagHasher.Write([]byte{0})
			_, _ = etagHasher.Write([]byte(step.Kind))
			_, _ = etagHasher.Write([]byte{0})
			_, _ = etagHasher.Write([]byte(step.Step))
			_, _ = etagHasher.Write([]byte{0})
			_, _ = etagHasher.Write([]byte(step.Status))
			_, _ = etagHasher.Write([]byte{0})
			_, _ = etagHasher.Write([]byte(step.StartedAt))
			_, _ = etagHasher.Write([]byte{0})
			_, _ = etagHasher.Write([]byte(step.CompletedAt))
			_, _ = etagHasher.Write([]byte{0})
			_, _ = etagHasher.Write([]byte(step.Error))
			_, _ = etagHasher.Write([]byte{0})
		}
	}
	return fmt.Sprintf(`W/"jobruns-%x"`, etagHasher.Sum64())
}

func parseJobRunRange(c *core.RequestEvent) (startAt, endAt *time.Time, err error) {
	startAt, endAt, err = queryhelpers.ParseFlexibleDateRangeUTC(
		c.Request.URL.Query(),
		"startAt",
		"endAt",
		"startDate",
		"endDate",
	)
	if err == nil {
		return startAt, endAt, nil
	}
	var parseErr *queryhelpers.DateRangeParseError
	if !errors.As(err, &parseErr) || parseErr == nil {
		return nil, nil, router.NewBadRequestError("Invalid date range.", logging.Fields{"error": err.Error()})
	}
	switch parseErr.Field {
	case "startAt":
		return nil, nil, router.NewBadRequestError("Invalid start time.", logging.Fields{"startAt": parseErr.Value})
	case "endAt":
		return nil, nil, router.NewBadRequestError("Invalid end time.", logging.Fields{"endAt": parseErr.Value})
	case "startDate":
		return nil, nil, router.NewBadRequestError("Invalid start date.", logging.Fields{"startDate": parseErr.Value})
	case "endDate":
		return nil, nil, router.NewBadRequestError("Invalid end date.", logging.Fields{"endDate": parseErr.Value})
	default:
		return nil, nil, router.NewBadRequestError("Invalid date range.", logging.Fields{"error": err.Error()})
	}
}

func toJobRunEntry(record *core.Record) jobRunEntry {
	started := record.GetDateTime("started_at")
	completed := record.GetDateTime("completed_at")
	startedAt := ""
	completedAt := ""
	if !started.IsZero() {
		startedAt = started.Time().Format(time.RFC3339)
	}
	if !completed.IsZero() {
		completedAt = completed.Time().Format(time.RFC3339)
	}
	durationMs := format.CoerceInt64(record.Get("duration_ms"))
	return jobRunEntry{
		ID:               record.Id,
		JobID:            record.GetString("job_id"),
		Kind:             record.GetString("kind"),
		Step:             record.GetString("step"),
		Trigger:          record.GetString("trigger"),
		Status:           record.GetString("status"),
		ActorID:          record.GetString("actor_id"),
		ActorDisplayName: record.GetString("actor_display_name"),
		StartedAt:        startedAt,
		CompletedAt:      completedAt,
		DurationMs:       durationMs,
		Error:            record.GetString("error"),
	}
}

func (h *Handler) JobRuns(c *core.RequestEvent) error {
	opts, optionsErr := parseJobRunsOptions(c)
	if optionsErr != nil {
		return optionsErr
	}
	if rangeErr := validateJobRunsRange(opts.startAt, opts.endAt); rangeErr != nil {
		return rangeErr
	}
	offset := pagination.OffsetForPage(opts.page, opts.limit)
	targetCount := offset + opts.limit + 1
	scanFilter, scanParams := buildJobRunsScanFilter(opts.startAt, opts.endAt, opts.excludeKinds)
	jobIDOrder, collectErr := h.collectJobIDOrder(scanFilter, scanParams, targetCount)
	if collectErr != nil {
		return collectErr
	}
	jobIDs, hasMore := paginateJobIDs(jobIDOrder, offset, opts.limit)
	if len(jobIDs) == 0 {
		return c.JSON(http.StatusOK, map[string]any{
			"jobs":    []jobRunGroup{},
			"page":    opts.page,
			"limit":   opts.limit,
			"hasMore": hasMore,
		})
	}
	filter, params := buildJobIDFilter(jobIDs)
	records, recordsErr := h.App.FindRecordsByFilter("job_runs", filter, "", 0, 0, params)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to load jobs.", logging.Fields{
			"error": recordsErr,
		})
	}
	groups, ordered, latest := groupJobRunRecords(records, jobIDs)
	etag := computeJobRunsETag(opts.excludeKinds, jobIDs, groups)
	if match := c.Request.Header.Get("If-None-Match"); match != "" && match == etag {
		return c.NoContent(http.StatusNotModified)
	}
	c.Response.Header().Set("ETag", etag)
	if !latest.IsZero() {
		c.Response.Header().Set("Last-Modified", latest.UTC().Format(time.RFC1123))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"jobs":    ordered,
		"page":    opts.page,
		"limit":   opts.limit,
		"hasMore": hasMore,
	})
}

func (h *Handler) RefreshCharacter(c *core.RequestEvent) error {
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
func actorIDFromRequest(c *core.RequestEvent) string {
	if c != nil && c.Auth != nil {
		return c.Auth.Id
	}
	return ""
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
	for idx, character := range records {
		if ctx.Err() != nil {
			return success, failed, ctx.Err()
		}
		if character == nil {
			continue
		}
		stepSuccess, stepFailed := runSingleRefreshStep(stepper, character, refresher)
		success += stepSuccess
		failed += stepFailed
		if waitErr := waitRefreshBatchBoundary(ctx, idx+1, refreshAllBatchSize, refreshAllBatchPause); waitErr != nil {
			return success, failed, waitErr
		}
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

func waitRefreshBatchBoundary(ctx context.Context, index, batchSize int, pause time.Duration) error {
	if index%batchSize != 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(pause):
		return nil
	}
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
	return counts, nil
}
