package collections

func MarkSeen[T comparable](seen map[T]struct{}, value T) bool {
	if seen == nil {
		return false
	}

	if _, ok := seen[value]; ok {
		return false
	}
	seen[value] = struct{}{}
	return true
}

// AppendUnique appends value to dst when unseen and marks it in seen.
func AppendUnique[T comparable](dst *[]T, seen map[T]struct{}, value T) bool {
	if dst == nil || seen == nil {
		return false
	}

	if !MarkSeen(seen, value) {
		return false
	}
	*dst = append(*dst, value)
	return true
}

// Dedupe returns input values with duplicates removed while preserving order.
func Dedupe[T comparable](items []T) []T {
	if len(items) == 0 {
		return nil
	}
	out := make([]T, 0, len(items))
	seen := make(map[T]struct{}, len(items))
	for _, item := range items {
		_ = AppendUnique(&out, seen, item)
	}
	return out
}

// ToSet builds a set map from a slice.
func ToSet[T comparable](items []T) map[T]struct{} {
	if len(items) == 0 {
		return nil
	}
	set := make(map[T]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}
