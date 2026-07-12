package auth

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/audit"
	"sentinel2/internal/esi"
	"sentinel2/internal/intel"
	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

type CharacterRefresher struct {
	App       *pocketbase.PocketBase
	EVE       *EVEProvider
	ESI       esi.ESIClient
	PublicESI *esi.ESIPublicClient
	Intel     *intel.IntelService
	Audit     *audit.Service
	logger    *logging.Logger
}

type throttleDelayProvider interface {
	ThrottleDelay() time.Duration
}

type refreshJobMeta struct {
	Trigger string
	ActorID string
}

type refreshJobMetaKey struct{}

const (
	defaultRefreshBatchSize        = 25
	defaultRefreshPause            = 350 * time.Millisecond
	defaultRefreshRetries          = 2
	defaultRefreshBackoffMs        = 250
	maxRefreshJitterMs      uint32 = 800
)

func WithRefreshJobMeta(ctx context.Context, trigger, actorID string) context.Context {
	return context.WithValue(ctx, refreshJobMetaKey{}, refreshJobMeta{
		Trigger: trigger,
		ActorID: actorID,
	})
}

func getRefreshJobMeta(ctx context.Context) refreshJobMeta {
	value := ctx.Value(refreshJobMetaKey{})
	meta, ok := value.(refreshJobMeta)
	if !ok {
		return refreshJobMeta{}
	}
	return meta
}

func NewCharacterRefresher(app *pocketbase.PocketBase, eve *EVEProvider, esiClient esi.ESIClient, publicESI *esi.ESIPublicClient, intelService *intel.IntelService, auditSvc *audit.Service) *CharacterRefresher {
	return &CharacterRefresher{
		App:       app,
		EVE:       eve,
		ESI:       esiClient,
		PublicESI: publicESI,
		Intel:     intelService,
		Audit:     auditSvc,
		logger: logging.New(app).WithFields(logging.Fields{
			"component": "auth.character_refresh",
		}),
	}
}

func (r *CharacterRefresher) RefreshAll(ctx context.Context) (success, failed int) {
	if r.App == nil {
		return 0, 0
	}

	records, recordsErr := r.App.FindRecordsByFilter(
		store.CollectionCharacters,
		"auth_provider = {:provider}",
		"",
		0,
		0,
		map[string]any{"provider": AuthProviderEVE},
	)
	if recordsErr != nil {
		r.logger.
			WithErr(recordsErr).
			Warn("character refresh failed to load records")
		return 0, 0
	}

	meta := getRefreshJobMeta(ctx)
	if meta.Trigger == "" {
		ctx = WithRefreshJobMeta(ctx, jobs.TriggerCronSchedule, "")
	}
	jitter := refreshJitter()
	return r.RefreshAllBatched(ctx, records, defaultRefreshBatchSize, defaultRefreshPause+jitter)
}

func (r *CharacterRefresher) RefreshAllBatched(ctx context.Context, records []*core.Record, batchSize int, pause time.Duration) (success, failed int) {
	if batchSize <= 0 {
		batchSize = defaultRefreshBatchSize
	}

	if pause < 0 {
		pause = 0
	}

	for i, record := range records {
		if ctx.Err() != nil {
			return success, failed
		}

		if err := r.refreshOneRecord(ctx, record); err != nil {
			failed++
		} else {
			success++
		}

		waitErr := r.waitAtBatchBoundary(ctx, i+1, batchSize, pause)
		if waitErr != nil {
			return success, failed
		}
	}
	return success, failed
}

func (r *CharacterRefresher) RefreshCharacter(ctx context.Context, character *core.Record) error {
	if character == nil {
		return fmt.Errorf("missing character")
	}

	if r.EVE == nil {
		return fmt.Errorf("eve provider unavailable")
	}

	userID, user := r.findCharacterUser(character)

	refreshToken := character.GetString("oauth_refresh_token")
	if refreshToken == "" {
		r.logger.
			WithFields(logging.Fields{
				"character_record_id": character.Id,
				"character_id":        character.GetInt("eve_character_id"),
			}).
			Warn("character refresh missing refresh token")
		return r.refreshAffiliationOnly(ctx, character, fmt.Errorf("missing refresh token"))
	}

	_, refreshErr := r.EVE.RefreshCharacter(ctx, user, character)
	if refreshErr != nil {
		if errors.Is(refreshErr, esi.ErrNotModified) {
			r.logger.
				WithFields(logging.Fields{
					"character_record_id": character.Id,
					"character_id":        character.GetInt("eve_character_id"),
				}).
				Info("character refresh skipped: affiliation not modified")
			return nil
		}
		r.handleRefreshDenied(userID, character, refreshErr)
		return r.refreshAffiliationOnly(ctx, character, refreshErr)
	}
	return nil
}

func (r *CharacterRefresher) refreshOneRecord(ctx context.Context, record *core.Record) error {
	runner, started := r.tryStartRefreshJob(ctx, record)
	if !started {
		return nil
	}

	if runner == nil {
		return r.refreshWithRetry(ctx, record, defaultRefreshRetries)
	}
	var refreshErr error
	//nolint:contextcheck // runner.Run manages callback context.
	_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
		refreshErr = r.refreshWithRetry(ctx, record, defaultRefreshRetries)
		return refreshErr
	})
	return refreshErr
}

func (r *CharacterRefresher) waitAtBatchBoundary(ctx context.Context, index, batchSize int, pause time.Duration) error {
	if index%batchSize != 0 {
		return nil
	}
	waitFor := pause
	if provider, ok := r.ESI.(throttleDelayProvider); ok {
		if delay := provider.ThrottleDelay(); delay > waitFor {
			waitFor = delay
		}
	}

	if waitFor <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitFor):
		return nil
	}
}

func (r *CharacterRefresher) tryStartRefreshJob(ctx context.Context, record *core.Record) (*jobs.Runner, bool) {
	if r.App == nil || record == nil {
		return nil, true
	}

	step := "character:" + record.Id
	tracker := jobs.NewJobTracker(r.App)
	running, err := tracker.IsRunning(jobs.JobCharacterRefresh, step)
	if err == nil && running {
		r.logger.
			WithFields(logging.Fields{
				"character_record_id": record.Id,
			}).
			Info("character refresh already running; skipping")
		return nil, false
	}

	meta := getRefreshJobMeta(ctx)
	runner := jobs.NewRunner(r.App, &jobs.RunOptions{
		JobName: jobs.JobCharacterRefresh,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobCharacterRefresh,
			Step:    step,
			Trigger: meta.Trigger,
			ActorID: meta.ActorID,
			Hidden:  meta.Trigger == jobs.TriggerCronSchedule,
		},
		Parent:  ctx,
		Timeout: jobs.NoTimeout,
	})
	return runner, true
}

func (r *CharacterRefresher) refreshWithRetry(ctx context.Context, record *core.Record, retries int) error {
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		refreshErr := r.RefreshCharacter(ctx, record)
		if refreshErr == nil {
			return nil
		}

		lastErr = refreshErr
		if attempt < retries {
			backoff := time.Duration(defaultRefreshBackoffMs*(1<<attempt)) * time.Millisecond
			select {
			case <-ctx.Done():
				r.logger.
					WithFields(logging.Fields{
						"character_record_id": record.Id,
						"attempt":             attempt,
					}).
					WithErr(ctx.Err()).
					Warn(jobs.MessageCanceled)
				return ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}
	}
	r.logger.
		WithFields(logging.Fields{
			"character_record_id": record.Id,
		}).
		WithErr(lastErr).
		Warn("character refresh failed after retries")
	return lastErr
}

func refreshJitter() time.Duration {
	var buf [4]byte
	if _, err := cryptorand.Read(buf[:]); err != nil {
		return 0
	}
	jitterMs := binary.LittleEndian.Uint32(buf[:]) % maxRefreshJitterMs
	return time.Duration(jitterMs) * time.Millisecond
}

func (r *CharacterRefresher) findCharacterUser(character *core.Record) (string, *core.Record) {
	if character == nil {
		return "", nil
	}
	userID := character.GetString("user")
	if userID == "" {
		return "", nil
	}
	user, userErr := r.App.FindRecordById(store.CollectionUsers, userID)
	if userErr != nil {
		return userID, nil
	}
	return userID, user
}

func (r *CharacterRefresher) handleRefreshDenied(userID string, character *core.Record, refreshErr error) {
	if userID == "" || !errors.Is(refreshErr, ErrAccessDenied) {
		return
	}

	if r.Intel != nil {
		_ = r.Intel.RevokeUploaderTokensForUser(userID)
	}

	if r.Audit != nil {
		r.Audit.LogEvent(&audit.Event{
			Action:          audit.ActionUserRevokeUploadTokens,
			Summary:         "Revoked uploader tokens (allowlist)",
			TargetUserID:    userID,
			TargetCharacter: character,
		})
	}
	r.logger.
		WithFields(logging.Fields{
			"user_id":             userID,
			"character_record_id": character.Id,
			"character_id":        character.GetInt("eve_character_id"),
		}).
		Info("revoked uploader tokens due to access denied")
}

func (r *CharacterRefresher) refreshAffiliationOnly(ctx context.Context, character *core.Record, refreshErr error) error {
	charID := character.GetInt("eve_character_id")
	if charID > 0 {
		r.refreshCharacterAffiliation(ctx, character, charID)
	}

	refreshAt, _ := types.ParseDateTime(time.Now())
	character.Set("esi_last_refresh_at", refreshAt)
	character.Set("esi_token_valid", false)
	if refreshErr != nil {
		character.Set("esi_last_error", refreshErr.Error())
	}

	return r.App.Save(character)
}

func (r *CharacterRefresher) refreshCharacterAffiliation(ctx context.Context, character *core.Record, charID int) {
	var (
		corpID         int
		allianceID     int
		affiliationErr error
	)
	switch {
	case r != nil && r.PublicESI != nil:
		corpID, allianceID, affiliationErr = r.PublicESI.CharacterAffiliation(ctx, charID)
	case r != nil && r.ESI != nil:
		corpID, allianceID, affiliationErr = r.ESI.CharacterAffiliation(ctx, charID)
	default:
		affiliationErr = fmt.Errorf("missing esi client")
	}
	if affiliationErr != nil {
		r.logger.
			WithFields(logging.Fields{
				"character_record_id": character.Id,
				"character_id":        charID,
			}).
			WithErr(affiliationErr).
			Warn("character affiliation refresh failed")
		return
	}

	character.Set("eve_corporation_id", corpID)
	character.Set("eve_alliance_id", allianceID)
	if err := store.WarmCorporationCache(ctx, r.App, r.PublicESI, corpID); err != nil {
		r.logger.WithFields(logging.Fields{"corporation_id": corpID}).WithErr(err).Warn("failed to warm corporation cache")
	}

	if err := store.WarmAllianceCache(ctx, r.App, r.PublicESI, allianceID); err != nil {
		r.logger.WithFields(logging.Fields{"alliance_id": allianceID}).WithErr(err).Warn("failed to warm alliance cache")
	}
}
