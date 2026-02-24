package timers

import (
	"context"
	"testing"
	"time"
)

func TestCampaignSeverityByStanding(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		standing string
		want     string
	}{
		{name: "ours is critical", standing: TimerStandingOurs, want: "critical"},
		{name: "friendly is high", standing: TimerStandingFriendly, want: "high"},
		{name: "neutral is medium", standing: TimerStandingNeutral, want: "medium"},
		{name: "complicated is medium", standing: TimerStandingComplicated, want: "medium"},
		{name: "hostile is medium", standing: TimerStandingHostile, want: "medium"},
		{name: "invalid defaults to medium", standing: "bogus", want: "medium"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := campaignSeverity(tc.standing)
			if got != tc.want {
				t.Fatalf("campaignSeverity(%q) = %q, want %q", tc.standing, got, tc.want)
			}
		})
	}
}

func TestCampaignProgressClampsRange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		score float64
		want  int
	}{
		{name: "below zero", score: -12.4, want: 0},
		{name: "within range", score: 42.2, want: 42},
		{name: "rounds up", score: 42.6, want: 43},
		{name: "above max", score: 133.9, want: 100},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stage, total := campaignProgress(tc.score)
			if stage != tc.want {
				t.Fatalf("campaignProgress(%f) stage = %d, want %d", tc.score, stage, tc.want)
			}
			if total != scoreTotalStages {
				t.Fatalf("campaignProgress(%f) total = %d, want %d", tc.score, total, scoreTotalStages)
			}
		})
	}
}

func TestNormalizeWatchlistStanding(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{in: TimerStandingOurs, want: TimerStandingOurs},
		{in: " Friendly ", want: TimerStandingFriendly},
		{in: "COMPLICATED", want: TimerStandingComplicated},
		{in: "invalid", want: TimerStandingHostile},
		{in: "", want: TimerStandingHostile},
	}

	for _, tc := range cases {
		got := normalizeWatchlistStanding(tc.in)
		if got != tc.want {
			t.Fatalf("normalizeWatchlistStanding(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCampaignNotesNoParticipants(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	got := svc.campaignNotes(context.Background(), nil, sovWatchlist{})
	if got != "" {
		t.Fatalf("campaignNotes(nil) = %q, want empty string", got)
	}
}

func TestRoundRobinSelection(t *testing.T) {
	t.Parallel()

	items := []notificationSource{
		{CharacterID: 10},
		{CharacterID: 11},
		{CharacterID: 12},
	}

	selected, next := roundRobinSelection(items, 1, 2)
	if len(selected) != 2 {
		t.Fatalf("selected len = %d, want 2", len(selected))
	}
	if selected[0].CharacterID != 11 || selected[1].CharacterID != 12 {
		t.Fatalf("unexpected selection order: %#v", selected)
	}
	if next != 0 {
		t.Fatalf("next cursor = %d, want 0", next)
	}

	selected, next = roundRobinSelection(items, -7, 2)
	if len(selected) != 2 {
		t.Fatalf("selected len with negative cursor = %d, want 2", len(selected))
	}
	if selected[0].CharacterID != 10 || selected[1].CharacterID != 11 {
		t.Fatalf("unexpected selection order with negative cursor: %#v", selected)
	}
	if next != 2 {
		t.Fatalf("next cursor with negative cursor = %d, want 2", next)
	}

	selected, next = roundRobinSelection(nil, 0, 2)
	if len(selected) != 0 || next != 0 {
		t.Fatalf("empty input should return empty selection and zero cursor, got len=%d next=%d", len(selected), next)
	}
}

func TestUnixToTimeSecondsAndMilliseconds(t *testing.T) {
	t.Parallel()

	if !unixToTime(0).IsZero() {
		t.Fatalf("unixToTime(0) should be zero time")
	}

	seconds := int64(1_700_000_123)
	gotSeconds := unixToTime(seconds)
	wantSeconds := time.Unix(seconds, 0).UTC()
	if !gotSeconds.Equal(wantSeconds) {
		t.Fatalf("unixToTime(seconds) = %s, want %s", gotSeconds, wantSeconds)
	}

	millis := int64(1_700_000_123_456)
	gotMillis := unixToTime(millis)
	wantMillis := time.UnixMilli(millis).UTC()
	if !gotMillis.Equal(wantMillis) {
		t.Fatalf("unixToTime(milliseconds) = %s, want %s", gotMillis, wantMillis)
	}
}

func TestNotificationKeyBuilders(t *testing.T) {
	t.Parallel()

	if got := notificationRRCursorKey(); got != notificationRRCursorKeyPrefix {
		t.Fatalf("notificationRRCursorKey() = %q, want %q", got, notificationRRCursorKeyPrefix)
	}
	if got := notificationETagKey(90000001); got != "skyhook_notification_etag:90000001" {
		t.Fatalf("notificationETagKey() = %q, want %q", got, "skyhook_notification_etag:90000001")
	}
}
