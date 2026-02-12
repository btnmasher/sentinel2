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

func NewJumpbridgeHandler(app *pocketbase.PocketBase, service *jumpbridges.JumpbridgeService) *JumpbridgeHandler {
	return &JumpbridgeHandler{App: app, Service: service}
}

func (h *JumpbridgeHandler) Import(c *core.RequestEvent) error {
	payload := struct {
		Jumpbridges string `json:"jumpbridges"`
	}{}
	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{
			"error": bindErr.Error(),
		})
	}

	if strings.TrimSpace(payload.Jumpbridges) == "" {
		return router.NewBadRequestError("Jumpbridge import failed: empty input.", logging.Fields{
			"jumpbridges": payload.Jumpbridges,
		})
	}

	lines := strings.Split(strings.ReplaceAll(payload.Jumpbridges, "\r", ""), "\n")
	if h.Service == nil {
		return router.NewInternalServerError("Jumpbridge service unavailable.", logging.Fields{
			"line_count": len(lines),
		})
	}
	count, updateErr := h.Service.UpdateFromLines(lines)
	logFields := logging.Fields{
		"line_count": len(lines),
		"count":      count,
	}

	if updateErr != nil {
		logFields["error"] = updateErr.Error()
		return router.NewInternalServerError("Failed to import jumpbridges.", logFields)
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
	if h.Service == nil {
		return router.NewInternalServerError("Jumpbridge service unavailable.", logging.Fields{
			"operation": "clear",
		})
	}
	_, updateErr := h.Service.UpdateFromLines([]string{})
	if updateErr != nil {
		return router.NewInternalServerError("Failed to clear jumpbridges.", logging.Fields{
			"error": updateErr.Error(),
		})
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
		return router.NewBadRequestError("Invalid payload.", logging.Fields{
			"error": bindErr.Error(),
		})
	}
	if payload.FromID <= 0 || payload.ToID <= 0 {
		return router.NewBadRequestError("Both from_id and to_id are required.", logging.Fields{
			"from_id": payload.FromID,
			"to_id":   payload.ToID,
		})
	}

	if h.Service == nil {
		return router.NewInternalServerError("Jumpbridge service unavailable.", logging.Fields{
			"operation": "add",
		})
	}
	changed, addErr := h.Service.AddPair(payload.FromID, payload.ToID)
	if addErr != nil {
		return router.NewBadRequestError(addErr.Error(), logging.Fields{
			"from_id": payload.FromID,
			"to_id":   payload.ToID,
			"error":   addErr.Error(),
		})
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
		return router.NewBadRequestError("Invalid payload.", logging.Fields{
			"error": bindErr.Error(),
		})
	}
	if payload.FromID <= 0 || payload.ToID <= 0 {
		return router.NewBadRequestError("Both from_id and to_id are required.", logging.Fields{
			"from_id": payload.FromID,
			"to_id":   payload.ToID,
		})
	}

	if h.Service == nil {
		return router.NewInternalServerError("Jumpbridge service unavailable.", logging.Fields{
			"operation": "remove",
		})
	}
	deleted, removeErr := h.Service.RemovePair(payload.FromID, payload.ToID)
	if removeErr != nil {
		return router.NewInternalServerError("Failed to remove jumpbridge.", logging.Fields{
			"from_id": payload.FromID,
			"to_id":   payload.ToID,
			"error":   removeErr.Error(),
		})
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
		return router.NewBadRequestError("Invalid payload.", logging.Fields{
			"error": bindErr.Error(),
		})
	}
	if payload.OldFromID <= 0 || payload.OldToID <= 0 || payload.FromID <= 0 || payload.ToID <= 0 {
		return router.NewBadRequestError("old_from_id, old_to_id, from_id, and to_id are required.", logging.Fields{
			"old_from_id": payload.OldFromID,
			"old_to_id":   payload.OldToID,
			"from_id":     payload.FromID,
			"to_id":       payload.ToID,
		})
	}

	if h.Service == nil {
		return router.NewInternalServerError("Jumpbridge service unavailable.", logging.Fields{
			"operation": "update",
		})
	}
	changed, updateErr := h.Service.UpdatePair(payload.OldFromID, payload.OldToID, payload.FromID, payload.ToID)
	if updateErr != nil {
		return router.NewBadRequestError(updateErr.Error(), logging.Fields{
			"old_from_id": payload.OldFromID,
			"old_to_id":   payload.OldToID,
			"from_id":     payload.FromID,
			"to_id":       payload.ToID,
			"error":       updateErr.Error(),
		})
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
