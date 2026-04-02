# Superpowers Comparison

Comparison of the Fab workflow with [Superpowers](https://github.com/obra/superpowers) (by Jesse Vincent) — an agentic skills framework and software development methodology for AI coding agents. Both emerged from the same insight: constrain the agent with process, not cleverness.

## Common ground

Both are "pure prompt play" — markdown skill files + shell scripts, no runtime frameworks. Both enforce design-before-code, use git worktrees for isolation, decompose work into atomic tasks, and use subagents for parallel execution with review gates.

## Key differences

| Dimension | Fab | Superpowers |
|-----------|-----|-------------|
| **Pipeline model** | 6 explicit stages (intake → spec → tasks → apply → review → hydrate) with YAML state machine tracking progress | 7 phases but loosely coupled — skills invoke each other by convention, no formal state tracking |
| **Autonomy framework** | SRAD scoring (Signal, Reversibility, Agent Competence, Disambiguation) with numeric confidence gates that block fast-forward if decisions are under-resolved | No formal autonomy scoring — relies on human approval checkpoints (brainstorming approval, plan approval) |
| **Memory / knowledge management** | First-class `docs/memory/` system — hydrate stage writes post-implementation truth back into memory files, creating institutional knowledge | No equivalent — knowledge lives in the codebase and git history only |
| **Specs as separate artifact** | Distinct pre-implementation specs (`docs/specs/`) vs post-implementation memory — the gap between intent and reality is explicitly tracked | Design document produced during brainstorming, but no persistent spec layer separate from code |
| **TDD enforcement** | No dedicated TDD skill — tests are part of apply/review but not a rigid RED-GREEN-REFACTOR cycle | Core differentiator — strict TDD with a dedicated skill that will *delete code written before tests* |
| **Multi-agent coordination** | Operator system (`/fab-operator`) with dependency-aware spawning, tmux pane routing, autopilot queues, proactive monitoring | Simpler model — fresh subagent per task, parallel dispatch for independent tasks, two-stage review |
| **Portability** | Self-contained `fab/.kit/` — `cp -r` into any project | Platform shims for Claude Code, Cursor, Codex, Gemini CLI — broader agent compatibility |
| **Assumption tracking** | Explicit assumption tables with grades (Certain/Confident/Tentative/Unresolved) persisted in artifacts, scannable by `/fab-clarify` | Implicit — assumptions surface during brainstorming discussion but aren't formally tracked |
| **Change lifecycle** | Full lifecycle: backlog → change → archive, with status tracking, checklist scoring, and PR integration | Per-branch lifecycle: worktree → implement → merge/discard |

## What fab can learn from Superpowers

### TDD as a first-class skill

Superpowers' strict RED-GREEN-REFACTOR enforcement is its sharpest edge. Fab's apply stage could benefit from an explicit TDD mode — write failing test, implement, verify green — rather than leaving test strategy to the `code-quality.md` policy file.

### The "junior engineer" planning standard

Superpowers' plans are described as "clear enough for an enthusiastic junior engineer with poor taste and no project context." This is a useful litmus test for task granularity — fab's `tasks.md` could adopt a similar explicitness bar (exact file paths, complete code specifications, not pseudo-code).

### Fresh-agent-per-task isolation

Superpowers dispatches a clean subagent for each task specifically to prevent context drift. Fab's operator system is more sophisticated (dependency-aware, monitoring), but the "clean slate per task" principle is worth preserving — accumulated context can cause subtle regressions.

### Platform portability

Superpowers works across 5+ agent platforms. Fab is currently Claude Code-specific. If portability matters, the skill invocation layer could be abstracted (though this trades simplicity for reach).

### The 1% Rule meta-skill

Superpowers' `using-superpowers` skill hooks into session start and forces the agent to always check for relevant skills before acting. Fab's preamble serves a similar role but only activates when a `/fab-*` command is invoked — there's no ambient "always check fab" behavior.

## Where fab is ahead

- **SRAD** gives fab a principled, numeric answer to "should I ask or assume?" — Superpowers relies entirely on human checkpoints
- **Memory hydration** creates durable institutional knowledge that compounds across changes
- **Confidence gating** prevents fast-forward pipelines from running with under-resolved decisions
- **Operator coordination** handles true multi-agent parallelism with dependency graphs, not just "dispatch N independent tasks"
- **Assumption tracking** makes the gap between "what we decided" and "what we guessed" visible and auditable
