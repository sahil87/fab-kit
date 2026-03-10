# Tasks: Build fab Go Binary

**Change**: 260305-bhd6-1-build-fab-go-binary
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Scaffold Go module at `src/go/fab/`: create `go.mod` (module `github.com/wvrdz/fab-kit/src/go/fab`, Go 1.22), `cmd/fab/main.go` (cobra root command), and `internal/` package directories. Run `go mod tidy` to fetch cobra and yaml.v3 dependencies.

## Phase 2: Core Implementation

- [x] T002 Implement `internal/statusfile/statusfile.go`: define `StatusFile` struct matching `.status.yaml` schema (name, created, created_by, change_type, issues, progress as ordered map, checklist, confidence with dimensions, stage_metrics, prs, last_updated). Implement `Load(path) (*StatusFile, error)` and `Save(path) error` with atomic temp+rename writes. Add `statusfile_test.go` with round-trip fidelity test.
- [x] T003 [P] Implement `internal/resolve/resolve.go`: change folder resolution logic — read `fab/current` line 2 as default, match by exact name → case-insensitive substring → 4-char ID extraction from folder names. Handle multiple matches (error), no matches (error), single-change fallback. Output modes: id, folder, dir, status. Register `fab resolve` cobra command at `cmd/fab/resolve.go`.
- [x] T004 [P] Implement `internal/log/log.go`: JSON-line append to `.history.jsonl`. Four log types: command, confidence, review, transition. Each appends `{"ts":"...","event":"..."}` JSON line. Handle graceful degradation for command subtype with no change arg. Register `fab log` cobra command at `cmd/fab/log.go`.
- [x] T005 Implement `internal/status/status.go`: full stage state machine. Query subcommands: all-stages, progress-map, progress-line, current-stage, display-stage, checklist, confidence, validate-status-file, get-issues, get-prs. Event subcommands: start, advance, finish (with auto-activate next + review auto-log), reset (with downstream cascade), skip (with downstream cascade), fail (review/review-pr only + auto-log). Write subcommands: set-change-type, set-checklist, set-confidence, set-confidence-fuzzy, add-issue, add-pr. Stage metrics tracking (started_at, driver, iterations, completed_at). Register `fab status` cobra command group at `cmd/fab/status.go`.
- [x] T006 Implement `internal/preflight/preflight.go`: validation checks (config.yaml exists, constitution.md exists, sync version staleness warning, change resolution, change dir exists, .status.yaml exists, schema validation). Produce structured YAML output matching preflight.sh format exactly. Register `fab preflight` cobra command at `cmd/fab/preflight.go`.
- [x] T007 [P] Implement `internal/change/change.go`: change lifecycle — `new` (slug validation, random 4-char ID generation, collision detection, template initialization, created_by detection via `gh api user` → `git config` → "unknown", logman integration), `rename` (folder rename, .status.yaml name update, fab/current update), `switch` (write fab/current, display summary with stage/confidence/next), `list` (scan directories, format output), `resolve` (passthrough). Register `fab change` cobra command group at `cmd/fab/change.go`.
- [x] T008 [P] Implement `internal/score/score.go`: parse `## Assumptions` markdown table from spec.md/intake.md, count grades, compute score with penalty weights (confident=0.3, tentative=1.0) and coverage factor (expected_min lookup by stage+type), handle dimension means for fuzzy scores, gate check mode. Register `fab score` cobra command at `cmd/fab/score.go`.
- [x] T009 Implement `internal/archive/archive.go`: archive (clean .pr-done, move to archive/, create/update index.md, backfill unindexed, clear fab/current), restore (move from archive/, remove index entry, optional switch), list (enumerate archive/ directories). Register `fab archive` cobra command group at `cmd/fab/archive.go`.

## Phase 3: Integration & Edge Cases

- [x] T010 Wire all subcommands into `cmd/fab/main.go` root command. Verify `fab --help` lists all subcommands. Verify `fab <subcmd> --help` works for each.
- [x] T011 Add integration tests: create test fixtures (sample .status.yaml, sample change directories) and test key end-to-end flows — preflight happy path, status finish with cascade, change new + switch, score computation, archive + restore round-trip. Tests in `internal/*/..._test.go`.
- [x] T012 Verify build: `cd src/go/fab && go build -o fab ./cmd/fab/` produces a single static binary. Verify `go vet ./...` and `go test ./...` pass clean.

---

## Execution Order

- T001 blocks all other tasks (module scaffold required first)
- T002 blocks T005, T006, T007, T008, T009 (statusfile is the shared foundation)
- T003, T004 are independent of T002 — can run in parallel with it
- T005 depends on T002 (statusfile), T003 (resolve), T004 (log)
- T006 depends on T005 (status), T003 (resolve), T007 (change)
- T007 depends on T002 (statusfile), T003 (resolve), T004 (log), T005 (status)
- T008 depends on T002 (statusfile), T003 (resolve), T004 (log), T005 (status)
- T009 depends on T003 (resolve), T007 (change)
- T010 depends on all Phase 2 tasks
- T011 depends on T010
- T012 depends on T010
