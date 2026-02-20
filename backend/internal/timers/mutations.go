package timers

import (
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/shared/queryhelpers"
	"sentinel2/internal/store"
)

func (s *Service) Create(input *CreateInput, auth *core.Record) (*core.Record, error) {
	if input == nil {
		return nil, ErrMissingCreateInput
	}
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}

	system, err := s.ResolveSystem(input.SystemID, input.System)
	if err != nil {
		return nil, err
	}
	collection, err := s.App.FindCollectionByNameOrId(store.CollectionTimers)
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	s.applyCreateInput(record, system, input, auth)
	if err := s.App.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Service) Update(id string, input *UpdateInput) (*core.Record, error) {
	if input == nil {
		return nil, ErrMissingUpdateInput
	}
	record, err := s.App.FindRecordById(store.CollectionTimers, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if err := validateUpdateInput(record, input); err != nil {
		return nil, err
	}

	queryhelpers.SetOptional(record, "title", input.Title, trimValue)
	queryhelpers.SetOptional(record, "standing_type", input.Standing, trimValue)
	queryhelpers.SetOptional(record, "timer_kind", input.TimerKind, trimValue)
	queryhelpers.SetOptional(record, "structure_type", input.StructureType, trimValue)
	queryhelpers.SetOptional(record, "stage_label", input.StageLabel, trimValue)
	queryhelpers.SetOptional(record, "planet_id", input.PlanetID, nil)
	queryhelpers.SetOptional(record, "planet_name", input.PlanetName, trimValue)
	queryhelpers.SetOptional(record, "moon_id", input.MoonID, nil)
	queryhelpers.SetOptional(record, "moon_name", input.MoonName, trimValue)
	queryhelpers.SetOptional(record, "owner_corporation_id", input.OwnerCorporationID, nil)
	queryhelpers.SetOptional(record, "owner_corporation_name", input.OwnerCorporationName, trimValue)
	queryhelpers.SetOptional(record, "owner_corporation_ticker", input.OwnerCorporationTicker, trimValue)
	queryhelpers.SetOptional(record, "owner_alliance_id", input.OwnerAllianceID, nil)
	queryhelpers.SetOptional(record, "owner_alliance_name", input.OwnerAllianceName, trimValue)
	queryhelpers.SetOptional(record, "owner_alliance_ticker", input.OwnerAllianceTicker, trimValue)
	queryhelpers.SetOptional(record, "skyhook_fullness_pct", input.SkyhookFullnessPct, nil)
	queryhelpers.SetOptional(record, "stage", input.Stage, nil)
	queryhelpers.SetOptional(record, "total_stages", input.TotalStages, nil)
	queryhelpers.SetOptional(record, "severity", input.Severity, trimValue)
	queryhelpers.SetOptional(record, "status", input.Status, trimValue)
	if input.ExpiresAt != nil {
		record.Set("expires_at", input.ExpiresAt.Format(time.RFC3339))
	}
	queryhelpers.SetOptional(record, "source_ref", input.SourceRef, trimValue)
	queryhelpers.SetOptional(record, "notes", input.Notes, trimValue)
	queryhelpers.SetOptional(record, "raw_text", input.RawText, trimValue)
	queryhelpers.SetOptional(record, "replacement_action", input.ReplacementAction, trimValue)

	if err := s.App.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Service) Delete(id string) (*core.Record, error) {
	record, err := s.App.FindRecordById(store.CollectionTimers, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if err := s.App.Delete(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Service) Cancel(id string, auth *core.Record) (*core.Record, error) {
	record, err := s.App.FindRecordById(store.CollectionTimers, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}

	record.Set("status", timerStatusCanceled)
	if auth != nil {
		record.Set("canceled_by", auth.Id)
	}
	record.Set("canceled_at", time.Now().UTC().Format(time.RFC3339))
	if saveErr := s.App.Save(record); saveErr != nil {
		return nil, saveErr
	}
	return record, nil
}

func (s *Service) Uncancel(id string) (*core.Record, error) {
	record, err := s.App.FindRecordById(store.CollectionTimers, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}

	record.Set("status", timerStatusActive)
	record.Set("canceled_by", nil)
	record.Set("canceled_at", nil)
	if saveErr := s.App.Save(record); saveErr != nil {
		return nil, saveErr
	}
	return record, nil
}

func (s *Service) applyCreateInput(record, system *core.Record, input *CreateInput, auth *core.Record) {
	record.Set("title", queryhelpers.ValueOrTrim(input.Title, system.GetString("name")))
	record.Set("system_id", system.GetInt("eve_id"))
	record.Set("system_name", system.GetString("name"))
	regionID := system.GetInt("region_id")
	record.Set("region_id", regionID)
	record.Set("region_name", s.resolveRegionName(regionID, system.GetString("region_name")))
	record.Set("standing_type", queryhelpers.ValueOrTrim(input.Standing, "hostile"))
	record.Set("timer_kind", queryhelpers.ValueOrTrim(input.TimerKind, "reinforcement"))
	record.Set("structure_type", queryhelpers.ValueOrTrim(input.StructureType, "custom"))
	record.Set("stage_label", strings.TrimSpace(input.StageLabel))
	record.Set("planet_id", input.PlanetID)
	record.Set("planet_name", strings.TrimSpace(input.PlanetName))
	record.Set("moon_id", input.MoonID)
	record.Set("moon_name", strings.TrimSpace(input.MoonName))
	record.Set("owner_corporation_id", input.OwnerCorporationID)
	record.Set("owner_corporation_name", strings.TrimSpace(input.OwnerCorporationName))
	record.Set("owner_corporation_ticker", strings.TrimSpace(input.OwnerCorporationTicker))
	record.Set("owner_alliance_id", input.OwnerAllianceID)
	record.Set("owner_alliance_name", strings.TrimSpace(input.OwnerAllianceName))
	record.Set("owner_alliance_ticker", strings.TrimSpace(input.OwnerAllianceTicker))
	if input.SkyhookFullnessPct != nil {
		record.Set("skyhook_fullness_pct", *input.SkyhookFullnessPct)
	}
	record.Set("stage", input.Stage)
	record.Set("total_stages", max(1, input.TotalStages))
	record.Set("severity", queryhelpers.ValueOrTrim(input.Severity, "medium"))
	record.Set("status", queryhelpers.ValueOrTrim(input.Status, timerStatusActive))
	record.Set("expires_at", input.ExpiresAt.UTC().Format(time.RFC3339))
	record.Set("source", queryhelpers.ValueOrTrim(input.Source, "manual"))
	record.Set("source_ref", strings.TrimSpace(input.SourceRef))
	record.Set("notes", strings.TrimSpace(input.Notes))
	record.Set("raw_text", strings.TrimSpace(input.RawText))
	record.Set("replacement_action", queryhelpers.ValueOrTrim(input.ReplacementAction, "not_replaceable"))
	if auth != nil {
		record.Set("created_by", auth.Id)
	}
}

func trimValue(value string) any {
	return strings.TrimSpace(value)
}
