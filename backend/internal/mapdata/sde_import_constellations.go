package mapdata

import (
	"context"
	"io"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

func (s *SDEImporter) importConstellations(ctx context.Context, r io.Reader) error {
	var missingID, missingRegion, missingName, saved int
	rowCount, scanErr := forEachJSONLRow(r, func(i int, row map[string]any) error {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		id := getInt(row, "constellationID", "id", "_key")
		regionID := getInt(row, "regionID")
		name := getString(row, "constellationName", "name")
		if id == 0 {
			s.logger.
				WithFields(logging.Fields{"row_id": getString(row, "id", "_key")}).
				Warn("constellation missing id")
			missingID++
			return nil
		}
		if regionID == 0 {
			s.logger.
				WithFields(logging.Fields{"constellation_id": id}).
				Warn("constellation missing region")
			missingRegion++
			return nil
		}
		if name == "" {
			s.logger.
				WithFields(logging.Fields{"constellation_id": id}).
				Warn("constellation missing name")
			missingName++
			return nil
		}
		if upsertErr := s.upsertNumberRecord(store.CollectionConstellations, id, map[string]any{
			"eve_id":    id,
			"name":      name,
			"region_id": regionID,
		}); upsertErr != nil {
			s.logger.WithErr(upsertErr).
				WithFields(logging.Fields{"constellation_id": id}).
				Error("constellation upsert failed")
			return upsertErr
		}
		saved++
		return nil
	})
	if scanErr != nil {
		return scanErr
	}
	log := s.logger.WithFields(logging.Fields{
		"sde":       "constellations",
		"row_count": rowCount,
	})
	log.WithFields(logging.Fields{
		"saved_count":                  saved,
		"skipped_missing_id_count":     missingID,
		"skipped_missing_region_count": missingRegion,
		"skipped_missing_name_count":   missingName,
	}).Info("constellations import complete")
	return nil
}
