package timers

import (
	"database/sql"
	"errors"
	stdmaps "maps"
	"math"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/format"
	"sentinel2/internal/shared/queryhelpers"
	"sentinel2/internal/store"
)

const signalPreviewLimit = 3

func (s *Service) ActiveSignalsByRegions(regionIDs []int, now time.Time) (map[int]Signal, error) {
	exists, err := s.hasTimersCollection()
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[int]Signal{}, nil
	}

	regionFilter, params := queryhelpers.BuildOrEqualsFilter("region_id", regionIDs)
	filter := queryhelpers.AppendAnd(regionFilter, "status = {:status}")
	params["status"] = timerStatusActive

	records, err := s.App.FindRecordsByFilter(store.CollectionTimers, filter, "expires_at", 0, 0, params)
	if err != nil {
		return nil, err
	}

	metaBySystem := map[int]Signal{}
	for _, record := range records {
		systemID, expiresAt, ok := parseSignalRecord(record, now)
		if !ok {
			continue
		}

		current := metaBySystem[systemID]
		current.SystemID = systemID
		current.Count++
		if current.NextExpiresAt.IsZero() || expiresAt.Before(current.NextExpiresAt) {
			current.NextExpiresAt = expiresAt
			current.StandingType = record.GetString("standing_type")
			current.TimerKind = record.GetString("timer_kind")
			current.Title = record.GetString("title")
			current.StructureType = record.GetString("structure_type")
			current.StageLabel = record.GetString("stage_label")
			current.PlanetName = record.GetString("planet_name")
			current.MoonName = record.GetString("moon_name")
			current.SkyhookFullnessPct = parseSkyhookFullness(record)
		}
		if severityRank(record.GetString("severity")) > severityRank(current.Severity) {
			current.Severity = record.GetString("severity")
		}
		if len(current.Timers) < signalPreviewLimit {
			current.Timers = append(current.Timers, SignalTimerPreview{
				Title:              record.GetString("title"),
				ExpiresAt:          expiresAt,
				Severity:           record.GetString("severity"),
				StandingType:       record.GetString("standing_type"),
				TimerKind:          record.GetString("timer_kind"),
				StructureType:      record.GetString("structure_type"),
				StageLabel:         record.GetString("stage_label"),
				PlanetName:         record.GetString("planet_name"),
				MoonName:           record.GetString("moon_name"),
				SkyhookFullnessPct: parseSkyhookFullness(record),
			})
		}
		metaBySystem[systemID] = current
	}

	for key, value := range metaBySystem {
		value.Severity = queryhelpers.ValueOrTrim(value.Severity, "medium")
		value.StandingType = queryhelpers.ValueOrTrim(value.StandingType, "hostile")
		value.TimerKind = queryhelpers.ValueOrTrim(value.TimerKind, "custom")
		value.StructureType = queryhelpers.ValueOrTrim(value.StructureType, "custom")
		value.StageLabel = queryhelpers.ValueOrTrim(value.StageLabel, "not_applicable")
		value.RemainingCount = value.Count - len(value.Timers)
		for i := range value.Timers {
			value.Timers[i].Severity = queryhelpers.ValueOrTrim(value.Timers[i].Severity, "medium")
			value.Timers[i].StandingType = queryhelpers.ValueOrTrim(value.Timers[i].StandingType, "hostile")
			value.Timers[i].TimerKind = queryhelpers.ValueOrTrim(value.Timers[i].TimerKind, "custom")
			value.Timers[i].StructureType = queryhelpers.ValueOrTrim(value.Timers[i].StructureType, "custom")
			value.Timers[i].StageLabel = queryhelpers.ValueOrTrim(value.Timers[i].StageLabel, "not_applicable")
		}
		metaBySystem[key] = value
	}
	return metaBySystem, nil
}

func (s *Service) ActiveSystemsByStructureTypes(structureTypes []string, now time.Time, regionIDs []int) (map[int]struct{}, error) {
	exists, err := s.hasTimersCollection()
	if err != nil {
		return nil, err
	}
	if !exists || len(structureTypes) == 0 {
		return map[int]struct{}{}, nil
	}

	structureFilter, structureParams := queryhelpers.BuildOrEqualsFilter("structure_type", structureTypes)
	filter := queryhelpers.AppendAnd("status = {:status}", "expires_at > {:now}")
	filter = queryhelpers.AppendAnd(filter, "("+structureFilter+")")
	params := dbx.Params{
		"status": timerStatusActive,
		"now":    now.UTC().Format(time.RFC3339),
	}
	stdmaps.Copy(params, structureParams)
	filter = appendRegionFilter(filter, params, regionIDs)

	records, recordsErr := s.App.FindRecordsByFilter(store.CollectionTimers, filter, "", 0, 0, params)
	if recordsErr != nil {
		return nil, recordsErr
	}

	out := make(map[int]struct{}, len(records))
	for _, record := range records {
		systemID := record.GetInt("system_id")
		if systemID <= 0 {
			continue
		}
		out[systemID] = struct{}{}
	}
	return out, nil
}

func (s *Service) hasTimersCollection() (bool, error) {
	if _, err := s.App.FindCollectionByNameOrId(store.CollectionTimers); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func parseSignalRecord(record *core.Record, now time.Time) (int, time.Time, bool) {
	systemID := record.GetInt("system_id")
	if systemID == 0 {
		return 0, time.Time{}, false
	}

	expiresAt, err := format.ParseDateTimeFlexibleUTC(record.GetString("expires_at"))
	if err != nil || expiresAt.Before(now) {
		return 0, time.Time{}, false
	}
	return systemID, expiresAt, true
}

func severityRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return severityRankCritical
	case "high":
		return severityRankHigh
	case "medium":
		return severityRankMedium
	case "low":
		return severityRankLow
	default:
		return 0
	}
}

func parseSkyhookFullness(record *core.Record) *int {
	value := record.Get("skyhook_fullness_pct")
	if value == nil {
		return nil
	}
	fullness := min(100, max(0, int(math.Round(record.GetFloat("skyhook_fullness_pct")))))
	return &fullness
}
