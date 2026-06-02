# Plan: Fix stale `fab status` CLI invocations in fab-new and fab-draft

**Change**: 260601-w5sr-fix-stale-status-cli-args
**Status**: In Progress
**Intake**: `intake.md`
**Spec**: `spec.md`

## Tasks

<!-- Sequential work items for the apply stage. Checked off [x] as completed. -->

### Phase 1: Core Implementation

<!-- Mechanical signature corrections: replace the `.status.yaml` path arg with the `{name}` change reference. -->

- [x] T001 [P] In `src/kit/skills/fab-new.md`, replace the three `.status.yaml`-path `fab status` invocations with the `{name}` change-reference form: line ~56 `add-issue` (`fab status add-issue {name} DEV-988`), line ~89 `set-change-type` (`fab status set-change-type {name} <type>`), line ~113 `advance` (`fab status advance {name} intake`). Edit surrounding prose only as needed for these exact substitutions. <!-- A-001, A-003, A-005 -->
- [x] T002 [P] In `src/kit/skills/fab-draft.md`, replace the three `.status.yaml`-path `fab status` invocations with the `{name}` change-reference form: line ~58 `add-issue` (`fab status add-issue {name} DEV-988`), line ~91 `set-change-type` (`fab status set-change-type {name} <type>`), line ~115 `advance` (`fab status advance {name} intake`). Edit surrounding prose only as needed for these exact substitutions. <!-- A-001, A-003, A-005 -->

### Phase 2: Verification

<!-- Grep sweeps and spec alignment check. No source mutation unless a discrepancy is found. -->

- [x] T003 Run `grep -rn "fab status \(add-issue\|set-change-type\|advance\) .*\.status\.yaml" src/kit/skills/*.md` and confirm zero matches across the skill source tree. <!-- A-006, A-007 -->
- [x] T004 Verify `docs/specs/skills/SPEC-fab-new.md` and `docs/specs/skills/SPEC-fab-draft.md` already use the `<change>` form for `fab status set-change-type|advance|add-issue`; edit only if a `.status.yaml`-path discrepancy is found (constitutional obligation: skill changes update the corresponding SPEC). <!-- A-008, A-010 -->

## Acceptance

<!-- Declarative acceptance criteria used by the review stage. -->

### Functional Completeness

<!-- Every requirement in spec.md has working implementation. -->

- [x] A-001 Status subcommands receive a change reference: in both `fab-new.md` and `fab-draft.md`, all `fab status add-issue`, `set-change-type`, and `advance` invocations pass `{name}` as the first positional argument, matching the binary signatures documented in `_cli-fab.md`.
- [x] A-002 Source-only edits: only `src/kit/skills/fab-new.md` and `src/kit/skills/fab-draft.md` are changed; no file under `.claude/skills/` is touched (constitution §V).

### Behavioral Correctness

<!-- Changed requirements behave as specified, not as before. -->

- [x] A-003 Path form removed: no `fab/changes/{name}/.status.yaml` appears as the first argument to any `fab status` invocation in the two edited files; the literal `{name}` placeholder is preserved (these are skill instruction templates, not executed shell).

### Scenario Coverage

<!-- Key scenarios from spec.md have been exercised. -->

- [x] A-004 Each of the three subcommand forms is correct per spec scenarios: `fab status set-change-type {name} <type>`, `fab status advance {name} intake`, `fab status add-issue {name} DEV-988` appear in both files.
- [x] A-005 The corrected forms align with the installed binary (1.9.3+) `<change>` signatures, so an agent following the text would execute without the prior runtime error.

### Edge Cases & Error Handling

<!-- Error states, boundary conditions, suite-wide consistency. -->

- [x] A-006 Suite-wide consistency: a `grep` across all `src/kit/skills/*.md` for `fab status (add-issue|set-change-type|advance) .*\.status\.yaml` returns zero matches.
- [x] A-007 No collateral edits: surrounding prose is unchanged beyond the six exact substitutions; no other `fab status` call sites were altered.

### Code Quality

<!-- Baseline pattern-consistency items plus relevant code-quality.md principles. -->

- [x] A-008 Pattern consistency: the corrected invocations follow the same `<change>`-reference convention already used in `fab-continue.md`, `fab-ff.md`, `fab-fff.md`, and `_cli-fab.md`.
- [x] A-009 No unnecessary duplication / no new variance: reuses the already-in-scope `{name}` capture (no new variable introduced), per the spec Design Decision favoring `{name}` over `{id}`.

### Documentation Accuracy

<!-- config.yaml checklist.extra_categories: documentation_accuracy -->

- [x] A-010 SPEC alignment: `SPEC-fab-new.md` and `SPEC-fab-draft.md` document the `<change>` form and match the corrected skill sources; the constitutional "update SPEC-*" obligation is satisfied (already-correct, verified, no edit required unless a discrepancy was found).

### Cross-References

<!-- config.yaml checklist.extra_categories: cross_references -->

- [x] A-011 Cross-reference integrity: the corrected command forms are consistent with the signatures in `_cli-fab.md` (`fab status set-change-type <change> <type>`, `fab status advance <change> <stage> [driver]`, `fab status add-issue <change> <id>`) and with how the rest of the skill suite references `<change>`.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
