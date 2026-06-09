package zkill

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"
	esiapi "github.com/fnt-eve/goesi-openapi/esi"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/config"
	"sentinel2/internal/intel"
	"sentinel2/internal/logging"
	"sentinel2/internal/realtime"
	"sentinel2/internal/store"
)

const (
	defaultPollInterval      = 10 * time.Second
	minimumPollInterval      = 6 * time.Second
	maxFetchPerTick          = 25
	stateKeyMain             = "main"
	defaultMaxEventAge       = 5 * time.Minute
	standingsReloadInterval  = 1 * time.Minute
	sequenceRequestTimeout   = 8 * time.Second
	killmailRequestTimeout   = 8 * time.Second
	nameLookupRequestTimeout = 5 * time.Second
	backgroundHTTPIdleConn   = 30 * time.Second
	defaultHTTPTimeout       = 10 * time.Second
	defaultIdleConns         = 10
	defaultHeaderTimeout     = 5 * time.Second
	errorBodyPreviewBytes    = 512
	jSpaceSystemNameLength   = 7
	capsuleShipTypeID        = 670
	lowValueCapsuleISKMax    = 1_000_000
	zkillReportAuthor        = "zKillboard"
	zkillReportSource        = "zkill_feed"
	standingOurs             = "ours"
	standingFriendly         = "friendly"
	standingNeutral          = "neutral"
	standingComplicated      = "complicated"
	standingHostile          = "hostile"
	recentKillmailLRUSize    = 50000
)

type FeedIngestor struct {
	App    *pocketbase.PocketBase
	Config *config.Config
	Intel  *intel.IntelService
	Topics *realtime.Publisher
	logger *logging.Logger

	httpClient *http.Client
	resolver   *nameResolver

	systemsByID map[int]intel.IntelSystem
	itemTypes   map[int]string
	standings   standingsCache
	regionNames map[int]string
	recentKills *killmailLRU
}

type standingsCache struct {
	loadedAt     time.Time
	alliances    map[int]string
	corporations map[int]string
}

type killmailLRU struct {
	order *list.List
	seen  map[int64]*list.Element
	limit int
}

type nameResolver struct {
	client *esiapi.APIClient
	mu     sync.RWMutex
	cache  map[int64]string
}

type sequenceResponse struct {
	Sequence   int64 `json:"sequence"`
	SequenceID int64 `json:"sequence_id"`
}

type rawKillmailEnvelope struct {
	SequenceID    int64         `json:"sequence_id"`
	KillmailID    int64         `json:"killmail_id"`
	KillmailTime  string        `json:"killmail_time"`
	SolarSystemID int           `json:"solar_system_id"`
	Victim        rawVictim     `json:"victim"`
	Attackers     []rawAttacker `json:"attackers"`
	Killmail      *rawKillmail  `json:"killmail"`
	ESI           *rawKillmail  `json:"esi"`
	ZKB           *rawZKB       `json:"zkb"`
}

type rawKillmail struct {
	KillmailID    int64         `json:"killmail_id"`
	KillmailTime  string        `json:"killmail_time"`
	SolarSystemID int           `json:"solar_system_id"`
	Victim        rawVictim     `json:"victim"`
	Attackers     []rawAttacker `json:"attackers"`
}

type rawVictim struct {
	CharacterID   int `json:"character_id"`
	CorporationID int `json:"corporation_id"`
	AllianceID    int `json:"alliance_id"`
	ShipTypeID    int `json:"ship_type_id"`
}

type rawAttacker struct {
	CharacterID   int  `json:"character_id"`
	CorporationID int  `json:"corporation_id"`
	AllianceID    int  `json:"alliance_id"`
	FinalBlow     bool `json:"final_blow"`
}

type rawZKB struct {
	Solo          bool     `json:"solo"`
	AttackerCount int      `json:"attackerCount"`
	TotalValue    *float64 `json:"totalValue"`
}

type killmailEvent struct {
	SequenceID    int64
	KillmailID    int64
	KillmailTime  time.Time
	SolarSystemID int
	Victim        rawVictim
	Attackers     []rawAttacker
	Solo          bool
	AttackerCount int
	TotalValue    *float64
}

func NewFeedIngestor(
	app *pocketbase.PocketBase,
	cfg *config.Config,
	intelSvc *intel.IntelService,
	publisher *realtime.Publisher,
) *FeedIngestor {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          defaultIdleConns,
		MaxIdleConnsPerHost:   defaultIdleConns,
		IdleConnTimeout:       backgroundHTTPIdleConn,
		TLSHandshakeTimeout:   defaultHeaderTimeout,
		ResponseHeaderTimeout: defaultHeaderTimeout,
	}
	resolverClient := goesi.NewPublicESIClient(cfg.ESIUserAgent)
	return &FeedIngestor{
		App:    app,
		Config: cfg,
		Intel:  intelSvc,
		Topics: publisher,
		logger: logging.New(app).WithFields(logging.Fields{
			"source": "zkill.feed",
		}),
		httpClient: &http.Client{
			Timeout:   defaultHTTPTimeout,
			Transport: transport,
		},
		resolver: &nameResolver{
			client: resolverClient,
			cache:  map[int64]string{},
		},
		systemsByID: map[int]intel.IntelSystem{},
		itemTypes:   map[int]string{},
		regionNames: map[int]string{},
		standings: standingsCache{
			alliances:    map[int]string{},
			corporations: map[int]string{},
		},
		recentKills: newKillmailLRU(recentKillmailLRUSize),
	}
}

//nolint:gocognit // worker loop intentionally handles all poll states in one place.
func (i *FeedIngestor) Run(ctx context.Context) {
	if i == nil || i.App == nil || i.Config == nil {
		return
	}

	if !i.Config.ZKillFeedEnabled {
		return
	}

	if err := i.loadSystems(); err != nil {
		i.logger.WithErr(err).Error("zkill worker failed to load systems")
		return
	}

	if err := i.loadItemTypes(); err != nil {
		i.logger.WithErr(err).Warn("zkill worker failed to load item types; ship names will fall back to ESI")
	}

	if err := i.loadRegionNames(); err != nil {
		i.logger.WithErr(err).Warn("zkill worker failed to load regions")
	}

	pollInterval := i.pollInterval()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	i.logger.WithFields(logging.Fields{"poll_interval_seconds": int(pollInterval.Seconds())}).Info("zkill worker started")

	nextSequence := int64(0)
	for {
		if ctx.Err() != nil {
			return
		}

		if nextSequence <= 0 {
			sequence, sequenceErr := i.bootstrapSequence(ctx)
			if sequenceErr != nil {
				i.logger.WithErr(sequenceErr).Warn("zkill worker failed to bootstrap sequence")
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				continue
			}
			nextSequence = sequence
		}

		if refreshErr := i.refreshStandingsIfStale(); refreshErr != nil {
			i.logger.WithErr(refreshErr).Warn("zkill worker failed to refresh standings")
		}

		pollStartSequence := nextSequence
		processed := 0
		ingested := 0
		pollState := "tick_limit"
		for processed < maxFetchPerTick {
			if ctx.Err() != nil {
				return
			}
			event, state, fetchErr := i.fetchSequenceEvent(ctx, nextSequence)
			if fetchErr != nil {
				i.logger.WithFields(logging.Fields{"sequence_id": nextSequence}).WithErr(fetchErr).Warn("zkill sequence fetch failed")
				pollState = "fetch_error"
				break
			}
			if state == "empty" || state == "rate_limited" {
				pollState = state
				break
			}
			if event == nil {
				if checkpointErr := i.saveCheckpoint(nextSequence); checkpointErr != nil {
					i.logger.WithFields(logging.Fields{"sequence_id": nextSequence}).WithErr(checkpointErr).Warn("zkill checkpoint save failed")
					pollState = "checkpoint_error"
					break
				}
				nextSequence++
				processed++
				continue
			}

			created, handleErr := i.handleEvent(ctx, event)
			if handleErr != nil {
				i.logger.WithFields(logging.Fields{"sequence_id": nextSequence, "killmail_id": event.KillmailID}).WithErr(handleErr).Warn("zkill event handling failed")
				pollState = "handle_error"
				break
			}
			if created {
				ingested++
			}
			if checkpointErr := i.saveCheckpoint(nextSequence); checkpointErr != nil {
				i.logger.WithFields(logging.Fields{"sequence_id": nextSequence}).WithErr(checkpointErr).Warn("zkill checkpoint save failed")
				pollState = "checkpoint_error"
				break
			}
			nextSequence++
			processed++
		}
		i.logger.WithFields(logging.Fields{
			"sequence_start":     pollStartSequence,
			"next_sequence":      nextSequence,
			"sequences_advanced": processed,
			"ingested_events":    ingested,
			"state":              pollState,
		}).Debug("zkill poll complete")

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (i *FeedIngestor) pollInterval() time.Duration {
	if i == nil || i.Config == nil {
		return defaultPollInterval
	}
	seconds := i.Config.ZKillFeedPollSeconds
	if seconds <= 0 {
		return defaultPollInterval
	}
	d := time.Duration(seconds) * time.Second
	if d < minimumPollInterval {
		return minimumPollInterval
	}
	return d
}

func (i *FeedIngestor) maxEventAge() time.Duration {
	if i == nil || i.Config == nil {
		return defaultMaxEventAge
	}
	seconds := i.Config.ZKillMaxEventAgeSec
	if seconds <= 0 {
		return defaultMaxEventAge
	}
	return time.Duration(seconds) * time.Second
}

func (i *FeedIngestor) baseURL() string {
	if i == nil || i.Config == nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(i.Config.ZKillFeedBaseURL), "/")
}

func (i *FeedIngestor) bootstrapSequence(ctx context.Context) (int64, error) {
	current, err := i.fetchCurrentSequence(ctx)
	if err != nil {
		return 0, err
	}

	if current <= 0 {
		return 0, fmt.Errorf("invalid sequence response")
	}
	return current, nil
}

func (i *FeedIngestor) fetchCurrentSequence(ctx context.Context) (int64, error) {
	requestCtx, cancel := context.WithTimeout(ctx, sequenceRequestTimeout)
	defer cancel()

	url := i.baseURL() + "/sequence.json"
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return 0, err
	}
	resp, err := i.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("sequence request returned status %d", resp.StatusCode)
	}
	payload := sequenceResponse{}

	if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
		return 0, decodeErr
	}

	if payload.Sequence > 0 {
		return payload.Sequence, nil
	}

	if payload.SequenceID > 0 {
		return payload.SequenceID, nil
	}
	return 0, nil
}

func (i *FeedIngestor) fetchSequenceEvent(ctx context.Context, sequenceID int64) (*killmailEvent, string, error) {
	requestCtx, cancel := context.WithTimeout(ctx, killmailRequestTimeout)
	defer cancel()
	url := fmt.Sprintf("%s/%d.json", i.baseURL(), sequenceID)
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, "", err
	}
	resp, err := i.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, "empty", nil
	case http.StatusTooManyRequests:
		return nil, "rate_limited", nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, errorBodyPreviewBytes))
		return nil, "", fmt.Errorf("sequence request %d returned status %d: %s", sequenceID, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	rawBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, "", readErr
	}
	payload := rawKillmailEnvelope{}

	if decodeErr := json.Unmarshal(rawBody, &payload); decodeErr != nil {
		return nil, "", decodeErr
	}
	event, eventErr := decodeKillmailEnvelope(&payload)
	if eventErr != nil {
		i.logger.WithFields(logging.Fields{"sequence_id": sequenceID}).WithErr(eventErr).Warn("zkill event decode skipped")
		return nil, "ok", nil
	}

	if event.SequenceID == 0 {
		event.SequenceID = sequenceID
	}
	return event, "ok", nil
}

func decodeKillmailEnvelope(payload *rawKillmailEnvelope) (*killmailEvent, error) {
	if payload == nil {
		return nil, fmt.Errorf("missing payload")
	}
	resolved := killmailEvent{SequenceID: payload.SequenceID}
	container := payload.ESI
	if container == nil {
		container = payload.Killmail
	}

	if container != nil {
		if err := applyContainerToEvent(&resolved, container, payload.KillmailTime); err != nil {
			return nil, err
		}
	} else {
		resolved.KillmailID = payload.KillmailID
		resolved.SolarSystemID = payload.SolarSystemID
		resolved.Victim = payload.Victim
		resolved.Attackers = payload.Attackers
		t, err := parseKillmailTime(payload.KillmailTime)
		if err != nil {
			return nil, err
		}
		resolved.KillmailTime = t
	}

	if payload.ZKB != nil {
		resolved.Solo = payload.ZKB.Solo
		resolved.AttackerCount = payload.ZKB.AttackerCount
		resolved.TotalValue = payload.ZKB.TotalValue
	}

	if resolved.KillmailID <= 0 {
		return nil, fmt.Errorf("missing killmail_id")
	}

	if resolved.SolarSystemID <= 0 {
		return nil, fmt.Errorf("missing solar_system_id")
	}

	if resolved.KillmailTime.IsZero() {
		return nil, fmt.Errorf("missing killmail_time")
	}
	return &resolved, nil
}

func applyContainerToEvent(resolved *killmailEvent, container *rawKillmail, fallbackTime string) error {
	if resolved == nil || container == nil {
		return nil
	}
	resolved.KillmailID = container.KillmailID
	resolved.SolarSystemID = container.SolarSystemID
	resolved.Victim = container.Victim
	resolved.Attackers = container.Attackers
	killmailTime := strings.TrimSpace(container.KillmailTime)
	if killmailTime == "" {
		killmailTime = strings.TrimSpace(fallbackTime)
	}
	t, err := parseKillmailTime(killmailTime)
	if err != nil {
		return err
	}
	resolved.KillmailTime = t
	return nil
}

func parseKillmailTime(raw string) (time.Time, error) {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return time.Time{}, fmt.Errorf("missing killmail time")
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, clean); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid killmail time %q", clean)
}

//nolint:gocognit // event filtering + enrichment logic is intentionally centralized.
func (i *FeedIngestor) handleEvent(ctx context.Context, event *killmailEvent) (bool, error) {
	if event == nil {
		return false, nil
	}

	if i.recentKills != nil && i.recentKills.Seen(event.KillmailID) {
		i.logger.WithFields(logging.Fields{
			"killmail_id": event.KillmailID,
			"sequence_id": event.SequenceID,
			"reason":      "already_seen_recent",
		}).Debug("zkill event skipped")
		return false, nil
	}

	if i.recentKills != nil {
		i.recentKills.Add(event.KillmailID)
	}

	age := time.Since(event.KillmailTime)
	if age < -2*time.Minute {
		i.logger.WithFields(logging.Fields{
			"killmail_id":   event.KillmailID,
			"sequence_id":   event.SequenceID,
			"reason":        "future_timestamp",
			"age_seconds":   int(age.Seconds()),
			"killmail_time": event.KillmailTime.UTC().Format(time.RFC3339),
		}).Debug("zkill event skipped")
		return false, nil
	}

	if age > i.maxEventAge() {
		i.logger.WithFields(logging.Fields{
			"killmail_id":     event.KillmailID,
			"sequence_id":     event.SequenceID,
			"reason":          "outside_age_window",
			"age_seconds":     int(age.Seconds()),
			"max_age_seconds": int(i.maxEventAge().Seconds()),
			"killmail_time":   event.KillmailTime.UTC().Format(time.RFC3339),
		}).Debug("zkill event skipped")
		return false, nil
	}

	system, ok := i.systemsByID[event.SolarSystemID]
	if !ok {
		i.logger.WithFields(logging.Fields{
			"killmail_id":     event.KillmailID,
			"sequence_id":     event.SequenceID,
			"reason":          "unknown_solar_system",
			"solar_system_id": event.SolarSystemID,
		}).Debug("zkill event skipped")
		return false, nil
	}

	if isJSpaceSystemName(system.Name) {
		return false, nil
	}

	if isLowValueCapsuleKill(event) {
		return false, nil
	}
	regionTopic := realtime.TopicIntelZKillRegion(system.Region)
	if i.Topics == nil {
		return false, fmt.Errorf("missing realtime publisher")
	}

	if !i.Topics.HasSubscribers(regionTopic) {
		return false, nil
	}

	finalBlow := selectFinalBlowAttacker(event.Attackers)
	otherAttackers := otherAttackerCount(event)
	killerHostility := i.resolveStanding(
		finalBlow.CorporationID,
		finalBlow.AllianceID,
	)
	victimHostility := i.resolveStanding(
		event.Victim.CorporationID,
		event.Victim.AllianceID,
	)

	idsToResolve := []int64{}

	if finalBlow.CharacterID > 0 {
		idsToResolve = append(idsToResolve, int64(finalBlow.CharacterID))
	}

	if event.Victim.CharacterID > 0 {
		idsToResolve = append(idsToResolve, int64(event.Victim.CharacterID))
	}

	if finalBlow.CorporationID > 0 {
		idsToResolve = append(idsToResolve, int64(finalBlow.CorporationID))
	}

	if finalBlow.AllianceID > 0 {
		idsToResolve = append(idsToResolve, int64(finalBlow.AllianceID))
	}

	if event.Victim.CorporationID > 0 {
		idsToResolve = append(idsToResolve, int64(event.Victim.CorporationID))
	}

	if event.Victim.AllianceID > 0 {
		idsToResolve = append(idsToResolve, int64(event.Victim.AllianceID))
	}
	shipName := i.shipTypeName(event.Victim.ShipTypeID)
	if event.Victim.ShipTypeID > 0 && shipName == "" {
		idsToResolve = append(idsToResolve, int64(event.Victim.ShipTypeID))
	}
	resolvedNames, _ := i.resolver.ResolveNames(ctx, idsToResolve)

	killerName := "Unknown Killer"
	if finalBlow.CharacterID > 0 {
		if value := strings.TrimSpace(resolvedNames[int64(finalBlow.CharacterID)]); value != "" {
			killerName = value
		}
	}
	victimName := "Unknown Victim"
	if event.Victim.CharacterID > 0 {
		if value := strings.TrimSpace(resolvedNames[int64(event.Victim.CharacterID)]); value != "" {
			victimName = value
		}
	}

	if shipName == "" {
		shipName = "Unknown Ship"
	}

	if event.Victim.ShipTypeID > 0 && (shipName == "" || shipName == "Unknown Ship") {
		if value := strings.TrimSpace(resolvedNames[int64(event.Victim.ShipTypeID)]); value != "" {
			shipName = value
		} else {
			shipName = "Ship " + strconv.Itoa(event.Victim.ShipTypeID)
		}
	}
	killerCorporationName := ""
	if finalBlow.CorporationID > 0 {
		killerCorporationName = strings.TrimSpace(resolvedNames[int64(finalBlow.CorporationID)])
	}
	killerAllianceName := ""
	if finalBlow.AllianceID > 0 {
		killerAllianceName = strings.TrimSpace(resolvedNames[int64(finalBlow.AllianceID)])
	}
	victimCorporationName := ""
	if event.Victim.CorporationID > 0 {
		victimCorporationName = strings.TrimSpace(resolvedNames[int64(event.Victim.CorporationID)])
	}
	victimAllianceName := ""
	if event.Victim.AllianceID > 0 {
		victimAllianceName = strings.TrimSpace(resolvedNames[int64(event.Victim.AllianceID)])
	}
	regionName := strings.TrimSpace(i.regionNames[system.Region])

	involvedLabel := "(Solo)"
	if otherAttackers > 0 {
		involvedLabel = fmt.Sprintf("(+%d)", otherAttackers)
	}
	displayText := fmt.Sprintf("%s %s -> %s (%s)", killerName, involvedLabel, victimName, shipName)

	reportText := fmt.Sprintf("%s in %s", displayText, system.Name)
	report := intel.IntelReport{
		ID: event.SequenceID,
		RecordID: fmt.Sprintf(
			"zkill-%d",
			event.KillmailID,
		),
		Time:      event.KillmailTime.Unix(),
		Author:    zkillReportAuthor,
		Text:      reportText,
		Systems:   []intel.IntelSystem{system},
		Regions:   []int{system.Region},
		ChannelID: "zkill",
		Meta: map[string]any{
			"source": zkillReportSource,
			"zkill": map[string]any{
				"killmail_id":             event.KillmailID,
				"url":                     fmt.Sprintf("https://zkillboard.com/kill/%d/", event.KillmailID),
				"display_text":            displayText,
				"sequence_id":             event.SequenceID,
				"killmail_time":           event.KillmailTime.UTC().Format(time.RFC3339),
				"solo":                    event.Solo,
				"attacker_count":          event.AttackerCount,
				"solar_system_id":         event.SolarSystemID,
				"system_id":               system.System,
				"system_name":             system.Name,
				"region_id":               system.Region,
				"region_name":             regionName,
				"killer_name":             killerName,
				"killer_character_id":     finalBlow.CharacterID,
				"killer_alliance_id":      finalBlow.AllianceID,
				"killer_alliance_name":    killerAllianceName,
				"killer_corporation_id":   finalBlow.CorporationID,
				"killer_corporation_name": killerCorporationName,
				"victim_name":             victimName,
				"victim_character_id":     event.Victim.CharacterID,
				"victim_alliance_id":      event.Victim.AllianceID,
				"victim_alliance_name":    victimAllianceName,
				"victim_corporation_id":   event.Victim.CorporationID,
				"victim_corporation_name": victimCorporationName,
				"victim_ship_type_id":     event.Victim.ShipTypeID,
				"victim_ship_name":        shipName,
				"involved_attackers":      otherAttackers,
				"killer_hostility":        killerHostility,
				"victim_hostility":        victimHostility,
			},
		},
	}

	if _, publishErr := i.Topics.PublishJSON(regionTopic, report); publishErr != nil {
		return false, publishErr
	}
	return true, nil
}

func selectFinalBlowAttacker(attackers []rawAttacker) rawAttacker {
	for _, attacker := range attackers {
		if attacker.FinalBlow {
			return attacker
		}
	}

	if len(attackers) > 0 {
		return attackers[0]
	}
	return rawAttacker{}
}

func countPlayerAttackers(attackers []rawAttacker) int {
	seen := map[int]struct{}{}
	for _, attacker := range attackers {
		if attacker.CharacterID <= 0 {
			continue
		}
		seen[attacker.CharacterID] = struct{}{}
	}
	return len(seen)
}

func otherAttackerCount(event *killmailEvent) int {
	if event == nil || event.Solo {
		return 0
	}

	if event.AttackerCount > 1 {
		return event.AttackerCount - 1
	}
	playerCount := countPlayerAttackers(event.Attackers)
	if playerCount > 1 {
		return playerCount - 1
	}
	return 0
}

func isLowValueCapsuleKill(event *killmailEvent) bool {
	if event == nil {
		return false
	}

	if event.Victim.ShipTypeID != capsuleShipTypeID {
		return false
	}

	if event.TotalValue == nil {
		return false
	}
	return *event.TotalValue <= lowValueCapsuleISKMax
}

func isJSpaceSystemName(name string) bool {
	clean := strings.TrimSpace(name)
	if len(clean) != jSpaceSystemNameLength {
		return false
	}

	if clean[0] != 'J' && clean[0] != 'j' {
		return false
	}
	for i := 1; i < len(clean); i++ {
		if clean[i] < '0' || clean[i] > '9' {
			return false
		}
	}
	return true
}

func (i *FeedIngestor) loadSystems() error {
	records, err := i.App.FindRecordsByFilter(store.CollectionSolarSystems, "", "eve_id", 0, 0, nil)
	if err != nil {
		return err
	}
	next := make(map[int]intel.IntelSystem, len(records))
	for _, record := range records {
		eid := record.GetInt("eve_id")
		if eid <= 0 {
			continue
		}
		next[eid] = intel.IntelSystem{
			System:        eid,
			Name:          record.GetString("name"),
			Constellation: record.GetInt("constellation"),
			Region:        record.GetInt("region_id"),
		}
	}
	i.systemsByID = next
	return nil
}

func (i *FeedIngestor) loadItemTypes() error {
	records, err := i.App.FindRecordsByFilter(store.CollectionItemTypes, "", "eve_id", 0, 0, nil)
	if err != nil {
		return err
	}
	next := make(map[int]string, len(records))
	for _, record := range records {
		typeID := record.GetInt("eve_id")
		if typeID <= 0 {
			continue
		}
		name := strings.TrimSpace(record.GetString("name"))
		if name == "" {
			continue
		}
		next[typeID] = name
	}
	i.itemTypes = next
	return nil
}

func (i *FeedIngestor) loadRegionNames() error {
	records, err := i.App.FindRecordsByFilter(store.CollectionRegions, "", "eve_id", 0, 0, nil)
	if err != nil {
		return err
	}
	next := make(map[int]string, len(records))
	for _, record := range records {
		regionID := record.GetInt("eve_id")
		if regionID <= 0 {
			continue
		}
		name := strings.TrimSpace(record.GetString("name"))
		if name == "" {
			continue
		}
		next[regionID] = name
	}
	i.regionNames = next
	return nil
}

func (i *FeedIngestor) shipTypeName(shipTypeID int) string {
	if shipTypeID <= 0 {
		return ""
	}

	if i == nil || i.itemTypes == nil {
		return ""
	}
	return strings.TrimSpace(i.itemTypes[shipTypeID])
}

func (i *FeedIngestor) refreshStandingsIfStale() error {
	if time.Since(i.standings.loadedAt) < standingsReloadInterval {
		return nil
	}
	alliances, err := i.loadAllowedSet(store.CollectionAllowedAlliances)
	if err != nil {
		return err
	}
	corporations, err := i.loadAllowedSet(store.CollectionAllowedCorporations)
	if err != nil {
		return err
	}
	orgAlliances, orgCorporations, err := i.loadOrganizationStandings()
	if err != nil {
		return err
	}
	maps.Copy(alliances, orgAlliances)
	maps.Copy(corporations, orgCorporations)
	i.standings = standingsCache{
		loadedAt:     time.Now().UTC(),
		alliances:    alliances,
		corporations: corporations,
	}
	return nil
}

func (i *FeedIngestor) loadAllowedSet(collection string) (map[int]string, error) {
	records, err := i.App.FindRecordsByFilter(collection, "", "eve_id", 0, 0, nil)
	if err != nil {
		return nil, err
	}
	out := make(map[int]string, len(records))
	for _, record := range records {
		id := record.GetInt("eve_id")
		if id > 0 {
			out[id] = standingOurs
		}
	}
	return out, nil
}

func (i *FeedIngestor) loadOrganizationStandings() (alliances, corporations map[int]string, err error) {
	records, err := i.App.FindRecordsByFilter(store.CollectionOrganizationStandings, "", "", 0, 0, nil)
	if err != nil {
		return nil, nil, err
	}
	alliances = make(map[int]string, len(records))
	corporations = make(map[int]string, len(records))
	for _, record := range records {
		standing := normalizeStanding(record.GetString("hostility"))
		allianceID := record.GetInt("alliance_id")
		if allianceID > 0 {
			alliances[allianceID] = standing
		}
		corporationID := record.GetInt("corporation_id")
		if corporationID > 0 {
			corporations[corporationID] = standing
		}
	}
	return alliances, corporations, nil
}

func normalizeStanding(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case standingOurs:
		return standingOurs
	case standingFriendly:
		return standingFriendly
	case standingComplicated:
		return standingComplicated
	case standingHostile:
		return standingHostile
	default:
		return standingNeutral
	}
}

func (i *FeedIngestor) resolveStanding(corporationID, allianceID int) string {
	if corporationID > 0 {
		if standing, ok := i.standings.corporations[corporationID]; ok {
			return standing
		}
	}

	if allianceID > 0 {
		if standing, ok := i.standings.alliances[allianceID]; ok {
			return standing
		}
	}
	return standingNeutral
}

func (i *FeedIngestor) loadCheckpoint() (sequenceID int64, found bool, err error) {
	records, err := i.App.FindRecordsByFilter(
		store.CollectionZKillFeedState,
		"key = {:key}",
		"",
		1,
		0,
		dbx.Params{"key": stateKeyMain},
	)
	if err != nil {
		return 0, false, err
	}

	if len(records) == 0 {
		return 0, false, nil
	}
	return int64(records[0].GetInt("sequence_id")), true, nil
}

func (i *FeedIngestor) saveCheckpoint(sequenceID int64) error {
	records, err := i.App.FindRecordsByFilter(
		store.CollectionZKillFeedState,
		"key = {:key}",
		"",
		1,
		0,
		dbx.Params{"key": stateKeyMain},
	)
	if err != nil {
		return err
	}
	collection, err := i.App.FindCollectionByNameOrId(store.CollectionZKillFeedState)
	if err != nil {
		return err
	}
	var record *core.Record
	if len(records) > 0 {
		record = records[0]
	} else {
		record = core.NewRecord(collection)
		record.Set("key", stateKeyMain)
	}
	record.Set("sequence_id", sequenceID)
	updatedAt, _ := types.ParseDateTime(time.Now().UTC())
	record.Set("updated_at", updatedAt)
	return i.App.Save(record)
}

func newKillmailLRU(limit int) *killmailLRU {
	if limit <= 0 {
		limit = 1
	}
	return &killmailLRU{
		order: list.New(),
		seen:  make(map[int64]*list.Element, limit),
		limit: limit,
	}
}

func (l *killmailLRU) Seen(killmailID int64) bool {
	if l == nil || killmailID <= 0 {
		return false
	}
	_, ok := l.seen[killmailID]
	return ok
}

func (l *killmailLRU) Add(killmailID int64) {
	if l == nil || killmailID <= 0 {
		return
	}

	if existing, ok := l.seen[killmailID]; ok {
		l.order.MoveToBack(existing)
		return
	}
	element := l.order.PushBack(killmailID)
	l.seen[killmailID] = element
	if l.order.Len() <= l.limit {
		return
	}
	oldest := l.order.Front()
	if oldest == nil {
		return
	}
	oldestID, _ := oldest.Value.(int64)
	delete(l.seen, oldestID)
	l.order.Remove(oldest)
}

//nolint:gocognit // combines dedupe, cache hit, and one-shot ESI lookup.
func (r *nameResolver) ResolveNames(ctx context.Context, ids []int64) (map[int64]string, error) {
	if r == nil || r.client == nil || len(ids) == 0 {
		return map[int64]string{}, nil
	}

	unique := make([]int64, 0, len(ids))
	seen := map[int64]struct{}{}
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	slices.Sort(unique)

	result := make(map[int64]string, len(unique))
	missing := make([]int64, 0)

	r.mu.RLock()
	for _, id := range unique {
		if value, ok := r.cache[id]; ok && strings.TrimSpace(value) != "" {
			result[id] = value
		} else {
			missing = append(missing, id)
		}
	}
	r.mu.RUnlock()

	if len(missing) == 0 {
		return result, nil
	}

	requestCtx, cancel := context.WithTimeout(ctx, nameLookupRequestTimeout)
	defer cancel()
	response, httpResp, err := r.client.UniverseAPI.PostUniverseNames(requestCtx).RequestBody(missing).Execute()
	if httpResp != nil {
		_ = httpResp.Body.Close()
	}

	if err != nil {
		return result, err
	}
	r.mu.Lock()
	for _, item := range response {
		name := strings.TrimSpace(item.GetName())
		if name == "" {
			continue
		}
		r.cache[item.GetId()] = name
		result[item.GetId()] = name
	}
	r.mu.Unlock()
	return result, nil
}
