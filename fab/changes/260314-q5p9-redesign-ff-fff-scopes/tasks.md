# Tasks: Redesign FF and FFF Pipeline Scopes

**Change**: 260314-q5p9-redesign-ff-fff-scopes
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Skill Files

- [x] T001 Update `fab/.kit/skills/fab-ff.md` — remove Steps 8 (Ship) and 9 (Review-PR), update frontmatter description to "through hydrate" scope, update Purpose section to describe "intake → spec → tasks → apply → review → hydrate", add `--force` flag to Arguments section, update Resumability to check `hydrate: done` instead of `review-pr: done`, update Output format to end at `--- Hydrate ---`, remove ship/review-pr rows from Error Handling table
- [x] T002 Update `fab/.kit/skills/fab-fff.md` — add confidence gates to Pre-flight (intake gate step 3, spec gate after spec generation), remove Step 1 (Frontload All Questions) and renumber remaining steps, add `--force` flag to Arguments section, update frontmatter description to reflect gates + no frontloaded questions, update Purpose to describe "with confidence gates" and remove "No confidence gates" language, update Output header from "no gate" to include gate info, add intake/spec gate rows to Error Handling table

## Phase 2: Preamble and Schema

- [x] T003 [P] Update `fab/.kit/skills/_preamble.md` — update SRAD Skill-Specific Autonomy Levels table: change fab-ff column from "from-spec, gated" to "Gated on confidence; stops at hydrate", change fab-fff column from "full pipeline" to "Gated on confidence; extends through ship + review-pr". Update Confidence Scoring section: change "/fab-ff has two confidence gates. /fab-fff has no confidence gates" to describe both having identical gates. Update "/fab-fff does not gate or recompute" to reflect new gate behavior. Document --force bypass.
- [x] T004 [P] Update `fab/.kit/schemas/workflow.yaml` — add `fab-ff` and `fab-fff` to intake stage `commands` array. Remove `fab-ff` from tasks stage `commands` array.

## Phase 3: Documentation

- [x] T005 [P] Update `docs/specs/user-flow.md` — Diagram 2: change fab-ff shortcut arrow to point to H (hydrate) instead of sharing endpoint with fff, add fff arrow to RP (review-pr), update both labels to mention "confidence-gated". Diagram 3: update FF node text to "fast-forward through hydrate", FFF node text to "fast-forward further through review-pr", update arrow destinations (FF→HYD, FFF→RP). Diagram 4: update transition labels — `intake --> hydrate: /fab-ff (fast-forward, confidence-gated)`, `intake --> review_pr: /fab-fff (fast-forward-further, confidence-gated)`.
- [x] T006 [P] Update `docs/specs/skills.md` — update `/fab-ff` section: Purpose to "Fast-forward from intake through hydrate", Flow to "intake → spec → tasks → apply → review → hydrate", Behavior step 9 (hydrate) as final step, add --force mention. Update `/fab-fff` section: Purpose to remove "No confidence gate" and add "Confidence-gated", remove frontloaded questions from behavior, add --force, update "Key difference" text to reflect scope-only differentiation.
- [x] T007 [P] Update `docs/specs/skills/SPEC-fab-ff.md` — update Summary to "through hydrate" scope, remove Steps 8 (Ship) and 9 (Review-PR) from flow diagram, remove /git-pr and /git-pr-review from Sub-agents table, add --force note.
- [x] T008 [P] Update `docs/specs/skills/SPEC-fab-fff.md` — update Summary to reflect confidence gates + no frontloaded questions, add intake gate and spec gate to flow, remove Frontload Questions step, add --force note.
- [x] T009 [P] Update `docs/specs/srad.md` — update Skill-Specific Autonomy Levels table: fab-ff and fab-fff columns to match updated _preamble.md values. Update Gate Threshold section to mention both /fab-ff and /fab-fff.

## Phase 4: Memory Files

- [x] T010 [P] Update `docs/memory/fab-workflow/planning-skills.md` — update Overview paragraph: change `/fab-fff` description from "full-pipeline command (intake → hydrate, no confidence gate, frontloaded questions, interactive rework)" to "full-pipeline command (intake → review-pr, confidence-gated, no frontloaded questions, autonomous rework)". Change `/fab-ff` description from "fast-forward-from-spec command (spec → hydrate, confidence-gated, no frontloaded questions, bail on failure)" to "fast-forward command (intake → hydrate, confidence-gated, autonomous rework)". Update `/fab-fff` requirements section: remove "Frontloaded Questions" subsection, add confidence gates to "Minimum Prerequisite" or pipeline flow, update "When to Use" to remove "No confidence gate". Update `/fab-ff` requirements section: update "Minimum Prerequisite" from spec-only to intake, remove confidence gate from spec-only framing. Update "Scope Differentiation" design decision to reflect new scope-only axis. Update Changelog with this change entry.
- [x] T011 [P] Update `docs/memory/fab-workflow/execution-skills.md` — update Overview paragraph references to `/fab-ff` and `/fab-fff` pipeline scope if present. Update any references to fff having no gates or ff stopping at different endpoints. Update Changelog with this change entry.

---

## Execution Order

- T001 and T002 are independent — both modify separate skill files
- T003 and T004 are independent of each other (separate files) and can run alongside T001/T002
- T005-T011 are all independent (separate files) and can run in parallel
- No strict dependency chain — all tasks touch separate files
