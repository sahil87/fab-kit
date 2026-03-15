# Spec: Redesign FF and FFF Pipeline Scopes

**Change**: 260314-q5p9-redesign-ff-fff-scopes
**Created**: 2026-03-14
**Affected memory**: `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Changing the Go binary (`fab-go`) — this is a docs/skills-only change
- Modifying `/fab-continue` behavior — it has no gates to bypass
- Changing auto-clarify behavior — both skills keep identical auto-clarify between spec and tasks
- Changing the auto-rework loop — both skills keep the same 3-cycle cap with escalation rule

## Pipeline Skills: Scope Redesign

### Requirement: fab-ff SHALL stop at hydrate

`/fab-ff` SHALL execute stages intake through hydrate only. The pipeline SHALL NOT include ship or review-pr stages.

The pipeline steps SHALL be:
1. Intake gate check
2. Generate `spec.md` + spec gate check + auto-clarify
3. Generate `tasks.md` + auto-clarify
4. Generate `checklist.md`
5. Update `.status.yaml` (planning complete)
6. Implementation (apply)
7. Review (with auto-rework loop, max 3 cycles)
8. Hydrate

#### Scenario: fab-ff completes at hydrate
- **GIVEN** a change with `intake.md` and indicative confidence >= 3.0
- **WHEN** `/fab-ff` runs to completion
- **THEN** the pipeline ends after hydrate
- **AND** `progress.hydrate` is `done`
- **AND** `progress.ship` remains `pending`
- **AND** the output ends with `--- Hydrate ---` section (no `--- Ship ---` or `--- Review-PR ---`)

#### Scenario: fab-ff resumability checks hydrate as terminal
- **GIVEN** a change where `progress.hydrate` is `done`
- **WHEN** `/fab-ff` is re-invoked
- **THEN** the pipeline reports "Pipeline complete." without attempting ship or review-pr

### Requirement: fab-fff SHALL extend through ship and review-pr

`/fab-fff` SHALL execute stages intake through review-pr. This is the distinguishing feature — fff goes further than ff.

The pipeline steps SHALL be:
1. Intake gate check
2. Generate `spec.md` + spec gate check + auto-clarify
3. Generate `tasks.md` + auto-clarify
4. Generate `checklist.md`
5. Update `.status.yaml` (planning complete)
6. Implementation (apply)
7. Review (with auto-rework loop, max 3 cycles)
8. Hydrate
9. Ship (dispatch `/git-pr`)
10. Review-PR (dispatch `/git-pr-review`)

#### Scenario: fab-fff extends through ship and review-pr
- **GIVEN** a change with `intake.md` and indicative confidence >= 3.0
- **WHEN** `/fab-fff` runs to completion
- **THEN** the pipeline ends after review-pr
- **AND** `progress.review-pr` is `done`
- **AND** the output includes `--- Ship ---` and `--- Review-PR ---` sections

#### Scenario: fab-fff resumability checks review-pr as terminal
- **GIVEN** a change where `progress.review-pr` is `done`
- **WHEN** `/fab-fff` is re-invoked
- **THEN** the pipeline reports "Pipeline complete."

### Requirement: fab-fff SHALL NOT frontload questions

`/fab-fff` SHALL NOT include a "Frontload All Questions" step. The current Step 1 (SRAD batch question collection) SHALL be removed entirely. `/fab-fff` SHALL proceed directly to spec generation, matching `/fab-ff` behavior.

#### Scenario: fab-fff proceeds without questions
- **GIVEN** a change intake with Unresolved SRAD decisions
- **WHEN** `/fab-fff` is invoked
- **THEN** the pipeline proceeds directly to spec generation
- **AND** no batch question round is presented to the user

### Requirement: fab-fff SHALL use identical confidence gates as fab-ff

`/fab-fff` SHALL include both confidence gates:
1. **Intake gate**: `fab score --check-gate --stage intake` with fixed threshold >= 3.0
2. **Spec gate**: `fab score --check-gate` with per-type dynamic threshold

These are identical to the gates in `/fab-ff`.

#### Scenario: fab-fff intake gate blocks low-confidence change
- **GIVEN** a change with indicative confidence 2.0
- **WHEN** `/fab-fff` is invoked
- **THEN** the pipeline STOPS with: "Indicative confidence is 2.0 of 5.0 (need >= 3.0). Run /fab-clarify to resolve, then retry."

#### Scenario: fab-fff spec gate blocks low-confidence spec
- **GIVEN** a change with spec confidence below the per-type threshold
- **WHEN** `/fab-fff` reaches the spec gate
- **THEN** the pipeline STOPS with score, threshold, and guidance

### Requirement: Both skills SHALL accept a --force flag

Both `/fab-ff` and `/fab-fff` SHALL accept an optional `--force` flag that bypasses all confidence gates.

When `--force` is passed:
- The intake gate check SHALL be skipped entirely
- The spec gate check SHALL be skipped entirely
- All other behavior SHALL remain unchanged (auto-clarify, rework loop, etc.)
- The output header SHALL include "(force mode -- gates bypassed)"

#### Scenario: --force bypasses intake gate
- **GIVEN** a change with indicative confidence 1.5
- **WHEN** `/fab-ff --force` is invoked
- **THEN** the pipeline proceeds past the intake gate without checking
- **AND** the output header reads `/fab-ff --force -- (force mode -- gates bypassed)`

#### Scenario: --force bypasses spec gate
- **GIVEN** a change where spec confidence is below the per-type threshold
- **WHEN** `/fab-fff --force` reaches the spec gate point
- **THEN** the pipeline proceeds without checking the spec gate

#### Scenario: --force does not affect non-gate behavior
- **GIVEN** `/fab-ff --force` is invoked
- **WHEN** auto-clarify finds blocking issues
- **THEN** the pipeline still bails (--force only bypasses confidence gates, not auto-clarify)

## Pipeline Skills: Behavioral Alignment

### Requirement: Both skills SHALL use auto-rework loop on review failure

Both `/fab-ff` and `/fab-fff` SHALL use the autonomous rework loop on review failure:
- Max 3 rework cycles
- Escalation after 2 consecutive fix-code attempts
- STOP after 3 failed cycles with per-cycle summary

This is unchanged from the current behavior — both `/fab-ff` and `/fab-fff` already use autonomous rework with the same parameters. The redesign preserves this as-is.

#### Scenario: Auto-rework on review failure in fab-ff
- **GIVEN** `/fab-ff` reaches review and the sub-agent reports must-fix findings
- **WHEN** the review fails
- **THEN** the agent autonomously triages findings and selects a rework path
- **AND** the cycle repeats up to 3 times

### Requirement: Both skills SHALL use identical auto-clarify

Both `/fab-ff` and `/fab-fff` SHALL interleave auto-clarify after spec generation and after tasks generation using `[AUTO-MODE]` prefix. Same bail-on-blocking behavior for both.

#### Scenario: Auto-clarify in fab-fff matches fab-ff
- **GIVEN** `/fab-fff` generates `spec.md`
- **WHEN** auto-clarify is dispatched
- **THEN** it uses `[AUTO-MODE]` and returns `{resolved, blocking, non_blocking}`
- **AND** the pipeline bails if `blocking > 0`

## Preamble: SRAD Autonomy Table Update

### Requirement: _preamble.md SHALL reflect new skill postures

The SRAD Skill-Specific Autonomy Levels table in `_preamble.md` SHALL be updated:

| Aspect | fab-ff | fab-fff |
|--------|--------|---------|
| **Posture** | Gated on confidence; stops at hydrate | Gated on confidence; extends through ship + review-pr |
| **Interruption budget** | 0 (interactive rework on failure) | 0 (interactive rework on failure) |
| **Escape valve** | `/fab-clarify` | `/fab-clarify` |
| **Recomputes confidence?** | No | No |

The `fab-ff` and `fab-fff` columns SHALL be updated as shown. The `fab-new` and `fab-continue` columns remain unchanged. The table SHALL remove the "from-spec, gated" and "full pipeline" characterizations. Both ff and fff are now "gated" — the difference is scope only.

#### Scenario: Preamble SRAD table consistency
- **GIVEN** the updated `_preamble.md`
- **WHEN** a reader compares fab-ff and fab-fff rows
- **THEN** the only difference visible is scope (hydrate vs review-pr)
- **AND** both show "Gated on confidence" posture

### Requirement: _preamble.md Confidence Scoring section SHALL reflect both skills having gates

The Confidence Scoring section in `_preamble.md` currently states "/fab-ff has two confidence gates. /fab-fff has no confidence gates" and "/fab-fff does not gate or recompute." These statements SHALL be updated to reflect that both `/fab-ff` and `/fab-fff` have identical confidence gates (intake gate and spec gate). The `--force` flag bypass SHALL also be mentioned.

#### Scenario: Preamble Confidence Scoring reflects both skills
- **GIVEN** the updated `_preamble.md` Confidence Scoring section
- **WHEN** the gate description text is read
- **THEN** both `/fab-ff` and `/fab-fff` are described as having intake and spec gates
- **AND** the `--force` bypass is documented

## Documentation Updates

### Requirement: user-flow.md diagrams SHALL reflect new scopes

`docs/specs/user-flow.md` SHALL be updated:
- **Diagram 2** ("The Same Flow, With Fab"): `/fab-ff` arrow SHALL point to hydrate (H). `/fab-fff` arrow SHALL point to review-pr (RP). Both labels SHALL mention "confidence-gated".
- **Diagram 3** ("Change Flow"): FF node SHALL show "fast-forward through hydrate". FFF node SHALL show "fast-forward further through review-pr". Both SHALL note confidence gates.
- **Diagram 4** ("Change State Diagram"): `intake --> hydrate` transition labeled `/fab-ff (fast-forward, confidence-gated)`. `intake --> review_pr` transition labeled `/fab-fff (fast-forward-further, confidence-gated)`.

#### Scenario: Diagram 2 reflects scope difference
- **GIVEN** the updated `user-flow.md`
- **WHEN** Diagram 2 is rendered
- **THEN** fab-ff arrow ends at hydrate (not review-pr)
- **AND** fab-fff arrow ends at review-pr
- **AND** both mention "confidence-gated"

### Requirement: skills.md SHALL reflect redesigned skills

`docs/specs/skills.md` SHALL update the `/fab-ff` and `/fab-fff` sections:
- `/fab-ff` description: stops at hydrate, with confidence gates and `--force` flag
- `/fab-fff` description: extends through review-pr, with same confidence gates, no frontloaded questions, and `--force` flag
- "Key difference" text updated to reflect scope-only differentiation

#### Scenario: skills.md key difference text
- **GIVEN** the updated `skills.md`
- **WHEN** the `/fab-fff` section is read
- **THEN** the "Key difference" text explains: fff extends through ship and review-pr (ff stops at hydrate), both have identical gates, no frontloaded questions

### Requirement: Per-skill SPEC files SHALL be updated

- `docs/specs/skills/SPEC-fab-ff.md`: Remove Steps 8 (Ship) and 9 (Review-PR) from the flow diagram. Update summary to mention "through hydrate" scope. Add `--force` flag. Remove /git-pr and /git-pr-review from sub-agents table.
- `docs/specs/skills/SPEC-fab-fff.md`: Add confidence gates to flow. Remove frontload questions step. Add `--force` flag. Update summary to reflect gates + scope.

#### Scenario: SPEC-fab-ff flow ends at hydrate
- **GIVEN** the updated `SPEC-fab-ff.md`
- **WHEN** the flow diagram is read
- **THEN** the last step is Step 7: Hydrate
- **AND** no Step 8 or Step 9 exists

### Requirement: srad.md autonomy table SHALL be updated

`docs/specs/srad.md` SHALL update the Skill-Specific Autonomy Levels table to match the `_preamble.md` changes. The `/fab-ff` and `/fab-fff` columns SHALL reflect the new postures.

#### Scenario: srad.md table matches preamble
- **GIVEN** the updated `srad.md`
- **WHEN** the autonomy table is compared with `_preamble.md`
- **THEN** the fab-ff and fab-fff rows are consistent

## Schema Update

### Requirement: workflow.yaml commands fields SHALL be updated

`fab/.kit/schemas/workflow.yaml` SHALL be updated:
- `intake` stage: add `fab-ff` and `fab-fff` to `commands` array
- `tasks` stage: remove `fab-ff` from `commands` array

Both ff and fff start from intake and run through their respective endpoints.

#### Scenario: intake stage lists both pipeline skills
- **GIVEN** the updated `workflow.yaml`
- **WHEN** the `intake` stage is inspected
- **THEN** `commands` includes `fab-ff` and `fab-fff`

#### Scenario: tasks stage does not list fab-ff
- **GIVEN** the updated `workflow.yaml`
- **WHEN** the `tasks` stage is inspected
- **THEN** `commands` does not include `fab-ff`

## Skill File: fab-ff.md Updates

### Requirement: fab-ff.md frontmatter and purpose SHALL be updated

The frontmatter description SHALL change to reflect "through hydrate" scope. The Purpose section SHALL describe the pipeline as intake through hydrate (not through review-pr). The `--force` flag SHALL be documented in Arguments.

#### Scenario: fab-ff.md purpose describes hydrate endpoint
- **GIVEN** the updated `fab-ff.md`
- **WHEN** the Purpose section is read
- **THEN** it describes "intake -> spec -> tasks -> apply -> review -> hydrate"
- **AND** does not mention ship or review-pr

### Requirement: fab-ff.md SHALL remove ship and review-pr steps

Steps 8 (Ship) and 9 (Review-PR) SHALL be removed. The Resumability section SHALL check `hydrate: done` (not `review-pr: done`). The Output format SHALL end at `--- Hydrate ---`. The Error Handling table SHALL remove ship and review-pr rows.

#### Scenario: fab-ff.md output format ends at hydrate
- **GIVEN** the updated `fab-ff.md`
- **WHEN** the Output section is read
- **THEN** the last section is `--- Hydrate ---`
- **AND** `--- Ship ---` and `--- Review-PR ---` do not appear

## Skill File: fab-fff.md Updates

### Requirement: fab-fff.md SHALL add confidence gates and remove frontloaded questions

The frontmatter description SHALL reflect confidence gates and no frontloaded questions. The Purpose section SHALL describe gates. Step 1 (Frontload All Questions) SHALL be removed. Pre-flight SHALL add intake gate. Spec generation SHALL add spec gate. The `--force` flag SHALL be documented in Arguments.

#### Scenario: fab-fff.md has no frontload step
- **GIVEN** the updated `fab-fff.md`
- **WHEN** the Behavior section is read
- **THEN** no "Frontload All Questions" step exists
- **AND** the first behavioral step is spec generation

#### Scenario: fab-fff.md pre-flight includes intake gate
- **GIVEN** the updated `fab-fff.md`
- **WHEN** the Pre-flight section is read
- **THEN** step 3 reads: "Intake gate: Run `fab score --check-gate --stage intake`"

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | ff stops at hydrate, fff extends through ship + review-pr | Confirmed from intake #1 — user explicitly proposed and confirmed | S:95 R:70 A:95 D:95 |
| 2 | Certain | Both use identical confidence gates | Confirmed from intake #2 — user confirmed "the ones in fab-ff right now" | S:95 R:60 A:90 D:95 |
| 3 | Certain | Drop frontloaded questions from fff | Confirmed from intake #3 — user said "Drop this entirely" | S:95 R:75 A:90 D:95 |
| 4 | Certain | --force flag bypasses gates on both skills | Confirmed from intake #4 — user confirmed "--force flag is ok" | S:90 R:80 A:85 D:90 |
| 5 | Certain | Naming: ff = fast-forward, fff = fast-forward-further | Confirmed from intake #5 — user confirmed naming philosophy | S:90 R:90 A:90 D:95 |
| 6 | Confident | Auto-clarify behavior is identical in both | Confirmed from intake #6 — no objection when raised; spec confirms same [AUTO-MODE] dispatch | S:75 R:80 A:85 D:80 |
| 7 | Confident | No changes to Go binary, scripts, or templates | Confirmed from intake #7 — scope is docs/skills only | S:75 R:85 A:80 D:85 |
| 8 | Certain | --force only on ff/fff, not on /fab-continue | Confirmed from intake #8 — user confirmed scope | S:95 R:85 A:90 D:95 |
| 9 | Confident | Auto-rework loop is identical in both (same 3-cycle cap, same escalation) | Current ff behavior is the baseline; fff previously had same behavior; user didn't specify any difference | S:70 R:75 A:85 D:85 |
| 10 | Certain | Diagram updates follow the scope change (ff→hydrate, fff→review-pr) | Directly derived from the scope split — no alternative interpretation | S:95 R:85 A:95 D:95 |

10 assumptions (7 certain, 3 confident, 0 tentative, 0 unresolved).
