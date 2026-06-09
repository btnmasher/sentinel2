package intel

import (
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/store"
	_ "sentinel2/pb_migrations"
)

func TestNormalizeSystemNamesUsesEnglishNameWhenAlreadyCanonical(t *testing.T) {
	t.Parallel()

	app := newIntelParserTestApp(t)
	mustCreateParserTestSystem(t, app, 30000303, "ALPHA-1", map[string]string{
		"ja": "アルファ-1",
		"ko": "알파-1",
		"zh": "阿尔法-1",
	})

	text := "[20:13:43] Test Reporter > ALPHA-1 ess intrusion"
	gotText, gotSystems, err := NormalizeSystemNames(app, text)
	if err != nil {
		t.Fatalf("NormalizeSystemNames() error = %v", err)
	}

	if gotText != text {
		t.Fatalf("NormalizeSystemNames() text = %q, want %q", gotText, text)
	}
	if len(gotSystems) != 1 || gotSystems[0].System != 30000303 || gotSystems[0].Name != "ALPHA-1" {
		t.Fatalf("NormalizeSystemNames() systems = %#v", gotSystems)
	}
}

func TestNormalizeSystemNamesResolvesLowercaseEnglishName(t *testing.T) {
	t.Parallel()

	app := newIntelParserTestApp(t)
	mustCreateParserTestSystem(t, app, 30000303, "ALPHA-1", map[string]string{
		"ja": "アルファ-1",
		"ko": "알파-1",
		"zh": "阿尔法-1",
	})

	text := "[20:13:43] Test Reporter > alpha-1 ess intrusion"
	gotText, gotSystems, err := NormalizeSystemNames(app, text)
	if err != nil {
		t.Fatalf("NormalizeSystemNames() error = %v", err)
	}

	if gotText != "[20:13:43] Test Reporter > ALPHA-1 ess intrusion" {
		t.Fatalf("NormalizeSystemNames() text = %q", gotText)
	}
	if len(gotSystems) != 1 || gotSystems[0].System != 30000303 || gotSystems[0].Name != "ALPHA-1" {
		t.Fatalf("NormalizeSystemNames() systems = %#v", gotSystems)
	}
}

func TestNormalizeSystemNamesTrimsAutolinkSuffixAsterisk(t *testing.T) {
	t.Parallel()

	app := newIntelParserTestApp(t)
	mustCreateParserTestSystem(t, app, 30000284, "XD-TOV", map[string]string{
		"ja": "XD-TOV",
		"ko": "XD-TOV",
		"zh": "XD-TOV",
	})

	text := "[20:13:43] Test Reporter > XD-TOV*"
	gotText, gotSystems, err := NormalizeSystemNames(app, text)
	if err != nil {
		t.Fatalf("NormalizeSystemNames() error = %v", err)
	}

	if gotText != "[20:13:43] Test Reporter > XD-TOV" {
		t.Fatalf("NormalizeSystemNames() text = %q", gotText)
	}
	if len(gotSystems) != 1 || gotSystems[0].System != 30000284 {
		t.Fatalf("NormalizeSystemNames() systems = %#v", gotSystems)
	}
}

func TestContainsNonASCIIIgnoresAsciiAlphaNumericSystemNames(t *testing.T) {
	t.Parallel()

	if containsNonASCII("ALPHA-2") {
		t.Fatalf("containsNonASCII() reported ALPHA-2 as localized")
	}
	if containsNonASCII("RO-0PZ") {
		t.Fatalf("containsNonASCII() reported RO-0PZ as localized")
	}
	if !containsNonASCII("アルファ-2") {
		t.Fatalf("containsNonASCII() failed to detect Japanese text")
	}
}

func TestNormalizeSystemNamesRewritesLocalizedMatchToEnglish(t *testing.T) {
	t.Parallel()

	app := newIntelParserTestApp(t)
	mustCreateParserTestSystem(t, app, 30000282, "ALPHA-2", map[string]string{
		"ja": "アルファ-2",
		"ko": "알파-2",
		"zh": "阿尔法-2",
	})

	text := "[20:20:12] Test Reporter > アルファ-2*, +7 ALPHA-2"
	gotText, gotSystems, err := NormalizeSystemNames(app, text)
	if err != nil {
		t.Fatalf("NormalizeSystemNames() error = %v", err)
	}

	if gotText != "[20:20:12] Test Reporter > ALPHA-2, +7 ALPHA-2" {
		t.Fatalf("NormalizeSystemNames() text = %q", gotText)
	}
	if len(gotSystems) != 1 || gotSystems[0].System != 30000282 || gotSystems[0].Name != "ALPHA-2" {
		t.Fatalf("NormalizeSystemNames() systems = %#v", gotSystems)
	}
}

func TestNormalizeSystemNamesResolvesMultipleSystems(t *testing.T) {
	t.Parallel()

	app := newIntelParserTestApp(t)
	mustCreateParserTestSystem(t, app, 30000282, "ALPHA-2", map[string]string{
		"ja": "アルファ-2",
	})
	mustCreateParserTestSystem(t, app, 30000283, "BETA-3", map[string]string{
		"zh": "贝塔-3",
	})

	text := "[20:20:12] Test Reporter > アルファ-2 and 贝塔-3 are active"
	gotText, gotSystems, err := NormalizeSystemNames(app, text)
	if err != nil {
		t.Fatalf("NormalizeSystemNames() error = %v", err)
	}

	if gotText != "[20:20:12] Test Reporter > ALPHA-2 and BETA-3 are active" {
		t.Fatalf("NormalizeSystemNames() text = %q", gotText)
	}
	if len(gotSystems) != 2 {
		t.Fatalf("NormalizeSystemNames() systems len = %d, want 2", len(gotSystems))
	}
	if gotSystems[0].System != 30000282 || gotSystems[1].System != 30000283 {
		t.Fatalf("NormalizeSystemNames() systems = %#v", gotSystems)
	}
}

func TestNormalizeSystemNamesLeavesUnknownTokensUntouched(t *testing.T) {
	t.Parallel()

	app := newIntelParserTestApp(t)

	text := "[20:13:43] Test Reporter > UNKNOWN-1 ess intrusion"
	gotText, gotSystems, err := NormalizeSystemNames(app, text)
	if err != nil {
		t.Fatalf("NormalizeSystemNames() error = %v", err)
	}

	if gotText != text {
		t.Fatalf("NormalizeSystemNames() text = %q, want %q", gotText, text)
	}
	if len(gotSystems) != 0 {
		t.Fatalf("NormalizeSystemNames() systems = %#v, want empty", gotSystems)
	}
}

func TestNormalizeSystemNamesWithHintsReturnsSystemMetadata(t *testing.T) {
	t.Parallel()

	app := newIntelParserTestApp(t)
	mustCreateParserTestSystem(t, app, 30000865, "H-W9TY", map[string]string{
		"ja": "H-W9TY",
		"ko": "H-W9TY",
		"zh": "H-W9TY",
	})

	report := IntelReport{
		ID:      42,
		Time:    1710000000,
		Author:  "Test Reporter",
		Text:    "H-W9TY* incoming",
		Regions: []int{10000005},
	}

	normalizedText, systems, hints, err := NormalizeSystemNamesWithHints(app, report.Text)
	if err != nil {
		t.Fatalf("NormalizeSystemNamesWithHints() error = %v", err)
	}

	if normalizedText != "H-W9TY incoming" {
		t.Fatalf("NormalizeSystemNamesWithHints() text = %q", normalizedText)
	}
	if len(systems) != 1 || systems[0].System != 30000865 {
		t.Fatalf("NormalizeSystemNamesWithHints() systems = %#v", systems)
	}
	if len(hints) != 1 || hints[0].SystemID != 30000865 || hints[0].Name != "H-W9TY" {
		t.Fatalf("NormalizeSystemNamesWithHints() hints = %#v", hints)
	}
}

func newIntelParserTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:       t.TempDir(),
		DefaultEncryptionEnv: "pb_test_env",
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("RunAllMigrations() error = %v", err)
	}

	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Fatalf("ResetBootstrapState() error = %v", err)
		}
	})
	return app
}

func mustCreateParserTestSystem(t *testing.T, app *pocketbase.PocketBase, eveID int, name string, localized map[string]string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(store.CollectionSolarSystems)
	if err != nil {
		t.Fatalf("FindCollectionByNameOrId(%q) error = %v", store.CollectionSolarSystems, err)
	}

	record := core.NewRecord(collection)
	record.Set("eve_id", eveID)
	record.Set("name", name)
	record.Set("constellation", 20000041)
	record.Set("region_id", 10000005)
	for locale, value := range localized {
		switch locale {
		case "ja":
			record.Set("name_ja", value)
		case "ko":
			record.Set("name_ko", value)
		case "zh":
			record.Set("name_zh", value)
		}
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("Save(%q) error = %v", name, err)
	}
}
