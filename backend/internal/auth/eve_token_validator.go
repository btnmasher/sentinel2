package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

const (
	eveMetadataURLDefault = "https://login.eveonline.com/.well-known/oauth-authorization-server"
	eveAudienceStatic     = "EVE Online"
	eveRequestTimeout     = 5 * time.Second
)

var eveAcceptedIssuers = map[string]struct{}{
	"https://login.eveonline.com":  {},
	"https://login.eveonline.com/": {},
	"login.eveonline.com":          {},
}

type EVETokenValidator interface {
	ValidateAccessToken(ctx context.Context, accessToken string) (*eveTokenClaims, error)
}

type eveTokenValidator struct {
	clientID string
	keyfunc  keyfunc.Keyfunc
}

type eveTokenJWTClaims struct {
	jwt.RegisteredClaims

	Name string   `json:"name"`
	Scp  []string `json:"scp"`
}

type eveOAuthMetadata struct {
	JWKSURI string `json:"jwks_uri"`
}

func NewEVETokenValidator(clientID string) (*eveTokenValidator, error) {
	trimmedClientID := strings.TrimSpace(clientID)
	if trimmedClientID == "" {
		return nil, errors.New("missing EVE OAuth client ID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), eveRequestTimeout)
	defer cancel()

	jwksURI, metadataErr := fetchEVEJWKSURI(ctx, eveMetadataURLDefault, &http.Client{
		Timeout: eveRequestTimeout,
	})
	if metadataErr != nil {
		return nil, metadataErr
	}
	k, keyfuncErr := keyfunc.NewDefaultCtx(ctx, []string{jwksURI})
	if keyfuncErr != nil {
		return nil, keyfuncErr
	}
	return newEVETokenValidatorForTests(trimmedClientID, k)
}

func newEVETokenValidatorForTests(clientID string, k keyfunc.Keyfunc) (*eveTokenValidator, error) {
	trimmedClientID := strings.TrimSpace(clientID)
	if trimmedClientID == "" {
		return nil, errors.New("missing EVE OAuth client ID")
	}
	if k == nil {
		return nil, errors.New("missing keyfunc")
	}
	return &eveTokenValidator{
		clientID: trimmedClientID,
		keyfunc:  k,
	}, nil
}

func (v *eveTokenValidator) ValidateAccessToken(ctx context.Context, accessToken string) (*eveTokenClaims, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("missing access token")
	}
	claims := &eveTokenJWTClaims{}
	_, parseErr := jwt.ParseWithClaims(
		accessToken,
		claims,
		v.keyfunc.KeyfuncCtx(ctx),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
	)
	if parseErr != nil {
		return nil, parseErr
	}

	if claims.ExpiresAt == nil {
		return nil, errors.New("missing exp")
	}
	if _, ok := eveAcceptedIssuers[claims.Issuer]; !ok {
		return nil, fmt.Errorf("invalid issuer: %q", claims.Issuer)
	}
	if !audienceContains(claims.Audience, v.clientID) || !audienceContains(claims.Audience, eveAudienceStatic) {
		return nil, errors.New("invalid audience")
	}
	if _, subErr := parseCharacterID(claims.Subject); subErr != nil {
		return nil, subErr
	}

	return &eveTokenClaims{
		Sub:  claims.Subject,
		Name: claims.Name,
		Scp:  claims.Scp,
	}, nil
}

func fetchEVEJWKSURI(ctx context.Context, metadataURL string, httpClient *http.Client) (string, error) {
	if strings.TrimSpace(metadataURL) == "" {
		return "", errors.New("missing metadata url")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: eveRequestTimeout}
	}

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, http.NoBody)
	if reqErr != nil {
		return "", reqErr
	}
	resp, respErr := httpClient.Do(req)
	if respErr != nil {
		return "", respErr
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("metadata status %d", resp.StatusCode)
	}

	var metadata eveOAuthMetadata
	if decodeErr := json.NewDecoder(resp.Body).Decode(&metadata); decodeErr != nil {
		return "", decodeErr
	}
	if strings.TrimSpace(metadata.JWKSURI) == "" {
		return "", errors.New("missing jwks_uri")
	}
	return metadata.JWKSURI, nil
}

func audienceContains(audience jwt.ClaimStrings, target string) bool {
	return slices.Contains(audience, target)
}
