package auth

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
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
}

type throttleDelayProvider interface {
	ThrottleDelay() time.Duration
}

type refreshJobMeta struct {
	Trigger string
	ActorID string
}

type refreshJobMetaKey struct{}

func WithRefreshJobMeta(ctx context.Context, trigger string, actorID string) context.Context {
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

func NewCharacterRefresher(app *pocketbase.PocketBase, eve *EVEProvider, esi esi.ESIClient, publicESI *esi.ESIPublicClient) *CharacterRefresher {
	return &CharacterRefresher{App: app, EVE: eve, ESI: esi, PublicESI: publicESI}
}

func (r *CharacterRefresher) RefreshAll(ctx context.Context) (int, int) {
	if r.App == nil {
		return 0, 0
	}

	records, recordsErr := r.App.FindRecordsByFilter(store.CollectionCharacters, "", "", 0, 0, nil)
	if recordsErr != nil {
		logging.New(r.App).
			WithErr(recordsErr).
			Warn("character refresh failed to load records")
		return 0, 0
	}

	meta := getRefreshJobMeta(ctx)
	if meta.Trigger == "" {
		ctx = WithRefreshJobMeta(ctx, jobs.TriggerCronSchedule, "")
	}
	jitter := time.Duration(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(800)) * time.Millisecond
	return r.RefreshAllBatched(ctx, records, 25, 350*time.Millisecond+jitter)
}

func (r *CharacterRefresher) RefreshAllBatched(ctx context.Context, records []*core.Record, batchSize int, pause time.Duration) (int, int) {
	success := 0
	failed := 0
	if batchSize <= 0 {
		batchSize = 25
	}
	if pause < 0 {
		pause = 0
	}

	for i, record := range records {
		if ctx.Err() != nil {
			return success, failed
		}

		runner, started := r.tryStartRefreshJob(ctx, record)
		if !started {
			continue
		}

		var refreshErr error
		if runner != nil {
			_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
				refreshErr = r.refreshWithRetry(ctx, record, 2)
				return refreshErr
			})
		} else {
			refreshErr = r.refreshWithRetry(ctx, record, 2)
		}
		if refreshErr != nil {
			failed++
		} else {
			success++
		}

		if (i+1)%batchSize == 0 {
			waitFor := pause
			if provider, ok := r.ESI.(throttleDelayProvider); ok {
				if delay := provider.ThrottleDelay(); delay > waitFor {
					waitFor = delay
				}
			}
			if waitFor > 0 {
				select {
				case <-ctx.Done():
					return success, failed
				case <-time.After(waitFor):
				}
			}
		}
	}
	return success, failed
}

func (r *CharacterRefresher) tryStartRefreshJob(ctx context.Context, record *core.Record) (*jobs.Runner, bool) {
	if r.App == nil || record == nil {
		return nil, true
	}

	step := "character:" + record.Id
	tracker := jobs.NewJobTracker(r.App)
	running, err := tracker.IsRunning(jobs.JobCharacterRefresh, step)
	if err == nil && running {
		logging.New(r.App).
			WithFields(logging.Fields{
				"character_record_id": record.Id,
			}).
			Info("character refresh already running; skipping")
		return nil, false
	}

	meta := getRefreshJobMeta(ctx)
	runner := jobs.NewRunner(r.App, jobs.RunOptions{
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
			backoff := time.Duration(250*(1<<attempt)) * time.Millisecond
			select {
			case <-ctx.Done():
				logging.New(r.App).
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
	logging.New(r.App).
		WithFields(logging.Fields{
			"character_record_id": record.Id,
		}).
		WithErr(lastErr).
		Warn("character refresh failed after retries")
	return lastErr
}

func (r *CharacterRefresher) RefreshCharacter(ctx context.Context, character *core.Record) error {
	if character == nil {
		return fmt.Errorf("missing character")
	}

	if r.EVE == nil {
		return fmt.Errorf("eve provider unavailable")
	}

	userID := character.GetString("user")
	var user *core.Record
	if userID != "" {
		u, userErr := r.App.FindRecordById(store.CollectionUsers, userID)
		if userErr == nil {
			user = u
		}
	}

	refreshToken := character.GetString("oauth_refresh_token")
	if refreshToken == "" {
		logging.New(r.App).
			WithFields(logging.Fields{
				"character_record_id": character.Id,
				"character_id":        character.GetInt("eve_character_id"),
			}).
			Warn("character refresh missing refresh token")
		return r.refreshAffiliationOnly(ctx, character, fmt.Errorf("missing refresh token"))
	}

	_, refreshErr := r.EVE.RefreshCharacter(ctx, user, character)
	if refreshErr != nil {
		accessDenied := errors.Is(refreshErr, ErrAccessDenied)
		if accessDenied && userID != "" {
			_ = intel.NewIntelService(r.App).RevokeUploaderTokensForUser(userID)
			audit.New(r.App).Log(
				"user.revoke_upload_tokens",
				"Revoked uploader tokens (allowlist)",
				userID,
				"",
				character,
			)
			logging.New(r.App).
				WithFields(logging.Fields{
					"user_id":             userID,
					"character_record_id": character.Id,
					"character_id":        character.GetInt("eve_character_id"),
				}).
				Info("revoked uploader tokens due to access denied")
		}

		return r.refreshAffiliationOnly(ctx, character, refreshErr)
	}
	return nil
}

func (r *CharacterRefresher) refreshAffiliationOnly(ctx context.Context, character *core.Record, refreshErr error) error {
	charID := character.GetInt("eve_character_id")
	if charID != 0 {
		corpID, allianceID, affiliationErr := r.ESI.CharacterAffiliation(ctx, charID)
		if affiliationErr == nil {
			character.Set("eve_corporation_id", corpID)
			character.Set("eve_alliance_id", allianceID)
			ensureOrgName(ctx, r.App, r.PublicESI, store.CollectionCorporations, corpID)
			ensureOrgName(ctx, r.App, r.PublicESI, store.CollectionAlliances, allianceID)
		} else {
			logging.New(r.App).
				WithFields(logging.Fields{
					"character_record_id": character.Id,
					"character_id":        charID,
				}).
				WithErr(affiliationErr).
				Warn("character affiliation refresh failed")
		}
	}

	refreshAt, _ := types.ParseDateTime(time.Now())
	character.Set("esi_last_refresh_at", refreshAt)
	character.Set("esi_token_valid", false)
	if refreshErr != nil {
		character.Set("esi_last_error", refreshErr.Error())
	}

	return r.App.Save(character)
}
