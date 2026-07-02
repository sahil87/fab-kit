# Plan: Freeze-on-write log.md generation

**Change**: 260616-tayp-freeze-on-write-logs
**Intake**: `intake.md`

## Requirements

<!-- Derived from intake §1–§8. RFC-2119 statements with stable R# IDs and at
     least one GIVEN/WHEN/THEN scenario each. The intake §8 test matrix (TC1–TC12)
     is mapped to these requirements and to the Acceptance items below. -->

### Generation: Freeze-on-write

#### R1: Existing log is authoritative and write-once
`fab memory-index` SHALL read each folder's existing `log.md` (when present), parse it back
into `[]LogEntry` via `parseLog` (the inverse of `RenderLog`), and treat those entries as
immutable: they MUST NOT be rewritten, reworded, re-dated, or dropped during a normal
(non-`--rebuild`) regeneration. Regeneration MUST only **append** newly-discovered entries.

- **GIVEN** a folder with a committed `log.md` containing entries
- **WHEN** `fab memory-index` runs again on the same git state
- **THEN** every existing entry is preserved byte-for-byte in the merged output
- **AND** the second run is a byte-for-byte no-op (idempotence — Constitution III)

#### R2: Append/dedup key is `(file-base, change-id)`
The append guard SHALL key on the `(file-base, change-id)` pair. An **attributable** projected
entry (one whose commit resolves via `attributeCommit` to a registry change-id) MUST be appended
only if no existing entry already records that `(file-base, change-id)` pair. The git commit hash
(`%H`) MUST NOT be used as the key (squash + branch-delete makes it unreachable — intake Origin #4).

- **GIVEN** a frozen `log.md` with an entry for `(foo, a1b2)`
- **WHEN** the projection contains a commit for `(foo, a1b2)` and a commit for `(foo, c3d4)`
- **THEN** `(foo, a1b2)` is a no-op (already present) and `(foo, c3d4)` is appended exactly once
- **AND** a squash that preserved the `a1b2` token under a single new commit is still a no-op for `(foo, a1b2)`

#### R3: Unattributable commits are frozen, not re-projected
After first write, **new** unattributable commits (those `attributeCommit` cannot resolve to a
registry change-id) MUST NOT be projected into the log. Unattributable entries **already present**
in `log.md` MUST stay verbatim (frozen). Projection of unattributable commits is permitted ONLY at
**bootstrap** (the existing `log.md` is empty/absent) or under `--rebuild`.

- **GIVEN** a frozen `log.md` carrying unattributable lines
- **WHEN** a re-run projects different (squash-reworded) unattributable subjects for the same files
- **THEN** the frozen unattributable lines are unchanged
- **AND** the new unattributable commit is NOT appended to the log

#### R4: Seed-merge is preserved
The `log.seed.md` seed-merge (`readSeedEntries` → `mergeSeedEntries`) MUST continue to merge seed
entries beneath the git projection at first write and under `--rebuild`. After first write, a seed
entry already present in the frozen log MUST de-duplicate (no regression to `mergeSeedEntries`
idempotence). The merged set (existing ∪ appended-projection ∪ seed) MUST render through the
unchanged pure `RenderLog`.

- **GIVEN** a folder with a `log.seed.md` at first write (or under `--rebuild`)
- **WHEN** `fab memory-index` generates the log
- **THEN** seed entries appear beneath the projected entries
- **AND** a re-run does not duplicate any seed entry

### Parsing: `parseLog`

#### R5: `parseLog` is a faithful inverse of `RenderLog`
A new `parseLog(content string) []LogEntry` SHALL parse a rendered `log.md` body back into
`[]LogEntry` such that `parseLog(RenderLog(entries))` recovers the same entry set (set-wise; render
re-sorts). It MUST share `parseSeedLog`'s entry-line grammar (verb / `[base](/bundle-rel.md)` /
` — summary` / optional trailing `(id)`). Malformed lines MUST degrade gracefully (skipped, no panic).

- **GIVEN** entries with a verb, a bundle-relative path, a summary, and an `(id)` token
- **WHEN** `parseLog(RenderLog(entries))` runs
- **THEN** the parsed set equals the input set
- **AND** a malformed or non-entry line (header comment, blank, stray prose) is skipped without panic

### CLI: `--rebuild`

#### R6: `--rebuild` flag re-projects destructively
`fab memory-index` SHALL accept a `--rebuild` flag that discards the accumulated frozen state and
re-projects every `log.md` from current git (the pre-freeze behavior, made explicit and opt-in).
It MUST project unattributable commits (R3 gate open) and MUST be documented as destructive. It
MUST NOT be the default path; without `--rebuild`, freeze-on-write is in effect. There SHALL be NO
`--first-generation` flag (bootstrap is the first append into an empty log).

- **GIVEN** a frozen `log.md` containing lines unreachable from current git (squash-stale)
- **WHEN** `fab memory-index --rebuild` runs
- **THEN** the log is re-projected from current git, dropping the now-unreachable lines
- **AND** a plain `fab memory-index` run would instead have preserved those lines

### CLI: `--check` redesign

#### R7: `--check` PASSES on a valid superset
Under freeze-on-write, `--check` MUST NOT fail merely because the committed `log.md` contains
entries not derivable from current git (legitimately-frozen, squashed-away history). A committed
log that is a valid **superset** of the freeze-on-write merge result (it contains every entry the
merge would produce, plus extra frozen entries) MUST classify as **clean** (exit 0) for that
`log.md` target.

- **GIVEN** a committed `log.md` with frozen lines absent from the live projection
- **WHEN** `fab memory-index --check` runs and the merge would append nothing new
- **THEN** the `log.md` target does not contribute drift (the case that false-fails byte-equality today)

#### R8: `--check` FAILS on a missing attributable entry
`--check` MUST report a non-clean result when the projection contains an attributable
`(file-base, change-id)` entry that the committed `log.md` lacks (a genuine gap — someone forgot to
regenerate-and-commit). The report MUST name the gap.

- **GIVEN** a committed `log.md` missing a `(file, change-id)` the projection would append
- **WHEN** `fab memory-index --check` runs
- **THEN** the result is non-clean (drift) and the report names the missing entry

#### R9: `--check` FAILS on a hand-edited frozen line
`--check` MUST report a non-clean result when an existing committed `log.md` line was hand-altered
in a way that breaks the freeze-on-write merge identity (the single-writer discipline was violated).

- **GIVEN** a committed `log.md` whose freeze-on-write merge does not reproduce its current bytes
- **WHEN** `fab memory-index --check` runs
- **THEN** the result is non-clean (drift)

#### R10: `log.md` drift stays benign (no new tier-2 category)
The `--check` redesign MUST keep `log.md` drift in the **benign** tier (never destructive-loss /
tier 2). The three index-only destructive-loss detectors (description / tombstone / grouping) MUST
remain skipped for `log.md` targets (the `IsLog` guard, intake OQ4 / assumption #9). Index drift
classification (tiers for `index.md`) MUST be unchanged.

- **GIVEN** a `log.md` target that drifts (missing entry, hand-edit, or stale projection)
- **WHEN** `Classify` runs
- **THEN** the highest tier contributed by that `log.md` is benign drift (1), never destructive loss (2)
- **AND** a byte-identical `log.md` contributes no drift (tier 0)

### Distribution: Re-baseline migration

#### R11: Ship a re-baseline migration with a binary pre-check
A migration file SHALL be shipped in `src/kit/migrations/` (version `2.5.5` → next) that
re-baselines existing projects to freeze-on-write by running `fab memory-index --rebuild` once and
committing the result. The migration MUST include a **pre-check** that the running binary
understands `--rebuild` (probe `fab memory-index --help`); if not, it MUST abort with a clear
"upgrade the binary first" message and perform no partial rewrite. It MUST be idempotent and follow
the existing migration format (`2.4.2-to-2.5.0.md` precedent). `src/kit/VERSION` SHALL be bumped to
the migration's target version.

- **GIVEN** a project on an older binary that lacks `--rebuild`
- **WHEN** the migration's pre-check runs
- **THEN** it aborts with the upgrade-first message and rewrites nothing
- **AND** with a `--rebuild`-aware binary the migration re-projects + commits one clean baseline

### Docs: Spec + CLI reference

#### R12: Spec (`fkf.md` §6) and CLI reference (`_cli-fab.md`) document the new model
`docs/specs/fkf.md` §6 MUST describe freeze-on-write, the `(file-base, change-id)` key, the
unattributable-freeze rule, `--rebuild`, and the new `--check` superset/missing/hand-edit semantics.
`src/kit/skills/_cli-fab.md` § `fab memory-index` MUST document the `--rebuild` flag and the changed
`--check` contract (Constitution: CLI-change rule MUST update `_cli-fab.md`).

- **GIVEN** the freeze-on-write behavior has shipped in the binary
- **WHEN** a reader consults `fkf.md` §6 or `_cli-fab.md`
- **THEN** both describe freeze-on-write, the change-id key, unattributable-freeze, `--rebuild`, and the new `--check` semantics

### Non-Goals

- No `--first-generation` flag (R6 — bootstrap is not a special mode).
- No commit-hash key (R2 — change-id is the only squash-survivable key).
- No in-place update of a reworded summary on an already-frozen attributable entry (Assumption 1 below — keep-frozen, per intake OQ1).
- No live loom dependency in CI (TC10 fixture is synthesized git history).

### Design Decisions

1. **Unattributable-projection gate is a parameter on `gatherLogEntries`**: approach — thread a
   `projectUnattributable bool` through `gatherLogEntries`/`buildLogTarget`/`GatherLogs` rather than
   a package global or a duplicate projection function. *Why*: keeps the projection pure and a
   single source of truth; the gate is open only at bootstrap (empty existing log) or `--rebuild`.
   *Rejected*: a second `gatherLogEntriesRebuild` copy (duplicates the registry-join logic — code-quality anti-pattern).

2. **Append guard correlates on `(FileBase, ChangeID)`**: approach — build a `set[fileBase|changeID]`
   from the parsed existing entries and append a projected attributable entry only when its key is
   absent. *Why*: R2's squash-survivable identity. *Rejected*: byte-equality of the whole `LogEntry`
   (would re-append a reworded-summary line, defeating freeze-on-write).

3. **`--check` reuses the freeze-on-write merge as the "rendered" content**: approach — under
   `--check`, the `Rendered` field of each log `CheckTarget` is the freeze-on-write merge result
   (existing ∪ new-append ∪ seed), so the existing byte-compare in `Classify` already yields the
   right verdict — clean when the merge reproduces the committed bytes (R7), drift when it appends a
   missing entry (R8) or cannot reproduce a hand-edited line (R9). *Why*: one render pass serves both
   write and `--check`, no duplicate classifier; `log.md` stays `IsLog` (benign-only, R10). *Rejected*:
   a bespoke subset/superset comparator in `loss.go` (the merge-as-rendered approach makes the
   existing byte-compare express exactly subset/superset without new tier machinery — see Assumption 4).

## Tasks

<!-- Grouped by phase. Each item carries a <!-- R# --> trace annotation. -->

### Phase 1: Parsing primitive

- [x] T001 Add `parseLog(content string) []LogEntry` to `src/go/fab/internal/memoryindex/log.go` as the inverse of `RenderLog`, sharing `parseSeedLog`'s entry-line grammar from `seed.go` (extract a shared `parseLogEntryLine`/date-walk helper so the grammar is single-sourced, not copy-pasted). <!-- R5 -->

### Phase 2: Freeze-on-write generation core

- [x] T002 Add `projectUnattributable bool` parameter to `gatherLogEntries` in `src/go/fab/internal/memoryindex/memoryindex.go`; gate the unattributable branch (the `else` that sets `summary = touch.Subject`) so an unattributable commit is projected only when `projectUnattributable` is true (drop the entry otherwise). <!-- R3 -->
- [x] T003 Rework `buildLogTarget` in `src/go/fab/internal/memoryindex/memoryindex.go` to: read the existing `log.md` via `parseLog`; decide `bootstrap := len(existing)==0`; project with `projectUnattributable = bootstrap || rebuild`; append only attributable projected entries whose `(FileBase, ChangeID)` key is absent from the existing set (existing entries are authoritative); merge the seed beneath; under `--rebuild` discard the existing set and re-project everything. Render the union via `RenderLog`. Thread a `rebuild bool` parameter through `buildLogTarget` and `GatherLogs`. <!-- R1 R2 R3 R4 R6 -->
- [x] T004 Add an append-merge helper (e.g. `appendNewEntries(existing, projected []LogEntry) []LogEntry`) in `src/go/fab/internal/memoryindex/memoryindex.go` (or `log.go`) that keys on `(FileBase, ChangeID)` and preserves existing entries verbatim ahead of appended ones. <!-- R1 R2 -->

### Phase 3: CLI wiring

- [x] T005 Add the `--rebuild` bool flag to `memoryIndexCmd` in `src/go/fab/cmd/fab/memory_index.go`; thread it into the `GatherLogs(repoRoot, fabRoot, rebuild)` call; update the command Long text to describe freeze-on-write + `--rebuild`. <!-- R6 -->
- [x] T006 In the `--check` branch of `src/go/fab/cmd/fab/memory_index.go`, ensure the log `CheckTarget.Rendered` is the freeze-on-write merge result (rebuild=false), so `Classify` byte-compares against the merge (superset PASS / missing-entry & hand-edit FAIL), keeping `IsLog` benign-only. <!-- R7 R8 R9 R10 -->

### Phase 4: Tests (all 12 TCs — first-class deliverable, Constitution VII)

- [x] T007 [P] Add `parseLog` round-trip + malformed-line tests to `src/go/fab/internal/memoryindex/log_test.go` (TC5). <!-- R5 -->
- [x] T008 [P] Add freeze-on-write generation tests to `src/go/fab/internal/memoryindex/memoryindex_test.go` (or a new `freeze_test.go`): TC1 idempotence, TC2 append-on-new-change-id, TC3 no-op on squashed-but-attributable, TC4 freeze-of-unattributable, TC6 `--rebuild` re-projects, TC12 seed-merge-preserved. Build from synthesized git history using the `gitDateRun`/`writeFile` helpers. <!-- R1 R2 R3 R4 R6 -->
- [x] T009 [P] Add the loom regression fixture test (TC10) to `src/go/fab/internal/memoryindex/` built from synthesized git history (squash collapses Part 2a/Part 2b → #1721); assert 0 churn (merged log identical to frozen input; 0 appended, 0 destroyed) — NO live loom dependency. <!-- R1 R2 R3 -->
- [x] T010 [P] Add `--check` classification tests to `src/go/fab/internal/memoryindex/loss_test.go` (or alongside): TC7 PASS on valid superset, TC8 FAIL on missing attributable entry, TC9 FAIL on hand-edit, TC10/R10 log drift stays benign. <!-- R7 R8 R9 R10 -->
- [x] T011 Add the migration pre-check test (TC11) — verify the migration aborts against an old binary lacking `--rebuild` with the "upgrade the binary first" message and no partial rewrite. Implemented as a Go test exercising the pre-check probe logic (no live old binary needed) and/or asserted in the migration file's verification section. <!-- R11 -->

### Phase 5: Distribution + docs

- [x] T012 Create the re-baseline migration `src/kit/migrations/2.5.5-to-2.6.0.md` (FKF-cutover format precedent `2.4.2-to-2.5.0.md`): pre-check the binary understands `--rebuild`; run `fab memory-index --rebuild`; commit the baseline; idempotent; documents the one-time churn. <!-- R11 -->
- [x] T013 Bump `src/kit/VERSION` to `2.6.0` (the migration target version). <!-- R11 -->
- [x] T014 Update `docs/specs/fkf.md` §6 to describe freeze-on-write, the `(file-base, change-id)` key, the unattributable-freeze rule, `--rebuild`, and the new `--check` superset/missing/hand-edit semantics. <!-- R12 -->
- [x] T015 Update `src/kit/skills/_cli-fab.md` § `fab memory-index`: document the `--rebuild` flag and the changed `--check` contract. <!-- R12 -->

## Execution Order

- T001 (parseLog) blocks T003/T004 (generation reads it back) and T007 (its test).
- T002 (projection gate) blocks T003.
- T003/T004 (generation core) block T005/T006 (CLI wiring) and T008/T009 (generation tests).
- T012 (migration) and T013 (VERSION) pair; T011 (pre-check test) relates to T012.
- T014/T015 (docs) are independent and can run last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab memory-index` reads the existing `log.md`, treats its entries as immutable, and appends-only on regeneration; a second run on the same state is a byte-for-byte no-op (TC1).
- [x] A-002 R2: the append guard keys on `(file-base, change-id)`; a new change-id is appended exactly once and an already-present pair is a no-op (TC2); commit-hash is not used as the key.
- [x] A-003 R3: new unattributable commits are not projected after first write, and frozen unattributable lines are preserved verbatim (TC4).
- [x] A-004 R4: the `log.seed.md` seed-merge still merges beneath the projection at first write / `--rebuild`, without regression (TC12).
- [x] A-005 R5: `parseLog` is a faithful inverse of `RenderLog` and degrades gracefully on malformed lines (TC5).
- [x] A-006 R6: `fab memory-index --rebuild` re-projects every `log.md` from current git, dropping now-unreachable lines (TC6); no `--first-generation` flag exists.
- [x] A-007 R7: `--check` exits clean for a committed log that is a valid superset of the merge result (TC7 — the case that false-fails today).
- [x] A-008 R8: `--check` reports drift (non-clean) when an attributable `(file, change-id)` the projection has is missing from the committed log, naming the gap (TC8).
- [x] A-009 R9: `--check` reports drift when a frozen line was hand-edited (TC9).
- [x] A-010 R11: a re-baseline migration ships with a binary pre-check that aborts on an old binary; `src/kit/VERSION` is bumped (TC11).
- [x] A-011 R12: `docs/specs/fkf.md` §6 and `src/kit/skills/_cli-fab.md` document freeze-on-write, the change-id key, unattributable-freeze, `--rebuild`, and the new `--check` semantics.

### Behavioral Correctness

- [x] A-012 R2: a squash that preserves the change-id token under a single new commit is a no-op for that `(file, change-id)` pair (TC3).
- [x] A-013 R10: `log.md` drift remains in the benign tier (never destructive-loss); the index destructive-loss detectors remain skipped for `log.md` (`IsLog`), and index-target classification is unchanged.

### Scenario Coverage

- [x] A-014 R1 R2 R3: the loom regression fixture (squash collapses Part 2a/Part 2b → #1721) yields 0 churn across the folder set — merged log identical to the frozen input, 0 appended, 0 destroyed — built from synthesized git history with no live loom dependency (TC10).

### Edge Cases & Error Handling

- [x] A-015 R5: malformed / non-entry `log.md` lines are skipped without panic during `parseLog`.
- [x] A-016 R3 R6: bootstrap (empty/absent existing `log.md`) projects unattributable commits through the same code path as `--rebuild`; a non-empty existing log does not.
- [x] A-017 R11: the migration is idempotent — re-running `--rebuild` + commit on an already-clean tree is a no-op diff.

### Code Quality

- [x] A-018 Pattern consistency: new code follows the `memoryindex` package's doc-comment density and pure/I-O split idiom (pure parse/merge helpers, I/O in the Gather orchestrators).
- [x] A-019 No unnecessary duplication: `parseLog` shares `parseSeedLog`'s entry-line grammar rather than re-implementing it; the unattributable gate is a parameter, not a duplicated projection function.
- [x] A-020 No god functions / magic strings: the append-merge and projection-gate logic are focused helpers; the `(file-base, change-id)` key and bootstrap condition use clear named locals.

### Documentation Accuracy

- [x] A-021 R12: the `fkf.md` §6 and `_cli-fab.md` updates accurately describe the shipped binary behavior (flags, exit semantics) with no drift from the implementation.

### Cross-References

- [x] A-022 R11 R12: the migration references `docs/specs/fkf.md` §6 and the `--rebuild` flag accurately; the spec/CLI cross-references resolve.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | A reworded `.status.yaml` summary on an already-frozen attributable change keeps the FROZEN text (no in-place update); the `(file-base, change-id)` append key makes the re-projected entry a no-op | Intake OQ1 leans keep-frozen (simplest, matches immutability rule R1); the key naturally de-dups so no extra code path is needed; reversible later | S:70 R:65 A:75 D:65 |
| 2 | Confident | Bootstrap is detected as `len(existingParsed)==0` (empty/absent `log.md`); the unattributable-projection gate opens at bootstrap OR `--rebuild` | Intake §4 states bootstrap is "the first append into an empty log" through the same code path; an empty parse is the natural signal | S:80 R:70 A:85 D:80 |
| 3 | Confident | `--check` uses the freeze-on-write merge (rebuild=false) as each log target's `Rendered`, so the existing byte-compare in `Classify` expresses superset-PASS / missing-FAIL / hand-edit-FAIL without new tier machinery | Intake §5 requires the redesign; reusing the merge-as-rendered is the minimal change that satisfies R7/R8/R9 while keeping `IsLog` benign-only (R10); intake OQ2 (benign vs destructive missing) resolves to benign since `log.md` is always `IsLog` | S:65 R:60 A:70 D:60 |
| 4 | Confident | `loss.go`'s `Classify`/tier machinery is NOT structurally redesigned; the subset/superset semantics live in the cmd's choice of `Rendered` (the merge) plus the existing `IsLog` benign-only path | Intake §5 says "reworks loss.go's tier machinery and emitCheckReport" but the proven-minimal realization is merge-as-rendered; the `IsLog` guard already exists and keeps logs benign — redesigning the tier enum would be churn without behavioral gain (Constitution: follow existing patterns) | S:60 R:60 A:65 D:55 |
| 5 | Confident | The migration target version is `2.6.0` (next minor after `2.5.5`) | Intake §6 says "next version after 2.5.5"; the migration catalog uses minor bumps for feature migrations (`2.4.2-to-2.5.0`, `2.2.0-to-2.3.0`); a behavior change to a shipped CLI is minor-worthy | S:75 R:80 A:80 D:70 |
| 6 | Confident | TC11 (migration pre-check) is realized as a Go test over the pre-check probe semantics plus the migration's own Verification section, not by spawning a real old binary in CI | Intake §8 TC11 tests the abort behavior; spawning an historical binary in CI is infeasible, so the probe logic (parse `fab memory-index --help` for `--rebuild`) is unit-tested and the migration documents the manual abort | S:65 R:70 A:70 D:60 |

6 assumptions (0 certain, 6 confident, 0 tentative).
