package esi

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestShouldRetryPublicESI(t *testing.T) {
	if shouldRetryPublicESI(nil, nil) {
		t.Fatalf("nil error should not retry")
	}
	if !shouldRetryPublicESI(nil, errors.New("network")) {
		t.Fatalf("error without response should retry")
	}
	if !shouldRetryPublicESI(nil, timeoutErr{}) {
		t.Fatalf("timeout error should retry")
	}
	if !shouldRetryPublicESI(&http.Response{StatusCode: http.StatusTooManyRequests}, errors.New("rate limit")) {
		t.Fatalf("429 should retry")
	}
	if !shouldRetryPublicESI(&http.Response{StatusCode: http.StatusBadGateway}, errors.New("upstream")) {
		t.Fatalf("5xx should retry")
	}
	if shouldRetryPublicESI(&http.Response{StatusCode: http.StatusBadRequest}, errors.New("bad request")) {
		t.Fatalf("4xx should not retry")
	}
}

func TestExecuteWithPublicRetryRetriesThenSucceeds(t *testing.T) {
	client := &ESIPublicClient{
		throttle: newESIThrottle(nil),
	}
	attempts := 0

	//nolint:bodyclose // Callback contract is to close response bodies.
	resp, err := executeWithPublicRetry(
		client,
		context.Background(),
		publicRequestOptions{operation: "test request"},
		func(context.Context) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return &http.Response{StatusCode: http.StatusInternalServerError}, errors.New("temporary")
			}
			return &http.Response{StatusCode: http.StatusOK}, nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestExecuteWithPublicRetryReturnsNotFoundMapping(t *testing.T) {
	client := &ESIPublicClient{
		throttle: newESIThrottle(nil),
	}
	notFoundErr := errors.New("mapped not found")

	//nolint:bodyclose // Callback contract is to close response bodies.
	_, err := executeWithPublicRetry(
		client,
		context.Background(),
		publicRequestOptions{
			operation:   "test request",
			notFoundErr: notFoundErr,
		},
		func(context.Context) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusNotFound}, errors.New("missing")
		},
	)
	if !errors.Is(err, notFoundErr) {
		t.Fatalf("expected mapped notFound error, got %v", err)
	}
}

func TestExecuteWithPublicRetryDoesNotRetryNonRetryError(t *testing.T) {
	client := &ESIPublicClient{
		throttle: newESIThrottle(nil),
	}
	attempts := 0

	//nolint:bodyclose // Callback contract is to close response bodies.
	_, err := executeWithPublicRetry(
		client,
		context.Background(),
		publicRequestOptions{operation: "test request"},
		func(context.Context) (*http.Response, error) {
			attempts++
			return &http.Response{StatusCode: http.StatusOK}, newNonRetryPublicError("stop")
		},
	)
	if err == nil {
		t.Fatalf("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected single attempt, got %d", attempts)
	}
}

func TestNormalizeCharacterSearchInput(t *testing.T) {
	q, token, ok := normalizeCharacterSearchInput(1, "  tok  ", "  test  ")
	if !ok || q != "test" || token != "tok" {
		t.Fatalf("unexpected normalize result: q=%q token=%q ok=%v", q, token, ok)
	}

	_, _, ok = normalizeCharacterSearchInput(0, "tok", "q")
	if ok {
		t.Fatalf("expected invalid input for character id 0")
	}
}

func TestRetryBackoffIsBounded(t *testing.T) {
	backoff := publicRetryBackoff()

	for i := range 3 {
		next, stop := backoff.Next()
		if stop {
			t.Fatalf("unexpected stop at retry %d", i)
		}
		if next <= 0 || next > publicBackoffCap {
			t.Fatalf("unexpected backoff duration: %s", next)
		}
	}
}
