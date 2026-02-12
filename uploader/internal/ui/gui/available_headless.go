//go:build headless

package gui

import "sentinel2-uploader/internal/config"

func Available() bool {
	return false
}

func Run(_ string, _ config.Options) {}
