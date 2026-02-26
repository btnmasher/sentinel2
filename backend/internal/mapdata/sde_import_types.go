package mapdata

import (
	"context"
	"io"
	"strings"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

func (s *SDEImporter) importTypes(ctx context.Context, r io.Reader) error {
	var missingID, missingName, saved, unchanged int
	rowCount, scanErr := forEachJSONLRow(r, func(i int, row map[string]any) error {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		typeID := getInt(row, "typeID", "id", "_key")
		if typeID == 0 {
			missingID++
			return nil
		}
		name := getEnglishString(row, "name", "typeName")
		if name == "" {
			missingName++
			return nil
		}
		changed, upsertErr := s.upsertNumberRecordIfChanged(store.CollectionItemTypes, typeID, map[string]any{
			"eve_id": typeID,
			"name":   name,
		})
		if upsertErr != nil {
			s.logger.WithErr(upsertErr).
				WithFields(logging.Fields{"type_id": typeID}).
				Error("item type upsert failed")
			return upsertErr
		}
		if !changed {
			unchanged++
			return nil
		}
		saved++
		return nil
	})
	if scanErr != nil {
		return scanErr
	}
	log := s.logger.WithFields(logging.Fields{
		"sde":       "types",
		"row_count": rowCount,
	})
	log.WithFields(logging.Fields{
		"saved_count":                saved,
		"skipped_missing_id_count":   missingID,
		"skipped_missing_name_count": missingName,
		"skipped_unchanged_count":    unchanged,
	}).Info("types import complete")
	return nil
}

func getEnglishString(row map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := row[key]
		if !ok {
			continue
		}
		if text, ok := asString(value); ok {
			return text
		}
		labels, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if en, ok := labels["en"].(string); ok {
			return strings.TrimSpace(en)
		}
	}
	return ""
}
