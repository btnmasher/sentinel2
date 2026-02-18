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

type searchOptions struct {
	query      string
	startsWith string
	page       int
	limit      int
}

const (
	defaultSeedCount         = 50
	maxSeedCount             = 1000
	characterIDMultiplier    = 10000
	baseCharacterIDOffset    = 900000000
	defaultSearchLimit       = 25
	maxSearchLimit           = 100
	searchPrefixTokenCapHint = 36
)

func (h *Handler) Search(c *core.RequestEvent) error {
	opts := parseSearchOptions(c.Request.URL.Query())
	offset := pagination.OffsetForPage(opts.page, opts.limit)
	filter, params := buildCharacterSearchFilter(opts.query, opts.startsWith)
	records, recordsErr := h.App.FindRecordsByFilter(
		store.CollectionCharacters,
		filter,
		"eve_character_name",
		pagination.LimitPlusOne(opts.limit),
		offset,
		params,
	)
	if recordsErr != nil {
		return router.NewInternalServerError("Failed to search characters.", logging.Fields{
			"query":      opts.query,
			"startsWith": opts.startsWith,
			"page":       opts.page,
			"limit":      opts.limit,
		})
	}
	records, hasMore := pagination.TrimToLimit(records, opts.limit)
	results := h.buildSearchItems(records)

	return c.JSON(http.StatusOK, map[string]any{
		"results":          results,
		"page":             opts.page,
		"limit":            opts.limit,
		"hasMore":          hasMore,
		"availableLetters": h.availableSearchLetters(),
	})
}

func (h *Handler) SeedSearchUsers(c *core.RequestEvent) error {
	payload := struct {
		Count  int    `json:"count"`
		Prefix string `json:"prefix"`
	}{}
	if c.Request.ContentLength > 0 {
		if bindErr := c.BindBody(&payload); bindErr != nil {
			return router.NewBadRequestError("Invalid payload.", logging.Fields{
				"error": bindErr,
			})
		}
	}
	if payload.Count <= 0 {
		payload.Count = defaultSeedCount
	}
	if payload.Count > maxSeedCount {
		payload.Count = maxSeedCount
	}
	payload.Prefix = strings.TrimSpace(payload.Prefix)
	if payload.Prefix == "" {
		payload.Prefix = "Debug"
	}

	userCollection, userCollectionErr := h.App.FindCollectionByNameOrId(store.CollectionUsers)
	if userCollectionErr != nil {
		return router.NewInternalServerError("Failed to load users collection.", logging.Fields{
			"error": userCollectionErr.Error(),
		})
	}
	charCollection, charCollectionErr := h.App.FindCollectionByNameOrId(store.CollectionCharacters)
	if charCollectionErr != nil {
		return router.NewInternalServerError("Failed to load characters collection.", logging.Fields{
			"error": charCollectionErr.Error(),
		})
	}

	created := 0
	letters := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	baseCharacterID := int(time.Now().Unix())*characterIDMultiplier + baseCharacterIDOffset
	for i := range payload.Count {
		letter := string(letters[i%len(letters)])
		name := fmt.Sprintf("%s %s%03d", payload.Prefix, letter, i+1)
		characterID := baseCharacterID + i

		userRecord := core.NewRecord(userCollection)
		userRecord.Set("sub", fmt.Sprintf("debug-%d", characterID))
		userRecord.Set("auth_provider", "debug")
		userRecord.Set("auth_provider_sub", fmt.Sprintf("%d", characterID))
		userRecord.SetEmail(fmt.Sprintf("debug-%d@auth.invalid", characterID))
		userRecord.SetRandomPassword()
		userRecord.Set("created_at", time.Now())
		userRecord.Set("access_level", "user")
		userRecord.Set("eve_character_id", characterID)
		userRecord.Set("eve_character_name", name)
		if saveErr := h.App.Save(userRecord); saveErr != nil {
			return router.NewInternalServerError("Failed to create debug user.", logging.Fields{
				"error": saveErr.Error(),
				"name":  name,
			})
		}

		charRecord := core.NewRecord(charCollection)
		charRecord.Set("user", userRecord.Id)
		charRecord.Set("eve_character_id", characterID)
		charRecord.Set("eve_character_name", name)
		charRecord.Set("is_main", true)
		charRecord.Set("esi_token_valid", true)
		if saveErr := h.App.Save(charRecord); saveErr != nil {
			return router.NewInternalServerError("Failed to create debug character.", logging.Fields{
				"error":        saveErr.Error(),
				"name":         name,
				"character_id": characterID,
			})
		}

		created++
	}

	h.logAction(
		c,
		&audit.Event{
			Action:  audit.ActionDebugSearchSeed,
			Summary: fmt.Sprintf("Seeded %d debug search users", created),
		},
	)

	return c.JSON(http.StatusOK, map[string]any{
		"created": created,
		"prefix":  payload.Prefix,
	})
}

func (h *Handler) availableSearchLetters() []string {
	letters := []string{}
	for _, token := range searchPrefixTokens() {
		letterString := token
		filter, params := buildCharacterSearchFilter("", letterString)
		records, recordsErr := h.App.FindRecordsByFilter(
			store.CollectionCharacters,
			filter,
			"",
			1,
			0,
			params,
		)
		if recordsErr == nil && len(records) > 0 {
			letters = append(letters, letterString)
		}
	}
	return letters
}

func (h *Handler) buildSearchItems(records []*core.Record) []searchItem {
	mainNames := map[string]string{}
	results := make([]searchItem, 0, len(records))
	for _, rec := range records {
		userID := rec.GetString("user")
		if userID == "" {
			continue
		}
		mainName := mainNames[userID]
		if mainName == "" {
			main, _ := h.findMainCharacter(userID)
			if main != nil {
				mainName = main.GetString("eve_character_name")
				mainNames[userID] = mainName
			}
		}
		results = append(results, searchItem{
			CharacterRecordID: rec.Id,
			CharacterID:       rec.GetInt("eve_character_id"),
			Name:              rec.GetString("eve_character_name"),
			UserID:            userID,
			IsMain:            rec.GetBool("is_main"),
			MainName:          mainName,
		})
	}
	return results
}

func parseSearchOptions(values url.Values) searchOptions {
	startsWith := strings.ToUpper(strings.TrimSpace(values.Get("startsWith")))
	if len(startsWith) > 1 {
		startsWith = ""
	}
	return searchOptions{
		query:      strings.TrimSpace(values.Get("q")),
		startsWith: startsWith,
		page:       format.GetPositiveInt(values, "page", 1, 0),
		limit:      format.GetPositiveInt(values, "limit", defaultSearchLimit, maxSearchLimit),
	}
}

func buildCharacterSearchFilter(query, startsWith string) (string, dbx.Params) {
	filter := "user != \"\""
	params := dbx.Params{}
	if query != "" {
		filter = queryhelpers.AppendAnd(filter, "(eve_character_name ~ {:q} || user ~ {:q} || user = {:userID})")
		params["q"] = "%" + query + "%"
		params["userID"] = query
	}
	if isSearchPrefix(startsWith) {
		filter = queryhelpers.AppendAnd(filter, "eve_character_name ~ {:startsWith}")
		params["startsWith"] = startsWith + "%"
	}
	return filter, params
}

func searchPrefixTokens() []string {
	tokens := make([]string, 0, searchPrefixTokenCapHint)
	for digit := '0'; digit <= '9'; digit++ {
		tokens = append(tokens, string(digit))
	}
	for letter := 'A'; letter <= 'Z'; letter++ {
		tokens = append(tokens, string(letter))
	}
	return tokens
}

func isSearchPrefix(value string) bool {
	if len(value) != 1 {
		return false
	}
	ch := value[0]
	return (ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'Z')
}
