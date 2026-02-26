package staff

import (
	"fmt"
	"net/http"
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

const (
	standingOwnerAlliance    = "alliance"
	standingOwnerCorporation = "corporation"
)

type organizationStandingCreatePayload struct {
	OwnerType         string `json:"owner_type"`
	Hostility         string `json:"hostility"`
	IncludeInSovSync  bool   `json:"include_in_sov_sync"`
	CorporationID     int    `json:"corporation_id"`
	CorporationName   string `json:"corporation_name"`
	CorporationTicker string `json:"corporation_ticker"`
	AllianceID        int    `json:"alliance_id"`
	AllianceName      string `json:"alliance_name"`
	AllianceTicker    string `json:"alliance_ticker"`
}

type organizationStandingUpdatePayload struct {
	Hostility        string `json:"hostility"`
	IncludeInSovSync bool   `json:"include_in_sov_sync"`
}

func NewOrganizationStandingsHandler(app *pocketbase.PocketBase, auditSvc *audit.Service) *OrganizationStandingsHandler {
	return &OrganizationStandingsHandler{App: app, Audit: auditSvc}
}

func (h *OrganizationStandingsHandler) List(c *core.RequestEvent) error {
	records, err := h.App.FindRecordsByFilter(store.CollectionOrganizationStandings, "", "owner_type,alliance_name,corporation_name", 0, 0, nil)
	if err != nil {
		return router.NewInternalServerError("Failed to load organization standings.", logging.Fields{"error": err.Error()})
	}
	entities := make([]OrganizationStandingDTO, 0, len(records))
	for _, record := range records {
		entities = append(entities, organizationStandingRecordToDTO(record))
	}
	return c.JSON(http.StatusOK, OrganizationStandingsResponse{Entities: entities})
}

func (h *OrganizationStandingsHandler) Create(c *core.RequestEvent) error {
	payload := organizationStandingCreatePayload{}
	if err := c.BindBody(&payload); err != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{"error": err.Error()})
	}
	input, validationErr := validateOrganizationStandingPayload(&payload)
	if validationErr != nil {
		return validationErr
	}

	filter := "owner_type = {:owner_type} && corporation_id = {:corporation_id}"
	params := dbx.Params{
		"owner_type":     input.OwnerType,
		"corporation_id": input.CorporationID,
		"alliance_id":    input.AllianceID,
	}
	if input.OwnerType == standingOwnerAlliance {
		filter = "owner_type = {:owner_type} && alliance_id = {:alliance_id}"
	}
	existing, err := h.App.FindRecordsByFilter(store.CollectionOrganizationStandings, filter, "", 1, 0, params)
	if err != nil {
		return router.NewInternalServerError("Failed to validate organization standing.", logging.Fields{"error": err.Error()})
	}
	if len(existing) > 0 {
		return router.NewBadRequestError("Organization standing already exists.", logging.Fields{
			"owner_type":     input.OwnerType,
			"corporation_id": input.CorporationID,
			"alliance_id":    input.AllianceID,
		})
	}

	collection, err := h.App.FindCollectionByNameOrId(store.CollectionOrganizationStandings)
	if err != nil {
		return router.NewInternalServerError("Organization standings collection not available.", logging.Fields{"error": err.Error()})
	}
	record := core.NewRecord(collection)
	applyOrganizationStandingRecord(record, &input)
	if saveErr := h.App.Save(record); saveErr != nil {
		return router.NewInternalServerError("Failed to save organization standing.", logging.Fields{"error": saveErr.Error()})
	}

	if h.Audit != nil {
		h.Audit.LogRequest(c, &audit.Event{
			Action:      audit.ActionStaffSovWatchlistAdd,
			Summary:     fmt.Sprintf("Added organization standing %s (%s)", standingInputDisplayName(&input), input.OwnerType),
			TargetType:  audit.TargetTypeSovWatchEntity,
			TargetID:    record.Id,
			TargetLabel: standingInputDisplayName(&input),
			TargetMeta: map[string]any{
				"owner_type":          input.OwnerType,
				"hostility":           input.Hostility,
				"corporation_id":      input.CorporationID,
				"alliance_id":         input.AllianceID,
				"include_in_sov_sync": input.IncludeInSovSync,
			},
		})
	}

	return c.JSON(http.StatusCreated, OrganizationStandingCreateResponse{ID: record.Id})
}

func (h *OrganizationStandingsHandler) Update(c *core.RequestEvent) error {
	id := strings.TrimSpace(c.Request.PathValue("id"))
	if id == "" {
		return router.NewBadRequestError("Missing id.", logging.Fields{"required_field": "id"})
	}

	record, err := h.App.FindRecordById(store.CollectionOrganizationStandings, id)
	if err != nil {
		return router.NewNotFoundError("Organization standing not found.", logging.Fields{"id": id, "error": err.Error()})
	}

	payload := organizationStandingUpdatePayload{}
	if err := c.BindBody(&payload); err != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{"error": err.Error()})
	}
	input, validationErr := validateOrganizationStandingUpdatePayload(&payload)
	if validationErr != nil {
		return validationErr
	}
	if input.IncludeInSovSync && record.GetInt("alliance_id") <= 0 {
		return router.NewBadRequestError("include_in_sov_sync requires an alliance to be set.", nil)
	}

	record.Set("hostility", input.Hostility)
	record.Set("include_in_sov_sync", input.IncludeInSovSync)
	if saveErr := h.App.Save(record); saveErr != nil {
		return router.NewInternalServerError("Failed to update organization standing.", logging.Fields{"error": saveErr.Error()})
	}

	if h.Audit != nil {
		h.Audit.LogRequest(c, &audit.Event{
			Action:      audit.ActionStaffSovWatchlistUpdate,
			Summary:     fmt.Sprintf("Updated organization standing %s (%s)", standingRecordDisplayName(record), record.GetString("owner_type")),
			TargetType:  audit.TargetTypeSovWatchEntity,
			TargetID:    record.Id,
			TargetLabel: standingRecordDisplayName(record),
			TargetMeta: map[string]any{
				"owner_type":          record.GetString("owner_type"),
				"hostility":           record.GetString("hostility"),
				"corporation_id":      record.GetInt("corporation_id"),
				"alliance_id":         record.GetInt("alliance_id"),
				"include_in_sov_sync": record.GetBool("include_in_sov_sync"),
			},
		})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *OrganizationStandingsHandler) Delete(c *core.RequestEvent) error {
	id := strings.TrimSpace(c.Request.PathValue("id"))
	if id == "" {
		return router.NewBadRequestError("Missing id.", logging.Fields{"required_field": "id"})
	}
	record, err := h.App.FindRecordById(store.CollectionOrganizationStandings, id)
	if err != nil {
		return router.NewNotFoundError("Organization standing not found.", logging.Fields{"id": id, "error": err.Error()})
	}
	if err := h.App.Delete(record); err != nil {
		return router.NewInternalServerError("Failed to delete organization standing.", logging.Fields{"id": id, "error": err.Error()})
	}
	if h.Audit != nil {
		h.Audit.LogRequest(c, &audit.Event{
			Action:      audit.ActionStaffSovWatchlistDelete,
			Summary:     fmt.Sprintf("Removed organization standing %s (%s)", standingRecordDisplayName(record), record.GetString("owner_type")),
			TargetType:  audit.TargetTypeSovWatchEntity,
			TargetID:    id,
			TargetLabel: standingRecordDisplayName(record),
			TargetMeta: map[string]any{
				"owner_type":          record.GetString("owner_type"),
				"hostility":           record.GetString("hostility"),
				"corporation_id":      record.GetInt("corporation_id"),
				"alliance_id":         record.GetInt("alliance_id"),
				"include_in_sov_sync": record.GetBool("include_in_sov_sync"),
			},
		})
	}
	return c.NoContent(http.StatusNoContent)
}

type validatedOrganizationStandingInput struct {
	OwnerType         string
	Hostility         string
	IncludeInSovSync  bool
	CorporationID     int
	CorporationName   string
	CorporationTicker string
	AllianceID        int
	AllianceName      string
	AllianceTicker    string
}

func standingInputDisplayName(input *validatedOrganizationStandingInput) string {
	if input == nil {
		return ""
	}
	if input.OwnerType == standingOwnerAlliance {
		return input.AllianceName
	}
	return input.CorporationName
}

func validateOrganizationStandingPayload(payload *organizationStandingCreatePayload) (validatedOrganizationStandingInput, error) {
	if payload == nil {
		return validatedOrganizationStandingInput{}, router.NewBadRequestError("Invalid payload.", nil)
	}
	input := validatedOrganizationStandingInput{
		OwnerType:         strings.TrimSpace(strings.ToLower(payload.OwnerType)),
		Hostility:         strings.TrimSpace(strings.ToLower(payload.Hostility)),
		IncludeInSovSync:  payload.IncludeInSovSync,
		CorporationID:     payload.CorporationID,
		CorporationName:   strings.TrimSpace(payload.CorporationName),
		CorporationTicker: strings.TrimSpace(payload.CorporationTicker),
		AllianceID:        payload.AllianceID,
		AllianceName:      strings.TrimSpace(payload.AllianceName),
		AllianceTicker:    strings.TrimSpace(payload.AllianceTicker),
	}
	if input.Hostility == "" {
		input.Hostility = timercore.TimerStandingHostile
	}
	if !timercore.IsStandingType(input.Hostility) {
		return validatedOrganizationStandingInput{}, router.NewBadRequestError("hostility must be one of ours/friendly/neutral/complicated/hostile.", logging.Fields{"hostility": payload.Hostility})
	}
	switch input.OwnerType {
	case standingOwnerAlliance:
		if input.AllianceID <= 0 {
			return validatedOrganizationStandingInput{}, router.NewBadRequestError("alliance_id must be positive for alliance entries.", logging.Fields{"alliance_id": payload.AllianceID})
		}
		if input.AllianceName == "" {
			return validatedOrganizationStandingInput{}, router.NewBadRequestError("alliance_name is required for alliance entries.", logging.Fields{"alliance_name": payload.AllianceName})
		}
		input.CorporationID = 0
		input.CorporationName = ""
		input.CorporationTicker = ""
	case standingOwnerCorporation:
		if input.CorporationID <= 0 {
			return validatedOrganizationStandingInput{}, router.NewBadRequestError("corporation_id must be positive for corporation entries.", logging.Fields{"corporation_id": payload.CorporationID})
		}
		if input.CorporationName == "" {
			return validatedOrganizationStandingInput{}, router.NewBadRequestError("corporation_name is required for corporation entries.", logging.Fields{"corporation_name": payload.CorporationName})
		}
	default:
		return validatedOrganizationStandingInput{}, router.NewBadRequestError("owner_type must be 'alliance' or 'corporation'.", logging.Fields{"owner_type": payload.OwnerType})
	}
	if input.IncludeInSovSync && input.AllianceID <= 0 {
		return validatedOrganizationStandingInput{}, router.NewBadRequestError("include_in_sov_sync requires an alliance to be set.", nil)
	}
	return input, nil
}

func validateOrganizationStandingUpdatePayload(payload *organizationStandingUpdatePayload) (validatedOrganizationStandingInput, error) {
	if payload == nil {
		return validatedOrganizationStandingInput{}, router.NewBadRequestError("Invalid payload.", nil)
	}
	input := validatedOrganizationStandingInput{
		Hostility:        strings.TrimSpace(strings.ToLower(payload.Hostility)),
		IncludeInSovSync: payload.IncludeInSovSync,
	}
	if input.Hostility == "" {
		input.Hostility = timercore.TimerStandingHostile
	}
	if !timercore.IsStandingType(input.Hostility) {
		return validatedOrganizationStandingInput{}, router.NewBadRequestError("hostility must be one of ours/friendly/neutral/complicated/hostile.", logging.Fields{"hostility": payload.Hostility})
	}
	return input, nil
}

func applyOrganizationStandingRecord(record *core.Record, input *validatedOrganizationStandingInput) {
	if record == nil || input == nil {
		return
	}
	record.Set("owner_type", input.OwnerType)
	record.Set("hostility", input.Hostility)
	record.Set("include_in_sov_sync", input.IncludeInSovSync)
	record.Set("corporation_id", input.CorporationID)
	record.Set("corporation_name", input.CorporationName)
	record.Set("corporation_ticker", input.CorporationTicker)
	record.Set("alliance_id", input.AllianceID)
	record.Set("alliance_name", input.AllianceName)
	record.Set("alliance_ticker", input.AllianceTicker)
}

func standingRecordDisplayName(record *core.Record) string {
	if record == nil {
		return ""
	}
	if strings.TrimSpace(record.GetString("owner_type")) == standingOwnerAlliance {
		return strings.TrimSpace(record.GetString("alliance_name"))
	}
	return strings.TrimSpace(record.GetString("corporation_name"))
}

func organizationStandingRecordToDTO(record *core.Record) OrganizationStandingDTO {
	if record == nil {
		return OrganizationStandingDTO{}
	}
	return OrganizationStandingDTO{
		ID:                record.Id,
		OwnerType:         strings.TrimSpace(record.GetString("owner_type")),
		Hostility:         strings.TrimSpace(record.GetString("hostility")),
		IncludeInSovSync:  record.GetBool("include_in_sov_sync"),
		CorporationID:     record.GetInt("corporation_id"),
		CorporationName:   strings.TrimSpace(record.GetString("corporation_name")),
		CorporationTicker: strings.TrimSpace(record.GetString("corporation_ticker")),
		AllianceID:        record.GetInt("alliance_id"),
		AllianceName:      strings.TrimSpace(record.GetString("alliance_name")),
		AllianceTicker:    strings.TrimSpace(record.GetString("alliance_ticker")),
	}
}
