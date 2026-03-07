package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/config"
)

func TestRequireTimersWebhookToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		TimerSource:        config.TimerSourceWebhook,
		TimersWebhookToken: "secret-token",
	}
	middlewareFn := RequireTimersWebhookToken(cfg)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{name: "missing header", wantStatus: http.StatusUnauthorized},
		{name: "invalid header", authHeader: "Bearer wrong", wantStatus: http.StatusUnauthorized},
		{name: "valid bearer", authHeader: "Bearer secret-token", wantStatus: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := middlewareFn(newWebhookRequestEvent(tc.authHeader))
			assertMiddlewareResult(t, err, tc.wantStatus)
		})
	}
}

func newWebhookRequestEvent(authHeader string) *core.RequestEvent {
	req := httptest.NewRequest(http.MethodPut, "/api/webhooks/timers", http.NoBody)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	return &core.RequestEvent{Event: router.Event{Request: req}}
}

func assertMiddlewareResult(t *testing.T, err error, wantStatus int) {
	t.Helper()
	if wantStatus == 0 {
		if err != nil {
			t.Fatalf("RequireTimersWebhookToken() error = %v, want nil", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("RequireTimersWebhookToken() error = nil, want status %d", wantStatus)
	}
	apiErr := router.ToApiError(err)
	if apiErr.Status != wantStatus {
		t.Fatalf("status = %d, want %d", apiErr.Status, wantStatus)
	}
}
