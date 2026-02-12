package jumpbridges

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

var jumpbridgePattern = regexp.MustCompile(`^(?:(?P<structure_id>\d+)\s+)?(?P<from>\S+)\s*(?:»|->|-->|—>|=>|→)\s*(?P<to>\S+)(?:\s+-\s+.*)?$`)

const (
	maxJumpbridgeDistanceLY = 5.0
	metersPerLightYear      = 9.4607304725808e15
)

type JumpbridgeService struct {
	App *pocketbase.PocketBase
}

func NewJumpbridgeService(app *pocketbase.PocketBase) *JumpbridgeService {
	return &JumpbridgeService{App: app}
}

func (s *JumpbridgeService) UpdateFromLines(lines []string) (int, error) {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionJumpbridges)
	if collErr != nil {
		return 0, collErr
	}

	parsed, parseErr := parseJumpbridgeLines(lines)
	if parseErr != nil {
		return 0, parseErr
	}
	pairs, validationErr := buildJumpbridgePairs(parsed)
	if validationErr != nil {
		return 0, validationErr
	}
	resolvedPairs, resolveErr := s.resolvePairs(pairs)
	if resolveErr != nil {
		return 0, resolveErr
	}

	existing, _ := s.App.FindRecordsByFilter(coll.Name, "", "", 0, 0, nil)
	deleteFailed := 0
	for _, rec := range existing {
		if deleteErr := s.App.Delete(rec); deleteErr != nil {
			deleteFailed++
			logging.New(s.App).
				WithFields(logging.Fields{
					"record_id":  rec.Id,
					"collection": coll.Name,
				}).
				WithErr(deleteErr).
				Debug("jumpbridge delete failed")
		}
	}
	if deleteFailed > 0 {
		logging.New(s.App).
			WithFields(logging.Fields{
				"failed": deleteFailed,
			}).
			Warn("jumpbridge delete failures")
	}

	count := 0
	saveFailed := 0
	for _, pair := range resolvedPairs {
		saved, savedErr := s.savePair(coll, pair.structureID, pair.fromSystem, pair.toSystem)
		if savedErr != nil {
			saveFailed += 2 - saved
			continue
		}
		count += saved
	}
	if saveFailed > 0 {
		logging.New(s.App).
			WithFields(logging.Fields{
				"failed": saveFailed,
			}).
			Warn("jumpbridge save failures")
	}

	return count / 2, nil
}

func (s *JumpbridgeService) AddPair(fromSystemID int, toSystemID int) (bool, error) {
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

	a := int(fromSystem.GetInt("eve_id"))
	b := int(toSystem.GetInt("eve_id"))
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

	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionJumpbridges)
	if collErr != nil {
		return false, collErr
	}
	_, saveErr := s.savePair(coll, "", fromSystem, toSystem)
	if saveErr != nil {
		return false, saveErr
	}
	return true, nil
}

func (s *JumpbridgeService) RemovePair(fromSystemID int, toSystemID int) (int, error) {
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

func (s *JumpbridgeService) UpdatePair(oldFromSystemID int, oldToSystemID int, newFromSystemID int, newToSystemID int) (bool, error) {
	if oldFromSystemID <= 0 || oldToSystemID <= 0 || newFromSystemID <= 0 || newToSystemID <= 0 {
		return false, fmt.Errorf("invalid system id")
	}
	if newFromSystemID == newToSystemID {
		return false, fmt.Errorf("jumpbridge endpoints must be different systems")
	}

	oldKey := pairKey(strconv.Itoa(oldFromSystemID), strconv.Itoa(oldToSystemID))
	newKey := pairKey(strconv.Itoa(newFromSystemID), strconv.Itoa(newToSystemID))
	if oldKey == newKey {
		return false, nil
	}

	existingBySystem, existingPairs, existingErr := s.loadExistingPairState()
	if existingErr != nil {
		return false, existingErr
	}
	if _, exists := existingPairs[oldKey]; !exists {
		return false, fmt.Errorf("original jumpbridge pair was not found")
	}

	// Exclude the old pair from uniqueness checks so one endpoint can be reassigned.
	delete(existingPairs, oldKey)
	if partner, ok := existingBySystem[oldFromSystemID]; ok && partner == oldToSystemID {
		delete(existingBySystem, oldFromSystemID)
	}
	if partner, ok := existingBySystem[oldToSystemID]; ok && partner == oldFromSystemID {
		delete(existingBySystem, oldToSystemID)
	}

	if _, exists := existingPairs[newKey]; exists {
		return false, nil
	}

	fromSystem, fromErr := s.findSystemByEveID(newFromSystemID)
	if fromErr != nil {
		return false, fmt.Errorf("unknown from system: %d", newFromSystemID)
	}
	toSystem, toErr := s.findSystemByEveID(newToSystemID)
	if toErr != nil {
		return false, fmt.Errorf("unknown to system: %d", newToSystemID)
	}

	a := int(fromSystem.GetInt("eve_id"))
	b := int(toSystem.GetInt("eve_id"))
	if partner, ok := existingBySystem[a]; ok && partner != b {
		return false, fmt.Errorf("system already linked: %s", fromSystem.GetString("name"))
	}
	if partner, ok := existingBySystem[b]; ok && partner != a {
		return false, fmt.Errorf("system already linked: %s", toSystem.GetString("name"))
	}
	if !withinMaxDistance(fromSystem, toSystem) {
		return false, fmt.Errorf("jumpbridge distance exceeds %.0f lightyears", maxJumpbridgeDistanceLY)
	}

	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionJumpbridges)
	if collErr != nil {
		return false, collErr
	}

	if _, saveErr := s.savePair(coll, "", fromSystem, toSystem); saveErr != nil {
		return false, saveErr
	}

	if _, removeErr := s.RemovePair(oldFromSystemID, oldToSystemID); removeErr != nil {
		return false, removeErr
	}
	return true, nil
}

type jumpbridgeEntry struct {
	structureID string
	from        string
	to          string
}

type resolvedJumpbridgeEntry struct {
	structureID string
	fromSystem  *core.Record
	toSystem    *core.Record
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
			structureID: data["structure_id"],
			from:        strings.TrimSpace(data["from"]),
			to:          strings.TrimSpace(data["to"]),
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
		if int(fromSystem.GetInt("eve_id")) == int(toSystem.GetInt("eve_id")) {
			return nil, fmt.Errorf("jumpbridge endpoints must be different systems: %s", entry.from)
		}
		if !withinMaxDistance(fromSystem, toSystem) {
			return nil, fmt.Errorf("jumpbridge distance exceeds %.0f lightyears: %s <-> %s", maxJumpbridgeDistanceLY, entry.from, entry.to)
		}
		resolved = append(resolved, resolvedJumpbridgeEntry{
			structureID: entry.structureID,
			fromSystem:  fromSystem,
			toSystem:    toSystem,
		})
	}
	return resolved, nil
}

func pairKey(a string, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

func parseStructureID(raw string, from string, to string) int64 {
	if raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed != 0 {
			return parsed
		}
	}
	if from == "" || to == "" {
		return 0
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(strings.ToLower(from)))
	_, _ = hasher.Write([]byte("->"))
	_, _ = hasher.Write([]byte(strings.ToLower(to)))
	return -int64(hasher.Sum64())
}

func withinMaxDistance(from *core.Record, to *core.Record) bool {
	dx := from.GetFloat("raw_x") - to.GetFloat("raw_x")
	dy := from.GetFloat("raw_y") - to.GetFloat("raw_y")
	dz := from.GetFloat("raw_z") - to.GetFloat("raw_z")
	meters := math.Sqrt(dx*dx + dy*dy + dz*dz)
	lightyears := meters / metersPerLightYear
	return lightyears <= maxJumpbridgeDistanceLY
}

func (s *JumpbridgeService) savePair(coll *core.Collection, structureID string, fromSystem *core.Record, toSystem *core.Record) (int, error) {
	saved := 0
	directions := []struct {
		fromName   string
		toName     string
		fromRecord *core.Record
		toRecord   *core.Record
	}{
		{fromName: fromSystem.GetString("name"), toName: toSystem.GetString("name"), fromRecord: fromSystem, toRecord: toSystem},
		{fromName: toSystem.GetString("name"), toName: fromSystem.GetString("name"), fromRecord: toSystem, toRecord: fromSystem},
	}

	for _, direction := range directions {
		record := core.NewRecord(coll)
		recordStructureID := parseStructureID(structureID, direction.fromName, direction.toName)
		if recordStructureID == 0 {
			continue
		}
		record.Set("structure_id", recordStructureID)
		record.Set("from_solarsystem", direction.fromRecord.GetInt("eve_id"))
		record.Set("to_solarsystem", direction.toRecord.GetInt("eve_id"))
		record.Set("from_region", direction.fromRecord.GetInt("region_id"))
		record.Set("to_region", direction.toRecord.GetInt("region_id"))
		record.Set("is_friendly", true)
		createdAt, _ := types.ParseDateTime(time.Now())
		record.Set("created_date", createdAt)
		if saveErr := s.App.Save(record); saveErr != nil {
			logging.New(s.App).
				WithFields(logging.Fields{
					"structure_id": recordStructureID,
				}).
				WithErr(saveErr).
				Debug("jumpbridge save failed")
			return saved, saveErr
		}
		saved += 1
	}
	return saved, nil
}

func (s *JumpbridgeService) loadExistingPairState() (map[int]int, map[string]struct{}, error) {
	records, recordsErr := s.App.FindRecordsByFilter(store.CollectionJumpbridges, "", "", 0, 0, nil)
	if recordsErr != nil {
		return nil, nil, recordsErr
	}

	bySystem := make(map[int]int)
	pairs := make(map[string]struct{})
	for _, rec := range records {
		from := int(rec.GetInt("from_solarsystem"))
		to := int(rec.GetInt("to_solarsystem"))
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
		key := bridgeKey(int(bridge.GetInt("from_solarsystem")), int(bridge.GetInt("to_solarsystem")))
		pairs[key] = bridge
	}

	deleted := 0
	deleteFailed := 0
	for _, bridge := range bridges {
		from := int(bridge.GetInt("from_solarsystem"))
		to := int(bridge.GetInt("to_solarsystem"))
		if _, ok := pairs[bridgeKey(to, from)]; !ok {
			if deleteErr := s.App.Delete(bridge); deleteErr != nil {
				deleteFailed++
				logging.New(s.App).
					WithFields(logging.Fields{
						"structure_id": bridge.GetInt("structure_id"),
						"record_id":    bridge.Id,
					}).
					WithErr(deleteErr).
					Debug("jumpbridge delete failed")
			} else {
				deleted += 1
			}
		}
	}
	if deleteFailed > 0 {
		logging.New(s.App).
			WithFields(logging.Fields{
				"failed": deleteFailed,
			}).
			Warn("jumpbridge delete failures")
	}

	return deleted
}

func bridgeKey(from int, to int) string {
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
