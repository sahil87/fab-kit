# fab-operator3

## Summary

Multi-agent coordination layer with proactive monitoring and auto-nudge capability. Extends `/fab-operator2` with question detection and answer confidence — when a monitored agent is idle and its terminal shows a question prompt, operator3 either auto-answers routine questions or escalates judgment calls to the user.

**Key difference from operator2**: Where operator2's stuck detection is advisory-only (report and wait), operator3 adds input-waiting detection that actively resolves blocks for routine questions. All operator2 behavior is inherited unchanged.

Not a lifecycle enforcer — the operator answers operational questions as a proxy for the user, not advancing stages or making pipeline decisions.

---

## Primitives

Inherits all operator2 primitives plus:

| Primitive | Source | Purpose |
|-----------|--------|---------|
| `tmux capture-pane -t <pane> -p -l 10` | tmux | Capture last ~10 lines of an agent's terminal for question detection |
| `tmux send-keys -t <pane> "<answer>" Enter` | tmux | Send auto-answer to an agent's terminal (same mechanism as operator2 interaction) |

All operator2 primitives remain: `fab pane-map`, `fab resolve --pane`, `wt list` + `fab change list`, `/loop`.

---

## Question Detection

### Terminal Heuristic

For each monitored agent that is **idle** (per pane-map Agent column), the monitoring tick captures the last ~10 lines of the agent's terminal via `tmux capture-pane -t <pane> -p -l 10` and scans for question indicators.

### Question Indicator Patterns

- Lines ending with `?`
- `[Y/n]`, `[y/N]`, `(y/n)`, `(yes/no)` patterns
- `Allow?`, `Approve?`, `Confirm?`, `Proceed?` keywords
- Claude Code permission prompts (tool approval patterns)
- `Do you want to...`, `Should I...`, `Would you like...` phrasing

The heuristic is intentionally broad — false positives are handled by the answer confidence model (escalate when unsure), not by tightening detection.

### Bottom-Most Indicator Rule

When multiple question indicators appear in the captured output, the operator evaluates the most recent (bottom-most) indicator, consistent with standard terminal interaction patterns.

### Idle-Only Guard

Question detection runs ONLY for agents that are idle in the pane map. Active agents are not scanned.

---

## Answer Confidence Model

### Two-Tier Classification

When a question is detected, operator3 classifies it into one of two tiers:

**Auto-answer** — one obvious right answer given the pipeline context:
- Permission/approval prompts → approve
- "Should I commit before rebasing?" → yes
- "Proceed with the changes?" → yes
- "Continue?" → yes
- Simple operational confirmations where saying "no" would stall the pipeline

**Escalate** — multiple valid paths, judgment required:
- Questions involving tradeoffs or architecture decisions
- Questions involving destructive or irreversible actions ("Should I delete these files?")
- Questions offering multiple approaches ("Which approach: A or B?")
- Questions about failing tests ("Tests are failing, should I fix or skip?")

### Classification Heuristic

If the question is a binary yes/no where "yes" continues the work and "no" stalls it, AND the question does NOT involve destructive or branching choices → auto-answer. Everything else → escalate.

### Auto-Answer Action

Send the appropriate affirmative response via `tmux send-keys -t <pane> "<answer>" Enter`. Report: `"{change}: auto-answered '{summary of question}' → {answer}"`.

### Escalation Action

Do NOT send any answer. Report: `"{change}: waiting for input — '{summary of question}'. Please respond."`.

### No Nudge Budget

No cooldown or retry limit. Each question is evaluated independently on its own merits by the confidence model. The PR review step serves as the safety net against compounding bad auto-answers.

---

## Monitoring Tick Changes

### Input-Waiting Detection Step

The monitoring tick gains a new step (step 5) between pane death detection and stuck detection. The full tick order:

1. Stage advance detection *(inherited)*
2. Pipeline completion detection *(inherited)*
3. Review failure detection *(inherited)*
4. Pane death detection *(inherited)*
5. **Input-waiting detection** *(new)* — for idle agents, run the terminal heuristic. If question detected: auto-answer or escalate per the confidence model. Mark agent as input-waiting.
6. Stuck detection *(inherited, modified)* — only for agents NOT detected as input-waiting

**Key invariant**: Input-waiting detection runs before stuck detection. An agent waiting for input is not "stuck" — it's blocked on user input. Stuck detection only flags agents that are idle without a visible question.

---

## Relationship to Operator2

| Aspect | Operator2 | Operator3 |
|--------|-----------|-----------|
| Monitoring | Proactive via `/loop` | Same — inherited |
| Use cases (UC1–UC8) | All 8 | Same — inherited |
| Stuck detection | Advisory only — report, don't act | Modified — skips input-waiting agents |
| Input-waiting detection | Not present | New — terminal heuristic + confidence model |
| Auto-answer | Not present | New — auto-answers routine questions |
| Escalation | Not present | New — reports judgment-required questions |
| Confirmation model | 3-tier (read-only, recoverable, destructive) | Same — inherited |
| Pre-send validation | Pane exists + agent idle | Same — inherited |
| Context discipline | No change artifacts | Same — inherited |
| Configuration | Monitoring interval, stuck threshold, autopilot tick | Same — inherited |

### One Operator at a Time

Like operator1 and operator2, operator3 uses the shared singleton tmux tab name `operator`. Only one operator (1, 2, or 3) runs at a time.

---

## Resolved Design Decisions

1. **Terminal heuristic over time-based detection.** Detection uses `tmux capture-pane` content analysis rather than idle-time thresholds. Terminal content directly reveals whether an agent is waiting for input; time-based detection cannot distinguish "waiting for input" from "slow but working."

2. **Two-tier model over multi-tier.** A binary classification (auto-answer vs escalate) is sufficient for v1. Simpler mental model, easier to reason about safety; the critical distinction is "one obvious answer" vs "judgment needed."

3. **No audit trail for v1.** Auto-answer actions are reported inline in the operator's output but not logged to a persistent file. PR review catches compounding errors.

4. **Hardcoded question patterns.** Detection patterns are embedded in the skill file, not configurable via config.yaml. Simpler for v1; patterns are stable across projects.

5. **No cooldown between auto-answers.** Each question is evaluated independently; PR review is the safety net. Cooldowns reduce utility without meaningfully improving safety when each question is independently assessed.

6. **Skill file references operator2 rather than duplicating.** Operator3's skill file starts with a directive to read operator2.md first, then defines only the new behavior. This avoids content duplication and ensures operator2 updates automatically propagate.
