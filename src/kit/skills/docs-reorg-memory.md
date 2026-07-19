---
name: docs-reorg-memory
description: "Analyze memory files for themes and suggest reorganization. Read-only unless user approves changes. Also the memory rebalancer — diagnoses folder shape and splits/merges/flattens domains, rewriting links, on approval."
---

# /docs-reorg-memory

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

Read all memory files across all domains in `docs/memory/`, identify themes (up to 10), diagnose **tree shape** (folder fan-out/depth **and** over-size files), detect **duplicate coverage** (the same topic in 2+ files), triage `_unsorted/` staging, and propose a reorganization plan. Read-only by default — files only moved/rewritten with explicit user approval.

This is also the **memory rebalancer**: when a domain folder grows too wide or too deep, this skill proposes splitting it into sub-domains (or merging trivially-small siblings); when a single topic file grows into several topics, it proposes a **file split** (`split-file`); when two files cover the same topic, it proposes a **file merge** (`merge-file`) — and, on approval, **performs the moves**, rewrites the **bundle-relative** links they break, and regenerates indexes via `fab memory-index`. The ideal-shape bounds below (folder **and** file granularity) are the trigger.

> **FKF-aware moves (`$(fab kit-path)/reference/fkf.md` §7).** Memory↔memory links are **bundle-relative** (`](/{domain}[/{sub}]/{file}.md)`, resolved from `docs/memory/`), so a moved file's *inbound* links only need the path-after-`/` updated when the file changes domain/sub-domain, and **sibling-relative breakage largely disappears** — that is the §7 rationale ("reorg rewrites *far* fewer links"). Every move MUST **preserve the moved file's FKF frontmatter** (`type: memory` + `description:`) verbatim — never strip or regenerate it. The bulk of link rewriting under FKF is updating bundle-relative targets *to* a moved file; bare/relative sibling links no longer dominate the blast radius.

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

**Compatibility detection (pre-fab-kit tree) — mechanical, via `fab memory-index --check --json`.** Detection of the three ways a hand-curated, pre-fab-kit tree diverges from the convention `fab memory-index` depends on is a **Go primitive**, not prose the agent re-derives. Run:

```sh
fab memory-index --check --json
```

and read its exit code + JSON loss report (see `_cli-fab` § fab memory-index):

- **Exit 0** (clean) / **exit 1** (benign drift): **no compatibility findings** — nothing to relocate or backfill. Skip the Compatibility report (Step 3) entirely. This is the born-compatible fab-kit case.
- **Exit 2** (destructive loss): the JSON `losses[]` enumerates each divergence by `category` — `description` (a curated description that would regenerate to `—`), `tombstone` (a row whose `docs/memory/`-relative link target is absent on disk — the removal-history rows the generator drops; external/absolute links are already excluded by the primitive), and `grouping` (a custom structural heading in the root `index.md` the domains-only regen would flatten). Each loss carries `path` (the index file) and `detail` (the lost text / dropped link target / flattened heading). Map these directly into the findings report (Step 3); tombstone candidates are still **surfaced for explicit user confirmation** before any relocation (Step 5).

The primitive is the single source of truth for what regen would destroy — reorg consumes its classification rather than re-implementing `frontmatter.Field` semantics, tombstone heuristics, or the flatten rule in prose. Record the parsed findings alongside the shape measurements; they feed the findings report in Step 3.

**One call feeds three consumers.** The same `fab memory-index --check --json` invocation additionally records the report's **`warnings[]`** array — the advisory machine surface — alongside the `losses[]` compatibility findings:

- **`file-size`** findings (a topic file over ~400 lines OR ~15KB, carrying `count` = lines and `bytes`) → the **Shape Report file rows** (split candidates, Step 3).
- **`unsorted-nonempty`** finding (a non-empty `docs/memory/_unsorted/`) → the **`_unsorted/` staging triage** pass (Step 3/4).

So one call feeds compatibility detection (`losses[]`), the file-split Shape Report rows (`warnings[]` `file-size`), and the `_unsorted/` triage (`warnings[]` `unsorted-nonempty`). Record all three alongside the shape/theme measurements. The **completion chain to `/docs-distill-memory`** (§ Output) reuses this **same** call's `malformed[]` + `warnings[]` output once more at completion — a fourth reuse, no second `fab memory-index` call.

> **Older-binary fallback.** If `fab memory-index` does not support `--check`'s loss tiers / `--json` (an older binary — `--json` is unknown, the exit code is binary clean/dirty with no tier 2, or the `warnings` key is absent), fall back to the **legacy prose detection** during the read-all-files pass: a topic file with no frontmatter or no `description:` key is a missing-description finding (`frontmatter.Field` semantics); a row in an existing index whose `docs/memory/`-relative link target is absent on disk is a tombstone candidate (relative-target-absent is the primary signal, strikethrough `~~...~~` a non-required corroborating hint, external/absolute links excluded); a `### Apps`/`### Packages`-style heading in the root `index.md` beyond the domains-only table is a custom-grouping finding. **The `warnings[]` consumers also fall back**: file-size split candidates are measured from the read-all-files pass's approximate line counts (Step 1 already records them); `_unsorted/` staging is read by a direct `docs/memory/_unsorted/` folder listing. Then warn the user to upgrade `fab` so detection becomes mechanical.

### Step 2: Identify Themes (up to 10)

Analyze content for recurring topics, conceptual clusters, cross-cutting concerns. For each theme: name (2-4 words), description, source locations, cohesion (concentrated / scattered).

```
## Themes Found

| # | Theme | Description | Current Location(s) | Cohesion |
|---|-------|-------------|---------------------|----------|
```

### Step 3: Diagnose Current Structure

Brief assessment (5-7 bullets max): what works well, pain points (files too large, topics split across files, domain boundaries unclear, duplicated content), missing connections.

Then emit an explicit **Shape Report** flagging every folder that violates the Ideal Shape Bounds **and every over-size topic file** (the file-split candidates):

```
## Shape Report

| Folder | Files | Depth | Status | Suggested action |
|--------|-------|-------|--------|------------------|
| fab-workflow | 20 | 1 | ⚠ over width (~12) | split into sub-domains |
| <domain>/<sub> | 3 | 2 | ⚠ under floor (~5) | merge into sibling |

### Over-size files (split candidates)

| File | Lines | Size | Status | Suggested action |
|------|-------|------|--------|------------------|
| pipeline/test-suite-reference.md | 2757 | 96KB | ⚠ over size — 5 topic clusters | split-file into 5 topic files |
| runtime/big-but-cohesive.md | 612 | 22KB | ⚠ over size — long but cohesive; no split proposed | leave (report only) |
```

A folder is `✓ ok`, `⚠ over width`, `⚠ over depth`, or `⚠ under floor`. The `Depth` column counts folder levels (domain = 1, sub-domain = 2); since the ≤3 bound counts the *topic file's* path segments, `⚠ over depth` fires for any folder deeper than 2 — its files sit at ≥4 segments. Reserved domains (`_shared`, `_unsorted`) are listed as `— exempt`.

**File rows — reactive, not prophylactic.** A topic file over `[mxgu]`'s soft cap (~400 lines OR ~15KB — sourced from the Step 1 `warnings[]` `file-size` findings, `count` = lines / `bytes` = size; older-binary fallback measures during the read-all-files pass) is a **split candidate**, but a split is **proposed only when its heading clusters reveal ≥2 genuine topics**. A long-but-cohesive file is reported (`⚠ over size — long but cohesive; no split proposed`) and **left alone** — the same soft-SHOULD stance as the folder bounds (split reactively when a genuine cluster emerges, never prophylactically). If every folder is within bounds AND no file is over size, say so and skip straight to "structure is fine".

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

**Duplicate-coverage detection.** Alongside theme identification (Step 2), flag the **same topic covered in 2+ files**. Signals: near-identical filenames or `description:` frontmatter (e.g. `mock-infrastructure.md` vs `msw-mock-infrastructure.md`), the **same filename in two domains** (e.g. `right-panel-sections.md` under two domains), and heavy heading overlap between two files. Emit a `## Duplicate Coverage` table (omit entirely when no duplication is found):

```
## Duplicate Coverage

| Topic | Files | Evidence | Proposed canonical home |
|-------|-------|----------|-------------------------|
| Mock infrastructure | mock-infrastructure.md, msw-mock-infrastructure.md | near-identical filenames + 6 shared headings | mock-infrastructure.md (merge-file) |
| Right-panel sections | app-a/right-panel-sections.md, app-b/right-panel-sections.md | same filename, two domains; partial overlap | move-section (partial) |
```

Remediation rides the Migration Map: a **`merge-file`** row (move B's unique sections into canonical file A via the `move-section` machinery, rewrite all inbound links to A, delete the emptied B — parallel to `merge-domain` at file granularity, with the Link Impact note + the no-dangling-link guard), or plain **`move-section`** rows for partial overlap. The report **notes the tie to the open single-sourcing seam audit** as a cross-reference (not scope — this pass surfaces cross-file duplication; the audit is a separate effort).

**`_unsorted/` staging triage.** `_unsorted/` keeps its bounds exemption (never split/merged/flattened — Reserved Domains), but gains a **triage listing**: every staged topic file gets a per-file proposal. Signal: the Step 1 `warnings[]` `unsorted-nonempty` finding; fallback: a direct `docs/memory/_unsorted/` folder listing. Emit a `## _unsorted/ Triage` block (omit when `_unsorted/` is empty or absent):

```
## _unsorted/ Triage — staging should trend to empty

| File | Proposal | Rationale |
|------|----------|-----------|
| session-notes-infra-505.md | delete | session notes for shipped change infra-505; superseded, recorded in log/git |
| new-feature-note.md | move → runtime | belongs to the runtime domain |
```

Per-file proposal is **`move`** to a named domain (the existing `move` kind — the **default**) or **`delete`** for stale ephemera whose content is superseded or recorded elsewhere (e.g. session notes for a shipped change). **Every `delete` requires explicit per-file confirmation** (Step 5); git recoverability bounds the risk. `move` proposals ride the existing `move` Migration Map kind.

### Step 4: Propose Reorganization

```
## Proposed Structure

| Domain | File | Description | Change |
|--------|------|-------------|--------|

## Migration Map

| # | Item | From | To | Kind | Rationale |
|---|------|------|----|------|-----------|
```

`Kind` is one of: `move-section` (relocate a `##`/`###` block between files), `split-domain` (fan out an over-width folder into sub-domains), `merge-domain` (fold an under-floor folder into a sibling), `flatten` (reduce depth > 3), `move` (relocate a single file between domains/sub-domains without a split/merge/flatten), `split-file` (fan one multi-topic file into ≥2 topic files in the same domain/sub-domain — file-granularity parallel of `split-domain`), `merge-file` (fold a duplicate-coverage file's unique sections into a canonical sibling and delete the emptied file — file-granularity parallel of `merge-domain`; see the duplicate-coverage detection pass in Step 3).

#### `split-file` (file-granularity split)

A `split-file` row fans one over-size, multi-topic file into **≥2 topic files in the same domain/sub-domain** (parallel to `split-domain` at file granularity), proposed only for a file whose heading clusters show ≥2 genuine topics (Shape Report file rows). Rules:

- **Body content moves verbatim** — each cluster's `##`/`###` blocks move byte-for-byte to their new file. Restyling prose to present-truth remains **`/docs-distill-memory`'s** job, preserving the skills' division of labor (reorg moves structure; distill rewrites prose).
- **New files carry `type: memory` + a fresh change-id-free `description:`** — the same authoring rule as `split-domain`'s new files (a routing signal, not a provenance record, FKF §3.2).
- **The original path is kept for the dominant topic** when one exists (the largest / titular cluster stays at the original filename), else the emptied original is removed.
- **New files target ~300 lines** — detection flags at 400 lines / 15KB; authoring aims at ~300 (the existing "keep files under ~300 lines" constraint).
- **Width chaining** — a split that pushes the folder past the ~12 width bound MAY chain into the existing `split-domain` flow **in the same proposal**.
- **Link Impact** (below) extends to `split-file`: an **anchored** inbound link (`#heading`) follows the file its heading moved to; an **un-anchored** inbound link retargets to the **dominant-topic file**. Ambiguity (no dominant topic AND un-anchored inbound links) → the existing **abort escape** (roll back that migration, regenerate, continue).

For any `split-domain` / `merge-domain` / `flatten` / `move` / `split-file` / `merge-file` row (any move-bearing migration), the proposal MUST also list, in a **Link Impact** note, **every bundle-relative link that would break** when files move (dominant case: links *to* a moved file, whose path-after-`/` changes — per the §7 blockquote above; for `split-file`, per the anchored-vs-un-anchored rule above). Each entry pairs the current link with its rewrite, so the user sees the full blast radius before approving:

```
## Link Impact
{N} bundle-relative links point at files whose bundle path changes. They are rewritten on apply:
- in `execution-skills.md`: `](/pipeline/runtime-agents.md)` → `](/pipeline/runtime/runtime-agents.md)`
- in `clarify.md`: `](/pipeline/runtime-agents.md#gc)` → `](/pipeline/runtime/runtime-agents.md#gc)` (anchor preserved)
```

A link from a moved file to a sibling that did NOT move needs no rewrite (the §7 payoff — see the FKF-aware-moves blockquote above). If a move-bearing migration breaks zero links, state "Link Impact: none" explicitly so approval is informed.

Constraints: prefer fewer files per domain; preserve existing domain names where possible; keep files under ~300 lines (`split-file` targets ~300 for its new files); respect the Ideal Shape Bounds (split over-width, merge under-floor, flatten over-depth) but only when a genuine cluster justifies it; never **split/merge/flatten** `_shared`/`_unsorted` (they keep their bounds exemption — but `_unsorted/` staged files ARE triaged per-file via the `_unsorted/` triage pass: `move` to a domain, or `delete` with per-file confirmation); say so if the current structure is fine.

> **Index previews are NOT hand-authored here.** Indexes are a generated artifact — `fab memory-index` writes them. The proposal shows the *post-migration folder layout*, not a hand-rolled index preview; the actual `index.md` files (domain and sub-domain tiers) are regenerated on apply (Step 5).

### Step 5: User Confirmation & Apply

Options: **Apply all**, **Cherry-pick** (select specific migrations), **Skip** (keep analysis only).

#### Compatibility orchestration (when Step 1 found a pre-fab-kit divergence)

When the Compatibility report (Step 3) listed findings, the proposed remediation runs **only on explicit user approval** (reorg's existing posture). **If the user declines, report the compatibility findings and stop without mutating any file** — no `removed-domains.md` written, no backfill dispatched, no `fab memory-index` run; the user keeps their hand-curated tree intact. On approval, run the remediation in this **strict order**, *before* the normal rebalance/regenerate of the approved structural migrations below:

1. **Relocate tombstones → `docs/memory/_shared/removed-domains.md`.** reorg authors this **single mechanical file** — the **only** per-file content-authoring action reorg performs, bounded to mechanical row relocation and explicitly NOT per-file description synthesis:
   - A leading `description:` frontmatter one-liner **free of change-ids** (a routing signal, FKF §3.2 — so the file round-trips through `fab memory-index` like any topic file), then an H1, then the **user-confirmed tombstone rows lifted verbatim from the old index** — preserving the change IDs that explain each removal in the **body rows** (a verbatim tombstone row is a citation-carrying record of a removal, not transition narration; the ban is on change-ids in the `description:` frontmatter, not on these body citations).
   - **Merge-not-duplicate**: if `docs/memory/_shared/removed-domains.md` already exists, merge the new tombstone rows without duplicating any row already present (match on the row's identifying content). A re-run with no new tombstones leaves the file byte-stable (idempotency — a fab-kit design principle).
2. **Dispatch backfill to `/docs-hydrate-memory`** as a **general-purpose sub-agent** (per `_preamble.md` § Subagent Dispatch — pass the standard subagent context, the 5 project files). The dispatch prompt:
   - Instructs the sub-agent to run `/docs-hydrate-memory`'s **backfill mode** over the tree, **naming the operation** ("backfill this tree — add `description:` frontmatter to files missing it") — it does **NOT** pass a file manifest; backfill **re-scans `docs/memory/` itself** (the loose, idempotent seam). Synthesis lives there, not in reorg.
   - **Signals the reorg-dispatched (defer-regen) caller form** so backfill does NOT run `fab memory-index` — reorg owns the single regen in step 3.
3. **Rebalance + regenerate.** After backfill returns, run reorg's existing split/merge/flatten logic for any approved structural migrations (the per-migration apply below — whose item 4 runs `fab memory-index`). **The backfill sub-agent ran zero regens** (it was dispatched defer-regen), so reorg owns the regeneration. If there are **no** structural migrations (compatibility findings only), run `fab memory-index` **once** here directly to pick up the new `description:` frontmatter, the `removed-domains.md` row, and the regenerated indexes. Either way the tree is regenerated exactly once after backfill, never by the sub-agent. (`fab memory-index` is byte-stable, so an extra idempotent run is harmless — but the contract is one regen, owned by reorg.)

> The `description:` frontmatter on `removed-domains.md` (step 1) is authored **before** this regeneration, the same stub-before-index discipline reorg already uses for new sub-domains — so `fab memory-index` reads its description and lists `_shared/removed-domains.md` like any topic file. (`_shared/` is width-exempt, so it is never flagged or split.)

After this orchestration completes (or if there were no compatibility findings), proceed with the normal per-migration apply for any approved structural migrations:

On approval, for each approved migration:

1. **Move files / sections, preserving FKF frontmatter.** Execute section moves (`move-section`) and file moves (`split-domain` / `merge-domain` / `flatten` / `move` / `split-file` / `merge-file`) to their new paths. Use `git mv` semantics where possible to preserve history; a plain move is acceptable when `git mv` is unavailable. **A moved file keeps its FKF frontmatter (`type: memory` + `description:`) byte-for-byte** — moving never strips, regenerates, or re-stamps it (FKF §3; only `fab memory-index` round-trips index/log frontmatter, never topic-file `type:`). For a **`split-file`**, each cluster's blocks move **verbatim** into a new topic file (`type: memory` + a fresh change-id-free `description:`); the original path stays for the dominant topic or is removed. For a **`merge-file`**, B's unique sections move (via the `move-section` machinery) into canonical file A, then the emptied B is deleted.
2. **Rewrite bundle-relative links** broken by the move — every link in the proposal's **Link Impact** note. Edit each link's path-after-`/` to the moved file's new bundle path (`](/{new-domain}[/{sub}]/{file}.md)`), preserving any `#anchor`. A link to a sibling whose bundle path did NOT change needs no edit (§7). The guard below confirms no `](/…)` memory link dangles.
3. **Author `description:` frontmatter** — the single hand-curated index field. For any new topic file a split creates, add a `description:` frontmatter line (copy or synthesize from the source file's existing description) **alongside `type: memory`** (the FKF constant — stamp it on any genuinely new topic file so it is FKF-conforming; a *moved* file already carries it from step 1). Any `description:` you author **carries no change-ids** — it is a routing signal, not a provenance record (FKF §3.2). For each **new sub-domain**, create a **stub `index.md` BEFORE `fab memory-index` runs (step 4)**: the stub is only the `description:` frontmatter block (a one-liner summarizing the cluster), nothing else — the same stub-before-index pattern as `/docs-hydrate-memory` (FKF §5 stub-before-index). Step 4's regeneration fills in the generated body and round-trips the description.
4. **Regenerate indexes (and logs)**: run `fab memory-index`. It rewrites the root, every domain `index.md`, AND every sub-domain `index.md` from folder contents (including the new sub-domain reference rows in the parent), and regenerates each folder's `log.md` (merging any `log.seed.md` seed input beneath the git-projected entries — FKF §6). Generated content is never hand-edited (see the Index-previews blockquote above) — the `description:` frontmatter from step 3 (including the sub-domain stubs) is the only hand-curated part.
5. **Verify (no-dangling-link guard).** Confirm no headings were lost AND no broken bundle-relative link remains. **A remaining dangling `](/…)` memory link is a hard block** — do NOT finalize that migration until every broken link is rewritten. Report any dangling link found and the file it is in. **Abort escape**: if a dangling link cannot be rewritten (its target is genuinely gone, or the correct target is ambiguous), abort that migration instead of blocking indefinitely — roll back its moves and link rewrites (restore original paths), re-run `fab memory-index`, report the rollback, and continue with the remaining approved migrations.

For a **`split-file`**, the Link Impact rewrite (step 2) applies the anchored-vs-un-anchored rule: an **anchored** inbound link (`#heading`) is rewritten to the new file its heading moved to; an **un-anchored** inbound link is rewritten to the **dominant-topic file**. A `split-file` with **no dominant topic AND un-anchored inbound links** is ambiguous → take the abort escape above (roll back that migration, regenerate, continue). A **`merge-file`** rewrites every inbound link that pointed at the emptied B to canonical A (the no-dangling-link guard confirms none dangle after B is deleted).

**`_unsorted/` triage apply.** For each approved `_unsorted/` triage proposal: a **`move`** rides the normal per-migration apply above (it is a `move` Migration Map row — relocate the file into the named domain, rewrite bundle-relative links, regenerate). A **`delete`** is applied **only after explicit per-file confirmation** — confirm each staged file individually, then remove it (git recoverability bounds the risk). `_unsorted/` itself is never split/merged/flattened; only its staged files are triaged.

Present a change summary after all approved migrations are finalized.

---

## Output

```
Scanned {D} domains, {N} memory files ({L} total lines).

{Themes table}
{Shape Report — incl. over-size file rows (split candidates)}
{Compatibility report — only when a pre-fab-kit divergence was found}
{Duplicate Coverage table — only when the same topic is covered in 2+ files}
{_unsorted/ Triage — only when _unsorted/ is non-empty}
{Diagnosis}
{Proposal — incl. Link Impact for any move-bearing migration}

Apply this reorganization? (apply all / cherry-pick / skip)
```

After apply: `Reorganization complete: {M} sections moved, {F} files moved, {SF} files split, {MF} files merged, {S} files modified, {C} files/sub-domains created, {L2} links rewritten, {D2} domains split/merged, {UT} _unsorted/ files triaged ({UD} deleted). Indexes regenerated via fab memory-index; no dangling links.`

When compatibility remediation ran, also report: `Compatibility migration: {T} tombstone rows relocated to _shared/removed-domains.md, {B} files backfilled with description: frontmatter (via docs-hydrate-memory sub-agent).` When the user declined remediation: `Compatibility findings reported; tree left intact (no files mutated).`

If no changes needed: `Current structure is well-organized — no reorganization needed.` (and the Shape Report shows all folders within bounds).

### Completion chain → `/docs-distill-memory`

After the completion output above (regardless of whether any migration ran — the composition order is *structure first (reorg), prose second (distill)*), emit a `Next:` line pointing at `/docs-distill-memory` so the fixed reorg → distill order is self-guiding with zero new command surface. This is the other half of the bidirectional chain — distill already points back at reorg in its own `Next:` line.

**Reuse the Step 1 call — no second survey.** Reorg already runs a single `fab memory-index --check --json` (Step 1, one call feeding three consumers). Its `warnings[]` array already carries the `narration-density` and description-tier findings, so the chain reuses **that call's output** — it does **not** run a second `fab memory-index` call. The counts therefore reflect the **Step 1 pre-migration snapshot** — best-effort: approved migrations (moves, `split-file`/`merge-file`, and especially the compatibility description-backfill) can make them stale by completion; that is fine, because distill re-surveys at entry. Aggregate the flagged files with **distill's survey rule** so the two skills' counts agree:

- Count four finding kinds — `malformed[]` `description-change-id` + `description-over-cap` (blocking) and `warnings[]` `description-length` (501–1000 advisory) + `narration-density`.
- A file with **multiple findings counts once** (dedupe by `path`); a **sub-domain file rolls up to its domain** (first path segment under `docs/memory/`).
- **Re-apply the distillation exclusion set** — drop any finding whose path is an `index.md` or `_shared/removed-domains.md` before counting.

Then emit, when N ≥ 1 flagged files across M domains (listed **first**, before the other options):

```
Next: /docs-distill-memory (N files flagged across M domains), or /fab-new
```

When N = 0 (nothing flagged), omit the count-bearing pointer and emit the normal completion `Next:` (e.g. `Next: /fab-new`).

**Older-binary fallback (graceful degradation).** On the older-binary path (no `warnings[]` machine surface — the same fallback Step 1 already handles, with its upgrade warning), the counts are unavailable, so **omit them**: emit a plain pointer `Next: /docs-distill-memory, or /fab-new` (or the normal `Next:` line) — never a fabricated `(N files flagged …)` count. This follows the `_preamble` § Next Steps Convention skill-file-wins carve-out (reorg defines its own completion ending).

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `docs/memory/index.md` missing | Abort: "Run /fab-setup first." |
| No memory domains or files besides indexes | Abort: "Nothing to reorganize." |
| File write/move fails during apply | Report error, roll back that migration, continue |
| Content verification fails | Warn, show missing heading, ask to proceed |
| `fab memory-index` unavailable (older binary) | Warn; fall back to hand-updating affected `index.md` files (legacy path) and tell the user to upgrade `fab` |
| `fab memory-index --check --json` loss tiers / `--json` / `warnings` key unavailable (older binary) | See Step 1 older-binary fallback (legacy prose detection for `losses[]`; read-all-files line counts for `file-size`; direct folder listing for `_unsorted/`) + upgrade warning |
| `split-file` with no dominant topic AND un-anchored inbound links (ambiguous retarget) | Take the abort escape (Step 5 Verify item): roll back that migration, regenerate indexes, continue with the rest |
| `_unsorted/` triage `delete` proposal | Apply **only after explicit per-file confirmation**; never batch-delete staged files. `move` proposals ride the normal per-migration apply |
| Broken bundle-relative link remains after a move | **Hard block** — report the dangling `](/…)` link; do not finalize that migration until it is rewritten. If it cannot be rewritten, take the abort escape defined in Step 5's Verify item (apply item 5): roll back that migration, regenerate indexes, continue with the rest |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Requires active change? | No |
| Idempotent? | Yes — a balanced tree proposes nothing; re-running `fab memory-index` is byte-stable; an already-converted tree re-runs as a no-op (no duplicate tombstone rows in `removed-domains.md`, backfill skips frontmatter-present files) |
| Modifies memory files? | Yes — moves + bundle-relative link rewrites, only with explicit confirmation. Moved files keep their FKF frontmatter (`type: memory` + `description:`) verbatim. On compatibility approval also authors the ONE mechanical file `docs/memory/_shared/removed-domains.md` (tombstone relocation); per-file `description:` synthesis is delegated to the backfill sub-agent, not authored here (Step 5 item 1) |
| FKF-aware moves? | Yes — links are **bundle-relative** (`](/{domain}[/{sub}]/{file}.md)`, FKF §7); a move rewrites only links whose bundle path changes (far fewer than relative links would), preserves the moved file's `type: memory` + `description:` frontmatter, and stamps `type: memory` on any genuinely new topic file |
| Requires config/constitution? | No |
| Is the memory rebalancer? | Yes — supersedes any separate `/fab-rebalance-memory`; shape diagnosis (folder **and** file granularity) + split/merge/flatten + `split-file`/`merge-file` + the file-moving apply path live here |
| Splits over-size files? | Yes — the Shape Report flags a topic file over ~400 lines / ~15KB (from `warnings[]` `file-size`; older-binary ⇒ read-pass line counts) as a `split-file` candidate, proposed only for a ≥2-topic file; a long-but-cohesive file is reported, not split. Bodies move verbatim (restyling is `/docs-distill-memory`'s job) |
| Detects duplicate coverage? | Yes — flags the same topic in 2+ files (`## Duplicate Coverage` table); remediation is a `merge-file` (or `move-section` for partial overlap), with a cross-reference to the open single-sourcing seam audit (not scope) |
| Triages `_unsorted/`? | Yes — per-file `move`-to-domain (default) or `delete` (explicit per-file confirmation) for stale ephemera; `_unsorted/` keeps its width/depth exemption (never split/merged/flattened). Signal: `warnings[]` `unsorted-nonempty`; older-binary ⇒ direct folder listing |
| Orchestrates a pre-fab-kit compatibility migration? | Yes — mechanical detection via `fab memory-index --check --json` (exit 2 + `losses[]`: `description`/`tombstone`/`grouping`; older-binary ⇒ legacy prose); on approval runs the strict-order remediation (relocate tombstones → backfill sub-agent → regenerate once) per Step 5's Compatibility orchestration. Decline = report and stop, no mutation |
| Dispatches sub-agents? | Yes — `/docs-hydrate-memory` backfill mode (general-purpose sub-agent, standard subagent context) for per-file description synthesis during compatibility orchestration |
| Link rewriting | Skill-driven (the agent edits links per the Link Impact list) — NOT a `fab` subcommand |
| Indexes hand-edited? | No — regenerated by `fab memory-index` (domain + sub-domain tiers) |
| Chains to `/docs-distill-memory`? | Yes — completion emits `Next: /docs-distill-memory (N files flagged across M domains)` (N ≥ 1) reusing the Step 1 `--check --json` `warnings[]` (no second call, distill's four-kind aggregation); the fixed structure-then-prose order made self-guiding. Older-binary ⇒ plain pointer, no counts |
