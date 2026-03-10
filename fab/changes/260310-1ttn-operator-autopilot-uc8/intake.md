# Intake: Operator Autopilot UC8

**Change**: 260310-1ttn-operator-autopilot-uc8
**Created**: 2026-03-10
**Status**: Draft

## Origin

> User asked for a way to execute multiple changes one after another. After discussing options (fab-operator extension, standalone `/fab-batch` skill, shell script), the user chose to extend `fab-operator1` with a new UC8. The design was collaboratively refined through several decision points: multi-worktree model (each change gets its own worktree + agent pane), full autopilot with merge after each success, all three ordering strategies, and Approach B (UC8 stub in the use cases list with a detailed "Autopilot Behavior" section).

## Why

1. **Problem**: Running multiple changes through the fab pipeline currently requires manual repetition — `/fab-switch` + `/fab-ff` for each change, one at a time. For a queue of 3-5 changes, this is tedious and error-prone.
2. **Consequence**: Users either process changes one at a time (slow) or try to manually coordinate parallel agents without a coordination layer (risky).
3. **Approach**: Extend `fab-operator1` rather than creating a new skill, because the operator already has the primitives needed (pane-map, send-keys, status observation, pre-send validation, confirmation model) and its spec already contains a detailed UC7 "Sequential pipeline execution (autopilot)" design that was never ported to the skill file. This avoids a new entrypoint and builds on existing coordination infrastructure.

## What Changes

### UC8 Stub in Use Cases Section

Rename "Seven Use Cases" heading to "Use Cases". Add UC8 after the existing UC7 (Notification surface):

- UC8 accepts a list of changes (IDs, names, or "all idle")
- Resolves ordering via one of three strategies
- Confirms the full queue at start (destructive — merges PRs)
- Delegates to the Autopilot Behavior section for execution

### Autopilot Behavior Section (new top-level section)

A detailed behavior section (like how `/fab-continue` has "Apply Behavior" and "Review Behavior") containing:

**Ordering strategies:**
- **User-provided**: Run in the exact order given. `"run bh45, qkov, ab12"` → that order.
- **Confidence-based**: Sort by confidence score descending via `fab status show --all`. Highest confidence changes merge first (easy wins reduce rebase churn for harder ones).
- **Hybrid**: User provides ordering constraints (`"bh45 before qkov"`), operator sorts the rest by confidence. Constraints as partial order, confidence as tiebreaker.

**The autopilot loop** (per change):

```
For each change in resolved order:
  1. Spawn       → wt-create --non-interactive
  2. Open tab    → tmux new-window -n "fab-<id>" -c <worktree> "claude --dangerously-skip-permissions '/fab-switch <change>'"
  3. Gate check  → fab status show <change> for confidence
     - confidence >= gate → fab send-keys <change> "/fab-ff"
     - confidence < gate  → flag: "{change} confidence {score}, below {type} gate ({threshold}). Run /fab-fff or skip?"
  4. Monitor     → poll fab pane-map + fab runtime is-idle on each user interaction
     - Stage reaches hydrate/ship → change succeeded
     - Review fails after rework budget → flag and skip
     - Agent idle >15min at non-terminal stage → nudge once, then flag
     - Pane dies → flag and skip
  5. On success  → gh pr merge from operator shell (destructive — already confirmed at start)
  6. Rebase next → fab send-keys <next-change> "git fetch origin main && git rebase origin/main"
     - If conflict → flag to user, skip to next (never auto-resolve)
  7. Cleanup     → wt-delete (optional, after merge)
  8. Progress    → "bh45: merged. 1 of 3 complete. Starting qkov."
```

**Failure matrix:**

| Failure | Action | Resume? |
|---------|--------|---------|
| Confidence below gate | Flag to user: run `/fab-fff` or skip | Wait for user input |
| Review fails (rework exhausted) | Flag, skip to next change | Yes |
| Rebase conflict | Flag, skip to next change | Yes |
| Agent pane dies | 1 respawn attempt, then flag and skip | Yes |
| Stage timeout (>30 min same stage) | Flag regardless of retry state | Yes |
| Total timeout (>2 hr per change) | Flag for review | Yes |

**Interruptibility:**
- `"stop after current"` — finish active change, halt queue
- `"skip qkov"` — remove from queue, proceed to next
- `"pause"` — stop sending new commands, running agents continue
- `"resume"` — pick up from where paused

**Resumability:** If operator session restarts, reconstruct state from `fab pane-map` — merged changes show as archived/shipped, in-progress changes show their current stage. Resume from first non-completed change.

**Progress reporting:** After each change completes (success or skip), output a one-line status. Final summary lists all changes with outcome.

### Updates to Existing Sections

1. **Confirmation Model table** — add row for autopilot: destructive, confirm full queue at start, per-PR confirmation not required
2. **Rename** "Seven Use Cases" → "Use Cases"

### Spec Update

Per constitution constraint: changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-fab-operator1.md`. The spec already contains the UC7 autopilot design — the spec update is primarily to renumber (spec's UC7 → UC8 to match the skill's numbering where UC7 is Notification surface) and ensure alignment with the skill's final wording.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document operator autopilot UC8 behavior and its relationship to existing UC4 (spawn) and UC2 (sequenced actions)

## Impact

- **Skill file**: `fab/.kit/skills/fab-operator1.md` — primary change target
- **Spec file**: `docs/specs/skills/SPEC-fab-operator1.md` — alignment update
- **Deployed copy**: `.claude/skills/fab-operator1.md` — regenerated by `fab-sync.sh`
- **No CLI changes**: All primitives (`pane-map`, `send-keys`, `runtime`, `status show`) already exist
- **No template changes**: No new artifact templates needed
- **No migration needed**: This adds new behavior to an existing skill, no user data restructuring

## Open Questions

None — all resolved during clarification.

## Clarifications

### Session 2026-03-10

| # | Action | Detail |
|---|--------|--------|
| 8 | Confirmed | User chose conversation-only (option A) for v1 queue state |
| 9 | Resolved | Validated spawn pattern from `batch-fab-new-backlog.sh` lines 135-144 |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Extend fab-operator1 skill, not a new skill | Discussed — user explicitly chose operator extension over standalone `/fab-batch` | S:95 R:80 A:90 D:95 |
| 2 | Certain | Multi-worktree model (each change gets own worktree + pane) | Discussed — user chose option 1 (multi-worktree) over single-pane sequential | S:95 R:70 A:85 D:95 |
| 3 | Certain | Full autopilot with merge after each success | Discussed — user chose full autopilot over checkpoint-after-each | S:95 R:60 A:80 D:95 |
| 4 | Certain | All three ordering strategies (user-provided, confidence-based, hybrid) | Discussed — user explicitly requested all three | S:95 R:75 A:85 D:90 |
| 5 | Certain | Approach B: UC8 stub + separate Autopilot Behavior section | Discussed — user chose Approach B over inlining | S:95 R:85 A:90 D:95 |
| 6 | Confident | Renumber spec UC7 → UC8 to match skill numbering | Spec's UC7 (autopilot) maps to skill's UC8 since skill's UC7 is Notification surface; renumbering avoids confusion | S:70 R:90 A:80 D:75 |
| 7 | Confident | No new CLI primitives needed | All required commands (pane-map, send-keys, status show, runtime) already exist in the fab CLI | S:80 R:85 A:90 D:80 |
| 8 | Certain | Conversation context + pane-map re-derivation is sufficient for v1 queue state recovery | Clarified — user chose conversation-only (option A) for v1; file-backed queue is a clean upgrade path if context compression causes friction | S:95 R:60 A:55 D:50 |
| 9 | Certain | Worktree spawn uses `wt create --non-interactive` + `tmux new-window` + `claude --dangerously-skip-permissions` | Clarified — validated from `batch-fab-new-backlog.sh` lines 135-144 which implements this exact pattern | S:95 R:70 A:90 D:90 |

9 assumptions (7 certain, 2 confident, 0 tentative, 0 unresolved).
