package cleanup

import (
	"path/filepath"
	"strings"
)

func matchesAnyRule(rel string, isDir bool, rules []cleanRule, include bool) bool {
	for _, r := range rules {
		if r.include != include {
			continue
		}
		if ruleMatches(rel, isDir, r) {
			return true
		}
	}
	return false
}

func ruleMatches(rel string, isDir bool, rule cleanRule) bool {
	rel = normalizePattern(rel)
	pattern := normalizePattern(rule.pattern)
	rel, pattern = normalizeForMatch(rel, pattern)

	dirOnly := strings.HasSuffix(pattern, "/")
	if dirOnly {
		pattern = strings.TrimSuffix(pattern, "/")
	}
	if !hasGlob(pattern) {
		if dirOnly && !strings.Contains(pattern, "/") {
			if rule.anchored {
				return rel == pattern || strings.HasPrefix(rel, pattern+"/")
			}
			return pathHasComponent(rel, pattern)
		}
		return rel == pattern || strings.HasPrefix(rel, pattern+"/")
	}

	ok, err := filepath.Match(filepath.FromSlash(pattern), filepath.FromSlash(rel))
	if err == nil && ok {
		return true
	}
	ok, err = pathMatch(pattern, rel)
	return err == nil && ok
}

func pathHasComponent(rel, component string) bool {
	if rel == component {
		return true
	}
	parts := strings.Split(rel, "/")
	for _, p := range parts {
		if p == component {
			return true
		}
	}
	return false
}

func pathMatch(pattern, rel string) (bool, error) {
	if !strings.Contains(pattern, "**") {
		return false, nil
	}
	parts := strings.Split(pattern, "**")
	if len(parts) == 2 {
		return strings.HasPrefix(rel, strings.TrimSuffix(parts[0], "*")) &&
			strings.HasSuffix(rel, strings.TrimPrefix(parts[1], "*")), nil
	}
	return false, nil
}

func normalizeForMatch(rel, pattern string) (string, string) {
	if caseInsensitivePatterns {
		return strings.ToLower(rel), strings.ToLower(pattern)
	}
	return rel, pattern
}

func normalizePattern(pattern string) string {
	p := strings.TrimSpace(filepath.ToSlash(pattern))
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimSuffix(p, "/.")
	return p
}

func hasGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}
