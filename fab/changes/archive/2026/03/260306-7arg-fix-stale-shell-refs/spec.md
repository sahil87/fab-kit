# Spec: Fix stale shell script references

**Change**: 260306-7arg-fix-stale-shell-refs
**Created**: 2026-03-06
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Modifying Go parity tests in `src/fab-go/test/parity/` — they test the Go binary directly and are unrelated to the removed shell scripts
- Removing bats submodules from `src/packages/tests/libs/bats*` — still used by active hook tests and package tests
- Modifying `test-bash.sh` or `test-parallel.sh` — they handle empty globs gracefully after the deletions

## Shell Test Infrastructure Cleanup

### Requirement: Remove orphaned shell test files

All files in `src/lib/` and `src/sync/test-5-sync-hooks.bats` SHALL be deleted. These are test artifacts for removed shell scripts (statusman.sh, changeman.sh, logman.sh, archiveman.sh, resolve.sh, preflight.sh, calc-score.sh, sync-workspace.sh) and have no valid test targets.

The following files MUST be deleted:

**Bats test files (9)**:
- `src/lib/statusman/test.bats`
- `src/lib/changeman/test.bats`
- `src/lib/logman/test.bats`
- `src/lib/archiveman/test.bats`
- `src/lib/resolve/test.bats`
- `src/lib/preflight/test.bats`
- `src/lib/calc-score/test.bats`
- `src/lib/sync-workspace/test.bats`
- `src/sync/test-5-sync-hooks.bats`

**SPEC files (5)**:
- `src/lib/statusman/SPEC-statusman.md`
- `src/lib/changeman/SPEC-changeman.md`
- `src/lib/preflight/SPEC-preflight.md`
- `src/lib/calc-score/SPEC-calc-score.md`
- `src/lib/sync-workspace/SPEC-sync-workspace.md`

**Helper scripts (5)**:
- `src/lib/statusman/test-simple.sh`
- `src/lib/logman/test-simple.sh`
- `src/lib/resolve/test-simple.sh`
- `src/lib/preflight/test-simple.sh`
- `src/lib/calc-score/test-simple.sh`

**Utility script (1)**:
- `src/lib/calc-score/sensitivity.sh`

#### Scenario: All orphaned files deleted

- **GIVEN** the shell scripts (statusman.sh, changeman.sh, etc.) have been removed and the Go binary is the sole backend
- **WHEN** this change is applied
- **THEN** all 20 files listed above are deleted from the repository
- **AND** no files remain in `src/lib/` or `src/sync/`

### Requirement: Remove empty directories

After file deletion, the following directories SHALL be removed: `src/lib/statusman/`, `src/lib/changeman/`, `src/lib/logman/`, `src/lib/archiveman/`, `src/lib/resolve/`, `src/lib/preflight/`, `src/lib/calc-score/`, `src/lib/sync-workspace/`, `src/lib/`, and `src/sync/`.

#### Scenario: Empty directories cleaned up

- **GIVEN** all files in `src/lib/` and `src/sync/` have been deleted
- **WHEN** directory cleanup runs
- **THEN** all 8 subdirectories of `src/lib/`, the `src/lib/` parent directory, and the `src/sync/` directory are removed
- **AND** `src/` still exists (it contains other directories like `src/fab-go/`, `src/hooks/`, etc.)

### Requirement: Test pipeline passes after cleanup

`just test` SHALL pass after the orphaned files are deleted. The test runner (`src/scripts/just/test-bash.sh`) globs `src/lib/*/test.bats` and `src/sync/test-*.bats` — with the files removed, these globs return empty results, which the script handles gracefully.

#### Scenario: just test passes

- **GIVEN** all orphaned test files have been deleted
- **WHEN** `just test` is executed
- **THEN** all test suites pass
- **AND** the remaining hook tests (`src/hooks/test-on-session-start.bats`, `test-on-stop.bats`) still run and pass

## Script Invocation Guide Updates

### Requirement: Document issue/PR metadata subcommands

`fab/.kit/skills/_scripts.md` SHALL include 4 new rows in the `fab status` "Key subcommands" table for issue and PR metadata operations:

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `add-issue` | `add-issue <change> <id>` | Append issue ID to issues array (idempotent) |
| `get-issues` | `get-issues <change>` | List issue IDs (one per line) |
| `add-pr` | `add-pr <change> <url>` | Append PR URL to prs array (idempotent) |
| `get-prs` | `get-prs <change>` | List PR URLs (one per line) |

These rows SHALL be placed after the `set-confidence-fuzzy` row and before `progress-line`, maintaining logical grouping (metadata write/query operations together).

#### Scenario: New subcommands appear in table

- **GIVEN** the current `_scripts.md` has no rows for `add-issue`, `get-issues`, `add-pr`, or `get-prs`
- **WHEN** this change is applied
- **THEN** the 4 new rows appear in the `fab status` Key subcommands table
- **AND** they are placed between `set-confidence-fuzzy` and `progress-line`

## git-pr Skill Fix

### Requirement: Pass change reference instead of status file path

In `fab/.kit/skills/git-pr.md` Step 4, the `add-pr` call SHALL pass a `<change>` reference (resolved folder name) instead of constructing a `.status.yaml` path. The intermediate path derivation step SHALL be removed.

**Before** (current lines 229-231):
```
1. Resolve the active change: `fab/.kit/bin/fab change resolve 2>/dev/null`
2. If resolution succeeds (exit 0), derive the status file path: `fab/changes/{name}/.status.yaml`
3. Call: `fab/.kit/bin/fab status add-pr <status_file> <pr_url>`
```

**After** (fixed):
```
1. Resolve the active change: `fab/.kit/bin/fab change resolve 2>/dev/null`
2. If resolution succeeds (exit 0), call: `fab/.kit/bin/fab status add-pr <name> <pr_url>`
```

#### Scenario: add-pr uses change reference

- **GIVEN** `git-pr.md` Step 4 currently derives a `.status.yaml` path and passes it to `add-pr`
- **WHEN** this change is applied
- **THEN** Step 4 passes the resolved change name directly to `fab status add-pr`
- **AND** the intermediate "derive the status file path" step is removed
- **AND** the step count in Step 4 goes from 4 items to 3 items

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Delete all 20 listed files in src/lib/ and src/sync/ | Confirmed from intake #1 — user verified shell scripts removed, Go is sole backend | S:95 R:90 A:95 D:95 |
| 2 | Certain | Add exactly 4 subcommands (add-issue, get-issues, add-pr, get-prs) | Confirmed from intake #2 — user explicitly excluded read-only query commands | S:95 R:95 A:90 D:95 |
| 3 | Certain | Fix git-pr.md to pass `<change>` not `<status_file>` | Confirmed from intake #3 — verified against Go source | S:95 R:85 A:95 D:95 |
| 4 | Certain | Delete entire src/lib/ tree including SPECs, test-simple.sh, sensitivity.sh | Confirmed from intake #4 — all files are shell-script artifacts | S:90 R:85 A:90 D:90 |
| 5 | Certain | Verify just test passes after cleanup | Confirmed from intake #5 — user explicitly requested | S:95 R:95 A:90 D:95 |
| 6 | Confident | Place new subcommands after set-confidence-fuzzy row | Confirmed from intake #6 — logical grouping of metadata operations | S:65 R:95 A:80 D:75 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
