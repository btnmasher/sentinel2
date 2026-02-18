package staff

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/audit"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"

	"github.com/pocketbase/pocketbase/tools/router"
)

func NewChannelsHandler(app *pocketbase.PocketBase, auditSvc *audit.Service) *ChannelsHandler {
	return &ChannelsHandler{App: app, Audit: auditSvc}
}

func (h *ChannelsHandler) List(c *core.RequestEvent) error {
	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionIntelChannels, "", "channel_name", 0, 0, nil)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to load channels.", logging.Fields{
			"error": recordsErr.Error(),
		})
	}

	channels := []channelDTO{}
	for _, rec := range records {
		channels = append(channels, channelDTO{
			ID:          rec.Id,
			ChannelName: rec.GetString("channel_name"),
		})
	}

	return c.JSON(http.StatusOK, channelListResponse{Channels: channels})
}

func (h *ChannelsHandler) Create(c *core.RequestEvent) error {
	payload := struct {
		ChannelName string `json:"channel_name"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{
			"error": bindErr.Error(),
		})
	}
	if payload.ChannelName == "" {
		return router.NewBadRequestError("Missing channel name.", logging.Fields{
			"channel_name": payload.ChannelName,
		})
	}

	coll, collErr := h.App.FindCollectionByNameOrId(store.CollectionIntelChannels)
	if collErr != nil {
		return router.NewBadRequestError("Missing collection.", logging.Fields{
			"collection": store.CollectionIntelChannels,
			"error":      collErr.Error(),
		})
	}

	record := core.NewRecord(coll)
	record.Set("channel_name", payload.ChannelName)
	if saveErr := h.App.Save(record); saveErr != nil {
		return router.NewInternalServerError("Failed to save channel.", logging.Fields{
			"channel_name": payload.ChannelName,
			"error":        saveErr.Error(),
		})
	}

	logFields := logging.Fields{
		"channel_id": record.Id,
		"channel":    payload.ChannelName,
	}
	if c.Auth != nil {
		logFields["user_id"] = c.Auth.Id
	}
	logging.WithRequest(h.App, c).
		WithFields(logFields).
		Info("channel created")
	if h.Audit != nil {
		h.Audit.LogRequest(c, &audit.Event{
			Action:      audit.ActionStaffChannelCreate,
			Summary:     "Created channel " + payload.ChannelName,
			TargetType:  audit.TargetTypeChannel,
			TargetID:    record.Id,
			TargetLabel: payload.ChannelName,
			TargetMeta: map[string]any{
				"channel_id": record.Id,
			},
		})
	}

	return c.JSON(http.StatusCreated, channelCreateResponse{ID: record.Id})
}

func (h *ChannelsHandler) Delete(c *core.RequestEvent) error {
	id := c.Request.PathValue("id")
	if id == "" {
		return router.NewBadRequestError("Missing id.", logging.Fields{
			"required_field": "id",
		})
	}

	record, recordErr := h.App.FindRecordById(store.CollectionIntelChannels, id)
	if recordErr != nil {
		return router.NewNotFoundError("Not found", logging.Fields{
			"channel_id": id,
			"error":      recordErr.Error(),
		})
	}

	if deleteErr := h.App.Delete(record); deleteErr != nil {
		return router.NewInternalServerError("Failed to delete record.", logging.Fields{
			"channel_id": id,
			"error":      deleteErr.Error(),
		})
	}

	logFields := logging.Fields{
		"channel_id": id,
		"channel":    record.GetString("channel_name"),
	}
	if c.Auth != nil {
		logFields["user_id"] = c.Auth.Id
	}
	logging.WithRequest(h.App, c).
		WithFields(logFields).
		Info("channel deleted")
	if h.Audit != nil {
		h.Audit.LogRequest(c, &audit.Event{
			Action:      audit.ActionStaffChannelDelete,
			Summary:     "Deleted channel " + record.GetString("channel_name"),
			TargetType:  audit.TargetTypeChannel,
			TargetID:    id,
			TargetLabel: record.GetString("channel_name"),
			TargetMeta: map[string]any{
				"channel_id": id,
			},
		})
	}

	return c.NoContent(http.StatusNoContent)
}
