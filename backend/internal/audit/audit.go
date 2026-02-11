package audit

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

type Service struct {
	App *pocketbase.PocketBase
}

func New(app *pocketbase.PocketBase) *Service {
	return &Service{App: app}
}

func (s *Service) Log(action, summary, userID, targetUserName string, character *core.Record) {
	if s.App == nil {
		return
	}
	collection, collectionErr := s.App.FindCollectionByNameOrId(store.CollectionAuditLogs)
	if collectionErr != nil {
		logging.New(s.App).
			WithFields(logging.Fields{
				"action": action,
				"user":   userID,
			}).
			WithErr(collectionErr).
			Warn("audit log collection lookup failed")
		return
	}
	record := core.NewRecord(collection)
	record.Set("action", action)
	record.Set("summary", summary)
	record.Set("target_user_id", userID)
	if character != nil {
		record.Set("target_character_id", character.GetInt("eve_character_id"))
		record.Set("target_character_name", character.GetString("eve_character_name"))
	}
	if targetUserName != "" {
		record.Set("target_user_name", targetUserName)
	} else if userID != "" {
		user, userErr := s.App.FindRecordById(store.CollectionUsers, userID)
		if userErr == nil {
			record.Set("target_user_name", user.GetString("eve_character_name"))
		}
	}
	if saveErr := s.App.Save(record); saveErr != nil {
		logging.New(s.App).
			WithFields(logging.Fields{
				"action": action,
				"user":   userID,
			}).
			WithErr(saveErr).
			Warn("audit log save failed")
	}
}
