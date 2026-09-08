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

### Human-Curated Spec Rules

> **Specs remain human-curated; navigation may be generated.**
> 1. **Generated navigation, no required backfill.** For configured roots, `fab docs-index docs/specs`
> owns index generation. Consult `_cli-fab` § fab docs-index for landing ownership, seed-import,
> sparse-description handling, superseded patterns, and the exit-2 guard. No frontmatter
> backfill is required to index specs.
> 2. **Frontmatter-neutral moves.** Moving a spec MUST NOT stamp, add, or synthesize
> `type:` / `description:` frontmatter. Keep its bytes, updating links only when the
> approved reorganization requires it. FKF applies to memory, not spec content.

---

## Pre-flight

1. `docs/specs/index.md` must exist and be readable
2. `docs/specs/` must contain at least one `.md` file besides `index.md`

If either fails, STOP with appropriate message.

---

## Context Loading

Loads `docs/specs/index.md` and every `.md` file in `docs/specs/`. Also reads `fab/project/config.yaml` when present to determine whether `docs/specs` is configured in `docs_index.roots`. Does NOT require `.fab-status.yaml` or constitution.

---

## Behavior

### Step 1: Read All Spec Files

Read `docs/specs/index.md` and every `.md` file, **recursing into subfolders** (e.g., `findings/`). For each: extract `##`/`###` headings, brief section summaries, and approximate line count.

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

## Navigation Plan
(folder/landing layout; generator output is not hand-authored)
```

Constraints: prefer fewer files, preserve existing names, keep files under ~300 lines, say so if current structure is fine.

### Step 5: User Confirmation

Options: **Apply all**, **Cherry-pick** (select specific migrations), **Skip** (keep analysis only).

On approval: execute migrations under Human-Curated Spec Rules rule 2. For a configured specs root, regenerate navigation using the `_cli-fab` § fab docs-index procedure referenced in rule 1. For an unconfigured root, retain the human-maintained index and update its approved navigation manually; include adding the root configuration in a future proposal if generation is desired. Verify no headings or links were lost, then present the change summary.

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
| Modifies spec files? | Yes — only with explicit confirmation; a moved spec keeps its exact bytes (no FKF stamped — Human-Curated Spec Rules rule 2) |
| Stamps FKF frontmatter? | No — see Human-Curated Spec Rules rule 2 |
| Requires config/constitution? | Reads optional config for generator root selection; no constitution requirement |
