package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"golang.org/x/oauth2"

	"sentinel2/internal/intel"
	"sentinel2/internal/logging"
	"sentinel2/internal/oidc"
	"sentinel2/internal/store"
)

type TestAuthProvider struct {
	App   *pocketbase.PocketBase
	OIDC  *oidc.Client
	Intel *intel.IntelService
}

type idTokenClaims struct {
	Sub string `json:"sub"`
}

func NewTestAuthProvider(app *pocketbase.PocketBase, oidcClient *oidc.Client, intelService *intel.IntelService) *TestAuthProvider {
	return &TestAuthProvider{App: app, OIDC: oidcClient, Intel: intelService}
}

func (p *TestAuthProvider) Name() string {
	return AuthProviderTestAuth
}

func (p *TestAuthProvider) Authenticate(c *core.RequestEvent, flow AuthFlow) error {
	authURL, err := p.BuildAuthURL(c, flow)
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, authURL)
}

func (p *TestAuthProvider) BuildAuthURL(c *core.RequestEvent, flow AuthFlow) (string, error) {
	state, stateErr := oidc.RandomState()
	if stateErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(stateErr).
			Warn("oidc state generation failed")
		return "", ErrFailedCreateState
	}

	nonce, nonceErr := oidc.RandomNonce()
	if nonceErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(nonceErr).
			Warn("oidc nonce generation failed")
		return "", ErrFailedCreateNonce
	}

	saveAuthFlow(p.App, state, flow)

	redirectURL := absoluteURL(c)
	p.OIDC.OAuth2Config.RedirectURL = redirectURL

	authURL := p.OIDC.OAuth2Config.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce))
	return authURL, nil
}

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
	p.OIDC.OAuth2Config.RedirectURL = redirectURL

	token, tokenErr := p.OIDC.OAuth2Config.Exchange(c.Request.Context(), code)
	if tokenErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(tokenErr).
			Warn("oidc token exchange failed")
		return nil, AuthFlow{}, ErrFailedExchangeToken
	}

	rawIDToken, _ := token.Extra("id_token").(string)
	if rawIDToken == "" {
		logging.WithRequest(p.App, c).
			Warn("oidc missing id_token")
		return nil, AuthFlow{}, ErrMissingIDToken
	}

	idToken, idTokenErr := p.OIDC.Verifier.Verify(c.Request.Context(), rawIDToken)
	if idTokenErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(idTokenErr).
			Warn("oidc id_token verify failed")
		return nil, AuthFlow{}, ErrInvalidIDToken
	}

	claims := idTokenClaims{}
	if claimsErr := idToken.Claims(&claims); claimsErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(claimsErr).
			Warn("oidc id_token claims failed")
		return nil, AuthFlow{}, ErrInvalidClaims
	}

	sub := claims.Sub
	if sub == "" {
		return nil, AuthFlow{}, ErrMissingSub
	}

	accessToken := token.AccessToken
	rolesOK, rolesErr := oidc.VerifyRoles(accessToken, p.OIDC.Config.RequiredRoles())
	if rolesErr != nil || !rolesOK {
		if rolesErr != nil {
			logging.WithRequest(p.App, c).
				WithErr(rolesErr).
				Warn("oidc role check failed")
		}
		return nil, AuthFlow{}, ErrMissingRequiredRoles
	}

	user, userErr := p.findOrCreateUser(sub)
	if userErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(userErr).
			Warn("oidc user lookup failed")
		return nil, AuthFlow{}, ErrFailedPersistUser
	}

	accessLevel := p.resolveAccessLevel(c, user.GetString("access_level"), accessToken)

	accessExpiry := tokenExpiry(token)
	refreshExpiry := refreshExpiry(token)
	user.Set("auth_provider", p.Name())
	user.Set("auth_provider_sub", sub)
	user.Set("sub", sub)
	user.Set("access_level", accessLevel)
	user.Set("oauth_access_token", accessToken)
	accessExpiresAt, _ := types.ParseDateTime(time.Unix(accessExpiry, 0))
	user.Set("oauth_access_expires_at", accessExpiresAt)
	if token.RefreshToken != "" {
		user.Set("oauth_refresh_token", token.RefreshToken)
		refreshExpiresAt, _ := types.ParseDateTime(time.Unix(refreshExpiry, 0))
		user.Set("oauth_refresh_expires_at", refreshExpiresAt)
	}
	if saveErr := p.App.Save(user); saveErr != nil {
		logging.WithRequest(p.App, c).
			WithErr(saveErr).
			Warn("oidc user save failed")
		return nil, AuthFlow{}, ErrFailedPersistUser
	}

	return &AuthResult{
		Provider: p.Name(),
		UserID:   user.Id,
		Tokens: AuthTokens{
			AccessToken:   accessToken,
			AccessExpiry:  time.Unix(accessExpiry, 0),
			RefreshToken:  token.RefreshToken,
			RefreshExpiry: time.Unix(refreshExpiry, 0),
			IDToken:       rawIDToken,
		},
	}, flow, nil
}

func (p *TestAuthProvider) Refresh(ctx context.Context, user *core.Record) (AuthTokens, error) {
	refreshToken := user.GetString("oauth_refresh_token")
	if refreshToken == "" {
		logging.New(p.App).
			WithFields(logging.Fields{"user_id": user.Id}).
			Warn("oidc refresh missing refresh token")
		return AuthTokens{}, errors.New("missing refresh token")
	}

	token, tokenErr := p.OIDC.OAuth2Config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}).Token()
	if tokenErr != nil {
		logging.New(p.App).
			WithFields(logging.Fields{"user_id": user.Id}).
			WithErr(tokenErr).
			Warn("oidc refresh token exchange failed")
		return AuthTokens{}, tokenErr
	}

	rawIDToken, _ := token.Extra("id_token").(string)
	if rawIDToken == "" {
		logging.New(p.App).
			WithFields(logging.Fields{"user_id": user.Id}).
			Warn("oidc refresh missing id_token")
		return AuthTokens{}, errors.New("missing id_token")
	}

	if _, verifyErr := p.OIDC.Verifier.Verify(ctx, rawIDToken); verifyErr != nil {
		logging.New(p.App).
			WithFields(logging.Fields{"user_id": user.Id}).
			WithErr(verifyErr).
			Warn("oidc refresh id_token verify failed")
		return AuthTokens{}, verifyErr
	}

	accessExpiry := tokenExpiry(token)
	refreshExpiry := refreshExpiry(token)
	accessLevel := p.resolveAccessLevel(nil, user.GetString("access_level"), token.AccessToken)
	user.Set("access_level", accessLevel)
	user.Set("oauth_access_token", token.AccessToken)
	accessExpiresAt, _ := types.ParseDateTime(time.Unix(accessExpiry, 0))
	user.Set("oauth_access_expires_at", accessExpiresAt)
	if token.RefreshToken != "" {
		user.Set("oauth_refresh_token", token.RefreshToken)
		refreshExpiresAt, _ := types.ParseDateTime(time.Unix(refreshExpiry, 0))
		user.Set("oauth_refresh_expires_at", refreshExpiresAt)
	}
	if saveErr := p.App.Save(user); saveErr != nil {
		logging.New(p.App).
			WithFields(logging.Fields{"user_id": user.Id}).
			WithErr(saveErr).
			Warn("oidc refresh user save failed")
		return AuthTokens{}, saveErr
	}

	return AuthTokens{
		AccessToken:   token.AccessToken,
		AccessExpiry:  time.Unix(accessExpiry, 0),
		RefreshToken:  token.RefreshToken,
		RefreshExpiry: time.Unix(refreshExpiry, 0),
		IDToken:       rawIDToken,
	}, nil
}

func (p *TestAuthProvider) Logout(c *core.RequestEvent) error {
	return c.NoContent(http.StatusNoContent)
}

func (p *TestAuthProvider) resolveAccessLevel(c *core.RequestEvent, currentLevel, accessToken string) string {
	if currentLevel == "admin" {
		return currentLevel
	}
	staffRoles := p.OIDC.Config.StaffRoles()
	if len(staffRoles) == 0 {
		return currentLevel
	}
	staffOK, staffErr := oidc.VerifyRoles(accessToken, staffRoles)
	if staffErr != nil {
		if c != nil {
			logging.WithRequest(p.App, c).
				WithErr(staffErr).
				Warn("oidc staff role check failed")
		} else {
			logging.New(p.App).
				WithFields(logging.Fields{"reason": "refresh"}).
				WithErr(staffErr).
				Warn("oidc staff role check failed on refresh")
		}
		return currentLevel
	}
	if staffOK {
		return "staff"
	}
	return "user"
}

func (p *TestAuthProvider) findOrCreateUser(sub string) (*core.Record, error) {
	coll, collErr := p.App.FindCollectionByNameOrId(store.CollectionUsers)
	if collErr != nil {
		logging.New(p.App).
			WithErr(collErr).
			Warn("oidc user collection lookup failed")
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
		logging.New(p.App).
			WithErr(recordsErr).
			Warn("oidc user query failed")
		return nil, recordsErr
	}

	if len(records) > 0 {
		return records[0], nil
	}

	legacy, legacyErr := p.App.FindRecordsByFilter(
		coll.Name,
		"sub = {:sub}",
		"",
		1,
		0,
		map[string]any{"sub": sub},
	)
	if legacyErr == nil && len(legacy) > 0 {
		return legacy[0], nil
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
		logging.New(p.App).
			WithErr(saveErr).
			Warn("oidc user create failed")
		return nil, saveErr
	}
	if p.Intel != nil {
		if _, tokenErr := p.Intel.GetOrCreateUploaderToken(record.Id); tokenErr != nil {
			logging.New(p.App).
				WithFields(logging.Fields{"user_id": record.Id}).
				WithErr(tokenErr).
				Warn("oidc uploader token seed failed")
			return nil, tokenErr
		}
	}
	return record, nil
}

func refreshExpiry(token *oauth2.Token) int64 {
	value := token.Extra("refresh_expires_in")
	if v, ok := value.(float64); ok {
		return time.Now().Add(time.Duration(v) * time.Second).Unix()
	}
	return time.Now().Add(30 * 24 * time.Hour).Unix()
}
