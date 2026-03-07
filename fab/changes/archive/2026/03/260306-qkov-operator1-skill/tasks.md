# Tasks: Add `/fab-operator1` Skill

**Change**: 260306-qkov-operator1-skill
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Rename `docs/specs/skills/SPEC-fab-conductor.md` to `docs/specs/skills/SPEC-fab-operator1.md` and update all "conductor" references to "operator1" inside the file

## Phase 2: Core Implementation

- [x] T002 Implement `fab send-keys` subcommand in `src/fab-go/cmd/fab/sendkeys.go` — two positional args (`<change>`, `<text>`), tmux session guard, pane resolution (reuse pane-map discovery logic), pane existence validation, multi-pane warning, and `tmux send-keys -t <pane> "<text>" Enter` execution
- [x] T003 Register `sendKeysCmd()` in `src/fab-go/cmd/fab/main.go` root command
- [x] T004 Write tests for `fab send-keys` in `src/fab-go/cmd/fab/sendkeys_test.go` — test pane matching logic, multi-pane warning, no-pane-found error, tmux session guard
- [x] T005 Create the operator skill file at `fab/.kit/skills/fab-operator1.md` — frontmatter, context loading (always-load layer only), orientation on start (pane map + status), seven use cases, confirmation model, pre-send validation, bounded retries, context discipline, not-a-lifecycle-enforcer, key properties table

## Phase 3: Integration

- [x] T006 Update `fab/.kit/skills/_scripts.md` — add `fab send-keys` to the command reference table and add a dedicated section documenting usage, arguments, and error messages
- [x] T007 Update `docs/specs/skills/index.md` (if it exists) or `docs/specs/skills/` to reference the new SPEC-fab-operator1.md. Update `docs/specs/index.md` if the skills/ directory has a row in the specs index table

---

## Execution Order

- T001 is independent, can run in parallel with T002-T004
- T002 blocks T003 (sendKeysCmd must exist before registering)
- T002 blocks T004 (implementation before tests)
- T005 is independent of T002-T004
- T006 depends on T002 (need to know final command signature)
- T007 depends on T001 (rename must be done first)
