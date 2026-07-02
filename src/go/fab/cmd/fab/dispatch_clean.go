package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func dispatchCleanCmd() *cobra.Command {
	var orphans bool
	cmd := &cobra.Command{
		Use:   "clean [change]",
		Short: "Remove dispatch state dirs — named change, all, or orphaned (--orphans)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			changeArg := ""
			if len(args) > 0 {
				changeArg = args[0]
			}
			return runDispatchClean(cmd, changeArg, orphans)
		},
	}
	cmd.Flags().BoolVar(&orphans, "orphans", false, "Prune only dirs whose ID no longer resolves to a non-archived change")
	return cmd
}

// runDispatchClean has three modes (per intake §7b):
//   - clean <change>       → remove that change's .fab-dispatch/{id}/
//   - clean                → remove all .fab-dispatch/*/
//   - clean --orphans      → prune any .fab-dispatch/{id}/ whose ID no longer
//     resolves to a non-archived change (--orphans ignores any positional arg)
func runDispatchClean(cmd *cobra.Command, changeArg string, orphans bool) error {
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return err
	}
	repoRoot := filepath.Dir(fabRoot)
	root := filepath.Join(repoRoot, dispatch.DirName)

	// Named-change mode: remove exactly that change's dir.
	if changeArg != "" && !orphans {
		folder, err := resolve.ToFolder(fabRoot, changeArg)
		if err != nil {
			return err
		}
		id := resolve.ExtractID(folder)
		if id == "" {
			// Guard against an empty ID: DirFor(repoRoot, "") would resolve to
			// the root .fab-dispatch/ dir, and removeDispatchDir would then wipe
			// ALL dispatch state instead of one change's. Mirrors the same guard
			// in resolveDispatchDir (dispatch.go).
			return fmt.Errorf("could not extract change ID from %q", folder)
		}
		dir := dispatch.DirFor(repoRoot, id)
		return removeDispatchDir(cmd, dir, id)
	}

	// All / orphans modes iterate the .fab-dispatch/ dir.
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(cmd.OutOrStdout(), "no .fab-dispatch/ state to clean")
			return nil
		}
		return fmt.Errorf("read %s: %w", dispatch.DirName, err)
	}

	removed := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if orphans && !isOrphanedID(fabRoot, id) {
			continue
		}
		dir := dispatch.DirFor(repoRoot, id)
		if err := removeDispatchDir(cmd, dir, id); err != nil {
			return err
		}
		removed++
	}

	if removed == 0 {
		if orphans {
			fmt.Fprintln(cmd.OutOrStdout(), "no orphaned dispatch state to clean")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "no dispatch state to clean")
		}
	}
	return nil
}

// isOrphanedID reports whether the change ID no longer resolves to a
// non-archived change. resolve.ToFolder scans fab/changes/ (excluding archive/),
// so any resolution failure — not-found (archived/deleted) or ambiguous — makes
// the dir orphaned. A dir whose ID still resolves to an active change is kept.
func isOrphanedID(fabRoot, id string) bool {
	_, err := resolve.ToFolder(fabRoot, id)
	return err != nil
}

func removeDispatchDir(cmd *cobra.Command, dir, id string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %s: %w", dir, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "removed dispatch state for %s\n", id)
	return nil
}
