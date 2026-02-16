# Tasks: Standardize State-Keyed Next-Step Suggestions

**Change**: 260216-7ltw-DEV-1038-standardize-state-keyed-suggestions
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Foundation

- [x] T001 Replace skill-keyed lookup table in `fab/.kit/skills/_context.md` with the canonical state-keyed table. Replace the "Next Steps Convention" section (lines 62-86): update the convention text to instruct skills to look up by state reached, add the 9-row state table (from spec), add activation preamble convention, add default-first ordering rule. Remove the old "Lookup Table" subsection with the "After skill | Stage reached | Next line" table.

## Phase 2: Structural Simplifications

- [x] T002 Remove `--switch` flag from `fab/.kit/skills/fab-new.md`: remove `--switch` from the `## Arguments` section (line 22), remove Step 6 "Activate Change (Conditional)" (lines 63-65), remove conditional output `{if switched: "Branch: {name} (created)\n"}` (line 75), remove two-path `Next` lines at bottom (lines 104-106). Replace with single `Next:` using the activation preamble: `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-fff or /fab-clarify`.

- [x] T003 Remove private stage→suggestion table from `fab/.kit/skills/fab-switch.md`: remove the inline table at lines 61-68, replace step 5 "Suggest next command based on stage" with instruction to derive from canonical state table in `_context.md`. Update the output template `Next:` at line 96 to reference state-table-derived suggestion instead of hardcoded `/fab-continue`.

## Phase 3: Update Skill Next: Lines

- [x] T004 [P] Update `fab/.kit/skills/fab-ff.md`: replace hardcoded `Next: /fab-archive` in the Output section (line 118) with `Next: {per state table}`. Also update the contextual bail/resume Next line note (line 121) to reference the state table.

- [x] T005 [P] Update `fab/.kit/skills/fab-fff.md`: replace hardcoded `Next: /fab-archive` in the Output section (line 146) with `Next: {per state table}`. Also update the contextual bail/resume note (line 149).

- [x] T006 [P] Update `fab/.kit/skills/fab-archive.md`: replace archive mode `Next: /fab-new <description>` (line 104) with `Next: {per state table — initialized}`. Replace restore mode `Next: /fab-switch {name}` (line 188) with state-derived suggestion using activation preamble when `--switch` is not used, and direct state lookup when `--switch` is used.

- [x] T007 [P] Update `fab/.kit/skills/fab-setup.md`: replace hardcoded `Next: /fab-new <description> or /docs-hydrate-memory <sources>` in Bootstrap Output (line 162) with `Next: {per state table — initialized}`. Update the Next Steps Reference section (lines 573-579) to reference the state table.

- [x] T008 [P] Update `fab/.kit/skills/fab-clarify.md`: replace hardcoded `Next: /fab-clarify or /fab-continue or /fab-ff` in the Coverage Summary (line 90) with state-aware derivation. Extend the stage guard in Pre-flight (lines 33-37) to include `intake` as a valid stage alongside `spec` and `tasks`. Ensure taxonomy scan categories for intake are documented (scope boundaries, affected areas, blocking questions, impact, memory coverage — these already exist in the spec-stage taxonomy; at intake stage only the intake subset applies).

- [x] T009 [P] Update `fab/.kit/skills/fab-continue.md`: replace hardcoded `Next: /fab-continue` in Review Verdict Pass (line 150) with `Next: {per state table}`. Update Step 5 Output description (line 78) to say "End with `Next:` per state table in `_context.md`" instead of "per `_context.md` lookup table".

- [x] T010 [P] Update `fab/.kit/skills/docs-hydrate-memory.md`: replace hardcoded `Next:` at line 183 with `Next: {per state table — initialized}`.

- [x] T011 [P] Update `fab/.kit/skills/fab-status.md`: add explicit reference that the "suggested next command" output is derived from the canonical state table in `_context.md`, eliminating any custom or implicit suggestion logic.

## Phase 4: Verification

- [x] T012 Cross-check all updated skill files: verify every previously hardcoded `Next:` line has been replaced with a state-table reference or derivation, no skill suggests commands not listed in its state's available set, the state table covers all reachable states, and `/fab-clarify` stage guard now includes `intake`. Check `docs/specs/skills.md` and `docs/specs/user-flow.md` for any documented suggestion behavior that may need updating — flag if found but do not modify (specs are human-curated).

---

## Execution Order

- T001 blocks T002-T011 (all updates reference the state table introduced by T001)
- T002-T011 are independent of each other (different files)
- T012 depends on all of T001-T011
