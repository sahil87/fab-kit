# Intake: Redesign Hooks Strategy

**Change**: 260306-6bba-redesign-hooks-strategy
**Created**: 2026-03-06
**Status**: Draft

## Origin

> Redesign fab-kit's use of Claude Code hooks. Add PostToolUse hook for automatic artifact bookkeeping, migrate existing hooks from yq to fab CLI, archive stale changes, update Constitution §I.

Interaction mode: deep conversational. Preceded by:
1. `/fab-discuss` analysis of four related intakes (4vj0, qg80, rwt1, shk2) — identified they're overlapping/rejected
2. Discussion of Constitution §I wording — "shell scripts" → "scripts"
3. Comprehensive research of all 18 Claude Code hook events
4. Per-skill walkthrough of all fab-* and git-* skills identifying hook candidates and non-candidates
5. Detailed review of yq dependency in hook scripts
6. Created `docs/specs/skills/` with per-skill SPEC files and `SPEC-hooks.md` as the design reference

Key decisions from conversation:
- User-configured stage hooks (shk2) rejected — "simply too unreliable." User enforcement stays in `project/*` files
- Language-specific templates (rwt1, qg80, 4vj0) rejected — fab-kit stays language-neutral
- PostToolUse (Write/Edit) is the only new hook event to add — all others assessed and rejected
- PreCompact dropped — change folder artifacts survive compaction
- SessionEnd dropped — thin value, nowhere useful to log
- SessionStart enhancement dropped — skills already handle context loading
- `fab status advance` stays in skills — must happen after SRAD questions, not on artifact write
- Existing hooks should migrate from `yq` to `fab runtime` subcommands
- Hooks are a **reliability layer**, not a replacement — skills keep bookkeeping instructions for agent-agnostic portability (Claude Code hooks are not available on Codex, Gemini CLI, Cursor, etc.)

## Why

Fab-kit skills contain scattered bookkeeping instructions that the agent must remember to execute after generating artifacts. These are fragile — the agent may forget, skip under context pressure, or execute them in the wrong order:

- "After generating intake, run `fab score --stage intake`" (fab-new Step 7)
- "After generating intake, run `fab status set-change-type`" (fab-new Step 6)
- "After generating spec, run `fab score`" (fab-continue, fab-ff, fab-fff)
- "After generating tasks/checklist, run `fab status set-checklist`" (fab-continue, fab-ff, fab-fff)
- "After editing spec, run `fab score`" (fab-clarify)

When the agent misses these, `.status.yaml` desyncs from reality — confidence reads 0.0, checklist shows 0/0, change type stays `feat` regardless of actual type.

Additionally, the existing hooks (`on-stop.sh`, `on-session-start.sh`) depend on `yq` as an external tool. If yq isn't installed, they silently do nothing (`command -v yq || exit 0`). The Go binary is always present and should handle all YAML operations.

## What Changes

### 1. New hook: `on-artifact-write.sh` (PostToolUse for Write and Edit)

A single hook script registered for both PostToolUse `Write` and PostToolUse `Edit` matchers. When the agent writes or edits a file, the hook checks if the path matches a fab artifact and triggers bookkeeping.

**Flow:**

```
PostToolUse fires (Write or Edit)
  → hook reads tool_input.file_path from stdin JSON
  → pattern-match against fab/changes/*/intake.md|spec.md|tasks.md|checklist.md
  → if no match: exit 0 (fast path, no-op)
  → if match: derive change name from path, run bookkeeping
```

**Bookkeeping per artifact:**

| Artifact | Commands run by hook |
|----------|---------------------|
| `intake.md` | `fab status set-change-type <change> <inferred-type>` (keyword scan on content) + `fab score --stage intake <change>` |
| `spec.md` | `fab score <change>` |
| `tasks.md` | `fab status set-checklist <change> total <N>` (count `- [ ]` lines in content) |
| `checklist.md` | `fab status set-checklist <change> generated true` + `fab status set-checklist <change> total <N>` + `fab status set-checklist <change> completed 0` |

**What the hook does NOT do** (stays in skills):
- `fab status advance` — must happen after SRAD questions in fab-new, not on write
- `fab status finish/start/fail/reset` — stage transitions are agent decisions
- `fab log command` — skill name can't be detected from hook events

**Hook properties:**
- Receives JSON on stdin with `tool_input.file_path` and `tool_input.content`
- For Edit events: uses `file_path` only (reads file from disk for content if needed)
- Derives change name from path: extract folder between `fab/changes/` and `/artifact.md`
- Returns `additionalContext` in stdout JSON to inform agent (e.g., "Bookkeeping: score 4.2/5.0, type: refactor")
- Exits 0 always — bookkeeping failures must not interrupt the agent
- Uses `fab` CLI exclusively — no `yq` dependency
- **All commands are idempotent** — running `fab score` or `fab status set-checklist` twice produces identical results. This means the hook and the skill can both run the same command without conflict

**Agent-agnostic portability:**

Claude Code's hook system is unique — no equivalent exists in Codex, Gemini CLI, Cursor, or other agent platforms. The hook is a **reliability layer** that catches bookkeeping the agent forgets, not a replacement for skill instructions. Skills KEEP their bookkeeping instructions so that:

- **Claude Code users** get hook-backed bookkeeping (automatic) + skill-instructed bookkeeping (agent-directed). Doubling up is harmless due to idempotency.
- **Non-Claude-Code agents** get skill-instructed bookkeeping only (current behavior, works most of the time).
- **No agent is required to have hooks** for the workflow to function correctly.

### 2. New Go subcommands: `fab runtime`

Absorb the yq-based YAML operations from existing hooks into the Go binary:

| Command | Purpose | Replaces |
|---------|---------|----------|
| `fab runtime set-idle <change>` | Write `agent.idle_since` timestamp to `.fab-runtime.yaml` | `yq -i ".\"$change_folder\".agent.idle_since = $(date +%s)"` in on-stop.sh |
| `fab runtime clear-idle <change>` | Delete `agent` block from `.fab-runtime.yaml` | `yq -i "del(.\"$change_folder\".agent)"` in on-session-start.sh |

These operate on `.fab-runtime.yaml` at the repo root (per change 1lwf). The Go binary already knows how to find the repo root and resolve changes. ~10 lines of Go each.

### 3. Update existing hooks to use `fab runtime`

**on-stop.sh** (before → after):
```bash
# Before: 31 lines, depends on yq
command -v yq >/dev/null 2>&1 || exit 0
runtime_file="$repo_root/.fab-runtime.yaml"
[ -f "$runtime_file" ] || echo '{}' > "$runtime_file"
yq -i ".\"$change_folder\".agent.idle_since = $(date +%s)" "$runtime_file"

# After: ~15 lines, no yq
"$fab_cmd" runtime set-idle "$change_folder" 2>/dev/null || true
```

**on-session-start.sh** (same pattern):
```bash
# Before:
yq -i "del(.\"$change_folder\".agent)" "$runtime_file"

# After:
"$fab_cmd" runtime clear-idle "$change_folder" 2>/dev/null || true
```

### 4. Update hook sync script

`fab/.kit/sync/5-sync-hooks.sh` needs to register PostToolUse hooks with **matchers**. Current script creates matcher-less entries. New registration output:

```json
{
  "PostToolUse": [
    {"matcher": "Write", "hooks": [{"type": "command", "command": "bash fab/.kit/hooks/on-artifact-write.sh"}]},
    {"matcher": "Edit", "hooks": [{"type": "command", "command": "bash fab/.kit/hooks/on-artifact-write.sh"}]}
  ],
  "Stop": [
    {"matcher": "", "hooks": [{"type": "command", "command": "bash fab/.kit/hooks/on-stop.sh"}]}
  ],
  "SessionStart": [
    {"matcher": "", "hooks": [{"type": "command", "command": "bash fab/.kit/hooks/on-session-start.sh"}]}
  ]
}
```

The `map_event()` function needs extending to return event+matcher pairs. One approach: a mapping table in the script instead of a case statement, or a naming convention like `on-posttooluse-write-artifact-write.sh`.

### 5. Skills: no changes (hooks are additive)

Skills **keep** their existing bookkeeping instructions unchanged. The PostToolUse hook supplements them as a reliability layer — it catches what the agent forgets. Since all bookkeeping commands are idempotent, the hook and the skill can both run the same command without conflict.

This preserves portability: non-Claude-Code agents (Codex, Gemini CLI, etc.) continue to work with skill-instructed bookkeeping only.

### 6. Delete stale changes

Delete these change folders (not archive — they were never implemented, just intakes):

| Change | Reason |
|---|---|
| ~~`260305-shk2-pipeline-stage-hooks`~~ | Already deleted |
| `260305-4vj0-react-template` | Language-specific templates rejected — fab-kit stays neutral |
| `260305-qg80-node-typescript-template` | Same |
| `260305-rwt1-rust-project-template` | Same |

### 7. Update Constitution §I

```markdown
# Before
All workflow logic MUST live in markdown skill files and shell scripts.

# After
All workflow logic MUST live in markdown skill files and scripts.
```

### 8. Remove language detection from fab-setup.md

Remove Phase 1b-lang (language detection and template application) from the bootstrap flow. Detection logic was added by the rejected template changes. The templates directory (`fab/.kit/templates/constitutions/`, `fab/.kit/templates/configs/`) can be cleaned up or left for future use.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document new hooks, updated hook sync, hook-to-event mapping, `fab runtime` subcommands
- `fab-workflow/schemas`: (modify) No longer needs `compact_context` — just document `fab runtime` commands for `.fab-runtime.yaml`
- `fab-workflow/planning-skills`: (modify) Note that bookkeeping is now hook-backed (reliability layer, not replacement)
- `fab-workflow/execution-skills`: (modify) Note hook-backed bookkeeping for review checklist updates
- `fab-workflow/setup`: (modify) Remove language detection section

## Impact

- **New hook script**: 1 file — `fab/.kit/hooks/on-artifact-write.sh` (~60 lines)
- **New Go code**: `fab runtime set-idle` + `fab runtime clear-idle` (~20 lines total)
- **Modified hooks**: `on-stop.sh`, `on-session-start.sh` — replace yq with `fab runtime`
- **Modified sync**: `fab/.kit/sync/5-sync-hooks.sh` — support matchers
- **No skill changes**: Skills keep bookkeeping instructions for agent-agnostic portability; hooks are additive
- **Modified constitution**: 1 word change
- **Modified fab-setup.md**: Remove language detection phase
- **Deleted changes**: 3 change folders (shk2 already deleted)
- **Dependency**: Requires 1lwf (`.fab-runtime.yaml`) to be implemented first

## Open Questions

None — all design decisions resolved in preceding discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Hooks are for kit-internal mechanics only — user enforcement stays in project/* files | Discussed — user explicitly rejected user-configured hooks as "too unreliable" | S:95 R:85 A:95 D:95 |
| 2 | Certain | PostToolUse (Write/Edit) is the only new hook event | All 18 events assessed per-skill. PreCompact, SessionEnd, SessionStart enhancement, SubagentStop all rejected with reasons | S:95 R:85 A:95 D:95 |
| 3 | Certain | Delete shk2, 4vj0, qg80, rwt1 (not archive — never implemented) | Discussed — user agreed. shk2 already deleted | S:95 R:90 A:95 D:95 |
| 4 | Certain | Constitution §I changes "shell scripts" → "scripts" | Discussed — user explicitly requested | S:95 R:90 A:95 D:95 |
| 5 | Certain | Stage transitions remain agent-directed | Discussed — advance/finish/fail/reset are intentional decisions | S:90 R:85 A:90 D:90 |
| 6 | Certain | `fab status advance` stays in fab-new (not hookable) | Discussed — must happen after SRAD questions (Step 8), not on intake write | S:90 R:80 A:90 D:90 |
| 7 | Certain | Existing hooks migrate from yq to `fab runtime` subcommands | Discussed — eliminates silent-fail yq dependency, consistent with all-through-fab-CLI pattern | S:90 R:85 A:90 D:90 |
| 8 | Certain | Single script `on-artifact-write.sh` handles both Write and Edit matchers | Same logic — detect path, run bookkeeping. No reason for separate scripts | S:85 R:90 A:85 D:90 |
| 9 | Certain | Hooks are a reliability layer, not a replacement — skills keep bookkeeping instructions | Discussed — Claude Code hooks are platform-specific (no equivalent in Codex, Gemini CLI, Cursor). Constitution §I requires agent-agnostic portability. All bookkeeping commands are idempotent so doubling up is harmless | S:95 R:90 A:95 D:90 |
| 10 | Confident | Hook returns `additionalContext` to inform agent what was auto-handled | PostToolUse hooks can return JSON with `additionalContext` field. Agent sees it but doesn't need to act on it | S:80 R:90 A:80 D:85 |
| 11 | Confident | Pipeline orchestrator keeps yq (~40 uses) — separate concern | Absorbing manifest parsing into `fab pipeline` is a much larger change. Pipeline already lists yq as a prerequisite | S:80 R:85 A:85 D:80 |
| 12 | Confident | Remove language detection from fab-setup.md | Template changes are rejected. Detection logic has no purpose without templates to apply | S:80 R:85 A:80 D:80 |

12 assumptions (9 certain, 3 confident, 0 tentative, 0 unresolved).
