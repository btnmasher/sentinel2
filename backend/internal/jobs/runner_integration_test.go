package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type finishKind string

const (
	finishSuccess  finishKind = "finish"
	finishPartial  finishKind = "partial"
	finishSkipped  finishKind = "skipped"
	finishCanceled finishKind = "canceled"
)

type finishCall struct {
	kind    finishKind
	step    string
	message string
}

type fakeTracker struct {
	startErrByStep map[string]error
	records        map[*core.Record]JobOptions
	finishes       []finishCall
}

func newFakeTracker() *fakeTracker {
	return &fakeTracker{
		startErrByStep: map[string]error{},
		records:        map[*core.Record]JobOptions{},
	}
}

func (f *fakeTracker) IsRunning(_, _ string) (bool, error) {
	return false, nil
}

func (f *fakeTracker) Start(_ string, opts JobOptions) (*core.Record, error) {
	if err := f.startErrByStep[opts.Step]; err != nil {
		return nil, err
	}
	rec := core.NewRecord(core.NewBaseCollection("job_runs"))
	rec.Set("started_at", time.Now().UTC())
	rec.Set("step", opts.Step)
	f.records[rec] = opts
	return rec, nil
}

func (f *fakeTracker) Finish(record *core.Record, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	f.finishes = append(f.finishes, finishCall{kind: finishSuccess, step: f.stepOf(record), message: msg})
}

func (f *fakeTracker) FinishPartial(record *core.Record, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	f.finishes = append(f.finishes, finishCall{kind: finishPartial, step: f.stepOf(record), message: msg})
}

func (f *fakeTracker) FinishSkipped(record *core.Record, reason string) {
	f.finishes = append(f.finishes, finishCall{kind: finishSkipped, step: f.stepOf(record), message: reason})
}

func (f *fakeTracker) FinishCanceled(record *core.Record, reason string) {
	f.finishes = append(f.finishes, finishCall{kind: finishCanceled, step: f.stepOf(record), message: reason})
}

func (f *fakeTracker) stepOf(record *core.Record) string {
	if record == nil {
		return ""
	}
	if opts, ok := f.records[record]; ok {
		return opts.Step
	}
	if step := record.GetString("step"); step != "" {
		return step
	}
	return ""
}

func (f *fakeTracker) parentFinish() (finishCall, bool) {
	for i := len(f.finishes) - 1; i >= 0; i-- {
		if f.finishes[i].step == "" {
			return f.finishes[i], true
		}
	}
	return finishCall{}, false
}

func newRunnerWithFakeTracker(ft *fakeTracker) *Runner {
	r := NewRunner(nil, &RunOptions{
		JobOptions: JobOptions{
			Kind: JobUploaderReleases,
		},
	})
	r.tracker = ft
	return r
}

func TestRunnerRun_NonCriticalSingleFailure_FinalizesFailed(t *testing.T) {
	ft := newFakeTracker()
	r := newRunnerWithFakeTracker(ft)

	err := r.Run(func(ctx context.Context, stepper Stepper) error {
		return stepper.Run("s1", false, func(context.Context) error {
			return errors.New("noncritical boom")
		})
	})
	if err == nil {
		t.Fatalf("Run() err = nil, want non-nil")
	}
	parent, ok := ft.parentFinish()
	if !ok {
		t.Fatalf("expected parent finish call")
	}
	if parent.kind != finishSuccess {
		t.Fatalf("parent finish kind = %q, want %q (Finish with error path)", parent.kind, finishSuccess)
	}
	if parent.message != "noncritical boom" {
		t.Fatalf("parent finish message = %q", parent.message)
	}
}

func TestRunnerRun_NonCriticalFailureWithSuccess_FinalizesPartial(t *testing.T) {
	ft := newFakeTracker()
	r := newRunnerWithFakeTracker(ft)

	err := r.Run(func(ctx context.Context, stepper Stepper) error {
		if runErr := stepper.Run("s1", false, func(context.Context) error { return nil }); runErr != nil {
			return runErr
		}
		return stepper.Run("s2", false, func(context.Context) error { return errors.New("noncritical boom") })
	})
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	parent, ok := ft.parentFinish()
	if !ok {
		t.Fatalf("expected parent finish call")
	}
	if parent.kind != finishPartial {
		t.Fatalf("parent finish kind = %q, want %q", parent.kind, finishPartial)
	}
}

func TestRunnerRun_CriticalFailure_FinalizesFailed(t *testing.T) {
	ft := newFakeTracker()
	r := newRunnerWithFakeTracker(ft)

	err := r.Run(func(ctx context.Context, stepper Stepper) error {
		return stepper.Run("s1", true, func(context.Context) error {
			return errors.New("critical boom")
		})
	})
	if err == nil {
		t.Fatalf("Run() err = nil, want non-nil")
	}
	parent, ok := ft.parentFinish()
	if !ok {
		t.Fatalf("expected parent finish call")
	}
	if parent.kind != finishSuccess {
		t.Fatalf("parent finish kind = %q, want %q (Finish with error path)", parent.kind, finishSuccess)
	}
	if parent.message != "critical boom" {
		t.Fatalf("parent finish message = %q", parent.message)
	}
}

func TestRunnerRun_SkippedOnlyStep_FinalizesSuccess(t *testing.T) {
	ft := newFakeTracker()
	r := newRunnerWithFakeTracker(ft)

	err := r.Run(func(ctx context.Context, stepper Stepper) error {
		return stepper.SkipStep("s1", "not needed")
	})
	if err != nil {
		t.Fatalf("Run() err = %v, want nil", err)
	}
	parent, ok := ft.parentFinish()
	if !ok {
		t.Fatalf("expected parent finish call")
	}
	if parent.kind != finishSuccess {
		t.Fatalf("parent finish kind = %q, want %q", parent.kind, finishSuccess)
	}
	if parent.message != "" {
		t.Fatalf("parent finish message = %q, want empty", parent.message)
	}
}

func TestRunnerRun_ManualPartialWithoutFailedSteps_FinalizesFailed(t *testing.T) {
	ft := newFakeTracker()
	r := newRunnerWithFakeTracker(ft)

	err := r.Run(func(ctx context.Context, stepper Stepper) error {
		if runErr := stepper.Run("s1", false, func(context.Context) error { return nil }); runErr != nil {
			return runErr
		}
		stepper.Partial(errors.New("manual partial"))
		return nil
	})
	if err == nil {
		t.Fatalf("Run() err = nil, want non-nil")
	}
	parent, ok := ft.parentFinish()
	if !ok {
		t.Fatalf("expected parent finish call")
	}
	if parent.kind != finishSuccess {
		t.Fatalf("parent finish kind = %q, want %q (Finish with error path)", parent.kind, finishSuccess)
	}
	if parent.message != "manual partial" {
		t.Fatalf("parent finish message = %q", parent.message)
	}
}

func TestRunnerRun_ManualPartialWithoutSteps_FinalizesFailed(t *testing.T) {
	ft := newFakeTracker()
	r := newRunnerWithFakeTracker(ft)

	err := r.Run(func(ctx context.Context, stepper Stepper) error {
		stepper.Partial(errors.New("manual partial"))
		return nil
	})
	if err == nil {
		t.Fatalf("Run() err = nil, want non-nil")
	}
	parent, ok := ft.parentFinish()
	if !ok {
		t.Fatalf("expected parent finish call")
	}
	if parent.kind != finishSuccess {
		t.Fatalf("parent finish kind = %q, want %q (Finish with error path)", parent.kind, finishSuccess)
	}
	if parent.message != "manual partial" {
		t.Fatalf("parent finish message = %q", parent.message)
	}
}

func TestRunnerRun_PanicRecovered_FinalizesFailed(t *testing.T) {
	ft := newFakeTracker()
	r := newRunnerWithFakeTracker(ft)

	err := r.Run(func(context.Context, Stepper) error {
		panic("kaboom")
	})
	if err == nil {
		t.Fatalf("Run() err = nil, want non-nil")
	}
	if got := err.Error(); got != "job panic recovered: kaboom" {
		t.Fatalf("Run() err = %q, want %q", got, "job panic recovered: kaboom")
	}
	parent, ok := ft.parentFinish()
	if !ok {
		t.Fatalf("expected parent finish call")
	}
	if parent.kind != finishSuccess {
		t.Fatalf("parent finish kind = %q, want %q (Finish with error path)", parent.kind, finishSuccess)
	}
	if parent.message != "job panic recovered: kaboom" {
		t.Fatalf("parent finish message = %q", parent.message)
	}
}
