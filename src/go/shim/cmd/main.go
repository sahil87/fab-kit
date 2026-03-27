package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/wvrdz/fab-kit/src/go/shim/internal"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	// Handle --version (no config needed).
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v") {
		fmt.Printf("fab-shim %s\n", version)
		return nil
	}

	// Handle --help (no config needed).
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h")) {
		printHelp()
		return nil
	}

	// Handle `fab init` (no config needed — creates it).
	if args[0] == "init" {
		return internal.RunInit()
	}

	// All other commands require config discovery.
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	cfg, err := internal.DiscoverConfig(cwd)
	if err != nil {
		if errors.Is(err, internal.ErrNoConfig) {
			return fmt.Errorf("Not in a fab-managed repo. Run 'fab init' to get started.")
		}
		return err
	}

	if cfg.FabVersion == "" {
		return fmt.Errorf("No fab_version in config.yaml. Run 'fab init' to set one.")
	}

	// Ensure the version is cached (downloads if needed).
	runtimePath, err := internal.EnsureCached(cfg.FabVersion)
	if err != nil {
		return err
	}

	// Exec into the per-repo runtime with all original args.
	return execRuntime(runtimePath, args)
}

func execRuntime(runtimePath string, args []string) error {
	argv := append([]string{runtimePath}, args...)
	return syscall.Exec(runtimePath, argv, os.Environ())
}

func printHelp() {
	fmt.Printf(`fab-shim %s — version-aware fab-kit dispatcher

Usage:
  fab <command> [args...]    Dispatch to the repo's pinned fab-kit version
  fab init                   Initialize a new fab-kit project
  fab --version              Show shim version
  fab --help                 Show this help

The shim reads fab_version from fab/project/config.yaml, ensures that
version is cached locally, and dispatches all commands to the cached
per-repo runtime at ~/.fab-kit/versions/<version>/fab/.kit/bin/fab.
`, version)
}
