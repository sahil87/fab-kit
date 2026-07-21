# Plan: Fix tmux socket leak in Go integration tests

**Change**: 260721-0j0t-fix-tmux-test-socket-leak
**Intake**: `intake.md`

## Requirements

### Test Infrastructure: tmux socket lifecycle

#### R1: Per-test private TMUX_TMPDIR via process env
Each of the three tmux integration tests (`TestReadAgentStateOption_Integration` in `src/go/fab/internal/pane/pane_test.go`, `TestPaneSendGate_Integration` in `src/go/fab/cmd/fab/pane_send_test.go`, `TestMapSendAgentAgreement_Integration` in `src/go/fab/cmd/fab/panemap_test.go`) MUST set `TMUX_TMPDIR` to a per-test private directory via `t.Setenv` (process-level env, NOT `cmd.Env` scoped to the test's local `tmux` closure), placed after the `exec.LookPath("tmux")` skip guard and before the first `tmux(...)` invocation. The private directory (and the socket tmux creates at `$TMUX_TMPDIR/tmux-$UID/<name>` inside it) MUST be deleted when the test ends regardless of how tmux exits.

- **GIVEN** a machine with tmux installed
- **WHEN** any of the three integration tests runs to completion (pass, fail, or abrupt teardown after server start)
- **THEN** the tmux socket file is created under the test's private `TMUX_TMPDIR`, not the shared `/tmp/tmux-$UID`
- **AND** the socket file no longer exists after the test ends (the temp dir is removed)
- **AND** the production code under test (`ReadAgentStateOption`, `fab pane send`, `fab pane map`) resolves the SAME private socket dir via inherited process env — the test and the code-under-test talk to one server

#### R2: Short fixed socket names replace nano-timestamp names
The nano-timestamp server names (`"fabtest-" + strconv.FormatInt(time.Now().UnixNano(), 36)` and variants) MUST be replaced with short fixed names — `fabtest`, `fabtest-send`, `fabtest-agree` — kept distinct across the three tests for debuggability. Uniqueness now comes from the per-test private `TMUX_TMPDIR`, and short names keep the socket path inside the `sun_path` budget (R3).

- **GIVEN** the three integration tests after this change
- **WHEN** grepping the three files for `UnixNano`-based server names
- **THEN** none remain; each test uses its short fixed name

#### R3: sun_path length guard with short-dir fallback — never a silent skip
The `TMUX_TMPDIR` selection MUST guard the Unix socket path length: compute the candidate dir (`t.TempDir()`); if the full socket path `{dir}/tmux-{uid}/{name}` would exceed a conservative budget of 103 bytes (macOS caps `sun_path` at 104 bytes including the terminating NUL), fall back to a short directory created via `os.MkdirTemp("/tmp", "fabtest-")` and registered with `t.Cleanup(func() { os.RemoveAll(dir) })` so the leak guarantee (R1) holds on the fallback path too. The guard MUST NOT let an over-long path degrade into the existing `t.Skipf("could not start tmux server ...")` — path-length problems are detectable and fixable, unlike a missing tmux binary (which keeps its `t.Skip`). The budget MUST be a named constant, not a magic number.

- **GIVEN** a macOS machine whose `$TMPDIR` base makes `t.TempDir()` for the longest test name exceed the budget (measured: 49-char base → ~98-char temp dir)
- **WHEN** the integration test selects its `TMUX_TMPDIR`
- **THEN** the short `/tmp`-based fallback dir is used, tmux binds successfully, and the test RUNS (is not skipped)
- **AND** the fallback dir is removed at test end via `t.Cleanup`

#### R4: Existing kill-server cleanup retained
The `t.Cleanup(func() { _, _ = tmux("kill-server") })` in all three tests MUST be retained — it still frees the server *process* promptly; the temp-dir deletion is what now removes the socket *file*.

- **GIVEN** any of the three integration tests
- **WHEN** the test ends
- **THEN** `tmux kill-server` still runs against the test's private server

#### R5: Verified leak-free and exercised
The change MUST be verified by: (a) `go test ./internal/pane/ ./cmd/fab/ -run '_Integration'` from `src/go/fab` passing with tmux installed, with the integration paths actually exercised (PASS, not SKIP); (b) a before/after snapshot of `/tmp/tmux-$UID` showing **zero new socket files** from a test run; (c) all existing unit tests in the three files remaining green.

- **GIVEN** a tmux-equipped machine
- **WHEN** the integration suite runs and `/tmp/tmux-$UID` is snapshotted before and after
- **THEN** the file count and names are unchanged (zero new sockets) and the `-v` output shows the three integration tests PASS

### Non-Goals

- No cleanup of already-accumulated stale sockets in `/tmp/tmux-$UID` — machine state, removable manually; not a repo artifact (intake assumption 8)
- No production `.go` behavior changes, no CLI signature changes (no `_cli-fab.md` update), no skill/SPEC/memory updates — test-infrastructure-only change

### Design Decisions

#### Per-package test helper instead of triplicated inline logic
**Decision**: Extract a small unexported helper (`tmuxSocketDir(t, name)` + `tmuxSocketPathLen(dir, name)` + a named budget constant) once per package — in `internal/pane/pane_test.go` and in `cmd/fab/pane_send_test.go` (serving both `cmd/fab` tests) — rather than inlining the dir-selection/length-guard logic at each of the three call sites or exporting a shared cross-package helper.
**Why**: The guard logic is ~20 lines with a fallback branch; triplicating it inline invites drift, and `cmd/fab` has two consumers in the same package. The intake explicitly permits per-package helper extraction and explicitly rules out requiring a shared cross-package helper (which would need production/exported code for a test-only concern).
**Rejected**: Inlining thrice (drift-prone, duplicates the budget constant); a shared exported helper package (over-engineering a test-only concern across a package boundary).
*Introduced by*: 260721-0j0t-fix-tmux-test-socket-leak

## Tasks

### Phase 2: Core Implementation

- [x] T001 [P] In `src/go/fab/internal/pane/pane_test.go`: add the unexported `tmuxSocketDir` helper (named `sun_path` budget constant, `tmuxSocketPathLen`, `t.TempDir()` candidate with `os.MkdirTemp("/tmp", "fabtest-")` fallback registered via `t.Cleanup`), then wire `TestReadAgentStateOption_Integration`: `t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))` before the first tmux call, server name shortened to `fabtest`, kill-server cleanup kept. Add `path/filepath` import. <!-- R1 R2 R3 R4 -->
- [x] T002 [P] In `src/go/fab/cmd/fab/pane_send_test.go`: add the same per-package helper trio (serving both `cmd/fab` integration tests), then wire `TestPaneSendGate_Integration`: `t.Setenv("TMUX_TMPDIR", ...)` before the first tmux call, server name shortened to `fabtest-send`, kill-server cleanup kept. Add `os` + `path/filepath` imports. <!-- R1 R2 R3 R4 -->
- [x] T003 In `src/go/fab/cmd/fab/panemap_test.go`: wire `TestMapSendAgentAgreement_Integration` using the helper from T002: `t.Setenv("TMUX_TMPDIR", ...)` before the first tmux call, server name shortened to `fabtest-agree`, kill-server cleanup kept. <!-- R1 R2 R4 -->

### Phase 3: Integration & Edge Cases

- [x] T004 In `src/go/fab/cmd/fab/pane_send_test.go`: add a unit test for the length guard (`TestTmuxSocketDirLengthGuard`) asserting that (a) an over-budget candidate (long server name forcing `t.TempDir()` past the budget) falls back to a dir whose socket path fits the budget, and (b) the returned dir exists in both the fit and fallback cases. <!-- R3 -->
- [x] T005 Verify end-to-end from `src/go/fab`: snapshot `/tmp/tmux-$UID`, run `go test ./internal/pane/ ./cmd/fab/ -run '_Integration' -v` confirming the three integration tests PASS (not SKIP), re-snapshot `/tmp/tmux-$UID` confirming zero new socket files, then run the full `go test ./internal/pane/ ./cmd/fab/` for the existing unit tests. <!-- R5 -->

## Execution Order

- T001 and T002 are independent ([P] — different packages)
- T003 depends on T002 (uses the `cmd/fab` helper)
- T004 depends on T002 (tests the `cmd/fab` helper)
- T005 runs last (verifies everything)

## Acceptance

### Functional Completeness

- [x] A-001 R1: All three integration tests call `t.Setenv("TMUX_TMPDIR", <private dir>)` after the `exec.LookPath("tmux")` guard and before the first tmux invocation; the env is process-level so the code under test resolves the same private socket dir
- [x] A-002 R2: Server names are the short fixed distinct strings `fabtest` / `fabtest-send` / `fabtest-agree`; no `UnixNano`-based server name remains in the three files
- [x] A-003 R3: The length guard exists with a named budget constant (103); an over-budget candidate falls back to a short `os.MkdirTemp("/tmp", "fabtest-")` dir removed via `t.Cleanup`; no path-length condition can reach `t.Skipf`
- [x] A-004 R4: The `t.Cleanup(kill-server)` registration is retained in all three tests

### Behavioral Correctness

- [x] A-005 R1: A full run of the three integration tests creates zero new files in `/tmp/tmux-$UID` (before/after snapshot identical) — verified: 6 sockets before, 6 after, zero new (`comm -13` empty)

### Scenario Coverage

- [x] A-006 R3: A unit test deterministically exercises the fallback branch (over-budget candidate → returned dir's socket path fits the budget) and the returned-dir-exists contract — `TestTmuxSocketDirLengthGuard` PASS (both subtests)
- [x] A-007 R5: On this tmux-equipped machine the three integration tests PASS (verbose output shows PASS, not SKIP) — proving the macOS `sun_path` edge case is handled, since the naive temp dir here exceeds the budget

### Edge Cases & Error Handling

- [x] A-008 R3: On macOS with a long `$TMPDIR` base, the longest-named test triggers the fallback rather than a tmux bind failure that would degrade into `t.Skipf` — proven by A-007 (integration tests PASS on this macOS machine where `t.TempDir()` exceeds the budget)

### Code Quality

- [x] A-009 Pattern consistency: New helper code follows the naming and structural patterns of the surrounding test code (unexported helpers, `t.Helper()`, table-less small tests)
- [x] A-010 No unnecessary duplication: The guard logic appears once per package (shared within `cmd/fab`); no third inline copy
- [x] A-011 No magic numbers: The `sun_path` budget is a named constant with a comment explaining the 104-byte macOS cap

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change removes the nano-timestamp server-name mechanism inline (`"fabtest-" + strconv.FormatInt(time.Now().UnixNano(), 36)` and variants) as part of the diff; nothing is left dangling. The `strconv` and `time` imports remain in use by other tests in all three files, so no import becomes dead. No existing helper, file, branch, or config was made redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Extract the per-package helper (intake marked extraction optional) rather than inlining at all three call sites | Two consumers in `cmd/fab` plus a ~20-line guard with a fallback branch make inline triplication drift-prone; intake explicitly permits per-package helpers | S:70 R:90 A:85 D:65 |
| 2 | Certain | Budget constant is 103 bytes for the full socket path (`macOS sun_path` 104 incl. terminating NUL); conservative on Linux (108) | Intake names "~103 bytes" as the conservative budget; measured platform data confirms | S:80 R:90 A:90 D:85 |
| 3 | Confident | Server names are exactly `fabtest` / `fabtest-send` / `fabtest-agree` | Intake gives these as "e.g." examples; they are the obvious short-fixed-distinct choice preserving the current prefixes | S:75 R:95 A:90 D:80 |
| 4 | Confident | `t.Setenv` placed immediately after the `server := ...` assignment (the length guard needs the name), still before the first tmux command | Intake's binding constraint is "before the first `tmux(...)` call"; the "before `server := ...`" phrasing is incidental ordering the length guard makes impossible to honor literally | S:65 R:90 A:85 D:70 |
| 5 | Confident | The length-guard unit test lives in `cmd/fab` only; the identical `internal/pane` copy is covered by the same logic plus its integration run | Duplicating an identical unit test across packages adds no coverage; the integration tests exercise both helpers for real (A-007) | S:60 R:95 A:85 D:65 |

5 assumptions (1 certain, 4 confident, 0 tentative).
