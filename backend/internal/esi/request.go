package esi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	retry "github.com/sethvargo/go-retry"
)

type publicRequestOptions struct {
	operation   string
	notFoundErr error
}

type nonRetryPublicError struct {
	message string
}

type publicAuthTransport struct {
	Base  http.RoundTripper
	Token string
}

func newNonRetryPublicError(message string) error {
	return &nonRetryPublicError{message: message}
}

func (e *nonRetryPublicError) Error() string {
	return e.message
}

func (a *publicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	if clone.Header.Get("Authorization") == "" {
		clone.Header.Set("Authorization", "Bearer "+a.Token)
	}
	return a.base().RoundTrip(clone)
}

func (a *publicAuthTransport) base() http.RoundTripper {
	if a.Base == nil {
		return http.DefaultTransport
	}
	return a.Base
}

func (c *ESIPublicClient) ensureConfigured() error {
	if c == nil || c.client == nil {
		return fmt.Errorf("esi public client not configured")
	}
	return nil
}

func publicRetryBackoff() retry.Backoff {
	backoff := retry.NewExponential(publicBackoffBase)
	backoff = retry.WithJitter(publicBackoffJitter, backoff)
	backoff = retry.WithCappedDuration(publicBackoffCap, backoff)
	backoff = retry.WithMaxRetries(publicBackoffMaxRetryCount, backoff)
	return backoff
}

func executeWithPublicRetry(
	c *ESIPublicClient,
	ctx context.Context,
	opts publicRequestOptions,
	call func(context.Context) (*http.Response, error),
) (*http.Response, error) {
	var resultHTTPResp *http.Response
	retryErr := retry.Do(ctx, publicRetryBackoff(), func(ctx context.Context) error {
		c.limiter.wait(ctx)
		c.throttle.wait(ctx)
		//nolint:bodyclose // The contract requires call() implementations to close response bodies.
		httpResp, respErr := call(ctx)
		if httpResp != nil {
			c.throttle.update(httpResp)
		}
		var nonRetryErr *nonRetryPublicError
		if errors.As(respErr, &nonRetryErr) {
			return nonRetryErr
		}
		if respErr != nil {
			if opts.notFoundErr != nil && httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
				return opts.notFoundErr
			}
			wrapped := fmt.Errorf("%s: %w", opts.operation, respErr)
			if shouldRetryPublicESI(httpResp, respErr) {
				return retry.RetryableError(wrapped)
			}
			return wrapped
		}
		resultHTTPResp = httpResp
		return nil
	})
	if retryErr != nil {
		return nil, retryErr
	}
	return resultHTTPResp, nil
}

func normalizeCharacterSearchInput(characterID int, accessToken, query string) (queryOut, tokenOut string, ok bool) {
	queryOut = strings.TrimSpace(query)
	tokenOut = strings.TrimSpace(accessToken)
	if characterID <= 0 || queryOut == "" || tokenOut == "" {
		return "", "", false
	}
	return queryOut, tokenOut, true
}

func normalizeCharacterStructureSearchInput(characterID int, accessToken, query string) (queryOut, tokenOut string, ok bool) {
	queryOut = query
	tokenOut = strings.TrimSpace(accessToken)
	if characterID <= 0 || strings.TrimSpace(queryOut) == "" || tokenOut == "" {
		return "", "", false
	}
	return queryOut, tokenOut, true
}

func shouldRetryPublicESI(resp *http.Response, err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	if resp == nil {
		return true
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return resp.StatusCode >= http.StatusInternalServerError
}

func closeResponseBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
