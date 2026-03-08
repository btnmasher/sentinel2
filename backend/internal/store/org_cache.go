package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/shared/collections"
)

const orgCacheChunk = 50
const orgUpsertChunk = 80

type OrgCacheEntry struct {
	EveID                  int
	Name                   string
	Ticker                 string
	Closed                 bool
	AllianceID             int
	MemberCount            int
	MemberCorporationCount int
}

type CorporationProfileUpsert struct {
	EveID       int
	Name        string
	Ticker      string
	AllianceID  int
	MemberCount int
}

type AllianceProfileUpsert struct {
	EveID                  int
	Name                   string
	Ticker                 string
	MemberCorporationCount int
}

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

func GetOrg(app *pocketbase.PocketBase, collection string, eveID int) (name, ticker string, ok bool) {
	if app == nil || eveID == 0 {
		return "", "", false
	}
	records, err := app.FindRecordsByFilter(collection, "eve_id = {:id}", "", 1, 0, dbx.Params{"id": eveID})
	if err != nil || len(records) == 0 {
		return "", "", false
	}
	record := records[0]
	name = strings.TrimSpace(record.GetString("name"))
	if name == "" {
		return "", "", false
	}
	ticker = strings.TrimSpace(record.GetString("ticker"))
	return name, ticker, true
}

func IsOrgClosed(app *pocketbase.PocketBase, collection string, eveID int) bool {
	if app == nil || eveID == 0 {
		return false
	}
	records, err := app.FindRecordsByFilter(collection, "eve_id = {:id}", "", 1, 0, dbx.Params{"id": eveID})
	if err != nil || len(records) == 0 {
		return false
	}
	return records[0].GetBool("closed")
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
		_ = collections.AppendUnique(&unique, seen, id)
	}
	for start := 0; start < len(unique); start += orgCacheChunk {
		end := min(start+orgCacheChunk, len(unique))
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

func GetOrgCacheEntries(app *pocketbase.PocketBase, collection string, ids []int) map[int]OrgCacheEntry {
	entries := map[int]OrgCacheEntry{}
	if app == nil || len(ids) == 0 {
		return entries
	}
	unique := make([]int, 0, len(ids))
	seen := map[int]struct{}{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		_ = collections.AppendUnique(&unique, seen, id)
	}
	for start := 0; start < len(unique); start += orgCacheChunk {
		end := min(start+orgCacheChunk, len(unique))
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
			eveID := record.GetInt("eve_id")
			if eveID <= 0 {
				continue
			}
			entry := OrgCacheEntry{
				EveID:  eveID,
				Name:   strings.TrimSpace(record.GetString("name")),
				Ticker: strings.TrimSpace(record.GetString("ticker")),
			}
			if record.Collection().Fields.GetByName("closed") != nil {
				entry.Closed = record.GetBool("closed")
			}
			if record.Collection().Fields.GetByName("alliance_id") != nil {
				entry.AllianceID = record.GetInt("alliance_id")
			}
			if record.Collection().Fields.GetByName("member_count") != nil {
				entry.MemberCount = record.GetInt("member_count")
			}
			if record.Collection().Fields.GetByName("member_corporation_count") != nil {
				entry.MemberCorporationCount = record.GetInt("member_corporation_count")
			}
			entries[eveID] = entry
		}
	}
	return entries
}

func UpsertOrg(app *pocketbase.PocketBase, collection string, eveID int, name, ticker string) error {
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
		if record.Collection().Fields.GetByName("ticker") != nil {
			record.Set("ticker", ticker)
		}
		if record.Collection().Fields.GetByName("closed") != nil {
			record.Set("closed", false)
		}
		record.Set("updated_at", updatedAt)
		return app.Save(record)
	}
	record := core.NewRecord(collectionRecord)
	record.Set("eve_id", eveID)
	record.Set("name", name)
	if record.Collection().Fields.GetByName("ticker") != nil {
		record.Set("ticker", ticker)
	}
	if record.Collection().Fields.GetByName("closed") != nil {
		record.Set("closed", false)
	}
	record.Set("updated_at", updatedAt)
	return app.Save(record)
}

func UpsertOrgName(app *pocketbase.PocketBase, collection string, eveID int, name string) error {
	return UpsertOrg(app, collection, eveID, name, "")
}

func UpsertCorporationProfile(
	app *pocketbase.PocketBase,
	eveID int,
	name, ticker string,
	allianceID, memberCount int,
) error {
	if app == nil || eveID == 0 || strings.TrimSpace(name) == "" {
		return nil
	}
	collectionRecord, err := app.FindCollectionByNameOrId(CollectionCorporations)
	if err != nil {
		return err
	}
	records, findErr := app.FindRecordsByFilter(CollectionCorporations, "eve_id = {:id}", "", 1, 0, dbx.Params{"id": eveID})
	if findErr != nil {
		return findErr
	}
	updatedAt, _ := types.ParseDateTime(time.Now())
	if len(records) > 0 {
		record := records[0]
		record.Set("name", strings.TrimSpace(name))
		if record.Collection().Fields.GetByName("ticker") != nil {
			record.Set("ticker", strings.TrimSpace(ticker))
		}
		if record.Collection().Fields.GetByName("closed") != nil {
			record.Set("closed", false)
		}
		if record.Collection().Fields.GetByName("alliance_id") != nil {
			record.Set("alliance_id", allianceID)
		}
		if record.Collection().Fields.GetByName("member_count") != nil {
			record.Set("member_count", memberCount)
		}
		record.Set("updated_at", updatedAt)
		return app.Save(record)
	}
	record := core.NewRecord(collectionRecord)
	record.Set("eve_id", eveID)
	record.Set("name", strings.TrimSpace(name))
	if record.Collection().Fields.GetByName("ticker") != nil {
		record.Set("ticker", strings.TrimSpace(ticker))
	}
	if record.Collection().Fields.GetByName("closed") != nil {
		record.Set("closed", false)
	}
	if record.Collection().Fields.GetByName("alliance_id") != nil {
		record.Set("alliance_id", allianceID)
	}
	if record.Collection().Fields.GetByName("member_count") != nil {
		record.Set("member_count", memberCount)
	}
	record.Set("updated_at", updatedAt)
	return app.Save(record)
}

func UpsertAllianceProfile(
	app *pocketbase.PocketBase,
	eveID int,
	name, ticker string,
	memberCorporationCount int,
) error {
	if app == nil || eveID == 0 || strings.TrimSpace(name) == "" {
		return nil
	}
	collectionRecord, err := app.FindCollectionByNameOrId(CollectionAlliances)
	if err != nil {
		return err
	}
	records, findErr := app.FindRecordsByFilter(CollectionAlliances, "eve_id = {:id}", "", 1, 0, dbx.Params{"id": eveID})
	if findErr != nil {
		return findErr
	}
	updatedAt, _ := types.ParseDateTime(time.Now())
	if len(records) > 0 {
		record := records[0]
		record.Set("name", strings.TrimSpace(name))
		if record.Collection().Fields.GetByName("ticker") != nil {
			record.Set("ticker", strings.TrimSpace(ticker))
		}
		if record.Collection().Fields.GetByName("closed") != nil {
			record.Set("closed", false)
		}
		if record.Collection().Fields.GetByName("member_corporation_count") != nil {
			record.Set("member_corporation_count", memberCorporationCount)
		}
		record.Set("updated_at", updatedAt)
		return app.Save(record)
	}
	record := core.NewRecord(collectionRecord)
	record.Set("eve_id", eveID)
	record.Set("name", strings.TrimSpace(name))
	if record.Collection().Fields.GetByName("ticker") != nil {
		record.Set("ticker", strings.TrimSpace(ticker))
	}
	if record.Collection().Fields.GetByName("closed") != nil {
		record.Set("closed", false)
	}
	if record.Collection().Fields.GetByName("member_corporation_count") != nil {
		record.Set("member_corporation_count", memberCorporationCount)
	}
	record.Set("updated_at", updatedAt)
	return app.Save(record)
}

func UpsertCorporationProfiles(app *pocketbase.PocketBase, profiles []CorporationProfileUpsert) error {
	if app == nil || len(profiles) == 0 {
		return nil
	}

	uniqueProfiles := make(map[int]CorporationProfileUpsert, len(profiles))
	ordered := make([]CorporationProfileUpsert, 0, len(profiles))
	for _, profile := range profiles {
		if profile.EveID <= 0 || strings.TrimSpace(profile.Name) == "" {
			continue
		}
		if _, exists := uniqueProfiles[profile.EveID]; !exists {
			ordered = append(ordered, profile)
		}
		profile.Name = strings.TrimSpace(profile.Name)
		profile.Ticker = strings.TrimSpace(profile.Ticker)
		uniqueProfiles[profile.EveID] = profile
	}
	if len(ordered) == 0 {
		return nil
	}
	for i := range ordered {
		ordered[i] = uniqueProfiles[ordered[i].EveID]
	}

	updatedAt := time.Now().UTC().Format(time.RFC3339)
	for start := 0; start < len(ordered); start += orgUpsertChunk {
		end := min(start+orgUpsertChunk, len(ordered))
		rows := ordered[start:end]
		valueClauses := make([]string, 0, len(rows))
		params := dbx.Params{}
		for index, profile := range rows {
			valueClauses = append(valueClauses, fmt.Sprintf("({:id_%d}, {:eve_id_%d}, {:name_%d}, {:ticker_%d}, {:closed_%d}, {:alliance_id_%d}, {:member_count_%d}, {:updated_at_%d})", index, index, index, index, index, index, index, index))
			params[fmt.Sprintf("id_%d", index)] = security.RandomString(15)
			params[fmt.Sprintf("eve_id_%d", index)] = profile.EveID
			params[fmt.Sprintf("name_%d", index)] = profile.Name
			params[fmt.Sprintf("ticker_%d", index)] = profile.Ticker
			params[fmt.Sprintf("closed_%d", index)] = false
			params[fmt.Sprintf("alliance_id_%d", index)] = profile.AllianceID
			params[fmt.Sprintf("member_count_%d", index)] = profile.MemberCount
			params[fmt.Sprintf("updated_at_%d", index)] = updatedAt
		}
		query := "INSERT INTO `corporations` (`id`, `eve_id`, `name`, `ticker`, `closed`, `alliance_id`, `member_count`, `updated_at`) VALUES " +
			strings.Join(valueClauses, ", ") +
			" ON CONFLICT(`eve_id`) DO UPDATE SET `name`=excluded.`name`, `ticker`=excluded.`ticker`, `closed`=excluded.`closed`, `alliance_id`=excluded.`alliance_id`, `member_count`=excluded.`member_count`, `updated_at`=excluded.`updated_at`"
		if _, execErr := app.NonconcurrentDB().NewQuery(query).Bind(params).Execute(); execErr != nil {
			return execErr
		}
	}
	return nil
}

func UpsertAllianceProfiles(app *pocketbase.PocketBase, profiles []AllianceProfileUpsert) error {
	if app == nil || len(profiles) == 0 {
		return nil
	}

	uniqueProfiles := make(map[int]AllianceProfileUpsert, len(profiles))
	ordered := make([]AllianceProfileUpsert, 0, len(profiles))
	for _, profile := range profiles {
		if profile.EveID <= 0 || strings.TrimSpace(profile.Name) == "" {
			continue
		}
		if _, exists := uniqueProfiles[profile.EveID]; !exists {
			ordered = append(ordered, profile)
		}
		profile.Name = strings.TrimSpace(profile.Name)
		profile.Ticker = strings.TrimSpace(profile.Ticker)
		uniqueProfiles[profile.EveID] = profile
	}
	if len(ordered) == 0 {
		return nil
	}
	for i := range ordered {
		ordered[i] = uniqueProfiles[ordered[i].EveID]
	}

	updatedAt := time.Now().UTC().Format(time.RFC3339)
	for start := 0; start < len(ordered); start += orgUpsertChunk {
		end := min(start+orgUpsertChunk, len(ordered))
		rows := ordered[start:end]
		valueClauses := make([]string, 0, len(rows))
		params := dbx.Params{}
		for index, profile := range rows {
			valueClauses = append(valueClauses, fmt.Sprintf("({:id_%d}, {:eve_id_%d}, {:name_%d}, {:ticker_%d}, {:closed_%d}, {:member_corporation_count_%d}, {:updated_at_%d})", index, index, index, index, index, index, index))
			params[fmt.Sprintf("id_%d", index)] = security.RandomString(15)
			params[fmt.Sprintf("eve_id_%d", index)] = profile.EveID
			params[fmt.Sprintf("name_%d", index)] = profile.Name
			params[fmt.Sprintf("ticker_%d", index)] = profile.Ticker
			params[fmt.Sprintf("closed_%d", index)] = false
			params[fmt.Sprintf("member_corporation_count_%d", index)] = profile.MemberCorporationCount
			params[fmt.Sprintf("updated_at_%d", index)] = updatedAt
		}
		query := "INSERT INTO `alliances` (`id`, `eve_id`, `name`, `ticker`, `closed`, `member_corporation_count`, `updated_at`) VALUES " +
			strings.Join(valueClauses, ", ") +
			" ON CONFLICT(`eve_id`) DO UPDATE SET `name`=excluded.`name`, `ticker`=excluded.`ticker`, `closed`=excluded.`closed`, `member_corporation_count`=excluded.`member_corporation_count`, `updated_at`=excluded.`updated_at`"
		if _, execErr := app.NonconcurrentDB().NewQuery(query).Bind(params).Execute(); execErr != nil {
			return execErr
		}
	}
	return nil
}

func SetOrgClosed(app *pocketbase.PocketBase, collection string, eveID int, closed bool) error {
	if app == nil || eveID == 0 {
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
		if record.Collection().Fields.GetByName("closed") != nil {
			record.Set("closed", closed)
		}
		record.Set("updated_at", updatedAt)
		return app.Save(record)
	}
	record := core.NewRecord(collectionRecord)
	record.Set("eve_id", eveID)
	if record.Collection().Fields.GetByName("closed") != nil {
		record.Set("closed", closed)
	}
	record.Set("updated_at", updatedAt)
	return app.Save(record)
}
