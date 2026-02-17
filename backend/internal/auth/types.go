package auth

import "time"

type AuthTokens struct {
	AccessToken   string
	AccessExpiry  time.Time
	RefreshToken  string
	RefreshExpiry time.Time
	IDToken       string
}

type AuthResult struct {
	Provider      string
	UserID        string
	CharacterID   int
	CharacterName string
	Tokens        AuthTokens
}
