# Tasks: Ship fab Go Binary

**Change**: 260305-g0uq-2-ship-fab-go-binary
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create parity test directory structure at `src/go/fab/test/parity/` with fixture scaffolding under `src/go/fab/test/parity/fixtures/`
- [x] T002 [P] Create parity test fixtures: `.status.yaml` variants (pending, active, done stages), `config.yaml`, `constitution.md`, `workflow.yaml`, `.history.jsonl`, and change directory structures under `src/go/fab/test/parity/fixtures/`
- [x] T003 [P] Create `fab/.kit/bin/` directory with `.gitkeep` to establish the binary location convention

## Phase 2: Core Implementation

- [x] T004 Implement parity test harness at `src/go/fab/test/parity/parity_test.go` — shared helpers for running bash vs Go binary, diffing stdout/stderr/exit code/file mutations, temp dir isolation, prerequisite skip logic
- [x] T005 Add parity tests for `resolve.sh` vs `fab resolve` — cover `--id`, `--folder`, `--dir`, `--status` with change ID, substring, and full name inputs
- [x] T006 [P] Add parity tests for `logman.sh` vs `fab log` — cover `command`, `confidence`, `review`, `transition` subcommands
- [x] T007 [P] Add parity tests for `statusman.sh` vs `fab status` — cover `progress-map`, `progress-line`, `current-stage`, `start`, `advance`, `finish`, `reset`, `skip`, `fail`, `set-change-type`, `set-checklist`, `set-confidence`
- [x] T008 [P] Add parity tests for `preflight.sh` vs `fab preflight` — cover valid change, missing config, missing change, override resolution
- [x] T009 [P] Add parity tests for `changeman.sh` vs `fab change` — cover `new`, `rename`, `switch`, `list`, `resolve`
- [x] T010 [P] Add parity tests for `calc-score.sh` vs `fab score` — cover normal scoring, `--check-gate`, `--stage intake`, gate pass/fail
- [x] T011 [P] Add parity tests for `archiveman.sh` vs `fab archive` — cover `archive`, `restore`, `list`
- [x] T012 Add shim block to all 7 shell scripts in `fab/.kit/scripts/lib/`: `resolve.sh`, `logman.sh`, `statusman.sh`, `preflight.sh`, `changeman.sh`, `calc-score.sh`, `archiveman.sh` — consistent pattern with relative binary path resolution and `exec` delegation
- [x] T013 Add Go cross-compilation to `src/scripts/fab-release.sh` — build for darwin/arm64, darwin/amd64, linux/arm64, linux/amd64 with `CGO_ENABLED=0`, add Go toolchain pre-flight check
- [x] T014 Add per-platform archive packaging to `src/scripts/fab-release.sh` — produce 5 archives (4 platform-specific with binary at `.kit/bin/fab`, 1 generic without binary), upload all to GitHub Release, cleanup after
- [x] T015 Add platform detection and platform-specific download to `fab/.kit/scripts/fab-upgrade.sh` — detect OS/arch via `uname`, try `kit-{os}-{arch}.tar.gz` first, fall back to `kit.tar.gz`

## Phase 3: Integration & Edge Cases

- [x] T016 Update `fab/.kit/skills/_scripts.md` to document both legacy (bash scripts) and direct (Go binary) calling conventions with mapping table
- [x] T017 Update `fab/.kit/skills/_preamble.md` — replace `bash fab/.kit/scripts/lib/preflight.sh` with `fab/.kit/bin/fab preflight` and `bash fab/.kit/scripts/lib/logman.sh` with `fab/.kit/bin/fab log` in all invocation instructions
- [x] T018 Update skill callers in `fab/.kit/skills/`: `fab-new.md`, `fab-continue.md`, `fab-ff.md`, `fab-fff.md`, `fab-clarify.md`, `fab-switch.md`, `fab-status.md`, `fab-setup.md`, `fab-help.md`, `fab-discuss.md`, `fab-archive.md`, `git-branch.md`, `git-pr.md`, `git-pr-review.md` — replace `bash fab/.kit/scripts/lib/{script}.sh` invocations with `fab/.kit/bin/fab {command}` equivalents

## Phase 4: Polish

- [x] T019 Update `README.md` bootstrap one-liner to platform-aware version using `uname` detection and `kit-{os}-{arch}.tar.gz`
- [x] T020 Run `fab/.kit/scripts/fab-sync.sh` to deploy updated skill files to `.claude/skills/`

---

## Execution Order

- T001 blocks T004 (test harness needs directory structure)
- T002 blocks T005–T011 (tests need fixtures)
- T004 blocks T005–T011 (individual tests need shared harness)
- T005–T011 are parallelizable (independent script coverage)
- T012 is independent (shims don't depend on tests, but tests validate them)
- T013 blocks T014 (packaging needs built binaries)
- T016 blocks T017–T018 (preamble and skills reference _scripts.md conventions)
- T017 blocks T018 (skills derive patterns from preamble)
- T020 depends on T016–T018 (sync deploys updated skills)
