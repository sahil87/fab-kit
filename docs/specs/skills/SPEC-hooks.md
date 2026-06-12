# Hooks in Fab-Kit Skills

## Summary

Fab's Claude Code hooks are Go subcommands of the `fab` binary — `fab hook <subcommand>` — registered as inline command entries in `.claude/settings.local.json` by `fab hook sync` (invoked during `fab sync`). There are four event handlers (`session-start`, `stop`, `user-prompt`, `artifact-write`) plus the setup-facing `sync`. All hook subcommands exit 0 so they can never block the agent: the event handlers swallow errors silently; `sync` surfaces failures on stderr but still exits 0. The former shell-script hooks (`on-stop.sh`, `on-session-start.sh`, registered by `5-sync-hooks.sh`) and the proposed `fab runtime` subcommands are gone — handlers call the internal `runtime` package directly, with no `yq` dependency anywhere on the hook path.

Canonical command reference: `src/kit/skills/_cli-fab.md` § fab hook. Go source: `src/go/fab/cmd/fab/hook.go` + `src/go/fab/internal/hooklib/`. Runtime-file schema: `docs/memory/runtime/runtime-agents.md`.

## Registered Hooks

`fab hook sync` registers these mappings (one settings entry per row; `artifact-write` gets two — one per matcher):

| Subcommand | Claude Code event | Matcher | What it does |
|------------|-------------------|---------|--------------|
| `session-start` | **SessionStart** | — | Delete `_agents[session_id]` from `.fab-runtime.yaml` |
| `stop` | **Stop** | — | Write `_agents[session_id]` with `idle_since` plus optional `change`/`pid`/`tmux_server`/`tmux_pane`/`transcript_path` |
| `user-prompt` | **UserPromptSubmit** | — | Remove only `idle_since` from `_agents[session_id]`; other fields preserved |
| `artifact-write` | **PostToolUse** | `Write`, `Edit` | Per-artifact bookkeeping (see below) |
| `sync` | n/a (setup) | — | Register the rows above in `.claude/settings.local.json`; idempotent |

The three session-scoped handlers (`session-start`, `stop`, `user-prompt`) read a JSON payload on stdin with at least `session_id` (and optionally `transcript_path`). Malformed JSON or a missing `session_id` is silently skipped. Each invocation also runs a throttled GC sweep (at most once per 180 s, tracked via `last_run_gc`) that prunes `_agents` entries whose stored `pid` no longer exists.

## Runtime File (`.fab-runtime.yaml`)

Ephemeral per-worktree state written by the session-scoped hooks and consumed by `fab pane map` / `fab pane send` (agent idle detection):

```yaml
_agents:
  "<session_id>":
    idle_since: <unix-ts>       # present when the agent is idle
    change: "<folder-name>"     # optional — absent in discussion mode
    pid: <int>                  # optional — Claude's PID, used for GC liveness
    tmux_server: "<label>"      # optional
    tmux_pane: "%15"            # optional
    transcript_path: "..."      # optional
last_run_gc: <unix-ts>          # throttles GC sweeps
```

Per-session map keyed by `session_id` — not the legacy `agent.idle_since` singleton. Full schema: `docs/memory/runtime/runtime-agents.md`.

## artifact-write Bookkeeping

`artifact-write` parses a different payload shape — `tool_input.file_path` from the PostToolUse JSON — and does not participate in `_agents` writes. It pattern-matches the path against `fab/changes/{name}/intake.md|plan.md` (`spec.md` was dropped as a recognized artifact in 1.10.0), then mutates `.status.yaml` under the cross-process status lock:

| Artifact written | Bookkeeping performed |
|------------------|----------------------|
| `intake.md` | Infer `change_type` from content (`fab status set-change-type` equivalent) + recompute the authoritative intake confidence (`fab score` equivalent) |
| `plan.md` | Set `plan.generated: true` (when `## Tasks` exists), re-count `plan.task_count` (checkbox items under `## Tasks`), `plan.acceptance_count` / `plan.acceptance_completed` (under `## Acceptance`) |

Side effects: auto-stages the change's `.status.yaml` and `.history.jsonl` via `git add` (so status writes never block git operations), and emits `{"additionalContext": "Bookkeeping: ..."}` on stdout to inform the agent of what was recorded.

This is why skills no longer carry post-write bookkeeping instructions for these fields — the hook owns them mechanically.

## Registration (`fab hook sync`)

- Runs as part of `fab sync`; can be invoked standalone.
- Merges the inline `fab hook <subcommand>` entries into `.claude/settings.local.json`, deduplicating by matcher + command pair — re-running is a no-op (`.claude/settings.local.json hooks: OK`).
- Migrates old-style entries (`bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-*.sh` and relative variants) to the inline commands.
- Output: `Created`, `Updated`, or the OK line; on failure (no fab root, unwritable settings) a `hook sync: {error}` line on stderr — exit code stays 0 either way.

## What Stays in Skills (not hooks)

Agent decisions remain skill-instructed; only mechanical bookkeeping moved into hooks:

| Command | Why not a hook |
|---------|----------------|
| `fab status finish/start/advance/fail/reset/skip` | Stage transitions are intentional agent judgments |
| `fab log command "<skill>" "<change>"` | Skill invocation can't be detected from hook events (no "which skill is running" matcher); best-effort, always exits 0 |

One skill still writes status YAML directly via `yq`: `/git-pr-review` tracks ephemeral sub-state in `stage_metrics.review-pr.phase` (`received` → `triaging` → `fixing` → `pushed` → `replying`) and `stage_metrics.review-pr.reviewer` — best-effort `yq -i` writes, since no `fab status` subcommand covers arbitrary metric fields.

## Event Coverage

Of the Claude Code hook events, fab-kit uses four. The rest were assessed and rejected:

| Event | Status | Rationale |
|-------|--------|-----------|
| **Stop** | In use (`stop`) | Idle tracking |
| **SessionStart** | In use (`session-start`) | Clear stale agent entry on session start/resume/clear |
| **UserPromptSubmit** | In use (`user-prompt`) | Clear `idle_since` the moment the user engages (agent no longer idle) |
| **PostToolUse** (Write/Edit) | In use (`artifact-write`) | Artifact bookkeeping + `additionalContext` |
| PreToolUse | Not used | Guardrails belong in `fab/project/*` policy files |
| PermissionRequest | Not used | Fab shouldn't auto-approve tool calls |
| SubagentStop / SubagentStart | Not used | Skills already handle subagent results |
| SessionEnd / PreCompact / Notification / others | Not used | Thin value; change-folder artifacts survive compaction |

All handlers are command-type hooks (shell command, JSON on stdin) — fab uses no prompt/agent/HTTP hook types.
