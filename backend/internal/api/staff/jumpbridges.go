package staff

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/jobs"
	"sentinel2/internal/jumpbridges"
	"sentinel2/internal/logging"
	"sentinel2/internal/mapdata"
)

type jumpbridgeImportResponse struct {
	Count int `json:"count"`
}

type JumpbridgeHandler struct {
	App *pocketbase.PocketBase
}

func NewJumpbridgeHandler(app *pocketbase.PocketBase) *JumpbridgeHandler {
	return &JumpbridgeHandler{App: app}
}

func (h *JumpbridgeHandler) Import(c *core.RequestEvent) error {
	payload := struct {
		Jumpbridges string `json:"jumpbridges"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		logging.WithRequest(h.App, c).
			WithErr(bindErr).
			Warn("jumpbridge import malformed payload")
		return router.NewBadRequestError("Invalid payload.", nil)
	}

	if strings.TrimSpace(payload.Jumpbridges) == "" {
		return router.NewBadRequestError("Jumpbridge import failed: empty input.", nil)
	}

	lines := strings.Split(strings.ReplaceAll(payload.Jumpbridges, "\r", ""), "\n")
	service := jumpbridges.NewJumpbridgeService(h.App)
	count, updateErr := service.UpdateFromLines(lines)
	if updateErr != nil {
		logFields := logging.Fields{
			"line_count": len(lines),
		}
		if c.Auth != nil {
			logFields["user_id"] = c.Auth.Id
		}
		logging.WithRequest(h.App, c).
			WithFields(logFields).
			WithErr(updateErr).
			Warn("jumpbridge import failed")
		return router.NewInternalServerError("Failed to import jumpbridges.", nil)
	}

	logFields := logging.Fields{
		"line_count": len(lines),
		"count":      count,
	}
	if c.Auth != nil {
		logFields["user_id"] = c.Auth.Id
	}
	logging.WithRequest(h.App, c).
		WithFields(logFields).
		Info("jumpbridge import completed")

	actorID := ""
	if c.Auth != nil {
		actorID = c.Auth.Id
	}
	_ = mapdata.TriggerMapDataStep(h.App, mapdata.StepTriggerOptions{
		Step:    mapdata.StepBuildGraph,
		Trigger: jobs.TriggerStaffJumpbridgeImport,
		ActorID: actorID,
		JobName: mapdata.JobMapDataStep,
		Logger:  logging.WithRequest(h.App, c),
	})

	return c.JSON(http.StatusOK, jumpbridgeImportResponse{Count: count})
}

func (h *JumpbridgeHandler) Clear(c *core.RequestEvent) error {
	service := jumpbridges.NewJumpbridgeService(h.App)
	_, updateErr := service.UpdateFromLines([]string{})
	if updateErr != nil {
		logging.WithRequest(h.App, c).
			WithErr(updateErr).
			Warn("jumpbridge clear failed")
		return router.NewInternalServerError("Failed to clear jumpbridges.", nil)
	}

	actorID := ""
	if c.Auth != nil {
		actorID = c.Auth.Id
	}
	_ = mapdata.TriggerMapDataStep(h.App, mapdata.StepTriggerOptions{
		Step:    mapdata.StepBuildGraph,
		Trigger: jobs.TriggerStaffJumpbridgeImport,
		ActorID: actorID,
		JobName: mapdata.JobMapDataStep,
		Logger:  logging.WithRequest(h.App, c),
	})

	return c.JSON(http.StatusOK, jumpbridgeImportResponse{Count: 0})
}
