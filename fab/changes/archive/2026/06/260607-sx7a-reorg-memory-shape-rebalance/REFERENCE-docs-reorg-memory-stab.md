---
name: docs-reorg-memory
description: "Analyze memory files for themes and suggest reorganization. Read-only unless user approves changes."
---

# /docs-reorg-memory

---

## Purpose

Read all memory files across all domains in `docs/memory/`, identify themes (up to 10), diagnose **tree shape** (fan-out and depth), and propose a reorganization plan. Read-only by default — files only moved/rewritten with explicit user approval.

This is also the **memory rebalancer**: when a domain folder grows too wide or too deep, this skill proposes splitting it into sub-domains (or merging trivially-small siblings). The ideal-shape bounds below are the trigger.

### Ideal Shape Bounds

| Dimension | Bound | Meaning |
|-----------|-------|---------|
| Folder width | **~12 topic files (soft upper)** | Over this, propose splitting the folder into cohesive sub-domains |
| Sub-domain worth | **≥8 cohesive files** | A sub-domain earns its own `index.md` only once a real cluster this size exists — don't pre-build hierarchy |
| Folder floor | **~5 files (soft lower)** | Trivially-small sibling folders are split candidates for *merging*, not keeping |
| Tree depth | **≤3** (`{domain}/{sub-domain}/{topic}.md`) | Deeper than this, propose flattening |

Bounds are **soft SHOULD guidance**, not hard gates — a folder at 13 files is fine if the files don't cluster. Split *reactively* when a genuine cluster emerges, never prophylactically.

### Reserved Domains (exempt from bounds)

`docs/memory/_shared/` (cross-cutting concerns that map to no single domain) and `docs/memory/_unsorted/` (staging area for not-yet-placed notes) are **exempt** from the width/depth bounds. Never propose splitting, merging, or flattening them.

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

Read `docs/memory/index.md` and every domain directory. For each memory file: extract `##`/`###` headings, brief section summaries, and approximate line count. For each folder, record its **topic-file count** (excluding `index.md`) and its **depth** relative to `docs/memory/`. Exclude `_shared/` and `_unsorted/` from shape measurement.

### Step 2: Identify Themes (up to 10)

Analyze content for recurring topics, conceptual clusters, cross-cutting concerns. For each theme: name (2-4 words), description, source locations, cohesion (concentrated / scattered).

```
## Themes Found

| # | Theme | Description | Current Location(s) | Cohesion |
|---|-------|-------------|---------------------|----------|
```

### Step 3: Diagnose Current Structure

Brief assessment (5-7 bullets max): what works well, pain points (files too large, topics split across files, domain boundaries unclear, duplicated content), missing connections.

Then emit an explicit **Shape Report** flagging every folder that violates the Ideal Shape Bounds:

```
## Shape Report

| Folder | Files | Depth | Status | Suggested action |
|--------|-------|-------|--------|------------------|
| fab-workflow | 20 | 1 | ⚠ over width (~12) | split into sub-domains |
| <domain>/<sub> | 3 | 2 | ⚠ under floor (~5) | merge into sibling |
```

A folder is `✓ ok`, `⚠ over width`, `⚠ over depth`, or `⚠ under floor`. Reserved domains (`_shared`, `_unsorted`) are listed as `— exempt`. If every folder is within bounds, say so and skip straight to "structure is fine".

### Step 4: Propose Reorganization

```
## Proposed Structure

| Domain | File | Description | Change |
|--------|------|-------------|--------|

## Migration Map

| # | Item | From | To | Kind | Rationale |
|---|------|------|----|------|-----------|
```

`Kind` is one of: `move-section` (relocate a `##`/`###` block between files), `split-domain` (fan out an over-width folder into sub-domains), `merge-domain` (fold an under-floor folder into a sibling), `flatten` (reduce depth > 3).

For any `split-domain` / `merge-domain` / `flatten` row, the proposal MUST also list, in a **Link Impact** note, every relative link that would break when files move:

```
## Link Impact
{N} intra-domain relative links point at files being moved (e.g. `](runtime-agents.md)` in execution-skills.md → must become `](runtime/runtime-agents.md)`). These are rewritten on apply.
```

Constraints: prefer fewer files per domain; preserve existing domain names where possible; keep files under ~300 lines; respect the Ideal Shape Bounds (split over-width, merge under-floor, flatten over-depth) but only when a genuine cluster justifies it; never touch `_shared`/`_unsorted`; say so if the current structure is fine.

> **Index previews are NOT hand-authored here.** Indexes are a generated artifact — `fab memory-index` writes them. The proposal shows the *post-migration folder layout*, not a hand-rolled index preview; the actual `index.md` files are regenerated on apply (Step 5).

### Step 5: User Confirmation

Options: **Apply all**, **Cherry-pick** (select specific migrations), **Skip** (keep analysis only).

On approval:

1. Execute the section moves and any file moves (`split-domain` / `merge-domain` / `flatten`).
2. **Rewrite relative links** broken by file moves (every link listed in the Link Impact note), so no cross-file reference dangles.
3. **Regenerate indexes**: run `fab memory-index` to rewrite the root and every affected domain/sub-domain `index.md` from folder contents. Do **not** hand-edit index files — they are generated.
4. Verify no headings lost; present change summary.

> **`description:` frontmatter**: any new file created by a split MUST carry a `description:` frontmatter field (the generated index reads it). Copy/synthesize it from the source file's existing description.

---

## Output

```
Scanned {D} domains, {N} memory files ({L} total lines).

{Themes table}
{Shape Report}
{Diagnosis}
{Proposal}

Apply this reorganization? (apply all / cherry-pick / skip)
```

After apply: `Reorganization complete: {M} sections moved, {S} files modified, {C} files created, {D2} domains split/merged. Indexes regenerated via fab memory-index.`

If no changes needed: `Current structure is well-organized — no reorganization needed.` (and the Shape Report shows all folders within bounds).

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `docs/memory/index.md` missing | Abort: "Run /fab-setup first." |
| No memory domains or files besides indexes | Abort: "Nothing to reorganize." |
| File write fails during apply | Report error, roll back that migration, continue |
| Content verification fails | Warn, show missing heading, ask to proceed |
| `fab memory-index` unavailable (older binary) | Warn; fall back to hand-updating affected `index.md` files (legacy path) and tell the user to upgrade `fab` |
| Broken relative link remains after a move | Report the dangling link; do not finalize that migration until rewritten |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Requires active change? | No |
| Idempotent? | Yes — a balanced tree proposes nothing; re-running `fab memory-index` is byte-stable |
| Modifies memory files? | Yes — only with explicit confirmation |
| Requires config/constitution? | No |
| Is the memory rebalancer? | Yes — supersedes any separate `/fab-rebalance-memory`; shape diagnosis + split/merge/flatten proposals live here |
| Indexes hand-edited? | No — regenerated by `fab memory-index` |
