---
name: docs-reorg-memory
description: "Analyze memory files for themes and suggest reorganization. Read-only unless user approves changes. Also the memory rebalancer — diagnoses folder shape and splits/merges/flattens domains, rewriting links, on approval."
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
| Tree depth | **≤3 path segments** under `docs/memory/` (`{domain}/{sub-domain}/{topic}.md`) — equivalently, **folder depth ≤2** | Deeper than this, propose flattening |

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

Read `docs/memory/index.md` and every domain directory (recursing into sub-domain folders). For each memory file: extract `##`/`###` headings, brief section summaries, and approximate line count. For each folder, record its **topic-file count** (excluding `index.md`) and its **depth** in folder levels relative to `docs/memory/` (a domain is depth 1, a sub-domain depth 2). Exclude `_shared/` and `_unsorted/` from shape measurement.

**Compatibility detection (pre-fab-kit tree).** The read-all-files pass already reads every file, so during it also detect the three ways a hand-curated, pre-fab-kit tree diverges from the convention `fab memory-index` depends on — these are shape diagnoses, surfaced in the findings report (Step 3) for approval like any other:

- **Missing `description:` frontmatter.** A topic file (a non-`index.md` `.md` file) that has no frontmatter, or frontmatter without a `description:` key, counts as **missing** — the same `frontmatter.Field` semantics `fab memory-index` uses. The generator reads descriptions exclusively from this field, so a missing one renders `—` (wiping any curated description) on the next regen.
- **Tombstone rows.** A row in an *existing* hand-curated index file whose **`docs/memory/`-relative link target is absent on disk** is a tombstone candidate — the removal-history rows the generator silently drops (it walks only folders that exist). The unresolved relative-link target is the **primary signal**; strikethrough syntax (`~~lib-bdash~~`) is a **corroborating hint** that raises confidence but is **not required** — un-struck tombstones are still caught. Scope the signal to `docs/memory/`-relative paths only, so intentional external links (URLs, absolute paths) never false-positive. Tombstone candidates are **surfaced for explicit user confirmation** before any relocation (Step 5).
- **Custom structural groupings.** Structural headings in the existing root `index.md` beyond the generated domains-only table (e.g., `### Apps`, `### Packages`, `### Cross-cutting`) — content the domains-only regen will flatten.

Record these as compatibility findings alongside the shape measurements; they feed the findings report in Step 3.

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

A folder is `✓ ok`, `⚠ over width`, `⚠ over depth`, or `⚠ under floor`. The `Depth` column counts folder levels (domain = 1, sub-domain = 2); since the ≤3 bound counts the *topic file's* path segments, `⚠ over depth` fires for any folder deeper than 2 — its files sit at ≥4 segments. Reserved domains (`_shared`, `_unsorted`) are listed as `— exempt`. If every folder is within bounds, say so and skip straight to "structure is fine".

**Compatibility report (only when a pre-fab-kit divergence was found in Step 1).** When the Step 1 compatibility detection found any missing-frontmatter files, tombstone rows, or custom groupings, emit a Compatibility section enumerating them and the proposed remediation. **Omit this section entirely when no compatibility findings exist** — a born-compatible fab-kit tree sees no behavioral change.

```
## Compatibility (pre-fab-kit memory tree detected)

- 12 topic files lack `description:` frontmatter (will render as — on regen)
- 6 tombstone rows reference removed folders (will be dropped by fab memory-index)
- Grouped layout (Apps / Packages / Cross-cutting) will flatten to a domains-only table

Proposed remediation (on approval):
  1. Relocate tombstone rows → docs/memory/_shared/removed-domains.md
  2. Backfill description: frontmatter (12 files, via docs-hydrate-memory)
  3. Rebalance + regenerate indexes (fab memory-index)
```

List the tombstone candidates explicitly (with their source index and link target) so the user can confirm them before relocation — un-struck candidates especially, since they are the easiest to misjudge.

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

#### Compatibility orchestration (when Step 1 found a pre-fab-kit divergence)

When the Compatibility report (Step 3) listed findings, the proposed remediation runs **only on explicit user approval** (reorg's existing posture). **If the user declines, report the compatibility findings and stop without mutating any file** — no `removed-domains.md` written, no backfill dispatched, no `fab memory-index` run; the user keeps their hand-curated tree intact. On approval, run the remediation in this **strict order**, *before* the normal rebalance/regenerate of the approved structural migrations below:

1. **Relocate tombstones → `docs/memory/_shared/removed-domains.md`.** reorg authors this **single mechanical file** — the **only** per-file content-authoring action reorg performs, bounded to mechanical row relocation and explicitly NOT per-file description synthesis:
   - A leading `description:` frontmatter one-liner (so the file round-trips through `fab memory-index` like any topic file), then an H1, then the **user-confirmed tombstone rows lifted verbatim from the old index** — preserving the change IDs that explain each removal (the decision context derivable from nothing else).
   - **Merge-not-duplicate**: if `docs/memory/_shared/removed-domains.md` already exists, merge the new tombstone rows without duplicating any row already present (match on the row's identifying content). A re-run with no new tombstones leaves the file byte-stable (idempotency, Constitution III).
2. **Dispatch backfill to `/docs-hydrate-memory`** as a **general-purpose sub-agent** (per `_preamble.md` § Subagent Dispatch — pass the standard subagent context, the 5 project files). The dispatch prompt:
   - Instructs the sub-agent to run `/docs-hydrate-memory`'s **backfill mode** over the tree, **naming the operation** ("backfill this tree — add `description:` frontmatter to files missing it") — it does **NOT** pass a file manifest; backfill **re-scans `docs/memory/` itself** (the loose, idempotent seam). Synthesis lives there, not in reorg.
   - **Signals the reorg-dispatched (defer-regen) caller form** so backfill does NOT run `fab memory-index` — reorg owns the single regen in step 3.
3. **Rebalance + regenerate.** After backfill returns, run reorg's existing split/merge/flatten logic for any approved structural migrations (the per-migration apply below — whose item 4 runs `fab memory-index`). **The backfill sub-agent ran zero regens** (it was dispatched defer-regen), so reorg owns the regeneration. If there are **no** structural migrations (compatibility findings only), run `fab memory-index` **once** here directly to pick up the new `description:` frontmatter, the `removed-domains.md` row, and the regenerated indexes. Either way the tree is regenerated exactly once after backfill, never by the sub-agent. (`fab memory-index` is byte-stable, so an extra idempotent run is harmless — but the contract is one regen, owned by reorg.)

> The `description:` frontmatter on `removed-domains.md` (step 1) is authored **before** this regeneration, the same stub-before-index discipline reorg already uses for new sub-domains — so `fab memory-index` reads its description and lists `_shared/removed-domains.md` like any topic file. (`_shared/` is width-exempt, so it is never flagged or split.)

After this orchestration completes (or if there were no compatibility findings), proceed with the normal per-migration apply for any approved structural migrations:

On approval, for each approved migration:

1. **Move files / sections.** Execute section moves (`move-section`) and file moves (`split-domain` / `merge-domain` / `flatten` / `move`) to their new paths. Use `git mv` semantics where possible to preserve history; a plain move is acceptable when `git mv` is unavailable.
2. **Rewrite relative links** broken by the move — every link in the proposal's **Link Impact** note, in both directions (links *from* moved files and links *to* moved files). Edit each link to its computed new relative target so no cross-file reference dangles.
3. **Author `description:` frontmatter** — the single hand-curated index field. For any new topic file a split creates, add a `description:` frontmatter line (copy or synthesize from the source file's existing description). For each **new sub-domain**, create a **stub `index.md` BEFORE `fab memory-index` runs (step 4)**: the stub is only the `description:` frontmatter block (a one-liner summarizing the cluster), nothing else — the same stub-before-index pattern as `/docs-hydrate-memory`. Step 4's regeneration fills in the generated body and round-trips the description.
4. **Regenerate indexes**: run `fab memory-index`. It rewrites the root, every domain `index.md`, AND every sub-domain `index.md` from folder contents (including the new sub-domain reference rows in the parent). Generated index content (file rows, "Last Updated" cells) is never hand-edited — the `description:` frontmatter authored in step 3 (including the sub-domain stubs) is the only hand-curated part of any index file.
5. **Verify (no-dangling-link guard).** Confirm no headings were lost AND no broken relative link remains. **A remaining dangling relative link is a hard block** — do NOT finalize that migration until every broken link is rewritten. Report any dangling link found and the file it is in. **Abort escape**: if a dangling link cannot be rewritten (its target is genuinely gone, or the correct target is ambiguous), abort that migration instead of blocking indefinitely — roll back its moves and link rewrites (restore original paths), re-run `fab memory-index`, report the rollback, and continue with the remaining approved migrations.

Present a change summary after all approved migrations are finalized.

---

## Output

```
Scanned {D} domains, {N} memory files ({L} total lines).

{Themes table}
{Shape Report}
{Compatibility report — only when a pre-fab-kit divergence was found}
{Diagnosis}
{Proposal — incl. Link Impact for any move-bearing migration}

Apply this reorganization? (apply all / cherry-pick / skip)
```

After apply: `Reorganization complete: {M} sections moved, {F} files moved, {S} files modified, {C} files/sub-domains created, {L2} links rewritten, {D2} domains split/merged. Indexes regenerated via fab memory-index; no dangling links.`

When compatibility remediation ran, also report: `Compatibility migration: {T} tombstone rows relocated to _shared/removed-domains.md, {B} files backfilled with description: frontmatter (via docs-hydrate-memory sub-agent).` When the user declined remediation: `Compatibility findings reported; tree left intact (no files mutated).`

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
| Broken relative link remains after a move | **Hard block** — report the dangling link; do not finalize that migration until it is rewritten. If it cannot be rewritten, take the abort escape defined in Step 5's Verify item (apply item 5): roll back that migration, regenerate indexes, continue with the rest |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Requires active change? | No |
| Idempotent? | Yes — a balanced tree proposes nothing; re-running `fab memory-index` is byte-stable; an already-converted tree re-runs as a no-op (no duplicate tombstone rows in `removed-domains.md`, backfill skips frontmatter-present files) |
| Modifies memory files? | Yes — moves + link rewrites, only with explicit confirmation. On compatibility approval also authors the ONE mechanical file `docs/memory/_shared/removed-domains.md` (tombstone relocation); per-file `description:` synthesis is delegated to the `/docs-hydrate-memory` backfill sub-agent, not authored here |
| Requires config/constitution? | No |
| Is the memory rebalancer? | Yes — supersedes any separate `/fab-rebalance-memory`; shape diagnosis + split/merge/flatten + the file-moving apply path live here |
| Orchestrates a pre-fab-kit compatibility migration? | Yes — detects missing `description:` frontmatter, tombstone rows, and custom groupings; on approval relocates tombstones (`_shared/removed-domains.md`), dispatches `/docs-hydrate-memory` backfill mode as a sub-agent, then rebalances + regenerates once. Decline = report and stop, no mutation |
| Dispatches sub-agents? | Yes — `/docs-hydrate-memory` backfill mode (general-purpose sub-agent, standard subagent context) for per-file description synthesis during compatibility orchestration |
| Link rewriting | Skill-driven (the agent edits links per the Link Impact list) — NOT a `fab` subcommand |
| Indexes hand-edited? | No — regenerated by `fab memory-index` (domain + sub-domain tiers) |
