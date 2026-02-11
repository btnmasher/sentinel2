package intel

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/store"
)

var reportPattern = regexp.MustCompile(`\[ (?P<date>.*) \] (?P<author>[\s\w\-']+) > (?P<text>.*)`)

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

	if len(body) > 256 {
		body = body[:256]
	}

	return ParsedReport{Date: parsed, Author: author, Text: body}, nil
}

func LinkSystemNames(app *pocketbase.PocketBase, text string) ([]IntelSystem, error) {
	words := strings.Fields(strings.ReplaceAll(text, "*", ""))
	filters := []string{}
	params := map[string]any{}
	for i, word := range words {
		if len(word) < 3 {
			continue
		}
		key := "w" + strconv.Itoa(i)
		filters = append(filters, "name ~ {:"+key+"}")
		params[key] = word + "%"
	}

	if len(filters) == 0 {
		return []IntelSystem{}, nil
	}

	filter := "(" + strings.Join(filters, " || ") + ")"
	records, recordsErr := app.FindRecordsByFilter(store.CollectionSolarSystems, filter, "", 0, 0, params)
	if recordsErr != nil {
		return nil, recordsErr
	}

	systems := []IntelSystem{}
	for _, rec := range records {
		name := rec.GetString("name")
		if strings.Contains(name, " ") || !strings.Contains(name, "-") {
			continue
		}
		systems = append(systems, IntelSystem{
			System:        int(rec.GetInt("eve_id")),
			Name:          name,
			Constellation: int(rec.GetInt("constellation")),
			Region:        int(rec.GetInt("region_id")),
		})
	}
	return systems, nil
}
