package timers

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	esipkg "sentinel2/internal/esi"
	"sentinel2/internal/store"
)

func (s *Service) searchEntitiesFromESI(
	ctx context.Context,
	query string,
	limit int,
	requester *EntitySearchRequester,
	scope EntitySearchScope,
) ([]EntitySearchItem, error) {
	q := strings.TrimSpace(query)
	if s == nil || s.App == nil || limit <= 0 || q == "" {
		return []EntitySearchItem{}, nil
	}
	return s.searchEntitiesFromAuthenticatedSearch(ctx, q, limit, requester, scope)
}

func (s *Service) searchEntitiesFromAuthenticatedSearch(
	ctx context.Context,
	query string,
	limit int,
	requester *EntitySearchRequester,
	scope EntitySearchScope,
) ([]EntitySearchItem, error) {
	if s == nil || s.App == nil || s.ESI == nil || limit <= 0 || requester == nil {
		return []EntitySearchItem{}, nil
	}
	categories := entitySearchCategories(scope)
	corporationIDs, allianceIDs, err := s.ESI.SearchOrganizations(
		ctx,
		requester.CharacterID,
		requester.AccessToken,
		query,
		false,
		categories,
	)
	if err != nil {
		if errors.Is(err, esipkg.ErrAffiliationUnsupported) {
			return []EntitySearchItem{}, nil
		}
		return []EntitySearchItem{}, err
	}
	s.App.Logger().Debug(
		"timer entity search authenticated esi raw matches",
		slog.String("query", query),
		slog.Int("character_id", requester.CharacterID),
		slog.Int("corporation_match_count", len(corporationIDs)),
		slog.Int("alliance_match_count", len(allianceIDs)),
	)
	return s.resolveEntityMatches(ctx, query, limit, corporationIDs, allianceIDs, "authenticated", scope)
}

func (s *Service) resolveEntityMatches(
	ctx context.Context,
	query string,
	limit int,
	corporationIDs,
	allianceIDs []int,
	source string,
	scope EntitySearchScope,
) ([]EntitySearchItem, error) {
	results := make([]EntitySearchItem, 0, limit)
	switch scope {
	case EntitySearchScopeAlliance:
		results = s.resolveAllianceMatches(ctx, source, allianceIDs, results, limit)
	case EntitySearchScopeCorporation:
		results = s.resolveCorporationMatches(ctx, source, corporationIDs, results, limit)
	default:
		results = s.resolveCorporationMatches(ctx, source, corporationIDs, results, limit)
		results = s.resolveAllianceMatches(ctx, source, allianceIDs, results, limit)
	}

	s.App.Logger().Debug(
		"timer entity search esi resolved matches",
		slog.String("source", source),
		slog.String("query", query),
		slog.Int("result_count", len(results)),
	)
	return results, nil
}

func entitySearchCategories(scope EntitySearchScope) []string {
	switch scope {
	case EntitySearchScopeAlliance:
		return []string{"alliance"}
	case EntitySearchScopeCorporation:
		return []string{"corporation"}
	default:
		return []string{"corporation", "alliance"}
	}
}

func (s *Service) resolveCorporationMatches(ctx context.Context, source string, corporationIDs []int, results []EntitySearchItem, limit int) []EntitySearchItem {
	for _, corporationID := range corporationIDs {
		if len(results) >= limit {
			return results
		}
		entity, resolveErr := s.resolveCorporation(ctx, corporationID)
		if resolveErr != nil {
			s.logOrganizationResolveError("corporation", source, corporationID, resolveErr)
			continue
		}
		results = appendEntitySearchResult(results, entity, limit)
	}
	return results
}

func (s *Service) resolveAllianceMatches(ctx context.Context, source string, allianceIDs []int, results []EntitySearchItem, limit int) []EntitySearchItem {
	for _, allianceID := range allianceIDs {
		if len(results) >= limit {
			return results
		}
		entity, resolveErr := s.resolveAlliance(ctx, allianceID)
		if resolveErr != nil {
			s.logOrganizationResolveError("alliance", source, allianceID, resolveErr)
			continue
		}
		results = appendEntitySearchResult(results, entity, limit)
	}
	return results
}

func appendEntitySearchResult(results []EntitySearchItem, entity EntitySearchItem, limit int) []EntitySearchItem {
	if len(results) >= limit || entity.ID <= 0 || entity.Name == "" {
		return results
	}
	return append(results, entity)
}

func (s *Service) logOrganizationResolveError(kind, source string, eveID int, resolveErr error) {
	if s == nil || s.App == nil || resolveErr == nil {
		return
	}
	if errors.Is(resolveErr, esipkg.ErrOrganizationInactive) {
		s.App.Logger().Info(
			"timer entity search filtered inactive "+kind,
			slog.String("source", source),
			slog.Int(kind+"_id", eveID),
		)
		return
	}
	s.App.Logger().Warn(
		"timer entity search "+kind+" resolve failed",
		slog.String("source", source),
		slog.Int(kind+"_id", eveID),
		slog.Any("error", resolveErr),
	)
}

func (s *Service) resolveCorporation(ctx context.Context, corporationID int) (EntitySearchItem, error) {
	if s == nil {
		return EntitySearchItem{}, ErrESIPublicClientNotConfigured
	}
	if s.App != nil {
		if name, ticker, ok := store.GetOrg(s.App, store.CollectionCorporations, corporationID); ok {
			return EntitySearchItem{
				Type:   "corporation",
				ID:     corporationID,
				Name:   name,
				Ticker: ticker,
			}, nil
		}
	}
	if s.PublicESI == nil {
		return EntitySearchItem{}, ErrESIPublicClientNotConfigured
	}
	name, ticker, allianceID, err := s.PublicESI.CorporationDetails(ctx, corporationID)
	if err != nil {
		return EntitySearchItem{}, err
	}
	if upsertErr := upsertOrganizationRecord(s.App, store.CollectionCorporations, corporationID, name, ticker); upsertErr != nil {
		return EntitySearchItem{}, upsertErr
	}
	var parentAlliance *EntitySearchAlliance
	if allianceID > 0 {
		alliance, allianceErr := s.resolveAlliance(ctx, allianceID)
		if allianceErr == nil {
			parentAlliance = &EntitySearchAlliance{
				ID:     alliance.ID,
				Name:   alliance.Name,
				Ticker: alliance.Ticker,
			}
		}
	}
	return EntitySearchItem{
		Type:           "corporation",
		ID:             corporationID,
		Name:           name,
		Ticker:         ticker,
		ParentAlliance: parentAlliance,
	}, nil
}

func (s *Service) resolveAlliance(ctx context.Context, allianceID int) (EntitySearchItem, error) {
	if s == nil {
		return EntitySearchItem{}, ErrESIPublicClientNotConfigured
	}
	if s.App != nil {
		if name, ticker, ok := store.GetOrg(s.App, store.CollectionAlliances, allianceID); ok {
			return EntitySearchItem{
				Type:   "alliance",
				ID:     allianceID,
				Name:   name,
				Ticker: ticker,
			}, nil
		}
	}
	if s.PublicESI == nil {
		return EntitySearchItem{}, ErrESIPublicClientNotConfigured
	}
	name, ticker, err := s.PublicESI.AllianceDetails(ctx, allianceID)
	if err != nil {
		return EntitySearchItem{}, err
	}
	if upsertErr := upsertOrganizationRecord(s.App, store.CollectionAlliances, allianceID, name, ticker); upsertErr != nil {
		return EntitySearchItem{}, upsertErr
	}
	return EntitySearchItem{
		Type:   "alliance",
		ID:     allianceID,
		Name:   name,
		Ticker: ticker,
	}, nil
}

func upsertOrganizationRecord(app core.App, collectionName string, eveID int, name, ticker string) error {
	if app == nil || eveID <= 0 || strings.TrimSpace(name) == "" {
		return nil
	}
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return err
	}
	records, recordsErr := app.FindRecordsByFilter(collectionName, "eve_id = {:id}", "", 1, 0, dbx.Params{"id": eveID})
	if recordsErr != nil {
		return recordsErr
	}
	var record *core.Record
	if len(records) > 0 {
		record = records[0]
	} else {
		record = core.NewRecord(collection)
		record.Set("eve_id", eveID)
	}
	record.Set("name", name)
	if record.Collection().Fields.GetByName("ticker") != nil {
		record.Set("ticker", ticker)
	}
	if record.Collection().Fields.GetByName("updated_at") != nil {
		record.Set("updated_at", types.NowDateTime())
	}
	return app.Save(record)
}
