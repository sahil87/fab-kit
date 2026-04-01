package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	wt "github.com/sahil87/fab-kit/src/go/wt/internal/worktree"
)

func deleteCmd() *cobra.Command {
	var (
		worktreeName   string
		deleteBranch   string
		deleteRemote   string
		deleteAll      bool
		stashFlag      bool
		nonInteractive bool
	)

	cmd := &cobra.Command{
		Use:   "delete [worktree-names...]",
		Short: "Delete a git worktree",
		Long: `Delete one or more git worktrees with optional branch cleanup.

Positional arguments are interpreted as worktree names to delete.
Resolution order: --delete-all, positional args, --worktree-name (deprecated), current worktree, interactive selection.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Apply defaults (deleteBranch "" = auto mode, handled by handleBranchCleanup)
			if deleteRemote == "" {
				deleteRemote = "true"
			}

			if err := wt.ValidateGitRepo(); err != nil {
				wt.ExitWithError(wt.ExitGitError,
					"Not a git repository",
					"This command requires a git repository",
					"Navigate to a git repository and try again")
			}

			// Set up signal handling
			rb := wt.NewRollback()
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				fmt.Println()
				rb.Execute()
				os.Exit(130)
			}()

			stashMode := ""
			if stashFlag {
				stashMode = "stash"
			}

			if deleteAll {
				return handleDeleteAll(nonInteractive, deleteBranch, deleteRemote, stashMode)
			}

			if len(args) > 0 && worktreeName != "" {
				wt.ExitWithError(wt.ExitInvalidArgs,
					"Cannot mix positional arguments and --worktree-name",
					"Use either positional arguments or --worktree-name, not both",
					"Example: wt delete alpha bravo --non-interactive")
			}

			if len(args) > 0 {
				return handleDeleteMultiple(args, nonInteractive, deleteBranch, deleteRemote, stashMode)
			}

			if worktreeName != "" {
				return handleDeleteByName(worktreeName, nonInteractive, deleteBranch, deleteRemote, stashMode, rb)
			}

			if wt.IsWorktree() {
				return handleDeleteCurrent(nonInteractive, deleteBranch, deleteRemote, stashMode, rb)
			}

			if nonInteractive {
				wt.ExitWithError(wt.ExitInvalidArgs,
					"No worktree specified",
					"In non-interactive mode, specify worktree names as arguments (or run from within a worktree)",
					"Example: wt delete my-feature --non-interactive")
			}

			return handleDeleteMenu(nonInteractive, deleteBranch, deleteRemote, stashMode)
		},
	}

	cmd.Flags().StringVar(&worktreeName, "worktree-name", "", "Worktree to delete")
	cmd.Flags().StringVar(&deleteBranch, "delete-branch", "", "Delete associated branch: true, false, or auto (default: auto — deletes only when branch matches worktree name)")
	cmd.Flags().StringVar(&deleteRemote, "delete-remote", "", "Delete remote branch: true (default) or false")
	cmd.Flags().BoolVar(&deleteAll, "delete-all", false, "Delete all worktrees")
	cmd.Flags().BoolVarP(&stashFlag, "stash", "s", false, "Stash uncommitted changes before deleting")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "No prompts, use defaults")

	cmd.Flags().MarkDeprecated("worktree-name", "use positional arguments instead")

	return cmd
}

func handleDeleteCurrent(nonInteractive bool, deleteBranch, deleteRemote, stashMode string, rb *wt.Rollback) error {
	if !wt.IsWorktree() {
		wt.ExitWithError(wt.ExitGeneralError,
			"Not in a worktree",
			"wt delete without --worktree-name only works from within a worktree",
			"Specify a worktree: wt delete --worktree-name <name>")
	}

	ctx, err := wt.GetRepoContext()
	if err != nil {
		wt.ExitWithError(wt.ExitGeneralError, "Cannot get repo context", err.Error(), "")
	}

	wtPath, err := wt.CurrentWorktreeTopLevel()
	if err != nil {
		wt.ExitWithError(wt.ExitGeneralError, "Cannot determine worktree path", err.Error(), "")
	}
	wtName := filepath.Base(wtPath)

	branch, err := wt.CurrentBranch()
	if err != nil {
		wt.ExitWithError(wt.ExitGeneralError, "Cannot determine current branch", err.Error(), "")
	}

	fmt.Printf("Worktree: %s%s%s\n", wt.ColorBold, wtName, wt.ColorReset)
	fmt.Printf("Branch: %s\n", branch)
	fmt.Printf("Path: %s\n\n", wtPath)

	// Handle uncommitted changes
	if wt.HasUncommittedChanges() || wt.HasUntrackedFiles() {
		if err := handleUncommittedChanges(wtName, stashMode, nonInteractive, rb); err != nil {
			return err
		}
	}

	// Check for unpushed commits
	if wt.HasUnpushedCommits(branch) {
		if err := handleUnpushedCommits(branch, nonInteractive); err != nil {
			return err
		}
	}

	// Confirmation
	if !nonInteractive {
		choice, err := wt.ShowMenu("Delete this worktree?", []string{"Yes, delete"}, 0)
		if err != nil {
			return err
		}
		if choice == 0 {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Change to main repo before deletion
	if err := os.Chdir(ctx.RepoRoot); err != nil {
		wt.ExitWithError(wt.ExitGeneralError, "Cannot change to main repo",
			fmt.Sprintf("Failed to cd to %s", ctx.RepoRoot),
			"Check if the main repository still exists")
	}

	fmt.Println("Removing worktree...")
	if err := wt.RemoveWorktree(wtPath, true); err != nil {
		wt.ExitWithError(wt.ExitGitError, "Failed to remove worktree", err.Error(), "")
	}
	fmt.Printf("Deleted worktree: %s%s%s\n", wt.ColorGreen, wtName, wt.ColorReset)

	handleBranchCleanup(branch, wtName, deleteBranch, deleteRemote)

	fmt.Println()
	fmt.Println("You are no longer in a valid directory.")
	fmt.Printf("Run: %scd %s%s\n", wt.ColorBold, ctx.RepoRoot, wt.ColorReset)

	return nil
}

func handleDeleteByName(name string, nonInteractive bool, deleteBranch, deleteRemote, stashMode string, rb *wt.Rollback) error {
	if err := wt.ValidateGitRepo(); err != nil {
		wt.ExitWithError(wt.ExitGitError, "Not a git repository", err.Error(), "")
	}

	entries, err := listWorktreeEntries()
	if err != nil {
		wt.ExitWithError(wt.ExitGitError, "Cannot list worktrees", err.Error(), "")
	}

	var wtPath, branch string
	for _, e := range entries {
		if filepath.Base(e.path) == name {
			wtPath = e.path
			branch = e.branch
			break
		}
	}

	if wtPath == "" {
		wt.PrintError(
			fmt.Sprintf("Worktree '%s' not found", name),
			"No worktree with that name exists",
			"Use 'wt list' to see available worktrees")
		os.Exit(wt.ExitGeneralError)
	}

	fmt.Printf("Worktree: %s%s%s\n", wt.ColorBold, name, wt.ColorReset)
	fmt.Printf("Branch: %s\n", branch)
	fmt.Printf("Path: %s\n\n", wtPath)

	// Handle stash
	if stashMode == "stash" {
		handleStashInDir(wtPath, name)
	}

	// Confirmation
	if !nonInteractive {
		choice, err := wt.ShowMenu("Delete this worktree?", []string{"Yes, delete"}, 0)
		if err != nil {
			return err
		}
		if choice == 0 {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	fmt.Println("Removing worktree...")
	if err := wt.RemoveWorktree(wtPath, true); err != nil {
		wt.ExitWithError(wt.ExitGitError, "Failed to remove worktree", err.Error(), "")
	}
	fmt.Printf("Deleted worktree: %s%s%s\n", wt.ColorGreen, name, wt.ColorReset)

	handleBranchCleanup(branch, name, deleteBranch, deleteRemote)

	return nil
}

func handleDeleteMultiple(names []string, nonInteractive bool, deleteBranch, deleteRemote, stashMode string) error {
	ctx, err := wt.GetRepoContext()
	if err != nil {
		wt.ExitWithError(wt.ExitGeneralError, "Cannot get repo context", err.Error(), "")
	}

	// If running from inside a worktree that may be deleted, chdir to main repo first
	if wt.IsWorktree() {
		if err := os.Chdir(ctx.RepoRoot); err != nil {
			wt.ExitWithError(wt.ExitGeneralError, "Cannot change to main repo",
				fmt.Sprintf("Failed to cd to %s", ctx.RepoRoot),
				"Check if the main repository still exists")
		}
	}

	entries, err := listWorktreeEntries()
	if err != nil {
		wt.ExitWithError(wt.ExitGitError, "Cannot list worktrees", err.Error(), "")
	}

	// Build lookup map: name -> rawEntry (excluding main worktree)
	entryMap := make(map[string]rawEntry)
	for _, e := range entries {
		if e.path == ctx.RepoRoot {
			continue
		}
		entryMap[filepath.Base(e.path)] = e
	}

	// Deduplicate names preserving order
	seen := make(map[string]bool)
	var unique []string
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			unique = append(unique, n)
		}
	}

	// Resolve all names upfront — fail-fast if any are invalid
	type wtInfo struct {
		name   string
		path   string
		branch string
	}
	var resolved []wtInfo
	var unresolved []string
	for _, n := range unique {
		if e, ok := entryMap[n]; ok {
			resolved = append(resolved, wtInfo{
				name:   n,
				path:   e.path,
				branch: e.branch,
			})
		} else {
			unresolved = append(unresolved, n)
		}
	}

	if len(unresolved) > 0 {
		for _, n := range unresolved {
			wt.PrintError(
				fmt.Sprintf("Worktree '%s' not found", n),
				"No worktree with that name exists",
				"Use 'wt list' to see available worktrees")
		}
		os.Exit(wt.ExitGeneralError)
	}

	// Display summary
	fmt.Printf("Worktrees to delete (%d):\n", len(resolved))
	for _, w := range resolved {
		fmt.Printf("  %s%s%s  (branch: %s, path: %s)\n", wt.ColorBold, w.name, wt.ColorReset, w.branch, w.path)
	}
	fmt.Println()

	// Single confirmation prompt
	if !nonInteractive {
		choice, err := wt.ShowMenu(
			fmt.Sprintf("Delete these %d worktrees?", len(resolved)),
			[]string{"Yes, delete all"},
			0)
		if err != nil {
			return err
		}
		if choice == 0 {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Sequential deletion with continue-on-error
	for _, w := range resolved {
		fmt.Printf("\n--- Deleting: %s ---\n", w.name)
		fmt.Printf("Worktree: %s%s%s\n", wt.ColorBold, w.name, wt.ColorReset)
		fmt.Printf("Branch: %s\n", w.branch)
		fmt.Printf("Path: %s\n\n", w.path)

		// Handle stash per worktree
		if stashMode == "stash" {
			handleStashInDir(w.path, w.name)
		}

		fmt.Println("Removing worktree...")
		if err := wt.RemoveWorktree(w.path, true); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %s\n", w.name, err)
			continue
		}
		fmt.Printf("Deleted worktree: %s%s%s\n", wt.ColorGreen, w.name, wt.ColorReset)
		handleBranchCleanup(w.branch, w.name, deleteBranch, deleteRemote)
	}

	return nil
}

func handleDeleteAll(nonInteractive bool, deleteBranch, deleteRemote, stashMode string) error {
	ctx, err := wt.GetRepoContext()
	if err != nil {
		wt.ExitWithError(wt.ExitGeneralError, "Cannot get repo context", err.Error(), "")
	}

	// If in a worktree, cd to main repo
	if wt.IsWorktree() {
		if err := os.Chdir(ctx.RepoRoot); err != nil {
			wt.ExitWithError(wt.ExitGeneralError, "Cannot change to main repo",
				fmt.Sprintf("Failed to cd to %s", ctx.RepoRoot), "")
		}
	}

	entries, err := listWorktreeEntries()
	if err != nil {
		wt.ExitWithError(wt.ExitGitError, "Cannot list worktrees", err.Error(), "")
	}

	// Collect non-main worktrees
	type wtInfo struct {
		name   string
		path   string
		branch string
	}
	var worktrees []wtInfo
	for _, e := range entries {
		if e.path != ctx.RepoRoot {
			worktrees = append(worktrees, wtInfo{
				name:   filepath.Base(e.path),
				path:   e.path,
				branch: e.branch,
			})
		}
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found.")
		return nil
	}

	fmt.Printf("Found %d worktree(s):\n", len(worktrees))
	for _, w := range worktrees {
		fmt.Printf("  %s\n", w.name)
	}
	fmt.Println()

	// Confirmation
	if !nonInteractive {
		choice, err := wt.ShowMenu(
			fmt.Sprintf("Delete ALL %d worktree(s)?", len(worktrees)),
			[]string{"Yes, delete all"},
			0)
		if err != nil {
			return err
		}
		if choice == 0 {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	for _, w := range worktrees {
		fmt.Printf("\n--- Deleting: %s ---\n", w.name)
		fmt.Printf("Worktree: %s%s%s\n", wt.ColorBold, w.name, wt.ColorReset)
		fmt.Printf("Branch: %s\n", w.branch)
		fmt.Printf("Path: %s\n\n", w.path)

		fmt.Println("Removing worktree...")
		if err := wt.RemoveWorktree(w.path, true); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %s\n", w.name, err)
			continue
		}
		fmt.Printf("Deleted worktree: %s%s%s\n", wt.ColorGreen, w.name, wt.ColorReset)
		handleBranchCleanup(w.branch, w.name, deleteBranch, deleteRemote)
	}

	return nil
}

func handleDeleteMenu(nonInteractive bool, deleteBranch, deleteRemote, stashMode string) error {
	ctx, err := wt.GetRepoContext()
	if err != nil {
		wt.ExitWithError(wt.ExitGeneralError, "Cannot get repo context", err.Error(), "")
	}

	entries, err := listWorktreeEntries()
	if err != nil {
		wt.ExitWithError(wt.ExitGitError, "Cannot list worktrees", err.Error(), "")
	}

	type wtOption struct {
		name   string
		path   string
		branch string
	}

	var options []wtOption
	var newestPath string
	var newestTime int64

	for _, e := range entries {
		if e.path == ctx.RepoRoot {
			continue
		}
		name := filepath.Base(e.path)
		options = append(options, wtOption{name: name, path: e.path, branch: e.branch})

		if info, err := os.Stat(e.path); err == nil {
			mtime := info.ModTime().Unix()
			if mtime > newestTime {
				newestTime = mtime
				newestPath = e.path
			}
		}
	}

	if len(options) == 0 {
		fmt.Println("No worktrees found.")
		return nil
	}

	// Find default index (offset by 1 for "All" option)
	defaultIdx := 2
	for i, o := range options {
		if o.path == newestPath {
			defaultIdx = i + 2
			break
		}
	}

	// Prepend "All" option
	allLabel := fmt.Sprintf("All (%d worktrees)", len(options))
	menuNames := []string{allLabel}
	for _, o := range options {
		menuNames = append(menuNames, fmt.Sprintf("%s (%s)", o.name, o.branch))
	}

	choice, err := wt.ShowMenu("Select worktree to delete:", menuNames, defaultIdx)
	if err != nil {
		return err
	}
	if choice == 0 {
		fmt.Println("Cancelled.")
		return nil
	}

	if choice == 1 {
		return handleDeleteAll(nonInteractive, deleteBranch, deleteRemote, stashMode)
	}

	selected := options[choice-2]
	rb := wt.NewRollback()
	return handleDeleteByName(selected.name, nonInteractive, deleteBranch, deleteRemote, stashMode, rb)
}

func handleUncommittedChanges(wtName, stashMode string, nonInteractive bool, rb *wt.Rollback) error {
	dateStr := time.Now().Format("2006-01-02")

	if stashMode == "stash" {
		fmt.Println("Stashing changes...")
		hash, err := wt.StashCreate(fmt.Sprintf("wt-delete: saved from worktree '%s' on %s", wtName, dateStr))
		if err != nil {
			return err
		}
		if hash != "" {
			fmt.Printf("Changes stashed (hash: %s). Recover with 'git stash list' or 'git stash apply %s'\n", hash, hash)
		}
		return nil
	}

	if nonInteractive {
		fmt.Println("Discarding uncommitted changes...")
		return nil
	}

	fmt.Printf("\n%sWarning:%s Worktree has uncommitted changes\n\n", wt.ColorYellow, wt.ColorReset)

	choice, err := wt.ShowMenu("What would you like to do?", []string{
		"Stash changes and delete (Recommended)",
		"Discard changes and delete",
	}, 1)
	if err != nil {
		return err
	}

	switch choice {
	case 1:
		fmt.Println("Stashing changes...")
		hash, err := wt.StashCreate(fmt.Sprintf("wt-delete: saved from worktree '%s' on %s", wtName, dateStr))
		if err != nil {
			return err
		}
		if hash != "" {
			fmt.Printf("Changes stashed (hash: %s). Recover with 'git stash list' or 'git stash apply %s'\n", hash, hash)
		}
	case 2:
		fmt.Println("Discarding changes...")
	case 0:
		fmt.Println("Cancelled.")
		os.Exit(wt.ExitSuccess)
	}
	return nil
}

func handleUnpushedCommits(branch string, nonInteractive bool) error {
	if nonInteractive {
		return nil
	}

	count := wt.GetUnpushedCount(branch)
	fmt.Printf("\n%sWarning:%s Branch has %d unpushed commit(s)\n\n", wt.ColorYellow, wt.ColorReset, count)

	fmt.Println("Commits that will be lost:")
	lines := wt.GetUnpushedCommitLines(branch, 5)
	for _, line := range lines {
		fmt.Printf("  %s\n", line)
	}
	if count > 5 {
		fmt.Printf("  ... and %d more\n", count-5)
	}
	fmt.Println()

	choice, err := wt.ShowMenu("Continue anyway?", []string{
		"Yes, delete (commits will be lost)",
	}, 0)
	if err != nil {
		return err
	}
	if choice == 0 {
		fmt.Println("Cancelled.")
		os.Exit(wt.ExitSuccess)
	}
	return nil
}

func handleBranchCleanup(branch, wtName, deleteBranch, deleteRemote string) {
	if branch == "" {
		return
	}

	// Tri-state logic for deleteBranch:
	//   ""      = auto mode: delete only if branch == wtName
	//   "true"  = force delete regardless of name match
	//   "false" = skip deletion
	shouldDelete := false
	switch deleteBranch {
	case "true":
		shouldDelete = true
	case "false":
		shouldDelete = false
	default: // "" = auto mode
		if branch == wtName {
			shouldDelete = true
		} else {
			fmt.Printf("Skipped branch deletion: %s ≠ worktree name (%s). Use --delete-branch true to force.\n", branch, wtName)
		}
	}

	if shouldDelete {
		if err := wt.DeleteLocalBranch(branch, true); err == nil {
			fmt.Printf("Deleted branch: %s (local)\n", branch)
		}

		if deleteRemote == "true" && wt.BranchExistsRemotely(branch) {
			if err := wt.DeleteRemoteBranch(branch); err == nil {
				fmt.Printf("Deleted branch: %s (remote)\n", branch)
			} else {
				fmt.Printf("%sNote:%s Could not delete remote branch\n", wt.ColorYellow, wt.ColorReset)
			}
		}
	}

	// Clean up orphaned wt/ branch (always runs regardless of deleteBranch)
	wtOriginBranch := "wt/" + wtName
	if wtOriginBranch != branch {
		if err := wt.DeleteLocalBranch(wtOriginBranch, true); err == nil {
			fmt.Printf("Deleted branch: %s (local)\n", wtOriginBranch)
		}
		if deleteRemote == "true" && wt.BranchExistsRemotely(wtOriginBranch) {
			if err := wt.DeleteRemoteBranch(wtOriginBranch); err == nil {
				fmt.Printf("Deleted branch: %s (remote)\n", wtOriginBranch)
			}
		}
	}
}

func handleStashInDir(wtPath, name string) {
	// Check if there are changes to stash
	cmd := exec.Command("git", "diff", "--quiet", "HEAD")
	cmd.Dir = wtPath
	hasChanges := cmd.Run() != nil

	if !hasChanges {
		cmd = exec.Command("git", "diff", "--cached", "--quiet", "HEAD")
		cmd.Dir = wtPath
		hasChanges = cmd.Run() != nil
	}

	if !hasChanges {
		cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard")
		cmd.Dir = wtPath
		out, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			hasChanges = true
		}
	}

	if !hasChanges {
		return
	}

	fmt.Println("Stashing changes...")
	dateStr := time.Now().Format("2006-01-02")
	msg := fmt.Sprintf("wt-delete: saved from worktree '%s' on %s", name, dateStr)

	// Run stash in the worktree directory
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = wtPath
	addCmd.Run()

	createCmd := exec.Command("git", "stash", "create", msg)
	createCmd.Dir = wtPath
	out, err := createCmd.Output()
	if err != nil {
		return
	}

	hash := strings.TrimSpace(string(out))
	if hash == "" {
		return
	}

	storeCmd := exec.Command("git", "stash", "store", hash, "-m", msg)
	storeCmd.Dir = wtPath
	storeCmd.Run()

	resetCmd := exec.Command("git", "reset", "--hard", "HEAD")
	resetCmd.Dir = wtPath
	resetCmd.Run()

	cleanCmd := exec.Command("git", "clean", "-fd")
	cleanCmd.Dir = wtPath
	cleanCmd.Run()

	fmt.Printf("Changes stashed (hash: %s). Recover with 'git stash list' or 'git stash apply %s'\n", hash, hash)
}
