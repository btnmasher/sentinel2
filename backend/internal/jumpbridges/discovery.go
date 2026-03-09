package jumpbridges

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/esi"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

const (
	minDirectionalFields = 3
	expectedPairParts    = 2
)

type DiscoveryImportSummary struct {
	CharacterIDs              []int
	SkippedOrganizations      int
	AllowedSovereigntySystems int
	StructureCandidates       int
	CandidatePairs            int
	AddedPairs                int
	UpgradedPairs             int
	SkippedPairs              int
	AddedKeys                 []string
	AddedNames                []string
	UpgradedKeys              []string
	UpgradedNames             []string
}

type discoveredPair struct {
	fromSystemID     int
	toSystemID       int
	fromStructureID  int64
	toStructureID    int64
	fromOwnerAllowed bool
	toOwnerAllowed   bool
}

type globalSearchAggregate struct {
	query          string
	totalRaw       int
	uniqueCount    int
	queried        bool
	firstErr       error
	perStructure   map[int64][]structureCharacterCandidate
	perCharSeen    map[int64]map[int]struct{}
	characterCount int
}

type globalCandidateFilterStats struct {
	filteredNonAnsiblex      int
	filteredDestinationParse int
	filteredOwnerNotAllowed  int
	acceptedProposalUpdates  int
	filteredPairsByOwner     int
}

type systemDiscoveryAggregate struct {
	totalRaw       int
	queried        bool
	firstErr       error
	seenStructures map[int64]struct{}
}

func (s *JumpbridgeService) DiscoverAndImportAllowedSovPairs(ctx context.Context) (DiscoveryImportSummary, error) {
	if s.PublicESI == nil {
		return DiscoveryImportSummary{}, fmt.Errorf("esi public client unavailable for jumpbridge discovery")
	}
	candidates, skippedOrganizations, err := s.pickAllowedStructureCharactersDetailed()
	if err != nil {
		return DiscoveryImportSummary{}, err
	}
	characterIDs := make([]int, 0, len(candidates))
	for _, candidate := range candidates {
		characterIDs = append(characterIDs, candidate.CharacterID)
	}

	allowedAllianceIDs, allowedCorporationIDs, err := s.loadAllowedOrganizationIDs()
	if err != nil {
		return DiscoveryImportSummary{}, err
	}
	ownerAllowedAllianceIDs, ownerAllowedCorporationIDs, ownerAllowedErr := s.loadOwnerAllowedOrganizationIDsFromBase(allowedAllianceIDs, allowedCorporationIDs)
	if ownerAllowedErr != nil {
		return DiscoveryImportSummary{}, ownerAllowedErr
	}
	allowedAllianceSet := toIntSet(ownerAllowedAllianceIDs)
	allowedCorporationSet := toIntSet(ownerAllowedCorporationIDs)
	allowedSystems, err := s.loadAllowedSovereigntySystems(ctx, allowedAllianceIDs)
	if err != nil {
		return DiscoveryImportSummary{}, err
	}
	systemByID, systemNameToID, err := s.loadSolarSystemIndexes()
	if err != nil {
		return DiscoveryImportSummary{}, err
	}
	validation := structureValidationContext{
		characters:         candidates,
		systemStructureIDs: map[int]int64{},
	}
	proposals, candidateCount, proposalErr := s.discoverPairProposalsAcrossAllowedSovSystems(
		ctx,
		&validation,
		allowedSystems,
		systemNameToID,
		allowedAllianceSet,
		allowedCorporationSet,
	)
	if proposalErr != nil {
		fallbackProposals, fallbackCandidateCount := s.discoverPairProposalsBySystemNames(
			ctx,
			&validation,
			allowedSystems,
			systemByID,
			systemNameToID,
			allowedAllianceSet,
			allowedCorporationSet,
		)
		s.logger.WithFields(logging.Fields{
			"error":                    proposalErr.Error(),
			"fallback_candidate_count": fallbackCandidateCount,
			"fallback_pair_count":      len(fallbackProposals),
		}).Warn("jumpbridge discovery global query failed; used per-system fallback")
		proposals = fallbackProposals
		candidateCount = fallbackCandidateCount
	}

	existingBySystem, existingPairs, err := s.loadExistingPairState()
	if err != nil {
		return DiscoveryImportSummary{}, err
	}
	existingPairHasStructureIDs, err := s.loadExistingPairStructureCompleteness()
	if err != nil {
		return DiscoveryImportSummary{}, err
	}
	coll, err := s.App.FindCollectionByNameOrId(store.CollectionJumpbridges)
	if err != nil {
		return DiscoveryImportSummary{}, err
	}
	keys := make([]string, 0, len(proposals))
	for key := range proposals {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	summary := DiscoveryImportSummary{
		CharacterIDs:              characterIDs,
		SkippedOrganizations:      skippedOrganizations,
		AllowedSovereigntySystems: len(allowedSystems),
		StructureCandidates:       candidateCount,
		CandidatePairs:            len(keys),
		AddedKeys:                 []string{},
		AddedNames:                []string{},
		UpgradedKeys:              []string{},
		UpgradedNames:             []string{},
	}
	s.logger.WithFields(logging.Fields{
		"character_ids":               summary.CharacterIDs,
		"character_count":             len(summary.CharacterIDs),
		"skipped_organizations":       summary.SkippedOrganizations,
		"allowed_sovereignty_systems": summary.AllowedSovereigntySystems,
		"structure_candidates":        summary.StructureCandidates,
		"candidate_pairs":             summary.CandidatePairs,
	}).Debug("jumpbridge discovery started")
	for _, key := range keys {
		s.processDiscoveredPair(
			ctx,
			&validation,
			coll,
			key,
			proposals[key],
			systemByID,
			existingBySystem,
			existingPairs,
			existingPairHasStructureIDs,
			&summary,
		)
	}
	s.logger.WithFields(logging.Fields{
		"character_ids":               summary.CharacterIDs,
		"skipped_organizations":       summary.SkippedOrganizations,
		"allowed_sovereignty_systems": summary.AllowedSovereigntySystems,
		"structure_candidates":        summary.StructureCandidates,
		"candidate_pairs":             summary.CandidatePairs,
		"added_pairs":                 summary.AddedPairs,
		"upgraded_pairs":              summary.UpgradedPairs,
		"skipped_pairs":               summary.SkippedPairs,
		"added_keys":                  summary.AddedKeys,
		"added_names":                 summary.AddedNames,
		"upgraded_keys":               summary.UpgradedKeys,
		"upgraded_names":              summary.UpgradedNames,
	}).Info("jumpbridge discovery completed")
	return summary, nil
}

func (s *JumpbridgeService) processDiscoveredPair(
	ctx context.Context,
	validation *structureValidationContext,
	coll *core.Collection,
	key string,
	pair discoveredPair,
	systemByID map[int]*core.Record,
	existingBySystem map[int]int,
	existingPairs map[string]struct{},
	existingPairHasStructureIDs map[string]bool,
	summary *DiscoveryImportSummary,
) {
	fromSystem := systemByID[pair.fromSystemID]
	toSystem := systemByID[pair.toSystemID]
	if fromSystem == nil || toSystem == nil {
		s.logger.WithFields(logging.Fields{
			"pair_key":       key,
			"from_system_id": pair.fromSystemID,
			"to_system_id":   pair.toSystemID,
		}).Debug("jumpbridge discovery pair skipped: missing system record")
		summary.SkippedPairs++
		return
	}

	existingPair, existingPairComplete := checkExistingPairStatus(existingPairs, existingPairHasStructureIDs, key)
	if existingPairComplete {
		s.logger.WithFields(logging.Fields{
			"pair_key":       key,
			"from_system_id": pair.fromSystemID,
			"to_system_id":   pair.toSystemID,
		}).Debug("jumpbridge discovery pair skipped: already exists with structure ids")
		summary.SkippedPairs++
		return
	}

	if pairedElsewhere(pair, existingBySystem) {
		s.logPairEndpointConflict(key, pair, existingBySystem)
		summary.SkippedPairs++
		return
	}

	if !withinMaxDistance(fromSystem, toSystem) {
		s.logger.WithFields(logging.Fields{
			"pair_key":         key,
			"from_system_id":   pair.fromSystemID,
			"to_system_id":     pair.toSystemID,
			"from_system_name": strings.TrimSpace(fromSystem.GetString("name")),
			"to_system_name":   strings.TrimSpace(toSystem.GetString("name")),
			"max_distance_ly":  maxJumpbridgeDistanceLY,
		}).Debug("jumpbridge discovery pair skipped: exceeds max distance")
		summary.SkippedPairs++
		return
	}

	fromStructureID, toStructureID, validateErr := s.resolvePairStructureIDsWithContext(
		ctx,
		validation,
		fromSystem,
		toSystem,
		pair.fromStructureID,
		pair.toStructureID,
	)
	if validateErr != nil {
		s.logger.WithFields(logging.Fields{
			"pair_key":       key,
			"from_system_id": pair.fromSystemID,
			"to_system_id":   pair.toSystemID,
			"error":          validateErr.Error(),
		}).Debug("jumpbridge discovery pair skipped: could not resolve/validate structure ids")
		summary.SkippedPairs++
		return
	}

	if existingPair && !existingPairComplete {
		if s.upgradeDiscoveredPair(key, pair, fromSystem, toSystem, fromStructureID, toStructureID, existingPairHasStructureIDs, summary) {
			return
		}
		summary.SkippedPairs++
		return
	}

	if s.addDiscoveredPair(coll, key, pair, fromSystem, toSystem, fromStructureID, toStructureID, existingBySystem, existingPairs, summary) {
		return
	}
	summary.SkippedPairs++
}

func checkExistingPairStatus(existingPairs map[string]struct{}, existingPairHasStructureIDs map[string]bool, key string) (exists, complete bool) {
	if _, ok := existingPairs[key]; !ok {
		return false, false
	}
	return true, existingPairHasStructureIDs[key]
}

func pairedElsewhere(pair discoveredPair, existingBySystem map[int]int) bool {
	if partner, ok := existingBySystem[pair.fromSystemID]; ok && partner != pair.toSystemID {
		return true
	}

	if partner, ok := existingBySystem[pair.toSystemID]; ok && partner != pair.fromSystemID {
		return true
	}
	return false
}

func (s *JumpbridgeService) logPairEndpointConflict(key string, pair discoveredPair, existingBySystem map[int]int) {
	if partner, ok := existingBySystem[pair.fromSystemID]; ok && partner != pair.toSystemID {
		s.logger.WithFields(logging.Fields{
			"pair_key":             key,
			"from_system_id":       pair.fromSystemID,
			"candidate_partner_id": pair.toSystemID,
			"existing_partner_id":  partner,
		}).Debug("jumpbridge discovery pair skipped: from endpoint already paired elsewhere")
		return
	}

	if partner, ok := existingBySystem[pair.toSystemID]; ok && partner != pair.fromSystemID {
		s.logger.WithFields(logging.Fields{
			"pair_key":             key,
			"to_system_id":         pair.toSystemID,
			"candidate_partner_id": pair.fromSystemID,
			"existing_partner_id":  partner,
		}).Debug("jumpbridge discovery pair skipped: to endpoint already paired elsewhere")
	}
}

func (s *JumpbridgeService) upgradeDiscoveredPair(
	key string,
	pair discoveredPair,
	fromSystem *core.Record,
	toSystem *core.Record,
	fromStructureID int64,
	toStructureID int64,
	existingPairHasStructureIDs map[string]bool,
	summary *DiscoveryImportSummary,
) bool {
	if backfillErr := s.backfillPairStructureIDs(pair.fromSystemID, pair.toSystemID, fromStructureID, toStructureID); backfillErr != nil {
		return false
	}
	summary.UpgradedPairs++
	summary.UpgradedKeys = append(summary.UpgradedKeys, key)
	summary.UpgradedNames = append(summary.UpgradedNames, formatPairName(fromSystem, toSystem))
	s.logger.WithFields(logging.Fields{
		"pair_key":          key,
		"from_system_id":    pair.fromSystemID,
		"to_system_id":      pair.toSystemID,
		"from_structure_id": fromStructureID,
		"to_structure_id":   toStructureID,
	}).Info("jumpbridge discovery upgraded existing pair with structure ids")
	existingPairHasStructureIDs[key] = true
	return true
}

func (s *JumpbridgeService) addDiscoveredPair(
	coll *core.Collection,
	key string,
	pair discoveredPair,
	fromSystem *core.Record,
	toSystem *core.Record,
	fromStructureID int64,
	toStructureID int64,
	existingBySystem map[int]int,
	existingPairs map[string]struct{},
	summary *DiscoveryImportSummary,
) bool {
	saved, saveErr := s.savePair(coll, fromStructureID, toStructureID, fromSystem, toSystem)
	if saveErr != nil || saved != pairDirectionCount {
		s.logger.WithFields(logging.Fields{
			"pair_key":          key,
			"from_system_id":    pair.fromSystemID,
			"to_system_id":      pair.toSystemID,
			"from_structure_id": fromStructureID,
			"to_structure_id":   toStructureID,
			"saved_count":       saved,
			"error":             fmt.Sprintf("%v", saveErr),
		}).Warn("jumpbridge discovery failed to save pair")
		return false
	}
	summary.AddedPairs++
	summary.AddedKeys = append(summary.AddedKeys, key)
	summary.AddedNames = append(summary.AddedNames, formatPairName(fromSystem, toSystem))
	s.logger.WithFields(logging.Fields{
		"pair_key":          key,
		"from_system_id":    pair.fromSystemID,
		"to_system_id":      pair.toSystemID,
		"from_structure_id": fromStructureID,
		"to_structure_id":   toStructureID,
	}).Debug("jumpbridge discovery added pair")
	existingPairs[key] = struct{}{}
	existingBySystem[pair.fromSystemID] = pair.toSystemID
	existingBySystem[pair.toSystemID] = pair.fromSystemID
	return true
}

func (s *JumpbridgeService) discoverPairProposalsAcrossAllowedSovSystems(
	ctx context.Context,
	validation *structureValidationContext,
	allowedSystems map[int]struct{},
	systemNameToID map[string]int,
	allowedAllianceSet map[int]struct{},
	allowedCorporationSet map[int]struct{},
) (proposals map[string]discoveredPair, candidateCount int, err error) {
	if validation == nil {
		return map[string]discoveredPair{}, 0, nil
	}
	aggregate := s.collectGlobalStructureCandidates(ctx, validation)
	if !aggregate.queried && aggregate.firstErr != nil {
		return nil, aggregate.uniqueCount, aggregate.firstErr
	}
	s.logger.WithFields(logging.Fields{
		"query":                    aggregate.query,
		"character_count":          aggregate.characterCount,
		"raw_result_count_total":   aggregate.totalRaw,
		"unique_structure_count":   aggregate.uniqueCount,
		"allowed_sovereignty_size": len(allowedSystems),
	}).Info("jumpbridge discovery global search aggregate")
	proposals, stats := s.buildGlobalProposals(
		ctx,
		aggregate.perStructure,
		systemNameToID,
		allowedAllianceSet,
		allowedCorporationSet,
	)
	s.logger.WithFields(logging.Fields{
		"query":                      aggregate.query,
		"raw_result_count_total":     aggregate.totalRaw,
		"unique_structure_count":     aggregate.uniqueCount,
		"filtered_non_ansiblex":      stats.filteredNonAnsiblex,
		"filtered_bad_destination":   stats.filteredDestinationParse,
		"filtered_owner_not_allowed": stats.filteredOwnerNotAllowed,
		"filtered_pairs_owner_gate":  stats.filteredPairsByOwner,
		"accepted_updates":           stats.acceptedProposalUpdates,
		"candidate_pairs":            len(proposals),
	}).Info("jumpbridge discovery global candidate filtering summary")
	return proposals, aggregate.uniqueCount, nil
}

func (s *JumpbridgeService) collectGlobalStructureCandidates(
	ctx context.Context,
	validation *structureValidationContext,
) globalSearchAggregate {
	aggregate := globalSearchAggregate{
		query:          " » ",
		characterCount: len(validation.characters),
		perStructure:   map[int64][]structureCharacterCandidate{},
		perCharSeen:    map[int64]map[int]struct{}{},
	}
	for _, character := range validation.characters {
		s.collectGlobalStructureCandidatesForCharacter(ctx, &aggregate, character)
	}
	aggregate.uniqueCount = len(aggregate.perStructure)
	return aggregate
}

func (s *JumpbridgeService) collectGlobalStructureCandidatesForCharacter(
	ctx context.Context,
	aggregate *globalSearchAggregate,
	character structureCharacterCandidate,
) {
	s.logger.WithFields(logging.Fields{
		"character_id": character.CharacterID,
		"query":        aggregate.query,
	}).Debug("jumpbridge discovery global search attempt")
	structureIDs, searchErr := s.ESI.SearchStructures(ctx, character.CharacterID, character.Token, aggregate.query, false)
	if searchErr != nil {
		s.logger.WithFields(logging.Fields{
			"character_id": character.CharacterID,
			"query":        aggregate.query,
			"error":        searchErr.Error(),
		}).Warn("jumpbridge discovery global search failed")
		if aggregate.firstErr == nil {
			aggregate.firstErr = searchErr
		}
		return
	}
	aggregate.queried = true
	aggregate.totalRaw += len(structureIDs)
	s.logger.WithFields(logging.Fields{
		"character_id":     character.CharacterID,
		"query":            aggregate.query,
		"raw_result_count": len(structureIDs),
	}).Info("jumpbridge discovery global search returned candidates")
	for _, structureID := range structureIDs {
		s.registerStructureCandidate(aggregate, structureID, character)
	}
}

func (s *JumpbridgeService) registerStructureCandidate(
	aggregate *globalSearchAggregate,
	structureID int64,
	character structureCharacterCandidate,
) {
	if _, exists := aggregate.perStructure[structureID]; !exists {
		aggregate.perStructure[structureID] = []structureCharacterCandidate{}
	}
	seenByCharacter := aggregate.perCharSeen[structureID]
	if seenByCharacter == nil {
		seenByCharacter = map[int]struct{}{}
		aggregate.perCharSeen[structureID] = seenByCharacter
	}

	if _, exists := seenByCharacter[character.CharacterID]; exists {
		return
	}
	seenByCharacter[character.CharacterID] = struct{}{}
	aggregate.perStructure[structureID] = append(aggregate.perStructure[structureID], character)
}

func (s *JumpbridgeService) buildGlobalProposals(
	ctx context.Context,
	structureCandidates map[int64][]structureCharacterCandidate,
	systemNameToID map[string]int,
	allowedAllianceSet map[int]struct{},
	allowedCorporationSet map[int]struct{},
) (map[string]discoveredPair, globalCandidateFilterStats) {
	proposals := map[string]discoveredPair{}
	stats := globalCandidateFilterStats{}
	structureIDs := sortedStructureIDs(structureCandidates)
	corporationAllianceCache := map[int]int{}
	for _, structureID := range structureIDs {
		s.applyGlobalStructureCandidate(
			ctx,
			structureID,
			structureCandidates[structureID],
			systemNameToID,
			allowedAllianceSet,
			allowedCorporationSet,
			corporationAllianceCache,
			proposals,
			&stats,
		)
	}
	filtered := make(map[string]discoveredPair, len(proposals))
	for key, pair := range proposals {
		if pair.fromOwnerAllowed || pair.toOwnerAllowed {
			filtered[key] = pair
			continue
		}
		stats.filteredPairsByOwner++
	}
	return filtered, stats
}

func sortedStructureIDs(perStructure map[int64][]structureCharacterCandidate) []int64 {
	ids := make([]int64, 0, len(perStructure))
	for structureID := range perStructure {
		ids = append(ids, structureID)
	}
	slices.Sort(ids)
	return ids
}

func (s *JumpbridgeService) applyGlobalStructureCandidate(
	ctx context.Context,
	structureID int64,
	candidates []structureCharacterCandidate,
	systemNameToID map[string]int,
	allowedAllianceSet map[int]struct{},
	allowedCorporationSet map[int]struct{},
	corporationAllianceCache map[int]int,
	proposals map[string]discoveredPair,
	stats *globalCandidateFilterStats,
) {
	structure, resolvedBy, fetchErr := s.fetchStructureAcrossCandidates(ctx, structureID, candidates)
	if fetchErr != nil {
		s.logger.WithFields(logging.Fields{
			"structure_id":       structureID,
			"candidate_char_ids": candidateCharacterIDs(candidates),
			"error":              fetchErr.Error(),
		}).Debug("jumpbridge discovery global candidate fetch failed across all candidate characters")
		return
	}

	if structure.TypeID != ansiblexTypeID {
		stats.filteredNonAnsiblex++
		s.logger.WithFields(logging.Fields{
			"structure_id": structureID,
			"type_id":      structure.TypeID,
			"resolved_by":  resolvedBy,
		}).Debug("jumpbridge discovery global candidate filtered: non-ansiblex")
		return
	}
	sourceSystemID := structure.SystemID
	destinationID := parseAnsiblexDestinationSystemID(structure.Name, sourceSystemID, systemNameToID)
	if destinationID <= 0 || destinationID == sourceSystemID {
		stats.filteredDestinationParse++
		s.logger.WithFields(logging.Fields{
			"structure_id":   structureID,
			"resolved_by":    resolvedBy,
			"source_system":  sourceSystemID,
			"structure_name": structure.Name,
		}).Debug("jumpbridge discovery global candidate destination parse failed")
		return
	}
	ownerAllowed, ownerReason := s.isStructureOwnerAllowed(
		ctx,
		structure.OwnerID,
		allowedAllianceSet,
		allowedCorporationSet,
		corporationAllianceCache,
	)
	if !ownerAllowed {
		stats.filteredOwnerNotAllowed++
		s.logger.WithFields(logging.Fields{
			"structure_id":   structureID,
			"resolved_by":    resolvedBy,
			"source_system":  sourceSystemID,
			"destination_id": destinationID,
			"owner_id":       structure.OwnerID,
			"reason":         ownerReason,
		}).Debug("jumpbridge discovery global candidate filtered: structure owner not allowed")
		return
	}
	s.logger.WithFields(logging.Fields{
		"structure_id":   structureID,
		"resolved_by":    resolvedBy,
		"source_system":  sourceSystemID,
		"destination_id": destinationID,
		"owner_id":       structure.OwnerID,
	}).Debug("jumpbridge discovery global candidate resolved")
	key := pairKey(strconv.Itoa(sourceSystemID), strconv.Itoa(destinationID))
	pair := mergeStructureIntoPair(proposals[key], sourceSystemID, destinationID, structureID)
	proposals[key] = pair
	stats.acceptedProposalUpdates++
}

func mergeStructureIntoPair(pair discoveredPair, sourceSystemID, destinationID int, structureID int64) discoveredPair {
	if pair.fromSystemID == 0 || pair.toSystemID == 0 {
		pair = discoveredPair{fromSystemID: sourceSystemID, toSystemID: destinationID}
	}
	switch sourceSystemID {
	case pair.fromSystemID:
		pair.fromStructureID = structureID
		pair.fromOwnerAllowed = true
	case pair.toSystemID:
		pair.toStructureID = structureID
		pair.toOwnerAllowed = true
	}
	return pair
}

func (s *JumpbridgeService) discoverPairProposalsBySystemNames(
	ctx context.Context,
	validation *structureValidationContext,
	allowedSystems map[int]struct{},
	systemByID map[int]*core.Record,
	systemNameToID map[string]int,
	allowedAllianceSet map[int]struct{},
	allowedCorporationSet map[int]struct{},
) (proposals map[string]discoveredPair, seenStructureCount int) {
	proposals = map[string]discoveredPair{}
	seenStructures := map[int64]struct{}{}
	for systemID := range allowedSystems {
		s.collectSystemFallbackProposals(
			ctx,
			systemID,
			systemByID[systemID],
			validation,
			systemNameToID,
			allowedAllianceSet,
			allowedCorporationSet,
			proposals,
			seenStructures,
		)
	}
	filtered := filterOwnerAllowedPairs(proposals)
	return filtered, len(seenStructures)
}

func (s *JumpbridgeService) collectSystemFallbackProposals(
	ctx context.Context,
	systemID int,
	systemRecord *core.Record,
	validation *structureValidationContext,
	systemNameToID map[string]int,
	allowedAllianceSet map[int]struct{},
	allowedCorporationSet map[int]struct{},
	proposals map[string]discoveredPair,
	seenStructures map[int64]struct{},
) {
	if systemRecord == nil {
		return
	}
	endpoints, discoverErr := s.discoverPairEndpointsFromSystem(
		ctx,
		validation,
		systemRecord,
		systemNameToID,
		allowedAllianceSet,
		allowedCorporationSet,
	)
	if discoverErr != nil {
		return
	}
	for destinationID, endpoint := range endpoints {
		if destinationID <= 0 || destinationID == systemID {
			continue
		}
		seenStructures[endpoint.structureID] = struct{}{}
		key := pairKey(strconv.Itoa(systemID), strconv.Itoa(destinationID))
		pair := mergeStructureIntoPair(proposals[key], systemID, destinationID, endpoint.structureID)
		if systemID == pair.toSystemID {
			pair.toOwnerAllowed = endpoint.ownerAllowed
		} else {
			pair.fromOwnerAllowed = endpoint.ownerAllowed
		}
		proposals[key] = pair
	}
}

func filterOwnerAllowedPairs(proposals map[string]discoveredPair) map[string]discoveredPair {
	filtered := make(map[string]discoveredPair, len(proposals))
	for key, pair := range proposals {
		if pair.fromOwnerAllowed || pair.toOwnerAllowed {
			filtered[key] = pair
		}
	}
	return filtered
}

type discoveredEndpoint struct {
	structureID  int64
	ownerAllowed bool
}

func (s *JumpbridgeService) fetchStructureAcrossCandidates(
	ctx context.Context,
	structureID int64,
	candidates []structureCharacterCandidate,
) (esi.UniverseStructure, int, error) {
	var firstErr error
	for _, character := range candidates {
		structure, fetchErr := s.ESI.UniverseStructure(ctx, character.CharacterID, character.Token, structureID)
		if fetchErr != nil {
			if firstErr == nil {
				firstErr = fetchErr
			}
			s.logger.WithFields(logging.Fields{
				"character_id": character.CharacterID,
				"structure_id": structureID,
				"error":        fetchErr.Error(),
			}).Debug("jumpbridge discovery candidate structure fetch failed for character")
			continue
		}
		return structure, character.CharacterID, nil
	}

	if firstErr == nil {
		firstErr = fmt.Errorf("no candidate characters available for structure %d", structureID)
	}
	return esi.UniverseStructure{}, 0, firstErr
}

func candidateCharacterIDs(candidates []structureCharacterCandidate) []int {
	out := make([]int, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, candidate.CharacterID)
	}
	return out
}

func (s *JumpbridgeService) loadExistingPairStructureCompleteness() (map[string]bool, error) {
	records, err := s.App.FindRecordsByFilter(store.CollectionJumpbridges, "", "", 0, 0, nil)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, record := range records {
		from := record.GetInt("from_solarsystem")
		to := record.GetInt("to_solarsystem")
		if from <= 0 || to <= 0 || from == to {
			continue
		}
		key := pairKey(strconv.Itoa(from), strconv.Itoa(to))
		fromStructureID := int64(record.GetInt("from_structure_id"))
		toStructureID := int64(record.GetInt("to_structure_id"))
		if fromStructureID > 0 && toStructureID > 0 {
			out[key] = true
			continue
		}
		if _, exists := out[key]; !exists {
			out[key] = false
		}
	}
	return out, nil
}

func (s *JumpbridgeService) loadAllowedSovereigntySystems(ctx context.Context, allowedAllianceIDs []int) (map[int]struct{}, error) {
	allowedAlliances := make(map[int]struct{}, len(allowedAllianceIDs))
	for _, allianceID := range allowedAllianceIDs {
		allowedAlliances[allianceID] = struct{}{}
	}
	rows, err := s.PublicESI.SovereigntyMap(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int]struct{}{}
	for _, row := range rows {
		if row.SystemID <= 0 {
			continue
		}
		if _, ok := allowedAlliances[row.AllianceID]; ok {
			out[row.SystemID] = struct{}{}
		}
	}
	return out, nil
}

func (s *JumpbridgeService) loadSolarSystemIndexes() (byID map[int]*core.Record, byName map[string]int, err error) {
	records, err := s.App.FindRecordsByFilter(store.CollectionSolarSystems, "", "", 0, 0, nil)
	if err != nil {
		return nil, nil, err
	}
	byID = make(map[int]*core.Record, len(records))
	byName = make(map[string]int, len(records))
	for _, record := range records {
		systemID := record.GetInt("eve_id")
		name := strings.TrimSpace(record.GetString("name"))
		if systemID <= 0 || name == "" {
			continue
		}
		byID[systemID] = record
		byName[strings.ToLower(name)] = systemID
	}
	return byID, byName, nil
}

func (s *JumpbridgeService) discoverPairEndpointsFromSystem(
	ctx context.Context,
	validation *structureValidationContext,
	systemRecord *core.Record,
	systemNameToID map[string]int,
	allowedAllianceSet map[int]struct{},
	allowedCorporationSet map[int]struct{},
) (map[int]discoveredEndpoint, error) {
	if validation == nil || systemRecord == nil {
		return map[int]discoveredEndpoint{}, nil
	}
	systemID := systemRecord.GetInt("eve_id")
	systemName := strings.TrimSpace(systemRecord.GetString("name"))
	if systemID <= 0 || systemName == "" {
		return map[int]discoveredEndpoint{}, nil
	}
	endpoints := map[int]discoveredEndpoint{}
	aggregate := systemDiscoveryAggregate{seenStructures: map[int64]struct{}{}}
	corporationAllianceCache := map[int]int{}
	for _, character := range validation.characters {
		s.discoverEndpointsForCharacter(
			ctx,
			character,
			systemID,
			systemName,
			systemNameToID,
			allowedAllianceSet,
			allowedCorporationSet,
			corporationAllianceCache,
			endpoints,
			&aggregate,
		)
	}

	if !aggregate.queried && aggregate.firstErr != nil {
		return nil, aggregate.firstErr
	}
	s.logger.WithFields(logging.Fields{
		"system_id":              systemID,
		"system_name":            systemName,
		"character_count":        len(validation.characters),
		"raw_result_count_total": aggregate.totalRaw,
		"unique_structure_count": len(aggregate.seenStructures),
		"endpoint_count":         len(endpoints),
	}).Info("jumpbridge discovery system search aggregate")
	return endpoints, nil
}

func (s *JumpbridgeService) discoverEndpointsForCharacter(
	ctx context.Context,
	character structureCharacterCandidate,
	systemID int,
	systemName string,
	systemNameToID map[string]int,
	allowedAllianceSet map[int]struct{},
	allowedCorporationSet map[int]struct{},
	corporationAllianceCache map[int]int,
	endpoints map[int]discoveredEndpoint,
	aggregate *systemDiscoveryAggregate,
) {
	s.logger.WithFields(logging.Fields{
		"character_id": character.CharacterID,
		"system_id":    systemID,
		"system_name":  systemName,
	}).Debug("jumpbridge discovery search attempt")
	structureIDs, err := s.ESI.SearchStructures(ctx, character.CharacterID, character.Token, systemName, false)
	if err != nil {
		s.logger.WithFields(logging.Fields{
			"character_id": character.CharacterID,
			"system_id":    systemID,
			"system_name":  systemName,
			"error":        err.Error(),
		}).Warn("jumpbridge discovery search failed")
		if aggregate.firstErr == nil {
			aggregate.firstErr = err
		}
		return
	}
	aggregate.queried = true
	aggregate.totalRaw += len(structureIDs)
	s.logger.WithFields(logging.Fields{
		"character_id":     character.CharacterID,
		"system_id":        systemID,
		"system_name":      systemName,
		"raw_result_count": len(structureIDs),
	}).Info("jumpbridge discovery search returned candidates")
	for _, structureID := range structureIDs {
		if _, exists := aggregate.seenStructures[structureID]; exists {
			continue
		}
		aggregate.seenStructures[structureID] = struct{}{}
		s.addEndpointFromStructure(
			ctx,
			character,
			systemID,
			structureID,
			systemNameToID,
			allowedAllianceSet,
			allowedCorporationSet,
			corporationAllianceCache,
			endpoints,
		)
	}
}

func (s *JumpbridgeService) addEndpointFromStructure(
	ctx context.Context,
	character structureCharacterCandidate,
	systemID int,
	structureID int64,
	systemNameToID map[string]int,
	allowedAllianceSet map[int]struct{},
	allowedCorporationSet map[int]struct{},
	corporationAllianceCache map[int]int,
	endpoints map[int]discoveredEndpoint,
) {
	structure, err := s.ESI.UniverseStructure(ctx, character.CharacterID, character.Token, structureID)
	if err != nil {
		s.logger.WithFields(logging.Fields{
			"character_id": character.CharacterID,
			"system_id":    systemID,
			"structure_id": structureID,
			"error":        err.Error(),
		}).Debug("jumpbridge discovery structure fetch failed")
		return
	}

	if structure.TypeID != ansiblexTypeID || structure.SystemID != systemID {
		return
	}
	toSystemID := parseAnsiblexDestinationSystemID(structure.Name, systemID, systemNameToID)
	if toSystemID <= 0 || toSystemID == systemID {
		s.logger.WithFields(logging.Fields{
			"character_id":   character.CharacterID,
			"system_id":      systemID,
			"structure_id":   structureID,
			"structure_name": structure.Name,
		}).Debug("jumpbridge discovery could not parse destination system from structure name")
		return
	}
	ownerAllowed, ownerReason := s.isStructureOwnerAllowed(
		ctx,
		structure.OwnerID,
		allowedAllianceSet,
		allowedCorporationSet,
		corporationAllianceCache,
	)
	if !ownerAllowed {
		s.logger.WithFields(logging.Fields{
			"character_id": character.CharacterID,
			"system_id":    systemID,
			"structure_id": structureID,
			"owner_id":     structure.OwnerID,
			"reason":       ownerReason,
		}).Debug("jumpbridge discovery system candidate filtered: structure owner not allowed")
		return
	}

	if _, exists := endpoints[toSystemID]; !exists {
		endpoints[toSystemID] = discoveredEndpoint{
			structureID:  structureID,
			ownerAllowed: true,
		}
	}
}

func (s *JumpbridgeService) isStructureOwnerAllowed(
	ctx context.Context,
	ownerCorporationID int,
	allowedAllianceSet map[int]struct{},
	allowedCorporationSet map[int]struct{},
	corporationAllianceCache map[int]int,
) (allowed bool, reason string) {
	if ownerCorporationID <= 0 {
		return false, "missing owner corporation id"
	}

	if _, ok := allowedCorporationSet[ownerCorporationID]; ok {
		return true, "owner corporation is explicitly allowed"
	}

	if len(allowedAllianceSet) == 0 {
		return false, "no allowed alliances configured"
	}

	if allianceID, ok := corporationAllianceCache[ownerCorporationID]; ok {
		return s.allowByAllianceIDOrRefresh(
			ctx,
			ownerCorporationID,
			allianceID,
			allowedAllianceSet,
			corporationAllianceCache,
			"cache",
		)
	}

	_, _, allianceID, ok, err := store.GetOrFetchCorporation(ctx, s.App, s.PublicESI, ownerCorporationID)
	if err != nil {
		return false, fmt.Sprintf("failed resolving owner corporation alliance: %v", err)
	}

	if !ok {
		return false, "owner corporation details unavailable"
	}
	corporationAllianceCache[ownerCorporationID] = allianceID
	return s.allowByAllianceIDOrRefresh(
		ctx,
		ownerCorporationID,
		allianceID,
		allowedAllianceSet,
		corporationAllianceCache,
		"resolved",
	)
}

func (s *JumpbridgeService) refreshOwnerCorporationAlliance(ctx context.Context, ownerCorporationID int) (allianceID int, refreshed bool, reason string) {
	if ownerCorporationID <= 0 {
		return 0, false, "missing owner corporation id"
	}

	if s.PublicESI == nil {
		return 0, false, "owner corporation alliance unresolved: public esi client unavailable"
	}

	profile, err := s.PublicESI.CorporationProfile(ctx, ownerCorporationID)
	if err != nil {
		return 0, false, fmt.Sprintf("failed refreshing owner corporation alliance: %v", err)
	}

	if strings.TrimSpace(profile.Name) == "" {
		return 0, false, "owner corporation alliance unresolved: empty corporation profile"
	}

	if s.App != nil {
		_ = store.UpsertCorporationProfile(
			s.App,
			ownerCorporationID,
			profile.Name,
			profile.Ticker,
			profile.AllianceID,
			profile.MemberCount,
		)
	}
	return profile.AllianceID, true, "owner corporation alliance refreshed"
}

func (s *JumpbridgeService) allowByAllianceIDOrRefresh(
	ctx context.Context,
	ownerCorporationID int,
	allianceID int,
	allowedAllianceSet map[int]struct{},
	corporationAllianceCache map[int]int,
	source string,
) (allowed bool, reason string) {
	if _, ok := allowedAllianceSet[allianceID]; ok {
		return true, fmt.Sprintf("owner corporation alliance is allowed (%s)", source)
	}

	if allianceID > 0 {
		return false, fmt.Sprintf("owner corporation alliance not allowed (%s)", source)
	}

	refreshedAllianceID, refreshed, refreshReason := s.refreshOwnerCorporationAlliance(ctx, ownerCorporationID)
	if !refreshed {
		return false, refreshReason
	}
	corporationAllianceCache[ownerCorporationID] = refreshedAllianceID
	if _, ok := allowedAllianceSet[refreshedAllianceID]; ok {
		return true, "owner corporation alliance is allowed (refreshed)"
	}
	return false, "owner corporation alliance not allowed (refreshed)"
}

func toIntSet(values []int) map[int]struct{} {
	out := make(map[int]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}

func parseAnsiblexDestinationSystemID(structureName string, sourceSystemID int, systemNameToID map[string]int) int {
	value := strings.TrimSpace(structureName)
	if value == "" || len(systemNameToID) == 0 {
		return 0
	}

	if destinationID := parseDirectionalFieldsDestination(value, sourceSystemID, systemNameToID); destinationID > 0 {
		return destinationID
	}
	return parseSeparatedDestination(value, sourceSystemID, systemNameToID)
}

func parseDirectionalFieldsDestination(structureName string, sourceSystemID int, systemNameToID map[string]int) int {
	// Preferred parse path for names like:
	// "P-E9GN » H-EY0P - Penguin Chute"
	// where fields[0] and fields[2] are the endpoint system names.
	fields := strings.Fields(structureName)
	if len(fields) < minDirectionalFields {
		return 0
	}
	switch fields[1] {
	case "»", "<->", "↔", "->":
		return matchPairSystemIDs(fields[0], fields[2], sourceSystemID, systemNameToID)
	default:
		return 0
	}
}

func parseSeparatedDestination(structureName string, sourceSystemID int, systemNameToID map[string]int) int {
	separators := []string{" <-> ", " ↔ ", " -> ", " » ", " - ", "|", "/"}
	for _, separator := range separators {
		parts := strings.Split(structureName, separator)
		if len(parts) != expectedPairParts {
			continue
		}
		destinationID := matchPairSystemIDs(parts[0], parts[1], sourceSystemID, systemNameToID)
		if destinationID > 0 {
			return destinationID
		}
	}
	return 0
}

func matchPairSystemIDs(leftRaw, rightRaw string, sourceSystemID int, systemNameToID map[string]int) int {
	leftID := lookupSystemID(leftRaw, systemNameToID)
	rightID := lookupSystemID(rightRaw, systemNameToID)
	if leftID == sourceSystemID && rightID > 0 {
		return rightID
	}

	if rightID == sourceSystemID && leftID > 0 {
		return leftID
	}
	return 0
}

func lookupSystemID(raw string, systemNameToID map[string]int) int {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return 0
	}
	systemID, ok := systemNameToID[key]
	if !ok {
		return 0
	}
	return systemID
}
