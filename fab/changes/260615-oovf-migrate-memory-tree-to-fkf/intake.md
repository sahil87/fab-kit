# Intake: FKF Change 4/4 — Migrate docs/memory/ Tree to FKF + FKF-Aware Reorg Skills

**Change**: 260615-oovf-migrate-memory-tree-to-fkf
**Created**: 2026-06-16

## Origin

> Backlog `[oovf]` (2026-06-15): "FKF Change 4/4 (refactor, CUTOVER — depends on Change 2 + Change 3, LAND LAST): Migrate the existing docs/memory/ tree to FKF + make reorg skills FKF-aware. … Tasks: (a) one-time conversion (migration file or a docs-reorg-memory orchestration pass) that strips the `## Changelog` table from all ~20 memory files, converts each existing changelog ROW into a seed log.md entry (DECISION b: preserve existing rows faithfully rather than collapsing pre-FKF history to git+slug — this is why we chose C-lite over pure-B; rich rows have no live .status.yaml to source from), converts all memory↔memory links from relative to bundle-relative (/...), ensures `type: memory` on every file, regenerates indexes + logs once; (b) src/kit/skills/docs-reorg-memory.md + docs-reorg-specs.md — handle FKF frontmatter during file moves, optionally add the generated-index borrow for docs/specs/ (fkf.md §9 optional — confirm with owner); (c) SPEC-docs-reorg-memory.md + SPEC-docs-reorg-specs.md mirrors. Verify no dangling links post-migration. Spec: docs/specs/fkf.md §10. Depends-on: Change 2 + Change 3. Context PR #419."

One-shot invocation via `/fab-new oovf`. This is the **final** change (4 of 4) in the FKF adoption arc defined by `docs/specs/fkf.md`. The three prior changes are landed/cherry-picked on this branch:

- **Change 1 (5943)** — `.status.yaml` `summary:` field + migration — SHIPPED (`004aabba`, `feat: Add .status.yaml summary field + 2.4.2-to-2.5.0 migration (#421)`).
- **Change 2 (bmzo)** — `fab memory-index` emits per-folder `log.md` (C-lite) + stamps FKF frontmatter — SHIPPED (`02f3ab28`, `#422`).
- **Change 3 (8fr5)** — doc-skill prose authors FKF / stops per-file changelog writes — cherry-picked onto this branch (`17e80668 operator: cherry-pick 8fr5 dependency`, content commit `de7567a6`).

Both dependencies are stable here, so the CUTOVER can proceed. The format machinery now exists; this change moves the **existing data** onto it and closes the last skills (`docs-reorg-*`) that were not touched by Change 3.

## Why

**Problem.** The FKF spec (`docs/specs/fkf.md`) and its tooling are live (Changes 1–3), but the *actual* `docs/memory/` tree is still pre-FKF and inconsistent with what the tooling now produces:

- **20 of 20** memory files still carry a per-file `## Changelog` table (FKF §3.3 removes these — change history must live in the generated per-folder `log.md`).
- Only **3 of 20** files carry the required `type: memory` frontmatter (Change 3's example writes touched a few; the other 17 lack it — FKF §3.1 requires it on every file).
- memory↔memory cross-links are still **relative** (`](../runtime/operator.md)`) rather than the FKF §7 **bundle-relative** (`](/runtime/operator.md)`) form. There are dozens of these (sampled across `pipeline/`, `_shared/`, `memory-docs/`).
- The 5 generated `log.md` files exist (Change 2's `fab memory-index` produced them) but their entries are **git-commit-subject projections** for all pre-`summary:` history — the rich pre-FKF `## Changelog` prose (651 data rows across the 20 files) is *not yet* represented in the logs.

**Consequence if not fixed.** A partially-migrated tree degrades gracefully (OKF/FKF permissive model — §10 closing note), so nothing *breaks*. But: (1) the tree contradicts the spec and the skills that now author against it, so every future hydrate write produces FKF-shaped files sitting next to 17 non-FKF ones; (2) the `docs-reorg-*` skills still rewrite *relative* links and are unaware of FKF frontmatter, so a future reorg move would re-introduce relative links and could drop `type:`/`description:` on moved files; (3) the 651 rows of curated changelog prose — real archaeology value ("where did `cssMarker` go?") — are stranded in per-file tables that the spec says to remove, with no path into the logs once removed.

**Approach over alternatives.** The arc deliberately split the contract (the spec, Change 0/#419) from its implementations (Changes 1–4), and front-loaded the *machinery* (1–3) before the *data cutover* (4) so the migrated tree matches exactly what the now-FKF-aware skills produce going forward. The one genuinely-debated sub-decision — how to treat pre-FKF history — was already resolved in the backlog as **DECISION b: preserve the existing rich rows faithfully** rather than collapse them to a git+slug projection. This is the stated reason C-lite (descriptive line per change) was chosen over pure-B (slug-only git projection): a slug-only projection loses the what-changed signal, and pre-FKF rows have **no live `.status.yaml` `summary:`** to regenerate from — so the only way to preserve them is to capture them once, at cutover, as seed log content. This change is where that capture happens.

## What Changes

This change has three task groups, mirroring the backlog: **(a)** one-time data conversion of the tree, **(b)** make the two `docs-reorg-*` skills FKF-aware, **(c)** sync the two SPEC mirrors. The whole thing must end with **zero dangling memory↔memory links**.

### (a) One-time data conversion of `docs/memory/`

A single idempotent conversion (per Constitution III) that takes the tree from pre-FKF to fully-FKF. Five mechanical sub-parts (FKF §10 enumerates the same list):

1. **Strip `## Changelog` from every memory file.** All 20 files: `docs/memory/{_shared,distribution,memory-docs,pipeline,runtime}/*.md` (excluding `index.md`/`log.md`). The section starts at the `## Changelog` heading and runs to EOF (it is the trailing section in every file — verified on `memory-docs/hydrate-specs.md`). Remove the heading and its table.

2. **Convert each existing changelog ROW into a seed `log.md` entry** (DECISION b — the load-bearing, design-tension sub-part). Each row today looks like:

   ```
   | 260612-d9rs-docs-reality-sweep | 2026-06-12 | **No-target branch added** … |
   ```

   and must become a `log.md` entry under that date, newest-first, in the FKF §6.2 format:

   ```markdown
   ## 2026-06-12
   - **Update** [hydrate-specs](/memory-docs/hydrate-specs.md) — No-target branch added … (260612-d9rs)
   ```

   The change-id in parens is derived from the row's `Change` column prefix (`260612-d9rs-…` → `d9rs`, or keep the full `YYMMDD-XXXX-slug` if that is what the existing generated logs use — see Open Questions). The descriptive line is the row's Summary cell (which may be multi-line / contain bold markup). **651 data rows** across 20 files feed this (largest: `pipeline/execution-skills.md` and `distribution/kit-architecture.md` at 110 rows each).

   **The design tension (RESOLVED at intake — owner chose the seed-merge mechanism):** `fab memory-index` is the **single, byte-stable writer** of `log.md` (FKF §5/§6; `--help` confirms "Output is byte-stable / idempotent"; today it has **no seed-injection input**). It assembles `log.md` purely from (git history) ⋈ (`.status.yaml summary:`). Pre-FKF changes have **no live `.status.yaml`** (they are shipped/archived), so there is nowhere for the *current* generator to read the rich row from. The resolution: **teach `fab memory-index` a seed-merge** — the generator reads an existing hand-seeded `log.md` (or a sidecar seed file) and *merges* seed entries beneath the git-projected ones, preserving them on every regen.
   <!-- clarified: Q1 — owner chose mechanism (i) generator seed-merge over (ii) archived-summary back-fill and (iii) guarded hand-seed; faithfulness prioritized, scope deliberately expanded into Go -->

   **Scope consequence:** this re-opens the Change-2 Go package (`src/go/fab/internal/memoryindex/`) — it is no longer a skills+data-only change. The work: (1) add a seed-read input (existing `log.md` parse, or a dedicated seed file format); (2) merge-beneath-projected logic preserving seed entries idempotently across regens; (3) extend the loss tiers (`loss.go` `--check`/`--json`) so a seed-merge run still classifies clean/benign and never reports the preserved seed as destructive loss; (4) tests for the merge path; (5) update `_cli-fab.md` § fab memory-index. The chosen mechanism is the most faithful to DECISION b (preserves all 651 rows verbatim under their real dates) and keeps the single-writer/idempotency discipline intact (the generator stays the writer; the seed is an *input*, not a hand-edit of the output).

3. **Convert memory↔memory links from relative to bundle-relative** (FKF §7). Every `](../{domain}/{file}.md)` or `](./{file}.md)` or `]({file}.md)` pointing at another memory file becomes `](/{domain}/{file}.md)`. Links *out* of the bundle (to `src/`, `docs/specs/`, README, external URLs) stay repo-relative/absolute-URL — unchanged. Note these conversions also appear **inside the changelog Summary cells** being migrated to logs, so the link rewrite must run on the seed entries too (the sampled rows contain `[schemas.md](schemas.md)`, `[pane-commands.md](../runtime/pane-commands.md)`, etc.).

4. **Ensure `type: memory` on every file** (FKF §3.1). Add it to the 17 files missing it (alongside the existing `description:`); leave the 3 that already have it. Reserved files (`index.md`, `log.md`) are exempt.

5. **Regenerate indexes + logs once** via `fab memory-index`, and confirm `--check` returns 0 (clean) or 1 (benign drift only) — never 2 (destructive loss).

**Conversion vehicle (Open Questions Q2):** the backlog offers a choice — "a migration file **or** a `docs-reorg-memory` orchestration pass." A migration file (`src/kit/migrations/`) is the project's standard data-migration vehicle (idempotent, version-gated, applied by `/fab-setup`), but migrations are markdown *instructions* an agent executes, and a 651-row faithful-prose conversion across 20 files is a large agent-driven pass — well-suited to being **driven by this change's own apply stage** with the migration file documenting the steps for *other* projects. The two are not exclusive (the migration file can codify what apply does here). Leaning migration-file-as-record + apply-as-executor; flagging for confirmation.

### (b) Make `docs-reorg-memory.md` + `docs-reorg-specs.md` FKF-aware

- **`src/kit/skills/docs-reorg-memory.md`** (230 lines): the rebalancer moves files between domains/sub-domains and **rewrites the links they break**. Today it rewrites *relative* links (its Link Impact note shows `](../execution-skills.md)` rewrites). Under FKF, memory↔memory links are bundle-relative and **survive a move** (that is the §7 rationale — "reorg rewrites *far* fewer links"). So:
  - The move logic must **preserve FKF frontmatter** (`type: memory` + `description:`) on every moved file — never strip or regenerate it.
  - The Link Impact analysis must understand that bundle-relative links to a *moved* file still need rewriting (the path after `/` changes when a file changes domain/sub-domain), but sibling-relative breakage largely disappears. Update the Link Impact note's examples and the rewrite rule to the bundle-relative form.
  - Stub-before-index (FKF §5) already governs new sub-domain index creation; confirm the skill's split/merge flow writes the `description:`-only stub before `fab memory-index`.
- **`src/kit/skills/docs-reorg-specs.md`** (122 lines): specs are **out of FKF scope** (§9, Constitution VI — no frontmatter, human-curated). The optional generated-index borrow for `docs/specs/index.md` (§9) was **declined by the owner** — **skip it**.
  <!-- clarified: Q3 — owner chose minimal scope; no specs-index generator, specs stay hand-curated -->
  The only change here is a **guard**: `docs-reorg-specs` must NOT stamp FKF frontmatter (`type:`/`description:`) on spec files during moves, keeping specs frontmatter-free per Constitution VI. No new generator, no flip of `memory-docs/specs-index.md`'s existing "no specs-index generator" note. This keeps the CUTOVER focused on `docs/memory/`.

### (c) SPEC mirrors (Constitution: skill changes MUST update the SPEC-*.md mirror)

- `docs/specs/skills/SPEC-docs-reorg-memory.md` — mirror the (b) changes.
- `docs/specs/skills/SPEC-docs-reorg-specs.md` — mirror the (b) changes (likely minimal, tracking whatever Q3 resolves to).

Both mirror files already exist.

### Closing verification

- **Zero dangling memory↔memory links** post-migration (backlog requirement). Mechanically: every `](/...)` link resolves to an extant file under `docs/memory/`. The `fab memory-index --check` tombstone tier (exit 2 `tombstone` loss category) catches index-row dangling targets; a separate sweep is needed for body-prose links.
- `fab memory-index --check` exits 0 or 1 (not 2).
- `fab preflight` / kit still loads cleanly.

## Affected Memory

This change edits the memory **tree shape/frontmatter** mechanically (every file) and edits the two reorg **skills** — so the skills' own memory docs need updating:

- `memory-docs/templates.md`: (modify) — the memory-file format/template no longer carries `## Changelog` (existing files cut over). This file ALSO homes `docs-reorg-memory`'s file-moving rebalancer + compatibility orchestration ("Memory Tree Shape" per the domain index), so the reorg FKF-awareness from task (b) is recorded here: frontmatter-preserving moves, bundle-relative link rewrites, optional specs-index borrow per Q3.
- `memory-docs/hydrate.md`: (modify) — record that the tree is now fully FKF (all 20 files `type: memory`, no `## Changelog`, bundle-relative links); the backfill mode and reorg compatibility orchestration prose stay consistent with the post-cutover state.
- `memory-docs/specs-index.md`: (modify, **only if Q3 → add specs-index borrow**) — this file already carries the explicit "no specs-index generator / no-symmetry note" for `docs-reorg-specs`; it would flip only if the optional §9 borrow is approved.
- Every memory file under `_shared/`, `distribution/`, `memory-docs/`, `pipeline/`, `runtime/`: (modify) — mechanical frontmatter + link + changelog-strip conversion (task a). These are content edits to existing files, not memory *spec*-behavior changes, but they ARE memory-tree edits and will surface in the regenerated logs.

> Note: confirmed against `docs/memory/memory-docs/index.md` — there is **no dedicated `reorg.md`**; the reorg rebalancer lives in `templates.md` (Memory Tree Shape) and `hydrate.md` (compatibility orchestration). Routing targets above reflect that.

## Impact

- **Code**: `src/kit/skills/docs-reorg-memory.md`, `src/kit/skills/docs-reorg-specs.md`, **and** `src/go/fab/internal/memoryindex/` (`memoryindex.go`, `loss.go`, `indexparse.go` + tests) — the seed-merge mechanism (Q1 resolution) re-opens the Change-2 Go package. This is the largest piece of the change and the main scope/risk item.
- **Migrations**: possibly a new `src/kit/migrations/*.md` (or fold into the existing `2.4.2-to-2.5.0.md`) per Q2 — still open.
- **Specs**: `docs/specs/skills/SPEC-docs-reorg-memory.md`, `docs/specs/skills/SPEC-docs-reorg-specs.md` (mirrors, Constitution-required). `docs/specs/fkf.md` §10 is the contract — read, not edited (it already describes this migration).
- **Data**: all 20 `docs/memory/**/*.md` topic files (frontmatter, links, changelog strip); all `log.md` files (seed entries); all `index.md` files (regenerated).
- **`_cli-fab.md`**: § fab memory-index MUST be updated — the seed-merge changes the generator's documented behavior (and the loss-tier semantics).
- **Version skew caveat**: `fab status set-summary` (Change 1) and the `log.md`/FKF-frontmatter generator behavior (Change 2) are in `src/kit/VERSION 2.5.0`, but the **installed binary is `fab 2.4.2`** (`fab status set-summary --help` is not recognized by the installed binary). Apply/hydrate steps that invoke these must use the 2.5.0-source behavior — verify the binary in use before relying on Change 1/2 CLI surfaces. (Recurring pattern — see memory `score-binary-source-version-skew`.)
- **Idempotency (Constitution III)**: the conversion MUST be safe to re-run — a second pass over an already-FKF file must be a no-op (no double-stripped sections, no doubled `type:`, no re-relativized links).

## Open Questions

- **Q1 — RESOLVED at intake → teach the generator a seed-merge.** Pre-FKF rows become seed `log.md` entries via a new seed-read input to `fab memory-index` that merges seed entries beneath the git-projected ones, idempotently. This re-opens `src/go/fab/internal/memoryindex/` (the most faithful option; deliberately expands scope into Go).
- **Q3 — RESOLVED at intake → skip the specs-index borrow.** `docs-reorg-specs.md` gets only a guard against stamping FKF frontmatter on spec moves; no `fab specs-index` generator. Specs stay human-curated (Constitution VI / FKF §9).
- **Q2 (open, Tentative default recorded):** Conversion vehicle — a `src/kit/migrations/` file, a `docs-reorg-memory` orchestration pass driven by this change's apply, or both (migration file documents what apply executes)? Default leaning: migration-as-record + apply-as-executor. Resolve at `/fab-clarify` if the default is wrong.
- **Q4 (open, Tentative default recorded):** Change-id form in seed log entries — bare 4-char id `(d9rs)` (FKF §6.2 example text) or full `(260612-d9rs-slug)`? Mechanical; align with whatever Change-2's generator already emits (verify at apply).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Both dependencies (Change 2 `bmzo`, Change 3 `8fr5`) are stable on this branch; CUTOVER may proceed | Verified in git log: `02f3ab28` (Change 2), `de7567a6`+`17e80668` cherry-pick (Change 3). The backlog's `Depends-on` gate is satisfied | S:95 R:90 A:100 D:95 |
| 2 | Certain | DECISION b stands — preserve existing 651 changelog rows faithfully as seed log entries, do NOT collapse to git+slug | Stated verbatim in the backlog as a settled decision and is the cited reason C-lite was chosen over pure-B; not re-litigated here | S:100 R:70 A:100 D:100 |
| 3 | Certain | FKF §10 is the migration contract and §3.1/§3.3/§7 the per-file rules; spec is read-not-edited | `docs/specs/fkf.md` already enumerates the exact 6-part migration; Constitution VI forbids tooling editing specs | S:100 R:85 A:100 D:100 |
| 4 | Confident | The conversion must be idempotent (re-run-safe): no double-strip, no doubled `type:`, no re-relativized links | Constitution III (Idempotent Operations) is a MUST; every fab skill/migration already honors it | S:80 R:75 A:100 D:90 |
| 5 | Confident | Scope is all 20 memory topic files + 5 log.md + all index.md; specs untouched except the two SPEC mirrors (and Q3) | FKF §9 + Constitution VI keep specs out of scope; grep confirms exactly 20 files carry `## Changelog` | S:85 R:75 A:95 D:85 |
| 6 | Confident | `docs-reorg-memory` link-rewrite logic moves from relative to bundle-relative form; moves must preserve FKF frontmatter | FKF §7 rationale is explicitly "reorg rewrites far fewer links"; the skill's current Link Impact note rewrites relative links | S:80 R:65 A:90 D:80 |
| 7 | Tentative | Conversion vehicle = migration file as the record + this change's apply stage as the executor (Q2) | Backlog offers either/both; migrations are agent-executed markdown, and a 651-row prose conversion fits apply better than a pure declarative migration — but not yet confirmed <!-- assumed: migration-file-documents + apply-executes; confirm at clarify (Q2) --> | S:55 R:55 A:60 D:50 |
| 8 | Tentative | Seed-log change-id form = bare 4-char id `(d9rs)` per FKF §6.2 examples (Q4) | §6.2 shows `(260613-l3ja)` full form in one example and the spec text says "the `(change-id)`"; mild ambiguity, mechanical to flip <!-- assumed: align to whatever Change-2 generator emits; verify at apply --> | S:50 R:90 A:70 D:55 |
| 9 | Certain | Q1 RESOLVED → seed pre-FKF rows by teaching `fab memory-index` a seed-merge (reads seed entries, merges beneath git-projected, idempotent) | Asked — owner chose mechanism (i) over archived-summary back-fill and guarded hand-seed; faithfulness prioritized | S:95 R:55 A:100 D:100 |
| 10 | Certain | Q3 RESOLVED → skip the optional specs-index borrow; `docs-reorg-specs` gets only a "don't stamp FKF frontmatter on spec moves" guard | Asked — owner chose minimal scope; specs stay human-curated (Constitution VI / FKF §9) | S:90 R:80 A:100 D:100 |
| 11 | Confident | Q1's resolution re-opens `src/go/fab/internal/memoryindex/` — this is now a skills + data + Go change, not skills+data only | Direct consequence of assumption 9; the seed-merge lives in the Change-2 Go package + tests + loss tiers + `_cli-fab.md` | S:90 R:50 A:85 D:80 |

11 assumptions (5 certain, 4 confident, 2 tentative, 0 unresolved).
