package mapdata

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"os"
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
	LatestJSONLZip    = "https://developers.eveonline.com/static-data/eve-online-static-data-latest-jsonl.zip"
	LatestBuildURL    = "https://developers.eveonline.com/static-data/tranquility/latest.jsonl"
	sdeHTTPTimeout    = 120 * time.Second
	jsonlScanBufBytes = 1024 * 1024
	jsonlScanMaxBytes = 1024 * 1024 * 20
)

type SDEImporter struct {
	App    *pocketbase.PocketBase
	Client *http.Client
	logger *logging.Logger
	cache  sdeNameCache
}

type sdeNameCache struct {
	systemNames map[int]string
	planetNames map[int]string
}

type SDEImportFileResult struct {
	Name    string
	Skipped bool
	Reason  string
}

type SDEImportReport struct {
	Files []SDEImportFileResult
}

func NewSDEImporter(app *pocketbase.PocketBase) *SDEImporter {
	return &SDEImporter{
		App:    app,
		Client: &http.Client{Timeout: sdeHTTPTimeout},
		logger: logging.New(app).WithFields(logging.Fields{
			"component": "sde_importer",
		}),
		cache: sdeNameCache{
			systemNames: map[int]string{},
			planetNames: map[int]string{},
		},
	}
}

func (s *SDEImporter) NeedsUpdate(ctx context.Context) (needs bool, etag string, err error) {
	etag, etagErr := s.fetchETag(ctx, LatestJSONLZip)
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
	_, err := s.DownloadAndImportWithReport(ctx, etag)
	return err
}

func (s *SDEImporter) DownloadAndImportWithReport(ctx context.Context, etag string) (SDEImportReport, error) {
	s.resetNameCache()
	report := SDEImportReport{}

	if etag == "" {
		fetched, fetchErr := s.fetchETag(ctx, LatestJSONLZip)
		if fetchErr == nil {
			etag = fetched
		}
	}

	zipPath, loadErr := s.downloadSDEZip(ctx)
	if loadErr != nil {
		return report, loadErr
	}
	defer func() { _ = os.Remove(zipPath) }()

	skipped, reason, importErr := s.importJSONLIfChanged(ctx, zipPath, "mapRegions.jsonl", func(r io.Reader) error {
		return s.importRegions(ctx, r)
	})
	if importErr != nil {
		return report, importErr
	}
	report.Files = append(report.Files, SDEImportFileResult{Name: "mapRegions.jsonl", Skipped: skipped, Reason: reason})

	skipped, reason, importErr = s.importJSONLIfChanged(ctx, zipPath, "mapConstellations.jsonl", func(r io.Reader) error {
		return s.importConstellations(ctx, r)
	})
	if importErr != nil {
		return report, importErr
	}
	report.Files = append(report.Files, SDEImportFileResult{Name: "mapConstellations.jsonl", Skipped: skipped, Reason: reason})

	skipped, reason, importErr = s.importJSONLIfChanged(ctx, zipPath, "mapSolarSystems.jsonl", func(r io.Reader) error {
		return s.importSystems(ctx, r)
	})
	if importErr != nil {
		return report, importErr
	}
	report.Files = append(report.Files, SDEImportFileResult{Name: "mapSolarSystems.jsonl", Skipped: skipped, Reason: reason})

	skipped, reason, importErr = s.importJSONLIfChanged(ctx, zipPath, "mapStargates.jsonl", func(r io.Reader) error {
		return s.importGates(ctx, r)
	})
	if importErr != nil {
		return report, importErr
	}
	report.Files = append(report.Files, SDEImportFileResult{Name: "mapStargates.jsonl", Skipped: skipped, Reason: reason})

	skipped, reason, importErr = s.importJSONLIfChanged(ctx, zipPath, "types.jsonl", func(r io.Reader) error {
		return s.importTypes(ctx, r)
	})
	if importErr != nil {
		return report, importErr
	}
	report.Files = append(report.Files, SDEImportFileResult{Name: "types.jsonl", Skipped: skipped, Reason: reason})

	build := s.fetchBuildNumber(ctx)
	_ = s.saveMeta("last_sde_update", time.Now().UTC().Format(time.RFC3339))
	if etag != "" {
		_ = s.saveMeta("sde_zip_etag", etag)
	}

	if build != "" {
		_ = s.saveMeta("sde_build", build)
	}

	s.logSDECollectionCounts()
	return report, nil
}

func (s *SDEImporter) ImportPlanetsFromLatest(ctx context.Context) error {
	zipPath, loadErr := s.downloadSDEZip(ctx)
	if loadErr != nil {
		return loadErr
	}
	defer func() { _ = os.Remove(zipPath) }()
	_, _, importErr := s.importJSONLIfChanged(ctx, zipPath, "mapPlanets.jsonl", func(r io.Reader) error {
		return s.importPlanets(ctx, r)
	})
	return importErr
}

func (s *SDEImporter) ImportMoonsFromLatest(ctx context.Context) error {
	zipPath, loadErr := s.downloadSDEZip(ctx)
	if loadErr != nil {
		return loadErr
	}
	defer func() { _ = os.Remove(zipPath) }()
	_, _, importErr := s.importJSONLIfChanged(ctx, zipPath, "mapMoons.jsonl", func(r io.Reader) error {
		return s.importMoons(ctx, r)
	})
	return importErr
}

func (s *SDEImporter) ImportTypesFromLatest(ctx context.Context) error {
	zipPath, loadErr := s.downloadSDEZip(ctx)
	if loadErr != nil {
		return loadErr
	}
	defer func() { _ = os.Remove(zipPath) }()
	_, _, importErr := s.importJSONLIfChanged(ctx, zipPath, "types.jsonl", func(r io.Reader) error {
		return s.importTypes(ctx, r)
	})
	return importErr
}

func (s *SDEImporter) ShouldImportJSONLFromLatest(ctx context.Context, name string) (shouldImport bool, reason string, err error) {
	if ctx.Err() != nil {
		return false, "", ctx.Err()
	}
	zipPath, loadErr := s.downloadSDEZip(ctx)
	if loadErr != nil {
		return false, "", loadErr
	}
	defer func() { _ = os.Remove(zipPath) }()

	hash, hashErr := hashJSONLInZip(zipPath, name)
	if hashErr != nil {
		return false, "", hashErr
	}

	metaKey := sdeFileHashMetaKey(name)
	previousHash := strings.TrimSpace(s.getMeta(metaKey))
	if previousHash != "" && previousHash == hash {
		return false, "skipped (hash unchanged)", nil
	}
	return true, "", nil
}

func (s *SDEImporter) importJSONLIfChanged(ctx context.Context, zipPath, name string, importFn func(io.Reader) error) (skipped bool, reason string, err error) {
	if ctx.Err() != nil {
		return false, "", ctx.Err()
	}
	hash, hashErr := hashJSONLInZip(zipPath, name)
	if hashErr != nil {
		return false, "", hashErr
	}

	metaKey := sdeFileHashMetaKey(name)
	previousHash := strings.TrimSpace(s.getMeta(metaKey))
	if previousHash != "" && previousHash == hash {
		reason := "skipped (hash unchanged)"
		s.logger.
			WithFields(logging.Fields{
				"file": name,
				"hash": hash,
			}).
			Info("skipping unchanged sde file import")
		return true, reason, nil
	}

	if importErr := withJSONLFileFromZip(zipPath, name, importFn); importErr != nil {
		return false, "", importErr
	}

	if saveErr := s.saveMeta(metaKey, hash); saveErr != nil {
		s.logger.
			WithErr(saveErr).
			WithFields(logging.Fields{"file": name}).
			Warn("failed to persist sde file hash meta")
	}
	return false, "", nil
}

func sdeFileHashMetaKey(name string) string {
	sanitized := strings.ToLower(strings.TrimSpace(name))
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	return "sde_file_hash_" + sanitized
}

func (s *SDEImporter) downloadSDEZip(ctx context.Context) (string, error) {
	req, reqErr := http.NewRequestWithContext(ctx, "GET", LatestJSONLZip, http.NoBody)
	if reqErr != nil {
		return "", reqErr
	}
	resp, respErr := s.Client.Do(req)
	if respErr != nil {
		return "", respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		return "", ErrSDEDownloadFailed
	}

	tmp, tmpErr := os.CreateTemp("", "sentinel2-sde-*.zip")
	if tmpErr != nil {
		return "", tmpErr
	}
	tmpPath := tmp.Name()

	written, copyErr := io.Copy(tmp, resp.Body)
	closeErr := tmp.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return "", copyErr
	}

	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return "", closeErr
	}

	s.logger.
		WithFields(logging.Fields{
			"sde_zip_bytes": written,
			"zip_path":      tmpPath,
		}).
		Info("zip downloaded to temp file")
	return tmpPath, nil
}

func (s *SDEImporter) logSDECollectionCounts() {
	counts := map[string]int{}
	for _, collection := range []string{
		store.CollectionRegions,
		store.CollectionConstellations,
		store.CollectionSolarSystems,
		store.CollectionPlanets,
		store.CollectionMoons,
		store.CollectionGates,
		store.CollectionItemTypes,
	} {
		records, recordsErr := s.App.FindRecordsByFilter(collection, "", "", 0, 0, nil)
		if recordsErr != nil {
			s.logger.
				WithErr(recordsErr).
				WithFields(logging.Fields{"collection": collection}).
				Warn("collection count failed")
			continue
		}
		counts[collection] = len(records)
	}

	if len(counts) > 0 {
		s.logger.
			WithFields(logging.Fields{"collection_counts": counts}).
			Info("collection counts")
	}
}

func (s *SDEImporter) fetchETag(ctx context.Context, url string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "HEAD", url, http.NoBody)
	resp, respErr := s.Client.Do(req)
	if respErr != nil {
		return "", respErr
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusBadRequest {
		return "", ErrETagRequestFailed
	}
	return resp.Header.Get("ETag"), nil
}

func withJSONLFileFromZip(zipPath, name string, fn func(io.Reader) error) error {
	if fn == nil {
		return nil
	}

	if !isSupportedJSONLFile(name) {
		return ErrMapJSONLNotFound
	}
	reader, readerErr := zip.OpenReader(zipPath)
	if readerErr != nil {
		return readerErr
	}
	defer func() { _ = reader.Close() }()

	for _, file := range reader.File {
		if file.Name != name {
			continue
		}

		rc, openErr := file.Open()
		if openErr != nil {
			return openErr
		}
		callErr := fn(rc)
		closeErr := rc.Close()
		if callErr != nil {
			return callErr
		}
		return closeErr
	}

	return ErrMapJSONLNotFound
}

func hashJSONLInZip(zipPath, name string) (string, error) {
	hash := sha256.New()
	if err := withJSONLFileFromZip(zipPath, name, func(r io.Reader) error {
		_, copyErr := io.Copy(hash, r)
		return copyErr
	}); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func isSupportedJSONLFile(name string) bool {
	switch name {
	case "mapRegions.jsonl", "mapConstellations.jsonl", "mapSolarSystems.jsonl", "mapPlanets.jsonl", "mapMoons.jsonl", "mapStargates.jsonl", "types.jsonl":
		return true
	default:
		return false
	}
}

func forEachJSONLRow(r io.Reader, fn func(i int, row map[string]any) error) (int, error) {
	if fn == nil {
		return 0, nil
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, jsonlScanBufBytes), jsonlScanMaxBytes)
	rowIndex := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var payload map[string]any
		if unmarshalErr := json.Unmarshal([]byte(line), &payload); unmarshalErr != nil {
			return rowIndex, unmarshalErr
		}
		if rowErr := fn(rowIndex, normalizeJSONL(payload)); rowErr != nil {
			return rowIndex + 1, rowErr
		}
		rowIndex++
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return rowIndex, scanErr
	}
	return rowIndex, nil
}

func normalizeJSONL(payload map[string]any) map[string]any {
	if key, ok := payload["_key"]; ok {
		payload["_key"] = key
	}

	if value, ok := payload["_value"]; ok {
		if m, ok := value.(map[string]any); ok {
			maps.Copy(payload, m)
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

func (s *SDEImporter) findPlanet(planetID int) (*core.Record, error) {
	records, recordsErr := s.App.FindRecordsByFilter(store.CollectionPlanets, "eve_id = {:id}", "", 1, 0, map[string]any{"id": planetID})
	if recordsErr != nil {
		return nil, recordsErr
	}

	if len(records) == 0 {
		return nil, ErrSystemNotFound
	}
	return records[0], nil
}

func (s *SDEImporter) gateExists(fromID, toID int) bool {
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

func (s *SDEImporter) upsertNumberRecordIfChanged(collection string, id int, data map[string]any) (bool, error) {
	records, recordsErr := s.App.FindRecordsByFilter(collection, "eve_id = {:id}", "", 1, 0, map[string]any{"id": id})
	if recordsErr != nil {
		return false, recordsErr
	}

	coll := s.collection(collection)
	if len(records) == 0 {
		record := core.NewRecord(coll)
		for key, value := range data {
			record.Set(key, value)
		}
		return true, s.App.Save(record)
	}

	record := records[0]
	if !recordHasFieldChanges(record, data) {
		return false, nil
	}
	for key, value := range data {
		record.Set(key, value)
	}
	return true, s.App.Save(record)
}

func recordHasFieldChanges(record *core.Record, data map[string]any) bool {
	for key, incoming := range data {
		if !valuesEqual(record.Get(key), incoming) {
			return true
		}
	}
	return false
}

func valuesEqual(current, incoming any) bool {
	if incoming == nil {
		return current == nil
	}

	if current == nil {
		return false
	}

	if incomingNumber, incomingIsNumber := toFloat64(incoming); incomingIsNumber {
		currentNumber, currentIsNumber := toFloat64(current)
		return currentIsNumber && math.Abs(currentNumber-incomingNumber) < 1e-9
	}

	switch v := incoming.(type) {
	case string:
		currentString, ok := current.(string)
		return ok && currentString == v
	case bool:
		currentBool, ok := current.(bool)
		return ok && currentBool == v
	default:
		return fmt.Sprintf("%v", current) == fmt.Sprintf("%v", incoming)
	}
}

func toFloat64(value any) (float64, bool) {
	switch n := value.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

func (s *SDEImporter) saveMeta(key, value string) error {
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

func (s *SDEImporter) fetchBuildNumber(ctx context.Context) string {
	req, reqErr := http.NewRequestWithContext(ctx, "GET", LatestBuildURL, http.NoBody)
	if reqErr != nil {
		return ""
	}
	resp, respErr := s.Client.Do(req)
	if respErr != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= http.StatusBadRequest {
		return ""
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if build, ok := parseBuildNumberLine(scanner.Text()); ok {
			return build
		}
	}
	return ""
}

func (s *SDEImporter) resetNameCache() {
	if s == nil {
		return
	}
	s.cache.systemNames = map[int]string{}
	s.cache.planetNames = map[int]string{}
}

func (s *SDEImporter) rememberSystemName(systemID int, name string) {
	if s == nil || systemID <= 0 {
		return
	}
	s.cache.systemNames[systemID] = strings.TrimSpace(name)
}

func (s *SDEImporter) rememberPlanetName(planetID int, name string) {
	if s == nil || planetID <= 0 {
		return
	}
	s.cache.planetNames[planetID] = strings.TrimSpace(name)
}

func (s *SDEImporter) lookupSystemName(systemID int) string {
	if s == nil || systemID <= 0 {
		return ""
	}

	if value, ok := s.cache.systemNames[systemID]; ok {
		return value
	}
	name := ""
	if system, err := s.findSystem(systemID); err == nil {
		name = strings.TrimSpace(system.GetString("name"))
	}
	s.cache.systemNames[systemID] = name
	return name
}

func (s *SDEImporter) lookupPlanetName(planetID int) string {
	if s == nil || planetID <= 0 {
		return ""
	}

	if value, ok := s.cache.planetNames[planetID]; ok {
		return value
	}
	name := ""
	if planet, err := s.findPlanet(planetID); err == nil {
		name = strings.TrimSpace(planet.GetString("name"))
	}
	s.cache.planetNames[planetID] = name
	return name
}

func parseBuildNumberLine(line string) (string, bool) {
	if !strings.Contains(line, "\"sde\"") {
		return "", false
	}
	var payload map[string]any
	if unmarshalErr := json.Unmarshal([]byte(line), &payload); unmarshalErr != nil {
		return "", false
	}

	if payload["_key"] != "sde" {
		return "", false
	}
	value, ok := payload["_value"].(map[string]any)
	if !ok {
		return "", false
	}
	build, ok := value["build"]
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%v", build), true
}

func getInt(row map[string]any, keys ...string) int {
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

func getFloat(row map[string]any, keys ...string) float64 {
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

func getString(row map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := row[key]
		if !ok {
			continue
		}
		if text, ok := asString(value); ok {
			return text
		}
		if text, ok := localizedString(value); ok {
			return text
		}
	}
	return ""
}

func asString(value any) (string, bool) {
	text, ok := value.(string)
	return text, ok && text != ""
}

func localizedString(value any) (string, bool) {
	labels, ok := value.(map[string]any)
	if !ok {
		return "", false
	}

	if en, ok := labels["en"].(string); ok && en != "" {
		return en, true
	}
	for _, nested := range labels {
		if text, ok := nested.(string); ok && text != "" {
			return text, true
		}
	}
	return "", false
}

func getPositionXYZ(row map[string]any) (x, y, z float64) {
	if value, ok := row["position"]; ok {
		if m, ok := value.(map[string]any); ok {
			x := getFloat(m, "x")
			y := getFloat(m, "y")
			z := getFloat(m, "z")
			return x, y, z
		}
	}
	return getFloat(row, "x"), getFloat(row, "y"), getFloat(row, "z")
}

func getPosition2D(row map[string]any) (x, y float64, ok bool) {
	if value, ok := row["position2D"]; ok {
		if m, ok := value.(map[string]any); ok {
			return getFloat(m, "x"), getFloat(m, "y"), true
		}
	}

	if value, ok := row["position2d"]; ok {
		if m, ok := value.(map[string]any); ok {
			return getFloat(m, "x"), getFloat(m, "y"), true
		}
	}
	return 0, 0, false
}

func getNestedInt(row map[string]any, first string, keys ...string) int {
	if value, ok := row[first]; ok {
		if m, ok := value.(map[string]any); ok {
			return getInt(m, keys...)
		}
	}
	return 0
}
