package timers

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/audit"
	"sentinel2/internal/auth"
	"sentinel2/internal/format"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
	timerssvc "sentinel2/internal/timers"
)

const defaultTimerListLimit = 200
const (
	defaultSearchSystemsLimit  = 20
	defaultSearchEntitiesLimit = 20
	defaultMoonListLimit       = 200
	scorePercentMax            = 100
	tokenRefreshLeeway         = 2 * time.Minute
)

func NewHandler(service *timerssvc.Service, auditSvc *audit.Service, provider auth.Provider) *Handler {
	return &Handler{Service: service, Audit: auditSvc, Provider: provider}
}

func (h *Handler) List(c *core.RequestEvent) error {
	values := c.Request.URL.Query()
	statuses := format.SplitTokens(strings.TrimSpace(values.Get("status")))
	regionIDs := splitInts(values.Get("region_ids"))

	var fromAt *time.Time
	if fromRaw := strings.TrimSpace(values.Get("from")); fromRaw != "" {
		parsed, err := format.ParseDateTimeFlexibleUTC(fromRaw)
		if err == nil {
			fromAt = &parsed
		}
	}

	var toAt *time.Time
	if toRaw := strings.TrimSpace(values.Get("to")); toRaw != "" {
		parsed, err := format.ParseDateTimeFlexibleUTC(toRaw)
		if err == nil {
			toAt = &parsed
		}
	}

	records, err := h.Service.List(timerssvc.ListInput{
		Statuses:  statuses,
		RegionIDs: regionIDs,
		From:      fromAt,
		To:        toAt,
		Limit:     parsePositiveInt(values.Get("limit"), defaultTimerListLimit),
	})
	if err != nil {
		return router.NewInternalServerError("Failed to load timers.", logging.Fields{"error": err.Error()})
	}

	out := make([]timerDTO, 0, len(records))
	for _, record := range records {
		out = append(out, timerToDTO(record))
	}
	return c.JSON(http.StatusOK, listResponse{Timers: out})
}

func (h *Handler) Parse(c *core.RequestEvent) error {
	payload := struct {
		Text string `json:"text"`
	}{}

	if err := c.BindBody(&payload); err != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{"error": err.Error()})
	}
	result, err := h.Service.ParseText(payload.Text)
	if err != nil {
		return router.NewBadRequestError("Unable to parse timer text.", logging.Fields{"error": err.Error()})
	}
	systemID, _ := h.Service.ResolveSystemID(result.System)
	return c.JSON(http.StatusOK, parseResponse{
		Title:      result.Title,
		System:     result.System,
		SystemID:   systemID,
		TimerKind:  result.TimerKind,
		Standing:   timerssvc.TimerStandingHostile,
		ExpiresAt:  result.ExpiresAt.Format(time.RFC3339),
		RawExtract: result.Raw,
	})
}

func (h *Handler) SearchSystems(c *core.RequestEvent) error {
	query := strings.TrimSpace(c.Request.URL.Query().Get("query"))
	limit := parsePositiveInt(c.Request.URL.Query().Get("limit"), defaultSearchSystemsLimit)
	systems, err := h.Service.SearchSystems(query, limit)
	if err != nil {
		return router.NewInternalServerError("Failed to search systems.", logging.Fields{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, systemSearchResponse{Systems: systems})
}

func (h *Handler) SearchEntities(c *core.RequestEvent) error {
	query := strings.TrimSpace(c.Request.URL.Query().Get("query"))
	limit := parsePositiveInt(c.Request.URL.Query().Get("limit"), defaultSearchEntitiesLimit)
	scope := parseEntitySearchScope(c.Request.URL.Query().Get("scope"))
	requester, requesterErr := h.entitySearchRequester(c)
	if requesterErr != nil {
		return router.NewInternalServerError("Failed to resolve requester context.", logging.Fields{"error": requesterErr.Error()})
	}
	entities, err := h.Service.SearchEntitiesWithScope(c.Request.Context(), query, limit, requester, scope)
	if err != nil {
		return router.NewInternalServerError("Failed to search entities.", logging.Fields{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, entitySearchResponse{Entities: entities})
}

func parseEntitySearchScope(raw string) timerssvc.EntitySearchScope {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case string(timerssvc.EntitySearchScopeAlliance):
		return timerssvc.EntitySearchScopeAlliance
	case string(timerssvc.EntitySearchScopeCorporation):
		return timerssvc.EntitySearchScopeCorporation
	default:
		return timerssvc.EntitySearchScopeBoth
	}
}

func (h *Handler) entitySearchRequester(c *core.RequestEvent) (*timerssvc.EntitySearchRequester, error) {
	if h == nil || h.Service == nil || h.Service.App == nil || c == nil || c.Auth == nil {
		return nil, nil
	}

	record, err := h.findMainCharacter(c.Auth.Id)
	if err != nil {
		return nil, err
	}

	if record == nil {
		return nil, nil
	}

	if h.needsEntitySearchTokenRefresh(record) {
		refreshed := h.refreshEntitySearchToken(c, c.Auth)
		if !refreshed {
			return nil, nil
		}
		record, err = h.findMainCharacter(c.Auth.Id)
		if err != nil {
			return nil, err
		}
		if record == nil {
			return nil, nil
		}
	}

	token := strings.TrimSpace(record.GetString("oauth_access_token"))
	characterID := record.GetInt("eve_character_id")
	if characterID <= 0 || token == "" {
		return nil, nil
	}
	return &timerssvc.EntitySearchRequester{
		CharacterID: characterID,
		AccessToken: token,
	}, nil
}

func (h *Handler) findMainCharacter(userID string) (*core.Record, error) {
	records, err := h.Service.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user} && is_main = true",
		"",
		1,
		0,
		map[string]any{"user": userID},
	)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

func (h *Handler) needsEntitySearchTokenRefresh(record *core.Record) bool {
	if h == nil || h.Provider == nil || record == nil {
		return false
	}
	expiresAt := record.GetDateTime("oauth_access_expires_at")
	return expiresAt.IsZero() || expiresAt.Time().Before(time.Now().Add(tokenRefreshLeeway))
}

func (h *Handler) refreshEntitySearchToken(c *core.RequestEvent, authRecord *core.Record) bool {
	if h == nil || h.Provider == nil || c == nil || authRecord == nil {
		return false
	}
	_, refreshErr := h.Provider.Refresh(c.Request.Context(), authRecord)
	if errors.Is(refreshErr, auth.ErrAccessDenied) {
		return false
	}

	if refreshErr != nil {
		h.Service.App.Logger().Warn("timer entity search token refresh failed", "error", refreshErr.Error(), "user_id", authRecord.Id)
		return false
	}
	return true
}

func (h *Handler) SearchMoons(c *core.RequestEvent) error {
	systemID := parsePositiveInt(c.Request.URL.Query().Get("system_id"), 0)
	limit := parsePositiveInt(c.Request.URL.Query().Get("limit"), defaultMoonListLimit)
	moons, err := h.Service.ListMoonsBySystem(systemID, limit)
	if err != nil {
		return router.NewInternalServerError("Failed to load moons.", logging.Fields{"error": err.Error(), "system_id": systemID})
	}
	return c.JSON(http.StatusOK, moonSearchResponse{Moons: moons})
}

func (h *Handler) SearchPlanets(c *core.RequestEvent) error {
	systemID := parsePositiveInt(c.Request.URL.Query().Get("system_id"), 0)
	limit := parsePositiveInt(c.Request.URL.Query().Get("limit"), defaultMoonListLimit)
	planets, err := h.Service.ListPlanetsBySystem(systemID, limit)
	if err != nil {
		return router.NewInternalServerError("Failed to load planets.", logging.Fields{"error": err.Error(), "system_id": systemID})
	}
	return c.JSON(http.StatusOK, planetSearchResponse{Planets: planets})
}

func (h *Handler) Create(c *core.RequestEvent) error {
	payload := createPayload{}

	if err := c.BindBody(&payload); err != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{"error": err.Error()})
	}

	expiresAt, err := format.ParseDateTimeFlexibleUTC(payload.ExpiresAt)
	if err != nil {
		return router.NewBadRequestError("Invalid expires_at.", logging.Fields{"expires_at": payload.ExpiresAt})
	}

	record, err := h.Service.Create(&timerssvc.CreateInput{
		Title:                  payload.Title,
		SystemID:               payload.SystemID,
		System:                 payload.System,
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
		Stage:                  payload.Stage,
		TotalStages:            payload.TotalStages,
		Severity:               payload.Severity,
		Status:                 payload.Status,
		ExpiresAt:              expiresAt,
		Source:                 payload.Source,
		SourceRef:              payload.SourceRef,
		Notes:                  payload.Notes,
		RawText:                payload.RawText,
		ReplacementAction:      payload.ReplacementAction,
	}, c.Auth)
	if err != nil {
		if isUserInputError(err) {
			return router.NewBadRequestError("Invalid timer payload.", logging.Fields{"error": err.Error(), "system_id": payload.SystemID, "system": payload.System})
		}
		return router.NewInternalServerError("Failed to save timer.", logging.Fields{"error": err.Error()})
	}

	h.logAudit(c, audit.ActionTimerCreate, "Created timer", record)
	return c.JSON(http.StatusCreated, timerToDTO(record))
}

func (h *Handler) Update(c *core.RequestEvent) error {
	id := strings.TrimSpace(c.Request.PathValue("id"))
	if id == "" {
		return router.NewBadRequestError("Missing id.", nil)
	}

	payload := updatePayload{}

	if err := c.BindBody(&payload); err != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{"error": err.Error()})
	}

	var expiresAt *time.Time
	if payload.ExpiresAt != nil {
		parsed, err := format.ParseDateTimeFlexibleUTC(*payload.ExpiresAt)
		if err != nil {
			return router.NewBadRequestError("Invalid expires_at.", logging.Fields{"expires_at": *payload.ExpiresAt})
		}
		expiresAt = &parsed
	}

	record, err := h.Service.Update(id, &timerssvc.UpdateInput{
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
	if err != nil {
		if isUserInputError(err) {
			return router.NewBadRequestError("Invalid timer payload.", logging.Fields{"error": err.Error()})
		}
		return h.handleMutationError(err, "Failed to update timer.")
	}

	h.logAudit(c, audit.ActionTimerUpdate, "Updated timer", record)
	return c.JSON(http.StatusOK, timerToDTO(record))
}

func (h *Handler) Delete(c *core.RequestEvent) error {
	id := strings.TrimSpace(c.Request.PathValue("id"))
	if id == "" {
		return router.NewBadRequestError("Missing id.", nil)
	}
	record, err := h.Service.Delete(id)
	if err != nil {
		return h.handleMutationError(err, "Failed to delete timer.")
	}

	h.logAudit(c, audit.ActionTimerDelete, "Deleted timer", record)
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) Cancel(c *core.RequestEvent) error {
	id := strings.TrimSpace(c.Request.PathValue("id"))
	if id == "" {
		return router.NewBadRequestError("Missing id.", nil)
	}
	record, err := h.Service.Cancel(id, c.Auth)
	if err != nil {
		return h.handleMutationError(err, "Failed to cancel timer.")
	}
	h.logAudit(c, audit.ActionTimerCancel, "Canceled timer", record)
	return c.JSON(http.StatusOK, timerToDTO(record))
}

func (h *Handler) Uncancel(c *core.RequestEvent) error {
	id := strings.TrimSpace(c.Request.PathValue("id"))
	if id == "" {
		return router.NewBadRequestError("Missing id.", nil)
	}
	record, err := h.Service.Uncancel(id)
	if err != nil {
		return h.handleMutationError(err, "Failed to uncancel timer.")
	}
	h.logAudit(c, audit.ActionTimerUncancel, "Uncancelled timer", record)
	return c.JSON(http.StatusOK, timerToDTO(record))
}

func (h *Handler) handleMutationError(err error, fallback string) error {
	if err == nil {
		return nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return router.NewNotFoundError("Timer not found.", nil)
	}
	return router.NewInternalServerError(fallback, logging.Fields{"error": err.Error()})
}

func (h *Handler) logAudit(c *core.RequestEvent, action, summary string, record *core.Record) {
	if h.Audit == nil || record == nil {
		return
	}
	h.Audit.LogRequest(c, &audit.Event{
		Action:      action,
		Summary:     summary + " " + record.GetString("title"),
		TargetType:  audit.TargetTypeTimer,
		TargetID:    record.Id,
		TargetLabel: record.GetString("title"),
		TargetMeta: map[string]any{
			"timer_id":    record.Id,
			"system_id":   record.GetInt("system_id"),
			"system_name": record.GetString("system_name"),
		},
	})
}

func isUserInputError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, timerssvc.ErrMissingSystem) ||
		errors.Is(err, timerssvc.ErrSystemNotFound) ||
		errors.Is(err, timerssvc.ErrMoonRequired) ||
		errors.Is(err, timerssvc.ErrPlanetRequired) ||
		errors.Is(err, timerssvc.ErrInvalidStageLabel) ||
		errors.Is(err, timerssvc.ErrInvalidTimerContext) ||
		errors.Is(err, timerssvc.ErrInvalidSkyhookFullnessPercentage) ||
		errors.Is(err, timerssvc.ErrMissingExpiresAt) ||
		errors.Is(err, timerssvc.ErrMissingDate) ||
		errors.Is(err, timerssvc.ErrInvalidDate) ||
		errors.Is(err, timerssvc.ErrEmptyTimerText) ||
		errors.Is(err, timerssvc.ErrNoTimerDateFound) ||
		errors.Is(err, timerssvc.ErrInvalidTimerDate) ||
		errors.Is(err, timerssvc.ErrMissingCreateInput) ||
		errors.Is(err, timerssvc.ErrMissingUpdateInput) ||
		errors.Is(err, timerssvc.ErrMissingInput) ||
		errors.Is(err, strconv.ErrSyntax)
}

func timerToDTO(record *core.Record) timerDTO {
	return timerDTO{
		ID:                     record.Id,
		Title:                  record.GetString("title"),
		SystemID:               record.GetInt("system_id"),
		SystemName:             record.GetString("system_name"),
		RegionID:               record.GetInt("region_id"),
		RegionName:             record.GetString("region_name"),
		Standing:               record.GetString("standing_type"),
		TimerKind:              record.GetString("timer_kind"),
		StructureType:          record.GetString("structure_type"),
		StageLabel:             record.GetString("stage_label"),
		PlanetID:               record.GetInt("planet_id"),
		PlanetName:             record.GetString("planet_name"),
		MoonID:                 record.GetInt("moon_id"),
		MoonName:               record.GetString("moon_name"),
		OwnerCorporationID:     record.GetInt("owner_corporation_id"),
		OwnerCorporationName:   record.GetString("owner_corporation_name"),
		OwnerCorporationTicker: record.GetString("owner_corporation_ticker"),
		OwnerAllianceID:        record.GetInt("owner_alliance_id"),
		OwnerAllianceName:      record.GetString("owner_alliance_name"),
		OwnerAllianceTicker:    record.GetString("owner_alliance_ticker"),
		SkyhookFullnessPct:     recordSkyhookFullnessPct(record),
		AttackersScorePct:      recordScorePct(record, "attackers_score_pct"),
		DefenderScorePct:       recordScorePct(record, "defender_score_pct"),
		Stage:                  record.GetInt("stage"),
		TotalStages:            record.GetInt("total_stages"),
		Severity:               record.GetString("severity"),
		Status:                 record.GetString("status"),
		ExpiresAt:              record.GetString("expires_at"),
		Source:                 record.GetString("source"),
		SourceRef:              record.GetString("source_ref"),
		Notes:                  record.GetString("notes"),
		RawText:                record.GetString("raw_text"),
		ReplacementAction:      record.GetString("replacement_action"),
		CreatedBy:              record.GetString("created_by"),
		CreatedByName:          record.GetString("created_by_name"),
		CanceledBy:             record.GetString("canceled_by"),
		CanceledAt:             record.GetString("canceled_at"),
		Created:                record.GetString("created"),
		Updated:                record.GetString("updated"),
	}
}

func splitInts(value string) []int {
	parts := strings.Split(value, ",")
	out := []int{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if n, err := strconv.Atoi(part); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func parsePositiveInt(value string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func recordSkyhookFullnessPct(record *core.Record) int {
	if record == nil {
		return 0
	}
	value := int(math.Round(record.GetFloat("skyhook_fullness_pct")))
	return min(scorePercentMax, max(0, value))
}

func recordScorePct(record *core.Record, field string) int {
	if record == nil {
		return 0
	}
	value := int(math.Round(record.GetFloat(field)))
	return min(scorePercentMax, max(0, value))
}
