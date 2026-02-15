---
name: fab-archive
description: "Archive a completed change or restore an archived change — move to/from archive folder, update index, mark backlog items, clear pointer."
---

# /fab-archive [<change-name>] | restore <change-name> [--switch]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Archive a completed change after hydrate, or restore an archived change back to active. Archive mode moves to archive, updates index, marks backlog items done, clears active pointer. Restore mode moves from archive back to changes, removes the index entry, and optionally activates. Both modes are safe to re-run after interruption.

---

## Arguments

### Archive Mode (default)

- **`<change-name>`** *(optional)* — target a specific change. Resolution per `_context.md` (Change-name override).

### Restore Mode

- **`restore`** — switches to restore mode.
- **`<change-name>`** *(required)* — name or substring of the archived change to restore.
- **`--switch`** *(optional)* — activate the restored change by writing its name to `fab/current`.

**Mode detection**: If the first positional argument is `restore`, use restore mode. Otherwise, use archive mode.

---

## Pre-flight

1. Run `fab/.kit/scripts/lib/preflight.sh [change-name]` per `_context.md`
2. **Hydrate Guard**: If `progress.hydrate` is not `done`, STOP: `Hydrate has not completed. Run /fab-continue to hydrate memory first.`

---

## Context Loading

Minimal: `intake.md` (for backlog ID + keywords), `.status.yaml`, `fab/backlog.md` (if exists), `fab/current`.

---

## Behavior

### Resumability

If folder already at `fab/changes/archive/{name}/`, skip move and complete remaining steps.

### Step 1: Move Change Folder

`fab/changes/{name}/` → `fab/changes/archive/{name}/`. Create `archive/` if needed. Do NOT rename.

### Step 2: Update Archive Index

Maintain `fab/changes/archive/index.md`:
- Doesn't exist → create with header, backfill all existing archived folders
- Exists → prepend entry (most-recent-first)

Entry format: `- **{folder-name}** — {1-2 sentence description from intake Why}`

### Step 3: Mark Backlog Items Done

Skip silently if `fab/backlog.md` doesn't exist.

**3a — Exact-ID**: If intake has backlog ID, find and mark done (`- [ ]` → `- [x]`), move to Done section.

**3b — Keyword Scan**: Extract keywords from intake title/Why (filter stop words). Match against unchecked items — candidate when ≥2 significant keywords overlap (exclude 3a matches). No candidates → proceed silently.

**3c — Interactive Confirmation** (if 3b found candidates):

```
Backlog matches found:
  1. [ID] {description (~80 chars)}
  2. [ID] {description (~80 chars)}

Mark as done? (comma-separated numbers, or "none")
```

### Step 4: Clear Pointer (Conditional)

If `fab/current` contains the archived change → delete `fab/current`. Otherwise no-op.

Steps execute 1→4 for safety. If interrupted, re-run completes remaining.

---

## Output

```
Archive: {change name}

Moved:    ✓ fab/changes/archive/{name}/        (or: ✓ already in archive)
Index:    ✓ fab/changes/archive/index.md updated
Backlog:  ✓ [ID] marked done                   (or: — no backlog file)
Scan:     ✓ {N} candidates, {M} marked done    (or: ✓ no matches)
Pointer:  ✓ fab/current cleared                 (or: — skipped, not active)

Archive complete.

Next: /fab-new <description>
```

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No — post-pipeline housekeeping |
| Idempotent? | Yes — detects already-moved folders |
| Modifies `.status.yaml`? | No (may update `last_updated`) |
| Modifies `fab/current`? | Yes — conditionally clears |
| Modifies `docs/memory/`? | No |
| Requires hydrate done? | Yes |

---

# /fab-archive restore <change-name> [--switch]

## Purpose

Restore an archived change back to the active changes folder. Inverse of the archive operation. Preserves all artifacts and status as-is — no status reset, no artifact regeneration.

---

## Arguments

- **`<change-name>`** *(required)* — name or substring of the archived change to restore. Resolved via case-insensitive substring matching against folder names in `fab/changes/archive/` (excluding `index.md`).
- **`--switch`** *(optional)* — activate the restored change by writing its name to `fab/current`.

---

## Pre-flight

1. Verify `fab/changes/archive/` exists. If not, STOP: `No archive folder found.`
2. Scan `fab/changes/archive/` for folders (excluding `index.md`).
3. If no folders found, STOP: `No archived changes found.`
4. Match `<change-name>` against archived folder names (case-insensitive substring):
   - **Exact/single match** → use that folder
   - **Multiple matches** → list matches, ask user to pick
   - **No match** → list all archived changes, inform user no match was found

---

## Context Loading

Minimal: the matched archive folder's `.status.yaml` (for output), `fab/changes/archive/index.md`.

---

## Behavior

### Resumability

If the folder already exists at `fab/changes/{name}/` (not in archive), skip the move and complete remaining steps (index cleanup, optional pointer update).

### Step 1: Move Change Folder

`fab/changes/archive/{name}/` → `fab/changes/{name}/`. Do NOT rename. All artifacts (`.status.yaml`, `intake.md`, `spec.md`, `tasks.md`, `checklist.md`, etc.) are preserved without modification.

### Step 2: Remove Archive Index Entry

Remove the entry for `{name}` from `fab/changes/archive/index.md`. If the index becomes empty (header only), preserve the file — do not delete it. If the entry is not found, skip silently.

### Step 3: Update Pointer (Conditional)

If `--switch` flag is provided → write `{name}` to `fab/current`. Otherwise no-op — `fab/current` is not modified.

Steps execute 1→3 for safety. If interrupted, re-run completes remaining.

---

## Output

```
Restore: {change name}

Moved:    ✓ fab/changes/{name}/                  (or: ✓ already in changes)
Index:    ✓ entry removed from archive/index.md  (or: — entry not found)
Pointer:  ✓ fab/current updated                  (or: — not requested)

Restore complete.

Next: /fab-switch {name}
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/changes/archive/` doesn't exist | `No archive folder found.` |
| No archived folders | `No archived changes found.` |
| No match for `<change-name>` | List all archived changes, inform user |
| Multiple matches | List matches, ask user to pick |
| Folder already in `fab/changes/` | Skip move, complete remaining steps |
| Index entry not found | Skip index removal silently |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No — post-archive housekeeping |
| Idempotent? | Yes — detects already-restored folders |
| Modifies `.status.yaml`? | No |
| Modifies `fab/current`? | Only with `--switch` flag |
| Modifies `docs/memory/`? | No |
| Requires hydrate done? | No — restores any archived change regardless of state |
