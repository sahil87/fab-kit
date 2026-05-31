# Plan: Add --skip-brew-update flag to update command

**Change**: 260531-vsge-skip-brew-update-flag
**Status**: In Progress
**Intake**: `intake.md`
**Spec**: `spec.md`

## Tasks

<!-- Sequential work items for the apply stage. Checked off [x] as completed. -->

### Phase 2: Core Implementation

<!-- Primary functionality. Order by dependency — earlier tasks are prerequisites for later ones. -->

- [x] T001 Change `Update()` signature in `src/go/fab-kit/internal/update.go` from `func Update(currentVersion string) error` to `func Update(currentVersion string, skipBrewUpdate bool) error` (plain positional bool, no options struct). <!-- A-001, A-004 -->
- [x] T002 In `src/go/fab-kit/internal/update.go`, wrap the existing `brew update --quiet` block (the `fmt.Println("Checking for updates...")` + `exec.Command("brew", "update", "--quiet")` + Stdout/Stderr wiring + `runWithTimeout`) in `if !skipBrewUpdate { ... }`, and change the later `brew upgrade` line from `cmd = exec.Command(...)` to `cmd := exec.Command(...)`. Leave the `isBrewInstalled()` guard, `brewLatestVersion()`/`brew info` check, the up-to-date short-circuit, and `brew upgrade` otherwise unchanged. Do NOT refactor the exec/subprocess style. <!-- A-002, A-005, A-006, A-007 -->
- [x] T003 In `src/go/fab-kit/cmd/fab-kit/main.go`, rewrite `updateCmd()` following the `syncCmd()` pattern: declare `var skipBrewUpdate bool`, build `cmd := &cobra.Command{...}` whose `RunE` calls `internal.Update(version, skipBrewUpdate)`, register `cmd.Flags().BoolVar(&skipBrewUpdate, "skip-brew-update", false, <help>)`, then `return cmd`. Flag name MUST be exactly `skip-brew-update`; default MUST be `false`. <!-- A-003, A-008 -->

### Phase 3: Integration & Edge Cases

<!-- Wire components together. Handle error states, edge cases, validation. -->

- [x] T004 Add the lowest-touch test seam in `src/go/fab-kit/internal/update.go` so the update-package test can drive the brew sequence under `go test` (where the binary path lacks `/Cellar/`): make `isBrewInstalled` a package-level function variable (`var isBrewInstalled = func() bool { ... }`). Production runtime behavior MUST be identical; no exec-style refactor. <!-- A-009, A-013 -->
- [x] T005 Create new file `src/go/fab-kit/internal/update_test.go` (package `internal`) following the `download_test.go`/`sync_test.go` convention (`t.TempDir()`, `t.Setenv()`, real fake `brew` executable on `PATH` — no exec-layer mocking). The fake `brew` logs each invocation's subcommand (`$1`) to a log file and emits valid `--json=v2` JSON for `info` reporting a stable version (`9.9.9`) different from `currentVersion` ("1.0.0"). Override the `isBrewInstalled` seam to return true (restore via `defer`). Two cases: (1) `Update("1.0.0", true)` — log contains `info` + `upgrade`, NOT `update`; (2) `Update("1.0.0", false)` — log contains `update` + `info` + `upgrade`. <!-- A-010, A-011, A-012, A-013 -->

### Phase 4: Polish

<!-- Build + test verification. -->

- [x] T006 From `src/go/fab-kit`, run `go build ./...`, `go test ./internal/ -run Update -v`, and `go vet ./...`; confirm all pass. <!-- A-014, A-015 -->

## Acceptance

<!-- Declarative acceptance criteria used by the review stage. ALL must pass before hydrate. -->

### Functional Completeness

<!-- Every requirement in spec.md has working implementation. -->

- [x] A-001 Update signature: `internal.Update` is defined as `func Update(currentVersion string, skipBrewUpdate bool) error` (plain positional bool). Confirmed update.go:16.
- [x] A-002 Skip gating: the `brew update --quiet` block (with its `fmt.Println("Checking for updates...")`) is wrapped in `if !skipBrewUpdate { ... }`; no other brew step is gated. Confirmed update.go:27-35.
- [x] A-003 Flag wiring: `updateCmd()` registers a flag named exactly `skip-brew-update` via `cmd.Flags().BoolVar`, default `false`, and `RunE` calls `internal.Update(version, skipBrewUpdate)`. Confirmed main.go:100-110; `--help` lists `--skip-brew-update` and unknown-flag typo is rejected (exit 1).
- [x] A-004 Caller updated: the package compiles and both callers pass the bool — `updateCmd()` (main.go:105, `skipBrewUpdate`) and the pre-existing `versionGuard()` (sync.go:118, `false`). NOTE: spec's "sole caller" wording is inaccurate (versionGuard also calls Update), but both call sites correctly thread the bool; behavior preserved.

### Behavioral Correctness

<!-- Changed requirements behave as specified, not as before. -->

- [x] A-005 Skip path: with `skipBrewUpdate == true`, `brew update` is not invoked while `brew info` and `brew upgrade` still run. Verified by test (skip case: log contains info+upgrade, not update; runtime output omits "Checking for updates...").
- [x] A-006 Default path: with `skipBrewUpdate == false`, the full sequence (`brew update` → `brew info` → up-to-date short-circuit → `brew upgrade`) runs byte-for-byte as before. Verified by test (default case: log contains update+info+upgrade) and code inspection (update.go:27-55).
- [x] A-007 Preserved steps: the `isBrewInstalled()` guard, `brewLatestVersion()`/`brew info` check, up-to-date short-circuit, and `brew upgrade` invocation are unchanged in behavior; only the `=`→`:=` change accompanies the `brew upgrade` line. Confirmed: `brew upgrade` now uses `cmd :=` (update.go:50); all other steps byte-identical.

### Removal Verification

<!-- No deprecated requirements in this change. -->

- [x] A-008 **N/A**: No requirements are removed by this additive, behavior-preserving change.

### Scenario Coverage

<!-- Key scenarios from spec.md have been exercised. -->

- [x] A-009 Brew-install guard preserved: when `isBrewInstalled()` is false, the manual-install guidance prints and no brew subcommand runs (production seam value returns the real check). Confirmed: default `isBrewInstalled` func var (update.go:88-98) performs the real `/Cellar/` resolution; guard returns early before any brew call (update.go:18-22). Not directly asserted in test, but logic is unchanged from prior behavior.
- [x] A-010 Test — skip omits update but keeps upgrade: the test asserts the brew log NOT containing `update` but containing `info` and `upgrade` for `Update("1.0.0", true)`. Confirmed update_test.go:30,64-73; test passes.
- [x] A-011 Test — default runs all three: the test asserts the brew log contains `update`, `info`, AND `upgrade` for `Update("1.0.0", false)`. Confirmed update_test.go:31,64-73; test passes.
- [x] A-012 Up-to-date short-circuit honored: the fake `brew info` returns a version different from `currentVersion` so `brew upgrade` is reached (short-circuit not triggered in test fixtures). Confirmed: fake brew emits stable `9.9.9` vs current `1.0.0` (update_test.go:19,54).

### Edge Cases & Error Handling

<!-- Error states, boundary conditions, failure modes. -->

- [x] A-013 Test seam isolation: the `isBrewInstalled` package-var seam is overridden only within the test and restored via `defer`; default (unset) behavior performs the real `/Cellar/` resolution. Confirmed update_test.go:50-52 (`orig := isBrewInstalled` … `defer func() { isBrewInstalled = orig }()`); production default at update.go:88-98.

### Code Quality

<!-- Baseline items + items derived from fab/project/code-quality.md Principles and Anti-Patterns. -->

- [x] A-014 Pattern consistency: new code follows the surrounding Go style — `updateCmd()` mirrors `syncCmd()`'s flag-wiring idiom (main.go:82-97 vs 99-111); the test mirrors `download_test.go`/`sync_test.go` conventions (`t.TempDir`, `t.Setenv`, fake executable on PATH). Confirmed.
- [x] A-015 No unnecessary duplication: existing helpers (`brewLatestVersion`, `runWithTimeout`, `brewFormula`) are reused; no reimplementation. Confirmed.
- [x] A-016 Readability over cleverness (Principle): the skip gate is a plain `if !skipBrewUpdate` wrapper; no clever indirection introduced. Confirmed update.go:27.
- [x] A-017 Follow existing patterns (Principle): the flag is threaded as a plain positional bool matching the existing `Update(currentVersion string)` signature style, not an options struct. Confirmed.
- [x] A-018 No God functions (Anti-Pattern): `Update()` remains focused and well under 50 lines of logic; no sprawling additions. Confirmed (Update is ~43 lines, update.go:16-59).
- [x] A-019 No magic strings/numbers (Anti-Pattern): the flag name `"skip-brew-update"` is a cobra registration literal (idiomatic, matches `syncCmd()`); timeouts and formula name reuse existing constants/literals already present — no new magic values introduced. Confirmed.

### Documentation Accuracy

<!-- extra_category from config.yaml. -->

- [x] A-020 **N/A**: Implementation-only, behavior-preserving change; no memory/spec docs document the internal `brew update` step (affected memory `(none)`), so no documentation changes are required.

### Cross References

<!-- extra_category from config.yaml. -->

- [x] A-021 **N/A**: No memory/spec docs are added or modified, so there are no cross-references to maintain.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
