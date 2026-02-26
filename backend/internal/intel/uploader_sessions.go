package intel

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

const (
	UploaderSessionScopeConfig          = "uploader:config"
	UploaderRealtimeSessionTTL          = 5 * time.Minute
	UploaderRealtimeSessionRefreshAfter = 4 * time.Minute
)

type UploaderRealtimeSession struct {
	Token        string
	ExpiresAt    time.Time
	RefreshAfter time.Duration
}

func (s *IntelService) IssueUploaderRealtimeSession(userID, uploaderTokenID string) (UploaderRealtimeSession, error) {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionUploaderSessions)
	if collErr != nil {
		return UploaderRealtimeSession{}, collErr
	}

	now := time.Now().UTC()
	expiresAt := now.Add(UploaderRealtimeSessionTTL)
	expiresAtValue, _ := types.ParseDateTime(expiresAt)

	record := core.NewRecord(coll)
	record.Set("email", fmt.Sprintf("uploader-session-%s-%d@auth.invalid", userID, now.UnixNano()))
	record.SetRandomPassword()
	record.Set("verified", true)
	record.Set("user", userID)
	record.Set("uploader_token", uploaderTokenID)
	record.Set("scope", UploaderSessionScopeConfig)
	record.Set("expires_at", expiresAtValue)
	record.Set("last_seen_at", types.NowDateTime())
	if saveErr := s.App.Save(record); saveErr != nil {
		return UploaderRealtimeSession{}, saveErr
	}

	token, tokenErr := record.NewStaticAuthToken(UploaderRealtimeSessionTTL)
	if tokenErr != nil {
		return UploaderRealtimeSession{}, tokenErr
	}

	return UploaderRealtimeSession{
		Token:        token,
		ExpiresAt:    expiresAt,
		RefreshAfter: UploaderRealtimeSessionRefreshAfter,
	}, nil
}

func (s *IntelService) RefreshUploaderRealtimeSession(session *core.Record) (UploaderRealtimeSession, error) {
	if session == nil || session.Collection() == nil || session.Collection().Name != store.CollectionUploaderSessions {
		return UploaderRealtimeSession{}, ErrExpiredOrRevoked
	}
	if session.GetString("scope") != UploaderSessionScopeConfig {
		return UploaderRealtimeSession{}, ErrExpiredOrRevoked
	}

	uploaderTokenID := session.GetString("uploader_token")
	if uploaderTokenID == "" {
		return UploaderRealtimeSession{}, ErrExpiredOrRevoked
	}
	if _, tokenErr := s.ValidateUploaderTokenID(uploaderTokenID); tokenErr != nil {
		return UploaderRealtimeSession{}, tokenErr
	}

	now := time.Now().UTC()
	expiresAt := now.Add(UploaderRealtimeSessionTTL)
	expiresAtValue, _ := types.ParseDateTime(expiresAt)

	session.Set("expires_at", expiresAtValue)
	session.Set("last_seen_at", types.NowDateTime())
	if saveErr := s.App.Save(session); saveErr != nil {
		return UploaderRealtimeSession{}, saveErr
	}

	token, tokenErr := session.NewStaticAuthToken(UploaderRealtimeSessionTTL)
	if tokenErr != nil {
		return UploaderRealtimeSession{}, tokenErr
	}

	return UploaderRealtimeSession{
		Token:        token,
		ExpiresAt:    expiresAt,
		RefreshAfter: UploaderRealtimeSessionRefreshAfter,
	}, nil
}

func (s *IntelService) RevokeUploaderSessionsForUser(userID string) error {
	return s.revokeUploaderSessions(
		"user = {:user}",
		map[string]any{"user": userID},
		logging.Fields{"user_id": userID},
	)
}

func (s *IntelService) RevokeUploaderSessionsForUploaderToken(uploaderTokenID string) error {
	return s.revokeUploaderSessions(
		"uploader_token = {:uploader_token}",
		map[string]any{"uploader_token": uploaderTokenID},
		logging.Fields{"uploader_token_id": uploaderTokenID},
	)
}

func (s *IntelService) revokeUploaderSessions(filter string, params map[string]any, fields logging.Fields) error {
	log := logging.New(s.App)
	if fields != nil {
		log = log.WithFields(fields)
	}
	records, recordsErr := s.App.FindRecordsByFilter(
		store.CollectionUploaderSessions,
		filter,
		"",
		0,
		0,
		params,
	)
	if recordsErr != nil {
		return recordsErr
	}

	failed := 0
	for _, rec := range records {
		if deleteErr := s.App.Delete(rec); deleteErr != nil {
			failed++
			log.WithFields(logging.Fields{"session_id": rec.Id}).WithErr(deleteErr).Debug("uploader session delete failed")
		}
	}

	if failed > 0 {
		log.WithFields(logging.Fields{"failed": failed}).Warn("uploader session delete failures")
	}

	return nil
}
