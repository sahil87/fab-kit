---
name: fab-status
description: "Show current change state at a glance — name, branch, stage, checklist status, and suggested next command."
---

# /fab:status

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Show the current change state at a glance — change name, branch, stage progress, checklist status, kit version, and suggested next command. Provides a quick orientation for where you are in the workflow without modifying anything.

---

## Context Loading

This skill uses **minimal context** — it does not need to load `fab/config.yaml` or `fab/constitution.md` (as noted in `_context.md`, status is exempt from the "Always Load" requirement).

The only context needed is:
1. `fab/current` — to identify the active change
2. `fab/changes/{name}/.status.yaml` — to read stage, progress, branch, and checklist info
3. `fab/.kit/VERSION` — to display the kit version

---

## Behavior

### Step 1: Check for Active Change

1. Read `fab/.kit/VERSION` to get the kit version string
2. Check if `fab/current` exists

**If `fab/current` does not exist, STOP.** Output:

```
Fab Kit v{version}

No active change. Run /fab:new to start one.
```

### Step 2: Load Change Status

1. Read the change name from `fab/current`
2. Read `fab/changes/{name}/.status.yaml`

**If the change directory or `.status.yaml` is missing, STOP.** Output:

```
Fab Kit v{version}

Active change: {name}
⚠ .status.yaml not found — change may be corrupted.

Run /fab:new to start a fresh change or /fab:switch to select another.
```

### Step 3: Parse Status

From `.status.yaml`, extract:
- `name` — change name
- `branch` — git branch (may not exist)
- `stage` — current stage keyword
- `progress` — map of stage → status (`pending`, `active`, `done`, `skipped`, `failed`)
- `checklist.completed` — number of completed checklist items
- `checklist.total` — total number of checklist items
- `checklist.generated` — whether the checklist has been generated

### Step 4: Render Progress Table

Build a progress table using these symbols:

| Symbol | Meaning | When to use |
|--------|---------|-------------|
| `✓` | Done | `progress.{stage}` is `done` |
| `●` | Active | `progress.{stage}` is `active` |
| `○` | Pending | `progress.{stage}` is `pending` |
| `—` | Skipped | `progress.{stage}` is `skipped` |
| `✗` | Failed | `progress.{stage}` is `failed` |

The stages in order are: `proposal`, `specs`, `plan`, `tasks`, `apply`, `review`, `archive`.

Map the current `stage` field to a stage number (1-7):

| Stage | Number |
|-------|--------|
| `proposal` | 1 |
| `specs` | 2 |
| `plan` | 3 |
| `tasks` | 4 |
| `apply` | 5 |
| `review` | 6 |
| `archive` | 7 |

### Step 5: Determine Next Command

Based on the current stage and its status, suggest the next command:

| Stage | Progress | Suggested next |
|-------|----------|---------------|
| `proposal` | `active` | `/fab:continue or /fab:ff` |
| `proposal` | `done` | `/fab:continue or /fab:ff` |
| `specs` | `active` | `/fab:continue` |
| `specs` | `done` | `/fab:continue (plan) or /fab:ff or /fab:clarify` |
| `plan` | `active` | `/fab:continue` |
| `plan` | `done` | `/fab:continue (tasks) or /fab:clarify` |
| `plan` | `skipped` | `/fab:continue (tasks)` |
| `tasks` | `active` | `/fab:continue` |
| `tasks` | `done` | `/fab:apply` |
| `apply` | `active` | `/fab:apply` |
| `apply` | `done` | `/fab:review` |
| `review` | `active` | `/fab:review` |
| `review` | `done` | `/fab:archive` |
| `review` | `failed` | `/fab:review (re-review after fixes)` |
| `archive` | `done` | `/fab:new <description>` |

---

## Output

### Active Change — Full Status

```
Fab Kit v{version}

Change:  {name}
Branch:  {branch}
Stage:   {stage} ({N}/7)

Progress:
  ✓ proposal
  ✓ specs
  ● plan
  ○ tasks
  ○ apply
  ○ review
  ○ archive

Checklist: {completed}/{total} items
           (or "not yet generated" if checklist.generated is false)

Next: {suggested command}
```

### Active Change — No Branch

```
Fab Kit v{version}

Change:  {name}
Branch:  (none)
Stage:   {stage} ({N}/7)

Progress:
  ✓ proposal
  ✓ specs
  ● plan
  ○ tasks
  ○ apply
  ○ review
  ○ archive

Checklist: not yet generated

Next: /fab:continue (tasks) or /fab:clarify
```

### Active Change — With Skipped Plan

```
Fab Kit v{version}

Change:  {name}
Branch:  feature/add-spinner
Stage:   tasks (4/7)

Progress:
  ✓ proposal
  ✓ specs
  — plan (skipped)
  ● tasks
  ○ apply
  ○ review
  ○ archive

Checklist: 0/15 items

Next: /fab:apply
```

### Active Change — Review Failed

```
Fab Kit v{version}

Change:  {name}
Branch:  feature/fix-checkout
Stage:   review (6/7)

Progress:
  ✓ proposal
  ✓ specs
  ✓ plan
  ✓ tasks
  ✓ apply
  ✗ review
  ○ archive

Checklist: 10/12 items

Next: /fab:review (re-review after fixes)
```

### No Active Change

```
Fab Kit v{version}

No active change. Run /fab:new to start one.
```

### Corrupted Change

```
Fab Kit v{version}

Active change: {name}
⚠ .status.yaml not found — change may be corrupted.

Run /fab:new to start a fresh change or /fab:switch to select another.
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/current` missing | Output "No active change" message with kit version |
| `fab/.kit/VERSION` missing | Use "unknown" as version string |
| `.status.yaml` missing | Output corrupted change warning |
| `.status.yaml` malformed | Output what can be parsed, warn about unreadable fields |
| `progress` map incomplete | Show `○` (pending) for missing stages |
| `checklist` section missing | Show "not yet generated" |
| `branch` field missing | Show "(none)" |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — purely informational, read-only |
| Idempotent? | **Yes** — no side effects, safe to call any number of times |
| Modifies `fab/current`? | **No** |
| Modifies `.status.yaml`? | **No** |
| Modifies source code? | **No** |
| Requires config/constitution? | **No** — operates only on `fab/current`, `.status.yaml`, and `VERSION` |

---

## Next Steps Reference

After `/fab:status`, the Next line is contextual based on the active change's current stage (see the Suggested next table above). If there is no active change, suggest `/fab:new`.
