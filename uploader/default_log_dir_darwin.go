//go:build darwin

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
	return filepath.Join(
		home,
		"Library",
		"Application Support",
		"EVE Online",
		"p_drive",
		"User",
		"My Documents",
		"EVE",
		"logs",
		"Chatlogs",
	)
}
