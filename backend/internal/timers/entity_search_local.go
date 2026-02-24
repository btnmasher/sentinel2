package timers

import (
	"context"
	"strings"
)

func (s *Service) SearchEntities(ctx context.Context, query string, limit int, requester *EntitySearchRequester) ([]EntitySearchItem, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []EntitySearchItem{}, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	return s.SearchEntitiesWithScope(ctx, q, limit, requester, EntitySearchScopeBoth)
}

func (s *Service) SearchEntitiesWithScope(
	ctx context.Context,
	query string,
	limit int,
	requester *EntitySearchRequester,
	scope EntitySearchScope,
) ([]EntitySearchItem, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []EntitySearchItem{}, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	return s.searchEntitiesFromESI(ctx, q, limit, requester, normalizeEntitySearchScope(scope))
}

func normalizeEntitySearchScope(scope EntitySearchScope) EntitySearchScope {
	switch strings.TrimSpace(strings.ToLower(string(scope))) {
	case string(EntitySearchScopeAlliance):
		return EntitySearchScopeAlliance
	case string(EntitySearchScopeCorporation):
		return EntitySearchScopeCorporation
	default:
		return EntitySearchScopeBoth
	}
}
