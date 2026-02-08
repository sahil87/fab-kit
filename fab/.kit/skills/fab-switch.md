---
name: fab-switch
description: "Switch the active change to a different one. Lists available changes when called with no argument. Handles branch integration."
---

# /fab-switch [change-name] [--branch <name>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Switch the active change to a different one. Accepts a full or partial change name, matches it against existing change folders in `fab/changes/`, updates `fab/current` to point to the selected change, and handles git branch integration. When invoked with no argument, lists all available changes and asks the user to pick one.

---

## Arguments

- **`<change-name>`** *(optional)* — full or partial name of the change to switch to. Supports:
  - Full folder name: `260202-m3x1-fix-checkout-bug`
  - Partial slug match: `fix-checkout`
  - Any substring: `checkout`
- **`--branch <name>`** *(optional)* — explicit branch name to use. Creates if new, checks out if existing. Skips the interactive branch prompt.

If no argument is provided, list all active changes and ask the user to pick one (see **No Argument Flow** below).

---

## Context Loading

This skill loads **minimal context plus config**:

1. `fab/config.yaml` — for `git.enabled` and `git.branch_prefix` (needed for branch integration)
2. The contents of `fab/changes/` directory (folder names)
3. The `.status.yaml` of the matched change (to display status after switching)

It does NOT load `fab/constitution.md`, `fab/docs/index.md`, or `fab/specs/index.md`.

---

## Behavior

### No Argument Flow

When invoked as `/fab-switch` with no argument:

1. **Scan `fab/changes/`** for all change folders (exclude the `archive/` subdirectory)
2. **If no change folders exist**, output:

   > `No active changes found. Run /fab-new <description> to start one.`

3. **If change folders exist**, list them with their current stage:

   For each folder in `fab/changes/` (excluding `archive/`):
   - Read `fab/changes/{name}/.status.yaml`
   - Extract the `stage` field
   - Display as a numbered list

   ```
   Active changes:
     1. 260202-m3x1-fix-checkout-bug (stage: apply)
     2. 260205-k9p2-add-oauth (stage: specs)
     3. 260206-r4t7-refactor-auth (stage: proposal)

   Which change? (1-3)
   ```

4. **Wait for user selection** and proceed to the Switch Flow with the selected change name

### Argument Flow

When invoked as `/fab-switch <change-name>`:

1. **Scan `fab/changes/`** for all change folders (exclude the `archive/` subdirectory)
2. **Match the argument** against the folder names:
   - **Exact match**: The argument matches a folder name exactly → use that folder
   - **Single partial match**: The argument is a substring of exactly one folder name → use that folder
   - **Multiple partial matches**: The argument matches more than one folder → list all matches and ask the user to pick (see Ambiguous Match below)
   - **No match**: The argument does not match any folder → list all available changes and inform the user (see No Match below)

### Match Rules

Matching is **case-insensitive** and checks if the argument is a **substring** of the folder name:

- `fix-checkout` matches `260202-m3x1-fix-checkout-bug` ✓
- `checkout` matches `260202-m3x1-fix-checkout-bug` ✓
- `260202` matches `260202-m3x1-fix-checkout-bug` ✓
- `m3x1` matches `260202-m3x1-fix-checkout-bug` ✓

### Ambiguous Match

When the argument matches multiple change folders:

```
Multiple changes match "add":
  1. 260205-k9p2-add-oauth (stage: specs)
  2. 260206-a1b2-add-spinner (stage: proposal)

Which change? (1-2)
```

Wait for user selection and proceed to the Switch Flow.

### No Match

When the argument does not match any change folder:

```
No change matches "xyz".

Active changes:
  1. 260202-m3x1-fix-checkout-bug (stage: apply)
  2. 260205-k9p2-add-oauth (stage: specs)

Run /fab-switch <name> with a matching name, or pick from the list above (1-2).
```

Wait for user selection or let the user re-invoke with a different argument.

### Switch Flow

Once a single change is identified:

1. **Write the change name** to `fab/current`:
   - Write just the folder name (not the full path) to `fab/current`
   - This overwrites any previous content in `fab/current`

2. **Branch Integration** (see Branch Integration section below)

3. **Read the change's `.status.yaml`** to get:
   - `stage` — current stage
   - `progress` — stage progress map

4. **Display confirmation** with status summary:

   ```
   fab/current now points to 260202-m3x1-fix-checkout-bug

   Stage:  apply (5/7)
   Branch: fix/checkout-bug (created)
   ```

   If branch integration was skipped:

   ```
   fab/current now points to 260202-m3x1-fix-checkout-bug

   Stage:  apply (5/7)
   ```

5. **Suggest next command** based on the change's current stage:

   | Stage | Suggested next |
   |-------|---------------|
   | `proposal` (active) | `Next: /fab-continue or /fab-ff` |
   | `proposal` (done) | `Next: /fab-continue or /fab-ff` |
   | `specs` (done) | `Next: /fab-continue (plan) or /fab-ff or /fab-clarify` |
   | `plan` (done) | `Next: /fab-continue (tasks) or /fab-clarify` |
   | `tasks` (done) | `Next: /fab-apply` |
   | `apply` (done) | `Next: /fab-review` |
   | `review` (done) | `Next: /fab-archive` |
   | `review` (failed) | `Next: /fab-review (re-review after fixes)` |

---

## Branch Integration

**Skip this step entirely if**:
- `git.enabled` is `false` in `fab/config.yaml`, OR
- The working directory is not inside a git repository

**If `--branch <name>` was provided**:
1. Use the provided name directly
2. If a git branch with that name already exists, check it out (if not already on it)
3. If it doesn't exist, create it: `git checkout -b <name>`
4. Skip the interactive prompt below

**If no `--branch` argument** (interactive flow):

Detect the current branch and offer options:

- **If on `main` or `master`**: Auto-create a branch without prompting
  - Branch name: `{branch_prefix}{change-name}` (using `git.branch_prefix` from config, which may be empty)
  - Run `git checkout -b {branch-name}`
  - This is a low-value prompt removed per SRAD scoring (high R, high A, high D — Certain grade)

- **If on a `wt/*` branch** (worktree base branch): **Present these options to the user (do NOT auto-select)**:
  - Show current branch name
  - Note: `wt/*` branches are worktree base branches — default to **Create new branch**
  - Options: **Create new branch** (default), **Adopt this branch**, **Skip**
  - If user chooses "Create new branch": create a new branch as above
  - If user chooses "Adopt": no git operation needed (already on the branch)
  - If user chooses "Skip": no branch change

- **If on a feature branch** (not main/master, not wt/*): **Present these options to the user (do NOT auto-select)**:
  - Show current branch name
  - Options: **Adopt this branch** (default), **Create new branch**, **Skip**
  - If user chooses "Adopt": no git operation needed (already on the branch)
  - If user chooses "Create new branch": create a new branch as above
  - If user chooses "Skip": no branch change

---

## Stage Number Mapping

Map stages to their numeric position for the `(N/7)` display:

| Stage | Number |
|-------|--------|
| `proposal` | 1 |
| `specs` | 2 |
| `plan` | 3 |
| `tasks` | 4 |
| `apply` | 5 |
| `review` | 6 |
| `archive` | 7 |

---

## Output

### Successful Switch (with branch)

```
fab/current now points to 260202-m3x1-fix-checkout-bug

Stage:  apply (5/7)
Branch: 260202-m3x1-fix-checkout-bug (created)

Next: /fab-review
```

### Successful Switch (branch adopted)

```
fab/current now points to 260202-m3x1-fix-checkout-bug

Stage:  apply (5/7)
Branch: feature/checkout-fix (adopted)

Next: /fab-review
```

### Successful Switch (no branch)

```
fab/current now points to 260202-m3x1-fix-checkout-bug

Stage:  apply (5/7)

Next: /fab-review
```

### No Changes Exist

```
No active changes found. Run /fab-new <description> to start one.
```

### Ambiguous Match

```
Multiple changes match "add":
  1. 260205-k9p2-add-oauth (stage: specs)
  2. 260206-a1b2-add-spinner (stage: proposal)

Which change? (1-2)
```

### No Match

```
No change matches "xyz".

Active changes:
  1. 260202-m3x1-fix-checkout-bug (stage: apply)
  2. 260205-k9p2-add-oauth (stage: specs)

Run /fab-switch <name> with a matching name, or pick from the list above (1-2).
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| No argument and no change folders | Output: "No active changes found. Run /fab-new \<description\> to start one." |
| No argument and changes exist | List all changes with stages, ask user to pick |
| Argument matches exactly one change | Switch to that change |
| Argument matches multiple changes | List matches with stages, ask user to pick |
| Argument matches no changes | List all available changes, inform user |
| Matched change folder missing `.status.yaml` | Switch anyway but warn: "Warning: .status.yaml not found for {name} — change may be corrupted." |
| `fab/changes/` directory does not exist | Output: "fab/changes/ not found. Run /fab-init to set up the project." |
| `fab/current` cannot be written | Report error with details |
| `fab/config.yaml` not found | Skip branch integration (git settings unknown) |
| Git branch creation fails | Report the error, continue without branch change. The switch itself still completes (fab/current is written, status is displayed). |
| `--branch` name invalid for git | Report the error, continue without branch change |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — switch only changes the active pointer, does not modify any change's stage |
| Idempotent? | **Yes** — switching to the same change multiple times has no side effects (branch integration is skipped if already on the target branch) |
| Modifies `fab/current`? | **Yes** — writes the selected change name |
| Modifies `.status.yaml`? | **No** — read-only access to display status |
| Modifies source code? | **No** |
| Modifies git state? | **Yes** — may create or check out a branch |
| Requires config/constitution? | **Config only** — reads `git.enabled` and `git.branch_prefix` from `fab/config.yaml` |

---

## Next Steps Reference

After `/fab-switch` completes, the Next line is contextual based on the switched-to change's current stage (see the Suggested next table above).
