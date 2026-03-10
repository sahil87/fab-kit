# Tasks: Redesign Hooks Strategy

**Change**: 260306-6bba-redesign-hooks-strategy
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Add `fab runtime` Cobra command group in `src/go/fab/cmd/fab/main.go` — register `runtimeCmd()` on the root command

## Phase 2: Core Implementation

- [x] T002 Implement `fab runtime set-idle <change>` in `src/go/fab/cmd/fab/runtime.go` — resolve change to folder name, read/create `.fab-runtime.yaml` at repo root, write `{folder}.agent.idle_since` with current Unix timestamp
- [x] T003 Implement `fab runtime clear-idle <change>` in `src/go/fab/cmd/fab/runtime.go` — resolve change to folder name, read `.fab-runtime.yaml`, delete `{folder}.agent` block, exit 0 if file doesn't exist
- [x] T004 Add tests for runtime commands in `src/go/fab/test/parity/runtime_test.go` — test set-idle (creates file, updates existing), clear-idle (removes entry, no-op on missing file), change resolution
- [x] T005 Create `fab/.kit/hooks/on-artifact-write.sh` — PostToolUse hook script: parse stdin JSON for `tool_input.file_path`, pattern-match against `fab/changes/*/intake.md|spec.md|tasks.md|checklist.md`, derive change name, run appropriate bookkeeping commands, return `additionalContext` JSON, exit 0 always
- [x] T006 [P] Migrate `fab/.kit/hooks/on-stop.sh` — replace yq calls with `"$fab_cmd" runtime set-idle "$change_folder"`, remove `command -v yq` guard and `runtime_file` variable
- [x] T007 [P] Migrate `fab/.kit/hooks/on-session-start.sh` — replace yq calls with `"$fab_cmd" runtime clear-idle "$change_folder"`, remove `command -v yq` guard and `runtime_file` variable

## Phase 3: Integration & Edge Cases

- [x] T008 Update `fab/.kit/sync/5-sync-hooks.sh` — extend `map_event()` to support matchers, register `on-artifact-write.sh` for both PostToolUse Write and PostToolUse Edit matchers using mapping table approach
- [x] T009 Update Constitution §I in `fab/project/constitution.md` — change "shell scripts" to "scripts", bump version <!-- clarified: constitution already reads "scripts" (v1.3.0, amended 2026-03-06) — verify at apply time and skip if already correct -->
- [x] T010 Remove Phase 1b-lang from `fab/.kit/skills/fab-setup.md` — delete the `#### 1b-lang. Language Detection and Convention Inference` section and its content, renumber subsequent steps
- [x] T011 Update `fab/.kit/skills/_scripts.md` — add `fab runtime` command reference (set-idle, clear-idle) per constitution constraint
- [x] T012 Update `docs/specs/skills/SPEC-fab-setup.md` — reflect removal of Phase 1b-lang per constitution constraint (skill file changes MUST update corresponding SPEC file)

## Phase 4: Polish

- [x] T013 Run `fab/.kit/sync/5-sync-hooks.sh` and verify `.claude/settings.local.json` contains correct hook registrations with matchers

---

## Execution Order

- T001 blocks T002, T003
- T002, T003 block T004
- T005 is independent of T001-T004 (uses fab CLI, not the Go source directly)
- T006, T007 depend on T002, T003 (runtime commands must exist)
- T008 depends on T005 (hook script must exist to register)
- T009, T010, T011, T012 are independent of each other
- T012 depends on T010 (SPEC update follows skill file change)
- T013 depends on T008
