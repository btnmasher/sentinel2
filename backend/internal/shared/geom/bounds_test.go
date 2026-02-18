package geom

import "testing"

func TestBoundsAdd(t *testing.T) {
	t.Parallel()

	var b Bounds[float64]
	b.Add(5, 10)
	b.Add(-3, 7)
	b.Add(8, 20)

	if !b.Seen {
		t.Fatal("expected Seen=true")
	}
	if b.Count != 3 {
		t.Fatalf("expected Count=3, got %d", b.Count)
	}
	if b.MinX != -3 || b.MinY != 7 || b.MaxX != 8 || b.MaxY != 20 {
		t.Fatalf("unexpected bounds: %+v", b)
	}
}
