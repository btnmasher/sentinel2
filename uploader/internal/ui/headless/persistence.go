package headless

import "sentinel2-uploader/internal/config"

func mergeHeadlessOptions(cli config.Options, saved config.UploaderSettings) config.Options {
	return config.MergeOptionsWithSettings(cli, saved)
}

func loadPersistedOptions() (config.UploaderSettings, error) {
	return config.LoadSettings()
}

func savePersistedOptions(opts config.Options) error {
	settings := config.SettingsFromOptions(opts)
	if saved, err := config.LoadSettings(); err == nil {
		settings.MinimizeToTray = saved.MinimizeToTray
		settings.StartMinimized = saved.StartMinimized
	}
	return config.SaveSettings(settings)
}
