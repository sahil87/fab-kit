# Intake: Rename Scaffold & Add Kit Script Tests

**Change**: 260216-b1k9-DEV-1028-rename-scaffold-add-kit-tests
**Created**: 2026-02-16
**Status**: Draft

## Origin

> Three-part request from active development session:
> 1. Rename `init-scaffold.sh` to `sync-workspace.sh` to better reflect its idempotent, convergence-oriented behavior
> 2. Add test suites and SPEC files for `sync-workspace.sh` and `changeman.sh`, following the established pattern in `src/lib/` (symlink to distributed script, SPEC-*.md, test.sh)
> 3. Improve the `just test` runner so the output clearly shows a pass/fail summary across all suites

The rename emerged from reviewing `init-scaffold.sh` responsibilities — "scaffold" implies one-time skeleton creation, but the script is idempotent and sync-oriented (skill distribution, agent wiring, gitignore convergence). "sync-workspace" communicates the convergence semantics accurately.

## Why

1. **Naming accuracy**: `init-scaffold.sh` is misleading — it's not a one-time scaffolder. It syncs kit assets (skills, agents, gitignore entries, envrc, version tracking) into the workspace and is explicitly safe to re-run. `sync-workspace.sh` communicates this.

2. **Test coverage gap**: `changeman.sh` and `init-scaffold.sh` (soon `sync-workspace.sh`) are the only two scripts in `fab/.kit/scripts/lib/` without test suites or SPEC files. The other four scripts (`preflight.sh`, `resolve-change.sh`, `stageman.sh`, `calc-score.sh`) all have `src/lib/{name}/` directories with comprehensive tests. This gap means regressions in change creation and workspace setup go undetected.

3. **Test runner UX**: The current `just test` output dumps individual test results but provides no aggregate summary. When 130+ tests run across 4 (soon 6) suites, it's hard to tell at a glance whether everything passed. A failed suite buried in the middle can be missed. The runner needs a summary line like `6/6 suites passed` or `FAIL: 2/6 suites failed (stageman, changeman)`.

## What Changes

### 1. Rename `init-scaffold.sh` → `sync-workspace.sh`

- Rename the file at `fab/.kit/scripts/lib/init-scaffold.sh` → `fab/.kit/scripts/lib/sync-workspace.sh`
- Update all internal references:
  - The comment header and `#` description line in the script itself
  - `changeman.sh` line 128 references `init-scaffold.sh` in a comment — update to `sync-workspace.sh`
  - Any other scripts, skills, or memory files that reference the old name
- Update docs/memory files that mention `init-scaffold.sh`

### 2. Adopt bats-core as bash testing framework

Install/vendor bats-core as a dev dependency. New tests (changeman, sync-workspace) will be written as `.bats` files. Existing hand-rolled test.sh files remain untouched — migration to bats is a separate follow-up (DEV-1029).

### 3. Add `src/lib/sync-workspace/` test directory

Following the established pattern, but with bats for the test file:

```
src/lib/sync-workspace/
  sync-workspace.sh → ../../../fab/.kit/scripts/lib/sync-workspace.sh  (symlink)
  SPEC-sync-workspace.md   (spec file documenting behavior, usage, sources of truth)
  test.bats                (bats-core test suite)
```

The SPEC file should follow the format of existing SPECs (e.g., `SPEC-stageman.md`): Sources of Truth, Usage, API/Behavior Reference, Requirements, Testing section.

The test suite should cover:
- Directory creation (fab/changes, docs/memory, docs/specs)
- VERSION file logic (new project vs existing project with config.yaml)
- .envrc symlink creation and repair
- Memory/specs index seeding
- Skill sync across all three platforms (Claude Code, OpenCode, Codex)
- Model-tier agent generation for fast-tier skills
- .gitignore entry management (creation, dedup, append)
- Idempotency (running twice produces same result)

### 4. Add `src/lib/changeman/` test directory

```
src/lib/changeman/
  changeman.sh → ../../../fab/.kit/scripts/lib/changeman.sh  (symlink)
  SPEC-changeman.md   (spec file)
  test.bats            (bats-core test suite)
```

The test suite should cover:
- `new` subcommand: slug validation, change-id validation, folder creation, .status.yaml initialization
- Random ID generation and collision detection
- `--help` flag
- Error cases: missing slug, invalid slug format, invalid change-id, duplicate change-id, unknown flags
- `detect_created_by` fallback chain (gh → git → "unknown")
- Stageman integration (set-state and log-command called correctly)

### 5. Restructure `just test` with two-tier runner and summary

Replace the current monolithic `just test` with a two-tier structure:

- **`just test-bash`** — runs both bats `.bats` files (new tests) and legacy `test.sh` files (existing tests) until DEV-1029 completes the migration
- **`just test-rust`** — placeholder/no-op for now; will run `cargo test` once Rust libs exist
- **`just test`** — runs both, with a combined summary showing per-suite pass/fail and an overall verdict:

```
── bash (bats) ──────────  sync-workspace, changeman     PASS
── bash (legacy) ────────  preflight, resolve-change, stageman, calc-score     FAIL (stageman)
═══════════════════════════════════════════════════
5/6 suites passed, 1 failed     FAIL
```

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update reference from `init-scaffold.sh` to `sync-workspace.sh`
- `fab-workflow/distribution`: (modify) Update script name reference

## Impact

- **Scripts**: `fab/.kit/scripts/lib/init-scaffold.sh` renamed; `changeman.sh` comment updated
- **Build**: `justfile` test recipe enhanced
- **Dev tooling**: Two new test directories in `src/lib/`
- **Docs/memory**: References to old name updated throughout

## Open Questions

- (None — scope is well-defined)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New name is `sync-workspace.sh` | Explicitly discussed and agreed in conversation | S:95 R:90 A:95 D:95 |
| 2 | Certain | Test structure follows existing `src/lib/*/` pattern (symlink + SPEC) | All 4 existing scripts use this layout | S:90 R:95 A:95 D:95 |
| 3 | Certain | SPEC files follow existing format (SPEC-stageman.md as reference) | Explicit in the description | S:85 R:90 A:90 D:95 |
| 4 | Certain | New tests use bats-core (.bats files), existing tests unchanged | Discussed and agreed — bats for new, migrate existing in DEV-1029 | S:95 R:90 A:90 D:95 |
| 5 | Certain | Two-tier justfile: test-bash (bats + legacy), test-rust (placeholder), test (both + summary) | Discussed and agreed — Rust libs coming soon, need separate runners | S:95 R:85 A:90 D:90 |
| 6 | Confident | Summary output format is suite-level pass/fail with totals and overall verdict | User described the problem (no summary); specific format is reasonable inference | S:70 R:90 A:80 D:70 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
