# Spec: Add fab-switch --blank to deactivate the current change

**Change**: 260212-egqa-switch-return-main
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/change-lifecycle.md`
<!-- clarified: Removed planning-skills.md — /fab-switch is documented in change-lifecycle.md, not planning-skills.md -->

## Non-Goals

- Pull/fetch latest from remote — the user controls remote sync
- Warn about uncommitted git changes — fab defers to git's own checkout behavior
- Add a `/fab-deactivate` or separate skill — this is a natural extension of `/fab-switch`
- Auto-checkout main/master on `--blank` — git branch management is a separate concern handled by `--branch`

## fab-switch: Deactivation Flag

### Requirement: Add `--blank` flag to deactivate the current change

`/fab-switch` SHALL accept a `--blank` flag that deactivates the current change by deleting `fab/current`. When `--blank` is provided, the skill SHALL NOT perform change folder matching — it operates solely on the fab change pointer.

`--blank` does NOT perform any git operations on its own. Git branch switching is handled by the existing `--branch` flag, which is orthogonal and can be combined with `--blank`.

#### Scenario: Deactivate with `--blank`

- **GIVEN** an active change pointed to by `fab/current`
- **WHEN** the user runs `/fab-switch --blank`
- **THEN** `fab/current` SHALL be deleted (not emptied or truncated)
- **AND** no git operations SHALL be performed
- **AND** the skill SHALL display a confirmation message

#### Scenario: Already deactivated (idempotent)

- **GIVEN** no `fab/current` file exists (no active change)
- **WHEN** the user runs `/fab-switch --blank`
- **THEN** the skill SHALL display: "No active change (already blank)."
- **AND** the skill SHALL NOT error

#### Scenario: Combine `--blank` with `--branch` to return to main

- **GIVEN** an active change pointed to by `fab/current`
- **WHEN** the user runs `/fab-switch --blank --branch main`
- **THEN** `fab/current` SHALL be deleted
- **AND** the skill SHALL checkout the `main` branch (via existing `--branch` behavior)
- **AND** the output SHALL show both the deactivation and the branch change

#### Scenario: Combine `--blank` with `--no-branch-change`

- **GIVEN** an active change exists
- **WHEN** the user runs `/fab-switch --blank --no-branch-change`
- **THEN** `fab/current` SHALL be deleted
- **AND** no git operations SHALL be performed (this is the same as `--blank` alone — `--no-branch-change` is redundant but harmless)

#### Scenario: Branch checkout fails when combined with `--branch`

- **GIVEN** an active change exists and the user runs `/fab-switch --blank --branch main`
- **WHEN** `git checkout main` fails (worktree conflict, branch doesn't exist, dirty tree)
- **THEN** `fab/current` SHALL still be deleted (deactivation succeeds regardless of git outcome)
- **AND** the skill SHALL remain on the current branch and report the reason
<!-- clarified: Deactivation always completes — git checkout failure is non-fatal. Covers worktrees, missing branches, and dirty trees in one condition. -->

### Requirement: Delete `fab/current` on deactivation

When deactivating, the skill SHALL **delete** the `fab/current` file entirely, consistent with the pattern used by `/fab-archive`. It SHALL NOT truncate or write an empty string to the file.

This ensures the preflight script (`fab-preflight.sh`) correctly reports "No active change" for all skills that require an active change.

#### Scenario: Preflight after deactivation

- **GIVEN** the user has run `/fab-switch --blank` and `fab/current` has been deleted
- **WHEN** any skill that requires an active change runs the preflight script
- **THEN** the preflight script SHALL exit non-zero with: "No active change. Run /fab-new to start one."

### Requirement: Deactivation output format

The skill SHALL display confirmation in the following format:

When deactivating (no branch change):

```
No active change.

Next: /fab-new <description> or /fab-switch <change-name>
```

When deactivating with branch change (`--blank --branch main`):

```
No active change.
Branch: main (checked out)

Next: /fab-new <description> or /fab-switch <change-name>
```

When deactivating but branch checkout failed:

```
No active change.
Branch: stayed on {current-branch} ({reason})

Next: /fab-new <description> or /fab-switch <change-name>
```

When already deactivated:

```
No active change (already blank).

Next: /fab-new <description> or /fab-switch <change-name>
```

#### Scenario: Output after deactivation

- **GIVEN** an active change exists
- **WHEN** the user runs `/fab-switch --blank`
- **THEN** the output SHALL show "No active change." and the Next suggestion

#### Scenario: Output with branch change

- **GIVEN** an active change exists
- **WHEN** the user runs `/fab-switch --blank --branch main`
- **THEN** the output SHALL show "No active change.", the Branch line, and the Next suggestion

## fab-switch: Integration with existing flows

### Requirement: No Argument Flow unchanged

The existing No Argument Flow (list all changes and ask user to pick) SHALL NOT be modified.

#### Scenario: No argument still lists changes

- **GIVEN** active changes exist
- **WHEN** the user runs `/fab-switch` with no argument
- **THEN** behavior SHALL be identical to the current implementation (list changes, prompt for selection)

### Requirement: Existing change-matching unchanged

The existing Argument Flow (positional change name matching with substring search) SHALL NOT be modified. `--blank` is a flag, not a positional argument, so there is no interaction with change name matching.

#### Scenario: Change matching unaffected

- **GIVEN** a change folder named `260212-xxxx-add-main-nav` exists
- **WHEN** the user runs `/fab-switch main-nav`
- **THEN** the skill SHALL match and switch to the change (existing behavior, unaffected by `--blank`)

### Requirement: Update skill documentation

The fab-switch skill file (`fab/.kit/skills/fab-switch.md`) SHALL be updated to document:
1. `--blank` flag in the Arguments section
2. The deactivation flow in the Behavior section (between Argument Flow and Switch Flow)
3. Deactivation output format in the Output section
4. Error handling for deactivation edge cases
5. Updated Key Properties noting that `fab/current` may now be deleted (not just written)

#### Scenario: Skill file reflects new behavior

- **GIVEN** the change is implemented
- **WHEN** a developer reads `fab/.kit/skills/fab-switch.md`
- **THEN** the `--blank` flag SHALL be fully documented alongside the existing switch behavior

### Requirement: Update centralized docs

`fab/docs/fab-workflow/change-lifecycle.md` SHALL be updated to:

1. Add `/fab-switch --blank` to the `fab/current` lifecycle section as a way to clear the pointer
2. Add `--blank` deactivation to the `/fab-switch` section

#### Scenario: Change lifecycle reflects deactivation

- **GIVEN** the change is archived
- **WHEN** a developer reads `change-lifecycle.md`
- **THEN** the `fab/current` lifecycle SHALL list `/fab-switch --blank` as a way to clear the pointer
- **AND** the `/fab-switch` section SHALL document the `--blank` flag

## Design Decisions

1. **`--blank` flag, not a positional keyword**: Use a flag rather than reserving `main`/`master` as special arguments.
   - *Why*: Separates concerns cleanly — `--blank` handles the fab state (change pointer), `--branch` handles the git state. No keyword precedence issues with change name matching. Composable: `--blank --branch main` is explicit and unambiguous.
   - *Rejected*: `/fab-switch main` as keyword — creates precedence conflict with change name substring matching (a change named `add-main-nav` would be shadowed). Also couples fab deactivation with git branch semantics.
   <!-- clarified: Changed from keyword-based to flag-based design per user feedback — orthogonal concerns should use orthogonal flags -->

2. **No default git checkout on `--blank`**: `--blank` alone only clears `fab/current`. Branch switching requires explicit `--branch`.
   - *Why*: Minimal surprise — `--blank` does one thing (deactivate). Users who want to also switch branches compose it with `--branch main`. This is especially important in worktree setups where checking out main is often impossible.
   - *Rejected*: Auto-checkout main — couples two operations, fails silently in worktrees.
   <!-- clarified: User specified minimal behavior — --blank only clears fab/current, no git operations -->

3. **Delete `fab/current` rather than emptying**: Consistent with `/fab-archive` behavior.
   - *Why*: The preflight script checks `[ ! -f "$current_file" ]` for existence, not just emptiness. Deletion is the established pattern.

## Clarifications

### Session 2026-02-12

- **Q**: The spec references `planning-skills.md` for `/fab-switch` docs, but `/fab-switch` is in `change-lifecycle.md`. Correct Affected Docs?
  **A**: Remove `planning-skills.md`, keep only `change-lifecycle.md`
- **Q**: Should `fab/current` be deleted even if git checkout fails?
  **A**: Yes — deactivation always succeeds. Git checkout failure is non-fatal. Especially relevant in worktrees where main is checked out elsewhere.
- **Q**: Missing scenario for neither main/master branch existing.
  **A**: Consolidated with checkout failure scenario — all "checkout not possible" cases (missing branch, worktree conflict, dirty tree) handled by one condition.
- **Q**: Command syntax — keyword (`main`) or flag (`--blank`)?
  **A**: `--blank` flag. Orthogonal to `--branch` (one is about fab state, other about git). No keyword precedence issues. Composable.
- **Q**: Default git behavior when `--blank` is used alone?
  **A**: No git change. `--blank` only clears `fab/current`. Use `--branch main` explicitly if also switching branches.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Certain | `--blank` flag rather than keyword-based deactivation | User-specified design direction — orthogonal flag for orthogonal concerns |
| 2 | Certain | No default git checkout on `--blank` | User-specified — minimal behavior, compose with `--branch` for git operations |

0 assumptions made (0 confident, 0 tentative). All decisions clarified.
