package cleanup

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func scanRoot(rootDir string) ([]cleanEntry, error) {
	entries := make([]cleanEntry, 0, 256)
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == rootDir {
			return nil
		}
		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, cleanEntry{
			abs:   path,
			rel:   filepath.ToSlash(filepath.Clean(rel)),
			isDir: d.IsDir(),
		})
		return nil
	})
	return entries, err
}

func hasIgnoredDescendant(rel string, ignored map[string]struct{}) bool {
	prefix := rel + "/"
	for p := range ignored {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

func depth(rel string) int {
	if rel == "" {
		return 0
	}
	return strings.Count(rel, "/") + 1
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
