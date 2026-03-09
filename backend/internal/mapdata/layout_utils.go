package mapdata

import "github.com/pocketbase/pocketbase/core"

func regionIDSet(regions []*core.Record) map[int]struct{} {
	regionSet := map[int]struct{}{}
	for _, region := range regions {
		regionSet[region.GetInt("eve_id")] = struct{}{}
	}
	return regionSet
}

func normalizeScale(minX, minY, maxX, maxY, target float64) float64 {
	scale := 1.0
	dx := maxX - minX
	dy := maxY - minY
	if dx <= 0 && dy <= 0 {
		return scale
	}

	if dx > dy {
		return target / dx
	}
	return target / dy
}

func extendFloatBounds(x, y, minX, minY, maxX, maxY float64, first bool) (nextMinX, nextMinY, nextMaxX, nextMaxY float64) {
	if first {
		return x, y, x, y
	}
	minX = min(minX, x)
	minY = min(minY, y)
	maxX = max(maxX, x)
	maxY = max(maxY, y)
	return minX, minY, maxX, maxY
}
