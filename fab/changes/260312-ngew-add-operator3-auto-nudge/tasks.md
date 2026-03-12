# Tasks: Add Operator3 — Auto-Nudge for Blocked Agents

**Change**: 260312-ngew-add-operator3-auto-nudge
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 [P] Create launcher script `fab/.kit/scripts/fab-operator3.sh` — mirror `fab-operator2.sh`, replace `/fab-operator2` with `/fab-operator3`
- [x] T002 [P] Create skeleton skill file `fab/.kit/skills/fab-operator3.md` — frontmatter, purpose section, inheritance directive to read `fab-operator2.md`, and section stubs for new behavior

## Phase 2: Core Implementation

- [x] T003 Add Question Detection section to `fab/.kit/skills/fab-operator3.md` — define the terminal heuristic: `tmux capture-pane -t <pane> -p -l 10`, list of question indicator patterns, idle-only guard, bottom-most indicator rule
- [x] T004 Add Answer Confidence Model section to `fab/.kit/skills/fab-operator3.md` — two-tier classification (auto-answer vs escalate), classification heuristic, auto-answer action (tmux send-keys), escalation reporting format, no nudge budget
- [x] T005 Add Updated Monitoring Tick section to `fab/.kit/skills/fab-operator3.md` — document the 6-step tick order with input-waiting detection as step 5, stuck detection modified to skip input-waiting agents

## Phase 3: Integration & Edge Cases

- [x] T006 Create per-skill spec `docs/specs/skills/SPEC-fab-operator3.md` — document behavior, primitives, question detection, answer confidence model, monitoring tick changes, relationship to operator2
- [x] T007 Update `docs/memory/fab-workflow/execution-skills.md` — add `/fab-operator3` section covering question detection, answer confidence model, monitoring tick changes, and inheritance from operator2; add changelog entry
- [x] T008 Run `fab/.kit/scripts/fab-sync.sh` to deploy skill to `.claude/skills/`

## Execution Order

- T001 and T002 are independent (parallel)
- T003 → T004 → T005 (sequential within the skill file, building on prior sections)
- T006 and T007 are independent of each other but depend on T005 (skill complete)
- T008 depends on T005 (skill file finalized)
