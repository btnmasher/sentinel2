package timers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	goesi "github.com/fnt-eve/goesi-openapi"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"gopkg.in/yaml.v3"

	esipkg "sentinel2/internal/esi"
	"sentinel2/internal/intel"
	"sentinel2/internal/store"
)

const (
	esiNotificationsScope           = "esi-characters.read_notifications.v1"
	skyhookIntelAuthor              = "System Skyhook Alert"
	skyhookLostShieldsIntelAuthor   = "System Skyhook Vulnerability"
	skyhookNotificationSourceName   = "esi"
	skyhookNotificationSourcePrefix = "esi:notification:"
	skyhookUnderAttackType          = "SkyhookUnderAttack"
	skyhookLostShieldsType          = "SkyhookLostShields"
	orbitalReinforcedType           = "OrbitalReinforced"

	notificationRRCursorKeyPrefix = "skyhook_notification_rr_cursor"
	notificationETagKeyPrefix     = "skyhook_notification_etag"

	notificationCoverageWindow = 10 * time.Minute
	notificationMinInterval    = 30 * time.Second
	defaultSinceWindow         = 2 * time.Minute
	skyhookSampleLimit         = 5
	skyhookSampleMaxText       = 120
	milliUnixThreshold         = int64(1000000000000)
)

type SkyhookNotificationSyncResult struct {
	WatchedCharacters int
	NotificationsSeen int
	IntelCreated      int
	TimersCreated     int
	TimersUpdated     int
	Skipped           int
}

type skyhookSyncDelta struct {
	seen          int
	intelCreated  int
	timersCreated int
	timersUpdated int
	skipped       int
}

type notificationSelection struct {
	watched  int
	selected []notificationSource
}

func (s *Service) SyncSkyhookNotifications(ctx context.Context, sinceWindow time.Duration) (SkyhookNotificationSyncResult, error) {
	result := SkyhookNotificationSyncResult{}
	if s == nil || s.App == nil || s.ESI == nil {
		return result, ErrESIClientNotConfigured
	}
	sinceWindow = resolvedSinceWindow(sinceWindow)
	cutoff := time.Now().UTC().Add(-sinceWindow)
	selection, err := s.selectNotificationSources(sinceWindow)
	if err != nil {
		return result, err
	}
	result.WatchedCharacters = selection.watched
	if len(selection.selected) == 0 {
		return result, nil
	}

	timersCollection, err := s.App.FindCollectionByNameOrId(store.CollectionTimers)
	if err != nil {
		return result, err
	}

	for _, source := range selection.selected {
		delta, stop, sourceErr := s.syncNotificationSource(ctx, timersCollection, source, cutoff)
		result.applyDelta(delta)
		if sourceErr != nil {
			return result, sourceErr
		}
		if stop {
			break
		}
	}
	return result, nil
}

func resolvedSinceWindow(sinceWindow time.Duration) time.Duration {
	if sinceWindow <= 0 {
		return defaultSinceWindow
	}
	return sinceWindow
}

type notificationSource struct {
	CharacterID int
	AccessToken string
}

func (r *SkyhookNotificationSyncResult) applyDelta(delta skyhookSyncDelta) {
	r.NotificationsSeen += delta.seen
	r.IntelCreated += delta.intelCreated
	r.TimersCreated += delta.timersCreated
	r.TimersUpdated += delta.timersUpdated
	r.Skipped += delta.skipped
}

func (s *Service) selectNotificationSources(sinceWindow time.Duration) (notificationSelection, error) {
	out := notificationSelection{}
	eligible, err := s.eligibleNotificationSources()
	if err != nil {
		return out, err
	}
	out.watched = len(eligible)
	if len(eligible) == 0 {
		return out, nil
	}

	perRequestInterval := notificationCoverageWindow / time.Duration(len(eligible))
	perRequestInterval = max(notificationMinInterval, perRequestInterval)
	requestsPerRun := int(sinceWindow / perRequestInterval)
	requestsPerRun = min(len(eligible), max(1, requestsPerRun))

	cursor, _ := s.getSyncMetaInt(notificationRRCursorKey())
	selected, nextCursor := roundRobinSelection(eligible, cursor, requestsPerRun)
	_ = s.saveSyncMetaInt(notificationRRCursorKey(), nextCursor)
	out.selected = selected
	return out, nil
}

func (s *Service) syncNotificationSource(
	ctx context.Context,
	timersCollection *core.Collection,
	source notificationSource,
	cutoff time.Time,
) (skyhookSyncDelta, bool, error) {
	delta := skyhookSyncDelta{}

	select {
	case <-ctx.Done():
		return delta, false, ctx.Err()
	default:
	}

	notifications, rateLimited, err := s.fetchCharacterNotifications(ctx, source)
	if err != nil {
		delta.skipped++
		if rateLimited {
			return delta, true, nil
		}
		return delta, false, nil
	}
	for _, notification := range notifications {
		update, processErr := s.processNotification(ctx, timersCollection, notification, cutoff)
		delta.seen += update.seen
		delta.intelCreated += update.intelCreated
		delta.timersCreated += update.timersCreated
		delta.timersUpdated += update.timersUpdated
		if processErr != nil {
			delta.skipped++
		}
	}
	return delta, false, nil
}

func (s *Service) fetchCharacterNotifications(
	ctx context.Context,
	source notificationSource,
) ([]esipkg.CharacterNotification, bool, error) {
	etagKey := notificationETagKey(source.CharacterID)
	priorETag, _ := s.getSyncMeta(etagKey)
	notifications, nextETag, notModified, err := s.ESI.CharacterNotifications(ctx, source.CharacterID, source.AccessToken, priorETag)
	if err != nil {
		return nil, errors.Is(err, esipkg.ErrRateLimited), err
	}
	if nextETag != "" && nextETag != priorETag {
		_ = s.saveSyncMeta(etagKey, nextETag)
	}
	if notModified {
		return []esipkg.CharacterNotification{}, false, nil
	}
	s.logSkyhookNotificationFetch(source.CharacterID, notifications)
	return notifications, false, nil
}

func (s *Service) processNotification(
	ctx context.Context,
	timersCollection *core.Collection,
	notification esipkg.CharacterNotification,
	cutoff time.Time,
) (skyhookSyncDelta, error) {
	delta := skyhookSyncDelta{seen: 1}
	if notification.Timestamp.UTC().Before(cutoff) {
		return delta, nil
	}

	var processErr error
	switch notification.Type {
	case skyhookUnderAttackType:
		processErr = s.applySkyhookUnderAttackNotification(&delta, notification)
	case skyhookLostShieldsType:
		processErr = s.applySkyhookLostShieldsNotification(&delta, notification)
	case orbitalReinforcedType:
		processErr = s.applyOrbitalReinforcedNotification(ctx, timersCollection, &delta, notification)
	default:
		return delta, nil
	}
	return delta, processErr
}

func (s *Service) applySkyhookUnderAttackNotification(delta *skyhookSyncDelta, notification esipkg.CharacterNotification) error {
	created, err := s.createSkyhookIntelReport(notification)
	if err != nil {
		return err
	}
	if created {
		delta.intelCreated++
	}
	return nil
}

func (s *Service) applySkyhookLostShieldsNotification(delta *skyhookSyncDelta, notification esipkg.CharacterNotification) error {
	created, err := s.createSkyhookLostShieldsIntelReport(notification)
	if err != nil {
		return err
	}
	if created {
		delta.intelCreated++
	}
	return nil
}

func (s *Service) applyOrbitalReinforcedNotification(
	ctx context.Context,
	timersCollection *core.Collection,
	delta *skyhookSyncDelta,
	notification esipkg.CharacterNotification,
) error {
	created, updated, err := s.upsertOrbitalReinforcementTimer(ctx, timersCollection, notification)
	if err != nil {
		return err
	}
	if created {
		delta.timersCreated++
	}
	if updated {
		delta.timersUpdated++
	}
	return nil
}

func (s *Service) NotificationSourceCount() (int, error) {
	eligible, err := s.eligibleNotificationSources()
	if err != nil {
		return 0, err
	}
	return len(eligible), nil
}

func (s *Service) eligibleNotificationSources() ([]notificationSource, error) {
	allowedAllianceIDs, err := s.allowedAllianceIDSet()
	if err != nil {
		return nil, err
	}
	if len(allowedAllianceIDs) == 0 {
		return []notificationSource{}, nil
	}

	records, err := s.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"esi_token_valid = true && oauth_access_token != '' && oauth_scopes ~ {:scope}",
		"eve_character_id",
		0,
		0,
		dbx.Params{"scope": "%" + esiNotificationsScope + "%"},
	)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]notificationSource, 0, len(records))
	for _, record := range records {
		allianceID := record.GetInt("eve_alliance_id")
		if allianceID <= 0 {
			continue
		}
		if _, ok := allowedAllianceIDs[allianceID]; !ok {
			continue
		}
		expiresAt := record.GetDateTime("oauth_access_expires_at")
		if expiresAt.IsZero() || expiresAt.Time().Before(now) {
			continue
		}
		characterID := record.GetInt("eve_character_id")
		token := strings.TrimSpace(record.GetString("oauth_access_token"))
		if characterID <= 0 || token == "" {
			continue
		}
		out = append(out, notificationSource{
			CharacterID: characterID,
			AccessToken: token,
		})
	}
	return out, nil
}

func (s *Service) allowedAllianceIDSet() (map[int]struct{}, error) {
	records, err := s.App.FindRecordsByFilter(store.CollectionAllowedAlliances, "", "", 0, 0, nil)
	if err != nil {
		return nil, err
	}
	out := make(map[int]struct{}, len(records))
	for _, record := range records {
		allianceID := record.GetInt("eve_id")
		if allianceID > 0 {
			out[allianceID] = struct{}{}
		}
	}
	return out, nil
}

func roundRobinSelection(items []notificationSource, cursor, count int) (selected []notificationSource, nextCursor int) {
	if len(items) == 0 || count <= 0 {
		return []notificationSource{}, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	cursor %= len(items)
	selected = make([]notificationSource, 0, count)
	for i := range count {
		selected = append(selected, items[(cursor+i)%len(items)])
	}
	nextCursor = (cursor + count) % len(items)
	return selected, nextCursor
}

func notificationRRCursorKey() string {
	return notificationRRCursorKeyPrefix
}

func notificationETagKey(characterID int) string {
	return notificationETagKeyPrefix + ":" + strconv.Itoa(characterID)
}

func (s *Service) getSyncMeta(key string) (string, error) {
	records, err := s.App.FindRecordsByFilter(store.CollectionSDEMeta, "key = {:key}", "", 1, 0, dbx.Params{"key": key})
	if err != nil || len(records) == 0 {
		return "", err
	}
	return strings.TrimSpace(records[0].GetString("value")), nil
}

func (s *Service) saveSyncMeta(key, value string) error {
	collection, err := s.App.FindCollectionByNameOrId(store.CollectionSDEMeta)
	if err != nil {
		return err
	}
	records, findErr := s.App.FindRecordsByFilter(store.CollectionSDEMeta, "key = {:key}", "", 1, 0, dbx.Params{"key": key})
	if findErr != nil {
		return findErr
	}
	var record *core.Record
	if len(records) > 0 {
		record = records[0]
	} else {
		record = core.NewRecord(collection)
		record.Set("key", key)
	}
	record.Set("value", value)
	record.Set("updated_at", types.NowDateTime())
	return s.App.Save(record)
}

func (s *Service) getSyncMetaInt(key string) (int, error) {
	raw, err := s.getSyncMeta(key)
	if raw == "" || err != nil {
		return 0, err
	}
	value, parseErr := strconv.Atoi(raw)
	if parseErr != nil {
		return 0, parseErr
	}
	return value, nil
}

func (s *Service) saveSyncMetaInt(key string, value int) error {
	return s.saveSyncMeta(key, strconv.Itoa(max(0, value)))
}

func (s *Service) logSkyhookNotificationFetch(characterID int, notifications []esipkg.CharacterNotification) {
	if s == nil || s.App == nil {
		return
	}
	samples := make([]string, 0, min(skyhookSampleLimit, len(notifications)))
	for i, notification := range notifications {
		if i >= skyhookSampleLimit {
			break
		}
		trimmed := strings.TrimSpace(notification.Text)
		if len(trimmed) > skyhookSampleMaxText {
			trimmed = trimmed[:skyhookSampleMaxText] + "..."
		}
		samples = append(samples, fmt.Sprintf("%s:%d@%s text=%q", notification.Type, notification.ID, notification.Timestamp.UTC().Format(time.RFC3339), trimmed))
	}
	s.App.Logger().Debug(
		"skyhook notifications fetched",
		slog.Int("character_id", characterID),
		slog.Int("count", len(notifications)),
		slog.Any("samples", samples),
	)
}

func (s *Service) createSkyhookIntelReport(notification esipkg.CharacterNotification) (bool, error) {
	if s == nil || s.App == nil {
		return false, fmt.Errorf("timers service not configured")
	}
	existing, err := s.App.FindRecordsByFilter(
		store.CollectionIntelReports,
		"report_id = {:id} && author = {:author}",
		"",
		1,
		0,
		dbx.Params{
			"id":     notification.ID,
			"author": skyhookIntelAuthor,
		},
	)
	if err != nil {
		return false, err
	}
	if len(existing) > 0 {
		return false, nil
	}

	payload := goesi.SkyhookUnderAttack{}
	if parseErr := yaml.Unmarshal([]byte(notification.Text), &payload); parseErr != nil {
		return false, parseErr
	}
	systemID := int(payload.SolarsystemID)
	if systemID <= 0 {
		return false, fmt.Errorf("missing system in skyhook notification")
	}

	system, resolveErr := s.ResolveSystem(systemID, "")
	if resolveErr != nil || system == nil {
		return false, resolveErr
	}
	systemName := system.GetString("name")
	intelReport := intel.IntelReport{
		ID:     notification.ID,
		Time:   notification.Timestamp.UTC().Unix(),
		Author: skyhookIntelAuthor,
		Text: fmt.Sprintf(
			"Skyhook under attack in %s (shield %.0f%% armor %.0f%% hull %.0f%%)",
			systemName,
			payload.ShieldPercentage,
			payload.ArmorPercentage,
			payload.HullPercentage,
		),
		Systems: []intel.IntelSystem{
			{
				System:        systemID,
				Name:          systemName,
				Constellation: system.GetInt("constellation"),
				Region:        system.GetInt("region_id"),
			},
		},
		Regions: []int{system.GetInt("region_id")},
	}
	if createErr := intel.NewIntelService(s.App).CreateReport(&intelReport); createErr != nil {
		return false, createErr
	}
	return true, nil
}

func (s *Service) createSkyhookLostShieldsIntelReport(notification esipkg.CharacterNotification) (bool, error) {
	if s == nil || s.App == nil {
		return false, fmt.Errorf("timers service not configured")
	}
	existing, err := s.App.FindRecordsByFilter(
		store.CollectionIntelReports,
		"report_id = {:id} && author = {:author}",
		"",
		1,
		0,
		dbx.Params{
			"id":     notification.ID,
			"author": skyhookLostShieldsIntelAuthor,
		},
	)
	if err != nil {
		return false, err
	}
	if len(existing) > 0 {
		return false, nil
	}

	payload := goesi.SkyhookLostShields{}
	if parseErr := yaml.Unmarshal([]byte(notification.Text), &payload); parseErr != nil {
		return false, parseErr
	}
	systemID := int(payload.SolarsystemID)
	if systemID <= 0 {
		return false, fmt.Errorf("missing system in skyhook lost shields notification")
	}
	system, resolveErr := s.ResolveSystem(systemID, "")
	if resolveErr != nil || system == nil {
		return false, resolveErr
	}
	systemName := system.GetString("name")
	intelReport := intel.IntelReport{
		ID:     notification.ID,
		Time:   notification.Timestamp.UTC().Unix(),
		Author: skyhookLostShieldsIntelAuthor,
		Text: fmt.Sprintf(
			"Skyhook lost shields in %s (vulnerable at %s)",
			systemName,
			unixToTime(payload.VulnerableTime).UTC().Format(time.RFC3339),
		),
		Systems: []intel.IntelSystem{
			{
				System:        systemID,
				Name:          systemName,
				Constellation: system.GetInt("constellation"),
				Region:        system.GetInt("region_id"),
			},
		},
		Regions: []int{system.GetInt("region_id")},
	}
	if createErr := intel.NewIntelService(s.App).CreateReport(&intelReport); createErr != nil {
		return false, createErr
	}
	return true, nil
}

func (s *Service) upsertOrbitalReinforcementTimer(ctx context.Context, timersCollection *core.Collection, notification esipkg.CharacterNotification) (created, updated bool, err error) {
	payload := goesi.OrbitalReinforced{}
	if parseErr := yaml.Unmarshal([]byte(notification.Text), &payload); parseErr != nil {
		return false, false, parseErr
	}
	expiresAt := unixToTime(payload.ReinforceExitTime)
	if expiresAt.IsZero() {
		return false, false, fmt.Errorf("orbital reinforced notification missing reinforce exit time")
	}
	return s.upsertSkyhookNotificationTimer(
		ctx,
		timersCollection,
		notification,
		int(payload.SolarSystemID),
		int(payload.PlanetID),
		int(payload.TypeID),
		expiresAt,
		int(payload.AggressorCorpID),
		int(payload.AggressorAllianceID),
	)
}

func (s *Service) upsertSkyhookNotificationTimer(
	ctx context.Context,
	timersCollection *core.Collection,
	notification esipkg.CharacterNotification,
	systemID,
	planetID,
	typeID int,
	expiresAt time.Time,
	fallbackCorpID,
	fallbackAllianceID int,
) (created, updated bool, err error) {
	system, resolveErr := s.ResolveSystem(systemID, "")
	if resolveErr != nil || system == nil {
		return false, false, resolveErr
	}
	owner := s.resolveSkyhookOwnerFromNotification(ctx, fallbackCorpID, fallbackAllianceID)

	sourceRef := skyhookNotificationSourcePrefix + strconv.FormatInt(notification.ID, 10)
	input := &CreateInput{
		Title:                  fmt.Sprintf("%s Skyhook Reinforcement", system.GetString("name")),
		SystemID:               systemID,
		Standing:               TimerStandingOurs,
		TimerKind:              TimerKindReinforcement,
		StructureType:          TimerStructureOrbitalSkyhook,
		StageLabel:             TimerStageReinforcement,
		PlanetID:               planetID,
		OwnerCorporationID:     owner.CorporationID,
		OwnerCorporationName:   owner.CorporationName,
		OwnerCorporationTicker: owner.CorporationTicker,
		OwnerAllianceID:        owner.AllianceID,
		OwnerAllianceName:      owner.AllianceName,
		OwnerAllianceTicker:    owner.AllianceTicker,
		Stage:                  1,
		TotalStages:            1,
		Severity:               TimerSeverityCritical,
		Status:                 timerStatusActive,
		ExpiresAt:              expiresAt.UTC(),
		Source:                 skyhookNotificationSourceName,
		SourceRef:              sourceRef,
		Notes:                  fmt.Sprintf("From %s notification", notification.Type),
		RawText:                strings.TrimSpace(notification.Text),
		ReplacementAction:      "alliance_replacement",
	}
	if typeID > 0 {
		input.Notes = fmt.Sprintf("%s | structure_type_id=%d", input.Notes, typeID)
	}

	desired := core.NewRecord(timersCollection)
	s.applyCreateInput(desired, system, input, nil)
	desired.Set("canceled_at", nil)
	desired.Set("canceled_by", nil)

	existing, err := s.App.FindRecordsByFilter(
		store.CollectionTimers,
		"source = {:source} && source_ref = {:source_ref}",
		"",
		1,
		0,
		dbx.Params{
			"source":     skyhookNotificationSourceName,
			"source_ref": sourceRef,
		},
	)
	if err != nil {
		return false, false, err
	}
	if len(existing) == 0 {
		if saveErr := s.App.Save(desired); saveErr != nil {
			return false, false, saveErr
		}
		return true, false, nil
	}
	if !applyTimerRecordFromSource(existing[0], desired) {
		return false, false, nil
	}
	if saveErr := s.App.Save(existing[0]); saveErr != nil {
		return false, false, saveErr
	}
	return false, true, nil
}

type skyhookOwner struct {
	CorporationID     int
	CorporationName   string
	CorporationTicker string
	AllianceID        int
	AllianceName      string
	AllianceTicker    string
}

func (s *Service) resolveSkyhookOwnerFromNotification(
	ctx context.Context,
	fallbackCorpID,
	fallbackAllianceID int,
) skyhookOwner {
	out := skyhookOwner{
		CorporationID: fallbackCorpID,
		AllianceID:    fallbackAllianceID,
	}
	if s == nil || s.App == nil {
		return out
	}
	if out.CorporationID > 0 {
		out.CorporationName, out.CorporationTicker = s.resolveCorporationNameTicker(ctx, out.CorporationID)
	}
	if out.AllianceID > 0 {
		out.AllianceName, out.AllianceTicker = s.resolveAllianceNameTicker(ctx, out.AllianceID, sovWatchlist{})
	}
	return out
}

func unixToTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	// Some payloads are seconds, others may be milliseconds.
	if value > milliUnixThreshold {
		return time.UnixMilli(value).UTC()
	}
	return time.Unix(value, 0).UTC()
}

func (s *Service) resolveCorporationNameTicker(ctx context.Context, corporationID int) (name, ticker string) {
	if corporationID <= 0 {
		return "", ""
	}
	if name, ticker, ok := store.GetOrg(s.App, store.CollectionCorporations, corporationID); ok {
		return name, ticker
	}
	if s.PublicESI == nil {
		return "", ""
	}
	name, ticker, allianceID, err := s.PublicESI.CorporationDetails(ctx, corporationID)
	if err != nil {
		return "", ""
	}
	_ = store.UpsertOrg(s.App, store.CollectionCorporations, corporationID, name, ticker)
	if allianceID > 0 {
		allianceName, allianceTicker := s.resolveAllianceNameTicker(ctx, allianceID, sovWatchlist{})
		if allianceName != "" {
			_ = store.UpsertOrg(s.App, store.CollectionAlliances, allianceID, allianceName, allianceTicker)
		}
	}
	return name, ticker
}
