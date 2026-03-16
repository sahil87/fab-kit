# Spec: Multi-Worktree Delete

**Change**: 260316-euw2-multi-worktree-delete
**Created**: 2026-03-16
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Interactive multi-select menu (checkbox UI) — separate UX change, out of scope
- Glob/pattern matching for worktree names — explicit names only

## wt delete: Positional Arguments

### Requirement: Accept Positional Arguments as Worktree Names

The `wt delete` command SHALL accept zero or more positional arguments, each interpreted as a worktree name to delete. When positional args are provided, they take precedence over the `--worktree-name` flag in the resolution order.

The Cobra command definition SHALL change from `cobra.NoArgs` to `cobra.ArbitraryArgs`.

#### Scenario: Single positional argument
- **GIVEN** a worktree named `swift-fox` exists
- **WHEN** the user runs `wt delete swift-fox --non-interactive`
- **THEN** the worktree `swift-fox` is deleted
- **AND** its branch is cleaned up per `--delete-branch` / `--delete-remote` defaults

#### Scenario: Multiple positional arguments
- **GIVEN** worktrees `swift-fox`, `calm-bear`, and `happy-finch` exist
- **WHEN** the user runs `wt delete swift-fox calm-bear --non-interactive`
- **THEN** both `swift-fox` and `calm-bear` are deleted
- **AND** `happy-finch` is NOT deleted

#### Scenario: No positional arguments and no flags
- **GIVEN** no positional args and no `--worktree-name` flag
- **WHEN** the user runs `wt delete` (interactive)
- **THEN** the existing resolution order applies (current worktree or menu)

### Requirement: Validate All Names Before Deleting

When multiple positional arguments are provided, the command SHALL resolve all names against the worktree list before performing any deletions. If any name does not match an existing worktree, the command SHALL print all unresolved names and exit with a non-zero status code without deleting anything.

#### Scenario: All names valid
- **GIVEN** worktrees `alpha` and `bravo` exist
- **WHEN** the user runs `wt delete alpha bravo --non-interactive`
- **THEN** both worktrees are deleted successfully

#### Scenario: One name invalid (fail-fast)
- **GIVEN** worktree `alpha` exists but `typo-name` does not
- **WHEN** the user runs `wt delete alpha typo-name --non-interactive`
- **THEN** the command exits non-zero
- **AND** stderr contains `Worktree 'typo-name' not found`
- **AND** worktree `alpha` is NOT deleted (no partial execution)

#### Scenario: Multiple invalid names
- **GIVEN** neither `foo` nor `bar` exist as worktrees
- **WHEN** the user runs `wt delete foo bar --non-interactive`
- **THEN** the command exits non-zero
- **AND** stderr lists both `foo` and `bar` as not found

#### Scenario: Duplicate names
- **GIVEN** worktree `alpha` exists
- **WHEN** the user runs `wt delete alpha alpha --non-interactive`
- **THEN** the command deduplicates and deletes `alpha` once

### Requirement: Single Confirmation for Multi-Delete

In interactive mode (no `--non-interactive`), the command SHALL display a summary of all worktrees to be deleted and present a single confirmation prompt. Per-worktree confirmation SHALL NOT be used.

#### Scenario: Interactive multi-delete confirmation
- **GIVEN** worktrees `alpha` and `bravo` exist
- **WHEN** the user runs `wt delete alpha bravo` (interactive)
- **THEN** a summary listing both worktrees (name, branch, path) is displayed
- **AND** a single "Delete these 2 worktrees?" prompt appears

#### Scenario: Non-interactive multi-delete skips prompt
- **GIVEN** worktrees `alpha` and `bravo` exist
- **WHEN** the user runs `wt delete alpha bravo --non-interactive`
- **THEN** both worktrees are deleted without any confirmation prompt

### Requirement: Sequential Deletion with Continue-on-Error

After validation passes and confirmation is given, worktrees SHALL be deleted sequentially in the order specified. If deletion of one worktree fails (e.g., git error), the command SHALL print a warning to stderr and continue with the remaining worktrees.

#### Scenario: One deletion fails mid-batch
- **GIVEN** worktrees `alpha`, `bravo`, and `charlie` exist
- **AND** `bravo` has a locked worktree (git removal will fail)
- **WHEN** the user runs `wt delete alpha bravo charlie --non-interactive`
- **THEN** `alpha` is deleted successfully
- **AND** a warning for `bravo` is printed to stderr
- **AND** `charlie` is deleted successfully

### Requirement: Branch Cleanup Per Worktree

Each deleted worktree SHALL have its branch cleaned up according to the `--delete-branch` and `--delete-remote` flags, consistent with existing single-delete behavior.

#### Scenario: Multi-delete with branch cleanup
- **GIVEN** worktrees `alpha` (branch `alpha`) and `bravo` (branch `bravo`) exist
- **WHEN** the user runs `wt delete alpha bravo --non-interactive --delete-branch true`
- **THEN** both worktrees are deleted
- **AND** both branches `alpha` and `bravo` are deleted locally

#### Scenario: Multi-delete preserving branches
- **GIVEN** worktrees `alpha` and `bravo` exist
- **WHEN** the user runs `wt delete alpha bravo --non-interactive --delete-branch false`
- **THEN** both worktrees are deleted
- **AND** both branches are preserved

### Requirement: Stash Flag with Multi-Delete

When `--stash` is provided with multiple worktrees, uncommitted changes in each worktree SHALL be stashed before that worktree is deleted.

#### Scenario: Multi-delete with stash
- **GIVEN** worktrees `alpha` and `bravo` exist
- **AND** both have uncommitted changes
- **WHEN** the user runs `wt delete alpha bravo --non-interactive --stash`
- **THEN** changes in `alpha` are stashed, then `alpha` is deleted
- **AND** changes in `bravo` are stashed, then `bravo` is deleted

## wt delete: Resolution Order Update

### Requirement: Updated Resolution Order

The `RunE` handler SHALL evaluate in this order:

1. `--delete-all` → `handleDeleteAll()` (unchanged)
2. Positional args (1+) → `handleDeleteMultiple()`
3. `--worktree-name` set → `handleDeleteByName()` (deprecated path)
4. In a worktree (no args, no flags) → `handleDeleteCurrent()` (unchanged)
5. Non-interactive, no target → error (unchanged)
6. Interactive, no target → `handleDeleteMenu()` (unchanged)

#### Scenario: --delete-all takes precedence over positional args
- **GIVEN** worktrees `alpha` and `bravo` exist
- **WHEN** the user runs `wt delete alpha --delete-all --non-interactive`
- **THEN** all worktrees are deleted (not just `alpha`)

#### Scenario: Positional args take precedence over --worktree-name
- **GIVEN** worktrees `alpha` and `bravo` exist
- **WHEN** the user runs `wt delete alpha --worktree-name bravo --non-interactive`
- **THEN** the command exits with an error: cannot mix positional args and --worktree-name

## wt delete: Deprecate --worktree-name

### Requirement: Deprecate --worktree-name Flag

The `--worktree-name` flag SHALL be marked as deprecated. When used, it SHALL print a deprecation notice to stderr: `Warning: --worktree-name is deprecated, use positional arguments instead`. The flag SHALL continue to work for backward compatibility.

#### Scenario: --worktree-name still works with deprecation warning
- **GIVEN** worktree `alpha` exists
- **WHEN** the user runs `wt delete --worktree-name alpha --non-interactive`
- **THEN** worktree `alpha` is deleted
- **AND** stderr contains "deprecated"

#### Scenario: Mixing positional args and --worktree-name is an error
- **GIVEN** worktrees `alpha` and `bravo` exist
- **WHEN** the user runs `wt delete alpha --worktree-name bravo --non-interactive`
- **THEN** the command exits non-zero
- **AND** stderr contains an error about mixing positional args and --worktree-name

## Clarifications

### Session 2026-03-16 (auto-clarify)

| # | Action | Detail |
|---|--------|--------|
| 4 | Resolved | Cobra MarkDeprecated natively supports deprecation with continued functionality; flag and handler confirmed in delete.go |
| 5 | Resolved | handleDeleteAll already uses single ShowMenu prompt for batch — direct precedent at delete.go:282-294 |
| 6 | Resolved | handleDeleteAll already implements sequential iteration with stderr warning + continue — exact pattern at delete.go:296-309 |
| 7 | Resolved | Existing resolution order in delete.go:66-86 already places --delete-all first; positional args slot naturally at position 2 |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use positional args as primary interface | Confirmed from intake #1. Standard Cobra pattern, matches `git worktree remove` | S:70 R:85 A:90 D:90 |
| 2 | Certain | Fail-fast validation before any deletion | Confirmed from intake #2. Validate-then-execute prevents partial deletes from typos | S:75 R:70 A:95 D:95 |
| 3 | Certain | Deduplicate repeated names silently | Obvious safe behavior — deleting the same worktree twice would error on the second attempt | S:80 R:95 A:90 D:95 |
| 4 | Certain | Deprecate --worktree-name with warning, keep working | Clarified — Cobra MarkDeprecated natively supports this; flag at delete.go:89, handleDeleteByName at :175 continues to work | S:60 R:60 A:95 D:80 |
| 5 | Certain | Single confirmation prompt for batch delete | Clarified — handleDeleteAll (delete.go:282-294) already uses single ShowMenu prompt for N worktrees; direct precedent | S:65 R:80 A:95 D:85 |
| 6 | Certain | Sequential deletion with continue-on-error | Clarified — handleDeleteAll (delete.go:296-309) iterates sequentially with stderr warning + continue on failure; exact pattern to reuse | S:60 R:85 A:95 D:75 |
| 7 | Certain | --delete-all takes precedence over positional args | Clarified — delete.go:66-86 already checks deleteAll first in resolution order; positional args slot in at position 2 naturally | S:55 R:85 A:95 D:80 |
| 8 | Confident | Mixing --worktree-name and positional args is an error | Ambiguous intent — better to fail clearly than guess which takes precedence | S:60 R:90 A:80 D:70 |
| 9 | Tentative | Do not add multi-select to interactive menu | Confirmed from intake #6. Deferred — separate UX change | S:50 R:80 A:60 D:55 |

9 assumptions (7 certain, 1 confident, 1 tentative, 0 unresolved).
