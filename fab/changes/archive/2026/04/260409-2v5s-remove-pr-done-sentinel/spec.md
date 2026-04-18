# Spec: Remove .pr-done Sentinel

**Change**: 260409-2v5s-remove-pr-done-sentinel
**Created**: 2026-04-09
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Skill: `/git-pr` — Remove Step 4d

### Requirement: Remove PR Sentinel Write
The `/git-pr` skill SHALL NOT write a `.pr-done` sentinel file. Step 4d ("Write PR Sentinel") SHALL be removed entirely. The step numbering SHALL be updated: Step 4c remains the final post-PR step. The PR URL recorded via `fab status add-pr` in Step 4a is the sole mechanism for determining whether a PR has been created.

#### Scenario: PR creation without sentinel
- **GIVEN** a change with an active ship stage
- **WHEN** `/git-pr` creates a PR successfully
- **THEN** Steps 4a (record PR URL), 4b (finish ship stage), and 4c (commit+push status) execute as before
- **AND** no `.pr-done` file is created in the change directory

#### Scenario: Existing .pr-done files are unaffected
- **GIVEN** a change directory that already contains a `.pr-done` file from a prior run
- **WHEN** `/git-pr` is re-run
- **THEN** the existing `.pr-done` file is neither read nor deleted by `/git-pr`

## Skill: `fab change archive` — Remove Clean Step

### Requirement: Remove .pr-done Cleanup from Archive
The `Archive()` function in `src/go/fab/internal/archive/archive.go` SHALL NOT check for or delete `.pr-done` files. The `Clean` field SHALL be removed from the `ArchiveResult` struct. The `FormatArchiveYAML` function SHALL NOT emit a `clean:` field.

#### Scenario: Archive without clean step
- **GIVEN** a change directory (with or without a `.pr-done` file)
- **WHEN** `fab change archive <change> --description "..."` is run
- **THEN** the archive operation moves the folder, updates the index, and clears the pointer
- **AND** no `.pr-done` check or deletion occurs
- **AND** the YAML output contains `action`, `name`, `move`, `index`, `pointer` fields only (no `clean` field)

### Requirement: Update Archive Tests
The `archive_test.go` tests SHALL NOT reference `.pr-done` or the `Clean` field. The `FormatArchiveYAML` test SHALL validate the updated output format without the `clean:` field.

#### Scenario: FormatArchiveYAML output
- **GIVEN** an `ArchiveResult` with `Action: "archive"`, `Name: "260310-abcd-my-change"`, `Move: "moved"`, `Index: "created"`, `Pointer: "cleared"`
- **WHEN** `FormatArchiveYAML` is called
- **THEN** the output contains exactly: `action`, `name`, `move`, `index`, `pointer` fields
- **AND** does not contain a `clean:` field

## Skill: `/fab-archive` — Remove Clean References

### Requirement: Remove Clean Step from Skill Description
The `/fab-archive` skill description SHALL NOT mention `.pr-done` cleanup. The "Clean" bullet under Step 2 SHALL be removed. The report format table SHALL NOT include `clean: removed` or `clean: not_present` rows. The output example SHALL NOT include a `Cleaned:` line.

#### Scenario: Archive skill report format
- **GIVEN** a successful archive operation
- **WHEN** the skill formats the report from YAML output
- **THEN** the report shows `Moved:`, `Index:`, `Backlog:`, `Scan:`, and `Pointer:` lines
- **AND** no `Cleaned:` line appears

## Docs: `_cli-fab.md` — Update Archive Description

### Requirement: Update Archive Command Row
The `archive` command description in the `fab change` table SHALL read "Move to archive/, update index, clear pointer" (removing "Clean .pr-done, " prefix).

#### Scenario: CLI reference accuracy
- **GIVEN** the `_cli-fab.md` command reference
- **WHEN** a reader looks up the `archive` subcommand
- **THEN** the description reads "Move to archive/, update index, clear pointer"

## Docs: Spec Diagrams — Update Flow Charts

### Requirement: Update SPEC-git-pr Flow Diagram
The `SPEC-git-pr.md` flow diagram SHALL NOT include Step 4d. The flow SHALL end at Step 4c.

#### Scenario: git-pr flow diagram accuracy
- **GIVEN** the `docs/specs/skills/SPEC-git-pr.md` flow diagram
- **WHEN** a reader traces the post-PR steps
- **THEN** the flow shows 4a → 4b → 4c as the final steps with no 4d

### Requirement: Update SPEC-fab-archive Flow Diagram
The `SPEC-fab-archive.md` flow diagram SHALL NOT reference `.pr-done` cleanup. The archive step description SHALL mention only move, index update, and pointer clearing.

#### Scenario: fab-archive flow diagram accuracy
- **GIVEN** the `docs/specs/skills/SPEC-fab-archive.md` flow diagram
- **WHEN** a reader traces the archive flow
- **THEN** no reference to `.pr-done` appears

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `.status.yaml` prs field is the authoritative PR record | Confirmed from intake #1 — `fab status add-pr` writes in step 4a, before `.pr-done` in step 4d | S:95 R:90 A:95 D:95 |
| 2 | Certain | No migration needed | Confirmed from intake #2 — 0.46.0-to-1.1.0 migration already cleans up `.pr-done` files and `.gitignore` entries | S:90 R:95 A:95 D:95 |
| 3 | Certain | Remove `Clean` field from `ArchiveResult` struct entirely | Confirmed from intake #3 — no downstream consumers parse the `clean` YAML field | S:85 R:85 A:90 D:90 |
| 4 | Certain | Historical changelog entries preserved | Confirmed from intake #4 — constitution says memory records what happened | S:90 R:95 A:90 D:95 |
| 5 | Certain | Archive tests need `Clean` field removal | Upgraded from intake Confident #5 — verified `archive_test.go` references `Clean` in `FormatArchiveYAML` test | S:90 R:85 A:90 D:85 |
| 6 | Certain | `fab-archive` Purpose paragraph needs update | Purpose mentions "clean" — removing reference to match new behavior | S:85 R:90 A:90 D:95 |

6 assumptions (6 certain, 0 confident, 0 tentative, 0 unresolved).