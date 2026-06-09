package intel

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/shared/eve"
	"sentinel2/internal/store"
)

var (
	reportPattern      = regexp.MustCompile(`\[ (?P<date>.*) \] (?P<author>[\s\w\-']+) > (?P<text>.*)`)
	systemTokenPattern = regexp.MustCompile(`\S+`)
)

const (
	maxReportBodyLen      = 256
	minSystemTokenLenHint = 3
)

type ParsedReport struct {
	Date   time.Time
	Author string
	Text   string
}

func ParseReportText(text string) (ParsedReport, error) {
	clean := strings.ReplaceAll(text, "\ufeff", "")
	matches := reportPattern.FindStringSubmatch(clean)
	if matches == nil {
		return ParsedReport{}, ErrInvalidLogFormat
	}

	dateText := matches[1]
	author := matches[2]
	body := matches[3]

	parsed, parseErr := time.Parse("2006.01.02 15:04:05", dateText)
	if parseErr != nil {
		return ParsedReport{}, parseErr
	}

	if len(body) > maxReportBodyLen {
		body = body[:maxReportBodyLen]
	}

	return ParsedReport{Date: parsed, Author: author, Text: body}, nil
}

// NormalizeSystemNames replaces matching system names in a report body with the canonical English names.
func NormalizeSystemNames(app *pocketbase.PocketBase, text string) (string, []IntelSystem, error) {
	normalizedText, systems, _, err := NormalizeSystemNamesWithHints(app, text)
	return normalizedText, systems, err
}

// IntelSystemHint describes a matched system occurrence in normalized report text.
type IntelSystemHint struct {
	SystemID int    `json:"system_id"`
	Name     string `json:"name"`
}

type reportToken struct {
	start       int
	end         int
	prefix      string
	core        string
	cleanSuffix string
	exactKey    string
	lowerKey    string
	isLocalized bool
	isCandidate bool
	displayText string
}

type reportSystemMatches struct {
	exact     map[string]IntelSystem
	lower     map[string]IntelSystem
	localized map[string]IntelSystem
	systems   []IntelSystem
}

// NormalizeSystemNamesWithHints replaces matching system names in a report body with the canonical English names.
// It also returns per-occurrence system hints for frontend enrichment.
func NormalizeSystemNamesWithHints(app *pocketbase.PocketBase, text string) (string, []IntelSystem, []IntelSystemHint, error) {
	if strings.TrimSpace(text) == "" {
		return text, nil, nil, nil
	}

	tokens := collectReportTokens(text)
	matches, err := resolveReportSystemMatches(app, tokens)
	if err != nil {
		return "", nil, nil, err
	}

	return rebuildReportText(text, tokens, matches), matches.systems, buildSystemHints(tokens, matches), nil
}

func collectReportTokens(text string) []reportToken {
	spans := systemTokenPattern.FindAllStringIndex(text, -1)
	tokens := make([]reportToken, 0, len(spans))
	for _, span := range spans {
		raw := text[span[0]:span[1]]
		prefix, tokenCore, suffix := splitTokenAffixes(raw)
		cleanSuffix := strings.TrimLeft(suffix, "*")
		displayText := prefix + tokenCore + cleanSuffix
		token := reportToken{
			start:       span[0],
			end:         span[1],
			prefix:      prefix,
			core:        tokenCore,
			cleanSuffix: cleanSuffix,
			exactKey:    tokenCore,
			lowerKey:    strings.ToLower(tokenCore),
			isLocalized: containsNonASCII(tokenCore),
			displayText: displayText,
		}
		token.isCandidate = len(token.core) >= minSystemTokenLenHint
		tokens = append(tokens, token)
	}
	return tokens
}

func resolveReportSystemMatches(app *pocketbase.PocketBase, tokens []reportToken) (reportSystemMatches, error) {
	englishExact := resolveExactEnglishCandidates(tokens)
	englishExactMatches, err := resolveEnglishExactSystemCandidates(app, englishExact)
	if err != nil {
		return reportSystemMatches{}, err
	}

	englishLower := resolveLowerEnglishCandidates(tokens, englishExactMatches)
	englishLowerMatches, err := resolveLowerEnglishSystemCandidates(app, englishLower)
	if err != nil {
		return reportSystemMatches{}, err
	}

	localized := resolveLocalizedCandidates(tokens, englishExactMatches)
	localizedMatches, err := resolveLocalizedSystemCandidates(app, localized)
	if err != nil {
		return reportSystemMatches{}, err
	}

	systems := make([]IntelSystem, 0, len(tokens))
	seenSystems := map[int]struct{}{}
	for _, token := range tokens {
		if system, ok := matchedSystemForToken(token.exactKey, token.lowerKey, englishExactMatches, englishLowerMatches, localizedMatches); ok {
			appendSystemMatch(&systems, seenSystems, system)
		}
	}

	return reportSystemMatches{
		exact:     englishExactMatches,
		lower:     englishLowerMatches,
		localized: localizedMatches,
		systems:   systems,
	}, nil
}

func resolveExactEnglishCandidates(tokens []reportToken) []string {
	return uniqueTokenKeys(tokens, func(token reportToken) (string, bool) {
		if !token.isCandidate {
			return "", false
		}
		return token.exactKey, true
	})
}

func resolveLowerEnglishCandidates(tokens []reportToken, englishExact map[string]IntelSystem) []string {
	return uniqueTokenKeys(tokens, func(token reportToken) (string, bool) {
		if !token.isCandidate || token.isLocalized {
			return "", false
		}
		if _, resolved := englishExact[token.exactKey]; resolved {
			return "", false
		}
		return token.lowerKey, true
	})
}

func resolveLocalizedCandidates(tokens []reportToken, englishExact map[string]IntelSystem) []string {
	return uniqueTokenKeys(tokens, func(token reportToken) (string, bool) {
		if !token.isCandidate || !token.isLocalized {
			return "", false
		}
		if _, resolved := englishExact[token.exactKey]; resolved {
			return "", false
		}
		return token.exactKey, true
	})
}

func resolveEnglishExactSystemCandidates(app *pocketbase.PocketBase, candidates []string) (map[string]IntelSystem, error) {
	return findSystemsByFieldValues(app, "name", candidates)
}

func resolveLowerEnglishSystemCandidates(app *pocketbase.PocketBase, candidates []string) (map[string]IntelSystem, error) {
	return findSystemsByFieldValuesLower(app, "name", candidates)
}

func resolveLocalizedSystemCandidates(app *pocketbase.PocketBase, candidates []string) (map[string]IntelSystem, error) {
	matches := map[string]IntelSystem{}
	for _, locale := range eve.SupportedSystemLocales() {
		field, fieldOK := eve.SystemNameField(locale)
		if !fieldOK {
			continue
		}

		fieldMatches, err := findSystemsByFieldValues(app, field, candidates)
		if err != nil {
			return nil, err
		}
		for key, system := range fieldMatches {
			if _, exists := matches[key]; exists {
				continue
			}
			matches[key] = system
		}
	}
	return matches, nil
}

func uniqueTokenKeys(tokens []reportToken, include func(reportToken) (string, bool)) []string {
	seen := map[string]struct{}{}
	keys := make([]string, 0, len(tokens))
	for _, token := range tokens {
		key, ok := include(token)
		if !ok || key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

func matchedSystemForToken(exactKey, lowerKey string, englishExact, englishLower, localized map[string]IntelSystem) (IntelSystem, bool) {
	if system, ok := englishExact[exactKey]; ok {
		return system, true
	}
	if system, ok := englishLower[lowerKey]; ok {
		return system, true
	}
	system, ok := localized[exactKey]
	return system, ok
}

func buildSystemHints(tokens []reportToken, matches reportSystemMatches) []IntelSystemHint {
	hints := make([]IntelSystemHint, 0, len(tokens))
	for _, token := range tokens {
		system, ok := matchedSystemForToken(token.exactKey, token.lowerKey, matches.exact, matches.lower, matches.localized)
		if !ok {
			continue
		}
		hints = append(hints, IntelSystemHint{
			SystemID: system.System,
			Name:     system.Name,
		})
	}
	return hints
}

func rebuildReportText(text string, tokens []reportToken, matches reportSystemMatches) string {
	var builder strings.Builder
	builder.Grow(len(text))

	last := 0
	for _, token := range tokens {
		builder.WriteString(text[last:token.start])
		if system, ok := matchedSystemForToken(token.exactKey, token.lowerKey, matches.exact, matches.lower, matches.localized); ok {
			builder.WriteString(token.prefix)
			builder.WriteString(system.Name)
			builder.WriteString(token.cleanSuffix)
		} else {
			builder.WriteString(token.displayText)
		}
		last = token.end
	}
	builder.WriteString(text[last:])
	return builder.String()
}

func appendSystemMatch(systems *[]IntelSystem, seenSystems map[int]struct{}, system IntelSystem) {
	if _, exists := seenSystems[system.System]; exists {
		return
	}
	seenSystems[system.System] = struct{}{}
	*systems = append(*systems, system)
}

func findSystemsByFieldValues(app *pocketbase.PocketBase, field string, values []string) (map[string]IntelSystem, error) {
	if len(values) == 0 {
		return map[string]IntelSystem{}, nil
	}

	matches := map[string]IntelSystem{}
	for start := 0; start < len(values); start += systemLookupBatchSize {
		end := min(start+systemLookupBatchSize, len(values))

		filter, params := buildExactMatchFilter(field, values[start:end])
		records, err := app.FindRecordsByFilter(store.CollectionSolarSystems, filter, "", 0, 0, params)
		if err != nil {
			return nil, err
		}
		for _, rec := range records {
			system := systemFromRecord(rec)
			fieldValue := strings.TrimSpace(rec.GetString(field))
			if fieldValue == "" {
				continue
			}
			if _, exists := matches[fieldValue]; exists {
				continue
			}
			matches[fieldValue] = system
		}
	}
	return matches, nil
}

func findSystemsByFieldValuesLower(app *pocketbase.PocketBase, field string, values []string) (map[string]IntelSystem, error) {
	if len(values) == 0 {
		return map[string]IntelSystem{}, nil
	}

	matches := map[string]IntelSystem{}
	for start := 0; start < len(values); start += systemLookupBatchSize {
		end := min(start+systemLookupBatchSize, len(values))

		filter, params := buildExactMatchFilterLower(field, values[start:end])
		records, err := app.FindRecordsByFilter(store.CollectionSolarSystems, filter, "", 0, 0, params)
		if err != nil {
			return nil, err
		}
		for _, rec := range records {
			system := systemFromRecord(rec)
			fieldValue := strings.TrimSpace(rec.GetString(field))
			if fieldValue == "" {
				continue
			}
			fieldKey := strings.ToLower(fieldValue)
			if _, exists := matches[fieldKey]; exists {
				continue
			}
			matches[fieldKey] = system
		}
	}
	return matches, nil
}

const systemLookupBatchSize = 32

func buildExactMatchFilter(field string, values []string) (filter string, params map[string]any) {
	clauses := make([]string, 0, len(values))
	params = make(map[string]any, len(values))
	for i, value := range values {
		key := "v" + strconv.Itoa(i)
		clauses = append(clauses, field+" = {:"+key+"}")
		params[key] = value
	}
	return "(" + strings.Join(clauses, " || ") + ")", params
}

func buildExactMatchFilterLower(field string, values []string) (filter string, params map[string]any) {
	clauses := make([]string, 0, len(values))
	params = make(map[string]any, len(values))
	for i, value := range values {
		key := "v" + strconv.Itoa(i)
		clauses = append(clauses, field+":lower = {:"+key+"}")
		params[key] = strings.ToLower(value)
	}
	return "(" + strings.Join(clauses, " || ") + ")", params
}

func splitTokenAffixes(token string) (prefix, tokenCore, suffix string) {
	start := 0
	end := len(token)

	for start < end {
		r, size := utf8.DecodeRuneInString(token[start:])
		if !isTokenBoundaryRune(r) {
			break
		}
		start += size
	}

	for end > start {
		r, size := utf8.DecodeLastRuneInString(token[:end])
		if !isTokenBoundaryRune(r) {
			break
		}
		end -= size
	}

	return token[:start], token[start:end], token[end:]
}

func isTokenBoundaryRune(r rune) bool {
	switch r {
	case '*', '"', '\'', '(', ')', '[', ']', '{', '}', '<', '>', '.', ',', ';', ':', '!', '?':
		return true
	default:
		return false
	}
}

func containsNonASCII(text string) bool {
	for _, r := range text {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

func systemFromRecord(rec *core.Record) IntelSystem {
	return IntelSystem{
		System:        rec.GetInt("eve_id"),
		Name:          rec.GetString("name"),
		Constellation: rec.GetInt("constellation"),
		Region:        rec.GetInt("region_id"),
	}
}
