package format

import (
	"net/url"
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
		for _, token := range SplitTokens(value) {
			out = append(out, token)
		}
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
