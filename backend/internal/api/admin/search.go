package admin

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

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
	defaultSearchLimit       = 25
	maxSearchLimit           = 100
	searchPrefixTokenCapHint = 36
)

func (h *Handler) Search(c *core.RequestEvent) error {
	opts := parseSearchOptions(c.Request.URL.Query())
	offset := pagination.OffsetForPage(opts.page, opts.limit)
	authProvider := h.currentAuthProvider(c)
	filter, params := buildCharacterSearchFilter(authProvider, opts.query, opts.startsWith)
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
		"provider":   authProvider,
		})
	}
	records, hasMore := pagination.TrimToLimit(records, opts.limit)
	results := h.buildSearchItems(records)

	return c.JSON(http.StatusOK, map[string]any{
		"results":          results,
		"page":             opts.page,
		"limit":            opts.limit,
		"hasMore":          hasMore,
		"availableLetters": h.availableSearchLetters(authProvider),
	})
}

func (h *Handler) availableSearchLetters(authProvider string) []string {
	letters := []string{}
	for _, token := range searchPrefixTokens() {
		letterString := token
		filter, params := buildCharacterSearchFilter(authProvider, "", letterString)
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
	seenUsers := map[string]struct{}{}
	results := make([]searchItem, 0, len(records))
	for _, rec := range records {
		userID := rec.GetString("user")
		if userID == "" {
			continue
		}
		if _, seen := seenUsers[userID]; seen {
			continue
		}
		seenUsers[userID] = struct{}{}

		mainRecord := rec
		if !rec.GetBool("is_main") {
			if main, _ := h.findMainCharacter(userID); main != nil {
				mainRecord = main
			}
		}
		results = append(results, searchItem{
			CharacterRecordID: mainRecord.Id,
			CharacterID:       mainRecord.GetInt("eve_character_id"),
			Name:              mainRecord.GetString("eve_character_name"),
			UserID:            userID,
			AuthProvider:      mainRecord.GetString("auth_provider"),
			IsMain:            true,
			MainName:          "",
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

func buildCharacterSearchFilter(authProvider, query, startsWith string) (string, dbx.Params) {
	filter := "user != \"\""
	params := dbx.Params{}

	if authProvider != "" {
		filter = queryhelpers.AppendAnd(filter, "auth_provider = {:authProvider}")
		params["authProvider"] = authProvider
	}

	if query != "" {
		filter = queryhelpers.AppendAnd(filter, "(eve_character_name ~ {:q} || user ~ {:q} || user = {:userID})")
		params["q"] = "%" + query + "%"
		params["userID"] = query
	}

	if isSearchPrefix(startsWith) {
		filter = queryhelpers.AppendAnd(filter, "is_main = true && eve_character_name ~ {:startsWith}")
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
