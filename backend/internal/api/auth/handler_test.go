package auth

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestNewAuthRecordResponseContainsOnlyBrowserSafeFields(t *testing.T) {
	collection := core.NewAuthCollection("users")
	collection.Fields.Add(
		&core.TextField{Name: "auth_provider"},
		&core.SelectField{Name: "access_level", Values: []string{"user"}, MaxSelect: 1},
		&core.TextField{Name: "oauth_access_token"},
		&core.TextField{Name: "oauth_refresh_token"},
	)
	record := core.NewRecord(collection)
	record.Set("auth_provider", "eve")
	record.Set("access_level", "user")
	record.Set("oauth_access_token", "access-secret")
	record.Set("oauth_refresh_token", "refresh-secret")

	response := newAuthRecordResponse(record)
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	serialized := string(encoded)
	if strings.Contains(serialized, "oauth_") || strings.Contains(serialized, "secret") {
		t.Fatalf("auth response contains sensitive fields: %s", serialized)
	}
	if response.AuthProvider != "eve" || response.AccessLevel != "user" {
		t.Fatalf("auth response lost safe identity fields: %+v", response)
	}
}
