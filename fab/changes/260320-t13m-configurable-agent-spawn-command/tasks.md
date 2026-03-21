# Tasks: Configurable Agent Spawn Command

**Change**: 260320-t13m-configurable-agent-spawn-command
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create shared helper `fab/.kit/scripts/lib/spawn.sh` with `fab_spawn_cmd` function that reads `agent.spawn_command` from a given config path via `yq`, falling back to `claude --dangerously-skip-permissions` when missing or empty

## Phase 2: Core Implementation

- [x] T002 [P] Update `fab/.kit/scripts/fab-operator4.sh` — source `lib/spawn.sh`, resolve config path, replace hardcoded `claude --dangerously-skip-permissions` with `$SPAWN_CMD` in the tmux `new-window` invocation
- [x] T003 [P] Update `fab/.kit/scripts/fab-operator5.sh` — same pattern as T002
- [x] T004 [P] Update `fab/.kit/scripts/batch-fab-new-backlog.sh` — source `lib/spawn.sh`, read spawn command once before the loop, replace hardcoded command in the tmux `new-window` call
- [x] T005 [P] Update `fab/.kit/scripts/batch-fab-switch-change.sh` — source `lib/spawn.sh`, read spawn command once before the loop, replace hardcoded command in the tmux `new-window` call
- [x] T006 [P] Update `fab/.kit/scripts/batch-fab-archive-change.sh` — source `lib/spawn.sh`, read spawn command, replace `exec claude --dangerously-skip-permissions "$prompt"` with `eval "exec $SPAWN_CMD \"\$prompt\""`

## Phase 3: Integration

- [x] T007 [P] Update `fab/.kit/scaffold/config.yaml` to include the `agent` section with `spawn_command` and explanatory comment
- [x] T008 [P] Update `fab/.kit/skills/_cli-external.md` tmux `new-window` usage note to reference configurable `$SPAWN_CMD` from `config.yaml` instead of hardcoded command
- [x] T009 Create migration file in `fab/.kit/migrations/` to add `agent` section to existing `config.yaml` files that lack it (skip if already present)

---

## Execution Order

- T001 blocks T002–T006 (all script updates source the helper)
- T002–T006 are independent of each other (parallel)
- T007–T009 are independent of each other and of T002–T006 (parallel)
