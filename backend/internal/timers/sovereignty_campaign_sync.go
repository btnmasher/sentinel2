package timers

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/fnt-eve/goesi-openapi/esi"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/store"
)

const (
	sovCampaignEventIHUBDefense = "ihub_defense"
	sovCampaignSourcePrefix     = "esi:sovereignty_campaign:"
	sovCampaignSourceName       = "esi"
	scorePercentCap             = 100.0
	scoreTotalStages            = 100
	sovCampaignLogSampleLimit   = 5
	attackerPriorityHostile     = 5
	attackerPriorityComplicated = 4
	attackerPriorityNeutral     = 3
	attackerPriorityFriendly    = 2
	attackerPriorityOurs        = 1
	maxAttackerNameSamples      = 4
)

type SovCampaignSyncResult struct {
	Fetched    int
	Considered int
	Created    int
	Updated    int
	Canceled   int
	Skipped    int
}

type sovWatchlist struct {
	alliances map[int]sovWatchlistAlliance
}

type sovWatchlistAlliance struct {
	standing string
	name     string
	ticker   string
}

type sovDesiredRecords struct {
	records map[string]*core.Record
	refs    map[string]struct{}
}

func (a sovWatchlist) standingForAlliance(allianceID int) (string, bool) {
	if allianceID <= 0 {
		return "", false
	}
	value, ok := a.alliances[allianceID]
	if !ok {
		return "", false
	}
	standing := normalizeWatchlistStanding(value.standing)
	return standing, standing != ""
}

func (a sovWatchlist) labelForAlliance(allianceID int) (name, ticker string, ok bool) {
	if allianceID <= 0 {
		return "", "", false
	}
	value, exists := a.alliances[allianceID]
	if !exists {
		return "", "", false
	}
	name = strings.TrimSpace(value.name)
	if name == "" {
		return "", "", false
	}
	return name, strings.TrimSpace(value.ticker), true
}

func (a sovWatchlist) standingForAttackers(participants []esi.SovereigntyCampaignsGetInnerParticipantsInner) (standing string, allianceID int, ok bool) {
	bestStanding := ""
	bestAllianceID := 0
	bestPriority := -1
	for _, participant := range participants {
		allianceID := int(participant.GetAllianceId())
		standing, ok := a.standingForAlliance(allianceID)
		if !ok {
			continue
		}
		priority := attackerStandingPriority(standing)
		if priority <= bestPriority {
			continue
		}
		bestPriority = priority
		bestStanding = standing
		bestAllianceID = allianceID
	}
	if bestPriority < 0 {
		return "", 0, false
	}
	return bestStanding, bestAllianceID, true
}

func attackerStandingPriority(standing string) int {
	switch normalizeWatchlistStanding(standing) {
	case TimerStandingHostile:
		return attackerPriorityHostile
	case TimerStandingComplicated:
		return attackerPriorityComplicated
	case TimerStandingNeutral:
		return attackerPriorityNeutral
	case TimerStandingFriendly:
		return attackerPriorityFriendly
	case TimerStandingOurs:
		return attackerPriorityOurs
	default:
		return 0
	}
}

func (s *Service) SyncSovereigntyCampaignTimers(ctx context.Context) (SovCampaignSyncResult, error) {
	result := SovCampaignSyncResult{}
	if s == nil || s.App == nil || s.PublicESI == nil {
		return result, ErrESIPublicClientNotConfigured
	}

	watchlist, err := s.loadSovWatchlist()
	if err != nil {
		return result, err
	}

	campaigns, err := s.PublicESI.SovereigntyCampaigns(ctx)
	if err != nil {
		return result, err
	}
	s.logSovCampaignFetch(campaigns)
	result.Fetched = len(campaigns)
	timersCollection, err := s.App.FindCollectionByNameOrId(store.CollectionTimers)
	if err != nil {
		return result, err
	}

	desired := s.buildDesiredSovRecords(ctx, campaigns, watchlist, timersCollection, &result)

	existingByRef, err := s.loadExistingSovRecords()
	if err != nil {
		return result, err
	}

	if err := s.applyDesiredSovRecords(desired.records, existingByRef, &result); err != nil {
		return result, err
	}
	if err := s.cancelStaleSovRecords(existingByRef, desired.refs, &result); err != nil {
		return result, err
	}

	return result, nil
}

func (s *Service) buildDesiredSovRecords(
	ctx context.Context,
	campaigns []esi.SovereigntyCampaignsGetInner,
	watchlist sovWatchlist,
	timersCollection *core.Collection,
	result *SovCampaignSyncResult,
) sovDesiredRecords {
	desired := sovDesiredRecords{
		records: map[string]*core.Record{},
		refs:    map[string]struct{}{},
	}
	for i := range campaigns {
		sourceRef, record, include := s.desiredSovRecordForCampaign(ctx, &campaigns[i], watchlist, timersCollection, result)
		if !include {
			continue
		}
		desired.records[sourceRef] = record
		desired.refs[sourceRef] = struct{}{}
	}
	return desired
}

func (s *Service) desiredSovRecordForCampaign(
	ctx context.Context,
	campaign *esi.SovereigntyCampaignsGetInner,
	watchlist sovWatchlist,
	timersCollection *core.Collection,
	result *SovCampaignSyncResult,
) (string, *core.Record, bool) {
	if campaign == nil {
		return "", nil, false
	}
	if strings.TrimSpace(strings.ToLower(campaign.GetEventType())) != sovCampaignEventIHUBDefense {
		return "", nil, false
	}
	defenderIDPtr, hasDefender := campaign.GetDefenderIdOk()
	if !hasDefender {
		result.Skipped++
		return "", nil, false
	}
	defenderAllianceID := int(*defenderIDPtr)
	standing, ownerAllianceID, allowed := campaignWatchMatch(campaign, watchlist, defenderAllianceID)
	if !allowed || ownerAllianceID <= 0 {
		return "", nil, false
	}

	systemID := int(campaign.GetSolarSystemId())
	system, resolveErr := s.ResolveSystem(systemID, "")
	if resolveErr != nil || system == nil {
		result.Skipped++
		return "", nil, false
	}

	expiresAt := campaign.GetStartTime().UTC()
	if expiresAt.IsZero() {
		result.Skipped++
		return "", nil, false
	}

	attackersScore := campaignScore(campaign.GetAttackersScoreOk())
	defenderScore := campaignScore(campaign.GetDefenderScoreOk())
	stage, totalStages := campaignProgress(attackersScore)
	severity := campaignSeverity(standing)
	notes := s.campaignNotes(ctx, campaign, watchlist)
	allianceName, allianceTicker := s.resolveAllianceNameTicker(ctx, ownerAllianceID, watchlist)
	sourceRef := sovCampaignSourcePrefix + strconv.FormatInt(campaign.GetCampaignId(), 10)
	result.Considered++

	input := &CreateInput{
		Title:               fmt.Sprintf("%s iHub Defense", system.GetString("name")),
		SystemID:            systemID,
		Standing:            standing,
		TimerKind:           TimerKindReinforcement,
		StructureType:       TimerStructureSovereigntyHub,
		StageLabel:          TimerStageReinforcement,
		OwnerAllianceID:     ownerAllianceID,
		OwnerAllianceName:   allianceName,
		OwnerAllianceTicker: allianceTicker,
		Stage:               stage,
		TotalStages:         totalStages,
		AttackersScorePct:   intPtr(stage),
		DefenderScorePct:    intPtr(int(math.Round(min(scorePercentCap, max(0.0, defenderScore))))),
		Severity:            severity,
		Status:              timerStatusActive,
		ExpiresAt:           expiresAt,
		Source:              sovCampaignSourceName,
		SourceRef:           sourceRef,
		Notes:               notes,
		RawText:             "esi sovereignty campaign",
		ReplacementAction:   "not_replaceable",
	}
	record := core.NewRecord(timersCollection)
	s.applyCreateInput(record, system, input, nil)
	record.Set("canceled_at", nil)
	record.Set("canceled_by", nil)
	return sourceRef, record, true
}

func campaignScore(score *float64, ok bool) float64 {
	if !ok || score == nil {
		return 0
	}
	return *score
}

func (s *Service) loadExistingSovRecords() (map[string]*core.Record, error) {
	existing, err := s.App.FindRecordsByFilter(
		store.CollectionTimers,
		"source = {:source} && source_ref ~ {:source_ref}",
		"",
		0,
		0,
		dbx.Params{
			"source":     sovCampaignSourceName,
			"source_ref": sovCampaignSourcePrefix + "%",
		},
	)
	if err != nil {
		return nil, err
	}
	existingByRef := map[string]*core.Record{}
	for _, record := range existing {
		ref := strings.TrimSpace(record.GetString("source_ref"))
		if ref == "" {
			continue
		}
		existingByRef[ref] = record
	}
	return existingByRef, nil
}

func (s *Service) applyDesiredSovRecords(desired, existingByRef map[string]*core.Record, result *SovCampaignSyncResult) error {
	for sourceRef, desiredRecord := range desired {
		existingRecord, exists := existingByRef[sourceRef]
		if !exists {
			if saveErr := s.App.Save(desiredRecord); saveErr != nil {
				return saveErr
			}
			result.Created++
			continue
		}
		if !applyTimerRecordFromSource(existingRecord, desiredRecord) {
			continue
		}
		if saveErr := s.App.Save(existingRecord); saveErr != nil {
			return saveErr
		}
		result.Updated++
	}
	return nil
}

func (s *Service) cancelStaleSovRecords(existingByRef map[string]*core.Record, desiredRefs map[string]struct{}, result *SovCampaignSyncResult) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for sourceRef, existingRecord := range existingByRef {
		if _, wanted := desiredRefs[sourceRef]; wanted {
			continue
		}
		if existingRecord.GetString("status") == timerStatusCanceled {
			continue
		}
		existingRecord.Set("status", timerStatusCanceled)
		existingRecord.Set("canceled_at", now)
		if saveErr := s.App.Save(existingRecord); saveErr != nil {
			return saveErr
		}
		result.Canceled++
	}
	return nil
}

func campaignWatchMatch(campaign *esi.SovereigntyCampaignsGetInner, watchlist sovWatchlist, defenderAllianceID int) (standing string, ownerAllianceID int, allowed bool) {
	if campaign == nil {
		return "", 0, false
	}
	if defenderStanding, ok := watchlist.standingForAlliance(defenderAllianceID); ok {
		return defenderStanding, defenderAllianceID, true
	}
	participants, hasParticipants := campaign.GetParticipantsOk()
	if !hasParticipants || len(participants) == 0 {
		return "", 0, false
	}
	attackerStanding, attackerAllianceID, ok := watchlist.standingForAttackers(participants)
	if !ok || attackerAllianceID <= 0 {
		return "", 0, false
	}
	return attackerStanding, attackerAllianceID, true
}

func (s *Service) loadSovWatchlist() (sovWatchlist, error) {
	result := sovWatchlist{
		alliances: map[int]sovWatchlistAlliance{},
	}
	records, err := s.App.FindRecordsByFilter(
		store.CollectionOrganizationStandings,
		"include_in_sov_sync = true && alliance_id > 0",
		"",
		0,
		0,
		nil,
	)
	if err != nil {
		return result, err
	}
	for _, record := range records {
		standing := normalizeWatchlistStanding(record.GetString("hostility"))
		entityID := record.GetInt("alliance_id")
		if entityID > 0 {
			result.alliances[entityID] = sovWatchlistAlliance{
				standing: standing,
				name:     strings.TrimSpace(record.GetString("alliance_name")),
				ticker:   strings.TrimSpace(record.GetString("alliance_ticker")),
			}
		}
	}
	return result, nil
}

func normalizeWatchlistStanding(value string) string {
	return NormalizeStanding(value)
}

func (s *Service) resolveAllianceNameTicker(ctx context.Context, allianceID int, watchlist sovWatchlist) (name, ticker string) {
	if allianceID <= 0 {
		return "", ""
	}
	if name, ticker, ok := watchlist.labelForAlliance(allianceID); ok {
		return name, ticker
	}
	name, ticker, ok, err := store.GetOrFetchAlliance(ctx, s.App, s.PublicESI, allianceID)
	if err != nil || !ok {
		return "", ""
	}
	return name, ticker
}

func applyTimerRecordFromSource(existing, desired *core.Record) bool {
	if existing == nil || desired == nil {
		return false
	}
	changed := false
	for _, field := range []string{
		"title",
		"system_id",
		"system_name",
		"region_id",
		"region_name",
		"standing_type",
		"timer_kind",
		"structure_type",
		"stage_label",
		"owner_alliance_id",
		"owner_alliance_name",
		"owner_alliance_ticker",
		"stage",
		"total_stages",
		"attackers_score_pct",
		"defender_score_pct",
		"severity",
		"expires_at",
		"source",
		"source_ref",
		"notes",
		"raw_text",
		"replacement_action",
		"created_by_name",
	} {
		desiredValue := desired.Get(field)
		if valuesEqual(existing.Get(field), desiredValue) {
			continue
		}
		existing.Set(field, desiredValue)
		changed = true
	}
	if existing.GetString("status") != timerStatusActive {
		existing.Set("status", timerStatusActive)
		existing.Set("canceled_at", nil)
		existing.Set("canceled_by", nil)
		changed = true
	}
	return changed
}

func campaignProgress(attackersScore float64) (stage, totalStages int) {
	clamped := min(scorePercentCap, max(0.0, attackersScore))
	return int(math.Round(clamped)), scoreTotalStages
}

func campaignSeverity(standing string) string {
	switch normalizeWatchlistStanding(standing) {
	case TimerStandingOurs:
		return TimerSeverityCritical
	case TimerStandingFriendly:
		return TimerSeverityHigh
	default:
		return TimerSeverityMedium
	}
}

func (s *Service) campaignNotes(
	ctx context.Context,
	campaign *esi.SovereigntyCampaignsGetInner,
	watchlist sovWatchlist,
) string {
	if campaign == nil {
		return ""
	}
	participants, ok := campaign.GetParticipantsOk()
	if !ok || len(participants) == 0 {
		return "Attackers: Unknown"
	}
	names, unknownCount := s.resolveParticipantAllianceLabels(ctx, participants, watchlist)
	return formatAttackerSummary(names, unknownCount)
}

func (s *Service) resolveParticipantAllianceLabels(
	ctx context.Context,
	participants []esi.SovereigntyCampaignsGetInnerParticipantsInner,
	watchlist sovWatchlist,
) (labels []string, unknownCount int) {
	seen := map[int]struct{}{}
	labels = make([]string, 0, len(participants))
	unknownCount = 0
	for _, participant := range participants {
		allianceID := int(participant.GetAllianceId())
		if allianceID <= 0 {
			continue
		}
		if _, exists := seen[allianceID]; exists {
			continue
		}
		seen[allianceID] = struct{}{}
		name, ticker := s.resolveAllianceNameTicker(ctx, allianceID, watchlist)
		if strings.TrimSpace(name) == "" {
			unknownCount++
			continue
		}
		label := strings.TrimSpace(name)
		if strings.TrimSpace(ticker) != "" {
			label = fmt.Sprintf("[%s] %s", strings.TrimSpace(ticker), label)
		}
		if standing, ok := watchlist.standingForAlliance(allianceID); ok && standing != "" {
			label = fmt.Sprintf("%s (%s)", label, standing)
		}
		labels = append(labels, label)
	}
	return labels, unknownCount
}

func formatAttackerSummary(names []string, unknownCount int) string {
	if len(names) == 0 && unknownCount == 0 {
		return "Attackers: Unknown"
	}
	if len(names) > maxAttackerNameSamples {
		extra := len(names) - maxAttackerNameSamples
		return fmt.Sprintf(
			"Attackers: %s (+%d more)",
			strings.Join(names[:maxAttackerNameSamples], ", "),
			extra,
		)
	}
	if len(names) == 0 {
		return "Attackers: Unknown"
	}
	if unknownCount > 0 {
		return fmt.Sprintf("Attackers: %s (+%d unknown)", strings.Join(names, ", "), unknownCount)
	}
	return "Attackers: " + strings.Join(names, ", ")
}

func valuesEqual(left, right any) bool {
	return fmt.Sprint(left) == fmt.Sprint(right)
}

func intPtr(value int) *int {
	return &value
}

func (s *Service) logSovCampaignFetch(campaigns []esi.SovereigntyCampaignsGetInner) {
	if s == nil || s.App == nil {
		return
	}
	samples := make([]map[string]any, 0, min(sovCampaignLogSampleLimit, len(campaigns)))
	for i, campaign := range campaigns {
		if i >= sovCampaignLogSampleLimit {
			break
		}
		attackersScore := 0.0
		if score, ok := campaign.GetAttackersScoreOk(); ok {
			attackersScore = *score
		}
		defenderScore := 0.0
		if score, ok := campaign.GetDefenderScoreOk(); ok {
			defenderScore = *score
		}
		samples = append(samples, map[string]any{
			"campaign_id":    campaign.GetCampaignId(),
			"event_type":     campaign.GetEventType(),
			"system_id":      campaign.GetSolarSystemId(),
			"start_time":     campaign.GetStartTime().UTC().Format(time.RFC3339),
			"attacker_score": attackersScore,
			"defender_score": defenderScore,
		})
	}
	s.App.Logger().Debug(
		"sov campaign fetch result sample",
		slog.Int("count", len(campaigns)),
		slog.Any("samples", samples),
	)
}
