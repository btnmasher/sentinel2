package timers

import (
	"errors"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/config"
	"sentinel2/internal/store"
)

const webhookLookupLimit = 2

func (s *Service) FindByWebhookID(webhookID string) (*core.Record, error) {
	id := strings.TrimSpace(webhookID)
	if id == "" {
		return nil, ErrMissingWebhookID
	}

	records, err := s.App.FindRecordsByFilter(
		store.CollectionTimers,
		"webhook_id = {:webhook_id}",
		"",
		webhookLookupLimit,
		0,
		dbx.Params{"webhook_id": id},
	)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, ErrTimerNotFound
	}
	return records[0], nil
}

func (s *Service) CreateWebhook(input *CreateInput) (*core.Record, error) {
	if input == nil {
		return nil, ErrMissingCreateInput
	}
	input.Source = config.TimerSourceWebhook
	input.WebhookID = strings.TrimSpace(input.WebhookID)
	if input.WebhookID == "" {
		return nil, ErrMissingWebhookID
	}
	return s.Create(input, nil)
}

func (s *Service) DeleteByWebhookID(webhookID string) error {
	record, err := s.FindByWebhookID(webhookID)
	if err != nil {
		if errors.Is(err, ErrTimerNotFound) {
			return nil
		}
		return err
	}
	return s.App.Delete(record)
}
