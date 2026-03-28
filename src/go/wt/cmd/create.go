package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	wt "github.com/wvrdz/fab-kit/src/go/wt/internal/worktree"
)

func createCmd() *cobra.Command {
	var (
		worktreeName   string
		worktreeInit   string
		worktreeOpen   string
		reuse          bool
		nonInteractive bool
		base           string
	)

	cmd := &cobra.Command{
		Use:   "create [branch]",
		Short: "Create a git worktree",
		Long: `Create a git worktree for parallel development.

When BRANCH is omitted, creates an exploratory worktree with a random name.
When BRANCH is provided, checks out that branch (fetching from remote if needed)
or creates a new branch.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var branchArg string
			if len(args) > 0 {
				branchArg = args[0]
			}

			// Apply defaults
			if worktreeInit == "" {
				worktreeInit = "true"
			}
			if worktreeOpen == "" {
				if nonInteractive {
					worktreeOpen = "skip"
				} else {
					worktreeOpen = "prompt"
				}
			}

			// Validate --reuse requires --worktree-name
			if reuse && worktreeName == "" {
				wt.PrintError(
					"--reuse requires --worktree-name",
					"--reuse only works with an explicit worktree name",
					"Example: wt create --reuse --worktree-name my-feature branch-name")
				os.Exit(wt.ExitInvalidArgs)
			}

			// Validate git repo
			if err := wt.ValidateGitRepo(); err != nil {
				wt.ExitWithError(wt.ExitGitError,
					"Not a git repository",
					"This command requires a git repository",
					"Navigate to a git repository and try again")
			}

			ctx, err := wt.GetRepoContext()
			if err != nil {
				wt.ExitWithError(wt.ExitGeneralError,
					"Not a git repository",
					"This command must be run from within a git repository",
					"Navigate to a git repository and try again")
			}

			// Set up rollback and signal handling
			rb := wt.NewRollback()
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				fmt.Println()
				rb.Execute()
				os.Exit(130)
			}()
			defer func() {
				rb.Execute()
			}()

			// Validate branch name
			if branchArg != "" {
				if err := wt.ValidateBranchName(branchArg); err != nil {
					wt.ExitWithError(wt.ExitInvalidArgs,
						"Invalid branch name",
						fmt.Sprintf("Branch name '%s' contains invalid characters", branchArg),
						"Use alphanumeric characters, hyphens, and single slashes")
				}
			}

			// Validate --base ref only when it will actually be used.
			// When --reuse is set, or when BRANCH already exists locally/remotely,
			// later logic ignores --base, so we skip validation here to avoid
			// failing commands like `wt create --reuse --base <bad>` or
			// `wt create <existing-branch> --base <bad>`.
			if base != "" && !reuse {
				existingBranch := false
				if branchArg != "" {
					if err := exec.Command("git", "rev-parse", "--verify", branchArg).Run(); err == nil {
						existingBranch = true
					}
				}
				if !existingBranch {
					if err := exec.Command("git", "rev-parse", "--verify", base).Run(); err != nil {
						wt.ExitWithError(wt.ExitInvalidArgs,
							fmt.Sprintf("Invalid --base ref: %s", base),
							fmt.Sprintf("'%s' does not resolve to a valid git object", base),
							"Provide a valid branch name, tag, or commit SHA")
					}
				}
			}

			// Dirty-state check
			if !nonInteractive && (wt.HasUncommittedChanges() || wt.HasUntrackedFiles()) {
				fmt.Fprintf(os.Stderr, "%sWarning: main repo has uncommitted changes%s\n",
					wt.ColorYellow, wt.ColorReset)
				choice, err := wt.ShowMenu("How to proceed?", []string{
					"Continue anyway",
					"Stash changes first",
					"Abort",
				}, -1)
				if err != nil {
					wt.ExitWithError(wt.ExitGeneralError, "Menu error", err.Error(), "")
				}
				switch choice {
				case 1: // continue
				case 2:
					stashID, err := wt.StashCreate("wt-create: pre-creation stash")
					if err != nil {
						wt.ExitWithError(wt.ExitGeneralError,
							"Failed to create stash",
							err.Error(),
							"Resolve any repository issues and try again")
					}
					if stashID != "" {
						fmt.Fprintf(os.Stderr, "Created stash %s for pre-creation changes.\n", stashID)
					}
				case 3, 0:
					rb.Disarm()
					return nil
				}
			}

			// Determine suggested name
			var suggestedName string
			if branchArg == "" {
				suggestedName, err = wt.GenerateUniqueName(ctx.WorktreesDir, 10)
				if err != nil {
					wt.ExitWithError(wt.ExitRetryExhausted,
						"Could not find unique worktree name",
						"All 10 random name attempts collided with existing worktrees",
						fmt.Sprintf("Remove some worktrees from %s or increase retries", ctx.WorktreesDir))
				}
			} else {
				suggestedName = wt.DeriveWorktreeName(branchArg)
			}

			// Resolve final name
			var finalName string
			if worktreeName != "" {
				finalName = worktreeName
			} else if nonInteractive {
				finalName = suggestedName
			} else {
				finalName = wt.PromptWithDefault("Worktree name", suggestedName)
			}

			// Check collision
			if wt.CheckNameCollision(ctx.WorktreesDir, finalName) {
				if reuse {
					fmt.Fprintf(os.Stderr, "Reusing existing worktree: %s\n", finalName)
					rb.Disarm()
					fmt.Println(filepath.Join(ctx.WorktreesDir, finalName))
					return nil
				}
				wt.ExitWithError(wt.ExitGeneralError,
					fmt.Sprintf("Worktree '%s' already exists", finalName),
					fmt.Sprintf("A worktree with this name already exists at %s/%s", ctx.WorktreesDir, finalName),
					"Remove the existing worktree or use a different branch name")
			}

			// Create worktree
			var wtPath string
			if branchArg == "" {
				wtPath, err = wt.CreateExploratoryWorktree(finalName, ctx, rb, base)
				if err != nil {
					wt.ExitWithError(wt.ExitGitError, "Failed to create worktree", err.Error(),
						"Check if the branch already exists or if there are permission issues")
				}
				fmt.Fprintf(os.Stderr, "Created worktree: %s\nPath: %s\nBranch: %s\n", finalName, wtPath, finalName)
			} else {
				// Warn-and-ignore --base for existing branches
				effectiveBase := base
				if base != "" {
					if wt.BranchExistsLocally(branchArg) {
						fmt.Fprintf(os.Stderr, "--base ignored: branch already exists locally\n")
						effectiveBase = ""
					} else if wt.BranchExistsRemotely(branchArg) {
						fmt.Fprintf(os.Stderr, "--base ignored: fetching existing remote branch\n")
						effectiveBase = ""
					}
				}
				wtPath, err = wt.CreateBranchWorktree(branchArg, finalName, ctx, rb, effectiveBase)
				if err != nil {
					wt.ExitWithError(wt.ExitGitError, "Failed to create worktree", err.Error(),
						"The branch may already be checked out in another worktree")
				}
				fmt.Fprintf(os.Stderr, "Created worktree: %s\nPath: %s\nBranch: %s\n", finalName, wtPath, branchArg)
			}

			// Setup
			if worktreeInit == "true" {
				initScript := wt.InitScriptPath()
				if nonInteractive || branchArg == "" {
					// Auto-run init
					if err := wt.RunWorktreeSetup(wtPath, "force", initScript, ctx.RepoRoot); err != nil {
						// Init failure triggers rollback — must execute before os.Exit
						rb.Execute()
						wt.ExitWithError(wt.ExitGeneralError, "Init script failed", err.Error(),
							"Check the init script for errors")
					}
				} else {
					// Prompt for init
					if err := wt.RunWorktreeSetup(wtPath, "", initScript, ctx.RepoRoot); err != nil {
						rb.Execute()
						wt.ExitWithError(wt.ExitGeneralError, "Init script failed", err.Error(),
							"Check the init script for errors")
					}
				}
			}

			// Open
			var suppressPath bool
			if worktreeOpen == "prompt" {
				apps := wt.BuildAvailableApps()
				if len(apps) > 0 {
					defaultIdx := wt.DetectDefaultApp(apps)
					appNames := make([]string, len(apps))
					for i, a := range apps {
						appNames[i] = a.Name
					}
					choice, err := wt.ShowMenu("Open in:", appNames, defaultIdx)
					if err == nil && choice > 0 && choice <= len(apps) {
						selected := apps[choice-1]
						wt.SaveLastApp(selected.Cmd)
						if openErr := wt.OpenInApp(selected.Cmd, wtPath, ctx.RepoName, finalName); openErr != nil {
							fmt.Fprintf(os.Stderr, "Warning: could not open in %s: %s\n", selected.Name, openErr)
						}
						if selected.Cmd == "open_here" {
							suppressPath = true
						}
					}
				}
			} else if worktreeOpen != "skip" {
				apps := wt.BuildAvailableApps()
				resolved, err := wt.ResolveApp(worktreeOpen, apps)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %s\n", err)
				} else {
					wt.SaveLastApp(resolved.Cmd)
					if openErr := wt.OpenInApp(resolved.Cmd, wtPath, ctx.RepoName, finalName); openErr != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not open in %s: %s\n", resolved.Name, openErr)
					}
					if resolved.Cmd == "open_here" {
						suppressPath = true
					}
				}
			}

			// Success — disarm rollback
			rb.Disarm()

			// Output the worktree path as the last line
			if !suppressPath {
				fmt.Println(wtPath)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&worktreeName, "worktree-name", "", "Set worktree name (skips name prompt)")
	cmd.Flags().StringVar(&worktreeInit, "worktree-init", "", "Run worktree init script: true (default) or false")
	cmd.Flags().StringVar(&worktreeOpen, "worktree-open", "", "Open in app after creation, or 'skip'")
	cmd.Flags().BoolVar(&reuse, "reuse", false, "Reuse existing worktree if name collides (requires --worktree-name)")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "No prompts, porcelain output")
	cmd.Flags().StringVar(&base, "base", "", "Git ref (branch, tag, SHA) to use as start-point for new branch")

	return cmd
}
