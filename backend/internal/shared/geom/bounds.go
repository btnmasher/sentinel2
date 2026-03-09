package geom

import "cmp"

type Bounds[T cmp.Ordered] struct {
	MinX  T
	MinY  T
	MaxX  T
	MaxY  T
	Seen  bool
	Count int
}

func (b *Bounds[T]) Add(x, y T) {
	if b == nil {
		return
	}

	if !b.Seen {
		b.MinX = x
		b.MinY = y
		b.MaxX = x
		b.MaxY = y
		b.Seen = true
		b.Count = 1
		return
	}
	b.MinX = min(b.MinX, x)
	b.MinY = min(b.MinY, y)
	b.MaxX = max(b.MaxX, x)
	b.MaxY = max(b.MaxY, y)
	b.Count++
}
