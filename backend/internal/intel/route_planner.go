package intel

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"sort"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

const RebuildExpiry = 300

type RoutePlanner struct {
	App             *pocketbase.PocketBase
	initializedHash string
	lastRebuild     time.Time
	graph           map[int][]edge
}

type edge struct {
	to   int
	kind string
}

func NewRoutePlanner(app *pocketbase.PocketBase) *RoutePlanner {
	return &RoutePlanner{App: app}
}

func (r *RoutePlanner) GenerateRoute(source int, destination int, avoid []int) ([]int, []int, error) {
	if r.graph == nil || time.Since(r.lastRebuild) > RebuildExpiry*time.Second {
		if populateErr := r.populateGraph(); populateErr != nil {
			return nil, nil, populateErr
		}
	}

	graph := r.copyGraph()
	for _, node := range avoid {
		delete(graph, node)
		for k := range graph {
			graph[k] = filterEdges(graph[k], node)
		}
	}

	path := bfsShortestPath(graph, source, destination)
	if len(path) == 0 {
		return nil, nil, sql.ErrNoRows
	}

	jumpgatePath := []int{}
	for i := 0; i < len(path)-1; i++ {
		edgeType := edgeTypeBetween(graph, path[i], path[i+1])
		if edgeType == "bridge" {
			gate, gateErr := r.findLatestJumpbridge(path[i], path[i+1])
			if gateErr == nil && gate != 0 {
				jumpgatePath = append(jumpgatePath, gate)
			}
		}
	}

	if len(jumpgatePath) == 0 || jumpgatePath[len(jumpgatePath)-1] != destination {
		jumpgatePath = append(jumpgatePath, destination)
	}

	return path, jumpgatePath, nil
}

func (r *RoutePlanner) populateGraph() error {
	r.lastRebuild = time.Now()

	jumpbridges, jumpErr := r.App.FindRecordsByFilter(store.CollectionJumpbridges, "", "", 0, 0, nil)
	if jumpErr != nil {
		return jumpErr
	}

	jumpHash := hashJumpbridges(jumpbridges)
	if jumpHash == r.initializedHash && r.graph != nil {
		return nil
	}

	gates, gatesErr := r.App.FindRecordsByFilter(store.CollectionGates, "", "", 0, 0, nil)
	if gatesErr != nil {
		return gatesErr
	}

	start := time.Now()
	logging.New(r.App).
		WithFields(logging.Fields{
			"jumpbridge_cnt":  len(jumpbridges),
			"gate_cnt":        len(gates),
			"jumpbridge_hash": jumpHash,
		}).
		Debug("route planner graph rebuild started")

	graph := map[int][]edge{}

	for _, gate := range gates {
		from := int(gate.GetInt("from_solarsystem"))
		to := int(gate.GetInt("to_solarsystem"))
		graph[from] = append(graph[from], edge{to: to, kind: "gate"})
		graph[to] = append(graph[to], edge{to: from, kind: "gate"})
	}

	for _, jb := range jumpbridges {
		from := int(jb.GetInt("from_solarsystem"))
		to := int(jb.GetInt("to_solarsystem"))
		graph[from] = append(graph[from], edge{to: to, kind: "bridge"})
		graph[to] = append(graph[to], edge{to: from, kind: "bridge"})
	}

	r.graph = graph
	r.initializedHash = jumpHash

	logging.New(r.App).
		WithFields(logging.Fields{
			"node_cnt":        len(graph),
			"jumpbridge_hash": jumpHash,
			"duration_ms":     time.Since(start).Milliseconds(),
		}).
		Debug("route planner graph rebuild completed")
	return nil
}

func (r *RoutePlanner) findLatestJumpbridge(from int, to int) (int, error) {
	records, recordsErr := r.App.FindRecordsByFilter(
		store.CollectionJumpbridges,
		"from_solarsystem = {:from} && to_solarsystem = {:to}",
		"-created_date",
		1,
		0,
		map[string]any{"from": from, "to": to},
	)
	if recordsErr != nil {
		return 0, recordsErr
	}
	if len(records) == 0 {
		return 0, nil
	}
	return int(records[0].GetInt("structure_id")), nil
}

func hashJumpbridges(records []*core.Record) string {
	if len(records) == 0 {
		return ""
	}
	ids := make([]int, 0, len(records))
	for _, rec := range records {
		ids = append(ids, int(rec.GetInt("structure_id")))
	}
	sort.Ints(ids)
	payload := ""
	for _, id := range ids {
		payload += strconv.Itoa(id) + ","
	}
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func (r *RoutePlanner) copyGraph() map[int][]edge {
	out := make(map[int][]edge, len(r.graph))
	for k, v := range r.graph {
		outEdges := make([]edge, len(v))
		copy(outEdges, v)
		out[k] = outEdges
	}
	return out
}

func filterEdges(edges []edge, remove int) []edge {
	out := edges[:0]
	for _, e := range edges {
		if e.to != remove {
			out = append(out, e)
		}
	}
	return out
}

func bfsShortestPath(graph map[int][]edge, source int, destination int) []int {
	if source == destination {
		return []int{source}
	}

	visited := map[int]bool{source: true}
	parent := map[int]int{}
	queue := []int{source}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		for _, e := range graph[node] {
			if visited[e.to] {
				continue
			}
			visited[e.to] = true
			parent[e.to] = node
			if e.to == destination {
				return buildPath(parent, source, destination)
			}
			queue = append(queue, e.to)
		}
	}
	return nil
}

func buildPath(parent map[int]int, source int, destination int) []int {
	path := []int{destination}
	for current := destination; current != source; {
		p, ok := parent[current]
		if !ok {
			return nil
		}
		path = append(path, p)
		current = p
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func edgeTypeBetween(graph map[int][]edge, from int, to int) string {
	for _, e := range graph[from] {
		if e.to == to {
			return e.kind
		}
	}
	return ""
}
