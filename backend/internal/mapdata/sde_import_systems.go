package mapdata

import (
	"context"
	"io"

	"sentinel2/internal/logging"
	"sentinel2/internal/shared/eve"
	"sentinel2/internal/store"
)

func (s *SDEImporter) importSystems(ctx context.Context, r io.Reader) error {
	var missingID, missingRegionConst, missingName, missingCoords, saved int
	var withPos2D, missingPos2D int
	rowCount, scanErr := forEachJSONLRow(r, func(i int, row map[string]any) error {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		system := parseSystemRow(row)
		switch validateSystemRow(&system) {
		case "missing_id":
			s.logger.WithFields(logging.Fields{"row_id": system.rowID}).Warn("system missing id")
			missingID++
			return nil
		case "missing_region_or_constellation":
			s.logger.
				WithFields(logging.Fields{
					"system_id":        system.id,
					"region_id":        system.regionID,
					"constellation_id": system.constID,
				}).
				Warn("system missing region or constellation")
			missingRegionConst++
			return nil
		case "missing_name":
			s.logger.WithFields(logging.Fields{"system_id": system.id}).Warn("system missing name")
			missingName++
			return nil
		}

		if system.coordsMissing() {
			missingCoords++
		}
		if system.hasPos2D {
			withPos2D++
		} else {
			missingPos2D++
		}

		if upsertErr := s.upsertNumberRecord(store.CollectionSolarSystems, system.id, system.payload()); upsertErr != nil {
			s.logger.WithErr(upsertErr).
				WithFields(logging.Fields{"system_id": system.id}).
				Error("system upsert failed")
			return upsertErr
		}
		s.rememberSystemName(system.id, system.name)
		saved++
		return nil
	})
	if scanErr != nil {
		return scanErr
	}
	log := s.logger.WithFields(logging.Fields{
		"sde":       "systems",
		"row_count": rowCount,
	})
	log.WithFields(logging.Fields{
		"saved_count":              saved,
		"skipped_missing_id_count": missingID,
		"skipped_missing_region_or_constellation_count": missingRegionConst,
		"skipped_missing_name_count":                    missingName,
		"missing_coordinates_count":                     missingCoords,
		"eve2d_present_count":                           withPos2D,
		"eve2d_missing_count":                           missingPos2D,
	}).Info("systems import complete")
	return nil
}

type systemRowData struct {
	id             int
	rowID          string
	constID        int
	regionID       int
	security       float64
	name           string
	localizedNames map[string]string
	x              float64
	y              float64
	z              float64
	pos2dX         float64
	pos2dY         float64
	hasPos2D       bool
}

func parseSystemRow(row map[string]any) systemRowData {
	x, y, z := getPositionXYZ(row)
	pos2dX, pos2dY, has2d := getPosition2D(row)
	localizedNames, _ := localizedStringMap(row["name"])
	return systemRowData{
		id:             getInt(row, "solarSystemID", "id", "_key"),
		rowID:          getString(row, "id", "_key"),
		constID:        getInt(row, "constellationID"),
		regionID:       getInt(row, "regionID"),
		security:       getFloat(row, "security", "securityStatus"),
		name:           getString(row, "solarSystemName", "name"),
		localizedNames: localizedNames,
		x:              x,
		y:              y,
		z:              z,
		pos2dX:         pos2dX,
		pos2dY:         pos2dY,
		hasPos2D:       has2d,
	}
}

func validateSystemRow(row *systemRowData) string {
	if row.id == 0 {
		return "missing_id"
	}

	if row.constID == 0 || row.regionID == 0 {
		return "missing_region_or_constellation"
	}

	if row.name == "" {
		return "missing_name"
	}
	return ""
}

func (row *systemRowData) coordsMissing() bool {
	return row.x == 0 && row.y == 0 && row.z == 0
}

func (row *systemRowData) payload() map[string]any {
	payload := map[string]any{
		"eve_id":          row.id,
		"name":            row.name,
		"security_status": row.security,
		"constellation":   row.constID,
		"region_id":       row.regionID,
		"dotlan_x":        0,
		"dotlan_y":        0,
		"metro_x":         0,
		"metro_y":         0,
	}

	for _, locale := range eve.SupportedSystemLocales() {
		field, ok := eve.SystemNameField(locale)
		if !ok {
			continue
		}
		payload[field] = row.localizedNames[locale]
	}

	if row.coordsMissing() {
		payload["raw_x"] = nil
		payload["raw_y"] = nil
		payload["raw_z"] = nil
	} else {
		payload["raw_x"] = row.x
		payload["raw_y"] = row.y
		payload["raw_z"] = row.z
	}

	if row.hasPos2D {
		payload["eve2d_x"] = row.pos2dX
		payload["eve2d_y"] = row.pos2dY
	} else {
		payload["eve2d_x"] = nil
		payload["eve2d_y"] = nil
	}
	return payload
}
