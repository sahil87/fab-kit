# Spec: Stageman Write API

**Change**: 260214-w3r8-stageman-write-api
**Created**: 2026-02-14
**Affected memory**: `fab/memory/fab-workflow/kit-architecture.md`, `fab/memory/fab-workflow/change-lifecycle.md`, `fab/memory/fab-workflow/planning-skills.md`, `fab/memory/fab-workflow/execution-skills.md`, `fab/memory/fab-workflow/preflight.md`

## Non-Goals

- Changing any read accessor behavior — existing `get_progress_map`, `get_checklist`, `get_confidence`, `get_current_stage` remain unchanged
- Adding write CLI commands to `_preflight.sh` — preflight remains read-only
- Adding new stages or states to the workflow schema
- Modifying `_resolve-change.sh` — change resolution is unrelated to writes

## Stageman: Write Functions

### Requirement: set_stage_state

`set_stage_state <status_file> <stage> <state>` SHALL set a single stage's progress value in `.status.yaml`.

The function MUST:
1. Validate `<stage>` exists via `validate_stage`
2. Validate `<state>` is allowed for `<stage>` via `validate_stage_state`
3. Replace the stage's current value in the progress block
4. Update `last_updated` to the current ISO 8601 timestamp
5. Use temp-file-then-mv for atomic writes

The function MUST validate that `<status_file>` exists before any parsing or writing; if not found, return exit code 1 with a "file not found" diagnostic to stderr.

The function SHALL return exit code 0 on success, 1 on validation failure (with diagnostic to stderr).

#### Scenario: Status file not found

- **GIVEN** a path to a nonexistent file
- **WHEN** `set_stage_state "/nonexistent/.status.yaml" spec active` is called
- **THEN** stderr contains a "file not found" message
- **AND** exit code is 1

#### Scenario: Valid state change

- **GIVEN** a `.status.yaml` with `spec: pending`
- **WHEN** `set_stage_state "$file" spec active` is called
- **THEN** the file contains `spec: active`
- **AND** `last_updated` is refreshed
- **AND** exit code is 0

#### Scenario: Invalid state for stage

- **GIVEN** a `.status.yaml` with `brief: active`
- **WHEN** `set_stage_state "$file" brief failed` is called
- **THEN** the file is NOT modified
- **AND** stderr contains an error message mentioning the invalid state
- **AND** exit code is 1

#### Scenario: Invalid stage name

- **GIVEN** any `.status.yaml`
- **WHEN** `set_stage_state "$file" nonexistent done` is called
- **THEN** the file is NOT modified
- **AND** stderr contains an error message mentioning the invalid stage
- **AND** exit code is 1

### Requirement: transition_stages

`transition_stages <status_file> <from_stage> <to_stage>` SHALL perform the two-write transition: set `<from_stage>` to `done` and `<to_stage>` to `active` in a single atomic file write.

The function MUST:
1. Validate both stages exist via `validate_stage`
2. Validate `done` is allowed for `<from_stage>` and `active` is allowed for `<to_stage>`
3. Validate `<from_stage>` is currently `active` in the file (read the current value and reject if not `active`)
4. Verify `<to_stage>` is the next stage after `<from_stage>` in the pipeline sequence (via `get_next_stage`)
5. Write both changes in a single temp-file-then-mv operation
6. Update `last_updated`

All preconditions from `set_stage_state` apply: the function MUST validate `<status_file>` exists before any parsing.

The function SHALL return exit code 0 on success, 1 on validation failure.

#### Scenario: Normal forward transition

- **GIVEN** a `.status.yaml` with `brief: active, spec: pending`
- **WHEN** `transition_stages "$file" brief spec` is called
- **THEN** the file contains `brief: done` and `spec: active`
- **AND** `last_updated` is refreshed
- **AND** exit code is 0

#### Scenario: from_stage not currently active

- **GIVEN** a `.status.yaml` with `brief: done, spec: pending`
- **WHEN** `transition_stages "$file" brief spec` is called
- **THEN** the file is NOT modified
- **AND** stderr contains an error that `brief` is not currently `active`
- **AND** exit code is 1

#### Scenario: Non-adjacent stages

- **GIVEN** a `.status.yaml` with `brief: active, tasks: pending`
- **WHEN** `transition_stages "$file" brief tasks` is called
- **THEN** the file is NOT modified
- **AND** stderr contains an error about non-adjacent stages
- **AND** exit code is 1

#### Scenario: Last pipeline transition

- **GIVEN** a `.status.yaml` with `review: active, hydrate: pending`
- **WHEN** `transition_stages "$file" review hydrate` is called
- **THEN** the file contains `review: done` and `hydrate: active`
- **AND** exit code is 0

### Requirement: set_checklist_field

`set_checklist_field <status_file> <field> <value>` SHALL update a single field in the `checklist:` block of `.status.yaml`.

The function MUST validate that `<status_file>` exists before any parsing.

The function MUST:
1. Validate `<field>` is one of: `generated`, `completed`, `total`
2. Validate value type: `generated` accepts `true`/`false`; `completed` and `total` accept non-negative integers
3. Replace the field's value in the checklist block
4. Update `last_updated`
5. Use temp-file-then-mv for atomic writes

#### Scenario: Set checklist generated

- **GIVEN** a `.status.yaml` with `checklist.generated: false`
- **WHEN** `set_checklist_field "$file" generated true` is called
- **THEN** the file contains `generated: true` in the checklist block
- **AND** `last_updated` is refreshed

#### Scenario: Update checklist total

- **GIVEN** a `.status.yaml` with `checklist.total: 0`
- **WHEN** `set_checklist_field "$file" total 15` is called
- **THEN** the file contains `total: 15` in the checklist block

#### Scenario: Invalid field name

- **GIVEN** any `.status.yaml`
- **WHEN** `set_checklist_field "$file" invalid_field 5` is called
- **THEN** the file is NOT modified
- **AND** stderr contains an error about invalid field
- **AND** exit code is 1

#### Scenario: Invalid value type for generated

- **GIVEN** any `.status.yaml`
- **WHEN** `set_checklist_field "$file" generated 42` is called
- **THEN** the file is NOT modified
- **AND** stderr contains an error about invalid value
- **AND** exit code is 1

### Requirement: set_confidence_block

`set_confidence_block <status_file> <certain> <confident> <tentative> <unresolved> <score>` SHALL replace the entire `confidence:` block in `.status.yaml`.

The function MUST validate that `<status_file>` exists before any parsing.

The function MUST:
1. Validate all count arguments are non-negative integers
2. Validate score is a non-negative float
3. Replace the entire confidence block (from `confidence:` to the next top-level key)
4. Update `last_updated`
5. Use temp-file-then-mv for atomic writes

#### Scenario: Write confidence block

- **GIVEN** a `.status.yaml` with default confidence values
- **WHEN** `set_confidence_block "$file" 12 3 2 0 2.1` is called
- **THEN** the confidence block contains `certain: 12`, `confident: 3`, `tentative: 2`, `unresolved: 0`, `score: 2.1`
- **AND** `last_updated` is refreshed

#### Scenario: Invalid count (negative)

- **GIVEN** any `.status.yaml`
- **WHEN** `set_confidence_block "$file" -1 3 2 0 2.1` is called
- **THEN** the file is NOT modified
- **AND** exit code is 1

#### Scenario: Non-numeric score

- **GIVEN** any `.status.yaml`
- **WHEN** `set_confidence_block "$file" 12 3 2 0 abc` is called
- **THEN** the file is NOT modified
- **AND** stderr contains an error about invalid score value
- **AND** exit code is 1

## Stageman: Atomic Write Mechanism

### Requirement: Temp-file-then-mv pattern

All write functions SHALL use the temp-file-then-mv pattern for atomicity:
1. Create a temp file (via `mktemp`) in the same directory as the target file
2. Write the modified content to the temp file
3. `mv` the temp file over the target file

This MUST prevent partial writes on interruption. The temp file SHOULD be created in the same directory as the target to ensure `mv` is a rename (not a cross-device copy).

#### Scenario: Interrupted write

- **GIVEN** a write function is in progress
- **WHEN** the process is interrupted after temp file creation but before mv
- **THEN** the original `.status.yaml` is unchanged
- **AND** only a temp file remains (can be cleaned up)

### Requirement: last_updated auto-refresh

Every write function SHALL update the `last_updated` field to the current ISO 8601 timestamp (with timezone offset, matching the format used by `/fab-new`: `YYYY-MM-DDTHH:MM:SS+HH:MM`).

#### Scenario: Timestamp format

- **GIVEN** any write function is called successfully
- **WHEN** the write completes
- **THEN** `last_updated` matches the pattern `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}`

## Stageman: CLI Write Commands

### Requirement: CLI dispatch for write operations

When `_stageman.sh` is executed directly (not sourced), it SHALL support write subcommands in addition to the existing `--help`, `--test`, `--version` flags.

The CLI SHALL support:
- `_stageman.sh set-state <file> <stage> <state>` — calls `set_stage_state`
- `_stageman.sh transition <file> <from> <to>` — calls `transition_stages`
- `_stageman.sh set-checklist <file> <field> <value>` — calls `set_checklist_field`
- `_stageman.sh set-confidence <file> <certain> <confident> <tentative> <unresolved> <score>` — calls `set_confidence_block`

Each CLI command SHALL validate argument count, delegate to the corresponding function, and exit with the function's return code.

#### Scenario: CLI set-state

- **GIVEN** `_stageman.sh` is invoked as `_stageman.sh set-state /path/.status.yaml spec active`
- **WHEN** the command executes
- **THEN** `set_stage_state` is called with the provided arguments
- **AND** exit code matches the function's return code

#### Scenario: CLI missing arguments

- **GIVEN** `_stageman.sh` is invoked as `_stageman.sh set-state /path/.status.yaml`
- **WHEN** the command executes
- **THEN** stderr shows usage help for the subcommand
- **AND** exit code is 1

#### Scenario: CLI help updated

- **GIVEN** `_stageman.sh --help` is invoked
- **WHEN** the help text is displayed
- **THEN** all write subcommands are listed under a "Write commands" section

## _calc-score.sh: Delegate Write to Stageman

### Requirement: Refactor _calc-score.sh write logic

`_calc-score.sh` SHALL delegate its `.status.yaml` write to `set_confidence_block` from `_stageman.sh` instead of using inline awk.

The script MUST:
1. Source `_stageman.sh` (it already resolves its directory via `BASH_SOURCE`)
2. Replace the inline awk block (lines ~120-141 of current `_calc-score.sh`) with a call to `set_confidence_block`
3. Preserve the stdout YAML output (including delta) unchanged — the emit block is separate from the write and SHALL remain inline

The scoring logic (grade parsing, counting, carry-forward, formula) SHALL remain in `_calc-score.sh` — only the file I/O is delegated.

**Note — behavioral change**: The current `_calc-score.sh` does NOT update `last_updated` when writing the confidence block. After this refactor, `set_confidence_block` will refresh `last_updated` on every score computation. This is intentional — any `.status.yaml` mutation SHOULD refresh `last_updated` for consistency.

#### Scenario: Identical output after refactor

- **GIVEN** a change directory with brief.md and spec.md containing Assumptions tables
- **WHEN** `_calc-score.sh <change-dir>` is run
- **THEN** the stdout YAML output is identical to the pre-refactor version
- **AND** the `.status.yaml` confidence block is identical to the pre-refactor version

#### Scenario: Sourcing stageman

- **GIVEN** `_calc-score.sh` is invoked
- **WHEN** it sources `_stageman.sh`
- **THEN** all stageman functions are available
- **AND** no conflict with existing variables or functions in `_calc-score.sh`

## Skill Prompt Updates

### Requirement: Replace ad-hoc YAML editing with Bash calls

Skill prompts that currently instruct the LLM to edit `.status.yaml` directly (via the Edit tool) SHALL be updated to instruct the LLM to call stageman CLI commands via the Bash tool instead.

The affected skills are:
- **`fab-continue.md`** — Step 4 (Update `.status.yaml`) currently describes ad-hoc editing of progress fields; SHALL be replaced with `_stageman.sh transition` or `_stageman.sh set-state` calls
- **`fab-ff.md`** and **`fab-fff.md`** — describe stage transitions inline as part of their full-pipeline behavior; SHOULD be updated to reference `_stageman.sh transition` or `_stageman.sh set-state` for consistency with `fab-continue.md`
<!-- clarified: fab-ff/fab-fff moved from "do NOT need changes" to affected skills — they describe transitions inline and should reference CLI -->
- **`fab-new.md`** — `.status.yaml` initialization; MAY remain as template-copy since it creates the file from scratch (not a mutation)
<!-- assumed: fab-new.md may not need changes — it initializes .status.yaml from a template copy, which is creation not mutation. The write API is for updating existing files. -->

Skills that do NOT need changes:
- **`fab-switch.md`** — does not modify `.status.yaml`
- **`fab-status.md`** — read-only
- **`fab-clarify.md`** — does not modify progress fields (only artifact content)

#### Scenario: fab-continue uses stageman CLI

- **GIVEN** the updated `fab-continue.md` skill prompt
- **WHEN** an LLM agent reads Step 4 instructions
- **THEN** the instructions reference `_stageman.sh transition` or `_stageman.sh set-state` commands
- **AND** no instructions for direct `.status.yaml` editing via Edit tool remain for progress/checklist/confidence fields

#### Scenario: fab-ff/fab-fff reference stageman CLI

- **GIVEN** the updated `fab-ff.md` and `fab-fff.md` skill prompts
- **WHEN** an LLM agent reads stage transition instructions
- **THEN** the instructions reference `_stageman.sh` CLI commands for state changes

## Design Decisions

1. **Extend `_stageman.sh` in-place**: Add write functions to the existing file rather than creating a new script. Stageman is already the single owner of `.status.yaml` read semantics; owning writes too maintains the "one file, one concern" principle. The alternative (`_stageman_write.sh`) would split the API across files for no benefit.
   - *Rejected*: Separate write script — fragments the API, requires sourcing two files.

2. **CLI subcommands over flags**: Write operations use positional subcommands (`set-state`, `transition`) rather than flags (`--set-state`). This matches how skills invoke via Bash (cleaner argument passing) and avoids conflicts with existing `--help`/`--test`/`--version` flags.
   - *Rejected*: Flag-based write invocation — ambiguous argument binding, conflicts with existing flags.

3. **Validate-then-write, not write-then-validate**: All write functions validate inputs before touching the file. This avoids partial writes from validation failures mid-write.
   - *Rejected*: Post-write validation — leaves corrupted file on failure.

4. **`transition_stages` enforces adjacency**: The function verifies that `to_stage` is the `get_next_stage` of `from_stage`. This prevents accidental stage-skipping. Reset flows (which skip stages) use `set_stage_state` directly.
   - *Rejected*: Allowing arbitrary stage pairs in `transition_stages` — loses the safety guarantee that transitions follow the pipeline graph.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | CLI uses subcommand pattern (`set-state`, `transition`, etc.) rather than flag pattern | Brief example shows `_stageman.sh transition <file> <from> <to>` — positional subcommand style. Consistent with how LLM agents invoke Bash commands. |
| 2 | Confident | `fab-new.md` may not need changes (template-copy creation, not mutation) | fab-new creates `.status.yaml` from the template file; the write API is designed for updating existing files. Creating from template is idempotent and well-defined. |
| 3 | Certain | `fab-ff.md` and `fab-fff.md` should also reference stageman CLI | Clarified: moved to affected skills list — they describe transitions inline and need CLI references. |
| 4 | Confident | `transition_stages` validates adjacency (next-stage check) | The two-write transition is specifically for forward pipeline flow. Resets use `set_stage_state` directly, which doesn't enforce adjacency. Prevents accidental stage-skipping in normal flow. |

4 assumptions made (4 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-14

Q: Should `transition_stages` validate the current state of `from_stage` before setting it to `done`?
A: Require `active` — `from_stage` must currently be `active` in the file; reject otherwise.

Q: Should write functions validate that `<status_file>` exists before any parsing?
A: Validate and fail fast — check file existence at the top of each write function; exit 1 with diagnostic.

Q: The `_calc-score.sh` refactor introduces `last_updated` refresh. Is this intentional?
A: Accept the change — any `.status.yaml` mutation should refresh `last_updated` for consistency.

Q: The scenario "Invalid target state" for `transition_stages` is actually a valid case. Rename?
A: Renamed to "Last pipeline transition" to match the scenario's actual behavior.

Q: `fab-ff.md` and `fab-fff.md` listed under "do NOT need changes" but spec says they SHOULD reference CLI. Contradictory?
A: Move to affected skills — reclassified as skills that need updates. Assumption #3 reclassified to Certain.

Q: Add a non-numeric score validation scenario for `set_confidence_block`?
A: Added "Non-numeric score" scenario with `abc` as invalid input.
