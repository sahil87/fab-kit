# Tasks: Go wt Binary

**Change**: 260310-qbiq-go-wt-binary
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `src/go/fab/cmd/wt/main.go` — cobra root command `wt` with `SilenceUsage`, `SilenceErrors`, register all subcommands. Build entry point for the wt binary.

## Phase 2: Core Implementation — internal/worktree/ extensions

- [x] T002 [P] Add `src/go/fab/internal/worktree/context.go` — `RepoContext` struct and `GetRepoContext()` function: derive main repo root, repo name, and `<repo>.worktrees/` dir from `git rev-parse --git-common-dir`. Also `IsWorktree()`, `DefaultBranch()`, `ValidateBranchName()`, `DeriveWorktreeName()`.
- [x] T003 [P] Add `src/go/fab/internal/worktree/names.go` — adjective/noun word lists (~50 each), `GenerateRandomName()`, `GenerateUniqueName(worktreesDir string, maxRetries int)` with collision detection.
- [x] T004 [P] Add `src/go/fab/internal/worktree/git.go` — `HasUncommittedChanges()`, `HasUntrackedFiles()`, `HasUnpushedCommits(branch)`, `GetUnpushedCount(branch)`, `BranchExistsLocally(branch)`, `BranchExistsRemotely(branch)`, `FetchRemoteBranch(branch)`, `DeleteLocalBranch(branch, force)`, `DeleteRemoteBranch(branch)`.
- [x] T005 [P] Add `src/go/fab/internal/worktree/stash.go` — `StashCreate(msg)` (hash-based via `git stash create`/`git stash store`/`git reset --hard`/`git clean -fd`) and `StashApply(hash)`.
- [x] T006 [P] Add `src/go/fab/internal/worktree/rollback.go` — `Rollback` type with LIFO stack: `Register(cmd)`, `Execute()`, `Disarm()`. `Execute` runs all commands even if individual ones fail.
- [x] T007 [P] Add `src/go/fab/internal/worktree/menu.go` — `ShowMenu(prompt, options, defaultIdx)` with Cancel option, numeric input validation, default on empty. Color support via `NO_COLOR` env var.
- [x] T008 [P] Add `src/go/fab/internal/worktree/platform.go` — `DetectOS()`, `IsByobuSession()`, `IsTmuxSession()`. Check `BYOBU_TTY`/`BYOBU_BACKEND`/`BYOBU_SESSION`/`TMUX` env vars.
- [x] T009 [P] Add `src/go/fab/internal/worktree/errors.go` — exit code constants (`ExitSuccess=0`, `ExitGeneralError=1`, `ExitInvalidArgs=2`, `ExitGitError=3`, `ExitRetryExhausted=4`, `ExitByobuTabError=5`, `ExitTmuxWindowError=6`), color constants, `WtError(what, why, fix)` structured error formatter.
- [x] T010 Add `src/go/fab/internal/worktree/crud.go` — `EnsureWorktreesDir(dir)`, `CheckNameCollision(worktreesDir, name)`, `CreateWorktree(path, branch, newBranch)`, `RemoveWorktree(path, force)` (includes prune), `CreateBranchWorktree(branch, name, repoCtx)`, `CreateExploratoryWorktree(name, repoCtx)`, `RunWorktreeSetup(wtPath, mode, initScript)`.
- [x] T011 Add `src/go/fab/internal/worktree/apps.go` — `AppInfo` struct, `BuildAvailableApps()` (detect VSCode, Cursor, Ghostty, iTerm2, Terminal.app, GNOME Terminal, Konsole, Finder/Nautilus/Dolphin, clipboard, byobu tab, tmux window), `ResolveApp(input)`, `DetectDefaultApp()`, `OpenInApp(cmd, path, repoName, wtName)`, `SaveLastApp(cmd)`. Platform-specific detection via `exec.LookPath`, macOS `mdfind`, Linux .desktop files.

## Phase 3: Subcommands

- [x] T012 Add `src/go/fab/cmd/wt/create.go` — `wt create [flags] [branch]` subcommand. Parse flags (`--worktree-name`, `--worktree-init`, `--worktree-open`, `--reuse`, `--non-interactive`). Dirty-state check, name resolution, worktree creation with rollback/signal handling, init, open, porcelain output.
- [x] T013 Add `src/go/fab/cmd/wt/list.go` — `wt list [flags]` subcommand. Parse flags (`--path <name>`, `--json`). Formatted table (name, branch, dirty `*`, unpushed `↑N`, path), JSON output, path lookup. Use existing `worktree.List()` for worktree discovery, extend with status checking.
- [x] T014 Add `src/go/fab/cmd/wt/open.go` — `wt open [flags] [name|path]` subcommand. Parse flags (`--app <name>`). Worktree resolution (name, path, current, selection menu). App detection, menu, open, last-app cache.
- [x] T015 Add `src/go/fab/cmd/wt/delete.go` — `wt delete [flags]` subcommand. Parse flags (`--worktree-name`, `--delete-branch`, `--delete-remote`, `--delete-all`, `--stash`/`-s`, `--non-interactive`). Resolution order, uncommitted changes handling, unpushed warning, confirmation, worktree removal, branch cleanup, orphaned branch cleanup.
- [x] T016 Add `src/go/fab/cmd/wt/init.go` — `wt init` subcommand. Find init script from main repo root, run in current directory. Guidance message if missing.

## Phase 4: Build & Integration

- [x] T017 [P] Update `justfile` — add `build-wt`, `build-wt-target os arch`, `build-wt-all` recipes. Update `build-all` to include `build-wt-all`. Update `package-kit` to verify and include wt binary in per-platform archives at `.kit/bin/wt`.
- [x] T018 [P] Remove legacy shell scripts — delete `fab/.kit/packages/wt/bin/wt-create`, `wt-delete`, `wt-init`, `wt-list`, `wt-open`, `wt-pr`, `fab/.kit/packages/wt/lib/wt-common.sh`. Remove `fab/.kit/packages/wt/` directory entirely.
- [x] T019 [P] Add tests — `src/go/fab/internal/worktree/context_test.go` (branch validation, name derivation), `src/go/fab/internal/worktree/names_test.go` (random name format, uniqueness retry), `src/go/fab/internal/worktree/git_test.go` (change detection helpers), `src/go/fab/internal/worktree/rollback_test.go` (LIFO order, disarm), `src/go/fab/internal/worktree/errors_test.go` (error format, NO_COLOR).

---

## Execution Order

- T001 blocks T012–T016 (subcommands need root command)
- T002–T011 are parallel (independent internal packages)
- T010 depends on T002 (uses RepoContext), T003 (uses name generation), T004 (uses git ops), T006 (uses rollback)
- T011 depends on T007 (uses menu), T008 (uses platform detection)
- T012–T016 depend on T002–T011 (subcommands use internal packages)
- T017–T019 are parallel and depend on T012–T016 (build/cleanup/tests after implementation)
