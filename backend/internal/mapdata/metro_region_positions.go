package mapdata

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-graphviz"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

var regionPosRe = regexp.MustCompile(`^\s*([0-9]+)\s+\[.*?pos="([0-9eE+.\-]+),([0-9eE+.\-]+)`)

func CalculateRegionLayouts(ctx context.Context, app *pocketbase.PocketBase) error {
	regions, regionsErr := app.FindRecordsByFilter(store.CollectionRegions, "eve_id < 11000000", "name", 0, 0, nil)
	if regionsErr != nil {
		logging.New(app).
			WithErr(regionsErr).
			Warn("region layout lookup failed")
		return regionsErr
	}

	gates, gatesErr := app.FindRecordsByFilter(store.CollectionGates, "", "", 0, 0, nil)
	if gatesErr != nil {
		logging.New(app).
			WithErr(gatesErr).
			Warn("region layout gate lookup failed")
		return gatesErr
	}

	start := time.Now()
	logging.New(app).
		WithFields(logging.Fields{
			"region_count": len(regions),
		}).
		Debug("region layout calc started")

	if ctx.Err() != nil {
		return ctx.Err()
	}
	positions := map[int]regionPos{}
	useEve2D := false
	for _, region := range regions {
		id := region.GetInt("eve_id")
		x := region.GetInt("eve2d_x")
		y := region.GetInt("eve2d_y")
		if x != 0 || y != 0 {
			positions[id] = regionPos{x: x, y: y}
			useEve2D = true
		}
	}
	if !useEve2D {
		var layoutErr error
		positions, layoutErr = runRegionLayout(ctx, regions, gates)
		if layoutErr != nil {
			logging.New(app).
				WithErr(layoutErr).
				Warn("region layout calc failed")
			return layoutErr
		}
	}

	regionSizes, sizeErr := buildMetroRegionSizes(ctx, app, regions)
	if sizeErr != nil {
		logging.New(app).
			WithErr(sizeErr).
			Warn("region layout size lookup failed")
		return sizeErr
	}

	minX, minY := 0.0, 0.0
	maxX, maxY := 0.0, 0.0
	first := true
	for _, pos := range positions {
		x := float64(pos.x)
		y := float64(pos.y)
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

	posScaled := map[int]regionPosF{}
	sizeScaled := map[int]regionSize{}
	for id, pos := range positions {
		posScaled[id] = regionPosF{
			x: (float64(pos.x) - minX) * scale,
			y: (float64(pos.y) - minY) * scale,
		}
		if size, ok := regionSizes[id]; ok {
			sizeScaled[id] = regionSize{
				w: size.w * scale,
				h: size.h * scale,
			}
		}
	}

	resolveRegionOverlaps(posScaled, sizeScaled)

	minX = 0
	minY = 0
	first = true
	for _, pos := range posScaled {
		if first {
			minX = pos.x
			minY = pos.y
			first = false
			continue
		}
		if pos.x < minX {
			minX = pos.x
		}
		if pos.y < minY {
			minY = pos.y
		}
	}

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
			logging.New(app).
				WithFields(logging.Fields{
					"region_id":   id,
					"region_name": region.GetString("name"),
				}).
				WithErr(saveErr).
				Debug("region layout save failed")
		}
	}

	logging.New(app).
		WithFields(logging.Fields{
			"duration_ms": time.Since(start).Milliseconds(),
		}).
		Debug("region layout calc completed")
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
	out := map[int]regionSize{}
	regionSet := map[int]struct{}{}
	for _, region := range regions {
		regionSet[region.GetInt("eve_id")] = struct{}{}
	}

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

	type bounds struct {
		minX int
		maxX int
		minY int
		maxY int
		seen bool
	}
	boundsByRegion := map[int]*bounds{}
	for i, record := range records {
		if i%1000 == 0 && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		regionID := record.GetInt("region_id")
		if _, ok := regionSet[regionID]; !ok {
			continue
		}
		x := record.GetInt("metro_x")
		y := record.GetInt("metro_y")
		b, ok := boundsByRegion[regionID]
		if !ok {
			b = &bounds{minX: x, maxX: x, minY: y, maxY: y, seen: true}
			boundsByRegion[regionID] = b
			continue
		}
		if !b.seen {
			b.minX = x
			b.maxX = x
			b.minY = y
			b.maxY = y
			b.seen = true
			continue
		}
		if x < b.minX {
			b.minX = x
		}
		if x > b.maxX {
			b.maxX = x
		}
		if y < b.minY {
			b.minY = y
		}
		if y > b.maxY {
			b.maxY = y
		}
	}

	const fallbackSize = 1000.0
	for _, region := range regions {
		id := region.GetInt("eve_id")
		if b, ok := boundsByRegion[id]; ok && b.seen {
			w := float64(b.maxX - b.minX)
			h := float64(b.maxY - b.minY)
			if w <= 0 {
				w = fallbackSize
			}
			if h <= 0 {
				h = fallbackSize
			}
			out[id] = regionSize{w: w, h: h}
		} else {
			out[id] = regionSize{w: fallbackSize, h: fallbackSize}
		}
	}

	return out, nil
}

func resolveRegionOverlaps(positions map[int]regionPosF, sizes map[int]regionSize) {
	if len(positions) < 2 {
		return
	}
	ids := make([]int, 0, len(positions))
	for id := range positions {
		ids = append(ids, id)
	}
	const padding = 600.0
	const iterations = 40
	const damping = 0.35

	for iter := 0; iter < iterations; iter++ {
		offsets := map[int]regionPosF{}
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				aID := ids[i]
				bID := ids[j]
				aPos := positions[aID]
				bPos := positions[bID]
				aSize := sizes[aID]
				bSize := sizes[bID]
				halfAX := aSize.w/2 + padding
				halfAY := aSize.h/2 + padding
				halfBX := bSize.w/2 + padding
				halfBY := bSize.h/2 + padding

				dx := bPos.x - aPos.x
				dy := bPos.y - aPos.y
				overlapX := halfAX + halfBX - math.Abs(dx)
				overlapY := halfAY + halfBY - math.Abs(dy)
				if overlapX <= 0 || overlapY <= 0 {
					continue
				}
				if dx == 0 {
					if aID%2 == 0 {
						dx = -1
					} else {
						dx = 1
					}
				}
				if dy == 0 {
					if aID%3 == 0 {
						dy = -1
					} else {
						dy = 1
					}
				}
				shiftX := overlapX * math.Copysign(1, dx)
				shiftY := overlapY * math.Copysign(1, dy)
				offsets[aID] = regionPosF{
					x: offsets[aID].x - shiftX*0.5*damping,
					y: offsets[aID].y - shiftY*0.5*damping,
				}
				offsets[bID] = regionPosF{
					x: offsets[bID].x + shiftX*0.5*damping,
					y: offsets[bID].y + shiftY*0.5*damping,
				}
			}
		}
		for id, offset := range offsets {
			positions[id] = regionPosF{
				x: positions[id].x + offset.x,
				y: positions[id].y + offset.y,
			}
		}
	}
}

func runRegionLayout(ctx context.Context, regions []*core.Record, gates []*core.Record) (map[int]regionPos, error) {
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
		key := fmt.Sprintf("%d-%d", minInt(from, to), maxInt(from, to))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		fmt.Fprintf(graph, "  %d -- %d;\n", from, to)
	}

	graph.WriteString("}\n")

	g, graphErr := graphviz.ParseBytes([]byte(graph.String()))
	if graphErr != nil {
		return nil, graphErr
	}
	defer g.Close()

	viz, vizErr := graphviz.New(ctx)
	if vizErr != nil {
		return nil, vizErr
	}
	defer viz.Close()

	viz.SetLayout(graphviz.NEATO)

	var output bytes.Buffer
	if renderErr := viz.Render(ctx, g, "plain", &output); renderErr != nil {
		return nil, renderErr
	}

	positions := map[int]regionPos{}
	scanner := bufio.NewScanner(bytes.NewReader(output.Bytes()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[0] != "node" {
			continue
		}
		id, idErr := strconv.Atoi(fields[1])
		if idErr != nil {
			continue
		}
		x, xErr := strconv.ParseFloat(fields[2], 64)
		if xErr != nil {
			continue
		}
		y, yErr := strconv.ParseFloat(fields[3], 64)
		if yErr != nil {
			continue
		}
		positions[id] = regionPos{x: int(x), y: int(y)}
	}
	if len(positions) > 0 {
		return positions, nil
	}

	output.Reset()
	if renderErr := viz.Render(ctx, g, "dot", &output); renderErr != nil {
		return nil, renderErr
	}

	scanner = bufio.NewScanner(bytes.NewReader(output.Bytes()))
	for scanner.Scan() {
		line := scanner.Text()
		match := regionPosRe.FindStringSubmatch(line)
		if len(match) != 4 {
			continue
		}
		id, idErr := strconv.Atoi(match[1])
		if idErr != nil {
			continue
		}
		x, xErr := strconv.ParseFloat(match[2], 64)
		if xErr != nil {
			continue
		}
		y, yErr := strconv.ParseFloat(match[3], 64)
		if yErr != nil {
			continue
		}
		positions[id] = regionPos{x: int(x), y: int(y)}
	}
	return positions, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
