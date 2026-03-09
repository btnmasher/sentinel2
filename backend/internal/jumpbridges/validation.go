package jumpbridges

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/format"
	"sentinel2/internal/logging"
	"sentinel2/internal/shared/queryhelpers"
	"sentinel2/internal/store"
)

const (
	ansiblexTypeID            = 35841
	scopeSearchStructures     = "esi-search.search_structures.v1"
	scopeReadStructures       = "esi-universe.read_structures.v1"
	maxCandidateCharacterScan = 50
)

type PairValidationSummary struct {
	CharacterIDs         []int
	SkippedOrganizations int
	TotalPairs           int
	ValidPairs           int
	InvalidPairs         int
	SkippedPairs         int
	RemovedPairs         int
	RemovedKeys          []string
	RemovedNames         []string
}

type structureCharacterCandidate struct {
	CharacterID int
	Token       string
}

type structureValidationContext struct {
	characters               []structureCharacterCandidate
	systemStructureIDs       map[int]int64
	allowedAllianceSet       map[int]struct{}
	allowedCorporationSet    map[int]struct{}
	corporationAllianceCache map[int]int
}

func (s *JumpbridgeService) ValidateExistingPairsWithAllowedCharacters(ctx context.Context) (PairValidationSummary, error) {
	allowedAllianceIDs, allowedCorporationIDs, allowedErr := s.loadOwnerAllowedOrganizationIDs()
	if allowedErr != nil {
		return PairValidationSummary{}, allowedErr
	}

	characters, skippedOrganizations, err := s.pickAllowedStructureCharactersDetailed()
	if err != nil {
		return PairValidationSummary{}, err
	}
	pairs, err := s.loadUniquePairs()
	if err != nil {
		return PairValidationSummary{}, err
	}
	characterIDs := make([]int, 0, len(characters))
	for _, character := range characters {
		characterIDs = append(characterIDs, character.CharacterID)
	}
	validation := structureValidationContext{
		characters:               characters,
		systemStructureIDs:       map[int]int64{},
		allowedAllianceSet:       toIntSet(allowedAllianceIDs),
		allowedCorporationSet:    toIntSet(allowedCorporationIDs),
		corporationAllianceCache: map[int]int{},
	}
	summary := PairValidationSummary{
		CharacterIDs:         characterIDs,
		SkippedOrganizations: skippedOrganizations,
		TotalPairs:           len(pairs),
		RemovedKeys:          []string{},
		RemovedNames:         []string{},
	}
	s.logger.WithFields(logging.Fields{
		"pair_count":            summary.TotalPairs,
		"character_ids":         summary.CharacterIDs,
		"character_count":       len(summary.CharacterIDs),
		"skipped_organizations": summary.SkippedOrganizations,
	}).Debug("jumpbridge validation started")
	for _, pair := range pairs {
		if processErr := s.validateExistingPair(ctx, &validation, pair, &summary); processErr != nil {
			return summary, processErr
		}
	}
	s.logger.WithFields(logging.Fields{
		"total_pairs":           summary.TotalPairs,
		"valid_pairs":           summary.ValidPairs,
		"invalid_pairs":         summary.InvalidPairs,
		"skipped_pairs":         summary.SkippedPairs,
		"skipped_organizations": summary.SkippedOrganizations,
		"removed_pairs":         summary.RemovedPairs,
		"removed_keys":          summary.RemovedKeys,
		"removed_names":         summary.RemovedNames,
	}).Info("jumpbridge validation completed")
	return summary, nil
}

func (s *JumpbridgeService) validateExistingPair(
	ctx context.Context,
	validation *structureValidationContext,
	pair jumpbridgePair,
	summary *PairValidationSummary,
) error {
	fromSystem, toSystem, lookupErr := s.lookupPairSystems(pair)
	if lookupErr != nil {
		return lookupErr
	}
	fromStructureID, toStructureID, resolveErr := s.resolvePairStructureIDsWithContext(
		ctx,
		validation,
		fromSystem,
		toSystem,
		pair.fromStructureID,
		pair.toStructureID,
	)
	if resolveErr != nil {
		return s.handlePairValidationResolutionError(ctx, validation, pair, fromSystem, toSystem, summary, resolveErr)
	}

	ownerAllowed, authoritative, ownerReason := s.isPairOwnerAllowedForValidation(ctx, validation, fromStructureID, toStructureID)
	if !ownerAllowed {
		if !authoritative {
			summary.SkippedPairs++
			s.logger.WithFields(logging.Fields{
				"from_system_id":    pair.from,
				"to_system_id":      pair.to,
				"pair_key":          formatPairKey(pair.from, pair.to),
				"from_structure_id": fromStructureID,
				"to_structure_id":   toStructureID,
				"reason":            ownerReason,
			}).Warn("jumpbridge validation kept pair due to non-authoritative owner check")
			return nil
		}

		summary.InvalidPairs++
		removed, removeErr := s.RemovePair(pair.from, pair.to)
		if removeErr != nil {
			return removeErr
		}
		if removed > 0 {
			summary.RemovedPairs += removed / pairDirectionCount
			summary.RemovedKeys = append(summary.RemovedKeys, formatPairKey(pair.from, pair.to))
			summary.RemovedNames = append(summary.RemovedNames, formatPairName(fromSystem, toSystem))
		}
		s.logger.WithFields(logging.Fields{
			"from_system_id":    pair.from,
			"to_system_id":      pair.to,
			"pair_key":          formatPairKey(pair.from, pair.to),
			"from_structure_id": fromStructureID,
			"to_structure_id":   toStructureID,
			"reason":            ownerReason,
			"removed":           removed / pairDirectionCount,
		}).Warn("jumpbridge validation removed pair failing owner allowlist gate")
		return nil
	}

	if backfillErr := s.backfillPairStructureIDs(pair.from, pair.to, fromStructureID, toStructureID); backfillErr != nil {
		return backfillErr
	}
	s.logger.WithFields(logging.Fields{
		"from_system_id":    pair.from,
		"to_system_id":      pair.to,
		"pair_key":          formatPairKey(pair.from, pair.to),
		"from_structure_id": fromStructureID,
		"to_structure_id":   toStructureID,
	}).Debug("jumpbridge validation pair confirmed")
	summary.ValidPairs++
	return nil
}

func (s *JumpbridgeService) isPairOwnerAllowedForValidation(
	ctx context.Context,
	validation *structureValidationContext,
	fromStructureID int64,
	toStructureID int64,
) (allowed, authoritative bool, reason string) {
	if validation == nil {
		return false, false, "missing validation context"
	}

	if len(validation.allowedAllianceSet) == 0 && len(validation.allowedCorporationSet) == 0 {
		return false, true, "no allowed organizations configured"
	}

	fromAllowed, fromAuthoritative, fromReason := s.isSingleStructureOwnerAllowedForValidation(ctx, validation, fromStructureID)
	toAllowed, toAuthoritative, toReason := s.isSingleStructureOwnerAllowedForValidation(ctx, validation, toStructureID)
	if fromAllowed || toAllowed {
		return true, fromAuthoritative && toAuthoritative, "at least one endpoint owner is allowed"
	}

	if !fromAuthoritative || !toAuthoritative {
		return false, false, fmt.Sprintf("non-authoritative owner check: from=%s; to=%s", fromReason, toReason)
	}
	return false, true, fmt.Sprintf("owner gate failed: from=%s; to=%s", fromReason, toReason)
}

func (s *JumpbridgeService) isSingleStructureOwnerAllowedForValidation(
	ctx context.Context,
	validation *structureValidationContext,
	structureID int64,
) (allowed, authoritative bool, reason string) {
	if structureID <= 0 {
		return false, false, "missing structure id"
	}

	structure, _, err := s.fetchStructureAcrossCandidates(ctx, structureID, validation.characters)
	if err != nil {
		if isNonAuthoritativeStructureLookupError(err) {
			return false, false, err.Error()
		}
		return false, true, fmt.Sprintf("failed loading structure: %v", err)
	}

	allowed, ownerReason := s.isStructureOwnerAllowed(
		ctx,
		structure.OwnerID,
		validation.allowedAllianceSet,
		validation.allowedCorporationSet,
		validation.corporationAllianceCache,
	)
	return allowed, true, ownerReason
}

func (s *JumpbridgeService) lookupPairSystems(pair jumpbridgePair) (fromSystem, toSystem *core.Record, err error) {
	fromSystem, fromErr := s.findSystemByEveID(pair.from)
	if fromErr != nil {
		return nil, nil, fmt.Errorf("pair validation failed resolving from system %d: %w", pair.from, fromErr)
	}
	toSystem, toErr := s.findSystemByEveID(pair.to)
	if toErr != nil {
		return nil, nil, fmt.Errorf("pair validation failed resolving to system %d: %w", pair.to, toErr)
	}
	return fromSystem, toSystem, nil
}

func (s *JumpbridgeService) handlePairValidationResolutionError(
	ctx context.Context,
	validation *structureValidationContext,
	pair jumpbridgePair,
	fromSystem *core.Record,
	toSystem *core.Record,
	summary *PairValidationSummary,
	resolveErr error,
) error {
	if isNonAuthoritativeStructureLookupError(resolveErr) {
		summary.SkippedPairs++
		s.logger.WithFields(logging.Fields{
			"from_system_id": pair.from,
			"to_system_id":   pair.to,
			"pair_key":       formatPairKey(pair.from, pair.to),
			"reason":         resolveErr.Error(),
		}).Warn("jumpbridge validation skipped pair due to non-authoritative lookup error")
		return nil
	}

	if pair.fromStructureID <= 0 || pair.toStructureID <= 0 {
		summary.SkippedPairs++
		s.logger.WithFields(logging.Fields{
			"from_system_id":    pair.from,
			"to_system_id":      pair.to,
			"pair_key":          formatPairKey(pair.from, pair.to),
			"from_structure_id": pair.fromStructureID,
			"to_structure_id":   pair.toStructureID,
			"reason":            resolveErr.Error(),
		}).Warn("jumpbridge validation kept pair with missing structure ids after unresolved lookup")
		return nil
	}
	shouldRemove, removeReason := s.shouldRemovePairForConfirmedMissingStructures(ctx, validation, pair)
	if !shouldRemove {
		summary.SkippedPairs++
		s.logger.WithFields(logging.Fields{
			"from_system_id":    pair.from,
			"to_system_id":      pair.to,
			"pair_key":          formatPairKey(pair.from, pair.to),
			"from_structure_id": pair.fromStructureID,
			"to_structure_id":   pair.toStructureID,
			"reason":            resolveErr.Error(),
			"remove_reason":     removeReason,
		}).Warn("jumpbridge validation kept pair because structure invalidation was not confirmed")
		return nil
	}
	summary.InvalidPairs++
	removed, removeErr := s.RemovePair(pair.from, pair.to)
	if removeErr != nil {
		return removeErr
	}

	if removed > 0 {
		summary.RemovedPairs += removed / pairDirectionCount
		summary.RemovedKeys = append(summary.RemovedKeys, formatPairKey(pair.from, pair.to))
		summary.RemovedNames = append(summary.RemovedNames, formatPairName(fromSystem, toSystem))
		s.logger.WithFields(logging.Fields{
			"from_system_id": pair.from,
			"to_system_id":   pair.to,
			"pair_key":       formatPairKey(pair.from, pair.to),
			"removed":        removed / pairDirectionCount,
			"reason":         removeReason,
		}).Warn("jumpbridge validation removed invalid pair")
	}
	return nil
}

func (s *JumpbridgeService) validateResolvedPairsWithAllowedCharacters(ctx context.Context, pairs []resolvedJumpbridgeEntry) error {
	if len(pairs) == 0 {
		return nil
	}
	characters, _, err := s.pickAllowedStructureCharactersDetailed()
	if err != nil {
		return err
	}
	validation := structureValidationContext{
		characters:         characters,
		systemStructureIDs: map[int]int64{},
	}
	for i := range pairs {
		fromStructureID, toStructureID, resolveErr := s.resolvePairStructureIDsWithContext(
			ctx,
			&validation,
			pairs[i].fromSystem,
			pairs[i].toSystem,
			pairs[i].fromStructureID,
			pairs[i].toStructureID,
		)
		if resolveErr != nil {
			return resolveErr
		}
		pairs[i].fromStructureID = fromStructureID
		pairs[i].toStructureID = toStructureID
	}
	return nil
}

func (s *JumpbridgeService) resolvePairStructureIDsWithAllowedCharacters(
	ctx context.Context,
	fromSystem,
	toSystem *core.Record,
	preferredFromStructureID,
	preferredToStructureID int64,
) (fromStructureID, toStructureID int64, err error) {
	characters, _, err := s.pickAllowedStructureCharactersDetailed()
	if err != nil {
		return 0, 0, err
	}
	validation := structureValidationContext{
		characters:         characters,
		systemStructureIDs: map[int]int64{},
	}
	return s.resolvePairStructureIDsWithContext(
		ctx,
		&validation,
		fromSystem,
		toSystem,
		preferredFromStructureID,
		preferredToStructureID,
	)
}

func (s *JumpbridgeService) resolvePairStructureIDsWithContext(
	ctx context.Context,
	validation *structureValidationContext,
	fromSystem,
	toSystem *core.Record,
	preferredFromStructureID,
	preferredToStructureID int64,
) (fromStructureID, toStructureID int64, err error) {
	if validation == nil {
		return 0, 0, fmt.Errorf("missing validation context")
	}

	if s.ESI == nil {
		return 0, 0, fmt.Errorf("esi client unavailable for jumpbridge validation")
	}

	if fromSystem == nil || toSystem == nil {
		return 0, 0, fmt.Errorf("missing system records")
	}
	fromID := fromSystem.GetInt("eve_id")
	toID := toSystem.GetInt("eve_id")
	fromName := strings.TrimSpace(fromSystem.GetString("name"))
	toName := strings.TrimSpace(toSystem.GetString("name"))

	fromStructureID, fromErr := s.resolveSystemStructureIDWithContext(ctx, validation, fromID, fromName, preferredFromStructureID)
	if fromErr != nil {
		return 0, 0, fmt.Errorf("failed validating %s ansiblex structures: %w", fromName, fromErr)
	}

	if fromStructureID <= 0 {
		return 0, 0, fmt.Errorf("no accessible ansiblex structure found in %s", fromName)
	}

	toStructureID, toErr := s.resolveSystemStructureIDWithContext(ctx, validation, toID, toName, preferredToStructureID)
	if toErr != nil {
		return 0, 0, fmt.Errorf("failed validating %s ansiblex structures: %w", toName, toErr)
	}

	if toStructureID <= 0 {
		return 0, 0, fmt.Errorf("no accessible ansiblex structure found in %s", toName)
	}

	return fromStructureID, toStructureID, nil
}

func (s *JumpbridgeService) resolveSystemStructureIDWithContext(
	ctx context.Context,
	validation *structureValidationContext,
	systemID int,
	systemName string,
	preferredStructureID int64,
) (int64, error) {
	if preferredStructureID > 0 {
		structure, err := s.fetchUniverseStructureByIDAcrossCandidates(ctx, validation, preferredStructureID)
		if err == nil && structure.TypeID == ansiblexTypeID && structure.SystemID == systemID {
			validation.systemStructureIDs[systemID] = preferredStructureID
			return preferredStructureID, nil
		}
	}

	if cachedStructureID, ok := validation.systemStructureIDs[systemID]; ok {
		return cachedStructureID, nil
	}

	structureID, err := s.searchSystemStructureAcrossCandidates(ctx, validation, systemID, systemName)
	if err != nil {
		return 0, err
	}
	validation.systemStructureIDs[systemID] = structureID
	return structureID, nil
}

func (s *JumpbridgeService) fetchUniverseStructureByIDAcrossCandidates(
	ctx context.Context,
	validation *structureValidationContext,
	structureID int64,
) (structureIDResponse struct {
	TypeID   int
	SystemID int
}, err error) {
	var firstErr error
	for _, character := range validation.characters {
		structure, fetchErr := s.ESI.UniverseStructure(ctx, character.CharacterID, character.Token, structureID)
		if fetchErr != nil {
			if firstErr == nil {
				firstErr = fetchErr
			}
			continue
		}
		return struct {
			TypeID   int
			SystemID int
		}{TypeID: structure.TypeID, SystemID: structure.SystemID}, nil
	}

	if firstErr != nil {
		return structureIDResponse, firstErr
	}
	return structureIDResponse, fmt.Errorf("no eligible characters for structure lookup")
}

func (s *JumpbridgeService) searchSystemStructureAcrossCandidates(
	ctx context.Context,
	validation *structureValidationContext,
	systemID int,
	systemName string,
) (int64, error) {
	var firstErr error
	queried := false
	for _, character := range validation.characters {
		structureID, searchErr := s.searchSystemStructureForCharacter(ctx, character, systemID, systemName)
		if searchErr != nil {
			if firstErr == nil {
				firstErr = searchErr
			}
			continue
		}
		queried = true
		if structureID > 0 {
			return structureID, nil
		}
	}

	if !queried && firstErr != nil {
		return 0, firstErr
	}
	return 0, nil
}

func (s *JumpbridgeService) searchSystemStructureForCharacter(
	ctx context.Context,
	character structureCharacterCandidate,
	systemID int,
	systemName string,
) (int64, error) {
	s.logger.WithFields(logging.Fields{
		"character_id": character.CharacterID,
		"system_id":    systemID,
		"system_name":  systemName,
	}).Debug("jumpbridge structure search attempt")
	structureIDs, searchErr := s.ESI.SearchStructures(ctx, character.CharacterID, character.Token, systemName, false)
	if searchErr != nil {
		s.logger.WithFields(logging.Fields{
			"character_id": character.CharacterID,
			"system_id":    systemID,
			"system_name":  systemName,
			"error":        searchErr.Error(),
		}).Warn("jumpbridge structure search failed")
		return 0, searchErr
	}
	s.logger.WithFields(logging.Fields{
		"character_id":    character.CharacterID,
		"system_id":       systemID,
		"system_name":     systemName,
		"structure_count": len(structureIDs),
	}).Debug("jumpbridge structure search returned candidates")
	seen := map[int64]struct{}{}
	for _, structureID := range structureIDs {
		if _, ok := seen[structureID]; ok {
			continue
		}
		seen[structureID] = struct{}{}
		matchID := s.matchAnsiblexStructureInSystem(ctx, character, systemID, structureID)
		if matchID > 0 {
			return matchID, nil
		}
	}
	return 0, nil
}

func (s *JumpbridgeService) matchAnsiblexStructureInSystem(
	ctx context.Context,
	character structureCharacterCandidate,
	systemID int,
	structureID int64,
) int64 {
	structure, fetchErr := s.ESI.UniverseStructure(ctx, character.CharacterID, character.Token, structureID)
	if fetchErr != nil {
		s.logger.WithFields(logging.Fields{
			"character_id": character.CharacterID,
			"system_id":    systemID,
			"structure_id": structureID,
			"error":        fetchErr.Error(),
		}).Debug("jumpbridge structure candidate fetch failed")
		return 0
	}

	if structure.TypeID != ansiblexTypeID || structure.SystemID != systemID {
		return 0
	}
	s.logger.WithFields(logging.Fields{
		"character_id": character.CharacterID,
		"system_id":    systemID,
		"structure_id": structureID,
	}).Debug("jumpbridge ansiblex structure match found")
	return structureID
}

func isNonAuthoritativeStructureLookupError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "401 unauthorized") ||
		strings.Contains(message, "403 forbidden") ||
		strings.Contains(message, "429") ||
		strings.Contains(message, "rate limit") ||
		strings.Contains(message, "timeout") ||
		strings.Contains(message, "deadline exceeded") ||
		strings.Contains(message, "temporarily unavailable") ||
		strings.Contains(message, "service unavailable")
}

func isConfirmedStructureMissingError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "404 not found") ||
		strings.Contains(message, "structure not found") ||
		strings.Contains(message, "requested resource was not found")
}

func (s *JumpbridgeService) shouldRemovePairForConfirmedMissingStructures(
	ctx context.Context,
	validation *structureValidationContext,
	pair jumpbridgePair,
) (shouldRemove bool, reason string) {
	fromMissing, fromReason := s.isStructureIDConfirmedMissing(ctx, validation, pair.fromStructureID)
	if fromMissing {
		return true, fmt.Sprintf("from structure %d missing: %s", pair.fromStructureID, fromReason)
	}
	toMissing, toReason := s.isStructureIDConfirmedMissing(ctx, validation, pair.toStructureID)
	if toMissing {
		return true, fmt.Sprintf("to structure %d missing: %s", pair.toStructureID, toReason)
	}
	return false, "neither structure ID was confirmed missing"
}

func (s *JumpbridgeService) isStructureIDConfirmedMissing(
	ctx context.Context,
	validation *structureValidationContext,
	structureID int64,
) (missing bool, reason string) {
	if structureID <= 0 {
		return false, "structure id not set"
	}
	_, err := s.fetchUniverseStructureByIDAcrossCandidates(ctx, validation, structureID)
	if err == nil {
		return false, "structure still resolves"
	}

	if isConfirmedStructureMissingError(err) {
		return true, err.Error()
	}

	if isNonAuthoritativeStructureLookupError(err) {
		return false, fmt.Sprintf("non-authoritative lookup failure: %s", err.Error())
	}
	return false, fmt.Sprintf("unconfirmed lookup failure: %s", err.Error())
}

func (s *JumpbridgeService) pickAllowedStructureCharacters() ([]structureCharacterCandidate, error) {
	records, _, err := s.listAllowedStructureCharacterRecordsWithStats()
	if err != nil {
		return nil, err
	}
	out := make([]structureCharacterCandidate, 0, len(records))
	for _, record := range records {
		characterID := record.GetInt("eve_character_id")
		token := strings.TrimSpace(record.GetString("oauth_access_token"))
		if characterID <= 0 || token == "" {
			continue
		}
		out = append(out, structureCharacterCandidate{
			CharacterID: characterID,
			Token:       token,
		})
	}
	return out, nil
}

func (s *JumpbridgeService) pickAllowedStructureCharactersDetailed() ([]structureCharacterCandidate, int, error) {
	records, skippedOrganizations, err := s.listAllowedStructureCharacterRecordsWithStats()
	if err != nil {
		return nil, 0, err
	}
	out := make([]structureCharacterCandidate, 0, len(records))
	for _, record := range records {
		characterID := record.GetInt("eve_character_id")
		token := strings.TrimSpace(record.GetString("oauth_access_token"))
		if characterID <= 0 || token == "" {
			continue
		}
		out = append(out, structureCharacterCandidate{
			CharacterID: characterID,
			Token:       token,
		})
	}
	return out, skippedOrganizations, nil
}

func (s *JumpbridgeService) ListAllowedStructureCharacterRecords() ([]*core.Record, error) {
	records, _, err := s.listAllowedStructureCharacterRecordsWithStats()
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (s *JumpbridgeService) listAllowedStructureCharacterRecordsWithStats() ([]*core.Record, int, error) {
	if s.App == nil {
		return nil, 0, fmt.Errorf("app unavailable")
	}

	allowedAllianceIDs, allowedCorporationIDs, err := s.loadAllowedOrganizationIDs()
	if err != nil {
		return nil, 0, err
	}

	if len(allowedAllianceIDs) == 0 && len(allowedCorporationIDs) == 0 {
		return nil, 0, fmt.Errorf("no allowed organizations configured for jumpbridge validation")
	}

	out := make([]*core.Record, 0, len(allowedAllianceIDs)+len(allowedCorporationIDs))
	seenCharacterIDs := map[int]struct{}{}
	out, skippedAlliances, collectAllianceErr := s.collectAllowedCharactersByField(
		"eve_alliance_id",
		"alliance",
		allowedAllianceIDs,
		seenCharacterIDs,
		out,
	)
	if collectAllianceErr != nil {
		return nil, 0, collectAllianceErr
	}

	out, skippedCorporations, collectCorporationErr := s.collectAllowedCharactersByField(
		"eve_corporation_id",
		"corporation",
		allowedCorporationIDs,
		seenCharacterIDs,
		out,
	)
	if collectCorporationErr != nil {
		return nil, 0, collectCorporationErr
	}

	skippedOrganizations := skippedAlliances + skippedCorporations
	if err := s.ensureAllowedCharacterCandidates(
		out,
		skippedOrganizations,
		len(allowedAllianceIDs),
		len(allowedCorporationIDs),
	); err != nil {
		return nil, skippedOrganizations, err
	}

	s.logger.WithFields(logging.Fields{
		"candidate_count":        len(out),
		"skipped_organizations":  skippedOrganizations,
		"allowed_alliance_count": len(allowedAllianceIDs),
		"allowed_corp_count":     len(allowedCorporationIDs),
	}).Debug("jumpbridge candidate character selection completed")
	slices.SortFunc(out, func(a, b *core.Record) int {
		aid := a.GetInt("eve_character_id")
		bid := b.GetInt("eve_character_id")
		if aid < bid {
			return -1
		}
		if aid > bid {
			return 1
		}
		return 0
	})

	return out, skippedOrganizations, nil
}

func (s *JumpbridgeService) collectAllowedCharactersByField(
	field string,
	organizationType string,
	organizationIDs []int,
	seenCharacterIDs map[int]struct{},
	candidates []*core.Record,
) (updatedCandidates []*core.Record, missingOrganizations int, err error) {
	updatedCandidates = candidates

	for _, organizationID := range organizationIDs {
		record, added, collectErr := s.collectAllowedCharacterRecord(
			field,
			organizationType,
			organizationID,
			seenCharacterIDs,
		)
		if collectErr != nil {
			return nil, 0, collectErr
		}

		if record == nil {
			missingOrganizations++
			continue
		}

		if !added {
			s.logger.WithFields(logging.Fields{
				"organization": fmt.Sprintf("%s:%d", organizationType, organizationID),
			}).Warn("jumpbridge candidate character found but not eligible")
			continue
		}
		updatedCandidates = append(updatedCandidates, record)
	}
	return updatedCandidates, missingOrganizations, nil
}

func (s *JumpbridgeService) ensureAllowedCharacterCandidates(
	candidates []*core.Record,
	skippedOrganizations int,
	allowedAllianceCount int,
	allowedCorporationCount int,
) error {
	if len(candidates) > 0 {
		return nil
	}

	s.logger.WithFields(logging.Fields{
		"skipped_organizations":  skippedOrganizations,
		"allowed_alliance_count": allowedAllianceCount,
		"allowed_corp_count":     allowedCorporationCount,
	}).Warn("jumpbridge candidate character selection found no eligible tokens")
	return fmt.Errorf(
		"allowed-organization characters missing esi auth data (allowed_alliances=%d allowed_corporations=%d skipped_orgs=%d)",
		allowedAllianceCount,
		allowedCorporationCount,
		skippedOrganizations,
	)
}

func (s *JumpbridgeService) collectAllowedCharacterRecord(
	field string,
	organizationType string,
	organizationID int,
	seenCharacterIDs map[int]struct{},
) (record *core.Record, added bool, err error) {
	record, lookupErr := s.findFirstAllowedOrganizationCharacter(field, organizationID)
	if lookupErr != nil {
		return nil, false, lookupErr
	}

	if record == nil {
		s.logger.WithFields(logging.Fields{
			"organization": fmt.Sprintf("%s:%d", organizationType, organizationID),
		}).Warn("jumpbridge candidate character missing for allowed organization")
		return nil, false, nil
	}

	characterID := record.GetInt("eve_character_id")
	token := strings.TrimSpace(record.GetString("oauth_access_token"))
	if characterID <= 0 || token == "" {
		return record, false, nil
	}

	if _, exists := seenCharacterIDs[characterID]; exists {
		return record, false, nil
	}

	seenCharacterIDs[characterID] = struct{}{}
	s.logger.WithFields(logging.Fields{
		"character_id":     characterID,
		"corporation_id":   record.GetInt("eve_corporation_id"),
		"alliance_id":      record.GetInt("eve_alliance_id"),
		"organization":     fmt.Sprintf("%s:%d", organizationType, organizationID),
		"token_expires_at": record.GetDateTime("oauth_access_expires_at").Time().UTC().Format(time.RFC3339),
		"scopes":           strings.TrimSpace(record.GetString("oauth_scopes")),
	}).Debug("jumpbridge candidate character selected")

	return record, true, nil
}

func (s *JumpbridgeService) findFirstAllowedOrganizationCharacter(field string, id int) (*core.Record, error) {
	if strings.TrimSpace(field) == "" || id <= 0 {
		return nil, nil
	}
	tokenNotBefore := time.Now().UTC().Add(1 * time.Minute)
	filter := "esi_token_valid = true"
	filter = queryhelpers.AppendAnd(filter, "oauth_access_token != ''")
	filter = queryhelpers.AppendAnd(filter, "oauth_scopes ~ {:scope_search}")
	filter = queryhelpers.AppendAnd(filter, "oauth_scopes ~ {:scope_read}")
	filter = queryhelpers.AppendAnd(filter, fmt.Sprintf("%s = {:org_id}", field))
	params := dbx.Params{
		"scope_search": scopeSearchStructures,
		"scope_read":   scopeReadStructures,
		"org_id":       id,
	}
	records, err := s.App.FindRecordsByFilter(
		store.CollectionCharacters,
		filter,
		"eve_character_id",
		maxCandidateCharacterScan,
		0,
		params,
	)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if record == nil {
			continue
		}
		expiresAt, parseErr := format.ParseDateTimeFlexibleUTC(record.GetString("oauth_access_expires_at"))
		if parseErr != nil {
			continue
		}
		if expiresAt.After(tokenNotBefore) {
			return record, nil
		}
	}
	return nil, nil
}

func (s *JumpbridgeService) loadAllowedOrganizationIDs() (allianceIDs, corporationIDs []int, err error) {
	alliances, allianceErr := s.App.FindRecordsByFilter(store.CollectionAllowedAlliances, "", "eve_id", 0, 0, nil)
	if allianceErr != nil {
		return nil, nil, allianceErr
	}
	corporations, corporationErr := s.App.FindRecordsByFilter(store.CollectionAllowedCorporations, "", "eve_id", 0, 0, nil)
	if corporationErr != nil {
		return nil, nil, corporationErr
	}
	allianceIDs = make([]int, 0, len(alliances))
	for _, record := range alliances {
		if eveID := record.GetInt("eve_id"); eveID > 0 {
			allianceIDs = append(allianceIDs, eveID)
		}
	}
	corporationIDs = make([]int, 0, len(corporations))
	for _, record := range corporations {
		if eveID := record.GetInt("eve_id"); eveID > 0 {
			corporationIDs = append(corporationIDs, eveID)
		}
	}
	slices.Sort(allianceIDs)
	slices.Sort(corporationIDs)
	return allianceIDs, corporationIDs, nil
}

func (s *JumpbridgeService) loadOwnerAllowedOrganizationIDs() (allianceIDs, corporationIDs []int, err error) {
	baseAllianceIDs, baseCorporationIDs, err := s.loadAllowedOrganizationIDs()
	if err != nil {
		return nil, nil, err
	}
	return s.loadOwnerAllowedOrganizationIDsFromBase(baseAllianceIDs, baseCorporationIDs)
}

func (s *JumpbridgeService) loadOwnerAllowedOrganizationIDsFromBase(baseAllianceIDs, baseCorporationIDs []int) (allianceIDs, corporationIDs []int, err error) {
	allianceSet := toIntSet(baseAllianceIDs)
	corporationSet := toIntSet(baseCorporationIDs)
	friendlyRecords, friendlyErr := s.App.FindRecordsByFilter(
		store.CollectionOrganizationStandings,
		"hostility = {:hostility}",
		"",
		0,
		0,
		dbx.Params{"hostility": "friendly"},
	)
	if friendlyErr != nil {
		return nil, nil, friendlyErr
	}

	for _, record := range friendlyRecords {
		mergeFriendlyStandingOwnerIDs(record, allianceSet, corporationSet)
	}

	allianceIDs = make([]int, 0, len(allianceSet))
	for allianceID := range allianceSet {
		allianceIDs = append(allianceIDs, allianceID)
	}
	corporationIDs = make([]int, 0, len(corporationSet))
	for corporationID := range corporationSet {
		corporationIDs = append(corporationIDs, corporationID)
	}
	slices.Sort(allianceIDs)
	slices.Sort(corporationIDs)
	return allianceIDs, corporationIDs, nil
}

func mergeFriendlyStandingOwnerIDs(record *core.Record, allianceSet, corporationSet map[int]struct{}) {
	if record == nil {
		return
	}
	ownerType := strings.TrimSpace(record.GetString("owner_type"))
	switch ownerType {
	case "alliance":
		if allianceID := record.GetInt("alliance_id"); allianceID > 0 {
			allianceSet[allianceID] = struct{}{}
		}
	case "corporation":
		if corporationID := record.GetInt("corporation_id"); corporationID > 0 {
			corporationSet[corporationID] = struct{}{}
		}
	default:
		if allianceID := record.GetInt("alliance_id"); allianceID > 0 {
			allianceSet[allianceID] = struct{}{}
		}
		if corporationID := record.GetInt("corporation_id"); corporationID > 0 {
			corporationSet[corporationID] = struct{}{}
		}
	}
}

type jumpbridgePair struct {
	from            int
	to              int
	fromStructureID int64
	toStructureID   int64
}

func (s *JumpbridgeService) loadUniquePairs() ([]jumpbridgePair, error) {
	records, err := s.App.FindRecordsByFilter(store.CollectionJumpbridges, "", "", 0, 0, nil)
	if err != nil {
		return nil, err
	}
	pairs := map[string]jumpbridgePair{}
	for _, record := range records {
		merged, ok := mergeJumpbridgePairRecord(record, pairs)
		if !ok {
			continue
		}
		pairs[merged.key] = merged.pair
	}
	out := make([]jumpbridgePair, 0, len(pairs))
	for _, pair := range pairs {
		out = append(out, pair)
	}
	return out, nil
}

type mergedJumpbridgePair struct {
	key  string
	pair jumpbridgePair
}

func mergeJumpbridgePairRecord(record *core.Record, pairs map[string]jumpbridgePair) (mergedJumpbridgePair, bool) {
	from := record.GetInt("from_solarsystem")
	to := record.GetInt("to_solarsystem")
	if from <= 0 || to <= 0 || from == to {
		return mergedJumpbridgePair{}, false
	}
	fromStructureID := int64(record.GetInt("from_structure_id"))
	toStructureID := int64(record.GetInt("to_structure_id"))
	canonicalFrom := min(from, to)
	canonicalTo := max(from, to)
	key := pairKey(fmt.Sprintf("%d", canonicalFrom), fmt.Sprintf("%d", canonicalTo))
	pair := pairs[key]
	if pair.from == 0 {
		pair.from = canonicalFrom
		pair.to = canonicalTo
	}
	mergeOrientedStructureIDs(&pair, from == canonicalFrom, fromStructureID, toStructureID)
	return mergedJumpbridgePair{key: key, pair: pair}, true
}

func mergeOrientedStructureIDs(pair *jumpbridgePair, sameDirection bool, fromStructureID, toStructureID int64) {
	if sameDirection {
		if pair.fromStructureID <= 0 && fromStructureID > 0 {
			pair.fromStructureID = fromStructureID
		}
		if pair.toStructureID <= 0 && toStructureID > 0 {
			pair.toStructureID = toStructureID
		}
		return
	}

	if pair.fromStructureID <= 0 && toStructureID > 0 {
		pair.fromStructureID = toStructureID
	}

	if pair.toStructureID <= 0 && fromStructureID > 0 {
		pair.toStructureID = fromStructureID
	}
}

func (s *JumpbridgeService) backfillPairStructureIDs(fromSystemID, toSystemID int, fromStructureID, toStructureID int64) error {
	if fromSystemID <= 0 || toSystemID <= 0 {
		return nil
	}
	filter := "(from_solarsystem = {:from} && to_solarsystem = {:to}) || (from_solarsystem = {:to} && to_solarsystem = {:from})"
	records, err := s.App.FindRecordsByFilter(
		store.CollectionJumpbridges,
		filter,
		"",
		0,
		0,
		map[string]any{"from": fromSystemID, "to": toSystemID},
	)
	if err != nil {
		return err
	}
	for _, record := range records {
		if !setBackfillStructureIDs(record, fromSystemID, toSystemID, fromStructureID, toStructureID) {
			continue
		}
		if saveErr := s.App.Save(record); saveErr != nil {
			return saveErr
		}
	}
	return nil
}

func setBackfillStructureIDs(
	record *core.Record,
	fromSystemID int,
	toSystemID int,
	fromStructureID int64,
	toStructureID int64,
) bool {
	rowFrom := record.GetInt("from_solarsystem")
	rowTo := record.GetInt("to_solarsystem")
	wantedFrom := fromStructureID
	wantedTo := toStructureID
	if rowFrom == toSystemID && rowTo == fromSystemID {
		wantedFrom, wantedTo = toStructureID, fromStructureID
	}
	changed := false
	if wantedFrom > 0 && record.GetInt("from_structure_id") <= 0 {
		record.Set("from_structure_id", wantedFrom)
		changed = true
	}

	if wantedTo > 0 && record.GetInt("to_structure_id") <= 0 {
		record.Set("to_structure_id", wantedTo)
		changed = true
	}
	return changed
}

func formatPairKey(fromSystemID, toSystemID int) string {
	if fromSystemID > toSystemID {
		fromSystemID, toSystemID = toSystemID, fromSystemID
	}
	return fmt.Sprintf("%d-%d", fromSystemID, toSystemID)
}

func formatPairName(fromSystem, toSystem *core.Record) string {
	fromLabel := formatSystemLabel(fromSystem)
	toLabel := formatSystemLabel(toSystem)
	if fromLabel > toLabel {
		fromLabel, toLabel = toLabel, fromLabel
	}
	return fmt.Sprintf("%s <-> %s", fromLabel, toLabel)
}

func formatSystemLabel(system *core.Record) string {
	if system == nil {
		return "unknown(0)"
	}
	name := strings.TrimSpace(system.GetString("name"))
	eveID := system.GetInt("eve_id")
	if name == "" {
		return fmt.Sprintf("unknown(%d)", eveID)
	}
	return fmt.Sprintf("%s(%d)", name, eveID)
}
