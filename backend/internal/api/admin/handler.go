package admin

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/auth"
	"sentinel2/internal/cleanup"
	"sentinel2/internal/format"
	"sentinel2/internal/intel"
	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
	"sentinel2/internal/middleware"
	"sentinel2/internal/store"
)

func NewHandler(app *pocketbase.PocketBase, refresher *auth.CharacterRefresher, provider *auth.EVEProvider, cleanupSvc *cleanup.Service, intelSvc *intel.IntelService) *Handler {
	return &Handler{
		App:       app,
		Refresher: refresher,
		Provider:  provider,
		Cleanup:   cleanupSvc,
		Intel:     intelSvc,
	}
}

func (h *Handler) Search(c *core.RequestEvent) error {
	query := strings.TrimSpace(c.Request.URL.Query().Get("q"))
	if len(query) < 2 {
		return c.JSON(http.StatusOK, map[string]any{"results": []searchItem{}})
	}

	filter := "eve_character_name ~ {:q}"
	params := dbx.Params{"q": "%" + query + "%"}
	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionCharacters, filter, "eve_character_name", 20, 0, params)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to search characters.", logging.Fields{
			"query": query,
		})
	}

	mainNames := map[string]string{}
	results := []searchItem{}
	for _, rec := range records {
		userID := rec.GetString("user")
		if userID == "" {
			continue
		}
		mainName := mainNames[userID]
		if mainName == "" {
			main, _ := h.findMainCharacter(userID)
			if main != nil {
				mainName = main.GetString("eve_character_name")
				mainNames[userID] = mainName
			}
		}
		results = append(results, searchItem{
			CharacterRecordID: rec.Id,
			CharacterID:       rec.GetInt("eve_character_id"),
			Name:              rec.GetString("eve_character_name"),
			UserID:            userID,
			IsMain:            rec.GetBool("is_main"),
			MainName:          mainName,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{"results": results})
}

func (h *Handler) AuditLogs(c *core.RequestEvent) error {
	userID := strings.TrimSpace(c.Request.URL.Query().Get("user_id"))
	action := strings.TrimSpace(c.Request.URL.Query().Get("action"))
	actor := strings.TrimSpace(c.Request.URL.Query().Get("actor"))
	summary := strings.TrimSpace(c.Request.URL.Query().Get("summary"))
	limit := 30
	page := 1
	if value := strings.TrimSpace(c.Request.URL.Query().Get("limit")); value != "" {
		parsed, limitErr := strconv.Atoi(value)
		if limitErr == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if value := strings.TrimSpace(c.Request.URL.Query().Get("page")); value != "" {
		parsed, pageErr := strconv.Atoi(value)
		if pageErr == nil && parsed > 0 {
			page = parsed
		}
	}

	filter := ""
	params := dbx.Params{}
	if userID != "" {
		filter = "target_user_id = {:user}"
		params["user"] = userID
	}
	if action != "" {
		filter = appendFilter(filter, "action ~ {:action}")
		params["action"] = "%" + action + "%"
	}
	if actor != "" {
		filter = appendFilter(filter, "actor_display_name ~ {:actor}")
		params["actor"] = "%" + actor + "%"
	}
	if summary != "" {
		filter = appendFilter(filter, "summary ~ {:summary}")
		params["summary"] = "%" + summary + "%"
	}
	offset := (page - 1) * limit
	records, recordsErr := h.App.FindRecordsByFilter(
		store.CollectionAuditLogs,
		filter,
		"-created",
		limit+1,
		offset,
		params,
	)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to load audit logs.", logging.Fields{
			"filter": filter,
			"page":   page,
			"limit":  limit,
			"error":  recordsErr.Error(),
		})
	}

	hasMore := false
	if len(records) > limit {
		hasMore = true
		records = records[:limit]
	}

	entries := []auditLogEntry{}
	for _, record := range records {
		created := ""
		if !record.GetDateTime("created").IsZero() {
			created = record.GetDateTime("created").Time().Format(time.RFC3339)
		}
		entries = append(entries, auditLogEntry{
			ID:                  record.Id,
			Action:              record.GetString("action"),
			Summary:             record.GetString("summary"),
			ActorID:             record.GetString("actor_id"),
			ActorDisplayName:    record.GetString("actor_display_name"),
			TargetUserID:        record.GetString("target_user_id"),
			TargetUserName:      record.GetString("target_user_name"),
			TargetCharacterID:   record.GetInt("target_character_id"),
			TargetCharacterName: record.GetString("target_character_name"),
			Created:             created,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"logs":    entries,
		"page":    page,
		"limit":   limit,
		"hasMore": hasMore,
	})
}

func (h *Handler) CancelJob(c *core.RequestEvent) error {
	jobID := strings.TrimSpace(c.Request.PathValue("id"))
	if jobID == "" {
		return router.NewBadRequestError("Missing job id.", nil)
	}

	records, recordsErr := h.App.FindRecordsByFilter(
		"job_runs",
		"job_id = {:job} && status = {:status}",
		"",
		1,
		0,
		dbx.Params{
			"job":    jobID,
			"status": jobs.StatusRunning,
		},
	)
	if recordsErr != nil || len(records) == 0 {
		return router.NewBadRequestError("Job is not running.", logging.Fields{
			"job_id": jobID,
		})
	}

	if !jobs.Cancel(jobID) {
		return router.NewBadRequestError("Job is not cancelable.", logging.Fields{
			"job_id": jobID,
		})
	}
	tracker := jobs.NewJobTracker(h.App)
	for _, record := range records {
		tracker.FinishCanceled(record, jobs.MessageCanceled)
	}

	h.logAction(
		c,
		"job.cancel",
		fmt.Sprintf("Canceled job %s", jobID),
		"",
		"",
		nil,
	)

	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) JobRuns(c *core.RequestEvent) error {
	page := 1
	limit := 20
	var startAt *time.Time
	var endAt *time.Time
	excludeKinds := format.GetQueryList(c.Request.URL.Query(), "kinds")
	if value := strings.TrimSpace(c.Request.URL.Query().Get("page")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if value := strings.TrimSpace(c.Request.URL.Query().Get("limit")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	startAt, endAt, err := parseJobRunRange(c)
	if err != nil {
		return err
	}
	if startAt != nil && endAt != nil && startAt.After(*endAt) {
		return router.NewBadRequestError("Start date must be before end date.", logging.Fields{
			"startAt": startAt.Format(time.RFC3339),
			"endAt":   endAt.Format(time.RFC3339),
		})
	}
	offset := (page - 1) * limit

	const scanBatch = 200
	targetCount := offset + limit + 1
	jobIDOrder := []string{}
	seen := map[string]struct{}{}
	recordOffset := 0
	scanParams := dbx.Params{}
	dateFilter := ""
	if startAt != nil {
		dateFilter = appendFilter(
			dateFilter,
			fmt.Sprintf(`started_at >= "%s"`, startAt.UTC().Format(time.RFC3339)),
		)
	}
	if endAt != nil {
		dateFilter = appendFilter(
			dateFilter,
			fmt.Sprintf(`started_at <= "%s"`, endAt.UTC().Format(time.RFC3339)),
		)
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
	parentFilter := `(step = "" || step = null || kind = "map_data_step")`
	parentOnlyFilter := ""
	if dateFilter != "" {
		parentOnlyFilter = appendFilter(parentFilter, dateFilter)
	} else {
		parentOnlyFilter = parentFilter
	}
	scanFilter := parentOnlyFilter
	if kindFilter != "" {
		scanFilter = appendFilter(scanFilter, kindFilter)
	}
	scanFilter = appendFilter(scanFilter, hiddenFilter)
	for {
		records, err := h.App.FindRecordsByFilter(
			"job_runs",
			scanFilter,
			"-started_at",
			scanBatch,
			recordOffset,
			scanParams,
		)
		if err != nil {
			return router.NewInternalServerError("Failed to load jobs.", logging.Fields{
				"error": err,
			})
		}
		if len(records) == 0 {
			break
		}
		for _, record := range records {
			jobID := record.GetString("job_id")
			if jobID == "" {
				continue
			}
			if _, ok := seen[jobID]; ok {
				continue
			}
			seen[jobID] = struct{}{}
			jobIDOrder = append(jobIDOrder, jobID)
			if len(jobIDOrder) >= targetCount {
				break
			}
		}
		if len(jobIDOrder) >= targetCount {
			break
		}
		recordOffset += len(records)
	}

	hasMore := len(jobIDOrder) > offset+limit
	if offset >= len(jobIDOrder) {
		return c.JSON(http.StatusOK, map[string]any{
			"jobs":    []jobRunGroup{},
			"page":    page,
			"limit":   limit,
			"hasMore": hasMore,
		})
	}

	end := offset + limit
	if end > len(jobIDOrder) {
		end = len(jobIDOrder)
	}
	jobIDs := jobIDOrder[offset:end]
	clauses := make([]string, 0, len(jobIDs))
	params := dbx.Params{}
	for i, jobID := range jobIDs {
		key := fmt.Sprintf("job_%d", i)
		clauses = append(clauses, fmt.Sprintf("job_id = {:%s}", key))
		params[key] = jobID
	}
	filter := strings.Join(clauses, " || ")
	filter = appendFilter(filter, hiddenFilter)
	records, recordsErr := h.App.FindRecordsByFilter("job_runs", filter, "", 0, 0, params)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to load jobs.", logging.Fields{
			"error": recordsErr,
		})
	}

	groups := map[string]*jobRunGroup{}
	for _, jobID := range jobIDs {
		groups[jobID] = &jobRunGroup{Steps: []jobRunEntry{}}
	}

	var latest time.Time
	for _, record := range records {
		if record.GetBool("hidden") {
			continue
		}
		jobID := record.GetString("job_id")
		group := groups[jobID]
		if group == nil {
			continue
		}
		startedAt := record.GetDateTime("started_at")
		if !startedAt.IsZero() && startedAt.Time().After(latest) {
			latest = startedAt.Time()
		}
		completedAt := record.GetDateTime("completed_at")
		if !completedAt.IsZero() && completedAt.Time().After(latest) {
			latest = completedAt.Time()
		}
		entry := toJobRunEntry(record)
		if entry.Step == "" {
			group.Parent = entry
		} else {
			group.Steps = append(group.Steps, entry)
		}
	}

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

	etagKey := strings.Join(excludeKinds, ",")
	etag := fmt.Sprintf(`W/"jobruns-%d-%d-%d-%s"`, latest.Unix(), len(jobIDs), len(records), etagKey)
	if match := c.Request.Header.Get("If-None-Match"); match != "" && match == etag {
		return c.NoContent(http.StatusNotModified)
	}
	c.Response.Header().Set("ETag", etag)
	if !latest.IsZero() {
		c.Response.Header().Set("Last-Modified", latest.UTC().Format(time.RFC1123))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"jobs":    ordered,
		"page":    page,
		"limit":   limit,
		"hasMore": hasMore,
	})
}

func appendFilter(filter, clause string) string {
	if filter == "" {
		return clause
	}
	return filter + " && " + clause
}

func parseJobRunRange(c *core.RequestEvent) (*time.Time, *time.Time, error) {
	query := c.Request.URL.Query()
	startAtRaw := strings.TrimSpace(query.Get("startAt"))
	endAtRaw := strings.TrimSpace(query.Get("endAt"))
	startDateRaw := strings.TrimSpace(query.Get("startDate"))
	endDateRaw := strings.TrimSpace(query.Get("endDate"))

	parseDateTime := func(value string) (time.Time, error) {
		if value == "" {
			return time.Time{}, nil
		}
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			return parsed, nil
		}
		if parsed, err := time.Parse("2006-01-02T15:04", value); err == nil {
			return time.Date(
				parsed.Year(),
				parsed.Month(),
				parsed.Day(),
				parsed.Hour(),
				parsed.Minute(),
				0,
				0,
				time.UTC,
			), nil
		}
		return time.Time{}, fmt.Errorf("invalid datetime")
	}

	var startAt *time.Time
	var endAt *time.Time
	if startAtRaw != "" {
		parsed, err := parseDateTime(startAtRaw)
		if err != nil {
			return nil, nil, router.NewBadRequestError("Invalid start time.", logging.Fields{
				"startAt": startAtRaw,
			})
		}
		startAt = &parsed
	} else if startDateRaw != "" {
		parsed, err := time.Parse("2006-01-02", startDateRaw)
		if err != nil {
			return nil, nil, router.NewBadRequestError("Invalid start date.", logging.Fields{
				"startDate": startDateRaw,
			})
		}
		startAt = &parsed
	}

	if endAtRaw != "" {
		parsed, err := parseDateTime(endAtRaw)
		if err != nil {
			return nil, nil, router.NewBadRequestError("Invalid end time.", logging.Fields{
				"endAt": endAtRaw,
			})
		}
		endAt = &parsed
	} else if endDateRaw != "" {
		parsed, err := time.Parse("2006-01-02", endDateRaw)
		if err != nil {
			return nil, nil, router.NewBadRequestError("Invalid end date.", logging.Fields{
				"endDate": endDateRaw,
			})
		}
		endOfDay := time.Date(
			parsed.Year(),
			parsed.Month(),
			parsed.Day(),
			23,
			59,
			59,
			int(time.Second-time.Nanosecond),
			time.UTC,
		)
		endAt = &endOfDay
	}

	return startAt, endAt, nil
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

func (h *Handler) logAction(c *core.RequestEvent, action, summary, userID, targetUserName string, character *core.Record) {
	collection, collectionErr := h.App.FindCollectionByNameOrId(store.CollectionAuditLogs)
	if collectionErr != nil {
		return
	}
	record := core.NewRecord(collection)
	record.Set("action", action)
	record.Set("summary", summary)
	record.Set("target_user_id", userID)
	if character != nil {
		record.Set("target_character_id", character.GetInt("eve_character_id"))
		record.Set("target_character_name", character.GetString("eve_character_name"))
	}
	value := c.Get("admin_record")
	admin, ok := value.(*core.Record)
	if ok {
		record.Set("actor_id", admin.Id)
		display := admin.GetString("eve_character_name")
		if display == "" {
			display = admin.Id
		}
		record.Set("actor_display_name", display)
	}
	if targetUserName != "" {
		record.Set("target_user_name", targetUserName)
	} else if userID != "" {
		user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
		if userErr == nil {
			record.Set("target_user_name", user.GetString("eve_character_name"))
		}
	}
	if saveErr := h.App.Save(record); saveErr != nil {
		logging.New(h.App).
			WithFields(logging.Fields{
				"action": action,
				"user":   userID,
			}).
			WithErr(saveErr).
			Warn("admin audit log save failed")
	}
}

func (h *Handler) UserDetails(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	record, recordErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if recordErr != nil {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}

	characters, _ := h.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user}",
		"-is_main",
		0,
		0, dbx.Params{"user": userID},
	)

	response := userResponse{
		UserID:      record.Id,
		AccessLevel: record.GetString("access_level"),
		Characters:  []characterResponse{},
	}
	if h.Intel != nil {
		if tokenValid, tokenErr := h.Intel.HasValidUploaderToken(record.Id); tokenErr == nil {
			response.UploaderTokenValid = tokenValid
		}
	}
	if revokedAt := record.GetDateTime("session_revoked_at").Time(); !revokedAt.IsZero() {
		response.SessionRevokedAt = revokedAt.Format(time.RFC3339)
	}

	for _, charRecord := range characters {
		response.Characters = append(response.Characters, newCharacter(charRecord, nil, nil))
	}

	hydrated := h.hydrateCharacterAffiliations(c, response.Characters)
	if len(hydrated) > 0 {
		response.Characters = hydrated
	}

	sort.Slice(response.Characters, func(i, j int) bool {
		if response.Characters[i].IsMain != response.Characters[j].IsMain {
			return response.Characters[i].IsMain
		}
		return response.Characters[i].CharacterID < response.Characters[j].CharacterID
	})

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) RefreshCharacter(c *core.RequestEvent) error {
	id := c.Request.PathValue("id")
	record, recordErr := h.App.FindRecordById(store.CollectionCharacters, id)
	if recordErr != nil {
		return router.NewNotFoundError("Character not found.", logging.Fields{
			"character_record_id": id,
		})
	}

	refreshErr := h.Refresher.RefreshCharacter(c.Request.Context(), record)
	if refreshErr != nil {
		return router.NewInternalServerError("Failed to refresh character.", logging.Fields{
			"character_id": record.GetInt("eve_character_id"),
		})
	}

	h.logAction(
		c,
		"character.refresh",
		"Refreshed character "+record.GetString("eve_character_name"),
		record.GetString("user"),
		"",
		record,
	)

	return c.JSON(http.StatusOK, newCharacter(record, nil, nil))
}

func (h *Handler) RefreshAllCharacters(c *core.RequestEvent) error {
	payload := struct {
		UserID string `json:"user_id"`
	}{}
	if c.Request.ContentLength > 0 {
		if bindErr := c.BindBody(&payload); bindErr != nil {
			return router.NewBadRequestError("Invalid payload.", logging.Fields{
				"error": bindErr,
			})
		}
	}
	payload.UserID = strings.TrimSpace(payload.UserID)

	var user *core.Record
	if payload.UserID != "" {
		userRecord, userErr := h.App.FindRecordById(store.CollectionUsers, payload.UserID)
		if userErr != nil {
			return router.NewNotFoundError("User not found.", logging.Fields{
				"user_id": payload.UserID,
			})
		}
		user = userRecord
	}

	filter := ""
	params := dbx.Params{}
	scope := "all"
	step := ""
	if payload.UserID != "" {
		pending, pendingErr := h.App.FindRecordsByFilter(
			"job_runs",
			"kind = {:kind} && step = {:step} && status = {:status}",
			"",
			1,
			0,
			dbx.Params{
				"kind":   "character_refresh",
				"step":   "user:" + payload.UserID,
				"status": jobs.StatusRunning,
			},
		)
		if pendingErr == nil && len(pending) > 0 {
			return router.NewBadRequestError("Character refresh already running for this user.", logging.Fields{
				"user_id": payload.UserID,
			})
		}

		filter = "user = {:user}"
		params["user"] = payload.UserID
		scope = "user"
		step = "user:" + payload.UserID
	}
	records, recordsErr := h.App.FindRecordsByFilter(
		store.CollectionCharacters,
		filter,
		"",
		0,
		0,
		params,
	)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to load characters.", logging.Fields{
			"user_id": payload.UserID,
		})
	}
	if payload.UserID != "" && len(records) > 0 {
		const chunkSize = 20
		for start := 0; start < len(records); start += chunkSize {
			end := start + chunkSize
			if end > len(records) {
				end = len(records)
			}
			clauses := make([]string, 0, end-start)
			params := dbx.Params{
				"kind":   "character_refresh",
				"status": jobs.StatusRunning,
			}
			for i, rec := range records[start:end] {
				key := fmt.Sprintf("step_%d", i)
				clauses = append(clauses, fmt.Sprintf("step = {:%s}", key))
				params[key] = "character:" + rec.Id
			}
			filter := fmt.Sprintf("kind = {:kind} && status = {:status} && (%s)", strings.Join(clauses, " || "))
			pending, pendingErr := h.App.FindRecordsByFilter(
				"job_runs",
				filter,
				"",
				1,
				0,
				params,
			)
			if pendingErr == nil && len(pending) > 0 {
				return router.NewBadRequestError("Character refresh already running for this user.", logging.Fields{
					"user_id": payload.UserID,
				})
			}
		}
	}

	actorID := ""
	if c.Auth != nil {
		actorID = c.Auth.Id
	}
	runner := jobs.NewRunner(h.App, jobs.RunOptions{
		JobName: "admin.character_refresh",
		JobOptions: jobs.JobOptions{
			Kind:    "character_refresh",
			Step:    step,
			Trigger: "admin.character_refresh",
			ActorID: actorID,
		},
		Timeout: 5 * time.Minute,
		JobFunc: func(ctx context.Context) context.Context {
			return auth.WithRefreshJobMeta(ctx, "admin.character_refresh", actorID)
		},
	})
	jobID := runner.JobID()
	targetName := ""
	if user != nil {
		targetName = user.GetString("eve_character_name")
	}
	suffix := ""
	if len(records) != 1 {
		suffix = "s"
	}
	summary := fmt.Sprintf("Queued refresh for %d character%s (job %s)", len(records), suffix, jobID)
	if payload.UserID != "" && targetName != "" {
		summary = fmt.Sprintf("Queued refresh for %s (%d character%s, job %s)", targetName, len(records), suffix, jobID)
	}
	h.logAction(
		c,
		"character.refresh_all",
		summary,
		payload.UserID,
		targetName,
		nil,
	)

	go func(records []*core.Record, jobID string, userID string, scope string, runner *jobs.Runner) {
		start := time.Now()
		var success int
		var failed int

		_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
			success, failed = h.Refresher.RefreshAllBatched(ctx, records, 25, 350*time.Millisecond)
			if ctx.Err() != nil {
				return ctx.Err()
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
	}(records, jobID, payload.UserID, scope, runner)

	return c.JSON(http.StatusAccepted, map[string]any{
		"job_id": jobID,
		"scope":  scope,
	})
}

func (h *Handler) RunCleanupJob(c *core.RequestEvent) error {
	actorID := ""
	if c.Auth != nil {
		actorID = c.Auth.Id
	}

	runner := jobs.NewRunner(h.App, jobs.RunOptions{
		JobName: jobs.JobCleanup,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobCleanup,
			Trigger: jobs.TriggerAdminManual,
			ActorID: actorID,
		},
		Timeout: jobs.NoTimeout,
	})

	jobID := runner.JobID()

	go func() {
		_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
			var reportHashCount int
			var intelUploaderCount int
			var uploaderTokenCount int
			var intelReportCount int

			if err := stepper.Run("cleanup_report_hashes", false, func(ctx context.Context) error {
				count, err := h.Cleanup.RemoveExpired(store.CollectionIntelReportHash)
				if err == nil {
					reportHashCount = count
				}
				return err
			}); err != nil {
				return err
			}

			if err := stepper.Run("cleanup_intel_uploaders", false, func(ctx context.Context) error {
				count, err := h.Cleanup.RemoveExpired(store.CollectionIntelUploaders)
				intelUploaderCount = count
				return err
			}); err != nil {
				return err
			}

			if err := stepper.Run("cleanup_uploader_tokens", false, func(ctx context.Context) error {
				count, err := h.Cleanup.RemoveRevokedUploaderTokens()
				uploaderTokenCount = count
				return err
			}); err != nil {
				return err
			}

			if err := stepper.Run("cleanup_intel_reports", false, func(ctx context.Context) error {
				count, err := h.Cleanup.RemoveOldIntelReports(cleanup.IntelReportRetention)
				intelReportCount = count
				return err
			}); err != nil {
				return err
			}

			runner.WithFields(logging.Fields{
				"intel_report_hash_count": reportHashCount,
				"intel_uploaders_count":   intelUploaderCount,
				"uploader_tokens_count":   uploaderTokenCount,
				"intel_reports_count":     intelReportCount,
			})

			return nil
		})
	}()

	return c.JSON(http.StatusAccepted, map[string]any{
		"job_id": jobID,
	})
}

func (h *Handler) SetMainCharacter(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	payload := struct {
		CharacterRecordID string `json:"character_record_id"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil || payload.CharacterRecordID == "" {
		return router.NewBadRequestError("Missing character record id.", logging.Fields{
			"user_id": userID,
		})
	}

	user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}

	character, characterErr := h.App.FindRecordById(store.CollectionCharacters, payload.CharacterRecordID)
	if characterErr != nil || character.GetString("user") != userID {
		return router.NewNotFoundError("Character not found.", logging.Fields{
			"user_id":              userID,
			"character_record_id":  payload.CharacterRecordID,
			"character_owner_user": character.GetString("user"),
		})
	}

	records, _ := h.App.FindRecordsByFilter(store.CollectionCharacters, "user = {:user}", "", 0, 0, dbx.Params{"user": userID})
	for _, rec := range records {
		rec.Set("is_main", rec.Id == character.Id)
		_ = h.App.Save(rec)
	}

	if updateErr := h.updateUserFromCharacter(user, character); updateErr != nil {
		return router.NewInternalServerError("Failed to update user.", logging.Fields{
			"user_id":       userID,
			"character_id":  character.GetInt("eve_character_id"),
			"character_rec": character.Id,
		})
	}
	h.logAction(
		c,
		"character.set_main",
		"Set main to "+character.GetString("eve_character_name"),
		userID,
		user.GetString("eve_character_name"),
		character,
	)

	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) SetAccessLevel(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	payload := struct {
		AccessLevel string `json:"access_level"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{
			"user_id": userID,
		})
	}
	if payload.AccessLevel != "" && payload.AccessLevel != "staff" && payload.AccessLevel != "admin" {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{
			"user_id":      userID,
			"access_level": payload.AccessLevel,
		})
	}
	if payload.AccessLevel == "admin" {
		return middleware.ErrForbidden
	}

	user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}
	if user.GetString("access_level") == "admin" {
		return middleware.ErrForbidden
	}
	user.Set("access_level", payload.AccessLevel)
	if saveErr := h.App.Save(user); saveErr != nil {
		return router.NewInternalServerError("Failed to update user.", logging.Fields{
			"user_id":      userID,
			"access_level": payload.AccessLevel,
		})
	}
	action := "user.access_level_cleared"
	summary := "Cleared access level"
	if payload.AccessLevel != "" {
		action = "user.access_level_set"
		summary = "Set access level to " + payload.AccessLevel
	}
	h.logAction(c, action, summary, userID, user.GetString("eve_character_name"), nil)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) RevokeSessions(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}
	user.RefreshTokenKey()
	sessionRevokedAt, _ := types.ParseDateTime(time.Now())
	user.Set("session_revoked_at", sessionRevokedAt)
	if saveErr := h.App.Save(user); saveErr != nil {
		return router.NewInternalServerError("Failed to revoke sessions.", logging.Fields{
			"user_id": userID,
		})
	}
	if h.Intel != nil {
		if revokeErr := h.Intel.RevokeUploaderSessionsForUser(userID); revokeErr != nil {
			return router.NewInternalServerError("Failed to revoke uploader sessions.", logging.Fields{
				"user_id": userID,
			})
		}
	}
	h.logAction(c, "user.revoke_sessions", "Revoked sessions", userID, user.GetString("eve_character_name"), nil)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) RevokeUploaderTokens(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	if h.Intel != nil {
		if revokeErr := h.Intel.RevokeUploaderTokensForUser(userID); revokeErr != nil {
			return router.NewInternalServerError("Failed to revoke uploader tokens.", logging.Fields{
				"user_id": userID,
			})
		}
	}
	h.logAction(c, "user.revoke_upload_tokens", "Revoked uploader tokens", userID, "", nil)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) RegenerateUploaderToken(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	_, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}
	if h.Intel == nil {
		return router.NewInternalServerError("Intel service unavailable.", logging.Fields{
			"user_id": userID,
		})
	}
	record, regenErr := h.Intel.RegenerateUploaderToken(userID)
	if regenErr != nil {
		return router.NewInternalServerError("Failed to regenerate uploader token.", logging.Fields{
			"user_id": userID,
		})
	}
	h.logAction(c, "user.regenerate_upload_token", "Regenerated uploader token", userID, "", nil)
	return c.JSON(http.StatusOK, map[string]any{"token": record.Id})
}

func (h *Handler) RevokeCharacterTokens(c *core.RequestEvent) error {
	id := c.Request.PathValue("id")
	record, recordErr := h.App.FindRecordById(store.CollectionCharacters, id)
	if recordErr != nil {
		return router.NewNotFoundError("Character not found.", logging.Fields{
			"character_record_id": id,
		})
	}
	h.clearCharacterTokens(record, "revoked")
	if saveErr := h.App.Save(record); saveErr != nil {
		return router.NewInternalServerError("Failed to revoke tokens.", logging.Fields{
			"character_id": record.GetInt("eve_character_id"),
		})
	}
	h.logAction(
		c,
		"character.revoke_tokens",
		"Revoked character tokens for "+record.GetString("eve_character_name"),
		record.GetString("user"),
		"",
		record,
	)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) RemoveCharacter(c *core.RequestEvent) error {
	id := c.Request.PathValue("id")
	record, recordErr := h.App.FindRecordById(store.CollectionCharacters, id)
	if recordErr != nil {
		return router.NewNotFoundError("Character not found.", logging.Fields{
			"character_record_id": id,
		})
	}
	userID := record.GetString("user")
	isMain := record.GetBool("is_main")
	if isMain {
		others, othersErr := h.App.FindRecordsByFilter(
			store.CollectionCharacters,
			"user = {:user} && id != {:id}",
			"",
			1,
			0, dbx.Params{"user": userID, "id": record.Id},
		)
		if othersErr == nil && len(others) > 0 {
			return router.NewBadRequestError("Cannot remove main character while other characters remain.", logging.Fields{
				"user_id":      userID,
				"character_id": record.GetInt("eve_character_id"),
			})
		}
	}
	targetUserName := ""
	if userID != "" {
		user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
		if userErr == nil {
			targetUserName = user.GetString("eve_character_name")
		}
	}
	deleteErr := h.App.Delete(record)
	if deleteErr != nil {
		return router.NewInternalServerError("Failed to delete character.", logging.Fields{
			"character_id": record.GetInt("eve_character_id"),
		})
	}
	if userID != "" && isMain {
		_ = h.deleteUserIfNoCharacters(userID)
	}
	h.logAction(
		c,
		"character.remove",
		"Removed character "+record.GetString("eve_character_name"),
		userID,
		targetUserName,
		record,
	)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) MoveCharacter(c *core.RequestEvent) error {
	id := c.Request.PathValue("id")
	payload := struct {
		TargetUserID string `json:"target_user_id"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil || payload.TargetUserID == "" {
		return router.NewBadRequestError("Missing target user.", logging.Fields{
			"character_record_id": id,
		})
	}

	record, recordErr := h.App.FindRecordById(store.CollectionCharacters, id)
	if recordErr != nil {
		return router.NewNotFoundError("Character not found.", logging.Fields{
			"character_record_id": id,
		})
	}

	sourceUserID := record.GetString("user")
	targetUser, targetErr := h.App.FindRecordById(store.CollectionUsers, payload.TargetUserID)
	if targetErr != nil {
		return router.NewNotFoundError("Target user not found.", logging.Fields{
			"target_user_id": payload.TargetUserID,
		})
	}
	targetMain, _ := h.findMainCharacter(targetUser.Id)
	if targetMain == nil {
		return router.NewBadRequestError("Target user missing main character.", logging.Fields{
			"target_user_id": payload.TargetUserID,
		})
	}

	record.Set("user", targetUser.Id)
	record.Set("is_main", false)
	if saveErr := h.App.Save(record); saveErr != nil {
		return router.NewInternalServerError("Failed to move character.", logging.Fields{
			"character_id":   record.GetInt("eve_character_id"),
			"source_user_id": sourceUserID,
			"target_user_id": targetUser.Id,
		})
	}

	if sourceUserID != "" && sourceUserID != targetUser.Id {
		// Post-move cleanup: delete the source account only if it's now empty.
		_ = h.deleteUserIfNoCharacters(sourceUserID)
	}
	if sourceUserID != "" {
		h.logAction(
			c,
			"character.move_out",
			"Moved character "+record.GetString("eve_character_name")+" to "+targetUser.Id,
			sourceUserID,
			"",
			record,
		)
	}
	h.logAction(
		c,
		"character.move_in",
		"Received character "+record.GetString("eve_character_name")+" from "+sourceUserID,
		targetUser.Id,
		targetUser.GetString("eve_character_name"),
		record,
	)

	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) MergeUsers(c *core.RequestEvent) error {
	sourceUserID := c.Request.PathValue("id")
	payload := struct {
		TargetUserID string `json:"target_user_id"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil || payload.TargetUserID == "" {
		return router.NewBadRequestError("Missing target user.", logging.Fields{
			"source_user_id": sourceUserID,
		})
	}

	if sourceUserID == payload.TargetUserID {
		return router.NewBadRequestError("Source and target must differ.", logging.Fields{
			"source_user_id": sourceUserID,
			"target_user_id": payload.TargetUserID,
		})
	}

	sourceUser, sourceErr := h.App.FindRecordById(store.CollectionUsers, sourceUserID)
	if sourceErr != nil {
		return router.NewNotFoundError("Source user not found.", logging.Fields{
			"source_user_id": sourceUserID,
		})
	}
	targetUser, targetErr := h.App.FindRecordById(store.CollectionUsers, payload.TargetUserID)
	if targetErr != nil {
		return router.NewNotFoundError("Target user not found.", logging.Fields{
			"target_user_id": payload.TargetUserID,
		})
	}
	targetMain, _ := h.findMainCharacter(targetUser.Id)
	if targetMain == nil {
		return router.NewBadRequestError("Target user missing main character.", logging.Fields{
			"target_user_id": payload.TargetUserID,
		})
	}

	records, recordsErr := h.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user}",
		"",
		0,
		0, dbx.Params{"user": sourceUser.Id},
	)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to load characters.", logging.Fields{
			"user_id": sourceUser.Id,
		})
	}

	for _, rec := range records {
		rec.Set("user", targetUser.Id)
		rec.Set("is_main", false)
		if saveErr := h.App.Save(rec); saveErr != nil {
			return router.NewInternalServerError("Failed to move character.", logging.Fields{
				"character_id":   rec.GetInt("eve_character_id"),
				"source_user_id": sourceUser.Id,
				"target_user_id": targetUser.Id,
			})
		}
	}

	targetUserName := targetUser.GetString("eve_character_name")
	sourceUserName := sourceUser.GetString("eve_character_name")
	h.logAction(
		c,
		"user.merge_out",
		"Merged account into "+targetUser.Id,
		sourceUser.Id,
		sourceUserName,
		nil,
	)
	h.logAction(
		c,
		"user.merge_in",
		"Merged account from "+sourceUser.Id,
		targetUser.Id,
		targetUserName,
		nil,
	)
	// Post-merge cleanup: delete the source account only if it's now empty.
	_ = h.deleteUserIfNoCharacters(sourceUser.Id)

	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) clearCharacterTokens(record *core.Record, reason string) {
	record.Set("oauth_access_token", "")
	record.Set("oauth_refresh_token", "")
	record.Set("oauth_access_expires_at", types.DateTime{})
	record.Set("oauth_refresh_expires_at", types.DateTime{})
	record.Set("esi_token_valid", false)
	record.Set("esi_last_error", reason)
	lastRefreshAt, _ := types.ParseDateTime(time.Now())
	record.Set("esi_last_refresh_at", lastRefreshAt)
}

// deleteUserIfNoCharacters removes users that have no linked characters.
func (h *Handler) deleteUserIfNoCharacters(userID string) error {
	user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return userErr
	}
	records, recordsErr := h.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user}",
		"",
		1,
		0, dbx.Params{"user": userID},
	)
	if recordsErr != nil {
		return recordsErr
	}
	if len(records) == 0 {
		return h.App.Delete(user)
	}
	return nil
}

func (h *Handler) findMainCharacter(userID string) (*core.Record, error) {
	records, recordsErr := h.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user} && is_main = true",
		"",
		1,
		0, dbx.Params{"user": userID},
	)
	if recordsErr != nil || len(records) == 0 {
		return nil, recordsErr
	}
	return records[0], nil
}

func (h *Handler) updateUserFromCharacter(user *core.Record, character *core.Record) error {
	user.Set("eve_character_id", character.GetInt("eve_character_id"))
	user.Set("eve_character_name", character.GetString("eve_character_name"))
	user.Set("eve_corporation_id", character.GetInt("eve_corporation_id"))
	user.Set("eve_alliance_id", character.GetInt("eve_alliance_id"))
	return h.App.Save(user)
}

func (h *Handler) hydrateCharacterAffiliations(c *core.RequestEvent, chars []characterResponse) []characterResponse {
	if len(chars) == 0 {
		return chars
	}
	corpIDs := make([]int, 0, len(chars))
	allianceIDs := make([]int, 0, len(chars))
	for _, char := range chars {
		if char.CorpID > 0 {
			corpIDs = append(corpIDs, char.CorpID)
		}
		if char.AllianceID > 0 {
			allianceIDs = append(allianceIDs, char.AllianceID)
		}
	}
	corpNames := store.GetOrgNames(h.App, store.CollectionCorporations, corpIDs)
	allianceNames := store.GetOrgNames(h.App, store.CollectionAlliances, allianceIDs)
	if len(corpNames) == 0 && len(allianceNames) == 0 {
		return chars
	}
	enriched := make([]characterResponse, 0, len(chars))
	for _, char := range chars {
		if name, ok := corpNames[char.CorpID]; ok {
			char.CorpName = name
		}
		if name, ok := allianceNames[char.AllianceID]; ok {
			char.AllianceName = name
		}
		enriched = append(enriched, char)
	}
	return enriched
}

func newCharacter(record *core.Record, corpName map[int]string, allianceName map[int]string) characterResponse {
	refresh := record.GetDateTime("esi_last_refresh_at")
	refreshAt := ""
	if !refresh.IsZero() {
		refreshAt = refresh.Time().Format(time.RFC3339)
	}
	corpID := record.GetInt("eve_corporation_id")
	allianceID := record.GetInt("eve_alliance_id")
	corp := ""
	alliance := ""
	if corpName != nil {
		corp = corpName[corpID]
	}
	if allianceName != nil {
		alliance = allianceName[allianceID]
	}
	return characterResponse{
		ID:               record.Id,
		CharacterID:      record.GetInt("eve_character_id"),
		Name:             record.GetString("eve_character_name"),
		CorpID:           corpID,
		CorpName:         corp,
		AllianceID:       allianceID,
		AllianceName:     alliance,
		IsMain:           record.GetBool("is_main"),
		Scopes:           record.GetString("oauth_scopes"),
		ESILastRefreshAt: refreshAt,
		ESILastError:     record.GetString("esi_last_error"),
		ESITokenValid:    record.GetBool("esi_token_valid"),
	}
}
