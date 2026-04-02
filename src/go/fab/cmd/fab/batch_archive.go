package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
)

func batchArchiveCmd() *cobra.Command {
	var listFlag, allFlag bool

	cmd := &cobra.Command{
		Use:   "archive [change...]",
		Short: "Archive multiple completed changes in one session",
		Long:  "Archives completed changes (hydrate done|skipped) by running /fab-archive for each.",
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
		out, err := exec.Command("fab", "change", "resolve", change).Output()
		if err != nil {
			fmt.Fprintf(errW, "Warning: could not resolve '%s', skipping\n", change)
			continue
		}
		match := strings.TrimSpace(string(out))

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

	// Build prompt for a single Claude session
	prompt := "Run /fab-archive for each of these changes, one at a time: " + strings.Join(resolved, " ")

	fmt.Fprintf(w, "Archiving: %s\n", strings.Join(resolved, " "))

	// Read spawn command and exec
	configPath := filepath.Join(fabRoot, "project", "config.yaml")
	spawnCmd := spawn.Command(configPath)

	// Exec the spawn command with the prompt
	bashBin, err := exec.LookPath("bash")
	if err != nil {
		return fmt.Errorf("bash not found: %w", err)
	}
	argv := []string{bashBin, "-c", spawnCmd + ` "$1"`, "--", prompt}
	return syscall.Exec(bashBin, argv, os.Environ())
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
