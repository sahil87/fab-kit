# Intake: Add Operator2 Monitoring Skill

**Change**: 260311-5c11-add-operator2-monitoring-skill
**Created**: 2026-03-11
**Status**: Draft

## Origin

> During a `/fab-discuss` session reviewing the `fab-operator1` skill, the user identified a fire-and-forget pattern: when operator1 sends `/fab-ff` to an agent, it announces the send and then goes idle — the user must manually ask "any updates?" or "tell me when X finishes" to learn the outcome. The user proposed that a new operator skill should use `/loop` to periodically check agent progress after sending commands, rather than going idle. The decision was to create `/fab-operator2` as a **workflow iteration** of operator1 — same full capabilities (send keys, create worktrees, merge PRs, all UCs) but with proactive monitoring baked into every action instead of fire-and-forget.

## Why

1. **Fire-and-forget is a bad UX pattern.** When operator1 sends `/fab-ff` to an agent, three things can happen (success, failure, stuck) — all three require the user to remember to check. Users shouldn't need to babysit.
2. **The monitoring pattern already exists but is locked inside autopilot.** Operator1's UC8 (autopilot) already uses `/loop` for periodic monitoring, but this capability is only available when running the full autopilot queue. Single-send monitoring doesn't exist.
3. **Iteration, not reduction.** Operator2 is the next version of the operator interaction model. It has all of operator1's capabilities — send keys, create worktrees, spawn agents, merge PRs, broadcast, sequenced rebases, autopilot — but replaces the fire-and-forget pattern with monitor-after-action. This is a workflow evolution, not an observer addon.

## What Changes

### New skill: `fab/.kit/skills/fab-operator2.md`

A full-capability operator with proactive monitoring. Inherits all of operator1's use cases (UC1–UC8) and adds automatic monitoring after any action that dispatches work to another agent.

#### Core behavioral difference from operator1

When operator2 sends a command to an agent (via `fab send-keys`), it does NOT simply report the send and go idle. Instead:

1. **Send the command** (same as operator1)
2. **Enroll the target change in monitoring** — capture its current state (stage, agent status) as a baseline
3. **Start or extend a `/loop`** at configurable interval (default 5m) to periodically check progress
4. **Each tick**: Read `fab pane-map`, compare against last known state for all monitored changes, report transitions:
   - Stage advances (e.g., "ftrh: spec → tasks")
   - Completions (e.g., "ftrh: reached hydrate — pipeline complete")
   - Failures (e.g., "ftrh: review failed")
   - Pane deaths (e.g., "ftrh: pane %689 is gone — agent exited")
   - Stuck detection (e.g., "ftrh: idle at apply for 15m — may be stuck")
5. **Terminal state**: Stop monitoring a change when it reaches hydrate/ship, the user says stop, or the pane dies

#### Inherited from operator1 (no changes)

All use cases carry over with the same behavior, same confirmation model, same pre-send validation, same bounded retries and escalation, same context discipline:

- **UC1**: Broadcast command to all idle agents
- **UC2**: Sequenced rebase after completion
- **UC3**: Merge completed PRs
- **UC4**: Spawn new worktree + agent from idea
- **UC5**: Status dashboard
- **UC6**: Unstick a stuck agent
- **UC7**: Notification surface — **upgraded**: instead of holding instructions in conversation context and checking on next user interaction, monitoring loop handles this automatically
- **UC8**: Autopilot — already uses `/loop`, behavior unchanged

#### What changes in inherited UCs

- **UC1 (Broadcast)**: After broadcasting, all recipients are auto-enrolled in monitoring
- **UC2 (Sequenced rebase)**: Monitoring detects when trigger change reaches target stage, operator sends the rebase automatically (already part of UC2 intent, but now driven by loop instead of next-user-interaction)
- **UC6 (Unstick)**: After nudging, the nudged agent is monitored for recovery
- **UC7 (Notification)**: "Tell me when X finishes" → immediate enrollment in monitoring loop. No need for "on next user interaction" caveat.

#### Design constraints

- **Pane-map only**: Uses `fab pane-map` as its sole observation primitive. No `fab runtime is-idle` (same cross-worktree issue that ftrh is fixing in operator1).
- **No change artifacts**: Like operator1, operator2 never reads intakes, specs, or tasks. Context window reserved for coordination state.
- **Stuck detection**: Flag agents idle at a non-terminal stage for >15m (configurable). Advisory — reports to user, does not auto-nudge (nudging requires explicit UC6 invocation or user instruction).
- **One operator at a time**: The operator tab runs either `/fab-operator1` or `/fab-operator2`. They do not run simultaneously.

### New launcher: `fab/.kit/scripts/fab-operator2.sh`

A launcher script that creates a singleton tmux tab named `operator` and invokes `/fab-operator2` in a new Claude session. Same singleton pattern as `fab-operator.sh` but invokes the new skill.

### Rename existing launcher: `fab-operator.sh` → `fab-operator1.sh`

Rename `fab/.kit/scripts/fab-operator.sh` to `fab/.kit/scripts/fab-operator1.sh`. Update the tab name from `operator1` to `operator` and keep the existing skill invocation (`/fab-operator1`). This makes the naming symmetric: `fab-operator1.sh` launches operator1, `fab-operator2.sh` launches operator2.

Also update the launcher reference in `fab/.kit/skills/fab-operator1.md` (line 61) from `fab-operator.sh` to `fab-operator1.sh`.

### New spec: `docs/specs/skills/SPEC-fab-operator2.md`

Per constitution requirement: changes to skill files must update the corresponding spec.

### What does NOT change

- `fab-operator1.md` — no modifications (operator1 remains as-is for users who prefer the reactive model)
- `fab pane-map` — no changes needed, already provides all required data
- `/loop` skill — consumed as-is
- All fab CLI primitives (`send-keys`, `pane-map`, `status`, `change`, etc.) — used as-is

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Add operator2 as a new execution skill — workflow iteration of operator1 with proactive monitoring

## Impact

- `fab/.kit/skills/fab-operator2.md` — new file (the skill)
- `fab/.kit/scripts/fab-operator2.sh` — new file (launcher)
- `fab/.kit/scripts/fab-operator.sh` → `fab/.kit/scripts/fab-operator1.sh` — rename + update tab name to `operator`
- `fab/.kit/skills/fab-operator1.md` — update launcher reference
- `docs/specs/skills/SPEC-fab-operator2.md` — new file (spec)
- `.claude/skills/` — deployed copy will be generated by `fab-sync.sh`
- `fab/.kit/skills/fab-operator1.md` — update launcher reference from `fab-operator.sh` to `fab-operator1.sh`

## Open Questions

None — approach was fully discussed before intake.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Separate skill, not a modification to operator1 | Discussed — user explicitly chose to create operator2 as a new skill | S:95 R:90 A:95 D:95 |
| 2 | Certain | Full capabilities — workflow iteration, not stripped-down observer | Discussed — user corrected the read-only assumption. Operator2 has all of operator1's capabilities plus monitoring | S:95 R:85 A:95 D:95 |
| 3 | Certain | Use `/loop` for periodic monitoring after every send | Discussed — pattern already proven in operator1's autopilot mode, now generalized | S:90 R:90 A:90 D:95 |
| 4 | Certain | Pane-map only, no `fab runtime is-idle` | Discussed — same cross-worktree issue that ftrh is fixing. Pane-map Agent column is the correct data source | S:95 R:90 A:95 D:95 |
| 5 | Certain | Default 5m monitoring interval | Discussed — user proposed 5m in the original conversation | S:90 R:95 A:85 D:90 |
| 6 | Certain | New launcher script `fab-operator2.sh` | Discussed — user explicitly requested a new launcher | S:95 R:90 A:90 D:95 |
| 7 | Certain | Tab name `operator` (shared with operator1 launcher) | Discussed — one operator tab, user chooses which skill to run | S:90 R:85 A:90 D:90 |
| 8 | Certain | One operator at a time in the shared tab | Discussed — they don't run simultaneously | S:90 R:85 A:90 D:90 |
| 9 | Certain | Constitution compliance: create spec alongside skill | Constitution requires skill changes to update corresponding spec | S:95 R:80 A:95 D:95 |
| 10 | Confident | Stuck detection threshold 15m, advisory only | Carried from operator1's autopilot guardrails — same threshold, no auto-nudge without explicit user instruction | S:75 R:90 A:80 D:85 |

10 assumptions (9 certain, 1 confident, 0 tentative, 0 unresolved).
