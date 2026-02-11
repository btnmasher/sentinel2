package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"sentinel2/internal/config"
	"sentinel2/internal/logging"
	"sentinel2/internal/server"
	_ "sentinel2/pb_migrations"
)

var BuildVersion = ""

func main() {
	loadEnvFiles()

	cfg := config.Load()
	cfg.SentinelVersion = BuildVersion
	cfg.EnsureESIUserAgent()
	logging.Configure(logging.Options{
		MinLevel:             logging.ParseLevel(cfg.LogLevel),
		PrettyEnabled:        cfg.LogPretty || cfg.DebugEnabled,
		UsePocketBasePrinter: cfg.LogPrettyPB,
		JSONEnabled:          cfg.LogJSON,
		JSONPath:             cfg.LogJSONPath,
		UsePocketBaseJSON:    cfg.LogJSONPB,
	})
	logAuthConfig(cfg)
	if err := server.Run(cfg); err != nil {
		log.Fatal(err)
	}
}

func loadEnvFiles() {
	cwd, err := os.Getwd()
	if err != nil {
		_ = godotenv.Load()
		return
	}

	loadList := findEnvFiles(cwd)
	if exePath, exeErr := os.Executable(); exeErr == nil {
		exeDir := filepath.Dir(exePath)
		loadList = append(loadList, findEnvFiles(exeDir)...)
	}
	loadList = uniqueExistingFiles(loadList)
	loadEnvFilesFromList(loadList)
}

func findEnvFiles(dir string) []string {
	if dir == "" {
		return nil
	}
	return []string{
		filepath.Join(dir, ".env"),
		filepath.Join(dir, ".env.local"),
	}
}

func uniqueExistingFiles(paths []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	return out
}

func loadEnvFilesFromList(loadList []string) {
	if len(loadList) == 0 {
		return
	}
	if err := godotenv.Load(loadList...); err != nil {
		log.Printf("startup: failed to load env files: %v", err)
		return
	}
}

func logAuthConfig(cfg config.Config) {
	switch cfg.AuthBackend {
	case "eve":
		log.Printf(
			"startup: auth_backend=eve client_id_set=%t client_secret_set=%t",
			cfg.EVEClientID != "",
			cfg.EVEClientSecret != "",
		)
	default:
		log.Printf(
			"startup: auth_backend=testauth client_id_set=%t client_secret_set=%t",
			cfg.OIDCClientID != "",
			cfg.OIDCClientSecret != "",
		)
	}
}
