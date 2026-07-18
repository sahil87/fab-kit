# Plan: Distill No-Arg Survey Mode

**Change**: 260718-ukpf-distill-noarg-survey
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md's four change areas. This is a `docs` change: skill
     markdown only — no Go/CLI, no `_cli-fab.md`, no migrations. Edit ONLY the
     canonical sources; never the `.claude/skills/` deployed copies. The
     `memory-docs/distill.md` memory file is hydrate-stage work, NOT touched at apply. -->

### Skill: No-Arg Survey Mode

#### R1: `<domain>` becomes optional
The `/docs-distill-memory` skill SHALL treat its `<domain>` argument as optional. The header SHALL read `# /docs-distill-memory [<domain>]` and the `## Arguments` entry SHALL be marked optional rather than required.

- **GIVEN** the `/docs-distill-memory` skill source
- **WHEN** a reader inspects the header and `## Arguments` section
- **THEN** the domain argument is documented as optional (`[<domain>]`) with the no-arg behavior described

#### R2: No-arg invocation runs a heuristic survey
WHEN `/docs-distill-memory` is invoked with no `<domain>` argument, the skill SHALL run a **survey mode** instead of aborting: a cheap heuristic scan over all domains that (a) reports per-domain status (which domains have flagged files and how many), (b) auto-selects the first domain with candidates, announces the pick, and proceeds into the existing one-domain flow (full read → per-file report → Step 3 approval gate, all unchanged), and (c) when nothing is flagged anywhere, reports the terminal "all domains distilled (survey heuristic)" case carrying the caveat (R6).

- **GIVEN** a memory tree with at least one domain carrying survey-flagged files
- **WHEN** `/docs-distill-memory` is invoked with no argument
- **THEN** the survey reports per-domain status, auto-picks the first flagged domain, announces it, and enters the existing one-domain flow
- **AND GIVEN** a fully-clean tree, **THEN** the survey reports the terminal all-clean case with the heuristic caveat

#### R3: Survey heuristic set, exclusions, and scan order
The survey SHALL detect exactly three defect classes: (1) `description:` frontmatter over the 500-char cap, (2) change-ids present in `description:` (a `— xu0k`-style suffix or a `(d9rs)`-style citation), and (3) narration markers in bodies (a grep seeded from the skill's own Step 1 narration-pattern list: `renamed`, `supersed`, `` was ` ``, `superseding the historical`, `inverts`). A missing `type: memory` is NOT a survey signal (the full read catches it once a domain is selected). The survey SHALL apply distillation's exclusion set — skip `index.md`, `log.md`, `log.seed.md`, and `_shared/removed-domains.md` — and recurse into sub-domains like Step 1. Survey scan/auto-pick order SHALL follow the domain order of `docs/memory/index.md`'s domain table.

- **GIVEN** the survey step
- **WHEN** it scans domains
- **THEN** it applies the three heuristic classes, honors the exclusion set, recurses sub-domains, and scans in `docs/memory/index.md` domain-table order

#### R4: Explicit `<domain>` remains the override
An explicit `<domain>` argument SHALL remain supported and SHALL force a full read of that domain regardless of survey heuristics (no upfront survey runs on an explicit invocation).

- **GIVEN** `/docs-distill-memory <domain>` with an explicit domain
- **WHEN** the skill runs
- **THEN** it reads that domain in full directly, bypassing the survey heuristic scan

#### R5: Dynamic `Next:` line
After a domain completes, the skill's final `Next:` line SHALL list the surveyed remaining candidate domains (with per-domain flagged-file counts, in `docs/memory/index.md` domain-table order), or report "all domains distilled" when none remain — replacing the static `Next: /docs-distill-memory {another-domain}, /docs-reorg-memory, or /fab-new` placeholder. On an explicit-`<domain>` invocation the completion step SHALL run the survey to populate the line; on a no-arg invocation it MAY reuse the initial survey results minus the completed domain. A skipped or partially cherry-picked domain SHALL stay listed while it still carries flagged files.

- **GIVEN** a completed distillation run
- **WHEN** the skill emits its closing line
- **THEN** the `Next:` line reflects the surveyed remaining candidates (or all-distilled), not the static placeholder

#### R6: Survey caveat stated in output
The survey output SHALL state that the survey is heuristic, and the terminal "survey says all clean" case MUST carry the caveat, e.g. `Survey is heuristic; run /docs-distill-memory <domain> to force a full read of a specific domain.`

- **GIVEN** survey output (per-domain report and especially the all-clean terminal case)
- **WHEN** the skill prints it
- **THEN** the heuristic caveat is present

#### R7: Guardrails and remaining aborts unchanged
The change SHALL preserve all existing guardrails: one domain per apply run, the Step 3 approval gate (apply all / cherry-pick / skip), the read-only-until-approval posture, and the ambiguous-domain / unknown-domain / multiple-domains aborts. ONLY the no-arg abort row is replaced (by survey mode).

- **GIVEN** the `## Error Handling` table and guardrail prose
- **WHEN** the change is applied
- **THEN** only the "No `<domain>` argument" abort row is removed; ambiguous / unknown / multiple-domains aborts and all guardrails remain

### Skill: Mirror & Aggregate Sweep

#### R8: SPEC mirror updated
`docs/specs/skills/SPEC-docs-distill-memory.md` SHALL be updated to reflect the optional-domain/survey mode: its Summary, the Flow diagram (survey branch on no-arg + dynamic `Next:`), and the Tools table if affected.

- **GIVEN** the SPEC mirror
- **WHEN** the skill source changes
- **THEN** the mirror documents optional `<domain>`, the survey branch, and the dynamic `Next:` line (Constitution: skill change MUST carry its SPEC mirror)

#### R9: skills.md aggregate updated
The `## /docs-distill-memory <domain>` section of `docs/specs/skills.md` SHALL be updated: the heading to `[<domain>]`, and the Purpose/Behavior/Key-properties restatement to describe optional-domain + survey mode (while keeping the still-true one-domain-per-run guardrail).

- **GIVEN** the `docs/specs/skills.md` aggregate section
- **WHEN** the skill source changes
- **THEN** the heading and restatement reflect optional `<domain>` + survey mode

### Non-Goals

- No Go/CLI changes, no new `fab` subcommand — the survey is inline agent work (frontmatter reads + grep).
- No `_cli-fab.md` update, no Go tests, no migrations.
- No edits to `.claude/skills/` deployed copies.
- No edits to `docs/memory/memory-docs/distill.md` — that is hydrate-stage work; the intake lists it in the sweep class only to signal hydrate must touch it.
- Multi-domain sequential invocation and a persistent distilled-state marker are rejected (see intake § Why / Rejected alternatives) — not implemented.

### Design Decisions

1. **Survey scan/auto-pick and `Next:` order = `docs/memory/index.md` domain-table order**: use the deterministic user-facing landscape order — *Why*: matches what the user sees, deterministic, trivially reversible prose — *Rejected*: alphabetical or filesystem order (less discoverable, no user-facing anchor).
2. **Narration-marker grep seeded from the skill's own Step 1 list**: reuse `renamed`/`supersed`/`` was ` ``/`superseding the historical`/`inverts` with "e.g." extensibility — *Why*: single-sources the pattern list already in the skill body; keeps survey and full-read classification aligned — *Rejected*: a separate, divergent survey pattern list (drift risk).
3. **A missing `type: memory` is not a survey signal**: the closed three-class defect set from the discussion — *Why*: the full read catches `type:` gaps once a domain is selected; silently widening survey scope deviates from the approved design — *Rejected*: adding `type:`-absence as a fourth survey class.

## Tasks

### Phase 1: Canonical skill source

- [x] T001 Update the header and `## Arguments` of `src/kit/skills/docs-distill-memory.md` — header `# /docs-distill-memory [<domain>]`; `## Arguments` entry changed from *(required)* to optional, documenting that no-arg triggers survey mode (keep the one-domain-per-run and match semantics). <!-- R1 -->
- [x] T002 Update the frontmatter `description:` of `src/kit/skills/docs-distill-memory.md` to advertise the optional-domain/survey capability while keeping the still-true "one domain per run" (per apply) phrasing, staying under the 500-char cap and free of change-ids. <!-- R1 -->

### Phase 2: Survey behavior

- [x] T003 Add a survey step to `## Behavior` in `src/kit/skills/docs-distill-memory.md`: no-arg invocation runs the heuristic scan (three defect classes; exclusion set; recurse sub-domains; scan in `docs/memory/index.md` domain-table order) → per-domain status report → auto-pick first candidate → announce → existing one-domain flow; explicit `<domain>` bypasses the scan and forces a full read. Wire the survey ahead of / around the existing Step 1–5 flow without altering the approval gate. <!-- R2 --> <!-- R3 --> <!-- R4 --> <!-- rework: should-fix — § Context Loading (src/kit/skills/docs-distill-memory.md:80-84) still says "Up front it reads only … the target domain"; add a survey-mode clause covering the no-arg all-domains frontmatter/body scan -->
- [x] T004 Update `## Pre-flight` in `src/kit/skills/docs-distill-memory.md` so the no-arg path is handled by survey mode (domain-required preconditions apply to the resolved/selected domain, not the invocation). <!-- R2 -->

### Phase 3: Output, caveat, dynamic Next, error handling

- [x] T005 Update `## Output` in `src/kit/skills/docs-distill-memory.md` to show the survey per-domain report, the heuristic caveat (R6), and the dynamic `Next:` line format; add the terminal all-clean survey output carrying the caveat. <!-- R2 --> <!-- R5 --> <!-- R6 --> <!-- rework: must-fix — survey-report example (:187-191) lists domains out of docs/memory/index.md table order (_shared, distribution, memory-docs, pipeline, runtime); reorder and harmonize the _shared flagged-count across examples -->
- [x] T006 Replace the static final line `Next: /docs-distill-memory {another-domain}, /docs-reorg-memory, or /fab-new` in `src/kit/skills/docs-distill-memory.md` with the dynamic surveyed-remaining-candidates line (or "all domains distilled"), noting explicit-invocation runs the survey at completion and skipped/partial domains stay listed while flagged. <!-- R5 --> <!-- rework: must-fix — the Next: example (:236) lists runtime before _shared, contradicting the domain-table-order rule stated directly above it (:233); reorder to table order -->
- [x] T007 Update `## Error Handling` in `src/kit/skills/docs-distill-memory.md`: remove the "No `<domain>` argument" abort row (replaced by survey mode); keep the ambiguous-domain, unknown-domain, and multiple-domains abort rows unchanged. <!-- R7 -->
- [x] T008 Update `## Key Properties` (scope row) in `src/kit/skills/docs-distill-memory.md` to reflect optional-domain/no-arg survey while keeping one-domain-per-apply-run and read-only-until-approval accurate. <!-- R7 --> <!-- R1 -->

### Phase 4: Mirror & aggregate sweep

- [x] T009 Update `docs/specs/skills/SPEC-docs-distill-memory.md` — Summary (optional `<domain>` + survey mode), Flow diagram (no-arg survey branch: scan → per-domain report → auto-pick → existing flow; explicit `<domain>` override; dynamic `Next:`), and the Tools table if affected. <!-- R8 --> <!-- rework: should-fix — SPEC Context Loading line (:41) still reads "Reads only … the target domain's"; add survey-mode clause; extend Tools table Read/Bash purpose text (:89-91) to cover the survey scan -->
- [x] T010 Update the `## /docs-distill-memory <domain>` section of `docs/specs/skills.md` — heading to `[<domain>]`; Purpose/Behavior/Key-properties restatement to optional-domain + survey mode (keep the still-true one-domain-per-run guardrail). <!-- R9 -->
- [x] T011 Sweep-verify: grep the repo (excluding `docs/memory/` and `.claude/skills/`) for restatements of the required-`<domain>` invocation, the no-arg abort, and the static `{another-domain}` Next-line, and confirm every occurrence in the sweep class is updated; confirm `glossary.md` / `fkf.md` need no change (they do not restate the changed claims). <!-- R8 --> <!-- R9 --> <!-- rework: must-fix — README.md:462 command quick-reference row still shows `/docs-distill-memory <domain>`; update the cell to `[<domain>]` (sibling rows use bracket notation for optional args; check `shll standards` for the README-governing standard first if available), then re-run the sweep grep including README.md -->

## Execution Order

- T001, T002 (Phase 1) before T003–T008 (they establish the argument/description framing the behavior section references).
- T003 before T004–T008 (survey step is the anchor the other sections point at).
- Phase 4 (T009–T011) after Phase 1–3 so the mirror/aggregate reflect the final canonical wording; T011 last (verification).

## Acceptance

### Functional Completeness

- [x] A-001 R1: The skill header reads `# /docs-distill-memory [<domain>]` and `## Arguments` documents the domain as optional with no-arg survey behavior.
- [x] A-002 R2: No-arg invocation is documented to run survey mode (heuristic scan → per-domain report → auto-pick first candidate → announce → existing one-domain flow; terminal all-clean case when nothing flagged), not the old abort.
- [x] A-003 R3: The survey documents exactly the three defect classes (over-cap `description:`, change-ids in `description:`, body narration markers), excludes `index.md`/`log.md`/`log.seed.md`/`_shared/removed-domains.md`, recurses sub-domains, scans in `docs/memory/index.md` domain-table order, and does NOT treat missing `type: memory` as a survey signal.
- [x] A-004 R4: An explicit `<domain>` is documented as the override that forces a full read regardless of survey heuristics.
- [x] A-005 R5: The final line is the dynamic surveyed-remaining-candidates `Next:` line (or "all domains distilled"), replacing the static `{another-domain}` placeholder; explicit-invocation-runs-survey-at-completion and skipped/partial-domain-stays-listed are documented.
- [x] A-006 R6: The heuristic caveat appears in survey output and specifically on the terminal all-clean case.
- [x] A-007 R8: `docs/specs/skills/SPEC-docs-distill-memory.md` Summary + Flow diagram (+ Tools table if affected) reflect optional `<domain>` + survey mode.
- [x] A-008 R9: The `## /docs-distill-memory <domain>` section of `docs/specs/skills.md` has the `[<domain>]` heading and an optional-domain/survey-mode restatement.

### Behavioral Correctness

- [x] A-009 R7: Only the "No `<domain>` argument" abort row is removed from `## Error Handling`; ambiguous-domain, unknown-domain, and multiple-domains aborts remain verbatim, and the one-domain-per-apply-run / approval-gate / read-only guardrails are intact.
- [x] A-010 R2: Existing explicit-`<domain>` behavior is unchanged except for the dynamic `Next:` line (purely additive for existing invocations).

### Edge Cases & Error Handling

- [x] A-011 R2: The fully-clean-tree terminal case ("all domains distilled") is documented and carries the heuristic caveat (idempotency: a distilled tree surveys clean every re-run).

### Code Quality

- [x] A-012 Pattern consistency: Edited markdown follows the surrounding skill/spec structure and CommonMark conventions (Constitution IV).
- [x] A-013 No unnecessary duplication: The survey narration-marker list single-sources the skill's own Step 1 pattern list rather than duplicating a divergent list.
- [x] A-014 Canonical source only: Only `src/kit/skills/docs-distill-memory.md`, `docs/specs/skills/SPEC-docs-distill-memory.md`, and `docs/specs/skills.md` are edited — no `.claude/skills/` copy and no `docs/memory/` file touched at apply (code-quality.md anti-patterns; Constitution V).

### documentation_accuracy

- [x] A-015 Every changed behavioral claim (optional argument, survey heuristics/exclusions/order, dynamic `Next:`, caveat, retained aborts/guardrails) is stated consistently across the skill source, SPEC mirror, and skills.md aggregate — no contradictory or stale restatement remains. <!-- re-verified (review cycle 2): survey-report example (src/kit/skills/docs-distill-memory.md:189-193) and Next: example (:238) now follow docs/memory/index.md domain-table order with a harmonized _shared count (2), consistent with the auto-pick announcement -->

### cross_references

- [x] A-016 The sweep class is complete: the skill source, SPEC mirror, and skills.md aggregate are all updated; a repo-wide grep (excluding `docs/memory/` and `.claude/skills/`) confirms no other file restates the changed required-`<domain>` / no-arg-abort / static-Next claims (glossary.md and fkf.md verified to need no change). <!-- re-verified (review cycle 2): README.md:462 now reads `/docs-distill-memory [<domain>]`; sweep grep re-run incl. README — remaining hits are change artifacts, an archived change, the hydrate-deferred memory file, and the new caveat's placeholder usage; glossary.md/fkf.md clean -->

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Survey scan/auto-pick order and `Next:` candidate order = `docs/memory/index.md` domain-table order (`_shared`, `distribution`, `memory-docs`, `pipeline`, `runtime`) | Intake Assumption 9; deterministic and matches user-facing landscape; presentational and reversible | S:60 R:90 A:80 D:70 |
| 2 | Confident | Narration-marker grep list seeded from the skill's own Step 1 patterns (`renamed`, `supersed`, `` was ` ``, `superseding the historical`, `inverts`) with "e.g." extensibility | Intake Assumption 10; single-sources the existing in-body pattern list | S:60 R:90 A:85 D:70 |
| 3 | Confident | A missing `type: memory` is NOT a survey signal; survey defect set is exactly the three discussed classes | Intake Assumption 13; closed defect list in the approved design, full read catches `type:` gaps | S:60 R:90 A:70 D:70 |
| 4 | Confident | `glossary.md` and `fkf.md` need no edit — glossary's "one domain per run" restatement stays true and does not assert a required argument or the static Next-line; fkf.md only name-lists the skill among forward writers | Repo grep verified; only the argument-required framing changes, which those files do not carry | S:65 R:85 A:85 D:75 |
| 5 | Confident | The `## Deletion Candidates` section is not added to plan.md (change_type `docs`) | plan.md template comment: parsimony/deletion-candidate passes are skipped for `docs`/`chore`/`ci` and the section is omitted entirely | S:80 R:85 A:90 D:85 |

5 assumptions (0 certain, 5 confident, 0 tentative).
