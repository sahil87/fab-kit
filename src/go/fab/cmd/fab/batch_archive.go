package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	archivePkg "github.com/sahil87/fab-kit/src/go/fab/internal/archive"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
	"github.com/spf13/cobra"
)

func batchArchiveCmd() *cobra.Command {
	var listFlag, allFlag bool

	cmd := &cobra.Command{
		Use:   "archive [change...]",
		Short: "Archive multiple completed changes in one pass",
		Long:  "Archives completed changes (hydrate done|skipped) mechanically (move, index, backlog, pointer) in a Go loop — no agent or Claude session is spawned.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatchArchive(cmd, args, listFlag, allFlag)
		},
	}

	cmd.Flags().BoolVar(&listFlag, "list", false, "Show archivable changes without archiving")
	cmd.Flags().BoolVar(&allFlag, "all", false, "Archive all archivable changes")

	return cmd
}

func runBatchArchive(cmd *cobra.Command, args []string, listFlag, allFlag bool) error {
	w := cmd.OutOrStdout()
	errW := cmd.ErrOrStderr()

	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return err
	}

	changesDir := filepath.Join(fabRoot, "changes")
	if _, err := os.Stat(changesDir); os.IsNotExist(err) {
		return fmt.Errorf("changes directory not found at %s", changesDir)
	}

	// No args defaults to --all (different from new/switch which default to --list)
	if len(args) == 0 && !listFlag {
		allFlag = true
	}

	if listFlag {
		return listArchivable(w, changesDir)
	}

	// Collect change names
	var changes []string
	if allFlag {
		changes = allArchivableNames(changesDir)
		if len(changes) == 0 {
			// A clean repo with nothing to archive is a benign no-op, not a
			// failure — exit 0 so the generic non-zero-means-STOP failure
			// rule does not escalate it.
			fmt.Fprintln(w, "No archivable changes found.")
			fmt.Fprintf(w, "\nArchived 0, skipped 0, failed 0.\n")
			return nil
		}
		fmt.Fprintf(w, "Archiving %d changes...\n", len(changes))
	} else {
		changes = args
	}

	// Resolve and validate each change
	var resolved []string
	for _, change := range changes {
		match, err := resolve.ToFolder(fabRoot, change)
		if err != nil {
			// A name that no longer resolves may already be archived — pass
			// it through so archiveLoop reports the soft skip (counted as
			// skipped, exit 0) instead of warning into the exit-1 path.
			if archivePkg.IsArchived(fabRoot, change) {
				resolved = append(resolved, change)
				continue
			}
			fmt.Fprintf(errW, "Warning: could not resolve '%s', skipping\n", change)
			continue
		}

		statusPath := filepath.Join(changesDir, match, ".status.yaml")
		if !isArchivable(statusPath) {
			fmt.Fprintf(errW, "Warning: '%s' not ready for archive (hydrate not done or skipped), skipping\n", match)
			continue
		}

		resolved = append(resolved, match)
	}

	if len(resolved) == 0 {
		fmt.Fprintln(errW, "No valid changes to archive.")
		os.Exit(1)
	}

	_, _, failed := archiveLoop(w, errW, fabRoot, resolved)
	if failed > 0 {
		os.Exit(1)
	}
	return nil
}

// archiveLoop archives each resolved change in-process via
// archive.ArchiveWithBacklog. A per-change failure is reported and does not
// abort the remaining changes. Already-archived changes are counted as skipped,
// not failed. It returns the (archived, skipped, failed) counts and never calls
// os.Exit so it can be unit-tested.
func archiveLoop(w, errW io.Writer, fabRoot string, resolved []string) (archived, skipped, failed int) {
	for _, name := range resolved {
		result, err := archivePkg.ArchiveWithBacklog(fabRoot, name, "")
		if err != nil {
			if errors.Is(err, archivePkg.ErrAlreadyArchived) {
				fmt.Fprintf(w, "  %s — already archived, skipping\n", name)
				skipped++
				continue
			}
			// A non-nil result means the archive move succeeded but a
			// post-archive step (index update or backlog mark) failed. The
			// folder is already archived and the move is irreversible within
			// this loop, so count it as archived and surface the failure as
			// a warning rather than failing the change.
			if result != nil {
				fmt.Fprintf(w, "  %s — archived\n", name)
				fmt.Fprintf(errW, "    warning: %v\n", err)
				archived++
				continue
			}
			fmt.Fprintf(errW, "  %s — FAILED: %v\n", name, err)
			failed++
			continue
		}
		line := fmt.Sprintf("  %s — archived", name)
		if result.Backlog == "marked" {
			line += " (backlog marked done)"
		}
		fmt.Fprintln(w, line)
		archived++
	}
	fmt.Fprintf(w, "\nArchived %d, skipped %d, failed %d.\n", archived, skipped, failed)
	return archived, skipped, failed
}

// listArchivable prints archivable changes.
func listArchivable(w interface{ Write([]byte) (int, error) }, changesDir string) error {
	fmt.Fprintln(w, "Archivable changes (hydrate done|skipped):")
	fmt.Fprintln(w)

	names := allArchivableNames(changesDir)
	if len(names) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		for _, name := range names {
			fmt.Fprintf(w, "  %s\n", name)
		}
	}
	return nil
}

// allArchivableNames returns change names where hydrate is done or skipped.
func allArchivableNames(changesDir string) []string {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		statusPath := filepath.Join(changesDir, e.Name(), ".status.yaml")
		if isArchivable(statusPath) {
			names = append(names, e.Name())
		}
	}
	return names
}

// isArchivable checks if a .status.yaml file has progress.hydrate done or
// skipped. It goes through internal/statusfile — the package that owns the
// .status.yaml schema — rather than a private line-scan, so batch archive
// applies the same parsing semantics as every other consumer.
func isArchivable(statusPath string) bool {
	sf, err := statusfile.Load(statusPath)
	if err != nil {
		return false
	}
	progress := sf.GetProgress("hydrate")
	return progress == "done" || progress == "skipped"
}
