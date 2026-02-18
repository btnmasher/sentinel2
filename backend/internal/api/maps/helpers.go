package maps

import (
	"strings"

	"sentinel2/internal/shared/geom"
)

func normalizeSystemsByRegion(systems map[int]SystemDTO, regionIDs []int, tx, ty int) {
	regionBounds := collectRegionBounds(systems, regionIDs)
	for regionID, regionBoundsValue := range regionBounds {
		applyRegionScale(systems, regionID, regionBoundsValue, tx, ty)
	}
}

func collectRegionBounds(systems map[int]SystemDTO, regionIDs []int) map[int]*geom.Bounds[int] {
	regionSet := map[int]struct{}{}
	for _, id := range regionIDs {
		regionSet[id] = struct{}{}
	}
	regionBoundsMap := map[int]*geom.Bounds[int]{}
	for _, system := range systems {
		regionID := system.Region
		if _, ok := regionSet[regionID]; !ok {
			continue
		}
		bounds := regionBoundsMap[regionID]
		if bounds == nil {
			bounds = &geom.Bounds[int]{}
			regionBoundsMap[regionID] = bounds
		}
		bounds.Add(system.Position.X, system.Position.Y)
	}
	return regionBoundsMap
}

func applyRegionScale(systems map[int]SystemDTO, regionID int, bounds *geom.Bounds[int], tx, ty int) {
	if bounds == nil || !bounds.Seen {
		return
	}
	dx := bounds.MaxX - bounds.MinX
	dy := bounds.MaxY - bounds.MinY
	if dx == 0 || dy == 0 {
		return
	}
	scale := float64(tx) / float64(dx)
	yScale := float64(ty) / float64(dy)
	scale = min(scale, yScale)
	for systemID, system := range systems {
		if system.Region != regionID {
			continue
		}
		system.Position.X = int(scale * float64(system.Position.X-bounds.MinX))
		system.Position.Y = int(scale * float64(system.Position.Y-bounds.MinY))
		systems[systemID] = system
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
		minX = min(minX, region.Position.X)
		minY = min(minY, region.Position.Y)
	}
	for id, region := range regions {
		region.Position.X -= minX
		region.Position.Y -= minY
		regions[id] = region
	}
}

func normalizeRegionToken(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
}

func overlap(a, b []int) bool {
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
