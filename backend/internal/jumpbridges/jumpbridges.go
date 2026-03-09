package jumpbridges

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/esi"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

var jumpbridgePattern = regexp.MustCompile(`^(?P<from>\S+)\s*(?:»|->|-->|—>|=>|→)\s*(?P<to>\S+)(?:\s+-\s+.*)?$`)

const (
	maxJumpbridgeDistanceLY = 5.0
	metersPerLightYear      = 9.4607304725808e15
	pairDirectionCount      = 2
)

type JumpbridgeService struct {
	App       *pocketbase.PocketBase
	ESI       esi.ESIClient
	PublicESI *esi.ESIPublicClient
	logger    *logging.Logger
}

type ImportPairFailure struct {
	FromSystemID   int
	ToSystemID     int
	FromSystemName string
	ToSystemName   string
	Reason         string
}

type UpdateFromLinesResult struct {
	LineCount     int
	ParsedPairs   int
	ImportedPairs int
	FailedPairs   int
	Failures      []ImportPairFailure
	Applied       bool
}

func NewJumpbridgeService(app *pocketbase.PocketBase, esiClient esi.ESIClient, publicESI *esi.ESIPublicClient) *JumpbridgeService {
	return &JumpbridgeService{
		App:       app,
		ESI:       esiClient,
		PublicESI: publicESI,
		logger: logging.New(app).WithFields(logging.Fields{
			"component": "jumpbridges",
		}),
	}
}

func (s *JumpbridgeService) UpdateFromLines(lines []string) (int, error) {
	return s.UpdateFromLinesWithContext(context.Background(), lines)
}

func (s *JumpbridgeService) UpdateFromLinesWithContext(ctx context.Context, lines []string) (int, error) {
	result, err := s.UpdateFromLinesDetailedWithContext(ctx, lines)
	if err != nil {
		return 0, err
	}
	return result.ImportedPairs, nil
}

func (s *JumpbridgeService) UpdateFromLinesDetailedWithContext(ctx context.Context, lines []string) (UpdateFromLinesResult, error) {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionJumpbridges)
	if collErr != nil {
		return UpdateFromLinesResult{}, collErr
	}
	parsed, parseErr := parseJumpbridgeLines(lines)
	if parseErr != nil {
		return UpdateFromLinesResult{}, parseErr
	}
	pairs, validationErr := buildJumpbridgePairs(parsed)
	if validationErr != nil {
		return UpdateFromLinesResult{}, validationErr
	}
	resolvedPairs, resolveErr := s.resolvePairs(pairs)
	if resolveErr != nil {
		return UpdateFromLinesResult{}, resolveErr
	}

	result := UpdateFromLinesResult{
		LineCount:   len(lines),
		ParsedPairs: len(resolvedPairs),
		Failures:    make([]ImportPairFailure, 0),
	}
	validatedPairs := s.validateImportPairs(ctx, resolvedPairs, &result)
	if len(validatedPairs) == 0 && len(resolvedPairs) > 0 {
		return result, nil
	}

	s.deleteExistingJumpbridgeRows(coll.Name)
	count := s.saveValidatedJumpbridgePairs(coll, validatedPairs, &result)
	result.ImportedPairs = count / pairDirectionCount
	result.Applied = true
	return result, nil
}

func (s *JumpbridgeService) AddPair(fromSystemID, toSystemID int) (bool, error) {
	return s.AddPairWithContext(context.Background(), fromSystemID, toSystemID)
}

func (s *JumpbridgeService) AddPairWithContext(ctx context.Context, fromSystemID, toSystemID int) (bool, error) {
	if fromSystemID <= 0 || toSystemID <= 0 {
		return false, fmt.Errorf("invalid system id")
	}

	if fromSystemID == toSystemID {
		return false, fmt.Errorf("jumpbridge endpoints must be different systems")
	}

	fromSystem, fromErr := s.findSystemByEveID(fromSystemID)
	if fromErr != nil {
		return false, fmt.Errorf("unknown from system: %d", fromSystemID)
	}
	toSystem, toErr := s.findSystemByEveID(toSystemID)
	if toErr != nil {
		return false, fmt.Errorf("unknown to system: %d", toSystemID)
	}

	existingBySystem, existingPairs, existingErr := s.loadExistingPairState()
	if existingErr != nil {
		return false, existingErr
	}

	a := fromSystem.GetInt("eve_id")
	b := toSystem.GetInt("eve_id")
	key := pairKey(strconv.Itoa(a), strconv.Itoa(b))
	if _, exists := existingPairs[key]; exists {
		return false, nil
	}

	if partner, ok := existingBySystem[a]; ok && partner != b {
		return false, fmt.Errorf("system already linked: %s", fromSystem.GetString("name"))
	}

	if partner, ok := existingBySystem[b]; ok && partner != a {
		return false, fmt.Errorf("system already linked: %s", toSystem.GetString("name"))
	}

	if !withinMaxDistance(fromSystem, toSystem) {
		return false, fmt.Errorf("jumpbridge distance exceeds %.0f lightyears", maxJumpbridgeDistanceLY)
	}
	fromStructureID, toStructureID, validateErr := s.resolvePairStructureIDsWithAllowedCharacters(ctx, fromSystem, toSystem, 0, 0)
	if validateErr != nil {
		return false, validateErr
	}

	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionJumpbridges)
	if collErr != nil {
		return false, collErr
	}
	_, saveErr := s.savePair(coll, fromStructureID, toStructureID, fromSystem, toSystem)
	if saveErr != nil {
		return false, saveErr
	}
	return true, nil
}

func (s *JumpbridgeService) RemovePair(fromSystemID, toSystemID int) (int, error) {
	if fromSystemID <= 0 || toSystemID <= 0 {
		return 0, fmt.Errorf("invalid system id")
	}
	filter := "(from_solarsystem = {:a} && to_solarsystem = {:b}) || (from_solarsystem = {:b} && to_solarsystem = {:a})"
	records, recordsErr := s.App.FindRecordsByFilter(
		store.CollectionJumpbridges,
		filter,
		"",
		0,
		0,
		map[string]any{"a": fromSystemID, "b": toSystemID},
	)
	if recordsErr != nil {
		return 0, recordsErr
	}

	deleted := 0
	for _, rec := range records {
		if deleteErr := s.App.Delete(rec); deleteErr != nil {
			return deleted, deleteErr
		}
		deleted += 1
	}
	return deleted, nil
}

func (s *JumpbridgeService) UpdatePair(oldFromSystemID, oldToSystemID, newFromSystemID, newToSystemID int) (bool, error) {
	return s.UpdatePairWithContext(context.Background(), oldFromSystemID, oldToSystemID, newFromSystemID, newToSystemID)
}

func (s *JumpbridgeService) UpdatePairWithContext(ctx context.Context, oldFromSystemID, oldToSystemID, newFromSystemID, newToSystemID int) (bool, error) {
	oldKey, newKey, validateErr := validateUpdatePairInput(oldFromSystemID, oldToSystemID, newFromSystemID, newToSystemID)
	if validateErr != nil || oldKey == newKey {
		return false, validateErr
	}
	existingBySystem, existingPairs, stateErr := s.updatePairState(oldFromSystemID, oldToSystemID, oldKey)
	if stateErr != nil {
		return false, stateErr
	}

	if _, exists := existingPairs[newKey]; exists {
		return false, nil
	}
	fromSystem, toSystem, pairErr := s.resolveUpdatePairEndpoints(existingBySystem, newFromSystemID, newToSystemID)
	if pairErr != nil {
		return false, pairErr
	}
	fromStructureID, toStructureID, validateErr := s.resolvePairStructureIDsWithAllowedCharacters(ctx, fromSystem, toSystem, 0, 0)
	if validateErr != nil {
		return false, validateErr
	}

	if saveErr := s.replacePair(oldFromSystemID, oldToSystemID, fromSystem, toSystem, fromStructureID, toStructureID); saveErr != nil {
		return false, saveErr
	}
	return true, nil
}

func validateUpdatePairInput(oldFromSystemID, oldToSystemID, newFromSystemID, newToSystemID int) (oldKey, newKey string, err error) {
	if oldFromSystemID <= 0 || oldToSystemID <= 0 || newFromSystemID <= 0 || newToSystemID <= 0 {
		return "", "", fmt.Errorf("invalid system id")
	}

	if newFromSystemID == newToSystemID {
		return "", "", fmt.Errorf("jumpbridge endpoints must be different systems")
	}
	oldKey = pairKey(strconv.Itoa(oldFromSystemID), strconv.Itoa(oldToSystemID))
	newKey = pairKey(strconv.Itoa(newFromSystemID), strconv.Itoa(newToSystemID))
	return oldKey, newKey, nil
}

func (s *JumpbridgeService) updatePairState(oldFromSystemID, oldToSystemID int, oldKey string) (existingBySystem map[int]int, existingPairs map[string]struct{}, err error) {
	existingBySystem, existingPairs, existingErr := s.loadExistingPairState()
	if existingErr != nil {
		return nil, nil, existingErr
	}

	if _, exists := existingPairs[oldKey]; !exists {
		return nil, nil, fmt.Errorf("original jumpbridge pair was not found")
	}
	// Exclude the old pair from uniqueness checks so one endpoint can be reassigned.
	delete(existingPairs, oldKey)
	if partner, ok := existingBySystem[oldFromSystemID]; ok && partner == oldToSystemID {
		delete(existingBySystem, oldFromSystemID)
	}

	if partner, ok := existingBySystem[oldToSystemID]; ok && partner == oldFromSystemID {
		delete(existingBySystem, oldToSystemID)
	}
	return existingBySystem, existingPairs, nil
}

func (s *JumpbridgeService) resolveUpdatePairEndpoints(existingBySystem map[int]int, newFromSystemID, newToSystemID int) (fromSystem, toSystem *core.Record, err error) {
	fromSystem, fromErr := s.findSystemByEveID(newFromSystemID)
	if fromErr != nil {
		return nil, nil, fmt.Errorf("unknown from system: %d", newFromSystemID)
	}
	toSystem, toErr := s.findSystemByEveID(newToSystemID)
	if toErr != nil {
		return nil, nil, fmt.Errorf("unknown to system: %d", newToSystemID)
	}
	a := fromSystem.GetInt("eve_id")
	b := toSystem.GetInt("eve_id")
	if partner, ok := existingBySystem[a]; ok && partner != b {
		return nil, nil, fmt.Errorf("system already linked: %s", fromSystem.GetString("name"))
	}

	if partner, ok := existingBySystem[b]; ok && partner != a {
		return nil, nil, fmt.Errorf("system already linked: %s", toSystem.GetString("name"))
	}

	if !withinMaxDistance(fromSystem, toSystem) {
		return nil, nil, fmt.Errorf("jumpbridge distance exceeds %.0f lightyears", maxJumpbridgeDistanceLY)
	}
	return fromSystem, toSystem, nil
}

func (s *JumpbridgeService) replacePair(oldFromSystemID, oldToSystemID int, fromSystem, toSystem *core.Record, fromStructureID, toStructureID int64) error {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionJumpbridges)
	if collErr != nil {
		return collErr
	}

	if _, saveErr := s.savePair(coll, fromStructureID, toStructureID, fromSystem, toSystem); saveErr != nil {
		return saveErr
	}
	_, removeErr := s.RemovePair(oldFromSystemID, oldToSystemID)
	return removeErr
}

type jumpbridgeEntry struct {
	from string
	to   string
}

type resolvedJumpbridgeEntry struct {
	fromStructureID int64
	toStructureID   int64
	fromSystem      *core.Record
	toSystem        *core.Record
}

func parseJumpbridgeLines(lines []string) ([]jumpbridgeEntry, error) {
	entries := make([]jumpbridgeEntry, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		match := jumpbridgePattern.FindStringSubmatch(trimmed)
		if match == nil {
			return nil, fmt.Errorf("invalid jumpbridge line: %s", trimmed)
		}
		data := map[string]string{}
		for i, name := range jumpbridgePattern.SubexpNames() {
			if i == 0 || name == "" {
				continue
			}
			data[name] = match[i]
		}
		entries = append(entries, jumpbridgeEntry{
			from: strings.TrimSpace(data["from"]),
			to:   strings.TrimSpace(data["to"]),
		})
	}
	return entries, nil
}

func buildJumpbridgePairs(entries []jumpbridgeEntry) ([]jumpbridgeEntry, error) {
	pairs := map[string]jumpbridgeEntry{}
	pairedWith := map[string]string{}
	for _, entry := range entries {
		if entry.from == "" || entry.to == "" {
			return nil, fmt.Errorf("invalid jumpbridge entry")
		}
		keyFrom := strings.ToLower(entry.from)
		keyTo := strings.ToLower(entry.to)
		if keyFrom == keyTo {
			return nil, fmt.Errorf("jumpbridge endpoints must be different systems: %s", entry.from)
		}
		pairKey := pairKey(keyFrom, keyTo)
		if partner, ok := pairedWith[keyFrom]; ok && partner != keyTo {
			return nil, fmt.Errorf("system appears in multiple pairs: %s", entry.from)
		}
		if partner, ok := pairedWith[keyTo]; ok && partner != keyFrom {
			return nil, fmt.Errorf("system appears in multiple pairs: %s", entry.to)
		}

		pairedWith[keyFrom] = keyTo
		pairedWith[keyTo] = keyFrom
		if _, ok := pairs[pairKey]; !ok {
			pairs[pairKey] = entry
		}
	}
	out := make([]jumpbridgeEntry, 0, len(pairs))
	for _, entry := range pairs {
		out = append(out, entry)
	}
	return out, nil
}

func (s *JumpbridgeService) resolvePairs(entries []jumpbridgeEntry) ([]resolvedJumpbridgeEntry, error) {
	resolved := make([]resolvedJumpbridgeEntry, 0, len(entries))
	for _, entry := range entries {
		fromSystem, fromErr := s.findSystemByName(entry.from)
		if fromErr != nil {
			return nil, fmt.Errorf("unknown from system: %s", entry.from)
		}
		toSystem, toErr := s.findSystemByName(entry.to)
		if toErr != nil {
			return nil, fmt.Errorf("unknown to system: %s", entry.to)
		}
		if fromSystem.GetInt("eve_id") == toSystem.GetInt("eve_id") {
			return nil, fmt.Errorf("jumpbridge endpoints must be different systems: %s", entry.from)
		}
		if !withinMaxDistance(fromSystem, toSystem) {
			return nil, fmt.Errorf("jumpbridge distance exceeds %.0f lightyears: %s <-> %s", maxJumpbridgeDistanceLY, entry.from, entry.to)
		}
		resolved = append(resolved, resolvedJumpbridgeEntry{
			fromSystem: fromSystem,
			toSystem:   toSystem,
		})
	}
	return resolved, nil
}

func pairKey(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

func withinMaxDistance(from, to *core.Record) bool {
	dx := from.GetFloat("raw_x") - to.GetFloat("raw_x")
	dy := from.GetFloat("raw_y") - to.GetFloat("raw_y")
	dz := from.GetFloat("raw_z") - to.GetFloat("raw_z")
	meters := math.Sqrt(dx*dx + dy*dy + dz*dz)
	lightyears := meters / metersPerLightYear
	return lightyears <= maxJumpbridgeDistanceLY
}

func (s *JumpbridgeService) savePair(
	coll *core.Collection,
	fromStructureID int64,
	toStructureID int64,
	fromSystem,
	toSystem *core.Record,
) (int, error) {
	saved := 0
	directions := []struct {
		fromRecord *core.Record
		toRecord   *core.Record
		fromID     int64
		toID       int64
	}{
		{fromRecord: fromSystem, toRecord: toSystem, fromID: fromStructureID, toID: toStructureID},
		{fromRecord: toSystem, toRecord: fromSystem, fromID: toStructureID, toID: fromStructureID},
	}

	for _, direction := range directions {
		record := core.NewRecord(coll)
		recordFromStructureID := direction.fromID
		recordToStructureID := direction.toID
		record.Set("from_structure_id", recordFromStructureID)
		record.Set("to_structure_id", recordToStructureID)
		record.Set("from_solarsystem", direction.fromRecord.GetInt("eve_id"))
		record.Set("to_solarsystem", direction.toRecord.GetInt("eve_id"))
		record.Set("from_region", direction.fromRecord.GetInt("region_id"))
		record.Set("to_region", direction.toRecord.GetInt("region_id"))
		record.Set("is_friendly", true)
		createdAt, _ := types.ParseDateTime(time.Now())
		record.Set("created_date", createdAt)
		if saveErr := s.App.Save(record); saveErr != nil {
			s.logger.
				WithFields(logging.Fields{
					"from_structure_id": recordFromStructureID,
					"to_structure_id":   recordToStructureID,
				}).
				WithErr(saveErr).
				Debug("jumpbridge save failed")
			return saved, saveErr
		}
		saved += 1
	}
	return saved, nil
}

func (s *JumpbridgeService) loadExistingPairState() (bySystem map[int]int, pairs map[string]struct{}, err error) {
	records, recordsErr := s.App.FindRecordsByFilter(store.CollectionJumpbridges, "", "", 0, 0, nil)
	if recordsErr != nil {
		return nil, nil, recordsErr
	}

	bySystem = make(map[int]int)
	pairs = make(map[string]struct{})
	for _, rec := range records {
		from := rec.GetInt("from_solarsystem")
		to := rec.GetInt("to_solarsystem")
		if from <= 0 || to <= 0 || from == to {
			continue
		}
		pairs[pairKey(strconv.Itoa(from), strconv.Itoa(to))] = struct{}{}
		if _, exists := bySystem[from]; !exists {
			bySystem[from] = to
		}
	}
	return bySystem, pairs, nil
}

func (s *JumpbridgeService) removeSingles() int {
	bridges, bridgesErr := s.App.FindRecordsByFilter(store.CollectionJumpbridges, "", "", 0, 0, nil)
	if bridgesErr != nil {
		return 0
	}

	pairs := map[string]*core.Record{}
	for _, bridge := range bridges {
		key := bridgeKey(bridge.GetInt("from_solarsystem"), bridge.GetInt("to_solarsystem"))
		pairs[key] = bridge
	}

	deleted := 0
	deleteFailed := 0
	for _, bridge := range bridges {
		from := bridge.GetInt("from_solarsystem")
		to := bridge.GetInt("to_solarsystem")
		if _, ok := pairs[bridgeKey(to, from)]; !ok {
			if deleteErr := s.App.Delete(bridge); deleteErr != nil {
				deleteFailed++
				s.logger.
					WithFields(logging.Fields{
						"from_structure_id": bridge.GetInt("from_structure_id"),
						"to_structure_id":   bridge.GetInt("to_structure_id"),
						"record_id":         bridge.Id,
					}).
					WithErr(deleteErr).
					Debug("jumpbridge delete failed")
			} else {
				deleted += 1
			}
		}
	}

	if deleteFailed > 0 {
		s.logger.
			WithFields(logging.Fields{
				"failed": deleteFailed,
			}).
			Warn("jumpbridge delete failures")
	}

	return deleted
}

func bridgeKey(from, to int) string {
	return strconv.Itoa(from) + "->" + strconv.Itoa(to)
}

func (s *JumpbridgeService) findSystemByName(name string) (*core.Record, error) {
	records, recordsErr := s.App.FindRecordsByFilter(store.CollectionSolarSystems, "name = {:name}", "", 1, 0, map[string]any{"name": name})
	if recordsErr != nil {
		return nil, recordsErr
	}

	if len(records) == 0 {
		lowerName := strings.ToLower(name)
		records, recordsErr = s.App.FindRecordsByFilter(
			store.CollectionSolarSystems,
			"lower(name) = {:name}",
			"",
			1,
			0,
			map[string]any{"name": lowerName},
		)
		if recordsErr != nil {
			return nil, recordsErr
		}
		if len(records) == 0 {
			return nil, sql.ErrNoRows
		}
	}
	return records[0], nil
}

func (s *JumpbridgeService) findSystemByEveID(id int) (*core.Record, error) {
	records, recordsErr := s.App.FindRecordsByFilter(
		store.CollectionSolarSystems,
		"eve_id = {:id}",
		"",
		1,
		0,
		map[string]any{"id": id},
	)
	if recordsErr != nil {
		return nil, recordsErr
	}

	if len(records) == 0 {
		return nil, sql.ErrNoRows
	}
	return records[0], nil
}

func (s *JumpbridgeService) validateImportPairs(
	ctx context.Context,
	resolvedPairs []resolvedJumpbridgeEntry,
	result *UpdateFromLinesResult,
) []resolvedJumpbridgeEntry {
	validatedPairs := make([]resolvedJumpbridgeEntry, 0, len(resolvedPairs))
	for _, pair := range resolvedPairs {
		fromName := strings.TrimSpace(pair.fromSystem.GetString("name"))
		toName := strings.TrimSpace(pair.toSystem.GetString("name"))
		fromID := pair.fromSystem.GetInt("eve_id")
		toID := pair.toSystem.GetInt("eve_id")
		fromStructureID, toStructureID, validateErr := s.resolvePairStructureIDsWithAllowedCharacters(
			ctx,
			pair.fromSystem,
			pair.toSystem,
			pair.fromStructureID,
			pair.toStructureID,
		)
		if validateErr != nil {
			// ESI structure validation is advisory for manual imports.
			// We keep the pair and report the validation issue in the response.
			result.FailedPairs++
			result.Failures = append(result.Failures, ImportPairFailure{
				FromSystemID:   fromID,
				ToSystemID:     toID,
				FromSystemName: fromName,
				ToSystemName:   toName,
				Reason:         validateErr.Error(),
			})
			s.logger.WithFields(logging.Fields{
				"from_system_id":   fromID,
				"to_system_id":     toID,
				"from_system_name": fromName,
				"to_system_name":   toName,
				"error":            validateErr.Error(),
			}).Warn("jumpbridge import validation failed; importing pair without structure ids")
			validatedPairs = append(validatedPairs, pair)
			continue
		}
		pair.fromStructureID = fromStructureID
		pair.toStructureID = toStructureID
		validatedPairs = append(validatedPairs, pair)
	}
	return validatedPairs
}

func (s *JumpbridgeService) deleteExistingJumpbridgeRows(collectionName string) {
	existing, _ := s.App.FindRecordsByFilter(collectionName, "", "", 0, 0, nil)
	deleteFailed := 0
	for _, rec := range existing {
		if deleteErr := s.App.Delete(rec); deleteErr != nil {
			deleteFailed++
			s.logger.
				WithFields(logging.Fields{
					"record_id":  rec.Id,
					"collection": collectionName,
				}).
				WithErr(deleteErr).
				Debug("jumpbridge delete failed")
		}
	}

	if deleteFailed > 0 {
		s.logger.WithFields(logging.Fields{"failed": deleteFailed}).Warn("jumpbridge delete failures")
	}
}

func (s *JumpbridgeService) saveValidatedJumpbridgePairs(
	coll *core.Collection,
	validatedPairs []resolvedJumpbridgeEntry,
	result *UpdateFromLinesResult,
) int {
	count := 0
	saveFailed := 0
	for _, pair := range validatedPairs {
		saved, savedErr := s.savePair(coll, pair.fromStructureID, pair.toStructureID, pair.fromSystem, pair.toSystem)
		if savedErr != nil {
			saveFailed += pairDirectionCount - saved
			result.FailedPairs++
			result.Failures = append(result.Failures, ImportPairFailure{
				FromSystemID:   pair.fromSystem.GetInt("eve_id"),
				ToSystemID:     pair.toSystem.GetInt("eve_id"),
				FromSystemName: strings.TrimSpace(pair.fromSystem.GetString("name")),
				ToSystemName:   strings.TrimSpace(pair.toSystem.GetString("name")),
				Reason:         savedErr.Error(),
			})
			continue
		}
		count += saved
	}

	if saveFailed > 0 {
		s.logger.WithFields(logging.Fields{"failed": saveFailed}).Warn("jumpbridge save failures")
	}
	return count
}
