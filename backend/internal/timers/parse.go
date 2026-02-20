package timers

import (
	"regexp"
	"strings"
	"time"
)

const (
	timerDateLayout = "2006.01.02 15:04:05"

	namedGroupEvent  = "event"
	namedGroupDate   = "date"
	namedGroupTitle  = "title"
	namedGroupSystem = "system"

	timerEventAnchoring  = "anchoring"
	timerEventReinforced = "reinforced"

	timerKindAnchoring     = "anchoring"
	timerKindReinforcement = "reinforcement"
	timerKindSkyhook       = "skyhook"
	timerKindMoonOre       = "moon_ore"
	timerKindCustom        = "custom"

	tokenSkyhook = "skyhook"
	tokenMoon    = "moon"
)

var (
	untilRe = regexp.MustCompile(`(?i)\b(?P<event>Reinforced|Anchoring)\s+until\s+(?P<date>\d{4}\.\d{2}\.\d{2}\s+\d{2}:\d{2}:\d{2})`)
	dateRe  = regexp.MustCompile(`\b(\d{4}\.\d{2}\.\d{2}\s+\d{2}:\d{2}:\d{2})\b`)

	structureWithParenSystemRe = regexp.MustCompile(`^\s*(?P<title>.*?)\s*\(\s*(?P<system>[^\s)]+)[^)]*\)`)
	systemDashStructureRe      = regexp.MustCompile(`^\s*(?P<system>[A-Za-z0-9-]+)\s*-\s*(?P<title>.*?)(?:\s+\d[\d,]*\s*(?:km|m)\b|\s+Sec\.\s*[\d.]+|\s+(?:Reinforced|Anchoring)\s+until\b|$)`)
	systemTokenRe              = regexp.MustCompile(`\(([A-Za-z0-9-]{2,})\)`)
)

func parseText(raw string) (ParseResult, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ParseResult{}, ErrEmptyTimerText
	}

	var (
		eventToken string
		dateToken  string
	)
	if groups := matchNamed(untilRe, text); groups != nil {
		eventToken = strings.ToLower(strings.TrimSpace(groups[namedGroupEvent]))
		dateToken = strings.TrimSpace(groups[namedGroupDate])
	}
	if dateToken == "" {
		matches := dateRe.FindStringSubmatch(text)
		if len(matches) > 1 {
			dateToken = strings.TrimSpace(matches[1])
		}
	}
	if dateToken == "" {
		return ParseResult{}, ErrNoTimerDateFound
	}

	expiresAt, err := time.Parse(timerDateLayout, dateToken)
	if err != nil {
		return ParseResult{}, ErrInvalidTimerDate
	}
	expiresAt = time.Date(expiresAt.Year(), expiresAt.Month(), expiresAt.Day(), expiresAt.Hour(), expiresAt.Minute(), expiresAt.Second(), 0, time.UTC)

	title, system := extractTitleAndSystem(text)
	timerKind := timerKindFromText(text, eventToken)

	return ParseResult{
		Title:     title,
		System:    system,
		TimerKind: timerKind,
		ExpiresAt: expiresAt,
		Raw:       text,
	}, nil
}

func extractTitleAndSystem(text string) (title, system string) {
	if groups := matchNamed(structureWithParenSystemRe, text); groups != nil {
		return normalizeToken(groups[namedGroupTitle]), normalizeToken(groups[namedGroupSystem])
	}
	if groups := matchNamed(systemDashStructureRe, text); groups != nil {
		return normalizeToken(groups[namedGroupTitle]), normalizeToken(groups[namedGroupSystem])
	}
	if match := systemTokenRe.FindStringSubmatch(text); len(match) > 1 {
		system = normalizeToken(match[1])
	}
	return "", system
}

func timerKindFromText(text, eventToken string) string {
	switch eventToken {
	case timerEventAnchoring:
		return timerKindAnchoring
	case timerEventReinforced:
		return timerKindReinforcement
	}

	lowered := strings.ToLower(text)
	switch {
	case strings.Contains(lowered, tokenSkyhook):
		return timerKindSkyhook
	case strings.Contains(lowered, tokenMoon):
		return timerKindMoonOre
	default:
		return timerKindCustom
	}
}

func normalizeToken(value string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}

func matchNamed(re *regexp.Regexp, text string) map[string]string {
	matches := re.FindStringSubmatch(text)
	if len(matches) == 0 {
		return nil
	}
	out := map[string]string{}
	for i, name := range re.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		out[name] = matches[i]
	}
	return out
}
