package timers

import (
	stdmaps "maps"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/shared/queryhelpers"
	"sentinel2/internal/store"
)

func (s *Service) List(input ListInput) ([]*core.Record, error) {
	filter, params := buildListFilter(input, time.Now().UTC())
	limit := input.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	records, recordsErr := s.App.FindRecordsByFilter(store.CollectionTimers, filter, "expires_at", limit, 0, params)
	if recordsErr != nil {
		return nil, recordsErr
	}
	s.hydrateRegionNames(records)
	return records, nil
}

func buildListFilter(input ListInput, now time.Time) (string, dbx.Params) {
	filter := ""
	params := dbx.Params{}

	filter = appendStatusFilter(filter, params, input.Statuses)
	filter = appendRegionFilter(filter, params, input.RegionIDs)
	cutoff := now.Add(-24 * time.Hour)
	params["cutoff"] = cutoff.UTC().Format(time.RFC3339)
	params["active_status"] = timerStatusActive
	filter = queryhelpers.AppendAnd(
		filter,
		"(status = {:active_status} || expires_at >= {:cutoff})",
	)
	filter = queryhelpers.AppendDateTimeClauseUTC(filter, params, input.From, "from", "expires_at >= {:from}")
	filter = queryhelpers.AppendDateTimeClauseUTC(filter, params, input.To, "to", "expires_at <= {:to}")
	return filter, params
}

func appendStatusFilter(filter string, params dbx.Params, statuses []string) string {
	if len(statuses) == 0 {
		params["status"] = timerStatusActive
		return queryhelpers.AppendAnd(filter, "status = {:status}")
	}

	var statusFilter strings.Builder
	for index, item := range statuses {
		key := "status" + strconv.Itoa(index)
		if index > 0 {
			statusFilter.WriteString(" || ")
		}
		statusFilter.WriteString("status = {:")
		statusFilter.WriteString(key)
		statusFilter.WriteString("}")
		params[key] = item
	}
	return queryhelpers.AppendAnd(filter, "("+statusFilter.String()+")")
}

func appendRegionFilter(filter string, params dbx.Params, regionIDs []int) string {
	if len(regionIDs) == 0 {
		return filter
	}
	regionFilter, regionParams := queryhelpers.BuildOrEqualsFilter("region_id", regionIDs)
	stdmaps.Copy(params, regionParams)
	return queryhelpers.AppendAnd(filter, "("+regionFilter+")")
}
