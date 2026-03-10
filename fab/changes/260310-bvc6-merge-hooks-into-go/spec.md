# Spec: Merge Claude Code Hooks Into Go Binary

**Change**: 260310-bvc6-merge-hooks-into-go
**Created**: 2026-03-10
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Rust binary parity — `fab-rust` is local-dev only; hook subcommands are Go-only for now
- Changing hook semantics — existing behavior is preserved exactly; no new bookkeeping logic beyond `user-prompt`
- Removing the shell scripts from `fab/.kit/hooks/` — they remain as thin wrappers for Claude Code's hook config

## Hook Subcommand Group

### Requirement: `fab hook` parent command

The Go binary SHALL expose a `hook` subcommand group with five subcommands: `session-start`, `stop`, `user-prompt`, `artifact-write`, and `sync`.

#### Scenario: Help output lists all hook subcommands
- **GIVEN** the Go binary is built
- **WHEN** a user runs `fab hook --help`
- **THEN** the output SHALL list `session-start`, `stop`, `user-prompt`, `artifact-write`, and `sync` as available subcommands

## Hook: session-start

### Requirement: Clear agent idle state on session start

`fab hook session-start` SHALL clear the agent idle state for the active change. It takes no arguments and no stdin. It MUST exit 0 always — errors are silently swallowed.

#### Scenario: Active change exists — clears idle state
- **GIVEN** `.fab-status.yaml` symlink exists and points to a valid change
- **AND** `.fab-runtime.yaml` has an `agent.idle_since` entry for that change
- **WHEN** `fab hook session-start` is invoked
- **THEN** the `agent` block for that change SHALL be deleted from `.fab-runtime.yaml`
- **AND** exit code SHALL be 0

#### Scenario: No active change — silent exit
- **GIVEN** `.fab-status.yaml` symlink does not exist
- **WHEN** `fab hook session-start` is invoked
- **THEN** exit code SHALL be 0
- **AND** no files are modified

#### Scenario: Runtime file missing — silent exit
- **GIVEN** `.fab-status.yaml` exists but `.fab-runtime.yaml` does not exist
- **WHEN** `fab hook session-start` is invoked
- **THEN** exit code SHALL be 0
- **AND** no runtime file is created

### Requirement: Internal resolution (no subprocess)

`fab hook session-start` SHALL resolve the active change and manipulate the runtime file using the Go `resolve` and `runtime` internal packages directly — NOT by spawning `fab resolve` or `fab runtime` as subprocesses. <!-- clarified: runtime logic currently in cmd/fab/runtime.go needs extraction to internal/runtime/ first; see "Runtime Package Extraction" section -->

#### Scenario: No subprocess spawned for resolution
- **GIVEN** the Go binary is built with the `hook` package
- **WHEN** `fab hook session-start` is invoked
- **THEN** the implementation SHALL call `resolve.FabRoot()` and `resolve.ToFolder()` directly
- **AND** SHALL manipulate `.fab-runtime.yaml` using the same logic as `runtimeClearIdleCmd` (load, delete agent block, save)

## Hook: stop

### Requirement: Set agent idle timestamp on stop

`fab hook stop` SHALL write an `agent.idle_since` Unix timestamp for the active change. It takes no arguments and no stdin. It MUST exit 0 always.

#### Scenario: Active change exists — sets idle timestamp
- **GIVEN** `.fab-status.yaml` symlink exists and points to a valid change
- **WHEN** `fab hook stop` is invoked
- **THEN** `.fab-runtime.yaml` SHALL have `{change_folder}.agent.idle_since` set to the current Unix timestamp
- **AND** exit code SHALL be 0

#### Scenario: No active change — silent exit
- **GIVEN** `.fab-status.yaml` symlink does not exist
- **WHEN** `fab hook stop` is invoked
- **THEN** exit code SHALL be 0
- **AND** no files are modified

#### Scenario: Runtime file missing — creates it
- **GIVEN** `.fab-status.yaml` exists but `.fab-runtime.yaml` does not
- **WHEN** `fab hook stop` is invoked
- **THEN** `.fab-runtime.yaml` SHALL be created with the idle timestamp
- **AND** exit code SHALL be 0

### Requirement: Internal resolution (no subprocess)

Same as session-start — uses `resolve` and `runtime` internal packages directly.

#### Scenario: No subprocess spawned
- **GIVEN** the Go binary is built
- **WHEN** `fab hook stop` is invoked
- **THEN** the implementation SHALL call `resolve.FabRoot()` and `resolve.ToFolder()` directly
- **AND** SHALL manipulate `.fab-runtime.yaml` using the same logic as `runtimeSetIdleCmd`

## Hook: user-prompt

### Requirement: Clear agent idle state on user prompt

`fab hook user-prompt` SHALL clear the agent idle state for the active change, identical to `session-start`. Registered for the `UserPromptSubmit` event. It takes no arguments and no stdin. It MUST exit 0 always.

#### Scenario: Active change exists — clears idle state
- **GIVEN** `.fab-status.yaml` symlink exists and points to a valid change
- **AND** `.fab-runtime.yaml` has an `agent.idle_since` entry for that change
- **WHEN** `fab hook user-prompt` is invoked
- **THEN** the `agent` block for that change SHALL be deleted from `.fab-runtime.yaml`
- **AND** exit code SHALL be 0

#### Scenario: No active change — silent exit
- **GIVEN** `.fab-status.yaml` symlink does not exist
- **WHEN** `fab hook user-prompt` is invoked
- **THEN** exit code SHALL be 0

### Requirement: Internal resolution (no subprocess)

Same as session-start and stop.

#### Scenario: No subprocess spawned
- **GIVEN** the Go binary is built
- **WHEN** `fab hook user-prompt` is invoked
- **THEN** the implementation SHALL use internal packages directly

## Hook: artifact-write

### Requirement: Parse PostToolUse JSON from stdin

`fab hook artifact-write` SHALL read the Claude Code PostToolUse JSON payload from stdin and extract the `tool_input.file_path` field using Go's `encoding/json` package.

#### Scenario: Valid JSON with file_path
- **GIVEN** stdin contains `{"tool_input":{"file_path":"fab/changes/260310-bvc6-test/intake.md"}}`
- **WHEN** `fab hook artifact-write` is invoked
- **THEN** the file_path SHALL be extracted as `fab/changes/260310-bvc6-test/intake.md`

#### Scenario: Invalid JSON — silent exit
- **GIVEN** stdin contains malformed JSON
- **WHEN** `fab hook artifact-write` is invoked
- **THEN** exit code SHALL be 0
- **AND** no output is produced

#### Scenario: Missing file_path — silent exit
- **GIVEN** stdin contains `{"tool_input":{}}`
- **WHEN** `fab hook artifact-write` is invoked
- **THEN** exit code SHALL be 0

### Requirement: Pattern match against fab artifact paths

The extracted `file_path` SHALL be matched against fab artifact path patterns. Only these patterns trigger bookkeeping:

- `*/fab/changes/*/intake.md` or `fab/changes/*/intake.md`
- `*/fab/changes/*/spec.md` or `fab/changes/*/spec.md`
- `*/fab/changes/*/tasks.md` or `fab/changes/*/tasks.md`
- `*/fab/changes/*/checklist.md` or `fab/changes/*/checklist.md`

Non-matching paths SHALL cause a silent exit 0.

#### Scenario: Absolute path to intake.md
- **GIVEN** file_path is `/home/user/project/fab/changes/260310-bvc6-test/intake.md`
- **WHEN** pattern matching runs
- **THEN** the change folder SHALL be extracted as `260310-bvc6-test`
- **AND** the artifact type SHALL be `intake.md`

#### Scenario: Relative path to spec.md
- **GIVEN** file_path is `fab/changes/260310-bvc6-test/spec.md`
- **WHEN** pattern matching runs
- **THEN** the change folder SHALL be `260310-bvc6-test` and artifact SHALL be `spec.md`

#### Scenario: Non-fab path
- **GIVEN** file_path is `src/main.go`
- **WHEN** pattern matching runs
- **THEN** no bookkeeping SHALL occur
- **AND** exit code SHALL be 0

### Requirement: Per-artifact bookkeeping

After extracting the change folder and artifact type, `fab hook artifact-write` SHALL perform artifact-specific bookkeeping by calling internal packages directly (not subprocesses):

**intake.md**:
1. Read the intake file content
2. Infer change type via case-insensitive keyword matching (first match wins):
   - `fix|bug|broken|regression` → `fix`
   - `refactor|restructure|consolidate|split|rename|redesign` → `refactor`
   - `docs|document|readme|guide` → `docs`
   - `test|spec|coverage` → `test`
   - `ci|pipeline|deploy|build` → `ci`
   - `chore|cleanup|maintenance|housekeeping` → `chore`
   - default → `feat`
3. Set change type via statusfile package (equivalent to `fab status set-change-type`)
4. Compute intake confidence score via score package (equivalent to `fab score --stage intake`)

**spec.md**:
1. Compute spec confidence score via score package (equivalent to `fab score`)

**tasks.md**:
1. Read the tasks file content
2. Count unchecked tasks (lines matching `^- \[ \]`)
3. Set checklist total via statusfile package (equivalent to `fab status set-checklist total`)

**checklist.md**:
1. Mark checklist as generated via statusfile package
2. Read the checklist file content
3. Count total checklist items (lines matching `^- \[(x| )\]`)
4. Set checklist total and completed=0 via statusfile package

#### Scenario: intake.md written with "refactor" keyword
- **GIVEN** stdin contains a PostToolUse payload with file_path pointing to an intake.md
- **AND** the intake content contains the word "refactor"
- **WHEN** `fab hook artifact-write` is invoked
- **THEN** the change type SHALL be set to `refactor`
- **AND** the intake confidence score SHALL be computed

#### Scenario: tasks.md written with 5 unchecked tasks
- **GIVEN** stdin contains a PostToolUse payload with file_path pointing to a tasks.md
- **AND** the file has 5 lines matching `^- \[ \]`
- **WHEN** `fab hook artifact-write` is invoked
- **THEN** checklist total SHALL be set to 5

#### Scenario: checklist.md written with 12 items
- **GIVEN** stdin contains a PostToolUse payload with file_path pointing to a checklist.md
- **AND** the file has 12 lines matching `^- \[(x| )\]`
- **WHEN** `fab hook artifact-write` is invoked
- **THEN** checklist generated SHALL be set to true
- **AND** checklist total SHALL be set to 12
- **AND** checklist completed SHALL be set to 0

### Requirement: JSON output

After successful bookkeeping, `fab hook artifact-write` SHALL output a JSON object to stdout:

```json
{"additionalContext":"Bookkeeping: type: fix, score: 4.1"}
```

The `additionalContext` field SHALL contain a comma-separated list of bookkeeping actions performed. If no bookkeeping occurred (non-matching path), no output is produced.

#### Scenario: intake.md bookkeeping output
- **GIVEN** bookkeeping for intake.md sets type to `feat` and score to `3.5`
- **WHEN** output is produced
- **THEN** stdout SHALL contain `{"additionalContext":"Bookkeeping: type: feat, score: 3.5"}`

### Requirement: Never fail

`fab hook artifact-write` MUST exit 0 always. Any error during JSON parsing, file reading, change resolution, or bookkeeping calls SHALL be silently swallowed.

#### Scenario: Change folder doesn't resolve
- **GIVEN** stdin contains a valid PostToolUse payload pointing to `fab/changes/nonexistent/intake.md`
- **WHEN** `fab hook artifact-write` is invoked
- **THEN** exit code SHALL be 0
- **AND** no output is produced

## Hook: sync

### Requirement: Register hooks into settings.local.json

`fab hook sync` SHALL replace the jq-dependent `fab/.kit/sync/5-sync-hooks.sh` by implementing the same hook registration logic in Go. It discovers `on-*.sh` files in `fab/.kit/hooks/`, maps them to Claude Code hook events via an internal mapping table, and merges entries into `.claude/settings.local.json`. Idempotent — running twice produces no changes.

The mapping table SHALL be:

| Script | Event | Matcher |
|--------|-------|---------|
| `on-session-start.sh` | `SessionStart` | (empty) |
| `on-stop.sh` | `Stop` | (empty) |
| `on-user-prompt.sh` | `UserPromptSubmit` | (empty) |
| `on-artifact-write.sh` | `PostToolUse` | `Write` |
| `on-artifact-write.sh` | `PostToolUse` | `Edit` |

Each hook entry in the JSON has the structure:
```json
{"matcher": "<matcher>", "hooks": [{"type": "command", "command": "bash fab/.kit/hooks/<script>"}]}
```

#### Scenario: Fresh settings file — creates hooks section
- **GIVEN** `.claude/settings.local.json` does not exist or is `{}`
- **WHEN** `fab hook sync` is invoked
- **THEN** `.claude/settings.local.json` SHALL be created/updated with a `hooks` object containing all mapped events
- **AND** stdout SHALL report `Created: .claude/settings.local.json hooks (N hook entries)`

#### Scenario: Existing hooks — deduplication
- **GIVEN** `.claude/settings.local.json` already contains the correct hook entries
- **WHEN** `fab hook sync` is invoked
- **THEN** no entries SHALL be duplicated
- **AND** stdout SHALL report `.claude/settings.local.json hooks: OK`

#### Scenario: Partial hooks — merges new entries
- **GIVEN** `.claude/settings.local.json` has some hook entries but is missing `UserPromptSubmit`
- **WHEN** `fab hook sync` is invoked
- **THEN** the missing `UserPromptSubmit` entry SHALL be added
- **AND** existing entries SHALL be preserved
- **AND** stdout SHALL report `Updated: .claude/settings.local.json hooks (added N hook entries)`

#### Scenario: Hook script missing — skips mapping
- **GIVEN** `on-user-prompt.sh` does not exist in `fab/.kit/hooks/`
- **WHEN** `fab hook sync` is invoked
- **THEN** the `UserPromptSubmit` mapping SHALL be skipped
- **AND** all other mappings for existing scripts SHALL be registered

#### Scenario: Existing non-hook settings preserved
- **GIVEN** `.claude/settings.local.json` contains other settings (e.g., `model`, `permissions`)
- **WHEN** `fab hook sync` is invoked
- **THEN** all non-hook settings SHALL be preserved unchanged

### Requirement: Duplicate detection by matcher + command pair

Deduplication SHALL match on both the `matcher` field AND the `hooks[].command` field. If an entry with the same matcher and command already exists in the event's array, it SHALL NOT be added again.

#### Scenario: Same command different matcher — both kept
- **GIVEN** `PostToolUse` has an entry for matcher `Write` with command `bash fab/.kit/hooks/on-artifact-write.sh`
- **AND** a new entry for matcher `Edit` with the same command
- **WHEN** `fab hook sync` runs
- **THEN** both entries SHALL be present (different matchers = different entries)

### Requirement: `5-sync-hooks.sh` becomes thin wrapper

`fab/.kit/sync/5-sync-hooks.sh` SHALL be rewritten to delegate to `fab hook sync`, following the same thin-wrapper pattern as the hook scripts:

```bash
#!/usr/bin/env bash
set -euo pipefail
sync_dir="$(cd "$(dirname "$0")" && pwd)"
kit_dir="$(dirname "$sync_dir")"
"$kit_dir/bin/fab" hook sync 2>/dev/null || echo "WARN: fab binary not found — skipping hook sync"
```

#### Scenario: fab-sync delegates to Go binary
- **GIVEN** `fab-sync.sh` runs sync steps including `5-sync-hooks.sh`
- **WHEN** `5-sync-hooks.sh` executes
- **THEN** it SHALL invoke `fab hook sync`
- **AND** if the binary is missing, it SHALL print a warning and exit 0

## Shell Script Wrappers

### Requirement: Existing scripts become thin wrappers

The three existing shell scripts in `fab/.kit/hooks/` SHALL be replaced with one-liners that delegate to the Go binary. A new `on-user-prompt.sh` SHALL be created for the `UserPromptSubmit` event.

Each wrapper uses the pattern:
```bash
#!/usr/bin/env bash
exec "$(dirname "$0")/../bin/fab" hook <subcommand> 2>/dev/null; exit 0
```

The `exec ... 2>/dev/null; exit 0` pattern ensures the hook never fails even if the binary is missing (exec replaces the process; if exec fails, the fallback `exit 0` runs).

#### Scenario: on-session-start.sh delegates to binary
- **GIVEN** the Go binary exists at `fab/.kit/bin/fab`
- **WHEN** `on-session-start.sh` is executed
- **THEN** it SHALL exec `fab hook session-start`

#### Scenario: on-stop.sh delegates to binary
- **GIVEN** the Go binary exists
- **WHEN** `on-stop.sh` is executed
- **THEN** it SHALL exec `fab hook stop`

#### Scenario: on-user-prompt.sh created and delegates to binary
- **GIVEN** the Go binary exists
- **WHEN** `on-user-prompt.sh` is executed
- **THEN** it SHALL exec `fab hook user-prompt`

#### Scenario: on-artifact-write.sh delegates to binary
- **GIVEN** the Go binary exists
- **WHEN** `on-artifact-write.sh` is executed
- **THEN** it SHALL exec `fab hook artifact-write`

#### Scenario: Binary missing — graceful fallback
- **GIVEN** the Go binary does not exist at `fab/.kit/bin/fab`
- **WHEN** any hook shell script is executed
- **THEN** exit code SHALL be 0

### Requirement: Hook sync mapping update

The `fab hook sync` subcommand's internal mapping table SHALL include a `UserPromptSubmit` event pointing to `on-user-prompt.sh`. The shell script `5-sync-hooks.sh` delegates to `fab hook sync` (see "Hook: sync" section above). <!-- clarified: mapping table now lives in Go binary, not shell script -->

#### Scenario: fab-sync deploys UserPromptSubmit hook
- **GIVEN** `fab-sync.sh` is run (which delegates to `5-sync-hooks.sh`, which calls `fab hook sync`)
- **WHEN** hook mappings are processed
- **THEN** `UserPromptSubmit` SHALL be mapped to `on-user-prompt.sh`

## jq Prerequisite Removal

### Requirement: Remove jq from fab-doctor.sh

`fab/.kit/scripts/fab-doctor.sh` SHALL remove the `jq` prerequisite check. With `on-artifact-write.sh` delegating to the Go binary and `5-sync-hooks.sh` delegating to `fab hook sync`, there are no remaining jq consumers in the kit. <!-- clarified: blocking resolved — 5-sync-hooks.sh rewritten to delegate to `fab hook sync`, eliminating its jq dependency -->

#### Scenario: fab-doctor no longer checks for jq
- **GIVEN** `fab-doctor.sh` is run
- **WHEN** prerequisite checks execute
- **THEN** `jq` SHALL NOT be listed as a prerequisite
- **AND** the absence of `jq` SHALL NOT produce a warning or failure

## _scripts.md Update

### Requirement: Update kit script invocation guide

`fab/.kit/skills/_scripts.md` SHALL be updated to document the `fab hook` subcommand group and its four subcommands (`session-start`, `stop`, `user-prompt`, `artifact-write`), per the constitution's requirement that CLI changes include `_scripts.md` updates. <!-- clarified: added missing requirement per constitution constraint — "Changes to the fab CLI (Go binary) MUST update _scripts.md" -->

#### Scenario: _scripts.md documents hook subcommands
- **GIVEN** the change is applied
- **WHEN** a developer reads `_scripts.md`
- **THEN** the `fab hook` command group SHALL be listed in the Command Reference table
- **AND** each subcommand's purpose and usage SHALL be documented

## Runtime Package Extraction

### Requirement: Extract runtime file logic into internal package

The runtime file manipulation functions (`loadRuntimeFile`, `saveRuntimeFile`, `runtimeFilePath`) SHALL be extracted from `cmd/fab/runtime.go` (package `main`) into a new `internal/runtime/` package so that both the existing `fab runtime` Cobra commands and the new `fab hook` commands can share the logic without duplication. <!-- clarified: current runtime file logic lives in cmd/fab/runtime.go (package main), not importable by other packages — extraction needed for hook subcommands to reuse it -->

#### Scenario: Hook subcommands use shared runtime package
- **GIVEN** the `internal/runtime/` package is extracted
- **WHEN** `fab hook session-start` clears idle state
- **THEN** it SHALL call `runtime.ClearIdle()` (or equivalent) from the shared package
- **AND** `fab runtime clear-idle` SHALL also use the same shared package

## Go Unit Tests

### Requirement: Test coverage for hook package

The `hook` package SHALL include Go unit tests covering:

1. **JSON parsing** — valid PostToolUse payload, malformed JSON, missing fields
2. **Path pattern matching** — absolute paths, relative paths, non-fab paths, edge cases
3. **Change type keyword matching** — each keyword category, case insensitivity, first-match-wins, default to `feat`
4. **Task counting** — unchecked task regex matching
5. **Checklist counting** — checked and unchecked item regex matching
6. **Never-fail behavior** — error scenarios return nil, not errors

#### Scenario: All keyword categories tested
- **GIVEN** test fixtures with intake content containing each keyword
- **WHEN** keyword matching is invoked
- **THEN** each keyword SHALL map to the correct change type

#### Scenario: Path matching edge cases
- **GIVEN** paths like `fab/changes//intake.md`, `fab/changes/name/other.md`, `not-fab/changes/name/intake.md`
- **WHEN** pattern matching runs
- **THEN** malformed and non-matching paths SHALL be rejected

## Bats Test Updates

### Requirement: Update existing bats tests

Existing bats tests in `src/hooks/` SHALL be updated to test the thin wrapper scripts. Tests SHOULD verify that the wrappers correctly delegate to the binary and handle binary-missing scenarios.

#### Scenario: Session-start bats test with binary
- **GIVEN** the Go binary exists at the expected path
- **WHEN** the bats test invokes `on-session-start.sh`
- **THEN** the test SHALL verify that idle state is cleared

#### Scenario: Stop bats test with binary
- **GIVEN** the Go binary exists
- **WHEN** the bats test invokes `on-stop.sh`
- **THEN** the test SHALL verify that idle timestamp is set

## Design Decisions

1. **Internal package calls over subprocesses**: Hook subcommands call `resolve`, `statusfile`, `score`, and `runtime` packages directly rather than spawning `fab resolve`, `fab status`, etc. as child processes.
   - *Why*: Hooks fire on every tool use — subprocess overhead (2-4 spawns per invocation) adds latency. The Go binary already has all packages available internally.
   - *Rejected*: Subprocess delegation — simpler code but measurably slower for the most frequently called subcommand.

2. **`user-prompt` as separate subcommand**: New `fab hook user-prompt` for `UserPromptSubmit` rather than reusing `session-start`.
   - *Why*: Different Claude Code events (`SessionStart` vs `UserPromptSubmit`) should map to distinct subcommands for clarity, even though both call `clear-idle`. This also allows future divergence if the events need different behavior.
   - *Rejected*: Aliasing to `session-start` — loses semantic clarity and makes hook config harder to audit.

3. **Shell wrappers kept, not deleted**: Hook `.sh` files remain as thin wrappers.
   - *Why*: Claude Code's hook config in `.claude/settings.json` points to `.sh` files. The sync script deploys hooks by copying `.sh` files. Removing them would break the hook registration mechanism.
   - *Rejected*: Direct binary invocation from hook config — would require changing the settings.json hook format and the sync mechanism, which is out of scope.

4. **`fab hook sync` replaces jq-dependent `5-sync-hooks.sh`**: Hook-to-event mapping and JSON merging moved to Go binary.
   - *Why*: `5-sync-hooks.sh` was the last remaining jq consumer after artifact-write moves to Go. Moving sync to Go eliminates jq as a kit prerequisite entirely. Go's `encoding/json` handles the JSON manipulation natively.
   - *Rejected*: Keeping jq prerequisite for sync only — defeats the goal of eliminating the jq dependency. Also rejected: rewriting sync in pure bash without jq — JSON manipulation in bash without jq is fragile and harder to maintain than Go.

## Clarifications

### Session 2026-03-10 (auto)

| # | Action | Detail |
|---|--------|--------|
| 1 | Resolved | Hook sync mapping: corrected file reference from `fab-sync.sh` to `fab/.kit/sync/5-sync-hooks.sh` |
| 2 | Resolved | jq prerequisite removal: `5-sync-hooks.sh` rewritten to delegate to `fab hook sync` — eliminates last jq consumer |
| 3 | Added | Missing `_scripts.md` update requirement per constitution |
| 4 | Added | Runtime package extraction requirement — runtime helpers in `cmd/fab/runtime.go` (package main) need extraction to `internal/runtime/` for hook subcommands to import |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All four hooks (session-start, stop, user-prompt, artifact-write) implemented as `fab hook` subcommands | Confirmed from intake #1 — user explicitly chose "move all hooks" to Go | S:95 R:80 A:90 D:95 |
| 2 | Certain | Shell scripts become thin `exec` wrappers | Confirmed from intake #2 — Claude Code config requires .sh files | S:85 R:90 A:85 D:90 |
| 3 | Certain | All hooks MUST exit 0 always | Confirmed from intake #3 — hooks must never block the agent | S:95 R:95 A:95 D:95 |
| 4 | Confident | Rust binary parity deferred | Confirmed from intake #4 — Rust is local-dev only | S:70 R:85 A:75 D:80 |
| 5 | Certain | Remove jq from fab-doctor.sh prerequisites | Clarified — user chose to rewrite 5-sync-hooks.sh as `fab hook sync`, eliminating last jq consumer | S:95 R:80 A:90 D:95 |
| 6 | Confident | artifact-write reads stdin; session-start, stop, user-prompt take no stdin | Confirmed from intake #6 — matches Claude Code hook API | S:80 R:85 A:80 D:90 |
| 7 | Certain | Hook subcommands use internal packages directly, no subprocess spawning | Codebase shows all needed packages (resolve, statusfile, score, runtime logic) already exist in Go | S:90 R:85 A:95 D:90 |
| 8 | Confident | Existing bats tests updated to test thin wrappers with real binary | Tests currently stub `fab` dispatcher; with thin wrappers, tests can use the real binary or continue stubbing | S:70 R:80 A:75 D:75 |
| 9 | Certain | `_scripts.md` updated with `fab hook` command documentation | Constitution requires CLI changes to update `_scripts.md` | S:95 R:90 A:95 D:95 |
| 10 | Certain | Runtime file logic extracted from `cmd/fab/runtime.go` to `internal/runtime/` package | Codebase shows runtime helpers in package main, not importable — extraction is the standard Go pattern for shared logic | S:90 R:85 A:90 D:90 |
| 11 | Certain | `fab hook sync` replaces jq-dependent `5-sync-hooks.sh` | User explicitly chose option (b) — rewrite sync to use Go binary, eliminating jq dependency | S:95 R:80 A:90 D:95 |
| 12 | Certain | `5-sync-hooks.sh` becomes thin wrapper delegating to `fab hook sync` | Same pattern as hook scripts — shell wrapper for backward compatibility with `fab-sync.sh` orchestrator | S:90 R:85 A:90 D:90 |

12 assumptions (9 certain, 3 confident, 0 tentative, 0 unresolved).
