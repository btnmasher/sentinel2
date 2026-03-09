package admin

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/audit"
	"sentinel2/internal/auth"
	"sentinel2/internal/cleanup"
	"sentinel2/internal/intel"
	"sentinel2/internal/jobs"
	"sentinel2/internal/jumpbridges"
	"sentinel2/internal/logging"
	"sentinel2/internal/middleware"
	"sentinel2/internal/store"
	"sentinel2/internal/timers"
)

func NewHandler(app *pocketbase.PocketBase, refresher *auth.CharacterRefresher, provider *auth.EVEProvider, cleanupSvc *cleanup.Service, intelSvc *intel.IntelService, timerSvc *timers.Service, jumpbridgeSvc *jumpbridges.JumpbridgeService, auditSvc *audit.Service) *Handler {
	return &Handler{
		App:         app,
		Refresher:   refresher,
		Provider:    provider,
		Cleanup:     cleanupSvc,
		Intel:       intelSvc,
		Timers:      timerSvc,
		Jumpbridges: jumpbridgeSvc,
		Audit:       auditSvc,
	}
}

func (h *Handler) CancelJob(c *core.RequestEvent) error {
	jobID := strings.TrimSpace(c.Request.PathValue("id"))
	if jobID == "" {
		return router.NewBadRequestError("Missing job id.", nil)
	}

	records, recordsErr := h.App.FindRecordsByFilter(
		"job_runs",
		"job_id = {:job} && status = {:status}",
		"",
		1,
		0,
		dbx.Params{
			"job":    jobID,
			"status": jobs.StatusRunning,
		},
	)
	if recordsErr != nil || len(records) == 0 {
		return router.NewBadRequestError("Job is not running.", logging.Fields{
			"job_id": jobID,
		})
	}

	if !jobs.Cancel(jobID) {
		return router.NewBadRequestError("Job is not cancelable.", logging.Fields{
			"job_id": jobID,
		})
	}
	tracker := jobs.NewJobTracker(h.App)
	for _, record := range records {
		tracker.FinishCanceled(record, jobs.MessageCanceled)
	}

	h.logAction(
		c,
		&audit.Event{
			Action:      audit.ActionJobCancel,
			Summary:     fmt.Sprintf("Canceled job %s", jobID),
			TargetType:  audit.TargetTypeJob,
			TargetID:    jobID,
			TargetLabel: "manual_cancel",
			TargetMeta:  map[string]any{"job_id": jobID},
		},
	)

	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) CreateSiteAnnouncement(c *core.RequestEvent) error {
	payload := siteAnnouncementPayload{}

	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{
			"error": bindErr.Error(),
		})
	}

	variant, message, payloadErr := normalizeAnnouncementPayload(payload)
	if errors.Is(payloadErr, errInvalidAnnouncementVariant) {
		return router.NewBadRequestError("Invalid announcement variant.", logging.Fields{
			"variant": payload.Variant,
		})
	}

	if errors.Is(payloadErr, errAnnouncementMessageRequired) {
		return router.NewBadRequestError("Announcement message is required.", nil)
	}

	if payloadErr != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{"error": payloadErr.Error()})
	}

	collection, collectionErr := h.App.FindCollectionByNameOrId(store.CollectionSiteAnnouncements)
	if collectionErr != nil {
		return router.NewInternalServerError("Failed to load announcement collection.", logging.Fields{
			"error": collectionErr.Error(),
		})
	}

	if archiveErr := h.archiveActiveAnnouncements(); archiveErr != nil {
		return router.NewInternalServerError("Failed to archive previous announcements.", logging.Fields{
			"error": archiveErr.Error(),
		})
	}

	record := core.NewRecord(collection)
	record.Set("variant", variant)
	record.Set("message", message)
	record.Set("archived", false)
	record.Set("published_at", types.NowDateTime())
	if saveErr := h.App.Save(record); saveErr != nil {
		return router.NewInternalServerError("Failed to save announcement.", logging.Fields{
			"error": saveErr.Error(),
		})
	}

	h.logAction(
		c,
		&audit.Event{
			Action:      audit.ActionAnnouncementCreate,
			Summary:     fmt.Sprintf("Published %s announcement", variant),
			TargetType:  audit.TargetTypeAnnouncement,
			TargetID:    record.Id,
			TargetLabel: variant,
			TargetMeta: map[string]any{
				"variant": variant,
			},
		},
	)

	return c.JSON(http.StatusOK, map[string]any{
		"id":      record.Id,
		"variant": variant,
	})
}

func (h *Handler) ArchiveLatestSiteAnnouncement(c *core.RequestEvent) error {
	latest, err := h.findLatestActiveAnnouncement()
	if err != nil {
		return router.NewInternalServerError("Failed to load latest announcement.", logging.Fields{
			"error": err.Error(),
		})
	}

	if latest == nil {
		return c.JSON(http.StatusOK, map[string]any{"archived": false})
	}
	latest.Set("archived", true)
	if saveErr := h.App.Save(latest); saveErr != nil {
		return router.NewInternalServerError("Failed to archive announcement.", logging.Fields{
			"error": saveErr.Error(),
		})
	}
	h.logAction(
		c,
		&audit.Event{
			Action:      audit.ActionAnnouncementArchiveLatest,
			Summary:     "Archived latest announcement",
			TargetType:  audit.TargetTypeAnnouncement,
			TargetID:    latest.Id,
			TargetLabel: latest.GetString("variant"),
			TargetMeta: map[string]any{
				"variant": latest.GetString("variant"),
			},
		},
	)
	return c.JSON(http.StatusOK, map[string]any{
		"archived": true,
		"id":       latest.Id,
	})
}

func (h *Handler) archiveActiveAnnouncements() error {
	records, err := h.App.FindRecordsByFilter(
		store.CollectionSiteAnnouncements,
		"(archived = false || archived = null)",
		"",
		0,
		0,
		nil,
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		record.Set("archived", true)
		if saveErr := h.App.Save(record); saveErr != nil {
			return saveErr
		}
	}
	return nil
}

func (h *Handler) findLatestActiveAnnouncement() (*core.Record, error) {
	records, err := h.App.FindRecordsByFilter(
		store.CollectionSiteAnnouncements,
		"(archived = false || archived = null)",
		"-created",
		1,
		0,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

func (h *Handler) logAction(c *core.RequestEvent, event *audit.Event) {
	if event == nil {
		return
	}
	explicitTarget := strings.TrimSpace(event.TargetType) != "" ||
		strings.TrimSpace(event.TargetID) != "" ||
		strings.TrimSpace(event.TargetLabel) != ""
	event.ResolveTargetCharacter = event.ResolveTargetCharacter || (event.TargetUserID != "" && event.TargetCharacter == nil && !explicitTarget)
	if h.Audit == nil {
		return
	}
	h.Audit.LogRequest(c, event)
}

func (h *Handler) applyAccountTarget(event *audit.Event, userID, fallbackMainName string) {
	if event == nil {
		return
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	mainName := strings.TrimSpace(fallbackMainName)
	if mainName == "" {
		if main, err := h.findMainCharacter(userID); err == nil && main != nil {
			mainName = strings.TrimSpace(main.GetString("eve_character_name"))
		}
	}

	if mainName == "" {
		if user, err := h.App.FindRecordById(store.CollectionUsers, userID); err == nil {
			mainName = strings.TrimSpace(user.GetString("eve_character_name"))
		}
	}

	if mainName == "" {
		mainName = "Unknown"
	}
	event.TargetUserID = userID
	if strings.TrimSpace(event.TargetUserName) == "" {
		event.TargetUserName = mainName
	}
	event.TargetType = audit.TargetTypeUser
	event.TargetID = userID
	event.TargetLabel = fmt.Sprintf("%s (Main: %s)", userID, mainName)
}

func (h *Handler) UserDetails(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	record, recordErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if recordErr != nil {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}

	characters, _ := h.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user}",
		"-is_main",
		0,
		0, dbx.Params{"user": userID},
	)

	response := userResponse{
		UserID:      record.Id,
		AccessLevel: record.GetString("access_level"),
		Characters:  []characterResponse{},
	}

	if h.Intel != nil {
		if tokenValid, tokenErr := h.Intel.HasValidUploaderToken(record.Id); tokenErr == nil {
			response.UploaderTokenValid = tokenValid
		}
	}

	if revokedAt := record.GetDateTime("session_revoked_at").Time(); !revokedAt.IsZero() {
		response.SessionRevokedAt = revokedAt.Format(time.RFC3339)
	}

	for _, charRecord := range characters {
		response.Characters = append(response.Characters, newCharacter(charRecord, nil, nil))
	}

	hydrated := h.hydrateCharacterAffiliations(response.Characters)
	if len(hydrated) > 0 {
		response.Characters = hydrated
	}

	sort.Slice(response.Characters, func(i, j int) bool {
		if response.Characters[i].IsMain != response.Characters[j].IsMain {
			return response.Characters[i].IsMain
		}
		return response.Characters[i].CharacterID < response.Characters[j].CharacterID
	})

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) SetMainCharacter(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	payload := struct {
		CharacterRecordID string `json:"character_record_id"`
	}{}

	if bindErr := c.BindBody(&payload); bindErr != nil || payload.CharacterRecordID == "" {
		return router.NewBadRequestError("Missing character record id.", logging.Fields{
			"user_id": userID,
		})
	}

	user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}

	character, characterErr := h.App.FindRecordById(store.CollectionCharacters, payload.CharacterRecordID)
	if characterErr != nil || character.GetString("user") != userID {
		return router.NewNotFoundError("Character not found.", logging.Fields{
			"user_id":              userID,
			"character_record_id":  payload.CharacterRecordID,
			"character_owner_user": character.GetString("user"),
		})
	}

	records, _ := h.App.FindRecordsByFilter(store.CollectionCharacters, "user = {:user}", "", 0, 0, dbx.Params{"user": userID})
	for _, rec := range records {
		rec.Set("is_main", rec.Id == character.Id)
		_ = h.App.Save(rec)
	}

	if updateErr := h.updateUserFromCharacter(user, character); updateErr != nil {
		return router.NewInternalServerError("Failed to update user.", logging.Fields{
			"user_id":       userID,
			"character_id":  character.GetInt("eve_character_id"),
			"character_rec": character.Id,
		})
	}
	h.logAction(
		c,
		&audit.Event{
			Action:          audit.ActionCharacterSetMain,
			Summary:         "Set main to " + character.GetString("eve_character_name"),
			TargetUserID:    userID,
			TargetUserName:  user.GetString("eve_character_name"),
			TargetCharacter: character,
		},
	)

	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) SetAccessLevel(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	payload := struct {
		AccessLevel string `json:"access_level"`
	}{}

	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{
			"user_id": userID,
		})
	}

	if payload.AccessLevel != "" && payload.AccessLevel != "staff" && payload.AccessLevel != "admin" {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{
			"user_id":      userID,
			"access_level": payload.AccessLevel,
		})
	}

	if payload.AccessLevel == "admin" {
		return middleware.ErrForbidden
	}

	user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}

	if user.GetString("access_level") == "admin" {
		return middleware.ErrForbidden
	}
	user.Set("access_level", payload.AccessLevel)
	if saveErr := h.App.Save(user); saveErr != nil {
		return router.NewInternalServerError("Failed to update user.", logging.Fields{
			"user_id":      userID,
			"access_level": payload.AccessLevel,
		})
	}
	action := audit.ActionUserAccessLevelCleared
	summary := "Cleared access level"
	if payload.AccessLevel != "" {
		action = audit.ActionUserAccessLevelSet
		summary = "Set access level to " + payload.AccessLevel
	}
	event := audit.Event{
		Action:         action,
		Summary:        summary,
		TargetUserID:   userID,
		TargetUserName: user.GetString("eve_character_name"),
	}
	h.applyAccountTarget(&event, userID, user.GetString("eve_character_name"))
	h.logAction(c, &event)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) RevokeSessions(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}
	user.RefreshTokenKey()
	sessionRevokedAt, _ := types.ParseDateTime(time.Now())
	user.Set("session_revoked_at", sessionRevokedAt)
	if saveErr := h.App.Save(user); saveErr != nil {
		return router.NewInternalServerError("Failed to revoke sessions.", logging.Fields{
			"user_id": userID,
		})
	}

	if h.Intel != nil {
		if revokeErr := h.Intel.RevokeUploaderSessionsForUser(userID); revokeErr != nil {
			return router.NewInternalServerError("Failed to revoke uploader sessions.", logging.Fields{
				"user_id": userID,
			})
		}
	}
	event := audit.Event{
		Action:         audit.ActionUserRevokeSessions,
		Summary:        "Revoked sessions",
		TargetUserID:   userID,
		TargetUserName: user.GetString("eve_character_name"),
	}
	h.applyAccountTarget(&event, userID, user.GetString("eve_character_name"))
	h.logAction(c, &event)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) RevokeUploaderTokens(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	if h.Intel != nil {
		if revokeErr := h.Intel.RevokeUploaderTokensForUser(userID); revokeErr != nil {
			return router.NewInternalServerError("Failed to revoke uploader tokens.", logging.Fields{
				"user_id": userID,
			})
		}
	}
	event := audit.Event{
		Action:       audit.ActionUserRevokeUploadTokens,
		Summary:      "Revoked uploader tokens",
		TargetUserID: userID,
	}
	h.applyAccountTarget(&event, userID, "")
	h.logAction(c, &event)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) RegenerateUploaderToken(c *core.RequestEvent) error {
	userID := c.Request.PathValue("id")
	_, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return router.NewNotFoundError("User not found.", logging.Fields{
			"user_id": userID,
		})
	}

	if h.Intel == nil {
		return router.NewInternalServerError("Intel service unavailable.", logging.Fields{
			"user_id": userID,
		})
	}
	record, regenErr := h.Intel.RegenerateUploaderToken(userID)
	if regenErr != nil {
		return router.NewInternalServerError("Failed to regenerate uploader token.", logging.Fields{
			"user_id": userID,
		})
	}
	event := audit.Event{
		Action:       audit.ActionUserRegenerateUploadToken,
		Summary:      "Regenerated uploader token",
		TargetUserID: userID,
	}
	h.applyAccountTarget(&event, userID, "")
	h.logAction(c, &event)
	return c.JSON(http.StatusOK, map[string]any{"token": record.Id})
}

func (h *Handler) RevokeCharacterTokens(c *core.RequestEvent) error {
	id := c.Request.PathValue("id")
	record, recordErr := h.App.FindRecordById(store.CollectionCharacters, id)
	if recordErr != nil {
		return router.NewNotFoundError("Character not found.", logging.Fields{
			"character_record_id": id,
		})
	}
	h.clearCharacterTokens(record, "revoked")
	if saveErr := h.App.Save(record); saveErr != nil {
		return router.NewInternalServerError("Failed to revoke tokens.", logging.Fields{
			"character_id": record.GetInt("eve_character_id"),
		})
	}
	h.logAction(
		c,
		&audit.Event{
			Action:          audit.ActionCharacterRevokeTokens,
			Summary:         "Revoked character tokens for " + record.GetString("eve_character_name"),
			TargetUserID:    record.GetString("user"),
			TargetCharacter: record,
		},
	)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) RemoveCharacter(c *core.RequestEvent) error {
	id := c.Request.PathValue("id")
	record, recordErr := h.App.FindRecordById(store.CollectionCharacters, id)
	if recordErr != nil {
		return router.NewNotFoundError("Character not found.", logging.Fields{
			"character_record_id": id,
		})
	}
	userID := record.GetString("user")
	isMain := record.GetBool("is_main")
	if isMain {
		others, othersErr := h.App.FindRecordsByFilter(
			store.CollectionCharacters,
			"user = {:user} && id != {:id}",
			"",
			1,
			0, dbx.Params{"user": userID, "id": record.Id},
		)
		if othersErr == nil && len(others) > 0 {
			return router.NewBadRequestError("Cannot remove main character while other characters remain.", logging.Fields{
				"user_id":      userID,
				"character_id": record.GetInt("eve_character_id"),
			})
		}
	}
	targetUserName := ""
	if userID != "" {
		user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
		if userErr == nil {
			targetUserName = user.GetString("eve_character_name")
		}
	}
	deleteErr := h.App.Delete(record)
	if deleteErr != nil {
		return router.NewInternalServerError("Failed to delete character.", logging.Fields{
			"character_id": record.GetInt("eve_character_id"),
		})
	}

	if userID != "" && isMain {
		_ = h.deleteUserIfNoCharacters(userID)
	}
	h.logAction(
		c,
		&audit.Event{
			Action:          audit.ActionCharacterRemove,
			Summary:         "Removed character " + record.GetString("eve_character_name"),
			TargetUserID:    userID,
			TargetUserName:  targetUserName,
			TargetCharacter: record,
		},
	)
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) MoveCharacter(c *core.RequestEvent) error {
	id := c.Request.PathValue("id")
	payload := struct {
		TargetUserID string `json:"target_user_id"`
	}{}

	if bindErr := c.BindBody(&payload); bindErr != nil || payload.TargetUserID == "" {
		return router.NewBadRequestError("Missing target user.", logging.Fields{
			"character_record_id": id,
		})
	}

	record, recordErr := h.App.FindRecordById(store.CollectionCharacters, id)
	if recordErr != nil {
		return router.NewNotFoundError("Character not found.", logging.Fields{
			"character_record_id": id,
		})
	}

	sourceUserID := record.GetString("user")
	targetUser, targetErr := h.App.FindRecordById(store.CollectionUsers, payload.TargetUserID)
	if targetErr != nil {
		return router.NewNotFoundError("Target user not found.", logging.Fields{
			"target_user_id": payload.TargetUserID,
		})
	}
	targetMain, _ := h.findMainCharacter(targetUser.Id)
	if targetMain == nil {
		return router.NewBadRequestError("Target user missing main character.", logging.Fields{
			"target_user_id": payload.TargetUserID,
		})
	}

	record.Set("user", targetUser.Id)
	record.Set("is_main", false)
	if saveErr := h.App.Save(record); saveErr != nil {
		return router.NewInternalServerError("Failed to move character.", logging.Fields{
			"character_id":   record.GetInt("eve_character_id"),
			"source_user_id": sourceUserID,
			"target_user_id": targetUser.Id,
		})
	}

	if sourceUserID != "" && sourceUserID != targetUser.Id {
		// Post-move cleanup: delete the source account only if it's now empty.
		_ = h.deleteUserIfNoCharacters(sourceUserID)
	}

	if sourceUserID != "" {
		h.logAction(
			c,
			&audit.Event{
				Action:          audit.ActionCharacterMoveOut,
				Summary:         "Moved character " + record.GetString("eve_character_name") + " to " + targetUser.Id,
				TargetUserID:    sourceUserID,
				TargetCharacter: record,
			},
		)
	}
	h.logAction(
		c,
		&audit.Event{
			Action:          audit.ActionCharacterMoveIn,
			Summary:         "Received character " + record.GetString("eve_character_name") + " from " + sourceUserID,
			TargetUserID:    targetUser.Id,
			TargetUserName:  targetUser.GetString("eve_character_name"),
			TargetCharacter: record,
		},
	)

	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) MergeUsers(c *core.RequestEvent) error {
	sourceUserID := c.Request.PathValue("id")
	payload := struct {
		TargetUserID string `json:"target_user_id"`
	}{}

	if bindErr := c.BindBody(&payload); bindErr != nil || payload.TargetUserID == "" {
		return router.NewBadRequestError("Missing target user.", logging.Fields{
			"source_user_id": sourceUserID,
		})
	}

	if sourceUserID == payload.TargetUserID {
		return router.NewBadRequestError("Source and target must differ.", logging.Fields{
			"source_user_id": sourceUserID,
			"target_user_id": payload.TargetUserID,
		})
	}

	sourceUser, sourceErr := h.App.FindRecordById(store.CollectionUsers, sourceUserID)
	if sourceErr != nil {
		return router.NewNotFoundError("Source user not found.", logging.Fields{
			"source_user_id": sourceUserID,
		})
	}
	targetUser, targetErr := h.App.FindRecordById(store.CollectionUsers, payload.TargetUserID)
	if targetErr != nil {
		return router.NewNotFoundError("Target user not found.", logging.Fields{
			"target_user_id": payload.TargetUserID,
		})
	}
	targetMain, _ := h.findMainCharacter(targetUser.Id)
	if targetMain == nil {
		return router.NewBadRequestError("Target user missing main character.", logging.Fields{
			"target_user_id": payload.TargetUserID,
		})
	}

	records, recordsErr := h.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user}",
		"",
		0,
		0, dbx.Params{"user": sourceUser.Id},
	)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to load characters.", logging.Fields{
			"user_id": sourceUser.Id,
		})
	}

	for _, rec := range records {
		rec.Set("user", targetUser.Id)
		rec.Set("is_main", false)
		if saveErr := h.App.Save(rec); saveErr != nil {
			return router.NewInternalServerError("Failed to move character.", logging.Fields{
				"character_id":   rec.GetInt("eve_character_id"),
				"source_user_id": sourceUser.Id,
				"target_user_id": targetUser.Id,
			})
		}
	}

	targetUserName := targetUser.GetString("eve_character_name")
	sourceUserName := sourceUser.GetString("eve_character_name")
	h.logAction(
		c,
		&audit.Event{
			Action:                 audit.ActionUserMergeOut,
			Summary:                "Merged account into " + targetUser.Id,
			TargetUserID:           sourceUser.Id,
			TargetUserName:         sourceUserName,
			ResolveTargetCharacter: true,
		},
	)
	h.logAction(
		c,
		&audit.Event{
			Action:                 audit.ActionUserMergeIn,
			Summary:                "Merged account from " + sourceUser.Id,
			TargetUserID:           targetUser.Id,
			TargetUserName:         targetUserName,
			ResolveTargetCharacter: true,
		},
	)
	// Post-merge cleanup: delete the source account only if it's now empty.
	_ = h.deleteUserIfNoCharacters(sourceUser.Id)

	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) clearCharacterTokens(record *core.Record, reason string) {
	record.Set("oauth_access_token", "")
	record.Set("oauth_refresh_token", "")
	record.Set("oauth_access_expires_at", types.DateTime{})
	record.Set("oauth_refresh_expires_at", types.DateTime{})
	record.Set("esi_token_valid", false)
	record.Set("esi_last_error", reason)
	lastRefreshAt, _ := types.ParseDateTime(time.Now())
	record.Set("esi_last_refresh_at", lastRefreshAt)
}

// deleteUserIfNoCharacters removes users that have no linked characters.
func (h *Handler) deleteUserIfNoCharacters(userID string) error {
	user, userErr := h.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return userErr
	}
	records, recordsErr := h.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user}",
		"",
		1,
		0, dbx.Params{"user": userID},
	)
	if recordsErr != nil {
		return recordsErr
	}

	if len(records) == 0 {
		return h.App.Delete(user)
	}
	return nil
}

func (h *Handler) findMainCharacter(userID string) (*core.Record, error) {
	records, recordsErr := h.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user} && is_main = true",
		"",
		1,
		0, dbx.Params{"user": userID},
	)
	if recordsErr != nil || len(records) == 0 {
		return nil, recordsErr
	}
	return records[0], nil
}

func (h *Handler) updateUserFromCharacter(user, character *core.Record) error {
	user.Set("eve_character_id", character.GetInt("eve_character_id"))
	user.Set("eve_character_name", character.GetString("eve_character_name"))
	user.Set("eve_corporation_id", character.GetInt("eve_corporation_id"))
	user.Set("eve_alliance_id", character.GetInt("eve_alliance_id"))
	return h.App.Save(user)
}

func (h *Handler) hydrateCharacterAffiliations(chars []characterResponse) []characterResponse {
	if len(chars) == 0 {
		return chars
	}
	corpIDs := make([]int, 0, len(chars))
	allianceIDs := make([]int, 0, len(chars))
	for i := range chars {
		if chars[i].CorpID > 0 {
			corpIDs = append(corpIDs, chars[i].CorpID)
		}
		if chars[i].AllianceID > 0 {
			allianceIDs = append(allianceIDs, chars[i].AllianceID)
		}
	}
	corpNames := store.GetOrgNames(h.App, store.CollectionCorporations, corpIDs)
	allianceNames := store.GetOrgNames(h.App, store.CollectionAlliances, allianceIDs)
	if len(corpNames) == 0 && len(allianceNames) == 0 {
		return chars
	}
	enriched := make([]characterResponse, 0, len(chars))
	for i := range chars {
		char := chars[i]
		if name, ok := corpNames[chars[i].CorpID]; ok {
			char.CorpName = name
		}
		if name, ok := allianceNames[chars[i].AllianceID]; ok {
			char.AllianceName = name
		}
		enriched = append(enriched, char)
	}
	return enriched
}

func newCharacter(record *core.Record, corpName, allianceName map[int]string) characterResponse {
	refresh := record.GetDateTime("esi_last_refresh_at")
	refreshAt := ""
	if !refresh.IsZero() {
		refreshAt = refresh.Time().Format(time.RFC3339)
	}
	corpID := record.GetInt("eve_corporation_id")
	allianceID := record.GetInt("eve_alliance_id")
	corp := ""
	alliance := ""
	if corpName != nil {
		corp = corpName[corpID]
	}

	if allianceName != nil {
		alliance = allianceName[allianceID]
	}
	return characterResponse{
		ID:               record.Id,
		CharacterID:      record.GetInt("eve_character_id"),
		Name:             record.GetString("eve_character_name"),
		CorpID:           corpID,
		CorpName:         corp,
		AllianceID:       allianceID,
		AllianceName:     alliance,
		IsMain:           record.GetBool("is_main"),
		Scopes:           record.GetString("oauth_scopes"),
		ESILastRefreshAt: refreshAt,
		ESILastError:     record.GetString("esi_last_error"),
		ESITokenValid:    record.GetBool("esi_token_valid"),
	}
}
