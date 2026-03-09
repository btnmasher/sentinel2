package mapdata

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
	"sentinel2/internal/shared/geom"
	"sentinel2/internal/store"
)

const (
	minOverlapRegionCount = 2
	halfDivisor           = 2.0
)

func CalculateRegionLayouts(ctx context.Context, app *pocketbase.PocketBase) error {
	log := logging.New(app).WithFields(logging.Fields{"component": "mapdata.metro_region_positions"})
	regions, gates, fetchErr := loadRegionLayoutInputs(app)
	if fetchErr != nil {
		return fetchErr
	}

	start := time.Now()
	log.
		WithFields(logging.Fields{
			"region_count": len(regions),
		}).
		Debug("region layout calc started")

	if ctx.Err() != nil {
		return ctx.Err()
	}
	positions, layoutErr := regionLayoutPositions(ctx, regions, gates, app)
	if layoutErr != nil {
		return layoutErr
	}

	regionSizes, sizeErr := buildMetroRegionSizes(ctx, app, regions)
	if sizeErr != nil {
		log.WithErr(sizeErr).Warn("region layout size lookup failed")
		return sizeErr
	}

	posScaled, sizeScaled := scaleRegionLayout(positions, regionSizes)
	resolveRegionOverlaps(posScaled, sizeScaled)
	minX, minY := minScaledRegionPosition(posScaled)
	if saveErr := saveMetroRegionLayouts(ctx, app, regions, posScaled, minX, minY, log); saveErr != nil {
		return saveErr
	}

	log.
		WithFields(logging.Fields{
			"duration_ms": time.Since(start).Milliseconds(),
		}).
		Debug("region layout calc completed")
	return nil
}

func loadRegionLayoutInputs(app *pocketbase.PocketBase) (regions, gates []*core.Record, err error) {
	log := logging.New(app).WithFields(logging.Fields{"component": "mapdata.metro_region_positions"})
	regions, regionsErr := app.FindRecordsByFilter(store.CollectionRegions, "eve_id < 11000000", "name", 0, 0, nil)
	if regionsErr != nil {
		log.WithErr(regionsErr).Warn("region layout lookup failed")
		return nil, nil, regionsErr
	}
	gates, gatesErr := app.FindRecordsByFilter(store.CollectionGates, "", "", 0, 0, nil)
	if gatesErr != nil {
		log.WithErr(gatesErr).Warn("region layout gate lookup failed")
		return nil, nil, gatesErr
	}
	return regions, gates, nil
}

func regionLayoutPositions(ctx context.Context, regions, gates []*core.Record, app *pocketbase.PocketBase) (map[int]regionPos, error) {
	log := logging.New(app).WithFields(logging.Fields{"component": "mapdata.metro_region_positions"})
	positions, useEve2D := eve2DRegionPositions(regions)
	if useEve2D {
		return positions, nil
	}
	layoutPositions, layoutErr := runRegionLayout(ctx, regions, gates)
	if layoutErr != nil {
		log.WithErr(layoutErr).Warn("region layout calc failed")
		return nil, layoutErr
	}
	return layoutPositions, nil
}

func eve2DRegionPositions(regions []*core.Record) (map[int]regionPos, bool) {
	positions := map[int]regionPos{}
	useEve2D := false
	for _, region := range regions {
		id := region.GetInt("eve_id")
		x := region.GetInt("eve2d_x")
		y := region.GetInt("eve2d_y")
		if x == 0 && y == 0 {
			continue
		}
		positions[id] = regionPos{x: x, y: y}
		useEve2D = true
	}
	return positions, useEve2D
}

func scaleRegionLayout(positions map[int]regionPos, regionSizes map[int]regionSize) (posScaled map[int]regionPosF, sizeScaled map[int]regionSize) {
	minX, minY, maxX, maxY := regionPositionBounds(positions)
	scale := normalizeScale(minX, minY, maxX, maxY, normalizedRegionTarget)
	posScaled = map[int]regionPosF{}
	sizeScaled = map[int]regionSize{}
	for id, pos := range positions {
		posScaled[id] = regionPosF{
			x: (float64(pos.x) - minX) * scale,
			y: (float64(pos.y) - minY) * scale,
		}
		if size, ok := regionSizes[id]; ok {
			sizeScaled[id] = regionSize{w: size.w * scale, h: size.h * scale}
		}
	}
	return posScaled, sizeScaled
}

func regionPositionBounds(positions map[int]regionPos) (minX, minY, maxX, maxY float64) {
	first := true
	for _, pos := range positions {
		x := float64(pos.x)
		y := float64(pos.y)
		minX, minY, maxX, maxY = extendFloatBounds(x, y, minX, minY, maxX, maxY, first)
		first = false
	}
	return minX, minY, maxX, maxY
}

func minScaledRegionPosition(positions map[int]regionPosF) (minX, minY float64) {
	minX, minY = 0.0, 0.0
	first := true
	for _, pos := range positions {
		if first {
			minX, minY, first = pos.x, pos.y, false
			continue
		}
		if pos.x < minX {
			minX = pos.x
		}
		if pos.y < minY {
			minY = pos.y
		}
	}
	return minX, minY
}

func saveMetroRegionLayouts(ctx context.Context, app *pocketbase.PocketBase, regions []*core.Record, posScaled map[int]regionPosF, minX, minY float64, log *logging.Logger) error {
	for i, region := range regions {
		if i%100 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		id := region.GetInt("eve_id")
		pos, ok := posScaled[id]
		if !ok {
			continue
		}
		region.Set("metro_x", int(math.Round(pos.x-minX)))
		region.Set("metro_y", int(math.Round(pos.y-minY)))
		if saveErr := app.Save(region); saveErr != nil {
			log.WithFields(logging.Fields{"region_id": id, "region_name": region.GetString("name")}).
				WithErr(saveErr).
				Debug("region layout save failed")
		}
	}
	return nil
}

type regionPos struct {
	x int
	y int
}

type regionPosF struct {
	x float64
	y float64
}

type regionSize struct {
	w float64
	h float64
}

func buildMetroRegionSizes(ctx context.Context, app *pocketbase.PocketBase, regions []*core.Record) (map[int]regionSize, error) {
	regionSet := regionIDSet(regions)
	records, err := app.FindRecordsByFilter(
		store.CollectionSolarSystems,
		"region_id < 11000000",
		"",
		0,
		0,
		nil,
	)
	if err != nil {
		return nil, err
	}
	boundsByRegion, boundsErr := collectMetroBounds(ctx, records, regionSet)
	if boundsErr != nil {
		return nil, boundsErr
	}
	return metroRegionSizes(regions, boundsByRegion), nil
}

func collectMetroBounds(ctx context.Context, records []*core.Record, regionSet map[int]struct{}) (map[int]*geom.Bounds[int], error) {
	boundsByRegion := map[int]*geom.Bounds[int]{}
	for i, record := range records {
		if i%1000 == 0 && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		regionID := record.GetInt("region_id")
		if _, ok := regionSet[regionID]; !ok {
			continue
		}
		b := boundsByRegion[regionID]
		if b == nil {
			b = &geom.Bounds[int]{}
			boundsByRegion[regionID] = b
		}
		b.Add(record.GetInt("metro_x"), record.GetInt("metro_y"))
	}
	return boundsByRegion, nil
}

func metroRegionSizes(regions []*core.Record, boundsByRegion map[int]*geom.Bounds[int]) map[int]regionSize {
	const fallbackSize = 1000.0
	out := map[int]regionSize{}
	for _, region := range regions {
		id := region.GetInt("eve_id")
		b, ok := boundsByRegion[id]
		if !ok || !b.Seen {
			out[id] = regionSize{w: fallbackSize, h: fallbackSize}
			continue
		}
		out[id] = regionSize{
			w: max(float64(b.MaxX-b.MinX), fallbackSize),
			h: max(float64(b.MaxY-b.MinY), fallbackSize),
		}
	}
	return out
}

func resolveRegionOverlaps(positions map[int]regionPosF, sizes map[int]regionSize) {
	if len(positions) < minOverlapRegionCount {
		return
	}
	ids := make([]int, 0, len(positions))
	for id := range positions {
		ids = append(ids, id)
	}
	const padding = 600.0
	const iterations = 40
	const damping = 0.35

	for range iterations {
		offsets := overlapOffsets(ids, positions, sizes, padding, damping)
		applyRegionOffsets(positions, offsets)
	}
}

func overlapOffsets(ids []int, positions map[int]regionPosF, sizes map[int]regionSize, padding, damping float64) map[int]regionPosF {
	offsets := map[int]regionPosF{}
	for i := range ids {
		for j := i + 1; j < len(ids); j++ {
			aID, bID := ids[i], ids[j]
			shiftX, shiftY, ok := overlapShift(aID, bID, positions, sizes, padding)
			if !ok {
				continue
			}
			offsets[aID] = regionPosF{x: offsets[aID].x - shiftX*0.5*damping, y: offsets[aID].y - shiftY*0.5*damping}
			offsets[bID] = regionPosF{x: offsets[bID].x + shiftX*0.5*damping, y: offsets[bID].y + shiftY*0.5*damping}
		}
	}
	return offsets
}

func overlapShift(aID, bID int, positions map[int]regionPosF, sizes map[int]regionSize, padding float64) (shiftX, shiftY float64, ok bool) {
	aPos, bPos := positions[aID], positions[bID]
	aSize, bSize := sizes[aID], sizes[bID]
	halfAX := aSize.w/halfDivisor + padding
	halfAY := aSize.h/halfDivisor + padding
	halfBX := bSize.w/halfDivisor + padding
	halfBY := bSize.h/halfDivisor + padding
	dx := bPos.x - aPos.x
	dy := bPos.y - aPos.y
	overlapX := halfAX + halfBX - math.Abs(dx)
	overlapY := halfAY + halfBY - math.Abs(dy)
	if overlapX <= 0 || overlapY <= 0 {
		return 0, 0, false
	}
	dx = nonZeroAxis(dx, aID%2 == 0)
	dy = nonZeroAxis(dy, aID%3 == 0)
	return overlapX * math.Copysign(1, dx), overlapY * math.Copysign(1, dy), true
}

func nonZeroAxis(delta float64, preferNegative bool) float64 {
	if delta != 0 {
		return delta
	}

	if preferNegative {
		return -1
	}
	return 1
}

func applyRegionOffsets(positions, offsets map[int]regionPosF) {
	for id, offset := range offsets {
		positions[id] = regionPosF{x: positions[id].x + offset.x, y: positions[id].y + offset.y}
	}
}

func runRegionLayout(ctx context.Context, regions, gates []*core.Record) (map[int]regionPos, error) {
	dot := buildRegionLayoutDOT(regions, gates)
	positions, layoutErr := runNeato(ctx, dot)
	if layoutErr != nil {
		return nil, layoutErr
	}
	out := map[int]regionPos{}
	for id, pos := range positions {
		out[id] = regionPos(pos)
	}
	return out, nil
}

func buildRegionLayoutDOT(regions, gates []*core.Record) string {
	graph := new(strings.Builder)
	graph.WriteString("graph G {\n")
	graph.WriteString("  overlap=false;\n")
	graph.WriteString("  node [fixedsize=true, shape=box, width=15, height=15];\n")

	for _, region := range regions {
		id := region.GetInt("eve_id")
		x := region.GetInt("metro_x")
		y := region.GetInt("metro_y")
		if x != 0 || y != 0 {
			fmt.Fprintf(graph, "  %d [pos=\"%d,%d\"];\n", id, x, y)
		} else {
			fmt.Fprintf(graph, "  %d;\n", id)
		}
	}

	seen := map[string]struct{}{}
	for _, gate := range gates {
		from := gate.GetInt("from_region")
		to := gate.GetInt("to_region")
		if from == 0 || to == 0 || from == to {
			continue
		}
		key := fmt.Sprintf("%d-%d", min(from, to), max(from, to))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		fmt.Fprintf(graph, "  %d -- %d;\n", from, to)
	}

	graph.WriteString("}\n")
	return graph.String()
}
