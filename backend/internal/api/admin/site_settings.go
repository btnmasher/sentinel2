package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/audit"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

type allowedOrganizationEntry struct {
	EVEID int    `json:"eve_id"`
	Name  string `json:"name"`
}

type allowedOrganizationsResponse struct {
	Alliances    []allowedOrganizationEntry `json:"alliances"`
	Corporations []allowedOrganizationEntry `json:"corporations"`
}

func (h *Handler) ListAllowedOrganizations(c *core.RequestEvent) error {
	alliances, err := h.listAllowedOrganizationsByCollection(store.CollectionAllowedAlliances)
	if err != nil {
		return router.NewInternalServerError("Failed to load allowed alliances.", logging.Fields{"error": err.Error()})
	}
	corporations, err := h.listAllowedOrganizationsByCollection(store.CollectionAllowedCorporations)
	if err != nil {
		return router.NewInternalServerError("Failed to load allowed corporations.", logging.Fields{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, allowedOrganizationsResponse{
		Alliances:    alliances,
		Corporations: corporations,
	})
}

func (h *Handler) UpsertAllowedOrganization(c *core.RequestEvent) error {
	payload := struct {
		Type  string `json:"type"`
		EVEID int    `json:"eve_id"`
		Name  string `json:"name"`
	}{}

	if bindErr := c.BindBody(&payload); bindErr != nil {
		return router.NewBadRequestError("Invalid payload.", logging.Fields{"error": bindErr.Error()})
	}
	collection, label, resolveErr := allowedOrgCollection(strings.TrimSpace(strings.ToLower(payload.Type)))
	if resolveErr != nil {
		return router.NewBadRequestError(resolveErr.Error(), logging.Fields{"type": payload.Type})
	}

	if payload.EVEID <= 0 {
		return router.NewBadRequestError("eve_id must be positive.", logging.Fields{"eve_id": payload.EVEID})
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return router.NewBadRequestError("name is required.", logging.Fields{"name": payload.Name})
	}

	records, findErr := h.App.FindRecordsByFilter(
		collection,
		"eve_id = {:id}",
		"",
		1,
		0,
		dbx.Params{"id": payload.EVEID},
	)
	if findErr != nil {
		return router.NewInternalServerError("Failed to validate allowed organization.", logging.Fields{"error": findErr.Error()})
	}

	var record *core.Record
	if len(records) > 0 {
		record = records[0]
		record.Set("name", name)
	} else {
		collectionRecord, collectionErr := h.App.FindCollectionByNameOrId(collection)
		if collectionErr != nil {
			return router.NewInternalServerError("Allowed organization collection unavailable.", logging.Fields{"error": collectionErr.Error()})
		}
		record = core.NewRecord(collectionRecord)
		record.Set("eve_id", payload.EVEID)
		record.Set("name", name)
	}

	if saveErr := h.App.Save(record); saveErr != nil {
		return router.NewInternalServerError("Failed to save allowed organization.", logging.Fields{"error": saveErr.Error()})
	}

	h.logAction(c, &audit.Event{
		Action:      audit.ActionAdminAllowedOrgAdd,
		Summary:     fmt.Sprintf("Added %s allow entry %s (%d)", label, name, payload.EVEID),
		TargetType:  audit.TargetTypeAllowedOrg,
		TargetID:    record.Id,
		TargetLabel: name,
		TargetMeta: map[string]any{
			"type":   payload.Type,
			"eve_id": payload.EVEID,
		},
	})

	return c.JSON(http.StatusCreated, map[string]any{"ok": true})
}

func (h *Handler) DeleteAllowedOrganization(c *core.RequestEvent) error {
	rawType := strings.TrimSpace(strings.ToLower(c.Request.PathValue("type")))
	rawID := strings.TrimSpace(c.Request.PathValue("eve_id"))

	collection, label, resolveErr := allowedOrgCollection(rawType)
	if resolveErr != nil {
		return router.NewBadRequestError(resolveErr.Error(), logging.Fields{"type": rawType})
	}
	eveID, parseErr := strconv.Atoi(rawID)
	if parseErr != nil || eveID <= 0 {
		return router.NewBadRequestError("Invalid eve_id.", logging.Fields{"eve_id": rawID})
	}

	records, findErr := h.App.FindRecordsByFilter(
		collection,
		"eve_id = {:id}",
		"",
		1,
		0,
		dbx.Params{"id": eveID},
	)
	if findErr != nil {
		return router.NewInternalServerError("Failed to load allowed organization.", logging.Fields{"error": findErr.Error()})
	}

	if len(records) == 0 {
		return router.NewNotFoundError("Allowed organization not found.", logging.Fields{"type": rawType, "eve_id": eveID})
	}
	record := records[0]
	name := record.GetString("name")
	if deleteErr := h.App.Delete(record); deleteErr != nil {
		return router.NewInternalServerError("Failed to delete allowed organization.", logging.Fields{"error": deleteErr.Error()})
	}

	h.logAction(c, &audit.Event{
		Action:      audit.ActionAdminAllowedOrgDelete,
		Summary:     fmt.Sprintf("Removed %s allow entry %s (%d)", label, name, eveID),
		TargetType:  audit.TargetTypeAllowedOrg,
		TargetID:    record.Id,
		TargetLabel: name,
		TargetMeta: map[string]any{
			"type":   rawType,
			"eve_id": eveID,
		},
	})

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) listAllowedOrganizationsByCollection(collection string) ([]allowedOrganizationEntry, error) {
	records, err := h.App.FindRecordsByFilter(collection, "", "name", 0, 0, nil)
	if err != nil {
		return nil, err
	}
	out := make([]allowedOrganizationEntry, 0, len(records))
	for _, record := range records {
		out = append(out, allowedOrganizationEntry{
			EVEID: record.GetInt("eve_id"),
			Name:  strings.TrimSpace(record.GetString("name")),
		})
	}
	return out, nil
}

func allowedOrgCollection(rawType string) (collection, label string, err error) {
	switch rawType {
	case "alliance":
		return store.CollectionAllowedAlliances, "alliance", nil
	case "corporation":
		return store.CollectionAllowedCorporations, "corporation", nil
	default:
		return "", "", fmt.Errorf("type must be 'alliance' or 'corporation'")
	}
}
