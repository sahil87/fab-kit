# Tasks: Three-Binary Architecture

**Change**: 260402-3ac3-three-binary-architecture
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Rename `src/go/shim/` to `src/go/fab-kit/`. Update `go.mod` module path. Update all `import` statements in existing files to reflect new module path.
- [x] T002 Create `src/go/fab-kit/cmd/fab/main.go` — stub router binary with Cobra root command, `--version`, and `--help`. Wire up the `fabKitCommands` allowlist map (same pattern as current `nonRepoCommands` in `dispatch.go`).
- [x] T003 Rename existing `src/go/fab-kit/cmd/main.go` to `src/go/fab-kit/cmd/fab-kit/main.go`. Update Cobra root command name from `fab` to `fab-kit`. Verify `init`, `upgrade`, `--version` subcommands still work.

## Phase 2: Core Implementation

- [x] T004 Implement router dispatch in `src/go/fab-kit/cmd/fab/main.go`: negative-match routing — if first arg is in fab-kit set, find `fab-kit` binary on PATH and `syscall.Exec`; otherwise call shared `internal.ResolveConfig()` + `internal.EnsureCached()` + `syscall.Exec` to fab-go. Handle not-in-repo error.
- [x] T005 Implement composed help in router (`src/go/fab-kit/cmd/fab/main.go`): detect if inside a repo (config.yaml exists), run `fab-go --help` to get workflow commands, merge with fab-kit commands into grouped output. Fall back to workspace-only when outside repo.
- [x] T006 Implement `sync` subcommand in `src/go/fab-kit/cmd/fab-kit/main.go` — directory scaffolding: create `fab/changes/`, `fab/changes/archive/`, `docs/memory/`, `docs/specs/` with `.gitkeep` files. Create `fab/.kit-migration-version` using dual-version model.
- [x] T007 Implement scaffold tree-walk in `src/go/fab-kit/internal/sync.go` — walk `fab/.kit/scaffold/`, dispatch by filename: `fragment-` + `.json` → `jsonMergePermissions()`, `fragment-` + other → `lineEnsureMerge()`, no prefix → copy-if-absent.
- [x] T008 Implement `jsonMergePermissions()` in `src/go/fab-kit/internal/sync.go` — read source and dest JSON, merge `permissions.allow` arrays (union, dedup), write back.
- [x] T009 Implement `lineEnsureMerge()` in `src/go/fab-kit/internal/sync.go` — read source lines, append non-duplicate non-comment lines to dest file.
- [x] T010 Implement multi-agent skill deployment in `src/go/fab-kit/internal/sync.go` — detect agents via PATH lookup (or `FAB_AGENTS` env override), deploy skills per agent format: Claude Code (copy to `.claude/skills/{name}/SKILL.md`), OpenCode (relative symlink to `.opencode/commands/{name}.md`), Codex (copy to `.agents/skills/{name}/SKILL.md`), Gemini (copy to `.gemini/skills/{name}/SKILL.md`).
- [x] T011 Implement stale skill cleanup in `src/go/fab-kit/internal/sync.go` — for each agent directory, remove skill entries not present in canonical `fab/.kit/skills/`. Write `fab/.kit-sync-version` stamp on completion.
- [x] T012 Implement prerequisites check in `src/go/fab-kit/internal/sync.go` — validate `git`, `bash`, `yq` (v4+), `jq`, `gh`, `direnv` are available. Run `direnv allow` after scaffold.
- [x] T013 Implement project-level sync script execution in `src/go/fab-kit/internal/sync.go` — discover `fab/sync/*.sh`, sort, exec each in order. Halt on non-zero exit.

## Phase 3: Integration & Edge Cases

- [x] T014 Update `justfile` — rename `build-shim` to `build-fab-kit`, add build recipe for router binary (`go build ./cmd/fab`), update `build-target` to build 5 binaries, update `build-all` for 20 cross-compiled binaries, update `build` for local dev.
- [x] T015 Update `scripts/just/package-brew.sh` — include `fab-kit` binary in brew archives (4 binaries: fab, fab-kit, wt, idea).
- [x] T016 Update `.github/workflows/release.yml` — upload 5th binary per platform, update brew archive creation, update Homebrew tap formula to install 4 binaries with `--version` tests.
- [x] T017 Remove absorbed shell scripts: `fab/.kit/scripts/fab-sync.sh`, `fab/.kit/sync/1-prerequisites.sh`, `fab/.kit/sync/2-sync-workspace.sh`, `fab/.kit/sync/3-direnv.sh`.
- [x] T018 Update `fab/.kit/scripts/fab-help.sh` — remove hardcoded `fab-sync.sh` entry from Setup group (replaced by `fab sync`).
- [x] T019 Update references to `fab-sync.sh` in `fab/project/context.md`, `fab/.kit/skills/_preamble.md` or other skill files that mention running `fab-sync.sh` — replace with `fab sync`.

## Phase 4: Polish

- [x] T020 Add unit tests for router dispatch logic in `src/go/fab-kit/cmd/fab/main_test.go` — test allowlist matching, not-in-repo error, config resolution.
- [x] T021 [P] Add unit tests for sync logic in `src/go/fab-kit/internal/sync_test.go` — test scaffold tree-walk strategies (JSON merge, line merge, copy-if-absent), skill deployment per agent, stale cleanup.
- [x] T022 [P] Add unit tests for `fab-kit` command in `src/go/fab-kit/cmd/fab-kit/main_test.go` — test `sync` subcommand registration, `--version`, `--help`.

---

## Execution Order

- T001 blocks all other tasks (module rename must happen first)
- T002 and T003 can run in parallel after T001
- T004 depends on T002 (router stub must exist)
- T005 depends on T004 (dispatch must work before composed help)
- T006 depends on T003 (fab-kit cmd must be renamed)
- T007–T013 depend on T006 (sync subcommand must be scaffolded)
- T007 blocks T008 and T009 (tree-walk dispatches to merge functions)
- T014–T019 can start after T013 (all Go code complete)
- T020–T022 run after their respective implementation tasks
