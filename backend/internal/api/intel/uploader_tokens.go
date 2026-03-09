package intel

import (
	"errors"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/auth"
	"sentinel2/internal/intel"
	"sentinel2/internal/logging"
	"sentinel2/internal/realtime"
	"sentinel2/internal/shared/requestctx"
)

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
		if errors.Is(recordErr, intel.ErrExpiredOrRevoked) {
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
		if errors.Is(recordErr, intel.ErrExpiredOrRevoked) {
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
	ctxUserID := requestctx.String(c, "uploader_user_id")
	ctxUploaderTokenID := requestctx.String(c, "uploader_token_id")
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

	return c.JSON(http.StatusOK, newUploaderRealtimeTokenResponse(session))
}

func (h *IntelHandler) UploaderSessionRefresh(c *core.RequestEvent) error {
	if c.Auth == nil {
		return router.NewUnauthorizedError("Invalid uploader session.", nil)
	}
	session, sessionErr := h.Service.RefreshUploaderRealtimeSession(c.Auth)
	if sessionErr != nil {
		if errors.Is(sessionErr, intel.ErrExpiredOrRevoked) {
			return router.NewUnauthorizedError("Uploader token revoked.", nil)
		}
		return router.NewInternalServerError("Failed to refresh realtime token.", logging.Fields{
			"error": sessionErr.Error(),
		})
	}

	return c.JSON(http.StatusOK, newUploaderRealtimeTokenResponse(session))
}

func newUploaderRealtimeTokenResponse(session intel.UploaderRealtimeSession) uploaderRealtimeTokenResponse {
	return uploaderRealtimeTokenResponse{
		Token:               session.Token,
		Topic:               realtime.TopicUploaderConfig,
		ExpiresAt:           session.ExpiresAt.Unix(),
		RefreshAfterSeconds: int64(session.RefreshAfter.Seconds()),
	}
}
