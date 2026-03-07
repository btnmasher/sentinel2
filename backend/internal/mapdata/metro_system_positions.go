package mapdata

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-graphviz"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

var dotPosRe = regexp.MustCompile(`^\s*(\d+)\s+\[.*?pos="([0-9eE+.\-]+),([0-9eE+.\-]+)`)

const dotMatchParts = 4
const regionGraphContextStride = 500

func CalculateSystemGraphs(ctx context.Context, app *pocketbase.PocketBase) error {
	log := logging.New(app).WithFields(logging.Fields{"component": "mapdata.metro_system_positions"})
	regions, regionsErr := app.FindRecordsByFilter(store.CollectionRegions, "eve_id < 11000000", "name", 0, 0, nil)
	if regionsErr != nil {
		log.WithErr(regionsErr).Warn("system graph region lookup failed")
		return regionsErr
	}

	start := time.Now()
	log.
		WithFields(logging.Fields{
			"region_count": len(regions),
		}).
		Debug("system graph calc started")

	for _, region := range regions {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if calcErr := calculateRegionGraph(ctx, app, region, log); calcErr != nil {
			log.
				WithFields(logging.Fields{
					"region_id":   region.GetInt("eve_id"),
					"region_name": region.GetString("name"),
				}).
				WithErr(calcErr).
				Warn("system graph calc failed")
			return calcErr
		}
	}
	log.
		WithFields(logging.Fields{
			"duration_ms": time.Since(start).Milliseconds(),
		}).
		Debug("system graph calc completed")
	return nil
}

func calculateRegionGraph(ctx context.Context, app *pocketbase.PocketBase, region *core.Record, log *logging.Logger) error {
	regionID := region.GetInt("eve_id")

	gateRecords, systems, loadErr := loadRegionGraphInputs(ctx, app, regionID, log)
	if loadErr != nil {
		return loadErr
	}
	dot, dotErr := buildRegionDOT(ctx, gateRecords, systems)
	if dotErr != nil {
		return dotErr
	}
	positions, positionsErr := runNeato(ctx, dot)
	if positionsErr != nil {
		log.WithFields(logging.Fields{
			"region_id": regionID,
		}).
			WithErr(positionsErr).
			Warn("system graph layout failed")
		return positionsErr
	}

	return saveRegionGraphPositions(ctx, app, region, positions, log)
}

func loadRegionGraphInputs(ctx context.Context, app *pocketbase.PocketBase, regionID int, log *logging.Logger) (gateRecords, systems []*core.Record, err error) {
	gateRecords, gateErr := app.FindRecordsByFilter(
		store.CollectionGates,
		"from_region = {:id} || to_region = {:id}",
		"",
		0,
		0,
		map[string]any{"id": regionID},
	)
	if gateErr != nil {
		log.WithFields(logging.Fields{"region_id": regionID}).WithErr(gateErr).Warn("system graph gate lookup failed")
		return nil, nil, gateErr
	}
	systems, systemsErr := app.FindRecordsByFilter(
		store.CollectionSolarSystems,
		"region_id = {:id}",
		"",
		0,
		0,
		map[string]any{"id": regionID},
	)
	if systemsErr != nil {
		log.WithFields(logging.Fields{"region_id": regionID}).WithErr(systemsErr).Warn("system graph system lookup failed")
		return nil, nil, systemsErr
	}
	if err := checkContextStride(ctx, len(gateRecords), regionGraphContextStride); err != nil {
		return nil, nil, err
	}
	if err := checkContextStride(ctx, len(systems), regionGraphContextStride); err != nil {
		return nil, nil, err
	}
	return gateRecords, systems, nil
}

func checkContextStride(ctx context.Context, total, stride int) error {
	if stride <= 0 {
		return nil
	}
	for i := range total {
		if i%stride == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return nil
}

func buildRegionDOT(ctx context.Context, gateRecords, systems []*core.Record) (string, error) {
	systemIDs, gatherErr := gatherRegionSystemIDs(ctx, gateRecords, systems)
	if gatherErr != nil {
		return "", gatherErr
	}
	edges, edgeErr := gatherRegionEdges(ctx, gateRecords)
	if edgeErr != nil {
		return "", edgeErr
	}
	dot := new(strings.Builder)
	dot.WriteString("graph G {\n")
	dot.WriteString("  overlap=false;\n")
	dot.WriteString("  start=1;\n")
	dot.WriteString("  node [shape=point];\n")
	appendSystemNodes(dot, systemIDs)
	appendGateEdges(dot, edges)
	dot.WriteString("}\n")
	return dot.String(), nil
}

type edge struct {
	from int
	to   int
}

func gatherRegionSystemIDs(ctx context.Context, gateRecords, systems []*core.Record) (map[int]struct{}, error) {
	systemIDs := map[int]struct{}{}
	for i, gate := range gateRecords {
		if i%500 == 0 && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		systemIDs[gate.GetInt("from_solarsystem")] = struct{}{}
		systemIDs[gate.GetInt("to_solarsystem")] = struct{}{}
	}
	for i, system := range systems {
		if i%500 == 0 && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		systemIDs[system.GetInt("eve_id")] = struct{}{}
	}
	return systemIDs, nil
}

func gatherRegionEdges(ctx context.Context, gateRecords []*core.Record) ([]edge, error) {
	edges := map[string]edge{}
	for i, gate := range gateRecords {
		if i%500 == 0 && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		from, to, ok := normalizeEdge(gate.GetInt("from_solarsystem"), gate.GetInt("to_solarsystem"))
		if !ok {
			continue
		}
		key := fmt.Sprintf("%d-%d", from, to)
		if _, exists := edges[key]; exists {
			continue
		}
		edges[key] = edge{from: from, to: to}
	}
	edgeList := make([]edge, 0, len(edges))
	for _, value := range edges {
		edgeList = append(edgeList, value)
	}
	sort.Slice(edgeList, func(i, j int) bool {
		if edgeList[i].from == edgeList[j].from {
			return edgeList[i].to < edgeList[j].to
		}
		return edgeList[i].from < edgeList[j].from
	})
	return edgeList, nil
}

func normalizeEdge(from, to int) (left, right int, ok bool) {
	if from == 0 || to == 0 {
		return 0, 0, false
	}
	if from > to {
		from, to = to, from
	}
	return from, to, true
}

func appendSystemNodes(dot *strings.Builder, systemIDs map[int]struct{}) {
	systemList := make([]int, 0, len(systemIDs))
	for id := range systemIDs {
		systemList = append(systemList, id)
	}
	sort.Ints(systemList)
	for _, id := range systemList {
		fmt.Fprintf(dot, "  %d;\n", id)
	}
}

func appendGateEdges(dot *strings.Builder, edgeList []edge) {
	for _, e := range edgeList {
		fmt.Fprintf(dot, "  %d -- %d;\n", e.from, e.to)
	}
}

func saveRegionGraphPositions(ctx context.Context, app *pocketbase.PocketBase, region *core.Record, positions map[int]nodePos, log *logging.Logger) error {
	regionID := region.GetInt("eve_id")
	for i, id := range sortedKeys(positions) {
		if i%200 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		pos := positions[id]
		system, systemErr := findSystemByID(app, id)
		if systemErr != nil {
			continue
		}
		if system.GetInt("region_id") != regionID {
			updateRegionGateCoords(app, system, region, pos.x, pos.y, "metro")
			continue
		}
		system.Set("metro_x", pos.x)
		system.Set("metro_y", pos.y)
		if saveErr := app.Save(system); saveErr != nil {
			log.WithFields(logging.Fields{
				"region_id":   regionID,
				"system_id":   system.GetInt("eve_id"),
				"system_name": system.GetString("name"),
			}).
				WithErr(saveErr).
				Debug("system graph save failed")
		}
	}
	return nil
}

func sortedKeys(values map[int]nodePos) []int {
	keys := make([]int, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}

type nodePos struct {
	x int
	y int
}

func runNeato(ctx context.Context, dot string) (positions map[int]nodePos, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("graphviz runtime failure: %v", recovered)
		}
	}()

	graph, graphErr := graphviz.ParseBytes([]byte(dot))
	if graphErr != nil {
		return nil, fmt.Errorf("graphviz parse failed: %w", graphErr)
	}
	defer func() {
		if graph != nil {
			_ = graph.Close()
		}
	}()

	g, graphvizErr := graphviz.New(ctx)
	if graphvizErr != nil {
		return nil, fmt.Errorf("graphviz init failed: %w", graphvizErr)
	}
	defer func() {
		if g != nil {
			_ = g.Close()
		}
	}()

	g.SetLayout(graphviz.NEATO)

	var output bytes.Buffer
	if renderErr := g.Render(ctx, graph, "plain", &output); renderErr != nil {
		return nil, fmt.Errorf("graphviz layout failed: %w", renderErr)
	}

	positions, parseErr := parsePlainPositions(output.Bytes())
	if parseErr != nil {
		return nil, parseErr
	}
	if len(positions) > 0 {
		return positions, nil
	}

	output.Reset()
	if renderErr := g.Render(ctx, graph, "dot", &output); renderErr != nil {
		return nil, fmt.Errorf("graphviz layout failed: %w", renderErr)
	}

	return parseDotPositions(output.Bytes())
}

func parseDotPositions(dot []byte) (map[int]nodePos, error) {
	positions := map[int]nodePos{}
	scanner := bufio.NewScanner(bytes.NewReader(dot))
	for scanner.Scan() {
		line := scanner.Text()
		match := dotPosRe.FindStringSubmatch(line)
		if len(match) != dotMatchParts {
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
		positions[id] = nodePos{x: int(x), y: int(y)}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, scanErr
	}
	return positions, nil
}

func parsePlainPositions(dot []byte) (map[int]nodePos, error) {
	positions := map[int]nodePos{}
	scanner := bufio.NewScanner(bytes.NewReader(dot))
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
		positions[id] = nodePos{x: int(x), y: int(y)}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, scanErr
	}
	return positions, nil
}
