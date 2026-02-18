package intel

import (
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/intel"
	"sentinel2/internal/logging"
	"sentinel2/internal/shared/collections"
	"sentinel2/internal/shared/requestctx"
	"sentinel2/internal/store"
)

const reportsListLimit = 50

func (h *IntelHandler) Submit(c *core.RequestEvent) error {
	userID, ctxErr := uploaderUserIDFromContext(c)
	if ctxErr != nil {
		return ctxErr
	}
	if refreshErr := h.refreshUploaderHeartbeat(userID); refreshErr != nil {
		return refreshErr
	}

	payload := submitPayload{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Malformed JSON.", logging.Fields{
			"error": bindErr.Error(),
		})
	}

	if submitErr := h.submitReportIfPresent(payload, userID); submitErr != nil {
		return submitErr
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *IntelHandler) Heartbeat(c *core.RequestEvent) error {
	userID, ctxErr := uploaderUserIDFromContext(c)
	if ctxErr != nil {
		return ctxErr
	}
	if refreshErr := h.refreshUploaderHeartbeat(userID); refreshErr != nil {
		return refreshErr
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *IntelHandler) ListReports(c *core.RequestEvent) error {
	reports, reportsErr := h.Service.ListReports(reportsListLimit)
	if reportsErr != nil {
		return router.NewInternalServerError("Failed to load reports.", logging.Fields{
			"error": reportsErr.Error(),
		})
	}
	uploaders, _ := h.Service.UploaderCount()
	return c.JSON(http.StatusOK, intelRetrieveResponse{
		Intel:     reports,
		Uploaders: uploaders,
		Version:   h.Config.SentinelVersion,
	})
}

func uploaderUserIDFromContext(c *core.RequestEvent) (string, error) {
	userID := requestctx.String(c, "uploader_user_id")
	if userID == "" {
		return "", router.NewUnauthorizedError("Invalid uploader token.", logging.Fields{
			"reason": "missing uploader user context",
		})
	}
	return userID, nil
}

func (h *IntelHandler) refreshUploaderHeartbeat(userID string) error {
	if updateErr := h.Service.UpdateUploader(userID); updateErr != nil {
		return router.NewInternalServerError("Failed to refresh uploader heartbeat.", logging.Fields{
			"uploader_user_id": userID,
			"error":            updateErr.Error(),
		})
	}
	return nil
}

func (h *IntelHandler) submitReportIfPresent(payload submitPayload, userID string) error {
	if strings.TrimSpace(payload.Text) == "" {
		return nil
	}

	channelID, channelErr := h.resolveSubmitChannelID(payload.ChannelID, userID)
	if channelErr != nil {
		return channelErr
	}
	parsed, parseErr := h.parseSubmitPayload(payload.Text, channelID, userID)
	if parseErr != nil {
		return parseErr
	}
	reportTime := parsed.Date.Unix()
	systems, systemsErr := intel.LinkSystemNames(h.App, parsed.Text)
	if systemsErr != nil {
		return router.NewInternalServerError("Failed to link systems.", logging.Fields{
			"author": parsed.Author,
		})
	}
	regions := submitRegions(systems)
	shouldCreate, shouldErr := h.Service.ShouldCreateReport(parsed.Author, parsed.Text, reportTime)
	if shouldErr != nil {
		return router.NewInternalServerError("Failed to store report.", logging.Fields{
			"author": parsed.Author,
		})
	}
	if !shouldCreate {
		return nil
	}

	report := intel.IntelReport{
		ID:        time.Now().UnixMilli(),
		Time:      reportTime,
		Author:    parsed.Author,
		Text:      parsed.Text,
		Systems:   systems,
		Regions:   regions,
		Uploader:  userID,
		ChannelID: channelID,
	}
	if createErr := h.Service.CreateReport(&report); createErr != nil {
		return router.NewInternalServerError("Failed to save report.", logging.Fields{
			"author":     parsed.Author,
			"channel_id": channelID,
		})
	}
	return nil
}

func (h *IntelHandler) resolveSubmitChannelID(rawChannelID, userID string) (string, error) {
	channelID := strings.TrimSpace(rawChannelID)
	if channelID == "" {
		return "", router.NewBadRequestError("Missing channel id.", logging.Fields{
			"uploader_user_id": userID,
		})
	}
	if _, channelErr := h.App.FindRecordById(store.CollectionIntelChannels, channelID); channelErr != nil {
		return "", router.NewBadRequestError("Invalid channel id.", logging.Fields{
			"channel_id":       channelID,
			"uploader_user_id": userID,
			"error":            channelErr.Error(),
		})
	}
	return channelID, nil
}

func (h *IntelHandler) parseSubmitPayload(text, channelID, userID string) (intel.ParsedReport, error) {
	parsed, parseErr := intel.ParseReportText(text)
	if parseErr != nil {
		return intel.ParsedReport{}, router.NewBadRequestError(parseErr.Error(), logging.Fields{
			"channel_id":       channelID,
			"uploader_user_id": userID,
			"error":            parseErr.Error(),
			"text":             text,
		})
	}
	parsed.Author = strings.TrimSpace(parsed.Author)
	parsed.Text = strings.TrimSpace(parsed.Text)
	if parsed.Author == "" {
		return intel.ParsedReport{}, router.NewBadRequestError("Missing report author.", logging.Fields{
			"channel_id":       channelID,
			"uploader_user_id": userID,
		})
	}
	if parsed.Text == "" {
		return intel.ParsedReport{}, router.NewBadRequestError("Missing report text.", logging.Fields{
			"channel_id":       channelID,
			"uploader_user_id": userID,
		})
	}
	return parsed, nil
}

func submitRegions(systems []intel.IntelSystem) []int {
	regions := make([]int, 0, len(systems))
	seen := make(map[int]struct{}, len(systems))
	for _, sys := range systems {
		_ = collections.AppendUnique(&regions, seen, sys.Region)
	}
	return regions
}
