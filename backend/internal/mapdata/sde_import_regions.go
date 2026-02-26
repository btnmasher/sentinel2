package mapdata

import (
	"context"
	"io"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

func (s *SDEImporter) importRegions(ctx context.Context, r io.Reader) error {
	var missingID, missingName, missingPos, saved int
	rowCount, scanErr := forEachJSONLRow(r, func(i int, row map[string]any) error {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		region := parseRegionRow(row)
		switch validateRegionRow(region) {
		case "missing_id":
			s.logger.
				WithFields(logging.Fields{"row_id": region.rowID}).
				Warn("region missing id")
			missingID++
			return nil
		case "missing_name":
			s.logger.
				WithFields(logging.Fields{"region_id": region.regionID}).
				Warn("region missing name")
			missingName++
			return nil
		}
		if region.rawX == 0 && region.rawY == 0 {
			missingPos++
		}
		if upsertErr := s.upsertNumberRecord(store.CollectionRegions, region.regionID, region.payload()); upsertErr != nil {
			s.logger.WithErr(upsertErr).
				WithFields(logging.Fields{"region_id": region.regionID}).
				Error("region upsert failed")
			return upsertErr
		}
		saved++
		return nil
	})
	if scanErr != nil {
		return scanErr
	}
	log := s.logger.WithFields(logging.Fields{
		"sde":       "regions",
		"row_count": rowCount,
	})
	log.WithFields(logging.Fields{
		"saved_count":                saved,
		"skipped_missing_id_count":   missingID,
		"skipped_missing_name_count": missingName,
		"missing_position_count":     missingPos,
	}).Info("regions import complete")
	return nil
}

type regionRowData struct {
	regionID int
	name     string
	rawX     float64
	rawY     float64
	rowID    string
}

func parseRegionRow(row map[string]any) regionRowData {
	rawX, rawY, _ := getPositionXYZ(row)
	return regionRowData{
		regionID: getInt(row, "regionID", "id", "_key"),
		name:     getString(row, "regionName", "name"),
		rawX:     rawX,
		rawY:     rawY,
		rowID:    getString(row, "id", "_key"),
	}
}

func validateRegionRow(row regionRowData) string {
	if row.regionID == 0 {
		return "missing_id"
	}
	if row.name == "" {
		return "missing_name"
	}
	return ""
}

func (row regionRowData) payload() map[string]any {
	payload := map[string]any{
		"eve_id": row.regionID,
		"name":   row.name,
	}
	if row.rawX == 0 && row.rawY == 0 {
		return payload
	}
	payload["raw_x"] = row.rawX
	payload["raw_y"] = row.rawY
	return payload
}
