# Plan: Native Apply-Worker Resume

**Change**: 260808-tv3g-native-apply-worker-resume
**Intake**: `intake.md`

## Requirements

### Skills: Worker-continuation mechanics (`_preamble.md`, owner)

#### R1: `_preamble.md` § Subagent Dispatch SHALL own the native-arm worker-continuation mechanics

`src/kit/skills/_preamble.md` § Subagent Dispatch MUST gain a subsection (**Worker Continuation (native arm)**) that states, as the single canonical source, all five mechanics: worker **naming** (`apply-{id}`, `{id}` = 4-char change ID), **continuation** (resume via SendMessage under the same block contract, carrying triaged findings + the chosen rework action, without re-carrying the standard subagent context files), the **fallback rule** (unreachable handle ⇒ fresh dispatch per the existing procedure, including the full dispatch-prompt obligations), **profile fixity** (`fab resolve-agent apply --alias` is NOT re-run on the resume path), and the **scope guard** (apply worker inside the auto-rework loop only; review/hydrate/other stages and the whole CLI-adapter branch excluded).

- **GIVEN** an agent reading `_preamble.md` § Subagent Dispatch
- **WHEN** it needs to know how a rework cycle reaches the apply worker
- **THEN** the subsection answers naming, continuation, fallback, profile fixity, and scope without consulting any other file
- **AND** no other skill file restates those mechanics (owner-or-pointer convention, `code-quality.md`)

#### R2: Continuation SHALL be an optimization with a mandatory fresh-dispatch fallback

The fallback MUST be stated as load-bearing: continuation is never a correctness dependency. The enumerated unreachable cases MUST include (a) the orchestrator session was resumed/restarted so handles did not survive, (b) the harness has no named-agent/SendMessage capability, (c) the send errors, and (d) the worker was never named (e.g. the stage went through the CLI adapter). A fresh fallback dispatch MUST re-establish the name for subsequent cycles where the capability exists.

- **GIVEN** a rework cycle whose `apply-{id}` worker is unreachable for any of the four reasons
- **WHEN** the orchestrator re-dispatches apply
- **THEN** it runs the existing Stage Dispatch Procedure verbatim — same prompt, same dispatch-prompt obligations, same transitions
- **AND** the observable pipeline behavior is identical to today's (Constitution III idempotency is not conditioned on a live handle)

#### R3: A continued worker SHALL remain bound by the block contract

The continuation prompt MUST restate the block contract — results only, no `fab status` **transition** commands (`start`/`advance`/`finish`/`reset`/`fail`/`skip`), terminal `fab status refresh` — while deliberately NOT re-carrying the standard subagent context files (the worker already holds them).

- **GIVEN** a resumed `apply-{id}` worker finishing a rework cycle
- **WHEN** it completes its edits
- **THEN** it returns its result and ends with `fab status refresh`, running no transition command
- **AND** the orchestrator still owns every `finish`/`fail`/`reset`

### Skills: Auto-Rework Loop wiring (`_pipeline.md`, pointer)

#### R4: `_pipeline.md` SHALL name the apply worker at Step 1 and resume it first at item 3

Step 1's apply dispatch MUST name the worker `apply-{id}` on the native branch, pointing at `_preamble.md` § Worker Continuation for the mechanics. Auto-Rework Loop item 3 MUST become resume-first: continue the reachable native-arm `apply-{id}` worker from this orchestrator session via SendMessage with the item-2 rework instructions; otherwise run the Stage Dispatch Procedure for `apply` with the Step 1 target (verbatim current behavior).

- **GIVEN** a rework cycle whose `apply-{id}` worker from this orchestrator session is reachable
- **WHEN** item 3 runs
- **THEN** the orchestrator sends the rework instructions to that worker instead of spawning a fresh one
- **AND** on success it still runs `fab status finish <change> apply {driver}`

#### R5: The status choreography and cycle-count invariant SHALL be semantically unchanged

Items 1 (fail+reset pair), 2 (triage), 4 (fresh re-review), and 5 (verdict handling) MUST stay byte-identical in semantics, and item 3's `finish apply` MUST remain the one counted review `→ active` transition per cycle.

- **GIVEN** a run of N rework cycles that resumes the apply worker every cycle
- **WHEN** the run completes
- **THEN** `stage_metrics.review.iterations` equals `N + 1`, exactly as with fresh dispatches
- **AND** item 4's "Never reuse a prior review worker's context" rule is untouched

#### R6: The release point SHALL be stated as a one-line rule

`_pipeline.md` MUST state that the orchestrator stops continuing the apply worker after `fab status finish <change> review {driver}` (review pass) or at the exhaustion Stop — release is passive (the handle is simply never used again); no teardown call exists or is needed, and hydrate always dispatches fresh.

- **GIVEN** review passes (or the loop exhausts)
- **WHEN** the pipeline proceeds to hydrate (or stops)
- **THEN** no further message is sent to `apply-{id}` and hydrate dispatches a fresh worker

### Specs: constitution-required mirrors and sweep

#### R7: The two SPEC mirrors SHALL reflect the skill edits

`docs/specs/skills/SPEC-_preamble.md` MUST record the new Worker Continuation subsection in its Summary/Subsection-Inventory/Flow surfaces; `docs/specs/skills/SPEC-_pipeline.md` MUST update its Per-Cycle Rework Choreography item 3 (currently asserting a **fresh** apply subagent) and its Flow/Sub-agents surfaces.

- **GIVEN** a reader of either SPEC mirror
- **WHEN** they read the rework choreography or the dispatch inventory
- **THEN** the resume-first apply path, fallback, profile fixity, and release point are described consistently with the skill sources
- **AND** no mirror still claims the rework apply worker is always fresh

#### R8: `docs/specs/harness-adapters.md` SHALL document continuation as a native-adapter property

The native Agent-tool adapter description MUST gain a continuation note (apply-worker reuse across rework cycles, fallback-to-fresh), and the spec MUST record that the headless adapter is deliberately non-resumable and the pane adapter is unchanged (its resume is a separate follow-up change).

- **GIVEN** a reader comparing the three adapters
- **WHEN** they read the native adapter section
- **THEN** apply-worker continuation is named as a native-only capability with a mandatory fresh fallback
- **AND** the other two adapters carry no resumability claim

#### R9: The aggregate-spec and twin sweep SHALL leave no stale "fresh apply worker" claim

`docs/specs/skills.md`, `docs/specs/glossary.md`, `docs/specs/architecture.md`, the `fab-ff`/`fab-fff` twin wrappers, and `fab-adopt.md` MUST be swept: any restated "re-dispatch apply fresh each cycle" claim is updated; review fresh-worker claims are preserved verbatim.

- **GIVEN** a repo-wide grep for restated apply-rework-dispatch prose
- **WHEN** the sweep completes
- **THEN** every occurrence in the class is consistent with the new behavior
- **AND** every *review* fresh-worker claim is unchanged

### Non-Goals

- No reviewer scope-brief / findings-forwarding change — discussed, not decided.
- No headless-CLI resume, no session-ID tracking, no provider `resume_command` grammar — user-decided out.
- No pane-mode resume — separate follow-up change (needs `fab dispatch resume`, stage-aware reap, contract amendment).
- No Go, CLI, config, or test changes; no `/fab-continue` manual-rework-menu change.
- No `docs/memory/` edits — memory hydration is the hydrate stage's job (`intake.md` § Affected Memory drives it).

### Design Decisions

#### Continuation mechanics live in `_preamble.md`, wiring in `_pipeline.md`
**Decision**: `_preamble.md` § Subagent Dispatch owns naming/continuation/fallback/profile-fixity/scope; `_pipeline.md` carries only the pointer plus the Step-1 naming and item-3 resume-first wiring.
**Why**: The owner-or-pointer convention in `fab/project/code-quality.md` — a skill may state a rule it owns or point at the owner, never both; restated copies are this repo's documented drift mechanism.
**Rejected**: Stating the mechanics inside the Auto-Rework Loop (the only consumer today) — it would have to be restated the moment a second consumer appears, and `/fab-adopt` already partially consumes that loop.
*Introduced by*: 260808-tv3g-native-apply-worker-resume

#### Resume skips `fab resolve-agent` re-resolution
**Decision**: The resume path does not re-run `fab resolve-agent apply --alias`; the profile is fixed at first dispatch and re-resolved only on a fresh dispatch (initial or fallback).
**Why**: A continued worker's model cannot change mid-session, so re-resolving would surface a value that cannot be applied — the opposite of the surfacing rule's purpose (make a mis-resolution *visible*).
**Rejected**: Re-resolving every cycle for uniformity — it would print a resolution the cycle demonstrably does not honor.
*Introduced by*: 260808-tv3g-native-apply-worker-resume

#### Fallback is the contract, not an error path
**Decision**: Every unreachable-handle case degrades silently to today's fresh dispatch, with the full dispatch-prompt obligations.
**Why**: Constitution III (idempotent operations) — correctness must not depend on a session surviving; the worst case of a broken resume path is the status quo.
**Rejected**: Treating an unreachable handle as a pipeline failure — it would convert a harmless optimization miss into a stop.
*Introduced by*: 260808-tv3g-native-apply-worker-resume

## Tasks

### Phase 1: Core Implementation

- [x] T001 <!-- rework: cycle 2 — continuation contract lacks the re-read-from-disk instruction for orchestrator-edited artifacts (must-fix); reachability clause (nice-to-have) --><!-- rework: cycle 1 — obligation-2 owner left unqualified (must-fix 1); naming re-establishment stated categorically (nice-to-have 1) --> Add the `### Worker Continuation (native arm)` subsection to `src/kit/skills/_preamble.md` § Subagent Dispatch (after § Per-Stage Model Resolution, before § CLI-Adapter Dispatch), stating naming, continuation, fallback rule, block-contract restatement, profile fixity, and scope guard <!-- R1, R2, R3 -->
- [x] T002 <!-- rework: cycle 1 — item 3 restates the four-case fallback enumeration owned by _preamble.md (owner-or-pointer violation, must-fix 2) --> Wire `src/kit/skills/_pipeline.md`: name the worker `apply-{id}` at Step 1 (native branch, pointer to `_preamble.md` § Worker Continuation), make Auto-Rework Loop item 3 resume-first with verbatim fresh-dispatch fallback, and add the one-line release rule after the review-pass/exhaustion boundary <!-- R4, R5, R6 -->

### Phase 2: Spec Mirrors (constitution-required)

- [x] T003 [P] <!-- rework: cycle 2 — mirror the re-read clause into the Worker Continuation summary + Flow continuation node --> Update `docs/specs/skills/SPEC-_preamble.md` — record the Worker Continuation subsection in the Summary and the `## Flow` Subagent Dispatch branch <!-- R7 -->
- [x] T004 [P] Update `docs/specs/skills/SPEC-_pipeline.md` — rewrite Per-Cycle Rework Choreography item 3 (drop the unconditional "fresh apply subagent" claim), add the release rule, and refresh the `## Flow` + `### Sub-agents` surfaces <!-- R7 -->
- [x] T005 [P] <!-- rework: cycle 2 — mirror the re-read clause into §1 continuation paragraph; fix the amendment-note ordering claim (should-fix); rewrap :94 (nice-to-have) --><!-- rework: cycle 1 — obligations section not amended with the continuation carve-out (must-fix 1); opening 'fresh context' framing contradicts the amendment (should-fix 1) --> Update `docs/specs/harness-adapters.md` — add the native-adapter continuation note (apply-worker reuse + fallback-to-fresh) and record headless as deliberately non-resumable / pane unchanged <!-- R8 -->

### Phase 3: Sibling Sweep & Verification

- [x] T006 <!-- rework: cycle 2 — mirror the re-read clause into glossary.md § Worker continuation; rewrap stage-models.md:46 (nice-to-have) --> Sweep `docs/specs/skills.md`, `docs/specs/glossary.md`, `docs/specs/architecture.md`, `src/kit/skills/fab-ff.md`, `src/kit/skills/fab-fff.md`, `src/kit/skills/fab-adopt.md` (and `docs/specs/skills/SPEC-fab-adopt.md`) for restated apply-rework-dispatch prose; update every hit, leave review fresh-worker claims verbatim <!-- R9 -->
- [x] T007 Verify the change: grep repo-wide that no stale "fresh apply worker per cycle" claim remains, that no file under `.claude/skills/` was edited, and that the diff is markdown-only (no `.go`, config, or template changes) <!-- R1, R9 -->

## Execution Order

- T001 blocks T002 (the pointer must have an owner to point at) and T003/T004
- T003, T004, T005 are independent of one another
- T006 runs after T001–T005; T007 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/_preamble.md` § Subagent Dispatch contains a Worker Continuation subsection stating naming (`apply-{id}`), continuation via SendMessage, the fallback rule, profile fixity, and the scope guard
- [x] A-002 R2: The fallback rule enumerates all four unreachable cases and states that a fresh fallback dispatch re-establishes the name
- [x] A-003 R3: The continuation prompt's obligations are stated as results-only + no transition commands + terminal `fab status refresh`, with the standard context files explicitly not re-carried
- [x] A-004 R4: `src/kit/skills/_pipeline.md` Step 1 names the worker and item 3 is resume-first with a verbatim fresh-dispatch fallback
- [x] A-005 R6: `_pipeline.md` carries a one-line release rule (stop continuing after review pass or the exhaustion Stop; hydrate always fresh; no teardown call)
- [x] A-006 R7: `docs/specs/skills/SPEC-_preamble.md` and `docs/specs/skills/SPEC-_pipeline.md` both reflect the skill edits
- [x] A-007 R8: `docs/specs/harness-adapters.md` describes continuation on the native adapter and records headless as non-resumable / pane unchanged

### Behavioral Correctness

- [x] A-008 R5: `_pipeline.md` items 1, 2, 4, 5 and the cycle-count invariant are semantically unchanged — item 3's `finish apply` remains the single counted review `→ active` transition per cycle
- [x] A-009 R5: Item 4's "Never reuse a prior review worker's context" rule is present and unmodified
- [x] A-010 R1: The mechanics appear in exactly one skill file — `_pipeline.md` points at `_preamble.md` rather than restating (owner-or-pointer convention)

### Scenario Coverage

- [x] A-011 R2: The prose makes the unreachable-handle path indistinguishable from today's behavior (same procedure, same dispatch-prompt obligations, same transitions)
- [x] A-012 R4: The resume path's inputs are named — the triaged findings and the item-2 rework action

### Edge Cases & Error Handling

- [x] A-013 R1: The scope guard excludes review workers, hydrate, all other stages, and the entire CLI-adapter (`dispatch=` present) branch, naming headless-by-decision and pane-as-separate-change
- [x] A-014 R2: The prose states that continuation failure is never a pipeline failure (Constitution III)

### Removal Verification

- [x] A-015 R7: No SPEC mirror still asserts the rework apply worker is unconditionally fresh (`SPEC-_pipeline.md` item 3)
- [x] A-016 R9: A repo-wide grep finds no remaining restated "fresh apply worker each cycle" claim; every *review* fresh-worker claim is unchanged

### Code Quality

- [x] A-017 Pattern consistency: New prose matches surrounding skill/spec voice — table-or-bullet structure, RFC-2119 keywords where normative, existing cross-reference style
- [x] A-018 No unnecessary duplication: The continuation mechanics are stated once; consumers carry pointers only
- [x] A-019 Canonical source only: No file under `.claude/skills/` was edited (deployed copies; Constitution V and code-quality.md § Anti-Patterns)
- [x] A-020 Sibling & mirror sweeps: Every `src/kit/skills/*.md` edit carries its `docs/specs/skills/SPEC-*.md` mirror update in this same change, and the aggregate-spec class was swept up front
- [x] A-021 Markdown-only: The diff contains no `.go`, config, template, or migration changes — the CLI/test constitution constraints are not triggered

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. The only path it could plausibly have superseded — the fresh apply dispatch at `_pipeline.md` Auto-Rework Loop item 3 — is retained verbatim as the mandatory fallback, so no file, section, or rule becomes unused.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The new subsection is placed after § Per-Stage Model Resolution and before § CLI-Adapter Dispatch in `_preamble.md` | Profile fixity is stated in terms of the resolution section immediately above it, and the scope guard forward-references the CLI branch that follows; trivially relocatable | S:60 R:95 A:85 D:75 |
| 2 | Certain | Apply performs no `docs/memory/` edits for this change | The intake's Affected Memory list is consumed by the hydrate stage; apply owns source/spec artifacts only | S:90 R:90 A:95 D:95 |
| 3 | Certain | `fab sync` is not run as part of apply | `.claude/skills/` is a gitignored deployment of the *installed* kit version, not of `src/kit/`; syncing would overwrite nothing useful and edits there are forbidden | S:85 R:90 A:95 D:90 |
| 4 | Confident | `docs/specs/skills.md`'s existing "spawn fresh sub-agent for re-review" wording needs no rewrite — its "fresh" binds to re-review only | Reading the sentence, `fresh` qualifies the review sub-agent; the apply half says only "re-apply" | S:70 R:90 A:85 D:70 |

4 assumptions (2 certain, 2 confident, 0 tentative).
