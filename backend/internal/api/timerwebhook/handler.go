package timerwebhook

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/format"
	"sentinel2/internal/logging"
	timerssvc "sentinel2/internal/timers"
)

type Handler struct {
	Service *timerssvc.Service
}

type timerPayload struct {
	Title                  *string `json:"title"`
	SystemID               *int    `json:"system_id"`
	Standing               *string `json:"standing_type"`
	TimerKind              *string `json:"timer_kind"`
	StructureType          *string `json:"structure_type"`
	StageLabel             *string `json:"stage_label"`
	PlanetID               *int    `json:"planet_id"`
	PlanetName             *string `json:"planet_name"`
	MoonID                 *int    `json:"moon_id"`
	MoonName               *string `json:"moon_name"`
	OwnerCorporationID     *int    `json:"owner_corporation_id"`
	OwnerCorporationName   *string `json:"owner_corporation_name"`
	OwnerCorporationTicker *string `json:"owner_corporation_ticker"`
	OwnerAllianceID        *int    `json:"owner_alliance_id"`
	OwnerAllianceName      *string `json:"owner_alliance_name"`
	OwnerAllianceTicker    *string `json:"owner_alliance_ticker"`
	SkyhookFullnessPct     *int    `json:"skyhook_fullness_pct"`
	AttackersScorePct      *int    `json:"attackers_score_pct"`
	DefenderScorePct       *int    `json:"defender_score_pct"`
	Stage                  *int    `json:"stage"`
	TotalStages            *int    `json:"total_stages"`
	Severity               *string `json:"severity"`
	Status                 *string `json:"status"`
	ExpiresAt              *string `json:"expires_at"`
	SourceRef              *string `json:"source_ref"`
	Notes                  *string `json:"notes"`
	RawText                *string `json:"raw_text"`
	ReplacementAction      *string `json:"replacement_action"`
}

type createPayload struct {
	timerPayload

	ID string `json:"id"`
}

type patchPayload struct {
	timerPayload

	ID *string `json:"id"`
}

type webhookResponse struct {
	Operation string `json:"operation"`
	ID        string `json:"id"`
}

func NewHandler(service *timerssvc.Service) *Handler {
	return &Handler{Service: service}
}

func (h *Handler) Create(c *core.RequestEvent) error {
	payload := createPayload{}

	if err := c.BindBody(&payload); err != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{"error": err.Error()})
	}

	webhookID := strings.TrimSpace(payload.ID)
	if webhookID == "" {
		return router.NewBadRequestError("Missing id.", nil)
	}

	existing, err := h.Service.FindByWebhookID(webhookID)
	if err != nil && !errors.Is(err, timerssvc.ErrTimerNotFound) {
		return router.NewInternalServerError("Failed to load timer.", logging.Fields{"error": err.Error()})
	}

	if existing != nil {
		return router.NewApiError(http.StatusConflict, "Timer already exists.", logging.Fields{"id": webhookID})
	}

	_, createErr := h.create(webhookID, &payload)
	if createErr != nil {
		return createErr
	}
	return c.JSON(http.StatusCreated, webhookResponse{
		Operation: "created",
		ID:        webhookID,
	})
}

func (h *Handler) Patch(c *core.RequestEvent) error {
	webhookID := strings.TrimSpace(c.Request.PathValue("id"))
	if webhookID == "" {
		return router.NewBadRequestError("Missing id.", nil)
	}

	payload := patchPayload{}

	if err := c.BindBody(&payload); err != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{"error": err.Error()})
	}

	if payload.ID != nil && strings.TrimSpace(*payload.ID) != webhookID {
		return router.NewBadRequestError("Path id must match body id.", nil)
	}

	existing, err := h.Service.FindByWebhookID(webhookID)
	if err != nil {
		if errors.Is(err, timerssvc.ErrTimerNotFound) {
			return router.NewNotFoundError("Timer not found.", logging.Fields{"id": webhookID})
		}
		return router.NewInternalServerError("Failed to load timer.", logging.Fields{"error": err.Error(), "id": webhookID})
	}

	_, updateErr := h.update(existing.Id, webhookID, &payload)
	if updateErr != nil {
		return updateErr
	}
	return c.JSON(http.StatusOK, webhookResponse{
		Operation: "updated",
		ID:        webhookID,
	})
}

func (h *Handler) Delete(c *core.RequestEvent) error {
	webhookID := strings.TrimSpace(c.Request.PathValue("id"))
	if webhookID == "" {
		return router.NewBadRequestError("Missing id.", nil)
	}

	if err := h.Service.DeleteByWebhookID(webhookID); err != nil {
		return router.NewInternalServerError("Failed to delete timer.", logging.Fields{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) create(webhookID string, payload *createPayload) (*core.Record, error) {
	if payload == nil {
		return nil, router.NewBadRequestError("Invalid payload.", nil)
	}

	if payload.SystemID == nil || *payload.SystemID <= 0 {
		return nil, router.NewBadRequestError("Missing system_id.", nil)
	}

	expiresAt, err := parseRequiredExpiresAt(payload.ExpiresAt)
	if err != nil {
		return nil, err
	}

	input := &timerssvc.CreateInput{
		WebhookID:              webhookID,
		Title:                  stringValue(payload.Title),
		SystemID:               intValue(payload.SystemID),
		Standing:               stringValue(payload.Standing),
		TimerKind:              stringValue(payload.TimerKind),
		StructureType:          stringValue(payload.StructureType),
		StageLabel:             stringValue(payload.StageLabel),
		PlanetID:               intValue(payload.PlanetID),
		PlanetName:             stringValue(payload.PlanetName),
		MoonID:                 intValue(payload.MoonID),
		MoonName:               stringValue(payload.MoonName),
		OwnerCorporationID:     intValue(payload.OwnerCorporationID),
		OwnerCorporationName:   stringValue(payload.OwnerCorporationName),
		OwnerCorporationTicker: stringValue(payload.OwnerCorporationTicker),
		OwnerAllianceID:        intValue(payload.OwnerAllianceID),
		OwnerAllianceName:      stringValue(payload.OwnerAllianceName),
		OwnerAllianceTicker:    stringValue(payload.OwnerAllianceTicker),
		SkyhookFullnessPct:     payload.SkyhookFullnessPct,
		AttackersScorePct:      payload.AttackersScorePct,
		DefenderScorePct:       payload.DefenderScorePct,
		Stage:                  intValue(payload.Stage),
		TotalStages:            intValue(payload.TotalStages),
		Severity:               stringValue(payload.Severity),
		Status:                 stringValue(payload.Status),
		ExpiresAt:              expiresAt,
		SourceRef:              stringValue(payload.SourceRef),
		Notes:                  stringValue(payload.Notes),
		RawText:                stringValue(payload.RawText),
		ReplacementAction:      stringValue(payload.ReplacementAction),
	}
	record, createErr := h.Service.CreateWebhook(input)
	if createErr != nil {
		if isWebhookUserInputError(createErr) {
			return nil, router.NewBadRequestError("Invalid timer payload.", logging.Fields{"error": createErr.Error()})
		}
		return nil, router.NewInternalServerError("Failed to save timer.", logging.Fields{"error": createErr.Error()})
	}
	return record, nil
}

func (h *Handler) update(recordID, webhookID string, payload *patchPayload) (*core.Record, error) {
	expiresAt, err := parseOptionalExpiresAt(payload.ExpiresAt)
	if err != nil {
		return nil, err
	}

	record, updateErr := h.Service.Update(recordID, &timerssvc.UpdateInput{
		Title:                  payload.Title,
		Standing:               payload.Standing,
		TimerKind:              payload.TimerKind,
		StructureType:          payload.StructureType,
		StageLabel:             payload.StageLabel,
		PlanetID:               payload.PlanetID,
		PlanetName:             payload.PlanetName,
		MoonID:                 payload.MoonID,
		MoonName:               payload.MoonName,
		OwnerCorporationID:     payload.OwnerCorporationID,
		OwnerCorporationName:   payload.OwnerCorporationName,
		OwnerCorporationTicker: payload.OwnerCorporationTicker,
		OwnerAllianceID:        payload.OwnerAllianceID,
		OwnerAllianceName:      payload.OwnerAllianceName,
		OwnerAllianceTicker:    payload.OwnerAllianceTicker,
		SkyhookFullnessPct:     payload.SkyhookFullnessPct,
		AttackersScorePct:      payload.AttackersScorePct,
		DefenderScorePct:       payload.DefenderScorePct,
		Stage:                  payload.Stage,
		TotalStages:            payload.TotalStages,
		Severity:               payload.Severity,
		Status:                 payload.Status,
		ExpiresAt:              expiresAt,
		SourceRef:              payload.SourceRef,
		Notes:                  payload.Notes,
		RawText:                payload.RawText,
		ReplacementAction:      payload.ReplacementAction,
	})
	if updateErr != nil {
		if isWebhookUserInputError(updateErr) {
			return nil, router.NewBadRequestError("Invalid timer payload.", logging.Fields{"error": updateErr.Error(), "id": webhookID})
		}
		return nil, router.NewInternalServerError("Failed to update timer.", logging.Fields{"error": updateErr.Error(), "id": webhookID})
	}
	return record, nil
}

func parseRequiredExpiresAt(raw *string) (time.Time, error) {
	if raw == nil {
		return time.Time{}, router.NewBadRequestError("Missing expires_at.", nil)
	}
	parsed, err := format.ParseDateTimeFlexibleUTC(*raw)
	if err != nil {
		return time.Time{}, router.NewBadRequestError("Invalid expires_at.", logging.Fields{"expires_at": *raw})
	}
	return parsed, nil
}

func parseOptionalExpiresAt(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	parsed, err := format.ParseDateTimeFlexibleUTC(*raw)
	if err != nil {
		return nil, router.NewBadRequestError("Invalid expires_at.", logging.Fields{"expires_at": *raw})
	}
	return &parsed, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func isWebhookUserInputError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, timerssvc.ErrMissingWebhookID) ||
		errors.Is(err, timerssvc.ErrMissingSystem) ||
		errors.Is(err, timerssvc.ErrMissingStructureType) ||
		errors.Is(err, timerssvc.ErrSystemNotFound) ||
		errors.Is(err, timerssvc.ErrMoonRequired) ||
		errors.Is(err, timerssvc.ErrPlanetRequired) ||
		errors.Is(err, timerssvc.ErrInvalidStageLabel) ||
		errors.Is(err, timerssvc.ErrInvalidTimerContext) ||
		errors.Is(err, timerssvc.ErrInvalidSkyhookFullnessPercentage) ||
		errors.Is(err, timerssvc.ErrMissingExpiresAt) ||
		errors.Is(err, timerssvc.ErrMissingCreateInput) ||
		errors.Is(err, timerssvc.ErrMissingUpdateInput) ||
		errors.Is(err, timerssvc.ErrMissingInput)
}
