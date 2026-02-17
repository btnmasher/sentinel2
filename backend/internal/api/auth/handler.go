package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/audit"
	"sentinel2/internal/auth"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"

	"github.com/pocketbase/pocketbase/tools/router"
)

type AuthHandler struct {
	Auth  *auth.Manager
	Audit *audit.Service
}

func NewAuthHandler(manager *auth.Manager, auditSvc *audit.Service) *AuthHandler {
	return &AuthHandler{Auth: manager, Audit: auditSvc}
}

func (h *AuthHandler) Authenticate(c *core.RequestEvent) error {
	if err := h.Auth.Authenticate(c, auth.AuthFlow{Type: auth.FlowLogin}); err != nil {
		logging.WithRequest(h.Auth.App, c).
			WithErr(err).
			Debug("auth login failed")
		return err
	}
	return nil
}

func (h *AuthHandler) Callback(c *core.RequestEvent) error {
	result, callbackErr := h.Auth.Callback(c)
	if callbackErr != nil {
		logging.WithRequest(h.Auth.App, c).
			WithErr(callbackErr).
			Debug("auth callback failed")
		return callbackErr
	}
	if result.IsLink {
		linkedCharacter := findCharacterByUserAndEVEID(h.Auth.App, result.UserID, result.CharacterID)
		linkSummary := "Linked character"
		if result.CharacterName != "" {
			linkSummary = fmt.Sprintf("Linked character %s", result.CharacterName)
		}
		if h.Audit != nil {
			h.Audit.LogEvent(audit.Event{
				Action:                 audit.ActionUserAuthLinkCharacter,
				Summary:                linkSummary,
				TargetUserID:           result.UserID,
				TargetCharacter:        linkedCharacter,
				TargetType:             audit.TargetTypeCharacter,
				TargetLabel:            result.CharacterName,
				ResolveTargetCharacter: linkedCharacter == nil,
				TargetMeta: map[string]any{
					"eve_character_id": result.CharacterID,
				},
				ActorID:          result.UserID,
				ActorDisplayName: result.CharacterName,
			})
		}
		logging.WithRequest(h.Auth.App, c).
			Info("auth link completed")
		return c.Redirect(http.StatusFound, "/profile?linked=1")
	}
	logging.WithRequest(h.Auth.App, c).
		Info("auth login completed")
	return c.Redirect(http.StatusFound, "/auth/complete?code="+result.ExchangeCode)
}

func (h *AuthHandler) Logout(c *core.RequestEvent) error {
	if err := h.Auth.Logout(c); err != nil {
		logging.WithRequest(h.Auth.App, c).
			WithErr(err).
			Debug("auth logout failed")
		return err
	}
	return nil
}

func (h *AuthHandler) Link(c *core.RequestEvent) error {
	if h.Auth.Provider.Name() != auth.AuthProviderEVE {
		return router.NewNotFoundError("Not found", nil)
	}
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		logging.WithRequest(h.Auth.App, c).
			WithErr(userErr).
			Debug("auth link unauthorized")
		return userErr
	}
	authURL, err := h.Auth.BuildAuthURL(
		c,
		auth.AuthFlow{Type: auth.FlowLink, LinkUserID: user.Id},
	)
	if err != nil {
		logging.WithRequest(h.Auth.App, c).
			WithErr(err).
			Debug("auth link failed")
		return err
	}
	return c.JSON(200, map[string]string{"url": authURL})
}

type authMeResponse struct {
	UserID      string `json:"user_id"`
	AccessLevel string `json:"access_level"`
	Provider    string `json:"auth_provider"`
}

func (h *AuthHandler) Me(c *core.RequestEvent) error {
	record, recordErr := auth.CurrentUser(c)
	if recordErr != nil {
		return recordErr
	}

	return c.JSON(200, authMeResponse{
		UserID:      record.Id,
		AccessLevel: record.GetString("access_level"),
		Provider:    record.GetString("auth_provider"),
	})
}

type characterProfile struct {
	RecordID         string `json:"record_id"`
	CharacterID      int    `json:"character_id"`
	Name             string `json:"name"`
	CorpID           int    `json:"corp_id"`
	CorpName         string `json:"corp_name"`
	AllianceID       int    `json:"alliance_id"`
	AllianceName     string `json:"alliance_name"`
	IsMain           bool   `json:"is_main"`
	ESITokenValid    bool   `json:"esi_token_valid"`
	ESILastError     string `json:"esi_last_error"`
	ESILastRefreshAt string `json:"esi_last_refresh_at"`
}

type profileResponse struct {
	UserID     string             `json:"user_id"`
	Characters []characterProfile `json:"characters"`
}

func (h *AuthHandler) Profile(c *core.RequestEvent) error {
	record, recordErr := auth.CurrentUser(c)
	if recordErr != nil {
		return recordErr
	}

	response := profileResponse{
		UserID:     record.Id,
		Characters: []characterProfile{},
	}

	charRecords, recordsErr := h.Auth.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user}",
		"-is_main",
		0,
		0, dbx.Params{"user": record.Id},
	)
	if recordsErr == nil && len(charRecords) > 0 {
		for _, charRecord := range charRecords {
			corpID := charRecord.GetInt("eve_corporation_id")
			allianceID := charRecord.GetInt("eve_alliance_id")
			allianceName := store.GetOrgName(h.Auth.App, store.CollectionAlliances, allianceID)
			corpName := store.GetOrgName(h.Auth.App, store.CollectionCorporations, corpID)
			refresh := charRecord.GetDateTime("esi_last_refresh_at")
			refreshAt := ""
			if !refresh.IsZero() {
				refreshAt = refresh.Time().Format(time.RFC3339)
			}
			response.Characters = append(response.Characters, characterProfile{
				RecordID:         charRecord.Id,
				CharacterID:      charRecord.GetInt("eve_character_id"),
				Name:             charRecord.GetString("eve_character_name"),
				CorpID:           corpID,
				CorpName:         corpName,
				AllianceID:       allianceID,
				AllianceName:     allianceName,
				IsMain:           charRecord.GetBool("is_main"),
				ESITokenValid:    charRecord.GetBool("esi_token_valid"),
				ESILastError:     charRecord.GetString("esi_last_error"),
				ESILastRefreshAt: refreshAt,
			})
		}
		return c.JSON(200, response)
	}

	charID := record.GetInt("eve_character_id")
	if charID != 0 {
		corpID := record.GetInt("eve_corporation_id")
		allianceID := record.GetInt("eve_alliance_id")
		allianceName := store.GetOrgName(h.Auth.App, store.CollectionAlliances, allianceID)
		corpName := store.GetOrgName(h.Auth.App, store.CollectionCorporations, corpID)
		refresh := record.GetDateTime("esi_last_refresh_at")
		refreshAt := ""
		if !refresh.IsZero() {
			refreshAt = refresh.Time().Format(time.RFC3339)
		}
		response.Characters = append(response.Characters, characterProfile{
			CharacterID:      charID,
			Name:             record.GetString("eve_character_name"),
			CorpID:           corpID,
			CorpName:         corpName,
			AllianceID:       allianceID,
			AllianceName:     allianceName,
			IsMain:           true,
			ESITokenValid:    record.GetBool("esi_token_valid"),
			ESILastError:     record.GetString("esi_last_error"),
			ESILastRefreshAt: refreshAt,
		})
	}
	return c.JSON(200, response)
}

type authExchangeResponse struct {
	Token     string         `json:"token"`
	Record    map[string]any `json:"record"`
	ExpiresAt string         `json:"expires_at"`
}

func (h *AuthHandler) Exchange(c *core.RequestEvent) error {
	code := c.Request.URL.Query().Get("code")
	if code == "" {
		return router.NewBadRequestError("Missing code.", logging.Fields{
			"required_field": "code",
		})
	}
	user, token, exchangeErr := h.Auth.Exchange(code)
	if exchangeErr != nil {
		return exchangeErr
	}
	expiresAt := time.Now().Add(auth.PBTokenTTL()).UTC().Format(time.RFC3339)
	return c.JSON(http.StatusOK, authExchangeResponse{
		Token:     token,
		Record:    user.PublicExport(),
		ExpiresAt: expiresAt,
	})
}

func (h *AuthHandler) Refresh(c *core.RequestEvent) error {
	user, userErr := auth.CurrentUser(c)
	if userErr != nil {
		return userErr
	}
	token, tokenErr := h.Auth.IssueToken(user)
	if tokenErr != nil {
		return tokenErr
	}
	expiresAt := time.Now().Add(auth.PBTokenTTL()).UTC().Format(time.RFC3339)
	return c.JSON(http.StatusOK, authExchangeResponse{
		Token:     token,
		Record:    user.PublicExport(),
		ExpiresAt: expiresAt,
	})
}

func findCharacterByUserAndEVEID(app *pocketbase.PocketBase, userID string, eveCharacterID int) *core.Record {
	if app == nil || userID == "" || eveCharacterID <= 0 {
		return nil
	}
	records, err := app.FindRecordsByFilter(
		store.CollectionCharacters,
		"user = {:user} && eve_character_id = {:character_id}",
		"",
		1,
		0,
		dbx.Params{
			"user":         userID,
			"character_id": eveCharacterID,
		},
	)
	if err != nil || len(records) == 0 {
		return nil
	}
	return records[0]
}
