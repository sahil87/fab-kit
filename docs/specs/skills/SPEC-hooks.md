# Hooks in Fab-Kit Skills

## Current Hooks

Two Claude Code hooks exist today, both registered via `fab/.kit/sync/5-sync-hooks.sh`:

| Hook | Event | File | Fires |
|------|-------|------|-------|
| Agent idle tracking | **Stop** | `fab/.kit/hooks/on-stop.sh` | Every agent response turn |
| Agent session clear | **SessionStart** | `fab/.kit/hooks/on-session-start.sh` | Every new/resumed session |

**What they do**: Write/clear `agent.idle_since` in `.fab-runtime.yaml` (implemented in change 1lwf).

**Problem**: Both hooks use `yq` directly instead of the `fab` CLI. If `yq` isn't installed, they silently do nothing (`command -v yq || exit 0`). See [yq Dependency](#yq-dependency-in-hooks) below.

---

## Hooks Embedded in Skills (via yq, not fab CLI)

`/git-pr-review` writes directly to `.status.yaml` using `yq -i` (bypassing the Go binary):

| Field | When | Why not fab CLI |
|-------|------|-----------------|
| `stage_metrics.review-pr.phase` | At each phase transition (waiting/received/triaging/fixing/pushed) | No `fab status` subcommand for arbitrary metric fields |
| `stage_metrics.review-pr.reviewer` | When reviews detected | Same |

These are ephemeral runtime state (only meaningful during review-pr execution). Good candidate for `.fab-runtime.yaml`.

---

## Bookkeeping Commands in Skills (hook candidates)

These are `fab` CLI calls that skills instruct the agent to run after generating artifacts. They're fragile because the agent may forget or skip them.

### Grouped by artifact trigger

**After `intake.md` is written** (fab-new):
| Command | Purpose | Hookable? |
|---------|---------|-----------|
| `fab status set-change-type <change> <type>` | Infer and record change type | Yes — hook reads content, does keyword match |
| `fab score --stage intake <change>` | Compute indicative confidence | Yes — hook calls fab score |
| `fab status advance <change> intake` | Signal intake artifact exists | **No** — must happen after SRAD questions (Step 8), not on write |

**After `spec.md` is written** (fab-continue, fab-ff, fab-fff):
| Command | Purpose | Hookable? |
|---------|---------|-----------|
| `fab score <change>` | Compute spec confidence score | Yes |

**After `tasks.md` is written** (fab-continue, fab-ff, fab-fff):
| Command | Purpose | Hookable? |
|---------|---------|-----------|
| `fab status set-checklist <change> total <N>` | Record task count | Yes — hook counts `- [ ]` lines |

**After `checklist.md` is written** (fab-continue, fab-ff, fab-fff):
| Command | Purpose | Hookable? |
|---------|---------|-----------|
| `fab status set-checklist <change> generated true` | Mark checklist as generated | Yes |
| `fab status set-checklist <change> total <N>` | Record checklist item count | Yes — hook counts `- [ ]` lines |
| `fab status set-checklist <change> completed 0` | Initialize completed count | Yes |

**After `spec.md` is edited** (fab-clarify, suggest mode only):
| Command | Purpose | Hookable? |
|---------|---------|-----------|
| `fab score <change>` | Recompute confidence after clarification | Yes |

---

## Command Logging (every skill)

Every skill per `_preamble.md` §2 runs:

```bash
fab log command "<skill-name>" "<change-id>" 2>/dev/null || true
```

This is best-effort, scattered across all skills. Not a hook candidate — skill invocation can't be detected from hook events (no matcher for "which skill is running").

---

## Stage Transitions (NOT hook candidates)

These are intentional agent decisions — the agent decides when a stage is complete:

| Command | Skills | Why not a hook |
|---------|--------|----------------|
| `fab status finish <change> <stage>` | fab-continue, fab-ff, fab-fff, git-pr, git-pr-review | Agent must judge when work is actually complete |
| `fab status start <change> <stage>` | fab-continue, git-pr, git-pr-review | Agent must decide when to begin |
| `fab status advance <change> <stage>` | fab-new, fab-continue | Agent signals artifact readiness |
| `fab status fail <change> <stage>` | fab-continue, fab-ff, fab-fff | Agent/sub-agent determines review failed |
| `fab status reset <change> <stage>` | fab-continue, fab-ff, fab-fff | Agent decides to restart from a stage |

---

## Possible Events to Use

All 18 Claude Code hook events, assessed for fab-kit relevance.

### Can block/deny actions (8 events)

| Event | Fires when | Matcher | Can block | Hook types | Fab-kit fit |
|-------|-----------|---------|-----------|------------|-------------|
| **PreToolUse** | Before any tool executes | Tool name (`Write`, `Bash`, `Edit`, `Glob`, `Grep`, `Agent`, `mcp__*`) | Yes — deny or modify input | Command, prompt, agent | No — guardrails belong in `project/*` files |
| **PostToolUse** | After a tool succeeds | Tool name | No (tool already ran) — but can return `additionalContext` | Command, prompt, agent | **High** — detect artifact writes/edits, trigger bookkeeping automatically |
| **UserPromptSubmit** | User submits a prompt, before Claude processes it | None (fires on every prompt) | Yes — block prompt | Command, prompt, agent | No — context injection is skill-level (preamble handles it) |
| **PermissionRequest** | Permission dialog about to show | Tool name | Yes — allow/deny on behalf of user | Command, prompt, agent | No — fab-kit shouldn't auto-approve tool calls |
| **Stop** | Agent finishes responding | None | Yes — force continuation | Command, prompt, agent | **In use** — idle tracking |
| **SubagentStop** | Subagent finishes | Agent type (`Explore`, `Plan`, custom) | Yes — force continuation | Command, prompt, agent | No — skills already handle subagent results |
| **TaskCompleted** | Todo item marked complete | None | Yes — force continuation | Command, prompt, agent | No — fab doesn't use Claude's built-in todos |
| **ConfigChange** | Settings file changes | Config source (`user_settings`, `project_settings`, etc.) | Yes (except policy) | Command | No |

### Observe only, cannot block (10 events)

| Event | Fires when | Matcher | Hook types | Fab-kit fit |
|-------|-----------|---------|------------|-------------|
| **SessionStart** | Session begins/resumes/clears/compacts | Source (`startup`, `resume`, `clear`, `compact`) | Command only | **In use** — clear idle state |
| **SessionEnd** | Session terminates | Exit reason (`clear`, `logout`, `prompt_input_exit`, etc.) | Command only | No — thin value, nowhere useful to log |
| **PreCompact** | Before context compaction | Trigger (`manual`, `auto`) | Command only | No — change folder artifacts survive compaction |
| **PostToolUseFailure** | After a tool fails | Tool name | Command, prompt, agent | No — not enough value to justify a hook |
| **InstructionsLoaded** | CLAUDE.md or rules load | None (no matcher) | Command only | No |
| **SubagentStart** | Subagent spawned | Agent type | Command only | No |
| **Notification** | Notification sent | Type (`permission_prompt`, `idle_prompt`, etc.) | Command only | No |
| **TeammateIdle** | Teammate about to idle | None | Command, prompt, agent | No |
| **WorktreeCreate** | Worktree being created | None | Command only (must print path to stdout) | No |
| **WorktreeRemove** | Worktree being removed | None | Command only | No |

### Hook types available

| Type | How it works | Available for | Speed |
|------|-------------|---------------|-------|
| **Command** | Shell script, JSON on stdin, exit code controls decision | All 18 events | Fast |
| **HTTP** | POST to endpoint, JSON body, response controls decision | All 18 events | Medium |
| **Prompt** | Single LLM call (Haiku) for yes/no decision | PreToolUse, PostToolUse, PostToolUseFailure, UserPromptSubmit, PermissionRequest, Stop, SubagentStop, TaskCompleted | Slow |
| **Agent** | Multi-turn subagent with Read/Grep/Glob tools | Same as Prompt | Slowest |

### Key data available in PostToolUse (Write/Edit)

The primary hook for fab-kit's bookkeeping redesign:

```json
{
  "hook_event_name": "PostToolUse",
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/path/to/fab/changes/260306-1lwf-extract-agent-runtime-file/intake.md",
    "content": "..."
  },
  "tool_response": {
    "filePath": "/path/...",
    "success": true
  }
}
```

For `Edit`, `tool_input` contains `file_path`, `old_string`, `new_string` instead of `content`.

The hook script can:
1. Extract `file_path` from the JSON
2. Pattern-match against `fab/changes/*/intake.md|spec.md|tasks.md|checklist.md`
3. Derive the change name from path components
4. Run the appropriate `fab` CLI bookkeeping commands
5. Return `additionalContext` in stdout JSON to inform the agent

---

## yq Dependency in Hooks

### Problem

Both existing hooks (`on-stop.sh`, `on-session-start.sh`) and `/git-pr-review` use `yq` directly to write YAML. This is problematic:

1. **Silent failure** — hooks do `command -v yq || exit 0`. If yq isn't installed, they silently do nothing
2. **Inconsistency** — every other status operation uses the `fab` CLI. Hooks bypassing it is an anomaly
3. **Extra dependency** — the Go binary is always present (it's the kit's own binary). `yq` is an external tool

### Current yq usage in fab/.kit/

| Location | Uses | Purpose |
|----------|------|---------|
| `hooks/on-stop.sh` | 1 | Write `agent.idle_since` to `.fab-runtime.yaml` |
| `hooks/on-session-start.sh` | 1 | Delete `agent` block from `.fab-runtime.yaml` |
| `scripts/fab-doctor.sh` | 1 | Check yq is installed (diagnostic) |

### Proposal: `fab runtime` subcommands

Absorb the hook `yq` calls into the Go binary:

| Command | Purpose | Replaces |
|---------|---------|----------|
| `fab runtime set-idle <change>` | Write `agent.idle_since` timestamp to `.fab-runtime.yaml` | `yq -i` in on-stop.sh |
| `fab runtime clear-idle <change>` | Delete `agent` block from `.fab-runtime.yaml` | `yq -i del()` in on-session-start.sh |

The hooks simplify from ~30 lines (with yq dependency check, file creation, quoting) to ~15 lines calling `fab runtime`.

The pipeline orchestrator (`run.sh`, `dispatch.sh`) was removed in change o1tu, eliminating its ~43 `yq` calls.

---

## Proposed Hook Architecture (Trimmed)

```
Claude Code Hook Events                    Fab-Kit Hook Scripts
─────────────────────────                  ─────────────────────

SessionStart ──────────► on-session-start.sh        (existing, updated)
                         └─ fab runtime clear-idle <change>

Stop ──────────────────► on-stop.sh                 (existing, updated)
                         └─ fab runtime set-idle <change>

PostToolUse (Write) ───► on-artifact-write.sh       ◄── NEW
                         ├─ intake.md → fab status set-change-type + fab score --stage intake
                         ├─ spec.md → fab score
                         ├─ tasks.md → fab status set-checklist total <N>
                         └─ checklist.md → fab status set-checklist generated + total + completed

PostToolUse (Edit) ────► on-artifact-write.sh       (same script, both matchers)
                         └─ spec.md → fab score
```

**Dropped** (from earlier proposal):
- ~~PreCompact~~ — change folder artifacts survive compaction
- ~~SessionEnd~~ — thin value, nowhere useful to log
- ~~SessionStart enhancement~~ — skills already handle context loading

### What this changes in skills

Skills that currently contain bookkeeping instructions can drop them:

| Skill | Steps removed | Net effect |
|-------|---------------|------------|
| fab-new | Steps 6 (type inference), 7 (confidence) | Simpler, 2 fewer Bash calls |
| fab-continue | Score after spec, checklist after tasks | Fewer bookkeeping instructions |
| fab-ff | Step 4 (3 set-checklist calls) | Cleaner pipeline |
| fab-fff | Same as fab-ff | Cleaner pipeline |
| fab-clarify | Step 7 (recompute confidence) | One fewer Bash call |
| _generation.md | Checklist procedure step 6 | 3 fewer Bash calls |

**Stays in skills** (agent decisions, not mechanical bookkeeping):
- `fab status advance` — agent signals artifact readiness (e.g., after SRAD questions)
- `fab status finish/start/fail/reset` — stage transitions are intentional
- `fab log command` — skill invocation logging (can't detect skill name from hooks)

### Registration in 5-sync-hooks.sh

The sync script needs updating to support **matchers** for PostToolUse hooks:

```json
{
  "PostToolUse": [
    {
      "matcher": "Write",
      "hooks": [{"type": "command", "command": "bash fab/.kit/hooks/on-artifact-write.sh"}]
    },
    {
      "matcher": "Edit",
      "hooks": [{"type": "command", "command": "bash fab/.kit/hooks/on-artifact-write.sh"}]
    }
  ]
}
```

The current `map_event()` function maps filenames to events. This needs extending to handle event+matcher pairs.
