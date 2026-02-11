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

type esiThrottle struct {
	mu           sync.Mutex
	throttleTill time.Time
	logger       *logging.Logger
}

type esiRateLimiter struct {
	limiter *rate.Limiter
}

const (
	defaultESIRate  = 10
	defaultESIBurst = 20
)

var globalESILimiter = newESIRateLimiter(defaultESIRate, defaultESIBurst)

func newESIRateLimiter(rps int, burst int) *esiRateLimiter {
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
	var throttle time.Time
	reasons := []string{}

	remainHeader := resp.Header.Get("X-ESI-Error-Limit-Remain")
	resetHeader := resp.Header.Get("X-ESI-Error-Limit-Reset")
	if remainHeader != "" && resetHeader != "" {
		remain, remainErr := strconv.Atoi(remainHeader)
		resetSeconds, resetErr := strconv.Atoi(resetHeader)

		if remainErr == nil && resetErr == nil && remain <= 5 && resetSeconds > 0 {
			throttle = now.Add(time.Duration(resetSeconds) * time.Second)
			reasons = append(reasons, "error_limit")
		}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retry := resp.Header.Get("Retry-After")
		if retry != "" {
			seconds, retryErr := strconv.Atoi(retry)
			if retryErr == nil && seconds > 0 {
				retryAt := now.Add(time.Duration(seconds) * time.Second)
				if retryAt.After(throttle) {
					throttle = retryAt
				}
				reasons = append(reasons, "retry_after")
			}
		}
	}

	remaining := resp.Header.Get("X-Ratelimit-Remaining")
	if remaining != "" {
		value, remainingErr := strconv.Atoi(remaining)
		if remainingErr == nil {
			var delay time.Duration
			switch {
			case value <= 1:
				delay = 5 * time.Second
			case value <= 5:
				delay = 2 * time.Second
			}
			if delay > 0 {
				next := now.Add(delay)
				if next.After(throttle) {
					throttle = next
				}
				reasons = append(reasons, "ratelimit_remaining")
			}
		}
	}

	if throttle.IsZero() {
		return
	}

	t.mu.Lock()
	if throttle.After(t.throttleTill) {
		t.throttleTill = throttle
		if t.logger != nil {
			delayMs := time.Until(throttle).Milliseconds()
			fields := logging.Fields{
				"throttle_until": throttle.UTC().Format(time.RFC3339),
				"delay_ms":       delayMs,
			}
			if len(reasons) > 0 {
				fields["reason"] = reasons
			}
			log := t.logger.WithFields(fields)
			if delayMs >= 5000 {
				log.Warn("esi throttle updated")
			} else {
				log.Debug("esi throttle updated")
			}
		}
	}
	t.mu.Unlock()
}
