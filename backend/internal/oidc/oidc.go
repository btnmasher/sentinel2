package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"sentinel2/internal/config"
)

type Client struct {
	Config       config.Config
	Provider     *oidc.Provider
	Verifier     *oidc.IDTokenVerifier
	OAuth2Config oauth2.Config
}

func New(ctx context.Context, cfg config.Config) (*Client, error) {
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
		Scopes:      splitScopes(cfg.OIDCScopes),
	}

	return &Client{
		Config:       cfg,
		Provider:     provider,
		Verifier:     provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID}),
		OAuth2Config: oauth2Config,
	}, nil
}

func splitScopes(value string) []string {
	if value == "" {
		return []string{"openid"}
	}
	out := []string{}
	current := ""
	for _, r := range value {
		if r == ' ' {
			if current != "" {
				out = append(out, current)
				current = ""
			}
			continue
		}
		current += string(r)
	}
	if current != "" {
		out = append(out, current)
	}
	return out
}

func RandomState() (string, error) {
	b := make([]byte, 32)
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

	rolesAny := false
	if realm, ok := claims["realm_access"].(map[string]interface{}); ok {
		if roles, ok := realm["roles"].([]interface{}); ok {
			roleSet := map[string]struct{}{}
			for _, r := range roles {
				if s, ok := r.(string); ok {
					roleSet[s] = struct{}{}
				}
			}
			for _, req := range required {
				if _, ok := roleSet[req]; ok {
					rolesAny = true
					break
				}
			}
		}
	}

	return rolesAny, nil
}

func parseUnverifiedJWT(token string) (map[string]interface{}, error) {
	parts := split(token, '.')
	if len(parts) < 2 {
		return nil, errors.New("invalid jwt")
	}
	payload, decodeErr := base64.RawURLEncoding.DecodeString(parts[1])
	if decodeErr != nil {
		return nil, decodeErr
	}
	return decodeJSON(payload)
}

func decodeJSON(payload []byte) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	if decodeErr := json.Unmarshal(payload, &out); decodeErr != nil {
		return nil, decodeErr
	}
	return out, nil
}

func split(value string, sep byte) []string {
	out := []string{}
	current := ""
	for i := 0; i < len(value); i++ {
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
