package devlogs

import (
	"os"
	"path/filepath"
	"testing"

	"sentinel2-taskutil/internal/project"
)

func TestSplitLines(t *testing.T) {
	got := splitLines("a\r\nb\n")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("splitLines() = %#v, want [a b]", got)
	}
}

func TestSplitLinesKeepRemainder(t *testing.T) {
	got := splitLinesKeepRemainder("a\r\nb")
	if len(got.lines) != 1 || got.lines[0] != "a" || got.remainder != "b" {
		t.Fatalf("splitLinesKeepRemainder() = %#v", got)
	}

	got = splitLinesKeepRemainder("a\nb\n")
	if len(got.lines) != 2 || got.remainder != "" {
		t.Fatalf("splitLinesKeepRemainder() trailing newline = %#v", got)
	}
}

func TestReadNewLines_RespectsOffsetAndRemainder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.log")
	if err := os.WriteFile(path, []byte("a\nbc"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	next, rem, err := readNewLines(path, 0, "", "x")
	if err != nil {
		t.Fatalf("readNewLines() error = %v", err)
	}

	if next <= 0 {
		t.Fatalf("next offset = %d, want > 0", next)
	}

	if rem != "bc" {
		t.Fatalf("remainder = %q, want bc", rem)
	}

	if err := os.WriteFile(path, []byte("a\nbcX\n"), 0o644); err != nil {
		t.Fatalf("append file: %v", err)
	}
	next2, rem2, err := readNewLines(path, next, rem, "x")
	if err != nil {
		t.Fatalf("readNewLines() error with appended data = %v", err)
	}

	if next2 <= next {
		t.Fatalf("next2 offset = %d, want > %d", next2, next)
	}

	if rem2 != "" {
		t.Fatalf("remainder after full lines = %q, want empty", rem2)
	}
}

func TestResolveDevLogDir_UsesLatestThenFallback(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "logs")
	cfg := project.Config{RootDir: root, LogDir: "logs"}

	mustMkdirAll(t, filepath.Join(base, "20260101-010101"))
	mustMkdirAll(t, filepath.Join(base, "20260102-010101"))
	latest := filepath.Join(base, "20260101-010101")
	if err := os.WriteFile(filepath.Join(base, "latest"), []byte(latest), 0o644); err != nil {
		t.Fatalf("write latest: %v", err)
	}
	got, err := resolveDevLogDir(cfg)
	if err != nil {
		t.Fatalf("resolveDevLogDir() error = %v", err)
	}

	if got != latest {
		t.Fatalf("resolveDevLogDir() = %q, want %q", got, latest)
	}

	if err := os.WriteFile(filepath.Join(base, "latest"), []byte(filepath.Join(base, "missing")), 0o644); err != nil {
		t.Fatalf("rewrite latest: %v", err)
	}
	got, err = resolveDevLogDir(cfg)
	if err != nil {
		t.Fatalf("resolveDevLogDir() fallback error = %v", err)
	}
	want := filepath.Join(base, "20260102-010101")
	if got != want {
		t.Fatalf("resolveDevLogDir() fallback = %q, want %q", got, want)
	}
}

func TestBaseLogDir_RelativeAndAbsolute(t *testing.T) {
	root := t.TempDir()

	rel := BaseLogDir(project.Config{RootDir: root, LogDir: "logs/dev"})
	if rel != filepath.Join(root, "logs", "dev") {
		t.Fatalf("BaseLogDir(relative) = %q", rel)
	}

	abs := filepath.Join(root, "abs", "logs")
	gotAbs := BaseLogDir(project.Config{RootDir: root, LogDir: abs})
	if gotAbs != abs {
		t.Fatalf("BaseLogDir(absolute) = %q, want %q", gotAbs, abs)
	}
}

func TestBaseLogDir_Default(t *testing.T) {
	root := t.TempDir()
	got := BaseLogDir(project.Config{RootDir: root})
	want := filepath.Join(root, "dev-logs")
	if got != want {
		t.Fatalf("BaseLogDir(default) = %q, want %q", got, want)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", path, err)
	}
}
