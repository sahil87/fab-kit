# Plan: Skills Twins & Self-Duplication Refactor

**Change**: 260611-szxd-skills-twins-self-duplication-refactor
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### Pipeline Skills: Twins

#### R1: fab-draft is a thin delta over fab-new (f031)
`src/kit/skills/fab-draft.md`'s body SHALL be replaced with a delta instruction over `fab-new.md`: read `.claude/skills/fab-new/SKILL.md`, execute its Pre-flight/Arguments/Steps 0–9 with deltas — Step 9 tail = change NOT activated (user must `/fab-switch`), **skip Steps 10–11** (stated explicitly and prominently), Output + `Next:` per the Activation Preamble convention, drop the activation/git error rows. The shared steps MUST NOT move into `_generation.md`. `helpers: [_generation, _srad]` stays (the executed fab-new steps need both).

- **GIVEN** the deployed `.claude/skills/fab-draft/SKILL.md` after `fab sync`
- **WHEN** an agent executes `/fab-draft <description>` cold
- **THEN** it reads fab-new's SKILL.md, runs Steps 0–9, stops after `fab status advance {name} intake`, never runs `fab change switch` or any git command, and ends with `Next: /fab-switch {name} to make it active, then /fab-continue, /fab-fff, /fab-ff, or /fab-clarify`

#### R2: Shared ff/fff bracket extracted to a `_pipeline` helper (f007)
A new `src/kit/skills/_pipeline.md` internal partial SHALL hold the shared pipeline bracket: pre-flight (intake prerequisite + intake gate via `fab score --check-gate --stage intake`), context loading, the behavior/dispatch note, resumability, Steps 1–3 (apply → review → hydrate), the auto-rework loop + escalation rule, the exhaustion stop message, and the shared error rows. It is parameterized by **`{driver}`** (`fab-ff` | `fab-fff`) and **`{terminal}`** (`hydrate` | `review-pr`). `fab-ff.md`/`fab-fff.md` SHALL shrink to Purpose + Arguments + a parameter table invoking the bracket (+ fab-fff's Steps 4–5 ship/review-pr and driver-specific Output/error rows). Both declare `helpers: [_generation, _review, _srad, _pipeline]`. `_preamble.md`'s allowed `helpers:` values SHALL gain `_pipeline` (6 values). Extraction resolves documented drift: gate terminology unified to the constitution's single-intake-gate framing (no "Two gates"), and fab-ff's post-bail `/fab-clarify` guidance lives in the shared stop text so fff regains it.

- **GIVEN** the deployed `.claude/skills/fab-ff/SKILL.md` and `.claude/skills/_pipeline/SKILL.md` after `fab sync`
- **WHEN** an agent executes `/fab-ff` cold
- **THEN** the frontmatter `helpers:` leads it to `_pipeline`, the parameter table makes driver = `fab-ff` and terminal = `hydrate` unambiguous, and the bracket executes apply → review → hydrate with `fab-ff` as every event-command driver
- **GIVEN** `/fab-fff`
- **WHEN** the bracket's Step 3 completes
- **THEN** fab-fff.md's own Steps 4–5 (ship, review-pr) run with their existing semantics (self-managed transitions, timeout outcome) unchanged

#### R3: Rework-cycle choreography stated once, explicitly (f071)
`_pipeline.md` SHALL state the per-cycle choreography exactly once: each cycle = (1) `fab status fail <change> review` then `fab status reset <change> apply {driver}` — the pair repeats on **every** failed review verdict, not just the first; (2) triage + one rework action; (3) re-dispatch apply via a `/fab-continue` Apply Behavior subagent (same no-`fab status` prompt contract), then `fab status finish <change> apply {driver}`; (4) dispatch a **fresh** `/fab-continue` Review Behavior subagent; (5) verdict — pass finishes review and proceeds; fail starts the next cycle or exhausts. At exhaustion (3rd cycle's re-review fails) the orchestrator SHALL run `fab status fail <change> review` only (no reset), leaving review in the `failed` state as the defined terminal state, and stop with the per-cycle summary. `_review.md`'s stale rework-loop pointer ("fab-ff.md Step 3, fab-fff.md Step 3") SHALL point at `_pipeline.md`'s Auto-Rework Loop instead.

- **GIVEN** a review failure on cycle 2 of `/fab-fff`
- **WHEN** the orchestrator processes the verdict
- **THEN** it fires the same fail+reset pair as cycle 1, re-dispatches apply, finishes apply, and dispatches a fresh review subagent — two conforming implementations leave identical `.status.yaml` histories
- **GIVEN** the 3rd cycle's re-review fails
- **WHEN** the orchestrator exhausts
- **THEN** `.status.yaml` shows `review: failed` (apply remains `done`), and the stop message describes what `/fab-continue` will actually do

#### R4: fab-continue gains a review-failed dispatch row (f019)
`fab-continue.md` Step 1 SHALL handle `progress.review == failed` with a dispatch row that presents the existing rework menu directly: run `fab status reset <change> apply fab-continue` (the same post-fail reset the Verdict fail path runs — review cascades to pending, apply re-activates), present the Verdict-fail rework options table, and stop for the user's choice — it MUST NOT re-run review first. This row replaces the former failed→`start review` resume guard (review=failed is now a deliberate resting state per R3); `_pipeline`'s Resumability keeps the `start review` recovery for orchestrator re-runs.

- **GIVEN** a change whose `/fab-ff` run exhausted rework (review `failed`)
- **WHEN** the user runs `/fab-continue`
- **THEN** the rework menu (fix code / revise plan / revise requirements) is presented directly — no autonomous re-review — making the exhaustion guidance truthful

### Pipeline Skills: Self-Duplication

#### R5: fab-new Step 11 compressed to a single table (f032)
`fab-new.md` Step 11's inlined 5-case branch logic SHALL become one condition/command/report table (~15 lines) annotated "evaluate in order, first match wins", preceded by the context commands and a keep-in-sync comment referencing `git-branch.md` Step 4. The five case outcomes and report strings are preserved verbatim (already-on-target / target-exists / on-main / local-only rename with guard / new-branch-leaving-old-intact). The logic stays inline — no runtime `/git-branch` delegation. `git-branch.md` itself is untouched.

- **GIVEN** `/fab-new` Step 11 on a local-only branch belonging to another change
- **WHEN** the table is evaluated in order
- **THEN** the first matching row yields `git checkout -b "{name}"` and `Branch: {name} (created, leaving {old_branch} intact)` — byte-identical report strings to today

#### R6: fab-archive single-document merge (f087)
`fab-archive.md` SHALL contain one `#`-level document: mode detection + both argument lists once at the top; the second document demoted to a `## Restore Mode` section holding only its unique Behavior/Output/Error-Handling/Key-Properties content. The restore-mode pre-flight waiver (no standard preflight, no hydrate guard — opposite of archive mode) is mode-specific content and MUST be preserved.

- **GIVEN** the merged file
- **WHEN** an agent runs `/fab-archive restore <name>`
- **THEN** it skips preflight and the hydrate guard (restores any archived change regardless of state) exactly as before

#### R7: git-pr resolves change context once (f094)
`git-pr.md` SHALL resolve change context ONCE in a unified Step 0 producing `{name}`, `{has_fab}`, `{has_intake}`, `{change_type}`; Steps 0a/0b/1/1b/3c/4a reference those variables and MUST NOT re-run `fab change resolve`. The Step 0b and Step 3c step names stay intact (`_cli-fab.md` and `prmeta.go` cite them by name).

- **GIVEN** `/git-pr` on a branch with an active change
- **WHEN** the pipeline executes Steps 0–4
- **THEN** `fab change resolve` runs exactly once (Step 0) and every later step consumes the stored variables, with identical observable behavior (same nudges, same Meta gating, same add-pr path)

#### R8: git-pr-review states the triage taxonomy once (f098)
`git-pr-review.md` Step 4 items 1 and 3 SHALL merge into one classify-and-assign list (keeping the examples); the Disposition Reference table is the single reply-format source (Step 5.5 item 1 drops the formats but preserves the 7-char-SHA + description detail); the Rules section is cut to fully-autonomous, the general fail-fast line (it has no other general statement), and targeted-fixes-only.

- **GIVEN** the revised skill
- **WHEN** an agent triages and replies
- **THEN** disposition intents and reply outcomes are unchanged (`Fixed — {description}. ({sha})` / `Deferred — {reason}.` / `Skipped — {reason}.`; informational gets no reply)

### Runtime: Operator

#### R9: Spawn sequence stated once + entry-form table (f049)
`fab-operator.md` SHALL keep the canonical 6-step spawn sequence only in §6 "Spawning an Agent"; the three Working-a-Change walkthroughs become a 3-row table mapping entry form → initial command (`/fab-switch <change> && /fab-proceed`, `/fab-new <escaped-text>`, `/fab-new <id>`) + "run the §6 spawn sequence"; Autopilot steps 1–2 and Watches step 4 become one-line §6 references. Variant-specific extras are preserved: shell-escaping note, idea-lookup pre-step, `--reuse`, watch-enrollment extras (`stop_stage`/`spawned_by`).

- **GIVEN** a raw-text work request
- **WHEN** the operator consults Working a Change
- **THEN** the table row yields the §6 sequence with initial command `/fab-new <shell_escaped_description>` and the shell-escaping requirement intact

#### R10: Status Frame Format extracted; render-path rationale collapsed (f116)
The ~74-line status-frame spec SHALL move out of tick step 1 into a `Status Frame Format` subsection (tick step 1 ends "emit the status frame — see Status Frame Format"). The 4x-repeated render-path rationale collapses into one rule: emit bare markdown (no code fence, no headings, no ANSI); channels: tables, emoji, bold, italic, code spans, plain URLs. The runtime no-fence rule (agent-critical, distinct), the frame example, and the two column tables are kept; the "Why emoji + table, not ANSI" design-history paragraph is dropped (it lives in `runtime/operator.md`).

- **GIVEN** a tick
- **WHEN** the operator emits the frame
- **THEN** the emitted markdown is unchanged (same header, anchors, tables, emoji, footnote behavior) — only the skill's internal organization changed

### Distribution: Setup

#### R11: Migrations version handling single-sourced to the binary (f080)
`fab-setup.md` SHALL delete the triplicated version read/parse/compare — pre-flight checks 1/2/4, the Compare Versions step, and the Semver Comparison section — and drop Migrations Context Loading item 1. The skill runs `fab migrations-status --json` once and branches on its returned `local`/`engine` fields (one-line rule) to pick the equal / ahead / no-op output; the binary exits non-zero with remediation hints on missing version files.

- **GIVEN** `/fab-setup migrations` with `fab/.kit-migration-version` missing
- **WHEN** the skill runs `fab migrations-status --json`
- **THEN** the binary's non-zero exit + stderr is surfaced and the skill stops — no hand-rolled existence check fired first

#### R12: Bootstrap sync-first reorder (f077) — flagged behavior-ORDER change
`fab sync` SHALL move from last (step 1j) to immediately after the interactive config/constitution creation (1a/1b — sync requires `config.yaml` `fab_version`), with a sync-failure guard (non-zero exit → STOP, surface output). Steps 1c–1g, 1i, and 1k are deleted (sync's `scaffoldTreeWalk` copy-if-absent installs context/code-quality/code-review/index files; `scaffoldDirectories` creates `fab/changes/` + archive + `.gitkeep`; the `.gitignore` fragment merge covers `.fab-*`, subsuming 1k). The migration-version note (old 1h) is renumbered and its "step 1j" references repointed. Bootstrap Output is rewritten accordingly. Outcome is identical via idempotency; this is the batch's one explicit behavior-ORDER change and MUST be flagged in the PR description (ship stage).

- **GIVEN** a fresh bootstrap
- **WHEN** `/fab-setup` runs with no arguments
- **THEN** the order is doctor → config (1a) → constitution (1b) → `fab sync` (1c) and the resulting file tree is identical to the old order's
- **GIVEN** `fab sync` exits non-zero during bootstrap
- **WHEN** the guard fires
- **THEN** the bootstrap stops and surfaces sync's output

### Docs: SPEC Mirrors & Verification

#### R13: Every touched skill's SPEC mirror updated; new SPEC-_pipeline.md
Each touched skill file SHALL have its `docs/specs/skills/SPEC-*.md` mirror updated, including a new `SPEC-_pipeline.md` (underscore-helper mirrors per the `SPEC-_srad.md` precedent). `docs/specs/skills.md` § Skill Helpers SHALL reflect the 6-value allowlist and the fab-ff/fab-fff mapping row.

- **GIVEN** the constitution constraint "Changes to skill files MUST update the corresponding SPEC"
- **WHEN** the diff is reviewed
- **THEN** every `src/kit/skills/*.md` change pairs with its SPEC mirror change, and `SPEC-_pipeline.md` exists

#### R14: Post-sync verification of the changed read paths
After all edits, `fab sync` SHALL be run, then the two changed indirections dry-run-verified: (a) the deployed `fab-draft` delta reads coherently against the deployed `fab-new` (manual read-through); (b) deployed `fab-ff`/`fab-fff` resolve `.claude/skills/_pipeline/SKILL.md` and the driver/terminal parameterization is unambiguous when read cold. `git diff --stat` SHALL be checked against the intake Impact list.

- **GIVEN** `fab sync` has run
- **WHEN** the deployed files are read cold
- **THEN** the fab-draft delta and the `_pipeline` indirection hold with no dangling references

### Non-Goals

- docs-reorg-memory vs docs-reorg-specs divergence (f197 — adversarially verified as justified) — untouched
- f107 specs-ward port (Kind column, Link Impact note, no-dangling-link verify) — excluded
- Pre-Go-CLI staleness residue sweep (uliv Non-Goals list) — left for a later docs sweep
- No Go code changes; no `.claude/skills/` direct edits (refresh via `fab sync` only)
- No memory-file edits (hydrate stage owns `docs/memory/`)

### Design Decisions

1. **`_pipeline` loaded via frontmatter `helpers:`, not in-body**: the bracket is the entire skill body of ff/fff, so the load is unconditional — frontmatter is the honest declaration per the zc9m contract. — *Rejected*: in-body point-of-use read (that pattern is for conditional loads).
2. **Exhaustion terminal state = review `failed`** (fail without reset on the final failure): the only state from which `/fab-continue`'s new review-failed row (and `_preamble`'s "review (fail)" state) can present the rework menu, making the stop message truthful. — *Rejected*: fail+reset on the final failure too (leaves apply `active`; `/fab-continue` would silently re-run apply→review instead of showing the menu).
3. **fab-continue's failed→`start review` resume guard replaced by the menu row**: both the interrupted-fail→reset case and the exhaustion case land on the same deliberate resting state; the menu is strictly more useful than an unconditional re-review. Orchestrators keep the `start review` recovery in `_pipeline` Resumability (autonomous re-run wants re-review). — *Rejected*: keeping both paths in fab-continue (contradictory dispatch for the same state).
4. **Shared error rows live in `_pipeline`** with `{driver}`-parameterized message text; each wrapper keeps only driver-specific rows (fff: ship/review-pr). — *Rejected*: duplicating the 5 shared rows per wrapper (re-creates the drift surface).

## Tasks

### Phase 1: Helper + Twins

- [x] T001 Create `src/kit/skills/_pipeline.md` — shared bracket (pre-flight gate, context loading, dispatch note, resumability, Steps 1–3, auto-rework loop with explicit per-cycle choreography, exhaustion stop w/ post-bail `/fab-clarify` guidance, shared error rows), parameterized by `{driver}`/`{terminal}` <!-- R2, R3 --> <!-- rework cycle 1: Behavior note claimed bracket "always passes {driver}" but the preserved Resumability start-review command passes none — soften claim toward preserved command -->
- [x] T002 Rewrite `src/kit/skills/fab-ff.md` as thin wrapper (Purpose, Arguments, parameter table → `_pipeline`, driver Output block, driver error rows); `helpers: [_generation, _review, _srad, _pipeline]` <!-- R2 -->
- [x] T003 Rewrite `src/kit/skills/fab-fff.md` as thin wrapper + fff-only Steps 4–5 (ship/review-pr incl. timeout outcome), driver Output + error rows; same helpers <!-- R2 -->
- [x] T004 Add `_pipeline` to `src/kit/skills/_preamble.md` § Skill Helper Declaration allowed values <!-- R2 -->
- [x] T005 Fix `src/kit/skills/_review.md` trailing note — rework loop pointer → `fab-continue.md` Verdict + `_pipeline.md` § Auto-Rework Loop <!-- R3 -->
- [x] T006 Rewrite `src/kit/skills/fab-draft.md` as thin delta over fab-new Steps 0–9 (prominent skip-10–11, Activation Preamble Next, delta Key Properties) <!-- R1 -->
- [x] T007 `src/kit/skills/fab-continue.md` — replace the failed→start-review resume guard with the review-failed dispatch row (reset apply + present rework menu, no re-review) in Step 1 prose + dispatch table <!-- R4 --> <!-- rework cycle 1: add parenthetical noting the review/failed row keys on progress.review via the Step 1 guard (derived stage never yields failed) -->

### Phase 2: Self-Duplication Consolidations

- [x] T008 `src/kit/skills/fab-new.md` Step 11 — context commands + keep-in-sync comment + 5-row first-match-wins table, report strings verbatim <!-- R5 -->
- [x] T009 `src/kit/skills/fab-archive.md` — merge to one document; demote restore to `## Restore Mode` with unique content + preserved pre-flight waiver <!-- R6 -->
- [x] T010 `src/kit/skills/git-pr.md` — unified Step 0 (`{name}`/`{has_fab}`/`{has_intake}`/`{change_type}`); repoint Steps 0a/0b/1/1b/3c/4a to the variables; keep Step 0b/3c names <!-- R7 -->
- [x] T011 `src/kit/skills/git-pr-review.md` — merge Step 4 items 1+3; Step 5.5 reply formats → Disposition Reference (keep SHA detail); trim Rules to 3 lines <!-- R8 --> <!-- rework cycle 1: restore the umbrella idempotency Rules line the consolidation deleted (SPEC + skills.md still cite it) -->
- [x] T012 `src/kit/skills/fab-operator.md` — Working-a-Change 3-row entry-form table; Autopilot 1–2 and Watches step 4 one-line §6 refs; preserve extras <!-- R9 -->
- [x] T013 `src/kit/skills/fab-operator.md` — extract `### Status Frame Format` after Tick Behavior; collapse render-path rationale to one rule; keep no-fence rule, example, column tables <!-- R10 -->
- [x] T014 `src/kit/skills/fab-setup.md` migrations — drop Context Loading item 1, pre-flight checks 1/2/4, Compare Versions step, Semver Comparison; single migrations-status branch rule <!-- R11 --> <!-- rework cycle 1: A-012 must-fix — two imperative migrations-status runs remained (Context Loading item 2 + Step 1); make Context Loading a pointer -->
- [x] T015 `src/kit/skills/fab-setup.md` bootstrap — sync to step 1c (after 1a/1b) with failure guard; delete old 1c–1g/1i/1k; renumber old 1h refs; rewrite Bootstrap Output <!-- R12 -->

### Phase 3: SPEC Mirrors

- [x] T016 Create `docs/specs/skills/SPEC-_pipeline.md` (Summary, parameters, Flow, sub-agents, bookkeeping) <!-- R13 -->
- [x] T017 [P] Update `docs/specs/skills/SPEC-fab-ff.md` + `SPEC-fab-fff.md` (bracket → `_pipeline`, helpers, choreography, terminal state) <!-- R13 -->
- [x] T018 [P] Update `docs/specs/skills/SPEC-fab-draft.md` (thin-delta model) + `SPEC-fab-new.md` (Step 11 table) <!-- R13 -->
- [x] T019 [P] Update `docs/specs/skills/SPEC-fab-continue.md` (review-failed row) + `SPEC-_review.md` (pointer) + `SPEC-_preamble.md` (6 allowed values) <!-- R13 --> <!-- rework cycle 1: repair garbled clause in SPEC-fab-continue review-failed note -->
- [x] T020 [P] Update `docs/specs/skills/SPEC-fab-archive.md` + `SPEC-git-pr.md` + `SPEC-git-pr-review.md` <!-- R13 -->
- [x] T021 [P] Update `docs/specs/skills/SPEC-fab-operator.md` + `SPEC-fab-setup.md` <!-- R13 --> <!-- rework cycle 1: SPEC-fab-setup stale mechanism names (symlinks→copies, fab-sync.sh→fab sync, gitignore Edit row→sync-owned) -->
- [x] T022 Update `docs/specs/skills.md` § Skill Helpers (6 allowed values; fab-ff/fab-fff mapping row) <!-- R13 --> <!-- rework cycle 1: skills.md fab-draft Behavior line still cited "Identical to /fab-new Steps 1-7" — align with thin-delta interface -->

### Phase 4: Verification

- [x] T023 Run `fab sync`; confirm `.claude/skills/_pipeline/SKILL.md` deployed <!-- R14 -->
- [x] T024 Dry-run (a): read deployed `fab-draft` delta against deployed `fab-new` — verify Steps 0–9 executable, skip-10–11 unambiguous, Next line correct <!-- R14 -->
- [x] T025 Dry-run (b): read deployed `fab-ff`/`fab-fff` cold — verify `_pipeline` resolution and driver/terminal parameterization unambiguous <!-- R14 -->
- [x] T026 `git diff --stat` — confirm touched files match intake Impact (12 skill sources + SPEC mirrors + skills.md + plan.md) <!-- R14 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab-draft.md` body is a delta instruction (no duplicated Steps 0–9); skip-Steps-10–11 is explicit and prominent; activation/git error rows absent
- [x] A-002 R2: `_pipeline.md` exists with the full shared bracket parameterized by driver + terminal; `fab-ff.md`/`fab-fff.md` contain no copy of Steps 1–3, the rework loop, or the gate pre-flight
- [x] A-003 R2: `_preamble.md` allowed `helpers:` values include `_pipeline` (6 values); fab-ff/fab-fff frontmatter declares it
- [x] A-004 R3: per-cycle choreography appears exactly once (in `_pipeline.md`), names the repeating fail+reset pair, the Apply Behavior re-dispatch, the fresh review subagent, and the exact exhaustion terminal state (review `failed`)
- [x] A-005 R4: `fab-continue.md` Step 1 has a `review`/`failed` dispatch row presenting the rework menu directly (reset apply, no re-review)
- [x] A-006 R5: fab-new Step 11 is one first-match-wins table (~15 lines) with a keep-in-sync comment referencing `git-branch.md` Step 4
- [x] A-007 R6: `fab-archive.md` has a single `#` title; restore content lives in `## Restore Mode` with no duplicated Purpose/Arguments
- [x] A-008 R7: `git-pr.md` contains exactly one `fab change resolve` instruction (Step 0); Steps 0b/1/1b/3c/4a consume `{name}`/`{has_fab}`/`{has_intake}`/`{change_type}`
- [x] A-009 R8: taxonomy definitions appear once (merged Step 4 list); reply formats appear only in Disposition Reference; Rules ≤ 3 lines <!-- review rework cycle 1: Rules is now 4 lines — the umbrella idempotency line was deliberately restored per T011 rework (SPEC-git-pr-review.md + skills.md Key properties cite it; byte-identical to pre-change HEAD line 234). Consolidation criterion met in substance: only non-restated general lines remain -->
- [x] A-010 R9: spawn sequence steps appear only in §6; Working a Change is a 3-row table; Autopilot 1–2/Watches 4 are one-line refs
- [x] A-011 R10: frame spec lives in `### Status Frame Format`; render-path rule stated once; no-fence rule + example + both column tables retained
- [x] A-012 R11: no version read/parse/compare prose remains in fab-setup migrations; Semver Comparison section gone; one migrations-status invocation instruction <!-- review rework cycle 1: resolved — Migrations Context Loading item 2 (fab-setup.md:290) is now a pointer to Step 1's single run; exactly one imperative "Run `fab migrations-status --json`" remains (fab-setup.md:296), verified across the full file -->
- [x] A-013 R12: bootstrap order is 1a → 1b → 1c (`fab sync` + failure guard); steps 1c–1g/1i/1k content gone; no stale "step 1j" references; Bootstrap Output matches
- [x] A-014 R13: every touched `src/kit/skills/*.md` has a paired SPEC mirror edit; `SPEC-_pipeline.md` exists; `skills.md` allowlist updated

### Behavioral Correctness

- [x] A-015 R2: report strings, commands, and case semantics in the extracted bracket are unchanged except the unified gate framing and the parameterized driver tokens
- [x] A-016 R5: all five Step 11 report strings byte-match the originals
- [x] A-017 R6: restore mode still waives preflight + hydrate guard
- [x] A-018 R12: sync-first bootstrap produces an identical file tree via idempotency (copy-if-absent, line-ensure merges); this is the only behavior-ORDER change in the batch

### Removal Verification

- [x] A-019 R1: no byte-copy of fab-new Steps 0–9 remains in fab-draft.md
- [x] A-020 R11: pre-flight checks 1/2/4, Compare Versions, and Semver Comparison are gone from fab-setup.md
- [x] A-021 R12: steps 1c–1g, 1i, 1k are gone from the bootstrap sequence

### Scenario Coverage

- [x] A-022 R14: `fab sync` deploys `_pipeline`; dry-run read-throughs of deployed fab-draft and fab-ff/fab-fff confirm both indirections hold
- [x] A-023 R14: `git diff --stat` matches the intake Impact section

### Code Quality

- [x] A-024 Pattern consistency: `_pipeline.md` follows the `_generation`/`_review` internal-partial conventions (frontmatter flags, orchestration-note preamble)
- [x] A-025 No unnecessary duplication: no shared block is left duplicated between fab-ff/fab-fff or fab-new/fab-draft (the refactor's own anti-pattern check)

### Documentation Accuracy

- [x] A-026 R13: SPEC mirrors describe the post-refactor structure (delta model, bracket parameters, dispatch row, reordered bootstrap) without stale step letters/numbers

### Cross References

- [x] A-027 R7: `_cli-fab.md`/`prmeta.go` citations of git-pr Step 0b/3c still resolve (names kept); `_review.md` rework-loop pointer resolves to `_pipeline.md`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- The f077 behavior-ORDER change MUST be called out in the PR description (ship stage responsibility).

## Deletion Candidates

This refactor's purpose was deleting duplicated content, and it consumed almost all of its own candidates (~750 lines removed). One small residual redundancy remains (a second — the duplicate migrations-status run instruction at fab-setup.md:290 — was consumed by rework cycle 1, which made it a pointer to Step 1's single run):

- `src/kit/skills/fab-operator.md:459` (Working a Change preamble) — the parenthetical arrow-restatement of the §6 spawn sequence (`establish target repo → wt create → … → enroll`); the "§6 spawn sequence above" pointer alone suffices and the restatement re-creates a small keep-in-sync surface.

Intentionally NOT candidates: `git-branch.md` Step 4 vs fab-new Step 11 (verifier-sanctioned twin, kept in sync via in-file comment); the stale memory passages in `docs/memory/` (hydrate owns those — listed in the review's memory-drift warnings); `docs/specs/findings/skills-review-2026-06-11.md` (historical record of the 4-batch remediation, archival is a backlog decision).

## Assumptions

<!-- SCORING SOURCE NOTE: as of 1.10.0, `fab score` reads intake.md only — this
     ## Assumptions section is the apply-agent's record of graded decisions made
     while co-generating ## Requirements (under-specified points resolved inline),
     NOT a scoring source. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | fab-ff/fab-fff load `_pipeline` via frontmatter `helpers:` (unconditional), not in-body | The bracket is the entire skill body; zc9m contract says frontmatter = unconditional pre-body loads; intake names the allowlist addition | S:85 R:90 A:90 D:90 |
| 2 | Certain | Exhaustion terminal state = review `failed` (final failure fires `fail` only, no reset) | f019 requires the rework menu reachable from exhaustion; the menu row keys on review=failed; fail+reset would leave apply active and re-run autonomously | S:80 R:85 A:90 D:85 |
| 3 | Confident | fab-continue's failed→`start review` resume guard is replaced by the menu row (not kept alongside) | Two dispatch behaviors for one state would be contradictory; menu strictly dominates (its options re-enter apply); orchestrators retain start-review recovery in `_pipeline` | S:70 R:80 A:80 D:75 |
| 4 | Confident | fff's intake-missing STOP tail ", then run /fab-fff" unified into the shared parameterized message | Same drift class as the gate-terminology unification the intake mandates; message differs only by a driver-pointing tail | S:65 R:90 A:80 D:75 |
| 5 | Confident | Shared error rows (preflight/intake-missing/gate/task/review) live in `_pipeline` with `{driver}` tokens; wrappers keep only driver-specific rows | "Shrink to wrappers" intent; duplicating rows re-creates the drift surface; findings list output/error rows as the only fff-specific content beyond Steps 4–5 | S:70 R:85 A:80 D:75 |
| 6 | Certain | "Immediately after Phase 0" means after 1a/1b interactive config/constitution — order doctor → config → constitution → sync | Intake verifier caveat states sync requires `config.yaml` `fab_version` and "must stay after Phase 0's interactive config creation (1a/1b)" | S:90 R:85 A:90 D:90 |
| 7 | Confident | Step 11 table merges old Case-4-else and Case 5 into one row (identical command + report string) — five outcomes preserved | Compression target ~15 lines; rows are byte-identical in command and report; first-match-wins ordering preserves semantics | S:75 R:85 A:85 D:80 |
| 8 | Confident | `docs/specs/skills.md` § Skill Helpers updated (allowlist + mapping) though not a SPEC-*.md mirror | Leaving it at 5 values would contradict the new `_preamble` allowlist; new-skill checklist item 7 directs helper rows there | S:70 R:90 A:85 D:80 |
| 9 | Confident | Dry-runs performed as cold read-throughs of deployed files (no scratch change created) | Task instructions explicitly accept manual read-through for the dry-run; avoids pointer churn on the active change | S:75 R:90 A:85 D:85 |

9 assumptions (3 certain, 6 confident, 0 tentative).
