package pagination

// OffsetForPage calculates offset for 1-based page and positive limit.
func OffsetForPage(page, limit int) int {
	if page <= 1 || limit <= 0 {
		return 0
	}
	return (page - 1) * limit
}

// LimitPlusOne returns limit+1 for sentinel paging fetches.
func LimitPlusOne(limit int) int {
	if limit < 0 {
		return 1
	}
	return limit + 1
}

// SliceByOffsetLimit returns a page slice and whether additional items exist.
func SliceByOffsetLimit[T any](items []T, offset, limit int) ([]T, bool) {
	if limit <= 0 {
		return nil, false
	}
	if offset < 0 {
		offset = 0
	}
	hasMore := len(items) > offset+limit
	if offset >= len(items) {
		return nil, hasMore
	}
	end := min(offset+limit, len(items))
	return items[offset:end], hasMore
}

// TrimToLimit returns up to limit items and whether additional items were trimmed.
func TrimToLimit[T any](items []T, limit int) ([]T, bool) {
	if limit <= 0 {
		return nil, len(items) > 0
	}
	if len(items) <= limit {
		return items, false
	}
	return items[:limit], true
}
