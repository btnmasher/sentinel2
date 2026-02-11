package mapdata

import (
	"context"
	"encoding/xml"
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

var sysRegex = regexp.MustCompile(`sys(\d{8})`)

func DownloadDotlan(ctx context.Context, app *pocketbase.PocketBase) error {
	regions, regionsErr := app.FindRecordsByFilter(store.CollectionRegions, "eve_id < 11000000", "name", 0, 0, nil)
	if regionsErr != nil {
		logging.New(app).
			WithErr(regionsErr).
			Warn("dotlan region lookup failed")
		return regionsErr
	}

	start := time.Now()
	logging.New(app).
		WithFields(logging.Fields{
			"region_count": len(regions),
		}).
		Debug("dotlan sync started")

	for _, region := range regions {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if updateErr := updateRegionDotlan(ctx, app, region); updateErr != nil {
			logging.New(app).
				WithFields(logging.Fields{
					"region_id":   region.GetInt("eve_id"),
					"region_name": region.GetString("name"),
				}).
				WithErr(updateErr).
				Warn("dotlan region sync failed")
			return updateErr
		}
	}

	logging.New(app).
		WithFields(logging.Fields{
			"duration_ms": time.Since(start).Milliseconds(),
		}).
		Debug("dotlan sync completed")
	return nil
}

func updateRegionDotlan(ctx context.Context, app *pocketbase.PocketBase, region *core.Record) error {
	name := strings.ReplaceAll(region.GetString("name"), " ", "_")
	req, reqErr := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(DotlanURL, name), nil)
	if reqErr != nil {
		return reqErr
	}
	resp, respErr := http.DefaultClient.Do(req)
	if respErr != nil {
		return respErr
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to fetch dotlan for %s (status %d)", name, resp.StatusCode)
	}

	decoder := xml.NewDecoder(resp.Body)
	inSysUse := false
	sysUseDepth := 0
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			id := attrValue(elem.Attr, "id")
			if id == "sysuse" {
				inSysUse = true
				sysUseDepth = 1
				continue
			}
			if !inSysUse {
				continue
			}
			sysUseDepth++
			if !strings.HasPrefix(id, "sys") {
				continue
			}

			match := sysRegex.FindStringSubmatch(id)
			if len(match) == 0 {
				continue
			}
			sysID, _ := strconv.Atoi(match[1])
			x, _ := strconv.Atoi(attrValue(elem.Attr, "x"))
			y, _ := strconv.Atoi(attrValue(elem.Attr, "y"))

			system, systemErr := findSystemByID(app, sysID)
			if systemErr != nil {
				continue
			}

			if skipDotlanRegion(system.GetInt("region_id")) {
				continue
			}

			if int(system.GetInt("region_id")) == int(region.GetInt("eve_id")) {
				system.Set("dotlan_x", x)
				system.Set("dotlan_y", y)
				if saveErr := app.Save(system); saveErr != nil {
					logging.New(app).
						WithFields(logging.Fields{
							"region_id":   region.GetInt("eve_id"),
							"system_id":   system.GetInt("eve_id"),
							"system_name": system.GetString("name"),
						}).
						WithErr(saveErr).
						Debug("dotlan system sync failed")
				}
				continue
			}

			updateRegionGateCoords(app, system, region, x, y, "dotlan")
		case xml.EndElement:
			if !inSysUse {
				continue
			}
			sysUseDepth--
			if sysUseDepth <= 0 {
				inSysUse = false
				sysUseDepth = 0
			}
		}
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
	switch regionID {
	case 10000019, 10000017, 10000004:
		return true
	default:
		return false
	}
}

func updateRegionGateCoords(app *pocketbase.PocketBase, system *core.Record, region *core.Record, x int, y int, mode string) {
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
		fromRegion := gate.GetInt("from_region")
		toRegion := gate.GetInt("to_region")
		if fromRegion == toRegion {
			continue
		}

		if gate.GetInt("to_solarsystem") == system.GetInt("eve_id") && fromRegion == region.GetInt("eve_id") {
			updateGateCoords(gate, "to", mode, x, y)
			if saveErr := app.Save(gate); saveErr != nil {
				logging.New(app).
					WithFields(logging.Fields{
						"gate_id":    gate.Id,
						"system_id":  system.GetInt("eve_id"),
						"region_id":  region.GetInt("eve_id"),
						"coord_mode": mode,
					}).
					WithErr(saveErr).
					Debug("dotlan gate sync failed")
			}
		}

		if gate.GetInt("from_solarsystem") == system.GetInt("eve_id") && toRegion == region.GetInt("eve_id") {
			updateGateCoords(gate, "from", mode, x, y)
			if saveErr := app.Save(gate); saveErr != nil {
				logging.New(app).
					WithFields(logging.Fields{
						"gate_id":    gate.Id,
						"system_id":  system.GetInt("eve_id"),
						"region_id":  region.GetInt("eve_id"),
						"coord_mode": mode,
					}).
					WithErr(saveErr).
					Debug("dotlan gate sync failed")
			}
		}
	}
}

func updateGateCoords(gate *core.Record, direction string, mode string, x int, y int) {
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
