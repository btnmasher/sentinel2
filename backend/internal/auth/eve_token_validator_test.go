package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

func TestEVETokenValidatorValidateAccessToken(t *testing.T) {
	t.Parallel()

	validator, privateKey, kid := newValidatorFixture(t)
	token := signedToken(t, privateKey, kid, jwt.MapClaims{
		"iss":  "https://login.eveonline.com",
		"sub":  "CHARACTER:EVE:4242",
		"name": "Pilot",
		"scp":  []string{"a", "b"},
		"aud":  []string{"client-123", "EVE Online"},
		"exp":  time.Now().Add(time.Hour).Unix(),
	})

	claims, err := validator.ValidateAccessToken(context.Background(), token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}
	if claims.Sub != "CHARACTER:EVE:4242" {
		t.Fatalf("claims.Sub = %q", claims.Sub)
	}
	if claims.Name != "Pilot" {
		t.Fatalf("claims.Name = %q", claims.Name)
	}
}

func TestEVETokenValidatorRejectsBadAudience(t *testing.T) {
	t.Parallel()

	validator, privateKey, kid := newValidatorFixture(t)
	token := signedToken(t, privateKey, kid, jwt.MapClaims{
		"iss": "https://login.eveonline.com",
		"sub": "CHARACTER:EVE:4242",
		"aud": []string{"client-123"},
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	_, err := validator.ValidateAccessToken(context.Background(), token)
	if err == nil || !strings.Contains(err.Error(), "invalid audience") {
		t.Fatalf("ValidateAccessToken() error = %v, want invalid audience", err)
	}
}

func TestEVETokenValidatorRejectsBadSub(t *testing.T) {
	t.Parallel()

	validator, privateKey, kid := newValidatorFixture(t)
	token := signedToken(t, privateKey, kid, jwt.MapClaims{
		"iss": "https://login.eveonline.com",
		"sub": "CHARACTER:OTHER:4242",
		"aud": []string{"client-123", "EVE Online"},
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	_, err := validator.ValidateAccessToken(context.Background(), token)
	if err == nil || !strings.Contains(err.Error(), "invalid sub") {
		t.Fatalf("ValidateAccessToken() error = %v, want invalid sub", err)
	}
}

func newValidatorFixture(t *testing.T) (*eveTokenValidator, *rsa.PrivateKey, string) {
	t.Helper()

	privateKey, keyErr := rsa.GenerateKey(rand.Reader, 2048)
	if keyErr != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", keyErr)
	}
	kid := "fixture-key"

	rawJWKS, marshalErr := json.Marshal(map[string]any{
		"keys": []map[string]string{
			{
				"kid": kid,
				"kty": "RSA",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
			},
		},
	})
	if marshalErr != nil {
		t.Fatalf("json.Marshal() error = %v", marshalErr)
	}

	k, keyfuncErr := keyfunc.NewJWKSetJSON(rawJWKS)
	if keyfuncErr != nil {
		t.Fatalf("keyfunc.NewJWKSetJSON() error = %v", keyfuncErr)
	}

	validator, validatorErr := newEVETokenValidatorForTests("client-123", k)
	if validatorErr != nil {
		t.Fatalf("newEVETokenValidatorForTests() error = %v", validatorErr)
	}
	return validator, privateKey, kid
}

func signedToken(t *testing.T, privateKey *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	out, signErr := token.SignedString(privateKey)
	if signErr != nil {
		t.Fatalf("SignedString() error = %v", signErr)
	}
	return out
}
