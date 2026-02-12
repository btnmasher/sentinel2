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

type jumpbridgeMutationResponse struct {
	Changed bool `json:"changed"`
	Count   int  `json:"count"`
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

func (h *JumpbridgeHandler) Add(c *core.RequestEvent) error {
	payload := struct {
		FromID int `json:"from_id"`
		ToID   int `json:"to_id"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Invalid payload.", nil)
	}
	if payload.FromID <= 0 || payload.ToID <= 0 {
		return router.NewBadRequestError("Both from_id and to_id are required.", nil)
	}

	service := jumpbridges.NewJumpbridgeService(h.App)
	changed, addErr := service.AddPair(payload.FromID, payload.ToID)
	if addErr != nil {
		return router.NewBadRequestError(addErr.Error(), nil)
	}

	if changed {
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
	}

	count := 0
	if changed {
		count = 1
	}
	return c.JSON(http.StatusOK, jumpbridgeMutationResponse{Changed: changed, Count: count})
}

func (h *JumpbridgeHandler) Remove(c *core.RequestEvent) error {
	payload := struct {
		FromID int `json:"from_id"`
		ToID   int `json:"to_id"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Invalid payload.", nil)
	}
	if payload.FromID <= 0 || payload.ToID <= 0 {
		return router.NewBadRequestError("Both from_id and to_id are required.", nil)
	}

	service := jumpbridges.NewJumpbridgeService(h.App)
	deleted, removeErr := service.RemovePair(payload.FromID, payload.ToID)
	if removeErr != nil {
		return router.NewInternalServerError("Failed to remove jumpbridge.", nil)
	}

	if deleted > 0 {
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
	}

	return c.JSON(http.StatusOK, jumpbridgeMutationResponse{Changed: deleted > 0, Count: deleted / 2})
}

func (h *JumpbridgeHandler) Update(c *core.RequestEvent) error {
	payload := struct {
		OldFromID int `json:"old_from_id"`
		OldToID   int `json:"old_to_id"`
		FromID    int `json:"from_id"`
		ToID      int `json:"to_id"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Invalid payload.", nil)
	}
	if payload.OldFromID <= 0 || payload.OldToID <= 0 || payload.FromID <= 0 || payload.ToID <= 0 {
		return router.NewBadRequestError("old_from_id, old_to_id, from_id, and to_id are required.", nil)
	}

	service := jumpbridges.NewJumpbridgeService(h.App)
	changed, updateErr := service.UpdatePair(payload.OldFromID, payload.OldToID, payload.FromID, payload.ToID)
	if updateErr != nil {
		return router.NewBadRequestError(updateErr.Error(), nil)
	}

	if changed {
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
	}

	count := 0
	if changed {
		count = 1
	}
	return c.JSON(http.StatusOK, jumpbridgeMutationResponse{Changed: changed, Count: count})
}
