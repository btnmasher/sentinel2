package mapdata

import (
	"context"
	"math"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
	"sentinel2/internal/shared/geom"
	"sentinel2/internal/store"
)

const (
	regionScaleTarget     = 1000.0
	centerMidpointDivisor = 2.0
)

func CalculateRealPositions(ctx context.Context, app *pocketbase.PocketBase) error {
	log := logging.New(app).WithFields(logging.Fields{"component": "mapdata.real_positions"})
	regions, regionsErr := app.FindRecordsByFilter(store.CollectionRegions, "eve_id < 11000000", "name", 0, 0, nil)
	if regionsErr != nil {
		return regionsErr
	}

	systems, systemsErr := app.FindRecordsByFilter(store.CollectionSolarSystems, "region_id > 0", "", 0, 0, nil)
	if systemsErr != nil {
		return systemsErr
	}

	start := time.Now()
	log.
		WithFields(logging.Fields{
			"region_count": len(regions),
			"system_count": len(systems),
		}).
		Debug("real positions calc started")

	regionSet := regionIDSet(regions)
	regionBounds, boundsErr := collectRealBounds(ctx, systems, regionSet)
	if boundsErr != nil {
		return boundsErr
	}

	regionScales := buildRegionScales(regionBounds)

	if saveErr := saveSystemRealPositions(ctx, app, systems, regionScales, log); saveErr != nil {
		return saveErr
	}

	centers, minX, minY, maxX, maxY := centersFromRealBounds(regionBounds)
	scale := normalizeScale(minX, minY, maxX, maxY, normalizedRegionTarget)
	if saveErr := saveRegionRealPositions(ctx, realRegionSaveInput{
		app:     app,
		regions: regions,
		centers: centers,
		minX:    minX,
		minY:    minY,
		scale:   scale,
		log:     log,
	}); saveErr != nil {
		return saveErr
	}

	log.
		WithFields(logging.Fields{
			"duration_ms":       time.Since(start).Milliseconds(),
			"regions_with_real": len(centers),
		}).
		Debug("real positions calc completed")

	return nil
}

type regionScale struct {
	minX  float64
	minY  float64
	scale float64
}

func collectRealBounds(ctx context.Context, systems []*core.Record, regionSet map[int]struct{}) (map[int]*geom.Bounds[float64], error) {
	regionBounds := map[int]*geom.Bounds[float64]{}
	for _, system := range systems {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		regionID := system.GetInt("region_id")
		if _, ok := regionSet[regionID]; !ok {
			continue
		}
		x := system.GetFloat("raw_x")
		y := -system.GetFloat("raw_z")
		if x == 0 && y == 0 {
			continue
		}
		b := regionBounds[regionID]
		if b == nil {
			b = &geom.Bounds[float64]{}
			regionBounds[regionID] = b
		}
		b.Add(x, y)
	}
	return regionBounds, nil
}

func buildRegionScales(regionBounds map[int]*geom.Bounds[float64]) map[int]regionScale {
	regionScales := map[int]regionScale{}
	for regionID, b := range regionBounds {
		dx := b.MaxX - b.MinX
		dy := b.MaxY - b.MinY
		if dx == 0 || dy == 0 {
			continue
		}
		scale := regionScaleTarget / dx
		yScale := regionScaleTarget / dy
		scale = min(scale, yScale)
		regionScales[regionID] = regionScale{minX: b.MinX, minY: b.MinY, scale: scale}
	}
	return regionScales
}

func saveSystemRealPositions(ctx context.Context, app *pocketbase.PocketBase, systems []*core.Record, regionScales map[int]regionScale, log *logging.Logger) error {
	for i, system := range systems {
		if i%1000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		regionID := system.GetInt("region_id")
		scale, ok := regionScales[regionID]
		if !ok {
			continue
		}
		x := system.GetFloat("raw_x")
		y := -system.GetFloat("raw_z")
		system.Set("real_x", int(math.Round((x-scale.minX)*scale.scale)))
		system.Set("real_y", int(math.Round((y-scale.minY)*scale.scale)))
		if saveErr := app.Save(system); saveErr != nil {
			log.WithFields(logging.Fields{"system_id": system.GetInt("eve_id")}).
				WithErr(saveErr).
				Debug("real system position save failed")
		}
	}
	return nil
}

func centersFromRealBounds(regionBounds map[int]*geom.Bounds[float64]) (centers map[int][2]float64, minX, minY, maxX, maxY float64) {
	centers = map[int][2]float64{}
	first := true
	for regionID, b := range regionBounds {
		if b == nil || b.Count == 0 {
			continue
		}
		x := b.MinX + (b.MaxX-b.MinX)/centerMidpointDivisor
		y := b.MinY + (b.MaxY-b.MinY)/centerMidpointDivisor
		centers[regionID] = [2]float64{x, y}
		minX, minY, maxX, maxY = extendFloatBounds(x, y, minX, minY, maxX, maxY, first)
		first = false
	}
	return centers, minX, minY, maxX, maxY
}

type realRegionSaveInput struct {
	app     *pocketbase.PocketBase
	regions []*core.Record
	centers map[int][2]float64
	minX    float64
	minY    float64
	scale   float64
	log     *logging.Logger
}

func saveRegionRealPositions(ctx context.Context, input realRegionSaveInput) error {
	for _, region := range input.regions {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		regionID := region.GetInt("eve_id")
		if center, ok := input.centers[regionID]; ok {
			region.Set("real_x", int(math.Round((center[0]-input.minX)*input.scale)))
			region.Set("real_y", int(math.Round((center[1]-input.minY)*input.scale)))
		}
		if saveErr := input.app.Save(region); saveErr != nil {
			input.log.WithFields(logging.Fields{
				"region_id":   regionID,
				"region_name": region.GetString("name"),
			}).
				WithErr(saveErr).
				Debug("real region position save failed")
		}
	}
	return nil
}
