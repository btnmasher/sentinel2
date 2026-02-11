//go:build windows

package main

import (
	"os"
	"path/filepath"
)

func defaultLogDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, "Documents", "EVE", "logs", "Chatlogs")
}
