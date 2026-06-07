package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	archivePkg "github.com/sahil87/fab-kit/src/go/fab/internal/archive"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
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

// hydrateStatusRe matches hydrate: done or hydrate: skipped in .status.yaml
var hydrateStatusRe = regexp.MustCompile(`^\s*hydrate:\s*(done|skipped)`)

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
			fmt.Fprintln(errW, "No archivable changes found.")
			os.Exit(1)
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
			// post-archive step (backlog mark) failed. The folder is already
			// archived and the move is irreversible within this loop, so count
			// it as archived and surface the backlog failure as a warning
			// rather than failing the change.
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

// isArchivable checks if a .status.yaml file has hydrate: done or hydrate: skipped.
func isArchivable(statusPath string) bool {
	f, err := os.Open(statusPath)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if hydrateStatusRe.MatchString(scanner.Text()) {
			return true
		}
	}
	return false
}
