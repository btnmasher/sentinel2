package uploader

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/uploaderrelease"
)

type Handler struct {
	Service *uploaderrelease.Service
}

func NewHandler(service *uploaderrelease.Service) *Handler {
	return &Handler{Service: service}
}

func (h *Handler) DownloadLinks(c *core.RequestEvent) error {
	return c.JSON(http.StatusOK, h.Service.Snapshot())
}
