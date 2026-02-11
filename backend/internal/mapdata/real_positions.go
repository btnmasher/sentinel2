package mapdata

import (
	"context"
	"math"
	"time"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

type regionFloatBounds struct {
	minX  float64
	minY  float64
	maxX  float64
	maxY  float64
	count int
}

func CalculateRealPositions(ctx context.Context, app *pocketbase.PocketBase) error {
	regions, regionsErr := app.FindRecordsByFilter(store.CollectionRegions, "eve_id < 11000000", "name", 0, 0, nil)
	if regionsErr != nil {
		return regionsErr
	}

	systems, systemsErr := app.FindRecordsByFilter(store.CollectionSolarSystems, "region_id > 0", "", 0, 0, nil)
	if systemsErr != nil {
		return systemsErr
	}

	start := time.Now()
	logging.New(app).
		WithFields(logging.Fields{
			"region_count": len(regions),
			"system_count": len(systems),
		}).
		Debug("real positions calc started")

	regionSet := map[int]struct{}{}
	for _, region := range regions {
		regionSet[int(region.GetInt("eve_id"))] = struct{}{}
	}

	regionBounds := map[int]*regionFloatBounds{}
	for _, system := range systems {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		regionID := int(system.GetInt("region_id"))
		if _, ok := regionSet[regionID]; !ok {
			continue
		}
		x := system.GetFloat("raw_x")
		y := -system.GetFloat("raw_z")
		if x == 0 && y == 0 {
			continue
		}
		b, exists := regionBounds[regionID]
		if !exists {
			regionBounds[regionID] = &regionFloatBounds{
				minX:  x,
				minY:  y,
				maxX:  x,
				maxY:  y,
				count: 1,
			}
			continue
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

	type regionScale struct {
		minX  float64
		minY  float64
		scale float64
	}
	regionScales := map[int]regionScale{}
	for regionID, b := range regionBounds {
		dx := b.maxX - b.minX
		dy := b.maxY - b.minY
		if dx == 0 || dy == 0 {
			continue
		}
		scale := 1000.0 / dx
		if yScale := 1000.0 / dy; yScale < scale {
			scale = yScale
		}
		regionScales[regionID] = regionScale{minX: b.minX, minY: b.minY, scale: scale}
	}

	for i, system := range systems {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		regionID := int(system.GetInt("region_id"))
		scale, ok := regionScales[regionID]
		if !ok {
			continue
		}
		x := system.GetFloat("raw_x")
		y := -system.GetFloat("raw_z")
		system.Set("real_x", int(math.Round((x-scale.minX)*scale.scale)))
		system.Set("real_y", int(math.Round((y-scale.minY)*scale.scale)))
		if saveErr := app.Save(system); saveErr != nil {
			logging.New(app).
				WithFields(logging.Fields{
					"system_id": system.GetInt("eve_id"),
				}).
				WithErr(saveErr).
				Debug("real system position save failed")
		}
	}

	centers := map[int][2]float64{}
	minX, minY := 0.0, 0.0
	maxX, maxY := 0.0, 0.0
	first := true
	for regionID, b := range regionBounds {
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
	if dx > 0 || dy > 0 {
		if dx > dy {
			scale = target / dx
		} else {
			scale = target / dy
		}
	}

	for _, region := range regions {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		regionID := int(region.GetInt("eve_id"))
		if center, ok := centers[regionID]; ok {
			region.Set("real_x", int(math.Round((center[0]-minX)*scale)))
			region.Set("real_y", int(math.Round((center[1]-minY)*scale)))
		}
		if saveErr := app.Save(region); saveErr != nil {
			logging.New(app).
				WithFields(logging.Fields{
					"region_id":   regionID,
					"region_name": region.GetString("name"),
				}).
				WithErr(saveErr).
				Debug("real region position save failed")
		}
	}

	logging.New(app).
		WithFields(logging.Fields{
			"duration_ms":       time.Since(start).Milliseconds(),
			"regions_with_real": len(centers),
		}).
		Debug("real positions calc completed")

	return nil
}
