# Hooks in Fab-Kit Skills

## Summary

Fab's Claude Code hooks are Go subcommands of the `fab` binary — `fab hook <subcommand>` — historically registered as inline command entries in `.claude/settings.local.json` by `fab hook sync` (invoked during `fab sync`). **As of the agent-state divestment (`ioku`), fab registers no hooks at all.** The three former session-scoped event handlers (`session-start`, `stop`, `user-prompt`) that wrote agent active/idle state, plus the earlier `artifact-write` PostToolUse handler, are all removed. The subcommands survive one release as silent no-op shims for un-migrated settings; the setup-facing `sync` is retained one release but now fully inert — it registers nothing and no longer rewrites legacy scripts (the rewrite rows were dropped so sync cannot re-mint the entries the `2.13.6-to-2.14.0` migration deletes); it has no removal path. All hook subcommands exit 0 so they can never block the agent.

**Hooks may enhance, never own — and fab now produces no agent state at all.** The three session handlers used to be push-by-nature runtime telemetry: they wrote `.fab-runtime.yaml` `_agents` entries (idle/active liveness) that the `fab pane` commands read. That whole *producer* subsystem was divested. Agent-state detection was never core fab — it is a tmux-context observation feature that got bolted onto fab because no owner existed. run-kit's `rk agent-setup` is that owner now. fab-kit stopped PRODUCING agent lifecycle state and became a pure CONSUMER of a shared tmux pane-option convention: the pane commands read `@rk_agent_state` (see `docs/specs/skills/SPEC-_cli-fab.md` § fab pane and `docs/memory/runtime/pane-commands.md`). The earlier `artifact-write` handler (a PostToolUse Write/Edit hook that recomputed artifact-derived `.status.yaml` state) was removed for a different reason: a hook fires only in the Claude harness, so correctness-critical pipeline state (change type, intake confidence, plan counts) must be pull-based via `fab status refresh`, not written only behind a hook (see § Artifact bookkeeping is pull-based).

Canonical command reference: `src/kit/skills/_cli-fab.md` § fab hook. Go source: `src/go/fab/cmd/fab/hook.go` + `src/go/fab/internal/hooklib/` + `src/go/fab/internal/refresh/`. Agent-state convention (read side): `docs/memory/runtime/runtime-agents.md`.

## Registered Hooks

`fab hook sync` registers **no hook mappings** — `hooklib.DefaultMappings` is empty. `sync` is retained one release but now **fully inert** — it registers nothing and no longer rewrites legacy scripts (the rewrite rows were dropped so sync cannot re-mint the entries the `2.13.6-to-2.14.0` migration deletes); it has **no removal path** — it merges desired entries (now none) but never deletes stale fab-managed entries. Full removal of the `sync` path and the no-op subcommand shims is a follow-up.

The deprecated no-op shims retained for one release:

| Subcommand | Former Claude Code event | Current behavior |
|------------|--------------------------|------------------|
| `session-start` | SessionStart | No-op — exits 0, emits nothing |
| `stop` | Stop | No-op — exits 0, emits nothing |
| `user-prompt` | UserPromptSubmit | No-op — exits 0, emits nothing |
| `artifact-write` | PostToolUse (Write/Edit) | No-op — exits 0, emits nothing (removed earlier, y022) |
| `sync` | n/a (setup) | Fully inert — registers nothing (empty mapping table) and no longer rewrites legacy scripts; idempotent |

The silence matters: an *unregistered* `fab hook <x>` subcommand exits 0 but prints cobra help to stdout, which a still-registered hook entry would feed to Claude Code as noisy non-JSON `additionalContext`. Each shim emits nothing, avoiding that until the `2.13.6-to-2.14.0` migration removes the three session settings entries (the `2.10.1-to-2.11.0` migration removed the `artifact-write` entries earlier).

## Agent state is a consumed convention (not a produced hook)

Agent active/idle/waiting state is no longer written by a fab hook. The `fab pane` commands READ a tmux **pane user option** `@rk_agent_state` (`"<state>:<epoch_seconds>"`, `state ∈ active | waiting | idle`) written by run-kit's `rk agent-setup` global agent-harness hooks (which cover Claude Code, Codex, Copilot, Gemini, OpenCode — not just Claude). fab reads it with plain tmux commands (`tmux show-options -pv` / the `#{@rk_agent_state}` list-panes format field), so there is no dependency on run-kit software being installed. The deleted producer subsystem was: the three hook writes, `WriteAgent`/`ClearAgent`/`ClearAgentIdle`, the throttled GC sweep (`last_run_gc`), the grandparent PID walker (`internal/proc`), the runtime file (`internal/runtime` / `.fab-runtime.yaml`), and the `_agents` matching in `internal/pane`. See `docs/specs/skills/SPEC-_cli-fab.md` § fab pane and `docs/memory/runtime/runtime-agents.md` for the read-side contract.

## Artifact bookkeeping is pull-based (not a hook)

Artifact-derived `.status.yaml` state — `change_type` + `confidence` (from `intake.md`) and `plan.generated`/`plan.task_count`/`plan.acceptance_count`/`plan.acceptance_completed` (from `plan.md`) — is recomputed on demand by **`fab status refresh <change>`** (`src/go/fab/internal/refresh`), not by a hook. `refresh` inspects both artifacts on disk under the cross-process status lock (single load-mutate-save), respecting the `change_type_source: explicit` guard (a human's `set-change-type` is never overwritten) and leaving a field untouched when its section heading is absent:

| Artifact | Recomputed by `fab status refresh` |
|----------|------------------------------------|
| `intake.md` | Infer `change_type` from content (only when `change_type_source` is absent/`inferred`) + recompute the authoritative intake confidence (`fab score` equivalent) |
| `plan.md` | Set `plan.generated: true` (when `## Tasks` exists), re-count `plan.task_count` (checkbox items under `## Tasks`), `plan.acceptance_count` / `plan.acceptance_completed` (under `## Acceptance`) |

`refresh` is **self-healed at the transition seams** — `fab status advance`, `fab status finish`, and `fab preflight` each run it before their read/transition — so no skill has to remember to call it, and a hook-bypassing edit (sed, direct write) or a non-Claude agent's artifact write is reflected before the next stage reads the fields. This is why skills no longer carry post-write bookkeeping instructions for these fields, and why the removed hook did not need to be replaced with another hook.

No git staging is performed on this path (the removed hook's best-effort `git add` of `.status.yaml`/`.history.jsonl` is dropped — `/git-pr` stages status/history at ship). A one-release no-op shim `fab hook artifact-write` is retained for un-migrated projects whose settings still register it (it exits 0 and emits nothing on stdout); the `2.10.1-to-2.11.0` migration removes the settings entry.

## Registration (`fab hook sync`)

- Runs as part of `fab sync`; can be invoked standalone.
- Registers **no** inline `fab hook <subcommand>` entries — the mapping table is empty. Re-running is a no-op (`.claude/settings.local.json hooks: OK`).
- Does **not** rewrite legacy old-style entries (`bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-*.sh` and relative variants) into inline `fab hook <sub>` commands — those rewrite rows were dropped so sync cannot re-mint the entries the `2.13.6-to-2.14.0` migration deletes. The `sync` path lives one more release only to tolerate un-migrated stale settings (it leaves them untouched), not to migrate anything.
- Output: the OK line (the `Created`/`Updated` branches are unreachable with the empty mapping table — a fresh sync always reports OK, nothing to register); on failure (no fab root, unwritable settings) a `hook sync: {error}` line on stderr — exit code stays 0 either way.

## What Stays in Skills (not hooks)

Agent decisions remain skill-instructed; only mechanical bookkeeping ever moved into hooks (and that too is gone now):

| Command | Why not a hook |
|---------|----------------|
| `fab status finish/start/advance/fail/reset/skip` | Stage transitions are intentional agent judgments |
| `fab log command "<skill>" "<change>"` | Skill invocation can't be detected from hook events (no "which skill is running" matcher); best-effort, always exits 0 |

One skill still writes status YAML directly via `yq`: `/git-pr-review` tracks ephemeral sub-state in `stage_metrics.review-pr.phase` (`received` → `triaging` → `fixing` → `pushed` → `replying`) and `stage_metrics.review-pr.reviewer` — best-effort `yq -i` writes, since no `fab status` subcommand covers arbitrary metric fields.

## Event Coverage

fab-kit uses **no** Claude Code hook events. Every event was assessed; the two that were once in use are now divested:

| Event | Status | Rationale |
|-------|--------|-----------|
| **Stop** | Removed (`stop` now a no-op shim) | Agent active/idle state divested to run-kit's `@rk_agent_state` pane-option convention — fab is a consumer, not a producer |
| **SessionStart** | Removed (`session-start` now a no-op shim) | Same — cleared the `_agents` entry, which no longer exists |
| **UserPromptSubmit** | Removed (`user-prompt` now a no-op shim) | Same — cleared `idle_since`, which no longer exists |
| **PostToolUse** (Write/Edit) | Removed (`artifact-write` no-op shim, y022) | Artifact-derived state is correctness-critical, so it is pull-based (`fab status refresh`, self-healed at the transition seams), not owned by a hook that fires only in the Claude harness |
| PreToolUse | Not used | Guardrails belong in `fab/project/*` policy files |
| PermissionRequest | Not used | Fab shouldn't auto-approve tool calls |
| SubagentStop / SubagentStart | Not used | Skills already handle subagent results |
| SessionEnd / PreCompact / Notification / others | Not used | Thin value; change-folder artifacts survive compaction |

The no-op shims are command-type hooks in name only — they consume no stdin and emit no stdout. Full removal of the shims and the `sync` registration path is a follow-up once the `2.13.6-to-2.14.0` migration has propagated.
