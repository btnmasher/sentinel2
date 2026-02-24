package staff

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/audit"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
	timercore "sentinel2/internal/timers"
)

func NewSovereigntyCampaignWatchlistHandler(app *pocketbase.PocketBase, auditSvc *audit.Service) *SovereigntyCampaignWatchlistHandler {
	return &SovereigntyCampaignWatchlistHandler{App: app, Audit: auditSvc}
}

func (h *SovereigntyCampaignWatchlistHandler) List(c *core.RequestEvent) error {
	records, err := h.App.FindRecordsByFilter(store.CollectionSovereigntyCampaignWatchlist, "", "alliance_name", 0, 0, nil)
	if err != nil {
		return router.NewInternalServerError("Failed to load sovereignty campaign watchlist.", logging.Fields{"error": err.Error()})
	}
	entities := make([]SovereigntyCampaignWatchlistEntityDTO, 0, len(records))
	for _, record := range records {
		entities = append(entities, sovereigntyWatchlistRecordToDTO(record))
	}
	return c.JSON(http.StatusOK, SovereigntyCampaignWatchlistResponse{Entities: entities})
}

type sovereigntyWatchlistCreatePayload struct {
	Hostility      string `json:"hostility"`
	AllianceID     int    `json:"alliance_id"`
	AllianceName   string `json:"alliance_name"`
	AllianceTicker string `json:"alliance_ticker"`
}

func (h *SovereigntyCampaignWatchlistHandler) Create(c *core.RequestEvent) error {
	payload := sovereigntyWatchlistCreatePayload{}
	if err := c.BindBody(&payload); err != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{"error": err.Error()})
	}
	hostility, allianceName, validationErr := validateSovWatchlistPayload(&payload)
	if validationErr != nil {
		return validationErr
	}

	key := strconv.Itoa(payload.AllianceID)
	existing, err := h.App.FindRecordsByFilter(
		store.CollectionSovereigntyCampaignWatchlist,
		"key = {:key}",
		"",
		1,
		0,
		dbx.Params{"key": key},
	)
	if err != nil {
		return router.NewInternalServerError("Failed to validate watchlist entity.", logging.Fields{"error": err.Error()})
	}
	if len(existing) > 0 {
		return c.JSON(http.StatusOK, SovereigntyCampaignWatchlistCreateResponse{ID: existing[0].Id})
	}

	collection, err := h.App.FindCollectionByNameOrId(store.CollectionSovereigntyCampaignWatchlist)
	if err != nil {
		return router.NewInternalServerError("Watchlist collection not available.", logging.Fields{"error": err.Error()})
	}

	record := core.NewRecord(collection)
	record.Set("key", key)
	record.Set("hostility", hostility)
	record.Set("alliance_id", payload.AllianceID)
	record.Set("alliance_name", allianceName)
	record.Set("alliance_ticker", strings.TrimSpace(payload.AllianceTicker))
	if err := h.App.Save(record); err != nil {
		return router.NewInternalServerError("Failed to save watchlist entity.", logging.Fields{"error": err.Error(), "key": key})
	}

	if h.Audit != nil {
		h.Audit.LogRequest(c, &audit.Event{
			Action:      audit.ActionStaffSovWatchlistAdd,
			Summary:     fmt.Sprintf("Added sovereignty watchlist alliance %s (%d)", allianceName, payload.AllianceID),
			TargetType:  audit.TargetTypeSovWatchEntity,
			TargetID:    record.Id,
			TargetLabel: allianceName,
			TargetMeta: map[string]any{
				"hostility":   hostility,
				"alliance_id": payload.AllianceID,
			},
		})
	}

	return c.JSON(http.StatusCreated, SovereigntyCampaignWatchlistCreateResponse{ID: record.Id})
}

func validateSovWatchlistPayload(payload *sovereigntyWatchlistCreatePayload) (hostility, allianceName string, err error) {
	if payload == nil {
		return "", "", router.NewBadRequestError("Invalid payload.", nil)
	}
	hostility = strings.TrimSpace(strings.ToLower(payload.Hostility))
	if hostility == "" {
		hostility = timercore.TimerStandingHostile
	}
	if !timercore.IsStandingType(hostility) {
		return "", "", router.NewBadRequestError("hostility must be one of ours/friendly/neutral/complicated/hostile.", logging.Fields{"hostility": payload.Hostility})
	}
	if payload.AllianceID <= 0 {
		return "", "", router.NewBadRequestError("alliance_id must be positive.", logging.Fields{"alliance_id": payload.AllianceID})
	}
	allianceName = strings.TrimSpace(payload.AllianceName)
	if allianceName == "" {
		return "", "", router.NewBadRequestError("alliance_name is required.", logging.Fields{"alliance_name": payload.AllianceName})
	}
	return hostility, allianceName, nil
}

func (h *SovereigntyCampaignWatchlistHandler) Delete(c *core.RequestEvent) error {
	id := strings.TrimSpace(c.Request.PathValue("id"))
	if id == "" {
		return router.NewBadRequestError("Missing id.", logging.Fields{"required_field": "id"})
	}
	record, err := h.App.FindRecordById(store.CollectionSovereigntyCampaignWatchlist, id)
	if err != nil {
		return router.NewNotFoundError("Watchlist entity not found.", logging.Fields{"id": id, "error": err.Error()})
	}
	if err := h.App.Delete(record); err != nil {
		return router.NewInternalServerError("Failed to delete watchlist entity.", logging.Fields{"id": id, "error": err.Error()})
	}
	if h.Audit != nil {
		h.Audit.LogRequest(c, &audit.Event{
			Action:      audit.ActionStaffSovWatchlistDelete,
			Summary:     fmt.Sprintf("Removed sovereignty watchlist alliance %s (%d)", record.GetString("alliance_name"), record.GetInt("alliance_id")),
			TargetType:  audit.TargetTypeSovWatchEntity,
			TargetID:    id,
			TargetLabel: record.GetString("alliance_name"),
			TargetMeta: map[string]any{
				"hostility":   record.GetString("hostility"),
				"alliance_id": record.GetInt("alliance_id"),
			},
		})
	}
	return c.NoContent(http.StatusNoContent)
}

func sovereigntyWatchlistRecordToDTO(record *core.Record) SovereigntyCampaignWatchlistEntityDTO {
	if record == nil {
		return SovereigntyCampaignWatchlistEntityDTO{}
	}
	return SovereigntyCampaignWatchlistEntityDTO{
		ID:             record.Id,
		Hostility:      record.GetString("hostility"),
		AllianceID:     record.GetInt("alliance_id"),
		AllianceName:   record.GetString("alliance_name"),
		AllianceTicker: record.GetString("alliance_ticker"),
	}
}
