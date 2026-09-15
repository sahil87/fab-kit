# Intake: Distill/Reorg Extensions — Drain the Accretion Debt

**Change**: 260718-dsrx-distill-reorg-extensions
**Created**: 2026-07-19

## Origin

Backlog ID `[dsrx]` via `/fab-new dsrx` (one-shot invocation, no prior conversation). Change C of the three-change series from the 2026-07-18 cross-repo memory audit (loom / run-kit / idea) — after Change A `[mxgu]` (memory-index guards, PR #501, merged) and Change B `[wrct]` (memory writer contract, PR #500, merged). Backlog entry verbatim:

> [dsrx] 2026-07-18: Change C: distill/reorg extensions — drain the accretion debt (cross-repo memory audit 2026-07-18; NOTE: the no-arg survey mode + dynamic Next line already SHIPPED via 260718-ukpf / PR #498 — NOT in scope here). (1) DISTILL new removal classes in docs-distill-memory Step 1 taxonomy: strip change-id suffixes from headings (run-kit: 60 such headings); dedupe literal duplicate headings/blocks (run-kit ui-patterns.md carries two duplicated heading pairs at lines 382/388 + 384/392); rewrite Design-Decisions-changelog-bullets to proper Decision/Why/Rejected form (idea ci+release pipeline.md); RELOCATE embedded operational TODOs to fab/backlog.md — relocate, never delete. (2) REORG owns FILE-splitting: extend docs-reorg-memory Shape Report to take over-long files as split candidates — today reorg splits domains (≥8 cohesive files) and distill rewrites in place, so nobody owns "this 2,000-line file is five topics" (loom test-suite-reference.md 2,757 lines / 83 headings; run-kit ui-patterns.md 2,033 lines = 45% of its corpus; idea structure.md 60KB = half its corpus; distilling without splitting still leaves mega-files). (3) REORG passes: duplicate-coverage detection (same topic in 2+ files — loom mock-infrastructure.md vs msw-mock-infrastructure.md, right-panel-sections.md in two domains; ties to the open single-sourcing seam audit) and _unsorted/ staging triage (loom: 4 stale infra-505 session notes parked since May). (4) OPTIONAL consolidation: switch the #498 survey narration heuristic from agent-side grep to consuming the [mxgu] warning output — one canonical signal source. DEPS: after BOTH [mxgu] (consumes its per-file-size / _unsorted / narration-density warnings) and [wrct] (distill reports cite the deployed FKF rule for every proposed rewrite — the new removal classes need their §3.3 rules written first). Surfaces: docs-distill-memory + docs-reorg-memory skill sources + SPEC-*.md mirrors + docs/specs/skills.md aggregate + the memory-docs domain memory files.

Dependency check at intake time: both deps are merged on `main` and present on this branch — `[wrct]` at 021d32b0 (PR #500) and `[mxgu]` at af95e001 (PR #501). The FKF §3.3 rules the new distill classes cite (heading change-id ban, no-operational-TODOs rule, changelog-bullet ban in Design Decisions) are verified present in the shipped extract `src/kit/reference/fkf.md`, and `fab memory-index --check --json` already emits the `warnings[]` machine surface (`narration-density` / `file-size` / `unsorted-nonempty` / `broken-link`) that `_cli-fab.md` explicitly documents as "the machine surface [dsrx] consumes instead of parsing stderr".

## Why

1. **The pain point.** The 2026-07-18 cross-repo audit showed the memory corpus disease is **accretion and form, not truth** (staleness 0/18 spot-checks). `[mxgu]` shipped *detection* (the memory-index warning meters) and `[wrct]` shipped *prevention* (the writer contract — hydrate rewrites, never appends). But the *remediation* skills that drain existing debt have gaps:
   - `/docs-distill-memory`'s Step 1 taxonomy names five defect classes; it does NOT name change-id heading suffixes (run-kit: 60 such headings), literal duplicate blocks (run-kit ui-patterns.md carries two byte-duplicated heading pairs), changelog-bullet Design Decisions (idea ci+release pipeline.md), or embedded operational TODOs (idea: a gh-secret-delete follow-up living in a memory body). A distill run over those files leaves all four in place.
   - Nobody owns file splits: `/docs-reorg-memory` splits *domains* (folder width/depth) and distill rewrites *in place*, so a 2,757-line five-topic mega-file (loom test-suite-reference.md) survives both skills. Distilling without splitting still leaves mega-files.
   - Nobody detects duplicate coverage across files (loom `mock-infrastructure.md` vs `msw-mock-infrastructure.md`) or triages `_unsorted/` staging (loom: 4 stale infra-505 session notes parked since May — `_unsorted` is bounds-exempt so reorg never even lists it).
2. **The consequence of not fixing.** The audit's debt persists indefinitely: `[mxgu]`'s warnings fire on every regen with no skill that acts on them, and the survey heuristic in distill duplicates (in agent-side grep) signal logic the Go binary now computes canonically — two implementations of the same defect classes that will drift.
3. **Why this approach.** Extend the two existing remediation skills rather than adding a new one: distill already owns in-place prose rewrites (the four new classes are exactly that, plus one relocation), and reorg already owns structure moves + link rewrites + the propose-then-apply gate (file splits/merges are structure). Item 4 consolidates on the `[mxgu]` machine surface because `_cli-fab.md` already names `[dsrx]` as its intended consumer.

## What Changes

### 1. `/docs-distill-memory` — four new removal classes

Extend Step 1 (identify), Step 2 (per-file report), and Step 4 (apply) with four classes, each citing the deployed rule in `$(fab kit-path)/reference/fkf.md` §3.3 (shipped by `[wrct]`):

- **(a) Change-id heading suffixes.** A heading carrying a change-id token — `### Dispatch States (xu0k)`, `## Foo — 260718-mxgu` — has the token stripped, keeping the heading text (§3.3: "a heading is `## Dispatch States`, never `### Dispatch States (xu0k)`"). Token recognition follows the registry-gated posture: a full `YYMMDD-XXXX-slug` token always matches; a bare 4-char id only when it is registry-plausible (the propose-then-apply human gate covers residual false positives). If the stripped suffix carried provenance worth keeping, it becomes a trailing `(change-id)` citation in the section body — allowed provenance, per the existing keep-list.
- **(b) Literal duplicate headings/blocks.** **Byte-identical** duplicated heading pairs/blocks (run-kit ui-patterns.md lines 382/388 + 384/392) → remove the later duplicate. **Near-duplicates are flagged in the Step 2 report for manual review, never auto-merged** — content judgment stays with the human gate.
- **(c) Design-Decisions changelog bullets.** A `- **{change-id} — retired X**`-shaped bullet inside `## Design Decisions` (the shape §3.3 bans there): when it encodes a durable decision → rewrite to the four-field entry (**Decision** / **Why** / **Rejected** / *Introduced by* — the change-id moves into *Introduced by* or a trailing citation); when it is pure change history already recorded in `log.md`/git → remove under the existing deletion-safety rule. **Never fabricate rationale**: when Why/Rejected content is not derivable from the bullet or surrounding context, the rewritten entry carries only the fields that exist (Decision + *Introduced by*).
- **(d) Embedded operational TODOs → RELOCATE to `fab/backlog.md`, never delete** (§3.3: follow-up work items belong in the project backlog or the originating change folder). Relocation appends a standard backlog entry:

  ```markdown
  - [ ] [{fresh-4char-id}] {YYYY-MM-DD}: {TODO text} (relocated from docs/memory/{domain}/{file}.md by /docs-distill-memory)
  ```

  If `fab/backlog.md` does not exist (user repos), create it with a minimal `# Backlog` header. Relocation honors the Step 3 approval unit: a file the user skips or cherry-picks away keeps its TODOs (no orphaned relocations).

Step 2 report lines gain matching entries (e.g. `- strip change-id heading suffixes: 3 headings`, `- dedupe byte-identical blocks: 2 (near-duplicates flagged: 1)`, `- rewrite DD changelog bullets: 2`, `- RELOCATE TODOs → fab/backlog.md: 1`), and the completion line's counters extend accordingly. The survey heuristics need no new signal: heading change-id tokens already count toward the narration-marker density (change-id occurrences in the body), and duplicates/TODOs are full-read findings — the survey is a ranking heuristic, not an exhaustive classifier.

### 2. `/docs-reorg-memory` — file-splitting (Shape Report + Migration Map)

Today the Shape Report is folder-only (width / depth / floor). Extend it with **file rows**:

- **Detection**: any topic file exceeding `[mxgu]`'s thresholds (~400 lines OR ~15KB) is a split *candidate*, sourced from the same `fab memory-index --check --json` call Step 1 already makes — `warnings[]` kind `file-size` (carries `count` = lines and `bytes`). Older-binary fallback: measure during the read-all-files pass (Step 1 already records approximate line counts).
- **Reactive posture preserved**: a flagged file is *proposed* for splitting only when its heading clusters show ≥2 genuine topics; a long-but-cohesive file is reported (e.g. `⚠ over size — long but cohesive; no split proposed`) and left alone. Same soft-SHOULD stance as the folder bounds.
- **New Migration Map `Kind`: `split-file`** — fan one multi-topic file into ≥2 topic files in the same domain/sub-domain (parallel to `split-domain` at file granularity). Each new file gets `type: memory` + a fresh change-id-free `description:` (same authoring rule as split-domain's new files). Body content moves **verbatim** — restyling prose remains distill's job, preserving the skills' division of labor. The original path is kept for the dominant topic when one exists, else removed. A split that pushes folder width past ~12 can chain into the existing `split-domain` flow in the same proposal. Newly split files target the existing "keep files under ~300 lines" constraint (detection flags at 400/15KB; authoring aims at ~300).
- **Link Impact extends to `split-file`**: inbound bundle-relative links to the split file are retargeted per destination — an **anchored** link (`#heading`) follows the file its heading moved to; an **un-anchored** link retargets to the dominant-topic file. Ambiguity (no dominant topic, un-anchored inbound links) → the existing abort escape (roll back that migration, regenerate, continue).

### 3. `/docs-reorg-memory` — duplicate-coverage detection + `_unsorted/` triage

Two new analysis passes feeding the existing report → confirm → apply flow:

- **Duplicate-coverage detection** (new step alongside theme identification): flag the same topic covered in 2+ files. Signals: near-identical filenames/descriptions (loom `mock-infrastructure.md` vs `msw-mock-infrastructure.md`), the same filename in two domains (loom `right-panel-sections.md`), heavy heading overlap. Output: a `## Duplicate Coverage` table (topic / files / evidence / proposed canonical home). Remediation rides the Migration Map: a **new `Kind`: `merge-file`** (move B's unique sections into canonical file A via the move-section machinery, rewrite all inbound links to A, delete the emptied B — parallel to `merge-domain` at file granularity, with Link Impact + the no-dangling-link guard), or plain `move-section` rows for partial overlap. The report notes the tie to the open single-sourcing seam audit (cross-reference, not scope).
- **`_unsorted/` staging triage**: `_unsorted/` keeps its bounds exemption (never split/merged/flattened), but gains a **triage listing** — every staged topic file with a per-file proposal: **`move`** to a named domain (existing kind — the default), or **delete** for stale ephemera whose content is superseded or recorded elsewhere (e.g. session notes for a shipped change), each deletion requiring explicit per-file confirmation. Signal: `warnings[]` kind `unsorted-nonempty`; fallback: direct folder listing. Staging should trend to empty.

### 4. Survey consumes the machine surface (consolidation, backlog item 4 — included)

- **`/docs-distill-memory` Step 0 survey**: replace the agent-side grep heuristics with **one `fab memory-index --check --json` invocation**. Per-domain flagged-file counts aggregate from: `malformed[]` kinds `description-change-id` + `description-over-cap` (blocking class), and `warnings[]` kinds `description-length` (advisory 501–1000 band — see the Go note below) + `narration-density`. A file with multiple findings counts once; a sub-domain file rolls up to its domain (first path segment under `docs/memory/`). Survey output format and auto-pick semantics are unchanged. The check's exit code does not gate the survey (the survey consumes the report; it is not a regen guard — exit 1/2 still surveys).
- **Small additive Go change**: include `KindDescriptionLength` in the JSON `warnings[]` switch (`src/go/fab/cmd/fab/memory_index.go` — today only the four mxgu debt-meter kinds join the array), so the survey's existing over-cap heuristic (the 501–1000 advisory band) has a machine surface instead of a residual agent-side frontmatter check. Additive per the established `warnings[]` contract (existing consumers unaffected). Constitution obligations: Go test update + `_cli-fab.md` § fab memory-index JSON-shape doc update (kind enum + count semantics), with its SPEC mirror.
- **Older-binary fallback**: when `--json` is unavailable or the `warnings` key is absent, the survey falls back to the current grep heuristics verbatim and warns to upgrade `fab` — mirroring `/docs-reorg-memory`'s Step 1 older-binary fallback posture.
- **`/docs-reorg-memory` Step 1**: its existing `--check --json` parse additionally records `warnings[]` (`file-size` → Shape Report file rows; `unsorted-nonempty` → triage pass) — one call feeds compatibility detection and the new passes.

### Out of scope

- The no-arg survey mode + dynamic `Next:` line themselves (shipped via 260718-ukpf / PR #498) — item 4 only swaps the survey's *signal source*.
- `[mxgu]`'s warning thresholds and blocking/advisory classification (shipped; consumed as-is).
- `[wrct]`'s writer rules and FKF §3.3 text (shipped; cited as-is).
- Actually running the remediation over any corpus (this change extends the skills; running them is per-repo operator work).
- The single-sourcing seam audit itself (the duplicate-coverage report cross-references it only).

## Affected Memory

- `memory-docs/distill`: (modify) — four new removal classes in the taxonomy; survey signal source switches to `fab memory-index --check --json` with grep fallback
- `memory-docs/templates`: (modify) — Memory Tree Shape section: reorg rebalancer gains file-level splitting (`split-file`/`merge-file`), duplicate-coverage detection, `_unsorted` triage; JSON `warnings[]` gains the `description-length` kind

## Impact

- **Skill sources**: `src/kit/skills/docs-distill-memory.md`, `src/kit/skills/docs-reorg-memory.md`, `src/kit/skills/_cli-fab.md` (§ fab memory-index `--json` shape)
- **Spec mirrors** (constitution: every changed skill source updates its mirror): `docs/specs/skills/SPEC-docs-distill-memory.md`, `SPEC-docs-reorg-memory.md`, `SPEC-_cli-fab.md`, plus the `docs/specs/skills.md` aggregate — sweep the whole mirror class per `fab/project/code-quality.md` § Sibling & Mirror Sweeps
- **Go**: `src/go/fab/cmd/fab/memory_index.go` (one switch case adds `KindDescriptionLength` to the JSON `warnings[]`) + corresponding test update (constitution: CLI changes ship with tests and `_cli-fab.md` updates)
- **Memory** (at hydrate): `docs/memory/memory-docs/distill.md`, `docs/memory/memory-docs/templates.md`
- **No migration**: no user-data restructuring; the JSON change is additive; no CLI signature changes
- **No behavior change for born-clean trees**: all new classes/passes propose nothing on a corpus without the defects (idempotency, Constitution III)

## Open Questions

*(none — all decision points resolved from the backlog entry, the shipped dep surfaces, and existing skill patterns; see Assumptions)*

## Assumptions

<!-- STATE TRANSFER: This table is the sole continuity mechanism between the intake-stage
     agent and the apply-entry agent (which co-generates plan.md). -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Deps satisfied: build on merged `[mxgu]` (#501) + `[wrct]` (#500); the cited FKF §3.3 rules and the `warnings[]` JSON surface are verified present on this branch | Verified in git history, `src/kit/reference/fkf.md`, and `src/go/fab/internal/memoryindex` at intake time | S:95 R:90 A:95 D:95 |
| 2 | Certain | #498's survey mode + dynamic `Next:` line are out of scope; item 4 only swaps the survey's signal source | Backlog entry states this explicitly | S:95 R:95 A:95 D:95 |
| 3 | Confident | Include backlog item 4 (marked OPTIONAL) — survey consumes the machine surface | `_cli-fab.md` documents `warnings[]` as "the machine surface [dsrx] consumes instead of parsing stderr" — the surface was built for this change | S:60 R:85 A:85 D:75 |
| 4 | Confident | Add `description-length` to the JSON `warnings[]` switch (small additive Go change) rather than keeping a hybrid agent-side length check | "One canonical signal source" is the item's stated goal; the kind constant already exists; `warnings[]` is additive by contract. Rejected: hybrid grep (perpetuates the dual-source drift this item removes) | S:55 R:85 A:80 D:65 |
| 5 | Confident | Survey older-binary fallback: current grep heuristics verbatim + upgrade warning when `--json`/`warnings` unavailable | Mirrors `/docs-reorg-memory`'s established older-binary fallback posture | S:60 R:90 A:85 D:80 |
| 6 | Confident | Two new Migration Map kinds `split-file` / `merge-file`, file-granularity parallels of `split-domain` / `merge-domain`, both with Link Impact + the no-dangling-link guard + abort escape | Cleanest fit to the existing Kind enum and apply machinery; reuses proven link-rewrite rules | S:65 R:80 A:80 D:70 |
| 7 | Confident | `split-file` proposed only for genuine multi-topic files (heading clusters); anchored inbound links follow their heading, un-anchored retarget to the dominant-topic file, ambiguity → abort escape; detection at mxgu's 400-line/15KB, new files target ~300 lines | Preserves reorg's reactive-not-prophylactic posture and existing constraints; abort escape already handles ambiguity | S:60 R:75 A:75 D:65 |
| 8 | Confident | Distill dedupe (class b) auto-removes byte-identical duplicates only; near-duplicates flagged for manual review. Cross-file duplication belongs to reorg's duplicate-coverage pass | Byte-equality is mechanically safe; near-duplicate merging needs content judgment the human gate should see. Within-file = distill (prose), cross-file = reorg (structure) matches the skills' division of labor | S:60 R:75 A:80 D:70 |
| 9 | Confident | TODO relocation format: append `- [ ] [{fresh-id}] {date}: {text} (relocated from {path})` to `fab/backlog.md`, creating it with a minimal header when absent; relocations follow per-file cherry-pick approval | Backlog names the destination; format matches existing `fab/backlog.md` entries; never-delete is the backlog's own constraint | S:70 R:90 A:75 D:75 |
| 10 | Confident | `_unsorted` triage default is `move`-to-domain; `delete` offered only for ephemera superseded/recorded elsewhere, with explicit per-file confirmation; `_unsorted` keeps its bounds exemption | Loom evidence (stale session notes) shows move-only would relocate garbage into curated domains; per-file confirmation + git recoverability bound the risk | S:55 R:55 A:60 D:50 |

10 assumptions (2 certain, 8 confident, 0 tentative, 0 unresolved).
