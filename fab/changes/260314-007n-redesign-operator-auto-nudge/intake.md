# Intake: Create fab-operator4 — Redesigned Auto-Nudge Operator

**Change**: 260314-007n-redesign-operator-auto-nudge
**Created**: 2026-03-14
**Status**: Draft

## Origin

> Create fab-operator4, a new operator skill that extends operator3 with improvements from a multi-agent review session.

Conversational — extensive prior discussion via 5 parallel review agents (safety, heuristic robustness, architecture, spec-skill consistency, optimization) followed by user design decisions on each finding.

## Why

1. **operator3's two-tier confidence model is unnecessarily conservative.** The escalation tier (withhold answers for "judgment calls") stalls the pipeline for questions that are safe to answer, because all agent work happens in worktrees and PRs require human merge. The safety gate is at the PR level, not the operator level.
2. **The terminal heuristic has known false-positive and false-negative gaps.** Claude's own conversational output ("Would you like me to run the tests?") triggers detection even though the agent isn't blocked. The `?` pattern fires on code comments, log output, and compiler warnings. Several common prompt formats (`:` endings, numbered menus, `Press any key`) are missed entirely.
3. **No audit trail.** When auto-answers go wrong, there's no reconstruction path — the user doesn't know what was answered.
4. **Race condition between detection and send.** The agent can transition states between the idle check and the `tmux send-keys`, injecting keystrokes into a different context.
5. **The skill is ~26% larger than necessary** due to redundant inherited-step listings, oversized key properties table, and standalone sections for single-sentence policies.
6. **No routing discipline.** Nothing prevents the operator from executing user instructions directly instead of routing them to the appropriate agent.

## What Changes

### New skill: `fab/.kit/skills/fab-operator4.md`

Extends operator3 (which extends operator2). Inherits all operator3 behavior, then overrides or adds the following:

### Routing Discipline (new section)

The operator MUST NOT execute user instructions directly. When the user gives an instruction (e.g., "fix the tests", "add error handling"), the operator:

1. Determines which running tmux session/agent the request corresponds to
2. Routes the instruction to that agent via `tmux send-keys`
3. Enrolls the agent in monitoring

The operator's job is coordination, not implementation.

### Autopilot Pipeline Skill (override)

Operator4 uses `/fab-fff` instead of `/fab-ff` for autopilot gate checks. An upcoming change will make `/fab-fff` confidence-gated, aligning it with the operator's needs. The inherited autopilot behavior (from operator2) is unchanged except for this skill substitution.

### Simplified Answer Model (replaces operator3's two-tier confidence model)

Remove the auto-answer/escalate classification entirely. All questions are auto-answered. The only escalation case is when the operator literally cannot determine what keystrokes to send.

What to send (decision list replaces prose heuristic):

1. Binary yes/no or confirmation prompt → `y`
2. `[Y/n]` or `[y/N]` prompt → `y`
3. Claude Code permission/approval prompt → `y`
4. Numbered menu or multi-choice → `1` (first/default option)
5. Open-ended question where a concrete answer is determinable from visible terminal context → send that answer
6. Question where the operator cannot determine what keystrokes to send → escalate: `"{change}: can't determine answer for '{summary}'. Please respond."`

### Improved Question Detection

**Capture window**: Increase from `-l 10` to `-l 20` — compensates for line wrapping and verbose preambles that push prompts off the 10-line window.

**Claude turn boundary guard** (new): If a Claude Code `>` prompt cursor (`^\s*>\s*$`) appears in the last 2 lines of capture, skip detection. The agent is at a normal human-turn boundary, not a blocking prompt. This prevents false positives from Claude's own conversational output.

**Tightened `?` pattern**: Match only on the **last non-empty line** (not any of the 20 lines). Must be <120 chars. Skip lines starting with `#`, `//`, `*`, `>`, or timestamp patterns (these are comments, log output, or search results).

**New question indicator patterns** (additions to operator3's list):
- Lines ending with `:` or `:\s*$` (CLI input prompts)
- Enumerated options (`[1-9]\)` patterns)
- `Press.*key`, `press.*enter`, `hit.*enter` (case-insensitive)

**Blank capture guard** (new): If captured output is entirely blank or whitespace, skip question detection for this tick (treat as "cannot determine", not "no question").

### Re-Capture Before Send (new guard)

Before sending an auto-answer via `tmux send-keys`, re-capture the terminal (`tmux capture-pane -t <pane> -p -l 20`). If output changed since the initial capture, abort — the agent is no longer waiting. This eliminates the race condition between idle check and send.

### Per-Answer Logging (new requirement)

Every auto-answer MUST be reported inline in the operator's terminal output: `"{change}: auto-answered '{summary}' → {answer}"`. This is the reconstruction path for debugging. No cooldown or retry limit — each question is evaluated independently.

### Optimized Skill Structure

Apply token-efficiency improvements (~26% reduction from operator3):
- **Delta-only monitoring tick**: Describe only the two modifications to operator2's tick, not the full 6-step listing
- **Reduced Key Properties table**: Only 4 rows (novel/modified properties), not 11
- **Condensed Purpose section**: Drop redundant "Key difference" paragraph and inline launcher info
- **Integrated guards**: Idle-only guard and bottom-most indicator rule are integrated into the detection steps, not separate paragraphs

### New spec: `docs/specs/skills/SPEC-fab-operator4.md`

Create a matching spec file with: summary, primitives table (with `-l 20` flags), routing discipline, question detection (with all guards as named subsections: Claude Turn Boundary Guard, Blank Capture Guard, Idle-Only Guard), answer model, re-capture before send, logging, monitoring tick changes, relationship to operator3 table, launcher details, one-operator-at-a-time note, key properties, and resolved design decisions.

### Launcher script: `fab/.kit/scripts/fab-operator4.sh`

Create launcher script following the same pattern as `fab-operator3.sh` — creates a singleton tmux tab named `operator` and invokes `/fab-operator4` in a new Claude session.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Add operator4 documentation — routing discipline, simplified answer model, improved detection, re-capture guard, per-answer logging

## Impact

- **New files**: `fab/.kit/skills/fab-operator4.md`, `docs/specs/skills/SPEC-fab-operator4.md`, `fab/.kit/scripts/fab-operator4.sh`
- **Modified files**: `docs/memory/fab-workflow/execution-skills.md` (add operator4 section)
- **No changes to operator1, operator2, or operator3** — operator4 is purely additive
- **Skill registry**: The new skill will be discovered automatically by the skill system via frontmatter

## Open Questions

None — all design decisions were resolved during the discussion session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Auto-answer all questions, no escalation tier | Discussed — user decided: worktree isolation + human PR merge is the safety gate, not the operator withholding answers | S:95 R:85 A:90 D:95 |
| 2 | Certain | Add routing discipline section | Discussed — user explicitly requested: operator must route instructions to agents, never execute directly | S:95 R:90 A:90 D:95 |
| 3 | Certain | Add per-answer inline logging | Discussed — user agreed: every auto-answer must be reported for debugging reconstruction | S:90 R:90 A:85 D:95 |
| 4 | Certain | Add Claude turn boundary guard (`>` prompt check) | Discussed — user agreed: prevents false positives from Claude's conversational output | S:90 R:90 A:85 D:90 |
| 5 | Certain | Add re-capture before send guard | Discussed — user agreed (re-capture yes, single-tick grace period no) | S:90 R:85 A:85 D:90 |
| 6 | Certain | Increase capture window to `-l 20` | Discussed — user agreed: compensates for wrapping and verbose preambles | S:85 R:90 A:85 D:90 |
| 7 | Certain | Tighten `?` to last non-empty line only, <120 chars, skip comment prefixes | Discussed — user agreed: reduces false positives from log/comment/search output | S:90 R:90 A:85 D:90 |
| 8 | Certain | Add `:` endings, enumerated options, `Press.*key` patterns | Discussed — user agreed: covers common prompt formats that operator3 misses | S:85 R:90 A:80 D:85 |
| 9 | Certain | Apply all optimization suggestions (~26% token reduction) | Discussed — user said "great! optimize pls" | S:90 R:90 A:90 D:95 |
| 10 | Certain | Keep all operators independent (no base extraction, no folding) | Discussed — user explicitly rejected merging operators or extracting shared base | S:95 R:85 A:85 D:95 |
| 11 | Certain | Create as operator4 (new skill), not modify operator3 | Discussed — user explicitly said "instead of making any changes to operator3, I want to do all this directly in operator4" | S:95 R:90 A:90 D:95 |
| 12 | Confident | Launcher script follows operator3.sh pattern | Strong precedent from operator1/2/3 launcher scripts — same singleton tab pattern | S:75 R:90 A:85 D:90 |
| 13 | Confident | Numbered decision list for "what to send" (items 1-6) | Discussed as "decision tree" replacement for prose heuristic — user agreed (#8 ok). Exact ordering and items derived from discussion | S:80 R:85 A:80 D:80 |
| 14 | Certain | Autopilot uses `/fab-fff` instead of `/fab-ff` | Discussed — user decided: fff will become confidence-gated in an upcoming change, making it the right fit for operator autopilot | S:95 R:80 A:85 D:95 |

14 assumptions (12 certain, 2 confident, 0 tentative, 0 unresolved).
