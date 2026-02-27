# Spec: Fix Kit Scripts

**Change**: 260227-yobi-fix-kit-scripts
**Created**: 2026-02-27
**Affected memory**: `docs/memory/fab-workflow/kit-scripts.md`

## Non-Goals

- Changing `resolve_change_arg` behavior for non-history commands — already correct
- Adding new stageman subcommands
- Changing `changeman.sh` (already works correctly, `rename` already exists)

## Stageman: Unified Argument Resolution

### Requirement: History commands SHALL use `resolve_change_arg`

The three history commands (`log-command`, `log-confidence`, `log-review`) SHALL resolve their first argument through `resolve_change_arg()` instead of `resolve_change_dir()`. The resolved `.status.yaml` path SHALL be converted to a change directory via `dirname` before appending to `.history.jsonl`.

#### Scenario: History command with change ID
- **GIVEN** a change `260227-yobi-fix-kit-scripts` exists
- **WHEN** `stageman.sh log-command yobi "fab-ff"` is called
- **THEN** `resolve_change_arg` resolves `yobi` to `/repo/fab/changes/260227-yobi-fix-kit-scripts/.status.yaml`
- **AND** `dirname` derives `/repo/fab/changes/260227-yobi-fix-kit-scripts`
- **AND** the event is appended to `/repo/fab/changes/260227-yobi-fix-kit-scripts/.history.jsonl`

#### Scenario: History command with .status.yaml path (backward compat)
- **GIVEN** a change exists at `fab/changes/260227-yobi-fix-kit-scripts/`
- **WHEN** `stageman.sh log-review fab/changes/260227-yobi-fix-kit-scripts/.status.yaml "passed"` is called
- **THEN** `resolve_change_arg` accepts the existing file path directly
- **AND** `dirname` derives the change directory
- **AND** the event is appended to `.history.jsonl` in that directory

#### Scenario: History command with folder name (primary convention)
- **GIVEN** a change `260227-yobi-fix-kit-scripts` exists
- **WHEN** `stageman.sh log-command 260227-yobi-fix-kit-scripts "fab-ff"` is called
- **THEN** `resolve_change_arg` delegates to `changeman.sh resolve` which returns the folder name
- **AND** the `.status.yaml` path is derived and `dirname` gives the change directory
- **AND** the event is appended to `.history.jsonl`

All internal callers (scripts, skills) MUST pass change identifiers (change IDs, folder names) or `.status.yaml` paths — not bare directory paths.

### Requirement: `resolve_change_dir` SHALL be removed

After migrating history commands to `resolve_change_arg`, the `resolve_change_dir()` function SHALL be removed. No deprecation shim — dead code is deleted.

#### Scenario: No remaining callers
- **GIVEN** all history commands use `resolve_change_arg`
- **WHEN** `resolve_change_dir` is grep'd across the codebase
- **THEN** zero references remain outside the function definition itself
- **AND** the function and its doc comment are deleted

### Requirement: Help text SHALL show `<change>` for all commands

The `show_help()` function SHALL display `<change>` as the argument placeholder for history commands, replacing `<change_dir>`. The help examples SHALL use change IDs.

#### Scenario: Updated help text
- **GIVEN** a user runs `stageman.sh --help`
- **WHEN** the help text is displayed
- **THEN** history commands show: `log-command <change> <cmd> [args]`, `log-confidence <change> <score> <delta> <trigger>`, `log-review <change> <result> [rework]`
- **AND** examples show change IDs: `stageman.sh log-review yobi "passed"`

### Requirement: Internal function signatures SHALL accept status file path

The `log_command()`, `log_confidence()`, and `log_review()` internal functions SHALL accept a `.status.yaml` file path as their first argument and derive the change directory via `dirname`. Their doc comments SHALL be updated to reflect this.

#### Scenario: Function receives resolved path
- **GIVEN** the CLI dispatch resolved `yobi` to `/repo/fab/changes/260227-yobi-fix-kit-scripts/.status.yaml`
- **WHEN** `log_command "/repo/fab/changes/260227-yobi-fix-kit-scripts/.status.yaml" "fab-ff"` is called
- **THEN** the function computes `change_dir=$(dirname "$1")`
- **AND** appends to `${change_dir}/.history.jsonl`

## Calc-Score: Self-Contained Spec Gate

### Requirement: `--check-gate` SHALL parse the artifact for all stages

When `--check-gate` is passed, `calc-score.sh` SHALL parse the relevant artifact (`intake.md` or `spec.md`) and compute the score inline, regardless of stage. The spec gate path SHALL NOT read pre-computed values from `.status.yaml`.

#### Scenario: Spec gate without prior scoring run
- **GIVEN** a change with `spec.md` containing an Assumptions table
- **AND** `.status.yaml` has `confidence.score: 0.0` (never scored)
- **WHEN** `calc-score.sh --check-gate fab/changes/{name}` is called
- **THEN** the script parses `spec.md`'s `## Assumptions` table directly
- **AND** computes certain/confident/tentative/unresolved counts from the table
- **AND** applies the coverage-weighted formula
- **AND** compares against the per-type threshold
- **AND** outputs the gate result with the freshly computed score

#### Scenario: Spec gate matches intake gate pattern
- **GIVEN** the intake gate path (lines 137-171) parses `intake.md` inline
- **WHEN** the spec gate path is executed
- **THEN** it follows the same pattern: parse artifact → count grades → compute score → compare threshold
- **AND** does NOT call `grep '^ *score:'` on `.status.yaml`

#### Scenario: Gate check is read-only
- **GIVEN** `--check-gate` is passed
- **WHEN** the script executes
- **THEN** `.status.yaml` is NOT modified
- **AND** the only output is the YAML gate result on stdout

## Shipped Script Guide: `_scripts.md`

### Requirement: `fab/.kit/skills/_scripts.md` SHALL document all kit scripts

A new file `fab/.kit/skills/_scripts.md` SHALL be created containing:
1. The unified `<change>` argument convention with accepted forms
2. Per-script summaries for `changeman.sh`, `stageman.sh`, `calc-score.sh`, and `preflight.sh`
3. Stage transition side effects (which `finish` auto-activates which next stage)
4. Common error messages and their meanings

#### Scenario: Agent invokes stageman correctly after reading `_scripts.md`
- **GIVEN** a skill loads `_preamble.md` which references `_scripts.md`
- **WHEN** the agent needs to call `stageman.sh finish` for the intake stage
- **THEN** it finds the convention: `stageman.sh finish <change> <stage> [driver]`
- **AND** uses a change ID (e.g., `yobi`) as the `<change>` argument
- **AND** the call succeeds

#### Scenario: Agent understands stage transitions
- **GIVEN** the agent reads `_scripts.md`'s stage transition section
- **WHEN** it calls `stageman.sh finish yobi intake fab-ff`
- **THEN** it knows spec becomes `active` automatically
- **AND** does NOT call `stageman.sh start yobi spec` (redundant)

### Requirement: `_preamble.md` SHALL reference `_scripts.md`

The always-load section of `_preamble.md` SHALL include an instruction to also read `_scripts.md` for script invocation conventions.

#### Scenario: Preamble chain loads scripts guide
- **GIVEN** a skill begins with "Read `_preamble.md`"
- **WHEN** the agent reads `_preamble.md`'s always-load section
- **THEN** it finds: "Also read `fab/.kit/skills/_scripts.md` for script invocation conventions"
- **AND** reads `_scripts.md` before proceeding

### Requirement: `_generation.md` SHALL use `<change>` placeholders

Lines 95-97 of `_generation.md` currently show `<file>` as the stageman argument placeholder. These SHALL be updated to `<change>` to match the unified convention.

#### Scenario: Updated generation procedure
- **GIVEN** an agent reads `_generation.md`'s Checklist Generation Procedure
- **WHEN** it sees the stageman invocation examples
- **THEN** they show `fab/.kit/scripts/lib/stageman.sh set-checklist <change> generated true`
- **AND** NOT `<file>`

## Internal Callers: Update to Change ID Convention

### Requirement: All internal callers SHALL pass change identifiers or `.status.yaml` paths

Scripts that call stageman history commands currently pass directory paths. Each SHALL be updated to pass a change identifier (folder name) or `.status.yaml` path instead. `resolve_change_arg` does NOT accept bare directory paths — this is by design (separation of concerns).

#### Scenario: calc-score.sh calls log-confidence with `.status.yaml` path
- **GIVEN** `calc-score.sh` is invoked with `fab/changes/{name}` as change-dir
- **AND** it has `$status_file` = `fab/changes/{name}/.status.yaml`
- **WHEN** it reaches the `log-confidence` call (line 329)
- **THEN** it passes `"$status_file"` instead of `"$change_dir"`
- **AND** `resolve_change_arg` accepts the existing file path directly

#### Scenario: changeman.sh calls log-command with folder name
- **GIVEN** `changeman.sh new` creates a change with folder name `260227-yobi-fix-kit-scripts`
- **WHEN** it reaches the `log-command` call (line 398)
- **THEN** it passes `"$folder_name"` (e.g., `260227-yobi-fix-kit-scripts`) instead of `"$changes_dir/$folder_name"`
- **AND** `resolve_change_arg` resolves the folder name via `changeman.sh resolve`

#### Scenario: changeman.sh rename calls log-command with new folder name
- **GIVEN** `changeman.sh rename` renames a change to folder name `260227-yobi-new-slug`
- **WHEN** it reaches the `log-command` call (line 486)
- **THEN** it passes `"$new_name"` instead of `"$changes_dir/$new_name"`
- **AND** `resolve_change_arg` resolves the folder name via `changeman.sh resolve`

### Requirement: Skill prompts SHALL use `<change>` for all stageman commands

Skill files that show stageman history command examples currently use `<change_dir>`. These SHALL be updated to `<change>` to match the unified convention. Affected files: `fab-ff.md`, `fab-fff.md`, `fab-continue.md`, `fab-clarify.md`.

#### Scenario: Skill prompt shows unified convention
- **GIVEN** an agent reads `fab-ff.md`
- **WHEN** it encounters a stageman invocation example
- **THEN** all commands show `<change>` as the argument: `stageman.sh log-command <change> "fab-ff"`, `stageman.sh log-review <change> "passed"`
- **AND** no instances of `<change_dir>` or `<file>` remain

## Memory Reference: `kit-scripts.md`

### Requirement: Deep reference SHALL be created in memory

`docs/memory/fab-workflow/kit-scripts.md` SHALL be created as a comprehensive reference covering:
- Internal function architecture (`resolve_change_arg`, event functions, etc.)
- State machine details (valid transitions, auto-activation rules)
- History logging format (`.history.jsonl` schema)
- Design rationale for unified argument resolution

This is supplementary to `_scripts.md` — the shipped guide covers "how to call", the memory file covers "how it works".

#### Scenario: Memory index updated
- **GIVEN** `kit-scripts.md` is created
- **WHEN** `docs/memory/fab-workflow/index.md` is checked
- **THEN** it contains an entry for `kit-scripts` with description and date

## Design Decisions

1. **Route history through `resolve_change_arg` + `dirname` rather than creating a new resolver**
   - *Why*: `resolve_change_arg` already handles all input forms and is battle-tested across all other commands. Adding `dirname` is trivial.
   - *Rejected*: Creating a `resolve_change_to_dir()` that combines both — unnecessary abstraction for a one-liner derivation.

2. **`_scripts.md` as separate file rather than section in `_preamble.md`**
   - *Why*: Separation of concerns — preamble covers workflow conventions, scripts covers tool usage. Preamble is already 316 lines.
   - *Rejected*: Adding to `_preamble.md` directly — would bloat an already dense file. Adding to `_generation.md` — only loaded during generation, but scripts are called by every skill.

3. **Keep `resolve_change_arg` strict — no directory path support**
   - *Why*: Separation of concerns. Logging, stage machinery, and change management are distinct concerns — all should work with change identifiers at the boundary. Adding directory support would make the interface permissive and hide caller bugs. Internal callers pass folder names (which are valid change identifiers) or `.status.yaml` paths.
   - *Rejected*: Adding `[ -d "$arg" ] && [ -f "$arg/.status.yaml" ]` check — would avoid breaking callers but muddies the interface and creates ambiguity between directory paths and change name substrings.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | History commands route through `resolve_change_arg` | Confirmed from intake #1 — user explicitly chose change ID convention. All non-history commands already use this | S:90 R:85 A:95 D:95 |
| 2 | Certain | Spec gate parses spec.md inline | Confirmed from intake #2 — intake gate demonstrates the pattern, spec gate mirrors it | S:85 R:80 A:90 D:95 |
| 3 | Certain | `changeman.sh rename` dropped from scope | Confirmed from intake #3 — already exists in source | S:95 R:95 A:95 D:95 |
| 4 | Certain | Script guide is `_scripts.md` (separate shipped file) | Confirmed from intake #4 — discussed in conversation, user chose separate file | S:90 R:85 A:90 D:90 |
| 5 | Certain | `_preamble.md` chain-loads `_scripts.md` | Confirmed from intake #5 — one-liner reference in always-load section | S:85 R:90 A:90 D:95 |
| 6 | Certain | Keep `resolve_change_arg` strict — no directory support | Confirmed from intake #6 — user chose separation of concerns. Each script accepts change IDs at boundary | S:85 R:85 A:90 D:90 |
| 7 | Certain | Internal callers pass folder names or `.status.yaml` paths | Confirmed from intake #7 — changeman passes `$folder_name`, calc-score passes `$status_file`. No directory paths | S:80 R:85 A:90 D:90 |
| 8 | Confident | Remove `resolve_change_dir` (no deprecation) | Upgraded from intake #9 — verified no external callers after migration. Dead code deletion is clean | S:70 R:85 A:85 D:90 |
| 9 | Confident | `dirname` on resolved `.status.yaml` for change dir | Confirmed from intake #8 — natural, one-liner derivation. No alternatives needed | S:70 R:90 A:85 D:90 |
| 10 | Confident | Update all skill prompts and `_generation.md` to `<change>` | Confirmed from intake #11 — 5 files with inconsistent placeholders (`<change_dir>`, `<file>`) | S:70 R:90 A:85 D:90 |
| 11 | Confident | Memory file is supplementary deep reference | Confirmed from intake #10 — `_scripts.md` is the primary deliverable, memory covers internals | S:75 R:85 A:80 D:80 |

11 assumptions (7 certain, 4 confident, 0 tentative, 0 unresolved).
