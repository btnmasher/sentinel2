package maps

import "testing"

func TestNormalizeRegionToken(t *testing.T) {
	t.Parallel()

	got := normalizeRegionToken("  The_Forge ")
	if got != "The Forge" {
		t.Fatalf("normalizeRegionToken() = %q, want %q", got, "The Forge")
	}
}

func TestOverlap(t *testing.T) {
	t.Parallel()

	if !overlap([]int{1, 2, 3}, []int{3, 4}) {
		t.Fatal("expected overlap to return true")
	}

	if overlap([]int{1, 2}, []int{3, 4}) {
		t.Fatal("expected overlap to return false")
	}
}

func TestNormalizeRegions(t *testing.T) {
	t.Parallel()

	regions := map[int]Region{
		1: {Position: struct {
			X int `json:"x"`
			Y int `json:"y"`
		}{X: -5, Y: 10}},
		2: {Position: struct {
			X int `json:"x"`
			Y int `json:"y"`
		}{X: 10, Y: -2}},
	}

	normalizeRegions(regions)

	if regions[1].Position.X != 0 || regions[2].Position.Y != 0 {
		t.Fatalf("normalizeRegions() failed: %#v", regions)
	}
}

func TestCollectRegionBoundsAndScale(t *testing.T) {
	t.Parallel()

	systems := map[int]System{
		1: systemWithPos(1, 10, 10),
		2: systemWithPos(1, 20, 30),
		3: systemWithPos(2, 100, 100), // out-of-scope region
	}

	bounds := collectRegionBounds(systems, []int{1})
	if bounds[1] == nil || bounds[1].MinX != 10 || bounds[1].MaxY != 30 {
		t.Fatalf("collectRegionBounds() unexpected bounds: %#v", bounds[1])
	}

	applyRegionScale(systems, 1, bounds[1], 100, 100)
	if systems[1].Position.X != 0 || systems[1].Position.Y != 0 {
		t.Fatalf("expected first system normalized to origin, got %+v", systems[1].Position)
	}

	if systems[2].Position.X != 50 || systems[2].Position.Y != 100 {
		t.Fatalf("expected scaled system to (50,100), got %+v", systems[2].Position)
	}
}

func systemWithPos(regionID, x, y int) System {
	system := System{Region: regionID}
	system.Position.X = x
	system.Position.Y = y
	return system
}
