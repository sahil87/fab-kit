package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	archivePkg "github.com/sahil87/fab-kit/src/go/fab/internal/archive"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
	"github.com/spf13/cobra"
)

// isStdinTTY reports whether the command's input is an interactive terminal.
// It is a package-level seam so tests can force the TTY / non-TTY branches
// deterministically (a cobra test sets cmd.SetIn(buf), which is never a tty).
// The default uses only the standard library — no golang.org/x/term or
// go-isatty — mirroring src/go/fab-kit/internal/upgrade.go's isTTY (Constitution
// I: minimal single-binary dependencies). A non-*os.File reader (the test
// buffer) is treated as non-interactive.
var isStdinTTY = func(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func batchArchiveCmd() *cobra.Command {
	var yesFlag, dryRunFlag, quietFlag bool

	cmd := &cobra.Command{
		Use:   "archive [change...]",
		Short: "Archive multiple completed changes in one pass",
		Long:  "Archives completed changes (hydrate done|skipped) mechanically (move, index, backlog, pointer) in a Go loop — no agent or Claude session is spawned.",
		Example: `  # Preview what would be archived, without archiving
  fab batch archive --dry-run

  # Archive all archivable changes without prompting
  fab batch archive --yes

  # Archive two specific changes (4-char ID or folder substring)
  fab batch archive b91h ptwh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatchArchive(cmd, args, yesFlag, dryRunFlag, quietFlag)
		},
	}

	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Archive all archivable changes without prompting")
	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be archived without archiving")
	cmd.Flags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress per-change progress output (keep the summary footer and all stderr)")

	return cmd
}

func runBatchArchive(cmd *cobra.Command, args []string, yesFlag, dryRunFlag, quietFlag bool) error {
	w := cmd.OutOrStdout()

	// --quiet routes per-change progress to a discard writer; data (the footer,
	// the empty-set no-op, the --dry-run listing) and the consent flow keep
	// writing to the real w, and stderr is never touched (principle №9: what
	// survives --quiet is the data and the errors, never progress).
	pw := w
	if quietFlag {
		pw = io.Discard
	}

	// --dry-run (preview-only) and --yes (assume-yes and do it) are
	// contradictory. Error rather than silently picking one.
	if dryRunFlag && yesFlag {
		return fmt.Errorf("--dry-run and --yes are mutually exclusive")
	}

	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return err
	}

	changesDir := filepath.Join(fabRoot, "changes")
	if _, err := os.Stat(changesDir); os.IsNotExist(err) {
		return fmt.Errorf("changes directory not found at %s", changesDir)
	}

	// --dry-run lists what would be archived; no prompt, no action.
	if dryRunFlag {
		return listArchivable(w, changesDir)
	}

	// Explicit args are their own opt-in: naming the changes IS the
	// confirmation, so we archive them directly — no prompt, no TTY guard
	// (preserves the pre-redesign explicit-args behavior, incl. warn-and-skip
	// and the No-valid-changes exit-1).
	if len(args) > 0 {
		return archiveResolvedNames(cmd, pw, fabRoot, changesDir, args)
	}

	// Bare / --yes path: archive ALL archivable changes. archive is the one
	// bulk-mutating member of the batch family whose moves are effectively
	// irreversible within archiveLoop, so — unlike new/switch which stay
	// list-by-default behind --all — it earns an interactive confirm: a bare
	// invocation lists the set and prompts (default No), while --yes/-y is the
	// non-interactive escape hatch (replacing the old --all). This is the
	// well-understood list-then-confirm-with-a-yes-escape-hatch pattern
	// (apt/npm/gh); it replaces the 260612-ye8r explicit-or---all model, giving
	// zero-flag ergonomics for the common case without firing a destructive-ish
	// bulk op on a bare command (--dry-run replaces the old --list preview).
	changes := allArchivableNames(changesDir)
	if len(changes) == 0 {
		// A clean repo with nothing to archive is a benign no-op, not a
		// failure — exit 0 (before any prompt or non-TTY guard) so the generic
		// non-zero-means-STOP failure rule does not escalate it (finding F49).
		fmt.Fprintln(w, "No archivable changes found.")
		fmt.Fprintf(w, "\nArchived 0, skipped 0, failed 0.\n")
		return nil
	}

	if !yesFlag {
		// Without --yes we need a human to confirm the bulk move. Prompting
		// against a non-interactive stdin would hang on EOF (or read an empty
		// line and silently abort) — both wrong for the tmux/operator runtime,
		// where stdin is frequently not a tty — so refuse with guidance and a
		// non-zero exit instead. --yes is the automation escape hatch.
		if !isStdinTTY(cmd.InOrStdin()) {
			// Return a single (multi-line) error and let main()'s centralized
			// "ERROR: %s" printing own the prefix — emitting our own ERROR:
			// lines here would double the prefix on this one failure path.
			return fmt.Errorf("refusing to prompt for confirmation on a non-interactive stdin.\n" +
				"Re-run with --yes to archive non-interactively")
		}

		// List the set, then prompt with default No — a bare Enter (or any
		// non-y/yes answer) is the safe abort. Print the already-computed
		// `changes` slice (not a fresh scan) so the listed set and the prompt
		// count below cannot disagree.
		printArchivable(w, changes)
		fmt.Fprintf(w, "Archive these %d? [y/N] ", len(changes))
		reader := bufio.NewReader(cmd.InOrStdin())
		line, _ := reader.ReadString('\n')
		answer := strings.ToLower(strings.TrimSpace(line))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(w, "Aborted; nothing archived.")
			return nil
		}
	}

	if !quietFlag {
		fmt.Fprintf(w, "Archiving %d changes...\n", len(changes))
	}
	return archiveResolvedNames(cmd, pw, fabRoot, changesDir, changes)
}

// archiveResolvedNames resolves and validates each named change, then archives
// the valid ones via archiveLoop. It is shared by the explicit-args path and
// the bare/--yes archive-all path (the latter passes pre-filtered archivable
// folder names). Unresolvable/not-ready names warn-and-skip; if nothing
// resolves it returns the No-valid-changes error (exit 1). pw is the progress
// writer for per-change lines (io.Discard under --quiet); the footer stays on
// the real stdout writer and errW is never gated.
func archiveResolvedNames(cmd *cobra.Command, pw io.Writer, fabRoot, changesDir string, changes []string) error {
	w := cmd.OutOrStdout()
	errW := cmd.ErrOrStderr()

	// Resolve and validate each change
	var resolved []string
	for _, change := range changes {
		match, err := resolve.ToFolder(fabRoot, change)
		if err != nil {
			// Distinguish not-found from ambiguous (jznd (d)). A genuine
			// not-found name may already be archived — pass it through so
			// archiveLoop reports the idempotent soft skip (counted as skipped,
			// exit 0). An AMBIGUOUS name is a real user error: surface it as its
			// own warning instead of misreporting it as "could not resolve" or
			// silently soft-skipping it as already-archived.
			if errors.Is(err, resolve.ErrAmbiguous) {
				fmt.Fprintf(errW, "Warning: %v — skipping (use a 4-char ID or full folder name)\n", err)
				continue
			}
			if errors.Is(err, resolve.ErrNotFound) && archivePkg.IsArchived(fabRoot, change) {
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
		return fmt.Errorf("No valid changes to archive.")
	}

	_, _, failed := archiveLoop(w, pw, errW, fabRoot, resolved)
	if failed > 0 {
		return fmt.Errorf("%d change(s) failed to archive", failed)
	}
	return nil
}

// archiveLoop archives each resolved change in-process via
// archive.ArchiveWithBacklog. A per-change failure is reported and does not
// abort the remaining changes. Already-archived changes are counted as skipped,
// not failed. It returns the (archived, skipped, failed) counts and never calls
// os.Exit so it can be unit-tested. Per-change progress lines go to pw (which is
// io.Discard under --quiet); the summary footer always writes to w and warnings
// always write to errW — neither is gated by --quiet (principle №9).
func archiveLoop(w, pw, errW io.Writer, fabRoot string, resolved []string) (archived, skipped, failed int) {
	for _, name := range resolved {
		result, err := archivePkg.ArchiveWithBacklog(fabRoot, name, "")
		if err != nil {
			if errors.Is(err, archivePkg.ErrAlreadyArchived) {
				fmt.Fprintf(pw, "  %s — already archived, skipping\n", name)
				skipped++
				continue
			}
			// A non-nil result means the archive move succeeded but a
			// post-archive step (index update or backlog mark) failed. The
			// folder is already archived and the move is irreversible within
			// this loop, so count it as archived and surface the failure as
			// a warning rather than failing the change.
			if result != nil {
				fmt.Fprintf(pw, "  %s — archived\n", name)
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
		fmt.Fprintln(pw, line)
		archived++
	}
	fmt.Fprintf(w, "\nArchived %d, skipped %d, failed %d.\n", archived, skipped, failed)
	return archived, skipped, failed
}

// listArchivable scans for archivable changes and prints them. Used by the
// --dry-run path, which has no precomputed set.
func listArchivable(w interface{ Write([]byte) (int, error) }, changesDir string) error {
	printArchivable(w, allArchivableNames(changesDir))
	return nil
}

// printArchivable renders an already-computed set of archivable change names.
// The bare-prompt path passes the same slice it counts for "Archive these N?"
// so the listed set and the prompt count come from a single scan (no second
// filesystem read that could disagree if the set changes mid-command).
func printArchivable(w interface{ Write([]byte) (int, error) }, names []string) {
	fmt.Fprintln(w, "Archivable changes (hydrate done|skipped):")
	fmt.Fprintln(w)

	if len(names) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		for _, name := range names {
			fmt.Fprintf(w, "  %s\n", name)
		}
	}
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
