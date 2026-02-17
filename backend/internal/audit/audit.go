package audit

import (
	"strings"

	"github.com/pocketbase/dbx"
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

type Event struct {
	Action                 string
	Summary                string
	TargetUserID           string
	TargetUserName         string
	TargetCharacter        *core.Record
	TargetType             string
	TargetID               string
	TargetLabel            string
	TargetMeta             any
	ActorID                string
	ActorDisplayName       string
	ActorRecord            *core.Record
	ResolveTargetCharacter bool
}

func (s *Service) LogRequest(c *core.RequestEvent, event Event) {
	if c == nil {
		s.LogEvent(event)
		return
	}
	if event.ActorRecord == nil {
		value := c.Get("admin_record")
		if admin, ok := value.(*core.Record); ok {
			event.ActorRecord = admin
		} else if c.Auth != nil {
			event.ActorRecord = c.Auth
		}
	}
	s.LogEvent(event)
}

func (s *Service) LogEvent(event Event) {
	if s.App == nil {
		return
	}
	collection, collectionErr := s.App.FindCollectionByNameOrId(store.CollectionAuditLogs)
	if collectionErr != nil {
		logging.New(s.App).
			WithFields(logging.Fields{
				"action": event.Action,
				"user":   event.TargetUserID,
			}).
			WithErr(collectionErr).
			Warn("audit log collection lookup failed")
		return
	}
	record := core.NewRecord(collection)
	record.Set("action", event.Action)
	record.Set("summary", event.Summary)
	record.Set("target_user_id", strings.TrimSpace(event.TargetUserID))

	targetCharacter := event.TargetCharacter
	if targetCharacter == nil && event.ResolveTargetCharacter && event.TargetUserID != "" {
		targetCharacter = s.resolveTargetCharacter(event.TargetUserID)
	}
	if targetCharacter != nil {
		record.Set("target_character_id", targetCharacter.GetInt("eve_character_id"))
		record.Set("target_character_name", targetCharacter.GetString("eve_character_name"))
	}

	targetUserName := strings.TrimSpace(event.TargetUserName)
	if targetUserName != "" {
		record.Set("target_user_name", targetUserName)
	} else if event.TargetUserID != "" {
		user, userErr := s.App.FindRecordById(store.CollectionUsers, event.TargetUserID)
		if userErr == nil {
			record.Set("target_user_name", user.GetString("eve_character_name"))
		}
	}

	targetType := strings.TrimSpace(event.TargetType)
	targetID := strings.TrimSpace(event.TargetID)
	targetLabel := strings.TrimSpace(event.TargetLabel)
	targetMeta := event.TargetMeta

	targetType, targetID, targetLabel, targetMeta = normalizeTargetFields(
		event,
		targetCharacter,
		record.GetString("target_user_name"),
		targetType,
		targetID,
		targetLabel,
		targetMeta,
	)

	if targetType != "" {
		record.Set("target_type", targetType)
	}
	if targetID != "" {
		record.Set("target_id", targetID)
	}
	if targetLabel != "" {
		record.Set("target_label", targetLabel)
	}
	if targetMeta != nil {
		record.Set("target_meta", targetMeta)
	}

	actorID := strings.TrimSpace(event.ActorID)
	actorDisplayName := strings.TrimSpace(event.ActorDisplayName)
	actorID, actorDisplayName = normalizeActorFields(actorID, actorDisplayName, event.ActorRecord)
	if actorID != "" {
		record.Set("actor_id", actorID)
	}
	if actorDisplayName != "" {
		record.Set("actor_display_name", actorDisplayName)
	}

	if saveErr := s.App.Save(record); saveErr != nil {
		logging.New(s.App).
			WithFields(logging.Fields{
				"action": event.Action,
				"user":   event.TargetUserID,
			}).
			WithErr(saveErr).
			Warn("audit log save failed")
	}
}

func (s *Service) resolveTargetCharacter(userID string) *core.Record {
	if strings.TrimSpace(userID) == "" {
		return nil
	}
	records, err := s.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user}",
		"-is_main",
		1,
		0,
		dbx.Params{"user": userID},
	)
	if err != nil || len(records) == 0 {
		return nil
	}
	return records[0]
}

func normalizeTargetFields(
	event Event,
	targetCharacter *core.Record,
	targetUserName string,
	targetType string,
	targetID string,
	targetLabel string,
	targetMeta any,
) (string, string, string, any) {
	if targetType == "" {
		if targetCharacter != nil {
			targetType = TargetTypeCharacter
			if targetID == "" {
				targetID = targetCharacter.Id
			}
			if targetLabel == "" {
				targetLabel = targetCharacter.GetString("eve_character_name")
			}
			if targetMeta == nil {
				targetMeta = map[string]any{
					"eve_character_id": targetCharacter.GetInt("eve_character_id"),
				}
			}
		} else if event.TargetUserID != "" {
			targetType = TargetTypeUser
			if targetID == "" {
				targetID = event.TargetUserID
			}
			if targetLabel == "" {
				targetLabel = targetUserName
			}
		}
	}
	return targetType, targetID, targetLabel, targetMeta
}

func normalizeActorFields(actorID string, actorDisplayName string, actorRecord *core.Record) (string, string) {
	if actorRecord != nil {
		if actorID == "" {
			actorID = actorRecord.Id
		}
		if actorDisplayName == "" {
			actorDisplayName = actorRecord.GetString("eve_character_name")
			if actorDisplayName == "" {
				actorDisplayName = actorRecord.Id
			}
		}
	}
	return actorID, actorDisplayName
}
