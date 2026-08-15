# Plan: Right-Size Fab Routing — Anti-Trigger Descriptions + Micro-Change Backstop

**Change**: 260815-slpa-right-size-fab-routing
**Intake**: `intake.md`

## Requirements

### Routing: Anti-Trigger Frontmatter Descriptions

#### R1: Four routing skills carry an identical anti-trigger in their `description:` frontmatter
The `description:` frontmatter of exactly `src/kit/skills/fab-proceed.md`, `src/kit/skills/fab-fff.md`, `src/kit/skills/fab-ff.md`, and `src/kit/skills/fab-new.md` MUST gain an appended anti-trigger sentence whose micro-criteria core is byte-identical across all four files. The sentence MUST be compressed and self-contained (descriptions are read without loading files — a pointer cannot work there), MUST name the three micro criteria (no memory/spec impact, no behavior-contract change, single-spot/~1-task edit), MUST carry the default-closed tie-breaker ("when unsure, use fab"), and MUST distinguish the two conflated cases (standalone micro fix → direct edit + commit, no fab; follow-up tweak to a change still in flight → amend that change).

- **GIVEN** a host agent deciding how to route a user request, with only skill descriptions in context
- **WHEN** the request is a trivial single-spot fix (e.g., "fix this 2px offset")
- **THEN** the four routing skills' descriptions tell it fab is not for this — make the edit directly and commit
- **AND** an ambiguous case reads "when unsure, use fab" and routes into the pipeline

### Backstop: Intake-Entry Micro-Change Check

#### R2: `fab-new.md` owns the canonical micro-criteria + backstop text
`src/kit/skills/fab-new.md` MUST gain a **Micro-Change Backstop** section in its Behavior, evaluated BEFORE the Steps 0–9 `_intake` call-site. It is the single owner of the skill-body criteria text (owner-or-pointer rule, `fab/project/code-quality.md`); `fab-proceed.md` points at it. The backstop: if the described change meets ALL THREE micro criteria, `/fab-new` — interactive by posture — confirms inline (e.g., "This looks like a direct fix — handle it without fab, or continue with a tracked change?"); an inline "continue" answer IS the explicit go-ahead and proceeds to Steps 0–9; otherwise the skill stops having created nothing. `_intake.md` itself is NOT modified (the backstop stays at the call sites per the EXTRACTION BOUNDARY — `fab-draft`/`fab-dedupe` are deliberately not gated).

- **GIVEN** `/fab-new fix the 2px offset on the settings button`
- **WHEN** the backstop evaluates the description against the three criteria and all three hold
- **THEN** the skill asks the inline confirm instead of creating a change folder
- **AND** on "continue anyway" it proceeds into Steps 0–9 unchanged; on decline it stops with no pipeline state created

#### R3: `fab-proceed.md` backstop on the create-new path only, zero-prompt preserved
`src/kit/skills/fab-proceed.md` MUST evaluate the micro-change backstop only when the Step 5 dispatch table selects a create-new (`_intake`) row, BEFORE dispatching the create-intake subagent. It MUST point at `fab-new.md`'s owner section for the criteria (never restating them) and state only its posture delta: on trigger it STOPs with a gate-failure-style message (`This looks like a direct fix — handle it without fab unless you want a tracked change. Say "use fab anyway" to proceed.`), asks nothing interactively, performs no fix itself, and adds no argument or flag. An explicit user go-ahead already present in the conversation (e.g., "use fab anyway", or an explicit instruction to run the pipeline for this work) suppresses the backstop on (re-)invocation — conversation is already the skill's inference surface. Resume/active-change rows never trigger it.

- **GIVEN** a conversation whose only substantive content is a trivial single-spot fix, no active change, no relevant draft
- **WHEN** `/fab-proceed` reaches the create-new dispatch row
- **THEN** it stops with the direct-fix message instead of dispatching `_intake`, creating no state
- **AND** re-invocation after the user says "use fab anyway" dispatches the create-new path normally
- **GIVEN** an active change or a clearly relevant unactivated draft
- **WHEN** `/fab-proceed` runs
- **THEN** the backstop never evaluates (those are amend/resume paths)

### Docs: Spec + Sweep Obligations

#### R4: `docs/specs/skills.md` reflects the flow change; changed claims swept repo-wide
The backstop is a flow change for `/fab-proceed` and `/fab-new`: their sections in `docs/specs/skills.md` MUST be updated, and any restatement of the four skills' one-line purposes that the new descriptions or backstop contradict MUST be swept repo-wide (aggregate specs `skills.md`/`glossary.md`/`architecture.md`, user-facing string literals, `*_test.go` comments — the `fab-ff` ↔ `fab-fff` twin class swept together). Memory files (`pipeline/planning-skills.md`, `pipeline/execution-skills.md`) are hydrate's job, not apply's.

- **GIVEN** the apply edits are complete
- **WHEN** the changed phrases and contradicted claims are grepped repo-wide
- **THEN** every occurrence outside `docs/memory/` (hydrate-owned) and `.claude/skills/` (deployed copies) is updated or verified non-contradicted

### Non-Goals

- No third pipeline lane ("nano-flow") — rejected in discussion; no-flow means fab is not invoked at all.
- No `docs/specs/change-types.md` micro-tier taxonomy entry (deferred to a later change).
- No `/fab-setup` host-CLAUDE.md right-sizing rule (deferred).
- No Go/CLI changes, no migration, no `_intake.md` edit, no backstop for `fab-draft`/`fab-dedupe`.

### Design Decisions

#### Owner file for the skill-body micro-criteria text
**Decision**: `fab-new.md` owns the canonical criteria + backstop text; `fab-proceed.md` carries a pointer plus its posture delta only.
**Why**: The backstop is an intake-entry check ("should this be a tracked change at all?") and `/fab-new` is the intake-entry skill; its interactive posture also makes it the natural home of the confirm flow, with `/fab-proceed`'s stop-only form the derived variant.
**Rejected**: `_intake.md` (would leak the backstop to `fab-draft`/`fab-dedupe` or need a second knob against its single-fork design); `_preamble.md` (expands the always-load surface for all skills with routing-only content); `fab-proceed.md` as owner (its file is a lean dispatch table, and the criteria would live far from the interactive flow that uses them most).
*Introduced by*: 260815-slpa-right-size-fab-routing

## Tasks

### Phase 2: Core Implementation

- [x] T001 Append the identical anti-trigger sentence to the `description:` frontmatter of `src/kit/skills/fab-proceed.md`, `src/kit/skills/fab-fff.md`, `src/kit/skills/fab-ff.md`, `src/kit/skills/fab-new.md` <!-- R1 -->
- [x] T002 Add the owner `### Step -1: Micro-Change Backstop` section to `src/kit/skills/fab-new.md` Behavior (before Steps 0–9), with the criteria canon, inline-confirm flow, and Contents/Key Properties touch-ups <!-- R2 -->
- [x] T003 Add the create-new-path backstop to `src/kit/skills/fab-proceed.md` (Step 5 dispatch-decision hook, stop message, conversation go-ahead rule, Error Handling row, Key Properties row) pointing at fab-new.md's owner section <!-- R3 -->

### Phase 3: Integration & Edge Cases

- [x] T004 Update `docs/specs/skills.md` `/fab-proceed` and `/fab-new` sections for the backstop flow change and the four new description anti-triggers <!-- R4 -->
- [x] T005 Repo-wide sweep: grep the changed description phrases and backstop-contradicted claims across `src/kit/`, `docs/specs/`, `README.md`, `docs/site/` (excluding `docs/memory/` and `.claude/skills/`); fix or verify each hit; confirm criteria wording is byte-identical everywhere it appears <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: All four `description:` lines carry the appended anti-trigger with a byte-identical micro-criteria core (verifiable by grepping the core phrase — exactly 4 hits in `src/kit/skills/`)
- [x] A-002 R2: `fab-new.md` contains the single owner backstop section, positioned before the Steps 0–9 call-site, with the inline-confirm flow and no change to `_intake.md`
- [x] A-003 R3: `fab-proceed.md` evaluates the backstop only on create-new rows, stops with the specified message, adds no args/flags, and references fab-new.md's section rather than restating criteria

### Behavioral Correctness

- [x] A-004 R3: `/fab-proceed`'s zero-prompt posture is intact — the backstop is a stop-with-message (like the empty/thin error row), never an interactive prompt; the conversation go-ahead rule is stated
- [x] A-005 R4: `docs/specs/skills.md` `/fab-proceed` + `/fab-new` sections describe the backstop; no aggregate-spec restatement contradicts the new descriptions

### Scenario Coverage

- [x] A-006 R4: Repo-wide grep (excluding `docs/memory/`, `.claude/skills/`) shows zero stale contradicted claims about the four skills' routing scope

### Code Quality

- [x] A-007 Canonical source only: no edits under `.claude/skills/`
- [x] A-008 Owner-or-pointer: the criteria text has exactly one skill-body owner; `fab-proceed.md` points, never restates; frontmatter descriptions are the sanctioned compressed self-contained form
- [x] A-009 Sibling sweep done up front: `fab-ff` ↔ `fab-fff` twins edited together; identical wording verified by grep, not by eye

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before hydrate

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Owner file = `fab-new.md`; `fab-proceed.md` points (resolves intake's deferred row 10) | Backstop is an intake-entry check and fab-new is the intake-entry skill; rejected candidates carry the structural costs the intake named | S:55 R:75 A:70 D:60 |
| 2 | Confident | Backstop placement in fab-new is a `### Step -1` before the Steps 0–9 call-site heading | "Before Steps 0–9" is the intake's stated trigger point; a pre-step numbered -1 keeps the existing step numbering untouched | S:70 R:85 A:80 D:75 |
| 3 | Confident | fab-proceed's go-ahead suppression keys on explicit conversation instruction (e.g., "use fab anyway", or the user explicitly asked for the pipeline), evaluated at the create-new row | Intake row 6 (Confident) fixed the mechanic; the evaluation point follows from the trigger point | S:60 R:75 A:75 D:65 |
| 4 | Confident | Anti-trigger sentence final wording authored at apply within the intake's candidate shape, criteria core identical across all four descriptions | Intake row 9 granted authoring latitude within a fixed shape | S:65 R:85 A:75 D:60 |

4 assumptions (0 certain, 4 confident, 0 tentative).
