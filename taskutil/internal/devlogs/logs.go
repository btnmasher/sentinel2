package devlogs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"sentinel2-taskutil/internal/project"
)

const (
	defaultTailLines = 200
	tailPollInterval = 400 * time.Millisecond
	hoursPerDay      = 24
	yearPrefix       = "20"
	unixTmpPrefix    = "/tmp/"
)

func Tail(cfg project.Config) error {
	logDir, err := resolveDevLogDir(cfg)
	if err != nil {
		return err
	}
	lines := cfg.TailLines
	if lines <= 0 {
		lines = defaultTailLines
	}
	files := []struct {
		label string
		path  string
	}{
		{label: "vite", path: filepath.Join(logDir, "vite.log")},
		{label: "backend", path: filepath.Join(logDir, "backend.log")},
	}
	offsets := map[string]int64{}
	for _, f := range files {
		content, readErr := os.ReadFile(f.path)
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				continue
			}
			return readErr
		}
		linesOut := splitLines(string(content))
		linesOut = linesOut[max(0, len(linesOut)-lines):]
		for _, line := range linesOut {
			fmt.Printf("[%s] %s\n", f.label, line)
		}
		offsets[f.path] = int64(len(content))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ticker := time.NewTicker(tailPollInterval)
	defer ticker.Stop()
	remainders := map[string]string{}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for _, f := range files {
				next, rem, tailErr := readNewLines(f.path, offsets[f.path], remainders[f.path], f.label)
				if tailErr != nil {
					if errors.Is(tailErr, os.ErrNotExist) {
						continue
					}
					return tailErr
				}
				offsets[f.path] = next
				remainders[f.path] = rem
			}
		}
	}
}

func Clean(cfg project.Config) error {
	baseLogDir := BaseLogDir(cfg)
	keepDays := cfg.KeepDays
	keepDays = max(keepDays, 0)
	entries, err := os.ReadDir(baseLogDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	cutoff := time.Now().Add(-time.Duration(keepDays) * hoursPerDay * time.Hour)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, yearPrefix) {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		path := filepath.Join(baseLogDir, name)
		fmt.Println(path)
		if remErr := os.RemoveAll(path); remErr != nil {
			return remErr
		}
	}
	return nil
}

func BaseLogDir(cfg project.Config) string {
	baseLogDir := strings.TrimSpace(cfg.LogDir)
	if baseLogDir == "" {
		baseLogDir = "dev-logs"
	}
	if runtime.GOOS == "windows" && strings.HasPrefix(baseLogDir, unixTmpPrefix) {
		baseLogDir = filepath.Join(os.TempDir(), filepath.Base(baseLogDir))
	}
	if !filepath.IsAbs(baseLogDir) {
		baseLogDir = filepath.Join(cfg.RootDir, baseLogDir)
	}
	return baseLogDir
}

func resolveDevLogDir(cfg project.Config) (string, error) {
	baseLogDir := BaseLogDir(cfg)
	latestPath := filepath.Join(baseLogDir, "latest")
	if data, err := os.ReadFile(latestPath); err == nil {
		p := strings.TrimSpace(string(data))
		if p != "" {
			if info, statErr := os.Stat(p); statErr == nil && info.IsDir() {
				return p, nil
			}
		}
	}
	if info, err := os.Stat(baseLogDir); err == nil && info.IsDir() {
		entries, readErr := os.ReadDir(baseLogDir)
		if readErr == nil {
			dirs := make([]string, 0, len(entries))
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), yearPrefix) {
					dirs = append(dirs, entry.Name())
				}
			}
			sort.Strings(dirs)
			if len(dirs) > 0 {
				return filepath.Join(baseLogDir, dirs[len(dirs)-1]), nil
			}
		}
		return baseLogDir, nil
	}
	return "", fmt.Errorf("no dev logs found at %s", baseLogDir)
}

func readNewLines(path string, offset int64, remainder string, label string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return offset, remainder, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return offset, remainder, err
	}
	size := info.Size()
	offset = min(offset, size)
	if _, err = file.Seek(offset, io.SeekStart); err != nil {
		return offset, remainder, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return offset, remainder, err
	}
	if len(data) == 0 {
		return size, remainder, nil
	}
	chunk := remainder + string(data)
	lines := splitLinesKeepRemainder(chunk)
	for _, line := range lines.lines {
		fmt.Printf("[%s] %s\n", label, line)
	}
	return size, lines.remainder, nil
}

type lineSplitResult struct {
	lines     []string
	remainder string
}

func splitLinesKeepRemainder(s string) lineSplitResult {
	if s == "" {
		return lineSplitResult{}
	}
	hasTrailingNewline := strings.HasSuffix(s, "\n")
	raw := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if hasTrailingNewline {
		raw = raw[:len(raw)-1]
		return lineSplitResult{lines: raw}
	}
	if len(raw) == 0 {
		return lineSplitResult{}
	}
	return lineSplitResult{
		lines:     raw[:len(raw)-1],
		remainder: raw[len(raw)-1],
	}
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
