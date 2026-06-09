package intel

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/store"
)

func TestListReportsReturnsSystemsRegionsAndMeta(t *testing.T) {
	t.Parallel()

	app := newIntelParserTestApp(t)

	channelCollection, err := app.FindCollectionByNameOrId(store.CollectionIntelChannels)
	if err != nil {
		t.Fatalf("FindCollectionByNameOrId(%q) error = %v", store.CollectionIntelChannels, err)
	}

	channel := core.NewRecord(channelCollection)
	channel.Set("channel_name", "Intel Channel")
	if err := app.Save(channel); err != nil {
		t.Fatalf("Save(channel) error = %v", err)
	}

	reportCollection, err := app.FindCollectionByNameOrId(store.CollectionIntelReports)
	if err != nil {
		t.Fatalf("FindCollectionByNameOrId(%q) error = %v", store.CollectionIntelReports, err)
	}

	record := core.NewRecord(reportCollection)
	record.Set("report_id", 1780985302192)
	record.Set("report_time", 1780985300)
	record.Set("author", "Biscuit Toralen")
	record.Set("text", "4-HWWF 2x imperial navy slicer camping 4h gate")
	record.Set("systems", []IntelSystem{{
		System:        30000127,
		Name:          "Sirseshin",
		Constellation: 20000018,
		Region:        10000002,
	}})
	record.Set("regions", []int{10000002})
	record.Set("meta", map[string]any{
		"source": "zkill_feed",
		"zkill": map[string]any{
			"killmail_id":      99,
			"url":              "https://zkillboard.com/kill/99/",
			"display_text":     "test",
			"killer_name":      "Pilot A",
			"killer_hostility": "hostile",
			"victim_hostility": "hostile",
			"system_name":      "Sirseshin",
		},
	})
	record.Set("channel", channel.Id)

	if err := app.Save(record); err != nil {
		t.Fatalf("Save(report) error = %v", err)
	}

	service := NewIntelService(app)
	reports, err := service.ListReports(10)
	if err != nil {
		t.Fatalf("ListReports() error = %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("ListReports() len = %d, want 1", len(reports))
	}

	got := reports[0]
	if len(got.Systems) != 1 || got.Systems[0].System != 30000127 || got.Systems[0].Name != "Sirseshin" {
		t.Fatalf("ListReports() systems = %#v", got.Systems)
	}
	if len(got.Regions) != 1 || got.Regions[0] != 10000002 {
		t.Fatalf("ListReports() regions = %#v", got.Regions)
	}
	if got.Meta == nil {
		t.Fatalf("ListReports() meta = nil, want data")
	}
	if got.Meta["source"] != "zkill_feed" {
		t.Fatalf("ListReports() meta.source = %#v, want zkill_feed", got.Meta["source"])
	}
	if zkill, ok := got.Meta["zkill"].(map[string]any); !ok || zkill["system_name"] != "Sirseshin" {
		t.Fatalf("ListReports() meta.zkill = %#v", got.Meta["zkill"])
	}
}
