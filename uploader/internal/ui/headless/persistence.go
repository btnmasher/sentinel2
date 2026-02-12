package headless

import (
	"encoding/json"
	"os"
	"path/filepath"

	"sentinel2-uploader/internal/config"
)

type persistedOptions struct {
	BaseURL     string `json:"base_url"`
	Token       string `json:"token"`
	LogDir      string `json:"log_dir"`
	AutoConnect bool   `json:"auto_connect"`
}

func mergeHeadlessOptions(cli config.Options, saved persistedOptions) config.Options {
	if cli.BaseURL == "" {
		cli.BaseURL = saved.BaseURL
	}
	if cli.Token == "" {
		cli.Token = saved.Token
	}
	if cli.LogDir == "" {
		cli.LogDir = saved.LogDir
	}
	if !cli.AutoConnect {
		cli.AutoConnect = saved.AutoConnect
	}
	cli.LogFile = ""
	return cli
}

func settingsPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "sentinel2", "uploader-headless.json"), nil
}

func loadPersistedOptions() (persistedOptions, error) {
	path, err := settingsPath()
	if err != nil {
		return persistedOptions{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return persistedOptions{}, err
	}
	var saved persistedOptions
	if err := json.Unmarshal(data, &saved); err != nil {
		return persistedOptions{}, err
	}
	return saved, nil
}

func savePersistedOptions(opts config.Options) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(persistedOptions{
		BaseURL:     opts.BaseURL,
		Token:       opts.Token,
		LogDir:      opts.LogDir,
		AutoConnect: opts.AutoConnect,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}
