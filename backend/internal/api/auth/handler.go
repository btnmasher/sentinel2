package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/audit"
	"sentinel2/internal/auth"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"

	"github.com/pocketbase/pocketbase/tools/router"
)

type AuthHandler struct {
	Auth  *auth.Manager
	Audit *audit.Service
}

func NewAuthHandler(manager *auth.Manager, auditSvc *audit.Service) *AuthHandler {
	return &AuthHandler{Auth: manager, Audit: auditSvc}
}

func (h *AuthHandler) Authenticate(c *core.RequestEvent) error {
	if err := h.Auth.Authenticate(c, auth.AuthFlow{Type: auth.FlowLogin}); err != nil {
		logging.WithRequest(h.Auth.App, c).
			WithErr(err).
			Debug("auth login failed")
		return err
	}
	return nil
}

func (h *AuthHandler) Callback(c *core.RequestEvent) error {
	result, callbackErr := h.Auth.Callback(c)
	if callbackErr != nil {
		logging.WithRequest(h.Auth.App, c).
			WithErr(callbackErr).
			Debug("auth callback failed")
		return callbackErr
	}

	redirectBaseURL := strings.TrimRight(strings.TrimSpace(result.RedirectBaseURL), "/")
	if redirectBaseURL == "" {
		redirectBaseURL = auth.RequestBaseURL(c)
	}

	if result.IsLink {
		linkedCharacter := findCharacterByUserAndProviderAndEVEID(h.Auth.App, result.UserID, h.Auth.Provider.Name(), result.CharacterID)
		linkSummary := "Linked character"
		if result.CharacterName != "" {
			linkSummary = fmt.Sprintf("Linked character %s", result.CharacterName)
		}
		if h.Audit != nil {
			h.Audit.LogEvent(&audit.Event{
				Action:                 audit.ActionUserAuthLinkCharacter,
				Summary:                linkSummary,
				TargetUserID:           result.UserID,
				TargetCharacter:        linkedCharacter,
				TargetType:             audit.TargetTypeCharacter,
				TargetLabel:            result.CharacterName,
				ResolveTargetCharacter: linkedCharacter == nil,
				TargetMeta: map[string]any{
					"eve_character_id": result.CharacterID,
				},
				ActorID:          result.UserID,
				ActorDisplayName: result.CharacterName,
			})
		}
		logging.WithRequest(h.Auth.App, c).
			Info("auth link completed")
		return c.Redirect(http.StatusFound, redirectBaseURL+"/profile?linked=1")
	}
	logging.WithRequest(h.Auth.App, c).
		Info("auth login completed")
	return c.Redirect(http.StatusFound, redirectBaseURL+"/auth/complete?code="+result.ExchangeCode)
}

func (h *AuthHandler) Logout(c *core.RequestEvent) error {
	if err := h.Auth.Logout(c); err != nil {
		logging.WithRequest(h.Auth.App, c).
			WithErr(err).
			Debug("auth logout failed")
		return err
	}
	return nil
}

func (h *AuthHandler) Link(c *core.RequestEvent) error {
	if h.Auth.Provider.Name() != auth.AuthProviderEVE {
		return router.NewNotFoundError("Not found", nil)
	}
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		logging.WithRequest(h.Auth.App, c).
			WithErr(userErr).
			Debug("auth link unauthorized")
		return userErr
	}
	authURL, err := h.Auth.BuildAuthURL(
		c,
		auth.AuthFlow{Type: auth.FlowLink, LinkUserID: user.Id},
	)
	if err != nil {
		logging.WithRequest(h.Auth.App, c).
			WithErr(err).
			Debug("auth link failed")
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"url": authURL})
}

type linkableCharacterProfile struct {
	CharacterID   int    `json:"character_id"`
	Name          string `json:"name"`
	IsPrimary     bool   `json:"is_primary"`
	HasValidToken bool   `json:"has_valid_token"`
}

type linkableCharactersResponse struct {
	Characters []linkableCharacterProfile `json:"characters"`
}

func (h *AuthHandler) LinkableCharacters(c *core.RequestEvent) error {
	record, recordErr := auth.CurrentUser(c)
	if recordErr != nil {
		return recordErr
	}

	testAuthProvider, ok := h.Auth.Provider.(*auth.TestAuthProvider)
	if !ok || record.GetString("auth_provider") != auth.AuthProviderTestAuth {
		return router.NewNotFoundError("Not found", nil)
	}

	characters, charErr := testAuthProvider.LinkableCharacters(c.Request.Context(), record)
	if charErr != nil {
		return charErr
	}

	response := linkableCharactersResponse{
		Characters: make([]linkableCharacterProfile, 0, len(characters)),
	}
	for _, character := range characters {
		response.Characters = append(response.Characters, linkableCharacterProfile{
			CharacterID:   int(character.CharacterID),
			Name:          character.CharacterName,
			IsPrimary:     character.IsPrimary,
			HasValidToken: character.HasValidToken,
		})
	}

	return c.JSON(http.StatusOK, response)
}

type linkCharactersRequest struct {
	CharacterIDs []int `json:"character_ids"`
}

func (h *AuthHandler) LinkCharacters(c *core.RequestEvent) error {
	record, recordErr := auth.CurrentUser(c)
	if recordErr != nil {
		return recordErr
	}

	testAuthProvider, ok := h.Auth.Provider.(*auth.TestAuthProvider)
	if !ok || record.GetString("auth_provider") != auth.AuthProviderTestAuth {
		return router.NewNotFoundError("Not found", nil)
	}

	var payload linkCharactersRequest
	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Invalid request body.", logging.Fields{
			"error": bindErr.Error(),
		})
	}
	if len(payload.CharacterIDs) == 0 {
		return router.NewBadRequestError("Missing character ids.", logging.Fields{
			"required_field": "character_ids",
		})
	}

	characters, linkErr := testAuthProvider.LinkCharacters(c.Request.Context(), record, payload.CharacterIDs)
	if linkErr != nil {
		return linkErr
	}

	if h.Audit != nil {
		for _, character := range characters {
			linkedCharacter := findCharacterByUserAndProviderAndEVEID(h.Auth.App, record.Id, auth.AuthProviderTestAuth, int(character.CharacterID))
			linkSummary := "Linked character"
			if character.CharacterName != "" {
				linkSummary = fmt.Sprintf("Linked character %s", character.CharacterName)
			}
			h.Audit.LogEvent(&audit.Event{
				Action:                 audit.ActionUserAuthLinkCharacter,
				Summary:                linkSummary,
				TargetUserID:           record.Id,
				TargetCharacter:        linkedCharacter,
				TargetType:             audit.TargetTypeCharacter,
				TargetLabel:            character.CharacterName,
				ResolveTargetCharacter: linkedCharacter == nil,
				TargetMeta: map[string]any{
					"eve_character_id": character.CharacterID,
				},
				ActorID:          record.Id,
				ActorDisplayName: character.CharacterName,
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"ok":     true,
		"linked": len(characters),
	})
}

type authMeResponse struct {
	UserID      string `json:"user_id"`
	AccessLevel string `json:"access_level"`
	Provider    string `json:"auth_provider"`
}

func (h *AuthHandler) Me(c *core.RequestEvent) error {
	record, recordErr := auth.CurrentUser(c)
	if recordErr != nil {
		return recordErr
	}

	return c.JSON(http.StatusOK, authMeResponse{
		UserID:      record.Id,
		AccessLevel: record.GetString("access_level"),
		Provider:    record.GetString("auth_provider"),
	})
}

type characterProfile struct {
	RecordID         string `json:"record_id"`
	CharacterID      int    `json:"character_id"`
	Name             string `json:"name"`
	CorpID           int    `json:"corp_id"`
	CorpName         string `json:"corp_name"`
	AllianceID       int    `json:"alliance_id"`
	AllianceName     string `json:"alliance_name"`
	IsMain           bool   `json:"is_main"`
	ESITokenValid    bool   `json:"esi_token_valid"`
	ESILastError     string `json:"esi_last_error"`
	ESILastRefreshAt string `json:"esi_last_refresh_at"`
}

type profileResponse struct {
	UserID     string             `json:"user_id"`
	Characters []characterProfile `json:"characters"`
}

func (h *AuthHandler) Profile(c *core.RequestEvent) error {
	record, recordErr := auth.CurrentUser(c)
	if recordErr != nil {
		return recordErr
	}

	response := profileResponse{
		UserID:     record.Id,
		Characters: []characterProfile{},
	}

	if h.appendTestAuthProfileCharacters(c, &response, record) {
		return c.JSON(http.StatusOK, response)
	}

	if h.appendStandaloneProfileCharacters(&response, record.Id) {
		return c.JSON(http.StatusOK, response)
	}

	charID := record.GetInt("eve_character_id")
	if charID != 0 {
		corpID := record.GetInt("eve_corporation_id")
		allianceID := record.GetInt("eve_alliance_id")
		allianceName := store.GetOrgName(h.Auth.App, store.CollectionAlliances, allianceID)
		corpName := store.GetOrgName(h.Auth.App, store.CollectionCorporations, corpID)
		refresh := record.GetDateTime("esi_last_refresh_at")
		refreshAt := ""
		if !refresh.IsZero() {
			refreshAt = refresh.Time().Format(time.RFC3339)
		}
		response.Characters = append(response.Characters, characterProfile{
			CharacterID:      charID,
			Name:             record.GetString("eve_character_name"),
			CorpID:           corpID,
			CorpName:         corpName,
			AllianceID:       allianceID,
			AllianceName:     allianceName,
			IsMain:           true,
			ESITokenValid:    record.GetBool("esi_token_valid"),
			ESILastError:     record.GetString("esi_last_error"),
			ESILastRefreshAt: refreshAt,
		})
	}
	return c.JSON(http.StatusOK, response)
}

func buildCharacterProfileFromRecord(app *pocketbase.PocketBase, charRecord *core.Record) characterProfile {
	corpID := charRecord.GetInt("eve_corporation_id")
	allianceID := charRecord.GetInt("eve_alliance_id")
	allianceName := store.GetOrgName(app, store.CollectionAlliances, allianceID)
	corpName := store.GetOrgName(app, store.CollectionCorporations, corpID)
	refresh := charRecord.GetDateTime("esi_last_refresh_at")
	refreshAt := ""
	if !refresh.IsZero() {
		refreshAt = refresh.Time().Format(time.RFC3339)
	}

	return characterProfile{
		RecordID:         charRecord.Id,
		CharacterID:      charRecord.GetInt("eve_character_id"),
		Name:             charRecord.GetString("eve_character_name"),
		CorpID:           corpID,
		CorpName:         corpName,
		AllianceID:       allianceID,
		AllianceName:     allianceName,
		IsMain:           charRecord.GetBool("is_main"),
		ESITokenValid:    charRecord.GetBool("esi_token_valid"),
		ESILastError:     charRecord.GetString("esi_last_error"),
		ESILastRefreshAt: refreshAt,
	}
}

func buildTestAuthCharacterProfile(
	app *pocketbase.PocketBase,
	character auth.CharacterInfo,
	charRecord *core.Record,
) characterProfile {
	if charRecord != nil {
		return buildCharacterProfileFromRecord(app, charRecord)
	}

	return characterProfile{
		CharacterID:   int(character.CharacterID),
		Name:          character.CharacterName,
		IsMain:        character.IsPrimary,
		ESITokenValid: character.HasValidToken,
	}
}

func (h *AuthHandler) RemoveCharacter(c *core.RequestEvent) error {
	recordID := c.Request.PathValue("id")
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		return userErr
	}

	record, recordErr := h.Auth.App.FindRecordById(store.CollectionCharacters, recordID)
	if recordErr != nil || record.GetString("user") != user.Id {
		return router.NewNotFoundError("Character not found.", logging.Fields{
			"user_id":              user.Id,
			"character_record_id":  recordID,
			"character_owner_user": record.GetString("user"),
		})
	}

	isMain := record.GetBool("is_main")
	if isMain && record.GetString("auth_provider") == auth.AuthProviderTestAuth {
		return router.NewBadRequestError("Cannot remove the main character in TestAuth mode.", logging.Fields{
			"user_id":      user.Id,
			"character_id": record.GetInt("eve_character_id"),
		})
	}

	if isMain {
		others, othersErr := h.Auth.App.FindRecordsByFilter(
			store.CollectionCharacters,
			"user = {:user} && id != {:id}",
			"",
			1,
			0, dbx.Params{"user": user.Id, "id": record.Id},
		)
		if othersErr == nil && len(others) > 0 {
			return router.NewBadRequestError("Cannot remove main character while other characters remain.", logging.Fields{
				"user_id":      user.Id,
				"character_id": record.GetInt("eve_character_id"),
			})
		}
	}

	userName := user.GetString("eve_character_name")
	deleteErr := h.Auth.App.Delete(record)
	if deleteErr != nil {
		return router.NewInternalServerError("Failed to delete character.", logging.Fields{
			"character_id": record.GetInt("eve_character_id"),
		})
	}

	deletedUser := false
	if isMain {
		deletedUser, _ = h.deleteUserIfNoCharacters(user.Id)
	}

	if deletedUser {
		return c.JSON(http.StatusOK, map[string]any{"ok": true, "deleted_user": true})
	}

	if h.Audit != nil {
		h.Audit.LogEvent(&audit.Event{
			Action:          audit.ActionCharacterRemove,
			Summary:         "Removed character " + record.GetString("eve_character_name"),
			TargetUserID:    user.Id,
			TargetUserName:  userName,
			TargetCharacter: record,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "deleted_user": false})
}

type authExchangeResponse struct {
	Token     string         `json:"token"`
	Record    map[string]any `json:"record"`
	ExpiresAt string         `json:"expires_at"`
}

func (h *AuthHandler) Exchange(c *core.RequestEvent) error {
	code := c.Request.URL.Query().Get("code")
	if code == "" {
		return router.NewBadRequestError("Missing code.", logging.Fields{
			"required_field": "code",
		})
	}
	user, token, exchangeErr := h.Auth.Exchange(code)
	if exchangeErr != nil {
		return exchangeErr
	}
	expiresAt := time.Now().Add(auth.PBTokenTTL()).UTC().Format(time.RFC3339)
	return c.JSON(http.StatusOK, authExchangeResponse{
		Token:     token,
		Record:    user.PublicExport(),
		ExpiresAt: expiresAt,
	})
}

func (h *AuthHandler) Refresh(c *core.RequestEvent) error {
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		return userErr
	}
	token, tokenErr := h.Auth.IssueToken(user)
	if tokenErr != nil {
		return tokenErr
	}
	expiresAt := time.Now().Add(auth.PBTokenTTL()).UTC().Format(time.RFC3339)
	return c.JSON(http.StatusOK, authExchangeResponse{
		Token:     token,
		Record:    user.PublicExport(),
		ExpiresAt: expiresAt,
	})
}

func (h *AuthHandler) appendTestAuthProfileCharacters(c *core.RequestEvent, response *profileResponse, record *core.Record) bool {
	testAuthProvider, ok := h.Auth.Provider.(*auth.TestAuthProvider)
	if !ok || record.GetString("auth_provider") != auth.AuthProviderTestAuth {
		return false
	}

	profileCharacters, profileErr := testAuthProvider.ProfileCharacters(c.Request.Context(), record)
	if profileErr != nil || len(profileCharacters) == 0 {
		return h.appendTestAuthProfileFallback(c, response, record)
	}

	for _, character := range profileCharacters {
		charRecord := findCharacterByUserAndProviderAndEVEID(h.Auth.App, record.Id, auth.AuthProviderTestAuth, int(character.CharacterID))
		response.Characters = append(
			response.Characters,
			buildTestAuthCharacterProfile(h.Auth.App, character, charRecord),
		)
	}

	return true
}

func (h *AuthHandler) appendStandaloneProfileCharacters(response *profileResponse, userID string) bool {
	charRecords, recordsErr := h.Auth.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user}",
		"-is_main",
		0,
		0,
		dbx.Params{"user": userID},
	)
	if recordsErr != nil || len(charRecords) == 0 {
		return false
	}

	for _, charRecord := range charRecords {
		response.Characters = append(response.Characters, buildCharacterProfileFromRecord(h.Auth.App, charRecord))
	}
	return true
}

func (h *AuthHandler) appendTestAuthProfileFallback(c *core.RequestEvent, response *profileResponse, record *core.Record) bool {
	testAuthProvider, ok := h.Auth.Provider.(*auth.TestAuthProvider)
	if !ok || record.GetString("auth_provider") != auth.AuthProviderTestAuth {
		return false
	}

	mainCharacter, mainErr := testAuthProvider.MainCharacter(c.Request.Context(), record)
	if mainErr != nil {
		return false
	}

	corpID := record.GetInt("eve_corporation_id")
	allianceID := record.GetInt("eve_alliance_id")

	allianceName := store.GetOrgName(h.Auth.App, store.CollectionAlliances, allianceID)
	corpName := store.GetOrgName(h.Auth.App, store.CollectionCorporations, corpID)
	refresh := record.GetDateTime("esi_last_refresh_at")
	refreshAt := ""
	if !refresh.IsZero() {
		refreshAt = refresh.Time().Format(time.RFC3339)
	}

	response.Characters = append(response.Characters, characterProfile{
		CharacterID:      int(mainCharacter.CharacterID),
		Name:             mainCharacter.CharacterName,
		CorpID:           corpID,
		CorpName:         corpName,
		AllianceID:       allianceID,
		AllianceName:     allianceName,
		IsMain:           true,
		ESITokenValid:    mainCharacter.HasValidToken,
		ESILastRefreshAt: refreshAt,
	})
	return true
}

func (h *AuthHandler) deleteUserIfNoCharacters(userID string) (bool, error) {
	user, userErr := h.Auth.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return false, userErr
	}
	records, recordsErr := h.Auth.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user}",
		"",
		1,
		0, dbx.Params{"user": userID},
	)
	if recordsErr != nil {
		return false, recordsErr
	}

	if len(records) == 0 {
		return true, h.Auth.App.Delete(user)
	}
	return false, nil
}

func findCharacterByUserAndProviderAndEVEID(app *pocketbase.PocketBase, userID, provider string, eveCharacterID int) *core.Record {
	if app == nil || userID == "" || provider == "" || eveCharacterID <= 0 {
		return nil
	}
	records, err := app.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user} && auth_provider = {:provider} && eve_character_id = {:character_id}",
		"",
		1,
		0,
		dbx.Params{
			"user":         userID,
			"provider":     provider,
			"character_id": eveCharacterID,
		},
	)
	if err != nil || len(records) == 0 {
		return nil
	}
	return records[0]
}
