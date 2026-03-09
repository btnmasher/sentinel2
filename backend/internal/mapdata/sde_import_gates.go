package mapdata

import (
	"context"
	"io"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

func (s *SDEImporter) importGates(ctx context.Context, r io.Reader) error {
	var missingID, missingFrom, missingTo, skippedExisting, saved int
	rowCount, scanErr := forEachJSONLRow(r, func(i int, row map[string]any) error {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		result, importErr := s.importGateRow(row)
		if importErr != nil {
			s.logger.WithErr(importErr).Error("gate save failed")
			return importErr
		}
		switch result {
		case gateImportSaved:
			saved++
		case gateImportMissingID:
			missingID++
		case gateImportMissingFrom:
			missingFrom++
		case gateImportMissingTo:
			missingTo++
		case gateImportSkippedExisting:
			skippedExisting++
		}
		return nil
	})
	if scanErr != nil {
		return scanErr
	}
	log := s.logger.WithFields(logging.Fields{
		"sde":       "gates",
		"row_count": rowCount,
	})
	log.WithFields(logging.Fields{
		"saved_count":                 saved,
		"skipped_missing_id_count":    missingID,
		"skipped_missing_from_count":  missingFrom,
		"skipped_missing_to_count":    missingTo,
		"skipped_existing_gate_count": skippedExisting,
	}).Info("gates import complete")
	return nil
}

type gateRowData struct {
	fromID int
	toID   int
}

func parseGateRow(row map[string]any) gateRowData {
	return gateRowData{
		fromID: getInt(row, "fromSolarSystemID", "solarSystemID"),
		toID:   getNestedInt(row, "destination", "solarSystemID", "toSolarSystemID"),
	}
}

type gateImportResult string

const (
	gateImportSaved           gateImportResult = "saved"
	gateImportMissingID       gateImportResult = "missing_id"
	gateImportMissingFrom     gateImportResult = "missing_from"
	gateImportMissingTo       gateImportResult = "missing_to"
	gateImportSkippedExisting gateImportResult = "skipped_existing"
)

func (s *SDEImporter) importGateRow(row map[string]any) (gateImportResult, error) {
	gate := parseGateRow(row)
	if gate.fromID == 0 || gate.toID == 0 {
		s.logger.WithFields(logging.Fields{"from_system": gate.fromID, "to_system": gate.toID}).Warn("gate missing system id")
		return gateImportMissingID, nil
	}

	if s.gateExists(gate.fromID, gate.toID) {
		return gateImportSkippedExisting, nil
	}
	fromSystem, toSystem, systemErr := s.gateSystems(gate)
	if systemErr != nil {
		s.logger.
			WithFields(logging.Fields{"from_system": gate.fromID, "to_system": gate.toID}).
			WithErr(systemErr.err).
			Warn(systemErr.message())
		if systemErr.kind == gateSystemErrFrom {
			return gateImportMissingFrom, nil
		}
		return gateImportMissingTo, nil
	}
	record := s.newGateRecord(gate, fromSystem, toSystem)
	if saveErr := s.App.Save(record); saveErr != nil {
		return "", saveErr
	}
	return gateImportSaved, nil
}

type gateSystemErrorKind string

const (
	gateSystemErrFrom gateSystemErrorKind = "from"
	gateSystemErrTo   gateSystemErrorKind = "to"
)

type gateSystemError struct {
	kind gateSystemErrorKind
	err  error
}

func (e gateSystemError) message() string {
	if e.kind == gateSystemErrFrom {
		return "gate missing from system"
	}
	return "gate missing to system"
}

func (s *SDEImporter) gateSystems(gate gateRowData) (fromSystem, toSystem *core.Record, err *gateSystemError) {
	fromSystem, fromErr := s.findSystem(gate.fromID)
	if fromErr != nil {
		return nil, nil, &gateSystemError{kind: gateSystemErrFrom, err: fromErr}
	}
	toSystem, toErr := s.findSystem(gate.toID)
	if toErr != nil {
		return nil, nil, &gateSystemError{kind: gateSystemErrTo, err: toErr}
	}
	return fromSystem, toSystem, nil
}

func (s *SDEImporter) newGateRecord(gate gateRowData, fromSystem, toSystem *core.Record) *core.Record {
	record := core.NewRecord(s.collection(store.CollectionGates))
	record.Set("from_solarsystem", gate.fromID)
	record.Set("to_solarsystem", gate.toID)
	record.Set("from_region", fromSystem.GetInt("region_id"))
	record.Set("to_region", toSystem.GetInt("region_id"))
	record.Set("from_constellation", fromSystem.GetInt("constellation"))
	record.Set("to_constellation", toSystem.GetInt("constellation"))
	record.Set("from_dotlan_x", fromSystem.GetInt("dotlan_x"))
	record.Set("from_dotlan_y", fromSystem.GetInt("dotlan_y"))
	record.Set("to_dotlan_x", toSystem.GetInt("dotlan_x"))
	record.Set("to_dotlan_y", toSystem.GetInt("dotlan_y"))
	record.Set("from_metro_x", fromSystem.GetInt("metro_x"))
	record.Set("from_metro_y", fromSystem.GetInt("metro_y"))
	record.Set("to_metro_x", toSystem.GetInt("metro_x"))
	record.Set("to_metro_y", toSystem.GetInt("metro_y"))
	return record
}
