package project

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func LoadDotEnv(rootDir string) error {
	paths := []string{
		filepath.Join(rootDir, ".env.taskutil"),
	}
	existing := make([]string, 0, len(paths))
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			existing = append(existing, p)
		}
	}
	if len(existing) == 0 {
		return nil
	}
	return godotenv.Load(existing...)
}
