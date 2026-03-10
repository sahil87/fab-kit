package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	wt "github.com/wvrdz/fab-kit/src/wt-go/internal/worktree"
)

func openCmd() *cobra.Command {
	var appFlag string

	cmd := &cobra.Command{
		Use:   "open [name|path]",
		Short: "Open a worktree in an application",
		Long: `Open a git worktree in a detected application (editor, terminal, file manager).

When called without arguments from a worktree, opens the current worktree.
When called without arguments from the main repo, shows a selection menu.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var target string
			if len(args) > 0 {
				target = args[0]
			}

			if err := wt.ValidateGitRepo(); err != nil {
				wt.ExitWithError(wt.ExitGitError,
					"Not a git repository",
					"This command requires a git repository",
					"Navigate to a git repository and try again")
			}

			ctx, err := wt.GetRepoContext()
			if err != nil {
				wt.ExitWithError(wt.ExitGeneralError, "Cannot get repo context", err.Error(), "")
			}

			var wtPath, wtName string

			if target != "" {
				// Check if it's a direct path
				if info, err := os.Stat(target); err == nil && info.IsDir() {
					wtPath = target
					wtName = filepath.Base(wtPath)
				} else {
					// Try as worktree name
					path, err := resolveWorktreeByName(target, ctx)
					if err != nil {
						wt.ExitWithError(wt.ExitGeneralError,
							fmt.Sprintf("Worktree '%s' not found", target),
							"No worktree with that name and not an existing directory",
							"Use 'wt list' to see available worktrees")
					}
					wtPath = path
					wtName = target
				}
			} else if wt.IsWorktree() {
				// In a worktree — open it
				wtPath, err = wt.CurrentWorktreeTopLevel()
				if err != nil {
					wt.ExitWithError(wt.ExitGeneralError, "Cannot determine worktree root", err.Error(), "")
				}
				wtName = filepath.Base(wtPath)
			} else {
				// In main repo — show selection
				if appFlag != "" {
					wt.ExitWithError(wt.ExitInvalidArgs,
						"No worktree specified",
						"--app requires a worktree name or path, or run from within a worktree",
						"Example: wt open --app code my-worktree")
				}
				return selectAndOpen(ctx)
			}

			// Open with specified app or show menu
			if appFlag != "" {
				apps := wt.BuildAvailableApps()
				resolved, err := wt.ResolveApp(appFlag, apps)
				if err != nil {
					wt.ExitWithError(wt.ExitGeneralError,
						fmt.Sprintf("Unknown app: %s", appFlag),
						fmt.Sprintf("App '%s' is not available on this system", appFlag),
						"Available apps can be seen with: wt open (then check the menu)")
				}
				if openErr := wt.OpenInApp(resolved.Cmd, wtPath, ctx.RepoName, wtName); openErr != nil {
					exitCode := wt.ExitGeneralError
					if strings.Contains(resolved.Cmd, "byobu") {
						exitCode = wt.ExitByobuTabError
					} else if strings.Contains(resolved.Cmd, "tmux") {
						exitCode = wt.ExitTmuxWindowError
					}
					wt.ExitWithError(exitCode,
						fmt.Sprintf("Failed to open in %s", resolved.Name),
						openErr.Error(),
						"Verify the application is running and retry")
				}
			} else {
				return handleAppMenu(wtPath, ctx.RepoName, wtName)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&appFlag, "app", "", "Open in specified app, skipping the menu")

	return cmd
}

func resolveWorktreeByName(name string, ctx *wt.RepoContext) (string, error) {
	entries, err := listWorktreeEntries()
	if err != nil {
		return "", err
	}

	for _, e := range entries {
		entryName := filepath.Base(e.path)
		if strings.EqualFold(entryName, name) {
			return e.path, nil
		}
	}

	return "", fmt.Errorf("not found")
}

func handleAppMenu(wtPath, repoName, wtName string) error {
	apps := wt.BuildAvailableApps()
	if len(apps) == 0 {
		fmt.Println("No supported applications detected.")
		return nil
	}

	defaultIdx := wt.DetectDefaultApp(apps)
	appNames := make([]string, len(apps))
	for i, a := range apps {
		appNames[i] = a.Name
	}

	choice, err := wt.ShowMenu("Open in:", appNames, defaultIdx)
	if err != nil {
		return err
	}
	if choice == 0 {
		return nil
	}

	selected := apps[choice-1]
	wt.SaveLastApp(selected.Cmd)

	if openErr := wt.OpenInApp(selected.Cmd, wtPath, repoName, wtName); openErr != nil {
		exitCode := wt.ExitGeneralError
		if strings.Contains(selected.Cmd, "byobu") {
			exitCode = wt.ExitByobuTabError
		} else if strings.Contains(selected.Cmd, "tmux") {
			exitCode = wt.ExitTmuxWindowError
		}
		wt.ExitWithError(exitCode,
			fmt.Sprintf("Failed to open in %s", selected.Name),
			openErr.Error(),
			"Verify the application is running and retry")
	}

	return nil
}

func selectAndOpen(ctx *wt.RepoContext) error {
	entries, err := listWorktreeEntries()
	if err != nil {
		return err
	}

	type wtOption struct {
		path string
		name string
	}

	var options []wtOption
	var newestPath string
	var newestTime int64

	for _, e := range entries {
		if e.path == ctx.RepoRoot {
			continue
		}
		name := filepath.Base(e.path)
		options = append(options, wtOption{path: e.path, name: name})

		// Track most recently modified
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

	// Find default index
	defaultIdx := 1
	for i, o := range options {
		if o.path == newestPath {
			defaultIdx = i + 1
			break
		}
	}

	// Build menu
	menuNames := make([]string, len(options))
	for i, o := range options {
		// Get branch for display
		branch := getBranchForPath(o.path)
		menuNames[i] = fmt.Sprintf("%s (%s)", o.name, branch)
	}

	choice, err := wt.ShowMenu("Select worktree to open:", menuNames, defaultIdx)
	if err != nil {
		return err
	}
	if choice == 0 {
		fmt.Println("Cancelled.")
		return nil
	}

	selected := options[choice-1]
	return handleAppMenu(selected.path, ctx.RepoName, selected.name)
}

func getBranchForPath(wtPath string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = wtPath
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
