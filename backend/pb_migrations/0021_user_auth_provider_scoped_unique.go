package pb_migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/store"
)

func init() {
	m.Register(migrateAuthProviderScopedUniqueForward, migrateAuthProviderScopedUniqueBackward)
}

func migrateAuthProviderScopedUniqueForward(app core.App) error {
	if err := migrateUserAuthProviderScopedUnique(app); err != nil {
		return err
	}

	return migrateCharacterAuthProviderScopedUnique(app)
}

func migrateAuthProviderScopedUniqueBackward(app core.App) error {
	if err := revertCharacterAuthProviderScopedUnique(app); err != nil {
		return err
	}

	return revertUserAuthProviderScopedUnique(app)
}

func migrateUserAuthProviderScopedUnique(app core.App) error {
	collection, err := app.FindCollectionByNameOrId(store.CollectionUsers)
	if err != nil {
		return err
	}

	collection.RemoveIndex("idx_users_auth_provider_sub")
	collection.AddIndex("idx_users_auth_provider_sub", true, "auth_provider,auth_provider_sub", "auth_provider != '' AND auth_provider_sub != ''")

	return app.Save(collection)
}

func revertUserAuthProviderScopedUnique(app core.App) error {
	collection, err := app.FindCollectionByNameOrId(store.CollectionUsers)
	if err != nil {
		return err
	}

	collection.RemoveIndex("idx_users_auth_provider_sub")
	collection.AddIndex("idx_users_auth_provider_sub", true, "auth_provider_sub", "auth_provider_sub != ''")

	return app.Save(collection)
}

func migrateCharacterAuthProviderScopedUnique(app core.App) error {
	if err := migrateCharacterAuthProvider(app); err != nil {
		return err
	}

	collection, err := app.FindCollectionByNameOrId(store.CollectionCharacters)
	if err != nil {
		return err
	}

	collection.RemoveIndex("idx_characters_eve_character_id")
	collection.AddIndex("idx_characters_auth_provider_eve_character_id", true, "auth_provider,eve_character_id", "auth_provider != '' AND eve_character_id != 0")

	return app.Save(collection)
}

func revertCharacterAuthProviderScopedUnique(app core.App) error {
	collection, err := app.FindCollectionByNameOrId(store.CollectionCharacters)
	if err != nil {
		return err
	}

	collection.RemoveIndex("idx_characters_auth_provider_eve_character_id")
	collection.AddIndex("idx_characters_eve_character_id", true, "eve_character_id", "")

	if collection.Fields.GetByName("auth_provider") != nil {
		collection.Fields.RemoveByName("auth_provider")
	}

	return app.Save(collection)
}

func migrateCharacterAuthProvider(app core.App) error {
	collection, err := app.FindCollectionByNameOrId(store.CollectionCharacters)
	if err != nil {
		return err
	}

	if collection.Fields.GetByName("auth_provider") == nil {
		collection.Fields.Add(&core.TextField{Name: "auth_provider"})
	}

	if err := app.Save(collection); err != nil {
		return err
	}

	records, err := app.FindRecordsByFilter(store.CollectionCharacters, "", "", 0, 0, nil)
	if err != nil {
		return err
	}

	for _, record := range records {
		record.Set("auth_provider", resolveCharacterAuthProvider(app, record))
		if saveErr := app.Save(record); saveErr != nil {
			return saveErr
		}
	}

	return nil
}

func resolveCharacterAuthProvider(app core.App, record *core.Record) string {
	provider := record.GetString("auth_provider")
	if provider != "" {
		return provider
	}

	userID := record.GetString("user")
	if userID == "" {
		return "eve"
	}

	user, err := app.FindRecordById(store.CollectionUsers, userID)
	if err != nil {
		return "eve"
	}

	userProvider := user.GetString("auth_provider")
	if userProvider != "" {
		return userProvider
	}

	return "eve"
}
