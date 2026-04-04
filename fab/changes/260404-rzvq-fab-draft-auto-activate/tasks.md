# Tasks: Fab Draft Auto Activate

**Change**: 260404-rzvq-fab-draft-auto-activate
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Skill Changes

<!-- Primary skill file modifications — all independent. -->

- [x] T001 [P] Update `src/kit/skills/fab-new.md`: add Step 10 (auto-activate via `fab change switch "{name}"`), update description to "Start a new change — creates the intake and activates it.", update Output section to show `Activated: {name}`, update final `Next:` line to `/fab-continue, /fab-fff, /fab-ff, or /fab-clarify` (no activation preamble)
- [x] T002 [P] Create `src/kit/skills/fab-draft.md`: new skill with Steps 0–9 identical to the current `/fab-new` (pre-activation behavior), description "Create a change intake without activating it.", `Next:` line retains activation preamble `/fab-switch {name} to make it active, then /fab-continue or /fab-fff or /fab-clarify`
- [x] T003 [P] Update `src/kit/skills/fab-switch.md`: change "No active changes found. Run /fab-new <description> to start one." to "No active changes found. Run /fab-new <description> to start one, or /fab-draft <description> to create without activating." and update error table to "Run /fab-new or /fab-draft."
- [x] T004 [P] Update `src/kit/skills/fab-proceed.md`: (a) remove `/fab-switch` from the "Conversation context" dispatch row — changes to `/fab-new → /git-branch → /fab-fff`; (b) update purpose description; (c) add note that fab-switch dispatch is only for `/fab-draft` intakes; (d) update empty-context error to "Nothing to proceed with — start a discussion or run /fab-new (or /fab-draft) first."
- [x] T005 [P] Update `src/kit/skills/_preamble.md`: change "This applies to `/fab-new` (always) and `/fab-archive restore` (without `--switch`)." to "This applies to `/fab-draft` (always) and `/fab-archive restore` (without `--switch`). `/fab-new` auto-activates and does not need the activation preamble."

## Phase 2: Supporting File Updates

<!-- Operator, CLI reference, docs — all independent. -->

- [x] T006 [P] Update `src/kit/skills/fab-operator.md`: (a) update routing note to remove `/fab-switch` from `/fab-new` chain; (b) update setup commands vocabulary to list `/fab-new` as "create + activate", `/fab-draft` as "create without activating", `/fab-switch` as "activate existing change"; (c) update pipeline description for `/fab-proceed` to show `/fab-new → /git-branch`
- [x] T007 [P] Update `src/kit/skills/_cli-fab.md`: change "No active changes found" error table entry action to "Run `/fab-new` or `/fab-draft` first"
- [x] T008 [P] Update `src/go/fab/cmd/fab/fabhelp.go`: add `"fab-draft": "Start & Navigate"` entry to `skillToGroupMap`
- [x] T009 [P] Update `src/go/fab/cmd/fab/fabhelp_test.go`: add `"fab-draft"` to the `expectedMapped` slice in `TestFabHelp_GroupMapping`

## Phase 3: Documentation Updates

<!-- Docs — all independent. -->

- [x] T010 [P] Update `README.md`: update mermaid block-beta diagram to 12 columns, add `/fab-draft` column (purple, change lifecycle) and `/fab-new` column (green, automation), show `fab-new` covering both intake + change-active rows, update arrows and styles accordingly
- [x] T011 [P] Update `docs/specs/skills.md`: (a) update Next Steps table to add `/fab-draft` row; (b) rename `/fab-new` section to remove `[--switch]` from signature; (c) update `/fab-new` description, artifacts (add `.fab-status.yaml`), remove `--switch` argument, update behavior step 8 to reflect auto-activation; (d) add new `/fab-draft` section; (e) update `/fab-proceed` section description to reflect simplified dispatch

---

## Execution Order

T001–T011 are all independent. Within each phase, all tasks are parallelizable. No blocking dependencies.
