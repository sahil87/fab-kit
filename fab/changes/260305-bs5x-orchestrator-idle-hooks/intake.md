# Intake: Orchestrator Idle Hooks

**Change**: 260305-bs5x-orchestrator-idle-hooks
**Created**: 2026-03-05
**Status**: Draft

## Origin

> Add SessionStart and Stop hooks (shell scripts in fab/.kit/) that write/clear `agent.idle_since` in the active change's `.status.yaml`. These hooks enable the existing pipeline orchestrator (and future multi-change coordination) to detect when an agent window is idle and ready for the next pipeline command — replacing fixed-delay heuristics with an explicit filesystem signal.

Discussion context: Explored the `agent-orchestrator` project (a TypeScript monorepo for coordinating 30+ AI agents via tmux). Concluded that fab-kit needs a much simpler approach — filesystem-as-bus, shell-as-glue — consistent with the constitution. The key missing piece: an explicit "agent is idle" signal. Claude Code's `SessionStart` and `Stop` hook events provide exactly this.

## Why

The existing pipeline orchestrator (`run.sh` + `dispatch.sh`) uses fixed delays to guess when an agent is ready:
- `CLAUDE_STARTUP_DELAY=3s` before sending fab-switch
- `POST_SWITCH_DELAY=5s` before sending fab-ff
- `PIPELINE_SHIP_DELAY=8s` after hydrate:done before sending /git-pr

These are timing heuristics — too short and the command is swallowed, too long and time is wasted. An explicit `idle_since` timestamp in `.status.yaml` lets the orchestrator (or any future coordination tool) know definitively when the agent stopped and is ready for input.

Without this, multi-change coordination remains dependent on fragile timing assumptions that vary by machine speed, model latency, and change complexity.

## What Changes

### New `agent` block in `.status.yaml`

Add an `agent` key to `.status.yaml` with a single field:

```yaml
agent:
  idle_since: 1741193400    # unix timestamp, set by Stop hook
```

When the agent is active (working), the `agent` block is absent or `idle_since` is cleared. When the agent stops (finishes its turn), the `Stop` hook writes the current unix timestamp.

The `agent` block is NOT part of the status template — it's ephemeral runtime state managed exclusively by hooks. It should not be initialized in `fab/.kit/templates/status.yaml`.

### SessionStart hook script

A shell script at `fab/.kit/hooks/on-session-start.sh` that:

1. Reads `fab/current` to get the active change ID (exit silently if no active change)
2. Resolves the change directory via `fab/.kit/bin/fab resolve --dir`
3. Clears `agent.idle_since` from `.status.yaml` using `yq`:
   ```bash
   yq -i 'del(.agent)' "$status_file"
   ```
4. Exits 0 always (hooks must not block the agent)

### Stop hook script

A shell script at `fab/.kit/hooks/on-stop.sh` that:

1. Reads `fab/current` to get the active change ID (exit silently if no active change)
2. Resolves the change directory via `fab/.kit/bin/fab resolve --dir`
3. Writes `agent.idle_since` to `.status.yaml` using `yq`:
   ```bash
   yq -i '.agent.idle_since = '$(date +%s)'' "$status_file"
   ```
4. Exits 0 always

### Hook registration

The hooks need to be registered in Claude Code's settings. This is a project-local configuration — the hooks reference scripts inside `fab/.kit/hooks/`. Registration format (in `.claude/settings.local.json`):

```json
{
  "hooks": {
    "SessionStart": [{ "command": "bash fab/.kit/hooks/on-session-start.sh" }],
    "Stop": [{ "command": "bash fab/.kit/hooks/on-stop.sh" }]
  }
}
```

Registration SHALL be handled by a new `fab/.kit/sync/5-sync-hooks.sh` script, following the existing numbered-step convention alongside `1-prerequisites.sh`, `2-sync-workspace.sh`, etc. The script:

1. Reads hook definitions from `fab/.kit/hooks/` (discovers available hook scripts)
2. Merges them into `.claude/settings.local.json` under the `hooks` key (same file as permissions scaffold)
3. Uses its own idempotent merge logic for `hooks.*` arrays — the existing `json_merge_permissions` only handles `permissions.allow` and won't work for hooks. Entries already present are not duplicated.
4. Requires `jq` (already used by `2-sync-workspace.sh` for JSON merging)

This keeps hook wiring automated and consistent with the existing sync workflow — users don't need to manually configure hooks.

### Path resolution

Both scripts must work from any working directory (worktrees, repo root). They should use `git rev-parse --show-toplevel` to find the repo root, then resolve paths relative to it.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document the `fab/.kit/hooks/` directory and hook scripts
- `fab-workflow/schemas`: (modify) Document the `agent` block in `.status.yaml` schema
- `fab-workflow/pipeline-orchestrator`: (modify) Note that `idle_since` hooks exist as an explicit idle signal, complementing the existing polling approach
- `fab-workflow/setup`: (modify) Document the new `5-sync-hooks.sh` sync step in fab-sync.sh's sub-step inventory
- `fab-workflow/distribution`: (modify) Note `fab/.kit/hooks/` as a new distributed directory

## Impact

- **`.status.yaml` schema**: New optional `agent` block — purely additive, no existing fields affected
- **Pipeline orchestrator**: Future changes can replace fixed delays with `idle_since` polling. This change does NOT modify `run.sh`/`dispatch.sh` — it only provides the signal.
- **New directory**: `fab/.kit/hooks/` — new convention for Claude Code hook scripts shipped with the kit
- **Dependencies**: Requires `yq` (already a fab-kit dependency) and `git` (always present)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use unix timestamp (not ISO 8601) for `idle_since` | Shell `date +%s` is simpler and sufficient; yq handles integers natively; orchestrator does arithmetic on timestamps | S:90 R:95 A:90 D:95 |
| 2 | Certain | Clear entire `agent` block on SessionStart (not just null the field) | Cleaner — absence means "active", presence means "idle". No stale partial state. | S:85 R:90 A:85 D:90 |
| 3 | Certain | Hook scripts live at `fab/.kit/hooks/` | Follows constitution (kit is self-contained, portable via cp -r). New directory is analogous to `fab/.kit/scripts/`. | S:80 R:90 A:90 D:90 |
| 4 | Certain | Don't modify status.yaml template | `agent` block is ephemeral runtime state, not part of change lifecycle. Template should only contain lifecycle fields. | S:85 R:95 A:90 D:90 |
| 5 | Certain | Hook registration via fab-sync.sh sub-step (new script) | Clarified — user changed to: registration handled by fab-sync.sh, not out of scope | S:95 R:85 A:80 D:75 |
| 6 | Certain | Don't modify run.sh/dispatch.sh in this change | Clarified — user confirmed | S:95 R:90 A:75 D:80 |
| 7 | Certain | Use `git rev-parse --show-toplevel` for path resolution | Clarified — user confirmed | S:95 R:85 A:85 D:80 |
| 8 | Confident | Per-change `.status.yaml` is sufficient (no shared file) | Deferred — orchestrator polls per-worktree, shared file is a future concern if multi-session coordination materializes | S:60 R:70 A:60 D:75 |
<!-- clarified: shared file deferred — orchestrator already tracks worktree paths, per-change status sufficient for v1 -->

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-03-05 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 5 | Changed | "Hook registration via fab-sync.sh sub-step (new script)" |
| 6 | Confirmed | — |
| 7 | Confirmed | — |

### Session 2026-03-05 (taxonomy scan)

| # | Question | Resolution |
|---|----------|------------|
| 1 | fab-sync.sh registration mechanism | New `5-sync-hooks.sh` in `fab/.kit/sync/`, idempotent merge into `.claude/settings.local.json` |
| 2 | Affected Memory gap for fab-sync scope | Added `fab-workflow/setup` and `fab-workflow/distribution` |
| 3 | Shared file open question (#8) | Explicitly deferred — upgraded #8 to Confident, removed from Open Questions |
