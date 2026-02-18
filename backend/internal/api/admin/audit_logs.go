package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"sentinel2/internal/audit"
	"sentinel2/internal/format"
	"sentinel2/internal/logging"
	"sentinel2/internal/shared/pagination"
	"sentinel2/internal/shared/queryhelpers"
	"sentinel2/internal/store"
)

type auditLogOptions struct {
	userID  string
	action  string
	actor   string
	summary string
	page    int
	limit   int
}

const (
	defaultAuditLogLimit = 30
	maxAuditLogLimit     = 100
)

func (h *Handler) AuditLogs(c *core.RequestEvent) error {
	opts := parseAuditLogOptions(c.Request.URL.Query())
	filter, params, filterErr := h.buildAuditLogFilter(&opts)
	if filterErr != nil {
		return filterErr
	}
	offset := pagination.OffsetForPage(opts.page, opts.limit)
	records, recordsErr := h.App.FindRecordsByFilter(
		store.CollectionAuditLogs,
		filter,
		"-created",
		pagination.LimitPlusOne(opts.limit),
		offset,
		params,
	)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to load audit logs.", logging.Fields{
			"filter": filter,
			"page":   opts.page,
			"limit":  opts.limit,
			"error":  recordsErr.Error(),
		})
	}

	records, hasMore := pagination.TrimToLimit(records, opts.limit)
	entries := buildAuditLogEntries(records)

	return c.JSON(http.StatusOK, map[string]any{
		"logs":    entries,
		"page":    opts.page,
		"limit":   opts.limit,
		"hasMore": hasMore,
	})
}

func (h *Handler) buildAuditLogFilter(opts *auditLogOptions) (string, dbx.Params, error) {
	if opts == nil {
		return "", dbx.Params{}, nil
	}
	filter := ""
	params := dbx.Params{}
	if opts.userID != "" {
		clauses := []string{
			"target_user_id = {:user}",
			"actor_id = {:user}",
			"(target_type = {:target_user_type} && target_id = {:user})",
		}
		params["user"] = opts.userID
		params["target_user_type"] = audit.TargetTypeUser
		params["target_character_type"] = audit.TargetTypeCharacter

		characters, charactersErr := h.App.FindRecordsByFilter(
			store.CollectionCharacters,
			"user = {:user}",
			"",
			0,
			0,
			dbx.Params{"user": opts.userID},
		)
		if charactersErr != nil {
			return "", nil, router.NewInternalServerError("Failed to load audit logs.", logging.Fields{
				"user_id": opts.userID,
				"error":   charactersErr.Error(),
			})
		}
		for i, character := range characters {
			characterIDParam := fmt.Sprintf("character_id_%d", i)
			characterRecordIDParam := fmt.Sprintf("character_record_id_%d", i)
			params[characterIDParam] = character.GetInt("eve_character_id")
			params[characterRecordIDParam] = character.Id
			clauses = append(
				clauses,
				"target_character_id = {:"+characterIDParam+"}",
				"(target_type = {:target_character_type} && target_id = {:"+characterRecordIDParam+"})",
			)
		}
		filter = "(" + strings.Join(clauses, " || ") + ")"
	}
	if opts.action != "" {
		filter = queryhelpers.AppendAnd(filter, "action ~ {:action}")
		params["action"] = "%" + opts.action + "%"
	}
	if opts.actor != "" {
		filter = queryhelpers.AppendAnd(filter, "actor_display_name ~ {:actor}")
		params["actor"] = "%" + opts.actor + "%"
	}
	if opts.summary != "" {
		filter = queryhelpers.AppendAnd(filter, "summary ~ {:summary}")
		params["summary"] = "%" + opts.summary + "%"
	}
	return filter, params, nil
}

func parseAuditLogOptions(values url.Values) auditLogOptions {
	return auditLogOptions{
		userID:  strings.TrimSpace(values.Get("user_id")),
		action:  strings.TrimSpace(values.Get("action")),
		actor:   strings.TrimSpace(values.Get("actor")),
		summary: strings.TrimSpace(values.Get("summary")),
		page:    format.GetPositiveInt(values, "page", 1, 0),
		limit:   format.GetPositiveInt(values, "limit", defaultAuditLogLimit, maxAuditLogLimit),
	}
}

func buildAuditLogEntries(records []*core.Record) []auditLogEntry {
	entries := make([]auditLogEntry, 0, len(records))
	for _, record := range records {
		created := ""
		if !record.GetDateTime("created").IsZero() {
			created = record.GetDateTime("created").Time().Format(time.RFC3339)
		}
		entries = append(entries, auditLogEntry{
			ID:                  record.Id,
			Action:              record.GetString("action"),
			Summary:             record.GetString("summary"),
			ActorID:             record.GetString("actor_id"),
			ActorDisplayName:    record.GetString("actor_display_name"),
			TargetUserID:        record.GetString("target_user_id"),
			TargetUserName:      record.GetString("target_user_name"),
			TargetCharacterID:   record.GetInt("target_character_id"),
			TargetCharacterName: record.GetString("target_character_name"),
			TargetType:          record.GetString("target_type"),
			TargetID:            record.GetString("target_id"),
			TargetLabel:         record.GetString("target_label"),
			TargetMeta:          record.Get("target_meta"),
			Created:             created,
		})
	}
	return entries
}
