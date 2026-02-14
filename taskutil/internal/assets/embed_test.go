package assets

import (
	"os"
	"path/filepath"
	"testing"

	"sentinel2-taskutil/internal/project"
)

func TestCopyDirContents(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")
	mustMkdirAll(t, filepath.Join(src, "a", "b"))
	if err := os.WriteFile(filepath.Join(src, "a", "b", "x.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src file: %v", err)
	}
	if err := copyDirContents(src, dst); err != nil {
		t.Fatalf("copyDirContents() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "a", "b", "x.txt"))
	if err != nil {
		t.Fatalf("read dst file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("copied content = %q, want hello", string(data))
	}
}

func TestPrepareEmbed_MissingSource(t *testing.T) {
	root := t.TempDir()
	cfg := project.Config{
		RootDir:       root,
		EmbedSrcPath:  filepath.Join(root, "missing"),
		EmbedDestPath: filepath.Join(root, "backend", "internal", "web", "dist"),
	}
	err := PrepareEmbed(cfg)
	if err == nil {
		t.Fatalf("PrepareEmbed() error = nil, want non-nil")
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", path, err)
	}
}
