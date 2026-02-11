package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"golang.org/x/oauth2"

	"sentinel2/internal/esi"
	"sentinel2/internal/intel"
	"sentinel2/internal/oidc"
	"sentinel2/internal/store"
)

type EVEProvider struct {
	App       *pocketbase.PocketBase
	OAuth2    oauth2.Config
	ESI       esi.ESIClient
	PublicESI *esi.ESIPublicClient
}

type eveTokenClaims struct {
	Sub  string   `json:"sub"`
	Name string   `json:"name"`
	Exp  int64    `json:"exp"`
	Scp  []string `json:"scp"`
}

func NewEVEProvider(app *pocketbase.PocketBase, oauthConfig oauth2.Config, esiClient esi.ESIClient, publicESI *esi.ESIPublicClient) *EVEProvider {
	return &EVEProvider{
		App:       app,
		OAuth2:    oauthConfig,
		ESI:       esiClient,
		PublicESI: publicESI,
	}
}

func (p *EVEProvider) Name() string {
	return AuthProviderEVE
}

func (p *EVEProvider) Authenticate(c *core.RequestEvent, flow AuthFlow) error {
	authURL, err := p.BuildAuthURL(c, flow)
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, authURL)
}

func (p *EVEProvider) BuildAuthURL(c *core.RequestEvent, flow AuthFlow) (string, error) {
	state, stateErr := oidc.RandomState()
	if stateErr != nil {
		return "", ErrFailedCreateState
	}
	saveAuthFlow(p.App, state, flow)

	redirectURL := absoluteURL(c, "/api/auth/callback")
	p.OAuth2.RedirectURL = redirectURL

	authURL := p.OAuth2.AuthCodeURL(state)
	return authURL, nil
}

func (p *EVEProvider) Callback(c *core.RequestEvent) (*AuthResult, AuthFlow, error) {
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

	redirectURL := absoluteURL(c, "/api/auth/callback")
	p.OAuth2.RedirectURL = redirectURL

	token, tokenErr := p.OAuth2.Exchange(c.Request.Context(), code)
	if tokenErr != nil {
		return nil, AuthFlow{}, ErrFailedExchangeToken
	}

	claims, claimsErr := parseEVEToken(token.AccessToken)
	if claimsErr != nil {
		return nil, AuthFlow{}, ErrFailedDecodeToken
	}

	charID, charIDErr := parseCharacterID(claims.Sub)
	if charIDErr != nil {
		return nil, AuthFlow{}, ErrInvalidCharacter
	}

	corpID, allianceID, affiliationErr := p.ESI.CharacterAffiliation(c.Request.Context(), charID)
	if affiliationErr != nil {
		return nil, AuthFlow{}, ErrFailedFetchCharacter
	}

	linkUserID := flow.LinkUserID
	existingChar, _ := p.findCharacterByID(charID)

	if linkUserID != "" {
		return p.handleLinkCallback(c, flow, linkContext{
			token:        token,
			claims:       claims,
			charID:       charID,
			corpID:       corpID,
			allianceID:   allianceID,
			existingChar: existingChar,
			linkUserID:   linkUserID,
		})
	}

	result, callbackErr := p.handleLoginCallback(c, loginContext{
		token:        token,
		claims:       claims,
		charID:       charID,
		corpID:       corpID,
		allianceID:   allianceID,
		existingChar: existingChar,
	})
	if callbackErr != nil {
		return nil, AuthFlow{}, callbackErr
	}
	return result, flow, nil
}

type linkContext struct {
	token        *oauth2.Token
	claims       *eveTokenClaims
	charID       int
	corpID       int
	allianceID   int
	existingChar *core.Record
	linkUserID   string
}

func (p *EVEProvider) handleLinkCallback(c *core.RequestEvent, flow AuthFlow, ctx linkContext) (*AuthResult, AuthFlow, error) {
	user, userErr := p.App.FindRecordById(store.CollectionUsers, ctx.linkUserID)
	if userErr != nil {
		return nil, AuthFlow{}, ErrUnauthorized
	}

	if ctx.existingChar != nil {
		existingUserID := ctx.existingChar.GetString("user")
		if existingUserID != "" && existingUserID != user.Id {
			return nil, AuthFlow{}, ErrCharacterAlreadyLinked
		}
	}

	mainChar, _ := p.findMainCharacter(user.Id)
	isMain := mainChar == nil
	if ctx.existingChar != nil && ctx.existingChar.GetBool("is_main") {
		isMain = true
	}
	if isMain {
		if authorizeErr := p.authorizeAccess(ctx.corpID, ctx.allianceID); authorizeErr != nil {
			return nil, AuthFlow{}, authorizeErr
		}
	}

	charRecord, upsertErr := p.upsertCharacterForUser(c.Request.Context(), user, characterUpsertInput{
		existing:   ctx.existingChar,
		charID:     ctx.charID,
		name:       ctx.claims.Name,
		corpID:     ctx.corpID,
		allianceID: ctx.allianceID,
		token:      ctx.token,
		scopes:     ctx.claims.Scp,
	}, isMain)
	if upsertErr != nil {
		return nil, AuthFlow{}, upsertErr
	}
	if isMain {
		if updateErr := p.updateUserFromCharacter(user, charRecord); updateErr != nil {
			return nil, AuthFlow{}, ErrFailedPersistUser
		}
	}

	return &AuthResult{
		Provider: p.Name(),
		UserID:   user.Id,
		Tokens:   oauthTokens(ctx.token),
	}, flow, nil
}

type loginContext struct {
	token        *oauth2.Token
	claims       *eveTokenClaims
	charID       int
	corpID       int
	allianceID   int
	existingChar *core.Record
}

func (p *EVEProvider) handleLoginCallback(c *core.RequestEvent, ctx loginContext) (*AuthResult, error) {
	user, userErr := p.userForLogin(ctx.existingChar, ctx.charID)
	if userErr != nil {
		return nil, ErrFailedPersistUser
	}

	var mainChar *core.Record
	if ctx.existingChar == nil {
		if authorizeErr := p.authorizeAccess(ctx.corpID, ctx.allianceID); authorizeErr != nil {
			return nil, authorizeErr
		}

		charRecord, upsertErr := p.upsertCharacterForUser(c.Request.Context(), user, characterUpsertInput{
			existing:   ctx.existingChar,
			charID:     ctx.charID,
			name:       ctx.claims.Name,
			corpID:     ctx.corpID,
			allianceID: ctx.allianceID,
			token:      ctx.token,
			scopes:     ctx.claims.Scp,
		}, true)
		if upsertErr != nil {
			return nil, upsertErr
		}
		mainChar = charRecord
		if updateErr := p.updateUserFromCharacter(user, charRecord); updateErr != nil {
			return nil, ErrFailedPersistUser
		}
	} else {
		charRecord, upsertErr := p.upsertCharacterForUser(c.Request.Context(), user, characterUpsertInput{
			existing:   ctx.existingChar,
			charID:     ctx.charID,
			name:       ctx.claims.Name,
			corpID:     ctx.corpID,
			allianceID: ctx.allianceID,
			token:      ctx.token,
			scopes:     ctx.claims.Scp,
		}, ctx.existingChar.GetBool("is_main"))
		if upsertErr != nil {
			return nil, upsertErr
		}
		ctx.existingChar = charRecord

		mainChar, _ = p.findMainCharacter(user.Id)
		if mainChar == nil {
			if authorizeErr := p.authorizeAccess(ctx.corpID, ctx.allianceID); authorizeErr != nil {
				return nil, authorizeErr
			}
			ctx.existingChar.Set("is_main", true)
			if saveErr := p.App.Save(ctx.existingChar); saveErr != nil {
				return nil, ErrFailedPersistCharacter
			}
			mainChar = ctx.existingChar
		}

		if mainChar != nil {
			if mainChar.Id != ctx.existingChar.Id {
				if ensureErr := p.refreshMainAffiliation(c.Request.Context(), mainChar); ensureErr != nil {
					return nil, ensureErr
				}
			} else {
				if authorizeErr := p.authorizeAccess(ctx.corpID, ctx.allianceID); authorizeErr != nil {
					return nil, authorizeErr
				}
			}
		}

		if mainChar != nil {
			if updateErr := p.updateUserFromCharacter(user, mainChar); updateErr != nil {
				return nil, ErrFailedPersistUser
			}
		}
	}

	sessionTokens := AuthTokens{
		AccessToken:   ctx.token.AccessToken,
		AccessExpiry:  time.Unix(tokenExpiry(ctx.token), 0),
		RefreshToken:  ctx.token.RefreshToken,
		RefreshExpiry: time.Unix(eveRefreshExpiry(), 0),
	}
	if mainChar != nil {
		if tokens, ok := tokensFromCharacter(mainChar); ok {
			sessionTokens = tokens
		}
	}

	return &AuthResult{
		Provider: p.Name(),
		UserID:   user.Id,
		Tokens:   sessionTokens,
	}, nil
}

type characterUpsertInput struct {
	existing   *core.Record
	charID     int
	name       string
	corpID     int
	allianceID int
	token      *oauth2.Token
	scopes     []string
}

func (p *EVEProvider) upsertCharacterForUser(ctx context.Context, user *core.Record, input characterUpsertInput, isMain bool) (*core.Record, error) {
	charRecord, upsertErr := p.upsertCharacter(
		user.Id,
		input.existing,
		input.charID,
		input.name,
		input.corpID,
		input.allianceID,
		input.token,
		input.scopes,
		isMain,
	)
	if upsertErr != nil {
		return nil, ErrFailedPersistCharacter
	}
	ensureOrgName(ctx, p.App, p.PublicESI, store.CollectionCorporations, input.corpID)
	ensureOrgName(ctx, p.App, p.PublicESI, store.CollectionAlliances, input.allianceID)
	return charRecord, nil
}

func (p *EVEProvider) authorizeAccess(corpID int, allianceID int) error {
	allowed, accessErr := p.allowedAccess(corpID, allianceID)
	if accessErr != nil {
		return ErrFailedCheckAccess
	}
	if !allowed {
		return ErrAccessDenied
	}
	return nil
}

func (p *EVEProvider) refreshMainAffiliation(ctx context.Context, mainChar *core.Record) error {
	mainCharID := mainChar.GetInt("eve_character_id")
	mainCorp, mainAlliance, affiliationErr := p.ESI.CharacterAffiliation(ctx, mainCharID)
	if affiliationErr != nil {
		return ErrFailedFetchMainCharacter
	}
	if authorizeErr := p.authorizeAccess(mainCorp, mainAlliance); authorizeErr != nil {
		return authorizeErr
	}
	mainChar.Set("eve_corporation_id", mainCorp)
	mainChar.Set("eve_alliance_id", mainAlliance)
	ensureOrgName(ctx, p.App, p.PublicESI, store.CollectionCorporations, mainCorp)
	ensureOrgName(ctx, p.App, p.PublicESI, store.CollectionAlliances, mainAlliance)
	if saveErr := p.App.Save(mainChar); saveErr != nil {
		return ErrFailedPersistCharacter
	}
	return nil
}

func oauthTokens(token *oauth2.Token) AuthTokens {
	return AuthTokens{
		AccessToken:   token.AccessToken,
		AccessExpiry:  time.Unix(tokenExpiry(token), 0),
		RefreshToken:  token.RefreshToken,
		RefreshExpiry: time.Unix(eveRefreshExpiry(), 0),
	}
}

func (p *EVEProvider) Refresh(ctx context.Context, user *core.Record) (AuthTokens, error) {
	mainChar, mainErr := p.findMainCharacter(user.Id)
	if mainErr != nil || mainChar == nil {
		return AuthTokens{}, errors.New("missing main character")
	}

	return p.refreshCharacter(ctx, user, mainChar)
}

func (p *EVEProvider) Logout(c *core.RequestEvent) error {
	return c.NoContent(http.StatusNoContent)
}

func parseEVEToken(accessToken string) (*eveTokenClaims, error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) < 2 {
		return nil, errors.New("invalid token")
	}
	payload, decodeErr := base64.RawURLEncoding.DecodeString(parts[1])
	if decodeErr != nil {
		return nil, decodeErr
	}
	var claims eveTokenClaims
	if unmarshalErr := json.Unmarshal(payload, &claims); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	return &claims, nil
}

func parseCharacterID(sub string) (int, error) {
	parts := strings.Split(sub, ":")
	if len(parts) < 3 {
		return 0, errors.New("invalid sub")
	}
	return strconv.Atoi(parts[len(parts)-1])
}

func (p *EVEProvider) userForLogin(existingChar *core.Record, characterID int) (*core.Record, error) {
	if existingChar != nil {
		userID := existingChar.GetString("user")
		if userID == "" {
			return nil, errors.New("missing character owner")
		}
		return p.App.FindRecordById(store.CollectionUsers, userID)
	}
	return p.findOrCreateUser(characterID)
}

func (p *EVEProvider) findCharacterByID(characterID int) (*core.Record, error) {
	records, recordsErr := p.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"eve_character_id = {:id}",
		"",
		1,
		0,
		map[string]any{"id": characterID},
	)
	if recordsErr != nil || len(records) == 0 {
		return nil, recordsErr
	}
	return records[0], nil
}

func (p *EVEProvider) findMainCharacter(userID string) (*core.Record, error) {
	records, recordsErr := p.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user} && is_main = true",
		"",
		1,
		0,
		map[string]any{"user": userID},
	)
	if recordsErr != nil || len(records) == 0 {
		return nil, recordsErr
	}
	return records[0], nil
}

func (p *EVEProvider) upsertCharacter(userID string, existing *core.Record, characterID int, name string, corpID int, allianceID int, token *oauth2.Token, scopes []string, isMain bool) (*core.Record, error) {
	coll, collErr := p.App.FindCollectionByNameOrId(store.CollectionCharacters)
	if collErr != nil {
		return nil, collErr
	}

	record := existing
	if record == nil {
		record = core.NewRecord(coll)
	}

	if existing != nil {
		owner := existing.GetString("user")
		if owner != "" && owner != userID {
			return nil, errors.New("character linked to another user")
		}
	}

	record.Set("user", userID)
	record.Set("eve_character_id", characterID)
	record.Set("eve_character_name", name)
	record.Set("eve_corporation_id", corpID)
	record.Set("eve_alliance_id", allianceID)
	record.Set("is_main", isMain)
	record.Set("oauth_access_token", token.AccessToken)
	accessExpiresAt, _ := types.ParseDateTime(time.Unix(tokenExpiry(token), 0))
	record.Set("oauth_access_expires_at", accessExpiresAt)
	if token.RefreshToken != "" {
		record.Set("oauth_refresh_token", token.RefreshToken)
		refreshExpiresAt, _ := types.ParseDateTime(time.Unix(eveRefreshExpiry(), 0))
		record.Set("oauth_refresh_expires_at", refreshExpiresAt)
	}
	record.Set("oauth_scopes", strings.Join(scopes, " "))
	lastRefreshAt, _ := types.ParseDateTime(time.Now())
	record.Set("esi_last_refresh_at", lastRefreshAt)
	record.Set("esi_last_error", "")
	record.Set("esi_token_valid", true)

	if saveErr := p.App.Save(record); saveErr != nil {
		return nil, saveErr
	}
	return record, nil
}

func (p *EVEProvider) updateUserFromCharacter(user *core.Record, character *core.Record) error {
	user.Set("auth_provider", p.Name())
	user.Set("auth_provider_sub", strconv.Itoa(character.GetInt("eve_character_id")))
	user.Set("sub", strconv.Itoa(character.GetInt("eve_character_id")))
	user.Set("eve_character_id", character.GetInt("eve_character_id"))
	user.Set("eve_character_name", character.GetString("eve_character_name"))
	user.Set("eve_corporation_id", character.GetInt("eve_corporation_id"))
	user.Set("eve_alliance_id", character.GetInt("eve_alliance_id"))
	return p.App.Save(user)
}

func tokensFromCharacter(record *core.Record) (AuthTokens, bool) {
	if record == nil {
		return AuthTokens{}, false
	}
	accessToken := record.GetString("oauth_access_token")
	if accessToken == "" {
		return AuthTokens{}, false
	}

	accessExpiry := record.GetDateTime("oauth_access_expires_at")
	refreshExpiry := record.GetDateTime("oauth_refresh_expires_at")

	return AuthTokens{
		AccessToken:   accessToken,
		AccessExpiry:  accessExpiry.Time(),
		RefreshToken:  record.GetString("oauth_refresh_token"),
		RefreshExpiry: refreshExpiry.Time(),
	}, true
}

func (p *EVEProvider) refreshCharacter(ctx context.Context, user *core.Record, character *core.Record) (AuthTokens, error) {
	if character == nil {
		return AuthTokens{}, errors.New("missing character")
	}

	refreshToken := character.GetString("oauth_refresh_token")
	if refreshToken == "" {
		return AuthTokens{}, errors.New("missing refresh token")
	}

	token, tokenErr := p.OAuth2.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}).Token()
	if tokenErr != nil {
		return AuthTokens{}, tokenErr
	}

	claims, claimsErr := parseEVEToken(token.AccessToken)
	if claimsErr != nil {
		return AuthTokens{}, claimsErr
	}
	charID, charIDErr := parseCharacterID(claims.Sub)
	if charIDErr != nil {
		return AuthTokens{}, charIDErr
	}

	corpID, allianceID, affiliationErr := p.ESI.CharacterAffiliation(ctx, charID)
	if affiliationErr != nil {
		return AuthTokens{}, affiliationErr
	}

	isMain := character.GetBool("is_main")
	if isMain {
		allowed, accessErr := p.allowedAccess(corpID, allianceID)
		if accessErr != nil {
			return AuthTokens{}, accessErr
		}
		if !allowed {
			return AuthTokens{}, ErrAccessDenied
		}
	}

	character.Set("eve_character_id", charID)
	character.Set("eve_character_name", claims.Name)
	character.Set("eve_corporation_id", corpID)
	character.Set("eve_alliance_id", allianceID)
	ensureOrgName(ctx, p.App, p.PublicESI, store.CollectionCorporations, corpID)
	ensureOrgName(ctx, p.App, p.PublicESI, store.CollectionAlliances, allianceID)
	character.Set("oauth_access_token", token.AccessToken)
	accessExpiresAt, _ := types.ParseDateTime(time.Unix(tokenExpiry(token), 0))
	character.Set("oauth_access_expires_at", accessExpiresAt)
	if token.RefreshToken != "" {
		character.Set("oauth_refresh_token", token.RefreshToken)
		refreshExpiresAt, _ := types.ParseDateTime(time.Unix(eveRefreshExpiry(), 0))
		character.Set("oauth_refresh_expires_at", refreshExpiresAt)
	}
	character.Set("oauth_scopes", strings.Join(claims.Scp, " "))
	lastRefreshAt, _ := types.ParseDateTime(time.Now())
	character.Set("esi_last_refresh_at", lastRefreshAt)
	character.Set("esi_last_error", "")
	character.Set("esi_token_valid", true)

	if saveErr := p.App.Save(character); saveErr != nil {
		return AuthTokens{}, saveErr
	}

	if user != nil && isMain {
		if updateErr := p.updateUserFromCharacter(user, character); updateErr != nil {
			return AuthTokens{}, updateErr
		}
	}

	return AuthTokens{
		AccessToken:   token.AccessToken,
		AccessExpiry:  time.Unix(tokenExpiry(token), 0),
		RefreshToken:  token.RefreshToken,
		RefreshExpiry: time.Unix(eveRefreshExpiry(), 0),
	}, nil
}

func (p *EVEProvider) RefreshCharacter(ctx context.Context, user *core.Record, character *core.Record) (AuthTokens, error) {
	return p.refreshCharacter(ctx, user, character)
}

func (p *EVEProvider) allowedAccess(corpID int, allianceID int) (bool, error) {
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

func (p *EVEProvider) hasAllowlist(collection string) bool {
	records, recordsErr := p.App.FindRecordsByFilter(collection, "", "", 1, 0, nil)
	return recordsErr == nil && len(records) > 0
}

func (p *EVEProvider) allowedID(collection string, id int) (bool, error) {
	if id == 0 {
		return false, nil
	}
	records, recordsErr := p.App.FindRecordsByFilter(collection, "eve_id = {:id}", "", 1, 0, map[string]any{"id": id})
	if recordsErr != nil {
		return false, recordsErr
	}
	return len(records) > 0, nil
}

func (p *EVEProvider) findOrCreateUser(characterID int) (*core.Record, error) {
	coll, findErr := p.App.FindCollectionByNameOrId(store.CollectionUsers)
	if findErr != nil {
		return nil, findErr
	}
	key := strconv.Itoa(characterID)
	records, recordsErr := p.App.FindRecordsByFilter(
		coll.Name,
		"auth_provider = {:provider} && auth_provider_sub = {:sub}",
		"",
		1,
		0,
		map[string]any{"provider": p.Name(), "sub": key},
	)
	if recordsErr != nil {
		return nil, recordsErr
	}
	if len(records) > 0 {
		return records[0], nil
	}
	record := core.NewRecord(coll)
	record.Set("auth_provider", p.Name())
	record.Set("auth_provider_sub", key)
	record.Set("sub", key)
	record.SetEmail(fmt.Sprintf("eve-%d@auth.invalid", characterID))
	record.SetRandomPassword()
	record.Set("created_at", time.Now())
	record.Set("access_level", "user")
	if saveErr := p.App.Save(record); saveErr != nil {
		return nil, saveErr
	}
	if _, tokenErr := intel.NewIntelService(p.App).GetOrCreateUploaderToken(record.Id); tokenErr != nil {
		return nil, tokenErr
	}
	return record, nil
}

func eveRefreshExpiry() int64 {
	return time.Now().Add(180 * 24 * time.Hour).Unix()
}
