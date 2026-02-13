---
name: fab-reorg-design
description: "Analyze design docs for themes and suggest reorganization. Read-only unless user approves changes."
---

# /fab-reorg-design

---

## Purpose

Read all design documents in `fab/specs/`, identify the main themes (up to 10), and propose a reorganization plan. Read-only by default — files are only moved or rewritten with explicit user approval.

---

## Pre-flight Check

Before doing anything else:

1. Check that `fab/specs/index.md` exists and is readable
2. Check that `fab/specs/` contains at least one `.md` file besides `index.md`

**If either check fails, STOP immediately.** Output:

> `fab/specs/index.md not found. Run /fab-init first.`

or

> `No design docs found in fab/specs/ besides index.md. Nothing to reorganize.`

---

## Context Loading

This skill loads:

1. `fab/specs/index.md` — to understand the current spec landscape
2. **Every `.md` file in `fab/specs/`** — read each file fully to build a complete picture of content

This skill does **not** require `fab/current`, `fab/config.yaml`, or `fab/constitution.md`.

---

## Behavior

### Step 1: Read All Design Docs

1. Read `fab/specs/index.md` to get the current file list and descriptions
2. Read every `.md` file in `fab/specs/` (excluding `index.md`)
3. For each file, extract:
   - File name and path
   - All `## ` and `### ` level headings
   - A brief summary of each section's content (what it covers, what decisions it captures)
   - Approximate line count

### Step 2: Identify Themes (up to 10)

Analyze the collected content across all files and identify the main themes — recurring topics, conceptual clusters, or cross-cutting concerns. A theme is a coherent grouping of related content that could logically live together.

For each theme, note:
- **Theme name** — a concise label (2-4 words)
- **Description** — one sentence explaining what falls under this theme
- **Source locations** — which files and sections contribute to this theme
- **Cohesion** — whether the theme's content is currently concentrated in one file or scattered across many

Present the themes as a numbered table:

```
## Themes Found

| # | Theme | Description | Current Location(s) | Cohesion |
|---|-------|-------------|---------------------|----------|
| 1 | ... | ... | ... | Concentrated / Scattered |
```

### Step 3: Diagnose Current Structure

Briefly assess the current organization:

- **What works well** — files that are well-scoped and self-contained
- **Pain points** — files that are too large, too broad, or contain unrelated topics; content that's duplicated or split awkwardly across files
- **Missing connections** — themes that lack a clear home or are buried inside unrelated files

Present this as a short bulleted assessment (not more than 5-7 bullets total).

### Step 4: Propose Reorganization

Based on the themes and diagnosis, propose a concrete reorganization plan. The plan should describe:

1. **Proposed file structure** — a table of target files with descriptions, showing what stays, what moves, what merges, and what splits
2. **Migration map** — for each proposed change, show `source → destination` with the specific sections affected
3. **Updated `index.md`** — a preview of what the new index would look like

Format the proposal as:

```
## Proposed Structure

| File | Description | Change |
|------|-------------|--------|
| overview.md | ... | Unchanged |
| architecture.md | ... | Split: sections X, Y move to new-file.md |
| new-file.md | ... | New: receives sections from architecture.md, skills.md |

## Migration Map

| # | Section | From | To | Rationale |
|---|---------|------|----|-----------|
| 1 | ## Foo | architecture.md | new-file.md | ... |

## Updated index.md Preview

(markdown preview of new index.md)
```

**Constraints on the proposal**:
- Prefer fewer files over more — don't fragment content unnecessarily
- Preserve existing file names where possible — renames have downstream cost
- Keep each file under ~300 lines where practical
- If the current structure is already well-organized, say so and suggest only minor tweaks (or nothing)

### Step 5: User Confirmation

After presenting the proposal, ask the user:

- **Apply all** — execute the full reorganization
- **Cherry-pick** — let the user select which migrations to apply
- **Skip** — do nothing, keep the analysis for reference

If the user approves (fully or partially):

1. Execute each migration in order (move sections, update files)
2. Rewrite `fab/specs/index.md` to reflect the new structure
3. Verify no content was lost — every heading from the original files should appear in the result
4. Present a summary of changes made

---

## Output

### Analysis Complete (before confirmation)

```
Scanned {N} design docs ({L} total lines).

## Themes Found
(theme table)

## Current Structure Assessment
(bulleted diagnosis)

## Proposed Structure
(file table + migration map + index preview)

Apply this reorganization? (apply all / cherry-pick / skip)
```

### After Apply

```
Reorganization complete: {M} sections moved, {S} files modified, {C} files created.
Updated fab/specs/index.md.
```

### No Changes Needed

```
Scanned {N} design docs ({L} total lines).

Current structure is well-organized — no reorganization needed.
(optional: minor suggestions as bullets)
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/specs/index.md` missing | Abort: "fab/specs/index.md not found. Run /fab-init first." |
| No `.md` files besides index | Abort: "No design docs found in fab/specs/ besides index.md." |
| File write fails during apply | Report error, roll back that migration, continue to next |
| Content verification fails (heading lost) | Warn user, show which heading is missing, ask whether to proceed |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — not part of the change lifecycle |
| Requires active change? | **No** — operates on project-level design docs |
| Idempotent? | **Yes** — safe to run multiple times; re-analyzes each time |
| Modifies `fab/current`? | **No** |
| Modifies `.status.yaml`? | **No** |
| Modifies source code? | **No** |
| Modifies design docs? | **Yes** — only with explicit user confirmation |
| Creates git branch? | **No** |
| Requires config/constitution? | **No** |
