---
name: fab-archive
description: "Archive a completed change or restore an archived change — move to/from archive folder, update index, mark backlog items, clear pointer."
---

# /fab-archive [<change-name>] | restore <change-name> [--switch]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

## Contents

- Purpose
- Arguments
- Pre-flight
- Context Loading
- Behavior
- Output
- Key Properties
- Restore Mode

---

## Purpose

Archive a completed change after hydrate, or restore an archived change back to active. Archive mode delegates all mechanical operations (move, index, backlog, pointer) to `fab change archive`; the skill only formats the YAML output into a user-facing report. Restore mode delegates entirely to `fab change restore`. Both modes are safe to re-run after interruption.

> **Dirty-tree disclosure**: "safe to re-run" covers fab state, not git state — both modes move tracked files and edit `fab/backlog.md` / `fab/changes/archive/index.md` with **no commit step**. Every archive or restore leaves uncommitted moves and backlog/index edits in the working tree for the caller to commit (e.g., via `/git-pr`). The skill deliberately does NOT commit autonomously — commit ownership stays with `/git-pr`.

---

## Arguments

### Archive Mode (default)

- **`<change-name>`** *(optional)* — target a specific change. Resolution per `_preamble.md` (Change-name override).

### Restore Mode

- **`restore`** — switches to restore mode.
- **`<change-name>`** *(required)* — name or substring of the archived change to restore.
- **`--switch`** *(optional)* — activate the restored change by creating the `.fab-status.yaml` symlink.

**Mode detection**: If the first positional argument is `restore`, use restore mode. Otherwise, use archive mode.

The sections below (Pre-flight through Key Properties) define **archive mode** — the default. Restore-specific behavior follows in **§ Restore Mode**.

---

## Pre-flight

1. Run `fab preflight [change-name]` per `_preamble.md`
2. **Hydrate Guard**: If `progress.hydrate` is not `done`, STOP: `Hydrate has not completed. Run /fab-continue to hydrate memory first.`

---

## Context Loading

None beyond preflight — `fab change archive` reads `intake.md` and `fab/backlog.md` itself.

---

## Behavior

### Step 1: Run Archive Command

Call `fab change archive` in a single invocation:

```bash
fab change archive <change>
```

Where `<change>` is the change ID or name from preflight. Pass no `--description` — the command derives it mechanically from the intake title (humanized-slug fallback). Parse the structured YAML output for the report.

The command handles everything mechanically:
- **Move**: `fab/changes/{name}/` → `fab/changes/archive/yyyy/mm/{name}/` (date-bucketed)
- **Dispatch state**: Delete `.fab-dispatch/{id}/` (the change's headless-dispatch state dir) — dispatch artifacts are transient comms, not history, so they are removed on archive (one of the two `fab dispatch` cleanup paths) and **not recreated on restore**. Best-effort: an absent dir is a no-op.
- **Index**: Create/update `fab/changes/archive/index.md` with entry + backfill
- **Backlog**: Mark the originating backlog item done (`- [ ]` → `- [x]`) by exact change-ID match, in place
- **Pointer**: Remove `.fab-status.yaml` symlink if this was the active change

If the command prints `already archived: ...` and exits 0, the change was already archived — report it as a soft skip.

### Step 2: Format Report

Construct the user-facing report from the command's YAML output fields:

| YAML field | Report line |
|------------|-------------|
| `move: moved` | `Moved:    ✓ fab/changes/archive/yyyy/mm/{name}/` |
| `index: created` | `Index:    ✓ fab/changes/archive/index.md created` |
| `index: updated` | `Index:    ✓ fab/changes/archive/index.md updated` |
| `index: failed` | `Index:    ✗ index update failed — see stderr` |
| `backlog: marked` | `Backlog:  ✓ [ID] marked done` |
| `backlog: already` | `Backlog:  — already done` |
| `backlog: not_found` | `Backlog:  — no match` |
| `pointer: cleared` | `Pointer:  ✓ .fab-status.yaml removed` |
| `pointer: skipped` | `Pointer:  — skipped, not active` |

All report lines are sourced from the command's YAML output — the skill performs no file edits and asks no interactive questions.

---

## Output

```
Archive: {change name}

Moved:    ✓ fab/changes/archive/yyyy/mm/{name}/
Index:    ✓ fab/changes/archive/index.md updated (or: created / ✗ index update failed — see stderr)
Backlog:  ✓ [ID] marked done                   (or: — already done / — no match)
Pointer:  ✓ .fab-status.yaml removed             (or: — skipped, not active)

Archive complete.

Next: {per state table — initialized}
```

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No — post-pipeline housekeeping |
| Idempotent? | Yes — re-archive is a soft skip that still re-attempts the backlog mark (recovering a previously-failed mark); `fab change archive` marks the backlog idempotently (`already`) |
| Leaves uncommitted changes? | Yes — moved files + backlog/index edits (see Dirty-tree disclosure in § Purpose) |
| Modifies `.status.yaml`? | No (may update `last_updated`) |
| Modifies `.fab-status.yaml`? | Yes — conditionally removes symlink (via command) |
| Modifies `docs/memory/`? | No |
| Uses `Edit`? | No — the skill only formats the command's YAML output |
| Requires hydrate done? | Yes |

---

## Restore Mode

Restore an archived change back to the active changes folder — the inverse of the archive operation. Delegates entirely to `fab change restore`. Preserves all artifacts and status as-is — no status reset, no artifact regeneration. The `.fab-dispatch/{id}/` state deleted at archive time is **not recreated** (dispatch artifacts are transient comms, not history) — re-dispatch a stage via `fab dispatch start` if needed. Arguments and mode detection are defined once in **§ Arguments** above (`<change-name>` is resolved by `fab change restore` via case-insensitive substring matching against `fab/changes/archive/`).

### Pre-flight (mode-specific — opposite of archive mode)

1. No standard preflight runs (no active change required), and the **hydrate guard is waived** — restore applies to any archived change regardless of state.
2. No context loading is required — `fab change restore` handles archive folder validation and all file operations internally.

### Behavior

#### Step 1: Resolve and Restore

Call `fab change restore` in a single invocation:

```bash
fab change restore <change-name> [--switch]
```

Parse the structured YAML output for the report. On non-zero exit, handle per § Error Handling (multiple-match, no-match, etc.).

#### Step 2: Format Report

Construct the user-facing report from the command's YAML output fields:

| YAML field | Report line |
|------------|-------------|
| `move: restored` | `Moved:    ✓ fab/changes/{name}/` |
| `move: already_in_changes` | `Moved:    ✓ already in changes` |
| `index: removed` | `Index:    ✓ entry removed from archive/index.md` |
| `index: not_found` | `Index:    — entry not found` |
| `index: failed` | `Index:    ✗ entry removal failed — see stderr` |
| `pointer: switched` | `Pointer:  ✓ .fab-status.yaml updated` |
| `pointer: skipped` | `Pointer:  — not requested` |
| `pointer: failed` | `Pointer:  ✗ activation failed — run /fab-switch {name} manually` |

### Output

```
Restore: {change name}

Moved:    ✓ fab/changes/{name}/                  (or: ✓ already in changes)
Index:    ✓ entry removed from archive/index.md  (or: — entry not found / ✗ entry removal failed — see stderr)
Pointer:  ✓ .fab-status.yaml updated              (or: — not requested / ✗ activation failed — run /fab-switch {name} manually)

Restore complete.

Next: {per state table — if --switch: restored change's state; otherwise: activation preamble + restored change's state}
```

### Error Handling

| Condition | Action |
|-----------|--------|
| Script exits 1: "No archive folder found" | Display error |
| Script exits 1: "No archived changes found" | Display error |
| Script exits 1: "No archive matches" | List available archives via `fab change archive-list` |
| Script exits 1: "Multiple archives match" | Parse matches from stderr, ask user to pick |
| Folder already in `fab/changes/` | Script handles — reports `already_in_changes` |
| Index entry not found | Script handles — reports `not_found` |

### Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No — post-archive housekeeping |
| Idempotent? | Yes — the command detects already-restored folders |
| Leaves uncommitted changes? | Yes — moved files + archive-index edits (see Dirty-tree disclosure in § Purpose) |
| Modifies `.status.yaml`? | No |
| Modifies `.fab-status.yaml`? | Only with `--switch` flag (via the command) |
| Modifies `docs/memory/`? | No |
| Requires hydrate done? | No — restores any archived change regardless of state |
