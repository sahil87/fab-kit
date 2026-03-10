# Tasks: Rust Binary Port

**Change**: 260307-bmp3-3-rust-binary-port
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Scaffold Rust project at `src/fab-rust/`: `Cargo.toml` (clap, serde, serde_yaml, anyhow deps; release profile with lto=true, strip=true), `Cargo.lock`, and `src/main.rs` with clap derive CLI skeleton defining all 9 subcommands and their sub-subcommands (stubs returning `todo!()`)
- [x] T002 Port shared types and utilities: `src/fab-rust/src/types.rs` (StatusFile, Checklist, Confidence, Dimensions, StageMetric, StageState structs with serde derives; StageOrder constant, NextStage, StageNumber, ValidStates, AllowedStates maps). Port from `src/go/fab/internal/statusfile/statusfile.go` (types) and `src/go/fab/internal/status/status.go` (constants/maps)
- [x] T003 Port change resolution: `src/fab-rust/src/resolve.rs` (fab_root, to_folder, extract_id, to_dir, to_abs_dir, to_status, to_abs_status, extract_folder_from_symlink). Port from `src/go/fab/internal/resolve/resolve.go` (192 lines)

## Phase 2: Core Implementation

- [x] T004 Port statusfile YAML serialization: `src/fab-rust/src/statusfile.rs` (load, save with atomic temp+rename, get_progress/set_progress on serde_yaml::Value, get_progress_map). Must preserve YAML formatting on round-trip. Port from `src/go/fab/internal/statusfile/statusfile.go` (459 lines)
- [x] T005 [P] Port config + hooks modules: `src/fab-rust/src/config.rs` (load config.yaml, stage_hooks map) and `src/fab-rust/src/hooks.rs` (run shell command via sh -c). Port from `src/go/fab/internal/config/config.go` (51 lines) and `src/go/fab/internal/hooks/hooks.go` (32 lines)
- [x] T006 [P] Port log module: `src/fab-rust/src/log.rs` (command, confidence, review, transition — append JSONL to .history.jsonl with ISO 8601 timestamps). Port from `src/go/fab/internal/log/log.go` (134 lines)
- [x] T007 Port status state machine: `src/fab-rust/src/status.rs` (start, advance, finish, reset, skip, fail transitions; cascading rules; auto-activate next; metrics side effects; hooks integration; set_change_type, set_checklist, set_confidence, set_confidence_fuzzy, add_issue, add_pr, get_issues, get_prs, validate_status_file, progress_map, progress_line, current_stage, display_stage, all_stages, checklist, confidence). Port from `src/go/fab/internal/status/status.go` (608 lines)
- [x] T007a Port worktree module: `src/fab-rust/src/worktree.rs` (List all git worktrees via `git worktree list --porcelain`, FindByName, Current, resolve fab state per worktree, FormatHuman/FormatAllHuman/FormatJSON/FormatAllJSON output). Used by `status show` subcommand (`--all`, `--json` flags). Port from `src/go/fab/internal/worktree/worktree.go` (231 lines)
<!-- clarified: worktree module was missing from tasks — Go's status show subcommand delegates to internal/worktree package for discovery and formatting -->
- [x] T008 Port preflight module: `src/fab-rust/src/preflight.rs` (validate config+constitution exist, check sync staleness, resolve change, load status, format YAML output). Port from `src/go/fab/internal/preflight/preflight.go` (144 lines)
- [x] T009 Port score module: `src/fab-rust/src/score.rs` (parse Assumptions table from intake.md/spec.md, count grades, extract S/R/A/D dimensions, compute confidence score with cover factor, gate checking, format YAML output). Port from `src/go/fab/internal/score/score.go` (379 lines)
- [x] T010 Port change module: `src/fab-rust/src/change.rs` (new with random 4-char ID + collision detection, rename, switch + display output, switch_blank, list with stage/state/confidence, resolve passthrough). Port from `src/go/fab/internal/change/change.go` (378 lines)
- [x] T011 Port archive module: `src/fab-rust/src/archive.rs` (archive with nested YYYY/MM dirs + index.md update + backfill + pointer cleanup, restore with optional switch, list flat+nested entries, YAML output formatting). Port from `src/go/fab/internal/archive/archive.go` (435 lines)
- [x] T012 Port runtime module: `src/fab-rust/src/runtime.rs` (set-idle writes agent.idle_since timestamp, clear-idle deletes agent block, atomic YAML write). Port from `src/go/fab/cmd/fab/runtime.go` (177 lines)
- [x] T013 Port pane-map module: `src/fab-rust/src/panemap.rs` (discover panes via `tmux list-panes -a`, resolve git roots via `git rev-parse`, find main worktree via `git worktree list --porcelain`, resolve fab state, format aligned table with Pane/Worktree/Change/Stage/Agent columns, idle duration formatting). Port from `src/go/fab/cmd/fab/panemap.go` (344 lines)
- [x] T014 Port send-keys module: `src/fab-rust/src/sendkeys.rs` (resolve change to tmux pane, send text via `tmux send-keys`, handle multiple pane matches with warning). Port from `src/go/fab/cmd/fab/sendkeys.go` (134 lines)

## Phase 3: Integration & Edge Cases

- [x] T015 Wire all subcommands in `src/fab-rust/src/main.rs`: replace `todo!()` stubs with calls to ported modules. Ensure all flags, positional args, and sub-subcommands match Go binary's CLI signatures
- [x] T016 [P] Modify dispatcher `fab/.kit/bin/fab`: add backend override mechanism (FAB_BACKEND env var check, .fab-backend file read with whitespace trim, invalid/unavailable values fall through to default priority)
- [x] T017 [P] Update `justfile`: add `build-rust` recipe (cargo build --release + copy to fab/.kit/bin/fab-rust), update `test-rust` recipe (cargo test). Add `.fab-backend` to `.gitignore`

## Phase 4: Testing

- [x] T018 Create test infrastructure at `src/fab-rust/tests/`: test helpers (setup_temp_repo, copy fixtures, run binary, assert output), test fixtures (copy/adapt from `src/go/fab/test/parity/fixtures/`)
- [x] T019 Integration tests for core subcommands: resolve (folder/id/dir/status output, substring match, ambiguous match, symlink resolution), status (finish with auto-activate, reset with cascading, skip with cascading, fail for review, validate-status-file, show with --all and --json), change (new with ID generation, switch + display, list, rename)
<!-- clarified: added status show tests to T019 to cover worktree integration added in T007a -->
- [x] T020 Integration tests for remaining subcommands: preflight (valid repo, missing config, missing constitution, sync staleness), score (intake scoring, spec scoring, gate check pass/fail, fuzzy dimensions), log (command/confidence/review/transition JSONL format), archive (archive with index, restore, backfill), runtime (set-idle, clear-idle), pane-map (outside tmux error), send-keys (missing change error)
<!-- clarified: added pane-map and send-keys test coverage — spec has explicit scenario for pane-map outside tmux -->

---

## Execution Order

- T001 blocks all others (project scaffold)
- T002, T003 block T004-T014 (shared types and resolution are used everywhere)
- T004 (statusfile) blocks T007 (status), T008 (preflight), T009 (score), T010 (change), T011 (archive)
- T005 (config+hooks) and T006 (log) are parallel, both block T007 (status)
- T007 (status) blocks T010 (change), T011 (archive)
- T007a (worktree) depends on T002, T004 (uses statusfile and status types); blocks T015 (wiring)
- T015 (wiring) depends on T003-T014 + T007a (all modules ported)
- T016, T017 are independent of T003-T015
- T018 blocks T019, T020 (test infrastructure first)
