package intel

import (
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/auth"
	"sentinel2/internal/config"
	"sentinel2/internal/intel"
	"sentinel2/internal/logging"
	"sentinel2/internal/realtime"
	"sentinel2/internal/store"
)

func NewIntelHandler(app *pocketbase.PocketBase, cfg config.Config, service *intel.IntelService) *IntelHandler {
	return &IntelHandler{
		App:     app,
		Config:  cfg,
		Service: service,
	}
}

func (h *IntelHandler) Submit(c *core.RequestEvent) error {
	ctxUserID, _ := c.Get("uploader_user_id").(string)
	payload := submitPayload{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Malformed JSON.", logging.Fields{
			"error": bindErr.Error(),
		})
	}

	userID := ctxUserID
	if userID == "" {
		return router.NewUnauthorizedError("Invalid uploader token.", logging.Fields{
			"reason": "missing uploader user context",
		})
	}
	if updateErr := h.Service.UpdateUploader(userID); updateErr != nil {
		return router.NewInternalServerError("Failed to refresh uploader heartbeat.", logging.Fields{
			"uploader_user_id": userID,
			"error":            updateErr.Error(),
		})
	}

	if payload.Text != "" {
		channelID := strings.TrimSpace(payload.ChannelID)
		if channelID == "" {
			return router.NewBadRequestError("Missing channel id.", logging.Fields{
				"uploader_user_id": userID,
			})
		}
		if _, channelErr := h.App.FindRecordById(store.CollectionIntelChannels, channelID); channelErr != nil {
			return router.NewBadRequestError("Invalid channel id.", logging.Fields{
				"channel_id":       channelID,
				"uploader_user_id": userID,
				"error":            channelErr.Error(),
			})
		}
		parsed, parseErr := intel.ParseReportText(payload.Text)
		if parseErr != nil {
			return router.NewBadRequestError(parseErr.Error(), logging.Fields{
				"channel_id":       channelID,
				"uploader_user_id": userID,
				"error":            parseErr.Error(),
				"text":             payload.Text,
			})
		}
		parsed.Author = strings.TrimSpace(parsed.Author)
		parsed.Text = strings.TrimSpace(parsed.Text)
		if parsed.Author == "" {
			return router.NewBadRequestError("Missing report author.", logging.Fields{
				"channel_id":       channelID,
				"uploader_user_id": userID,
			})
		}
		if parsed.Text == "" {
			return router.NewBadRequestError("Missing report text.", logging.Fields{
				"channel_id":       channelID,
				"uploader_user_id": userID,
			})
		}

		reportTime := parsed.Date.Unix()
		systems, systemsErr := intel.LinkSystemNames(h.App, parsed.Text)
		if systemsErr != nil {
			return router.NewInternalServerError("Failed to link systems.", logging.Fields{
				"author": parsed.Author,
			})
		}

		regions := []int{}
		regionSet := map[int]struct{}{}
		for _, sys := range systems {
			region := sys.Region
			if _, ok := regionSet[region]; !ok {
				regionSet[region] = struct{}{}
				regions = append(regions, region)
			}
		}

		shouldCreate, shouldErr := h.Service.ShouldCreateReport(parsed.Author, parsed.Text, reportTime)
		if shouldErr != nil {
			return router.NewInternalServerError("Failed to store report.", logging.Fields{
				"author": parsed.Author,
			})
		}

		if shouldCreate {
			reportID := time.Now().UnixMilli()
			report := intel.IntelReport{
				ID:        reportID,
				Time:      reportTime,
				Author:    parsed.Author,
				Text:      parsed.Text,
				Systems:   systems,
				Regions:   regions,
				Uploader:  userID,
				ChannelID: channelID,
			}
			if createErr := h.Service.CreateReport(report); createErr != nil {
				return router.NewInternalServerError("Failed to save report.", logging.Fields{
					"author":     parsed.Author,
					"channel_id": channelID,
				})
			}
		}
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *IntelHandler) Heartbeat(c *core.RequestEvent) error {
	ctxUserID, _ := c.Get("uploader_user_id").(string)
	if ctxUserID == "" {
		return router.NewUnauthorizedError("Invalid uploader token.", logging.Fields{
			"reason": "missing uploader user context",
		})
	}
	if updateErr := h.Service.UpdateUploader(ctxUserID); updateErr != nil {
		return router.NewInternalServerError("Failed to refresh uploader heartbeat.", logging.Fields{
			"uploader_user_id": ctxUserID,
			"error":            updateErr.Error(),
		})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *IntelHandler) ListReports(c *core.RequestEvent) error {
	reports, reportsErr := h.Service.ListReports(50)
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

func (h *IntelHandler) GetUploaderToken(c *core.RequestEvent) error {
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		logging.WithRequest(h.App, c).
			WithErr(userErr).
			Warn("uploader token request unauthorized")
		return userErr
	}

	record, recordErr := h.Service.GetValidUploaderToken(user.Id)
	if recordErr != nil {
		if recordErr == intel.ErrExpiredOrRevoked {
			return router.NewForbiddenError("Uploader token revoked.", logging.Fields{
				"user_id": user.Id,
			})
		}
		return router.NewInternalServerError("Failed to get uploader token.", logging.Fields{
			"user_id": user.Id,
			"error":   recordErr.Error(),
		})
	}

	return c.JSON(http.StatusOK, uploaderTokenResponse{Token: record.Id})
}

func (h *IntelHandler) RotateUploaderToken(c *core.RequestEvent) error {
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		logging.WithRequest(h.App, c).
			WithErr(userErr).
			Warn("uploader token rotate unauthorized")
		return userErr
	}

	record, recordErr := h.Service.RotateUploaderToken(user.Id)
	if recordErr != nil {
		if recordErr == intel.ErrExpiredOrRevoked {
			return router.NewForbiddenError("Uploader token revoked.", logging.Fields{
				"user_id": user.Id,
			})
		}
		return router.NewInternalServerError("Failed to rotate uploader token.", logging.Fields{
			"user_id": user.Id,
			"error":   recordErr.Error(),
		})
	}

	return c.JSON(http.StatusOK, uploaderTokenResponse{Token: record.Id})
}

func (h *IntelHandler) UploaderConfig(c *core.RequestEvent) error {
	cfg, cfgErr := h.Service.UploaderConfig()
	if cfgErr != nil {
		return router.NewInternalServerError("Failed to load channels.", logging.Fields{
			"error": cfgErr.Error(),
		})
	}

	return c.JSON(http.StatusOK, cfg)
}

func (h *IntelHandler) UploaderRealtimeToken(c *core.RequestEvent) error {
	ctxUserID, _ := c.Get("uploader_user_id").(string)
	ctxUploaderTokenID, _ := c.Get("uploader_token_id").(string)

	if ctxUserID == "" || ctxUploaderTokenID == "" {
		return router.NewUnauthorizedError("Invalid uploader token.", logging.Fields{
			"user_id":           ctxUserID,
			"uploader_token_id": ctxUploaderTokenID,
		})
	}

	session, sessionErr := h.Service.IssueUploaderRealtimeSession(ctxUserID, ctxUploaderTokenID)
	if sessionErr != nil {
		return router.NewInternalServerError("Failed to issue realtime token.", logging.Fields{
			"user_id":           ctxUserID,
			"uploader_token_id": ctxUploaderTokenID,
			"error":             sessionErr.Error(),
		})
	}

	return c.JSON(http.StatusOK, uploaderRealtimeTokenResponse{
		Token:               session.Token,
		Topic:               realtime.TopicUploaderConfig,
		ExpiresAt:           session.ExpiresAt.Unix(),
		RefreshAfterSeconds: int64(session.RefreshAfter.Seconds()),
	})
}

func (h *IntelHandler) UploaderSessionRefresh(c *core.RequestEvent) error {
	if c.Auth == nil {
		return router.NewUnauthorizedError("Invalid uploader session.", nil)
	}
	session, sessionErr := h.Service.RefreshUploaderRealtimeSession(c.Auth)
	if sessionErr != nil {
		if sessionErr == intel.ErrExpiredOrRevoked {
			return router.NewUnauthorizedError("Uploader token revoked.", nil)
		}
		return router.NewInternalServerError("Failed to refresh realtime token.", logging.Fields{
			"error": sessionErr.Error(),
		})
	}

	return c.JSON(http.StatusOK, uploaderRealtimeTokenResponse{
		Token:               session.Token,
		Topic:               realtime.TopicUploaderConfig,
		ExpiresAt:           session.ExpiresAt.Unix(),
		RefreshAfterSeconds: int64(session.RefreshAfter.Seconds()),
	})
}
