package admin

import (
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

	"sentinel2/internal/format"
	"sentinel2/internal/logging"
	"sentinel2/internal/shared/collections"
	"sentinel2/internal/shared/pagination"
	"sentinel2/internal/shared/queryhelpers"
)

type jobRunsOptions struct {
	page          int
	limit         int
	excludeKinds  []string
	startAt       *time.Time
	endAt         *time.Time
	includeHidden bool
}

type cleanupCounts struct {
	reportHashCount    int
	intelUploaderCount int
	uploaderTokenCount int
	intelReportCount   int
	timerCount         int
	staleJobRunCount   int
}

const (
	defaultJobRunsLimit       = 20
	maxJobRunsLimit           = 100
	adminSingleRefreshTimeout = 2 * time.Minute
	adminRefreshAllTimeout    = 5 * time.Minute
	refreshAllBatchSize       = 25
	refreshAllBatchPause      = 350 * time.Millisecond
	adminSovSyncTimeout       = 45 * time.Second
	adminSkyhookSyncTimeout   = 45 * time.Second
	adminSkyhookSyncWindow    = 2 * time.Minute
	staleJobRunTimeout        = 30 * time.Minute
)

func parseJobRunsOptions(c *core.RequestEvent) (jobRunsOptions, error) {
	values := c.Request.URL.Query()
	startAt, endAt, err := parseJobRunRange(c)
	if err != nil {
		return jobRunsOptions{}, err
	}
	return jobRunsOptions{
		page:          format.GetPositiveInt(values, "page", 1, 0),
		limit:         format.GetPositiveInt(values, "limit", defaultJobRunsLimit, maxJobRunsLimit),
		excludeKinds:  format.GetQueryList(values, "kinds"),
		startAt:       startAt,
		endAt:         endAt,
		includeHidden: parseIncludeHidden(values.Get("includeHidden"), values.Get("include_hidden")),
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

func buildJobRunsScanFilter(startAt, endAt *time.Time, excludeKinds []string, includeHidden bool) (string, dbx.Params) {
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

	parentFilter := `(step = "" || step = null || kind = "map_data_step" || (kind = "character_refresh" && step ~ "user:%"))`
	scanFilter := parentFilter
	if dateFilter != "" {
		scanFilter = queryhelpers.AppendAnd(scanFilter, dateFilter)
	}
	if kindFilter != "" {
		scanFilter = queryhelpers.AppendAnd(scanFilter, kindFilter)
	}
	if !includeHidden {
		scanFilter = queryhelpers.AppendAnd(scanFilter, `(hidden = false || hidden = null)`)
	}
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

func buildJobIDFilter(jobIDs []string, includeHidden bool) (string, dbx.Params) {
	filter, params := queryhelpers.BuildOrEqualsFilter("job_id", jobIDs)
	if !includeHidden {
		filter = queryhelpers.AppendAnd(filter, `(hidden = false || hidden = null)`)
	}
	return filter, params
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
		if group.Parent.JobID == "" && len(group.Steps) == 0 {
			continue
		}
		ordered = append(ordered, *group)
	}
	return ordered
}

func computeJobRunsETag(excludeKinds, jobIDs []string, groups map[string]*jobRunGroup, includeHidden bool) string {
	etagHasher := fnv.New64a()
	_, _ = etagHasher.Write([]byte(strings.Join(excludeKinds, ",")))
	_, _ = etagHasher.Write([]byte{0})
	if includeHidden {
		_, _ = etagHasher.Write([]byte("includeHidden=1"))
	} else {
		_, _ = etagHasher.Write([]byte("includeHidden=0"))
	}
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
	scanFilter, scanParams := buildJobRunsScanFilter(opts.startAt, opts.endAt, opts.excludeKinds, opts.includeHidden)
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
	filter, params := buildJobIDFilter(jobIDs, opts.includeHidden)
	records, recordsErr := h.App.FindRecordsByFilter("job_runs", filter, "", 0, 0, params)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to load jobs.", logging.Fields{
			"error": recordsErr,
		})
	}
	groups, ordered, latest := groupJobRunRecords(records, jobIDs)
	etag := computeJobRunsETag(opts.excludeKinds, jobIDs, groups, opts.includeHidden)
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

func parseIncludeHidden(values ...string) bool {
	for _, value := range values {
		switch strings.TrimSpace(strings.ToLower(value)) {
		case "1", "true", "yes", "on":
			return true
		}
	}
	return false
}
