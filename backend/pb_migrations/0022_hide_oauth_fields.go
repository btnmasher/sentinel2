package pb_migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"sentinel2/internal/store"
)

var sensitiveOAuthFields = map[string][]string{
	store.CollectionUsers: {
		"oauth_access_token",
		"oauth_refresh_token",
		"oauth_access_expires_at",
		"oauth_refresh_expires_at",
	},
	store.CollectionCharacters: {
		"oauth_access_token",
		"oauth_refresh_token",
		"oauth_access_expires_at",
		"oauth_refresh_expires_at",
		"oauth_scopes",
	},
}

func init() {
	m.Register(
		func(app core.App) error { return setOAuthFieldsHidden(app, true) },
		func(app core.App) error { return setOAuthFieldsHidden(app, false) },
	)
}

func setOAuthFieldsHidden(app core.App, hidden bool) error {
	for collectionName, fieldNames := range sensitiveOAuthFields {
		collection, err := app.FindCollectionByNameOrId(collectionName)
		if err != nil {
			return err
		}

		for _, fieldName := range fieldNames {
			if err := setFieldHidden(collection, fieldName, hidden); err != nil {
				return err
			}
		}

		if err := app.Save(collection); err != nil {
			return err
		}
	}

	return nil
}

func setFieldHidden(collection *core.Collection, fieldName string, hidden bool) error {
	field := collection.Fields.GetByName(fieldName)
	if field == nil {
		return nil
	}

	switch typedField := field.(type) {
	case *core.TextField:
		typedField.Hidden = hidden
	case *core.DateField:
		typedField.Hidden = hidden
	default:
		return fmt.Errorf("field %q in collection %q has unsupported type %T", fieldName, collection.Name, field)
	}

	return nil
}
