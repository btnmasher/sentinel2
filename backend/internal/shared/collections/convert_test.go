package collections

import "testing"

func TestToIntSlice(t *testing.T) {
	got32 := ToIntSlice([]int32{1, 2, 3})
	if len(got32) != 3 || got32[0] != 1 || got32[1] != 2 || got32[2] != 3 {
		t.Fatalf("unexpected int32 conversion: %#v", got32)
	}

	got64 := ToIntSlice([]int64{4, 5})
	if len(got64) != 2 || got64[0] != 4 || got64[1] != 5 {
		t.Fatalf("unexpected int64 conversion: %#v", got64)
	}
}
