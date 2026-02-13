---
name: fab-archive
description: "Archive a completed change — move to archive folder, update index, mark backlog items, clear pointer."
---

# /fab-archive [<change-name>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Archive a completed change after hydrate. This is a standalone housekeeping command — not a pipeline stage. It moves the change folder to archive, updates the archive index, marks related backlog items done, and clears the active pointer (if the archived change was active).

Safe to re-run after interruption — detects already-moved folders and completes remaining steps.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of the active one in `fab/current`. Supports full folder names, partial slug matches, or 4-char IDs (e.g., `r3m7`). When provided, passed to the preflight script as `$1` for transient resolution — `fab/current` is **not** modified.

If no argument is provided, the skill operates on the active change in `fab/current`.

---

## Pre-flight Check

Before doing anything else, run the preflight script:

1. Execute `fab/.kit/scripts/fab-preflight.sh [change-name]` via Bash — pass the change-name argument if one was provided
2. If the script exits non-zero, **STOP** and surface the stderr message to the user
3. Parse the stdout YAML to get `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence`

### Hydrate Guard

Check `progress.hydrate` from the preflight output.

**If `progress.hydrate` is not `done`, STOP.** Output:

> `Hydrate has not completed. Run /fab-continue to hydrate docs first.`

---

## Context Loading

This skill loads minimal context:

1. `fab/changes/{name}/brief.md` — for backlog ID extraction and keyword scanning
2. `fab/changes/{name}/.status.yaml` — for progress verification
3. `fab/backlog.md` — for backlog item matching (if exists)
4. `fab/current` — to determine if the archived change is the active one

---

## Behavior

### Resumability

On invocation, check whether the change folder has already been moved to `fab/changes/archive/{name}/`:

- **If folder is at `fab/changes/{name}/`**: normal flow — execute all steps
- **If folder is already at `fab/changes/archive/{name}/`**: resume — skip folder move, complete remaining steps (index, backlog, pointer)

This makes `/fab-archive` resumable after interruption.

### Step 1: Move Change Folder

Move `fab/changes/{name}/` → `fab/changes/archive/{name}/`. Create `archive/` directory if it doesn't exist. Do NOT rename the folder.

### Step 2: Update Archive Index

Maintain `fab/changes/archive/index.md`:

- **If file doesn't exist**: Create it with a header and backfill entries for all existing archived change folders
- **If file exists**: Prepend a new entry (most-recent-first)

Entry format: `- **{folder-name}** — {1-2 sentence description from brief Why section}`

### Step 3: Mark Backlog Items Done

#### Step 3a: Exact-ID Check

If the brief contains a backlog ID (in the Origin section), find the matching item in `fab/backlog.md` and mark it done:
- Change checkbox from `- [ ]` to `- [x]`
- Move from Backlog section to Done section (prepend, most-recent-first)

If `fab/backlog.md` does not exist, skip silently.

#### Step 3b: Keyword Scan

After the exact-ID check, perform a secondary keyword scan to surface related backlog items:

1. **Extract keywords**: Parse the brief's title (`# Brief: {title}` heading) and Why section content. Filter out common stop words (articles, prepositions, conjunctions, common verbs: "is", "are", "has", "should", "must", "the", "a", "an", "in", "of", "to", "for", "and", "or", "with", "from", "by", "on", "at", "this", "that", "it", "be", "not", "no"). Normalize remaining words to lowercase.
2. **Match candidates**: Compare extracted keywords against each unchecked (`- [ ]`) backlog item's description text (case-insensitive). A backlog item is a candidate when at least **2 significant keywords** overlap with the item's description. Exclude any item already marked done by the exact-ID check in Step 3a.
3. **No candidates**: If no items meet the 2-keyword threshold, proceed silently (no output for this sub-step).

#### Step 3c: Interactive Confirmation

If Step 3b produced candidate matches, present them to the user:

```
Backlog matches found:
  1. [ID] {description (truncated to ~80 chars)}
  2. [ID] {description (truncated to ~80 chars)}

Mark as done? (comma-separated numbers, or "none")
```

- **User responds with numbers** (e.g., "1" or "1,3"): Mark the selected items done — change checkbox from `- [ ]` to `- [x]`, move from Backlog section to Done section (prepend, most-recent-first)
- **User responds with "none"**: No items are marked done, archive proceeds normally

### Step 4: Clear Pointer (Conditional)

Read `fab/current` and check if it points to the change being archived:

- **If `fab/current` contains the archived change name**: Delete `fab/current`
- **If `fab/current` points to a different change or doesn't exist**: Do nothing (preserve the user's active context)

### Order of Operations (Fail-Safe)

Steps 1-4 execute in this order for safety: folder move first (primary action, makes state visible), index second (metadata), backlog third (cross-references), pointer last (cleanup). If interrupted mid-operation, re-running completes the remaining steps.

---

## Output

### Successful Archive (active change)

```
Archive: {change name}

Moved:    ✓ fab/changes/archive/{name}/
Index:    ✓ fab/changes/archive/index.md updated
Backlog:  ✓ [ID] marked done
Scan:     ✓ {N} candidates found, {M} marked done
Pointer:  ✓ fab/current cleared

Archive complete.

Next: /fab-new <description> (start next change)
```

### Successful Archive (non-active change)

```
Archive: {change name}

Moved:    ✓ fab/changes/archive/{name}/
Index:    ✓ fab/changes/archive/index.md updated
Backlog:  ✓ [ID] marked done
Scan:     ✓ no matches
Pointer:  — skipped (not the active change)

Archive complete.

Next: /fab-new <description> (start next change)
```

### Successful Archive (no backlog)

```
Archive: {change name}

Moved:    ✓ fab/changes/archive/{name}/
Index:    ✓ fab/changes/archive/index.md updated
Backlog:  — no backlog file
Pointer:  ✓ fab/current cleared

Archive complete.

Next: /fab-new <description> (start next change)
```

### Resume After Interruption

```
Archive: {change name}

Moved:    ✓ already in archive
Index:    ✓ fab/changes/archive/index.md updated
Backlog:  ✓ [ID] marked done
Pointer:  ✓ fab/current cleared

Archive complete.

Next: /fab-new <description> (start next change)
```

### Guard Failure

```
Hydrate has not completed. Run /fab-continue to hydrate docs first.
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight script exits non-zero | Abort with the stderr message from `fab-preflight.sh` |
| `progress.hydrate` is not `done` | Abort with: "Hydrate has not completed. Run /fab-continue to hydrate docs first." |
| Change folder already in `fab/changes/archive/` | Resume — skip move, complete remaining steps |
| `fab/changes/archive/` doesn't exist | Create it before moving |
| `fab/changes/archive/index.md` doesn't exist | Create with header and backfill |
| `fab/backlog.md` doesn't exist | Skip backlog steps silently |
| Keyword scan finds no matches | Proceed silently |
| User declines all keyword scan candidates | Proceed normally |
| `fab/current` points to a different change | Skip pointer clearing |
| `fab/current` doesn't exist | Skip pointer clearing |

---

## Key Properties

| Property | Value |
|----------|-------|
| Pipeline stage? | **No** — standalone housekeeping command |
| Advances stage? | **No** — does not modify progress map |
| Idempotent? | **Yes** — safe to re-invoke; detects already-moved folders |
| Modifies `.status.yaml`? | **No** — may update `last_updated` only |
| Moves change folder? | **Yes** — to `fab/changes/archive/{name}/` |
| Updates archive index? | **Yes** — `fab/changes/archive/index.md` |
| Clears `fab/current`? | **Conditional** — only if the archived change was active |
| Modifies `fab/backlog.md`? | **Yes** — marks matched items done |

---

## Next Steps Reference

After `/fab-archive` completes:

`Next: /fab-new <description> (start next change)`
