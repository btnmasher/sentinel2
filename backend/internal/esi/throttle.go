package esi

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"sentinel2/internal/logging"
)

const (
	defaultESIRate                   = 10
	defaultESIBurst                  = 20
	errorLimitRemainThreshold        = 5
	remainingLowThreshold            = 5
	remainingCriticalThreshold       = 1
	remainingLowDelay                = 2 * time.Second
	remainingCriticalDelay           = 5 * time.Second
	throttleWarnDelayMs        int64 = 5000
)

type esiThrottle struct {
	mu           sync.Mutex
	throttleTill time.Time
	logger       *logging.Logger
}

type esiRateLimiter struct {
	limiter *rate.Limiter
}

var globalESILimiter = newESIRateLimiter(defaultESIRate, defaultESIBurst)

func newESIRateLimiter(rps, burst int) *esiRateLimiter {
	if rps <= 0 {
		return &esiRateLimiter{limiter: nil}
	}
	if burst <= 0 {
		burst = rps
	}
	return &esiRateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

func (l *esiRateLimiter) wait(ctx context.Context) {
	if l == nil || l.limiter == nil {
		return
	}
	_ = l.limiter.Wait(ctx)
}

func newESIThrottle(logger *logging.Logger) *esiThrottle {
	return &esiThrottle{logger: logger}
}

func (t *esiThrottle) wait(ctx context.Context) {
	t.mu.Lock()
	until := t.throttleTill
	t.mu.Unlock()
	if until.IsZero() {
		return
	}
	delay := time.Until(until)
	if delay <= 0 {
		return
	}
	select {
	case <-ctx.Done():
		return
	case <-time.After(delay):
	}
}

func (t *esiThrottle) delay() time.Duration {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	until := t.throttleTill
	t.mu.Unlock()
	if until.IsZero() {
		return 0
	}
	delay := time.Until(until)
	if delay <= 0 {
		return 0
	}
	return delay
}

func (t *esiThrottle) update(resp *http.Response) {
	if resp == nil {
		return
	}

	now := time.Now()
	throttle := time.Time{}
	reasons := []string{}
	nextThrottle, reason := parseErrorLimitThrottle(resp, now)
	throttle, reasons = mergeThrottle(throttle, reasons, nextThrottle, reason)
	nextThrottle, reason = parseRetryAfterThrottle(resp, now)
	throttle, reasons = mergeThrottle(throttle, reasons, nextThrottle, reason)
	nextThrottle, reason = parseRemainingThrottle(resp, now)
	throttle, reasons = mergeThrottle(throttle, reasons, nextThrottle, reason)

	if throttle.IsZero() {
		return
	}

	t.mu.Lock()
	if !throttle.After(t.throttleTill) {
		t.mu.Unlock()
		return
	}
	t.throttleTill = throttle
	logger := t.logger
	t.mu.Unlock()
	logThrottleUpdate(logger, throttle, reasons)
}

func parseErrorLimitThrottle(resp *http.Response, now time.Time) (throttle time.Time, reason string) {
	if resp == nil {
		return time.Time{}, ""
	}
	remainHeader := resp.Header.Get("X-ESI-Error-Limit-Remain")
	resetHeader := resp.Header.Get("X-ESI-Error-Limit-Reset")
	if remainHeader == "" || resetHeader == "" {
		return time.Time{}, ""
	}
	remain, remainErr := strconv.Atoi(remainHeader)
	resetSeconds, resetErr := strconv.Atoi(resetHeader)
	if remainErr != nil || resetErr != nil || remain > errorLimitRemainThreshold || resetSeconds <= 0 {
		return time.Time{}, ""
	}
	return now.Add(time.Duration(resetSeconds) * time.Second), "error_limit"
}

func parseRetryAfterThrottle(resp *http.Response, now time.Time) (throttle time.Time, reason string) {
	if resp == nil || resp.StatusCode != http.StatusTooManyRequests {
		return time.Time{}, ""
	}
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return time.Time{}, ""
	}
	seconds, retryErr := strconv.Atoi(retryAfter)
	if retryErr != nil || seconds <= 0 {
		return time.Time{}, ""
	}
	return now.Add(time.Duration(seconds) * time.Second), "retry_after"
}

func parseRemainingThrottle(resp *http.Response, now time.Time) (throttle time.Time, reason string) {
	if resp == nil {
		return time.Time{}, ""
	}
	remaining := resp.Header.Get("X-Ratelimit-Remaining")
	if remaining == "" {
		return time.Time{}, ""
	}
	value, remainingErr := strconv.Atoi(remaining)
	if remainingErr != nil {
		return time.Time{}, ""
	}
	delay := time.Duration(0)
	switch {
	case value <= remainingCriticalThreshold:
		delay = remainingCriticalDelay
	case value <= remainingLowThreshold:
		delay = remainingLowDelay
	}
	if delay <= 0 {
		return time.Time{}, ""
	}
	return now.Add(delay), "ratelimit_remaining"
}

func mergeThrottle(current time.Time, reasons []string, candidate time.Time, reason string) (next time.Time, nextReasons []string) {
	if candidate.IsZero() {
		return current, reasons
	}
	if candidate.After(current) {
		current = candidate
	}
	if reason != "" {
		reasons = append(reasons, reason)
	}
	return current, reasons
}

func logThrottleUpdate(logger *logging.Logger, throttle time.Time, reasons []string) {
	if logger == nil {
		return
	}
	delayMs := time.Until(throttle).Milliseconds()
	fields := logging.Fields{
		"throttle_until": throttle.UTC().Format(time.RFC3339),
		"delay_ms":       delayMs,
	}
	if len(reasons) > 0 {
		fields["reason"] = reasons
	}
	log := logger.WithFields(fields)
	if delayMs >= throttleWarnDelayMs {
		log.Warn("esi throttle updated")
		return
	}
	log.Debug("esi throttle updated")
}
