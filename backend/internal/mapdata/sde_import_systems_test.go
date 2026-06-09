package mapdata

import "testing"

func TestParseSystemRowCapturesLocalizedNames(t *testing.T) {
	t.Parallel()

	row := map[string]any{
		"solarSystemID":   30002322,
		"constellationID": 20000340,
		"regionID":        10000027,
		"securityStatus":  -0.458969,
		"name": map[string]any{
			"en": "RO-0PZ",
			"ja": "RO-0PZ",
			"ko": "RO-0PZ",
			"zh": "RO-0PZ",
		},
	}

	system := parseSystemRow(row)
	if system.name != "RO-0PZ" {
		t.Fatalf("parseSystemRow().name = %q, want RO-0PZ", system.name)
	}
	if got := system.localizedNames["ja"]; got != "RO-0PZ" {
		t.Fatalf("parseSystemRow().localizedNames[ja] = %q, want RO-0PZ", got)
	}

	payload := system.payload()
	if got := payload["name_ja"]; got != "RO-0PZ" {
		t.Fatalf("payload[name_ja] = %q, want RO-0PZ", got)
	}
	if got := payload["name_ko"]; got != "RO-0PZ" {
		t.Fatalf("payload[name_ko] = %q, want RO-0PZ", got)
	}
	if got := payload["name_zh"]; got != "RO-0PZ" {
		t.Fatalf("payload[name_zh] = %q, want RO-0PZ", got)
	}
}
