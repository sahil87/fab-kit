# Plan: Scanner-Truncation Sweep + Score Truth-Telling

**Change**: 260612-hv7t-scanner-truncation-sweep-score-truth-telling
**Intake**: `intake.md`

## Requirements

### Shared Helpers: read-lines + atomic-write (`internal/lines`, `internal/atomicfile`)

#### R1: CRLF-preserving read-lines helper
The fab Go module SHALL provide a new `internal/lines` package exposing `ReadFileLines(path string) ([]string, error)` (full read via `os.ReadFile`, split on `"\n"`) and `Split(content string) []string` (same semantics for in-memory content). Each returned line MUST be `TrimSuffix`'d of a trailing `"\r"` so `bufio.ScanLines`' CRLF handling is preserved. Reads are all-or-nothing: a partial result is impossible, and read failures surface as errors.

- **GIVEN** a file whose lines end in `\r\n`
- **WHEN** `ReadFileLines` reads it
- **THEN** returned lines carry no trailing `\r` (byte-compatible with the former scanner sites)

- **GIVEN** a file containing a single line longer than bufio's 64KB `MaxScanTokenSize`
- **WHEN** `ReadFileLines` reads it
- **THEN** every line — including lines after the over-long one — is returned, with no error and no truncation

- **GIVEN** an unreadable or missing file
- **WHEN** `ReadFileLines` is called
- **THEN** it returns a non-nil error (never a silent empty slice)

#### R2: Shared atomic-write helper, extracted (no third copy)
The fab Go module SHALL provide a new `internal/atomicfile` package exposing `WriteFile(path string, data []byte, perm os.FileMode) error` implementing the temp-file + rename pattern (temp in destination dir, write, sync, close, chmod to `perm`, rename; temp removed on any failure). `statusfile.Save` and `runtime.SaveFile` MUST delegate to it — the helper is *extracted from* the two existing precedents, not added as a third duplicate.

- **GIVEN** an existing target file and a failed write (e.g., read-only directory)
- **WHEN** `atomicfile.WriteFile` fails
- **THEN** the original file is untouched and no temp file is left behind

- **GIVEN** the refactor is complete
- **WHEN** searching the module for the temp+rename idiom
- **THEN** exactly one implementation exists (`internal/atomicfile`), with `statusfile.Save` and `runtime.SaveFile` as thin delegators

### Score Truth-Telling (`internal/score`, `cmd/fab/score.go`)

#### R3: `countGrades` returns errors instead of partial counts (F09)
`countGrades` SHALL change signature to `countGrades(file string) (GradeCount, error)`, migrate to the read-lines helper, and propagate open/read errors. `CheckGate` and `Compute` MUST surface the error instead of scoring from a partial or unreadable Assumptions table — a read failure is distinguishable from an intake with no Assumptions table (zero `GradeCount`, nil error).

- **GIVEN** an intake.md whose Assumptions table contains a >64KB line between graded rows (including an Unresolved row after it)
- **WHEN** `fab score --check-gate` runs
- **THEN** every row is counted (the truncation-driven fail→pass gate flip is impossible)

- **GIVEN** an intake.md that exists (passes the `os.Stat` guard) but cannot be read
- **WHEN** `CheckGate` or `Compute` runs
- **THEN** it returns a non-nil error rather than reporting score 0.0 / zero counts

#### R4: `Compute` surfaces `.status.yaml` load failures (F11)
`score.Compute` SHALL return the `.status.yaml` load error instead of silently skipping the write-back block and defaulting `change_type` to `feat`. The documented contract ("compute, write `.status.yaml`") makes silent non-persistence a lie. `CheckGate`'s lenient change-type read (its `os.Stat` guard plus default-on-unparseable) is unchanged — only `countGrades` errors newly propagate there.

- **GIVEN** a change directory whose `.status.yaml` is missing or malformed
- **WHEN** `fab score <change>` runs
- **THEN** the command exits non-zero with the load error on stderr, and no score YAML is printed as success

#### R5: `Compute` propagates persistence failures (F11, k4ge exit scheme)
`Compute` SHALL propagate errors from `status.SetConfidence` / `status.SetConfidenceFuzzy` and `log.ConfidenceLog` instead of discarding them with `_ =`. Failure routing conforms to the k4ge exit-code scheme: success YAML on stdout; errors returned from cobra `RunE` reach stderr as `ERROR: ...` with non-zero exit. The hook caller (`cmd/fab/hook.go` `if err == nil` guard) stays best-effort with **zero hook changes**. No CLI signature changes — flags and success-output stay identical.

- **GIVEN** a `.status.yaml` that loads but cannot be re-saved (e.g., read-only change directory)
- **WHEN** `fab score <change>` runs
- **THEN** it exits non-zero with the persistence error on stderr instead of printing a score and persisting nothing

- **GIVEN** the artifact-write hook fires `score.Compute` and persistence fails
- **WHEN** the hook's `err == nil` guard evaluates
- **THEN** the hook silently skips the score context line (best-effort preserved, no hook code change)

### Archive Index Integrity (`internal/archive`, `cmd/fab/archive.go`)

#### R6: `removeFromIndex` safe read-modify-write with honest status (F10)
`removeFromIndex` SHALL migrate to the read-lines helper (rewrite always derived from the complete file), change signature to `(string, error)`, and write via the atomic helper. `Restore` MUST surface the failure: `RestoreResult.Index` reports `failed` and the error propagates (result returned alongside, mirroring the `ArchiveWithBacklog` partial-success pattern); the restore CLI prints the YAML report before surfacing the error.

- **GIVEN** an archive index containing a >64KB line and entries after it
- **WHEN** a change is restored (entry removed)
- **THEN** all other entries — including those after the long line — survive the rewrite

- **GIVEN** an index that cannot be read or rewritten
- **WHEN** `Restore` runs
- **THEN** the move still completes, `index: failed` appears in the YAML, and the command exits non-zero with the cause on stderr

#### R7: `updateIndex` stops lying about index writes (F15)
`updateIndex` SHALL check its ReadFile/WriteFile errors and return `(status, content, error)` instead of an unconditional `"updated"`/`"created"`. On index failure, `Archive` reports `index: failed` in its YAML result and returns the error alongside the (non-nil) result — the move has already happened. `ArchiveWithBacklog` MUST pass partial results through (still attempting the backlog mark once the move succeeded) and join both errors.

- **GIVEN** an index file that cannot be written (e.g., `index.md` is a directory)
- **WHEN** `Archive` runs
- **THEN** the YAML reports `move: moved` and `index: failed`, and the command exits non-zero

#### R8: Atomic index writes; `backfillIndex` reads once (F15)
All archive-index writes (`updateIndex`, `backfillIndex`, `removeFromIndex`) SHALL go through `atomicfile.WriteFile` (no in-place `os.WriteFile` truncation). `backfillIndex` SHALL take the just-written index content as a parameter instead of re-reading the file post-write, and append missing entries via one atomic rewrite.

- **GIVEN** an archive operation interrupted mid-write
- **WHEN** the index write is examined
- **THEN** the index contains either the old or the new content, never a truncated intermediate

### Backlog Parsing (`internal/backlog`, `cmd/fab/batch_new.go`)

#### R9: Backlog parse failures surface (F12)
`ParsePending` SHALL change signature to `([]Item, error)`, migrate to the read-lines helper, and stop swallowing open errors. `ExtractContent` SHALL migrate too, keeping `(string, error)` with errors made accurate: read failures return the real error; a genuinely missing ID returns `not found in backlog`. Callers in `batch_new.go` propagate `ParsePending` errors (`--list` and `--all` paths) and keep the per-ID warn-and-skip behavior, now printing the actual error.

- **GIVEN** a backlog with a >64KB line followed by pending items
- **WHEN** `fab batch new --list` runs
- **THEN** items below the long line are listed (nothing silently vanishes)

- **GIVEN** an ID positioned after a >64KB line
- **WHEN** `fab batch new <id>` extracts it
- **THEN** the content is found (no misleading `not found` error)

### Batch Archive: statusfile ownership (`cmd/fab/batch_archive.go`)

#### R10: `isArchivable` uses statusfile, not regex (F14)
`hydrateStatusRe` SHALL be deleted and `isArchivable` rewritten on `statusfile.Load` + `GetProgress("hydrate")` returning true only for `done`/`skipped`. The test fixtures that pin the regex's fragment-tolerant behavior (bare `"  hydrate: done\n"` without a `progress:` parent) MUST be rewritten as realistic `progress:` blocks (constitution VII — tests conform to spec).

- **GIVEN** a `.status.yaml` with `progress.hydrate: done` (or `skipped`)
- **WHEN** `isArchivable` evaluates it
- **THEN** it returns true

- **GIVEN** a `.status.yaml` with a `hydrate: done` key outside the `progress:` block (e.g., under `stage_metrics:`), or an unparseable file
- **WHEN** `isArchivable` evaluates it
- **THEN** it returns false (the regex's any-indentation match is gone; statusfile semantics hold everywhere)

### Mechanical Sweep (`internal/{hooklib,prmeta,frontmatter,memoryindex}`)

#### R11: Unchecked-scanner class eliminated (F13)
The remaining unchecked `bufio.Scanner` sites SHALL migrate to the helper, preserving each function's public signature and empty-on-error contract (the truncation class disappears because reads are all-or-nothing): `hooklib.HasSectionHeading` / `scanSectionItems` use `lines.Split` (in-memory); `prmeta.countCheckboxesInTasksSection` / `countCheckboxes`, `frontmatter.Field` / `HasFrontmatter`, and `memoryindex.readH1` use `lines.ReadFileLines`. `proc_linux.go`'s scanner is left as-is (the one correct site — checks `Err()`, streams /proc). After the sweep, `bufio.NewScanner` MUST NOT appear in fab-module production code outside `internal/proc/proc_linux.go`.

- **GIVEN** a plan.md whose `## Tasks` section contains a >64KB line followed by checkbox items
- **WHEN** the artifact-write hook counts section items
- **THEN** all items are counted and the persisted `.status.yaml` counters are correct

- **GIVEN** the sweep is complete
- **WHEN** grepping `src/go/fab` non-test code for `bufio.NewScanner`
- **THEN** the only hit is `internal/proc/proc_linux.go`

### Docs Conformance

#### R12: `_cli-fab.md` documents fab-score failure surfacing
`src/kit/skills/_cli-fab.md` § fab score SHALL document the new failure surfacing: normal mode exits non-zero with stderr detail on `.status.yaml` load failure, confidence-persist failure, or intake read failure (extending the k4ge gate-fail exit row). No command-signature rows change.

- **GIVEN** the updated `_cli-fab.md`
- **WHEN** reading the fab score section
- **THEN** the load/persist/read failure exit behavior is stated, and flags/YAML success output are documented unchanged

### Non-Goals

- The `gateThresholds`/`expectedMin` data maps in `internal/score/score.go` — owned by the sibling change ye8r [x8c9]; this change touches only scan/error logic
- Cross-process locking and the `.status.yaml` "status file not found" error-wording fix (F06) — batch B1 scope
- The fab-kit Go module (its `sync.go` already uses the safe idiom)
- `cmd/fab/hook.go` — already error-guarded; deliberately untouched
- `backlog.MarkDone` — already uses the safe ReadFile+Split idiom (the in-repo precedent); not migrated

### Design Decisions

1. **Read-fully over enlarged scanner buffers**: every scanned input is a small markdown/YAML file already loaded in full elsewhere — `os.ReadFile` + split removes the line-length failure class entirely instead of patching buffer sizes per site. — *Rejected*: `scanner.Buffer` + `Err()` checks (keeps a tunable failure mode and 12 per-site patches).
2. **Hard error over warn-and-exit-0 for fab score persistence** (intake Assumption #4): `_cli-fab.md` documents normal mode as "compute, write .status.yaml"; skill convention is non-zero → STOP; the hook's `err == nil` guard keeps the only best-effort path intact. — *Rejected*: stderr warning with exit 0 (still a lying success for scripted consumers).
3. **Extract the atomic-write helper; both precedents delegate**: `statusfile.Save` and `runtime.SaveFile` become thin wrappers over `atomicfile.WriteFile`, so archive-index writes are the helper's third *caller*, not a third *copy*. — *Rejected*: archive-local temp+rename (violates the duplicating-utilities anti-pattern).
4. **Partial-success result pattern for archive/restore index failures**: mirror the existing `ArchiveWithBacklog` contract (non-nil result + non-nil error after the irreversible move) so `archiveLoop`'s archived-with-warning path and the CLI's print-then-error path compose without new states.

## Tasks

### Phase 1: Setup — shared helpers

- [x] T001 Create `src/go/fab/internal/lines/lines.go` (`ReadFileLines`, `Split`, per-line `TrimSuffix "\r"`) + `lines_test.go` (CRLF, >64KB line, missing file, empty content, trailing newline semantics) <!-- R1 -->
- [x] T002 [P] Create `src/go/fab/internal/atomicfile/atomicfile.go` (`WriteFile(path, data, perm)` temp+sync+chmod+rename, cleanup on failure) + `atomicfile_test.go` (content/perm, overwrite, failure leaves original + no temp residue) <!-- R2 -->
- [x] T003 Refactor `src/go/fab/internal/statusfile/statusfile.go` `Save` and `src/go/fab/internal/runtime/runtime.go` `SaveFile` to delegate to `atomicfile.WriteFile`; existing package tests stay green <!-- R2 -->

### Phase 2: Core implementation — error propagation per finding

- [x] T004 `src/go/fab/internal/score/score.go`: `countGrades → (GradeCount, error)` via `lines.ReadFileLines`; `CheckGate`/`Compute` surface it; `Compute` returns `.status.yaml` load errors and propagates `SetConfidence`/`SetConfidenceFuzzy`/`ConfidenceLog` errors (drop the `_ =` discards). Tests in `score_test.go`: >64KB-line-in-table counting (gate-flip proof), load-failure error, persist-failure error <!-- R3 R4 R5 -->
- [x] T005 `src/go/fab/internal/archive/archive.go`: `removeFromIndex → (string, error)` via `lines.ReadFileLines` + `atomicfile.WriteFile`; `Restore` reports `index: failed` + returns result alongside error; `cmd/fab/archive.go` restore handler prints YAML on partial failure (mirror archive handler). Tests: oversized-line entry preservation, index-write failure surfaced <!-- R6 R8 -->
- [x] T006 `src/go/fab/internal/archive/archive.go`: `updateIndex → (status, content, error)` with checked ReadFile/atomic WriteFile; `backfillIndex(archiveDir, indexFile, indexContent string) error` (content passed in, atomic append-rewrite); `Archive` maps index errors to `index: failed` + partial result; `ArchiveWithBacklog` passes partial results through and joins errors. Tests: index failure honest YAML, backfill no re-read behavior preserved <!-- R7 R8 -->
- [x] T007 `src/go/fab/internal/backlog/backlog.go`: `ParsePending → ([]Item, error)`, `ExtractContent` migrated with accurate errors (`not found in backlog`); update callers in `src/go/fab/cmd/fab/batch_new.go` (`--all`, `--list` propagate; per-ID warn-and-skip prints the real error). Tests in `backlog_test.go` + `batch_new_test.go`: signature updates, >64KB-line item survival, missing-file error, read-error vs not-found distinction <!-- R9 -->
- [x] T008 `src/go/fab/cmd/fab/batch_archive.go`: delete `hydrateStatusRe`, rewrite `isArchivable` on `statusfile.Load` + `GetProgress("hydrate")`; rewrite fixtures at `batch_archive_test.go:62/67/85/97` as realistic `progress:` blocks; add non-progress `hydrate:` key and invalid-YAML not-archivable tests <!-- R10 -->

### Phase 3: Integration & sweep

- [x] T009 [P] `src/go/fab/internal/hooklib/artifact.go`: `HasSectionHeading` + `scanSectionItems` iterate `lines.Split(content)`; drop `bufio`. Test: >64KB line inside section, items after it still counted <!-- R11 -->
- [x] T010 [P] `src/go/fab/internal/prmeta/prmeta.go`: `countCheckboxesInTasksSection` + `countCheckboxes` via `lines.ReadFileLines` (keep `(done, total)` zero-on-error contract); drop `bufio` <!-- R11 -->
- [x] T011 [P] `src/go/fab/internal/frontmatter/frontmatter.go`: `Field` + `HasFrontmatter` via `lines.ReadFileLines`; drop `bufio`. Test: field after a >64KB frontmatter line is found <!-- R11 -->
- [x] T012 [P] `src/go/fab/internal/memoryindex/memoryindex.go`: `readH1` via `lines.ReadFileLines`; drop `bufio` <!-- R11 -->
- [x] T013 Class-elimination verification: `grep` confirms no production `bufio.NewScanner` outside `internal/proc/proc_linux.go`; run `go build ./...`, `go vet ./...`, `go test ./...` from `src/go/fab` — all green <!-- R11 -->

### Phase 4: Polish — docs conformance

- [x] T014 `src/kit/skills/_cli-fab.md` § fab score: document normal-mode non-zero exit on load/persist/read failure (extends the k4ge gate-fail exit row); no signature rows change <!-- R12 -->

## Execution Order

- T001 and T002 are independent [P]; T003 needs T002
- T004–T012 all need T001; T005/T006 also need T002 (atomic writes) and share `archive.go` (sequential)
- T013 needs T004–T012; T014 is independent of code tasks

## Acceptance

### Functional Completeness

- [x] A-001 R1: `internal/lines` exists with `ReadFileLines` + `Split`, trims trailing `\r` per line, returns errors on unreadable files, and handles >64KB lines without truncation
- [x] A-002 R2: `internal/atomicfile.WriteFile` exists; `statusfile.Save` and `runtime.SaveFile` delegate to it (no remaining inline temp+rename copies)
- [x] A-003 R3: `countGrades` has signature `(GradeCount, error)`; `CheckGate` and `Compute` return its error instead of scoring partial counts
- [x] A-004 R4: `Compute` returns `.status.yaml` load errors; no silent `feat` default + skipped write-back on load failure
- [x] A-005 R5: `Compute` propagates `SetConfidence`/`SetConfidenceFuzzy`/`ConfidenceLog` failures; `cmd/fab/score.go` RunE surfaces them (stderr + non-zero); `cmd/fab/hook.go` is unchanged
- [x] A-006 R6: `removeFromIndex` returns `(string, error)`, rewrites from the complete file via the helper, and `Restore` reports `index: failed` with the result returned alongside the error
- [x] A-007 R7: `updateIndex` checks read/write errors; `Archive` YAML reports honest index status; `ArchiveWithBacklog` passes partial results through
- [x] A-008 R8: all three index writers use `atomicfile.WriteFile`; `backfillIndex` takes content as a parameter and performs no post-write re-read
- [x] A-009 R9: `ParsePending` returns `([]Item, error)`; `ExtractContent` errors are accurate; `batch_new.go` propagates list/all errors and keeps per-ID warn-and-skip
- [x] A-010 R10: `hydrateStatusRe` is deleted; `isArchivable` uses `statusfile.Load` + `GetProgress`; fixtures write realistic `progress:` blocks
- [x] A-011 R11: hooklib/prmeta/frontmatter/memoryindex sites migrated; `proc_linux.go` untouched; no production `bufio.NewScanner` outside `internal/proc/`
- [x] A-012 R12: `_cli-fab.md` fab score section documents the failure-surfacing exit behavior

### Behavioral Correctness

- [x] A-013 R3: a >64KB line inside the Assumptions table no longer drops subsequent rows — gate verdict computed from the full table (truncation-driven fail→pass flip impossible)
- [x] A-014 R5: `fab score` success output (YAML on stdout) is byte-identical to before; only failure paths changed
- [x] A-015 R10: pipeline-produced `.status.yaml` files (block-style `progress:`) classify identically under statusfile as under the old regex

### Scenario Coverage

- [x] A-016 R6: test proves index entries beyond a removed entry (including after a >64KB line) survive restore
- [x] A-017 R9: test proves backlog items below a >64KB line appear in `ParsePending` and `ExtractContent` finds IDs after it
- [x] A-018 R11: test proves section-item counting is correct with a >64KB line inside the section

### Edge Cases & Error Handling

- [x] A-019 R6: missing index file on restore still yields `index: not_found` with nil error (no behavior change for the benign case)
- [x] A-020 R9: missing backlog.md yields an error from `ParsePending` (callers pre-check existence; `--list`/`--all` surface it)
- [x] A-021 R7: archive move succeeding + index write failing yields non-nil result (`move: moved`, `index: failed`) + non-nil error — `archiveLoop` counts it archived-with-warning

### Code Quality

- [x] A-022 Pattern consistency: new code follows surrounding naming, comment density, and error-message style (lowercase wrapped errors, existing fab strings)
- [x] A-023 No unnecessary duplication: one read-lines helper, one atomic-write helper; no per-site reimplementations (anti-pattern: duplicating existing utilities)
- [x] A-024 Readability over cleverness: helper functions stay small and focused; no god functions introduced
- [x] A-025 Test strategy (test-alongside): every behavior change ships with tests in the same package, per constitution "tests per Go change"

### Documentation Accuracy

- [x] A-026 R12: `_cli-fab.md` fab score rows match the implemented exit behavior exactly (no doc drift); no CLI signature changes documented because none were made

### Cross References

- [x] A-027: intake's cross-change seam respected — `gateThresholds`/`expectedMin` data maps untouched (`git diff` shows no hunks in score.go:344–356 region); `cmd/fab/hook.go` untouched

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `internal/backlog/backlog.go:117` (`MarkDone`'s inline `strings.Split(string(data), "\n")`) — the new `internal/lines` helper subsumes this hand-rolled ReadFile+Split idiom; deliberately left per the plan's Non-Goals, but it is now a redundant private copy (modulo CRLF trim)
- `internal/intake/intake.go:28` (inline `strings.Split(string(data), "\n")` line loop) — same idiom now centralized in `internal/lines.Split`; future cleanup once CRLF parity is confirmed acceptable there
- `internal/score/changetypes_doc_test.go:140` (test-local `bufio.NewScanner` parser + the :122 comment citing "the same bufio.Scanner idiom used by countGrades") — `countGrades` no longer uses a scanner, so the cited precedent is gone and `lines.ReadFileLines` can replace the test's scanner

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `atomicfile.WriteFile` adopts the fuller runtime.SaveFile variant (fsync + chmod `perm` before rename); `statusfile.Save` thereby gains fsync and a stable 0644 mode (previously inherited `os.CreateTemp`'s 0600) | Intake binds "extract, no third copy" but the two precedents differ; the stricter variant is crash-safer, and 0644 matches how every other fab artifact (and the pre-save template write) is created — unifying fixes a perms flip-flop rather than introducing one | S:70 R:85 A:80 D:65 |
| 2 | Confident | `ArchiveWithBacklog` still attempts the backlog mark when `Archive` returns a partial result (move done, index failed), combining both errors via `errors.Join` | The move is irreversible and a re-run soft-skips (`ErrAlreadyArchived`), so skipping the mark would strand the backlog item permanently; `errors.Is` detection through `Join` keeps the soft-skip path intact | S:65 R:80 A:80 D:60 |
| 3 | Confident | `ExtractContent`'s missing-ID error becomes `not found in backlog` and `batch_new.go`'s per-ID warning prints `Warning: [id] %v, skipping` — byte-identical output for the missing-ID case, honest for read errors | Verifier flagged the misleading `not found`; this preserves the documented warn-and-skip wording while making I/O failures truthful | S:70 R:90 A:85 D:70 |
| 4 | Confident | `Restore` mirrors the `ArchiveWithBacklog` partial-success pattern (non-nil result + error) and `changeRestoreCmd` prints the YAML before returning the error, mirroring `changeArchiveCmd` | Intake Assumption #6 requires "Restore YAML report honest index status"; the in-file archive CLI handler already established the print-then-error idiom | S:70 R:85 A:85 D:70 |
| 5 | Certain | `updateIndex` returns the rewritten content for `backfillIndex` to consume (3-value return) rather than `backfillIndex` re-reading the file | Intake §5 states "reads the index once per operation (content passed in)" verbatim; passing the just-written content is the only re-read-free shape | S:85 R:90 A:90 D:85 |
| 6 | Confident | Sweep sites with empty-on-error contracts (`prmeta` counts, `frontmatter.Field`/`HasFrontmatter`, `memoryindex.readH1`) keep their signatures — only the intake-mandated signature changes (`countGrades`, `ParsePending`, `removeFromIndex`, `updateIndex`) gain error returns | Intake §8 table lists their treatment as plain "helper" with no error-return note; `os.ReadFile` is all-or-nothing so the silent-truncation class is gone even with the preserved contracts | S:75 R:85 A:85 D:75 |

6 assumptions (1 certain, 5 confident, 0 tentative).
