# Spec: Centralize Current Pointer Format

**Change**: 260302-a8ay-centralize-current-pointer
**Created**: 2026-03-02
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md`, `docs/memory/fab-workflow/kit-scripts.md`

## Non-Goals

- Backward compatibility with old single-line `fab/current` format — this is a clean cut, not a migration
- Changing `changeman.sh new` — it calls `statusman.sh start` (which doesn't touch `fab/current`), so no change needed
- Changing `changeman.sh list` — it scans directories, never reads `fab/current`

## Active Change Pointer Format

### Requirement: Two-Line Plain Text Format

`fab/current` SHALL use a two-line plain text format:

- **Line 1**: 4-char change ID (e.g., `ye59`)
- **Line 2**: Full folder name (e.g., `260302-ye59-tu-fresh-flag-reduced-ttl`)

There SHALL be no trailing newline after line 2. The file SHALL contain no YAML, no delimiters, and no metadata beyond these two lines.

#### Scenario: Fresh switch writes both lines
- **GIVEN** a change `260302-ye59-example-change` exists in `fab/changes/`
- **WHEN** `changeman.sh switch ye59` is executed
- **THEN** `fab/current` line 1 SHALL be `ye59`
- **AND** `fab/current` line 2 SHALL be `260302-ye59-example-change`
- **AND** no trailing newline SHALL follow line 2

#### Scenario: Reading the file with standard tools
- **GIVEN** `fab/current` contains a valid two-line entry
- **WHEN** line 1 is read via `sed -n '1p'` or `head -1`
- **THEN** the output SHALL be the 4-char change ID
- **AND** when line 2 is read via `sed -n '2p'` or `tail -1`, the output SHALL be the full folder name

### Requirement: Centralized Access Pattern

Only `resolve.sh` SHALL read `fab/current` (for default-mode resolution). Only `changeman.sh` SHALL write `fab/current` (via `switch` and `rename` subcommands). No skill, no other script, and no agent instruction SHALL read or write `fab/current` directly.

#### Scenario: Skills delegate to resolve.sh
- **GIVEN** `fab-discuss` needs to know the active change
- **WHEN** it checks for an active change
- **THEN** it SHALL call `resolve.sh` (or `changeman.sh resolve`) instead of reading `fab/current` directly

#### Scenario: Skills delegate to changeman.sh for writes
- **GIVEN** `fab-archive` needs to clear the active pointer
- **WHEN** it deactivates the current change
- **THEN** it SHALL call `changeman.sh switch --blank` instead of deleting `fab/current` directly

## Script: resolve.sh

### Requirement: Default Mode Reads Line 2

When no override argument is provided, `resolve.sh` SHALL read line 2 of `fab/current` for the folder name. The current `tr -d '[:space:]' < "$current_file"` approach SHALL be replaced with line-specific reading.

The `extract_id` function and `--id` output mode SHALL remain unchanged — they derive the ID from the folder name, not from `fab/current` line 1.

#### Scenario: Default mode returns folder name from line 2
- **GIVEN** `fab/current` contains `ye59\n260302-ye59-example-change`
- **WHEN** `resolve.sh --folder` is called with no override
- **THEN** the output SHALL be `260302-ye59-example-change`

#### Scenario: Default mode with --id extracts from line 2 folder name
- **GIVEN** `fab/current` contains `ye59\n260302-ye59-example-change`
- **WHEN** `resolve.sh --id` is called with no override
- **THEN** the output SHALL be `ye59` (extracted via `extract_id` from line 2)

#### Scenario: Empty or missing fab/current falls back to single-change guess
- **GIVEN** `fab/current` does not exist or is empty
- **WHEN** `resolve.sh` is called with no override
- **THEN** the single-change guess fallback SHALL work as before (scan `fab/changes/` for single valid candidate)

#### Scenario: Whitespace handling on line 2
- **GIVEN** `fab/current` line 2 has trailing whitespace
- **WHEN** `resolve.sh --folder` is called with no override
- **THEN** the trailing whitespace SHALL be stripped before returning the folder name

### Requirement: Override Mode Unchanged

When an override argument is provided, `resolve.sh` SHALL continue to match against `fab/changes/` folder names using case-insensitive substring matching. `fab/current` is not consulted in override mode.

#### Scenario: Override bypasses fab/current
- **GIVEN** `fab/current` contains `ye59\n260302-ye59-old-change`
- **WHEN** `resolve.sh --folder a8ay` is called
- **THEN** the output SHALL be the folder matching `a8ay`, not the one in `fab/current`

## Script: changeman.sh

### Requirement: Switch Writes Both Lines

`changeman.sh switch <name>` SHALL write `fab/current` with the 4-char ID on line 1 and the full folder name on line 2. The ID SHALL be extracted from the resolved folder name using `extract_id` (or equivalent `cut -d'-' -f2` logic).

The current `printf '%s' "$resolved" > "$FAB_ROOT/current"` SHALL be replaced with a two-line write.

#### Scenario: Switch writes two-line format
- **GIVEN** `changeman.sh switch a8ay` resolves to `260302-a8ay-centralize-current-pointer`
- **WHEN** the switch completes
- **THEN** `fab/current` line 1 SHALL be `a8ay`
- **AND** `fab/current` line 2 SHALL be `260302-a8ay-centralize-current-pointer`

#### Scenario: Switch --blank unchanged
- **GIVEN** `fab/current` exists
- **WHEN** `changeman.sh switch --blank` is executed
- **THEN** `fab/current` SHALL be deleted (no format change — same behavior as before)

### Requirement: Rename Updates Line 2 Only

`changeman.sh rename` SHALL update `fab/current` line 2 to the new folder name when the active change is being renamed. Line 1 (the 4-char ID) SHALL remain unchanged because the `{YYMMDD}-{XXXX}` prefix is immutable across renames.

The comparison to detect "is this the active change?" SHALL use the ID from line 1 (or extract from the stored folder name) rather than comparing full folder names.

#### Scenario: Rename updates line 2, preserves line 1
- **GIVEN** `fab/current` contains `a8ay\n260302-a8ay-old-slug`
- **WHEN** `changeman.sh rename --folder 260302-a8ay-old-slug --slug new-slug` is executed
- **THEN** `fab/current` line 1 SHALL remain `a8ay`
- **AND** `fab/current` line 2 SHALL be `260302-a8ay-new-slug`

#### Scenario: Rename of non-active change leaves fab/current untouched
- **GIVEN** `fab/current` contains `x1y2\n260302-x1y2-other-change`
- **WHEN** `changeman.sh rename --folder 260302-a8ay-old-slug --slug new-slug` is executed
- **THEN** `fab/current` SHALL remain unchanged (both lines)

## Script: logman.sh

### Requirement: Remove Direct fab/current Read

The `command` subcommand's no-change-arg path SHALL delegate to `resolve.sh` instead of reading `fab/current` directly. The same best-effort, silent-exit-on-failure behavior SHALL be preserved: if resolution fails for any reason, `logman.sh` SHALL exit 0 silently.

The direct-read code block (lines 75–80: `current_file`, `tr -d '[:space:]'`, `resolve_change_dir`) SHALL be replaced with a single `resolve.sh` call wrapped in the same silent-failure guard.

#### Scenario: No change arg resolves via resolve.sh
- **GIVEN** `fab/current` contains a valid two-line entry
- **WHEN** `logman.sh command "fab-discuss"` is called (no change arg)
- **THEN** logman SHALL resolve the change via `resolve.sh --dir` (not direct file read)
- **AND** the command event SHALL be appended to `.history.jsonl`

#### Scenario: No change arg, fab/current missing — silent exit
- **GIVEN** `fab/current` does not exist
- **WHEN** `logman.sh command "fab-discuss"` is called
- **THEN** logman SHALL exit 0 silently (no error, no log entry)

#### Scenario: No change arg, stale fab/current — silent exit
- **GIVEN** `fab/current` contains a name that no longer resolves (deleted change)
- **WHEN** `logman.sh command "fab-discuss"` is called
- **THEN** logman SHALL exit 0 silently

## Script: dispatch.sh

### Requirement: Poll Line 1 for Switch Confirmation

`dispatch.sh` Step 3 (poll `fab/current` after sending `/fab-switch`) SHALL compare against line 1 (the 4-char ID) instead of the full file content.

The `CHANGE_ID` variable in dispatch.sh currently holds the full folder name (`fs-change-id` argument). The comparison SHALL extract the 4-char ID from `CHANGE_ID` (via `cut -d'-' -f2` or equivalent) and compare it against line 1 of `fab/current`.

#### Scenario: Polling matches 4-char ID
- **GIVEN** dispatch.sh was called with `fs-change-id` of `260302-ye59-example`
- **WHEN** `/fab-switch` completes in the interactive pane and `fab/current` line 1 becomes `ye59`
- **THEN** the poll SHALL detect a match and proceed to Step 4

#### Scenario: Polling reads line 1 only
- **GIVEN** `fab/current` contains `ye59\n260302-ye59-example`
- **WHEN** the poll reads `fab/current`
- **THEN** it SHALL compare `ye59` (line 1) against the expected 4-char ID, not the full content

## Script: preflight.sh

### Requirement: Emit id Field in YAML Output

`preflight.sh` SHALL emit an `id:` field in its YAML output, positioned before the `name:` field. The value SHALL be the 4-char change ID extracted from the resolved folder name via `resolve.sh --id` (or `extract_id` logic).

```yaml
id: ye59
name: 260302-ye59-example-change
change_dir: fab/changes/260302-ye59-example-change
```

#### Scenario: YAML output includes id field
- **GIVEN** preflight resolves to change `260302-ye59-example`
- **WHEN** preflight emits its YAML output
- **THEN** the output SHALL contain `id: ye59` before the `name:` line

#### Scenario: id field matches 4-char portion of name
- **GIVEN** preflight resolves to change `260302-a8ay-centralize-current-pointer`
- **WHEN** the `id` and `name` fields are parsed
- **THEN** `id` SHALL equal the second segment of `name` split by `-` (i.e., `a8ay`)

## Skill: _preamble.md

### Requirement: Agent Uses id for Script Calls

`_preamble.md` §2 (Change Context) SHALL instruct the agent to parse the `id` field from preflight's YAML output and use it (not `name`) when calling scripts. The `name` field SHALL remain available for display, path construction, and artifact metadata.

Specifically, step 4 SHALL change from:
```bash
bash fab/.kit/scripts/lib/logman.sh command "<skill-name>" "<name>"
```
to:
```bash
bash fab/.kit/scripts/lib/logman.sh command "<skill-name>" "<id>"
```

And all script invocation examples SHALL use `<id>` instead of `<name>`.

#### Scenario: Agent calls scripts with 4-char ID
- **GIVEN** preflight YAML contains `id: a8ay` and `name: 260302-a8ay-centralize-current-pointer`
- **WHEN** the agent calls `statusman.sh finish` for the intake stage
- **THEN** it SHALL use `bash fab/.kit/scripts/lib/statusman.sh finish a8ay intake fab-continue`
- **AND** NOT `bash fab/.kit/scripts/lib/statusman.sh finish 260302-a8ay-centralize-current-pointer intake fab-continue`

#### Scenario: Agent uses name for display and paths
- **GIVEN** preflight YAML contains `id: a8ay` and `name: 260302-a8ay-centralize-current-pointer`
- **WHEN** the agent reads an artifact file
- **THEN** it SHALL use `fab/changes/260302-a8ay-centralize-current-pointer/intake.md` (derived from `change_dir` or `name`, not `id`)

## Skill: fab-discuss

### Requirement: Use resolve.sh for Active Change Check

`fab-discuss` SHALL NOT read `fab/current` directly. Instead, it SHALL call `resolve.sh --folder` (or `changeman.sh resolve`) to determine the active change name. If resolution fails (exit non-zero), note "No active change."

The current instruction "Read `fab/current` — if the file does not exist or is empty, note 'No active change'" SHALL be replaced with a `resolve.sh` call with the same fallback behavior.

#### Scenario: Active change detected via resolve.sh
- **GIVEN** `fab/current` contains a valid entry
- **WHEN** `fab-discuss` checks for an active change
- **THEN** it SHALL call `resolve.sh --folder` and use the result
- **AND** it SHALL NOT read `fab/current` directly

#### Scenario: No active change
- **GIVEN** `fab/current` does not exist
- **WHEN** `fab-discuss` checks for an active change
- **THEN** `resolve.sh` SHALL exit non-zero and `fab-discuss` SHALL note "No active change"

## Skill: fab-archive

### Requirement: Use changeman.sh for Pointer Operations

`fab-archive` SHALL NOT read, write, or delete `fab/current` directly. All pointer operations SHALL go through `changeman.sh`:

1. **Check active change**: Use `changeman.sh resolve` (or `resolve.sh --folder`) instead of reading `fab/current`
2. **Clear pointer**: Use `changeman.sh switch --blank` instead of `rm fab/current`
3. **Activate on restore**: Use `changeman.sh switch <name>` instead of writing `fab/current`

#### Scenario: Archive clears pointer via changeman
- **GIVEN** `fab/current` points to the change being archived
- **WHEN** `fab-archive` Step 5 clears the pointer
- **THEN** it SHALL call `changeman.sh switch --blank`
- **AND** it SHALL NOT directly delete `fab/current`

#### Scenario: Restore with --switch activates via changeman
- **GIVEN** user runs `fab-archive restore <name> --switch`
- **WHEN** Step 3 activates the restored change
- **THEN** it SHALL call `changeman.sh switch <name>`
- **AND** it SHALL NOT directly write to `fab/current`

#### Scenario: Archive checks active change via resolve
- **GIVEN** `fab/current` points to a different change
- **WHEN** `fab-archive` Step 5 checks whether the archived change is active
- **THEN** it SHALL compare the archived change name against `changeman.sh resolve` output
- **AND** it SHALL NOT read `fab/current` directly

## Tests

### Requirement: Existing Tests Updated for Two-Line Format

All existing tests that write `fab/current` in single-line format SHALL be updated to use the two-line format. All assertions that read `fab/current` SHALL assert the correct line content.

#### Scenario: resolve test helpers write two-line format
- **GIVEN** `src/lib/resolve/test.bats` creates `fab/current` for testing
- **WHEN** the test writes to `fab/current`
- **THEN** it SHALL write both the 4-char ID (line 1) and the full folder name (line 2)

#### Scenario: changeman switch test asserts both lines
- **GIVEN** `src/lib/changeman/test.bats` tests the switch subcommand
- **WHEN** the test asserts `fab/current` content
- **THEN** it SHALL verify line 1 is the 4-char ID and line 2 is the full folder name

#### Scenario: preflight test helpers write two-line format
- **GIVEN** `src/lib/preflight/test.bats` has a `set_current` helper
- **WHEN** the helper writes `fab/current`
- **THEN** it SHALL write the two-line format

#### Scenario: logman test writes two-line format
- **GIVEN** `src/lib/logman/test.bats` tests the command subcommand with fab/current
- **WHEN** the test writes `fab/current`
- **THEN** it SHALL use the two-line format

### Requirement: New Tests for Format and Behavior

New test cases SHALL be added to validate the two-line format behavior and the `id` field in preflight output.

#### Scenario: New resolve test — two-line format folder name
- **GIVEN** `fab/current` is written in two-line format
- **WHEN** `resolve.sh --folder` is called with no override
- **THEN** it SHALL return the folder name from line 2

#### Scenario: New changeman test — switch writes correct lines
- **GIVEN** a change folder exists
- **WHEN** `changeman.sh switch` is executed
- **THEN** `sed -n '1p' fab/current` SHALL return the 4-char ID
- **AND** `sed -n '2p' fab/current` SHALL return the full folder name

#### Scenario: New changeman test — rename preserves line 1
- **GIVEN** `fab/current` points to the change being renamed
- **WHEN** `changeman.sh rename` completes
- **THEN** line 1 SHALL be unchanged (same 4-char ID)
- **AND** line 2 SHALL reflect the new folder name

#### Scenario: New preflight test — id field present
- **GIVEN** a valid change exists
- **WHEN** `preflight.sh` is run
- **THEN** the YAML output SHALL contain an `id:` field matching the 4-char portion of the change name

#### Scenario: New dispatch test — poll matches line 1
- **GIVEN** `dispatch.sh` polls `fab/current` after switch
- **WHEN** `fab/current` contains the expected change's ID on line 1
- **THEN** the poll SHALL detect a match

## Design Decisions

### 1. Two-Line Plain Text over YAML or Single-ID
**Decision**: `fab/current` uses two-line plain text (ID on line 1, folder name on line 2).
**Why**: Zero parsing dependencies — readable with `sed`/`head`/`tail`. Line 2 gives `resolve.sh` zero-cost folder-name reads (no directory scan). Line 1 gives `dispatch.sh` a short comparison value. YAML would require `yq` on the hot path. Single-ID-only would require a folder scan on every default-mode `resolve.sh` call.
**Rejected**: YAML (`yq` dependency in performance-critical path). Single-ID-only (requires folder scan on every resolve, breaks zero-cost reads).

### 2. Centralized Access via resolve.sh/changeman.sh
**Decision**: Only `resolve.sh` reads `fab/current`, only `changeman.sh` writes it. All other scripts and skills delegate.
**Why**: Encapsulates the file format as an implementation detail. Future format changes only require updating two scripts. Currently three non-resolve readers exist (`logman.sh`, `fab-discuss`, `fab-archive`) — this change eliminates all of them.
**Rejected**: Allowing direct reads "when convenient" — increases coupling surface, format changes become audit exercises.

### 3. No Backward Compatibility Layer
**Decision**: Clean cut to two-line format. No fallback to read single-line format.
**Why**: All `fab/current` writers go through `changeman.sh switch`, which will write the new format after this change. There's no migration scenario where a new `resolve.sh` sees an old-format file — `fab/current` is transient local state (gitignored), not persisted across installations. The moment a user switches to a change after upgrading, the file is rewritten in the new format.
**Rejected**: Backward compatibility detection (adds complexity for a scenario that can't happen in practice).

## Deprecated Requirements

### Direct `fab/current` Reads in logman.sh

**Reason**: logman.sh `command` subcommand's direct-read path (lines 75–80) is replaced with `resolve.sh` delegation to enforce the centralized access pattern.
**Migration**: `resolve.sh --dir` with silent-failure wrapping provides identical behavior.

### Direct `fab/current` Reads in fab-discuss

**Reason**: `fab-discuss` skill instructions that read `fab/current` directly are replaced with `resolve.sh` calls.
**Migration**: `resolve.sh --folder` (or `changeman.sh resolve`) provides identical resolution with single-change guess fallback.

### Direct `fab/current` Reads/Writes in fab-archive

**Reason**: `fab-archive` skill instructions that read/delete/write `fab/current` are replaced with `changeman.sh` calls.
**Migration**: `changeman.sh resolve` for reads, `changeman.sh switch --blank` for clears, `changeman.sh switch <name>` for writes.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Two-line plain text format (ID line 1, folder name line 2) | Confirmed from intake #1 — user chose over YAML and single-ID-only | S:95 R:85 A:90 D:95 |
| 2 | Certain | Skills must not read fab/current directly | Confirmed from intake #2 — user stated "zero contact with implementation details" | S:95 R:80 A:90 D:95 |
| 3 | Certain | resolve.sh reads line 2 for folder name | Confirmed from intake #3 — preserves zero-cost reads | S:90 R:90 A:85 D:90 |
| 4 | Certain | changeman.sh rename: ID unchanged, update folder name only | Confirmed from intake #4 — YYMMDD-XXXX prefix immutable | S:90 R:90 A:90 D:95 |
| 5 | Certain | dispatch.sh polls line 1 for switch confirmation | Confirmed from intake #5 — simpler short-string comparison | S:85 R:85 A:85 D:90 |
| 6 | Certain | logman.sh delegates to resolve.sh | Confirmed from intake #6 — same centralization principle | S:85 R:85 A:85 D:90 |
| 7 | Certain | fab-archive uses changeman.sh for pointer ops | Upgraded from intake #7 Confident — spec analysis confirms changeman.sh has all needed subcommands | S:90 R:80 A:85 D:90 |
| 8 | Confident | No trailing newline after line 2 | Confirmed from intake #8 — follows existing `printf '%s'` convention | S:70 R:95 A:80 D:85 |
| 9 | Certain | preflight.sh emits id: field | Confirmed from intake #9 — user chose over agent-side extraction | S:90 R:90 A:90 D:95 |
| 10 | Certain | Preamble §2 uses id for script calls | Confirmed from intake #10 — root fix for agent output noise | S:95 R:85 A:90 D:95 |
| 11 | Certain | No backward compatibility for old single-line format | Spec-level analysis: fab/current is gitignored, transient, rewritten on every switch — old format can't persist | S:85 R:90 A:90 D:95 |
| 12 | Confident | dispatch.sh extracts 4-char ID from CHANGE_ID variable | Spec-level: CHANGE_ID is the `fs-change-id` (full folder name), need `cut -d'-' -f2` to get 4-char ID for comparison | S:75 R:90 A:80 D:85 |

12 assumptions (10 certain, 2 confident, 0 tentative, 0 unresolved).
