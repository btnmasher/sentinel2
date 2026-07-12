package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"golang.org/x/oauth2"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

func (p *TestAuthProvider) persistUserSession(user *core.Record, sub string, token *oauth2.Token, accessLevel string, mainCharacter CharacterInfo, corpID, allianceID int) error {
	if user == nil {
		return fmt.Errorf("missing user")
	}
	if token == nil {
		return fmt.Errorf("missing token")
	}

	accessExpiry := tokenExpiry(token)
	refreshExpiry := refreshExpiry(token)
	user.Set("auth_provider", p.Name())
	user.Set("auth_provider_sub", sub)
	user.Set("sub", sub)
	user.Set("access_level", accessLevel)
	user.Set("oauth_access_token", token.AccessToken)
	accessExpiresAt, _ := types.ParseDateTime(time.Unix(accessExpiry, 0))
	user.Set("oauth_access_expires_at", accessExpiresAt)
	if token.RefreshToken != "" {
		user.Set("oauth_refresh_token", token.RefreshToken)
		refreshExpiresAt, _ := types.ParseDateTime(time.Unix(refreshExpiry, 0))
		user.Set("oauth_refresh_expires_at", refreshExpiresAt)
	}
	if mainCharacter.CharacterID > 0 {
		user.Set("eve_character_id", mainCharacter.CharacterID)
		user.Set("eve_character_name", mainCharacter.CharacterName)
		user.Set("eve_corporation_id", corpID)
		user.Set("eve_alliance_id", allianceID)
	} else {
		user.Set("eve_character_id", 0)
		user.Set("eve_character_name", "")
		user.Set("eve_corporation_id", 0)
		user.Set("eve_alliance_id", 0)
	}

	return p.App.Save(user)
}

func (p *TestAuthProvider) syncMainCharacter(ctx context.Context, user *core.Record, userInfo *UserInfo, corpID, allianceID int) error {
	if user == nil || userInfo == nil {
		return nil
	}

	mainCharacter, ok := SelectMainCharacter(userInfo)
	if !ok {
		return nil
	}

	coll, collErr := p.App.FindCollectionByNameOrId(store.CollectionCharacters)
	if collErr != nil {
		return collErr
	}

	now, _ := types.ParseDateTime(time.Now())
	if err := p.upsertLinkedCharacter(ctx, user, coll, mainCharacter, now, true, corpID, allianceID); err != nil {
		return err
	}

	records, recordsErr := p.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user} && auth_provider = {:provider}",
		"",
		0,
		0,
		dbx.Params{
			"user":     user.Id,
			"provider": p.Name(),
		},
	)
	if recordsErr != nil {
		return recordsErr
	}

	for _, record := range records {
		record.Set("is_main", int64(record.GetInt("eve_character_id")) == mainCharacter.CharacterID)
		if saveErr := p.App.Save(record); saveErr != nil {
			return saveErr
		}
	}

	return nil
}

func (p *TestAuthProvider) syncLinkedCharacters(ctx context.Context, user *core.Record, userInfo *UserInfo) error {
	if p == nil || p.App == nil || user == nil || userInfo == nil || len(userInfo.Characters) == 0 {
		return nil
	}

	linkedIDs, linkedErr := p.linkedCharacterIDs(user)
	if linkedErr != nil {
		return linkedErr
	}

	if len(linkedIDs) == 0 {
		return nil
	}

	currentByID := make(map[int64]CharacterInfo, len(userInfo.Characters))
	for _, character := range userInfo.Characters {
		currentByID[character.CharacterID] = character
	}

	coll, collErr := p.App.FindCollectionByNameOrId(store.CollectionCharacters)
	if collErr != nil {
		return collErr
	}

	now, _ := types.ParseDateTime(time.Now())
	records, recordsErr := p.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user} && auth_provider = {:provider}",
		"",
		0,
		0,
		dbx.Params{
			"user":     user.Id,
			"provider": p.Name(),
		},
	)
	if recordsErr != nil {
		return recordsErr
	}

	for _, record := range records {
		if syncErr := p.syncLinkedCharacterRecord(ctx, user, coll, record, currentByID, now); syncErr != nil {
			return syncErr
		}
	}

	return nil
}

func (p *TestAuthProvider) syncLinkedCharacterRecord(ctx context.Context, user *core.Record, coll *core.Collection, record *core.Record, currentByID map[int64]CharacterInfo, now types.DateTime) error {
	if record == nil || record.GetBool("is_main") {
		return nil
	}

	characterID := int64(record.GetInt("eve_character_id"))
	if characterID == 0 {
		return nil
	}

	characterInfo, ok := currentByID[characterID]
	if !ok {
		return p.App.Delete(record)
	}

	return p.upsertLinkedCharacter(ctx, user, coll, characterInfo, now, false, 0, 0)
}

func (p *TestAuthProvider) upsertLinkedCharacter(ctx context.Context, user *core.Record, coll *core.Collection, characterInfo CharacterInfo, now types.DateTime, isMain bool, corpID, allianceID int) error {
	records, recordsErr := p.App.FindRecordsByFilter(
		coll.Name,
		"auth_provider = {:provider} && eve_character_id = {:id}",
		"",
		1,
		0,
		dbx.Params{
			"provider": p.Name(),
			"id":       characterInfo.CharacterID,
		},
	)
	if recordsErr != nil {
		return recordsErr
	}

	record := core.NewRecord(coll)
	if len(records) > 0 {
		record = records[0]
		if owner := record.GetString("user"); owner != "" && owner != user.Id {
			return ErrCharacterAlreadyLinked
		}
	}

	corpID, allianceID = p.resolveLinkedCharacterAffiliation(ctx, user, record, characterInfo, corpID, allianceID)

	record.Set("user", user.Id)
	record.Set("auth_provider", p.Name())
	record.Set("eve_character_id", characterInfo.CharacterID)
	record.Set("eve_character_name", characterInfo.CharacterName)
	record.Set("eve_corporation_id", corpID)
	record.Set("eve_alliance_id", allianceID)
	record.Set("is_main", isMain)
	record.Set("esi_token_valid", characterInfo.HasValidToken)
	record.Set("esi_last_error", "")
	record.Set("esi_last_refresh_at", now)
	record.Set("oauth_access_token", "")
	record.Set("oauth_refresh_token", "")
	record.Set("oauth_scopes", "")
	if saveErr := p.App.Save(record); saveErr != nil {
		return saveErr
	}
	return nil
}

func (p *TestAuthProvider) resolveLinkedCharacterAffiliation(ctx context.Context, user, record *core.Record, characterInfo CharacterInfo, corpIDIn, allianceIDIn int) (corpID, allianceID int) {
	if corpIDIn != 0 || allianceIDIn != 0 || characterInfo.CharacterID <= 0 {
		return corpIDIn, allianceIDIn
	}

	fetchedCorpID, fetchedAllianceID, affiliationErr := p.resolveCharacterAffiliationByID(ctx, int(characterInfo.CharacterID))
	if affiliationErr == nil {
		return fetchedCorpID, fetchedAllianceID
	}

	if record != nil {
		corpID = record.GetInt("eve_corporation_id")
		allianceID = record.GetInt("eve_alliance_id")
	}

	p.logger.
		WithFields(logging.Fields{
			"user_id":      user.Id,
			"character_id": characterInfo.CharacterID,
		}).
		WithErr(affiliationErr).
		Warn("character affiliation lookup failed")

	return corpID, allianceID
}

func (p *TestAuthProvider) resolveCharacterAffiliationByID(ctx context.Context, characterID int) (corpID, allianceID int, err error) {
	if p == nil || p.PublicESI == nil {
		return 0, 0, fmt.Errorf("missing public esi client")
	}
	if characterID <= 0 {
		return 0, 0, fmt.Errorf("missing character id")
	}
	return p.PublicESI.CharacterAffiliation(ctx, characterID)
}
