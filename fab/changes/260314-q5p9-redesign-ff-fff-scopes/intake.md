# Intake: Redesign FF and FFF Pipeline Scopes

**Change**: 260314-q5p9-redesign-ff-fff-scopes
**Created**: 2026-03-14
**Status**: Draft

## Origin

> Redesign fab-ff and fab-fff pipeline scopes: ff stops at hydrate (with confidence gates), fff extends ff through ship + review-pr (same gates). Both use identical confidence gates. Drop frontloaded questions from fff. Add --force flag to bypass gates on both. Update user-flow.md diagrams and specs to reflect new philosophy.

Conversational. User proposed the redesign during a `/fab-discuss` session, iterating through three rounds:
1. Initial proposal: ff stops at hydrate, fff goes through ship + review-pr, both share gates
2. Confirmed: drop frontloaded questions entirely, naming is "fast-forward" vs "fast-forward-further"
3. Confirmed: `--force` flag is acceptable for bypassing gates

## Why

Currently `fab-ff` and `fab-fff` differ in two orthogonal ways: (1) gating philosophy (ff has confidence gates, fff doesn't) and (2) some behavioral differences (fff frontloads questions, interleaves auto-clarify differently). This makes the mental model unnecessarily complex — users must remember which pipeline does what along two axes.

The redesign simplifies to a single axis: **scope**. ff = "do the work" (through hydrate), fff = "do the work AND ship it" (through review-pr). Both use the same confidence gates. This is clearer, more memorable, and eliminates ~90% of the duplication between the two skills.

If we don't fix this, users will continue to be confused about when to use ff vs fff, and the two skill files will remain nearly identical with subtle behavioral differences that are hard to maintain.

## What Changes

### 1. `fab-ff` skill — scope reduction

Current: intake → spec → tasks → apply → review → hydrate → ship → review-pr (with confidence gates)
New: intake → spec → tasks → apply → review → hydrate (with confidence gates)

- Remove Steps 8 (Ship) and 9 (Review-PR) from `fab/.kit/skills/fab-ff.md`
- Update output format to end at hydrate
- Update error handling table (remove ship/review-pr rows)
- Add `--force` flag support to bypass both intake and spec confidence gates
- Keep auto-clarify behavior between spec and tasks (unchanged)
- Keep auto-rework loop on review failure (unchanged, 3-cycle cap)

### 2. `fab-fff` skill — unification with ff + ship/review-pr

Current: intake → spec → tasks → apply → review → hydrate → ship → review-pr (no gates, frontloaded questions)
New: intake → spec → tasks → apply → review → hydrate → ship → review-pr (with confidence gates, no frontloaded questions)

- Remove Step 1 (Frontload All Questions) from `fab/.kit/skills/fab-fff.md`
- Add confidence gates: intake gate (>= 3.0) and spec gate (per-type threshold) — same as ff
- Add `--force` flag support to bypass gates
- Keep auto-clarify between spec and tasks (same as ff)
- Keep auto-rework loop (same as ff, 3-cycle cap)
- Keep ship (Step 9) and review-pr (Step 10) — these become the distinguishing feature
- Update driver argument from "fab-fff" to "fab-fff" (unchanged)

### 3. `--force` flag on both skills

When `--force` is passed:
- Skip the intake gate check entirely (no `fab score --check-gate --stage intake`)
- Skip the spec gate check entirely (no `fab score --check-gate`)
- All other behavior unchanged (auto-clarify, rework loop, etc.)
- Output header includes "(force mode — gates bypassed)"

### 4. `_preamble.md` updates

Update the SRAD Autonomy Framework table to reflect new skill postures:

| Aspect | fab-ff | fab-fff |
|--------|--------|---------|
| **Posture** | Gated on confidence; stops at hydrate | Gated on confidence; extends through ship + review-pr |
| **Interruption budget** | 0 (interactive rework on failure) | 0 (interactive rework on failure) |
| **Escape valve** | `/fab-clarify` (interactive rework on review failure) | `/fab-clarify` (interactive rework on review failure) |
| **Recomputes confidence?** | No | No |

### 5. Documentation updates

- `docs/specs/user-flow.md` — update diagrams 2, 3B, and 4:
  - Diagram 2: ff arrow goes to hydrate (H), fff arrow goes beyond to ship/review-pr or archive
  - Diagram 3B: update ff/fff labels to reflect scope difference
  - Diagram 4: update state diagram arrows for ff (→ hydrate) and fff (→ review-pr)
- `docs/specs/skills.md` — update `/fab-ff` and `/fab-fff` sections
- `docs/specs/skills/SPEC-fab-ff.md` — update flow diagram (remove ship/review-pr steps)
- `docs/specs/skills/SPEC-fab-fff.md` — update to reflect gates + no frontloaded questions
- `docs/specs/srad.md` — update skill autonomy table if it duplicates preamble content

### 6. `fab/.kit/schemas/workflow.yaml` — update `commands` fields

Currently the `tasks` stage lists `fab-ff` in its `commands` array (implying ff starts from tasks), and no stage lists `fab-fff`. With the redesign, both ff and fff start from intake and run through their respective endpoints. Update:

- `intake` stage: add `fab-ff` and `fab-fff` to `commands`
- `tasks` stage: remove `fab-ff` from `commands`

### 7. State table updates in `_preamble.md`

The State Table's available commands may need updating. Currently both ff and fff appear at the intake state. With the redesign, this is still correct — both are valid from intake. The key difference is just how far they go.

## Affected Memory

- `fab-workflow/planning-skills`: (modify) Update ff/fff descriptions to reflect new scope/gate philosophy
- `fab-workflow/execution-skills`: (modify) Update references to ff/fff ship/review-pr behavior

## Impact

- **Skill files**: `fab/.kit/skills/fab-ff.md`, `fab/.kit/skills/fab-fff.md`, `fab/.kit/skills/_preamble.md`
- **Spec files**: `docs/specs/user-flow.md`, `docs/specs/skills.md`, `docs/specs/skills/SPEC-fab-ff.md`, `docs/specs/skills/SPEC-fab-fff.md`
- **Memory files**: `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/execution-skills.md`
- **Schema file**: `fab/.kit/schemas/workflow.yaml` (commands field updates only)
- **No Go binary changes**: Schema is consumed by scripts/skills, not compiled into the binary

## Open Questions

- Should the `--force` flag also be available on `/fab-continue` for individual stage transitions, or only on the pipeline shortcuts?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | ff stops at hydrate, fff extends through ship + review-pr | Discussed — user explicitly proposed and confirmed this scope split | S:95 R:70 A:95 D:95 |
| 2 | Certain | Both use identical confidence gates (intake >= 3.0, spec >= per-type threshold) | Discussed — user confirmed "the ones in fab-ff right now" | S:95 R:60 A:90 D:95 |
| 3 | Certain | Drop frontloaded questions from fff entirely | Discussed — user said "Drop this entirely" | S:95 R:75 A:90 D:95 |
| 4 | Certain | Add --force flag to bypass gates on both | Discussed — user confirmed "--force flag is ok" | S:90 R:80 A:85 D:90 |
| 5 | Certain | Naming: ff = fast-forward, fff = fast-forward-further | Discussed — user confirmed the naming philosophy | S:90 R:90 A:90 D:95 |
| 6 | Confident | Auto-clarify behavior is identical in both (after spec and after tasks) | Current ff behavior is the baseline; user didn't object when I raised this | S:70 R:80 A:85 D:80 |
| 7 | Confident | No changes to Go binary, scripts, or templates | Scope is docs/skills only — no CLI behavior changes needed | S:75 R:85 A:80 D:85 |
| 8 | Tentative | --force flag only on ff/fff, not on /fab-continue | User mentioned it in context of ff/fff only; /fab-continue doesn't have gates to bypass | S:60 R:85 A:70 D:60 |
<!-- assumed: --force scope limited to pipeline shortcuts — /fab-continue doesn't currently have gates, so the flag would be a no-op there -->

8 assumptions (5 certain, 2 confident, 1 tentative, 0 unresolved).
