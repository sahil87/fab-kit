# Plan: Binary Truth-Telling, Error-Surfacing & Inference Correctness

**Change**: 260615-jznd-binary-truth-telling
**Intake**: `intake.md`

## Requirements

### (a) change_type inference: regex tightening

#### R1: The `fix` keyword regex MUST NOT match `fix`/`bug` inside a hyphenated compound that signals non-fix intent
The change-type inference (`hooklib.InferChangeType`) SHALL classify an intake as `fix` for standalone fix-signalling tokens and for fix-describing compounds (`bug-fix`, `hot-fix`, `bug-free`), but SHALL NOT classify a feature intake as `fix` solely because it mentions `must-fix`/`must fix` in passing.

- **GIVEN** an intake whose only fix-adjacent token is `must-fix` (or `must fix`) in otherwise feature prose
- **WHEN** `InferChangeType` runs
- **THEN** the inferred type is NOT `fix` (falls through to a later pattern or `feat`)
- **AND GIVEN** an intake mentioning `bug-fix`, `hot-fix`, `bug-free`, or a standalone `fix`/`bug`/`broken`/`regression`
- **WHEN** `InferChangeType` runs
- **THEN** the inferred type IS `fix`

### (2/a) change_type inference: infer-once / respect-explicit-set guard

#### R2: `.status.yaml` MUST carry a `change_type_source` enum recording how the change_type was set
The status file schema SHALL gain an enum field `change_type_source` with values `inferred` or `explicit`. An absent/empty field SHALL be treated as `inferred` (back-compat). The field SHALL round-trip through `statusfile.Load`/`Save` and be inserted on write when absent from a sparse document.

- **GIVEN** a `.status.yaml` with no `change_type_source` key
- **WHEN** it is loaded
- **THEN** the in-memory value is the zero/absent equivalent of `inferred`
- **AND WHEN** a command mutates and saves it
- **THEN** `change_type_source` is serialized with the current value

#### R3: `fab status set-change-type` MUST mark the source `explicit`
When a human runs `set-change-type`, the binary SHALL set `change_type_source: explicit` alongside the type value.

- **GIVEN** any change
- **WHEN** `fab status set-change-type <change> <type>` runs
- **THEN** both `change_type` and `change_type_source: explicit` are persisted

#### R4: The PostToolUse intake-write hook MUST NOT overwrite an explicitly-set change_type
The artifact-write hook (`artifactBookkeeping`) SHALL apply inference and overwrite `change_type` ONLY when `change_type_source` is absent or `inferred`. When it is `explicit`, the hook SHALL skip both inference and the type overwrite (other bookkeeping such as scoring still runs).

- **GIVEN** a change with `change_type_source: explicit` and `change_type: feat`
- **WHEN** an intake.md write fires the hook and inference would yield `fix`
- **THEN** `change_type` stays `feat` and `change_type_source` stays `explicit`
- **AND GIVEN** a change with absent/`inferred` source
- **WHEN** the hook fires
- **THEN** inference runs and overwrites `change_type` (source stays `inferred`)

### (b) acceptance truth: derive from checkboxes on read

#### R5: Acceptance progress MUST be derived live from `plan.md` checkboxes at read time
A shared helper in `internal/status` (`LiveAcceptance(changeDir) (done, total int)`) SHALL count `## Acceptance` checkboxes in `{changeDir}/plan.md` via the existing `hooklib.CountSectionItemsBounded`/`CountCompletedSectionItemsBounded`. Read sites (`preflight`, `prmeta`, `cmd/fab/status.go status plan`) SHALL prefer the live count over the persisted `.status.yaml` counter when `plan.md` exists and has an `## Acceptance` section; the persisted counter remains a write-time cache. `fab score` is out of scope (reads `intake.md` only).

- **GIVEN** a `plan.md` whose `## Acceptance` checkboxes were toggled by a hook-bypassing edit (e.g. sed), leaving the `.status.yaml` counter stale
- **WHEN** preflight / prmeta / `status plan` report acceptance progress
- **THEN** the reported `done`/`total` match the checkboxes on disk, not the stale counter
- **AND GIVEN** a change with no `plan.md` (or no `## Acceptance` heading)
- **WHEN** a reader runs
- **THEN** it falls back to the persisted counter without error

### (c) F21-residue: surface swallowed WriteFile errors + fix lying comment

#### R6: Scaffold write failures in `lineEnsureMerge` MUST be propagated, and the `scaffoldDirectories` doc comment MUST be accurate
`lineEnsureMerge` SHALL return any `os.WriteFile` error (the symlink-resolve rewrite at ~L273 and the create-with-entry at ~L294) up the `scaffoldTreeWalk` chain instead of discarding it. The `scaffoldDirectories` doc comment's "Write failures are propagated" claim SHALL be made true for what it documents (it must not misdescribe `lineEnsureMerge`).

- **GIVEN** a scaffold fragment merge where `os.WriteFile` fails (e.g. read-only dest)
- **WHEN** `lineEnsureMerge` runs
- **THEN** it returns a non-nil error that `scaffoldTreeWalk` propagates
- **AND** the `scaffoldDirectories` comment accurately describes error propagation

### (d) resolve typed errors

#### R7: `internal/resolve` MUST expose `ErrNotFound`/`ErrAmbiguous` sentinels and archive soft-skip MUST branch on them
`resolve` SHALL declare `ErrNotFound` and `ErrAmbiguous` and wrap its "no change matches", "no active changes/change", and "multiple changes match" / "multiple changes exist" messages with `%w` so callers can `errors.Is`. The archive soft-skip callers (`internal/archive/archive.go` `Archive`, `cmd/fab/batch_archive.go` `runBatchArchive`) SHALL treat only `ErrNotFound` as the "maybe already archived" soft-skip path and SHALL surface `ErrAmbiguous` as a real error instead of conflating it with not-found.

- **GIVEN** an ambiguous change name passed to batch archive
- **WHEN** resolution returns `ErrAmbiguous`
- **THEN** the name is NOT soft-skipped as already-archived; the ambiguity surfaces (warning/error), not a silent skip
- **AND GIVEN** a not-found name that exists in the archive
- **WHEN** resolution returns `ErrNotFound`
- **THEN** the existing idempotent already-archived soft-skip applies

### (e) prmeta clampNonNeg: annotate when clamped

#### R8: PR-meta impl Impact MUST annotate the true value when the non-negative clamp engages
`prmeta.renderImpact` SHALL keep the non-negative clamp on the displayed impl `Added`/`Deleted`/`Net`, but when clamping actually changes a value (the pre-clamp value was negative) it SHALL surface the true pre-clamp value in the output (e.g. `net +0 (clamped from −42)`). This stops PR-meta from silently hiding net-deletion-in-production PRs.

- **GIVEN** a test-heavy diff where `total.Net - tests.Net` is negative
- **WHEN** `renderImpact` renders the impl row
- **THEN** the displayed net is `+0` AND the line annotates the true negative pre-clamp value
- **AND GIVEN** a non-negative impl net
- **WHEN** `renderImpact` renders
- **THEN** no clamp annotation appears (unchanged output)

### Design Decisions

1. **`change_type_source` as enum, not bool**: matches the intake's resolved design — leaves room for a future `linear`/imported source without a schema migration. *Rejected*: a `change_type_explicit` bool (less expressive).
2. **`LiveAcceptance` returns (done, total)** and lives in `internal/status` (not `hooklib`): readers already import `status`; `hooklib` owns the low-level counters that `LiveAcceptance` composes. Readers fall back to the persisted cache when `plan.md`/`## Acceptance` is absent, preserving current behavior for intake-only or pre-plan changes.
3. **(e) annotation, not signed reporting**: the binary-review Refuted section (R1-R8) does NOT adjudicate the clamp at all (it covers unrelated findings), so the intake's resolved "annotate when clamped" design stands unchanged — the clamp is kept (possibly load-bearing for downstream non-negative consumers) and the truth is surfaced alongside it. No escalation needed.

### Non-Goals

- `fab score` acceptance derivation — score reads `intake.md` only.
- Removing the `.status.yaml` write-time acceptance counter — it stays as a cache.
- Removing the clamp in prmeta — kept; only annotated.

## Tasks

### Phase 1: Schema & helpers (foundations other tasks build on)

- [x] T001 Add `ChangeTypeSource string` field to `StatusFile` in `src/go/fab/internal/statusfile/statusfile.go` (yaml tag `change_type_source`); parse it in `Load`, write it in `syncToRaw` (update existing key + insert-when-absent like `change_type`), and define exported `SourceInferred`/`SourceExplicit` constants <!-- R2 -->
- [x] T002 Add `LiveAcceptance(changeDir string) (done, total int, ok bool)` to `src/go/fab/internal/status` (new file `acceptance.go`) reading `{changeDir}/plan.md` `## Acceptance` via `hooklib.HasSectionHeading` + `CountSectionItemsBounded`/`CountCompletedSectionItemsBounded`; `ok=false` when plan.md absent or no `## Acceptance` heading <!-- R5 -->

### Phase 2: Core inference & guard

- [x] T003 Tighten the `fix` pattern in `src/go/fab/internal/hooklib/artifact.go:96` so hyphen-adjacent occurrences of `fix`/`bug` do not match (RE2: explicit non-hyphen boundary or post-match guard), keeping `bug-fix`/`hot-fix`/`bug-free`/standalone `fix`/`bug`/`broken`/`regression` → `fix` and excluding passing `must-fix`/`must fix` <!-- R1 -->
- [x] T004 In `src/go/fab/internal/status/status.go`, make `SetChangeType` set `ChangeTypeSource = SourceExplicit`; add an `ApplyChangeTypeSource` helper (or set the field in `ApplyChangeType`'s explicit-set path) so the explicit marker is set alongside the type and persisted <!-- R3 -->
- [x] T005 In `src/go/fab/cmd/fab/hook.go` `artifactBookkeeping` intake branch, skip `InferChangeType` + `ApplyChangeType` when `statusFile.ChangeTypeSource == SourceExplicit`; when absent/`inferred`, run inference as today (do not change the source) <!-- R4 -->

### Phase 3: Read-time acceptance derivation

- [x] T006 In `src/go/fab/internal/preflight/preflight.go`, prefer `status.LiveAcceptance(changeDir)` for `acceptance_completed`/`acceptance_count` in the `Result`/`FormatYAML` path, falling back to `statusFile.Plan` when `ok=false` <!-- R5 -->
- [x] T007 In `src/go/fab/internal/prmeta/prmeta.go` `Gather`, prefer `status.LiveAcceptance(changeDir)` for `d.AcceptanceDone`/`d.AcceptanceTotal` (both the plan.md and legacy tasks.md branches), falling back to `status.Plan.*` when `ok=false` <!-- R5 -->
- [x] T008 In `src/go/fab/cmd/fab/status.go` `statusPlanCmd`, prefer `status.LiveAcceptance(changeDir)` for the `acceptance_completed`/`acceptance_count` output, falling back to the persisted counter when `ok=false` <!-- R5 -->

### Phase 4: Scaffold errors, resolve sentinels, prmeta annotation

- [x] T009 In `src/go/fab-kit/internal/scaffold.go` `lineEnsureMerge`, propagate the two swallowed `os.WriteFile` errors (~L273 symlink-resolve, ~L294 create-with-entry) and check the appended-write/`f.Close()` errors; fix the `scaffoldDirectories` doc comment (~L12-14) so it no longer misdescribes propagation <!-- R6 -->
- [x] T010 In `src/go/fab/internal/resolve/resolve.go`, declare `ErrNotFound`/`ErrAmbiguous` and wrap the not-found / no-active-change(s) messages with `%w ErrNotFound` and the multiple-match / multiple-changes-exist messages with `%w ErrAmbiguous` <!-- R7 -->
- [x] T011 Update archive soft-skip callers to branch on `errors.Is(err, resolve.ErrNotFound)` (→ check archive, soft-skip) vs `errors.Is(err, resolve.ErrAmbiguous)` (→ surface) in `src/go/fab/internal/archive/archive.go` `Archive` (~L66-75) and `src/go/fab/cmd/fab/batch_archive.go` `runBatchArchive` (~L80-91) <!-- R7 -->
- [x] T012 In `src/go/fab/internal/prmeta/prmeta.go` `renderImpact`, compute pre-clamp impl Net/Added/Deleted and, when the clamp engages (pre-clamp < 0), annotate the impl row with the true value (e.g. `(net +0, clamped from −N)`); keep the clamped value as the displayed number <!-- R8 -->

### Phase 5: Tests & docs

- [x] T013 [P] Add tests: `InferChangeType` must-fix non-match + bug-fix/hot-fix/bug-free/standalone matches (`hooklib/artifact_test.go`); `change_type_source` round-trip (`statusfile/statusfile_test.go`); `LiveAcceptance` live count incl. sed-edited checkbox + absent-plan fallback (`status` package test); scaffold WriteFile error propagation (`fab-kit/internal/scaffold_test.go`); `errors.Is` resolve sentinels (`resolve/resolve_test.go`); prmeta clamp annotation under test-heavy diff (`prmeta/prmeta_test.go`); set-change-type marks explicit + hook respects explicit (`cmd/fab` or `status` test) <!-- R1 R2 R3 R4 R5 R6 R7 R8 -->
- [x] T014 [P] Update `src/kit/skills/_cli-fab.md`: `set-change-type` note (now marks `change_type_source: explicit`); document the new `.status.yaml` `change_type_source` field and read-time acceptance derivation; add `resolve` `ErrNotFound`/`ErrAmbiguous` note to Common Error Messages if behavior is observable <!-- R2 R3 R5 R7 -->

## Execution Order

- T001, T002 (Phase 1) block their consumers: T004/T005 depend on T001; T006/T007/T008 depend on T002.
- T013 depends on all implementation tasks (T001-T012); T014 is doc-only and independent.
- T003, T009, T010, T012 are mutually independent; T011 depends on T010.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `InferChangeType` returns non-`fix` for a feature intake whose only fix-adjacent token is `must-fix`/`must fix`, and `fix` for `bug-fix`/`hot-fix`/`bug-free`/standalone `fix`/`bug`/`broken`/`regression`
- [x] A-002 R2: `.status.yaml` carries `change_type_source`; absent field decodes as `inferred`-equivalent; the field round-trips through Load/Save and is inserted on a sparse document
- [x] A-003 R3: `fab status set-change-type` persists `change_type_source: explicit` alongside the type
- [x] A-004 R4: the intake-write hook skips inference + type overwrite when source is `explicit`, and runs inference when source is absent/`inferred`
- [x] A-005 R5: a shared `status.LiveAcceptance(changeDir)` helper exists and read sites (preflight, prmeta, `status plan`) prefer the live count, falling back to the persisted counter when plan.md/`## Acceptance` is absent
- [x] A-006 R6: `lineEnsureMerge` propagates both `os.WriteFile` errors and the `scaffoldDirectories` doc comment is accurate
- [x] A-007 R7: `resolve.ErrNotFound`/`ErrAmbiguous` exist, messages wrap them with `%w`, and archive soft-skip branches on `errors.Is`
- [x] A-008 R8: prmeta annotates the true pre-clamp impl value when the clamp engages, keeping the clamped display value

### Behavioral Correctness

- [x] A-009 R4: an explicit-set change_type survives a subsequent intake re-infer write (no silent revert)
- [x] A-010 R5: a sed-toggled `## Acceptance` checkbox is reflected by the readers even when the `.status.yaml` counter is stale
- [x] A-011 R7: an ambiguous change name during archive is surfaced (not soft-skipped as already-archived)
<!-- A-011: behavior implemented & verified (archive.go gates the already-archived guess on errors.Is(ErrNotFound); batch_archive.go surfaces ErrAmbiguous with a warning + continue). Resolve-layer classification is unit-tested (resolve_test.go), but no archive-level test exercises the ambiguous-surfacing scenario end-to-end — see Should-fix in review report. -->

### Edge Cases & Error Handling

- [x] A-012 R5: a change with no `plan.md` or no `## Acceptance` heading reads the persisted counter without error
- [x] A-013 R6: a scaffold write failure produces a non-nil error up the `scaffoldTreeWalk` chain (no silent half-scaffold)

### Code Quality

- [x] A-014 Pattern consistency: new code follows the package's existing naming, error-wrapping (`%w`), and yaml-node serialization patterns (e.g. `insertKey`/`syncToRaw` for the new field; `ErrAlreadyArchived` precedent for sentinels)
- [x] A-015 No unnecessary duplication: `LiveAcceptance` reuses the existing `hooklib` counters rather than reimplementing checkbox parsing; the new regex change reuses the existing `changeTypePatterns` structure
- [x] A-016 Readability over cleverness: the regex tightening is the simplest correct mechanism (no opaque RE2 gymnastics); functions stay under the >50-line god-function threshold

### Documentation Accuracy

- [x] A-017 `src/kit/skills/_cli-fab.md` reflects the `set-change-type` explicit-set behavior, the new `change_type_source` field, and read-time acceptance derivation (constitution: Go change MUST update `_cli-fab.md`)

### Cross References

- [x] A-018 The `scaffoldDirectories` doc comment and any `.status.yaml` schema references stay internally consistent with the implemented behavior

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- `src/kit/` is canonical — never edit `.claude/skills/` directly.

## Deletion Candidates

None — this change adds new functionality without making existing code redundant.

(Rationale: every change is additive or a tightening of an existing line. The old `fix` regex `\b(fix|bug|broken|regression)\b` was not deleted — it was renamed to `fixCandidateRegex` and reused inside `fixSignal`, with the `changeTypePatterns` "fix" entry's `Pattern` set to `nil` and special-cased in `InferChangeType` (still load-bearing as the first-match-wins ordering slot, not dead). `clampNonNeg` is kept (deliberately — downstream consumers may assume non-negative; only annotated). The `.status.yaml` write-time acceptance counter is kept as a cache (read-time derivation is preferred but the counter is the documented fallback). The two `resolve` `fmt.Errorf` call sites were converted to `notFoundf`/`ambiguousf`, not removed. No symbol, branch, file, or config became unreferenced.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Regex tightening uses an explicit non-hyphen boundary group with submatch extraction (RE2 has no lookbehind), keeping the first-match-wins ordering | Intake assumption #2/#8 confirmed; mechanism is the standard RE2 workaround | S:90 R:75 A:80 D:70 |
| 2 | Confident | `LiveAcceptance` returns `(done, total int, ok bool)` and lives in `internal/status`; readers fall back to the persisted cache when `ok=false` | Intake #5 confirmed counter-as-cache; the `ok` tri-state cleanly handles absent plan.md without sentinel ints | S:90 R:65 A:80 D:75 |
| 3 | Confident | `change_type_source` stored as a plain string field (empty == inferred) with `SourceInferred`/`SourceExplicit` constants, serialized only when non-empty-or-via-insert like `change_type` | Intake #6 confirmed enum; matches existing sparse-doc insert pattern (`change_type` is inserted only when non-empty) | S:90 R:55 A:80 D:80 |
| 4 | Confident | (e) keep clamp + annotate; the binary-review Refuted section (R1-R8) does NOT adjudicate the clamp, so the intake's resolved design stands with no escalation | Read the Refuted section per intake instruction; it covers unrelated findings, so neither "intentionally lossy" nor "safe to remove" was decided — annotate is the minimal honest change | S:95 R:55 A:75 D:80 |
| 5 | Confident | (e) annotation format renders inline on the impl row as `(net +N, clamped from −M)` (or per-field) rather than a separate line, to stay within the existing `**Impact**:` multi-row block | Lowest-churn placement; reversible formatting choice | S:80 R:85 A:75 D:65 |

5 assumptions (0 certain, 5 confident, 0 tentative).
