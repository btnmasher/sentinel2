package project

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseConfig_ResolvesPathsAndDefaults(t *testing.T) {
	root := t.TempDir()
	cfg, cmd, err := ParseConfig([]string{"dev"}, root)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cmd != "dev" {
		t.Fatalf("command = %q, want dev", cmd)
	}

	if cfg.FrontendDir() != filepath.Join(root, "frontend") {
		t.Fatalf("FrontendDir() = %q", cfg.FrontendDir())
	}

	if cfg.BackendDir() != filepath.Join(root, "backend") {
		t.Fatalf("BackendDir() = %q", cfg.BackendDir())
	}

	if cfg.BinDir() != filepath.Join(root, "bin") {
		t.Fatalf("BinDir() = %q", cfg.BinDir())
	}
	expectedBin := filepath.Join(root, "bin", "app-server")
	if runtime.GOOS == "windows" {
		expectedBin += ".exe"
	}

	if cfg.BackendBinary() != expectedBin {
		t.Fatalf("BackendBinary() = %q, want %q", cfg.BackendBinary(), expectedBin)
	}
}

func TestParseConfig_BackendBinaryOverride(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TASKUTIL_BACKEND_BIN_PATH", "custom/server")
	cfg, _, err := ParseConfig([]string{"dev"}, root)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	want := filepath.Join(root, "custom", "server")
	if cfg.BackendBinary() != want {
		t.Fatalf("BackendBinary() = %q, want %q", cfg.BackendBinary(), want)
	}
}

func TestParseConfig_DefaultBackendBinUsesAppName(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TASKUTIL_APP_NAME", "demo")
	cfg, _, err := ParseConfig([]string{"dev"}, root)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	want := filepath.Join(root, "bin", "demo-server")
	if runtime.GOOS == "windows" {
		want += ".exe"
	}

	if cfg.BackendBinary() != want {
		t.Fatalf("BackendBinary() = %q, want %q", cfg.BackendBinary(), want)
	}
}

func TestParseConfig_EnvOverrides(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TASKUTIL_BACKEND_BIN_NAME", "my-server")
	t.Setenv("LOG_DIR", "var/log/dev")

	cfg, _, err := ParseConfig([]string{"dev"}, root)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.BackendBinary() != filepath.Join(root, "bin", "my-server") {
		t.Fatalf("BackendBinary() = %q", cfg.BackendBinary())
	}

	if cfg.LogDir != "var/log/dev" {
		t.Fatalf("LogDir = %q", cfg.LogDir)
	}
}

func TestResolvePath(t *testing.T) {
	root := t.TempDir()
	got := resolvePath(root, "a/../b")
	want := filepath.Join(root, "b")
	if got != want {
		t.Fatalf("resolvePath() = %q, want %q", got, want)
	}
}

func TestParseConfig_ExperimentalPTYFlag(t *testing.T) {
	root := t.TempDir()
	t.Setenv("EXPERIMENTAL_PTY", "1")
	cfg, _, err := ParseConfig([]string{"dev"}, root)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if !cfg.ExperimentalPTY {
		t.Fatalf("ExperimentalPTY = false, want true")
	}
}

func TestParseConfig_RejectsEmptyRootMarkers(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TASKUTIL_ROOT_SIGNATURE", " , ")
	_, _, err := ParseConfig([]string{"dev"}, root)
	if err == nil {
		t.Fatalf("ParseConfig() expected error for empty root markers")
	}
}

func TestParseBootstrap_ReadsEnv(t *testing.T) {
	t.Setenv("TASKUTIL_ROOT", "/tmp/example-root")
	t.Setenv("TASKUTIL_ROOT_SIGNATURE", "Taskfile.yml,app")
	cfg, err := ParseBootstrap([]string{"dev"})
	if err != nil {
		t.Fatalf("ParseBootstrap() error = %v", err)
	}

	if cfg.RootOverride != "/tmp/example-root" {
		t.Fatalf("RootOverride = %q", cfg.RootOverride)
	}

	if cfg.RootMarkers != "Taskfile.yml,app" {
		t.Fatalf("RootMarkers = %q", cfg.RootMarkers)
	}
}

func TestParseConfig_LoadsDefaultCleanRulesFile(t *testing.T) {
	root := t.TempDir()
	content := "dist/**\n!dist/keep/**\n"
	if err := os.WriteFile(filepath.Join(root, ".cleanrules"), []byte(content), 0o644); err != nil {
		t.Fatalf("write .cleanrules: %v", err)
	}
	cfg, _, err := ParseConfig([]string{"dev"}, root)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.CleanRules != content {
		t.Fatalf("CleanRules = %q, want %q", cfg.CleanRules, content)
	}
}

func TestParseConfig_CleanRulesFileOverride(t *testing.T) {
	root := t.TempDir()
	alt := filepath.Join(root, "custom.rules")
	content := "bin/**\n!bin/.env\n"
	if err := os.WriteFile(alt, []byte(content), 0o644); err != nil {
		t.Fatalf("write custom.rules: %v", err)
	}
	t.Setenv("TASKUTIL_CLEAN_RULES_FILE", alt)
	cfg, _, err := ParseConfig([]string{"dev"}, root)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.CleanRules != content {
		t.Fatalf("CleanRules = %q, want %q", cfg.CleanRules, content)
	}
}

func TestParseConfig_MissingCustomCleanRulesFileErrors(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TASKUTIL_CLEAN_RULES_FILE", filepath.Join(root, "missing.rules"))
	_, _, err := ParseConfig([]string{"dev"}, root)
	if err == nil {
		t.Fatalf("ParseConfig() expected error for missing custom clean rules file")
	}
}
