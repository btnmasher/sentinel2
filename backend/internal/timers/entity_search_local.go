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
	return s.searchEntitiesFromESI(ctx, q, limit, requester)
}
