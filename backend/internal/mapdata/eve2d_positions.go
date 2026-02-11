package mapdata

import (
	"math"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

type regionBounds struct {
	minX  float64
	minY  float64
	maxX  float64
	maxY  float64
	count int
}

func UpdateRegionPositionsFromSystems(app core.App) error {
	regions, regionsErr := app.FindRecordsByFilter(store.CollectionRegions, "eve_id < 11000000", "name", 0, 0, nil)
	if regionsErr != nil {
		return regionsErr
	}

	systems, systemsErr := app.FindRecordsByFilter(store.CollectionSolarSystems, "region_id > 0", "", 0, 0, nil)
	if systemsErr != nil {
		return systemsErr
	}

	logger := logging.New(nil)
	if pb, ok := app.(*pocketbase.PocketBase); ok {
		logger = logging.New(pb)
	}

	start := time.Now()
	logger.
		WithFields(logging.Fields{
			"region_count": len(regions),
			"system_count": len(systems),
		}).
		Debug("region positions from systems started")

	regionSet := map[int]struct{}{}
	for _, region := range regions {
		regionSet[int(region.GetInt("eve_id"))] = struct{}{}
	}

	eve2dBounds := map[int]*regionBounds{}
	var systemsWithPos2D int

	for _, system := range systems {
		regionID := int(system.GetInt("region_id"))
		if _, ok := regionSet[regionID]; !ok {
			continue
		}

		eve2dX := system.GetFloat("eve2d_x")
		eve2dY := -system.GetFloat("eve2d_y")
		if eve2dX != 0 || eve2dY != 0 {
			updateRegionBounds(eve2dBounds, regionID, eve2dX, eve2dY)
			systemsWithPos2D++
		}
	}

	centers := map[int][2]float64{}
	minX, minY := 0.0, 0.0
	maxX, maxY := 0.0, 0.0
	first := true
	for regionID, b := range eve2dBounds {
		if b.count == 0 {
			continue
		}
		x := b.minX + (b.maxX-b.minX)/2
		y := b.minY + (b.maxY-b.minY)/2
		centers[regionID] = [2]float64{x, y}
		if first {
			minX, maxX = x, x
			minY, maxY = y, y
			first = false
			continue
		}
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}

	scale := 1.0
	dx := maxX - minX
	dy := maxY - minY
	const target = 10000.0
	const spacingFactor = 1.25
	if dx > 0 || dy > 0 {
		if dx > dy {
			scale = target / dx
		} else {
			scale = target / dy
		}
	}

	for _, region := range regions {
		regionID := int(region.GetInt("eve_id"))
		if center, ok := centers[regionID]; ok {
			region.Set("eve2d_x", int(math.Round((center[0]-minX)*scale*spacingFactor)))
			region.Set("eve2d_y", int(math.Round((center[1]-minY)*scale*spacingFactor)))
		}
		if saveErr := app.Save(region); saveErr != nil {
			logger.
				WithFields(logging.Fields{
					"region_id":   regionID,
					"region_name": region.GetString("name"),
				}).
				WithErr(saveErr).
				Debug("region position save failed")
		}
	}

	logger.
		WithFields(logging.Fields{
			"duration_ms":        time.Since(start).Milliseconds(),
			"eve2d_systems":      systemsWithPos2D,
			"regions_with_eve2d": len(centers),
		}).
		Debug("region positions from systems completed")

	return nil
}

func updateRegionBounds(bounds map[int]*regionBounds, regionID int, x float64, y float64) {
	b, exists := bounds[regionID]
	if !exists {
		bounds[regionID] = &regionBounds{
			minX:  x,
			minY:  y,
			maxX:  x,
			maxY:  y,
			count: 1,
		}
		return
	}
	if x < b.minX {
		b.minX = x
	}
	if y < b.minY {
		b.minY = y
	}
	if x > b.maxX {
		b.maxX = x
	}
	if y > b.maxY {
		b.maxY = y
	}
	b.count++
}
