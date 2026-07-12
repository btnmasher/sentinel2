package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"sentinel2/internal/esi"
)

const testAuthRequestTimeout = 30 * time.Second

// TestAuthClient wraps the standard OAuth2 client for the TestAuth provider.
// It uses RFC 8414 discovery to locate the authorization and token endpoints.
type TestAuthClient struct {
	// IssuerURL is the provider base URL used for API requests and redirect construction.
	IssuerURL string

	// Config is the standard oauth2.Config for token operations.
	Config oauth2.Config

	// Metadata stores the RFC 8414 discovery document for the provider.
	Metadata oauthAuthorizationServerMetadata
}

// UserInfo represents the response from the external auth platform's /oauth/api/me endpoint.
// Profile, groups, and permissions are derived live from Core/Groups and are not embedded in the token.
type UserInfo struct {
	Sub              string            `json:"sub"`
	ClientID         string            `json:"clientId"`
	Scope            []string          `json:"scope"`
	MainCharacterID  *int64            `json:"mainCharacterId,omitempty"`
	IsAdmin          bool              `json:"isAdmin"`
	Groups           []string          `json:"groups,omitempty"`
	Characters       []CharacterInfo   `json:"characters,omitempty"`
	GroupMemberships []GroupMembership `json:"groupMemberships,omitempty"`
	PermissionURNs   []string          `json:"permissionUrns,omitempty"`
}

// CharacterInfo represents a linked character from the external auth platform Core.
type CharacterInfo struct {
	CharacterID   int64  `json:"characterId"`
	CharacterName string `json:"characterName"`
	IsPrimary     bool   `json:"isPrimary"`
	HasValidToken bool   `json:"hasValidToken"`
}

// GroupMembership represents a group membership from the external auth platform Core.
type GroupMembership struct {
	GroupID         string `json:"groupId"`
	GroupName       string `json:"groupName"`
	MembershipLevel string `json:"membershipLevel"`
	JoinedAt        string `json:"joinedAt"`
}

type rawUserInfo struct {
	Sub              string               `json:"sub"`
	ClientID         string               `json:"clientId"`
	Scope            []string             `json:"scope"`
	MainCharacterID  string               `json:"mainCharacterId"`
	IsAdmin          bool                 `json:"isAdmin"`
	Groups           []string             `json:"groups,omitempty"`
	Characters       []rawCharacterInfo   `json:"characters,omitempty"`
	GroupMemberships []rawGroupMembership `json:"groupMemberships,omitempty"`
	PermissionURNs   []string             `json:"permissionUrns,omitempty"`
}

type rawCharacterInfo struct {
	CharacterID   string `json:"characterId"`
	CharacterName string `json:"characterName"`
	IsPrimary     bool   `json:"isPrimary"`
	HasValidToken bool   `json:"hasValidToken"`
}

type rawGroupMembership struct {
	GroupID         string `json:"groupId"`
	GroupName       string `json:"groupName"`
	MembershipLevel string `json:"membershipLevel"`
	JoinedAt        string `json:"joinedAt"`
}

// EsiProxyResponse represents the response from the external auth platform's ESI proxy.
type EsiProxyResponse struct {
	Status     int         `json:"status"`
	StatusText string      `json:"statusText"`
	Headers    [][2]string `json:"headers"`
	Body       string      `json:"body"`
}

type oauthAuthorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	ScopesSupported                   []string `json:"scopes_supported,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
}

// NewTestAuthClient creates an OAuth2 client for the TestAuth provider using standard libraries.
// It performs RFC 8414 discovery to locate the authorization and token endpoints.
func NewTestAuthClient(ctx context.Context, baseURL, clientID, clientSecret, redirectURI string, scopes []string) (*TestAuthClient, error) {
	issuerURL := strings.TrimRight(baseURL, "/")
	metadataURL := issuerURL + "/.well-known/oauth-authorization-server"
	metadata, metadataErr := fetchOAuthAuthorizationServerMetadata(ctx, metadataURL)
	if metadataErr != nil {
		return nil, fmt.Errorf("%s for %s: %w", ErrTestAuthDiscovery.Error(), metadataURL, metadataErr)
	}

	authStyle := oauth2.AuthStyleAutoDetect
	if clientSecret != "" {
		authStyle = oauth2.AuthStyleInHeader
	}

	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   metadata.AuthorizationEndpoint,
			TokenURL:  metadata.TokenEndpoint,
			AuthStyle: authStyle,
		},
	}

	if metadata.Issuer != "" {
		issuerURL = strings.TrimRight(metadata.Issuer, "/")
	}

	return &TestAuthClient{
		IssuerURL: issuerURL,
		Config:    config,
		Metadata:  metadata,
	}, nil
}

func fetchOAuthAuthorizationServerMetadata(ctx context.Context, metadataURL string) (oauthAuthorizationServerMetadata, error) {
	if strings.TrimSpace(metadataURL) == "" {
		return oauthAuthorizationServerMetadata{}, fmt.Errorf("missing metadata url")
	}

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, http.NoBody)
	if reqErr != nil {
		return oauthAuthorizationServerMetadata{}, reqErr
	}

	client := &http.Client{Timeout: testAuthRequestTimeout}
	resp, httpErr := client.Do(req)
	if httpErr != nil {
		return oauthAuthorizationServerMetadata{}, httpErr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return oauthAuthorizationServerMetadata{}, fmt.Errorf("metadata status %d: %s", resp.StatusCode, string(body))
	}

	var metadata oauthAuthorizationServerMetadata
	if decodeErr := json.NewDecoder(resp.Body).Decode(&metadata); decodeErr != nil {
		return oauthAuthorizationServerMetadata{}, decodeErr
	}

	if strings.TrimSpace(metadata.AuthorizationEndpoint) == "" {
		return oauthAuthorizationServerMetadata{}, fmt.Errorf("missing authorization_endpoint")
	}
	if strings.TrimSpace(metadata.TokenEndpoint) == "" {
		return oauthAuthorizationServerMetadata{}, fmt.Errorf("missing token_endpoint")
	}

	return metadata, nil
}

// ExchangeCode exchanges an authorization code for tokens using golang.org/x/oauth2.
// This is RFC 6749 compliant and does not use custom token exchange logic.
func (c *TestAuthClient) ExchangeCode(ctx context.Context, code, redirectURI string) (*oauth2.Token, error) {
	c.Config.RedirectURL = redirectURI
	token, exchangeErr := c.Config.Exchange(ctx, code)
	if exchangeErr != nil {
		return nil, fmt.Errorf("%s: %w", ErrTestAuthToken.Error(), exchangeErr)
	}
	return token, nil
}

// RefreshToken refreshes an access token using golang.org/x/oauth2.
// This is RFC 6749 compliant and does not use custom refresh logic.
func (c *TestAuthClient) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	refreshed, refreshErr := c.Config.TokenSource(ctx, token).Token()
	if refreshErr != nil {
		return nil, fmt.Errorf("%s: %w", ErrTestAuthRefresh.Error(), refreshErr)
	}
	return refreshed, nil
}

// GetUserInfo fetches user details from the external auth platform's /oauth/api/me endpoint.
// Profile, groups, and permissions are derived live from Core/Groups APIs and are not embedded in the token.
func (c *TestAuthClient) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	meURL := c.IssuerURL + "/oauth/api/me"

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, meURL, http.NoBody)
	if reqErr != nil {
		return nil, fmt.Errorf("create userinfo request: %w", reqErr)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: testAuthRequestTimeout}
	resp, httpErr := client.Do(req)
	if httpErr != nil {
		return nil, fmt.Errorf("%s: %w", ErrTestAuthUserInfo.Error(), httpErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrTestAuthUserInfo, resp.StatusCode, string(body))
	}

	var raw rawUserInfo
	decodeErr := json.NewDecoder(resp.Body).Decode(&raw)
	if decodeErr != nil {
		return nil, fmt.Errorf("%s: %w", ErrTestAuthUserInfo.Error(), decodeErr)
	}

	info, translateErr := translateUserInfo(&raw)
	if translateErr != nil {
		return nil, fmt.Errorf("%s: %w", ErrTestAuthUserInfo.Error(), translateErr)
	}

	return info, nil
}

// ProxyEsiRequest proxies a request to the external auth platform's ESI proxy at /oauth/api/esi-proxy/*.
// The auth platform handles scope enforcement, linked-character ownership validation,
// character path matching, method-aware write allowlisting, rate limiting, and response caching.
func (c *TestAuthClient) ProxyEsiRequest(ctx context.Context, accessToken, characterID, method, path string, body io.Reader) (*EsiProxyResponse, error) {
	resp, httpErr := esi.DoTestAuthProxyRequest(ctx, c.IssuerURL, accessToken, characterID, method, path, body, nil)
	if httpErr != nil {
		return nil, fmt.Errorf("%s: %w", ErrTestAuthEsiProxy.Error(), httpErr)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("%s: read response: %w", ErrTestAuthEsiProxy.Error(), readErr)
	}

	var headers [][2]string
	if resp.Header != nil {
		for k, vals := range resp.Header {
			for _, v := range vals {
				headers = append(headers, [2]string{k, v})
			}
		}
	}

	return &EsiProxyResponse{
		Status:     resp.StatusCode,
		StatusText: resp.Status,
		Headers:    headers,
		Body:       string(respBody),
	}, nil
}

func translateUserInfo(raw *rawUserInfo) (*UserInfo, error) {
	info := &UserInfo{
		Sub:            raw.Sub,
		ClientID:       raw.ClientID,
		Scope:          append([]string(nil), raw.Scope...),
		IsAdmin:        raw.IsAdmin,
		Groups:         append([]string(nil), raw.Groups...),
		PermissionURNs: append([]string(nil), raw.PermissionURNs...),
	}

	if trimmed := strings.TrimSpace(raw.MainCharacterID); trimmed != "" {
		mainID, parseErr := strconv.ParseInt(trimmed, 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("parse main character id: %w", parseErr)
		}
		info.MainCharacterID = &mainID
	}

	if raw.Characters != nil {
		info.Characters = make([]CharacterInfo, 0, len(raw.Characters))
		for _, character := range raw.Characters {
			charID, parseErr := strconv.ParseInt(strings.TrimSpace(character.CharacterID), 10, 64)
			if parseErr != nil {
				return nil, fmt.Errorf("parse character id %q: %w", character.CharacterID, parseErr)
			}
			info.Characters = append(info.Characters, CharacterInfo{
				CharacterID:   charID,
				CharacterName: character.CharacterName,
				IsPrimary:     character.IsPrimary,
				HasValidToken: character.HasValidToken,
			})
		}
	}

	if raw.GroupMemberships != nil {
		info.GroupMemberships = make([]GroupMembership, 0, len(raw.GroupMemberships))
		for _, membership := range raw.GroupMemberships {
			info.GroupMemberships = append(info.GroupMemberships, GroupMembership(membership))
		}
	}

	return info, nil
}

// TimeoutContext returns a context with a timeout suitable for auth platform API calls.
// This allows graceful shutdown to propagate to all auth platform requests.
func (c *TestAuthClient) TimeoutContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// Issuer returns the auth platform issuer URL for use in redirect URIs and other config.
func (c *TestAuthClient) Issuer() string {
	return c.IssuerURL
}

// AuthorizationURL returns the full authorization URL for the given state and scopes.
func (c *TestAuthClient) AuthorizationURL(state string, opts ...oauth2.AuthCodeOption) string {
	return c.Config.AuthCodeURL(state, opts...)
}

// RequiredScopes returns the scopes configured for this client.
func (c *TestAuthClient) RequiredScopes() []string {
	return c.Config.Scopes
}

// RedirectURL returns the configured redirect URL.
func (c *TestAuthClient) RedirectURL() string {
	return c.Config.RedirectURL
}

// SetRedirectURL updates the redirect URL used before token exchange.
func (c *TestAuthClient) SetRedirectURL(url string) {
	c.Config.RedirectURL = url
}
