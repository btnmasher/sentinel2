package maps

import (
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
)

func normalizeSystemsByRegion(systems map[int]SystemDTO, regionIDs []int, tx int, ty int) {
	regionSet := map[int]struct{}{}
	for _, id := range regionIDs {
		regionSet[id] = struct{}{}
	}

	type bounds struct {
		minX int
		minY int
		maxX int
		maxY int
	}
	regionBounds := map[int]*bounds{}

	for _, system := range systems {
		if _, ok := regionSet[int(system.Region)]; !ok {
			continue
		}
		b, exists := regionBounds[int(system.Region)]
		if !exists {
			regionBounds[int(system.Region)] = &bounds{
				minX: system.Position.X,
				minY: system.Position.Y,
				maxX: system.Position.X,
				maxY: system.Position.Y,
			}
			continue
		}
		if system.Position.X < b.minX {
			b.minX = system.Position.X
		}
		if system.Position.Y < b.minY {
			b.minY = system.Position.Y
		}
		if system.Position.X > b.maxX {
			b.maxX = system.Position.X
		}
		if system.Position.Y > b.maxY {
			b.maxY = system.Position.Y
		}
	}

	for id, b := range regionBounds {
		dx := b.maxX - b.minX
		dy := b.maxY - b.minY
		if dx == 0 || dy == 0 {
			continue
		}
		scale := float64(tx) / float64(dx)
		if yScale := float64(ty) / float64(dy); yScale < scale {
			scale = yScale
		}
		for systemID, system := range systems {
			if int(system.Region) != id {
				continue
			}
			system.Position.X = int(scale * float64(system.Position.X-b.minX))
			system.Position.Y = int(scale * float64(system.Position.Y-b.minY))
			systems[systemID] = system
		}
	}
}

func normalizeRegions(regions map[int]RegionDTO) {
	if len(regions) == 0 {
		return
	}
	minX := 0
	minY := 0
	first := true
	for _, region := range regions {
		if first {
			minX = region.Position.X
			minY = region.Position.Y
			first = false
			continue
		}
		if region.Position.X < minX {
			minX = region.Position.X
		}
		if region.Position.Y < minY {
			minY = region.Position.Y
		}
	}
	for id, region := range regions {
		region.Position.X -= minX
		region.Position.Y -= minY
		regions[id] = region
	}
}

func buildFilter[T filterValue](field string, ids []T) (string, dbx.Params) {
	if len(ids) == 0 {
		return "", dbx.Params{}
	}
	var filter strings.Builder
	filter.WriteString(field)
	filter.WriteString(" = {:id0}")
	params := dbx.Params{"id0": ids[0]}
	for i := 1; i < len(ids); i++ {
		key := "id" + strconv.Itoa(i)
		filter.WriteString(" || ")
		filter.WriteString(field)
		filter.WriteString(" = {:")
		filter.WriteString(key)
		filter.WriteString("}")
		params[key] = ids[i]
	}
	return filter.String(), params
}

func normalizeRegionToken(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
}

func overlap(a []int, b []int) bool {
	set := map[int]struct{}{}
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}
