# Spec: Extract Agent Runtime to Gitignored File

**Change**: 260306-1lwf-extract-agent-runtime-file
**Created**: 2026-03-06
**Affected memory**: `docs/memory/fab-workflow/schemas.md`, `docs/memory/fab-workflow/pipeline-orchestrator.md`

## Non-Goals

- Moving `stage_metrics`, `last_updated`, or `confidence` out of `.status.yaml` — only the `agent` block moves
- Adding cleanup/GC for stale entries in `.fab-runtime.yaml` — file is ephemeral and gitignored
- Modifying Go binary code (`statusfile.go`, `status.go`) — hooks use `yq` directly
- Changing the pipeline orchestrator's polling behavior — only memory docs update

## Hooks: Runtime File Target

### Requirement: on-stop.sh writes to .fab-runtime.yaml

The `on-stop.sh` hook SHALL write `agent.idle_since` to `.fab-runtime.yaml` at the repository root instead of the change's `.status.yaml`. The file SHALL be keyed by full change folder name (`YYMMDD-XXXX-slug` format).

#### Scenario: First write creates file
- **GIVEN** `.fab-runtime.yaml` does not exist at the repo root
- **WHEN** `on-stop.sh` fires
- **THEN** the hook creates `.fab-runtime.yaml` with an empty YAML object (`{}`)
- **AND** writes `{change_folder}.agent.idle_since` with the current unix timestamp

#### Scenario: Subsequent write updates existing entry
- **GIVEN** `.fab-runtime.yaml` exists with entries for other changes
- **WHEN** `on-stop.sh` fires for a different change
- **THEN** the hook writes only the current change's `agent.idle_since` key
- **AND** existing entries for other changes are preserved

#### Scenario: No active change
- **GIVEN** `fab/current` does not exist or is empty
- **WHEN** `on-stop.sh` fires
- **THEN** the hook exits 0 without modifying any file

### Requirement: on-session-start.sh clears from .fab-runtime.yaml

The `on-session-start.sh` hook SHALL clear the `agent` block from `.fab-runtime.yaml` instead of `.status.yaml`. It SHALL delete only the current change's `agent` key, leaving other changes' entries intact.

#### Scenario: Clear existing entry
- **GIVEN** `.fab-runtime.yaml` exists with an `agent` entry for the current change
- **WHEN** `on-session-start.sh` fires
- **THEN** the hook deletes `{change_folder}.agent` from `.fab-runtime.yaml`
- **AND** other changes' entries remain untouched

#### Scenario: File does not exist
- **GIVEN** `.fab-runtime.yaml` does not exist
- **WHEN** `on-session-start.sh` fires
- **THEN** the hook exits 0 without error

#### Scenario: No entry for current change
- **GIVEN** `.fab-runtime.yaml` exists but has no entry for the current change
- **WHEN** `on-session-start.sh` fires
- **THEN** the hook exits 0 without error (yq del on missing key is a no-op)

## Hooks: No .status.yaml Agent Writes

### Requirement: Hooks stop writing agent block to .status.yaml

Neither `on-stop.sh` nor `on-session-start.sh` SHALL read from or write to the `agent` block in `.status.yaml`. All agent runtime state operations MUST target `.fab-runtime.yaml` exclusively.

#### Scenario: on-stop.sh does not touch .status.yaml agent block
- **GIVEN** a valid active change with `.status.yaml`
- **WHEN** `on-stop.sh` fires
- **THEN** `.status.yaml` is not modified
- **AND** `agent.idle_since` is written only to `.fab-runtime.yaml`

#### Scenario: on-session-start.sh does not touch .status.yaml agent block
- **GIVEN** a valid active change with `.status.yaml` containing an `agent` block
- **WHEN** `on-session-start.sh` fires
- **THEN** `.status.yaml` is not modified
- **AND** only `.fab-runtime.yaml` is updated

## Runtime File: Structure and Location

### Requirement: .fab-runtime.yaml at repo root

The runtime file SHALL be located at the repository root as `.fab-runtime.yaml`. It SHALL be a YAML file keyed by full change folder name.

#### Scenario: Multi-change runtime state
- **GIVEN** two changes are active in separate worktrees
- **WHEN** each worktree's `on-stop.sh` fires
- **THEN** each worktree's `.fab-runtime.yaml` contains only its own change's entry
<!-- assumed: Each worktree has its own repo root, so separate .fab-runtime.yaml files — no cross-worktree contention -->

### Requirement: .fab-runtime.yaml is gitignored

`.fab-runtime.yaml` SHALL be listed in the repository's `.gitignore` file so it is never committed.

#### Scenario: Gitignore entry
- **GIVEN** the `.gitignore` file at repo root
- **WHEN** a user runs `git status` after `.fab-runtime.yaml` is created
- **THEN** `.fab-runtime.yaml` does not appear in untracked files

## Documentation: Memory Updates

### Requirement: schemas.md reflects new file location

The `docs/memory/fab-workflow/schemas.md` "Ephemeral Runtime State" section SHALL document that the `agent` block now lives in `.fab-runtime.yaml` (repo root, keyed by change folder name) instead of `.status.yaml`.

#### Scenario: Updated documentation
- **GIVEN** the schemas.md memory file
- **WHEN** a reader looks up agent runtime state
- **THEN** the documentation points to `.fab-runtime.yaml` with the keyed structure
- **AND** the old `.status.yaml` location is no longer referenced for agent state

### Requirement: pipeline-orchestrator.md reflects new location

The `docs/memory/fab-workflow/pipeline-orchestrator.md` "Agent idle signal" paragraph SHALL reference `.fab-runtime.yaml` instead of `.status.yaml` for the agent idle state location.

#### Scenario: Updated orchestrator docs
- **GIVEN** the pipeline-orchestrator.md memory file
- **WHEN** a reader looks up the agent idle signal
- **THEN** the documentation references `.fab-runtime.yaml` keyed by change folder name

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only `agent` block moves — workflow state stays in `.status.yaml` | Confirmed from intake #1 — user explicitly chose minimal scope | S:95 R:90 A:95 D:95 |
| 2 | Certain | Repo-root singleton `.fab-runtime.yaml` (Option B) | Confirmed from intake #2 — user chose Option B over per-change sidecar | S:95 R:85 A:90 D:95 |
| 3 | Certain | No Go binary changes needed | Confirmed from intake #3 — blast radius analysis verified zero Go consumers | S:95 R:95 A:95 D:95 |
| 4 | Certain | File keyed by full change folder name (`YYMMDD-XXXX-slug`) | Confirmed from intake #4 — consistent with `fab resolve --folder` output | S:90 R:85 A:90 D:90 |
| 5 | Certain | Hooks create `.fab-runtime.yaml` on first write with `{}` seed | Upgraded from intake Confident #5 — standard hook pattern, matches existing must-never-fail contract | S:85 R:90 A:90 D:90 |
| 6 | Confident | No cleanup/GC for stale entries | Confirmed from intake #6 — file is ephemeral, gitignored, harmless stale entries | S:70 R:95 A:80 D:80 |
| 7 | Certain | `.gitignore` entry at repo root level | Confirmed from intake #7 — follows existing pattern | S:90 R:95 A:90 D:95 |

7 assumptions (6 certain, 1 confident, 0 tentative, 0 unresolved).
