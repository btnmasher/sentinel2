package cleanup

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"sentinel2-taskutil/internal/project"
)

func TestCleanRoot_PreservesPBDataAndEnv(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	mustMkdirAll(t, filepath.Join(binDir, "pb_data"))
	if err := os.WriteFile(filepath.Join(binDir, ".env"), []byte("A=B\n"), 0o644); err != nil {
		t.Fatalf("write bin/.env: %v", err)
	}

	if err := os.WriteFile(filepath.Join(binDir, "delete.me"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write bin/delete.me: %v", err)
	}
	mustMkdirAll(t, filepath.Join(root, ".task"))
	mustMkdirAll(t, filepath.Join(root, ".tmp", "stamps"))

	cfg := project.Config{
		RootDir:       root,
		BinDirPath:    binDir,
		EmbedSrcPath:  filepath.Join(root, "frontend", "dist"),
		EmbedDestPath: filepath.Join(root, "backend", "internal", "web", "dist"),
		CleanYes:      true,
		CleanRules: strings.Join([]string{
			"bin/**",
			"!bin/pb_data",
			"!bin/pb_data/**",
			"!bin/.env",
			".task",
		}, ","),
	}

	if err := CleanRoot(cfg); err != nil {
		t.Fatalf("CleanRoot() error = %v", err)
	}

	assertExists(t, filepath.Join(binDir, "pb_data"))
	assertExists(t, filepath.Join(binDir, ".env"))
	assertNotExists(t, filepath.Join(binDir, "delete.me"))
	assertNotExists(t, filepath.Join(root, ".task"))
}

func TestCleanRoot_IgnorePriority(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "dist", "keep"))
	mustMkdirAll(t, filepath.Join(root, "dist", "drop"))
	writeTestFile(t, filepath.Join(root, "dist", "keep", "a.txt"))
	writeTestFile(t, filepath.Join(root, "dist", "drop", "b.txt"))
	cfg := project.Config{
		RootDir:  root,
		CleanYes: true,
		CleanRules: strings.Join([]string{
			"dist/**",
			"!dist/keep",
			"!dist/keep/**",
		}, ","),
	}

	if err := CleanRoot(cfg); err != nil {
		t.Fatalf("CleanRoot() error = %v", err)
	}
	assertExists(t, filepath.Join(root, "dist", "keep", "a.txt"))
	assertNotExists(t, filepath.Join(root, "dist", "drop", "b.txt"))
}

func TestParseCleanRules_Invalid(t *testing.T) {
	_, err := parseCleanRules("!")
	if err == nil {
		t.Fatalf("parseCleanRules should fail for empty ignore pattern")
	}
}

func TestParseCleanRules_InvalidGlob(t *testing.T) {
	_, err := parseCleanRules("dist/[")
	if err == nil {
		t.Fatalf("parseCleanRules should fail for invalid glob pattern")
	}
}

func TestParseCleanRules_CommentsAndEscapes(t *testing.T) {
	rules, err := parseCleanRules(strings.Join([]string{
		"# full line comment",
		`dist/** # inline comment`,
		`!dist/keep/**`,
		`\!literal-bang.txt`,
		`\#literal-hash.txt`,
	}, "\n"))
	if err != nil {
		t.Fatalf("parseCleanRules() error = %v", err)
	}

	if len(rules) != 4 {
		t.Fatalf("rule count = %d, want 4", len(rules))
	}

	if !rules[0].include || rules[0].pattern != "dist/**" {
		t.Fatalf("rules[0] = %#v", rules[0])
	}

	if rules[1].include || rules[1].pattern != "dist/keep/**" {
		t.Fatalf("rules[1] = %#v", rules[1])
	}

	if !rules[2].include || rules[2].pattern != "!literal-bang.txt" {
		t.Fatalf("rules[2] = %#v", rules[2])
	}

	if !rules[3].include || rules[3].pattern != "#literal-hash.txt" {
		t.Fatalf("rules[3] = %#v", rules[3])
	}
}

func TestParseCleanRules_SeparatorsAndWhitespace(t *testing.T) {
	rules, err := parseCleanRules("  dist/** ; !dist/keep/** ,  logs/*.log  ")
	if err != nil {
		t.Fatalf("parseCleanRules() error = %v", err)
	}

	if len(rules) != 3 {
		t.Fatalf("rule count = %d, want 3", len(rules))
	}

	if !rules[0].include || rules[0].pattern != "dist/**" {
		t.Fatalf("rules[0] = %#v", rules[0])
	}

	if rules[1].include || rules[1].pattern != "dist/keep/**" {
		t.Fatalf("rules[1] = %#v", rules[1])
	}

	if !rules[2].include || rules[2].pattern != "logs/*.log" {
		t.Fatalf("rules[2] = %#v", rules[2])
	}
}

func TestCleanRoot_IncludeDirIgnoreFile_BothOrders(t *testing.T) {
	cases := []struct {
		name  string
		rules []string
	}{
		{name: "include_then_ignore", rules: []string{"dist/**", "!dist/keep.txt"}},
		{name: "ignore_then_include", rules: []string{"!dist/keep.txt", "dist/**"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			mustMkdirAll(t, filepath.Join(root, "dist"))
			writeTestFile(t, filepath.Join(root, "dist", "keep.txt"))
			writeTestFile(t, filepath.Join(root, "dist", "drop.txt"))

			cfg := project.Config{RootDir: root, CleanYes: true, CleanRules: strings.Join(tc.rules, ",")}
			if err := CleanRoot(cfg); err != nil {
				t.Fatalf("CleanRoot() error = %v", err)
			}

			assertExists(t, filepath.Join(root, "dist", "keep.txt"))
			assertNotExists(t, filepath.Join(root, "dist", "drop.txt"))
		})
	}
}

func TestCleanRoot_PatternVariations(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "configs"))
	mustMkdirAll(t, filepath.Join(root, "backend", "internal", "keep"))
	mustMkdirAll(t, filepath.Join(root, "somedir", "keep"))
	mustMkdirAll(t, filepath.Join(root, "somedir", "drop", "nested"))

	writeTestFile(t, filepath.Join(root, "configs", "config.json"))
	writeTestFile(t, filepath.Join(root, "configs", "config.yaml"))
	writeTestFile(t, filepath.Join(root, "configs", "note.txt"))
	writeTestFile(t, filepath.Join(root, "backend", "internal", "drop.txt"))
	writeTestFile(t, filepath.Join(root, "backend", "internal", "keep", "ok.txt"))
	writeTestFile(t, filepath.Join(root, "somedir", "keep", "keep.txt"))
	writeTestFile(t, filepath.Join(root, "somedir", "drop", "nested", "gone.txt"))

	cfg := project.Config{
		RootDir:  root,
		CleanYes: true,
		CleanRules: strings.Join([]string{
			"configs/config.*",
			"/backend/internal",
			"!/backend/internal/keep/**",
			"somedir/",
			"!somedir/keep/**",
		}, "\n"),
	}

	if err := CleanRoot(cfg); err != nil {
		t.Fatalf("CleanRoot() error = %v", err)
	}

	assertNotExists(t, filepath.Join(root, "configs", "config.json"))
	assertNotExists(t, filepath.Join(root, "configs", "config.yaml"))
	assertExists(t, filepath.Join(root, "configs", "note.txt"))
	assertNotExists(t, filepath.Join(root, "backend", "internal", "drop.txt"))
	assertExists(t, filepath.Join(root, "backend", "internal", "keep", "ok.txt"))
	assertNotExists(t, filepath.Join(root, "somedir", "drop", "nested", "gone.txt"))
	assertExists(t, filepath.Join(root, "somedir", "keep", "keep.txt"))
}

func TestPromptYesNo(t *testing.T) {
	var out bytes.Buffer
	ok, err := promptYesNo(strings.NewReader("y\n"), &out, "")
	if err != nil {
		t.Fatalf("promptYesNo(y) error = %v", err)
	}

	if !ok {
		t.Fatalf("promptYesNo(y) = false, want true")
	}
	ok, err = promptYesNo(strings.NewReader("no\n"), &out, "")
	if err != nil {
		t.Fatalf("promptYesNo(no) error = %v", err)
	}

	if ok {
		t.Fatalf("promptYesNo(no) = true, want false")
	}
}

func TestSummarizePlan_MinimizesDirectoriesAndCountsSize(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "dist", "nested"))
	writeTestFile(t, filepath.Join(root, "dist", "a.txt"))
	writeTestFile(t, filepath.Join(root, "dist", "nested", "b.txt"))
	writeTestFile(t, filepath.Join(root, "top.txt"))

	entries := []cleanEntry{
		{abs: filepath.Join(root, "dist"), rel: "dist", isDir: true},
		{abs: filepath.Join(root, "dist", "a.txt"), rel: "dist/a.txt", isDir: false},
		{abs: filepath.Join(root, "dist", "nested"), rel: "dist/nested", isDir: true},
		{abs: filepath.Join(root, "dist", "nested", "b.txt"), rel: "dist/nested/b.txt", isDir: false},
		{abs: filepath.Join(root, "top.txt"), rel: "top.txt", isDir: false},
	}

	display, files, dirs, bytes := summarizePlan(entries)
	if files != 3 || dirs != 2 || bytes <= 0 {
		t.Fatalf("unexpected summary files=%d dirs=%d bytes=%d", files, dirs, bytes)
	}

	if len(display) != 2 || display[0] != "dist/*" || display[1] != "top.txt" {
		t.Fatalf("display = %#v, want [dist/* top.txt]", display)
	}
}

func TestRuleMatches_CaseInsensitiveMode(t *testing.T) {
	prev := caseInsensitivePatterns
	caseInsensitivePatterns = true
	t.Cleanup(func() { caseInsensitivePatterns = prev })

	if !ruleMatches("Dist/File.TXT", false, cleanRule{include: true, pattern: "dist/file.txt"}) {
		t.Fatalf("expected case-insensitive file match")
	}

	if !ruleMatches("Backend/Internal", true, cleanRule{include: true, pattern: "backend/internal/"}) {
		t.Fatalf("expected case-insensitive dir pattern match")
	}
}

func TestRuleMatches_DirectoryPatternMatchesDescendants(t *testing.T) {
	if !ruleMatches("somedir", true, cleanRule{include: true, pattern: "somedir/"}) {
		t.Fatalf("dir should match itself")
	}

	if !ruleMatches("somedir/file.txt", false, cleanRule{include: true, pattern: "somedir/"}) {
		t.Fatalf("dir pattern should match descendant file")
	}

	if !ruleMatches("adirectory/another/somedir/file.txt", false, cleanRule{include: true, pattern: "somedir/"}) {
		t.Fatalf("dir pattern should match same-name directory at any depth")
	}

	if ruleMatches("other/file.txt", false, cleanRule{include: true, pattern: "somedir/"}) {
		t.Fatalf("dir pattern should not match unrelated path")
	}
}

func TestRuleMatches_AnchoredDirectoryPattern(t *testing.T) {
	r := cleanRule{include: true, anchored: true, pattern: "somedir/"}

	if !ruleMatches("somedir/file.txt", false, r) {
		t.Fatalf("anchored dir pattern should match root-level directory")
	}

	if ruleMatches("nested/somedir/file.txt", false, r) {
		t.Fatalf("anchored dir pattern should not match nested directory")
	}
}

func TestCleanRoot_LoadedRulesText_ParsedAndApplied(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "dist"))
	writeTestFile(t, filepath.Join(root, "dist", "keep.txt"))
	writeTestFile(t, filepath.Join(root, "dist", "drop.txt"))

	rulesText := strings.Join([]string{
		"# clean dist except keep",
		"dist/**",
		"!dist/keep.txt",
	}, "\n")
	cfg := project.Config{RootDir: root, CleanYes: true, CleanRules: rulesText}

	if err := CleanRoot(cfg); err != nil {
		t.Fatalf("CleanRoot() error = %v", err)
	}
	assertExists(t, filepath.Join(root, "dist", "keep.txt"))
	assertNotExists(t, filepath.Join(root, "dist", "drop.txt"))
}

func TestRuleMatches_DoubleStarPrefixSuffixPattern(t *testing.T) {
	r := cleanRule{include: true, pattern: "cache/**/tmp"}

	if !ruleMatches("cache/a/tmp", true, r) {
		t.Fatalf("pattern should match one-segment nested path")
	}

	if !ruleMatches("cache/a/b/c/tmp", true, r) {
		t.Fatalf("pattern should match multi-segment nested path")
	}

	if ruleMatches("other/cache/a/tmp", true, r) {
		t.Fatalf("pattern should not match non-prefixed path")
	}
}

func TestParseCleanRules_AnchoredAndEscapedSpacePatterns(t *testing.T) {
	rules, err := parseCleanRules(strings.Join([]string{
		"/dist/**",
		"!/dist/keep.txt",
		`dir\ with\ space/**`,
	}, "\n"))
	if err != nil {
		t.Fatalf("parseCleanRules() error = %v", err)
	}

	if len(rules) != 3 {
		t.Fatalf("rule count = %d, want 3", len(rules))
	}

	if !rules[0].anchored || rules[0].pattern != "dist/**" {
		t.Fatalf("rules[0] = %#v", rules[0])
	}

	if !rules[1].anchored || rules[1].include {
		t.Fatalf("rules[1] = %#v", rules[1])
	}

	if rules[2].pattern != `dir\ with\ space/**` {
		t.Fatalf("rules[2].pattern = %q", rules[2].pattern)
	}
}

func TestCleanRoot_SymlinkTargetOutsideRootNotDeleted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on Windows CI")
	}

	root := t.TempDir()
	outside := t.TempDir()
	writeTestFile(t, filepath.Join(outside, "keep.txt"))

	linkPath := filepath.Join(root, "outside-link")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	cfg := project.Config{
		RootDir:    root,
		CleanYes:   true,
		CleanRules: "outside-link",
	}

	if err := CleanRoot(cfg); err != nil {
		t.Fatalf("CleanRoot() error = %v", err)
	}
	assertNotExists(t, linkPath)
	assertExists(t, filepath.Join(outside, "keep.txt"))
}

func TestPrintCleanPlan_StableOrderingAndCounts(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "zdir"))
	mustMkdirAll(t, filepath.Join(root, "adir", "nested"))
	writeSizedFile(t, filepath.Join(root, "zdir", "z.txt"), 2)
	writeSizedFile(t, filepath.Join(root, "adir", "nested", "a.txt"), 3)
	writeSizedFile(t, filepath.Join(root, "top.txt"), 5)

	entries := []cleanEntry{
		{abs: filepath.Join(root, "zdir", "z.txt"), rel: "zdir/z.txt", isDir: false},
		{abs: filepath.Join(root, "adir"), rel: "adir", isDir: true},
		{abs: filepath.Join(root, "adir", "nested"), rel: "adir/nested", isDir: true},
		{abs: filepath.Join(root, "adir", "nested", "a.txt"), rel: "adir/nested/a.txt", isDir: false},
		{abs: filepath.Join(root, "top.txt"), rel: "top.txt", isDir: false},
	}
	display, files, dirs, totalBytes := summarizePlan(entries)
	plan := cleanPlan{
		display: display,
		files:   files,
		dirs:    dirs,
		bytes:   totalBytes,
	}

	var out bytes.Buffer
	printCleanPlan(&out, plan)
	text := out.String()
	if !strings.Contains(text, "Clean plan: 3 files, 2 dirs, 10 B") {
		t.Fatalf("unexpected header output: %q", text)
	}
	expectedOrder := []string{"  - adir/*", "  - top.txt", "  - zdir/z.txt"}
	last := -1
	for _, needle := range expectedOrder {
		idx := strings.Index(text, needle)
		if idx == -1 {
			t.Fatalf("missing %q in output %q", needle, text)
		}
		if idx < last {
			t.Fatalf("output order is not stable: %q before prior item", needle)
		}
		last = idx
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected path to exist %q: %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected path to be removed %q, err=%v", path, err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", path, err)
	}
}

func writeSizedFile(t *testing.T, path string, size int) {
	t.Helper()
	data := strings.Repeat("x", size)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %q (%s): %v", path, fmt.Sprintf("%d bytes", size), err)
	}
}
