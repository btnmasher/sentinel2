package timers

import (
	"errors"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/config"
	"sentinel2/internal/store"
	_ "sentinel2/pb_migrations"
)

const (
	testRegionID = 10000001
	testSystemID = 30000001
)

func TestServiceCreate(t *testing.T) {
	svc, app, _, user := newTimerTestService(t)
	expiresAt := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)

	record, err := svc.Create(&CreateInput{
		Title:         "Alpha timer",
		SystemID:      testSystemID,
		Standing:      TimerStandingHostile,
		TimerKind:     TimerKindCustom,
		StructureType: TimerStructureCustom,
		StageLabel:    TimerStageCustom,
		Severity:      TimerSeverityHigh,
		ExpiresAt:     expiresAt,
		Notes:         "manual note",
	}, user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got := record.GetString("title"); got != "Alpha timer" {
		t.Fatalf("title = %q, want %q", got, "Alpha timer")
	}
	if got := record.GetInt("system_id"); got != testSystemID {
		t.Fatalf("system_id = %d, want %d", got, testSystemID)
	}
	if got := record.GetString("system_name"); got != "Alpha" {
		t.Fatalf("system_name = %q, want %q", got, "Alpha")
	}
	if got := record.GetInt("region_id"); got != testRegionID {
		t.Fatalf("region_id = %d, want %d", got, testRegionID)
	}
	if got := record.GetString("region_name"); got != "Test Region" {
		t.Fatalf("region_name = %q, want %q", got, "Test Region")
	}
	if got := record.GetString("source"); got != "manual" {
		t.Fatalf("source = %q, want %q", got, "manual")
	}
	if got := record.GetString("status"); got != timerStatusActive {
		t.Fatalf("status = %q, want %q", got, timerStatusActive)
	}
	if got := record.GetString("created_by"); got != user.Id {
		t.Fatalf("created_by = %q, want %q", got, user.Id)
	}
	if got := record.GetString("created_by_name"); got != "Test Pilot" {
		t.Fatalf("created_by_name = %q, want %q", got, "Test Pilot")
	}
	if got := record.GetDateTime("expires_at").Time().UTC(); !got.Equal(expiresAt) {
		t.Fatalf("expires_at = %s, want %s", got.Format(time.RFC3339), expiresAt.Format(time.RFC3339))
	}

	saved, err := app.FindRecordById(store.CollectionTimers, record.Id)
	if err != nil {
		t.Fatalf("FindRecordById() error = %v", err)
	}
	if saved.Id != record.Id {
		t.Fatalf("saved id = %q, want %q", saved.Id, record.Id)
	}
}

func TestServiceUpdatePartial(t *testing.T) {
	svc, _, _, user := newTimerTestService(t)
	record := mustCreateTimer(t, svc, user, "Before update", time.Now().UTC().Add(90*time.Minute))

	title := "After update"
	notes := "updated notes"
	sourceRef := "ext-123"
	attackers := 42
	defenders := 58
	expiresAt := time.Now().UTC().Add(4 * time.Hour).Truncate(time.Second)

	updated, err := svc.Update(record.Id, &UpdateInput{
		Title:             &title,
		Notes:             &notes,
		SourceRef:         &sourceRef,
		AttackersScorePct: &attackers,
		DefenderScorePct:  &defenders,
		ExpiresAt:         &expiresAt,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if got := updated.GetString("title"); got != title {
		t.Fatalf("title = %q, want %q", got, title)
	}
	if got := updated.GetString("notes"); got != notes {
		t.Fatalf("notes = %q, want %q", got, notes)
	}
	if got := updated.GetString("source_ref"); got != sourceRef {
		t.Fatalf("source_ref = %q, want %q", got, sourceRef)
	}
	if got := updated.GetInt("attackers_score_pct"); got != attackers {
		t.Fatalf("attackers_score_pct = %d, want %d", got, attackers)
	}
	if got := updated.GetInt("defender_score_pct"); got != defenders {
		t.Fatalf("defender_score_pct = %d, want %d", got, defenders)
	}
	if got := updated.GetDateTime("expires_at").Time().UTC(); !got.Equal(expiresAt) {
		t.Fatalf("expires_at = %s, want %s", got.Format(time.RFC3339), expiresAt.Format(time.RFC3339))
	}
	if got := updated.GetString("source"); got != "manual" {
		t.Fatalf("source = %q, want %q", got, "manual")
	}
	if got := updated.GetString("system_name"); got != "Alpha" {
		t.Fatalf("system_name = %q, want %q", got, "Alpha")
	}
}

func TestServiceCancelAndUncancel(t *testing.T) {
	svc, _, _, user := newTimerTestService(t)
	record := mustCreateTimer(t, svc, user, "Cancelable timer", time.Now().UTC().Add(2*time.Hour))

	canceled, err := svc.Cancel(record.Id, user)
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	if got := canceled.GetString("status"); got != timerStatusCanceled {
		t.Fatalf("status after cancel = %q, want %q", got, timerStatusCanceled)
	}
	if got := canceled.GetString("canceled_by"); got != user.Id {
		t.Fatalf("canceled_by = %q, want %q", got, user.Id)
	}
	if canceled.GetDateTime("canceled_at").IsZero() {
		t.Fatal("canceled_at is zero, want populated timestamp")
	}

	uncanceled, err := svc.Uncancel(record.Id)
	if err != nil {
		t.Fatalf("Uncancel() error = %v", err)
	}

	if got := uncanceled.GetString("status"); got != timerStatusActive {
		t.Fatalf("status after uncancel = %q, want %q", got, timerStatusActive)
	}
	if got := uncanceled.GetString("canceled_by"); got != "" {
		t.Fatalf("canceled_by after uncancel = %q, want empty", got)
	}
	if !uncanceled.GetDateTime("canceled_at").IsZero() {
		t.Fatal("canceled_at after uncancel is set, want zero value")
	}
}

func TestServiceDelete(t *testing.T) {
	svc, app, _, user := newTimerTestService(t)
	record := mustCreateTimer(t, svc, user, "Delete me", time.Now().UTC().Add(2*time.Hour))

	deleted, err := svc.Delete(record.Id)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted.Id != record.Id {
		t.Fatalf("deleted id = %q, want %q", deleted.Id, record.Id)
	}

	if _, err := app.FindRecordById(store.CollectionTimers, record.Id); err == nil {
		t.Fatal("FindRecordById() error = nil, want missing record error")
	}
}

func TestServiceListRespectsStatusAndRecentCanceledCutoff(t *testing.T) {
	svc, app, _, user := newTimerTestService(t)
	active := mustCreateTimer(t, svc, user, "Active timer", time.Now().UTC().Add(3*time.Hour))
	recentCanceled := mustCreateTimer(t, svc, user, "Recent canceled timer", time.Now().UTC().Add(4*time.Hour))
	oldCanceled := mustCreateTimer(t, svc, user, "Old canceled timer", time.Now().UTC().Add(5*time.Hour))

	setTimerState(t, app, recentCanceled, timerStatusCanceled, time.Now().UTC().Add(-2*time.Hour))
	setTimerState(t, app, oldCanceled, timerStatusCanceled, time.Now().UTC().Add(-48*time.Hour))

	activeOnly, err := svc.List(ListInput{Limit: 10})
	if err != nil {
		t.Fatalf("List(active only) error = %v", err)
	}
	if len(activeOnly) != 1 || activeOnly[0].Id != active.Id {
		t.Fatalf("active-only list ids = %v, want only %q", recordIDs(activeOnly), active.Id)
	}

	allVisible, err := svc.List(ListInput{
		Statuses: []string{timerStatusActive, timerStatusCanceled},
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("List(with canceled) error = %v", err)
	}

	got := recordTitles(allVisible)
	if len(got) != 2 {
		t.Fatalf("visible timer count = %d, want 2 (%v)", len(got), got)
	}
	if _, ok := got["Active timer"]; !ok {
		t.Fatalf("visible timers missing active timer: %v", got)
	}
	if _, ok := got["Recent canceled timer"]; !ok {
		t.Fatalf("visible timers missing recent canceled timer: %v", got)
	}
	if _, ok := got["Old canceled timer"]; ok {
		t.Fatalf("visible timers unexpectedly include old canceled timer: %v", got)
	}
}

func TestServiceCreateWebhookRequiresWebhookID(t *testing.T) {
	svc, _, _, _ := newTimerTestService(t)

	_, err := svc.CreateWebhook(&CreateInput{
		SystemID:      testSystemID,
		TimerKind:     TimerKindCustom,
		StructureType: TimerStructureCustom,
		StageLabel:    TimerStageCustom,
		ExpiresAt:     time.Now().UTC().Add(time.Hour),
	})
	if !errors.Is(err, ErrMissingWebhookID) {
		t.Fatalf("CreateWebhook() error = %v, want %v", err, ErrMissingWebhookID)
	}
}

func TestServiceWebhookLifecycle(t *testing.T) {
	svc, _, _, _ := newTimerTestService(t)
	expiresAt := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)

	record, err := svc.CreateWebhook(&CreateInput{
		WebhookID:     "webhook-123",
		Title:         "Webhook timer",
		SystemID:      testSystemID,
		TimerKind:     TimerKindCustom,
		StructureType: TimerStructureCustom,
		StageLabel:    TimerStageCustom,
		ExpiresAt:     expiresAt,
	})
	if err != nil {
		t.Fatalf("CreateWebhook() error = %v", err)
	}

	if got := record.GetString("source"); got != config.TimerSourceWebhook {
		t.Fatalf("source = %q, want %q", got, config.TimerSourceWebhook)
	}
	if got := record.GetString("webhook_id"); got != "webhook-123" {
		t.Fatalf("webhook_id = %q, want %q", got, "webhook-123")
	}
	if got := record.GetString("created_by_name"); got != systemCreatorName {
		t.Fatalf("created_by_name = %q, want %q", got, systemCreatorName)
	}

	found, err := svc.FindByWebhookID("webhook-123")
	if err != nil {
		t.Fatalf("FindByWebhookID() error = %v", err)
	}
	if found.Id != record.Id {
		t.Fatalf("FindByWebhookID() id = %q, want %q", found.Id, record.Id)
	}

	if err := svc.DeleteByWebhookID("webhook-123"); err != nil {
		t.Fatalf("DeleteByWebhookID() error = %v", err)
	}
	if err := svc.DeleteByWebhookID("webhook-123"); err != nil {
		t.Fatalf("DeleteByWebhookID() second call error = %v", err)
	}
	if _, err := svc.FindByWebhookID("webhook-123"); !errors.Is(err, ErrTimerNotFound) {
		t.Fatalf("FindByWebhookID() after delete error = %v, want %v", err, ErrTimerNotFound)
	}
}

func newTimerTestService(t *testing.T) (*Service, *pocketbase.PocketBase, *core.Record, *core.Record) {
	t.Helper()

	dataDir := t.TempDir()
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:       dataDir,
		DefaultEncryptionEnv: "pb_test_env",
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("RunAllMigrations() error = %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Fatalf("ResetBootstrapState() error = %v", err)
		}
	})

	region := mustCreateRegion(t, app, testRegionID, "Test Region")
	mustCreateSystem(t, app, testSystemID, "Alpha", region.GetInt("eve_id"), region.GetString("name"))
	user := mustCreateUser(t, app, "staff@example.com", "Test Pilot")

	return NewService(app, nil, nil), app, region, user
}

func mustCreateRegion(t *testing.T, app *pocketbase.PocketBase, eveID int, name string) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(store.CollectionRegions)
	if err != nil {
		t.Fatalf("FindCollectionByNameOrId(%q) error = %v", store.CollectionRegions, err)
	}
	record := core.NewRecord(collection)
	record.Set("eve_id", eveID)
	record.Set("name", name)
	mustSaveRecord(t, app, record)
	return record
}

func mustCreateSystem(t *testing.T, app *pocketbase.PocketBase, eveID int, name string, regionID int, regionName string) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(store.CollectionSolarSystems)
	if err != nil {
		t.Fatalf("FindCollectionByNameOrId(%q) error = %v", store.CollectionSolarSystems, err)
	}
	record := core.NewRecord(collection)
	record.Set("eve_id", eveID)
	record.Set("name", name)
	record.Set("constellation", 20000001)
	record.Set("region_id", regionID)
	record.Set("region_name", regionName)
	mustSaveRecord(t, app, record)
	return record
}

func mustCreateUser(t *testing.T, app *pocketbase.PocketBase, email, characterName string) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(store.CollectionUsers)
	if err != nil {
		t.Fatalf("FindCollectionByNameOrId(%q) error = %v", store.CollectionUsers, err)
	}
	record := core.NewRecord(collection)
	record.SetEmail(email)
	record.SetPassword("password123")
	record.SetVerified(true)
	record.Set("created_at", time.Now().UTC().Format(time.RFC3339))
	record.Set("access_level", "staff")
	record.Set("eve_character_name", characterName)
	mustSaveRecord(t, app, record)
	return record
}

func mustCreateTimer(t *testing.T, svc *Service, user *core.Record, title string, expiresAt time.Time) *core.Record {
	t.Helper()

	record, err := svc.Create(&CreateInput{
		Title:         title,
		SystemID:      testSystemID,
		Standing:      TimerStandingHostile,
		TimerKind:     TimerKindCustom,
		StructureType: TimerStructureCustom,
		StageLabel:    TimerStageCustom,
		Severity:      TimerSeverityMedium,
		ExpiresAt:     expiresAt.UTC().Truncate(time.Second),
	}, user)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", title, err)
	}
	return record
}

func mustSaveRecord(t *testing.T, app *pocketbase.PocketBase, record *core.Record) {
	t.Helper()

	if err := app.Save(record); err != nil {
		t.Fatalf("Save(%s) error = %v", record.Collection().Name, err)
	}
}

func setTimerState(t *testing.T, app *pocketbase.PocketBase, record *core.Record, status string, expiresAt time.Time) {
	t.Helper()

	record.Set("status", status)
	record.Set("expires_at", expiresAt.UTC().Format(time.RFC3339))
	if status == timerStatusCanceled {
		record.Set("canceled_at", time.Now().UTC().Format(time.RFC3339))
	}
	mustSaveRecord(t, app, record)
}

func recordIDs(records []*core.Record) []string {
	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, record.Id)
	}
	return out
}

func recordTitles(records []*core.Record) map[string]struct{} {
	out := make(map[string]struct{}, len(records))
	for _, record := range records {
		out[record.GetString("title")] = struct{}{}
	}
	return out
}
