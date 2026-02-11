package intel

import (
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/auth"
	"sentinel2/internal/config"
	"sentinel2/internal/intel"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

type IntelHandler struct {
	App     *pocketbase.PocketBase
	Config  config.Config
	Service *intel.IntelService
}

func NewIntelHandler(app *pocketbase.PocketBase, cfg config.Config) *IntelHandler {
	return &IntelHandler{
		App:     app,
		Config:  cfg,
		Service: intel.NewIntelService(app),
	}
}

type intelRetrieveResponse struct {
	Intel     []intel.IntelReport `json:"intel"`
	Uploaders int                 `json:"uploaders"`
	Version   string              `json:"version"`
}

type intelMetaResponse struct {
	Uploaders int    `json:"uploaders"`
	Version   string `json:"version"`
}

type uploaderTokenResponse struct {
	Token string `json:"token"`
}

type intelBroadcast struct {
	Intel     intel.IntelReport `json:"intel"`
	Uploaders int               `json:"uploaders"`
	Version   string            `json:"version"`
}
type submitPayload struct {
	Text   string `json:"text"`
	Status string `json:"status"`
}

func (h *IntelHandler) Submit(c *core.RequestEvent) error {
	log := logging.WithRequest(h.App, c)
	submitLog := log
	ctxUserID, _ := c.Get("uploader_user_id").(string)
	payload := submitPayload{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		submitLog.
			WithErr(bindErr).
			Warn("intel submit malformed payload")
		return router.NewBadRequestError("Malformed JSON.", nil)
	}

	userID := ctxUserID
	if userID == "" {
		submitLog.Warn("intel submit missing uploader context")
		return router.NewUnauthorizedError("Invalid uploader token.", nil)
	}
	_ = h.Service.UpdateUploader(userID, payload.Status)

	if payload.Text != "" {
		parsed, parseErr := intel.ParseReportText(payload.Text)
		if parseErr != nil {
			submitLog.
				WithErr(parseErr).
				Warn("intel submit parse failed")
			return router.NewBadRequestError(parseErr.Error(), nil)
		}

		reportTime := parsed.Date.Unix()
		systems, systemsErr := intel.LinkSystemNames(h.App, parsed.Text)
		if systemsErr != nil {
			submitLog.
				WithErr(systemsErr).
				Warn("intel submit link systems failed")
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
			submitLog.
				WithErr(shouldErr).
				Error("intel submit store check failed")
			return router.NewInternalServerError("Failed to store report.", logging.Fields{
				"author": parsed.Author,
			})
		}

		if shouldCreate {
			reportID := time.Now().UnixMilli()
			report := intel.IntelReport{
				ID:       reportID,
				Time:     reportTime,
				Author:   parsed.Author,
				Text:     parsed.Text,
				Systems:  systems,
				Regions:  regions,
				Uploader: userID,
			}
			if createErr := h.Service.CreateReport(report); createErr != nil {
				submitLog.
					WithErr(createErr).
					Error("intel submit create failed")
				return router.NewInternalServerError("Failed to save report.", logging.Fields{
					"author": parsed.Author,
				})
			}
			submitLog.WithFields(logging.Fields{
				"report_id":        reportID,
				"uploader_user_id": userID,
				"systems":          len(systems),
				"regions":          len(regions),
			}).Info("intel report created")

		}
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *IntelHandler) Meta(c *core.RequestEvent) error {
	uploaders, _ := h.Service.UploaderCount()
	return c.JSON(http.StatusOK, intelMetaResponse{
		Uploaders: uploaders,
		Version:   h.Config.SentinelVersion,
	})
}

func (h *IntelHandler) ListReports(c *core.RequestEvent) error {
	reports, reportsErr := h.Service.ListReports(50)
	if reportsErr != nil {
		logging.WithRequest(h.App, c).
			WithErr(reportsErr).
			Error("intel list reports failed")
		return router.NewInternalServerError("Failed to load reports.", nil)
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
			return router.NewForbiddenError("Uploader token revoked.", nil)
		}
		logging.WithRequest(h.App, c).
			WithErr(recordErr).
			Error("uploader token create failed")
		return router.NewInternalServerError("Failed to get uploader token.", logging.Fields{
			"user_id": user.Id,
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
			return router.NewForbiddenError("Uploader token revoked.", nil)
		}
		logging.WithRequest(h.App, c).
			WithErr(recordErr).
			Error("uploader token rotate failed")
		return router.NewInternalServerError("Failed to rotate uploader token.", logging.Fields{
			"user_id": user.Id,
		})
	}

	return c.JSON(http.StatusOK, uploaderTokenResponse{Token: record.Id})
}

func (h *IntelHandler) UploaderConfig(c *core.RequestEvent) error {
	records, recordsErr := h.App.FindRecordsByFilter(store.CollectionIntelChannels, "", "", 0, 0, nil)
	if recordsErr != nil {
		logging.WithRequest(h.App, c).
			WithErr(recordsErr).
			Error("uploader config load failed")
		return router.NewInternalServerError("Failed to load channels.", nil)
	}

	channels := []string{}
	for _, rec := range records {
		channels = append(channels, rec.GetString("channel_name"))
	}
	return c.JSON(http.StatusOK, uploaderConfigResponse{Channels: channels})
}

type uploaderConfigResponse struct {
	Channels []string `json:"channels"`
}
