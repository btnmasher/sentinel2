package timers

import (
	"sentinel2/internal/mapdata"
)

func (s *Service) ListMoonsBySystem(systemID, limit int) ([]MoonSearchItem, error) {
	items, err := mapdata.ListMoonsBySystem(s.App, systemID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]MoonSearchItem, 0, len(items))
	for _, item := range items {
		out = append(out, MoonSearchItem{
			ID:       item.ID,
			Name:     item.Name,
			SystemID: item.SystemID,
		})
	}
	return out, nil
}

func (s *Service) ListPlanetsBySystem(systemID, limit int) ([]PlanetSearchItem, error) {
	items, err := mapdata.ListPlanetsBySystem(s.App, systemID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]PlanetSearchItem, 0, len(items))
	for _, item := range items {
		out = append(out, PlanetSearchItem{
			ID:       item.ID,
			Name:     item.Name,
			SystemID: item.SystemID,
		})
	}
	return out, nil
}
