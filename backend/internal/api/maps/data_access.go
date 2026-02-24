package maps

import (
	"errors"
	stdmaps "maps"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/format"
	"sentinel2/internal/shared/collections"
	"sentinel2/internal/shared/queryhelpers"
	"sentinel2/internal/store"
)

const findByNameCandidatesLimit = 50
const ansiblexJumpBridgeStructureType = "ansiblex_jump_bridge"

func (h *MapHandler) fetchRegions(regionIDs []int, mode string) (map[int]Region, error) {
	filter, params := queryhelpers.BuildOrEqualsFilter("eve_id", regionIDs)
	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionRegions, filter, "name", 0, 0, params)
	if recordsErr != nil {
		return nil, recordsErr
	}

	out := map[int]Region{}
	for _, rec := range records {
		id := rec.GetInt("eve_id")
		dto := Region{
			Region: id,
			Name:   rec.GetString("name"),
		}
		switch mode {
		case "eve2d":
			dto.Position.X = rec.GetInt("eve2d_x")
			dto.Position.Y = rec.GetInt("eve2d_y")
			if dto.Position.X == 0 && dto.Position.Y == 0 {
				dto.Position.X = rec.GetInt("metro_x")
				dto.Position.Y = rec.GetInt("metro_y")
			}
		case "real":
			dto.Position.X = rec.GetInt("real_x")
			dto.Position.Y = rec.GetInt("real_y")
		case "dotlan":
			dto.Position.X = rec.GetInt("eve2d_x")
			dto.Position.Y = rec.GetInt("eve2d_y")
			if dto.Position.X == 0 && dto.Position.Y == 0 {
				dto.Position.X = rec.GetInt("metro_x")
				dto.Position.Y = rec.GetInt("metro_y")
			}
		case "metro":
			dto.Position.X = rec.GetInt("metro_x")
			dto.Position.Y = rec.GetInt("metro_y")
		default:
			dto.Position.X = rec.GetInt("metro_x")
			dto.Position.Y = rec.GetInt("metro_y")
		}
		out[id] = dto
	}
	return out, nil
}

func (h *MapHandler) fetchSystems(regionIDs []int, mode string) (map[int]System, error) {
	filter, params := queryhelpers.BuildOrEqualsFilter("region_id", regionIDs)
	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionSolarSystems, filter, "", 0, 0, params)
	if recordsErr != nil {
		return nil, recordsErr
	}

	out := map[int]System{}
	for _, rec := range records {
		id := rec.GetInt("eve_id")
		dto := System{
			Name:           rec.GetString("name"),
			SecurityStatus: rec.GetFloat("security_status"),
			Region:         rec.GetInt("region_id"),
			Constellation:  rec.GetInt("constellation"),
			System:         id,
		}
		switch mode {
		case "real":
			hasRealX := rec.Get("real_x") != nil
			hasRealY := rec.Get("real_y") != nil
			dto.Position.X = rec.GetInt("real_x")
			dto.Position.Y = rec.GetInt("real_y")
			// A real position of (0,0) is valid after normalization.
			// Fall back only when the real coordinates are missing in storage.
			if !hasRealX && !hasRealY {
				dto.Position.X = rec.GetInt("raw_x")
				dto.Position.Y = -rec.GetInt("raw_z")
			}
		case "eve2d":
			dto.Position.X = int(math.Round(rec.GetFloat("eve2d_x")))
			dto.Position.Y = -int(math.Round(rec.GetFloat("eve2d_y")))
		case "metro":
			dto.Position.X = rec.GetInt("metro_x")
			dto.Position.Y = rec.GetInt("metro_y")
		default:
			dto.Position.X = rec.GetInt("dotlan_x")
			dto.Position.Y = rec.GetInt("dotlan_y")
		}
		dto.Absolute.X = rec.GetInt("raw_x")
		dto.Absolute.Y = rec.GetInt("raw_y")
		dto.Absolute.Z = rec.GetInt("raw_z")
		out[id] = dto
	}
	return out, nil
}

func (h *MapHandler) fetchGates(regionIDs []int) ([]Gate, error) {
	filter, params := queryhelpers.BuildOrEqualsFilter("from_region", regionIDs)
	filterTo, paramsTo := queryhelpers.BuildOrEqualsFilter("to_region", regionIDs)
	stdmaps.Copy(params, paramsTo)

	combinedFilter := "(" + filter + ") || (" + filterTo + ")"

	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionGates, combinedFilter, "", 0, 0, params)
	if recordsErr != nil {
		return nil, recordsErr
	}

	out := []Gate{}
	for _, rec := range records {
		fromRegion := rec.GetInt("from_region")
		toRegion := rec.GetInt("to_region")
		gateType := "solarsystem"
		if fromRegion != toRegion {
			gateType = "region"
		} else if rec.GetInt("from_constellation") != rec.GetInt("to_constellation") {
			gateType = "constellation"
		}

		out = append(out, Gate{
			To:          rec.GetInt("to_solarsystem"),
			From:        rec.GetInt("from_solarsystem"),
			Type:        gateType,
			ToRegion:    toRegion,
			FromRegion:  fromRegion,
			ToDotlanX:   rec.GetInt("to_dotlan_x"),
			ToDotlanY:   rec.GetInt("to_dotlan_y"),
			ToMetroX:    rec.GetInt("to_metro_x"),
			ToMetroY:    rec.GetInt("to_metro_y"),
			FromDotlanX: rec.GetInt("from_dotlan_x"),
			FromDotlanY: rec.GetInt("from_dotlan_y"),
			FromMetroX:  rec.GetInt("from_metro_x"),
			FromMetroY:  rec.GetInt("from_metro_y"),
		})
	}

	return out, nil
}

func (h *MapHandler) fetchJumpbridges(regionIDs []int) ([]Jumpbridge, error) {
	disabledSystems, disabledErr := h.fetchDisabledAnsiblexSystems(regionIDs)
	if disabledErr != nil {
		disabledSystems = map[int]struct{}{}
	}

	filter, params := queryhelpers.BuildOrEqualsFilter("from_region", regionIDs)
	filterTo, paramsTo := queryhelpers.BuildOrEqualsFilter("to_region", regionIDs)
	stdmaps.Copy(params, paramsTo)

	combinedFilter := "(" + filter + ") || (" + filterTo + ")"

	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionJumpbridges, combinedFilter, "", 0, 0, params)
	if recordsErr != nil {
		return nil, recordsErr
	}

	out := []Jumpbridge{}
	seen := map[string]struct{}{}
	for _, rec := range records {
		fromRegion := rec.GetInt("from_region")
		toRegion := rec.GetInt("to_region")
		from := rec.GetInt("from_solarsystem")
		to := rec.GetInt("to_solarsystem")
		if from > to {
			from, to = to, from
			fromRegion, toRegion = toRegion, fromRegion
		}
		key := strconv.Itoa(from) + "-" + strconv.Itoa(to)
		if !collections.MarkSeen(seen, key) {
			continue
		}

		out = append(out, Jumpbridge{
			From:       from,
			To:         to,
			FromRegion: fromRegion,
			ToRegion:   toRegion,
			Friendly:   rec.GetBool("is_friendly"),
			Disabled:   jumpbridgeTouchesDisabledSystem(from, to, disabledSystems),
		})
	}
	return out, nil
}

func (h *MapHandler) fetchDisabledAnsiblexSystems(regionIDs []int) (map[int]struct{}, error) {
	if h.Timers == nil {
		return map[int]struct{}{}, nil
	}
	return h.Timers.ActiveSystemsByStructureTypes(
		[]string{ansiblexJumpBridgeStructureType},
		time.Now().UTC(),
		regionIDs,
	)
}

func jumpbridgeTouchesDisabledSystem(from, to int, disabledSystems map[int]struct{}) bool {
	if len(disabledSystems) == 0 {
		return false
	}
	_, fromDisabled := disabledSystems[from]
	_, toDisabled := disabledSystems[to]
	return fromDisabled || toDisabled
}

func (h *MapHandler) fetchTimerSignals(regionIDs []int) (map[int]TimerSignal, error) {
	if h.Timers == nil {
		return map[int]TimerSignal{}, nil
	}
	signals, err := h.Timers.ActiveSignalsByRegions(regionIDs, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	out := make(map[int]TimerSignal, len(signals))
	for systemID := range signals {
		signal := signals[systemID]
		previews := make([]TimerPreview, 0, len(signal.Timers))
		for i := range signal.Timers {
			timer := &signal.Timers[i]
			previews = append(previews, TimerPreview{
				Title:              timer.Title,
				NextExpiresAt:      timer.ExpiresAt.Format(time.RFC3339),
				Severity:           timer.Severity,
				StandingType:       timer.StandingType,
				TimerKind:          timer.TimerKind,
				StructureType:      timer.StructureType,
				StageLabel:         timer.StageLabel,
				PlanetName:         timer.PlanetName,
				MoonName:           timer.MoonName,
				SkyhookFullnessPct: timer.SkyhookFullnessPct,
			})
		}
		out[systemID] = TimerSignal{
			SystemID:           systemID,
			Count:              signal.Count,
			RemainingCount:     signal.RemainingCount,
			NextExpiresAt:      signal.NextExpiresAt.Format(time.RFC3339),
			Severity:           signal.Severity,
			StandingType:       signal.StandingType,
			TimerKind:          signal.TimerKind,
			Title:              signal.Title,
			StructureType:      signal.StructureType,
			StageLabel:         signal.StageLabel,
			PlanetName:         signal.PlanetName,
			MoonName:           signal.MoonName,
			SkyhookFullnessPct: signal.SkyhookFullnessPct,
			Timers:             previews,
		}
	}
	return out, nil
}

func (h *MapHandler) parseRegionIDs(value string) ([]int, error) {
	return h.parseIDsByName(value, store.CollectionRegions)
}

func (h *MapHandler) parseSystemIDs(value string) ([]int, error) {
	return h.parseIDsByName(value, store.CollectionSolarSystems)
}

func (h *MapHandler) parseIDsByName(value, collection string) ([]int, error) {
	parts := format.SplitTokens(value)
	out := []int{}
	for _, part := range parts {
		if part == "" {
			continue
		}
		num, parseErr := strconv.Atoi(part)
		if parseErr == nil {
			out = append(out, num)
			continue
		}
		name := normalizeRegionToken(part)
		record, recordErr := h.findRecordByName(collection, name)
		if recordErr != nil {
			return nil, recordErr
		}
		out = append(out, record.GetInt("eve_id"))
	}
	return out, nil
}

func (h *MapHandler) findRecordByName(collection, name string) (*core.Record, error) {
	records, recordsErr := h.App.FindRecordsByFilter(
		collection,
		"name = {:name}",
		"",
		1,
		0, dbx.Params{"name": name},
	)
	if recordsErr == nil && len(records) > 0 {
		return records[0], nil
	}
	records, recordsErr = h.App.FindRecordsByFilter(
		collection,
		"name ~ {:name}",
		"",
		findByNameCandidatesLimit,
		0, dbx.Params{"name": "%" + name + "%"},
	)
	if recordsErr != nil {
		return nil, recordsErr
	}
	for _, record := range records {
		if strings.EqualFold(record.GetString("name"), name) {
			return record, nil
		}
	}
	return nil, errors.New("not found")
}
