# Plan: Code-Reorg Structure Sweep

**Change**: 260824-ffny-code-reorg-structure-sweep
**Intake**: `intake.md`

## Requirements

### Source Tree: cmd/fab filename normalization

#### R1: Concatenated multi-word command files SHALL be renamed to the underscore convention
The four concatenated multi-word files in `src/go/fab/cmd/fab/` (and their `_test.go` twins — 8 files) MUST be renamed via `git mv` with **zero content change**: `panemap.go` → `pane_map.go`, `fabhelp.go` → `fab_help.go`, `helpdump.go` → `help_dump.go`, `shellinit.go` → `shell_init.go` (tests likewise). All are `package main`; no import or identifier changes. None of the new names may introduce an implicit Go build-constraint suffix.

- **GIVEN** the 35-of-39 underscore majority in `src/go/fab/cmd/fab/` and the `pane_*.go` prefix of all 9 `fab pane` sibling files
- **WHEN** the 8 files are renamed via `git mv`
- **THEN** `go build ./...` and `go test ./...` in `src/go/fab` pass unchanged, and `git log --follow` preserves each file's history

#### R2: Present-truth docs naming the old filenames SHALL be updated
Exactly the grep-verified present-truth occurrences MUST be updated to the new filenames: `docs/specs/skills.md:214` (`fabhelp.go`), `docs/memory/pipeline/code-reorg.md:17` (`fabhelp.go`), and `docs/memory/distribution/kit-architecture.md` lines 309 (`shellinit.go`), 311 (`helpdump.go`), 313/317/357 (`panemap.go`/`fabhelp` mentions). Historical records (`log.md`, `log.seed.md`, `docs/specs/findings/*`) MUST NOT be touched.

- **GIVEN** the sweep set verified by grep on 2026-08-24
- **WHEN** the doc sweep completes
- **THEN** `grep -rn 'panemap\.go\|fabhelp\.go\|helpdump\.go\|shellinit\.go' docs/ --include='*.md'` matches only `log.md`/`log.seed.md`/`docs/specs/findings/*` entries

### Repository Layout: benchmark decision record

#### R3: The src/benchmark decision record SHALL move to docs/findings
`src/benchmark/README.md` and `src/benchmark/RESULTS.md` MUST move via `git mv` to `docs/findings/statusman-benchmark/` (leaving `src/benchmark/` gone). `docs/memory/distribution/kit-architecture.md:516` MUST be repointed to the new path, and `docs/findings/index.md` MUST gain a row for the relocated record, marked as a historical/closed decision record. The files' content is unchanged (the README→RESULTS.md relative link survives because both move together).

- **GIVEN** `src/benchmark/` holds only the two markdown files
- **WHEN** the move and doc updates complete
- **THEN** `src/` contains only `go/` and `kit/`, the findings index lists the record, and no present-truth doc references `src/benchmark/`

### Go Packages: hooklib → artifact rename

#### R4: Package internal/hooklib SHALL be renamed to internal/artifact
`git mv src/go/fab/internal/hooklib src/go/fab/internal/artifact`; the package clause in `artifact.go` and `artifact_test.go` changes `hooklib` → `artifact`; the two importers update import path and selectors: `src/go/fab/internal/refresh/refresh.go` (line 19 import; selectors at 80, 143–144, 156, 162–163) and `src/go/fab/internal/status/acceptance.go` (line 7 import; selectors at 28, 31–32; line-14 comment). The comment-only mention at `src/go/fab/internal/prmeta/prmeta.go:538` MUST also be updated. No exported identifier changes; behavior is identical.

- **GIVEN** the verified complete importer set (refresh, status)
- **WHEN** the rename and import updates complete
- **THEN** `grep -rn hooklib src/go --include='*.go'` returns nothing and `go test ./...` in `src/go/fab` passes

#### R5: Present-truth docs mentioning hooklib SHALL be rewritten to the new name
The grep-verified present-truth mentions MUST be updated: `docs/memory/pipeline/schemas.md`, `docs/specs/change-types.md`, `docs/memory/pipeline/hooks-may-enhance-never-own.md`, `docs/memory/runtime/runtime-agents.md`, and `docs/memory/distribution/kit-architecture.md` — where the now-superseded claim "the package keeps its legacy name for now (ioku)" MUST be rewritten to present truth (the package is now `internal/artifact`), not merely token-swapped. Historical records stay untouched.

- **GIVEN** the sweep set verified by grep on 2026-08-24
- **WHEN** the doc sweep completes
- **THEN** `grep -rln hooklib docs/` matches only `log.md`/`log.seed.md`/`docs/specs/findings/*`

### Verification

#### R6: The change SHALL prove zero behavior change and index consistency
`go test ./...` MUST pass in `src/go/fab` (covering at minimum `cmd/fab`, `internal/artifact`, `internal/refresh`, `internal/status`); `gofmt -l` over changed `.go` files MUST report nothing; `fab memory-index` MUST be re-run after the memory sweeps so index descriptions don't drift.

- **GIVEN** all renames, moves, and sweeps are complete
- **WHEN** the verification suite runs
- **THEN** tests pass, gofmt is clean, and `fab memory-index --check` reports no drift

### Non-Goals

- Deleting the benchmark decision record (the 2026-06 binary-review alternative) — user chose MOVE
- Renaming any other file or package (e.g. `status/true_impact.go`, the `config*` package family) — evaluated and dropped by the /code-reorg taste guard
- Any CLI signature, help text, or `_cli-fab.md` change — no command surface is touched
- Rewriting historical `log.md` / `log.seed.md` / `docs/specs/findings/*` entries — dated records are frozen

### Design Decisions

#### New package name is `artifact`
**Decision**: `internal/hooklib` becomes `internal/artifact`.
**Why**: the package's sole file is `artifact.go` and its content is artifact-content analysis (change-type inference, plan-section counting); the name lets a reader predict the content from the path.
**Rejected**: folding into `internal/intake` (that package is about intake *metadata* — title/description — not artifact parsing, and refresh/status would then import an oddly-named home); keeping `hooklib` (describes machinery deleted in kit 2.14.0).
*Introduced by*: 260824-ffny-code-reorg-structure-sweep

## Tasks

### Phase 2: Core Implementation

- [x] T001 [P] Rename the 8 cmd files via `git mv` in `src/go/fab/cmd/fab/` (panemap→pane_map, fabhelp→fab_help, helpdump→help_dump, shellinit→shell_init, each with its `_test.go` twin); then sweep old-filename mentions in `docs/specs/skills.md:214`, `docs/memory/pipeline/code-reorg.md:17`, `docs/memory/distribution/kit-architecture.md` (lines 309/311/313/317/357) <!-- R1, R2 -->
- [x] T002 [P] `git mv src/benchmark/{README,RESULTS}.md` → `docs/findings/statusman-benchmark/`; repoint `docs/memory/distribution/kit-architecture.md:516`; add a historical-record row to `docs/findings/index.md` <!-- R3 -->
- [x] T003 [P] `git mv src/go/fab/internal/hooklib` → `internal/artifact`; change the package clause in both files; update import + selectors in `internal/refresh/refresh.go` and `internal/status/acceptance.go` (incl. line-14 comment); update the comment at `internal/prmeta/prmeta.go:538` <!-- R4 -->
- [x] T004 Sweep present-truth `hooklib` mentions in `docs/memory/pipeline/schemas.md`, `docs/specs/change-types.md`, `docs/memory/pipeline/hooks-may-enhance-never-own.md`, `docs/memory/runtime/runtime-agents.md`, `docs/memory/distribution/kit-architecture.md` (rewrite the "keeps its legacy name for now" claim to present truth) <!-- R5 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Run `go test ./...` in `src/go/fab`, `gofmt -l` on changed `.go` files, regenerate memory indexes (`fab memory-index`), and verify the R2/R4/R5 grep post-conditions <!-- R6 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: All 8 cmd files exist under their new underscore names with byte-identical content (`git mv` only), and no old filename remains in `src/go/fab/cmd/fab/`
- [x] A-002 R3: `docs/findings/statusman-benchmark/{README,RESULTS}.md` exist, `src/benchmark/` is gone, and `docs/findings/index.md` lists the record
- [x] A-003 R4: `internal/artifact` package exists with both files; `internal/hooklib` is gone; both importers compile against the new path

### Behavioral Correctness

- [x] A-004 R6: `go test ./...` passes in `src/go/fab` and `gofmt -l` reports no changed file — zero behavior change demonstrated

### Removal Verification

- [x] A-005 R2: `grep -rn 'panemap\.go\|fabhelp\.go\|helpdump\.go\|shellinit\.go' docs/` matches only historical records (`log.md`, `log.seed.md`, `docs/specs/findings/*`)
- [x] A-006 R5: `grep -rln hooklib src/go docs/` matches only historical records; zero `.go` matches — one deliberate exception pre-declared in `## Notes`: the historical attribution "then named hooklib" in `src/go/fab-kit/internal/lock.go:21`
- [x] A-007 R3: no present-truth doc references `src/benchmark/`

### Scenario Coverage

- [x] A-008 R6: `fab memory-index --check` reports no drift after regeneration

### Code Quality

- [x] A-009 Pattern consistency: new filenames follow the directory's underscore convention; the new package name follows the single-word lowercase convention of its `internal/` siblings
- [x] A-010 No unnecessary duplication: no content was copied — every move is a `git mv` preserving history
- [x] A-011 Canonical source only: no edits under `.claude/skills/`; no `src/kit/` edits needed (verified zero mentions)
- [x] A-012 Sibling sweeps up front: met after review — the two stale `panemap` code comments the reviewer caught (`src/go/fab/cmd/fab/tmuxsocket_test.go:28`, `src/go/fab/cmd/fab/operator_tick_start.go:222`) were fixed post-verdict (should-fix, clear + low-effort per review policy); `grep -rn panemap src/` now returns nothing

## Notes

- **Sweep expansion at apply (per intake assumption #7 — all grep-found present-truth occurrences, not only the enumerated lines)**: the T005 post-condition greps surfaced occurrences beyond the intake's enumerated set, all swept: `fabhelp.go` mentions in `docs/memory/pipeline/{issue-linking,dedupe,execution-skills}.md` and `docs/memory/memory-docs/distill.md`; `panemap.go` mentions in `docs/memory/runtime/pane-commands.md` (×2) and code comments in `src/go/fab/cmd/fab/{pane_questions,dispatch_start}.go`; a `hooklib` code comment in `src/go/fab-kit/internal/lock.go` (rewritten as historical attribution "then named hooklib"). Remaining `hooklib` matches in present-truth docs are deliberate rename attributions or historical-fact clauses about files deleted in hk7p; remaining old-filename matches are only `log.md`/`log.seed.md`/`docs/specs/findings/*` (frozen records).
- **Known flake**: `TestPaneReady_JSON/ready` failed once mid-run — verified intermittent on identical code (pass/fail/pass same tree, passes on clean main) and pre-recorded as the pre-existing zsh-wizard flake; unrelated to this change (pane_ready files untouched). Full `cmd/fab` suite passed on the final run.
- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant (rename/move-only refactor; the old paths are removed by the same `git mv` operations, leaving nothing orphaned behind).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Group the work as 5 tasks (one per proposal + combined doc sweep + verification) — each a coherent single-session unit | Three approved proposals with fully enumerated file sets; grouping mirrors the report structure | S:90 R:90 A:95 D:90 |
| 2 | Confident | `docs/findings/index.md` row wording/status label decided at apply against the index's existing table shape | Intake marked this Confident; the exact label depends on the table's live column values | S:60 R:95 A:75 D:65 |
| 3 | Certain | No new tests: structure-only change, existing tests move with their files and prove the rename compiles/behaves | code-review.md's "Go changes ship tests" targets behavior changes; here the moved tests ARE the coverage, per the intake | S:85 R:90 A:90 D:85 |

3 assumptions (2 certain, 1 confident, 0 tentative).
