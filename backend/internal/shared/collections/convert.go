package collections

// ToIntSlice converts int-like values to []int.
func ToIntSlice[T ~int32 | ~int64](values []T) []int {
	out := make([]int, 0, len(values))
	for _, value := range values {
		out = append(out, int(value))
	}
	return out
}
