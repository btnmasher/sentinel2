package esi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"
)

func TestCharacterAffiliationNotModifiedUsesCachedAffiliation(t *testing.T) {
	t.Parallel()

	const (
		charID     = 42
		wantCorp   = 1001
		wantAlli   = 2002
		cacheETag  = "etag-1"
		serverBody = `{"birthday":"2020-01-01T00:00:00Z","bloodline_id":1,"corporation_id":1001,"gender":"male","name":"Pilot","race_id":1,"alliance_id":2002}`
	)

	var requests atomic.Int32
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/characters/42" {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Status:     "404 Not Found",
					Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Request:    r,
				}, nil
			}

			switch requests.Add(1) {
			case 1:
				// Seed cache with an expired response so the second call uses If-None-Match.
				firstHeaders := http.Header{}
				firstHeaders.Set("Content-Type", "application/json")
				firstHeaders.Set("ETag", cacheETag)
				firstHeaders.Set("Expires", time.Now().Add(-time.Minute).UTC().Format(http.TimeFormat))
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(serverBody)),
					Header:     firstHeaders,
					Request:    r,
				}, nil
			case 2:
				if got := r.Header.Get("If-None-Match"); got != cacheETag {
					t.Errorf("If-None-Match = %q, want %q", got, cacheETag)
				}
				notModifiedHeaders := http.Header{}
				notModifiedHeaders.Set("Content-Type", "application/json")
				notModifiedHeaders.Set("Expires", time.Now().Add(10*time.Minute).UTC().Format(http.TimeFormat))
				return &http.Response{
					StatusCode: http.StatusNotModified,
					Status:     "304 Not Modified",
					Body:       io.NopCloser(strings.NewReader(`{"error":"Not modified"}`)),
					Header:     notModifiedHeaders,
					Request:    r,
				}, nil
			default:
				t.Errorf("unexpected extra request %d", requests.Load())
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Status:     "500 Internal Server Error",
					Body:       io.NopCloser(strings.NewReader(`{"error":"unexpected request"}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Request:    r,
				}, nil
			}
		}),
	}

	client := NewESIDirectClient("test-agent", nil)
	client.public = goesi.NewESIClientWithOptions(httpClient, goesi.ClientOptions{
		UserAgent: "test-agent",
		BaseURL:   "https://esi.test",
	})

	corpID, allianceID, err := client.CharacterAffiliation(context.Background(), charID)
	if err != nil {
		t.Fatalf("first CharacterAffiliation() error = %v", err)
	}
	if corpID != wantCorp || allianceID != wantAlli {
		t.Fatalf("first CharacterAffiliation() = (%d, %d), want (%d, %d)", corpID, allianceID, wantCorp, wantAlli)
	}

	corpID, allianceID, err = client.CharacterAffiliation(context.Background(), charID)
	if err != nil {
		t.Fatalf("second CharacterAffiliation() error = %v", err)
	}
	if corpID != wantCorp || allianceID != wantAlli {
		t.Fatalf("second CharacterAffiliation() = (%d, %d), want (%d, %d)", corpID, allianceID, wantCorp, wantAlli)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("request count = %d, want 2", got)
	}
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
