# Intake: Add fab-doctor.sh

**Change**: 260223-sr3u-add-fab-doctor
**Created**: 2026-02-23
**Status**: Draft

## Origin

> User requested a `fab-doctor.sh` script that validates the system has all required tools for fab-kit. Design was iterated in a `/fab-discuss` session covering location, tool list, output format, and direnv hook detection. User provided a concrete code snippet for direnv shell integration checking. Scope expanded in follow-up discussion to absorb `jq` and `bats` into doctor, sunset `sync/1-prerequisites.sh`, and improve `fab-upgrade.sh` post-upgrade output.

Interaction mode: conversational (multiple rounds of design refinement).

Key decisions from discussion:
- All checks required — no optional/soft distinction
- Accumulate all failures before reporting (not fail-fast)
- User provided specific direnv hook detection approach using interactive subshell
- `jq` and `bats` absorbed into doctor — `sync/1-prerequisites.sh` sunset
- `fab-upgrade.sh` migration reminder redesigned — prominent, final, actionable

## Why

There is no standalone diagnostic that validates the full fab-kit toolchain before use. The existing `sync/1-prerequisites.sh` checks tool presence via `command -v` but:

1. It runs only as part of the `fab-sync.sh` pipeline — not invocable standalone
2. It does not check versions (yq v3 vs v4 is a breaking syntax change)
3. It does not verify direnv shell hook integration (a common setup gap where direnv is installed but the hook is not wired into the shell rc file)
4. It does not check for `git` or `bash` (assumed present)
5. It provides minimal error output — a single line listing missing tools

Without this, users hit cryptic failures deep in the workflow (e.g., yq v3 parse errors in `stageman.sh`, missing direnv activation silently skipping `.envrc`).

Separately, `fab-upgrade.sh` buries the "run `/fab-setup migrations`" reminder in the middle of verbose sync output. Users miss it and don't run migrations after upgrading.

## What Changes

### New script: `fab/.kit/scripts/fab-doctor.sh`

A standalone prerequisite checker, sibling of `fab-help.sh`. Checks all 7 required tools:

1. **`git`** — present on PATH
2. **`bash`** — present on PATH
3. **`yq`** — present on PATH, version 4+ (Mike Farah Go binary)
4. **`jq`** — present on PATH
5. **`gh`** — present on PATH (GitHub CLI)
6. **`bats`** — present on PATH (test runner)
7. **`direnv`** — present on PATH AND shell hook active

Each check:
- Prints a checkmark line on success: `  ✓ {tool} {version}`
- Prints a cross line on failure: `  ✗ {tool} — {specific problem}`
- Prints an actionable fix hint on failure (install command, rc file line to add, etc.)

All failures are accumulated — the script runs all checks before exiting. Exit code = number of failed checks (0 = all pass).

Example output (all pass):
```
fab-doctor: checking prerequisites...
  ✓ git 2.43.0
  ✓ bash 5.2.15
  ✓ yq 4.35.1
  ✓ jq 1.7.1
  ✓ gh 2.40.1
  ✓ bats 1.11.0
  ✓ direnv 2.34.0 (zsh hook active)

7/7 checks passed.
```

Example output (failures):
```
fab-doctor: checking prerequisites...
  ✓ git 2.43.0
  ✓ bash 5.2.15
  ✗ yq — not found
    Install: brew install yq
  ✓ jq 1.7.1
  ✓ gh 2.40.1
  ✓ bats 1.11.0
  ✗ direnv shell hook not detected for zsh
    Add the following to your ~/.zshrc (or equivalent):
      eval "$(direnv hook zsh)"

5/7 checks passed. 2 issues found.
```

### Direnv shell hook detection

The direnv check has two parts:

1. **Binary presence**: `command -v direnv`
2. **Shell hook integration**: Spawn an interactive subshell to source rc files, then check for the hook function:
   - **zsh**: Check for `_direnv_hook` function via `zsh -i -c 'typeset -f _direnv_hook'`
   - **bash**: Check `PROMPT_COMMAND` contains "direnv" via `bash -i -c '[[ "${PROMPT_COMMAND:-}" == *direnv* ]]'`
   - Shell detected from `$SHELL` basename

### yq version check

yq v3 (Python) and v4 (Go) have incompatible syntax. `stageman.sh` requires v4. Detection approach:
- Run `yq --version` and parse for major version ≥ 4
- If v3 or unrecognized: report as failure with install hint

### Sunset `sync/1-prerequisites.sh`

`sync/1-prerequisites.sh` is replaced by `fab-doctor.sh`. The sync script `fab/.kit/sync/1-prerequisites.sh` is rewritten to delegate to `fab-doctor.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
exec "$(dirname "$0")/../scripts/fab-doctor.sh"
```

This keeps `fab-sync.sh`'s iteration model intact (it iterates `sync/*.sh` in order) while eliminating the duplicated tool list. Doctor's non-zero exit propagates through `set -e` in `fab-sync.sh`, halting sync on any missing prerequisite.

### `/fab-setup` integration

`/fab-setup` SHALL call `fab-doctor.sh` as an early gate before creating any project artifacts. If doctor exits non-zero, setup stops and surfaces the doctor output.

### Improve `fab-upgrade.sh` post-upgrade output

The version drift check (lines 99-113) is redesigned. The migration reminder moves to the **very last line** of output, with a prominent format:

Before:
```
Running fab-sync.sh to repair directories and agents...
[...sync output...]

Note: fab/project/VERSION (0.15.0) is behind engine (0.16.0). Run `/fab-setup migrations` to apply migrations.

Update complete: 0.15.0 → 0.16.0
```

After:
```
Running fab-sync.sh to repair directories and agents...
[...sync output...]

Update complete: 0.15.0 → 0.16.0

⚠ Run /fab-setup migrations to update project files (0.15.0 → 0.16.0)
```

When `fab/project/VERSION` is missing entirely:
```
Update complete: 0.15.0 → 0.16.0

⚠ Run /fab-setup to initialize, then /fab-setup migrations
```

When versions match: no warning, just the "Update complete" line.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Add `fab-doctor.sh` to scripts directory tree and Shell Scripts section; update `sync/1-prerequisites.sh` description to reflect delegation to doctor; update `fab-upgrade.sh` description to reflect improved output
- `fab-workflow/setup`: (modify) Document `/fab-setup` early gate integration with `fab-doctor.sh`

## Impact

- **New file**: `fab/.kit/scripts/fab-doctor.sh`
- **Modified file**: `fab/.kit/sync/1-prerequisites.sh` — rewritten to delegate to doctor
- **Modified file**: `fab/.kit/scripts/fab-upgrade.sh` — version drift output redesigned
- **Modified skill**: `fab/.kit/skills/fab-setup.md` — add doctor gate step

## Open Questions

None — all design decisions were resolved in the discussion session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Location is `fab/.kit/scripts/fab-doctor.sh` | Discussed — user explicitly chose "sibling of fab-help.sh" | S:95 R:90 A:95 D:95 |
| 2 | Certain | All 7 checks are required, no optional tier | Discussed — user said "Nothing optional. All required." and explicitly absorbed jq/bats | S:95 R:85 A:90 D:95 |
| 3 | Certain | Tool list: git, bash, yq v4+, jq, gh, bats, direnv+hook | Discussed — original 5 confirmed, jq/bats absorbed from 1-prerequisites.sh | S:95 R:80 A:90 D:90 |
| 4 | Certain | Accumulate all failures, report at end | Discussed — user agreed after reviewing fail-fast vs accumulate tradeoff | S:90 R:90 A:85 D:90 |
| 5 | Certain | Direnv hook detection via interactive subshell | Discussed — user provided the specific code snippet | S:95 R:85 A:90 D:95 |
| 6 | Certain | Output format: checkmark/cross per item with fix hints and summary | Discussed — user confirmed the proposed format | S:90 R:90 A:85 D:90 |
| 7 | Certain | Sunset `sync/1-prerequisites.sh` — delegate to doctor | Discussed — user agreed to absorb jq/bats and sunset the separate script | S:90 R:80 A:85 D:90 |
| 8 | Confident | Exit code = failure count (0 = all pass) | Standard convention, user confirmed in recap | S:80 R:90 A:85 D:85 |
| 9 | Confident | `/fab-setup` calls doctor as early gate | Discussed — user confirmed "integration: /fab-setup calls it as an early gate" | S:80 R:70 A:80 D:80 |
| 10 | Confident | `1-prerequisites.sh` becomes thin `exec` delegate | Preserves fab-sync.sh iteration model while eliminating duplication — simplest refactor | S:70 R:85 A:85 D:75 |
| 11 | Confident | `fab-upgrade.sh` migration reminder moves to last line with ⚠ prefix | Discussed — user agreed the current output buries the action; proposed format confirmed | S:85 R:90 A:80 D:80 |

11 assumptions (7 certain, 4 confident, 0 tentative, 0 unresolved).
