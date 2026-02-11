package mapdata

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

const (
	LatestJSONLZip = "https://developers.eveonline.com/static-data/eve-online-static-data-latest-jsonl.zip"
	LatestBuildURL = "https://developers.eveonline.com/static-data/tranquility/latest.jsonl"
)

type SDEImporter struct {
	App    *pocketbase.PocketBase
	Client *http.Client
}

func NewSDEImporter(app *pocketbase.PocketBase) *SDEImporter {
	return &SDEImporter{
		App:    app,
		Client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (s *SDEImporter) NeedsUpdate() (bool, string, error) {
	etag, etagErr := s.fetchETag(context.Background(), LatestJSONLZip)
	if etagErr != nil {
		return true, "", etagErr
	}

	stored := s.getMeta("sde_zip_etag")
	if stored == "" {
		return true, etag, nil
	}
	return stored != etag, etag, nil
}

func (s *SDEImporter) DownloadAndImport(ctx context.Context, etag string) error {
	if etag == "" {
		fetched, fetchErr := s.fetchETag(ctx, LatestJSONLZip)
		if fetchErr == nil {
			etag = fetched
		}
	}

	data, dataErr := s.downloadZip(ctx, LatestJSONLZip)
	if dataErr != nil {
		return dataErr
	}

	files, unzipErr := unzipJSONL(data)
	if unzipErr != nil {
		return unzipErr
	}
	logging.New(s.App).
		WithFields(logging.Fields{
			"sde_zip_bytes": len(data),
			"file_count":    len(files),
		}).
		Info("zip downloaded and unpacked")

	if importErr := s.importRegions(ctx, files["mapRegions.jsonl"]); importErr != nil {
		return importErr
	}
	if importErr := s.importConstellations(ctx, files["mapConstellations.jsonl"]); importErr != nil {
		return importErr
	}
	if importErr := s.importSystems(ctx, files["mapSolarSystems.jsonl"]); importErr != nil {
		return importErr
	}
	if importErr := s.importGates(ctx, files["mapStargates.jsonl"]); importErr != nil {
		return importErr
	}

	build := s.fetchBuildNumber()
	_ = s.saveMeta("last_sde_update", time.Now().UTC().Format(time.RFC3339))
	if etag != "" {
		_ = s.saveMeta("sde_zip_etag", etag)
	}
	if build != "" {
		_ = s.saveMeta("sde_build", build)
	}

	s.logSDECollectionCounts()
	return nil
}

func (s *SDEImporter) logSDECollectionCounts() {
	counts := map[string]int{}
	for _, collection := range []string{
		store.CollectionRegions,
		store.CollectionConstellations,
		store.CollectionSolarSystems,
		store.CollectionGates,
	} {
		records, recordsErr := s.App.FindRecordsByFilter(collection, "", "", 0, 0, nil)
		if recordsErr != nil {
			logging.New(s.App).
				WithErr(recordsErr).
				WithFields(logging.Fields{"collection": collection}).
				Warn("collection count failed")
			continue
		}
		counts[collection] = len(records)
	}
	if len(counts) > 0 {
		logging.New(s.App).
			WithFields(logging.Fields{"collection_counts": counts}).
			Info("collection counts")
	}
}

func (s *SDEImporter) importRegions(ctx context.Context, data []byte) error {
	rows, rowsErr := parseJSONL(data)
	if rowsErr != nil {
		return rowsErr
	}
	var missingID, missingName, missingPos, saved int
	log := logging.New(s.App).WithFields(logging.Fields{
		"sde":       "regions",
		"row_count": len(rows),
	})
	for i, row := range rows {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		regionID := getInt(row, "regionID", "id", "_key")
		if regionID == 0 {
			logging.New(s.App).
				WithFields(logging.Fields{"row_id": getString(row, "id", "_key")}).
				Warn("region missing id")
			missingID++
			continue
		}
		name := getString(row, "regionName", "name")
		if name == "" {
			logging.New(s.App).
				WithFields(logging.Fields{"region_id": regionID}).
				Warn("region missing name")
			missingName++
			continue
		}
		rawX, rawY, _ := getPositionXYZ(row)
		var rawXValue any = rawX
		var rawYValue any = rawY
		if rawX == 0 && rawY == 0 {
			rawXValue = nil
			rawYValue = nil
			missingPos++
		}
		payload := map[string]any{
			"eve_id": regionID,
			"name":   name,
		}
		if rawXValue != nil || rawYValue != nil {
			payload["raw_x"] = rawXValue
			payload["raw_y"] = rawYValue
		}
		if upsertErr := s.upsertNumberRecord(store.CollectionRegions, regionID, payload); upsertErr != nil {
			log.WithErr(upsertErr).
				WithFields(logging.Fields{"region_id": regionID}).
				Error("region upsert failed")
			return upsertErr
		}
		saved++
	}
	log.WithFields(logging.Fields{
		"saved_count":                saved,
		"skipped_missing_id_count":   missingID,
		"skipped_missing_name_count": missingName,
		"missing_position_count":     missingPos,
	}).Info("regions import complete")
	return nil
}

func (s *SDEImporter) importConstellations(ctx context.Context, data []byte) error {
	rows, rowsErr := parseJSONL(data)
	if rowsErr != nil {
		return rowsErr
	}
	var missingID, missingRegion, missingName, saved int
	log := logging.New(s.App).WithFields(logging.Fields{
		"sde":       "constellations",
		"row_count": len(rows),
	})
	for i, row := range rows {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		id := getInt(row, "constellationID", "id", "_key")
		regionID := getInt(row, "regionID")
		name := getString(row, "constellationName", "name")
		if id == 0 {
			logging.New(s.App).
				WithFields(logging.Fields{"row_id": getString(row, "id", "_key")}).
				Warn("constellation missing id")
			missingID++
			continue
		}
		if regionID == 0 {
			logging.New(s.App).
				WithFields(logging.Fields{"constellation_id": id}).
				Warn("constellation missing region")
			missingRegion++
			continue
		}
		if name == "" {
			logging.New(s.App).
				WithFields(logging.Fields{"constellation_id": id}).
				Warn("constellation missing name")
			missingName++
			continue
		}
		if upsertErr := s.upsertNumberRecord(store.CollectionConstellations, id, map[string]any{
			"eve_id":    id,
			"name":      name,
			"region_id": regionID,
		}); upsertErr != nil {
			log.WithErr(upsertErr).
				WithFields(logging.Fields{"constellation_id": id}).
				Error("constellation upsert failed")
			return upsertErr
		}
		saved++
	}
	log.WithFields(logging.Fields{
		"saved_count":                  saved,
		"skipped_missing_id_count":     missingID,
		"skipped_missing_region_count": missingRegion,
		"skipped_missing_name_count":   missingName,
	}).Info("constellations import complete")
	return nil
}

func (s *SDEImporter) importSystems(ctx context.Context, data []byte) error {
	rows, rowsErr := parseJSONL(data)
	if rowsErr != nil {
		return rowsErr
	}
	var missingID, missingRegionConst, missingName, missingCoords, saved int
	var withPos2D, missingPos2D int
	log := logging.New(s.App).WithFields(logging.Fields{
		"sde":       "systems",
		"row_count": len(rows),
	})
	for i, row := range rows {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		id := getInt(row, "solarSystemID", "id", "_key")
		if id == 0 {
			logging.New(s.App).
				WithFields(logging.Fields{"row_id": getString(row, "id", "_key")}).
				Warn("system missing id")
			missingID++
			continue
		}
		constID := getInt(row, "constellationID")
		regionID := getInt(row, "regionID")
		security := getFloat(row, "security", "securityStatus")
		x, y, z := getPositionXYZ(row)
		name := getString(row, "solarSystemName", "name")
		if constID == 0 || regionID == 0 {
			logging.New(s.App).
				WithFields(logging.Fields{
					"system_id":        id,
					"region_id":        regionID,
					"constellation_id": constID,
				}).
				Warn("system missing region or constellation")
			missingRegionConst++
			continue
		}
		if name == "" {
			logging.New(s.App).
				WithFields(logging.Fields{"system_id": id}).
				Warn("system missing name")
			missingName++
			continue
		}

		var posX any = x
		var posY any = y
		var posZ any = z
		if x == 0 && y == 0 && z == 0 {
			missingCoords++
			posX = nil
			posY = nil
			posZ = nil
		}

		var dotlanPosX any = 0
		var dotlanPosY any = 0
		var metroPosX any = 0
		var metroPosY any = 0
		pos2dX, pos2dY, has2d := getPosition2D(row)
		var pos2dXValue any = pos2dX
		var pos2dYValue any = pos2dY
		if !has2d {
			pos2dXValue = nil
			pos2dYValue = nil
			missingPos2D++
		} else {
			withPos2D++
		}

		if upsertErr := s.upsertNumberRecord(store.CollectionSolarSystems, id, map[string]any{
			"eve_id":          id,
			"name":            name,
			"security_status": security,
			"constellation":   constID,
			"region_id":       regionID,
			"raw_x":           posX,
			"raw_y":           posY,
			"raw_z":           posZ,
			"dotlan_x":        dotlanPosX,
			"dotlan_y":        dotlanPosY,
			"metro_x":         metroPosX,
			"metro_y":         metroPosY,
			"eve2d_x":         pos2dXValue,
			"eve2d_y":         pos2dYValue,
		}); upsertErr != nil {
			log.WithErr(upsertErr).
				WithFields(logging.Fields{"system_id": id}).
				Error("system upsert failed")
			return upsertErr
		}
		saved++
	}
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

func (s *SDEImporter) importGates(ctx context.Context, data []byte) error {
	rows, rowsErr := parseJSONL(data)
	if rowsErr != nil {
		return rowsErr
	}
	var missingID, missingFrom, missingTo, skippedExisting, saved int
	log := logging.New(s.App).WithFields(logging.Fields{
		"sde":       "gates",
		"row_count": len(rows),
	})
	for i, row := range rows {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		fromID := getInt(row, "fromSolarSystemID", "solarSystemID")
		toID := getNestedInt(row, "destination", "solarSystemID", "toSolarSystemID")
		if fromID == 0 || toID == 0 {
			logging.New(s.App).
				WithFields(logging.Fields{
					"from_system": fromID,
					"to_system":   toID,
				}).
				Warn("gate missing system id")
			missingID++
			continue
		}
		if s.gateExists(fromID, toID) {
			skippedExisting++
			continue
		}

		fromSystem, fromErr := s.findSystem(fromID)
		if fromErr != nil {
			logging.New(s.App).
				WithFields(logging.Fields{
					"from_system": fromID,
					"to_system":   toID,
				}).
				WithErr(fromErr).
				Warn("gate missing from system")
			missingFrom++
			continue
		}
		toSystem, toErr := s.findSystem(toID)
		if toErr != nil {
			logging.New(s.App).
				WithFields(logging.Fields{
					"from_system": fromID,
					"to_system":   toID,
				}).
				WithErr(toErr).
				Warn("gate missing to system")
			missingTo++
			continue
		}

		record := core.NewRecord(s.collection(store.CollectionGates))
		record.Set("from_solarsystem", fromID)
		record.Set("to_solarsystem", toID)
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
		if saveErr := s.App.Save(record); saveErr != nil {
			log.WithErr(saveErr).
				WithFields(logging.Fields{
					"from_system": fromID,
					"to_system":   toID,
				}).
				Error("gate save failed")
			return saveErr
		}
		saved++
	}
	log.WithFields(logging.Fields{
		"saved_count":                 saved,
		"skipped_missing_id_count":    missingID,
		"skipped_missing_from_count":  missingFrom,
		"skipped_missing_to_count":    missingTo,
		"skipped_existing_gate_count": skippedExisting,
	}).Info("gates import complete")
	return nil
}

func (s *SDEImporter) fetchETag(ctx context.Context, url string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	resp, respErr := s.Client.Do(req)
	if respErr != nil {
		return "", respErr
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", ErrETagRequestFailed
	}
	return resp.Header.Get("ETag"), nil
}

func (s *SDEImporter) downloadZip(ctx context.Context, url string) ([]byte, error) {
	req, reqErr := http.NewRequestWithContext(ctx, "GET", url, nil)
	if reqErr != nil {
		return nil, reqErr
	}
	resp, respErr := s.Client.Do(req)
	if respErr != nil {
		return nil, respErr
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, ErrSDEDownloadFailed
	}

	return io.ReadAll(resp.Body)
}

func unzipJSONL(data []byte) (map[string][]byte, error) {
	reader, readerErr := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if readerErr != nil {
		return nil, readerErr
	}

	needed := map[string][]byte{}
	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, ".jsonl") {
			continue
		}
		if file.Name != "mapRegions.jsonl" &&
			file.Name != "mapConstellations.jsonl" &&
			file.Name != "mapSolarSystems.jsonl" &&
			file.Name != "mapStargates.jsonl" {
			continue
		}

		rc, openErr := file.Open()
		if openErr != nil {
			return nil, openErr
		}
		body, bodyErr := io.ReadAll(rc)
		rc.Close()
		if bodyErr != nil {
			return nil, bodyErr
		}
		needed[file.Name] = body
	}

	if len(needed) == 0 {
		return nil, ErrMapJSONLNotFound
	}
	return needed, nil
}

func parseJSONL(data []byte) ([]map[string]interface{}, error) {
	rows := []map[string]interface{}{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024*20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var payload map[string]interface{}
		if unmarshalErr := json.Unmarshal([]byte(line), &payload); unmarshalErr != nil {
			return nil, unmarshalErr
		}
		rows = append(rows, normalizeJSONL(payload))
	}
	return rows, nil
}

func normalizeJSONL(payload map[string]interface{}) map[string]interface{} {
	if key, ok := payload["_key"]; ok {
		payload["_key"] = key
	}
	if value, ok := payload["_value"]; ok {
		if m, ok := value.(map[string]interface{}); ok {
			for k, v := range m {
				payload[k] = v
			}
		}
	}
	return payload
}

func (s *SDEImporter) collection(name string) *core.Collection {
	coll, _ := s.App.FindCollectionByNameOrId(name)
	return coll
}

func (s *SDEImporter) findSystem(systemID int) (*core.Record, error) {
	records, recordsErr := s.App.FindRecordsByFilter(store.CollectionSolarSystems, "eve_id = {:id}", "", 1, 0, map[string]any{"id": systemID})
	if recordsErr != nil {
		return nil, recordsErr
	}
	if len(records) == 0 {
		return nil, ErrSystemNotFound
	}
	return records[0], nil
}

func (s *SDEImporter) gateExists(fromID int, toID int) bool {
	records, recordsErr := s.App.FindRecordsByFilter(
		store.CollectionGates,
		"(from_solarsystem = {:from} && to_solarsystem = {:to}) || (from_solarsystem = {:to} && to_solarsystem = {:from})",
		"",
		1,
		0,
		map[string]any{"from": fromID, "to": toID},
	)
	if recordsErr != nil {
		return false
	}
	return len(records) > 0
}

func (s *SDEImporter) upsertNumberRecord(collection string, id int, data map[string]any) error {
	records, recordsErr := s.App.FindRecordsByFilter(collection, "eve_id = {:id}", "", 1, 0, map[string]any{"id": id})
	if recordsErr != nil {
		return recordsErr
	}

	var record *core.Record
	coll := s.collection(collection)
	if len(records) > 0 {
		record = records[0]
	} else {
		record = core.NewRecord(coll)
	}

	for key, value := range data {
		record.Set(key, value)
	}
	return s.App.Save(record)
}

func (s *SDEImporter) saveMeta(key string, value string) error {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionSDEMeta)
	if collErr != nil {
		return nil
	}
	records, recordsErr := s.App.FindRecordsByFilter(store.CollectionSDEMeta, "key = {:key}", "", 1, 0, map[string]any{"key": key})
	if recordsErr != nil {
		return recordsErr
	}
	var record *core.Record
	if len(records) > 0 {
		record = records[0]
	} else {
		record = core.NewRecord(coll)
		record.Set("key", key)
	}
	record.Set("value", value)
	record.Set("updated_at", types.NowDateTime())
	return s.App.Save(record)
}

func (s *SDEImporter) getMeta(key string) string {
	records, recordsErr := s.App.FindRecordsByFilter(store.CollectionSDEMeta, "key = {:key}", "", 1, 0, map[string]any{"key": key})
	if recordsErr != nil || len(records) == 0 {
		return ""
	}
	return records[0].GetString("value")
}

func (s *SDEImporter) fetchBuildNumber() string {
	resp, respErr := s.Client.Get(LatestBuildURL)
	if respErr != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ""
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "\"sde\"") {
			continue
		}
		var payload map[string]interface{}
		if unmarshalErr := json.Unmarshal([]byte(line), &payload); unmarshalErr != nil {
			continue
		}
		if payload["_key"] == "sde" {
			if value, ok := payload["_value"].(map[string]interface{}); ok {
				if build, ok := value["build"]; ok {
					return fmt.Sprintf("%v", build)
				}
			}
		}
	}
	return ""
}

func getInt(row map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			switch v := value.(type) {
			case float64:
				return int(v)
			case int:
				return v
			case string:
				num, _ := strconv.Atoi(v)
				return num
			}
		}
	}
	return 0
}

func getFloat(row map[string]interface{}, keys ...string) float64 {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			switch v := value.(type) {
			case float64:
				return v
			case int:
				return float64(v)
			case string:
				num, _ := strconv.ParseFloat(v, 64)
				return num
			}
		}
	}
	return 0
}

func getString(row map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			switch v := value.(type) {
			case string:
				return v
			case map[string]interface{}:
				// Prefer English localization if present.
				if en, ok := v["en"].(string); ok && en != "" {
					return en
				}
				for _, nested := range v {
					if s, ok := nested.(string); ok && s != "" {
						return s
					}
				}
			}
		}
	}
	return ""
}

func getPositionXYZ(row map[string]interface{}) (float64, float64, float64) {
	if value, ok := row["position"]; ok {
		if m, ok := value.(map[string]interface{}); ok {
			x := getFloat(m, "x")
			y := getFloat(m, "y")
			z := getFloat(m, "z")
			return x, y, z
		}
	}
	return getFloat(row, "x"), getFloat(row, "y"), getFloat(row, "z")
}

func getPosition2D(row map[string]interface{}) (float64, float64, bool) {
	if value, ok := row["position2D"]; ok {
		if m, ok := value.(map[string]interface{}); ok {
			return getFloat(m, "x"), getFloat(m, "y"), true
		}
	}
	if value, ok := row["position2d"]; ok {
		if m, ok := value.(map[string]interface{}); ok {
			return getFloat(m, "x"), getFloat(m, "y"), true
		}
	}
	return 0, 0, false
}

func getNestedInt(row map[string]interface{}, first string, keys ...string) int {
	if value, ok := row[first]; ok {
		if m, ok := value.(map[string]interface{}); ok {
			return getInt(m, keys...)
		}
	}
	return 0
}
