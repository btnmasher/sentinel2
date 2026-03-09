package timers

import (
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/store"
)

func (s *Service) ResolveSystemID(systemName string) (int, error) {
	record, err := s.ResolveSystem(0, systemName)
	if err != nil {
		return 0, err
	}
	return record.GetInt("eve_id"), nil
}

func (s *Service) ResolveSystem(systemID int, systemName string) (*core.Record, error) {
	if systemID > 0 {
		records, err := s.App.FindRecordsByFilter(store.CollectionSolarSystems, "eve_id = {:id}", "", 1, 0, dbx.Params{"id": systemID})
		if err == nil && len(records) > 0 {
			return records[0], nil
		}
	}

	name := strings.TrimSpace(systemName)
	if name == "" {
		return nil, ErrMissingSystem
	}

	records, err := s.App.FindRecordsByFilter(store.CollectionSolarSystems, "name = {:name}", "", 1, 0, dbx.Params{"name": name})
	if err == nil && len(records) > 0 {
		return records[0], nil
	}

	records, err = s.App.FindRecordsByFilter(
		store.CollectionSolarSystems,
		"name ~ {:name}",
		"name",
		systemNameCandidateLimit,
		0,
		dbx.Params{"name": "%" + name + "%"},
	)
	if err != nil || len(records) == 0 {
		return nil, ErrSystemNotFound
	}
	for _, record := range records {
		if strings.EqualFold(record.GetString("name"), name) {
			return record, nil
		}
	}
	return records[0], nil
}

func (s *Service) SearchSystems(query string, limit int) ([]SystemSearchItem, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []SystemSearchItem{}, nil
	}

	if limit <= 0 {
		limit = defaultSearchLimit
	}

	records, err := s.App.FindRecordsByFilter(
		store.CollectionSolarSystems,
		"name ~ {:query}",
		"name",
		limit,
		0,
		dbx.Params{"query": "%" + q + "%"},
	)
	if err != nil {
		return nil, err
	}

	out := make([]SystemSearchItem, 0, len(records))
	for _, record := range records {
		out = append(out, SystemSearchItem{
			ID:       record.GetInt("eve_id"),
			Name:     record.GetString("name"),
			RegionID: record.GetInt("region_id"),
			Region:   s.resolveRegionName(record.GetInt("region_id"), strconv.Itoa(record.GetInt("region_id"))),
		})
	}
	return out, nil
}
