# fab-operator4

## Summary

Standalone multi-agent coordination layer with proactive monitoring and auto-nudge. Runs in a dedicated tmux pane, observes all running fab agents via `fab pane-map`, routes commands via `tmux send-keys`, monitors progress via `/loop`, auto-answers routine agent questions, and drives autopilot queues through the full pipeline.

Self-contained — does not inherit from any other operator skill. All behavior is defined in `src/kit/skills/fab-operator4.md` plus the standard `_` files loaded via `_preamble.md`. External tool reference (`_cli-external.md`) is loaded in the operator's own startup section.

Not a lifecycle enforcer — the operator coordinates across agents and proxies routine user input, not advancing stages or making pipeline decisions.

**Helpers**: Declares `helpers: [_cli-fab, _cli-external]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

---

## Section Structure

The skill is organized into 9 sections:

1. **Principles** — identity (coordinate don't execute), routing discipline, context discipline (never loads change artifacts), state re-derivation (why: stale state = wrong actions)
2. **Startup** — always-load context layer, `_cli-external.md` load, orientation (pane map + ready signal), outside-tmux degradation
3. **Safety Model** — confirmation tiers (read-only / recoverable / destructive), pre-send validation (pane exists + agent idle), bounded retries & escalation table
4. **Monitoring System** — monitored set (fields, enrollment triggers, removal triggers), `/loop` lifecycle (start/extend/stop, one-loop invariant), monitoring tick with 6 steps. Window-name rename on enrollment: prefix `»` to the tmux window name (idempotent — skipped if already prefixed). Removal does not restore the original name.
5. **Auto-Nudge** — question detection (capture -S -20, guards, pattern matching), answer model (simplified decision list items 1-6), re-capture before send, per-answer logging
6. **Modes of Operation** — shared rhythm + compact table: broadcast, sequenced rebase, merge PRs, spawn agent, status dashboard, unstick agent, notification, rebase all, autopilot
7. **Autopilot** — queue ordering (user-provided / confidence-based / hybrid), per-change loop, failure matrix, interruptibility, resumability. Pipeline uses `/fab-fff`
8. **Configuration** — monitoring interval (5m), stuck threshold (15m), autopilot tick (2m), session-scoped
9. **Key Properties** — standard properties table

---

## Primitives

All tool references are in shared `_` files — operator4 does not duplicate tool tables.

| Primitive | Reference |
|-----------|-----------|
| `fab pane-map`, `fab resolve`, `fab change list`, `fab status`, `fab score` | `_cli-fab.md` |
| `wt list`, `wt create`, `wt delete`, `tmux` commands, `/loop` | `_cli-external.md` |
| Change folder, branch, worktree naming | `_preamble.md` § Naming Conventions |

---

## Monitoring Tick

All 6 steps are fully specified inline:

1. Stage advance detection
2. Pipeline completion detection
3. Review failure detection
4. Pane death detection
5. Auto-nudge (input-waiting detection + answer model) — includes sending `/git-branch` after detecting new change creation from backlog
6. Stuck detection (excludes input-waiting agents)

---

## Auto-Nudge

### Question Detection

- Capture window: `tmux capture-pane -t <pane> -p -S -20`
- Guards: Claude turn boundary (`>` cursor in last 2 lines), blank capture, idle-only
- Pattern matching: `?` on last non-empty line <120 chars with comment/log exclusions, plus inherited patterns (Y/n, approval, phrasing) and new patterns (`:` endings, enumerated options, `Press.*key`)
- Bottom-most indicator rule

### Answer Model

Decision list (all auto-answer except undeterminable or strategic):

1. Binary yes/no -> `y`
2. `[Y/n]`/`[y/N]` -> `y`
3. Claude Code permission -> `y`
4. Numbered menu -> classify then act:
   - **Routine** (tool/permission prompts, binary-framed menus, synonymous-option menus) -> `1`
   - **Strategic** (multi-option menus where options represent materially different directions — scope, PR split, pipeline shape, commit organization, spec/approach decisions) -> escalate to user
   - Classification uses LLM judgment over the terminal capture, weighing: option text length, semantic distinctness of options, surrounding agent context, and reversibility of the choice. No hardcoded keyword list. No agent-side sentinel/marker protocol.
   - On classification uncertainty, treat as Strategic and escalate. False-negative strategic commits the queue to an unchosen direction; false-positive strategic costs at most a user nudge, recovered by the 30m idle auto-default below.
5. Determinable from context -> send answer
6. Cannot determine keystrokes -> escalate

### Idle Auto-Default on Strategic Escalations

When rule 4 escalates as Strategic, the operator runs a per-prompt idle timer. If the prompt stays idle for 30 minutes, the operator auto-answers and logs with a distinct `auto-defaulted` format.

- **Threshold**: 30 minutes, hardcoded. No `.fab-operator.yaml` field, no per-change override, no environment variable. `.fab-operator.yaml` schema is unchanged.
- **Idle clock reset**: timer resets on any terminal-state change in the pane (new content appended by the agent, user keystrokes that alter the prompt display, prompt redraw). The timer watches pane-idle-ness, not escalation-open-ness.
- **Answer selection priority**: (1) if the prompt visibly states a default (e.g., `(default: 2)`, `Press enter for 2`, `[2]`), send that default; (2) otherwise, send `1`.
- **Scope exclusion**: applies ONLY to rule 4 Strategic escalations. Rule 6 ("cannot determine keystrokes") escalations MUST NOT trigger idle auto-default — sending `1` would emit nonsense into the pane. Rule-6 escalations remain open pending user action.
- **Distinct log format**: `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"`. This is grep-distinguishable from the normal `auto-answered` line for after-action review.

### Safety

- Re-capture before send eliminates detection-to-send race condition
- No cooldown or retry limit — PR review is the safety net
- Per-answer logging for all auto-answers, escalations, and auto-defaults

---

## Autopilot

- Pipeline: `/fab-fff` (not `/fab-ff`)
- Gate: confidence score threshold per change type
- Per-change loop: spawn -> gate -> monitor -> merge -> rebase -> cleanup -> progress
- Failure matrix covers: confidence below gate, review fails, rebase conflict, pane death, stage timeout, total timeout
- Interruptible: stop/skip/pause/resume
- Resumable from `fab pane-map` state reconstruction

---

## Key Properties

| Property | Value |
|----------|-------|
| Requires active change? | No |
| Runs preflight? | No |
| Read-only? | No — sends commands to other agents, auto-answers questions |
| Idempotent? | Yes — state is re-derived before every action |
| Advances stage? | No |
| Outputs `Next:` line? | No — ends with ready signal |
| Loads change artifacts? | No — coordination context only |
| Requires tmux? | Yes for pane-map, resolve --pane, monitoring, auto-nudge; status-only mode without |
| Uses `/loop`? | Yes — for proactive monitoring after every send |

---

## Resolved Design Decisions

1. **Standalone over inheritance chain.** Reading operator4 previously required mentally merging ~800 lines across 4 files (operator1->2->3->4). The standalone rewrite contains all behavior in ~280 lines by offloading tool references to shared `_` files and explaining constraints concisely.

2. **All-auto-answer over two-tier classification.** Worktree isolation and human PR merge provide the safety gate. The two-tier model added pipeline latency without meaningful safety improvement.

3. **Re-capture before send over single-tick grace period.** Eliminates the race condition between detection and send without adding latency.

4. **`/fab-fff` for autopilot.** The more autonomous pipeline variant, fitting for operator-driven autopilot where human interaction is minimized.

5. **`/git-branch` after new change.** The operator sends `/git-branch` to the agent after detecting intake stage advancement for backlog-spawned changes, aligning branch names with change folders per `_preamble.md` § Naming Conventions.
