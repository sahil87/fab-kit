---
name: docs-reorg-specs
description: "Analyze spec files for themes and suggest reorganization. Read-only unless user approves changes."
---

# /docs-reorg-specs

---

## Contents

- Purpose
- Pre-flight
- Context Loading
- Behavior
- Output
- Error Handling
- Key Properties

---

## Purpose

Read all spec files in `docs/specs/`, identify themes (up to 10), and propose a reorganization plan. Read-only by default — files only moved/rewritten with explicit user approval.

### Reserved Paths (exempt from reorganization)

> **Specs are human-curated (a fab-kit design principle), which drives three rules:**
> 1. **Reserved paths.** `docs/specs/skills/SPEC-*.md` mirrors are constitution-pinned: their names derive mechanically from their `src/kit/skills/` sources (`SPEC-{source-filename}.md`), and the constitution requires every skill edit to update its mirror. Never propose renaming, moving, merging, or splitting them — they may be *read* for theme analysis, but a Migration Map row targeting a reserved path is invalid.
> 2. **No compatibility/backfill step.** Unlike `/docs-reorg-memory` (which detects pre-fab-kit memory trees missing `description:` frontmatter and orchestrates a frontmatter backfill), `/docs-reorg-specs` has **no** compatibility or frontmatter-backfill step. There is no specs-index generator (no counterpart to `fab memory-index`); the specs index is hand-rewritten (Step 5), so a spec missing frontmatter breaks nothing downstream — there is no compatibility contract to violate, and no generated-index model for specs. Do not "fix the asymmetry" by adding a specs backfill — it would invent a non-problem and push specs toward the generated-index model the human-curated principle rejects.
> 3. **Frontmatter-neutral moves — no FKF on specs.** FKF (`type: memory` + `description:`) governs `docs/memory/` **only**; specs are out of FKF scope and stay frontmatter-free. When this skill moves a spec it MUST NOT stamp, add, or synthesize `type:` / `description:` frontmatter — a moved spec carries exactly the bytes it had before (only its path and the `index.md` row change). This mirrors `/docs-reorg-memory`'s frontmatter-*preserving* moves: memory moves keep FKF frontmatter, spec moves add none. Spec links stay ordinary repo-relative; the index stays hand-rewritten.

---

## Pre-flight

1. `docs/specs/index.md` must exist and be readable
2. `docs/specs/` must contain at least one `.md` file besides `index.md`

If either fails, STOP with appropriate message.

---

## Context Loading

Loads `docs/specs/index.md` and every `.md` file in `docs/specs/`. Does NOT require `.fab-status.yaml`, config, or constitution.

---

## Behavior

### Step 1: Read All Spec Files

Read `docs/specs/index.md` and every `.md` file, **recursing into subfolders** (e.g., `skills/`, `findings/`). For each: extract `##`/`###` headings, brief section summaries, and approximate line count. Reserved-path files (see Reserved Paths) are read for analysis only.

### Step 2: Identify Themes (up to 10)

Analyze content for recurring topics, conceptual clusters, cross-cutting concerns. For each theme: name (2-4 words), description, source locations, cohesion (concentrated / scattered).

```
## Themes Found

| # | Theme | Description | Current Location(s) | Cohesion |
|---|-------|-------------|---------------------|----------|
```

### Step 3: Diagnose Current Structure

Brief assessment (5-7 bullets max): what works well, pain points (too large, too broad, duplicated), missing connections.

### Step 4: Propose Reorganization

```
## Proposed Structure

| File | Description | Change |
|------|-------------|--------|

## Migration Map

| # | Section | From | To | Rationale |
|---|---------|------|----|-----------|

## Updated index.md Preview
(markdown preview)
```

Constraints: prefer fewer files, preserve existing names, keep files under ~300 lines, never migrate reserved paths (`docs/specs/skills/SPEC-*.md`), say so if current structure is fine.

### Step 5: User Confirmation

Options: **Apply all**, **Cherry-pick** (select specific migrations), **Skip** (keep analysis only).

On approval: execute migrations (a moved spec keeps its exact bytes — no FKF frontmatter stamped, per Reserved Paths rule 3), rewrite `docs/specs/index.md`, verify no headings lost, present change summary.

---

## Output

```
Scanned {N} spec files ({L} total lines).

{Themes table}
{Diagnosis}
{Proposal}

Apply this reorganization? (apply all / cherry-pick / skip)
```

After apply: `Reorganization complete: {M} sections moved, {S} files modified, {C} files created.`

If no changes needed: `Current structure is well-organized — no reorganization needed.`

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `docs/specs/index.md` missing | Abort: "Run /fab-setup first." |
| No spec files besides index | Abort: "Nothing to reorganize." |
| File write fails during apply | Report error, roll back that migration, continue |
| Content verification fails | Warn, show missing heading, ask to proceed |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Requires active change? | No |
| Idempotent? | Yes |
| Modifies spec files? | Yes — only with explicit confirmation; a moved spec keeps its exact bytes (no FKF stamped — Reserved Paths rule 3) |
| Stamps FKF frontmatter? | No — never adds `type:`/`description:` to a spec; no `fab specs-index` generator and no generated-index model for specs (Reserved Paths rules 2–3) |
| Requires config/constitution? | No |
