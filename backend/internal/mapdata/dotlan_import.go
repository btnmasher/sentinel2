package mapdata

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

const DotlanURL = "https://evemaps.dotlan.net/svg/%s.svg"
const (
	dotlanHTTPErrorStatus = http.StatusBadRequest
	regionIDA821A         = 10000019
	regionIDJ7HZF         = 10000017
	regionIDUUAF4         = 10000004
)

var sysRegex = regexp.MustCompile(`sys(\d{8})`)

func DownloadDotlan(ctx context.Context, app *pocketbase.PocketBase) error {
	log := logging.New(app).WithFields(logging.Fields{"component": "mapdata.dotlan"})
	regions, regionsErr := app.FindRecordsByFilter(store.CollectionRegions, "eve_id < 11000000", "name", 0, 0, nil)
	if regionsErr != nil {
		log.WithErr(regionsErr).Warn("dotlan region lookup failed")
		return regionsErr
	}

	start := time.Now()
	log.
		WithFields(logging.Fields{
			"region_count": len(regions),
		}).
		Debug("dotlan sync started")

	for _, region := range regions {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if updateErr := updateRegionDotlan(ctx, app, region, log); updateErr != nil {
			log.
				WithFields(logging.Fields{
					"region_id":   region.GetInt("eve_id"),
					"region_name": region.GetString("name"),
				}).
				WithErr(updateErr).
				Warn("dotlan region sync failed")
			return updateErr
		}
	}

	log.
		WithFields(logging.Fields{
			"duration_ms": time.Since(start).Milliseconds(),
		}).
		Debug("dotlan sync completed")
	return nil
}

func updateRegionDotlan(ctx context.Context, app *pocketbase.PocketBase, region *core.Record, log *logging.Logger) error {
	name := strings.ReplaceAll(region.GetString("name"), " ", "_")
	req, reqErr := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(DotlanURL, name), http.NoBody)
	if reqErr != nil {
		return reqErr
	}
	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		return respErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= dotlanHTTPErrorStatus {
		return fmt.Errorf("failed to fetch dotlan for %s (status %d)", name, resp.StatusCode)
	}

	decoder := xml.NewDecoder(resp.Body)
	state := dotlanParseState{}
	for {
		token, tokenErr := decoder.Token()
		if tokenErr != nil {
			if errors.Is(tokenErr, io.EOF) {
				break
			}
			return tokenErr
		}
		if handleErr := handleDotlanToken(app, region, &state, token, log); handleErr != nil {
			return handleErr
		}
	}

	return nil
}

type dotlanParseState struct {
	inSysUse    bool
	sysUseDepth int
}

func handleDotlanToken(app *pocketbase.PocketBase, region *core.Record, state *dotlanParseState, token xml.Token, log *logging.Logger) error {
	switch elem := token.(type) {
	case xml.StartElement:
		return handleDotlanStartElement(app, region, state, elem, log)
	case xml.EndElement:
		handleDotlanEndElement(state)
	}
	return nil
}

func handleDotlanStartElement(app *pocketbase.PocketBase, region *core.Record, state *dotlanParseState, elem xml.StartElement, log *logging.Logger) error {
	id := attrValue(elem.Attr, "id")
	if id == "sysuse" {
		state.inSysUse = true
		state.sysUseDepth = 1
		return nil
	}
	if !state.inSysUse {
		return nil
	}
	state.sysUseDepth++
	if !strings.HasPrefix(id, "sys") {
		return nil
	}
	match := sysRegex.FindStringSubmatch(id)
	if len(match) == 0 {
		return nil
	}
	sysID, _ := strconv.Atoi(match[1])
	x, _ := strconv.Atoi(attrValue(elem.Attr, "x"))
	y, _ := strconv.Atoi(attrValue(elem.Attr, "y"))
	return updateDotlanSystemCoords(app, region, sysID, x, y, log)
}

func handleDotlanEndElement(state *dotlanParseState) {
	if !state.inSysUse {
		return
	}
	state.sysUseDepth--
	if state.sysUseDepth <= 0 {
		state.inSysUse = false
		state.sysUseDepth = 0
	}
}

func updateDotlanSystemCoords(app *pocketbase.PocketBase, region *core.Record, sysID, x, y int, log *logging.Logger) error {
	system, systemErr := findSystemByID(app, sysID)
	if systemErr != nil || skipDotlanRegion(system.GetInt("region_id")) {
		return nil
	}
	if system.GetInt("region_id") == region.GetInt("eve_id") {
		return saveDotlanSystemCoords(app, system, region, x, y, log)
	}
	updateRegionGateCoords(app, system, region, x, y, "dotlan")
	return nil
}

func saveDotlanSystemCoords(app *pocketbase.PocketBase, system, region *core.Record, x, y int, log *logging.Logger) error {
	system.Set("dotlan_x", x)
	system.Set("dotlan_y", y)
	if saveErr := app.Save(system); saveErr != nil {
		log.WithFields(logging.Fields{
			"region_id":   region.GetInt("eve_id"),
			"system_id":   system.GetInt("eve_id"),
			"system_name": system.GetString("name"),
		}).
			WithErr(saveErr).
			Debug("dotlan system sync failed")
	}
	return nil
}

func attrValue(attrs []xml.Attr, key string) string {
	for _, attr := range attrs {
		if attr.Name.Local == key {
			return attr.Value
		}
	}
	return ""
}

func skipDotlanRegion(regionID int) bool {
	// These EVE regions don't have usable Dotlan overlays for our import flow.
	switch regionID {
	case regionIDA821A, regionIDJ7HZF, regionIDUUAF4:
		return true
	default:
		return false
	}
}

func updateRegionGateCoords(app *pocketbase.PocketBase, system, region *core.Record, x, y int, mode string) {
	log := logging.New(app).WithFields(logging.Fields{"component": "mapdata.dotlan"})
	records, recordsErr := app.FindRecordsByFilter(
		store.CollectionGates,
		"from_solarsystem = {:id} || to_solarsystem = {:id}",
		"",
		0,
		0,
		map[string]any{"id": system.GetInt("eve_id")},
	)
	if recordsErr != nil {
		return
	}

	for _, gate := range records {
		if !shouldUpdateGateCoords(gate) {
			continue
		}
		updateDirection := gateCoordDirection(gate, system, region)
		if updateDirection == "" {
			continue
		}
		updateGateCoords(gate, updateDirection, mode, x, y)
		saveDotlanGate(app, gate, system, region, mode, log)
	}
}

func shouldUpdateGateCoords(gate *core.Record) bool {
	return gate.GetInt("from_region") != gate.GetInt("to_region")
}

func gateCoordDirection(gate, system, region *core.Record) string {
	if gate.GetInt("to_solarsystem") == system.GetInt("eve_id") && gate.GetInt("from_region") == region.GetInt("eve_id") {
		return "to"
	}
	if gate.GetInt("from_solarsystem") == system.GetInt("eve_id") && gate.GetInt("to_region") == region.GetInt("eve_id") {
		return "from"
	}
	return ""
}

func saveDotlanGate(app *pocketbase.PocketBase, gate, system, region *core.Record, mode string, log *logging.Logger) {
	if saveErr := app.Save(gate); saveErr != nil {
		log.WithFields(logging.Fields{
			"gate_id":    gate.Id,
			"system_id":  system.GetInt("eve_id"),
			"region_id":  region.GetInt("eve_id"),
			"coord_mode": mode,
		}).
			WithErr(saveErr).
			Debug("dotlan gate sync failed")
	}
}

func updateGateCoords(gate *core.Record, direction, mode string, x, y int) {
	prefix := direction + "_" + mode
	gate.Set(prefix+"_x", x)
	gate.Set(prefix+"_y", y)
}

func findSystemByID(app *pocketbase.PocketBase, id int) (*core.Record, error) {
	records, recordsErr := app.FindRecordsByFilter(store.CollectionSolarSystems, "eve_id = {:id}", "", 1, 0, map[string]any{"id": id})
	if recordsErr != nil {
		return nil, recordsErr
	}
	if len(records) == 0 {
		return nil, ErrSystemNotFound
	}
	return records[0], nil
}
