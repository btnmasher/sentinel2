package jobs

import (
	"errors"
	"testing"
)

func TestShouldFinalizeAsPartial(t *testing.T) {
	errPartial := errors.New("partial")

	tests := []struct {
		name string
		r    Runner
		want bool
	}{
		{
			name: "no_partial_error",
			r: Runner{
				stepState: stepState{started: 2, succeeded: 1, nonCriticalFailed: 1},
			},
			want: false,
		},
		{
			name: "no_steps",
			r: Runner{
				partialErr: errPartial,
				stepState:  stepState{},
			},
			want: false,
		},
		{
			name: "noncritical_fail_without_success",
			r: Runner{
				partialErr: errPartial,
				stepState:  stepState{started: 1, nonCriticalFailed: 1},
			},
			want: false,
		},
		{
			name: "critical_failure_present",
			r: Runner{
				partialErr: errPartial,
				stepState:  stepState{started: 2, succeeded: 1, nonCriticalFailed: 1, criticalFailed: 1},
			},
			want: false,
		},
		{
			name: "only_skipped_steps",
			r: Runner{
				partialErr: errPartial,
				stepState:  stepState{started: 2, skipped: 2},
			},
			want: false,
		},
		{
			name: "valid_partial_noncritical_failed_plus_success",
			r: Runner{
				partialErr: errPartial,
				stepState:  stepState{started: 2, succeeded: 1, nonCriticalFailed: 1},
			},
			want: true,
		},
		{
			name: "manual_partial_without_failed_steps",
			r: Runner{
				partialErr: errPartial,
				stepState:  stepState{started: 1, succeeded: 1},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.r.shouldFinalizeAsPartial()
			if got != tc.want {
				t.Fatalf("shouldFinalizeAsPartial() = %v, want %v", got, tc.want)
			}
		})
	}
}
