package timers

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"

	esipkg "sentinel2/internal/esi"
	"sentinel2/internal/store"
)

const doomheimStationID = 60000001
const entitySearchCandidateFactor = 4
const entitySearchCandidateFloor = 40

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
		// Prefer alliance hits first for the mixed owner-search experience.
		results = s.resolveAllianceMatches(ctx, source, allianceIDs, results, limit)
		results = s.resolveCorporationMatches(ctx, source, corporationIDs, results, limit)
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
	capIDs := limitEntitySearchCandidates(corporationIDs, limit)
	cacheByID := store.GetOrgCacheEntries(s.App, store.CollectionCorporations, capIDs)
	upsertByCorpID := make(map[int]store.CorporationProfileUpsert, len(capIDs))

	type rankedCorp struct {
		corporationID int
		entity        EntitySearchItem
		memberCount   int
	}
	withAlliance := make([]rankedCorp, 0, len(capIDs))
	withoutAlliance := make([]rankedCorp, 0, len(capIDs))

	for _, corporationID := range capIDs {
		entry, exists := cacheByID[corporationID]
		if exists && entry.Closed {
			s.logOrganizationResolveError("corporation", source, corporationID, esipkg.ErrOrganizationInactive)
			continue
		}
		var profile *esipkg.CorporationProfile
		if exists && entry.Name != "" {
			profile = &esipkg.CorporationProfile{
				Name:        entry.Name,
				Ticker:      entry.Ticker,
				AllianceID:  entry.AllianceID,
				MemberCount: entry.MemberCount,
			}
		} else {
			if s.PublicESI == nil {
				s.logOrganizationResolveError("corporation", source, corporationID, ErrESIPublicClientNotConfigured)
				continue
			}
			profileFetched, fetchErr := s.PublicESI.CorporationProfile(ctx, corporationID)
			if fetchErr != nil {
				if errors.Is(fetchErr, esipkg.ErrOrganizationInactive) {
					_ = store.SetOrgClosed(s.App, store.CollectionCorporations, corporationID, true)
				}
				s.logOrganizationResolveError("corporation", source, corporationID, fetchErr)
				continue
			}
			if isClosedCorporationHeuristic(profileFetched.MemberCount, profileFetched.HomeStationID) {
				_ = store.SetOrgClosed(s.App, store.CollectionCorporations, corporationID, true)
				s.logOrganizationResolveError("corporation", source, corporationID, esipkg.ErrOrganizationInactive)
				continue
			}
			profile = &profileFetched
			upsertByCorpID[corporationID] = store.CorporationProfileUpsert{
				EveID:       corporationID,
				Name:        profileFetched.Name,
				Ticker:      profileFetched.Ticker,
				AllianceID:  profileFetched.AllianceID,
				MemberCount: profileFetched.MemberCount,
			}
		}

		entity, memberCount, resolveErr := s.resolveCorporation(ctx, corporationID, profile)
		if resolveErr != nil {
			s.logOrganizationResolveError("corporation", source, corporationID, resolveErr)
			continue
		}
		if entity.ParentAlliance != nil && entity.ParentAlliance.ID > 0 {
			withAlliance = append(withAlliance, rankedCorp{
				corporationID: corporationID,
				entity:        entity,
				memberCount:   memberCount,
			})
			continue
		}
		withoutAlliance = append(withoutAlliance, rankedCorp{
			corporationID: corporationID,
			entity:        entity,
			memberCount:   memberCount,
		})
	}

	sort.SliceStable(withAlliance, func(i, j int) bool {
		return withAlliance[i].memberCount > withAlliance[j].memberCount
	})
	sort.SliceStable(withoutAlliance, func(i, j int) bool {
		return withoutAlliance[i].memberCount > withoutAlliance[j].memberCount
	})

	selectedCorpIDs := make([]int, 0, limit)
	for _, ranked := range withAlliance {
		if len(results) >= limit {
			break
		}
		results = appendEntitySearchResult(results, ranked.entity, limit)
		selectedCorpIDs = append(selectedCorpIDs, ranked.corporationID)
	}
	for _, ranked := range withoutAlliance {
		if len(results) >= limit {
			break
		}
		results = appendEntitySearchResult(results, ranked.entity, limit)
		selectedCorpIDs = append(selectedCorpIDs, ranked.corporationID)
	}

	if len(upsertByCorpID) > 0 && len(selectedCorpIDs) > 0 {
		selectedUpserts := make([]store.CorporationProfileUpsert, 0, len(selectedCorpIDs))
		for _, corporationID := range selectedCorpIDs {
			upsert, ok := upsertByCorpID[corporationID]
			if !ok {
				continue
			}
			selectedUpserts = append(selectedUpserts, upsert)
		}
		if len(selectedUpserts) > 0 {
			if upsertErr := store.UpsertCorporationProfiles(s.App, selectedUpserts); upsertErr != nil {
				s.App.Logger().Warn(
					"timer entity search corporation bulk cache upsert failed",
					slog.Any("error", upsertErr),
					slog.Int("count", len(selectedUpserts)),
				)
			}
		}
	}
	return results
}

func (s *Service) resolveAllianceMatches(ctx context.Context, source string, allianceIDs []int, results []EntitySearchItem, limit int) []EntitySearchItem {
	capIDs := limitEntitySearchCandidates(allianceIDs, limit)
	cacheByID := store.GetOrgCacheEntries(s.App, store.CollectionAlliances, capIDs)
	type rankedAlliance struct {
		entity                 EntitySearchItem
		memberCorporationCount int
	}
	rankedAlliances := make([]rankedAlliance, 0, len(capIDs))
	bulkUpserts := make([]store.AllianceProfileUpsert, 0, len(capIDs))

	for _, allianceID := range capIDs {
		entry, exists := cacheByID[allianceID]
		if exists && entry.Closed {
			s.logOrganizationResolveError("alliance", source, allianceID, esipkg.ErrOrganizationInactive)
			continue
		}

		memberCorporationCount := 0
		if exists {
			memberCorporationCount = entry.MemberCorporationCount
		}

		entity := EntitySearchItem{}
		if exists && entry.Name != "" {
			entity = EntitySearchItem{
				Type:   "alliance",
				ID:     allianceID,
				Name:   entry.Name,
				Ticker: entry.Ticker,
			}
		} else {
			if s.PublicESI == nil {
				s.logOrganizationResolveError("alliance", source, allianceID, ErrESIPublicClientNotConfigured)
				continue
			}
			name, ticker, detailsErr := s.PublicESI.AllianceDetails(ctx, allianceID)
			if detailsErr != nil {
				if errors.Is(detailsErr, esipkg.ErrOrganizationInactive) {
					_ = store.SetOrgClosed(s.App, store.CollectionAlliances, allianceID, true)
				}
				s.logOrganizationResolveError("alliance", source, allianceID, detailsErr)
				continue
			}
			entity = EntitySearchItem{
				Type:   "alliance",
				ID:     allianceID,
				Name:   strings.TrimSpace(name),
				Ticker: strings.TrimSpace(ticker),
			}
			if entity.Name == "" {
				continue
			}
		}

		if memberCorporationCount <= 0 && s.PublicESI != nil {
			corporationIDs, corporationsErr := s.PublicESI.AllianceCorporationIDs(ctx, allianceID)
			if corporationsErr != nil {
				if errors.Is(corporationsErr, esipkg.ErrOrganizationInactive) {
					_ = store.SetOrgClosed(s.App, store.CollectionAlliances, allianceID, true)
					s.logOrganizationResolveError("alliance", source, allianceID, corporationsErr)
					continue
				}
				s.logOrganizationResolveError("alliance", source, allianceID, corporationsErr)
			} else {
				memberCorporationCount = len(corporationIDs)
			}
		}
		bulkUpserts = append(bulkUpserts, store.AllianceProfileUpsert{
			EveID:                  allianceID,
			Name:                   entity.Name,
			Ticker:                 entity.Ticker,
			MemberCorporationCount: memberCorporationCount,
		})

		rankedAlliances = append(rankedAlliances, rankedAlliance{
			entity:                 entity,
			memberCorporationCount: memberCorporationCount,
		})
	}

	sort.SliceStable(rankedAlliances, func(i, j int) bool {
		return rankedAlliances[i].memberCorporationCount > rankedAlliances[j].memberCorporationCount
	})

	if len(bulkUpserts) > 0 {
		if upsertErr := store.UpsertAllianceProfiles(s.App, bulkUpserts); upsertErr != nil {
			s.App.Logger().Warn(
				"timer entity search alliance bulk cache upsert failed",
				slog.Any("error", upsertErr),
				slog.Int("count", len(bulkUpserts)),
			)
		}
	}

	for _, ranked := range rankedAlliances {
		if len(results) >= limit {
			return results
		}
		results = appendEntitySearchResult(results, ranked.entity, limit)
	}
	return results
}

func limitEntitySearchCandidates(ids []int, limit int) []int {
	if len(ids) == 0 || limit <= 0 {
		return ids
	}
	candidateLimit := limit * entitySearchCandidateFactor
	if candidateLimit < entitySearchCandidateFloor {
		candidateLimit = entitySearchCandidateFloor
	}
	if candidateLimit > len(ids) {
		candidateLimit = len(ids)
	}
	return ids[:candidateLimit]
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

func (s *Service) resolveCorporation(
	ctx context.Context,
	corporationID int,
	profile *esipkg.CorporationProfile,
) (EntitySearchItem, int, error) {
	if s == nil {
		return EntitySearchItem{}, 0, ErrESIPublicClientNotConfigured
	}

	name := ""
	ticker := ""
	allianceID := 0
	memberCount := 0
	homeStationID := 0
	profileVerified := false

	if profile != nil {
		name = strings.TrimSpace(profile.Name)
		ticker = strings.TrimSpace(profile.Ticker)
		allianceID = profile.AllianceID
		memberCount = profile.MemberCount
	}

	if name == "" {
		if s.PublicESI == nil {
			return EntitySearchItem{}, 0, ErrESIPublicClientNotConfigured
		}
		profileFetched, fetchErr := s.PublicESI.CorporationProfile(ctx, corporationID)
		if fetchErr != nil {
			if s.App != nil && errors.Is(fetchErr, esipkg.ErrOrganizationInactive) {
				_ = store.SetOrgClosed(s.App, store.CollectionCorporations, corporationID, true)
			}
			return EntitySearchItem{}, 0, fetchErr
		}
		name = strings.TrimSpace(profileFetched.Name)
		if name == "" {
			return EntitySearchItem{}, 0, ErrESIPublicClientNotConfigured
		}
		ticker = strings.TrimSpace(profileFetched.Ticker)
		allianceID = profileFetched.AllianceID
		memberCount = profileFetched.MemberCount
		homeStationID = profileFetched.HomeStationID
		profileVerified = true
		if s.App != nil {
			_ = store.UpsertCorporationProfile(
				s.App,
				corporationID,
				name,
				ticker,
				profileFetched.AllianceID,
				profileFetched.MemberCount,
			)
		}
	}

	if profileVerified && isClosedCorporationHeuristic(memberCount, homeStationID) {
		if s != nil && s.App != nil {
			_ = store.SetOrgClosed(s.App, store.CollectionCorporations, corporationID, true)
		}
		return EntitySearchItem{}, 0, esipkg.ErrOrganizationInactive
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
	}, memberCount, nil
}

func isClosedCorporationHeuristic(memberCount, homeStationID int) bool {
	return memberCount == 0 || homeStationID == doomheimStationID
}

func (s *Service) resolveAlliance(ctx context.Context, allianceID int) (EntitySearchItem, error) {
	if s == nil {
		return EntitySearchItem{}, ErrESIPublicClientNotConfigured
	}
	name, ticker, ok, err := store.GetOrFetchAlliance(ctx, s.App, s.PublicESI, allianceID)
	if err != nil {
		if s.App != nil && errors.Is(err, esipkg.ErrOrganizationInactive) {
			_ = store.SetOrgClosed(s.App, store.CollectionAlliances, allianceID, true)
		}
		return EntitySearchItem{}, err
	}
	if !ok {
		return EntitySearchItem{}, ErrESIPublicClientNotConfigured
	}
	return EntitySearchItem{
		Type:   "alliance",
		ID:     allianceID,
		Name:   name,
		Ticker: ticker,
	}, nil
}
