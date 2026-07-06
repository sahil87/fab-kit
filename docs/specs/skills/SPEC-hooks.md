# Hooks in Fab-Kit Skills

## Summary

Fab-kit uses **no Claude Code hooks at all.** Historically fab registered inline `fab hook <subcommand>` command entries in `.claude/settings.local.json` (via `fab hook sync`, invoked during `fab sync`). The agent-state divestment (`ioku`) removed the last of them: the `fab hook` command family — the three former session-scoped handlers (`session-start`, `stop`, `user-prompt`) that wrote agent active/idle state, the earlier `artifact-write` PostToolUse handler, and the setup-facing `sync` — was **removed outright in 2.14.0, with no deprecation shim period.** fab registers, writes, and owns no hook.

**Hooks may enhance, never own — and fab now produces no agent state at all.** The three session handlers used to be push-by-nature runtime telemetry: they wrote `.fab-runtime.yaml` `_agents` entries (idle/active liveness) that the `fab pane` commands read. That whole *producer* subsystem was divested. Agent-state detection was never core fab — it is a tmux-context observation feature that got bolted onto fab because no owner existed. run-kit's `rk agent-setup` is that owner now. fab-kit stopped PRODUCING agent lifecycle state and became a pure CONSUMER of a shared tmux pane-option convention: the pane commands read `@rk_agent_state` (see `docs/specs/skills/SPEC-_cli-fab.md` § fab pane and `docs/memory/runtime/pane-commands.md`). The earlier `artifact-write` handler (a PostToolUse Write/Edit hook that recomputed artifact-derived `.status.yaml` state) was removed for a different reason: a hook fires only in the Claude harness, so correctness-critical pipeline state (change type, intake confidence, plan counts) must be pull-based via `fab status refresh`, not written only behind a hook (see § Artifact bookkeeping is pull-based).

Canonical command reference: `src/kit/skills/_cli-fab.md` § fab hook (removed in 2.14.0). Go source: `src/go/fab/internal/refresh/` (the pull-based successor). Agent-state convention (read side): `docs/memory/runtime/runtime-agents.md`.

## Removed command family (2.14.0)

The `fab hook` command family was deleted outright — command file, subcommands, and the `Sync` registration path all gone. There is **no shim period**: an un-migrated `.claude/settings.local.json` that still invokes any of these will get a cobra *unknown command* error (exit 1) until the `2.13.6-to-2.14.0` migration removes the entries.

| Subcommand | Former Claude Code event | Disposition |
|------------|--------------------------|-------------|
| `session-start` | SessionStart | Removed — agent-state production divested to `@rk_agent_state` |
| `stop` | Stop | Removed — same |
| `user-prompt` | UserPromptSubmit | Removed — same |
| `artifact-write` | PostToolUse (Write/Edit) | Removed — bookkeeping moved to pull-based `fab status refresh` (removed earlier as a shim in y022; the shim itself is now gone too) |
| `sync` | n/a (setup) | Removed — fab registers no hooks, so there is nothing to sync |

The `2.13.6-to-2.14.0` migration removes any lingering session-scoped settings entries (both the inline `fab hook …` and legacy `bash …/on-*.sh` forms) and deletes the dead `.fab-runtime.yaml`/`.fab-runtime.yaml.lock` files; the `2.10.1-to-2.11.0` migration removed the `artifact-write` PostToolUse entry earlier.

## Agent state is a consumed convention (not a produced hook)

Agent active/idle/waiting state is no longer written by a fab hook. The `fab pane` commands READ a tmux **pane user option** `@rk_agent_state` (`"<state>:<epoch_seconds>"`, `state ∈ active | waiting | idle`) written by run-kit's `rk agent-setup` global agent-harness hooks (which cover Claude Code, Codex, Copilot, Gemini, OpenCode — not just Claude). fab reads it with plain tmux commands (`tmux show-options -pv` / the `#{@rk_agent_state}` list-panes format field), so there is no dependency on run-kit software being installed. The deleted producer subsystem was: the three hook writes, `WriteAgent`/`ClearAgent`/`ClearAgentIdle`, the throttled GC sweep (`last_run_gc`), the grandparent PID walker (`internal/proc`), the runtime file (`internal/runtime` / `.fab-runtime.yaml`), and the `_agents` matching in `internal/pane`. See `docs/specs/skills/SPEC-_cli-fab.md` § fab pane and `docs/memory/runtime/runtime-agents.md` for the read-side contract.

## Artifact bookkeeping is pull-based (not a hook)

Artifact-derived `.status.yaml` state — `change_type` + `confidence` (from `intake.md`) and `plan.generated`/`plan.task_count`/`plan.acceptance_count`/`plan.acceptance_completed` (from `plan.md`) — is recomputed on demand by **`fab status refresh <change>`** (`src/go/fab/internal/refresh`), not by a hook. `refresh` inspects both artifacts on disk under the cross-process status lock (single load-mutate-save), respecting the `change_type_source: explicit` guard (a human's `set-change-type` is never overwritten) and leaving a field untouched when its section heading is absent:

| Artifact | Recomputed by `fab status refresh` |
|----------|------------------------------------|
| `intake.md` | Infer `change_type` from content (only when `change_type_source` is absent/`inferred`) + recompute the authoritative intake confidence (`fab score` equivalent) |
| `plan.md` | Set `plan.generated: true` (when `## Tasks` exists), re-count `plan.task_count` (checkbox items under `## Tasks`), `plan.acceptance_count` / `plan.acceptance_completed` (under `## Acceptance`) |

`refresh` is **self-healed at the transition seams** — `fab status advance`, `fab status finish`, and `fab preflight` each run it before their read/transition — so no skill has to remember to call it, and a hook-bypassing edit (sed, direct write) or a non-Claude agent's artifact write is reflected before the next stage reads the fields. This is why skills no longer carry post-write bookkeeping instructions for these fields, and why the removed hook did not need to be replaced with another hook.

No git staging is performed on this path (the removed hook's best-effort `git add` of `.status.yaml`/`.history.jsonl` is dropped — `/git-pr` stages status/history at ship). The `2.10.1-to-2.11.0` migration removed the `artifact-write` settings entry.

## What Stays in Skills (not hooks)

Agent decisions remain skill-instructed; only mechanical bookkeeping ever moved into hooks (and that too is gone now):

| Command | Why not a hook |
|---------|----------------|
| `fab status finish/start/advance/fail/reset/skip` | Stage transitions are intentional agent judgments |
| `fab log command "<skill>" "<change>"` | Skill invocation can't be detected from hook events (no "which skill is running" matcher); best-effort, always exits 0 |

One skill still writes status YAML directly via `yq`: `/git-pr-review` tracks ephemeral sub-state in `stage_metrics.review-pr.phase` (`received` → `triaging` → `fixing` → `pushed` → `replying`) and `stage_metrics.review-pr.reviewer` — best-effort `yq -i` writes, since no `fab status` subcommand covers arbitrary metric fields.

## Event Coverage

fab-kit uses **no** Claude Code hook events. Every event was assessed; the two that were once in use are now divested and their handlers removed:

| Event | Status | Rationale |
|-------|--------|-----------|
| **Stop** | Removed | Agent active/idle state divested to run-kit's `@rk_agent_state` pane-option convention — fab is a consumer, not a producer |
| **SessionStart** | Removed | Same — cleared the `_agents` entry, which no longer exists |
| **UserPromptSubmit** | Removed | Same — cleared `idle_since`, which no longer exists |
| **PostToolUse** (Write/Edit) | Removed (`artifact-write`, y022) | Artifact-derived state is correctness-critical, so it is pull-based (`fab status refresh`, self-healed at the transition seams), not owned by a hook that fires only in the Claude harness |
| PreToolUse | Not used | Guardrails belong in `fab/project/*` policy files |
| PermissionRequest | Not used | Fab shouldn't auto-approve tool calls |
| SubagentStop / SubagentStart | Not used | Skills already handle subagent results |
| SessionEnd / PreCompact / Notification / others | Not used | Thin value; change-folder artifacts survive compaction |

Note: `stage_hooks` in `fab/project/config.yaml` is a **separate, unrelated** mechanism — optional per-stage pre/post shell commands honored by `fab status start`/`finish`. It is not a Claude Code hook and was unaffected by this change.
