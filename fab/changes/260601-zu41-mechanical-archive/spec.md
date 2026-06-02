# Spec: Make fab archive (single + batch) fully mechanical

**Change**: 260601-zu41-mechanical-archive
**Created**: 2026-06-01
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- **Changing the archive directory layout** — the `archive/yyyy/mm/{name}/` date-bucketing, index format, and pointer-clearing behavior are unchanged. This change only alters *how* the description and backlog steps are computed, not where files land.
- **Restore-path changes** — `fab change restore` and `/fab-archive restore` are untouched.
- **A `## Done` section in the backlog** — items are flipped in place; no section migration.
- **Fuzzy/keyword backlog matching** — deliberately removed, not relocated. Only exact-ID matching survives, in Go.
- **Backfilling richer historical index descriptions** — existing `archive/index.md` entries keep their (agent-summarized) text; only new entries use the mechanical title-derived description.

## Archive: Mechanical Description Derivation

### Requirement: Archive description defaults to the intake title
`fab change archive` SHALL accept an optional `--description`. When `--description` is empty, the command SHALL derive the archive-index description mechanically from the change's `intake.md` title line (`# Intake: {title}`), with the `# Intake: ` prefix removed and internal whitespace collapsed. The derivation SHALL occur before the change folder is moved out of `fab/changes/`.

#### Scenario: Empty description derives from title
- **GIVEN** a change `260601-zu41-mechanical-archive` whose `intake.md` begins with `# Intake: Make fab archive (single + batch) fully mechanical`
- **WHEN** `fab change archive 260601-zu41-mechanical-archive` runs with no `--description`
- **THEN** the `archive/index.md` entry reads `- **260601-zu41-mechanical-archive** — Make fab archive (single + batch) fully mechanical`

#### Scenario: Explicit description overrides the title
- **GIVEN** the same change
- **WHEN** `fab change archive <change> --description "Custom text"` runs
- **THEN** the index entry uses `Custom text`, not the title

### Requirement: Description falls back to the humanized slug
When the intake title cannot be read (file missing, unreadable, or no `# Intake:` heading), `DescriptionFor` SHALL fall back to a humanized slug: the change-folder name with the `YYMMDD-XXXX-` prefix removed and hyphens replaced by spaces.

#### Scenario: Missing intake falls back to slug
- **GIVEN** a change folder `260601-abcd-fix-stale-status` with no readable `intake.md` title
- **WHEN** the change is archived with no `--description`
- **THEN** the index entry description reads `fix stale status`

### Requirement: `intake.Title` is a standalone, reusable function
A new package `internal/intake` SHALL provide `Title(changeDir string) string` returning the de-prefixed intake title (or `""` on any read failure or missing heading) and `DescriptionFor(fabRoot, folder string) string` implementing the title-then-slug preference. `internal/archive` SHALL depend on `internal/intake`; `internal/intake` SHALL NOT depend on `internal/archive`.

#### Scenario: Title with embedded markdown is preserved
- **GIVEN** an `intake.md` whose first line is `` # Intake: Fix stale `fab status` CLI ``
- **WHEN** `Title` reads it
- **THEN** it returns `` Fix stale `fab status` CLI `` verbatim (backticks intact)

## Archive: Mechanical Backlog Marking

### Requirement: Originating backlog item is marked done mechanically
A new package `internal/backlog` SHALL provide `MarkDone(backlogPath, id string) (string, error)` that flips the backlog line `- [ ] [<id>]` to `- [x] [<id>]` in place. It SHALL NOT move the item to any other section. `ArchiveWithBacklog` SHALL extract the 4-char change ID from the resolved folder name (via `resolve.ExtractID`) and call `MarkDone` after a successful archive.

#### Scenario: Exact-ID match flips the checkbox
- **GIVEN** `fab/backlog.md` contains `- [ ] [zu41] 2026-06-01: make archive mechanical`
- **WHEN** a change whose folder is `260601-zu41-mechanical-archive` is archived
- **THEN** that backlog line becomes `- [x] [zu41] 2026-06-01: make archive mechanical`
- **AND** the archive result reports `backlog: marked`

#### Scenario: Already-done item is a no-op
- **GIVEN** the backlog line is already `- [x] [zu41] ...`
- **WHEN** the change is archived
- **THEN** the file is not rewritten and the result reports `backlog: already`

### Requirement: Missing match or missing file is a silent no-op
`MarkDone` SHALL return `not_found` with a nil error when no line matches the ID, including when `fab/backlog.md` does not exist. Archiving SHALL still succeed in these cases.

#### Scenario: Change not from backlog
- **GIVEN** a change whose 4-char ID has no matching `[id]` in `fab/backlog.md`
- **WHEN** it is archived
- **THEN** the archive succeeds and reports `backlog: not_found`

#### Scenario: No backlog file
- **GIVEN** no `fab/backlog.md` exists
- **WHEN** a change is archived
- **THEN** the archive succeeds, reports `backlog: not_found`, and emits no error

### Requirement: Backlog parser is shared, not duplicated
The backlog parsing logic currently embedded in `cmd/fab/batch_new.go` (`package main`) SHALL be extracted into `internal/backlog` as `Item`, `ParsePending`, and `ExtractContent`. `batch_new.go` SHALL be refactored to import and call these instead of holding its own copies. No second copy of the `[a-z0-9]{4}` backlog regex SHALL exist after the change.

#### Scenario: batch new still parses pending items
- **GIVEN** `fab/backlog.md` with several `- [ ] [xxxx]` open items
- **WHEN** `fab batch new --list` runs
- **THEN** it lists the same pending items as before the refactor (behavior preserved via the shared parser)

## Archive: Idempotent Re-Archive (Soft Skip)

### Requirement: Re-archiving an already-archived change is a soft skip
`internal/archive.Archive` SHALL return a distinguishable sentinel `ErrAlreadyArchived` (replacing the ad-hoc "destination already exists" error) when the archive destination already exists. `fab change archive` SHALL treat this as a successful no-op: print an `already archived` note and exit 0.

#### Scenario: Single re-archive exits 0
- **GIVEN** a change already present under `archive/yyyy/mm/`
- **WHEN** `fab change archive <change>` runs again
- **THEN** it prints an `already archived` message and exits 0 (no error, no second move)

## Batch Archive: Pure Go Loop

### Requirement: Batch archive runs in-process without spawning an agent
`fab batch archive` SHALL archive each resolved change by calling `internal/archive.ArchiveWithBacklog` directly in a Go loop. It SHALL NOT build a `/fab-archive` prompt, SHALL NOT spawn a Claude session, and SHALL NOT depend on `internal/spawn`, `os/exec`, or `syscall` for the archive step. Change resolution SHALL use `resolve.ToFolder` rather than shelling out to `fab change resolve`.

#### Scenario: Batch archive does not spawn
- **GIVEN** two changes with `hydrate: done`
- **WHEN** `fab batch archive --all` runs
- **THEN** both folders are moved under `archive/yyyy/mm/` in-process
- **AND** no Claude session is spawned

### Requirement: Per-change failures are isolated; counts are reported
The batch loop SHALL be a testable helper returning `(archived, skipped, failed)` counts and SHALL NOT call `os.Exit` internally. A failure on one change SHALL be reported and SHALL NOT abort the remaining changes. Already-archived changes SHALL be counted as `skipped`, not `failed`. The loop SHALL print a footer `Archived {N}, skipped {M}, failed {K}.`. `fab batch archive` SHALL exit non-zero only when `failed > 0`.

#### Scenario: One failure does not abort the batch
- **GIVEN** three archivable changes where the second errors mid-archive
- **WHEN** `fab batch archive --all` runs
- **THEN** the first and third are archived, the second is reported `FAILED`, and the footer reads `Archived 2, skipped 0, failed 1.`
- **AND** the command exits non-zero

#### Scenario: Re-running a completed batch skips cleanly
- **GIVEN** changes already archived in a prior run
- **WHEN** `fab batch archive --all` runs again
- **THEN** each prints `already archived, skipping`, the footer reports them as `skipped`, and the command exits 0

### Requirement: List mode is unchanged
`fab batch archive --list` SHALL continue to show archivable changes (`hydrate: done|skipped`) without archiving anything.

#### Scenario: List archives nothing
- **GIVEN** archivable changes exist
- **WHEN** `fab batch archive --list` runs
- **THEN** it prints the archivable names and moves no folders

## Archive Result & YAML Contract

### Requirement: Archive YAML carries a backlog field
`ArchiveResult` SHALL include a `Backlog` field, and `FormatArchiveYAML` SHALL emit a `backlog: {marked|already|not_found}` line in addition to the existing `action`, `name`, `move`, `index`, and `pointer` lines.

#### Scenario: YAML includes backlog status
- **GIVEN** any successful archive
- **WHEN** the result is formatted
- **THEN** the YAML output contains a `backlog:` line with one of `marked`, `already`, or `not_found`

## Skill: `/fab-archive` Becomes a Thin Wrapper

### Requirement: The skill delegates all mechanics and only formats output
The `/fab-archive` skill SHALL remove the agent-driven description-extraction step and the agent-driven backlog-matching steps (exact-ID, fuzzy keyword scan, interactive confirm). Archive Mode SHALL reduce to: preflight → hydrate guard → `fab change archive <change>` (no `--description`) → format the YAML report. The skill SHALL map the `backlog:` YAML field to a report line (`marked` → `✓ [ID] marked done`, `already` → `— already done`, `not_found` → `— no match`) and SHALL NOT use the `Edit` tool.

#### Scenario: No interactive prompt during archive
- **GIVEN** a hydrated change
- **WHEN** `/fab-archive` runs on it
- **THEN** it performs no fuzzy keyword scan and asks no interactive confirmation question
- **AND** the `Backlog:` report line is sourced from the CLI's `backlog:` YAML field

## Documentation & Memory Consistency

### Requirement: Specs and memory reflect the mechanical archive
Because skill behavior changes, the corresponding spec (`docs/specs/skills/SPEC-fab-archive.md`) SHALL be updated, and the inaccurate "tmux tab per change" descriptions of `fab batch archive` in `docs/specs/overview.md` and `docs/specs/architecture.md` SHALL be corrected. `docs/memory/fab-workflow/kit-architecture.md` SHALL describe `fab batch archive` as a mechanical Go loop, scope `internal/spawn` and the batch tmux/spawn shared-pattern note to `new`/`switch` only, and list the new `internal/intake` and `internal/backlog` packages. `src/kit/skills/_cli-fab.md` SHALL document the new `fab batch archive` behavior and the optional `--description` / `backlog` field for `fab change archive`.

#### Scenario: Memory no longer claims batch archive spawns a session
- **GIVEN** the hydrate stage completes
- **WHEN** `docs/memory/fab-workflow/kit-architecture.md` is read
- **THEN** the `fab batch archive` bullet describes a mechanical Go loop with no spawned Claude session
- **AND** `internal/intake` and `internal/backlog` appear in the package and tested-packages lists

## Design Decisions

1. **Description source = intake title line (not first-sentence-of-Why)**: 
   - *Why*: The `## Why` section is heavily formatted prose (`**Problem**:`, numbered lists), so mechanical first-sentence extraction would capture markdown noise. The `# Intake: {title}` line is a single clean, structured string.
   - *Rejected*: First sentence of Why (markdown noise); raw slug as primary (terser, loses intent — used only as fallback).

2. **Keep `Archive()` pure; add `ArchiveWithBacklog()` orchestrator**:
   - *Why*: `Archive()` already owns move/index/pointer with no knowledge of backlog or intake-title. Bundling the backlog side-effect into it would couple unrelated concerns. A thin orchestrator isolates the side-effect and keeps `Archive()` independently testable.
   - *Rejected*: Adding backlog marking directly inside `Archive()` (couples filesystem move with backlog mutation).

3. **`ErrAlreadyArchived` sentinel for soft skip**:
   - *Why*: Callers (single CLI exit-0, batch `skipped` tally) must distinguish "already archived" from real failures. A typed sentinel via `errors.Is` is the idiomatic Go way; the prior ad-hoc string error couldn't be matched reliably.
   - *Rejected*: String-matching the error message (brittle); silently treating all errors as skips (hides real failures).

4. **Extract shared `internal/backlog` package**:
   - *Why*: The parser lives in `batch_new.go` as `package main` (unimportable). Reuse-over-duplication (constitution + code-quality.md anti-pattern: "Duplicating existing utilities").
   - *Rejected*: Copying the `[a-z0-9]{4}` regex into `internal/archive` (duplication, drift risk).

5. **Flip backlog item in place (no `## Done` section)**:
   - *Why*: The real `fab/backlog.md` has only `## Open`; completed items are `- [x]` interleaved in place. Moving to a `## Done` section would invent structure the project doesn't use.
   - *Rejected*: Moving marked items to a Done section.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Archive description derived from intake **title line**, de-prefixed, humanized-slug fallback | Confirmed from intake #1 — user chose this explicitly via structured question; the title is the clean structured source vs. formatted Why prose | S:95 R:80 A:90 D:95 |
| 2 | Certain | Backlog matching = exact-ID only in Go; fuzzy keyword scan + interactive confirm removed | Confirmed from intake #2 — folder 4-char ID *is* the backlog ID, deterministic string op | S:95 R:75 A:90 D:90 |
| 3 | Certain | `fab batch archive` = pure in-process Go loop; no spawn/tmux/exec | Confirmed from intake #3 — archiving is non-interactive file mechanics | S:95 R:70 A:95 D:95 |
| 4 | Certain | Extract shared `internal/backlog`; refactor `batch_new.go` to use it | Confirmed from intake #4 — reuse over duplication; parser currently in unimportable `package main` | S:90 R:70 A:90 D:90 |
| 5 | Certain | Already-archived = soft skip (exit 0 single; `skipped` count in batch) via `ErrAlreadyArchived` | Confirmed from intake #5 — constitution §III Idempotent Operations | S:90 R:80 A:90 D:90 |
| 6 | Certain | Keep `Archive()` pure; add `ArchiveWithBacklog()` orchestrator | Upgraded from intake #6 (Confident → Certain) — spec-level analysis confirms clean separation; both CLI and batch call the orchestrator, `Archive()` stays move/index/pointer | S:90 R:80 A:90 D:85 |
| 7 | Certain | `MarkDone` flips `[ ]`→`[x]` in place, never moving to `## Done` | Upgraded from intake #7 — verified against real backlog (no Done section exists) | S:90 R:85 A:90 D:90 |
| 8 | Certain | Change type = `refactor` | Confirmed from intake #8 — mechanizing existing behavior, no new user-facing capability; matches taxonomy (extract module, no behavior change to the *outcome* of archiving) | S:80 R:90 A:90 D:85 |
| 9 | Certain | `internal/intake` is a standalone package, not folded into `internal/archive` | Upgraded from intake #9 — keeps `archive` focused; `Title`/`DescriptionFor` independently testable and reusable | S:80 R:85 A:85 D:80 |
| 10 | Confident | `archiveLoop` is a testable helper returning `(archived, skipped, failed)`; `os.Exit` stays in `runBatchArchive` | New (spec-level) — separating pure logic from process exit is the standard testability pattern in this repo (mirrors how other cmd helpers are structured); enables `TestArchiveLoop` without subprocess | S:80 R:85 A:80 D:80 |
| 11 | Confident | Output format for `fab batch archive` is per-change lines + `Archived N, skipped M, failed K.` footer | New (spec-level) — no prior format to preserve (old path delegated to the agent); this is the natural mechanical reporting shape, low blast radius, easily adjusted | S:70 R:85 A:75 D:75 |

11 assumptions (9 certain, 2 confident, 0 tentative, 0 unresolved).
