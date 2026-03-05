# Spec: Extract Logman from Preflight

**Change**: 260302-9fnn-extract-logman-from-preflight
**Created**: 2026-03-02
**Affected memory**: `docs/memory/fab-workflow/preflight.md`, `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Changing the `confidence` or `review` logman subcommands — only `command` is affected
- Adding new log event types — this is a restructuring of existing logging
- Changing preflight's validation logic or YAML output format

## Scripts: logman.sh — Command Subcommand Redesign

### Requirement: Command subcommand SHALL accept cmd as first positional argument

The `command` subcommand signature SHALL change from `command <change> <cmd> [args]` to `command <cmd> [change] [args]`. The `<cmd>` argument (skill name) is always required. The `[change]` argument is optional.

#### Scenario: Skill calls logman with cmd and explicit change
- **GIVEN** a resolved change `260302-9fnn-extract-logman-from-preflight` exists in `fab/changes/`
- **WHEN** `logman.sh command "fab-continue" "9fnn"` is invoked
- **THEN** a JSON line is appended to `fab/changes/260302-9fnn-extract-logman-from-preflight/.history.jsonl`
- **AND** the JSON contains `"event":"command"` and `"cmd":"fab-continue"`

#### Scenario: Skill calls logman with cmd and args
- **GIVEN** a resolved change exists
- **WHEN** `logman.sh command "fab-continue" "9fnn" "spec"` is invoked
- **THEN** a JSON line is appended with `"cmd":"fab-continue"` and `"args":"spec"`

### Requirement: Command subcommand SHALL silently succeed when change is omitted and no active change exists

When the `[change]` argument is omitted, logman SHALL attempt to resolve the active change via `fab/current` (reading the file, then passing its content to `resolve.sh --dir`). If resolution fails at any point (no `fab/current`, empty file, directory doesn't exist, resolve.sh fails), logman SHALL exit 0 silently — no error output, no stderr.

#### Scenario: No change arg, active change exists via fab/current
- **GIVEN** `fab/current` contains `260302-9fnn-extract-logman-from-preflight`
- **AND** the corresponding change directory exists
- **WHEN** `logman.sh command "fab-discuss"` is invoked (no change arg)
- **THEN** a JSON line is appended to the active change's `.history.jsonl`
- **AND** the JSON contains `"cmd":"fab-discuss"`

#### Scenario: No change arg, no fab/current file
- **GIVEN** `fab/current` does not exist
- **WHEN** `logman.sh command "fab-setup"` is invoked (no change arg)
- **THEN** logman exits with code 0
- **AND** no output on stdout or stderr
- **AND** no `.history.jsonl` file is created or modified

#### Scenario: No change arg, fab/current points to nonexistent directory
- **GIVEN** `fab/current` contains `stale-change-name`
- **AND** `fab/changes/stale-change-name/` does not exist
- **WHEN** `logman.sh command "fab-switch"` is invoked (no change arg)
- **THEN** logman exits with code 0 silently

### Requirement: Command subcommand SHALL still fail loudly when explicit change doesn't resolve

When a `[change]` argument IS provided but doesn't resolve, logman SHALL exit with code 1 and an error message on stderr. This preserves the existing contract for explicit change references (used by changeman, calc-score, etc.).

#### Scenario: Explicit change arg doesn't resolve
- **GIVEN** no change matching `nonexistent` exists in `fab/changes/`
- **WHEN** `logman.sh command "fab-continue" "nonexistent"` is invoked
- **THEN** logman exits with code 1
- **AND** stderr contains an error message from resolve.sh

### Requirement: Arg count validation SHALL adapt to new signature

The `command` subcommand SHALL accept 2 to 4 positional arguments total (subcommand + 1 to 3 args): `command <cmd>`, `command <cmd> <change>`, `command <cmd> <change> <args>`.

#### Scenario: Only cmd provided (minimum valid)
- **GIVEN** any environment state
- **WHEN** `logman.sh command "fab-help"` is invoked
- **THEN** logman does not exit with a usage error (may exit 0 silently if no change resolves)

#### Scenario: No arguments after subcommand (invalid)
- **GIVEN** any environment state
- **WHEN** `logman.sh command` is invoked with no further arguments
- **THEN** logman exits with code 1
- **AND** stderr contains a usage message showing the new signature

### Requirement: Help text and usage messages SHALL reflect new signature

The `--help` output and error-path usage messages SHALL show `command <cmd> [change] [args]` instead of the old `command <change> <cmd> [args]`.

#### Scenario: Help output shows new signature
- **GIVEN** logman.sh is invoked with `--help`
- **WHEN** output is inspected
- **THEN** the command subcommand shows `command <cmd> [change] [args]`

## Scripts: preflight.sh — Remove Driver Flag

### Requirement: Preflight SHALL NOT accept the --driver flag

The `--driver` flag, its argument parsing block, the `LOGMAN` variable declaration, and the logman call (step 6 in the current script) SHALL be removed entirely. Preflight becomes purely: test-build guard → validation (steps 1-5) → structured YAML output.

#### Scenario: Preflight invoked without --driver
- **GIVEN** a valid project and active change
- **WHEN** `preflight.sh` is invoked (no flags)
- **THEN** it outputs structured YAML to stdout as before
- **AND** no logman call occurs

#### Scenario: Preflight invoked with --driver (removed flag)
- **GIVEN** a valid project and active change
- **WHEN** `preflight.sh --driver fab-continue` is invoked
- **THEN** `--driver` is NOT recognized as a flag
- **AND** `--driver` is treated as the `[change-name]` positional arg (which will fail resolution with an appropriate error)

### Requirement: Preflight validation and output SHALL be unchanged

All 5 validation checks (init, staleness, change resolution, directory, status file) and the YAML output format SHALL remain identical. Only the logging side-effect is removed.

#### Scenario: Validation and output unchanged
- **GIVEN** a valid project with active change `260302-9fnn-test`
- **WHEN** `preflight.sh` is invoked
- **THEN** the stdout YAML contains the same fields: `name`, `change_dir`, `stage`, `display_stage`, `display_state`, `progress`, `checklist`, `confidence`

## Scripts: changeman.sh — Update Logman Command Calls

### Requirement: changeman `new` SHALL call logman with flipped arg order

The `new` subcommand's logman call SHALL change from `logman command "$folder_name" "fab-new" "$log_args"` to `logman command "fab-new" "$folder_name" "$log_args"`.

#### Scenario: New change logs with correct arg order
- **GIVEN** changeman creates a new change folder `260302-a1b2-my-change`
- **WHEN** the `--log-args` flag provides a description
- **THEN** logman is called as `command "fab-new" "260302-a1b2-my-change" "<description>"`

### Requirement: changeman `rename` SHALL call logman with flipped arg order

The `rename` subcommand's logman call SHALL change from `logman command "$new_name" "changeman-rename" ...` to `logman command "changeman-rename" "$new_name" ...`.

#### Scenario: Rename logs with correct arg order
- **GIVEN** a change `260216-u6d5-old-slug` is renamed to `260216-u6d5-new-slug`
- **WHEN** the rename operation completes
- **THEN** logman is called as `command "changeman-rename" "260216-u6d5-new-slug" "--folder ... --slug ..."`

## Skills: Direct Logman Invocation Protocol

### Requirement: _preamble.md §2 SHALL include a logman call step after preflight

A new step SHALL be added to the Change Context protocol (§2) in `_preamble.md`, after the existing step 3 (parse YAML) and before step 4 (load artifacts). The step instructs the agent to call logman with the skill name and the resolved change name from preflight output.

Pattern: `bash fab/.kit/scripts/lib/logman.sh command "<skill-name>" "<name>" 2>/dev/null || true`

Where `<name>` is the `name` field parsed from preflight's YAML output.

#### Scenario: Preflight-calling skill logs after preflight
- **GIVEN** a skill runs preflight and gets `name: 260302-9fnn-extract-logman-from-preflight` in the YAML
- **WHEN** the skill proceeds with its behavior
- **THEN** it calls `logman.sh command "<skill-name>" "260302-9fnn-extract-logman-from-preflight" 2>/dev/null || true` before loading artifacts

### Requirement: Exempt skills SHALL call logman directly in their own skill files

Skills that do NOT call preflight (`/fab-new`, `/fab-switch`, `/fab-setup`, `/fab-discuss`, `/fab-help`) SHALL include their own logman call instruction. The call uses no change arg (or uses the newly-created change name for `/fab-new`).

Pattern for exempt skills: `bash fab/.kit/scripts/lib/logman.sh command "<skill-name>" 2>/dev/null || true`

Pattern for `/fab-new` (after folder creation): `bash fab/.kit/scripts/lib/logman.sh command "fab-new" "<folder-name>" 2>/dev/null || true`

#### Scenario: Exempt skill logs opportunistically
- **GIVEN** `/fab-discuss` is invoked with an active change in `fab/current`
- **WHEN** logman resolves via `fab/current`
- **THEN** a command entry is appended to the active change's `.history.jsonl`

#### Scenario: Exempt skill logs with no active change
- **GIVEN** `/fab-setup` is invoked with no `fab/current`
- **WHEN** logman's resolution fails silently
- **THEN** no log entry is created, the skill proceeds normally

## Documentation: _scripts.md Updates

### Requirement: _scripts.md SHALL reflect new logman command signature

The logman section in `_scripts.md` SHALL update:
- Command signature: `command <cmd> [change] [args]`
- Examples: show cmd first, change second
- Remove "Skills never call logman.sh directly" — replace with "Skills call `logman.sh command` directly"
- Update call graph: remove `preflight.sh (command log via --driver)`, add skills as direct logman callers

#### Scenario: Updated call graph
- **GIVEN** the _scripts.md call graph
- **WHEN** inspected after the change
- **THEN** it shows: `resolve.sh` called by every other script; `logman.sh` called by `statusman.sh` (review auto-log), `calc-score.sh` (confidence log), `changeman.sh` (new/rename log), and skills (command log directly)

### Requirement: _scripts.md SHALL remove --driver from preflight documentation

The preflight section SHALL remove the `--driver` flag from the usage line and its description. The signature becomes `preflight.sh [<change-name>]`.

#### Scenario: Preflight docs show no --driver
- **GIVEN** the _scripts.md preflight section
- **WHEN** inspected after the change
- **THEN** the usage shows `preflight.sh [<change-name>]` with no `--driver` flag

### Requirement: logman callers table SHALL be updated

The callers table in the logman section SHALL remove the `preflight.sh --driver <skill>` row and note that skills call logman directly (via _preamble.md convention and per-skill instructions).

#### Scenario: Callers table reflects new callers
- **GIVEN** the logman callers table in _scripts.md
- **WHEN** inspected
- **THEN** it does NOT include a `preflight.sh --driver` row
- **AND** it includes a row showing skills as direct callers

## Deprecated Requirements

### Preflight --driver flag
**Reason**: Logging concern is decoupled from validation. Skills call logman directly.
**Migration**: Skills that previously relied on `preflight.sh --driver <skill-name>` for command logging now call `logman.sh command "<skill-name>" [change] 2>/dev/null || true` directly.

### "Skills never call logman.sh directly" convention
**Reason**: Inverted — skills now call logman directly as the primary command logging mechanism.
**Migration**: The `_preamble.md` §2 protocol includes a logman call step. Exempt skills include logman calls in their own instructions.

## Design Decisions

1. **Position-based arg parsing for new `command` signature**: The `<cmd>` is always `$2` (first arg after subcommand), `[change]` is `$3`, `[args]` is `$4`.
   - *Why*: Simple, unambiguous. The skill name is always a known string. No flag parsing needed.
   - *Rejected*: Named flags (`--cmd`, `--change`) — overengineered for a 3-arg command. Heuristic detection (is this arg a change ID or a skill name?) — fragile and error-prone.

2. **Silent exit 0 when change is omitted and resolution fails**: logman handles its own "should I log?" decision rather than requiring callers to check.
   - *Why*: Zero-cost logging for callers. Every skill uses the same pattern (`2>/dev/null || true`) regardless of whether a change is active. The `|| true` is a safety net for unexpected errors; the silent exit 0 handles the common "no active change" case cleanly.
   - *Rejected*: Caller-decides (each skill checks for active change before calling logman) — more code in every skill, defeats the purpose of simplification.

3. **Explicit change arg still fails loudly**: When `[change]` IS provided and doesn't resolve, logman exits 1 with an error.
   - *Why*: Preserves the contract for callers who provide an explicit change reference (changeman, calc-score). Silent failure there would mask real bugs.
   - *Rejected*: Silent exit 0 for all resolution failures — would hide bugs in changeman/calc-score calls where the change should always exist.

4. **Logman call in _preamble.md §2 (not in individual preflight-calling skills)**: The logging instruction is centralized in the shared preamble rather than duplicated in each skill file.
   - *Why*: Single point of change. All preflight-calling skills inherit the convention automatically.
   - *Rejected*: Per-skill logman instructions for preflight-calling skills — would create N copies of the same instruction, prone to drift.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | logman silently exits 0 on failed resolution when change omitted | Confirmed from intake #1 — user explicitly chose "logman decides, silently" | S:95 R:90 A:95 D:95 |
| 2 | Certain | Arg order flips to `command <cmd> [change]` | Confirmed from intake #2 — positional parsing, cmd always first | S:90 R:85 A:90 D:90 |
| 3 | Certain | preflight drops --driver entirely | Confirmed from intake #3 — clean separation, preflight is pure validation | S:95 R:90 A:95 D:95 |
| 4 | Certain | All skills get a logman call (preflight-calling via preamble, exempt via own file) | Confirmed from intake #4 — user asked about fab-discuss logging opportunistically | S:90 R:90 A:90 D:90 |
| 5 | Confident | changeman.sh `new` and `rename` update to new `command` arg order | Consistent interface — confirmed from intake #5, only two `command` calls in changeman | S:75 R:80 A:90 D:85 |
| 6 | Confident | When change omitted, logman reads fab/current then calls resolve.sh (not direct fuzzy match) | Confirmed from intake #6 — fab/current is the standard active-change mechanism; resolve.sh handles path derivation | S:70 R:90 A:85 D:80 |
| 7 | Certain | confidence and review subcommands unchanged | Explicitly scoped out — only `command` subcommand is affected | S:95 R:95 A:95 D:95 |
| 8 | Confident | _preamble.md doesn't currently reference --driver, so the change is adding a new logman step rather than removing --driver | Verified by reading _preamble.md §2 — no --driver in invocation instructions | S:85 R:90 A:90 D:85 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
