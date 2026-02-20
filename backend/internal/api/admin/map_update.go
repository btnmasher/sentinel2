package admin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/audit"
	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
	"sentinel2/internal/mapdata"
)

func NewMapUpdateHandler(app *pocketbase.PocketBase, auditSvc *audit.Service) *MapUpdateHandler {
	return &MapUpdateHandler{App: app, Audit: auditSvc}
}

func (h *MapUpdateHandler) RunAll(c *core.RequestEvent) error {
	actorID := ""
	if c.Auth != nil {
		actorID = c.Auth.Id
	}
	runner := jobs.NewRunner(h.App, &jobs.RunOptions{
		JobName: mapdata.JobMapDataUpdate,
		JobOptions: jobs.JobOptions{
			Kind:    "map_data_update",
			Trigger: jobs.TriggerAdminManual,
			ActorID: actorID,
		},
		Timeout: jobs.NoTimeout,
	})
	jobID := runner.JobID()
	logFields := logging.Fields{
		"job_id": jobID,
		"cron":   jobs.TriggerAdminManual,
		"force":  true,
	}
	if c.Auth != nil {
		logFields["user_id"] = c.Auth.Id
	}
	logging.WithRequest(h.App, c).
		WithFields(logFields).
		Info("map data update requested")
	go func(jobID string, actorID string) {
		baseCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		localRunner := jobs.NewRunner(h.App, &jobs.RunOptions{
			JobID: jobID,
			JobOptions: jobs.JobOptions{
				Kind:    "map_data_update",
				Trigger: jobs.TriggerAdminManual,
				ActorID: actorID,
			},
			Timeout: jobs.NoTimeout,
			Parent:  baseCtx,
		})
		mapdata.RunMapDataUpdateWithContext(baseCtx, h.App, localRunner, jobs.TriggerAdminManual, true)
	}(jobID, actorID)
	if h.Audit != nil {
		h.Audit.LogRequest(c, &audit.Event{
			Action:      audit.ActionAdminMapDataRunAll,
			Summary:     "Triggered full map data update",
			TargetType:  audit.TargetTypeJob,
			TargetID:    jobID,
			TargetLabel: "all",
			TargetMeta: map[string]any{
				"job_id": jobID,
				"step":   "all",
			},
		})
	}

	return c.JSON(http.StatusAccepted, mapDataResponse{
		JobID: jobID,
		Step:  "all",
	})
}

func (h *MapUpdateHandler) RunSDEImport(c *core.RequestEvent) error {
	return h.runStep(c, mapdata.StepSDEImport)
}

func (h *MapUpdateHandler) RunDotlan(c *core.RequestEvent) error {
	return h.runStep(c, mapdata.StepDotlanImport)
}

func (h *MapUpdateHandler) RunPlanets(c *core.RequestEvent) error {
	return h.runStep(c, mapdata.StepPlanetsImport)
}

func (h *MapUpdateHandler) RunMoons(c *core.RequestEvent) error {
	return h.runStep(c, mapdata.StepMoonsImport)
}

func (h *MapUpdateHandler) RunEve2DPositions(c *core.RequestEvent) error {
	return h.runStep(c, mapdata.StepEve2DPositions)
}

func (h *MapUpdateHandler) RunRealPositions(c *core.RequestEvent) error {
	return h.runStep(c, mapdata.StepRealPositions)
}

func (h *MapUpdateHandler) RunMetroPositions(c *core.RequestEvent) error {
	return h.runStep(c, mapdata.StepMetroPositions)
}

func (h *MapUpdateHandler) RunSystemGraphs(c *core.RequestEvent) error {
	return h.runStep(c, mapdata.StepBuildGraph)
}

func (h *MapUpdateHandler) RunRegionLayout(c *core.RequestEvent) error {
	return h.runStep(c, mapdata.StepRegionLayout)
}

func (h *MapUpdateHandler) runStep(c *core.RequestEvent, step string) error {
	actorID := ""
	if c.Auth != nil {
		actorID = c.Auth.Id
	}
	jobID := mapdata.TriggerMapDataStep(h.App, mapdata.StepTriggerOptions{
		Step:    step,
		Trigger: jobs.TriggerAdminManual,
		ActorID: actorID,
		JobName: mapdata.JobMapDataStep,
		Logger:  logging.WithRequest(h.App, c),
	})
	if h.Audit != nil {
		h.Audit.LogRequest(c, &audit.Event{
			Action:      audit.ActionAdminMapDataRunStep,
			Summary:     fmt.Sprintf("Triggered map data step %s", step),
			TargetType:  audit.TargetTypeJob,
			TargetID:    jobID,
			TargetLabel: step,
			TargetMeta: map[string]any{
				"job_id": jobID,
				"step":   step,
			},
		})
	}

	return c.JSON(http.StatusAccepted, mapDataResponse{
		JobID: jobID,
		Step:  step,
	})
}
