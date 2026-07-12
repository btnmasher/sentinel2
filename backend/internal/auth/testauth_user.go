package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/oauth2"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

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
	record.SetEmail(fmt.Sprintf("%s%s@%s", testAuthEmailPrefix, base64.RawURLEncoding.EncodeToString([]byte(sub)), testAuthEmailDomain))
	record.SetRandomPassword()
	record.Set("created_at", time.Now())
	record.Set("access_level", accessLevelUser)
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

func refreshExpiry(token *oauth2.Token) int64 {
	value := token.Extra("refresh_expires_in")
	if v, ok := value.(float64); ok {
		return time.Now().Add(time.Duration(v) * time.Second).Unix()
	}
	return time.Now().Add(30 * 24 * time.Hour).Unix()
}

func generateState() (string, error) {
	b := make([]byte, testAuthStateBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
