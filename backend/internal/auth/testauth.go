package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"golang.org/x/oauth2"

	"sentinel2/internal/config"
	"sentinel2/internal/esi"
	"sentinel2/internal/intel"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

const testAuthStateBytes = 32

// TestAuthProvider implements the auth.Provider interface for the TestAuth backend.
// It communicates with the external auth platform as an OAuth client using standard
// libraries and derives identity from the /oauth/api/me response.
type TestAuthProvider struct {
	App       *pocketbase.PocketBase
	OAuth     *TestAuthClient
	PublicESI *esi.ESIPublicClient
	Intel     *intel.IntelService
	Config    *config.Config
	logger    *logging.Logger
}

// NewTestAuthProvider creates a new TestAuthProvider that communicates with the external auth platform.
func NewTestAuthProvider(app *pocketbase.PocketBase, oauth *TestAuthClient, publicESI *esi.ESIPublicClient, intelService *intel.IntelService, cfg *config.Config) *TestAuthProvider {
	return &TestAuthProvider{
		App:       app,
		OAuth:     oauth,
		PublicESI: publicESI,
		Intel:     intelService,
		Config:    cfg,
		logger: logging.New(app).WithFields(logging.Fields{
			"component": "auth.testauth",
		}),
	}
}

// Name returns the provider name constant for testauth.
func (p *TestAuthProvider) Name() string {
	return AuthProviderTestAuth
}

// Authenticate redirects the user to the authorization endpoint.
func (p *TestAuthProvider) Authenticate(c *core.RequestEvent, flow AuthFlow) error {
	authURL, err := p.BuildAuthURL(c, flow)
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, authURL)
}

// BuildAuthURL constructs the authorization URL for the external auth platform.
func (p *TestAuthProvider) BuildAuthURL(c *core.RequestEvent, flow AuthFlow) (string, error) {
	state, stateErr := generateState()
	if stateErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(stateErr).
			Warn("state generation failed")
		return "", ErrFailedCreateState
	}

	redirectURL := absoluteURL(c)
	p.OAuth.SetRedirectURL(redirectURL)
	flow.RedirectBaseURL = resolveRedirectBaseURL(c)
	saveAuthFlow(p.App, state, flow)

	return p.OAuth.AuthorizationURL(state), nil
}

// Callback handles the OAuth callback. It exchanges the authorization code for
// tokens, fetches user details from /oauth/api/me, and persists the user record.
func (p *TestAuthProvider) Callback(c *core.RequestEvent) (*AuthResult, AuthFlow, error) {
	state := c.Request.URL.Query().Get("state")
	if state == "" {
		return nil, AuthFlow{}, ErrInvalidState
	}
	flow, ok := loadAuthFlow(p.App, state)
	if !ok {
		return nil, AuthFlow{}, ErrInvalidState
	}
	deleteAuthFlow(p.App, state)

	code := c.Request.URL.Query().Get("code")
	if code == "" {
		return nil, AuthFlow{}, ErrMissingCode
	}

	redirectURL := absoluteURL(c)
	p.OAuth.SetRedirectURL(redirectURL)

	// Exchange authorization code for tokens using golang.org/x/oauth2 (RFC 6749).
	token, tokenErr := p.OAuth.ExchangeCode(c.Request.Context(), code, redirectURL)
	if tokenErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(tokenErr).
			Warn("token exchange failed")
		return nil, AuthFlow{}, ErrFailedExchangeToken
	}

	// Fetch user details from the auth platform /oauth/api/me.
	// Profile, groups, and permissions are derived live from Core/Groups.
	userInfo, userInfoErr := p.OAuth.GetUserInfo(c.Request.Context(), token.AccessToken)
	if userInfoErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(userInfoErr).
			Warn("user info fetch failed")
		return nil, AuthFlow{}, ErrUserInfoFetch
	}

	sub := strings.TrimSpace(userInfo.Sub)
	if sub == "" {
		return nil, AuthFlow{}, ErrMissingSub
	}

	corpID, allianceID, accessErr := p.resolveCharacterAffiliation(c.Request.Context(), userInfo)
	if accessErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(accessErr).
			Warn("access affiliation lookup failed")
		return nil, AuthFlow{}, accessErr
	}
	if authorizeErr := p.authorizeAccess(corpID, allianceID); authorizeErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(authorizeErr).
			Warn("access denied")
		return nil, AuthFlow{}, authorizeErr
	}

	// Find or create user by auth_provider_sub.
	user, userErr := p.findOrCreateUser(sub)
	if userErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(userErr).
			Warn("user lookup failed")
		return nil, AuthFlow{}, ErrFailedPersistUser
	}

	// Resolve access level from profile groups and permission URNs.
	accessLevel := p.resolveAccessLevel(userInfo)

	mainCharacter, _ := selectMainCharacter(userInfo)
	if saveErr := p.persistUserSession(user, sub, token, accessLevel, mainCharacter, corpID, allianceID); saveErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(saveErr).
			Warn("user save failed")
		return nil, AuthFlow{}, ErrFailedPersistUser
	}

	if syncErr := p.syncMainCharacter(c.Request.Context(), user, userInfo, corpID, allianceID); syncErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(syncErr).
			Warn("main character sync failed")
		return nil, AuthFlow{}, ErrFailedPersistUser
	}

	if syncErr := p.syncLinkedCharacters(c.Request.Context(), user, userInfo); syncErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(syncErr).
			Warn("linked character sync failed")
		return nil, AuthFlow{}, ErrFailedPersistUser
	}

	accessExpiry := tokenExpiry(token)
	refreshExpiry := refreshExpiry(token)

	return &AuthResult{
		Provider: p.Name(),
		UserID:   user.Id,
		Tokens: AuthTokens{
			AccessToken:   token.AccessToken,
			AccessExpiry:  time.Unix(accessExpiry, 0),
			RefreshToken:  token.RefreshToken,
			RefreshExpiry: time.Unix(refreshExpiry, 0),
		},
	}, flow, nil
}

// Refresh refreshes an access token using the auth platform's token endpoint and
// re-fetches user details to update groups and permissions.
func (p *TestAuthProvider) Refresh(ctx context.Context, user *core.Record) (AuthTokens, error) {
	refreshToken := user.GetString("oauth_refresh_token")
	if refreshToken == "" {
		p.logger.
			WithFields(logging.Fields{"user_id": user.Id}).
			Warn("refresh missing refresh token")
		return AuthTokens{}, errors.New("missing refresh token")
	}

	// Refresh using golang.org/x/oauth2 (RFC 6749).
	token, tokenErr := p.OAuth.RefreshToken(ctx, &oauth2.Token{RefreshToken: refreshToken})
	if tokenErr != nil {
		p.logger.
			WithFields(logging.Fields{"user_id": user.Id}).
			WithErr(tokenErr).
			Warn("refresh token exchange failed")
		return AuthTokens{}, tokenErr
	}

	// Re-fetch user details to update groups/permissions from the auth platform.
	userInfo, userInfoErr := p.OAuth.GetUserInfo(ctx, token.AccessToken)
	if userInfoErr != nil {
		p.logger.
			WithFields(logging.Fields{"user_id": user.Id}).
			WithErr(userInfoErr).
			Warn("refresh user info fetch failed")
		return AuthTokens{}, userInfoErr
	}

	corpID, allianceID, accessErr := p.resolveCharacterAffiliation(ctx, userInfo)
	if accessErr != nil {
		p.logger.
			WithFields(logging.Fields{"user_id": user.Id}).
			WithErr(accessErr).
			Warn("refresh affiliation lookup failed")
		return AuthTokens{}, accessErr
	}
	if authorizeErr := p.authorizeAccess(corpID, allianceID); authorizeErr != nil {
		p.logger.
			WithFields(logging.Fields{"user_id": user.Id}).
			WithErr(authorizeErr).
			Warn("refresh access denied")
		return AuthTokens{}, authorizeErr
	}

	accessLevel := p.resolveAccessLevel(userInfo)
	mainCharacter, _ := selectMainCharacter(userInfo)
	if saveErr := p.persistUserSession(user, user.GetString("auth_provider_sub"), token, accessLevel, mainCharacter, corpID, allianceID); saveErr != nil {
		p.logger.
			WithFields(logging.Fields{"user_id": user.Id}).
			WithErr(saveErr).
			Warn("refresh user save failed")
		return AuthTokens{}, saveErr
	}

	if syncErr := p.syncMainCharacter(ctx, user, userInfo, corpID, allianceID); syncErr != nil {
		p.logger.
			WithFields(logging.Fields{"user_id": user.Id}).
			WithErr(syncErr).
			Warn("refresh main character sync failed")
		return AuthTokens{}, syncErr
	}

	if syncErr := p.syncLinkedCharacters(ctx, user, userInfo); syncErr != nil {
		p.logger.
			WithFields(logging.Fields{"user_id": user.Id}).
			WithErr(syncErr).
			Warn("refresh linked character sync failed")
		return AuthTokens{}, syncErr
	}

	accessExpiry := tokenExpiry(token)
	refreshExpiry := refreshExpiry(token)

	return AuthTokens{
		AccessToken:   token.AccessToken,
		AccessExpiry:  time.Unix(accessExpiry, 0),
		RefreshToken:  token.RefreshToken,
		RefreshExpiry: time.Unix(refreshExpiry, 0),
	}, nil
}

// Logout returns a 204 No Content response (stateless logout).
func (p *TestAuthProvider) Logout(c *core.RequestEvent) error {
	return c.NoContent(http.StatusNoContent)
}

// LinkableCharacters returns the TestAuth characters that are not yet linked in Sentinel.
// The returned slice excludes the main character and characters without valid tokens.
func (p *TestAuthProvider) LinkableCharacters(ctx context.Context, user *core.Record) ([]CharacterInfo, error) {
	chars, _, err := p.linkableCharacters(ctx, user)
	if err != nil {
		return nil, err
	}
	return chars, nil
}

// ProfileCharacters returns the TestAuth characters that are currently linked in Sentinel.
func (p *TestAuthProvider) ProfileCharacters(ctx context.Context, user *core.Record) ([]CharacterInfo, error) {
	userInfo, infoErr := p.currentUserInfo(ctx, user)
	if infoErr != nil {
		return nil, infoErr
	}

	linkedIDs, linkedErr := p.linkedCharacterIDs(user)
	if linkedErr != nil {
		return nil, linkedErr
	}

	profileCharacters := make([]CharacterInfo, 0, len(userInfo.Characters))
	for _, character := range userInfo.Characters {
		if _, ok := linkedIDs[int(character.CharacterID)]; !ok {
			continue
		}
		profileCharacters = append(profileCharacters, character)
	}

	return profileCharacters, nil
}

// MainCharacter returns the current TestAuth main character for the user.
func (p *TestAuthProvider) MainCharacter(ctx context.Context, user *core.Record) (CharacterInfo, error) {
	userInfo, infoErr := p.currentUserInfo(ctx, user)
	if infoErr != nil {
		return CharacterInfo{}, infoErr
	}

	mainCharacter, ok := selectMainCharacter(userInfo)
	if !ok {
		return CharacterInfo{}, ErrFailedFetchMainCharacter
	}

	return mainCharacter, nil
}

// LinkCharacters links the requested TestAuth characters to the current Sentinel user.
// The provider validates that each requested character is present in the current TestAuth profile
// and not already linked before persisting the new character rows.
func (p *TestAuthProvider) LinkCharacters(ctx context.Context, user *core.Record, characterIDs []int) ([]CharacterInfo, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}

	_, linkableByID, err := p.linkableCharacters(ctx, user)
	if err != nil {
		return nil, err
	}

	coll, collErr := p.App.FindCollectionByNameOrId(store.CollectionCharacters)
	if collErr != nil {
		return nil, collErr
	}

	now, _ := types.ParseDateTime(time.Now())
	requested := make([]CharacterInfo, 0, len(characterIDs))
	seen := make(map[int]struct{}, len(characterIDs))
	for _, characterID := range characterIDs {
		if characterID <= 0 {
			return nil, ErrCharacterNotLinkable
		}
		if _, ok := seen[characterID]; ok {
			continue
		}
		seen[characterID] = struct{}{}

		characterInfo, ok := linkableByID[characterID]
		if !ok {
			return nil, ErrCharacterNotLinkable
		}
		requested = append(requested, characterInfo)
	}

	linked := make([]CharacterInfo, 0, len(characterIDs))
	for _, characterInfo := range requested {
		if saveErr := p.upsertLinkedCharacter(ctx, user, coll, characterInfo, now, false, 0, 0); saveErr != nil {
			return nil, saveErr
		}
		linked = append(linked, characterInfo)
	}

	return linked, nil
}

// resolveAccessLevel determines the user's access level from profile groups and permission URNs.
// Admin membership takes priority, then staff access, then falls back to user.
func (p *TestAuthProvider) resolveAccessLevel(userInfo *UserInfo) string {
	if userInfo == nil {
		return "user"
	}

	if userInfo.IsAdmin {
		return "admin"
	}

	if p.matchesAccessGrant(userInfo, p.Config.GetTestAuthAdminGroups(), p.Config.GetTestAuthAdminPermissionURNs()) {
		return "admin"
	}

	if p.matchesAccessGrant(userInfo, p.Config.GetTestAuthStaffGroups(), p.Config.GetStaffPermissionURNs()) {
		return "staff"
	}

	return "user"
}

func (p *TestAuthProvider) matchesAccessGrant(userInfo *UserInfo, groups, permissionURNs []string) bool {
	if userInfo == nil {
		return false
	}

	targets := normalizeAccessGrantTargets(groups, permissionURNs)
	if len(targets) == 0 {
		return false
	}

	return userInfoHasAccessGrant(userInfo, targets)
}

func (p *TestAuthProvider) resolveCharacterAffiliation(ctx context.Context, userInfo *UserInfo) (corpID, allianceID int, err error) {
	if p == nil || p.PublicESI == nil || userInfo == nil {
		return 0, 0, ErrAccessDenied
	}

	mainCharacter, ok := selectMainCharacter(userInfo)
	if !ok {
		return 0, 0, ErrAccessDenied
	}

	return p.PublicESI.CharacterAffiliation(ctx, int(mainCharacter.CharacterID))
}

func (p *TestAuthProvider) linkableCharacters(ctx context.Context, user *core.Record) ([]CharacterInfo, map[int]CharacterInfo, error) {
	userInfo, infoErr := p.currentUserInfo(ctx, user)
	if infoErr != nil {
		return nil, nil, infoErr
	}

	linkedIDs, linkedErr := p.linkedCharacterIDs(user)
	if linkedErr != nil {
		return nil, nil, linkedErr
	}

	mainCharacter, _ := selectMainCharacter(userInfo)
	mainCharacterID := int64(0)
	if mainCharacter.CharacterID > 0 {
		mainCharacterID = mainCharacter.CharacterID
	}

	linkable := make([]CharacterInfo, 0, len(userInfo.Characters))
	linkableByID := make(map[int]CharacterInfo, len(userInfo.Characters))
	for _, character := range userInfo.Characters {
		if !character.HasValidToken {
			continue
		}
		if character.IsPrimary || character.CharacterID == mainCharacterID {
			continue
		}
		if _, exists := linkedIDs[int(character.CharacterID)]; exists {
			continue
		}
		linkable = append(linkable, character)
		linkableByID[int(character.CharacterID)] = character
	}

	return linkable, linkableByID, nil
}

func (p *TestAuthProvider) currentUserInfo(ctx context.Context, user *core.Record) (*UserInfo, error) {
	if p == nil || p.OAuth == nil || user == nil {
		return nil, ErrUnauthorized
	}

	accessToken := strings.TrimSpace(user.GetString("oauth_access_token"))
	if accessToken == "" {
		return nil, ErrMissingAccessToken
	}

	return p.OAuth.GetUserInfo(ctx, accessToken)
}

func (p *TestAuthProvider) linkedCharacterIDs(user *core.Record) (map[int]struct{}, error) {
	if p == nil || p.App == nil || user == nil {
		return map[int]struct{}{}, ErrUnauthorized
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
		return nil, recordsErr
	}

	linkedIDs := make(map[int]struct{}, len(records))
	for _, record := range records {
		characterID := record.GetInt("eve_character_id")
		if characterID > 0 {
			linkedIDs[characterID] = struct{}{}
		}
	}
	return linkedIDs, nil
}

func (p *TestAuthProvider) authorizeAccess(corpID, allianceID int) error {
	allowed, accessErr := p.allowedAccess(corpID, allianceID)
	if accessErr != nil {
		return ErrFailedCheckAccess
	}
	if !allowed {
		return ErrAccessDenied
	}
	return nil
}

func (p *TestAuthProvider) allowedAccess(corpID, allianceID int) (bool, error) {
	corpAllowed, corpErr := p.allowedID("allowed_corporations", corpID)
	if corpErr != nil {
		return false, corpErr
	}
	allianceAllowed, allianceErr := p.allowedID("allowed_alliances", allianceID)
	if allianceErr != nil {
		return false, allianceErr
	}

	if corpAllowed || allianceAllowed {
		return true, nil
	}

	if !p.hasAllowlist("allowed_corporations") && !p.hasAllowlist("allowed_alliances") {
		return false, nil
	}
	return false, nil
}

func (p *TestAuthProvider) hasAllowlist(collection string) bool {
	records, recordsErr := p.App.FindRecordsByFilter(collection, "", "", 1, 0, nil)
	return recordsErr == nil && len(records) > 0
}

func (p *TestAuthProvider) allowedID(collection string, id int) (bool, error) {
	if id == 0 {
		return false, nil
	}
	records, recordsErr := p.App.FindRecordsByFilter(collection, "eve_id = {:id}", "", 1, 0, map[string]any{"id": id})
	if recordsErr != nil {
		return false, recordsErr
	}
	return len(records) > 0, nil
}

func normalizeAccessGrant(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeAccessGrantTargets(groups, permissionURNs []string) map[string]struct{} {
	targets := make(map[string]struct{}, len(groups)+len(permissionURNs))
	for _, entry := range groups {
		normalized := normalizeAccessGrant(entry)
		if normalized != "" {
			targets[normalized] = struct{}{}
		}
	}
	for _, entry := range permissionURNs {
		normalized := normalizeAccessGrant(entry)
		if normalized != "" {
			targets[normalized] = struct{}{}
		}
	}
	return targets
}

func userInfoHasAccessGrant(userInfo *UserInfo, targets map[string]struct{}) bool {
	if hasMatchingGrant(userInfo.Groups, targets) {
		return true
	}
	if hasMatchingMembership(userInfo.GroupMemberships, targets) {
		return true
	}
	return hasMatchingGrant(userInfo.PermissionURNs, targets)
}

func hasMatchingGrant(values []string, targets map[string]struct{}) bool {
	for _, value := range values {
		if _, ok := targets[normalizeAccessGrant(value)]; ok {
			return true
		}
	}
	return false
}

func hasMatchingMembership(memberships []GroupMembership, targets map[string]struct{}) bool {
	for _, membership := range memberships {
		if _, ok := targets[normalizeAccessGrant(membership.GroupID)]; ok {
			return true
		}
		if _, ok := targets[normalizeAccessGrant(membership.GroupName)]; ok {
			return true
		}
	}
	return false
}

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

	mainCharacter, ok := selectMainCharacter(userInfo)
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

func selectMainCharacter(userInfo *UserInfo) (CharacterInfo, bool) {
	if userInfo == nil || len(userInfo.Characters) == 0 {
		return CharacterInfo{}, false
	}

	if userInfo.MainCharacterID != nil {
		for _, character := range userInfo.Characters {
			if character.CharacterID == *userInfo.MainCharacterID {
				return character, true
			}
		}
	}

	for _, character := range userInfo.Characters {
		if character.IsPrimary {
			return character, true
		}
	}

	return CharacterInfo{}, false
}

// findOrCreateUser looks up a user by auth_provider_sub and creates a new user
// if none exists.
func (p *TestAuthProvider) findOrCreateUser(sub string) (*core.Record, error) {
	coll, collErr := p.App.FindCollectionByNameOrId(store.CollectionUsers)
	if collErr != nil {
		p.logger.
			WithErr(collErr).
			Warn("user collection lookup failed")
		return nil, collErr
	}

	records, recordsErr := p.App.FindRecordsByFilter(
		coll.Name,
		"auth_provider = {:provider} && auth_provider_sub = {:sub}",
		"",
		1,
		0,
		map[string]any{"provider": p.Name(), "sub": sub},
	)
	if recordsErr != nil {
		p.logger.
			WithErr(recordsErr).
			Warn("user query failed")
		return nil, recordsErr
	}

	if len(records) > 0 {
		return records[0], nil
	}

	record := core.NewRecord(coll)
	record.Set("sub", sub)
	record.Set("auth_provider", p.Name())
	record.Set("auth_provider_sub", sub)
	record.SetEmail(fmt.Sprintf("sso-%s@auth.invalid", base64.RawURLEncoding.EncodeToString([]byte(sub))))
	record.SetRandomPassword()
	record.Set("created_at", time.Now())
	record.Set("access_level", "user")
	if saveErr := p.App.Save(record); saveErr != nil {
		p.logger.
			WithErr(saveErr).
			Warn("user create failed")
		return nil, saveErr
	}

	if p.Intel != nil {
		if _, tokenErr := p.Intel.GetOrCreateUploaderToken(record.Id); tokenErr != nil {
			p.logger.
				WithFields(logging.Fields{"user_id": record.Id}).
				WithErr(tokenErr).
				Warn("uploader token seed failed")
			return nil, tokenErr
		}
	}
	return record, nil
}

// refreshExpiry returns the refresh token expiry timestamp.
func refreshExpiry(token *oauth2.Token) int64 {
	value := token.Extra("refresh_expires_in")
	if v, ok := value.(float64); ok {
		return time.Now().Add(time.Duration(v) * time.Second).Unix()
	}
	return time.Now().Add(30 * 24 * time.Hour).Unix()
}

// generateState generates a cryptographically secure random state string.
func generateState() (string, error) {
	b := make([]byte, testAuthStateBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
