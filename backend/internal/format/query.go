package format

import (
	"net/url"
	"strconv"
	"strings"
)

func GetQueryList(values url.Values, key string) []string {
	if values == nil {
		return nil
	}
	raw := append([]string{}, values[key]...)
	if extra := values[key+"[]"]; len(extra) > 0 {
		raw = append(raw, extra...)
	}
	if len(raw) == 0 {
		if single := strings.TrimSpace(values.Get(key)); single != "" {
			raw = append(raw, single)
		}
	}
	out := []string{}
	for _, value := range raw {
		out = append(out, SplitTokens(value)...)
	}
	return out
}

func SplitTokens(value string) []string {
	raw := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '+'
	})
	out := make([]string, 0, len(raw))
	for _, token := range raw {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		out = append(out, token)
	}
	return out
}

func GetPositiveInt(values url.Values, key string, defaultValue, maxValue int) int {
	if values == nil {
		return defaultValue
	}
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	if maxValue > 0 && parsed > maxValue {
		return defaultValue
	}
	return parsed
}
