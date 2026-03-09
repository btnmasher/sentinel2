package mapdata

import (
	"context"
	"io"
	"strconv"
	"strings"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

type planetImportStats struct {
	saved                int
	skippedMissingID     int
	skippedMissingSystem int
	skippedUnchanged     int
}

func (s *SDEImporter) importPlanets(ctx context.Context, r io.Reader) error {
	stats := planetImportStats{}
	rowCount, scanErr := forEachJSONLRow(r, func(i int, row map[string]any) error {
		return s.processPlanetRow(ctx, i, row, &stats)
	})
	if scanErr != nil {
		return scanErr
	}
	log := s.logger.WithFields(logging.Fields{
		"sde":       "planets",
		"row_count": rowCount,
	})
	log.WithFields(logging.Fields{
		"saved_count":                     stats.saved,
		"skipped_missing_id_count":        stats.skippedMissingID,
		"skipped_missing_system_id_count": stats.skippedMissingSystem,
		"skipped_unchanged_count":         stats.skippedUnchanged,
	}).Info("planets import complete")
	return nil
}

func (s *SDEImporter) processPlanetRow(ctx context.Context, i int, row map[string]any, stats *planetImportStats) error {
	if i%1000 == 0 && ctx.Err() != nil {
		return ctx.Err()
	}
	planet := parsePlanetRow(row)
	if planet.id == 0 {
		stats.skippedMissingID++
		return nil
	}

	if planet.systemID == 0 {
		stats.skippedMissingSystem++
		return nil
	}
	systemName := s.lookupSystemName(planet.systemID)
	name := planet.name
	if strings.TrimSpace(name) == "" {
		name = derivePlanetName(systemName, planet.celestialIndex, planet.id)
	}
	changed, upsertErr := s.upsertNumberRecordIfChanged(store.CollectionPlanets, planet.id, map[string]any{
		"eve_id":          planet.id,
		"name":            name,
		"system_id":       planet.systemID,
		"system_name":     systemName,
		"celestial_index": planet.celestialIndex,
	})
	if upsertErr != nil {
		s.logger.WithErr(upsertErr).
			WithFields(logging.Fields{"planet_id": planet.id, "system_id": planet.systemID}).
			Error("planet upsert failed")
		return upsertErr
	}
	s.rememberPlanetName(planet.id, name)
	if !changed {
		stats.skippedUnchanged++
		return nil
	}
	stats.saved++
	return nil
}

type planetRowData struct {
	id             int
	name           string
	systemID       int
	celestialIndex int
}

func parsePlanetRow(row map[string]any) planetRowData {
	return planetRowData{
		id:             getInt(row, "planetID", "itemID", "id", "_key"),
		name:           getString(row, "planetName", "itemName", "name"),
		systemID:       getInt(row, "solarSystemID", "locationID"),
		celestialIndex: getInt(row, "celestialIndex"),
	}
}

func derivePlanetName(systemName string, celestialIndex, planetID int) string {
	if systemName != "" && celestialIndex > 0 {
		return systemName + " " + intToRoman(celestialIndex)
	}
	return "Planet " + strconv.Itoa(planetID)
}

func intToRoman(value int) string {
	if value <= 0 {
		return strconv.Itoa(value)
	}
	var numerals = []struct {
		value int
		text  string
	}{
		{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"},
		{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"},
		{10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
	}
	var out strings.Builder
	for _, numeral := range numerals {
		for value >= numeral.value {
			out.WriteString(numeral.text)
			value -= numeral.value
		}
	}
	return out.String()
}
