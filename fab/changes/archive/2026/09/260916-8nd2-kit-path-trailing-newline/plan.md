# Plan: `fab kit-path` prints a newline-terminated line

**Change**: 260916-8nd2-kit-path-trailing-newline
**Intake**: `intake.md`

## Requirements

### Go Binary: `fab kit-path` output contract

#### R1: Output is one newline-terminated line
`fab kit-path` MUST print the absolute resolved kit directory followed by exactly one `\n` and
nothing else on stdout. Exit codes, stderr wording, and `FAB_KIT_PATH` resolution semantics
(`internal/kitpath.KitDir()`) MUST be unchanged.

- **GIVEN** a resolvable kit directory (exe-sibling `kit/` or a valid `FAB_KIT_PATH`)
- **WHEN** `fab kit-path` runs
- **THEN** stdout is `{dir}\n` — the last byte is `0a`
- **AND** `fab kit-path && echo NEXT` shows `NEXT` on its own line

- **GIVEN** `FAB_KIT_PATH` names a missing or non-directory path
- **WHEN** `fab kit-path` runs
- **THEN** the existing error (`cannot resolve kit path: …`) and non-zero exit are produced unchanged

#### R2: The success-path bytes are covered by a package test
A test in `src/go/fab/cmd/fab/` MUST execute `kitPathCmd()` against a temp `FAB_KIT_PATH` and assert
the captured stdout equals the directory plus exactly one `\n` (exact byte compare, so a
double-newline regression also fails). Constitution Additional Constraints: Go CLI changes ship tests.

- **GIVEN** `FAB_KIT_PATH` set via `t.Setenv` to `t.TempDir()`
- **WHEN** `kitPathCmd()` executes with `SetOut(&buf)`
- **THEN** `buf.String() == dir + "\n"`

### Kit Content: CLI reference

#### R3: `_cli-fab.md` § `fab kit-path` states the new contract
The sentence "No trailing newline or decoration." in `src/kit/skills/_cli-fab.md` § `fab kit-path`
MUST be replaced with wording that states the output is a single newline-terminated line (the path
plus exactly one `\n`) with no other decoration. The rest of the paragraph MUST be kept verbatim.

- **GIVEN** the updated `_cli-fab.md`
- **WHEN** grepping the file for `No trailing newline`
- **THEN** there are zero hits, and § `fab kit-path` describes a newline-terminated single line

#### R4: No other restatement of the old contract survives
No file under `docs/memory/**`, `docs/specs/**`, or `src/kit/**` (excluding `**/archive/**`) MAY
restate that `fab kit-path` prints without a trailing newline. Unrelated "trailing newline" hits
(the `fab/.fab-version` migration, `json.Encoder` note, `kit-architecture.md:185`) MUST be left alone.

- **GIVEN** the repo after apply
- **WHEN** grepping for kit-path newline claims per intake § 4
- **THEN** the only kit-path hit is the rewritten `_cli-fab.md` sentence

### Verification

#### R5: Scoped tests and gofmt pass
`go test ./...` run from `src/go/fab/` scoped to `./cmd/fab/...` MUST pass, and `gofmt -l` on the two
touched Go files MUST print nothing.

- **GIVEN** the applied change
- **WHEN** `cd src/go/fab && go test ./cmd/fab/...` and `gofmt -l cmd/fab/kitpath.go cmd/fab/kitpath_test.go` run
- **THEN** tests pass and gofmt prints nothing

### Non-Goals

- `src/kit/skills/fab-setup.md` Pre-flight Check wording — unchanged; kit and binary are version-pinned together, so a fixed skill never runs against an unfixed binary
- No migration file — no user data restructured
- No change to `internal/kitpath` or to any other `fmt.Fprint` site in `src/go/fab/cmd/fab/`

### Design Decisions

#### `fab kit-path` Prints Exactly One Trailing Newline
**Decision**: `fab kit-path` writes the resolved directory plus exactly one `\n` (`fmt.Fprintln`), matching every other scalar-printing `fab` command.
**Why**: POSIX `$(...)` substitution strips trailing newlines, so the former no-newline contract protected nothing, while chained invocations (`fab kit-path && cat "$(fab kit-path)/VERSION"` in `/fab-setup`'s pre-flight) fused the path with the next command's output and misled agents into a bogus `…/kit2.28.1` path on every `/fab-setup migrations` run.
**Rejected**: keeping the no-newline contract and rewording `fab-setup.md` (patches one caller of N and leaves the binary's sole scalar-output outlier in place); `fmt.Fprintf("%s\n", dir)` (equivalent, gains nothing).
*Introduced by*: 260916-8nd2-kit-path-trailing-newline

## Tasks

### Phase 2: Core Implementation

- [x] T001 Change `fmt.Fprint(cmd.OutOrStdout(), dir)` to `fmt.Fprintln(cmd.OutOrStdout(), dir)` in `src/go/fab/cmd/fab/kitpath.go`; nothing else in the file changes <!-- R1 -->
- [x] T002 Add `src/go/fab/cmd/fab/kitpath_test.go` (`package main`): `TestKitPathCmd_PrintsNewlineTerminatedLine` — `t.Setenv(kitpath.KitPathEnv, t.TempDir())`, `kitPathCmd()`, `SetOut(&buf)`, `SetArgs([]string{})`, `Execute()`, exact compare `buf.String() == dir+"\n"` <!-- R2 -->

### Phase 4: Polish

- [x] T003 [P] In `src/kit/skills/_cli-fab.md` § `fab kit-path`, replace "No trailing newline or decoration." with "Output is a single newline-terminated line — the path followed by exactly one `\n` — with no other decoration."; keep the rest of the paragraph verbatim <!-- R3 -->
- [x] T004 [P] Doc sweep: grep `docs/memory/**`, `docs/specs/**`, `src/kit/**` (excluding archives) for kit-path newline claims (`trailing newline`, `no newline`, `without a newline`, `no trailing`, and `kit-path` lines mentioning `newline`/`decoration`/`bare`/`raw`); confirm the only kit-path hit is T003's sentence and leave the unrelated hits listed in intake § 4 untouched <!-- R4 -->
- [x] T005 Run `cd src/go/fab && go test ./cmd/fab/...` and `gofmt -l cmd/fab/kitpath.go cmd/fab/kitpath_test.go`; then build a dev binary (`go build -o <scratch>/fab-go-dev ./cmd/fab`) and confirm `FAB_KIT_PATH=$PWD/../../kit <scratch>/fab-go-dev kit-path | xxd | tail -1` ends in `0a` and `… kit-path && echo NEXT` shows `NEXT` on its own line <!-- R5 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `kitpath.go` uses `fmt.Fprintln` and the command's `Use`, `Args`, `KitDir()` call, and error wrapping are unchanged
- [x] A-002 R2: `kitpath_test.go` exists in `src/go/fab/cmd/fab/`, targets `kitPathCmd()` via `FAB_KIT_PATH`, and asserts `dir + "\n"` exactly
- [x] A-003 R3: `_cli-fab.md` § `fab kit-path` contains the newline-terminated-line sentence and no longer contains "No trailing newline or decoration."
- [x] A-004 R4: no other file under `docs/memory/**`, `docs/specs/**`, `src/kit/**` (non-archive) restates a no-newline contract for `fab kit-path`
- [x] A-005 R5: `go test ./cmd/fab/...` passes and `gofmt -l` on the two Go files prints nothing

### Behavioral Correctness

- [x] A-006 R1: a dev-built `fab-go kit-path` piped through `xxd` ends in `0a`, and `kit-path && echo NEXT` prints `NEXT` on its own line
- [x] A-007 R1: the error path (`FAB_KIT_PATH` pointing at a missing path) still exits non-zero with the `cannot resolve kit path:` message (existing `internal/kitpath` tests still pass)

### Scenario Coverage

- [x] A-008 R2: the new test fails against the pre-change `fmt.Fprint` (verified by reasoning or a quick revert: the expected string carries `\n`, the old output does not)

### Edge Cases & Error Handling

- [x] A-009 R1: exactly one newline — no double newline (the exact byte compare in the test guards this)

### Code Quality

- [x] A-010 Pattern consistency: the test follows the existing `src/go/fab/cmd/fab/*_test.go` cobra constructor + `SetArgs`/`Execute` style and the `t.Setenv("FAB_KIT_PATH", …)` precedent in `setup_test.go`
- [x] A-011 No unnecessary duplication: no new helper introduced; `kitpath.KitPathEnv` constant reused rather than a string literal
- [x] A-012 CLI ⇒ docs + tests: the `fab` CLI behavior change ships with a test and an `_cli-fab.md` update (Constitution Additional Constraints / code-quality anti-pattern)
- [x] A-013 Canonical source only: the skill edit is in `src/kit/skills/`, not in `.agents/skills/` or `.claude/skills/`
- [x] A-014 No fab-kit-only path citations added to deployed content (`_cli-fab.md` edit cites nothing new)
- [x] A-015 Sibling sweep done up front (T004), not reactively

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change replaces the `fmt.Fprint` call in place and adds a test; no existing code (files, functions, branches, config) was made redundant or unused.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Test uses `t.Setenv` + `t.TempDir()` rather than `kitpath.SetOverride` | Matches the `FAB_KIT_PATH` precedent already in `setup_test.go`; no `t.Parallel` in the package so `t.Setenv` is safe; exercises the real env path instead of the test-only override | S:90 R:95 A:90 D:90 |
| 2 | Certain | Tasks grouped as Phase 2 (code) + Phase 4 (docs/verification); no Phase 1/3 | Nothing to scaffold or wire; five tasks total → light lane | S:90 R:95 A:95 D:90 |
| 3 | Certain | Verification runs from `src/go/fab/` (its own `go.mod`) with `./cmd/fab/...` scope | The repo has two Go modules (`src/go/fab`, `src/go/fab-kit`); only the fab module is touched | S:90 R:95 A:95 D:90 |

3 assumptions (3 certain, 0 confident, 0 tentative).
