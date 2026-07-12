package auth

import (
	"context"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/oauth2"

	"sentinel2/internal/store"
	_ "sentinel2/pb_migrations"
)

func TestSyncMainCharacterDemotesPreviousMain(t *testing.T) {
	t.Parallel()

	app := newTestAuthSyncTestApp(t)
	user := mustCreateTestAuthSyncUser(t, app)

	mustCreateTestAuthSyncCharacter(t, app, user.Id, 1001, "Old Main", true)
	mustCreateTestAuthSyncCharacter(t, app, user.Id, 1002, "Alt One", false)

	newMainID := int64(1003)
	provider := &TestAuthProvider{App: app}
	info := &UserInfo{
		MainCharacterID: &newMainID,
		Characters: []CharacterInfo{
			{CharacterID: 1001, CharacterName: "Old Main"},
			{CharacterID: 1002, CharacterName: "Alt One"},
			{CharacterID: 1003, CharacterName: "New Main", IsPrimary: true},
		},
	}

	if err := provider.syncMainCharacter(context.Background(), user, info, 9001, 8002); err != nil {
		t.Fatalf("syncMainCharacter() error = %v", err)
	}

	oldMain := mustFindTestAuthSyncCharacter(t, app, 1001)
	if oldMain.GetBool("is_main") {
		t.Fatal("old main remained marked as main")
	}

	alt := mustFindTestAuthSyncCharacter(t, app, 1002)
	if alt.GetBool("is_main") {
		t.Fatal("alt character became main")
	}

	newMain := mustFindTestAuthSyncCharacter(t, app, 1003)
	if !newMain.GetBool("is_main") {
		t.Fatal("new main was not marked as main")
	}
	if got := newMain.GetString("eve_character_name"); got != "New Main" {
		t.Fatalf("new main name = %q, want %q", got, "New Main")
	}
	if got := newMain.GetInt("eve_corporation_id"); got != 9001 {
		t.Fatalf("new main corp_id = %d, want %d", got, 9001)
	}
	if got := newMain.GetInt("eve_alliance_id"); got != 8002 {
		t.Fatalf("new main alliance_id = %d, want %d", got, 8002)
	}
}

func TestPersistUserSessionPopulatesMainCharacterFields(t *testing.T) {
	t.Parallel()

	app := newTestAuthSyncTestApp(t)
	user := mustCreateTestAuthSyncUser(t, app)
	provider := &TestAuthProvider{App: app}
	token := &oauth2.Token{AccessToken: "access-token", RefreshToken: "refresh-token"}
	mainCharacter := CharacterInfo{
		CharacterID:   1402766339,
		CharacterName: "Gothicus",
	}

	if err := provider.persistUserSession(user, "account-sub", token, "admin", mainCharacter, 9001, 8002); err != nil {
		t.Fatalf("persistUserSession() error = %v", err)
	}

	saved, err := app.FindRecordById(store.CollectionUsers, user.Id)
	if err != nil {
		t.Fatalf("FindRecordById() error = %v", err)
	}

	if got := saved.GetString("eve_character_name"); got != "Gothicus" {
		t.Fatalf("eve_character_name = %q, want %q", got, "Gothicus")
	}
	if got := saved.GetInt("eve_character_id"); got != 1402766339 {
		t.Fatalf("eve_character_id = %d, want %d", got, 1402766339)
	}
	if got := saved.GetInt("eve_corporation_id"); got != 9001 {
		t.Fatalf("eve_corporation_id = %d, want %d", got, 9001)
	}
	if got := saved.GetInt("eve_alliance_id"); got != 8002 {
		t.Fatalf("eve_alliance_id = %d, want %d", got, 8002)
	}
}

func newTestAuthSyncTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:       t.TempDir(),
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
	return app
}

func mustCreateTestAuthSyncUser(t *testing.T, app *pocketbase.PocketBase) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(store.CollectionUsers)
	if err != nil {
		t.Fatalf("FindCollectionByNameOrId(%q) error = %v", store.CollectionUsers, err)
	}

	record := core.NewRecord(collection)
	record.SetEmail("sync-test@example.com")
	record.SetPassword("password123")
	record.SetVerified(true)
	record.Set("created_at", time.Now().UTC().Format(time.RFC3339))
	record.Set("access_level", "admin")
	record.Set("eve_character_name", "Sync Test")

	if err := app.Save(record); err != nil {
		t.Fatalf("Save(user) error = %v", err)
	}
	return record
}

func mustCreateTestAuthSyncCharacter(t *testing.T, app *pocketbase.PocketBase, userID string, eveCharacterID int, name string, isMain bool) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(store.CollectionCharacters)
	if err != nil {
		t.Fatalf("FindCollectionByNameOrId(%q) error = %v", store.CollectionCharacters, err)
	}

	record := core.NewRecord(collection)
	record.Set("user", userID)
	record.Set("auth_provider", AuthProviderTestAuth)
	record.Set("eve_character_id", eveCharacterID)
	record.Set("eve_character_name", name)
	record.Set("is_main", isMain)
	record.Set("esi_token_valid", true)

	if err := app.Save(record); err != nil {
		t.Fatalf("Save(character) error = %v", err)
	}
}

func mustFindTestAuthSyncCharacter(t *testing.T, app *pocketbase.PocketBase, eveCharacterID int) *core.Record {
	t.Helper()

	records, err := app.FindRecordsByFilter(
		store.CollectionCharacters,
		"eve_character_id = {:id} && auth_provider = {:provider}",
		"",
		1,
		0,
		map[string]any{
			"id":       eveCharacterID,
			"provider": AuthProviderTestAuth,
		},
	)
	if err != nil {
		t.Fatalf("FindRecordsByFilter(%q) error = %v", store.CollectionCharacters, err)
	}
	if len(records) == 0 {
		t.Fatalf("character %d not found", eveCharacterID)
	}
	return records[0]
}
