package timers

import (
	"context"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/store"
)

func (s *Service) hydrateDisplayFields(record *core.Record) {
	if s == nil || s.App == nil || record == nil {
		return
	}

	s.hydrateSystemAndRegionFields(record)
	s.hydrateCelestialFields(record)
	s.hydrateOrganizationFields(record)
}

func (s *Service) hydrateSystemAndRegionFields(record *core.Record) {
	if record == nil {
		return
	}
	s.hydrateSystemFields(record)
	s.hydrateRegionFields(record)
}

func (s *Service) hydrateSystemFields(record *core.Record) {
	systemID := record.GetInt("system_id")
	if systemID <= 0 {
		return
	}
	systemRecord := s.findByEVEID(store.CollectionSolarSystems, systemID)
	if systemRecord == nil {
		return
	}

	if strings.TrimSpace(record.GetString("system_name")) == "" {
		record.Set("system_name", strings.TrimSpace(systemRecord.GetString("name")))
	}

	if record.GetInt("region_id") <= 0 {
		record.Set("region_id", systemRecord.GetInt("region_id"))
	}
}

func (s *Service) hydrateRegionFields(record *core.Record) {
	regionID := record.GetInt("region_id")
	if regionID <= 0 || strings.TrimSpace(record.GetString("region_name")) != "" {
		return
	}
	record.Set("region_name", s.resolveRegionName(regionID, ""))
}

func (s *Service) hydrateCelestialFields(record *core.Record) {
	if record == nil {
		return
	}
	s.hydratePlanetFields(record)
	s.hydrateMoonFields(record)
}

func (s *Service) hydratePlanetFields(record *core.Record) {
	planetID := record.GetInt("planet_id")
	if planetID <= 0 || strings.TrimSpace(record.GetString("planet_name")) != "" {
		return
	}
	planet := s.findByEVEID(store.CollectionPlanets, planetID)
	if planet != nil {
		record.Set("planet_name", strings.TrimSpace(planet.GetString("name")))
	}
}

func (s *Service) hydrateMoonFields(record *core.Record) {
	moonID := record.GetInt("moon_id")
	if moonID <= 0 || strings.TrimSpace(record.GetString("moon_name")) != "" {
		return
	}
	moon := s.findByEVEID(store.CollectionMoons, moonID)
	if moon != nil {
		record.Set("moon_name", strings.TrimSpace(moon.GetString("name")))
	}
}

func (s *Service) hydrateOrganizationFields(record *core.Record) {
	if record == nil {
		return
	}
	s.hydrateCorporationDisplay(record)
	s.hydrateAllianceDisplay(record)
}

func (s *Service) hydrateCorporationDisplay(record *core.Record) {
	corpID := record.GetInt("owner_corporation_id")
	needsName := strings.TrimSpace(record.GetString("owner_corporation_name")) == ""
	needsTicker := strings.TrimSpace(record.GetString("owner_corporation_ticker")) == ""
	if corpID <= 0 || (!needsName && !needsTicker) {
		return
	}
	name, ticker, _, ok, _ := store.GetOrFetchCorporation(context.Background(), s.App, s.PublicESI, corpID)
	if !ok {
		return
	}

	if needsName {
		record.Set("owner_corporation_name", name)
	}

	if needsTicker && ticker != "" {
		record.Set("owner_corporation_ticker", ticker)
	}
}

func (s *Service) hydrateAllianceDisplay(record *core.Record) {
	allianceID := record.GetInt("owner_alliance_id")
	needsName := strings.TrimSpace(record.GetString("owner_alliance_name")) == ""
	needsTicker := strings.TrimSpace(record.GetString("owner_alliance_ticker")) == ""
	if allianceID <= 0 || (!needsName && !needsTicker) {
		return
	}
	name, ticker, ok, _ := store.GetOrFetchAlliance(context.Background(), s.App, s.PublicESI, allianceID)
	if !ok {
		return
	}

	if needsName {
		record.Set("owner_alliance_name", name)
	}

	if needsTicker && ticker != "" {
		record.Set("owner_alliance_ticker", ticker)
	}
}

func (s *Service) findByEVEID(collection string, eveID int) *core.Record {
	if s == nil || s.App == nil || eveID <= 0 {
		return nil
	}

	records, err := s.App.FindRecordsByFilter(
		collection,
		"eve_id = {:id}",
		"",
		1,
		0,
		dbx.Params{"id": eveID},
	)
	if err != nil || len(records) == 0 {
		return nil
	}
	return records[0]
}
