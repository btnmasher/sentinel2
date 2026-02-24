package jobs

import (
	"context"
	"time"
)

// RunBatched executes run for each item and pauses at batch boundaries.
func RunBatched[T any](ctx context.Context, items []T, batchSize int, pause time.Duration, run func(context.Context, int, T) error) error {
	if batchSize <= 0 {
		batchSize = 1
	}
	for idx, item := range items {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := run(ctx, idx, item); err != nil {
			return err
		}
		if err := waitBatchBoundary(ctx, idx+1, batchSize, pause); err != nil {
			return err
		}
	}
	return nil
}

func waitBatchBoundary(ctx context.Context, index, batchSize int, pause time.Duration) error {
	if pause <= 0 || index%batchSize != 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(pause):
		return nil
	}
}
