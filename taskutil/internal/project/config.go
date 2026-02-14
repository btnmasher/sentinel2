package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	flags "github.com/jessevdk/go-flags"
)

type Config struct {
	RootDir string

	RootOverride string `long:"taskutil-root" env:"TASKUTIL_ROOT" description:"Explicit repository root path. Skips auto-discovery when set."`
	RootMarkers  string `long:"taskutil-root-signature" env:"TASKUTIL_ROOT_SIGNATURE" default:"Taskfile.yml,backend,frontend,taskutil" description:"Comma-separated files/directories that together identify repository root during auto-discovery."`

	AppName string `long:"taskutil-app-name" env:"TASKUTIL_APP_NAME" default:"app" description:"Application identifier used to derive backend binary name when no explicit backend binary is configured."`

	FrontendDirPath string `long:"taskutil-frontend-dir" env:"TASKUTIL_FRONTEND_DIR" default:"frontend" description:"Path to frontend project directory (absolute or relative to repository root)."`
	BackendDirPath  string `long:"taskutil-backend-dir" env:"TASKUTIL_BACKEND_DIR" default:"backend" description:"Path to backend project directory (absolute or relative to repository root)."`
	BinDirPath      string `long:"taskutil-bin-dir" env:"TASKUTIL_BIN_DIR" default:"bin" description:"Path to build output directory (absolute or relative to repository root)."`
	BackendBinName  string `long:"taskutil-backend-bin-name" env:"TASKUTIL_BACKEND_BIN_NAME" description:"Backend executable file name, without path. Overrides app-name derived default."`
	BackendBinPath  string `long:"taskutil-backend-bin-path" env:"TASKUTIL_BACKEND_BIN_PATH" description:"Backend executable full path. Overrides taskutil-backend-bin-name and app-name derived defaults."`
	EmbedSrcPath    string `long:"taskutil-embed-src" env:"TASKUTIL_EMBED_SRC" default:"frontend/dist" description:"Source directory for frontend embed assets (absolute or relative to repository root)."`
	EmbedDestPath   string `long:"taskutil-embed-dest" env:"TASKUTIL_EMBED_DEST" default:"backend/internal/web/dist" description:"Destination directory for embedded frontend assets (absolute or relative to repository root)."`

	LogDir string `long:"log-dir" env:"LOG_DIR" default:"dev-logs" description:"Directory used for dev supervisor session logs."`

	ViteHost       string `long:"vite-host" env:"VITE_HOST" default:"127.0.0.1" description:"Vite host value for frontend dev server startup."`
	VitePort       string `long:"vite-port" env:"VITE_PORT" default:"5173" description:"Vite port value for frontend dev server startup."`
	DevProxy       string `long:"dev-proxy" env:"DEV_PROXY" default:"127.0.0.1:5173" description:"Backend DEV_PROXY value that points to the frontend dev server."`
	LogJSONPath    string `long:"log-json-path" env:"LOG_JSON_PATH" description:"Optional backend JSON log output path. Defaults to backend.jsonl under the current session log directory."`
	CleanRulesFile string `long:"clean-rules-file" env:"TASKUTIL_CLEAN_RULES_FILE" description:"Path to cleanup rules file. Defaults to .cleanrules at repository root when present."`
	CleanRules     string `no-flag:"true"`
	CleanYes       bool   `long:"yes" env:"TASKUTIL_CLEAN_YES" description:"Skip clean-root confirmation prompt and proceed with deletion."`

	DevMigrations   bool `long:"dev-migrations" env:"DEV_MIGRATIONS" description:"Run backend migrate before starting dev processes."`
	ExperimentalPTY bool `long:"experimental-pty" env:"EXPERIMENTAL_PTY" description:"Enable experimental PTY process mode with stdio pipe fallback."`
	TailLines       int  `long:"tail-lines" env:"TAIL_LINES" default:"200" description:"Number of recent log lines to print at startup for dev-logs-tail."`
	KeepDays        int  `long:"keep-days" env:"KEEP_DAYS" default:"7" description:"Retention window in days for cleaning old dev log directories."`
}

type parseInput struct {
	Config
	Positional struct {
		Command string `positional-arg-name:"command" required:"yes"`
	} `positional-args:"yes"`
}

func ParseConfig(args []string, rootDir string) (Config, string, error) {
	input := parseInput{Config: Config{RootDir: rootDir}}
	parser := flags.NewParser(&input, flags.Default)
	_, err := parser.ParseArgs(args)
	if err != nil {
		return Config{}, "", err
	}
	cfg := input.Config
	cfg.RootDir = rootDir
	if err := cfg.resolvePaths(); err != nil {
		return Config{}, "", err
	}
	return cfg, input.Positional.Command, nil
}

func ParseBootstrap(args []string) (Config, error) {
	input := Config{}
	parser := flags.NewParser(&input, flags.IgnoreUnknown)
	_, err := parser.ParseArgs(args)
	if err != nil {
		return Config{}, err
	}
	return input, nil
}

func (c *Config) resolvePaths() error {
	c.RootMarkers = normalizeCSV(c.RootMarkers)
	if c.RootMarkers == "" {
		return fmt.Errorf("taskutil-root-signature cannot be empty")
	}
	c.FrontendDirPath = resolvePath(c.RootDir, c.FrontendDirPath)
	c.BackendDirPath = resolvePath(c.RootDir, c.BackendDirPath)
	c.BinDirPath = resolvePath(c.RootDir, c.BinDirPath)
	c.EmbedSrcPath = resolvePath(c.RootDir, c.EmbedSrcPath)
	c.EmbedDestPath = resolvePath(c.RootDir, c.EmbedDestPath)
	if c.BackendBinPath == "" {
		name := strings.TrimSpace(c.BackendBinName)
		if name == "" {
			name = defaultBackendBinName(c.AppName)
		}
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		c.BackendBinPath = filepath.Join(c.BinDirPath, name)
	} else {
		c.BackendBinPath = resolvePath(c.RootDir, c.BackendBinPath)
	}
	if strings.TrimSpace(c.LogDir) == "" {
		return fmt.Errorf("log-dir cannot be empty")
	}
	if err := c.loadCleanRules(); err != nil {
		return err
	}
	return nil
}

func resolvePath(rootDir, value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(rootDir, value))
}

func defaultBackendBinName(appName string) string {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		return "server"
	}
	return appName + "-server"
}

func (c Config) RootMarkerList() []string {
	parts := strings.Split(normalizeCSV(c.RootMarkers), ",")
	markers := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			markers = append(markers, p)
		}
	}
	return markers
}

func normalizeCSV(s string) string {
	parts := strings.Split(s, ",")
	normalized := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			normalized = append(normalized, p)
		}
	}
	return strings.Join(normalized, ",")
}

func (c Config) defaultCleanRules() string {
	embedSrcRel := pathRelativeToRoot(c.RootDir, c.EmbedSrcPath)
	embedDestRel := pathRelativeToRoot(c.RootDir, c.EmbedDestPath)
	return strings.Join([]string{
		"dist",
		embedSrcRel,
		embedDestRel,
		".tmp/go-build-cache",
		".tmp/golangci-lint-cache",
		".tmp/bun",
		".tmp/bun-install",
		".tmp/stamps",
		".task",
		".tmp/bin/**",
		"!.tmp/bin/taskutil",
		"!.tmp/bin/taskutil.exe",
		"bin/**",
		"!bin/pb_data",
		"!bin/pb_data/**",
		"!bin/.env",
	}, ",")
}

func pathRelativeToRoot(rootDir, value string) string {
	abs := resolvePath(rootDir, value)
	if rel, err := filepath.Rel(rootDir, abs); err == nil {
		return filepath.ToSlash(filepath.Clean(rel))
	}
	return filepath.ToSlash(filepath.Clean(value))
}

func (c *Config) loadCleanRules() error {
	if strings.TrimSpace(c.CleanRules) != "" {
		return nil
	}
	customFile := strings.TrimSpace(c.CleanRulesFile) != ""
	rulesFile := strings.TrimSpace(c.CleanRulesFile)
	if rulesFile == "" {
		rulesFile = ".cleanrules"
	}
	rulesFile = resolvePath(c.RootDir, rulesFile)
	data, err := os.ReadFile(rulesFile)
	if err == nil {
		c.CleanRules = string(data)
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read clean rules file %s: %w", rulesFile, err)
	}
	if customFile && errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clean rules file not found: %s", rulesFile)
	}
	c.CleanRules = c.defaultCleanRules()
	return nil
}

func (c Config) FrontendDir() string { return c.FrontendDirPath }
func (c Config) BackendDir() string  { return c.BackendDirPath }
func (c Config) BinDir() string      { return c.BinDirPath }
func (c Config) BackendBinary() string {
	return c.BackendBinPath
}
func (c Config) EmbedSrc() string  { return c.EmbedSrcPath }
func (c Config) EmbedDest() string { return c.EmbedDestPath }
