# Hooks in Fab-Kit Skills

## Summary

Fab's Claude Code hooks are Go subcommands of the `fab` binary — `fab hook <subcommand>` — registered as inline command entries in `.claude/settings.local.json` by `fab hook sync` (invoked during `fab sync`). There are three event handlers (`session-start`, `stop`, `user-prompt`) plus the setup-facing `sync`. All hook subcommands exit 0 so they can never block the agent: the event handlers swallow errors silently; `sync` surfaces failures on stderr but still exits 0. The former shell-script hooks (`on-stop.sh`, `on-session-start.sh`, registered by `5-sync-hooks.sh`) and the proposed `fab runtime` subcommands are gone — handlers call the internal `runtime` package directly, with no `yq` dependency anywhere on the hook path.

**Hooks may enhance, never own.** The three registered handlers are all push-by-nature runtime telemetry (liveness/idle tracking) — they own no correctness-critical state and degrade gracefully. The former fourth handler, `artifact-write` (a PostToolUse Write/Edit hook that recomputed artifact-derived `.status.yaml` state), was **removed**: a hook fires only in the Claude harness, so artifact-derived pipeline state (change type, intake confidence, plan counts) — which *is* correctness-critical — must be pull-based, not written only behind a hook. That recompute moved to the pull-based `fab status refresh`, self-healed at the transition seams (see § Artifact bookkeeping is pull-based).

Canonical command reference: `src/kit/skills/_cli-fab.md` § fab hook. Go source: `src/go/fab/cmd/fab/hook.go` + `src/go/fab/internal/hooklib/` + `src/go/fab/internal/refresh/`. Runtime-file schema: `docs/memory/runtime/runtime-agents.md`.

## Registered Hooks

`fab hook sync` registers these mappings (one settings entry per row):

| Subcommand | Claude Code event | Matcher | What it does |
|------------|-------------------|---------|--------------|
| `session-start` | **SessionStart** | — | Delete `_agents[session_id]` from `.fab-runtime.yaml` |
| `stop` | **Stop** | — | Write `_agents[session_id]` with `idle_since` plus optional `change`/`pid`/`tmux_server`/`tmux_pane`/`transcript_path` |
| `user-prompt` | **UserPromptSubmit** | — | Remove only `idle_since` from `_agents[session_id]`; other fields preserved |
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

## Artifact bookkeeping is pull-based (not a hook)

Artifact-derived `.status.yaml` state — `change_type` + `confidence` (from `intake.md`) and `plan.generated`/`plan.task_count`/`plan.acceptance_count`/`plan.acceptance_completed` (from `plan.md`) — is recomputed on demand by **`fab status refresh <change>`** (`src/go/fab/internal/refresh`), not by a hook. `refresh` inspects both artifacts on disk under the cross-process status lock (single load-mutate-save), respecting the `change_type_source: explicit` guard (a human's `set-change-type` is never overwritten) and leaving a field untouched when its section heading is absent:

| Artifact | Recomputed by `fab status refresh` |
|----------|------------------------------------|
| `intake.md` | Infer `change_type` from content (only when `change_type_source` is absent/`inferred`) + recompute the authoritative intake confidence (`fab score` equivalent) |
| `plan.md` | Set `plan.generated: true` (when `## Tasks` exists), re-count `plan.task_count` (checkbox items under `## Tasks`), `plan.acceptance_count` / `plan.acceptance_completed` (under `## Acceptance`) |

`refresh` is **self-healed at the transition seams** — `fab status advance`, `fab status finish`, and `fab preflight` each run it before their read/transition — so no skill has to remember to call it, and a hook-bypassing edit (sed, direct write) or a non-Claude agent's artifact write is reflected before the next stage reads the fields. This is why skills no longer carry post-write bookkeeping instructions for these fields, and why the removed hook did not need to be replaced with another hook.

No git staging is performed on this path (the removed hook's best-effort `git add` of `.status.yaml`/`.history.jsonl` is dropped — `/git-pr` stages status/history at ship). A one-release no-op shim `fab hook artifact-write` is retained for un-migrated projects whose settings still register it (it exits 0 and emits nothing on stdout, so it cannot feed noisy non-JSON `additionalContext`); the `2.10.1-to-2.11.0` migration removes the settings entry.

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

Of the Claude Code hook events, fab-kit uses three. The rest were assessed and rejected:

| Event | Status | Rationale |
|-------|--------|-----------|
| **Stop** | In use (`stop`) | Idle tracking |
| **SessionStart** | In use (`session-start`) | Clear stale agent entry on session start/resume/clear |
| **UserPromptSubmit** | In use (`user-prompt`) | Clear `idle_since` the moment the user engages (agent no longer idle) |
| **PostToolUse** (Write/Edit) | Not used (was `artifact-write`) | Artifact-derived state is correctness-critical, so it is pull-based (`fab status refresh`, self-healed at the transition seams), not owned by a hook that fires only in the Claude harness |
| PreToolUse | Not used | Guardrails belong in `fab/project/*` policy files |
| PermissionRequest | Not used | Fab shouldn't auto-approve tool calls |
| SubagentStop / SubagentStart | Not used | Skills already handle subagent results |
| SessionEnd / PreCompact / Notification / others | Not used | Thin value; change-folder artifacts survive compaction |

All handlers are command-type hooks (shell command, JSON on stdin) — fab uses no prompt/agent/HTTP hook types.
