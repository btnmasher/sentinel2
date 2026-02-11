package jumpbridges

import (
	"database/sql"
	"fmt"
	"hash/fnv"
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
	for _, pair := range pairs {
		fromSystem, fromErr := s.findSystemByName(pair.from)
		if fromErr != nil {
			return 0, fmt.Errorf("unknown from system: %s", pair.from)
		}
		toSystem, toErr := s.findSystemByName(pair.to)
		if toErr != nil {
			return 0, fmt.Errorf("unknown to system: %s", pair.to)
		}

		directions := []struct {
			fromName   string
			toName     string
			fromRecord *core.Record
			toRecord   *core.Record
		}{
			{fromName: pair.from, toName: pair.to, fromRecord: fromSystem, toRecord: toSystem},
			{fromName: pair.to, toName: pair.from, fromRecord: toSystem, toRecord: fromSystem},
		}

		for _, direction := range directions {
			record := core.NewRecord(coll)
			structureID := parseStructureID(pair.structureID, direction.fromName, direction.toName)
			if structureID == 0 {
				continue
			}
			record.Set("structure_id", structureID)
			record.Set("from_solarsystem", direction.fromRecord.GetInt("eve_id"))
			record.Set("to_solarsystem", direction.toRecord.GetInt("eve_id"))
			record.Set("from_region", direction.fromRecord.GetInt("region_id"))
			record.Set("to_region", direction.toRecord.GetInt("region_id"))
			record.Set("is_friendly", true)
			createdAt, _ := types.ParseDateTime(time.Now())
			record.Set("created_date", createdAt)
			if saveErr := s.App.Save(record); saveErr != nil {
				saveFailed++
				logging.New(s.App).
					WithFields(logging.Fields{
						"structure_id": structureID,
					}).
					WithErr(saveErr).
					Debug("jumpbridge save failed")
				continue
			}
			count += 1
		}
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

type jumpbridgeEntry struct {
	structureID string
	from        string
	to          string
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
