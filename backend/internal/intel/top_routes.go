package intel

import (
	"sort"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/store"
)

const (
	TopRoutesDays = 4
)

type TopRoutesService struct {
	App *pocketbase.PocketBase
}

func NewTopRoutesService(app *pocketbase.PocketBase) *TopRoutesService {
	return &TopRoutesService{App: app}
}

func (s *TopRoutesService) Add(systems []int) error {
	bucket := time.Now().UTC().Format("2006-01-02")

	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionTopRoutes)
	if collErr != nil {
		return collErr
	}

	for _, system := range systems {
		records, recordsErr := s.App.FindRecordsByFilter(
			coll.Name,
			"bucket = {:bucket} && system_id = {:system}",
			"",
			1,
			0,
			map[string]any{"bucket": bucket, "system": system},
		)
		if recordsErr != nil {
			return recordsErr
		}

		var rec *core.Record
		if len(records) > 0 {
			rec = records[0]
		} else {
			rec = core.NewRecord(coll)
			rec.Set("bucket", bucket)
			rec.Set("system_id", system)
			rec.Set("count", 0)
		}
		rec.Set("count", rec.GetInt("count")+1)
		rec.Set("updated_at", types.NowDateTime())
		if saveErr := s.App.Save(rec); saveErr != nil {
			return saveErr
		}
	}

	return nil
}

func (s *TopRoutesService) Top() ([]int, error) {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionTopRoutes)
	if collErr != nil {
		return nil, collErr
	}

	cutoff := time.Now().UTC().Add(-TopRoutesDays * 24 * time.Hour)
	cutoffDT, _ := types.ParseDateTime(cutoff)

	records, recordsErr := s.App.FindRecordsByFilter(
		coll.Name,
		"updated_at >= {:cutoff}",
		"",
		0,
		0,
		map[string]any{"cutoff": cutoffDT},
	)
	if recordsErr != nil {
		return nil, recordsErr
	}

	tally := map[int]int{}
	for _, rec := range records {
		system := int(rec.GetInt("system_id"))
		tally[system] += int(rec.GetInt("count"))
	}

	type pair struct {
		system int
		count  int
	}
	pairs := []pair{}
	for k, v := range tally {
		pairs = append(pairs, pair{system: k, count: v})
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i].count < pairs[j].count })

	top := []int{}
	for i := len(pairs) - 1; i >= 0 && len(top) < 4; i-- {
		top = append(top, pairs[i].system)
	}
	return top, nil
}
