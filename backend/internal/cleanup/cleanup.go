package cleanup

import (
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

type Service struct {
	App *pocketbase.PocketBase
}

func New(app *pocketbase.PocketBase) *Service {
	return &Service{App: app}
}

func (s *Service) RemoveExpired(collection string) (int, error) {
	records, recordsErr := s.App.FindRecordsByFilter(
		collection,
		"expires_at < {:now}",
		"",
		0,
		0,
		map[string]any{"now": types.NowDateTime()},
	)
	if recordsErr != nil {
		return 0, recordsErr
	}

	removed := 0
	failed := 0
	for _, rec := range records {
		if err := s.App.Delete(rec); err == nil {
			removed++
		} else {
			failed++
			logging.New(s.App).
				WithFields(logging.Fields{
					"collection": collection,
					"record_id":  rec.Id,
				}).
				WithErr(err).
				Debug("cleanup delete failed")
		}
	}
	if failed > 0 {
		logging.New(s.App).
			WithFields(logging.Fields{
				"collection": collection,
				"failed":     failed,
			}).
			Warn("cleanup delete failures")
	}
	return removed, nil
}

func (s *Service) RemoveRevokedUploaderTokens() (int, error) {
	records, recordsErr := s.App.FindRecordsByFilter(
		store.CollectionUploaderTokens,
		"revoked = true",
		"",
		0,
		0,
		nil,
	)
	if recordsErr != nil {
		return 0, recordsErr
	}

	removed := 0
	failed := 0
	for _, rec := range records {
		if err := s.App.Delete(rec); err == nil {
			removed++
		} else {
			failed++
			logging.New(s.App).
				WithFields(logging.Fields{
					"collection": store.CollectionUploaderTokens,
					"record_id":  rec.Id,
				}).
				WithErr(err).
				Debug("cleanup delete failed")
		}
	}
	if failed > 0 {
		logging.New(s.App).
			WithFields(logging.Fields{
				"collection": store.CollectionUploaderTokens,
				"failed":     failed,
			}).
			Warn("cleanup delete failures")
	}
	return removed, nil
}

func (s *Service) RemoveOldIntelReports(maxAge time.Duration) (int, error) {
	if maxAge <= 0 {
		return 0, nil
	}
	cutoff := time.Now().Add(-maxAge).Unix()
	records, recordsErr := s.App.FindRecordsByFilter(
		store.CollectionIntelReports,
		"report_time < {:cutoff}",
		"",
		0,
		0,
		map[string]any{"cutoff": cutoff},
	)
	if recordsErr != nil {
		return 0, recordsErr
	}
	removed := 0
	failed := 0
	for _, rec := range records {
		if err := s.App.Delete(rec); err == nil {
			removed++
		} else {
			failed++
			logging.New(s.App).
				WithFields(logging.Fields{
					"collection": store.CollectionIntelReports,
					"record_id":  rec.Id,
				}).
				WithErr(err).
				Debug("cleanup delete failed")
		}
	}
	if failed > 0 {
		logging.New(s.App).
			WithFields(logging.Fields{
				"collection": store.CollectionIntelReports,
				"failed":     failed,
			}).
			Warn("cleanup delete failures")
	}
	return removed, nil
}
