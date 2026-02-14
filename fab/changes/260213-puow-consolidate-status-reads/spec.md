# Spec: Consolidate .status.yaml Ownership into Stageman

**Change**: 260213-puow-consolidate-status-reads
**Created**: 2026-02-14
**Affected memory**: `fab/memory/fab-workflow/preflight.md` (modify), `fab/memory/fab-workflow/kit-architecture.md` (modify)

## Non-Goals

- Write/mutation functions for `.status.yaml` — this change is read-only accessors only
- Changing the output format of `fab-preflight.sh` or `fab-status.sh` — behavior is identical before and after
- Modifying `_fab-scaffold.sh`, agent files, or any skill markdown

## Stageman: .status.yaml Accessor API

### Requirement: Progress Map Accessor

`_stageman.sh` SHALL provide a `get_progress_map` function that extracts all stage→state pairs from a `.status.yaml` file. The function SHALL accept a status file path as its argument and output one `stage:state` pair per line (e.g., `brief:active`). Consumers iterate the output to populate their own data structures.

#### Scenario: Extract progress from a valid status file
- **GIVEN** a `.status.yaml` with `brief: active`, `spec: pending`, `tasks: pending`, `apply: pending`, `review: pending`, `hydrate: pending`
- **WHEN** `get_progress_map "$status_file"` is called
- **THEN** stdout contains six lines: `brief:active`, `spec:pending`, `tasks:pending`, `apply:pending`, `review:pending`, `hydrate:pending`

#### Scenario: Missing stage in status file
- **GIVEN** a `.status.yaml` that is missing the `tasks` progress entry
- **WHEN** `get_progress_map "$status_file"` is called
- **THEN** the `tasks` line outputs `tasks:pending` (defaults to `pending`)

### Requirement: Checklist Accessor

`_stageman.sh` SHALL provide a `get_checklist` function that extracts checklist fields from a `.status.yaml` file. The function SHALL accept a status file path as its argument and output three lines: `generated:{value}`, `completed:{value}`, `total:{value}`. Missing fields SHALL default to `false`, `0`, `0` respectively.

#### Scenario: Extract checklist from status file with checklist block
- **GIVEN** a `.status.yaml` with `generated: true`, `completed: 3`, `total: 10`
- **WHEN** `get_checklist "$status_file"` is called
- **THEN** stdout contains: `generated:true`, `completed:3`, `total:10`

#### Scenario: Missing checklist block
- **GIVEN** a `.status.yaml` with no checklist section
- **WHEN** `get_checklist "$status_file"` is called
- **THEN** stdout contains: `generated:false`, `completed:0`, `total:0`

### Requirement: Confidence Accessor

`_stageman.sh` SHALL provide a `get_confidence` function that extracts confidence fields from a `.status.yaml` file. The function SHALL accept a status file path as its argument and output five lines: `certain:{value}`, `confident:{value}`, `tentative:{value}`, `unresolved:{value}`, `score:{value}`. Missing fields SHALL default to `0`, `0`, `0`, `0`, `5.0` respectively.

#### Scenario: Extract confidence from status file with confidence block
- **GIVEN** a `.status.yaml` with `certain: 5`, `confident: 4`, `tentative: 0`, `unresolved: 0`, `score: 3.8`
- **WHEN** `get_confidence "$status_file"` is called
- **THEN** stdout contains: `certain:5`, `confident:4`, `tentative:0`, `unresolved:0`, `score:3.8`

#### Scenario: Missing confidence block (backwards compat)
- **GIVEN** a `.status.yaml` with no confidence section
- **WHEN** `get_confidence "$status_file"` is called
- **THEN** stdout contains: `certain:0`, `confident:0`, `tentative:0`, `unresolved:0`, `score:5.0`

### Requirement: Refactor get_current_stage to Use Accessor

`get_current_stage` in `_stageman.sh` SHALL be refactored to use `get_progress_map` internally instead of its current raw `grep | sed` parsing. Behavior SHALL be identical — the same fallback logic (active → first pending after last done → hydrate) is preserved.

#### Scenario: get_current_stage uses get_progress_map internally
- **GIVEN** a `.status.yaml` with `brief: done`, `spec: active`
- **WHEN** `get_current_stage "$status_file"` is called
- **THEN** the result is `spec`, derived by iterating `get_progress_map` output instead of raw grep

#### Scenario: Fallback behavior preserved
- **GIVEN** a `.status.yaml` with `brief: done`, all other stages `pending`, none `active`
- **WHEN** `get_current_stage "$status_file"` is called
- **THEN** the result is `spec` (first pending after last done), identical to pre-refactor

### Requirement: Rename to Underscore Prefix

`stageman.sh` SHALL be renamed to `_stageman.sh` to follow the internal library naming convention (underscore prefix = internal/sourced, `fab-` prefix = entry point). All scripts that source `stageman.sh` SHALL be updated to source `_stageman.sh`.

#### Scenario: Existing scripts source the renamed file
- **GIVEN** `fab-preflight.sh` sources `_stageman.sh`
- **WHEN** preflight runs
- **THEN** all stageman functions are available and behave identically to before the rename

#### Scenario: Dev symlink updated after rename
- **GIVEN** `src/stageman/stageman.sh` was a symlink to `../../fab/.kit/scripts/stageman.sh`
- **WHEN** the rename to `_stageman.sh` is applied
- **THEN** the dev symlink is updated to `src/stageman/_stageman.sh` → `../../fab/.kit/scripts/_stageman.sh`
- **AND** the old `src/stageman/stageman.sh` symlink is removed

## Preflight: Delegate to Stageman

### Requirement: Replace Inline Progress Extraction

`fab-preflight.sh` SHALL replace its inline `grep | sed` progress extraction loop (current lines ~108-112, the `declare -A progress` block) with a call to `get_progress_map` from `_stageman.sh`. The emitted YAML output SHALL be identical.

#### Scenario: Preflight output unchanged after refactor
- **GIVEN** a change `260213-puow-consolidate-status-reads` with `brief: active` and all other stages `pending`
- **WHEN** `fab-preflight.sh` is executed
- **THEN** the stdout YAML progress block is identical to the pre-refactor output

### Requirement: Replace Inline Stage Derivation

`fab-preflight.sh` SHALL replace its inline "derive current stage" loop (current lines ~114-146) with a call to `get_current_stage` from `_stageman.sh`. This function already exists in stageman — preflight currently reimplements it. `get_current_stage` itself SHALL be refactored to use `get_progress_map` internally instead of raw `grep | sed`, for consistency with the accessor API it sits alongside.

#### Scenario: Stage derivation delegates to stageman
- **GIVEN** a status file with `brief: done`, `spec: active`
- **WHEN** `fab-preflight.sh` runs
- **THEN** the `stage:` output is `spec`, derived via `get_current_stage`

### Requirement: Replace Inline Checklist and Confidence Extraction

`fab-preflight.sh` SHALL replace its inline `grep | sed` extraction of checklist fields (current lines ~149-151) and confidence fields (current lines ~153-158) with calls to `get_checklist` and `get_confidence` from `_stageman.sh`.

#### Scenario: Checklist and confidence fields via stageman
- **GIVEN** a status file with checklist `generated: true`, `completed: 5`, `total: 12` and confidence `score: 3.8`
- **WHEN** `fab-preflight.sh` runs
- **THEN** the emitted YAML checklist and confidence blocks match the pre-refactor output exactly

## Status: Delegate to Stageman

### Requirement: Remove get_field/get_nested Helpers

`fab-status.sh` SHALL remove its local `get_field` and `get_nested` helper functions (current lines 98-99) and replace all their usages with stageman accessor calls. The `created_by` field extraction (which is not stage/checklist/confidence related) MAY use a direct `grep` since it falls outside stageman's scope.

#### Scenario: Status output unchanged after removing helpers
- **GIVEN** a change with `created_by: sahil-weaver`
- **WHEN** `fab-status.sh` is executed
- **THEN** the "Created by: sahil-weaver" line appears in output, identical to before

### Requirement: Replace Inline Progress, Stage Derivation, Checklist, and Confidence

`fab-status.sh` SHALL replace its inline progress extraction loop, stage derivation loop, checklist extraction, and confidence extraction with calls to `get_progress_map`, `get_current_stage`, `get_checklist`, and `get_confidence` from `_stageman.sh`. Output SHALL be identical.

#### Scenario: Full status output unchanged
- **GIVEN** a change at stage `spec` (2/6) with checklist not generated and confidence score 3.8
- **WHEN** `fab-status.sh` is executed
- **THEN** all output lines (progress table, stage number, checklist line, confidence line) are identical to pre-refactor output

## Change Resolution: Shared Library

### Requirement: Extract to _resolve-change.sh

The duplicated change resolution logic (fuzzy matching against `fab/changes/` folders) SHALL be extracted from `fab-preflight.sh` (current lines ~17-83) and `fab-status.sh` (current lines ~21-87) into a new shared library `fab/.kit/scripts/_resolve-change.sh`. The library SHALL be sourced by both scripts.

#### Scenario: Preflight uses shared resolution
- **GIVEN** `fab/current` contains `260213-puow-consolidate-status-reads`
- **WHEN** `fab-preflight.sh` runs (no override argument)
- **THEN** the change name is resolved via `_resolve-change.sh` and output is identical

#### Scenario: Status uses shared resolution with override
- **GIVEN** the argument `puow` is passed to `fab-status.sh`
- **WHEN** `fab-status.sh` runs
- **THEN** `_resolve-change.sh` resolves `puow` to `260213-puow-consolidate-status-reads` via substring match

### Requirement: Resolution Library Interface

`_resolve-change.sh` SHALL provide a `resolve_change` function that accepts two arguments: `fab_root` (path to `fab/` directory) and an optional `override` (change name or substring). On success, it SHALL set the variable `RESOLVED_CHANGE_NAME` to the matched folder name. On failure, it SHALL print a diagnostic to stderr and return non-zero.

#### Scenario: Exact match
- **GIVEN** `fab/changes/260213-puow-consolidate-status-reads/` exists
- **WHEN** `resolve_change "$fab_root" "260213-puow-consolidate-status-reads"` is called
- **THEN** `RESOLVED_CHANGE_NAME` is set to `260213-puow-consolidate-status-reads` and return code is 0

#### Scenario: Substring match (single)
- **GIVEN** only one change folder contains `puow` in its name
- **WHEN** `resolve_change "$fab_root" "puow"` is called
- **THEN** `RESOLVED_CHANGE_NAME` is set to the matching folder name and return code is 0

#### Scenario: Multiple matches
- **GIVEN** two change folders contain `260213` in their names
- **WHEN** `resolve_change "$fab_root" "260213"` is called
- **THEN** return code is non-zero and stderr lists the ambiguous matches

#### Scenario: No override (read fab/current)
- **GIVEN** `fab/current` contains `my-change-name`
- **WHEN** `resolve_change "$fab_root" ""` is called (empty override)
- **THEN** `RESOLVED_CHANGE_NAME` is set to `my-change-name` read from `fab/current`

#### Scenario: No active change
- **GIVEN** `fab/current` does not exist
- **WHEN** `resolve_change "$fab_root" ""` is called
- **THEN** return code is non-zero and stderr says "No active change"

#### Scenario: Changes directory missing
- **GIVEN** `fab/changes/` does not exist
- **WHEN** `resolve_change "$fab_root" "anything"` is called
- **THEN** return code is non-zero and stderr says "fab/changes/ not found"

### Requirement: Error Messages Preserve Script Context

`_resolve-change.sh` error messages SHALL NOT include script-specific context (like "Run /fab-new" suggestions). The calling script SHALL add its own context-appropriate guidance after a resolution failure. This keeps the library generic.
<!-- assumed: Generic error messages without command suggestions — preflight and status have different error formatting (stderr vs. printf to stdout), so the library should emit minimal diagnostics and let callers wrap them -->

#### Scenario: Caller adds its own guidance
- **GIVEN** `resolve_change` fails with "No active change"
- **WHEN** `fab-preflight.sh` handles the failure
- **THEN** preflight adds "Run /fab-new to start one." to its stderr output
- **AND** `fab-status.sh` handling the same failure adds its own formatted message with the version header

## Development Folder: src/resolve-change/

### Requirement: Dev Folder Structure

A new `src/resolve-change/` directory SHALL be created following the pattern established by `src/stageman/` and `src/preflight/`. It SHALL contain:

- A symlink `_resolve-change.sh` → `../../fab/.kit/scripts/_resolve-change.sh`
- `README.md` with API documentation (function signatures, arguments, return values, examples)
- `test.sh` — comprehensive test suite covering all resolution scenarios (exact match, substring, multiple matches, no match, no fab/current, empty fab/current)
- `test-simple.sh` — smoke test for quick verification

#### Scenario: Dev symlink resolves correctly
- **GIVEN** `src/resolve-change/_resolve-change.sh` is a symlink to `../../fab/.kit/scripts/_resolve-change.sh`
- **WHEN** a developer sources it from `src/resolve-change/`
- **THEN** the `resolve_change` function is available and resolves changes against `fab/changes/` correctly

## Design Decisions

1. **Line-oriented output for accessors**: `get_progress_map`, `get_checklist`, and `get_confidence` output `key:value` pairs, one per line.
   - *Why*: Matches shell idiom — consumers parse with `while IFS=: read key val` or similar. No subshell needed for variable setting. Consistent with `get_all_stages` which also outputs one item per line.
   - *Rejected*: Setting variables directly (like `_resolve-change.sh` does with `RESOLVED_CHANGE_NAME`) — accessors return multiple values, so variable-setting would need N separate calls or a clunky multi-variable convention.

2. **Variable-setting for resolve_change**: `resolve_change` sets `RESOLVED_CHANGE_NAME` rather than echoing.
   - *Why*: Resolution has a single result (the name). Error handling needs return codes — if it echoed, callers would need `name=$(resolve_change ...)` which swallows the exit code in some shell patterns. Setting a variable keeps the return code clean.
   - *Rejected*: Echo to stdout — conflicts with error messages that go to stderr, and `$()` subshell masks return codes without `set -o pipefail` discipline.

3. **Generic error messages in _resolve-change.sh**: The library emits minimal diagnostics without command suggestions.
   - *Why*: Preflight and status have different error formatting conventions (stderr-only vs. printf with header). The library shouldn't dictate UX — callers add their own context.
   - *Rejected*: Preflight-style messages everywhere — would make status output inconsistent.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Accessor functions return `key:value` line-oriented output rather than setting variables | Multiple return values don't map cleanly to variable-setting; line output is idiomatic shell and matches existing `get_all_stages` pattern |
| 2 | Confident | `resolve_change` sets `RESOLVED_CHANGE_NAME` variable rather than echoing | Single return value + clean exit code handling; matches sourced-library convention |
| 3 | Confident | `_resolve-change.sh` error messages are generic (no "Run /fab-new" suggestions) | Callers have different error formatting; library stays reusable |
| 4 | Confident | `src/stageman/README.md` update is included in scope | Brief lists it explicitly; new functions need API reference entries |
| 5 | Certain | `_resolve-change.sh` does not source `_stageman.sh` itself — it is pure change-resolution logic with no stage awareness | Confirmed: resolution code in both preflight and status is purely filesystem + string matching with zero stageman calls |
<!-- clarified: _resolve-change.sh confirmed as pure filesystem/string matching — reclassified from Tentative to Certain -->

5 assumptions made (4 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-14

| # | Question | Resolution |
|---|----------|------------|
| 1 | Should `_resolve-change.sh` not sourcing `_stageman.sh` be reclassified from Tentative to Confident? | Yes — reclassified to Certain. Resolution code is purely filesystem + string matching. |
| 2 | Should `resolve_change` handle missing `fab/changes/` directory? | Yes — added "Changes directory missing" scenario. |
| 3 | Should `get_current_stage` be refactored to use `get_progress_map` internally? | Yes — added requirement and scenarios for internal consistency. |
| 4 | Dev symlink scenario for `src/resolve-change/` incorrectly references `WORKFLOW_SCHEMA` — fix? | Yes — corrected THEN clause to verify `resolve_change` availability, not schema resolution. |
| 5 | `src/stageman/` dev symlink needs updating after `stageman.sh` → `_stageman.sh` rename — add? | Yes — added scenario for symlink update and old symlink removal. |
