# Spec: Orchestrator Idle Hooks

**Change**: 260305-bs5x-orchestrator-idle-hooks
**Created**: 2026-03-05
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/schemas.md`, `docs/memory/fab-workflow/pipeline-orchestrator.md`, `docs/memory/fab-workflow/setup.md`, `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Modifying `run.sh` or `dispatch.sh` to consume the idle signal — this change provides the signal only; consumption is a separate change
- Adding a shared idle status file (`fab/.agent-status`) for multi-session visibility — deferred until multi-session coordination materializes
- Modifying `fab/.kit/templates/status.yaml` — the `agent` block is ephemeral runtime state, not part of change lifecycle

## Hook Scripts

### Requirement: Stop Hook Writes Idle Timestamp

The Stop hook script at `fab/.kit/hooks/on-stop.sh` SHALL write a unix timestamp to `.status.yaml` when the agent finishes a response turn. The timestamp SHALL be stored at the YAML path `agent.idle_since` as an integer (unix epoch seconds via `date +%s`).

#### Scenario: Agent Finishes a Turn with Active Change

- **GIVEN** `fab/current` contains a valid change name
- **AND** the change directory and `.status.yaml` exist
- **WHEN** the Claude Code `Stop` hook event fires
- **THEN** `agent.idle_since` is written to `.status.yaml` as the current unix timestamp
- **AND** the script exits 0

#### Scenario: No Active Change

- **GIVEN** `fab/current` does not exist or is empty
- **WHEN** the `Stop` hook event fires
- **THEN** the script exits 0 silently without modifying any files

#### Scenario: Change Directory Missing

- **GIVEN** `fab/current` contains a name but the change directory does not exist
- **WHEN** the `Stop` hook event fires
- **THEN** the script exits 0 silently

<!-- clarified: added missing edge case — .status.yaml absent within valid change directory -->

#### Scenario: Status File Missing

- **GIVEN** `fab/current` contains a valid change name
- **AND** the change directory exists but `.status.yaml` does not
- **WHEN** the `Stop` hook event fires
- **THEN** the script exits 0 silently without creating `.status.yaml`

### Requirement: SessionStart Hook Clears Idle State

The SessionStart hook script at `fab/.kit/hooks/on-session-start.sh` SHALL remove the entire `agent` block from `.status.yaml` when a new session begins. Absence of the `agent` block means "agent is active."

#### Scenario: Session Starts with Active Change

- **GIVEN** `fab/current` contains a valid change name
- **AND** `.status.yaml` exists and contains an `agent` block
- **WHEN** the Claude Code `SessionStart` hook event fires
- **THEN** the `agent` block is removed from `.status.yaml`
- **AND** the script exits 0

#### Scenario: Session Starts without Prior Idle State

- **GIVEN** `fab/current` contains a valid change name
- **AND** `.status.yaml` exists but has no `agent` block
- **WHEN** the `SessionStart` hook event fires
- **THEN** `.status.yaml` is unchanged (idempotent)
- **AND** the script exits 0

#### Scenario: No Active Change on Session Start

- **GIVEN** `fab/current` does not exist or is empty
- **WHEN** the `SessionStart` hook event fires
- **THEN** the script exits 0 silently

<!-- clarified: added missing edge case — .status.yaml absent within valid change directory -->

#### Scenario: Status File Missing on Session Start

- **GIVEN** `fab/current` contains a valid change name
- **AND** the change directory exists but `.status.yaml` does not
- **WHEN** the `SessionStart` hook event fires
- **THEN** the script exits 0 silently without modifying any files

### Requirement: Hook Scripts Must Never Block

All hook scripts MUST exit 0 regardless of any error encountered. Hooks run in the critical path of agent startup and response completion — a non-zero exit or hang would block the agent. All error paths (missing `yq`, missing files, resolution failures) SHALL exit 0 silently.

#### Scenario: yq Not Installed

- **GIVEN** `yq` is not available in PATH
- **WHEN** either hook fires
- **THEN** the script exits 0 without modifying any files

#### Scenario: fab Dispatcher Not Available

- **GIVEN** `fab/.kit/bin/fab` does not exist or is not executable
- **WHEN** either hook fires
- **THEN** the script exits 0 without modifying any files

### Requirement: Path Resolution via Git Root

Both hook scripts SHALL use `git rev-parse --show-toplevel` to find the repository root, then resolve all paths relative to it. This ensures hooks work correctly from any working directory (worktrees, subdirectories, repo root).

#### Scenario: Hook Fires from Worktree Subdirectory

- **GIVEN** the current working directory is a subdirectory within a worktree
- **WHEN** either hook fires
- **THEN** the script resolves the repo root via `git rev-parse --show-toplevel`
- **AND** reads `$repo_root/fab/current` for the active change
- **AND** invokes `$repo_root/fab/.kit/bin/fab resolve --dir` for change directory resolution

### Requirement: Change Resolution via Dispatcher

Hook scripts SHALL resolve the active change directory by reading `fab/current` to get the change name, then invoking `fab/.kit/bin/fab resolve --dir` with that name. This reuses the existing resolution logic rather than duplicating path construction.

#### Scenario: Resolution Succeeds

- **GIVEN** `fab/current` contains `260305-bs5x-orchestrator-idle-hooks`
- **WHEN** the hook invokes `fab/.kit/bin/fab resolve --dir`
- **THEN** it receives the path `fab/changes/260305-bs5x-orchestrator-idle-hooks/`
- **AND** uses `$change_dir/.status.yaml` for yq operations

#### Scenario: Resolution Fails

- **GIVEN** `fab/current` contains a name that doesn't match any change folder
- **WHEN** `fab resolve --dir` exits non-zero
- **THEN** the hook exits 0 silently

## Status Schema Extension

### Requirement: Ephemeral Agent Block

`.status.yaml` SHALL support an optional `agent` block at the top level with a single field `idle_since` (integer, unix timestamp). This block is ephemeral runtime state — it is NOT part of the status template, NOT initialized by `changeman new`, and NOT read by any existing status management script.

```yaml
agent:
  idle_since: 1741193400
```

#### Scenario: Orchestrator Reads Idle Timestamp

- **GIVEN** a `.status.yaml` file contains `agent.idle_since: 1741193400`
- **WHEN** an external tool reads `.status.yaml` via `yq '.agent.idle_since'`
- **THEN** it receives the integer `1741193400`

#### Scenario: Agent Block Absent

- **GIVEN** a `.status.yaml` file has no `agent` block
- **WHEN** an external tool reads `.status.yaml` via `yq '.agent.idle_since'`
- **THEN** it receives `null`
- **AND** this indicates the agent is currently active (or no hook has run)

### Requirement: No Template Modification

The status template at `fab/.kit/templates/status.yaml` SHALL NOT be modified. The `agent` block is runtime-only state that should never appear in newly created `.status.yaml` files.

#### Scenario: New Change Created

- **GIVEN** a user runs `/fab-new`
- **WHEN** `.status.yaml` is created from the template
- **THEN** it contains no `agent` block

## Hook Registration (Sync)

### Requirement: Automated Hook Registration via fab-sync

A new sync script at `fab/.kit/sync/5-sync-hooks.sh` SHALL register hook scripts from `fab/.kit/hooks/` into `.claude/settings.local.json`. This runs as part of the `fab-sync.sh` pipeline, after skill deployment (step 2) and before the version stamp (step 5).

The script SHALL:
1. Discover hook scripts in `fab/.kit/hooks/` by glob pattern (`*.sh`)
2. Map each script to its hook event based on filename convention: `on-session-start.sh` maps to `SessionStart`, `on-stop.sh` maps to `Stop`
3. Build the expected `hooks` entries with the command format: `bash fab/.kit/hooks/{filename}`
4. Merge into `.claude/settings.local.json` under the `hooks` key

#### Scenario: First Sync (No Existing Hooks)

- **GIVEN** `.claude/settings.local.json` exists with only `permissions` (no `hooks` key)
- **AND** `fab/.kit/hooks/` contains `on-session-start.sh` and `on-stop.sh`
- **WHEN** `5-sync-hooks.sh` runs
- **THEN** `.claude/settings.local.json` gains a `hooks` key with `SessionStart` and `Stop` arrays
- **AND** output: `Created: .claude/settings.local.json hooks (2 hook entries)`

#### Scenario: Hooks Already Registered (Idempotent)

- **GIVEN** `.claude/settings.local.json` already has the exact hook entries
- **WHEN** `5-sync-hooks.sh` runs
- **THEN** the file is unchanged
- **AND** output: `.claude/settings.local.json hooks: OK`

#### Scenario: User Has Custom Hooks

- **GIVEN** `.claude/settings.local.json` has a `Stop` array with a user-defined hook entry
- **AND** `fab/.kit/hooks/on-stop.sh` is not yet registered
- **WHEN** `5-sync-hooks.sh` runs
- **THEN** the fab hook entry is appended to the existing `Stop` array
- **AND** the user's existing hook entry is preserved
- **AND** output: `Updated: .claude/settings.local.json hooks (added 1 hook entry)`

#### Scenario: No Hook Scripts in Kit

- **GIVEN** `fab/.kit/hooks/` does not exist or is empty
- **WHEN** `5-sync-hooks.sh` runs
- **THEN** no changes to `.claude/settings.local.json`
- **AND** no output (silent skip)

#### Scenario: No .claude/settings.local.json

- **GIVEN** `.claude/settings.local.json` does not exist
- **AND** `fab/.kit/hooks/` contains hook scripts
- **WHEN** `5-sync-hooks.sh` runs
- **THEN** `.claude/settings.local.json` is created with the `hooks` key
- **AND** output: `Created: .claude/settings.local.json hooks (2 hook entries)`

#### Scenario: jq Not Available

- **GIVEN** `jq` is not in PATH
- **WHEN** `5-sync-hooks.sh` runs
- **THEN** output: `WARN: jq not found -- skipping hook sync`
- **AND** no file modifications

### Requirement: Filename-to-Event Mapping

The sync script SHALL map hook filenames to Claude Code hook events using a hardcoded convention:

| Filename | Hook Event |
|----------|-----------|
| `on-session-start.sh` | `SessionStart` |
| `on-stop.sh` | `Stop` |

Future hook scripts follow the same pattern: `on-{kebab-case-event}.sh` maps to `{PascalCaseEvent}`. The mapping is maintained in the sync script.

#### Scenario: Unknown Hook Script

- **GIVEN** `fab/.kit/hooks/` contains `on-session-start.sh`, `on-stop.sh`, and `helper.sh`
- **WHEN** `5-sync-hooks.sh` runs
- **THEN** only `on-session-start.sh` and `on-stop.sh` are registered
- **AND** `helper.sh` is ignored (does not match `on-*.sh` pattern)

### Requirement: Hook Entry Format

Each registered hook SHALL use the following JSON structure within the event array:

```json
{
  "type": "command",
  "command": "bash fab/.kit/hooks/{filename}"
}
```

Duplicate detection SHALL compare the `command` field value — if an entry with the same command string already exists in the event array, it is not added again.

#### Scenario: Duplicate Detection

- **GIVEN** `.claude/settings.local.json` has `hooks.Stop` containing `[{"type": "command", "command": "bash fab/.kit/hooks/on-stop.sh"}]`
- **WHEN** `5-sync-hooks.sh` runs
- **THEN** no duplicate entry is added to the `Stop` array

## Design Decisions

1. **Target `.claude/settings.local.json` (not `settings.json`)**:
   - *Why*: Consistent with the existing scaffold pattern (`fragment-settings.local.json` targets the same file). The `.local.json` file is the project-local settings file that `fab-sync.sh` already manages for permissions.
   - *Rejected*: `.claude/settings.json` — this is the user's personal settings file and should not be modified by automated tooling.

2. **Dedicated sync script (not scaffold fragment)**:
   - *Why*: The existing `json_merge_permissions` function only handles `permissions.allow` arrays. Hook merging needs per-event array merging (`hooks.SessionStart[]`, `hooks.Stop[]`), which is structurally different. A dedicated script keeps the logic clear and testable.
   - *Rejected*: Extending `fragment-` pattern — would require generalizing the merge function, adding complexity for a one-off need.

3. **Hardcoded filename mapping (not metadata/config)**:
   - *Why*: Only two hook scripts exist. A mapping table in the script is simpler than YAML metadata files or a `hooks.yaml` manifest. If more hooks are added, the mapping is a single `case` statement to extend.
   - *Rejected*: YAML manifest in `fab/.kit/hooks/hooks.yaml` — over-engineering for 2 entries.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use unix timestamp for `idle_since` | Confirmed from intake #1 — `date +%s` is simple, yq handles natively, orchestrator does arithmetic | S:90 R:95 A:90 D:95 |
| 2 | Certain | Clear entire `agent` block on SessionStart | Confirmed from intake #2 — absence means active, presence means idle, no stale state | S:85 R:90 A:85 D:90 |
| 3 | Certain | Hook scripts at `fab/.kit/hooks/` | Confirmed from intake #3 — constitution portability, analogous to `fab/.kit/scripts/` | S:80 R:90 A:90 D:90 |
| 4 | Certain | Don't modify status.yaml template | Confirmed from intake #4 — ephemeral runtime state, not lifecycle | S:85 R:95 A:90 D:90 |
| 5 | Certain | Hook registration via `5-sync-hooks.sh` | Confirmed from intake #5 (clarified) — registration handled by fab-sync.sh sub-step | S:95 R:85 A:80 D:75 |
| 6 | Certain | Don't modify run.sh/dispatch.sh | Confirmed from intake #6 — signal only, consumption is separate change | S:95 R:90 A:75 D:80 |
| 7 | Certain | Use `git rev-parse --show-toplevel` for path resolution | Confirmed from intake #7 — canonical way to find repo root from arbitrary CWD | S:95 R:85 A:85 D:80 |
| 8 | Confident | Per-change `.status.yaml` sufficient (no shared file) | Confirmed from intake #8 — orchestrator polls per-worktree, shared file deferred | S:60 R:70 A:60 D:75 |
| 9 | Certain | Target `.claude/settings.local.json` for hook registration | Existing scaffold targets the same file; consistent with `fragment-settings.local.json` pattern | S:90 R:85 A:90 D:90 |
| 10 | Certain | Use `on-*.sh` glob for hook discovery | Only hook scripts follow the `on-{event}.sh` naming; helper scripts and non-hook files are naturally excluded | S:85 R:90 A:85 D:90 |
| 11 | Certain | Hooks exit 0 on all errors | Hooks run in agent critical path — non-zero exit blocks agent startup/response. Silent failure is correct. | S:90 R:95 A:90 D:95 |
| 12 | Confident | Hardcoded filename-to-event mapping | Only 2 hooks exist; a `case` statement is simpler than metadata files. Easily extended if more hooks are added. | S:75 R:85 A:80 D:70 |

12 assumptions (10 certain, 2 confident, 0 tentative, 0 unresolved).
