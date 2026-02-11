package intel

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"sentinel2/internal/logging"
	"sentinel2/internal/store"
)

const (
	ReportTimeoutSeconds   = 60 * 30
	ReportHashExpiry       = 300
	DefaultReportHashSlots = 20
	UploaderExpiry         = 120
)

type IntelService struct {
	App             *pocketbase.PocketBase
	ReportHashSlots int
}

type IntelSystem struct {
	System        int    `json:"system"`
	Name          string `json:"name"`
	Constellation int    `json:"constellation"`
	Region        int    `json:"region"`
}

type IntelReport struct {
	ID        int64         `json:"id"`
	RecordID  string        `json:"record_id"`
	Time      int64         `json:"time"`
	Author    string        `json:"author"`
	Text      string        `json:"text"`
	Systems   []IntelSystem `json:"systems"`
	Regions   []int         `json:"regions"`
	Uploader  string        `json:"uploader"`
	ChannelID string        `json:"channel_id"`
}

func NewIntelService(app *pocketbase.PocketBase) *IntelService {
	return &IntelService{
		App:             app,
		ReportHashSlots: DefaultReportHashSlots,
	}
}

func (s *IntelService) SetReportHashSlots(slots int) {
	if slots < 1 {
		s.ReportHashSlots = DefaultReportHashSlots
		return
	}
	s.ReportHashSlots = slots
}

func (s *IntelService) reportHashSlots() int {
	if s.ReportHashSlots < 1 {
		return DefaultReportHashSlots
	}
	return s.ReportHashSlots
}

func (s *IntelService) GetOrCreateUploaderToken(userID string) (*core.Record, error) {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionUploaderTokens)
	if collErr != nil {
		return nil, collErr
	}
	records, recordsErr := s.App.FindRecordsByFilter(
		coll.Name,
		"user = {:user}",
		"-created_date",
		1,
		0,
		map[string]any{"user": userID},
	)
	if recordsErr != nil {
		return nil, recordsErr
	}
	if len(records) == 0 {
		return s.regenerateUploaderToken(userID)
	}
	if records[0].GetBool("revoked") {
		return nil, ErrExpiredOrRevoked
	}
	return records[0], nil
}

func (s *IntelService) GetValidUploaderToken(userID string) (*core.Record, error) {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionUploaderTokens)
	if collErr != nil {
		return nil, collErr
	}

	records, recordsErr := s.App.FindRecordsByFilter(
		coll.Name,
		"user = {:user}",
		"-created_date",
		1,
		0,
		map[string]any{"user": userID},
	)
	if recordsErr != nil {
		return nil, recordsErr
	}

	if len(records) == 0 {
		return nil, ErrExpiredOrRevoked
	}
	if records[0].GetBool("revoked") {
		return nil, ErrExpiredOrRevoked
	}
	return records[0], nil
}

func (s *IntelService) RotateUploaderToken(userID string) (*core.Record, error) {
	if _, err := s.GetValidUploaderToken(userID); err != nil {
		return nil, err
	}
	return s.regenerateUploaderToken(userID)
}

func (s *IntelService) RegenerateUploaderToken(userID string) (*core.Record, error) {
	return s.regenerateUploaderToken(userID)
}

func (s *IntelService) regenerateUploaderToken(userID string) (*core.Record, error) {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionUploaderTokens)
	if collErr != nil {
		return nil, collErr
	}

	records, recordsErr := s.App.FindRecordsByFilter(
		coll.Name,
		"user = {:user}",
		"",
		0,
		0,
		map[string]any{"user": userID},
	)
	if recordsErr != nil {
		return nil, recordsErr
	}

	failed := 0
	for _, rec := range records {
		rec.Set("revoked", true)
		if saveErr := s.App.Save(rec); saveErr != nil {
			failed++
			logging.New(s.App).
				WithFields(logging.Fields{
					"user_id":  userID,
					"token_id": rec.Id,
				}).
				WithErr(saveErr).
				Debug("uploader token revoke save failed")
		}
	}
	if failed > 0 {
		logging.New(s.App).
			WithFields(logging.Fields{
				"user_id": userID,
				"failed":  failed,
			}).
			Warn("uploader token revoke failures")
	}

	record := core.NewRecord(coll)
	record.Set("user", userID)
	record.Set("revoked", false)
	record.Set("created_date", types.NowDateTime())
	if saveErr := s.App.Save(record); saveErr != nil {
		return nil, saveErr
	}
	return record, nil
}

func (s *IntelService) HasValidUploaderToken(userID string) (bool, error) {
	_, err := s.GetValidUploaderToken(userID)
	if err == nil {
		return true, nil
	}
	if err == ErrExpiredOrRevoked {
		return false, nil
	}
	return false, err
}

func (s *IntelService) ValidateUploaderToken(token string) (*core.Record, error) {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionUploaderTokens)
	if collErr != nil {
		return nil, collErr
	}

	record, recordErr := s.App.FindRecordById(coll.Name, token)
	if recordErr != nil {
		return nil, recordErr
	}
	if record.GetBool("revoked") {
		return nil, ErrExpiredOrRevoked
	}
	return record, nil
}

func (s *IntelService) RevokeUploaderTokensForUser(userID string) error {
	records, recordsErr := s.App.FindRecordsByFilter(
		store.CollectionUploaderTokens,
		"user = {:user}",
		"",
		0,
		0,
		map[string]any{"user": userID},
	)
	if recordsErr != nil {
		return recordsErr
	}
	failed := 0
	for _, rec := range records {
		rec.Set("revoked", true)
		if saveErr := s.App.Save(rec); saveErr != nil {
			failed++
			logging.New(s.App).
				WithFields(logging.Fields{
					"user_id":  userID,
					"token_id": rec.Id,
				}).
				WithErr(saveErr).
				Debug("uploader token revoke save failed")
		}
	}
	if failed > 0 {
		logging.New(s.App).
			WithFields(logging.Fields{
				"user_id": userID,
				"failed":  failed,
			}).
			Warn("uploader token revoke failures")
	}
	return nil
}

func (s *IntelService) UpdateUploader(userID string) error {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionIntelUploaders)
	if collErr != nil {
		return collErr
	}

	records, recordsErr := s.App.FindRecordsByFilter(coll.Name, "user = {:user}", "", 1, 0, map[string]any{"user": userID})
	if recordsErr != nil {
		return recordsErr
	}
	var record *core.Record
	if len(records) > 0 {
		record = records[0]
	} else {
		record = core.NewRecord(coll)
		record.Set("user", userID)
	}
	expiresAt, _ := types.ParseDateTime(time.Now().Add(UploaderExpiry * time.Second))
	record.Set("expires_at", expiresAt)
	return s.App.Save(record)
}

func (s *IntelService) UploaderCount() (int, error) {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionIntelUploaders)
	if collErr != nil {
		return 0, collErr
	}
	records, recordsErr := s.App.FindRecordsByFilter(
		coll.Name,
		"expires_at >= {:now}",
		"",
		0,
		0,
		map[string]any{"now": types.NowDateTime()},
	)
	if recordsErr != nil {
		return 0, recordsErr
	}
	return len(records), nil
}

func (s *IntelService) CreateReport(report IntelReport) error {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionIntelReports)
	if collErr != nil {
		return collErr
	}
	record := core.NewRecord(coll)
	record.Set("report_id", report.ID)
	record.Set("report_time", report.Time)
	record.Set("author", report.Author)
	record.Set("text", report.Text)
	record.Set("systems", report.Systems)
	record.Set("regions", report.Regions)
	if report.Uploader != "" {
		record.Set("uploader_user", report.Uploader)
	}
	if report.ChannelID != "" {
		record.Set("channel", report.ChannelID)
	}
	return s.App.Save(record)
}

func (s *IntelService) ListReports(limit int) ([]IntelReport, error) {
	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionIntelReports)
	if collErr != nil {
		return nil, collErr
	}
	records, recordsErr := s.App.FindRecordsByFilter(
		coll.Name,
		"",
		"-report_time",
		limit,
		0,
		nil,
	)
	if recordsErr != nil {
		return nil, recordsErr
	}

	reports := make([]IntelReport, 0, len(records))
	for _, rec := range records {
		reports = append(reports, IntelReport{
			ID:        int64(rec.GetInt("report_id")),
			RecordID:  rec.Id,
			Time:      int64(rec.GetInt("report_time")),
			Author:    rec.GetString("author"),
			Text:      rec.GetString("text"),
			Systems:   decodeSystems(rec.Get("systems")),
			Regions:   toIntSlice(rec.Get("regions")),
			Uploader:  rec.GetString("uploader_user"),
			ChannelID: rec.GetString("channel"),
		})
	}

	sort.Slice(reports, func(i, j int) bool { return reports[i].Time > reports[j].Time })
	if limit > 0 && len(reports) > limit {
		reports = reports[:limit]
	}
	return reports, nil
}

func (s *IntelService) ShouldCreateReport(author string, text string, reportTime int64) (bool, error) {
	hash := sha256.Sum256([]byte(author + "+" + text))
	hashText := hex.EncodeToString(hash[:])

	coll, collErr := s.App.FindCollectionByNameOrId(store.CollectionIntelReportHash)
	if collErr != nil {
		return false, collErr
	}

	for i := 1; i <= s.reportHashSlots(); i++ {
		records, recordsErr := s.App.FindRecordsByFilter(
			coll.Name,
			"hash = {:hash} && hash_index = {:idx}",
			"",
			1,
			0,
			map[string]any{"hash": hashText, "idx": i},
		)
		if recordsErr != nil {
			return false, recordsErr
		}

		if len(records) == 0 {
			rec := core.NewRecord(coll)
			rec.Set("hash", hashText)
			rec.Set("hash_index", i)
			rec.Set("report_time", reportTime)
			expiresAt, _ := types.ParseDateTime(time.Now().Add(ReportHashExpiry * time.Second))
			rec.Set("expires_at", expiresAt)
			return true, s.App.Save(rec)
		}

		if absInt64(int64(records[0].GetInt("report_time"))-reportTime) < 10 {
			return false, nil
		}
	}

	return false, nil
}

func decodeSystems(value interface{}) []IntelSystem {
	out := []IntelSystem{}
	switch v := value.(type) {
	case []IntelSystem:
		return v
	case []map[string]interface{}:
		for _, item := range v {
			out = append(out, decodeSystemMap(item))
		}
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, decodeSystemMap(m))
			}
		}
	}
	return out
}

func decodeSystemMap(m map[string]interface{}) IntelSystem {
	sys := IntelSystem{}
	if value, ok := m["system"]; ok {
		sys.System = toInt(value)
	}
	if value, ok := m["name"]; ok {
		if name, ok := value.(string); ok {
			sys.Name = name
		}
	}
	if value, ok := m["constellation"]; ok {
		sys.Constellation = toInt(value)
	}
	if value, ok := m["region"]; ok {
		sys.Region = toInt(value)
	}
	return sys
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		num, _ := v.Int64()
		return int(num)
	case string:
		if n, parseErr := strconv.Atoi(v); parseErr == nil {
			return n
		}
	}
	return 0
}

func toIntSlice(value interface{}) []int {
	out := []int{}
	switch v := value.(type) {
	case []int:
		return v
	case []interface{}:
		for _, item := range v {
			switch num := item.(type) {
			case float64:
				out = append(out, int(num))
			case int:
				out = append(out, num)
			}
		}
	}
	return out
}

func absInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
