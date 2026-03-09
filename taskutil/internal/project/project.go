package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	getwdFn      = os.Getwd
	executableFn = os.Executable
)

func ResolveRootDir(cfg Config) (string, error) {
	if explicit := strings.TrimSpace(cfg.RootOverride); explicit != "" {
		return filepath.Abs(explicit)
	}
	start, err := getwdFn()
	if err != nil {
		return "", err
	}
	dir := start
	markers := cfg.RootMarkerList()
	for {
		if looksLikeRepoRoot(dir, markers) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if exe, exeErr := executableFn(); exeErr == nil {
		exeDir := filepath.Dir(exe)
		candidate := filepath.Clean(filepath.Join(exeDir, "..", ".."))
		if looksLikeRepoRoot(candidate, markers) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not resolve project root from %s", start)
}

func looksLikeRepoRoot(dir string, markers []string) bool {
	if dir == "" {
		return false
	}
	for _, marker := range markers {
		p := filepath.Join(dir, marker)
		if _, err := os.Stat(p); err != nil {
			return false
		}
	}
	return true
}
