---
name: docs-reorg-memory
description: "Analyze memory files for themes and suggest reorganization. Read-only unless user approves changes."
---

# /docs-reorg-memory

---

## Purpose

Read all memory files across all domains in `docs/memory/`, identify themes (up to 10), diagnose **tree shape** (fan-out and depth), and propose a reorganization plan. Read-only by default — files only moved/rewritten with explicit user approval.

This is also the **memory rebalancer**: when a domain folder grows too wide or too deep, this skill proposes splitting it into sub-domains (or merging trivially-small siblings) and, on approval, **performs the moves**, rewrites the relative links they break, and regenerates indexes via `fab memory-index`. The ideal-shape bounds below are the trigger.

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

### Sub-Domain Addressing (External)

A split creates `docs/memory/{domain}/{sub-domain}/{topic}.md`. Sub-domains are **first-class, externally addressed**: the rest of fab refers to a sub-domain file as `{domain}/{sub-domain}/{file-name}` (the flat `{domain}/{file-name}` form remains valid for un-split domains). `fab memory-index` generates a `{domain}/{sub-domain}/index.md` for every sub-domain and references it from the parent domain index, so the selective-load walk is: domain index → sub-domain index → file (see `_preamble` § Memory File Lookup).

---

## Pre-flight

1. `docs/memory/index.md` must exist and be readable
2. `docs/memory/` must contain at least one domain directory with `.md` files besides `index.md`

If either fails, STOP with appropriate message.

---

## Context Loading

Loads `docs/memory/index.md`, all domain (and sub-domain) `index.md` files, and every `.md` file in each domain. Does NOT require `.fab-status.yaml`, config, or constitution.

---

## Behavior

### Step 1: Read All Memory Files

Read `docs/memory/index.md` and every domain directory (recursing into sub-domain folders). For each memory file: extract `##`/`###` headings, brief section summaries, and approximate line count. For each folder, record its **topic-file count** (excluding `index.md`) and its **depth** relative to `docs/memory/`. Exclude `_shared/` and `_unsorted/` from shape measurement.

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

`Kind` is one of: `move-section` (relocate a `##`/`###` block between files), `split-domain` (fan out an over-width folder into sub-domains), `merge-domain` (fold an under-floor folder into a sibling), `flatten` (reduce depth > 3), `move` (relocate a single file between domains/sub-domains without a split/merge/flatten).

For any `split-domain` / `merge-domain` / `flatten` / `move` row (any move-bearing migration), the proposal MUST also list, in a **Link Impact** note, **every relative link that would break** when files move — in BOTH directions: links *from* moved files to their (now-relocated) siblings, and links *to* moved files from elsewhere in the domain. Each entry pairs the current link with its rewrite, so the user sees the full blast radius before approving:

```
## Link Impact
{N} intra-domain relative links point at files being moved. They are rewritten on apply:
- in `execution-skills.md`: `](runtime-agents.md)` → `](runtime/runtime-agents.md)`
- in `runtime/runtime-agents.md`: `](../execution-skills.md)` (link target preserved — file moved deeper, sibling did not)
```

If a move-bearing migration breaks zero links, state "Link Impact: none" explicitly so approval is informed.

Constraints: prefer fewer files per domain; preserve existing domain names where possible; keep files under ~300 lines; respect the Ideal Shape Bounds (split over-width, merge under-floor, flatten over-depth) but only when a genuine cluster justifies it; never touch `_shared`/`_unsorted`; say so if the current structure is fine.

> **Index previews are NOT hand-authored here.** Indexes are a generated artifact — `fab memory-index` writes them. The proposal shows the *post-migration folder layout*, not a hand-rolled index preview; the actual `index.md` files (domain and sub-domain tiers) are regenerated on apply (Step 5).

### Step 5: User Confirmation & Apply

Options: **Apply all**, **Cherry-pick** (select specific migrations), **Skip** (keep analysis only).

On approval, for each approved migration:

1. **Move files / sections.** Execute section moves (`move-section`) and file moves (`split-domain` / `merge-domain` / `flatten` / `move`) to their new paths. Use `git mv` semantics where possible to preserve history; a plain move is acceptable when `git mv` is unavailable.
2. **Rewrite relative links** broken by the move — every link in the proposal's **Link Impact** note, in both directions (links *from* moved files and links *to* moved files). Edit each link to its computed new relative target so no cross-file reference dangles.
3. **Add `description:` frontmatter** to any new file or new sub-domain `index.md` a split creates — the generated index reads it. Copy or synthesize it from the source file's existing description; for a new sub-domain, write a one-line `description:` summarizing the cluster.
4. **Regenerate indexes**: run `fab memory-index`. It rewrites the root, every domain `index.md`, AND every sub-domain `index.md` from folder contents (including the new sub-domain reference rows in the parent). Do **not** hand-edit index files — they are generated.
5. **Verify (no-dangling-link guard).** Confirm no headings were lost AND no broken relative link remains. **A remaining dangling relative link is a hard block** — do NOT finalize that migration until every broken link is rewritten. Report any dangling link found and the file it is in.

Present a change summary after all approved migrations are finalized.

---

## Output

```
Scanned {D} domains, {N} memory files ({L} total lines).

{Themes table}
{Shape Report}
{Diagnosis}
{Proposal — incl. Link Impact for any move-bearing migration}

Apply this reorganization? (apply all / cherry-pick / skip)
```

After apply: `Reorganization complete: {M} sections moved, {F} files moved, {S} files modified, {C} files/sub-domains created, {L2} links rewritten, {D2} domains split/merged. Indexes regenerated via fab memory-index; no dangling links.`

If no changes needed: `Current structure is well-organized — no reorganization needed.` (and the Shape Report shows all folders within bounds).

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `docs/memory/index.md` missing | Abort: "Run /fab-setup first." |
| No memory domains or files besides indexes | Abort: "Nothing to reorganize." |
| File write/move fails during apply | Report error, roll back that migration, continue |
| Content verification fails | Warn, show missing heading, ask to proceed |
| `fab memory-index` unavailable (older binary) | Warn; fall back to hand-updating affected `index.md` files (legacy path) and tell the user to upgrade `fab` |
| Broken relative link remains after a move | **Hard block** — report the dangling link; do not finalize that migration until it is rewritten |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Requires active change? | No |
| Idempotent? | Yes — a balanced tree proposes nothing; re-running `fab memory-index` is byte-stable |
| Modifies memory files? | Yes — moves + link rewrites, only with explicit confirmation |
| Requires config/constitution? | No |
| Is the memory rebalancer? | Yes — supersedes any separate `/fab-rebalance-memory`; shape diagnosis + split/merge/flatten + the file-moving apply path live here |
| Link rewriting | Skill-driven (the agent edits links per the Link Impact list) — NOT a `fab` subcommand |
| Indexes hand-edited? | No — regenerated by `fab memory-index` (domain + sub-domain tiers) |
