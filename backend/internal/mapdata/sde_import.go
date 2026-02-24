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
		logging.New(s.App).
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
		logging.New(s.App).
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

	logging.New(s.App).
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

func (s *SDEImporter) importRegions(ctx context.Context, r io.Reader) error {
	var missingID, missingName, missingPos, saved int
	rowCount, scanErr := forEachJSONLRow(r, func(i int, row map[string]any) error {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		region := parseRegionRow(row)
		switch validateRegionRow(region) {
		case "missing_id":
			logging.New(s.App).
				WithFields(logging.Fields{"row_id": region.rowID}).
				Warn("region missing id")
			missingID++
			return nil
		case "missing_name":
			logging.New(s.App).
				WithFields(logging.Fields{"region_id": region.regionID}).
				Warn("region missing name")
			missingName++
			return nil
		}
		if region.rawX == 0 && region.rawY == 0 {
			missingPos++
		}
		if upsertErr := s.upsertNumberRecord(store.CollectionRegions, region.regionID, region.payload()); upsertErr != nil {
			logging.New(s.App).WithErr(upsertErr).
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
	log := logging.New(s.App).WithFields(logging.Fields{
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
			logging.New(s.App).
				WithFields(logging.Fields{"row_id": getString(row, "id", "_key")}).
				Warn("constellation missing id")
			missingID++
			return nil
		}
		if regionID == 0 {
			logging.New(s.App).
				WithFields(logging.Fields{"constellation_id": id}).
				Warn("constellation missing region")
			missingRegion++
			return nil
		}
		if name == "" {
			logging.New(s.App).
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
			logging.New(s.App).WithErr(upsertErr).
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
	log := logging.New(s.App).WithFields(logging.Fields{
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
			logging.New(s.App).WithFields(logging.Fields{"row_id": system.rowID}).Warn("system missing id")
			missingID++
			return nil
		case "missing_region_or_constellation":
			logging.New(s.App).
				WithFields(logging.Fields{
					"system_id":        system.id,
					"region_id":        system.regionID,
					"constellation_id": system.constID,
				}).
				Warn("system missing region or constellation")
			missingRegionConst++
			return nil
		case "missing_name":
			logging.New(s.App).WithFields(logging.Fields{"system_id": system.id}).Warn("system missing name")
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
			logging.New(s.App).WithErr(upsertErr).
				WithFields(logging.Fields{"system_id": system.id}).
				Error("system upsert failed")
			return upsertErr
		}
		saved++
		return nil
	})
	if scanErr != nil {
		return scanErr
	}
	log := logging.New(s.App).WithFields(logging.Fields{
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
	id       int
	rowID    string
	constID  int
	regionID int
	security float64
	name     string
	x        float64
	y        float64
	z        float64
	pos2dX   float64
	pos2dY   float64
	hasPos2D bool
}

func parseSystemRow(row map[string]any) systemRowData {
	x, y, z := getPositionXYZ(row)
	pos2dX, pos2dY, has2d := getPosition2D(row)
	return systemRowData{
		id:       getInt(row, "solarSystemID", "id", "_key"),
		rowID:    getString(row, "id", "_key"),
		constID:  getInt(row, "constellationID"),
		regionID: getInt(row, "regionID"),
		security: getFloat(row, "security", "securityStatus"),
		name:     getString(row, "solarSystemName", "name"),
		x:        x,
		y:        y,
		z:        z,
		pos2dX:   pos2dX,
		pos2dY:   pos2dY,
		hasPos2D: has2d,
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

const moonGroupID = 8

type planetImportStats struct {
	saved                int
	skippedMissingID     int
	skippedMissingSystem int
	skippedUnchanged     int
}

func (s *SDEImporter) importPlanets(ctx context.Context, r io.Reader) error {
	stats := planetImportStats{}
	rowCount, scanErr := forEachJSONLRow(r, func(i int, row map[string]any) error {
		return s.processPlanetRow(ctx, i, row, &stats)
	})
	if scanErr != nil {
		return scanErr
	}
	log := logging.New(s.App).WithFields(logging.Fields{
		"sde":       "planets",
		"row_count": rowCount,
	})
	log.WithFields(logging.Fields{
		"saved_count":                     stats.saved,
		"skipped_missing_id_count":        stats.skippedMissingID,
		"skipped_missing_system_id_count": stats.skippedMissingSystem,
		"skipped_unchanged_count":         stats.skippedUnchanged,
	}).Info("planets import complete")
	return nil
}

func (s *SDEImporter) processPlanetRow(ctx context.Context, i int, row map[string]any, stats *planetImportStats) error {
	if i%1000 == 0 && ctx.Err() != nil {
		return ctx.Err()
	}
	planet := parsePlanetRow(row)
	if planet.id == 0 {
		stats.skippedMissingID++
		return nil
	}
	if planet.systemID == 0 {
		stats.skippedMissingSystem++
		return nil
	}
	systemName := ""
	if system, err := s.findSystem(planet.systemID); err == nil {
		systemName = system.GetString("name")
	}
	name := planet.name
	if strings.TrimSpace(name) == "" {
		name = derivePlanetName(systemName, planet.celestialIndex, planet.id)
	}
	changed, upsertErr := s.upsertNumberRecordIfChanged(store.CollectionPlanets, planet.id, map[string]any{
		"eve_id":          planet.id,
		"name":            name,
		"system_id":       planet.systemID,
		"system_name":     systemName,
		"celestial_index": planet.celestialIndex,
	})
	if upsertErr != nil {
		logging.New(s.App).WithErr(upsertErr).
			WithFields(logging.Fields{"planet_id": planet.id, "system_id": planet.systemID}).
			Error("planet upsert failed")
		return upsertErr
	}
	if !changed {
		stats.skippedUnchanged++
		return nil
	}
	stats.saved++
	return nil
}

func (s *SDEImporter) importMoons(ctx context.Context, r io.Reader) error {
	var saved, skippedNotMoon, skippedMissingID, skippedMissingSystem, skippedUnchanged int

	rowCount, scanErr := forEachJSONLRow(r, func(i int, row map[string]any) error {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		moon, reason := parseValidMoonRow(row)
		switch reason {
		case "not_moon":
			skippedNotMoon++
			return nil
		case "missing_id":
			skippedMissingID++
			return nil
		case "missing_system":
			skippedMissingSystem++
			return nil
		}
		payload := s.moonPayload(moon)
		changed, upsertErr := s.upsertNumberRecordIfChanged(store.CollectionMoons, moon.id, payload)
		if upsertErr != nil {
			logging.New(s.App).WithErr(upsertErr).
				WithFields(logging.Fields{"moon_id": moon.id, "system_id": moon.systemID}).
				Error("moon upsert failed")
			return upsertErr
		}
		if !changed {
			skippedUnchanged++
			return nil
		}
		saved++
		return nil
	})
	if scanErr != nil {
		return scanErr
	}
	log := logging.New(s.App).WithFields(logging.Fields{
		"sde":       "moons",
		"row_count": rowCount,
	})
	log.WithFields(logging.Fields{
		"saved_count":                     saved,
		"skipped_not_moon_count":          skippedNotMoon,
		"skipped_missing_id_count":        skippedMissingID,
		"skipped_missing_system_id_count": skippedMissingSystem,
		"skipped_unchanged_count":         skippedUnchanged,
	}).Info("moons import complete")
	return nil
}

func parseValidMoonRow(row map[string]any) (moon moonRowData, reason string) {
	moon = parseMoonRow(row)
	if moon.groupID != 0 && moon.groupID != moonGroupID {
		return moon, "not_moon"
	}
	if moon.id == 0 {
		return moon, "missing_id"
	}
	if moon.systemID == 0 {
		return moon, "missing_system"
	}
	return moon, ""
}

func (s *SDEImporter) moonPayload(moon moonRowData) map[string]any {
	systemName := ""
	if system, err := s.findSystem(moon.systemID); err == nil {
		systemName = system.GetString("name")
	}
	planetName := ""
	if moon.planetID > 0 {
		if planet, err := s.findPlanet(moon.planetID); err == nil {
			planetName = planet.GetString("name")
		}
	}
	name := moon.name
	if strings.TrimSpace(name) == "" {
		name = deriveMoonName(systemName, planetName, moon.orbitIndex, moon.id)
	}
	return map[string]any{
		"eve_id":      moon.id,
		"name":        name,
		"system_id":   moon.systemID,
		"system_name": systemName,
		"planet_id":   moon.planetID,
		"planet_name": planetName,
	}
}

type moonRowData struct {
	id         int
	name       string
	groupID    int
	systemID   int
	planetID   int
	orbitIndex int
}

func parseMoonRow(row map[string]any) moonRowData {
	return moonRowData{
		id:         getInt(row, "moonID", "itemID", "id", "_key"),
		name:       getString(row, "moonName", "itemName", "name"),
		groupID:    getInt(row, "groupID"),
		systemID:   getInt(row, "solarSystemID", "locationID"),
		planetID:   getInt(row, "orbitID", "planetID"),
		orbitIndex: getInt(row, "orbitIndex"),
	}
}

type planetRowData struct {
	id             int
	name           string
	systemID       int
	celestialIndex int
}

func parsePlanetRow(row map[string]any) planetRowData {
	return planetRowData{
		id:             getInt(row, "planetID", "itemID", "id", "_key"),
		name:           getString(row, "planetName", "itemName", "name"),
		systemID:       getInt(row, "solarSystemID", "locationID"),
		celestialIndex: getInt(row, "celestialIndex"),
	}
}

func derivePlanetName(systemName string, celestialIndex, planetID int) string {
	if systemName != "" && celestialIndex > 0 {
		return systemName + " " + intToRoman(celestialIndex)
	}
	return "Planet " + strconv.Itoa(planetID)
}

func deriveMoonName(systemName, planetName string, orbitIndex, moonID int) string {
	if planetName != "" {
		base := planetName
		if orbitIndex > 0 {
			return base + " - Moon " + strconv.Itoa(orbitIndex)
		}
		return base + " - Moon"
	}
	if systemName != "" {
		return systemName + " Moon " + strconv.Itoa(max(orbitIndex, 1))
	}
	return "Moon " + strconv.Itoa(moonID)
}

func intToRoman(value int) string {
	if value <= 0 {
		return strconv.Itoa(value)
	}
	var numerals = []struct {
		value int
		text  string
	}{
		{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"},
		{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"},
		{10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
	}
	var out strings.Builder
	for _, numeral := range numerals {
		for value >= numeral.value {
			out.WriteString(numeral.text)
			value -= numeral.value
		}
	}
	return out.String()
}

func (s *SDEImporter) importGates(ctx context.Context, r io.Reader) error {
	var missingID, missingFrom, missingTo, skippedExisting, saved int
	rowCount, scanErr := forEachJSONLRow(r, func(i int, row map[string]any) error {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		result, importErr := s.importGateRow(row)
		if importErr != nil {
			logging.New(s.App).WithErr(importErr).Error("gate save failed")
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
	log := logging.New(s.App).WithFields(logging.Fields{
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
		logging.New(s.App).WithFields(logging.Fields{"from_system": gate.fromID, "to_system": gate.toID}).Warn("gate missing system id")
		return gateImportMissingID, nil
	}
	if s.gateExists(gate.fromID, gate.toID) {
		return gateImportSkippedExisting, nil
	}
	fromSystem, toSystem, systemErr := s.gateSystems(gate)
	if systemErr != nil {
		logging.New(s.App).
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
	case "mapRegions.jsonl", "mapConstellations.jsonl", "mapSolarSystems.jsonl", "mapPlanets.jsonl", "mapMoons.jsonl", "mapStargates.jsonl":
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
