package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
)

var ErrJobSkipped = errors.New("job skipped")

const DefaultTimeout = 10 * time.Minute
const NoTimeout time.Duration = -1

type RunOptions struct {
	JobOptions
	JobName  string
	JobID    string
	StepKind string
	Timeout  time.Duration
	Parent   context.Context
	JobFunc  func(context.Context) context.Context
	Unique   bool
}

type Runner struct {
	app         *pocketbase.PocketBase
	tracker     tracker
	record      *core.Record
	opts        RunOptions
	fields      logging.Fields
	message     string
	cancel      context.CancelFunc
	timeoutStop context.CancelFunc
	partialErr  error
	skipped     bool
	skipReason  string
	stepState   stepState
}

type stepState struct {
	started           int
	succeeded         int
	skipped           int
	nonCriticalFailed int
	criticalFailed    int
}

type tracker interface {
	IsRunning(kind string, step string) (bool, error)
	Start(jobID string, opts JobOptions) (*core.Record, error)
	Finish(record *core.Record, err error)
	FinishPartial(record *core.Record, err error)
	FinishSkipped(record *core.Record, reason string)
	FinishCanceled(record *core.Record, reason string)
}

type Stepper interface {
	Run(name string, critical bool, fn func(context.Context) error) error
	Partial(err error)
	SkipParent(reason string) error
	SkipStep(name string, reason string) error
	WithMessage(message string)
}

type runnerSteps struct {
	runner *Runner
	ctx    context.Context
}

func NewRunner(app *pocketbase.PocketBase, opts RunOptions) *Runner {
	if opts.JobID == "" {
		name := opts.JobName
		if name == "" {
			name = opts.Kind
		}
		opts.JobID = newJobID(name)
	}
	return &Runner{
		app:     app,
		tracker: NewJobTracker(app),
		opts:    opts,
		fields:  logging.Fields{},
	}
}

func (r *Runner) JobID() string {
	if r == nil {
		return ""
	}
	return r.opts.JobID
}

func (r *Runner) WithFields(fields logging.Fields) *Runner {
	if r == nil || len(fields) == 0 {
		return r
	}
	if r.fields == nil {
		r.fields = logging.Fields{}
	}
	for key, value := range fields {
		r.fields[key] = value
	}
	return r
}

func (r *Runner) WithMessage(message string) *Runner {
	if r == nil {
		return r
	}
	r.message = message
	return r
}

func (r *Runner) Run(fn func(ctx context.Context, step Stepper) error) error {
	ctx, cancel, timeoutCancel := r.buildContext()
	r.cancel = cancel
	r.timeoutStop = timeoutCancel

	RegisterCancel(r.opts.JobID, cancel)
	defer UnregisterCancel(r.opts.JobID)
	defer cancel()
	if timeoutCancel != nil {
		defer timeoutCancel()
	}

	if r.opts.Unique {
		running, err := r.tracker.IsRunning(r.opts.Kind, r.opts.Step)
		if err != nil {
			return err
		}
		if running {
			r.logger().
				WithFields(logging.Fields{
					"kind": r.opts.Kind,
					"step": r.opts.Step,
				}).
				Info("job already running; skipping")
			return nil
		}
	}

	record, err := r.tracker.Start(r.opts.JobID, r.opts.JobOptions)
	if err != nil {
		return err
	}
	r.record = record
	log := r.logger()
	startedAt := recordTime(record.Get("started_at"))
	log.Info(MessageJobStarted)

	runErr := fn(ctx, runnerSteps{runner: r, ctx: ctx})

	if r.skipped || errors.Is(runErr, ErrJobSkipped) {
		r.tracker.FinishSkipped(r.record, r.skipReason)
		r.logCompletion(log, startedAt, StatusSkipped, r.skipReason, false, false)
		return nil
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		r.tracker.FinishPartial(r.record, ctx.Err())
		r.logCompletion(log, startedAt, StatusTimeout, ctx.Err().Error(), true, true)
		return runErr
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		r.tracker.FinishCanceled(r.record, MessageCanceled)
		r.logCompletion(log, startedAt, StatusCanceled, MessageCanceled, false, false)
		return runErr
	}
	if runErr != nil {
		r.tracker.Finish(r.record, runErr)
		r.logCompletion(log, startedAt, StatusFailed, runErr.Error(), true, false)
		return runErr
	}
	if r.partialErr != nil {
		if r.shouldFinalizeAsPartial() {
			r.tracker.FinishPartial(r.record, r.partialErr)
			r.logCompletion(log, startedAt, StatusPartial, r.partialErr.Error(), false, false)
			return nil
		}
		r.tracker.Finish(r.record, r.partialErr)
		r.logCompletion(log, startedAt, StatusFailed, r.partialErr.Error(), true, false)
		return r.partialErr
	}
	r.tracker.Finish(r.record, nil)
	r.logCompletion(log, startedAt, StatusSuccess, "", false, false)
	return nil
}

func (r *Runner) buildContext() (context.Context, context.CancelFunc, context.CancelFunc) {
	parent := r.opts.Parent
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	if r.opts.JobFunc != nil {
		ctx = r.opts.JobFunc(ctx)
	}
	timeout := r.opts.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	if timeout > 0 {
		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeout)
		return timeoutCtx, cancel, timeoutCancel
	}
	return ctx, cancel, nil
}

func (r *Runner) stepKind() string {
	if r.opts.StepKind != "" {
		return r.opts.StepKind
	}
	return r.opts.Kind
}

func (r *Runner) logger() *logging.Logger {
	fields := logging.Fields{
		"job_id": r.opts.JobID,
		"kind":   r.opts.Kind,
	}
	if r.opts.Step != "" {
		fields["step"] = r.opts.Step
	}
	if r.opts.Trigger != "" {
		fields["trigger"] = r.opts.Trigger
	}
	if r.opts.ActorID != "" {
		fields["actor_id"] = r.opts.ActorID
	}
	for key, value := range r.fields {
		fields[key] = value
	}
	return logging.New(r.app).WithFields(fields)
}

func (r *Runner) stepLogger(step string) *logging.Logger {
	fields := logging.Fields{
		"job_id": r.opts.JobID,
		"kind":   r.stepKind(),
		"step":   step,
	}
	if r.opts.Trigger != "" {
		fields["trigger"] = r.opts.Trigger
	}
	if r.opts.ActorID != "" {
		fields["actor_id"] = r.opts.ActorID
	}
	for key, value := range r.fields {
		fields[key] = value
	}
	return logging.New(r.app).WithFields(fields)
}

func (r *Runner) logCompletion(log *logging.Logger, startedAt time.Time, status string, message string, isError bool, isTimeout bool) {
	if log == nil {
		return
	}
	fields := logging.Fields{
		"duration_ms": time.Since(startedAt).Milliseconds(),
		"status":      status,
	}
	finalMessage := message
	if finalMessage == "" {
		finalMessage = r.message
	}
	if finalMessage != "" {
		fields["message"] = finalMessage
	}
	entry := log.WithFields(fields)
	switch status {
	case StatusSuccess, StatusSkipped:
		entry.Info(MessageJobCompleted)
	case StatusPartial:
		entry.Warn(MessageJobCompletedWithErrors)
	case StatusFailed:
		entry.Error(MessageJobFailed)
	case StatusTimeout:
		entry.Error(MessageJobTimedOut)
	case StatusCanceled:
		if isTimeout {
			entry.Error(MessageJobTimedOut)
		} else {
			entry.Warn(MessageJobCompleted)
		}
	default:
		if isError {
			entry.Error(MessageJobFailed)
		} else {
			entry.Info(MessageJobCompleted)
		}
	}
}

func recordTime(value any) time.Time {
	if parsed, ok := value.(time.Time); ok {
		return parsed
	}
	return time.Now().UTC()
}

func (r *Runner) markPartial(err error) {
	if err == nil {
		return
	}
	if r.partialErr == nil {
		r.partialErr = err
	}
}

func (r *Runner) markSkipped(reason string) {
	if r.skipped {
		return
	}
	r.skipped = true
	r.skipReason = reason
}

func (r *Runner) noteStepStart() {
	if r == nil {
		return
	}
	r.stepState.started++
}

func (r *Runner) noteStepSuccess() {
	if r == nil {
		return
	}
	r.stepState.succeeded++
}

func (r *Runner) noteStepSkipped() {
	if r == nil {
		return
	}
	r.stepState.skipped++
}

func (r *Runner) noteStepFailure(critical bool) {
	if r == nil {
		return
	}
	if critical {
		r.stepState.criticalFailed++
		return
	}
	r.stepState.nonCriticalFailed++
}

func (r *Runner) shouldFinalizeAsPartial() bool {
	if r == nil || r.partialErr == nil {
		return false
	}
	// Partial is only valid when non-critical step failures occurred alongside
	// at least one successful step, with no critical step failures.
	return r.stepState.started > 0 &&
		r.stepState.nonCriticalFailed > 0 &&
		r.stepState.succeeded > 0 &&
		r.stepState.criticalFailed == 0
}

func (s runnerSteps) Run(name string, critical bool, fn func(context.Context) error) error {
	if s.runner == nil {
		return nil
	}
	s.runner.noteStepStart()
	stepRecord, err := s.runner.tracker.Start(
		s.runner.opts.JobID,
		JobOptions{
			Kind:    s.runner.stepKind(),
			Step:    name,
			Trigger: s.runner.opts.Trigger,
			ActorID: s.runner.opts.ActorID,
			Hidden:  s.runner.opts.Hidden,
		},
	)
	if err != nil {
		if critical {
			s.runner.noteStepFailure(true)
			return err
		}
		s.runner.noteStepFailure(false)
		s.runner.markPartial(err)
		return nil
	}

	stepLog := s.runner.stepLogger(name)
	stepStartedAt := recordTime(stepRecord.Get("started_at"))
	stepLog.Info(MessageStepStarted)

	ctx := s.ctx
	runErr := fn(ctx)
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			s.runner.tracker.FinishCanceled(stepRecord, MessageCanceled)
			s.runner.logStepCompletion(stepLog, stepStartedAt, StatusCanceled, MessageCanceled, false, false)
		} else {
			s.runner.tracker.Finish(stepRecord, err)
			isTimeout := errors.Is(err, context.DeadlineExceeded)
			status := StatusFailed
			if isTimeout {
				status = StatusTimeout
			}
			s.runner.logStepCompletion(stepLog, stepStartedAt, status, err.Error(), true, isTimeout)
		}
		if critical {
			s.runner.noteStepFailure(true)
			return err
		}
		s.runner.noteStepFailure(false)
		s.runner.markPartial(err)
		return nil
	}
	if runErr != nil {
		s.runner.tracker.Finish(stepRecord, runErr)
		status := StatusFailed
		levelError := true
		if !critical {
			status = StatusPartial
			levelError = false
		}
		s.runner.logStepCompletion(stepLog, stepStartedAt, status, runErr.Error(), levelError, false)
		if critical {
			s.runner.noteStepFailure(true)
			return runErr
		}
		s.runner.noteStepFailure(false)
		s.runner.markPartial(runErr)
		return nil
	}
	s.runner.tracker.Finish(stepRecord, nil)
	s.runner.logStepCompletion(stepLog, stepStartedAt, StatusSuccess, "", false, false)
	s.runner.noteStepSuccess()
	return nil
}

func (s runnerSteps) Partial(err error) {
	if s.runner == nil {
		return
	}
	s.runner.markPartial(err)
}

func (s runnerSteps) SkipParent(reason string) error {
	if s.runner == nil {
		return ErrJobSkipped
	}
	s.runner.markSkipped(reason)
	return ErrJobSkipped
}

func (s runnerSteps) WithMessage(message string) {
	if s.runner == nil {
		return
	}
	s.runner.WithMessage(message)
}

func (s runnerSteps) SkipStep(name string, reason string) error {
	if s.runner == nil {
		return nil
	}
	s.runner.noteStepStart()
	stepRecord, err := s.runner.tracker.Start(
		s.runner.opts.JobID,
		JobOptions{
			Kind:    s.runner.stepKind(),
			Step:    name,
			Trigger: s.runner.opts.Trigger,
			ActorID: s.runner.opts.ActorID,
			Hidden:  s.runner.opts.Hidden,
		},
	)
	if err != nil {
		s.runner.noteStepFailure(false)
		s.runner.markPartial(err)
		return nil
	}

	stepLog := s.runner.stepLogger(name)
	stepStartedAt := recordTime(stepRecord.Get("started_at"))
	stepLog.Info(MessageStepStarted)
	s.runner.tracker.FinishSkipped(stepRecord, reason)
	s.runner.logStepCompletion(stepLog, stepStartedAt, StatusSkipped, reason, false, false)
	s.runner.noteStepSkipped()
	return nil
}

func (r *Runner) logStepCompletion(log *logging.Logger, startedAt time.Time, status string, message string, isError bool, isTimeout bool) {
	if log == nil {
		return
	}
	fields := logging.Fields{
		"duration_ms": time.Since(startedAt).Milliseconds(),
		"status":      status,
	}
	finalMessage := message
	if finalMessage == "" {
		finalMessage = r.message
	}
	if finalMessage != "" {
		fields["message"] = finalMessage
	}
	entry := log.WithFields(fields)
	switch status {
	case StatusSuccess:
		entry.Info(MessageStepCompleted)
	case StatusPartial:
		entry.Warn(MessageStepCompletedWithErrors)
	case StatusFailed:
		entry.Error(MessageStepFailed)
	case StatusTimeout:
		entry.Error(MessageStepTimedOut)
	case StatusCanceled:
		if isTimeout {
			entry.Error(MessageStepTimedOut)
		} else {
			entry.Warn(MessageStepCompleted)
		}
	default:
		if isError {
			entry.Error(MessageStepFailed)
		} else {
			entry.Info(MessageStepCompleted)
		}
	}
}
