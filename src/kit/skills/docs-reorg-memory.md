---
name: docs-reorg-memory
description: "Analyze memory files for themes and suggest reorganization. Read-only unless user approves changes."
---

# /docs-reorg-memory

---

## Purpose

Read all memory files across all domains in `docs/memory/`, identify themes (up to 10), and propose a reorganization plan. Read-only by default — files only moved/rewritten with explicit user approval.

---

## Pre-flight

1. `docs/memory/index.md` must exist and be readable
2. `docs/memory/` must contain at least one domain directory with `.md` files besides `index.md`

If either fails, STOP with appropriate message.

---

## Context Loading

Loads `docs/memory/index.md`, all domain `index.md` files, and every `.md` file in each domain. Does NOT require `.fab-status.yaml`, config, or constitution.

---

## Behavior

### Step 1: Read All Memory Files

Read `docs/memory/index.md` and every domain directory. For each memory file: extract `##`/`###` headings, brief section summaries, and approximate line count.

### Step 2: Identify Themes (up to 10)

Analyze content for recurring topics, conceptual clusters, cross-cutting concerns. For each theme: name (2-4 words), description, source locations, cohesion (concentrated / scattered).

```
## Themes Found

| # | Theme | Description | Current Location(s) | Cohesion |
|---|-------|-------------|---------------------|----------|
```

### Step 3: Diagnose Current Structure

Brief assessment (5-7 bullets max): what works well, pain points (files too large, topics split across files, domain boundaries unclear, duplicated content), missing connections.

#### Shape Report (read-only)

Compute per-folder file counts and tree depth, then flag every folder that violates the ideal-shape bounds. This is **detect/diagnose only** — it never moves files (see Scope below).

```
## Shape Report

| Folder | Files | Depth | Bound | Finding |
|--------|-------|-------|-------|---------|
| docs/memory/<domain> | 20 | 2 | ~12 max | Over-wide — candidate for sub-domain split |
| docs/memory/<domain>/<sub>/<deep> | 3 | 4 | depth ≤3 | Too deep — candidate for flattening |
| docs/memory/<thin> | 2 | 2 | ~5 min | Under floor — candidate for merge into a sibling |
```

Bounds (SHOULD guidance — the same ones `fab memory-index` warns on):
- **Width**: ~12 topic files per folder upper, ~5 lower.
- **Depth**: ≤3 (`docs/memory/{domain}/{sub-domain}/{topic}.md`).
- **Sub-domain earns its own index** only at a cohesive cluster of **≥8 files**.
- **Reserved domains `_shared/` and `_unsorted/` are exempt** from the width finding (cross-cutting / staging buckets).

If no folder violates the bounds, state "Shape is within bounds" and move on.

> **Scope (this skill, today)**: the Shape Report and the migration map below are **diagnostic** — they *propose* split / merge / flatten actions. The file-moving *apply* path (actually moving files into sub-domains and rewriting the ~intra-domain relative links a move breaks) is a deferred follow-up; until it lands, treat split/merge/flatten as proposals to review, and let `fab memory-index` regenerate indexes once any approved moves are made by hand.

### Step 4: Propose Reorganization

```
## Proposed Structure

| Domain | File | Description | Change |
|--------|------|-------------|--------|

## Migration Map

| # | Kind | Section/Files | From | To | Rationale |
|---|------|---------------|------|----|-----------|

(Migration **Kind** is one of: `split` (over-wide folder → sub-domain), `merge` (under-floor folder → sibling), `flatten` (over-depth → shallower), or `move` (re-home a topic). Indexes are NOT previewed here — `fab memory-index` regenerates them after any approved move.)
```

Constraints: prefer fewer files per domain (respect the ~5–12 / depth ≤3 bounds from the Shape Report), preserve existing domain names where possible, keep files under ~300 lines, say so if current structure is fine. Reserved domains `_shared/` and `_unsorted/` are exempt from the width bound.

### Step 5: User Confirmation

Options: **Apply all**, **Cherry-pick** (select specific migrations), **Skip** (keep analysis only).

On approval: execute the approved content migrations, verify no headings lost, then run **`fab memory-index`** to regenerate every affected `index.md` (domain-level and top-level) deterministically — never hand-edit index rows. Present a change summary.

---

## Output

```
Scanned {D} domains, {N} memory files ({L} total lines).

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
| `docs/memory/index.md` missing | Abort: "Run /fab-setup first." |
| No memory domains or files besides indexes | Abort: "Nothing to reorganize." |
| File write fails during apply | Report error, roll back that migration, continue |
| Content verification fails | Warn, show missing heading, ask to proceed |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Requires active change? | No |
| Idempotent? | Yes |
| Modifies memory files? | Yes — only with explicit confirmation |
| Requires config/constitution? | No |
