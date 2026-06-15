# Plan: Docs Reality Sweep

**Change**: 260612-d9rs-docs-reality-sweep
**Intake**: `intake.md`

## Requirements

### Docs: SPEC-hooks rewrite as-shipped

#### R1: SPEC-hooks describes the shipped Go hook system
`docs/specs/skills/SPEC-hooks.md` MUST be rewritten to describe the shipped Go hook system (`fab hook` subcommands registered inline in `.claude/settings.local.json`): the Current Hooks section MUST NOT list deleted shell scripts; shipped Go behavior MUST NOT be presented as a future "Proposed Hook Architecture"; the events table MUST rate UserPromptSubmit as registered. The superseded `fab runtime` proposal, dead yq inventory, stale phase list, and outdated runtime schema MUST be replaced with as-shipped content.

- **GIVEN** the shipped hook surface documented in `src/kit/skills/_cli-fab.md` § fab hook
- **WHEN** SPEC-hooks.md is read after this change
- **THEN** every hook it describes exists in the shipped system, and no shipped behavior is framed as proposed/future work

### Docs: architecture.md rewrite-in-place

#### R2: architecture.md describes the shipped distribution model
`docs/specs/architecture.md` MUST be rewritten in place (file and its specs-index row kept — user-confirmed) to describe the shipped system: the system-cache distribution model (`~/.fab-kit/versions/<version>/kit/`, `fab` router + per-version `fab-go`, brew install) instead of the pre-binary `.kit/` model, and a self-consistent Router Dispatch section.

- **GIVEN** the pre-binary `.kit/` content and the Router Dispatch self-contradiction
- **WHEN** architecture.md is read after this change
- **THEN** it describes the shipped cache/binary distribution model and contains no internal contradiction, and `docs/specs/index.md` still lists it

### Docs: legacy docs truth sweep

#### R3: Legacy docs point-fixes match the shipped 6-stage pipeline
The remaining legacy docs MUST be corrected: `assembly-line.md:121` spec/tasks stage narrative (6 stages, no spec stage since 1.10.0); `overview.md` 4-stage story plus omission of `/git-pr`, `/git-pr-review`, `/fab-proceed`, `/fab-operator`; `glossary.md:49/115` auto-clarify definitions that fab-ff disclaims; `user-flow.md:84/183` "failed is review-only" (review-pr also fails); `templates.md` pre-1.10.0 intake template and `.status.yaml` block missing `ready`/`id`/`issues`/`prs`; `skills.md:583` pre-date-bucketing archive path; `operator.md:11` v8→v9. When a stale claim is fixed, the same claim-class MUST be swept repo-wide within `docs/specs` + `src/kit/skills` (w7dp lesson), excluding §5 verified-clean areas.

- **GIVEN** the enumerated stale claims in report §2 Theme 7b
- **WHEN** each named doc is read after this change
- **THEN** the cited claim matches the shipped system and no same-class instance of that claim remains in docs/specs or src/kit/skills

### Skills: memory-index ownership (Theme 8)

#### R4: Index ownership defined once and self-contradictions resolved
The index-ownership model MUST be stated once and propagated: `description:` frontmatter is the single hand-curated field; a sub-domain index **stub is created BEFORE `fab memory-index` runs** (the stub pattern of `docs-hydrate-memory.md`). Specifically: `docs-reorg-memory.md` Step 5.3/5.4 contradiction resolved via stub-before-index; depth off-by-one between the ≤3 path-segment bound and the report's folder-depth column reconciled; dangling-link hard block gains an abort/rollback escape; `docs-hydrate-memory.md` generate mode gains placement rules (target path, domain creation, index stub, shape bounds) and sub-domain index stubs are instructed; the memory-index tier description includes sub-domain indexes in all 3 locations (incl. `fab-continue.md` hydrate step); `docs-hydrate-specs.md` gains the missing no-target-spec branch and Step 5/6 token alignment; `docs-reorg-specs.md` gains a reserved-path exemption for constitution-pinned SPEC mirrors and defines subfolder recursion. These are documentation-of-correct-behavior fixes — no new runtime mechanisms.

- **GIVEN** docs-reorg-memory Step 5.3 editing an index that Step 5.4 both generates and forbids editing
- **WHEN** the skill prose is followed after this change
- **THEN** the sub-domain index stub (frontmatter `description:` only) is created before `fab memory-index` runs, the generator round-trips it, and no step instructs hand-editing generated rows

#### R5: Quick-win Bundle A — stale pointers, counts, wording fixed
Every Bundle A item (report §3, full list) MUST be fixed in its `src/kit/skills/` source: fab-operator.md (`§5`→`§6`, status-frame count vs entries + schema-invalid `gmail-deploys` source, stale branch_map retention clause); fab-proceed.md (pointer → Bypass Notes, `sort -r` same-day tiebreak); fab-continue.md (dangling "Review Behavior" heading pointer); git-pr-review.md (unconsumed `node_id` in jq projection + SPEC, `--tool` header rewording); docs-hydrate-specs.md (prompt/handler token alignment); fab-help.md Purpose; internal-retrospect.md missing H1; _cli-external.md `--reuse` requires `--worktree-name`; _generation.md drop "auto-clarify"; fab-clarify.md protocol example citing a removed flow; fab-discuss.md stage-derivation rule; fab-switch.md missing "run the switch after selection" step; fab-status.md preflight does require config/constitution; git-pr.md `--fill` fallback vs STOP branch order; _cli-fab.md artifact-write git auto-staging. Items already fixed by post-audit batches are verified in place and skipped (audit line numbers refer to v2.1.6).

- **GIVEN** a Bundle A finding with file:line precision
- **WHEN** the cited location is read after this change
- **THEN** the pointer resolves / the count matches / the claim is true against the shipped system

#### R6: Quick-win Bundle B — enumerations completed or replaced by derivation rules
Every Bundle B enumeration MUST be made complete — and where a rule states the invariant without changing behavior, the enumeration MUST be replaced by the derivation rule (root-cause guard): _preamble.md Always-Load exceptions (+ give docs-hydrate-memory the Context Loading section the rule keys on), Subagent Dispatch orchestrator list (+ /fab-proceed), Confidence Scoring invokers (+ /fab-draft, clarify recompute scope); internal-skill-optimize.md partial enumerations (replace with "every `_*.md` file is a shared partial — reference, never target"); _generation.md consumer groups (fab-continue in both); _cli-fab.md in-file index (+ migrations-status/memory-index); `Next:` lines in fab-draft/fab-new/fab-setup (derive per the _preamble Lookup Procedure, + /fab-proceed); _srad.md autonomy-table covering note (verify — c5tr may have fixed it). Enumerations stay where no rule captures membership.

- **GIVEN** an enumeration that has drifted (one of them twice)
- **WHEN** the location is read after this change
- **THEN** either the list is complete as of this change, or it has been replaced by a derivation rule that makes the list unnecessary

### Specs: SPEC mirror conformance

#### R7: SPEC mirrors resynced and constitution rule honored
The ~18 drifted SPEC mirrors enumerated in Theme 7c MUST be resynced against their (post-this-change) `src/kit/skills/` sources — fixing the named drift, then diffing each mirror against its source for same-class residue: SPEC-fab-operator, SPEC-_preamble, SPEC-fab-proceed, SPEC-fab-clarify, SPEC-fab-continue, SPEC-fab-archive, SPEC-git-pr, SPEC-git-pr-review, SPEC-fab-status, SPEC-fab-help, SPEC-fab-new, SPEC-docs-hydrate-specs, SPEC-docs-hydrate-memory, SPEC-_review, SPEC-docs-reorg-memory, SPEC-fab-discuss. Additionally, every `src/kit/skills/*.md` file touched by R4–R6 MUST have its `docs/specs/skills/SPEC-*.md` mirror updated in the same change (constitution; `_cli-fab`/`_cli-external` are policy-exempt — no SPEC exists).

- **GIVEN** the final state of each touched skill source
- **WHEN** its SPEC mirror is compared against it
- **THEN** the mirror describes the source's current behavior (no removed artifacts, no rejected decisions recorded as current, no misquotes), and the touched-skill and touched-mirror lists match (minus the two policy-exempt partials)

### Hygiene

#### R8: Merged-but-unarchived changes archived with merge evidence
Change folders whose PRs are **actually MERGED** (verified per folder via `.status.yaml` `prs:` + `gh pr view <url> --json state,mergedAt`) MUST be archived via `fab change archive <id>` (which moves to archive/, updates the index, and marks the backlog box). The audit-time count of "four" is indicative — the set MUST be recounted at apply time. The `9u91`/`uliv`/`zc9m`/`szxd` backlog boxes MUST be checked only where the corresponding PR is merged. Changes with unmerged/open PRs MUST NOT be archived.

- **GIVEN** a change folder under `fab/changes/` with a non-empty `prs:` list
- **WHEN** every listed PR reports `state: MERGED`
- **THEN** the folder is archived via `fab change archive` and its backlog box is checked
- **AND** folders with any non-merged PR (or no PRs) are left in place

### Non-Goals

- No Go/binary changes, no `src/kit/templates/` edits, no test changes — markdown only
- Report §5 verified-clean areas (e.g., `_pipeline` bracket, f019/f051/f062 fixes, batch-2 naming, migration logic) — out of scope, do not "fix"
- Report §4 structural bets — design discussions, not drive-by fixes
- `.claude/skills/` deployed copies — never edited (regenerate via `fab sync`)

### Design Decisions

1. **Single bundled change** over the report's three-change split — *Why*: user-confirmed (intake assumption 4); the docs layer is internally cross-referential, so one change keeps the mirrors consistent — *Rejected*: three sequenced changes (spec-hooks-rewrite, legacy-docs-truth-sweep, spec-mirror-resync).
2. **Derivation rules over enumerations where the rule states the invariant** — *Why*: the `_pipeline` omission went stale twice; point-fixes don't hold — *Rejected*: completing every list verbatim (drift recurs).
3. **Mirror resync after skill edits** — *Why*: mirrors must describe the post-change sources; resyncing first would immediately re-drift — *Rejected*: parallel resync against pre-change sources.

## Tasks

### Phase 1: Skill-source edits (src/kit/skills)

- [x] T001 Theme 8: `src/kit/skills/docs-reorg-memory.md` — resolve Step 5.3/5.4 stub-before-index contradiction (stub with `description:` frontmatter created before `fab memory-index`; generated rows never hand-edited); reconcile depth off-by-one (≤3 path-segment bound vs report folder-depth column); add abort/rollback escape to the dangling-link hard block <!-- R4 -->
- [x] T002 Theme 8: `src/kit/skills/docs-hydrate-memory.md` — define the index-ownership model once (`description:` frontmatter = single hand-curated field; stub before `fab memory-index`); add generate-mode placement rules (target path, domain creation, index stub, shape bounds); instruct sub-domain index stubs; include sub-domain indexes in the memory-index tier descriptions <!-- R4 -->
- [x] T003 Theme 8 + Bundle A: `src/kit/skills/docs-hydrate-specs.md` — add the no-target-spec branch (Step 5); align Step 5's offered tokens with Step 6's four-token handler (drop or offer the "skip rest" token consistently) <!-- R4 -->
- [x] T004 Theme 8: `src/kit/skills/docs-reorg-specs.md` — add reserved-path exemption for constitution-pinned `docs/specs/skills/SPEC-*.md` mirrors; define recursion into subfolders <!-- R4 -->
- [x] T005 [P] `src/kit/skills/fab-continue.md` — fix the dangling "Review Behavior" heading pointer to a heading that exists in `_review.md`; include sub-domain indexes in the hydrate-step memory-index tier description <!-- R4 -->
- [x] T006 [P] Bundle A: `src/kit/skills/fab-operator.md` — `(see §5)`→`(see §6)`; status-frame example tracked-count vs entries + schema-valid `gmail-deploys` source; drop stale branch_map retention clause <!-- R5 -->
- [x] T007 [P] Bundle A: `src/kit/skills/fab-proceed.md` — `(see Output Format)`→`(see Bypass Notes)`; `sort -r` on full folder names for the same-day tiebreak <!-- R5 -->
- [x] T008 [P] Bundle A: `src/kit/skills/git-pr-review.md` — drop unconsumed `node_id` from the jq projection; reword the `--tool` header (no "bypasses automatic detection", no undefined "cascade") <!-- R5 -->
- [x] T009 [P] Bundle A one-liners: `fab-help.md` Purpose; `internal-retrospect.md` H1; `_cli-external.md` `--reuse` requires `--worktree-name`; `_generation.md` drop "auto-clarify"; `fab-clarify.md` protocol example removed-flow cite; `fab-discuss.md` stage-derivation rule; `fab-switch.md` run-the-switch-after-selection step; `fab-status.md` preflight config/constitution claim; `git-pr.md` `--fill` fallback vs STOP order; `_cli-fab.md` artifact-write git auto-staging <!-- R5 -->
- [x] T010 Bundle B: `src/kit/skills/_preamble.md` — Always-Load exceptions completed or rule-derived (+ /fab-proceed, /fab-help, /fab-archive, /docs-hydrate-specs, /docs-reorg-*); orchestrator list + /fab-proceed; Confidence Scoring invokers + /fab-draft and clarify recompute scope; give `docs-hydrate-memory.md` the Context Loading section the rule keys on <!-- R6 -->
- [x] T011 [P] Bundle B: `src/kit/skills/internal-skill-optimize.md` — replace partial-file enumerations with the derivation rule "every `_*.md` file is a shared partial — reference, never target" (3 sites) <!-- R6 -->
- [x] T012 [P] Bundle B: `src/kit/skills/_generation.md` — fab-continue belongs to both consumer groups (header + intro) <!-- R6 -->
- [x] T013 Bundle B: `src/kit/skills/_cli-fab.md` in-file index + migrations-status/memory-index; `Next:` lines in `fab-draft.md`, `fab-new.md`, `fab-setup.md` derived per the _preamble Lookup Procedure (+ /fab-proceed); verify `_srad.md` autonomy-table covering note (c5tr may have fixed it) <!-- R6 -->

### Phase 2: docs/specs legacy sweep (parallel with Phase 1)

- [x] T014 [P] Rewrite `docs/specs/skills/SPEC-hooks.md` as-shipped: Go `fab hook` handlers (session-start, stop, user-prompt, artifact-write, sync) registered inline in `.claude/settings.local.json`; UserPromptSubmit registered; no deleted shell scripts; no shipped-behavior-as-proposal; current runtime schema <!-- R1 -->
- [x] T015 [P] Rewrite `docs/specs/architecture.md` in place as-shipped: system-cache distribution (`~/.fab-kit/versions/<v>/kit/`), fab router + fab-kit/fab-go binaries, self-consistent Router Dispatch; keep file + specs-index row <!-- R2 -->
- [x] T016 [P] Point-fix legacy docs: `assembly-line.md` spec/tasks narrative; `overview.md` 6-stage story + missing skills; `glossary.md` auto-clarify defs; `user-flow.md` failed-is-review-only (×2); `templates.md` current intake template + `.status.yaml` block (`ready`/`id`/`issues`/`prs`); `skills.md` date-bucketed archive path; `operator.md` v8→v9 — sweeping each fixed claim-class repo-wide (docs/specs + src/kit/skills) <!-- R3 -->

### Phase 3: SPEC mirror resync (after Phase 1)

- [x] T017 Resync the enumerated drifted mirrors against final sources: SPEC-fab-operator, SPEC-_preamble, SPEC-fab-proceed, SPEC-fab-clarify, SPEC-fab-continue, SPEC-fab-archive, SPEC-git-pr, SPEC-git-pr-review, SPEC-fab-status, SPEC-fab-help, SPEC-fab-new, SPEC-docs-hydrate-specs, SPEC-docs-hydrate-memory, SPEC-_review, SPEC-docs-reorg-memory, SPEC-fab-discuss — fix named drift, then diff each against its source for same-class residue <!-- R7 -->
- [x] T018 Update SPEC mirrors for every Phase-1-touched skill not already covered by T017 (e.g., SPEC-docs-reorg-specs, SPEC-fab-switch, SPEC-internal-retrospect, SPEC-internal-skill-optimize, SPEC-_generation, SPEC-_srad if touched, SPEC-fab-draft, SPEC-fab-setup) — `_cli-fab`/`_cli-external` are policy-exempt <!-- R7 -->
- [x] T019 Conformance check: diff the list of touched `src/kit/skills/*.md` against touched `docs/specs/skills/SPEC-*.md`; every non-exempt touched skill has its mirror touched <!-- R7 -->

### Phase 4: Hygiene

- [x] T020 Recount merged-but-unarchived changes: for each folder under `fab/changes/` (non-archive), read `.status.yaml` `prs:`; `gh pr view <url> --json state,mergedAt` per PR; archive every all-MERGED folder via `fab change archive <id>` (verify YAML output incl. `backlog:` field) <!-- R8 -->
- [x] T021 Verify the `9u91`/`uliv`/`zc9m`/`szxd` backlog boxes are checked for merged work (archive auto-marks; hand-check only residue), and confirm no unmerged change was archived <!-- R8 -->

## Execution Order

- Phase 1 and Phase 2 are independent (different file sets) and may run in parallel
- T017/T018 require Phase 1 complete (mirrors must reflect final sources)
- T003 must precede T017's SPEC-docs-hydrate-specs resync; T005 precedes SPEC-fab-continue
- T019 runs after T017+T018; T020 precedes T021

## Acceptance

### Functional Completeness

- [x] A-001 R1: SPEC-hooks.md describes only the shipped Go hook system — no deleted shell scripts in Current Hooks, no shipped behavior framed as proposed, UserPromptSubmit listed as registered, current runtime schema
- [x] A-002 R2: architecture.md describes the system-cache/binary distribution model with a self-consistent Router Dispatch section; file and specs-index row retained
- [x] A-003 R3: all seven legacy-doc point-fix items match the shipped 6-stage pipeline (assembly-line, overview, glossary, user-flow, templates, skills.md, operator.md)
- [x] A-004 R4: the Theme-8 self-contradictions are resolved — stub-before-index ownership stated once and propagated (reorg + hydrate both modes), no-target-spec branch and token alignment in hydrate-specs, reserved-path exemption + recursion in reorg-specs, depth bound reconciled, dangling-link abort escape present
- [x] A-005 R5: every Bundle A item is fixed at its cited location (or verified already-fixed by a post-audit batch and noted)
- [x] A-006 R6: every Bundle B enumeration is complete or replaced by a derivation rule; enumerations remain only where no rule captures membership

### Behavioral Correctness

- [x] A-007 R4: skill edits resolve documented self-contradictions only — no new runtime mechanisms introduced; diff contains only .md files (plus fab/ bookkeeping and archive moves)
- [x] A-008 R6: derivation rules preserve current behavior (e.g., Next: lines still list the same default-first commands the State Table yields)

### Scenario Coverage

- [x] A-009 R7: each of the 16 enumerated drifted mirrors no longer carries its named drift (spot-verified per Theme 7c item)
- [x] A-010 R8: archived folders each have 100% MERGED PRs (evidence recorded); no folder with open/closed-unmerged PRs was archived

### Edge Cases & Error Handling

- [x] A-011 R5: items already fixed by post-audit batches (v2.1.6 line drift) are verified in place and skipped, not blindly re-edited
- [x] A-012 R8: changes with empty `prs:` lists are left unarchived regardless of stage state

### Code Quality

- [x] A-013 Pattern consistency: edits follow each file's existing structure, tone, and formatting conventions
- [x] A-014 No unnecessary duplication: ownership/derivation rules stated once and referenced, not copied

### Documentation Accuracy

- [x] A-015: every corrected claim is true against the shipped system (Go source / current skill prose), and each fixed claim-class was swept repo-wide (docs/specs + src/kit/skills) within scope
- [x] A-016: §5 verified-clean areas are untouched

### Cross References

- [x] A-017: every §-pointer and heading reference edited or added resolves to an existing target; no new dangling links introduced
- [x] A-018 R7: touched-skill list and touched-mirror list match (minus policy-exempt `_cli-fab`/`_cli-external`)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

<!-- SCORING SOURCE NOTE: as of 1.10.0, `fab score` reads intake.md only — this
     ## Assumptions section is the apply-agent's record of graded decisions made
     while co-generating ## Requirements (under-specified points resolved inline),
     NOT a scoring source. Three grades only (Certain/Confident/Tentative) —
     Unresolved is intake-only; apply decides and records, it never leaves a
     decision Unresolved. The Scores column is required for every row. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Audit items already fixed by post-audit batches (c5tr/ye8r/w7dp/g8st landed after the v2.1.6 audit tree) are verified in place and skipped, not re-edited | Audit header pins line numbers to commit 1431a9c3 (v2.1.6); intake assumption 7 establishes recount-at-apply honesty | S:90 R:95 A:90 D:90 |
| 2 | Confident | Bundle A's "~12 more one-liners" is bounded to the items explicitly enumerated in report §3 (mirrored verbatim into intake §5) | Intake says "§3 has the full list" and reproduces it; no open-ended discovery mandate | S:85 R:85 A:80 D:75 |
| 3 | Confident | Mirror resync = targeted drift-fix + per-mirror diff against final source, not full mirror regeneration | Intake calls it "mechanical resync... fix the named items, then diff each mirror against its source for same-class residue"; full regeneration risks losing curated flow diagrams | S:80 R:80 A:85 D:75 |
| 4 | Confident | architecture.md rewrite preserves the file's section skeleton where still valid (directory structure, naming, git/agent integration) and replaces only stale content | "Rewrite in place — keep the file and its specs-index row" implies continuity of purpose, not a blank-slate doc | S:75 R:80 A:80 D:70 |
| 5 | Confident | Archive eligibility = every PR in `prs:` reports MERGED; folders with zero PRs or any non-merged PR stay | Intake §7 names `gh pr view --json state,mergedAt` as the evidence source and "only actually-merged" | S:90 R:85 A:85 D:85 |
| 6 | Confident | Derivation rules keep one or two illustrative examples inline where the original enumeration served a navigational purpose | Root-cause guard says replace the list, not the reader's orientation; keeps prose usable | S:70 R:90 A:80 D:70 |

6 assumptions (1 certain, 5 confident, 0 tentative).
