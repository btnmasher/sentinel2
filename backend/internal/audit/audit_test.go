package audit

import (
	"reflect"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestNormalizeTargetFields_CharacterDefaults(t *testing.T) {
	character := core.NewRecord(core.NewBaseCollection("characters"))
	character.Set("eve_character_id", 90000001)
	character.Set("eve_character_name", "Pilot One")

	gotType, gotID, gotLabel, gotMeta := normalizeTargetFields(
		&Event{},
		character,
		"",
		"",
		"",
		"",
		nil,
	)

	if gotType != TargetTypeCharacter {
		t.Fatalf("targetType = %q, want %q", gotType, TargetTypeCharacter)
	}

	if gotID != character.Id {
		t.Fatalf("targetID = %q, want %q", gotID, character.Id)
	}

	if gotLabel != "Pilot One" {
		t.Fatalf("targetLabel = %q, want %q", gotLabel, "Pilot One")
	}
	wantMeta := map[string]any{"eve_character_id": 90000001}

	if !reflect.DeepEqual(gotMeta, wantMeta) {
		t.Fatalf("targetMeta = %#v, want %#v", gotMeta, wantMeta)
	}
}

func TestNormalizeTargetFields_UserDefaults(t *testing.T) {
	gotType, gotID, gotLabel, gotMeta := normalizeTargetFields(
		&Event{TargetUserID: "u_123"},
		nil,
		"Main Pilot",
		"",
		"",
		"",
		nil,
	)

	if gotType != TargetTypeUser {
		t.Fatalf("targetType = %q, want %q", gotType, TargetTypeUser)
	}

	if gotID != "u_123" {
		t.Fatalf("targetID = %q, want %q", gotID, "u_123")
	}

	if gotLabel != "Main Pilot" {
		t.Fatalf("targetLabel = %q, want %q", gotLabel, "Main Pilot")
	}

	if gotMeta != nil {
		t.Fatalf("targetMeta = %#v, want nil", gotMeta)
	}
}

func TestNormalizeTargetFields_ExplicitValuesPreserved(t *testing.T) {
	customMeta := map[string]any{"step": "all"}
	gotType, gotID, gotLabel, gotMeta := normalizeTargetFields(
		&Event{TargetUserID: "u_123"},
		nil,
		"Main Pilot",
		TargetTypeJob,
		"job_42",
		"cleanup",
		customMeta,
	)

	if gotType != TargetTypeJob {
		t.Fatalf("targetType = %q, want %q", gotType, TargetTypeJob)
	}

	if gotID != "job_42" {
		t.Fatalf("targetID = %q, want %q", gotID, "job_42")
	}

	if gotLabel != "cleanup" {
		t.Fatalf("targetLabel = %q, want %q", gotLabel, "cleanup")
	}

	if !reflect.DeepEqual(gotMeta, customMeta) {
		t.Fatalf("targetMeta = %#v, want %#v", gotMeta, customMeta)
	}
}

func TestNormalizeActorFields_FromActorRecord(t *testing.T) {
	actor := core.NewRecord(core.NewBaseCollection("users"))
	actor.Set("eve_character_name", "Admin Pilot")

	gotID, gotDisplay := normalizeActorFields("", "", actor)
	if gotID != actor.Id {
		t.Fatalf("actorID = %q, want %q", gotID, actor.Id)
	}

	if gotDisplay != "Admin Pilot" {
		t.Fatalf("actorDisplayName = %q, want %q", gotDisplay, "Admin Pilot")
	}
}

func TestNormalizeActorFields_FallbackDisplayToID(t *testing.T) {
	actor := core.NewRecord(core.NewBaseCollection("users"))

	gotID, gotDisplay := normalizeActorFields("", "", actor)
	if gotID != actor.Id {
		t.Fatalf("actorID = %q, want %q", gotID, actor.Id)
	}

	if gotDisplay != actor.Id {
		t.Fatalf("actorDisplayName = %q, want %q", gotDisplay, actor.Id)
	}
}

func TestNormalizeActorFields_ExplicitValuesWin(t *testing.T) {
	actor := core.NewRecord(core.NewBaseCollection("users"))
	actor.Set("eve_character_name", "Ignored Name")

	gotID, gotDisplay := normalizeActorFields("explicit-id", "explicit-name", actor)
	if gotID != "explicit-id" {
		t.Fatalf("actorID = %q, want %q", gotID, "explicit-id")
	}

	if gotDisplay != "explicit-name" {
		t.Fatalf("actorDisplayName = %q, want %q", gotDisplay, "explicit-name")
	}
}
