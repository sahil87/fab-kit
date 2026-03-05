# Spec: Add fab-doctor.sh

**Change**: 260223-sr3u-add-fab-doctor
**Created**: 2026-02-23
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/setup.md`

## Non-Goals

- Auto-installing missing tools — doctor diagnoses, it does not fix
- Replacing `sync/1-prerequisites.sh` logic — the file remains as a thin delegate
- Checking dev-environment-specific tools beyond the 7 required (e.g., editors, language runtimes)

## Scripts: fab-doctor.sh

### Requirement: Script Location and Structure

`fab-doctor.sh` SHALL be located at `fab/.kit/scripts/fab-doctor.sh`, as a sibling of `fab-help.sh`. The script SHALL use `#!/usr/bin/env bash` and `set -euo pipefail`. It SHALL be executable (`chmod +x`).

#### Scenario: Script exists at expected location
- **GIVEN** a user has `fab/.kit/` installed
- **WHEN** they run `fab/.kit/scripts/fab-doctor.sh`
- **THEN** the script executes and checks all 7 prerequisites

### Requirement: Tool Presence Checks

`fab-doctor.sh` SHALL check the following 7 tools in order: `git`, `bash`, `yq`, `jq`, `gh`, `bats`, `direnv`. Each check SHALL use `command -v {tool}` to verify the tool is on PATH. All checks are required — there is no optional tier.

#### Scenario: All tools present
- **GIVEN** all 7 tools are installed and on PATH
- **WHEN** `fab-doctor.sh` runs
- **THEN** each tool prints a checkmark line with version
- **AND** the summary shows "7/7 checks passed."
- **AND** exit code is 0

#### Scenario: One tool missing
- **GIVEN** `yq` is not installed
- **WHEN** `fab-doctor.sh` runs
- **THEN** the `yq` check prints a cross line: `  ✗ yq — not found`
- **AND** an actionable fix hint follows: `    Install: brew install yq`
- **AND** remaining checks still execute (no early exit)
- **AND** the summary shows "6/7 checks passed. 1 issue found."
- **AND** exit code is 1

#### Scenario: Multiple tools missing
- **GIVEN** `yq` and `gh` are not installed
- **WHEN** `fab-doctor.sh` runs
- **THEN** both missing tools print cross lines with fix hints
- **AND** the summary shows "5/7 checks passed. 2 issues found."
- **AND** exit code is 2

### Requirement: Version Display

For each tool that passes, `fab-doctor.sh` SHALL display the tool's version alongside the checkmark. Version extraction approaches:

| Tool | Version command | Parse |
|------|----------------|-------|
| `git` | `git --version` | Extract version number |
| `bash` | `bash --version` | Extract version number from first line |
| `yq` | `yq --version` | Extract version number (after v4 gate) |
| `jq` | `jq --version` | Strip leading `jq-` prefix |
| `gh` | `gh --version` | Extract version number from first line |
| `bats` | `bats --version` | Extract version number |
| `direnv` | `direnv version` | Use output directly |

#### Scenario: Version displayed for passing tool
- **GIVEN** `git` version 2.43.0 is installed
- **WHEN** `fab-doctor.sh` runs
- **THEN** the git check prints: `  ✓ git 2.43.0`

### Requirement: yq Version Gate

`fab-doctor.sh` SHALL verify that `yq` is version 4 or higher. yq v3 (Python-based) and v4 (Go-based, Mike Farah) have incompatible syntax. `stageman.sh` and other kit scripts require v4.

The script SHALL parse the major version from `yq --version` output and fail if the major version is less than 4.

#### Scenario: yq v4 installed
- **GIVEN** `yq` version 4.35.1 is on PATH
- **WHEN** `fab-doctor.sh` runs
- **THEN** the yq check prints: `  ✓ yq 4.35.1`

#### Scenario: yq v3 installed
- **GIVEN** `yq` version 3.4.1 (Python) is on PATH
- **WHEN** `fab-doctor.sh` runs
- **THEN** the yq check prints: `  ✗ yq 3.4.1 — version 4+ required (you have the Python version)`
- **AND** a fix hint follows: `    Install the Go version: brew install yq`

#### Scenario: yq not installed
- **GIVEN** `yq` is not on PATH
- **WHEN** `fab-doctor.sh` runs
- **THEN** the yq check prints: `  ✗ yq — not found`
- **AND** a fix hint follows: `    Install: brew install yq`

### Requirement: Direnv Shell Hook Detection

The direnv check SHALL have two parts: binary presence and shell hook integration. Both MUST pass for the direnv check to succeed.

Shell hook detection SHALL spawn an interactive subshell (`-i` flag) to source the user's rc files, then check for the hook:
- **zsh**: Check for `_direnv_hook` function via `zsh -i -c 'typeset -f _direnv_hook'`
- **bash**: Check `PROMPT_COMMAND` contains "direnv" via `bash -i -c '[[ "${PROMPT_COMMAND:-}" == *direnv* ]]'`

The target shell SHALL be detected from `$SHELL` basename.

All subshell output (motd, prompt rendering, rc file noise) SHALL be suppressed via `&>/dev/null`.

#### Scenario: direnv installed with hook active
- **GIVEN** `direnv` is on PATH and `eval "$(direnv hook zsh)"` is in the user's `.zshrc`
- **AND** the user's `$SHELL` is `/bin/zsh`
- **WHEN** `fab-doctor.sh` runs
- **THEN** the direnv check prints: `  ✓ direnv 2.34.0 (zsh hook active)`

#### Scenario: direnv installed but hook missing
- **GIVEN** `direnv` is on PATH but no hook is configured in the user's shell rc
- **AND** the user's `$SHELL` is `/bin/zsh`
- **WHEN** `fab-doctor.sh` runs
- **THEN** the direnv check prints: `  ✗ direnv shell hook not detected for zsh`
- **AND** a multi-line fix hint follows:
  ```
      Add the following to your ~/.zshrc (or equivalent):
        eval "$(direnv hook zsh)"
  ```

#### Scenario: direnv not installed
- **GIVEN** `direnv` is not on PATH
- **WHEN** `fab-doctor.sh` runs
- **THEN** the direnv check prints: `  ✗ direnv — not found`
- **AND** a fix hint follows: `    Install: brew install direnv`

### Requirement: Output Format

`fab-doctor.sh` SHALL produce structured output with a header, per-tool results, and a summary line.

The header line SHALL be: `fab-doctor: checking prerequisites...`

Each tool result SHALL be indented with 2 spaces and use `✓` (success) or `✗` (failure).

The summary line SHALL follow a blank line after the last tool result:
- All pass: `{N}/{N} checks passed.`
- Some fail: `{pass}/{total} checks passed. {fail} issue(s) found.`

#### Scenario: Output structure
- **GIVEN** any combination of pass/fail results
- **WHEN** `fab-doctor.sh` runs
- **THEN** output follows the format: header, per-tool lines, blank line, summary

### Requirement: Failure Accumulation

`fab-doctor.sh` SHALL execute all 7 checks regardless of individual failures. It SHALL NOT exit early on the first failure. A failure counter SHALL be incremented for each failed check. The script's exit code SHALL equal the failure count (0 = all pass, 1-7 = number of failures).

#### Scenario: Failures do not short-circuit
- **GIVEN** the first tool (`git`) is missing
- **WHEN** `fab-doctor.sh` runs
- **THEN** all 7 checks still execute
- **AND** the exit code reflects the total failure count

## Sync: Sunset 1-prerequisites.sh

### Requirement: Delegate to fab-doctor.sh

`fab/.kit/sync/1-prerequisites.sh` SHALL be rewritten to delegate to `fab-doctor.sh` via `exec`. The script body SHALL be:

```bash
#!/usr/bin/env bash
set -euo pipefail
exec "$(dirname "$0")/../scripts/fab-doctor.sh"
```

This preserves `fab-sync.sh`'s iteration model (`sync/*.sh` in sorted order) while eliminating the duplicated tool list. Doctor's non-zero exit propagates through `set -e` in `fab-sync.sh`, halting sync on any missing prerequisite.

#### Scenario: fab-sync.sh runs doctor via delegate
- **GIVEN** `fab-sync.sh` iterates `sync/*.sh` in sorted order
- **WHEN** `sync/1-prerequisites.sh` executes
- **THEN** it delegates to `fab-doctor.sh`
- **AND** a non-zero exit from doctor halts the sync pipeline

#### Scenario: Doctor replaces inline tool list
- **GIVEN** the previous `1-prerequisites.sh` checked `yq jq gh direnv bats` inline
- **WHEN** the new delegate version runs
- **THEN** tool checking is entirely handled by `fab-doctor.sh`
- **AND** no tool names are hardcoded in `1-prerequisites.sh`

## Upgrade: Improve fab-upgrade.sh Output

### Requirement: Migration Reminder as Final Output

`fab-upgrade.sh` SHALL print the version drift warning as the **last line of output**, after the "Update complete" line. The warning SHALL use a `⚠` prefix for visual prominence.

The current output (lines 99-113 of `fab-upgrade.sh`) SHALL be restructured:

**When `fab/project/VERSION` exists and is behind engine version:**
```
Update complete: {old} → {new}

⚠ Run /fab-setup migrations to update project files ({local} → {new})
```

**When `fab/project/VERSION` does not exist:**
```
Update complete: {old} → {new}

⚠ Run /fab-setup to initialize, then /fab-setup migrations
```

**When versions match (no drift):**
```
Update complete: {old} → {new}
```

No "Note:" prefix. No explanation paragraph. Just the command the user needs.

#### Scenario: Version drift detected after upgrade
- **GIVEN** `fab/project/VERSION` is `0.15.0` and the new engine is `0.16.0`
- **WHEN** `fab-upgrade.sh` completes successfully
- **THEN** the last line of output is: `⚠ Run /fab-setup migrations to update project files (0.15.0 → 0.16.0)`
- **AND** this line appears after "Update complete: ..."

#### Scenario: No version drift
- **GIVEN** `fab/project/VERSION` matches the new engine version
- **WHEN** `fab-upgrade.sh` completes successfully
- **THEN** no warning line is printed
- **AND** the output ends with: `Update complete: {old} → {new}`

#### Scenario: VERSION file missing
- **GIVEN** `fab/project/VERSION` does not exist
- **WHEN** `fab-upgrade.sh` completes successfully
- **THEN** the last line is: `⚠ Run /fab-setup to initialize, then /fab-setup migrations`

## Setup: Doctor as Early Gate

### Requirement: fab-setup Calls Doctor Before Bootstrap

`/fab-setup` (bare invocation — bootstrap behavior) SHALL call `fab/.kit/scripts/fab-doctor.sh` as the first step, before any structural bootstrap or interactive configuration. If doctor exits non-zero, `/fab-setup` SHALL stop immediately and surface the doctor output.

This applies only to the bare bootstrap flow, not to subcommands (`config`, `constitution`, `migrations`).

#### Scenario: Doctor passes — setup proceeds
- **GIVEN** all prerequisites are met
- **WHEN** the user runs `/fab-setup`
- **THEN** doctor runs and passes silently (output displayed)
- **AND** bootstrap continues with step 1a (config.yaml)

#### Scenario: Doctor fails — setup stops
- **GIVEN** `yq` is not installed
- **WHEN** the user runs `/fab-setup`
- **THEN** doctor runs and reports the failure
- **AND** `/fab-setup` stops without creating any project artifacts
- **AND** the user sees the doctor output with fix hints

## Design Decisions

1. **Single comprehensive script over modular checks**
   - *Why*: Constitution I (Pure Prompt Play) — keep it simple. One script, one concern, no import/source chain. The 7-check scope is small enough that splitting into modules adds complexity without benefit.
   - *Rejected*: Separate check functions sourced by both doctor and 1-prerequisites.sh — over-engineering for 7 checks.

2. **Delegate from 1-prerequisites.sh via exec, not delete**
   - *Why*: `fab-sync.sh` iterates `sync/*.sh` in sorted order. Deleting `1-prerequisites.sh` would require either renaming `fab-doctor.sh` into the sync directory (wrong — it's a user-facing script) or adding a call to `fab-sync.sh` itself (breaks the iteration model). `exec` delegation preserves the architecture.
   - *Rejected*: Moving doctor into `sync/` — doctor is user-facing and belongs in `scripts/`. Deleting `1-prerequisites.sh` and calling doctor from `fab-sync.sh` directly — breaks the generic iteration pattern.

3. **Interactive subshell for direnv hook detection**
   - *Why*: The hook is installed in shell rc files (`.zshrc`, `.bashrc`). The only reliable way to test if it's active is to spawn an interactive shell that sources those files. Non-interactive checks (looking for strings in rc files directly) are fragile — users may source from plugin managers, conditional blocks, or alternate rc paths (ZDOTDIR).
   - *Rejected*: Grep-based rc file scanning — too many edge cases (oh-my-zsh, zinit, bash-it, custom ZDOTDIR, etc.).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Location: `fab/.kit/scripts/fab-doctor.sh` | Confirmed from intake #1 — user explicitly chose sibling of fab-help.sh | S:95 R:90 A:95 D:95 |
| 2 | Certain | 7 required tools: git, bash, yq v4+, jq, gh, bats, direnv+hook | Confirmed from intake #3 — original 5 plus jq/bats absorbed from 1-prerequisites.sh | S:95 R:80 A:90 D:90 |
| 3 | Certain | Accumulate all failures, exit code = failure count | Confirmed from intake #4 | S:90 R:90 A:85 D:90 |
| 4 | Certain | Direnv hook detection via interactive subshell | Confirmed from intake #5 — user provided code snippet | S:95 R:85 A:90 D:95 |
| 5 | Certain | Sunset 1-prerequisites.sh via exec delegation | Confirmed from intake #7 — user agreed to absorb and sunset | S:90 R:80 A:85 D:90 |
| 6 | Certain | fab-upgrade.sh reminder as last line with ⚠ prefix | Confirmed from intake #11 — user agreed the current output buries the action | S:85 R:90 A:80 D:80 |
| 7 | Confident | `/fab-setup` gate applies to bare bootstrap only, not subcommands | Subcommands (config, constitution, migrations) have their own pre-flights; gating them on doctor would add friction for targeted edits | S:70 R:85 A:80 D:75 |
| 8 | Confident | Version parsing uses tool-specific commands per table | Standard approach — each tool has a different `--version` format, no universal parser | S:75 R:90 A:85 D:80 |
| 9 | Confident | `brew install` as default fix hint | Project context shows macOS-oriented dev environment; brew is the most common package manager. Users on other platforms will adapt | S:65 R:95 A:70 D:70 |

9 assumptions (6 certain, 3 confident, 0 tentative, 0 unresolved).
