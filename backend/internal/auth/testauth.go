package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

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
	App           *pocketbase.PocketBase
	OAuth         *TestAuthClient
	PublicESI     *esi.ESIPublicClient
	Intel         *intel.IntelService
	Config        *config.Config
	PublicBaseURL string
	DevMode       bool
	logger        *logging.Logger
}

// NewTestAuthProvider creates a new TestAuthProvider that communicates with the external auth platform.
func NewTestAuthProvider(app *pocketbase.PocketBase, oauth *TestAuthClient, publicESI *esi.ESIPublicClient, intelService *intel.IntelService, cfg *config.Config, publicBaseURL string, devMode bool) *TestAuthProvider {
	return &TestAuthProvider{
		App:           app,
		OAuth:         oauth,
		PublicESI:     publicESI,
		Intel:         intelService,
		Config:        cfg,
		PublicBaseURL: publicBaseURL,
		DevMode:       devMode,
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

	redirectURL, redirectURLErr := resolveCallbackURL(c, p.PublicBaseURL, p.DevMode)
	if redirectURLErr != nil {
		return "", redirectURLErr
	}
	p.OAuth.SetRedirectURL(redirectURL)
	redirectBaseURL, redirectErr := resolveRedirectBaseURL(c, p.PublicBaseURL, p.DevMode)
	if redirectErr != nil {
		return "", redirectErr
	}
	flow.RedirectBaseURL = redirectBaseURL
	saveAuthFlow(p.App, state, flow)
	saveAuthCallbackURL(p.App, state, redirectURL)

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

	code := c.Request.URL.Query().Get("code")
	if code == "" {
		return nil, AuthFlow{}, ErrMissingCode
	}

	redirectURL, callbackOK := loadAuthCallbackURL(p.App, state)
	if !callbackOK {
		return nil, AuthFlow{}, ErrInvalidState
	}
	deleteAuthFlow(p.App, state)
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

	mainCharacter, _ := SelectMainCharacter(userInfo)
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
	mainCharacter, _ := SelectMainCharacter(userInfo)
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

	mainCharacter, ok := SelectMainCharacter(userInfo)
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

	nowDT, _ := types.ParseDateTime(time.Now())
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
		if saveErr := p.upsertLinkedCharacter(ctx, user, coll, characterInfo, nowDT, false, 0, 0); saveErr != nil {
			return nil, saveErr
		}
		linked = append(linked, characterInfo)
	}

	return linked, nil
}
