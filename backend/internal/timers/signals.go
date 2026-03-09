package timers

import (
	"database/sql"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/format"
	"sentinel2/internal/shared/queryhelpers"
	"sentinel2/internal/store"
)

const (
	signalPreviewLimit    = 3
	signalScorePercentMax = 100
)

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
		updateSignalFromRecord(&current, record, systemID, expiresAt)
		metaBySystem[systemID] = current
	}
	normalizeSignalMap(metaBySystem)
	return metaBySystem, nil
}

func updateSignalFromRecord(current *Signal, record *core.Record, systemID int, expiresAt time.Time) {
	if current == nil {
		return
	}
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
}

func normalizeSignalMap(metaBySystem map[int]Signal) {
	for key := range metaBySystem {
		value := metaBySystem[key]
		normalizeSignal(&value)
		metaBySystem[key] = value
	}
}

func normalizeSignal(value *Signal) {
	value.Severity = queryhelpers.ValueOrTrim(value.Severity, TimerSeverityMedium)
	value.StandingType = queryhelpers.ValueOrTrim(value.StandingType, TimerStandingHostile)
	value.TimerKind = queryhelpers.ValueOrTrim(value.TimerKind, TimerKindCustom)
	value.StructureType = queryhelpers.ValueOrTrim(value.StructureType, TimerStructureCustom)
	value.StageLabel = queryhelpers.ValueOrTrim(value.StageLabel, TimerStageNotApplicable)
	value.RemainingCount = value.Count - len(value.Timers)
	for i := range value.Timers {
		value.Timers[i].Severity = queryhelpers.ValueOrTrim(value.Timers[i].Severity, TimerSeverityMedium)
		value.Timers[i].StandingType = queryhelpers.ValueOrTrim(value.Timers[i].StandingType, TimerStandingHostile)
		value.Timers[i].TimerKind = queryhelpers.ValueOrTrim(value.Timers[i].TimerKind, TimerKindCustom)
		value.Timers[i].StructureType = queryhelpers.ValueOrTrim(value.Timers[i].StructureType, TimerStructureCustom)
		value.Timers[i].StageLabel = queryhelpers.ValueOrTrim(value.Timers[i].StageLabel, TimerStageNotApplicable)
	}
}

func (s *Service) ActiveSystemsByStructureTypes(structureTypes []string, now time.Time, regionIDs []int) (map[int]struct{}, error) {
	exists, err := s.hasTimersCollection()
	if err != nil {
		return nil, err
	}

	if !exists || len(structureTypes) == 0 {
		return map[int]struct{}{}, nil
	}

	exprs := []dbx.Expression{
		dbx.HashExp{"status": timerStatusActive},
		queryhelpers.InExp("structure_type", structureTypes),
	}

	if len(regionIDs) > 0 {
		exprs = append(exprs, queryhelpers.InExp("region_id", regionIDs))
	}

	records, recordsErr := s.App.FindAllRecords(store.CollectionTimers, exprs...)
	if recordsErr != nil {
		return nil, recordsErr
	}

	out := make(map[int]struct{}, len(records))
	for _, record := range records {
		systemID, _, ok := parseSignalRecord(record, now)
		if !ok {
			continue
		}
		out[systemID] = struct{}{}
	}
	return out, nil
}

func (s *Service) ActiveSystemsByStructureTypesInSystems(structureTypes []string, now time.Time, systemIDs []int) (map[int]struct{}, error) {
	exists, err := s.hasTimersCollection()
	if err != nil {
		return nil, err
	}

	if !exists || len(structureTypes) == 0 || len(systemIDs) == 0 {
		return map[int]struct{}{}, nil
	}

	exprs := []dbx.Expression{
		dbx.HashExp{"status": timerStatusActive},
		queryhelpers.InExp("structure_type", structureTypes),
		queryhelpers.InExp("system_id", systemIDs),
	}

	records, recordsErr := s.App.FindAllRecords(store.CollectionTimers, exprs...)
	if recordsErr != nil {
		return nil, recordsErr
	}

	out := make(map[int]struct{}, len(records))
	for _, record := range records {
		systemID, _, ok := parseSignalRecord(record, now)
		if !ok {
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
	case TimerSeverityCritical:
		return severityRankCritical
	case TimerSeverityHigh:
		return severityRankHigh
	case TimerSeverityMedium:
		return severityRankMedium
	case TimerSeverityLow:
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
	fullness := min(signalScorePercentMax, max(0, int(math.Round(record.GetFloat("skyhook_fullness_pct")))))
	return &fullness
}
