package mapdata

import "testing"

func TestNormalizeScale(t *testing.T) {
	t.Parallel()

	scale := normalizeScale(0, 0, 10, 20, 100)
	if scale != 5 {
		t.Fatalf("normalizeScale() = %v, want 5", scale)
	}
}

func TestExtendFloatBounds(t *testing.T) {
	t.Parallel()

	minX, minY, maxX, maxY := extendFloatBounds(5, 10, 0, 0, 0, 0, true)
	minX, minY, maxX, maxY = extendFloatBounds(-3, 15, minX, minY, maxX, maxY, false)

	if minX != -3 || minY != 10 || maxX != 5 || maxY != 15 {
		t.Fatalf("extendFloatBounds() unexpected bounds: (%v,%v)-(%v,%v)", minX, minY, maxX, maxY)
	}
}

func TestParseBuildNumberLine(t *testing.T) {
	t.Parallel()

	build, ok := parseBuildNumberLine(`{"_key":"sde","_value":{"build":"12345"}}`)
	if !ok || build != "12345" {
		t.Fatalf("parseBuildNumberLine() = (%q,%v), want (12345,true)", build, ok)
	}

	if _, ok := parseBuildNumberLine(`{"_key":"other","_value":{"build":"1"}}`); ok {
		t.Fatal("expected non-sde line to return false")
	}
}
