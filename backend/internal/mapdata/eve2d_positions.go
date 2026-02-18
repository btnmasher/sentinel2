package mapdata

import (
	"math"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
	"sentinel2/internal/shared/geom"
	"sentinel2/internal/store"
)

const boundsMidpointDivisor = 2.0
const (
	normalizedRegionTarget = 10000.0
	eve2dSpacingFactor     = 1.25
)

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

	regionSet := regionIDSet(regions)
	eve2dBounds, systemsWithPos2D := collectEve2DBounds(systems, regionSet)
	centers, minX, minY, maxX, maxY := centerPointsFromBounds(eve2dBounds)
	scale := normalizeScale(minX, minY, maxX, maxY, normalizedRegionTarget)
	saveEve2DRegionCenters(&eve2DRegionSaveInput{
		app:           app,
		regions:       regions,
		centers:       centers,
		minX:          minX,
		minY:          minY,
		scale:         scale,
		spacingFactor: eve2dSpacingFactor,
		logger:        logger,
	})

	logger.
		WithFields(logging.Fields{
			"duration_ms":        time.Since(start).Milliseconds(),
			"eve2d_systems":      systemsWithPos2D,
			"regions_with_eve2d": len(centers),
		}).
		Debug("region positions from systems completed")

	return nil
}

func collectEve2DBounds(systems []*core.Record, regionSet map[int]struct{}) (eve2dBounds map[int]*geom.Bounds[float64], systemsWithPos2D int) {
	eve2dBounds = map[int]*geom.Bounds[float64]{}
	for _, system := range systems {
		regionID := system.GetInt("region_id")
		if _, ok := regionSet[regionID]; !ok {
			continue
		}
		eve2dX := system.GetFloat("eve2d_x")
		eve2dY := -system.GetFloat("eve2d_y")
		if eve2dX == 0 && eve2dY == 0 {
			continue
		}
		b := eve2dBounds[regionID]
		if b == nil {
			b = &geom.Bounds[float64]{}
			eve2dBounds[regionID] = b
		}
		b.Add(eve2dX, eve2dY)
		systemsWithPos2D++
	}
	return eve2dBounds, systemsWithPos2D
}

func centerPointsFromBounds(bounds map[int]*geom.Bounds[float64]) (centers map[int][2]float64, minX, minY, maxX, maxY float64) {
	centers = map[int][2]float64{}
	first := true
	for regionID, b := range bounds {
		if b == nil || b.Count == 0 {
			continue
		}
		x := b.MinX + (b.MaxX-b.MinX)/boundsMidpointDivisor
		y := b.MinY + (b.MaxY-b.MinY)/boundsMidpointDivisor
		centers[regionID] = [2]float64{x, y}
		minX, minY, maxX, maxY = extendFloatBounds(x, y, minX, minY, maxX, maxY, first)
		first = false
	}
	return centers, minX, minY, maxX, maxY
}

type eve2DRegionSaveInput struct {
	app           core.App
	regions       []*core.Record
	centers       map[int][2]float64
	minX          float64
	minY          float64
	scale         float64
	spacingFactor float64
	logger        *logging.Logger
}

func saveEve2DRegionCenters(input *eve2DRegionSaveInput) {
	for _, region := range input.regions {
		regionID := region.GetInt("eve_id")
		if center, ok := input.centers[regionID]; ok {
			region.Set("eve2d_x", int(math.Round((center[0]-input.minX)*input.scale*input.spacingFactor)))
			region.Set("eve2d_y", int(math.Round((center[1]-input.minY)*input.scale*input.spacingFactor)))
		}
		if saveErr := input.app.Save(region); saveErr != nil {
			input.logger.
				WithFields(logging.Fields{
					"region_id":   regionID,
					"region_name": region.GetString("name"),
				}).
				WithErr(saveErr).
				Debug("region position save failed")
		}
	}
}
