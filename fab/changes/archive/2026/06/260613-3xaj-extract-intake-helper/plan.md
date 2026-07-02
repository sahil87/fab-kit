# Plan: Extract `_intake.md` Shared Helper for Pre-Boundary Intake Creation

**Change**: 260613-3xaj-extract-intake-helper
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake.md's "What Changes" (6 items) + EXTRACTION BOUNDARY. Pure skill
     restructure — RFC-2119 statements describe the structural/behavioral contract of the
     extraction. Behavioral parity (no behavior change) is the cross-cutting acceptance bar. -->

### Helper: `_intake` Create-Intake Procedure

#### R1: New `_intake` internal helper exists at the canonical source path
A new internal helper SHALL be created at `src/kit/skills/_intake.md` (flat canonical-source layout — see Design Decisions / Assumptions row 1), carrying frontmatter `name: _intake`, `user-invocable: false`, `disable-model-invocation: true`, `metadata: internal: true`, and a `description:` describing the Create-Intake Procedure (Steps 0–9, parameterized by `{questioning-mode}`). It MUST NOT be created or edited under `.claude/skills/` (gitignored `fab sync` output).

- **GIVEN** the constitution names `src/kit/` canonical and every existing internal helper is a flat `src/kit/skills/_*.md` file
- **WHEN** the helper is authored
- **THEN** `src/kit/skills/_intake.md` exists with the internal-helper frontmatter matching `_pipeline.md`/`_review.md`/`_generation.md`/`_srad.md`
- **AND** no file is written under `.claude/skills/`

#### R2: `_intake` body defines Steps 0–9 parameterized by the single `{questioning-mode}` knob
The helper body SHALL define the "Create-Intake Procedure" = fab-new Steps 0–9, with exactly ONE parameter `{questioning-mode}` (`interactive` | `promptless-defer`) applied only at Step 8. Steps 0–7 and 9 SHALL be mode-invariant. The lifted text MUST be behaviorally identical to current fab-new Steps 0–9.

- **GIVEN** the intake's emphatic EXTRACTION BOUNDARY ("do NOT over-extract"; `{questioning-mode}` is the SOLE fork)
- **WHEN** the body is written
- **THEN** Steps 0, 1, 2, 3, 4, 5, 6, 7, 9 are reproduced from fab-new with no behavior change
- **AND** Step 8 branches: `interactive` → SRAD-driven question selection (no fixed cap, conversational mode when 5+ Unresolved); `promptless-defer` → record each would-be-asked Unresolved decision as a deferred Unresolved row per the `_srad.md` § Critical Rule promptless-dispatch carve-out (quoted verbatim)
- **AND** Step 5 references the already-extracted `_generation.md` § Intake Generation Procedure rather than inlining it

#### R3: Step 4 self-name references are genericized in the lifted body
The lifted Step 4 (Conversation Context Mining) SHALL refer to the invoking skill generically (e.g., "the invoking skill" / "this invocation") rather than "this `/fab-new` invocation". No `{self-name}` parameter SHALL be introduced. Step 4 SHALL also carry the finding's framing as the load-bearing context-flush at the boundary.

- **GIVEN** Assumption #9 (user-endorsed: genericize over parameterize) and `fab-draft`'s current "read self-name as `/fab-draft`" instruction
- **WHEN** Step 4 is lifted
- **THEN** the text reads invocation-agnostically and `fab-draft`'s per-consumer self-name instruction is structurally retired
- **AND** no `{self-name}` parameter exists in the helper

#### R4: `_intake` carries no `helpers:` frontmatter
`_intake.md` SHALL NOT declare a `helpers:` frontmatter list; it references `_generation` and `_srad` in-body and relies on the consumer having loaded them (the consumer-declared model, matching every existing internal helper).

- **GIVEN** Assumption #5 and that `_pipeline`/`_review`/`_generation` carry no `helpers:`
- **WHEN** the frontmatter is authored
- **THEN** `_intake.md` has only `name`/`description`/`user-invocable`/`disable-model-invocation`/`metadata`

### Consumer: `fab-new` thin call-site + retained tail

#### R5: `fab-new.md` becomes `_intake(interactive)` + activate/branch tail
`fab-new.md` SHALL replace its inline Steps 0–9 with a reference to read `_intake.md` and execute the Create-Intake Procedure with `{questioning-mode} = interactive`. Step 10 (activate), Step 11 (the full git-branch table incl. verify-in-repo guard, the 6-row evaluate-in-order table, the fab-new-specific `{dirty_count}` derivation excluding `fab/changes/{name}/`, the dirty-tree note, the keep-in-sync-with-git-branch.md comment), the Output block (with `Activated:`/`Branch:` lines), and the activation/git Error Handling rows SHALL STAY in `fab-new.md`. `fab-new.md`'s `helpers:` SHALL add `_intake` while keeping `_generation` and `_srad`.

- **GIVEN** the EXTRACTION BOUNDARY (activate/branch is a different responsibility, stays at the call site) and Assumption #4
- **WHEN** `fab-new.md` is rewired
- **THEN** Steps 0–9 are replaced by an `_intake(interactive)` call-site reference
- **AND** Steps 10–11, Output, and activation/git error rows remain verbatim
- **AND** frontmatter declares `helpers: [_generation, _srad, _intake]`

### Consumer: `fab-draft` thin call-site, momentum warning evaporates

#### R6: `fab-draft.md` becomes `_intake(interactive)`, stop at ready, momentum warning removed
`fab-draft.md` SHALL replace its prose-delta body with a reference to read `_intake.md` and execute the Create-Intake Procedure with `{questioning-mode} = interactive`; do NOT activate; do NOT create a git branch; stop after Step 9. The delta #2 "don't run Steps 10–11 by momentum" warning SHALL be removed (those steps no longer live in the body draft executes). `fab-draft` SHALL keep its own Output block (fab-new's minus `Activated:`/`Branch:`, ending with the Activation Preamble `Next:` line), Key Properties table, and Error Handling (no activation/git rows). Its `helpers:` SHALL add `_intake` while keeping `_generation` and `_srad`.

- **GIVEN** delta #2's warning exists only because the not-to-run steps shared the executed body
- **WHEN** `fab-draft.md` is rewired to call `_intake(interactive)`
- **THEN** the momentum warning is gone and the body references `_intake` instead of `fab-new.md`'s steps
- **AND** the Activation-Preamble `Next:` line and no-activation/git error posture are preserved
- **AND** frontmatter declares `helpers: [_generation, _srad, _intake]`

### Consumer: `fab-proceed` dispatch reroute, state-detection stays

#### R7: `fab-proceed.md`'s fab-new subagent dispatch reroutes to `_intake(promptless-defer)`
The subagent today dispatched as "`/fab-new` with a promptless defer-and-surface contract" SHALL instead dispatch the `_intake` Create-Intake Procedure with `{questioning-mode} = promptless-defer`. `/fab-proceed`'s state-detection (Steps 1–5, dispatch table), Relevance Assessment, asymmetric-bias rule, bypass notes, fab-switch/git-branch dispatch, Conversation Context Synthesis, and the terminal `/fab-fff` delegation SHALL all STAY unchanged. The defer-and-surface behavior is preserved (it is now `{questioning-mode}`-encoded in the called helper rather than a per-dispatch prompt contract over `/fab-new`).

- **GIVEN** the EXTRACTION BOUNDARY (proceed's state-detection decides *whether* to call `_intake`, stays at the call site)
- **WHEN** the fab-new Dispatch subsection is rewired
- **THEN** it dispatches `_intake(promptless-defer)` for the create-an-intake sub-operation
- **AND** the deferred-Unresolved surfacing + intake-gate backstop contract is preserved verbatim
- **AND** all state-detection / relevance / synthesis logic is unchanged

### Allowlist + Specs

#### R8: `_preamble.md` helpers Allowed-values allowlist gains `_intake`
The canonical `src/kit/skills/_preamble.md` § Skill Helper Declaration "Allowed values" line SHALL be updated from `_generation, _review, _cli-fab, _cli-external, _srad, _pipeline` to additionally include `_intake`.

- **GIVEN** Assumption #7 and that adding a new declared helper requires allowlist membership
- **WHEN** `_preamble.md` is edited
- **THEN** the Allowed-values line lists `_intake`

#### R9: All five constitution-mandated SPEC-* files are reconciled with the skill edits
Per the constitution ("Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md`"): a new `docs/specs/skills/SPEC-_intake.md` SHALL be created (mirroring `SPEC-_pipeline.md`/`SPEC-_generation.md` format), and `SPEC-fab-new.md`, `SPEC-fab-draft.md`, `SPEC-fab-proceed.md`, `SPEC-_preamble.md` SHALL be modified to reflect the rewires.

- **GIVEN** the constitution's SPEC-update mandate and Assumption #6
- **WHEN** the skill edits land
- **THEN** the five SPEC files mirror the skill-source changes and cross-references resolve

### Non-Goals

- Editing or generating any file under `.claude/skills/` — those are `fab sync` deployed copies (out of scope; the diff touches canonical sources + specs only).
- Running `fab sync` — explicitly out of scope for this diff.
- Touching any Go source or test — zero Go code (the finding confirms the state machine is already caller-agnostic).
- Changing intake-creation *behavior* — this is a pure restructure; behavioral parity is the bar.
- Hydrating `docs/memory/` — that is the hydrate stage's job (Affected Memory is resolved at hydrate).

### Design Decisions

1. **Canonical-source path is flat `src/kit/skills/_intake.md`, NOT `src/kit/skills/_intake/SKILL.md`**: *Why*: every existing canonical internal helper (`_pipeline.md`, `_generation.md`, `_srad.md`, `_review.md`, `_preamble.md`) is a flat `.md` file in `src/kit/skills/`; the `{name}/SKILL.md` directory-per-skill layout is the *deployed* form produced by `fab sync` under `.claude/skills/`. *Rejected*: `src/kit/skills/_intake/SKILL.md` (the intake's Assumption #1 path) — it would be inconsistent with the actual canonical source tree and would not deploy correctly. The canonical-source *intent* of Assumption #1 (never edit `.claude/skills/`) is fully honored; only the layout detail is corrected per apply-time evidence (Assumptions row 1).
2. **Mirror the `_pipeline.md` shape exactly**: shared body parameterized by one knob (`{questioning-mode}`, parallel to `_pipeline`'s `{driver}`/`{terminal}`); call-site-specific tails stay in the call-site files; consumers declare the helper via `helpers:` frontmatter AND keep declaring the underlying helpers (`_generation`, `_srad`) directly. *Why*: proven symmetry; low blast radius. *Rejected*: transitive inheritance of `_generation`/`_srad` through `_intake` — `_pipeline` precedent has consumers declare underlying helpers directly (Assumptions rows 4, 5).
3. **`fab-proceed` keeps declaring no `helpers:`**: it dispatches `_intake` as a subagent prompt (the subagent reads the helper), not as a frontmatter pre-load — same as today where it dispatched `/fab-new`. *Why*: proceed is an orchestrator that loads nothing for itself; the dispatched subagent loads what it needs. (Assumptions row 10.)

## Tasks

### Phase 1: Create the helper

- [x] T001 Create `src/kit/skills/_intake.md` with internal-helper frontmatter (`name: _intake`, `user-invocable: false`, `disable-model-invocation: true`, `metadata: internal: true`, descriptive `description:`) and a body defining the Create-Intake Procedure = Steps 0–9, parameterized by `{questioning-mode}` at Step 8, Step 4 genericized, Step 5 referencing `_generation.md`, the `_srad.md` carve-out quoted verbatim for `promptless-defer`. No `helpers:` frontmatter. <!-- R1 R2 R3 R4 -->

### Phase 2: Rewire consumers

- [x] T002 Rewire `src/kit/skills/fab-new.md`: replace inline Steps 0–9 with an `_intake(interactive)` call-site reference; keep Steps 10–11, Output, activation/git Error Handling, Key Properties, trailing `Next:`; add `_intake` to `helpers:` (now `[_generation, _srad, _intake]`). <!-- R5 -->
- [x] T003 Rewire `src/kit/skills/fab-draft.md`: replace the prose-delta body with an `_intake(interactive)` call-site reference (do not activate, no git branch, stop after Step 9); remove delta #2's momentum warning; keep Output (Activation-Preamble `Next:`), Key Properties, Error Handling; add `_intake` to `helpers:` (now `[_generation, _srad, _intake]`). <!-- R6 -->
- [x] T004 Rewire `src/kit/skills/fab-proceed.md` § fab-new Dispatch: reroute the subagent dispatch to `_intake(promptless-defer)`; keep all state-detection / relevance / synthesis / terminal-delegation logic unchanged. Dispatch-table create-new rows chain `_intake` → `/fab-switch` → `/git-branch` for activate/branch parity (Assumption 11). <!-- R7 -->

### Phase 3: Allowlist

- [x] T005 Update `src/kit/skills/_preamble.md` § Skill Helper Declaration "Allowed values" line to include `_intake` (and update the inline `helpers: [...]` example comment if it would otherwise drift). <!-- R8 -->

### Phase 4: Specs (constitution-mandated)

- [x] T006 [P] Create `docs/specs/skills/SPEC-_intake.md` mirroring `SPEC-_pipeline.md`/`SPEC-_generation.md` format (Summary, parameter, Flow, sub-agents, bookkeeping). <!-- R9 -->
- [x] T007 [P] Modify `docs/specs/skills/SPEC-fab-new.md` to reflect the thin-call-site rewire + retained activate/branch tail + `_intake` in helpers. <!-- R9 -->
- [x] T008 [P] Modify `docs/specs/skills/SPEC-fab-draft.md` to reflect the `_intake(interactive)` rewire + removed momentum warning + `_intake` in helpers. <!-- R9 -->
- [x] T009 [P] Modify `docs/specs/skills/SPEC-fab-proceed.md` to reflect the `_intake(promptless-defer)` dispatch reroute + dispatch-table chaining. <!-- R9 -->
- [x] T010 [P] Modify `docs/specs/skills/SPEC-_preamble.md` to reflect the updated 7-value helpers allowlist. <!-- R9 -->

### Phase 5: Verification

- [x] T011 Re-read all edited/created files; confirm cross-reference integrity (`_intake` references resolve, allowlist matches, the five specs mirror the skill edits), the EXTRACTION BOUNDARY held (no over-extraction of activate/branch or proceed state-detection), and behavioral parity vs. current fab-new Steps 0–9 (byte-level diff of mode-invariant Steps 0/1/2/3/6/7 = identical; Steps 4/8/9 differ only as intended). <!-- R1 R2 R3 R4 R5 R6 R7 R8 R9 -->

## Execution Order

- T001 blocks T002, T003, T004 (consumers reference the helper).
- T005 is independent of T001–T004.
- T006–T010 are [P] (independent spec files) but should follow T001–T005 so they mirror the final skill text.
- T011 runs last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/_intake.md` exists with correct internal-helper frontmatter; nothing was written under `.claude/skills/`. (Frontmatter verified: `name: _intake`, `user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`, descriptive `description:`. `git status` shows no `.claude/skills/` entries.)
- [x] A-002 R2: `_intake.md` body defines Steps 0–9 with `{questioning-mode}` as the sole parameter applied at Step 8; Steps 0–7 and 9 are mode-invariant; Step 5 references `_generation.md`. (Diff vs HEAD fab-new confirms Steps 0/1/2/3/5/6/7 byte-identical; Step 8 is the only branched step; Steps 4/9 differ only by intended genericization. Step 5 reads "Follow the Intake Generation Procedure (`_generation.md`)".)
- [x] A-003 R3: Step 4 in `_intake.md` is genericized (no "this `/fab-new` invocation"); no `{self-name}` parameter exists. (Step 4 reads "preceded the invoking skill's invocation" and "identical to a cold invocation"; the context-flush framing paragraph is present; no `{self-name}` token anywhere in the file.)
- [x] A-004 R4: `_intake.md` declares no `helpers:` frontmatter. (Frontmatter has only name/description/user-invocable/disable-model-invocation/metadata.)
- [x] A-005 R5: `fab-new.md` references `_intake(interactive)`, retains Steps 10–11 + Output + activation/git error rows, and declares `helpers: [_generation, _srad, _intake]`. (Step 11 git-branch table verbatim incl. the verify-in-repo guard, 6-row evaluate-in-order table, `{dirty_count}` derivation excluding `fab/changes/{name}/`, dirty-tree note, keep-in-sync comment; Output has Activated:/Branch: lines; error table keeps `fab change switch`/not-in-git-repo/`git checkout` rows.)
- [x] A-006 R6: `fab-draft.md` references `_intake(interactive)`, stops at ready, has NO momentum warning, retains its Activation-Preamble `Next:` and no-git/activation error posture, and declares `helpers: [_generation, _srad, _intake]`. (HEAD delta #2's "running activation or branch creation by momentum is the known failure mode" string is gone; line 47 explains why no hazard exists; Error Handling explicitly adds no activation/git rows.)
- [x] A-007 R7: `fab-proceed.md` dispatches `_intake(promptless-defer)`; state-detection / relevance / synthesis / terminal delegation are unchanged. (Steps 1–5 detection, Relevance Assessment, asymmetric-bias, Conversation Context Synthesis, terminal `/fab-fff` Skill-tool delegation all preserved; create-new rows chain `_intake → /fab-switch → /git-branch` per Assumption 11 — verified necessary: HEAD `/fab-new` rows activated+branched inline via Steps 10–11, and `/fab-switch` runs the same `fab change switch` (fab-switch.md L41) as fab-new Step 10, so the chain reaches the identical end state.)
- [x] A-008 R8: `_preamble.md` Allowed-values line lists `_intake` (7 values total). (`_preamble.md` L105: `_generation, _review, _cli-fab, _cli-external, _srad, _pipeline, _intake`.)
- [x] A-009 R9: All five SPEC files exist and mirror the skill edits. (`SPEC-_intake.md` new + `SPEC-fab-new`/`SPEC-fab-draft`/`SPEC-fab-proceed`/`SPEC-_preamble` modified; each mirrors its skill's rewire and cross-references resolve to `SPEC-_intake.md`.)

### Behavioral Correctness

- [x] A-010 R2: `interactive` mode reproduces current fab-new/fab-draft Step 8 behavior (SRAD-driven, no cap, conversational at 5+ Unresolved); `promptless-defer` reproduces fab-proceed's defer-and-surface contract (the `_srad.md` carve-out is quoted, not redefined). (`_intake.md` Step 8 interactive bullet is verbatim-equivalent to HEAD fab-new Step 8; the promptless-defer blockquote is a verbatim substring of `_srad.md` § Critical Rule "Promptless-dispatch carve-out" — confirmed by direct comparison.)
- [x] A-011 R5 R6 R7: Diffing the lifted Steps 0–9 against current fab-new.md shows zero intended behavior change for interactive consumers; fab-proceed's promptless contract is byte-equivalent in effect. (`diff` of HEAD fab-new Steps 0–9 vs `_intake.md` Steps 0–9: only Step 4 framing+genericization, Step 8 parameterization, and Step 9 call-site-agnostic closing differ — all intended. Steps 0/1/2/3/5/6/7 identical.)

### Scenario Coverage

- [x] A-012 R6: `fab-draft`'s removed momentum warning is justified — Steps 10–11 are not present in the body draft executes (they live only in fab-new.md's tail, which draft never reads). (fab-draft.md L38 reads `_intake.md` which contains Steps 0–9 only; L47 states the hazard is gone because the not-to-run steps live solely in fab-new.md's tail. Confirmed Steps 10–11 are absent from `_intake.md`.)

### Edge Cases & Error Handling

- [x] A-013 R5 R6: fab-new keeps activation/git error rows; fab-draft keeps the activation/git rows removed — error-handling tables match each consumer's retained responsibilities. (fab-new.md error table retains `fab change switch` (Step 10), not-in-git-repo (Step 11), and `git checkout`/`git branch` (Step 11) rows; fab-draft.md L61 explicitly states it adds no activation/git rows because those steps never run.)

### Code Quality

- [x] A-014 Pattern consistency: `_intake.md` and `SPEC-_intake.md` follow the established `_pipeline`/`_review`/`_generation` helper and spec conventions (frontmatter shape, parameter note, Flow/sub-agents/bookkeeping sections). (`_intake.md` frontmatter matches the internal-helper shape; intro blockquote documents the `{questioning-mode}` knob and helper model like `_pipeline.md`. `SPEC-_intake.md` has Summary + parameter table + Flow diagram + Tools used + Sub-agents + Bookkeeping commands — same structure as `SPEC-_pipeline.md`/`SPEC-_generation.md`.)
- [x] A-015 No unnecessary duplication: Steps 0–9 now live in exactly one place (`_intake.md`); consumers reference it rather than re-stating the steps; Step 5 references `_generation.md` rather than inlining intake generation. (Inline Steps 0–9 removed from fab-new.md (-89 lines net structure) and fab-draft's delta body replaced with a call-site reference; the lifted steps exist only in `_intake.md`. Step 5 delegates to `_generation.md` § Intake Generation Procedure, not inlined.)

### Documentation Accuracy

<!-- config.yaml checklist.extra_categories -->

- [x] A-016: Skill prose accurately describes the new structure (no stale references to "fab-new's Steps 0–9" from draft/proceed; no claim that the helper lives at a `.claude/skills/` or `_intake/SKILL.md` path in the canonical source). (Within the edited skill files this holds: fab-draft no longer says "execute fab-new's Steps 0–9"; the only `fab-new` mentions in draft/proceed are correct relationship descriptions; the authored canonical source `src/kit/skills/_intake.md` contains no `.claude/skills/` self-path. Rework cycle 1 RESOLVED the previously-flagged `docs/specs/skills.md` staleness: allowlist count `(6)`→`(7)` with `_intake` appended (L24), the `fab-new, fab-draft` helpers row → `[_generation, _srad, _intake]` (L48), and the partial-SPEC enumeration gained `SPEC-_intake.md` (L125). Now self-consistent with `_preamble.md`/`SPEC-_preamble.md`.)

### Cross-References

<!-- config.yaml checklist.extra_categories -->

- [x] A-017: Every cross-reference resolves — `_intake` deployed-path reference (`.claude/skills/_intake/SKILL.md`) in consumer bodies, `_generation`/`_srad` in-body references in `_intake.md`, the `_preamble.md` allowlist, and the five SPEC cross-references are mutually consistent. (All three consumers reference `.claude/skills/_intake/SKILL.md`; `_intake.md` Step 5 → `_generation.md`, Steps 4/8 → `_srad.md`; `_preamble.md` allowlist includes `_intake`; SPEC-fab-new/draft/proceed each cross-reference `SPEC-_intake.md`. The enumerated set is mutually consistent. Rework cycle 1 additionally RESOLVED two dangling back-references to the renamed `fab-proceed.md` heading — `_srad.md` and `docs/memory/pipeline/planning-skills.md` both updated `§ fab-new Dispatch` → `§ Create-Intake Dispatch` (Must-fix #1 + directly-caused fallout, Assumption #12) — and reconciled `docs/specs/skills.md`'s allowlist (Should-fix #2). Post-fix repo-wide grep of `src/kit/`+`docs/` for `fab-new Dispatch` returns zero.)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- Skill *bodies* reference the deployed helper path (`.claude/skills/_intake/SKILL.md`) per the established `_preamble` "Read the `_preamble` skill ... deployed to `.claude/skills/`" convention; the *file we author/edit* is the canonical flat source `src/kit/skills/_intake.md`. These are not in conflict — `fab sync` maps `src/kit/skills/_intake.md` → `.claude/skills/_intake/SKILL.md`.

## Assumptions

<!-- Intake's 9 assumptions transfer in (rows 1–9, with row 1's PATH detail corrected per
     apply-time evidence). Row 10 is a NEW apply-time decision. Three grades only. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Canonical-source path is the flat `src/kit/skills/_intake.md`, NOT `src/kit/skills/_intake/SKILL.md`. The canonical-source intent of intake Assumption #1 (never edit `.claude/skills/`) is honored; the directory-per-skill layout it named is the *deployed* `.claude/skills/{name}/SKILL.md` form, not the source tree. | Apply-time evidence: every existing canonical helper (`_pipeline.md`, `_generation.md`, `_srad.md`, `_review.md`, `_preamble.md`) is a flat `.md` in `src/kit/skills/`; `.claude/skills/_pipeline/SKILL.md` is the deployed copy. Constitution: `src/kit/` canonical, `.claude/skills/` is `fab sync` output. Corrects intake row 1's path detail while preserving its intent. | S:100 R:90 A:100 D:100 |
| 2 | Certain | `_intake` frontmatter is `user-invocable: false`, `disable-model-invocation: true`, `metadata: internal: true`. | Matches `_pipeline`/`_review`/`_generation`/`_srad` verbatim (verified by reading them). Intake row 2. | S:100 R:85 A:100 D:100 |
| 3 | Certain | `{questioning-mode}` (`interactive` \| `promptless-defer`) is the SOLE parameter; activate/branch and proceed's state-detection stay at the call site. | Intake's EXTRACTION BOUNDARY is explicit and emphatic. Mirrors `_pipeline`'s one-knob shape. Intake row 3. | S:100 R:80 A:95 D:100 |
| 4 | Confident | `fab-new.md` and `fab-draft.md` keep declaring `_generation` and `_srad` directly in `helpers:` AND add `_intake` (now `[_generation, _srad, _intake]`). | `_pipeline` precedent: `fab-ff`/`fab-fff` declare underlying helpers (`_generation`, `_review`, `_srad`) directly alongside `_pipeline`. Frontmatter-only, trivially reversible. Intake row 4. | S:55 R:90 A:70 D:65 |
| 5 | Confident | `_intake.md` carries NO `helpers:` frontmatter; references `_generation`/`_srad` in-body, relies on consumer having loaded them. | Matches `_pipeline`/`_review`/`_generation` (none carry `helpers:`). Consumer-declared model. Intake row 5. | S:60 R:85 A:75 D:70 |
| 6 | Confident | Spec updates: new `SPEC-_intake.md` + modify `SPEC-fab-new.md`, `SPEC-fab-draft.md`, `SPEC-fab-proceed.md`, `SPEC-_preamble.md`. | Constitution mandates SPEC-* updates for skill-file changes. Five edited skills map 1:1 (verified all four mods exist; `SPEC-_intake.md` is new). Intake row 6. | S:80 R:85 A:90 D:85 |
| 7 | Confident | `_preamble.md` Allowed-values line gains `_intake` (now `_generation, _review, _cli-fab, _cli-external, _srad, _pipeline, _intake`). | Current allowlist verbatim at `_preamble.md` line 105 (verified). Intake row 7. | S:90 R:90 A:95 D:90 |
| 8 | Confident | Behavioral parity is the acceptance bar: `interactive` reproduces current fab-new/fab-draft Steps 0–9 exactly; `promptless-defer` preserves `/fab-proceed`'s current contract exactly. | Finding frames de-duplication, not behavior change. The carve-out exists verbatim in `_srad.md` § Critical Rule; referenced not redefined. Intake row 8. | S:75 R:70 A:80 D:75 |
| 9 | Confident | Lifted Step 4 genericized ("the invoking skill" not "this `/fab-new` invocation"); `{self-name}` parameter rejected; structurally retires fab-draft's self-name instruction. | User-endorsed (2026-06-13): genericize over parameterize. Ordinary apply-time prose. Intake row 9. | S:80 R:80 A:75 D:80 |
| 10 | Confident | `fab-proceed.md` keeps declaring NO `helpers:` frontmatter; it dispatches `_intake(promptless-defer)` as a subagent prompt (the subagent reads the helper), exactly as it dispatched `/fab-new` today. | NEW apply-time decision. fab-proceed is an orchestrator that loads no helpers for itself (verified: no `helpers:` in its current frontmatter) and delegates loading to dispatched subagents. Adding `_intake` to its frontmatter would be incorrect — it never executes `_intake` in its own context. Frontmatter-only, trivially reversible. | S:75 R:90 A:80 D:80 |
| 11 | Confident | The `/fab-proceed` create-new dispatch-table rows now chain `_intake` → `/fab-switch` → `/git-branch` (previously `/fab-new` alone). REQUIRED for behavioral parity: the old dispatch invoked the FULL `/fab-new` skill, which activated (Step 10) + branched (Step 11) inline; `_intake` stops at `ready` and does neither (the EXTRACTION BOUNDARY keeps activate/branch as fab-new's call-site tail), so proceed must run the dedicated `/fab-switch`+`/git-branch` prefix steps (which it already has) to reach the same end state before `/fab-fff`. Without this, proceed's create-new path would hand `/fab-fff` an inactive, unbranched change — a parity regression. | NEW apply-time decision surfaced by the reroute. Directly serves Assumption #8 (preserve proceed's contract exactly). The `/fab-switch`+`/git-branch` steps already exist in proceed (used by the relevant-intake rows); this makes the create-new rows symmetric. Reversible (skill prose). The obsolete "fab-new rows skip /git-branch because Step 11 branches inline" rationale (lines 95/154) is updated accordingly. | S:80 R:85 A:80 D:70 |
| 12 | Certain | The dangling back-reference fix to the renamed `fab-proceed.md` heading (`§ fab-new Dispatch` → `§ Create-Intake Dispatch`) is applied not only in `src/kit/skills/_srad.md` but also in `docs/memory/pipeline/planning-skills.md:46`, which carried the identical parenthetical cross-reference. | Rework-cycle finding: this change's T004 renamed the `fab-proceed.md` heading, so EVERY back-reference to `§ fab-new Dispatch` dangles. A repo-wide grep of `src/kit/`+`docs/` found exactly two (`_srad.md` = Must-fix #1; `planning-skills.md` = directly-caused rename fallout). Both fixed; post-fix grep returns zero. Section-name-only edits — no surrounding prose changed; the memory file's editable-canonical status (not gitignored, not `.claude/skills/`) permits the edit. | S:100 R:95 A:100 D:100 |

12 assumptions (4 certain, 8 confident, 0 tentative).

## Deletion Candidates

None — this change IS a deletion (de-duplication). The redundant blocks the extraction targets — the inline Steps 0–9 in `fab-new.md` and `fab-draft.md`'s old prose-delta body (incl. delta #2's momentum warning) — were already removed by apply (modified-file diff: +121 / −223 lines; the lifted content now lives once in `_intake.md`). No leftover redundant block remains for the reviewer to additionally delete. The new `_intake.md` (137 lines) and `SPEC-_intake.md` (94 lines) are the single authoritative home for the lifted procedure, not new duplication. The `fab-new` mentions remaining in `fab-draft.md`/`fab-proceed.md` are correct relationship descriptions (e.g., "Steps 10–11 live only in `fab-new.md`'s tail"), not stale indirection, and must stay.
