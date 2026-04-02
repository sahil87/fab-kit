# Tasks: Migrate Kit Scripts to Go Binary

**Change**: 260402-41gc-migrate-kit-scripts
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create shared internal package `src/go/fab/internal/spawn/spawn.go` — `SpawnCommand(configPath string) string` function that reads `agent.spawn_command` from config.yaml via Go YAML parsing, falls back to `claude --dangerously-skip-permissions`. Add tests in `spawn_test.go`.
- [x] T002 Create shared internal package `src/go/fab/internal/frontmatter/frontmatter.go` — `Field(filePath, fieldName string) string` function that parses YAML frontmatter between `---` markers, handles quoted/unquoted values, strips inline comments. Add tests in `frontmatter_test.go`.

## Phase 2: Core Implementation

- [x] T003 Implement `fab doctor` in `fab-kit`: add `src/go/fab-kit/cmd/fab-kit/doctor.go` with 7 prerequisite checks (git, fab, bash, yq v4+, jq, gh, direnv+hook). Support `--porcelain` flag. Exit code = failure count. Register in `src/go/fab-kit/cmd/fab-kit/main.go`. Add `"doctor"` to `fabKitArgs` allowlist in `src/go/fab-kit/cmd/fab/main.go`. Add tests in `doctor_test.go`.
- [x] T004 [P] Implement `fab fab-help` in `fab-go`: add `src/go/fab/cmd/fab/fabhelp.go` that scans `fab/.kit/skills/*.md` frontmatter (using `internal/frontmatter`), groups by hardcoded category map, renders formatted output with version header, workflow diagram, grouped commands, typical flow, and packages section. Register in `src/go/fab/cmd/fab/main.go`. Add tests in `fabhelp_test.go`.
- [x] T005 [P] Implement `fab operator` in `fab-go`: add `src/go/fab/cmd/fab/operator.go` — singleton tmux tab launcher. Check `$TMUX`, check if "operator" window exists via `tmux select-window`, create via `tmux new-window` with spawn command (using `internal/spawn`). Register in `src/go/fab/cmd/fab/main.go`. Add tests in `operator_test.go`.
- [x] T006 [P] Implement `fab batch new` in `fab-go`: add `src/go/fab/cmd/fab/batch.go` (parent command) and `src/go/fab/cmd/fab/batch_new.go`. Parse `fab/backlog.md` for pending items, extract descriptions with continuation line handling. Support `--list`, `--all`, positional IDs. Check `$TMUX`. Create worktrees via `wt create --non-interactive --worktree-name {id}`, open tmux windows with spawn command. Register under `batchCmd()` in main.go. Add tests in `batch_new_test.go`.
- [x] T007 [P] Implement `fab batch switch` in `fab-go`: add `src/go/fab/cmd/fab/batch_switch.go`. Resolve changes via `fab change resolve`, read `branch_prefix` from config, create worktrees via `wt create --non-interactive --reuse --worktree-name {name} {branch}`, open tmux windows with spawn command. Support `--list`, `--all`, positional args. Check `$TMUX`. Add tests in `batch_switch_test.go`.
- [x] T008 [P] Implement `fab batch archive` in `fab-go`: add `src/go/fab/cmd/fab/batch_archive.go`. Scan `fab/changes/*/` for `.status.yaml` with `hydrate: done|skipped`, resolve via `fab change resolve`, spawn single Claude session with archive prompt. Support `--list`, `--all`, positional args. Default no-args = `--all`. Add tests in `batch_archive_test.go`.

## Phase 3: Integration & Edge Cases

- [x] T009 Update `/fab-setup` skill (`fab/.kit/skills/fab-setup.md`): change `fab/.kit/scripts/fab-doctor.sh` → `fab doctor`
- [x] T010 [P] Update `/fab-help` skill (`fab/.kit/skills/fab-help.md`): change `bash fab/.kit/scripts/fab-help.sh` → `fab fab-help`
- [x] T011 [P] Update `/fab-operator` skill (`fab/.kit/skills/fab-operator.md`): change `fab/.kit/scripts/fab-operator.sh` → `fab operator`
- [x] T012 Delete `fab/.kit/scripts/` directory (all 6 scripts + `lib/` with 2 files)
- [x] T013 Update `README.md` script reference table to reflect new `fab` subcommands instead of shell script paths

---

## Execution Order

- T001 and T002 are independent setup tasks (run first)
- T003 depends on nothing (fab-kit, no shared internals needed)
- T004 depends on T002 (uses frontmatter package)
- T005, T006, T007, T008 depend on T001 (use spawn package)
- T006, T007, T008 are independent batch subcommands (parallel within Phase 2)
- T009-T011 are independent skill updates (parallel)
- T012 depends on T003-T008 (scripts must be migrated before deletion)
- T013 depends on T012
