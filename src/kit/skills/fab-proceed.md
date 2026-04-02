---
name: fab-proceed
description: "Context-aware orchestrator — detects state, runs prefix steps (fab-new, fab-switch, git-branch), then delegates to fab-fff."
---

# /fab-proceed

Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

> `/fab-proceed` follows `_preamble.md` conventions but skips preflight/context loading itself — it delegates all pipeline context loading to `/fab-fff`.

---

## Purpose

Detect the current pipeline state and automatically run whatever prefix steps are needed (fab-new, fab-switch, git-branch) before handing off to `/fab-fff` for the full pipeline. Zero-argument, zero-flag — the skill infers everything from context. Idempotent — re-running detects completed steps and skips them.

---

## Arguments

None. `/fab-proceed` does not accept arguments or flags. Any arguments passed are silently ignored.

---

## State Detection

Detect the current state by executing the following checks in order. The skill MUST NOT prompt the user for input at any detection step — it either resolves automatically or errors.

### Step 1: Active Change Check

```bash
fab resolve --folder 2>/dev/null
```

If exits 0, an active change exists. Capture the folder name.

### Step 2: Branch Check

If an active change was found, compare the current git branch with the resolved change folder name:

```bash
git branch --show-current
```

If the current branch matches the change folder name, the branch is already set up.

### Step 3: Unactivated Intake Check

If no active change was found in Step 1, scan for unactivated intakes:

```bash
ls -d fab/changes/*/intake.md 2>/dev/null | grep -v archive/ | sed 's|fab/changes/||;s|/intake.md||' | sort -t- -k1,1r | head -1
```

This pipeline lists change folders with intakes, excludes archived changes, extracts folder names, and sorts by `YYMMDD` date prefix in descending order to select the most recent.

- If exactly one non-archived change folder exists, use it.
- If multiple exist, the sort selects the most recently created by folder date prefix (`YYMMDD` — higher date wins; on tie, lexicographic order breaks it deterministically).
- If none exist, proceed to Step 4.

### Step 4: Conversation Context Check

If no intake exists anywhere, evaluate whether the prior conversation contains substantive discussion. An empty conversation, a greeting-only conversation, or a conversation with no technical content SHALL be treated as "no context."

Substantive context means the conversation contains at least one of:
- Technical requirements or feature descriptions
- Design decisions or tradeoffs
- Specific values, constraints, or API shapes
- Problem statements with enough detail to generate an intake

### Dispatch Table

| Detected state | Steps to run |
|----------------|--------------|
| Active change + matching branch | `/fab-fff` only |
| Active change + no matching branch | `/git-branch` → `/fab-fff` |
| Unactivated intake (no active change) | `/fab-switch` → `/git-branch` → `/fab-fff` |
| Conversation context (no intake) | `/fab-new` → `/fab-switch` → `/git-branch` → `/fab-fff` |
| No context, no intake | Error — stop |

---

## Dispatch Behavior

### Subagent Dispatch (Prefix Steps)

Each prefix step (fab-new, fab-switch, git-branch) SHALL be dispatched as a subagent using the Agent tool (`subagent_type: "general-purpose"`) per `_preamble.md` § Subagent Dispatch. Each subagent prompt MUST include the standard subagent context files:

**Required** (subagent reports error if missing):
- `fab/project/config.yaml`
- `fab/project/constitution.md`

**Optional** (skip gracefully if missing):
- `fab/project/context.md`
- `fab/project/code-quality.md`
- `fab/project/code-review.md`

#### fab-new Dispatch

When conversation context exists but no intake:

1. Synthesize a description from the conversation (see Conversation Context Synthesis below)
2. Dispatch subagent: read `.claude/skills/fab-new.md`, invoke `/fab-new` with the synthesized description
3. Capture the created change folder name from the subagent result

#### fab-switch Dispatch

When an unactivated intake exists (or fab-new just created one):

1. Dispatch subagent: read `.claude/skills/fab-switch.md`, invoke `fab change switch "<change-name>"`
2. Capture the switch confirmation from the subagent result

#### git-branch Dispatch

When an active change exists but no matching branch:

1. Dispatch subagent: read `.claude/skills/git-branch.md`, follow its behavior for the active change
2. Capture the branch creation/checkout result from the subagent result

### Conversation Context Synthesis

When `/fab-proceed` needs to create an intake (no existing intake found), it SHALL synthesize a description from the conversation by extracting:

- **Decisions made** — specific choices with rationale
- **Alternatives rejected** — options considered and why they were ruled out
- **Constraints identified** — boundaries or requirements surfaced
- **Specific values agreed upon** — config structures, API shapes, exact behaviors

The synthesized description MUST be substantive enough for `/fab-new` to generate a complete intake without prompting. Do not fabricate details — capture what was said. If the conversation was minimal (e.g., "we should add retry logic"), capture that as-is without adding specifics.

### fab-fff Terminal Delegation

The final `/fab-fff` invocation is NOT dispatched as a subagent — it is invoked via the Skill tool in the current context. This ensures `/fab-fff` runs in the main context with full user visibility of its output, confidence gates, and pipeline progress.

The skill SHALL NOT pass `--force` or any other flags to `/fab-fff`. If `/fab-fff` fails a confidence gate, it stops normally and the user intervenes.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| No context and no intake | Output: `Nothing to proceed with — start a discussion or run /fab-new first.` Stop. |
| fab-new subagent fails | Surface the error from fab-new and stop. Do not proceed to further steps. |
| fab-switch subagent fails | Surface the error from fab-switch and stop. |
| git-branch subagent fails | Surface the error from git-branch and stop. |
| fab-fff gate failure | `/fab-fff` stops normally with its own gate failure message. `/fab-proceed` does not retry or bypass the gate. |

Errors from any sub-skill propagate to the user and halt execution. The skill does not retry failed steps.

---

## Output

```
/fab-proceed — detecting state...

{Step reports, one per line — only for steps actually executed}

Handing off to /fab-fff...
{fab-fff takes over and produces its own output}
```

Step report format (only for steps actually executed):
- `Created intake: {change-name}` (when fab-new ran)
- `Activated: {change-name}` (when fab-switch ran)
- `Branch: {branch-name} ({action})` (when git-branch ran; action = created / checked out / already active)

When only `/fab-fff` is needed (active change + matching branch), output shows only the detecting state line and the handoff line before `/fab-fff` output.

---

## Key Properties

| Property | Value |
|----------|-------|
| Arguments | None |
| Flags | None |
| Requires active change? | No — can create one from conversation context |
| Runs preflight? | No — delegates to `/fab-fff` |
| Read-only? | No — may create change, switch pointer, create branch |
| Idempotent? | Yes — re-running detects completed steps and skips them |
| Advances stage? | No directly — `/fab-fff` handles stage advancement |
| Outputs Next line? | Inherits from `/fab-fff` |
