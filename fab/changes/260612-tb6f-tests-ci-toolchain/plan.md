# Plan: Tests, CI & Toolchain

**Change**: 260612-tb6f-tests-ci-toolchain
**Intake**: `intake.md`

## Requirements

### State Machine: `internal/status` coverage (F39)

#### R1: Exhaustive state-machine tests
The `internal/status` package SHALL have an exhaustive table-driven test over `lookupTransition` (stage × event × from-state) asserting allowed targets and rejections against the transition tables — including the review/review-pr `failed→active` start override and the AllowedStates target validation (`advance ship`, `advance review-pr`, `skip intake` rejected). The `Skip` forward-cascade (pending→skipped ordered iteration over `sf.StageOrder`), and the currently-0.0% functions `SetChangeType`, `AddIssue`, `ProgressMap`, `ProgressLine`, `AllStages`, plus `Advance`'s remaining branches, MUST gain direct tests. Tests MUST target post-#395–#402 main semantics (AllowedStates hardening, Save-under-lock), using the existing `loadFixture` helper.

- **GIVEN** the full matrix of 6 stages × events (start/advance/finish/reset/skip/fail) × 6 from-states
- **WHEN** `lookupTransition` is invoked for each cell
- **THEN** every cell's outcome (target state or error) matches the transition tables + AllowedStates validation, with no untested cell

- **GIVEN** a status file with apply active and downstream stages pending
- **WHEN** `Skip(apply)` runs
- **THEN** apply → skipped and every downstream `pending` stage cascades to `skipped` in `StageOrder` order, while non-pending downstream states are untouched

#### R2: `stage_metrics` iterations truthfulness
A regression test SHALL assert `stage_metrics.review.iterations` accumulates across the fail→reset→re-finish rework choreography (`fail review` → `reset apply` → re-finish apply → review re-activates). Per constitution VII, if the test exposes the observed reset-to-1 bug, the implementation fix rides in this change; #395's actual implementation is checked first.

- **GIVEN** a change whose review has run once (iterations=1) and failed
- **WHEN** the choreography `fail review`, `reset apply`, `finish apply` (review re-activates) executes twice more
- **THEN** `stage_metrics.review.iterations` reads 3 after the third activation — preserved across each cascade to `pending`, incremented on each re-activation

### fab-kit: destructive workspace mutator coverage (F45)

#### R3: Sync orchestrator, cleanLegacyAgents, and Upgrade tests
The 0.0%-covered `Sync` orchestrator and `cleanLegacyAgents` (the file-deleting path) MUST gain integration-style tests: a temp git repo + fake cached kit dir, running `Sync` twice and asserting (a) the resulting tree is correct and (b) the second run is a content-identical no-op (constitution III idempotency — compare content, not mtimes). The same harness SHALL cover the `shimOnly`/`projectOnly` branch split and `cleanLegacyAgents`' deletion scoping (legacy targets deleted, project files never). `Upgrade`'s remaining branches (46.4%) SHOULD be covered, including dn2c's stamp-after-success ordering contract. Feasibility seams: `FAB_AGENTS` env override, `os.Setenv("HOME", tmp)` precedent in `cache_test.go`, `systemVersion="dev"` bypasses versionGuard, PATH shims for `checkPrerequisites` (git/bash/yq-v4/direnv).

- **GIVEN** a temp git repo and a fake `~/.fab-kit/versions/dev/kit/` cache
- **WHEN** `Sync("dev", "dev", false, false)` runs twice
- **THEN** the first run produces the expected workspace tree and the second run changes no file content

- **GIVEN** a workspace containing legacy agent files alongside project files
- **WHEN** `cleanLegacyAgents` runs
- **THEN** only the legacy-scoped files are deleted; nothing outside the documented deletion scope is touched

### CI/CD: release gate, darwin compile, toolchain (F40, F42, F41)

#### R4: Release workflow test gate + single-source Go version
`.github/workflows/release.yml` MUST run the test suite (`just test`) before any build/package/release step — positioned so a manual-dispatch release cannot even create the tag on a red suite — and MUST replace the hardcoded `go-version: '1.22'` with `go-version-file:` pointing at a module go.mod, making ci.yml's "single source of truth" comment true. The release-workflow step list in `docs/memory/distribution/distribution.md` (~lines 299-306) SHALL be updated.

- **GIVEN** a `v*` tag push (or manual dispatch) with a failing test in either Go module
- **WHEN** the release workflow runs
- **THEN** the workflow fails before `just build-all` and no release assets are published

#### R5: Darwin cross-compile in CI + race detector
`.github/workflows/ci.yml` MUST add a per-module-leg step `GOOS=darwin GOARCH=arm64 go build ./... && GOOS=darwin GOARCH=arm64 go vet ./...` (both matrix legs — harmless and future-proof; only fab strictly needs it for `proc_darwin.go`/`pane_process_darwin.go`), and `-race` SHALL be added to the existing linux `go test` step.

- **GIVEN** a PR introducing a type error in a `_darwin.go` file
- **WHEN** CI runs
- **THEN** the darwin cross-compile step fails on linux without needing a macOS runner

#### R6: Toolchain bump (go + cobra)
Both `go.mod` files MUST bump the `go` directive from 1.22 to the current supported line (1.26 — local toolchain 1.26.2) and cobra from v1.8.1 to the latest v1.10.x. CI picks the Go version up via `go-version-file`; ye8r's contract/collision drift tests guard the CLI surface through the cobra bump. The "Go 1.22" mentions in `docs/memory/distribution/distribution.md` (~:300) and `kit-architecture.md` (~:293) SHALL be updated.

- **GIVEN** the bumped go directive and cobra version
- **WHEN** `go test ./...` runs in both modules
- **THEN** the full suite passes, including ye8r's lifecycle-collision and contract drift tests

#### R7: yaml.v3 golden byte-stability tests, then evaluation
Golden round-trip tests asserting byte-stable output MUST land FIRST (before any toolchain/yaml change) for `.status.yaml` (statusfile load→Save) and the generated memory/archive indexes (byte-stability is a documented `fab memory-index` property). Only then is goccy/go-yaml evaluated: the swap happens ONLY on proven byte-identical round-trip parity under those tests; otherwise yaml.v3 stays pinned and the archived status (April 2025) + migration plan is recorded in memory.

- **GIVEN** the golden tests passing against yaml.v3
- **WHEN** goccy/go-yaml is substituted in a scratch evaluation
- **THEN** the swap is adopted only if every golden byte-comparison still passes; otherwise the pin + rationale is recorded in `docs/memory/distribution/kit-architecture.md`

### CLI: cmd/fab cobra wiring coverage (F46)

#### R8: Low-coverage RunE bodies exercised
The in-package cobra-execution pattern (exemplars: `memory_index_test.go:23-100`'s setupFabRepo + os.Chdir + SetArgs + stdout/file assertions, `pr_meta_test.go`, `shellinit_test.go` — NOT `batch_new_test.go`) SHALL be extended to the measured low-coverage RunE bodies: `changeArchiveListCmd` (10.0%), `logReviewCmd` (12.5%), `changeArchiveCmd` (23.5%), `changeSwitchCmd` (23.5%), `changeRestoreCmd` (28.6%), `changeListCmd` (35.7%), `changeRenameCmd` (38.5%), `listPendingItems` (27.3%). Tests MUST assert the exact stdout shapes skills parse: the archive command's structured YAML, the `already archived:` soft-skip line, and hv7t's `index: failed` print-then-error contract (non-zero exit with YAML still printed).

- **GIVEN** a temp fab repo with an archivable change
- **WHEN** `fab change archive <id>` executes via cobra
- **THEN** stdout carries the structured YAML skills parse; re-archiving emits the `already archived:` soft-skip (exit 0); an index write failure still prints YAML with `index: failed` and exits non-zero

### Structure: fab-kit sync.go split (F44)

#### R9: Mechanical split + semver compare bug fix
`src/go/fab-kit/internal/sync.go` (886 lines) SHALL be split within the existing flat internal package — no API changes, tests move with their functions: `semver.go` (parseSemver/compareSemver), `prereqs.go` (checkPrerequisites incl. yq version sniffing), `scaffold.go` (tree-walk + JSON-permissions merge + line-ensure merge), `skills.go` (deploySkills/listSkills/syncAgentSkills/cleanStaleSkills/cleanLegacyAgents), with `sync.go` keeping the ~100-line orchestrator + versionGuard. The `major < "4"` lexicographic string-compare bug in `checkPrerequisites` MUST be fixed while moving (with a regression test — it misorders a hypothetical yq v10). `docs/memory/distribution/migrations.md` (~:62) which cites "parseSemver/compareSemver helpers in sync.go" SHALL be touched up.

- **GIVEN** yq reporting version 10.0.0 on PATH
- **WHEN** `checkPrerequisites` evaluates the major version
- **THEN** yq v10 passes the v4+ requirement (numeric compare, not lexicographic)

### Cleanup: dead code & vestigial removal (F47)

#### R10: Verified dead code deleted
The following MUST be removed/consolidated (all verified present on this branch): `resolve.ToDir`/`resolve.ToStatus` deleted with their tests (cmd/fab/resolve.go re-inlines the format strings — delete per intake assumption #8); unused `change.List` wrapper deleted; the duplicate digit-only int parsers collapsed (`status.parseInt` deleted in favor of an exported `statusfile` parser — `status` already imports `statusfile`, no cycle); hv7t's three recorded deletion candidates absorbed (`backlog.go:117` MarkDone inline ReadFile+Split → `internal/lines`, `intake.go` Title same idiom → `internal/lines` with CRLF-trim parity confirmed, `changetypes_doc_test.go:140` test-local bufio.Scanner → `internal/lines`); `src/benchmark/` implementations deleted (statusman-go/node/rust/bash-opt + harness) keeping `RESULTS.md` + `README.md` as the decision record. Exclusions — `stage_hooks` and `model_tiers` MUST NOT be touched (c5tr owns those; re-verify residue only).

- **GIVEN** the cleanup applied
- **WHEN** `grep -r "ToDir\|ToStatus" src/go/fab --include="*.go"` and `go build ./...` run
- **THEN** no non-test references remain, both modules build, and `src/benchmark/` contains only RESULTS.md + README.md

### Docs: memory accuracy (cross-cutting)

#### R11: Affected memory updated in-change
The three `docs/memory/distribution/` files SHALL be updated where the intake's What Changes explicitly demands it (release step list + Go version in `distribution.md`; Go version, sync-split description, sole-survivor statusman record in `kit-architecture.md`; sync.go path cite in `migrations.md`). Hydrate later merges without duplication.

- **GIVEN** the CI/toolchain/split tasks completed
- **WHEN** the three memory files are read
- **THEN** no stale "Go 1.22", hardcoded-version, monolithic-sync.go, or dead benchmark-implementation references remain

### Non-Goals

- F43 (move domain logic out of cmd/fab) as a committed deliverable — only pieces that fall out naturally while testing; if nothing falls out, skip entirely
- Re-implementing tests sibling batches already added (refreshed coverage numbers in the intake define the actual gaps)
- Re-filing anything in the report's Refuted section (R1–R8) — notably the hooksync duplication between the two modules is a documented deliberate decision; do not "deduplicate" it while splitting sync.go
- New CLI commands or signature changes (pure test/CI/structure batch; `_cli-fab.md` untouched unless a signature unexpectedly changes)
- No skill-file (`src/kit/`) changes

### Design Decisions

1. **Golden tests before any yaml decision**: the byte-stability tests are written and passing against yaml.v3 before the goccy evaluation runs — *Why*: they are the objective parity arbiter (intake assumption #5) — *Rejected*: evaluating by inspection of goccy's docs (unverifiable)
2. **Delete `ToDir`/`ToStatus` rather than routing the command through them** — *Why*: user-confirmed (intake assumption #8); the inlined format strings in cmd/fab/resolve.go are the live spec — *Rejected*: re-routing (adds indirection for zero callers)
3. **Test step placement in release.yml before tag creation** — *Why*: on `workflow_dispatch` the tag is created by the workflow itself; testing first means a red suite never even mints the tag — *Rejected*: test step after checkout-only (would still tag on red for manual dispatch)

## Tasks

### Phase 1: Safety nets & mechanical restructure

- [x] T001 Golden byte-stability tests: `.status.yaml` load→Save round-trip golden test in `src/go/fab/internal/statusfile/` and generated memory/archive index byte-stability tests in `src/go/fab/internal/memoryindex/` (or alongside existing index tests). Must pass against current yaml.v3 before any toolchain task runs. <!-- R7 -->
- [x] T002 Split `src/go/fab-kit/internal/sync.go` into `semver.go`, `prereqs.go`, `scaffold.go`, `skills.go` (orchestrator + versionGuard stay in `sync.go`); move existing tests with their functions; fix the `major < "4"` lexicographic compare in `checkPrerequisites` with a regression test; update `docs/memory/distribution/migrations.md` ~:62 path cite. <!-- R9 -->

### Phase 2: Test coverage

- [x] T003 [P] Exhaustive table-driven `lookupTransition` matrix test in `src/go/fab/internal/status/status_test.go` (6 stages × start/advance/finish/reset/skip/fail × 6 from-states), incl. failed→active override and AllowedStates rejections. <!-- R1 -->
- [x] T004 [P] `Skip` forward-cascade tests (ordered pending→skipped cascade; non-pending downstream untouched) in `src/go/fab/internal/status/status_test.go`. <!-- R1 -->
- [x] T005 [P] Direct tests for `SetChangeType`, `AddIssue`, `ProgressMap`, `ProgressLine`, `AllStages` + `Advance` remaining branches in `src/go/fab/internal/status/status_test.go`. <!-- R1 -->
- [x] T006 `stage_metrics` iterations regression test (fail→reset→re-finish ×2 accumulates to 3) in `src/go/fab/internal/status/status_test.go`; check #395's implementation first; fix implementation if the test exposes the reset-to-1 bug (constitution VII). <!-- R2 -->
- [x] T007 `Sync` integration test in `src/go/fab-kit/internal/sync_test.go`: temp git repo + fake cached kit, run twice, assert correct tree + content-identical no-op second run; cover `shimOnly`/`projectOnly` branch split. <!-- R3 -->
- [x] T008 `cleanLegacyAgents` deletion-scoping tests in the same harness (`src/go/fab-kit/internal/sync_test.go` or `skills_test.go` post-split). <!-- R3 -->
- [x] T009 [P] `Upgrade` remaining-branch tests in `src/go/fab-kit/internal/upgrade_test.go` (stamp-after-success ordering, short-circuit, VERSION-file verification). <!-- R3 -->
- [x] T010 [P] Cobra-execution tests in `src/go/fab/cmd/fab/` for `changeArchiveCmd`, `changeArchiveListCmd`, `changeSwitchCmd`, `changeRestoreCmd`, `changeListCmd`, `changeRenameCmd`, `logReviewCmd`, `listPendingItems` — asserting archive structured YAML, `already archived:` soft-skip, `index: failed` print-then-error (non-zero exit, YAML still printed). <!-- R8 -->

### Phase 3: CI & toolchain

- [x] T011 `.github/workflows/release.yml`: add `just test` step before tag creation/build; replace `go-version: '1.22'` with `go-version-file: src/go/fab/go.mod`; update `docs/memory/distribution/distribution.md` release step list (~:299-306) and Go 1.22 mention (~:300). <!-- R4 -->
- [x] T012 [P] `.github/workflows/ci.yml`: add darwin cross-compile step (`GOOS=darwin GOARCH=arm64 go build ./... && ... go vet ./...`) to both matrix legs; add `-race` to the `go test` step. <!-- R5 -->
- [x] T013 Bump `go` directive to 1.26 + cobra to latest v1.10.x in both `src/go/fab/go.mod` and `src/go/fab-kit/go.mod`; run full suites; update "Go 1.22" in `docs/memory/distribution/kit-architecture.md` ~:293. <!-- R6 -->
- [x] T014 goccy/go-yaml evaluation against the T001 golden tests; swap only on byte-identical parity; otherwise stay pinned and record yaml.v3 archived status + migration plan in `docs/memory/distribution/kit-architecture.md`. <!-- R7 -->

### Phase 4: Cleanup & polish

- [x] T015 Delete `resolve.ToDir`/`resolve.ToStatus` (+ their tests in `src/go/fab/internal/resolve/resolve_test.go`) and unused `change.List` in `src/go/fab/internal/change/change.go`; collapse `status.parseInt` into an exported `statusfile` parser. <!-- R10 -->
- [x] T016 Absorb hv7t deletion candidates: `src/go/fab/internal/backlog/backlog.go` MarkDone → `internal/lines`; `src/go/fab/internal/intake/intake.go` Title → `internal/lines` (confirm CRLF-trim parity); `src/go/fab/internal/score/changetypes_doc_test.go` bufio.Scanner → `internal/lines`. <!-- R10 -->
- [x] T017 Delete `src/benchmark/` implementations (statusman-go, statusman-node, statusman-rust, statusman-bash-opt, bench.sh, fixtures) keeping `RESULTS.md` + `README.md`; re-verify `stage_hooks`/`model_tiers` residue untouched; update `kit-architecture.md` so its statusman decision record is the sole survivor reference. <!-- R10 -->
- [x] T018 Final verification: `gofmt -l` clean, `go vet ./...`, `go test ./...` in both modules, and local `GOOS=darwin GOARCH=arm64 go build ./... && GOOS=darwin GOARCH=arm64 go vet ./...` validating the F42 step. <!-- R5 -->

## Execution Order

- T001 blocks T013 and T014 (golden tests must exist before any toolchain/yaml change)
- T002 blocks T007, T008 (fab-kit tests written against the split file layout)
- T003–T006, T009, T010 are independent of Phase 3
- T011–T012 are independent of each other; T013 before T014 (evaluate goccy on the bumped toolchain)
- Phase 4 runs last; T018 is the final gate

## Acceptance

### Functional Completeness

- [x] A-001 R1: An exhaustive table-driven `lookupTransition` test covers every stage × event × from-state cell, incl. the review/review-pr failed→active start override and AllowedStates rejections (`advance ship`, `advance review-pr`, `skip intake`)
- [x] A-002 R1: `Skip` forward-cascade, `SetChangeType`, `AddIssue`, `ProgressMap`, `ProgressLine`, `AllStages`, and `Advance` branches have direct passing tests (none of the five 0.0% functions remains untested)
- [x] A-003 R2: A regression test asserts `stage_metrics.review.iterations` accumulates across fail→reset→re-finish; implementation conforms to spec (fixed in-change if the reset-to-1 bug was exposed)
- [x] A-004 R3: `Sync` twice-run integration test passes — correct tree, content-identical no-op second run (constitution III encoded)
- [x] A-005 R3: `cleanLegacyAgents` deletion scoping and `shimOnly`/`projectOnly` branch split are tested in the same harness; `Upgrade` stamp-after-success ordering covered
- [x] A-006 R4: release.yml runs the test suite before tag creation and build — a red suite ships nothing
- [x] A-007 R4: release.yml uses `go-version-file:` (no hardcoded Go version); ci.yml's "single source of truth" comment is now true
- [x] A-008 R5: ci.yml cross-compiles darwin (build + vet) on both matrix legs and runs the linux tests with `-race`
- [x] A-009 R6: Both go.mod files declare go 1.26 and cobra v1.10.x; full suites green post-bump
- [x] A-010 R7: Golden byte-stability tests for `.status.yaml` round-trip and generated indexes exist and pass, and landed before any yaml/toolchain change
- [x] A-011 R7: The yaml decision is recorded — swap only on byte-identical parity, else yaml.v3 stays pinned with archived status + migration plan in memory
- [x] A-012 R8: The eight measured low-coverage cmd/fab RunE bodies are exercised via cobra execution with exact stdout-shape assertions (archive YAML, `already archived:`, `index: failed` print-then-error)
- [x] A-013 R9: sync.go is split into the five named files with no API changes; tests moved with their functions; sync.go retains only the orchestrator, versionGuard, and the orchestrator-owned step helpers (gitRepoRoot, runDirenvAllow, runProjectSyncScripts) — ~220 lines, down from 886
- [x] A-014 R9: The yq lexicographic major-compare bug is fixed with a regression test (v10 passes the v4+ check)
- [x] A-015 R10: `resolve.ToDir`/`ToStatus` and `change.List` are gone with no non-test references; the duplicate int parsers are collapsed to one
- [x] A-016 R10: hv7t's three deletion candidates are absorbed (`internal/lines` reused; CRLF parity confirmed)
- [x] A-017 R10: `src/benchmark/` contains only RESULTS.md + README.md; `stage_hooks` and `model_tiers` are untouched

### Scenario Coverage

- [x] A-018 R2: The fail→reset→re-finish choreography scenario (iterations 1→2→3) is exercised end-to-end through the public Fail/Reset/Finish functions, not just unit internals
- [x] A-019 R3: The legacy-agent deletion scenario proves project files outside the documented scope survive

### Edge Cases & Error Handling

- [x] A-020 R8: `index: failed` contract verified — non-zero exit with YAML still printed on stdout
- [x] A-021 R1: Transition rejections assert error (no write), not just non-success

### Code Quality

- [x] A-022 Pattern consistency: New tests follow the existing in-package patterns (loadFixture, setupFabRepo, t.TempDir; table-driven style)
- [x] A-023 No unnecessary duplication: existing helpers (`internal/lines`, fixtures, PATH-shim precedents) reused; no parallel utilities introduced
- [x] A-024 No god functions: split sync.go files keep functions focused; no >50-line functions added without clear reason

### Documentation Accuracy

- [x] A-025 R11: The three `docs/memory/distribution/` files reflect the shipped state — release step list with test gate, go-version-file, no "Go 1.22", split sync.go paths, benchmark decision record as sole survivor

### Cross References

- [x] A-026 R11: Updated memory files' internal cross-references resolve (no stale `sync.go` path cites or links to deleted benchmark implementations)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- F43 stretch: only pieces that fall out naturally during T003–T010; none committed

## Deletion Candidates

- `src/go/fab/internal/status/status.go:49-51,56-58` (identical `advance`/`finish`/`reset` rows in `stageTransitions["review"]` and `["review-pr"]`) — duplicate `defaultTransitions` byte-for-byte (only `start`/`fail` genuinely differ); `lookupTransition` falls through to the default table for events absent from a stage override, so removal is behavior-neutral — the review-pr `advance` row is already a recorded k4ge deletion candidate (schemas.md §3), and this change's new exhaustive matrix test (`transitions_test.go`) now proves any such removal safe; F47's sweep absorbed hv7t's candidates but not this one
- `src/go/fab-kit/internal/skills.go:168-195` (`syncAgentSkills` symlink-mode branch + the `agentConfig.Mode` field that selects it) — all four agent configs deploy in copy mode and kit-architecture.md documents the symlink branch as "currently unused"; the F44 split moved it into `skills.go` without retiring it

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `src/benchmark/` deletion includes `bench.sh` + `fixtures/` alongside the four implementations | They exist solely to drive the deleted implementations (harness + test data); intake assumption #9 names "all four implementations, keep RESULTS.md + README.md" — a dead harness with nothing to run contradicts the cleanup intent | S:80 R:85 A:90 D:70 |
| 2 | Confident | Go directive set to `go 1.26` (line-level), matching the existing `go 1.22` line-level convention | Intake says "current supported line (1.26.x)"; line-level lets setup-go's `go-version-file` track 1.26 patches; exact-pin rejected as deviating from the file's existing convention | S:80 R:90 A:85 D:75 |
| 3 | Confident | parseInt consolidation direction: export `ParseIntStrict` from `statusfile`, delete `status.parseInt` | Intake fixes the direction ("status already imports statusfile, no cycle"); exporting is the minimal change preserving both call sites | S:85 R:90 A:90 D:80 |
| 4 | Confident | The intake-mandated memory-doc edits (distribution/kit-architecture/migrations) are performed during apply, not deferred to hydrate | The intake's What Changes lists them as in-change actions citing Docs Are Source of Truth; hydrate merges without duplication, so doing them now is safe and reviewable | S:80 R:95 A:90 D:75 |
| 5 | Confident | release.yml test step uses `just test` directly (not a reusable `needs:` job shared with ci.yml) | Intake offers either; `just test` keeps the "CI uses the exact same just commands a developer runs locally" principle and avoids restructuring ci.yml into a reusable workflow | S:80 R:90 A:90 D:70 |

5 assumptions (0 certain, 5 confident, 0 tentative).
