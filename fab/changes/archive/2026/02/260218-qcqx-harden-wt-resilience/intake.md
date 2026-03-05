# Intake: Harden wt Package Resilience

**Change**: 260218-qcqx-harden-wt-resilience
**Created**: 2026-02-18
**Status**: Draft

## Origin

> User compared the local `wt` package (`fab/.kit/packages/wt/`) with [johnlindquist/worktree-cli](https://github.com/johnlindquist/worktree-cli) (a TypeScript CLI for git worktrees). Analysis identified 6 improvements to adopt, drawing on patterns from that project. Backlog item `[qcqx]`.

The comparison was a structured review of both tools' API surfaces, covering command coverage, error handling, safety patterns, and UX. The local `wt` package already leads in app detection (`wt-open` supports 15+ editors/terminals), random name generation, and zero-dependency architecture. The gaps are in resilience and workflow coverage.

## Why

1. **Partial failure leaves orphaned state.** `wt-create` can fail mid-operation (after `git worktree add` but before init completes), leaving behind an orphaned worktree directory and local branch with no cleanup. The user must manually `git worktree remove` and `git branch -D`.

2. **Ctrl-C during creation is destructive.** No signal handlers exist — a SIGINT during `wt-create` or `wt-delete` can leave the repository in an inconsistent state (worktree metadata in `.git/worktrees/` with no matching directory, or vice versa).

3. **Index-based stash is fragile.** `wt-delete --stash` uses `git stash push`/`git stash pop` which relies on stack position. Concurrent operations in another terminal can shift the stash index, causing the wrong stash to be restored.

4. **No PR worktree workflow.** The most common reason to create a worktree is to review a PR without disrupting the current working state. This currently requires manually running `gh pr view`, copying the branch, and calling `wt-create <branch>`.

5. **Invalid branch names produce unclear errors.** Git's rejection messages for invalid characters (`~`, `^`, `..`, `.lock`) are cryptic. Pre-validation with a clear error would improve UX.

6. **`wt-create` doesn't check main repo state.** Creating a worktree while the main repo has uncommitted changes can cause confusion, especially if the new branch is based on a dirty HEAD.

## What Changes

### 1. New command: `wt-pr` (`fab/.kit/packages/wt/bin/wt-pr`)

A new script that creates a worktree from a GitHub PR number or interactive selection.

**Usage**: `wt-pr [OPTIONS] [PR_NUMBER]`

**Flags**:
- `--worktree-name <name>` — override worktree directory name
- `--worktree-init <bool>` — run init script (default: true)
- `--worktree-open <app>` — open after creation, or `skip`
- `--non-interactive` — use defaults, no prompts
- `help` / `--help` / `-h`

**Flow**:
1. If no PR number: run `gh pr list --json number,title,headRefName` and show via `wt_show_menu`
2. Get branch name: `gh pr view $PR --json headRefName --jq .headRefName`
3. Fetch without checkout: `git fetch origin refs/pull/$PR/head:<local-branch>`
4. Delegate to `wt-create` internals (reuse `wt_create_branch_worktree` or equivalent)

**Dependency**: Requires `gh` CLI. Should validate with `command -v gh` and exit with `wt_error` if missing.

### 2. Atomic rollback in `wt-create` (`fab/.kit/packages/wt/bin/wt-create`)

Add a rollback stack and `EXIT` trap to `wt-create`:

```bash
WT_ROLLBACK_STACK=()
wt_register_rollback() { WT_ROLLBACK_STACK+=("$1"); }
wt_rollback() {
  for ((i=${#WT_ROLLBACK_STACK[@]}-1; i>=0; i--)); do
    eval "${WT_ROLLBACK_STACK[$i]}" 2>/dev/null
  done
}
wt_disarm_rollback() { WT_ROLLBACK_STACK=(); }
```

Registration points in `wt-create`:
- After `git worktree add` → register `git worktree remove --force <path>`
- After local branch creation → register `git branch -D <branch>`
- After directory creation → register `rm -rf <dir>`

On successful completion, call `wt_disarm_rollback`. The `EXIT` trap fires the stack on any non-zero exit.

These functions should live in `fab/.kit/packages/wt/lib/wt-common.sh` so `wt-pr` and future commands can reuse them.

### 3. SIGINT/SIGTERM trap handlers (`fab/.kit/packages/wt/lib/wt-common.sh`)

Add signal trapping to the shared library:

```bash
wt_cleanup_on_signal() {
  echo "" # newline after ^C
  wt_rollback
  exit 130  # standard SIGINT exit code
}
trap wt_cleanup_on_signal INT TERM
```

Applied in `wt-create` and `wt-delete` — the two commands with multi-step destructive operations. The trap should restore stashed changes if a stash was created during the operation.

### 4. Hash-based stash (`fab/.kit/packages/wt/lib/wt-common.sh`)

Replace index-based stash in `wt_stash_changes`:

**Current** (in `wt-delete`):
```bash
git stash push -m "wt-delete: $wt_name"
# ... later ...
git stash pop
```

**Proposed**:
```bash
wt_stash_create() {
  local msg="$1"
  git add -A
  local hash
  hash=$(git stash create "$msg")
  if [[ -n "$hash" ]]; then
    git reset --hard HEAD
    git clean -fd
    echo "$hash"
  fi
}

wt_stash_apply() {
  local hash="$1"
  [[ -n "$hash" ]] && git stash apply "$hash"
}
```

The hash is stable regardless of concurrent stash operations in other terminals.

### 5. Branch name validation (`fab/.kit/packages/wt/lib/wt-common.sh`)

Add `wt_validate_branch_name()`:

```bash
wt_validate_branch_name() {
  local branch="$1"
  [[ -z "$branch" ]] && return 1
  # Reject invalid git ref characters and patterns
  if [[ "$branch" =~ [[:space:]~^:?*\[] ]] || \
     [[ "$branch" == *".."* ]] || \
     [[ "$branch" == *".lock" ]] || \
     [[ "$branch" == *"/"*"/"*"/"* ]] || \
     [[ "$branch" == "."* ]] || \
     [[ "$branch" == *"/." ]]; then
    return 1
  fi
  return 0
}
```

Called in `wt-create` and `wt-pr` before any git operations. On failure: `wt_error "Invalid branch" "Branch name '$branch' contains invalid characters" "Use alphanumeric characters, hyphens, and single slashes"`.

### 6. Dirty-state check in `wt-create` (`fab/.kit/packages/wt/bin/wt-create`)

Before creating a worktree, check if the main repo has uncommitted changes:

```bash
if wt_has_uncommitted_changes || wt_has_untracked_files; then
  echo "${WT_YELLOW}Warning: main repo has uncommitted changes${WT_RESET}"
  wt_show_menu "How to proceed?" "" \
    "Continue anyway" \
    "Stash changes first" \
    "Abort"
  case "$WT_MENU_CHOICE" in
    1) ;; # continue
    2) wt_stash_create "wt-create: pre-creation stash" ;;
    3) exit 0 ;;
  esac
fi
```

In `--non-interactive` mode, default to "Continue anyway" (creating a worktree from a dirty main is not inherently broken, just worth flagging).

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document rollback stack, signal handling, and hash-based stash as wt package patterns

## Impact

- **Files created**: `fab/.kit/packages/wt/bin/wt-pr` (new command)
- **Files modified**: `fab/.kit/packages/wt/lib/wt-common.sh` (rollback, signal, stash, validation functions), `fab/.kit/packages/wt/bin/wt-create` (rollback integration, dirty-state check, validation call), `fab/.kit/packages/wt/bin/wt-delete` (hash-based stash migration)
- **Test files**: Existing test infrastructure at `fab/.kit/packages/wt/tests/` should receive tests for the new functions
- **No external dependencies added** — `gh` is already used elsewhere in the project; the package remains pure bash

## Open Questions

- Should `wt-pr` support GitLab MRs (via `glab`) in addition to GitHub PRs, or is GitHub-only sufficient for now?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rollback functions go in `wt-common.sh` | Constitution requires shell scripts; `wt-common.sh` is the existing shared library — all bin scripts source it | S:90 R:95 A:95 D:95 |
| 2 | Certain | `wt-pr` follows existing command patterns (same flag style, sourcing, error functions) | Existing 5 commands establish a clear pattern — deviating would violate code-quality principles | S:85 R:90 A:95 D:95 |
| 3 | Confident | Hash-based stash uses `git stash create` + `git reset --hard` | This is the pattern from worktree-cli, avoids index-based fragility, and is well-documented in git internals | S:80 R:70 A:80 D:85 |
| 4 | Confident | `wt-pr` requires `gh` CLI (no REST API fallback) | Project already uses `gh` (constitution allows single-binary utilities). REST fallback adds complexity for a niche case | S:70 R:80 A:75 D:70 |
| 5 | Confident | Dirty-state check defaults to "continue" in non-interactive mode | Creating a worktree from dirty state is safe (worktree gets clean copy of committed state); the warning is UX, not correctness | S:75 R:90 A:80 D:75 |
| 6 | Tentative | GitLab MR support deferred to future change | No evidence of GitLab usage in the project, but question left open for user input | S:50 R:85 A:45 D:60 |
<!-- assumed: GitHub-only for wt-pr — no GitLab remote URLs detected in project -->

6 assumptions (2 certain, 3 confident, 1 tentative, 0 unresolved).
