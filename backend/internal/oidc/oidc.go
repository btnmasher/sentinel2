package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"sentinel2/internal/config"
	"sentinel2/internal/shared/collections"
)

type Client struct {
	Config       *config.Config
	Provider     *oidc.Provider
	Verifier     *oidc.IDTokenVerifier
	OAuth2Config oauth2.Config
}

const (
	randomStateBytes = 32
	minJWTParts      = 2
)

func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("missing oidc config")
	}
	provider, providerErr := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if providerErr != nil {
		return nil, providerErr
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.OIDCAuthURL,
			TokenURL: cfg.OIDCTokenURL,
		},
		RedirectURL: "",
		Scopes:      normalizeScopes(cfg.OIDCScopes),
	}

	return &Client{
		Config:       cfg,
		Provider:     provider,
		Verifier:     provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID}),
		OAuth2Config: oauth2Config,
	}, nil
}

func normalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return []string{"openid"}
	}
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}

	if len(out) == 0 {
		return []string{"openid"}
	}
	return out
}

func RandomState() (string, error) {
	b := make([]byte, randomStateBytes)
	if _, readErr := rand.Read(b); readErr != nil {
		return "", readErr
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func RandomNonce() (string, error) {
	return RandomState()
}

func VerifyRoles(accessToken string, required []string) (bool, error) {
	if len(required) == 0 {
		return true, nil
	}

	claims, claimsErr := parseUnverifiedJWT(accessToken)
	if claimsErr != nil {
		return false, claimsErr
	}

	roleSet := roleSetFromClaims(claims)
	if len(roleSet) == 0 {
		return false, nil
	}
	return hasRequiredRole(roleSet, required), nil
}

func roleSetFromClaims(claims map[string]any) map[string]struct{} {
	realm, ok := claims["realm_access"].(map[string]any)
	if !ok {
		return nil
	}
	roles, ok := realm["roles"].([]any)
	if !ok {
		return nil
	}
	roleNames := make([]string, 0, len(roles))
	for _, role := range roles {
		s, ok := role.(string)
		if !ok || s == "" {
			continue
		}
		roleNames = append(roleNames, s)
	}
	return collections.ToSet(roleNames)
}

func hasRequiredRole(roleSet map[string]struct{}, required []string) bool {
	for _, req := range required {
		if _, ok := roleSet[req]; ok {
			return true
		}
	}
	return false
}

func parseUnverifiedJWT(token string) (map[string]any, error) {
	parts := split(token, '.')
	if len(parts) < minJWTParts {
		return nil, errors.New("invalid jwt")
	}
	payload, decodeErr := base64.RawURLEncoding.DecodeString(parts[1])
	if decodeErr != nil {
		return nil, decodeErr
	}
	return decodeJSON(payload)
}

func decodeJSON(payload []byte) (map[string]any, error) {
	out := map[string]any{}

	if decodeErr := json.Unmarshal(payload, &out); decodeErr != nil {
		return nil, decodeErr
	}
	return out, nil
}

func split(value string, sep byte) []string {
	out := []string{}
	current := ""
	for i := range len(value) {
		if value[i] == sep {
			out = append(out, current)
			current = ""
			continue
		}
		current += string(value[i])
	}
	out = append(out, current)
	return out
}

var _ = time.Now
