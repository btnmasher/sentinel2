package mapdata

import (
	"context"
	"io"
	"strconv"
	"strings"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

const moonGroupID = 8

func (s *SDEImporter) importMoons(ctx context.Context, r io.Reader) error {
	var saved, skippedNotMoon, skippedMissingID, skippedMissingSystem, skippedUnchanged int

	rowCount, scanErr := forEachJSONLRow(r, func(i int, row map[string]any) error {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		moon, reason := parseValidMoonRow(row)
		switch reason {
		case "not_moon":
			skippedNotMoon++
			return nil
		case "missing_id":
			skippedMissingID++
			return nil
		case "missing_system":
			skippedMissingSystem++
			return nil
		}
		payload := s.moonPayload(moon)
		changed, upsertErr := s.upsertNumberRecordIfChanged(store.CollectionMoons, moon.id, payload)
		if upsertErr != nil {
			s.logger.WithErr(upsertErr).
				WithFields(logging.Fields{"moon_id": moon.id, "system_id": moon.systemID}).
				Error("moon upsert failed")
			return upsertErr
		}
		if !changed {
			skippedUnchanged++
			return nil
		}
		saved++
		return nil
	})
	if scanErr != nil {
		return scanErr
	}
	log := s.logger.WithFields(logging.Fields{
		"sde":       "moons",
		"row_count": rowCount,
	})
	log.WithFields(logging.Fields{
		"saved_count":                     saved,
		"skipped_not_moon_count":          skippedNotMoon,
		"skipped_missing_id_count":        skippedMissingID,
		"skipped_missing_system_id_count": skippedMissingSystem,
		"skipped_unchanged_count":         skippedUnchanged,
	}).Info("moons import complete")
	return nil
}

func parseValidMoonRow(row map[string]any) (moon moonRowData, reason string) {
	moon = parseMoonRow(row)
	if moon.groupID != 0 && moon.groupID != moonGroupID {
		return moon, "not_moon"
	}

	if moon.id == 0 {
		return moon, "missing_id"
	}

	if moon.systemID == 0 {
		return moon, "missing_system"
	}
	return moon, ""
}

func (s *SDEImporter) moonPayload(moon moonRowData) map[string]any {
	systemName := s.lookupSystemName(moon.systemID)
	planetName := s.lookupPlanetName(moon.planetID)
	name := moon.name
	if strings.TrimSpace(name) == "" {
		name = deriveMoonName(systemName, planetName, moon.orbitIndex, moon.id)
	}
	return map[string]any{
		"eve_id":      moon.id,
		"name":        name,
		"system_id":   moon.systemID,
		"system_name": systemName,
		"planet_id":   moon.planetID,
		"planet_name": planetName,
	}
}

type moonRowData struct {
	id         int
	name       string
	groupID    int
	systemID   int
	planetID   int
	orbitIndex int
}

func parseMoonRow(row map[string]any) moonRowData {
	return moonRowData{
		id:         getInt(row, "moonID", "itemID", "id", "_key"),
		name:       getString(row, "moonName", "itemName", "name"),
		groupID:    getInt(row, "groupID"),
		systemID:   getInt(row, "solarSystemID", "locationID"),
		planetID:   getInt(row, "orbitID", "planetID"),
		orbitIndex: getInt(row, "orbitIndex"),
	}
}

func deriveMoonName(systemName, planetName string, orbitIndex, moonID int) string {
	if planetName != "" {
		base := planetName
		if orbitIndex > 0 {
			return base + " - Moon " + strconv.Itoa(orbitIndex)
		}
		return base + " - Moon"
	}

	if systemName != "" {
		return systemName + " Moon " + strconv.Itoa(max(orbitIndex, 1))
	}
	return "Moon " + strconv.Itoa(moonID)
}
