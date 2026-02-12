//go:build !headless

package gui

import (
	"fyne.io/fyne/v2"

	"sentinel2-uploader/internal/config"
)

func applyPreferenceDefaults(prefs fyne.Preferences, opts config.Options) config.Options {
	opts.BaseURL = prefs.StringWithFallback(prefBaseURL, opts.BaseURL)
	opts.Token = prefs.StringWithFallback(prefToken, opts.Token)
	opts.LogDir = prefs.StringWithFallback(prefLogDir, opts.LogDir)
	opts.AutoConnect = prefs.BoolWithFallback(prefAutoConnect, opts.AutoConnect)
	opts.LogFile = ""
	if opts.LogDir == "" {
		opts.LogDir = config.DefaultLogDir()
	}
	return opts
}
