package internal

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// nonRepoCommands are commands the shim handles directly when outside a fab-managed repo.
var nonRepoCommands = map[string]bool{
	"init":      true,
	"upgrade":   true,
	"--version": true,
	"-v":        true,
	"--help":    true,
	"-h":        true,
	"help":      true,
}

// Dispatch resolves the fab_version from config.yaml, ensures the binary is cached,
// and execs the cached fab-go with all arguments passed through.
func Dispatch(args []string) error {
	// Check if this is a non-repo command
	if len(args) > 0 && nonRepoCommands[args[0]] {
		// These are handled by cobra subcommands or root flags
		// If we get here, cobra didn't catch it — pass through
		return nil
	}

	cfg, err := ResolveConfig()
	if err != nil {
		return err
	}

	if cfg == nil {
		return fmt.Errorf("not in a fab-managed repo. Run 'fab init' to set one up")
	}

	binaryPath, err := EnsureCached(cfg.FabVersion)
	if err != nil {
		return err
	}

	return execBinary(binaryPath, args)
}

// execBinary replaces the current process with the cached fab-go binary.
func execBinary(binaryPath string, args []string) error {
	argv := append([]string{binaryPath}, args...)
	return syscall.Exec(binaryPath, argv, os.Environ())
}

// RunBinary runs the cached fab-go binary as a subprocess (for use in init/upgrade).
func RunBinary(binaryPath string, args []string) error {
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
