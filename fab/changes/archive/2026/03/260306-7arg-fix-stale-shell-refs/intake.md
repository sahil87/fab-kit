# Intake: Fix stale shell script references

**Change**: 260306-7arg-fix-stale-shell-refs
**Created**: 2026-03-06
**Status**: Draft

## Origin

> Delete 9 orphaned bats test files (shell scripts were removed, Go binary is the sole backend now), add 4 missing status subcommands (add-issue, get-issues, add-pr, get-prs) to _scripts.md, and fix git-pr.md Step 4 to pass `<change>` instead of `<status_file>` path.

Discussion-driven: identified during `/fab-discuss` session. The screenshot showed `fab status add-pr` being called with a full `.status.yaml` path, which led to auditing `_scripts.md` and discovering the orphaned test files.

## Why

The shell scripts (`statusman.sh`, `changeman.sh`, `logman.sh`, `archiveman.sh`, `resolve.sh`, `preflight.sh`, `calc-score.sh`, sync scripts) were removed when the Go binary became the sole backend. Three categories of stale references remain:

1. **9 orphaned bats test files** — test suites for deleted shell scripts, ~2000+ lines of dead code that could confuse contributors and waste CI time if test runners scan for `.bats` files.
2. **`_scripts.md` missing 4 subcommands** — `add-issue`, `get-issues`, `add-pr`, `get-prs` are implemented in Go and used by skills (`/fab-new`, `/git-pr`) but not documented in the authoritative script invocation guide. Skills consulting `_scripts.md` won't know these exist.
3. **`git-pr.md` wrong signature** — Step 4 (lines 229-231) constructs a `.status.yaml` path and passes it to `fab status add-pr`, but the command expects a `<change>` argument (ID or folder name). This works today only because the Go binary's `loadStatus()` happens to accept paths, but it violates the documented `<change>` convention and will break if path acceptance is tightened.

## What Changes

### Delete orphaned shell test infrastructure

The entire `src/lib/` tree and `src/sync/` directory contain artifacts from the removed shell scripts. Delete the following:

**Bats test files** (9 files):
- `src/lib/statusman/test.bats`
- `src/lib/changeman/test.bats`
- `src/lib/logman/test.bats`
- `src/lib/archiveman/test.bats`
- `src/lib/resolve/test.bats`
- `src/lib/preflight/test.bats`
- `src/lib/calc-score/test.bats`
- `src/lib/sync-workspace/test.bats`
- `src/sync/test-5-sync-hooks.bats`

**Also stale in `src/lib/*/`** — SPEC files, test-simple scripts, and tooling for the removed shell scripts:
- `src/lib/statusman/SPEC-statusman.md`, `test-simple.sh`
- `src/lib/changeman/SPEC-changeman.md`
- `src/lib/logman/test-simple.sh`
- `src/lib/resolve/test-simple.sh`
- `src/lib/preflight/SPEC-preflight.md`, `test-simple.sh`
- `src/lib/calc-score/SPEC-calc-score.md`, `test-simple.sh`, `sensitivity.sh`
- `src/lib/sync-workspace/SPEC-sync-workspace.md`

After file deletion, remove the now-empty directories: all 8 `src/lib/*/` subdirs, the `src/lib/` parent, and `src/sync/`.

### Fix `just test` pipeline

`just test` currently fails because `test-bash.sh` globs `src/lib/*/test.bats` and `src/sync/test-*.bats`, finding the orphaned tests that reference deleted shell scripts.

After deleting the orphaned files:
- `test-bash.sh` (`src/scripts/just/test-bash.sh`) still globs `src/hooks/test-*.bats` (2 valid hook tests remain) — no change needed, glob handles empty results gracefully
- `test-parallel.sh` (`src/scripts/just/test-parallel.sh`) still runs `test-bash`, `test-packages`, `test-scripts` in parallel — still valid
- Justfile `test-rust` recipe is a no-op placeholder — leave as-is
- Verify `just test` passes after cleanup

### Add missing subcommands to `_scripts.md`

Add 4 rows to the `fab status` "Key subcommands" table in `fab/.kit/skills/_scripts.md`:

| Subcommand | Usage | Purpose |
|------------|-------|---------|
| `add-issue` | `add-issue <change> <id>` | Append issue ID to issues array (idempotent) |
| `get-issues` | `get-issues <change>` | List issue IDs (one per line) |
| `add-pr` | `add-pr <change> <url>` | Append PR URL to prs array (idempotent) |
| `get-prs` | `get-prs <change>` | List PR URLs (one per line) |

Place them after the `set-confidence-fuzzy` row and before `progress-line`, grouped as "Issue/PR metadata" operations.

### Fix `git-pr.md` Step 4 signature

In `fab/.kit/skills/git-pr.md`, simplify Step 4 (lines 227-234) to pass the resolved change name directly instead of constructing a `.status.yaml` path:

**Before** (current):
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

The intermediate `.status.yaml` path derivation is unnecessary — `fab status` resolves `<change>` internally.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update to reflect shell test files removed, note Go parity tests as sole test infrastructure
- `fab-workflow/execution-skills`: (modify) Update `/git-pr` `add-pr` call signature if documented there

## Impact

- **`fab/.kit/skills/_scripts.md`** — 4 new rows in status subcommands table
- **`fab/.kit/skills/git-pr.md`** — Step 4 simplified (fewer lines, correct convention)
- **`src/lib/`** — entire directory tree deleted (~20 files across 8 subdirs)
- **`src/sync/`** — directory deleted (sole file was orphaned test)
- **`just test`** — now passes (no more failing shell test suites)
- **Go parity tests** — NOT affected (they live in `src/go/fab/test/parity/` and test the Go binary directly)

### `.gitmodules` — no change needed

The 4 bats submodules (`src/packages/tests/libs/bats*`) are still required by:
- `src/hooks/test-on-session-start.bats`, `test-on-stop.bats` (2 active hook tests)
- `src/packages/idea/tests/*.bats` (package tests)

These remain valid consumers. No reduction possible.

## Open Questions

None — all decisions made during discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Delete all 9 listed bats test files | Discussed — user confirmed shell scripts removed, Go is sole backend | S:95 R:90 A:95 D:95 |
| 2 | Certain | Add only 4 subcommands (add-issue, get-issues, add-pr, get-prs), not read-only query commands | Discussed — user explicitly said "Don't add the read-only subcommands that aren't needed" | S:95 R:95 A:90 D:95 |
| 3 | Certain | Fix git-pr.md to pass `<change>` not `<status_file>` | Discussed — confirmed against Go source (`status.go:462`) and shell help text (`statusman.sh:1077`) | S:95 R:85 A:95 D:95 |
| 4 | Certain | Delete entire `src/lib/` tree including SPEC files, test-simple.sh, sensitivity.sh | Discussed — user said "stray test cases all need to be deleted"; all files in these dirs are shell-script artifacts | S:90 R:85 A:90 D:90 |
| 5 | Certain | Verify `just test` passes after cleanup | Discussed — user asked to "check and fix the output of just test" | S:95 R:95 A:90 D:95 |
| 6 | Confident | Place new subcommands after set-confidence-fuzzy row | Logical grouping — metadata writes together, then query commands | S:65 R:95 A:80 D:75 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
