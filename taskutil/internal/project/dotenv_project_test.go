package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv_LoadsExistingFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env.taskutil"), []byte("FOO=one\n"), 0o644); err != nil {
		t.Fatalf("write .env.taskutil: %v", err)
	}
	ensureUnset(t, "FOO")
	if err := LoadDotEnv(root); err != nil {
		t.Fatalf("LoadDotEnv() error = %v", err)
	}
	if got := os.Getenv("FOO"); got != "one" {
		t.Fatalf("FOO = %q, want one", got)
	}
}

func TestLooksLikeRepoRoot(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "backend"))
	mustMkdir(t, filepath.Join(root, "frontend"))
	mustMkdir(t, filepath.Join(root, "taskutil"))
	if err := os.WriteFile(filepath.Join(root, "Taskfile.yml"), []byte("version: '3'\n"), 0o644); err != nil {
		t.Fatalf("write Taskfile.yml: %v", err)
	}
	if !looksLikeRepoRoot(root, []string{"Taskfile.yml", "backend", "frontend", "taskutil"}) {
		t.Fatalf("looksLikeRepoRoot(%q) = false, want true", root)
	}
}

func TestRootMarkerList(t *testing.T) {
	got := (Config{RootMarkers: "Taskfile.yml,app,web"}).RootMarkerList()
	if len(got) != 3 || got[0] != "Taskfile.yml" || got[1] != "app" || got[2] != "web" {
		t.Fatalf("RootMarkerList() = %#v", got)
	}
}

func TestRootMarkerList_Empty(t *testing.T) {
	got := (Config{RootMarkers: " , "}).RootMarkerList()
	if len(got) != 0 {
		t.Fatalf("RootMarkerList() = %#v, want empty", got)
	}
}

func TestResolveRootDir_UsesEnvOverride(t *testing.T) {
	root := t.TempDir()
	got, err := ResolveRootDir(Config{RootOverride: root, RootMarkers: "Taskfile.yml"})
	if err != nil {
		t.Fatalf("ResolveRootDir() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveRootDir() = %q, want %q", got, root)
	}
}

func TestResolveRootDir_WalksUpUsingMarkers(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "app"))
	mustMkdir(t, filepath.Join(root, "web"))
	if err := os.WriteFile(filepath.Join(root, "Taskfile.yml"), []byte("version: '3'\n"), 0o644); err != nil {
		t.Fatalf("write Taskfile.yml: %v", err)
	}
	nested := filepath.Join(root, "web", "nested", "dir")
	mustMkdir(t, nested)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir nested: %v", err)
	}

	got, err := ResolveRootDir(Config{RootMarkers: "Taskfile.yml,app,web"})
	if err != nil {
		t.Fatalf("ResolveRootDir() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveRootDir() = %q, want %q", got, root)
	}
}

func TestResolveRootDir_UsesExecutableFallback(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "app"))
	if err := os.WriteFile(filepath.Join(root, "Taskfile.yml"), []byte("version: '3'\n"), 0o644); err != nil {
		t.Fatalf("write Taskfile.yml: %v", err)
	}

	other := t.TempDir()
	prevGetwd := getwdFn
	prevExecutable := executableFn
	getwdFn = func() (string, error) { return other, nil }
	executableFn = func() (string, error) {
		return filepath.Join(root, ".tmp", "bin", "taskutil"), nil
	}
	t.Cleanup(func() {
		getwdFn = prevGetwd
		executableFn = prevExecutable
	})

	got, err := ResolveRootDir(Config{RootMarkers: "Taskfile.yml,app"})
	if err != nil {
		t.Fatalf("ResolveRootDir() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveRootDir() = %q, want %q", got, root)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", path, err)
	}
}

func ensureUnset(t *testing.T, key string) {
	t.Helper()
	prev, had := os.LookupEnv(key)
	if had {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
			return
		}
		_ = os.Unsetenv(key)
	})
}
