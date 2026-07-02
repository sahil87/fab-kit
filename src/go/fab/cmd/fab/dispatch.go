package main

import (
	"fmt"
	"path/filepath"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

// dispatchCmd is the parent of the headless-process-manager command family:
// `fab dispatch <start|status|logs|kill|clean> [args...]`. It is the
// tmux-independent CLI adapter for cross-harness stage dispatch — parallel to,
// and independent of, `fab pane` / `fab operator` (which stay the interactive
// path). See docs/specs/harness-adapters.md for the cross-adapter contract.
func dispatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispatch",
		Short: "Headless process manager for CLI-dispatched pipeline stages",
		Long: "Headless, tmux-independent process manager: start/status/logs/kill/clean.\n" +
			"Launches a stage's resolved spawn command detached (setsid), tracks it under\n" +
			".fab-dispatch/{id}/, and exposes a byte-stable poll surface. POSIX-only (v1).",
	}

	cmd.AddCommand(
		dispatchStartCmd(),
		dispatchStatusCmd(),
		dispatchLogsCmd(),
		dispatchKillCmd(),
		dispatchCleanCmd(),
	)

	return cmd
}

// resolveDispatchDir resolves <change> to its 4-char ID and returns the
// absolute .fab-dispatch/{id}/ directory (DirFor joins onto the absolute
// repoRoot) plus the resolved ID. Shared
// by start/status/logs/kill (clean has its own multi-dir resolution). fabRoot
// is found via resolve.FabRoot; the repo root is its parent (the same
// derivation internal/archive uses for the .fab-status.yaml pointer).
func resolveDispatchDir(changeArg string) (dir, id string, err error) {
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return "", "", err
	}
	folder, err := resolve.ToFolder(fabRoot, changeArg)
	if err != nil {
		return "", "", err
	}
	id = resolve.ExtractID(folder)
	if id == "" {
		return "", "", fmt.Errorf("could not extract change ID from %q", folder)
	}
	repoRoot := filepath.Dir(fabRoot)
	return dispatch.DirFor(repoRoot, id), id, nil
}
