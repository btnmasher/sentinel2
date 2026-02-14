package cleanup

import (
	"fmt"
	"path/filepath"
	"strings"
)

func parseCleanRules(raw string) ([]cleanRule, error) {
	normalized := strings.NewReplacer("\n", ",", ";", ",").Replace(raw)
	parts := strings.Split(normalized, ",")
	rules := make([]cleanRule, 0, len(parts))
	for _, p := range parts {
		p = normalizeRuleToken(p)
		if p == "" {
			continue
		}
		r := cleanRule{include: true}
		if strings.HasPrefix(p, `\!`) || strings.HasPrefix(p, `\#`) {
			p = strings.TrimPrefix(p, `\`)
		} else if strings.HasPrefix(p, "!") {
			r.include = false
			p = strings.TrimSpace(strings.TrimPrefix(p, "!"))
		}
		if strings.HasPrefix(p, "/") {
			r.anchored = true
			p = strings.TrimPrefix(p, "/")
		}
		r.pattern = strings.ReplaceAll(p, `\#`, "#")
		r.pattern = normalizePattern(r.pattern)
		if r.pattern == "" {
			return nil, fmt.Errorf("invalid clean rule %q: empty pattern", p)
		}
		if hasGlob(r.pattern) {
			if _, err := filepath.Match(filepath.FromSlash(r.pattern), ""); err != nil {
				return nil, fmt.Errorf("invalid clean rule %q: %w", p, err)
			}
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func normalizeRuleToken(token string) string {
	return strings.TrimSpace(stripUnescapedComment(token))
}

func stripUnescapedComment(token string) string {
	var out strings.Builder
	escaped := false
	for _, r := range token {
		if escaped {
			out.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			out.WriteRune(r)
			continue
		}
		if r == '#' {
			break
		}
		out.WriteRune(r)
	}
	return out.String()
}
