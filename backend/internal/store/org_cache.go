package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const orgCacheChunk = 50

func GetOrgName(app *pocketbase.PocketBase, collection string, eveID int) string {
	if app == nil || eveID == 0 {
		return ""
	}
	records, err := app.FindRecordsByFilter(collection, "eve_id = {:id}", "", 1, 0, dbx.Params{"id": eveID})
	if err != nil || len(records) == 0 {
		return ""
	}
	return records[0].GetString("name")
}

func GetOrgNames(app *pocketbase.PocketBase, collection string, ids []int) map[int]string {
	names := map[int]string{}
	if app == nil || len(ids) == 0 {
		return names
	}
	unique := make([]int, 0, len(ids))
	seen := map[int]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	for start := 0; start < len(unique); start += orgCacheChunk {
		end := start + orgCacheChunk
		if end > len(unique) {
			end = len(unique)
		}
		clauses := make([]string, 0, end-start)
		params := dbx.Params{}
		for i, id := range unique[start:end] {
			key := fmt.Sprintf("id_%d", i)
			clauses = append(clauses, fmt.Sprintf("eve_id = {:%s}", key))
			params[key] = id
		}
		filter := strings.Join(clauses, " || ")
		records, err := app.FindRecordsByFilter(collection, filter, "", 0, 0, params)
		if err != nil {
			continue
		}
		for _, record := range records {
			names[record.GetInt("eve_id")] = record.GetString("name")
		}
	}
	return names
}

func UpsertOrgName(app *pocketbase.PocketBase, collection string, eveID int, name string) error {
	if app == nil || eveID == 0 || name == "" {
		return nil
	}
	collectionRecord, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		return err
	}
	records, findErr := app.FindRecordsByFilter(collection, "eve_id = {:id}", "", 1, 0, dbx.Params{"id": eveID})
	if findErr != nil {
		return findErr
	}
	updatedAt, _ := types.ParseDateTime(time.Now())
	if len(records) > 0 {
		record := records[0]
		record.Set("name", name)
		record.Set("updated_at", updatedAt)
		return app.Save(record)
	}
	record := core.NewRecord(collectionRecord)
	record.Set("eve_id", eveID)
	record.Set("name", name)
	record.Set("updated_at", updatedAt)
	return app.Save(record)
}
