package main

import (
	"errors"
	"fmt"
	"os"

	"sentinel2-taskutil/internal/assets"
	"sentinel2-taskutil/internal/buildmeta"
	"sentinel2-taskutil/internal/cleanup"
	"sentinel2-taskutil/internal/devconsole"
	"sentinel2-taskutil/internal/devlogs"
	"sentinel2-taskutil/internal/project"

	flags "github.com/jessevdk/go-flags"
)

func main() {
	bootstrapCfg, err := project.ParseBootstrap(os.Args[1:])
	if err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rootDir, err := project.ResolveRootDir(bootstrapCfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	_ = project.LoadDotEnv(rootDir)

	cfg, command, err := project.ParseConfig(os.Args[1:], rootDir)
	if err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	switch command {
	case "dev":
		if err := devconsole.Run(cfg, cfg.DevMigrations); err != nil {
			fmt.Fprintf(os.Stderr, "taskutil dev failed: %v\n", err)
			os.Exit(1)
		}
	case "version":
		version, err := buildmeta.DeriveBuildVersion()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(version)
	case "clean-root":
		if err := cleanup.CleanRoot(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "taskutil clean-root failed: %v\n", err)
			os.Exit(1)
		}
	case "prepare-embed":
		if err := assets.PrepareEmbed(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "taskutil prepare-embed failed: %v\n", err)
			os.Exit(1)
		}
	case "dev-logs-tail":
		if err := devlogs.Tail(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "taskutil dev-logs-tail failed: %v\n", err)
			os.Exit(1)
		}
	case "dev-logs-clean":
		if err := devlogs.Clean(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "taskutil dev-logs-clean failed: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: taskutil <command>")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  dev             run frontend+backend development processes")
	fmt.Fprintln(os.Stderr, "  version         derive build version from git state")
	fmt.Fprintln(os.Stderr, "  clean-root      clean root build artifacts/caches")
	fmt.Fprintln(os.Stderr, "  prepare-embed   copy frontend/dist into backend embed dir")
	fmt.Fprintln(os.Stderr, "  dev-logs-tail   tail dev logs for latest session")
	fmt.Fprintln(os.Stderr, "  dev-logs-clean  clean old dev log sessions")
}
