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

func (s *Service) LogRequest(c *core.RequestEvent, event *Event) {
	if event == nil {
		return
	}
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

func (s *Service) LogEvent(event *Event) {
	if event == nil {
		return
	}
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

	targetCharacter := s.resolveEventTargetCharacter(event)
	applyTargetCharacterRecord(record, targetCharacter)
	targetUserName := s.resolveTargetUserName(event)
	if targetUserName != "" {
		record.Set("target_user_name", targetUserName)
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

func (s *Service) resolveEventTargetCharacter(event *Event) *core.Record {
	if event == nil {
		return nil
	}
	targetCharacter := event.TargetCharacter
	if targetCharacter == nil && event.ResolveTargetCharacter && event.TargetUserID != "" {
		targetCharacter = s.resolveTargetCharacter(event.TargetUserID)
	}
	return targetCharacter
}

func applyTargetCharacterRecord(record, targetCharacter *core.Record) {
	if record == nil || targetCharacter == nil {
		return
	}
	record.Set("target_character_id", targetCharacter.GetInt("eve_character_id"))
	record.Set("target_character_name", targetCharacter.GetString("eve_character_name"))
}

func (s *Service) resolveTargetUserName(event *Event) string {
	if event == nil {
		return ""
	}
	targetUserName := strings.TrimSpace(event.TargetUserName)
	if targetUserName != "" || event.TargetUserID == "" {
		return targetUserName
	}
	user, userErr := s.App.FindRecordById(store.CollectionUsers, event.TargetUserID)
	if userErr != nil {
		return ""
	}
	return user.GetString("eve_character_name")
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
	event *Event,
	targetCharacter *core.Record,
	targetUserName string,
	targetType string,
	targetID string,
	targetLabel string,
	targetMeta any,
) (normalizedType, normalizedID, normalizedLabel string, normalizedMeta any) {
	if targetType != "" {
		return targetType, targetID, targetLabel, targetMeta
	}
	if targetCharacter != nil {
		return normalizeCharacterTarget(targetCharacter, targetID, targetLabel, targetMeta)
	}
	if event != nil && event.TargetUserID != "" {
		return normalizeUserTarget(event.TargetUserID, targetUserName, targetID, targetLabel, targetMeta)
	}
	return targetType, targetID, targetLabel, targetMeta
}

func normalizeCharacterTarget(targetCharacter *core.Record, targetID, targetLabel string, targetMeta any) (normalizedType, normalizedID, normalizedLabel string, normalizedMeta any) {
	normalizedType = TargetTypeCharacter
	normalizedID = targetID
	normalizedLabel = targetLabel
	normalizedMeta = targetMeta
	if targetID == "" {
		normalizedID = targetCharacter.Id
	}
	if targetLabel == "" {
		normalizedLabel = targetCharacter.GetString("eve_character_name")
	}
	if normalizedMeta == nil {
		normalizedMeta = map[string]any{
			"eve_character_id": targetCharacter.GetInt("eve_character_id"),
		}
	}
	return normalizedType, normalizedID, normalizedLabel, normalizedMeta
}

func normalizeUserTarget(targetUserID, targetUserName, targetID, targetLabel string, targetMeta any) (normalizedType, normalizedID, normalizedLabel string, normalizedMeta any) {
	normalizedType = TargetTypeUser
	normalizedID = targetID
	normalizedLabel = targetLabel
	normalizedMeta = targetMeta
	if targetID == "" {
		normalizedID = targetUserID
	}
	if targetLabel == "" {
		normalizedLabel = targetUserName
	}
	return normalizedType, normalizedID, normalizedLabel, normalizedMeta
}

func normalizeActorFields(actorID, actorDisplayName string, actorRecord *core.Record) (normalizedID, normalizedDisplayName string) {
	if actorRecord == nil {
		return actorID, actorDisplayName
	}
	if actorID == "" {
		actorID = actorRecord.Id
	}
	if actorDisplayName == "" {
		actorDisplayName = actorRecord.GetString("eve_character_name")
		if actorDisplayName == "" {
			actorDisplayName = actorRecord.Id
		}
	}
	return actorID, actorDisplayName
}
