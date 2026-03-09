package jobs

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunBatchedRunsAllItems(t *testing.T) {
	items := []int{1, 2, 3}
	seen := make([]int, 0, len(items))

	err := RunBatched(context.Background(), items, 2, 0, func(_ context.Context, _ int, item int) error {
		seen = append(seen, item)
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(seen) != len(items) {
		t.Fatalf("expected %d items, got %d", len(items), len(seen))
	}
	for i := range items {
		if seen[i] != items[i] {
			t.Fatalf("expected item %d at %d, got %d", items[i], i, seen[i])
		}
	}
}

func TestRunBatchedStopsOnRunError(t *testing.T) {
	items := []int{1, 2, 3}
	calls := 0
	expected := errors.New("stop")

	err := RunBatched(context.Background(), items, 2, 0, func(_ context.Context, _ int, item int) error {
		calls++
		if item == 2 {
			return expected
		}
		return nil
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}

	if calls != 2 {
		t.Fatalf("expected 2 calls before stop, got %d", calls)
	}
}

func TestRunBatchedRespectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := RunBatched(ctx, []int{1, 2, 3}, 2, 0, func(_ context.Context, _ int, _ int) error {
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestRunBatchedHonorsPauseAtBoundary(t *testing.T) {
	start := time.Now()

	err := RunBatched(context.Background(), []int{1, 2}, 1, 20*time.Millisecond, func(_ context.Context, _ int, _ int) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if time.Since(start) < 35*time.Millisecond {
		t.Fatalf("expected boundary pauses to elapse")
	}
}
