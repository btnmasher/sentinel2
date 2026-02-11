package staff

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"

	"github.com/pocketbase/pocketbase/tools/router"
)

type channelDTO struct {
	ID          string `json:"id"`
	ChannelName string `json:"channel_name"`
}

type channelListResponse struct {
	Channels []channelDTO `json:"channels"`
}

type channelCreateResponse struct {
	ID string `json:"id"`
}

type ChannelsHandler struct {
	App *pocketbase.PocketBase
}

func NewChannelsHandler(app *pocketbase.PocketBase) *ChannelsHandler {
	return &ChannelsHandler{App: app}
}

func (h *ChannelsHandler) List(c *core.RequestEvent) error {
	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionIntelChannels, "", "channel_name", 0, 0, nil)
	if recordsErr != nil {
		logging.WithRequest(h.App, c).
			WithErr(recordsErr).
			Warn("channels list failed")
		return router.NewInternalServerError("Failed to load channels.", nil)
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
		logging.WithRequest(h.App, c).
			WithErr(bindErr).
			Warn("channel create malformed payload")
		return router.NewBadRequestError("Invalid payload.", nil)
	}
	if payload.ChannelName == "" {
		return router.NewBadRequestError("Missing channel name.", nil)
	}

	coll, collErr := h.App.FindCollectionByNameOrId(store.CollectionIntelChannels)
	if collErr != nil {
		logging.WithRequest(h.App, c).
			WithErr(collErr).
			Warn("channel create missing collection")
		return router.NewBadRequestError("Missing collection.", logging.Fields{
			"collection": store.CollectionIntelChannels,
		})
	}

	record := core.NewRecord(coll)
	record.Set("channel_name", payload.ChannelName)
	if saveErr := h.App.Save(record); saveErr != nil {
		logging.WithRequest(h.App, c).
			WithErr(saveErr).
			Warn("channel create failed")
		return router.NewInternalServerError("Failed to save channel.", logging.Fields{
			"channel_name": payload.ChannelName,
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

	return c.JSON(http.StatusCreated, channelCreateResponse{ID: record.Id})
}

func (h *ChannelsHandler) Delete(c *core.RequestEvent) error {
	id := c.Request.PathValue("id")
	if id == "" {
		return router.NewBadRequestError("Missing id.", nil)
	}

	record, recordErr := h.App.FindRecordById(store.CollectionIntelChannels, id)
	if recordErr != nil {
		logging.WithRequest(h.App, c).
			WithFields(logging.Fields{"channel_id": id}).
			WithErr(recordErr).
			Warn("channel delete not found")
		return router.NewNotFoundError("Not found", nil)
	}

	if deleteErr := h.App.Delete(record); deleteErr != nil {
		logging.WithRequest(h.App, c).
			WithFields(logging.Fields{"channel_id": id}).
			WithErr(deleteErr).
			Warn("channel delete failed")
		return router.NewInternalServerError("Failed to delete record.", logging.Fields{
			"channel_id": id,
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

	return c.NoContent(http.StatusNoContent)
}
