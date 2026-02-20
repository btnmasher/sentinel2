package timers

import (
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/store"
)

func (s *Service) hydrateRegionNames(records []*core.Record) {
	if len(records) == 0 {
		return
	}

	regionNameCache := map[int]string{}
	for _, record := range records {
		if record == nil {
			continue
		}
		regionID := record.GetInt("region_id")
		if regionID <= 0 {
			continue
		}
		current := strings.TrimSpace(record.GetString("region_name"))
		if current != "" && current != strconv.Itoa(regionID) {
			continue
		}
		resolved := regionNameCache[regionID]
		if resolved == "" {
			resolved = s.lookupRegionName(regionID)
			if resolved == "" {
				resolved = strconv.Itoa(regionID)
			}
			regionNameCache[regionID] = resolved
		}
		record.Set("region_name", resolved)
	}
}

func (s *Service) resolveRegionName(regionID int, fallback string) string {
	name := strings.TrimSpace(fallback)
	if name != "" && name != strconv.Itoa(regionID) {
		return name
	}
	resolved := s.lookupRegionName(regionID)
	if resolved != "" {
		return resolved
	}
	return strconv.Itoa(regionID)
}

func (s *Service) lookupRegionName(regionID int) string {
	if regionID <= 0 {
		return ""
	}
	records, err := s.App.FindRecordsByFilter(
		store.CollectionRegions,
		"eve_id = {:id}",
		"",
		1,
		0,
		dbx.Params{"id": regionID},
	)
	if err != nil || len(records) == 0 {
		return ""
	}
	return strings.TrimSpace(records[0].GetString("name"))
}
