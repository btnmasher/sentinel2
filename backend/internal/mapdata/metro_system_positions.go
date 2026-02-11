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

var dotPosRe = regexp.MustCompile(`^\s*([0-9]+)\s+\[.*?pos="([0-9eE+.\-]+),([0-9eE+.\-]+)`)

func CalculateSystemGraphs(ctx context.Context, app *pocketbase.PocketBase) error {
	regions, regionsErr := app.FindRecordsByFilter(store.CollectionRegions, "eve_id < 11000000", "name", 0, 0, nil)
	if regionsErr != nil {
		logging.New(app).
			WithErr(regionsErr).
			Warn("system graph region lookup failed")
		return regionsErr
	}

	start := time.Now()
	logging.New(app).
		WithFields(logging.Fields{
			"region_count": len(regions),
		}).
		Debug("system graph calc started")

	for _, region := range regions {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if calcErr := calculateRegionGraph(ctx, app, region); calcErr != nil {
			logging.New(app).
				WithFields(logging.Fields{
					"region_id":   region.GetInt("eve_id"),
					"region_name": region.GetString("name"),
				}).
				WithErr(calcErr).
				Warn("system graph calc failed")
			return calcErr
		}
	}
	logging.New(app).
		WithFields(logging.Fields{
			"duration_ms": time.Since(start).Milliseconds(),
		}).
		Debug("system graph calc completed")
	return nil
}

func calculateRegionGraph(ctx context.Context, app *pocketbase.PocketBase, region *core.Record) error {
	regionID := int(region.GetInt("eve_id"))

	gateRecords, gateErr := app.FindRecordsByFilter(
		store.CollectionGates,
		"from_region = {:id} || to_region = {:id}",
		"",
		0,
		0,
		map[string]any{"id": regionID},
	)
	if gateErr != nil {
		logging.New(app).
			WithFields(logging.Fields{
				"region_id": regionID,
			}).
			WithErr(gateErr).
			Warn("system graph gate lookup failed")
		return gateErr
	}

	systemIDs := map[int]struct{}{}
	for i, gate := range gateRecords {
		if i%500 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		systemIDs[int(gate.GetInt("from_solarsystem"))] = struct{}{}
		systemIDs[int(gate.GetInt("to_solarsystem"))] = struct{}{}
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
		logging.New(app).
			WithFields(logging.Fields{
				"region_id": regionID,
			}).
			WithErr(systemsErr).
			Warn("system graph system lookup failed")
		return systemsErr
	}
	for i, system := range systems {
		if i%500 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		systemIDs[int(system.GetInt("eve_id"))] = struct{}{}
	}

	dot := new(strings.Builder)
	dot.WriteString("graph G {\n")
	dot.WriteString("  overlap=false;\n")
	dot.WriteString("  start=1;\n")
	dot.WriteString("  node [shape=point];\n")

	systemList := make([]int, 0, len(systemIDs))
	for id := range systemIDs {
		systemList = append(systemList, id)
	}
	sort.Ints(systemList)
	for _, id := range systemList {
		dot.WriteString(fmt.Sprintf("  %d;\n", id))
	}

	type edge struct {
		from int
		to   int
	}
	edges := map[string]edge{}
	for i, gate := range gateRecords {
		if i%500 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		from := int(gate.GetInt("from_solarsystem"))
		to := int(gate.GetInt("to_solarsystem"))
		if from == 0 || to == 0 {
			continue
		}
		if from > to {
			from, to = to, from
		}
		key := fmt.Sprintf("%d-%d", from, to)
		if _, ok := edges[key]; ok {
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
	for _, e := range edgeList {
		dot.WriteString(fmt.Sprintf("  %d -- %d;\n", e.from, e.to))
	}

	dot.WriteString("}\n")

	positions, positionsErr := runNeato(ctx, dot.String())
	if positionsErr != nil {
		logging.New(app).
			WithFields(logging.Fields{
				"region_id": regionID,
			}).
			WithErr(positionsErr).
			Warn("system graph layout failed")
		return positionsErr
	}

	for i, id := range sortedKeys(positions) {
		if i%200 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}
		pos := positions[id]
		system, systemErr := findSystemByID(app, id)
		if systemErr != nil {
			continue
		}

		if int(system.GetInt("region_id")) == regionID {
			system.Set("metro_x", pos.x)
			system.Set("metro_y", pos.y)
			if saveErr := app.Save(system); saveErr != nil {
				logging.New(app).
					WithFields(logging.Fields{
						"region_id":   regionID,
						"system_id":   system.GetInt("eve_id"),
						"system_name": system.GetString("name"),
					}).
					WithErr(saveErr).
					Debug("system graph save failed")
			}
		} else {
			updateRegionGateCoords(app, system, region, pos.x, pos.y, "metro")
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

func runNeato(ctx context.Context, dot string) (map[int]nodePos, error) {
	graph, graphErr := graphviz.ParseBytes([]byte(dot))
	if graphErr != nil {
		return nil, fmt.Errorf("graphviz parse failed: %w", graphErr)
	}
	defer graph.Close()

	g, graphvizErr := graphviz.New(ctx)
	if graphvizErr != nil {
		return nil, fmt.Errorf("graphviz init failed: %w", graphvizErr)
	}
	defer g.Close()

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
