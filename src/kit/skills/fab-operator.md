---
name: fab-operator
description: "Use when coordinating multiple fab agents across tmux panes — multi-agent monitoring, auto-answering prompts, routing commands, driving autopilot queues, and dependency-aware agent spawning."
helpers: [_cli-agents, _cli-fab, _cli-external]
---

# /fab-operator

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

## Contents

- 1. Principles
- 2. Startup
- 3. Safety
- 4. The Loop
- 5. Auto-Nudge
- 6. Coordination Patterns
- 7. Watches
- 8. Configuration
- 9. Key Properties

Multi-agent coordination layer. Runs in a dedicated tmux pane, observes agents across all sessions on its tmux server (per tick via `fab operator tick-start --diff --quiet`, on demand via `fab pane map --all-sessions`), routes commands and answers via `rk mux send` when rk is installed (`command -v rk`-gated — plain for command routing, `--answer` for prompt answers, `--key` for key-name input), degrading to raw `tmux send-keys` behind its own §3 state gate when rk is absent — never an error — and monitors progress via `/loop`. Spans multiple repos and sessions on one server. The loop is the heart of the operator.

Start via `fab operator` (singleton tmux tab named `operator`). The launcher requires **neither a git repo nor a resolvable `fab/` project** — matching the per-server, cross-repo singleton model, whose natural launch point is a neutral parent directory (e.g. `~/code`). Its exact degraded behavior (window cwd, session command, `operator`-role model resolution and built-in defaults) is documented in `_cli-fab.md` § fab operator and is the canonical §9 Key Properties rows below.

---

## 1. Principles

| Principle | Rule |
|-----------|------|
| Coordinate, don't execute | Route implementation to agents; ask when ambiguous. Perform only coordination-level maintenance such as merge, archive, and worktree deletion directly (§6). |
| Multi-repo aware | Address every agent as `(session, repo, pane)` on one tmux server, with pane ID primary and every monitored/watch/`branch_map` entry repo-qualified; state is one server-keyed file (§4, §8, §9). `session` is a **display/context dimension, never a join key** — a monitored agent's session can change mid-lifetime (`move-window` relocation), so correlation rides the pane ID (§ fab pane map's identity-key contract in `_cli-fab.md`). |
| Spawn in a worktree | Reserve the operator pane for coordination. Every pipeline command, including a one-line change, starts with `wt create --non-interactive` and runs in a fresh agent tab (§6). |
| Automate the routine | Auto-answer, nudge, rebase, and spawn for routine operations; PR review is the safety net. Every operator-spawned agent is monitored automatically (§4–§7). |
| Do not enforce lifecycle | Agents self-govern pipeline transitions; report unexpected stages factually (§4). |
| Keep context lean | Never read intake/spec/plan artifacts; retain only pane maps, snapshots, and operator state (§2, §4). |
| Re-derive state | Before every action, query `fab pane map --all-sessions`; never trust conversational pane/repo/session/stage values (§4). |
| Survive compaction | The agent cannot `/clear` itself. When a tick fires and §4 Tick Behavior is no longer in context (harness auto-compaction, or a session resumed from a summary), run `/fab-operator` **once** to reload, re-run §2 Init, then resume lean `operator tick` firings; monitored/autopilot/branch_map/notes survive in the server-keyed state file (§4 Post-Compaction Reload). |
| Route pipeline-first | New work MUST enter through `/fab-new`, then `/fab-fff`, `/fab-ff`, or `/fab-continue`; never send raw implementation instructions or use `/fab-continue` to skip intake. Coordination maintenance remains direct (§6). |

---

## 2. Startup

### Context Loading

Load only `fab/project/config.yaml`, `fab/project/constitution.md`, and `fab/project/context.md` (optional — skip gracefully if missing). The operator is a listed exception to the `_preamble.md` §1 always-load layer: code-quality, code-review, and the doc indexes serve artifact generation and review, which the operator never does (§1 Context discipline) — and a long-lived session re-pays any loaded file after every reload (compaction, `/clear`, or restart — §4 Post-Compaction Reload). Do not run `fab preflight`. Do not load change artifacts.

Helpers declared in frontmatter: `_cli-agents` (the generic agent-CLI interaction procedures — spawn composition, pre-send validation, delivery probe, peek, await — plus the per-provider grammar/discovery dictionary), `_cli-fab` (fab command reference), and `_cli-external` (wt, idea, tmux, /loop reference). Naming conventions are inlined in `_preamble.md` § Naming Conventions — already loaded.

The split between `_cli-agents` and this file is **agent primitives vs. operator orchestration**: `_cli-agents` owns *how* to talk to an agent CLI (the mechanics any session could reuse); this file owns *when and whether* to (confirmation tiers, retry budgets, repo targeting, enrollment, dependency resolution, autopilot).

The operator needs full command vocabulary to make routing decisions (e.g., knowing a fresh idea needs `/fab-new` → `/fab-fff` — fab-new creates the branch inline — while a mis-aligned tab needs `/git-branch` first).

After context loading, log the command invocation:

```bash
fab log command "fab-operator"
```

### Tmux Gate

If `$TMUX` is unset, STOP:

```
Error: operator requires tmux. Start a tmux session first.
```

### Role Mark

Mark this tmux window as the operator for run-kit's dashboard (the `@rk_win_role` window option — rk owns the option contract, the pinned rendering, and the one-operator-per-server radio semantics; fab is only the producer):

```bash
command -v rk >/dev/null 2>&1 && rk role operator >/dev/null 2>&1 || true
```

Fail-silent by contract (`_preamble.md` § Run-Kit (rk) Reference), extended to version skew: an absent rk, or an installed rk predating the `role` subcommand, degrades to a silent no-op — never an error, never a blocked startup. Idempotent: a restarted operator re-marks the same window harmlessly. There is no unmark step — the operator has no clean exit hook, and staleness and radio conflicts are rk's to resolve.

### wt Gate

`wt create` is the operator's first action for any new request (§1 Spawn-in-worktree), so probe it **once here** — not at each call site. `wt` ships as a standalone formula (not a `fab-kit` Homebrew dependency), so it may legitimately be absent. If `command -v wt >/dev/null 2>&1` fails, STOP:

```
Error: wt is required for operator spawning — install it via: brew install sahil87/tap/wt
```

This single preflight probe covers every later `wt create` call site; none is individually gated.

### Init

1. Run `fab operator state` to read (or create, on first run) the server-keyed operator state file — the binary derives the path and persists the empty skeleton when missing; the operator never computes the path or hand-creates the file (`_cli-fab.md` § fab operator state). Old repo-rooted `.fab-operator.yaml` files are not read or migrated
2. Restore monitored set, autopilot queue, branch_map, and notes from the file (this is what makes §4 Post-Compaction Reload lossless)
3. Run `fab pane map --all-sessions` and display the output (all sessions on this server, not just the operator's own)
4. If any tracked items exist (monitored set, autopilot queue, watches, or an in-progress merge sequence — an open merge-sequence `coordination` note), start the single loop per §4 Adaptive cadence, using the literal from §4 Loop Prompt
5. Output the ready line **with the loop literal** — the agent copies it later, never composes one:

   ```
   Operator ready. Loop active (3m) — /loop 3m "operator tick"
   Operator ready. Loop idle — start with /loop 3m "operator tick" on first enrollment
   ```

   (first form when step 4 started the loop, with `{interval}` = the active cadence, `3m` or `90s`; second form when nothing is tracked yet)

---

## 3. Safety

### Confirmation Tiers

| Tier | Examples | Behavior |
|------|----------|----------|
| Read-only | Status check, pane map | No confirmation |
| Recoverable | Send `/fab-continue`, rebase | Announce before sending |
| Destructive | Merge PR, archive, delete worktree | Confirm before executing |

### Pre-Send Validation

Before sending keys to any pane, run the two-step gate in **`_cli-agents.md` § Pre-Send Validation** (pane exists via a refreshed pane map → agent state fits the send intent per the three-state `@rk_pane_agent_state` read — the same mode-aware gate `rk mux send` enforces), then apply the operator's own policy on its outcome plus the two operator-specific checks:

1. **Pane gone** (gate step 1 fails) — report "Pane for {change} is gone." Do not send.
2. **Agent not `idle`** (gate step 2) — the operator does **not** silently proceed. If `active` or `waiting`: "{change} is {state}. Sending may corrupt its work / cut across a pending human answer. Send anyway?" — send only on explicit confirmation, and keep the confirmed send gated: with rk installed (`command -v rk`), a `waiting` target rides `rk mux send --answer`, an `active` target requires `rk mux send --force` (the deliberate skip-everything override); with rk absent, the confirmed send is raw `tmux send-keys` behind the operator's own state gate just performed. If unknown (`—`, no `@rk_pane_agent_state` on the pane): the agent isn't instrumented; confirm before sending (plain send then warns-and-sends — no override needed). Only `idle` sends unattended. *(This routed-command confirm policy is unchanged by `--answer` — the flag swaps the mechanism after confirmation, not the ask; the auto-answer flow in §5 is the unattended `--answer` consumer.)*
3. **Check change is active** — if the target change isn't the active change in that tab, send `/fab-switch <change>` first.
4. **Check branch alignment** — if the tab's git branch doesn't match the change folder name, send `/git-branch` to align it.

When a send appears to land but the agent never starts working, apply the **delivery probe** in `_cli-agents.md` § Delivery Probe (the printed-prompt trap: probe with a literal sentinel, `C-u`, retype, Enter, confirm via a working indicator) rather than re-sending blind.

### Branch Fallback

When `fab resolve` fails during a **user-initiated** action (not monitoring ticks):

1. Scan branches: `git for-each-ref --format='%(refname:short)' refs/heads/ refs/remotes/ | grep -iF "<query>"`
2. **Single match, read-only**: read `.status.yaml` via `git show <branch>:fab/changes/<folder>/.status.yaml`
3. **Single match, action**: create a worktree and proceed — probe-and-route the branch per `_cli-external.md` § wt (existing branch → `wt create --non-interactive --worktree-name <name> --checkout <branch>`; missing → the positional form)
4. **Multiple matches**: disambiguate. **No match**: report not found.

### Bounded Retries

| Situation | Max retries | Escalation |
|-----------|-------------|------------|
| Stuck agent nudge | 1 | "{change} appears stuck at {stage}. Manual investigation recommended." |
| Rebase conflict | 0 | Immediately flag to user |
| Pane death | 0 | Report gone. Respawn only in autopilot (1 attempt) |
| Agent exited (pane survives as a shell) | 0 | Report gone (pane kept, cwd intact). Respawn only in autopilot (1 attempt): kill the leftover shell pane first (`rk mux kill` when rk is installed — an uninstrumented/idle pane passes its gate — else `tmux kill-pane -t <pane>`), then spawn per §6 |
| Send to busy agent | 0 | Warn, require explicit confirmation |
| Cherry-pick conflict | 0 | Abort, log, escalate. Do not spawn. |

---

## 4. The Loop

The loop is the operator's heartbeat — a `/loop` whose prompt is the bare `operator tick` (§ Loop Prompt) that runs as long as the monitored set is non-empty, an autopilot queue is active, any watch is configured, or a merge sequence is in progress (an open merge-sequence `coordination` note — §6 Auto-Merge Choreography rule 4). When all four are empty, stop the loop. The loop starts when the first change is enrolled, an autopilot queue begins, a watch is created, or a merge sequence starts — always as `/loop 3m "operator tick"` (§ Loop Prompt). A user prompt can also restart it.

**Adaptive cadence.** The heartbeat interval is **not fixed** — it adapts to whether any monitored agent is `waiting` (blocked on a human):

- **Normal cadence: `3m`** (the default). Used when no monitored agent is `waiting` (or input-waiting).
- **Tightened cadence: `90s`** (§8, overridable). The moment a tick detects **any** monitored agent in the **`waiting`** Agent-column state (the pane's `@rk_pane_agent_state` is `waiting` — the agent is blocked on a permission prompt / menu / elicitation), the operator tightens the heartbeat to bound worst-case detection/pickup latency. `waiting` is the primary, event-driven trigger; capture-based §5 menu detection is the fallback for uninstrumented panes (`—`). When a later tick finds no monitored agent `waiting` (or menu-waiting), it relaxes back to `3m`.
- **One-loop invariant.** Adapting cadence means **re-establishing the single loop at the new interval** (e.g. restart `/loop 90s "operator tick"`), never running two loops concurrently (`_cli-external.md` § /loop — "one loop at a time"). The operator changes the interval of *the* loop; it does not add a second.
- **Autopilot composition.** Autopilot has no cadence of its own — a driving queue rides the same single `3m`/`90s` loop (§6 actions run at tick step 4). This section is the only cadence owner.

### Loop Prompt

The exact invocations — **copy one of these, never compose your own**:

```
/loop 3m "operator tick"
/loop 90s "operator tick"
```

The first is the normal cadence; the second is the tightened cadence (Adaptive cadence above, §8). The lines are comment-free on purpose — anything after the closing quote would ride into the slash command.

The loop prompt **MUST be the bare text `operator tick`**. It **MUST NOT** be `/fab-operator` or any other slash command. Reason: a slash command macro-expands its full source into the turn on **every** firing — this file alone is ~21k tokens, so a `/fab-operator` loop prompt re-pays the whole skill each tick (~400k tokens/hour at `3m`) and exhausts the context window in roughly ten ticks. The tick procedure (§4 Tick Behavior) is already in context; the prompt only needs to *name* it.

`/loop` also has a **self-paced (dynamic) mode** with no fixed interval, where the model hands a wakeup prompt back each tick. Either mode is permitted; in dynamic mode the wakeup prompt handed back **MUST likewise be the bare `operator tick`** — the same rule applied to the string the agent returns rather than the string it typed.

Recovery when this procedure is no longer in context: § Post-Compaction Reload.

### Post-Compaction Reload

**Trigger** — a tick (`operator tick`) arrives and §4 Tick Behavior is not in context: the agent cannot see the numbered Snapshot → Auto-nudge → Watches → Autopilot → Removals → Observed-field updates → Loop lifecycle list. Typical causes: harness auto-compaction of a long session, a fresh session resumed from a conversation summary, a user `/clear`.

**Procedure**:

1. Run `/fab-operator` exactly **once** — this reloads the skill body and its helpers and re-runs §2 Startup including Init (state file re-read via `fab operator state`, `fab pane map --all-sessions`, loop re-establishment per § Loop Prompt).
2. Treat the tick that triggered the reload as consumed — the next tick's `fab operator tick-start --diff` re-emits every level-triggered delta (§4 Tick Behavior step 1), so nothing durable is lost.
3. Continue with bare `operator tick` firings. **Never** put `/fab-operator` into the loop prompt as a way to "stay reloaded" — that is the failure mode this procedure replaces.

**Durable state** — monitored set, autopilot queue, `branch_map`, watches, and notes all live in the server-keyed operator state file and survive compaction, `/clear`, crash, and restart; only §8 session-scoped settings and in-conversation context are lost.

**`/clear` is a user action.** A user may `/clear` a bloated operator; it lands on this same procedure (the next tick, or the user's next message, finds no procedure in context). The skill never instructs the agent to `/clear` — the agent-side mechanism is *compaction → one-shot `/fab-operator` reload*.

### Operator State File

Persistent state, read on startup (§2 Init) and in the tick's watch pass via `fab operator state` — the tick's monitored-fleet data rides `fab operator tick-start --diff` instead (§4 Tick Behavior). The term **operator state file** used throughout this skill refers to the server-keyed file from §2 Init step 1 — one per tmux server spanning every repo it coordinates.

**The operator never hand-writes this file.** Every mutation goes through a `fab operator` subcommand (`enroll`/`update`/`remove`, the `note` verbs, the `watch` verbs, the `autopilot` verbs, `branch-map rm`) — agents state intent through flags; the binary owns the schema, the timestamps, the list-cap pruning, and the atomic write (same doctrine as `fab score`: agents never compute what the binary can own). The schema block below is *reference* documentation of what the binary maintains; the command contracts live in `_cli-fab.md` § fab operator.

```yaml
tick_count: 47
monitored:
  r3m7:
    pane: "%3"
    repo: /home/user/code/foo            # absolute main-worktree root for this agent's repo
    session: work                         # tmux session the agent's window lives in
    stage: apply
    agent: active
    stop_stage: null       # null = full pipeline, or a stage name to park at
    spawned_by: null       # watch name if spawned by a watch, null otherwise
    depends_on: []         # change IDs — same-repo deps cherry-pick, cross-repo deps are ordering-only; both gate on § Dependency satisfied (§6)
    branch: 260324-r3m7-add-retry-logic  # this change's branch name
    enrolled_at: "2026-03-23T17:30:00Z"
    last_transition: "2026-03-23T17:32:00Z"
autopilot:
  queue: [ab12, cd34, ef56]
  current: cd34
  completed: [ab12]
  state: running           # running | paused | null
branch_map:                # persists branch+repo after changes leave monitored set; value is { branch, repo }
  ab12: { branch: 260324-ab12-fix-auth, repo: /home/user/code/foo }
  cd34: { branch: 260324-cd34-add-oauth, repo: /home/user/code/bar }
watches:
  linear-bugs:
    enabled: true
    source: linear
    query: { project: "DEV", status: [Backlog, Todo], assignee: "@me" }
    target_repo: /home/user/code/foo   # repo the watch's spawned changes land in (§7)
    stop_stage: intake
    known: [DEV-988, DEV-992]  # capped at 200, oldest pruned first
    completed: [DEV-985]       # items that reached stop_stage
    last_checked: "2026-03-23T17:29:00Z"
    last_error: null
    instructions: >
      Spawn agents for issues older than 1 hour with label 'bug'.
      Max 2 concurrent agents from this watch.
notes_seq: 2             # persisted id counter — ids n<N> are never reused after prune
notes:
  - id: n1
    kind: phase_plan     # dependency_wait | phase_plan | coordination | correction
    text: Phase 2 of 4 — auto-merge armed behind s2gw
    refs: [s2gw, fab-kit]
    created_at: "2026-08-23T10:00:00Z"
    updated_at: "2026-08-23T12:00:00Z"
    resolved: false
    resolved_at: null
```

### Monitored Set

Each entry tracks: change ID, pane, **repo** (absolute main-worktree root), **session** (tmux session name), last-known stage, last-known agent state, stop_stage, spawned_by (watch name or null), depends_on (change IDs — same-repo cherry-pick, cross-repo ordering-only per §6), branch (this change's branch name), enrolled-at, last-transition-at. The pane ID is the server-global primary key; `repo` and `session` are the `(session, repo, pane)` addressing dimensions (§1). The recorded `session` is **context, not identity**: never re-derive which entry a pane is from a session name or window index — they are reassigned by `swap-window`/`move-window`/rename, and a monitored agent's session can change mid-lifetime (rk's planned `_rk-operator` relocation moves the window at enrollment); re-derive per tick from the tick's snapshot (`fab operator tick-start --diff`, § Tick Behavior) keyed on the pane ID (§ Re-derive state).

**Enrollment**: operator sends a command to a change, user requests monitoring, or operator triggers an automatic action (including autopilot and watch spawns). Read-only actions do not enroll. Enrollment is `fab operator enroll <change-id> --pane … --repo … --session … --branch … [--stage …] [--agent …] [--stop-stage …] [--spawned-by …] [--depends-on …]` (contract in `_cli-fab.md` § fab operator) — one command writes both the monitored entry and the `{ branch, repo }` pair in the top-level `branch_map`.

After the enroll call, the operator MUST prefix `»` (U+00BB) to the target tmux window's name via the `fab pane window-name ensure-prefix` primitive. The primitive enforces the idempotent literal-prefix check internally, so the rename applies to every enrollment path without the caller needing to guard:

```sh
fab pane window-name ensure-prefix <pane> »
```

Windows that already carry `»` (operator-spawned windows from §6, reload-restored entries, re-enrolled changes) no-op through the primitive's guard. A non-zero exit — pane vanished between refresh and rename (exit 2) or any other tmux error (exit 3, including tmux not running / socket unreachable) — causes the operator to log one line and continue. Enrollment itself is already durable from the preceding server-keyed state file write:

```
{change}: window rename skipped ({error}).
```

**Removal**: change completes (its `completion` delta — at/past its `stop_stage`, or `review-pr` done/skipped when `stop_stage` is null), pane dies, user explicitly stops. Removal is `fab operator remove <change-id>` — the `branch_map` entry is **not** removed by it; it persists for downstream dependency resolution. On every removal path, the operator MUST swap the active-monitoring `»` prefix for the done-marker `›` (U+203A, SINGLE RIGHT-POINTING ANGLE QUOTATION MARK) via the `replace-prefix` primitive:

```sh
fab pane window-name replace-prefix <pane> » ›
```

The primitive's literal-prefix guard protects user-renamed windows (if the user renamed the window mid-monitoring so it no longer starts with `»`, the call no-ops). Exit 2 (pane missing — window is gone anyway) is treated as successful removal; other non-zero exits log `"{change}: window rename skipped ({error})."` and the operator continues. This keeps the tab bar an accurate at-a-glance map of what is currently tracked (`»` active) vs. operator-touched (`›` trail).

**Stop stage**: when `stop_stage` is set on a monitored entry, the operator treats that stage as the terminal stage for that change. On reaching it, the operator reports completion and removes the change — it does not push the agent further. Default is `null` (full pipeline: the change completes only when `review-pr` is done/skipped — hydrate and ship are mid-pipeline and never complete an entry by themselves). Spawns that deliberately park earlier — e.g. a `/fab-ff` run, which stops after hydrate — MUST enroll with `--stop-stage hydrate`; otherwise the entry never completes and sits in the monitored set until the user stops it.

### Branch Map

The top-level `branch_map` persists change ID → `{ branch, repo }` mappings. Entries are added by `fab operator enroll` when changes are enrolled in the monitored set. Entries persist after changes leave the monitored set (merged, archived, pane died) — this is necessary so downstream changes can still look up dependency branches for cherry-picking. The `repo` is required to disambiguate a dependency's branch across repos and to decide same-repo (cherry-pick) vs. cross-repo (ordering-only) resolution per §6. Entries persist until the user explicitly clears them (`fab operator branch-map rm <change-id>` or `--all`) — the server-keyed state file survives operator sessions, so there is no session-end expiry.

### Notes

Notes are the operator's owned surface for cross-cutting narrative state — the entries that outlive the `monitored` set and have no other schema home. The binary owns everything mechanical: ids (`n<N>` from the persisted `notes_seq` counter — never reused after prune), timestamps, the 500-character text cap, and the resolved-history pruning. The text body is free prose evaluated by the operator as an LLM (same split as watches: structured fields are machine concerns, prose is operator judgment). Command contracts live in `_cli-fab.md` § fab operator note.

**Verbs:**

- `fab operator note add --kind <k> [--ref <r>]... <text>` — creates a note and prints its id.
- `fab operator note update <id> <text>` — replaces the text in place (evolving phase-plan notes update rather than accreting near-duplicates) and refreshes `updated_at`.
- `fab operator note resolve <id>` — marks resolved (idempotent); resolved notes past a cap of 50 are pruned oldest-first. **Open notes never auto-expire** — notes are decisions, not a dedupe cache.
- `fab operator note list [--open|--all] [--json]` — open by default; renders id · kind · age from `updated_at` · first line of text, flags stale ones (`⚠ 21d` past 14 days, display-only), and warns on stderr above 25 open notes.

`fab operator state` prints an OPEN NOTES header (one line per open note) before the dump in human mode; the lifecycle is **add → update → resolve**, with a bounded resolved history.

**What belongs where** (the routing doctrine):

| Content | Home |
|---|---|
| Passive narrative read on restart/orientation — phase progress + holds, peer scoping agreements, report-back promises, corrections to earlier conclusions, **merge-gate dependency waits** (checked by operator judgment per tick — no git/GitHub watch `source` exists today) | **note** |
| Standing concerns a watch source can actually express today (`linear`/`slack` queries with `instructions`) | **watch** — if a git-source watch ships later, merge-gate waits migrate to it as its own change |
| Anything still true for a different operator next month — process lessons | **not operator state** — route via an `idea` backlog entry → a fab change into docs/memory (the operator has no memory write path: 3-file context load, may run with no `fab/` project at all); deliberately **no `lesson` kind** — its absence is the guard against notes degrading into a reflexive scratchpad |

### Tick Behavior

On each tick:

1. **Snapshot** — run `fab operator tick-start --diff --quiet`: one command increments `tick_count`, writes `last_tick_at`, snapshots the fleet internally, diffs it against the monitored baseline, and writes the baseline back in the same atomic mutation (full contract in `_cli-fab.md` § fab operator tick-start). Drop `--quiet` only when the user asks for status ("status", "any updates?", "show the fleet") — the binary's built-in every-10th-tick full document is the periodic full refresh, so no skill-side counter is kept. Stdout is one document: the `tick: N` / `now: HH:MM` header lines, then three YAML blocks — `deltas:` (events), `candidates:` (the step-2 sweep population, always emitted), and the frame block: `fleet:` (one row per monitored entry, pre-ordered repo → session → enrollment — **the status frame's data source**) or, on a quiet tick (no deltas, tick count not a multiple of 10), `fleet_summary:` (five counts — tracked/waiting/idle/active/unknown) **in place of** `fleet:`. The skill branches its frame on which key is present — see **Status Frame Format** below. Act on `deltas:` **before any answers** (a completion removes the entry and skips its answer). Each delta is one of:
   - `completion` (`review-pr` done/skipped, or at/past the entry's `stop_stage`) — report, then remove via step 5;
   - `pane_death` (the entry's pane is absent) — report, then remove via step 5;
   - `pane_mismatch` (tmux recycled the `%N` pane ID — a different change, or none, now occupies it; the delta carries `found`) — report + remove via step 5; a mismatched pane is never diffed and never a candidate;
   - `agent_exited` (detection semantics owned by `_cli-fab.md` § `fab operator tick-start`; the delta carries `command`) — report, then remove via step 5; an exited pane is never diffed and never a candidate (the §5 sweep must never type into it);
   - `stage_advance {from, to}` / `review_fail {from: review, to: apply}` — report in the frame.
   `completion`, `pane_death`, `pane_mismatch`, and `agent_exited` are **level-triggered**: they re-emit on every tick until acted on — `fab operator remove` is the ack, so a crash between diff and action loses nothing. `stage_advance` and `review_fail` are **consumed-on-read** (baseline-diffed); a lost one costs a missed report only. Output the status frame from the frame block — see **Status Frame Format** below. **Version-skew fallback** (two rungs, softest first): if `--quiet` errors as an unknown flag (older binary), drop `--quiet` for the session (keep `--diff`) and report the mismatch once; if `tick-start --diff` or `fab pane questions` errors as an unknown flag/command (new skill, older installed binary), fall back to the flagless tick for the session — `fab operator tick-start` + `fab pane map --all-sessions --json` + per-pane manual capture-and-scan + step-6 `update` bookkeeping — and report the mismatch once.

2. **Auto-nudge** — the per-tick sweep population is the tick's `candidates:` block (waiting-first, then idle — the binary computes it to match §5's policy exactly). Run each candidate through question detection (§5 — `waiting` is the primary signal). (No post-intake `/git-branch` nudge — `/fab-new` Step 11 creates or renames the branch inline; only a detected branch/change mismatch warrants a `/git-branch` send, per §3 pre-send validation item 4.)
3. **Watches** — read `fab operator state` here (the watch pass's own state read — `known` / `completed` / `last_checked` / `last_error`), then for each watch, query the source, compare against `known` + `completed` (§7 step 2's dedupe rule), spawn on new matches (§7).
4. **Autopilot dispatch** — if an autopilot queue is active, run the next autopilot action (§6); if a merge sequence is in progress, run its per-tick check (§6 Auto-Merge Choreography). Autopilot-driven changes are visible in the frame via `▶`.
5. **Removals** — ack the level-triggered deltas from step 1: remove completed changes (`completion` delta observed), dead panes, mismatched panes, and exited agents (the pane survives as a shell — kill it only when respawning, per §3 Bounded Retries) from the monitored set via `fab operator remove`. The event stops re-emitting once the entry is gone.
6. **Observed-field updates** — the per-tick `stage`/`agent` baseline write is owned by `tick-start --diff` (step 1): on the diff path the skill does **no** per-tick `fab operator update` stage/agent bookkeeping (a hand-written baseline would make the next diff under-report). `fab operator update <change-id>` stays for non-baseline field edits (e.g. `stop_stage`; the binary touches `last_transition` on a stage change). There is no whole-file persist step — every action above already persisted through its own verb.
7. **Loop lifecycle** — stop when no tracked state remains (monitored set, autopilot queue, watches, in-progress merge sequence); otherwise apply §4 Adaptive cadence — the single cadence owner, autopilot included — re-establishing the loop with a § Loop Prompt literal when the interval changes

Actions (nudges, removals, autopilot progress) render as an *italic* footnote line below the frame as they happen, `·`-separated, keeping them visually subordinate to the table frame:

```
*k8ds: auto-answered 'Allow Bash: npm test?' → y · Removed ab12 (complete), ef56 (pane gone) · Autopilot: cd34 → next ef56*
```

When the action log is long, the operator MAY split it across several italic lines rather than one — but each remains italic to stay subordinate to the frame.

### Status Frame Format

The frame is emitted as an assistant message that the agent harness renders as GitHub-flavored markdown in the terminal. **Render rule** (the binding constraint on every styling choice below): emit **bare markdown** — no code fence, no headings, no ANSI escapes (none of these survive the render path); the channels that DO render are **tables**, **emoji** (the only color channel), **bold** (`**…**`), *italic*, `code spans`, and plain URLs. The frame uses exactly these.

The frame has **two shapes**, chosen by which key the tick document carries (tick step 1):

- **Full frame** (`fleet:` present — a delta tick, every 10th tick, or a user status request run without `--quiet`): a **header line**, one **repo section** per repo (an anchor line + a change table), then a **Watches** section (anchor line + table). **Data source**: the change tables render from the tick's `fleet:` block (already ordered repo → session → enrollment — no per-tick `fab pane map` or `fab operator state` call feeds the frame); the Watches table is fed by the watch pass (tick step 3).
- **Compact frame** (`fleet_summary:` present — a quiet tick: no deltas, not a 10th tick): exactly **ONE line** — no anchors, no repo tables:

  ```
  🛰️ **Operator** · {HH:MM} · tick #{N} · **{tracked} tracked** · no change
  ```

  Append ` · {waiting} waiting` only when `waiting > 0` (e.g. `… · **8 tracked** · no change · 1 waiting`). `{tracked}` keeps the full header's definition (changes + watches): changes = `fleet_summary.tracked`, watches = the count from step 3's `fab operator state` read. The **Watches table** renders on a compact tick ONLY if the watch pass (step 3) produced news this tick — new items, a `last_error`, or an auto-disable; otherwise it is omitted.

On either shape, the *italic* action-footnote line still renders whenever an action happened (nudge / answer / removal / autopilot).

> **Runtime no-fence rule (agent-critical)**: do NOT wrap the frame in a ` ``` ` code fence. The fenced block below is for *documentation* (so this skill file shows the literal source). At runtime the operator must emit the header, anchors, and tables directly into its message body — a fenced frame renders as literal text (the tables would not lay out and the emoji/bold would not style).

Example (this is the literal markdown the operator emits, shown fenced here only to display the source):

```
🛰️ **Operator** · 17:32 · tick #47 · **8 tracked**

📂 **~/code/foo** · work

| | ID | Health | Stage | PR |
|:--:|---|:--:|---|---|
| ▶ | `r3m7` | 🟢 | apply → review | |
| | `ab12` | ✅ | hydrate | https://github.com/acme/foo/pull/412 |

📂 **~/code/bar** · side

| | ID | Health | Stage | PR |
|:--:|---|:--:|---|---|
| ▶ | `k8ds` | 🟡 | review · idle 8m | |
| | `ef56` | 🔴 | apply · idle 32m ⚠️ | |
| | `cd34` | ✅ | review-pr | https://github.com/acme/bar/pull/408 |

👁️ **Watches**

| Watch | Target | Health | Status |
|---|---|:--:|---|
| `slack-deploys` | ~/code/foo | 🟡 | 1 new · 2m ago |
| `linear-bugs` | ~/code/foo | 🟢 | 2 known · 1 completed · 3m ago |
| `slack-alerts` | ~/code/bar | 🟢 | 0 new · 1m ago |
```

| Element | Format | Notes |
|---------|--------|-------|
| Header | `🛰️ **Operator** · {HH:MM} · tick #{N} · **{N} tracked**` | Total includes changes + watches; no per-type/repo count |
| Compact frame | `🛰️ **Operator** · {HH:MM} · tick #{N} · **{N} tracked** · no change[ · {W} waiting]` | Rendered from `fleet_summary:`; replaces header+tables on a quiet tick |
| Repo anchor | `📂 **{repo-path}** · {session}` | One per repo; omit `session:` label. Null roots render `📂 **(unresolved repo)**` |
| Change table | Headerless centered `▶`, `ID`, `Health`, `Stage`, `PR` | ID is a code span; Stage may trail `⚠️`; PR is the full `pr_url`, never markdown display text |
| Watches table | `Watch`, `Target`, `Health`, `Status` | Watch name is a code span; Target is `target_repo`; Status is counts + relative time |
| Ordering | Repo → session → change; then watches | Sort repos by path, sessions by name, changes by enrollment, watches by name |
| Styling | Emoji, bold, italic, code spans, plain URLs | Bold header/title/count/repo; health emoji is the color channel; action log stays italic |
| Stuck marker | `⚠️` after Stage | Same non-terminal >15m idle condition as 🔴 (§8) |
| Autopilot marker | `▶` or blank | Marks queue-driven changes; completion remains visible through ✅ |
| Exited agent marker | `⏏ shell` as the Stage value | An `agent_exited` row renders like a `pane_death` row — baseline identity (repo/session/stage), `—` agent state — with this marker in place of a live stage, so it never reads as a live agent |
| Watch timestamp | `{N}s ago` / `{N}m ago` / `{N}h ago` | Floor division at 60s and 60m |

**Health emoji** (geometric glyphs like `●◌✗` render monochrome and are NOT used):

| State | Change | Watch | Emoji |
|-------|--------|-------|:-----:|
| active / healthy | active | last query ok, no new items | 🟢 |
| waiting / idle / new-items | `waiting` (blocked on a human) or idle | has new unprocessed items | 🟡 |
| stuck / errored | >15m idle at non-terminal | `last_error` set | 🔴 |
| complete | `completion` delta (review-pr done/skipped, or at/past `stop_stage`) | — | ✅ |
| paused | — | `enabled: false` | ⚪ |

### Idle Message

Between ticks, the operator displays an idle message with the current time and next-tick time:

```
Waiting for next tick. Time: 08:26 · next tick: 08:29
```

Run `fab operator time --interval {interval}` (where `{interval}` is the **currently active** loop interval — `3m` normally, `90s` when the cadence is tightened per §4 Adaptive cadence) to get the `now:` and `next:` values to fill in the message. A tightened cadence therefore shows the nearer next-tick time. This lets the user gauge staleness at a glance without scrolling to the last tick frame.

The idle message is the **only other per-tick output** besides the frame (and the action footnote when an action happened): nothing else — no restating the tick document, no echoing `candidates:`, no per-candidate "no question detected" lines.

---

## 5. Auto-Nudge

The operator auto-answers routine prompts from monitored agents. The per-tick question-detection population (tick step 2) is each `waiting` agent (the primary signal — see below) plus, as a fallback, each idle agent. The `questions` sweep's capture-based patterns **remain applicable** to `active`/unknown (`—`) panes — an uninstrumented harness, or a mid-turn prompt not yet flipped to `waiting` — via an on-demand `fab pane questions --panes <id>` call, but those panes are **not swept every tick**; the per-tick sweep is `waiting`+idle only.

**The `waiting` Agent-column state is the primary signal.** When a monitored pane's `@rk_pane_agent_state` is `waiting`, the agent is blocked on a human (permission prompt / menu / elicitation) — this is event-driven and covers all instrumented harnesses (Claude/codex/copilot/gemini), so it is the first-class trigger for both the tightened cadence (§4) and question detection here. A `waiting` pane MUST be capture-scanned and run through the answer model, with each **idle** pane as the per-tick fallback (the population stated above).

### Question Detection

Detection is a single binary sweep, not per-pane manual work: run `fab pane questions --panes <ids>` over the tick's `candidates:` block from `fab operator tick-start --diff` (population policy unchanged — `waiting` first, then idle; re-expressed here as the command's input). The command applies the mechanical guards and indicator patterns itself and returns `matches:` (pane, agent_state, indicator, snippet) plus `skipped:` with reasons — the full contract (flags, guards, indicator classes, skip-reason enum, JSON fields, exit codes) is owned by `_cli-fab.md` § fab pane · questions. Capture and state-read mechanics (including the uninstrumented-pane state-writer caveat that makes capture the universal fallback) are in `_cli-agents.md` § Peek. Claude Code permission/tool-approval prompts are **not** mechanized as their own class — in practice they are covered by the yes/no, action-word, imperative, and enumerated classes; novel prompt shapes remain operator judgment via an on-demand `--panes` sweep or manual capture.

1. **Sweep**: `fab pane questions --panes <ids>` over the tick's `candidates:` block (full contract: `_cli-fab.md` § fab pane · questions)
2. **No match** → stuck detection applies
3. **Match** → answer model

### Answer Model

Evaluate in order:

1. Binary yes/no or confirmation → `y`
2. `[Y/n]` or `[y/N]` → `y`
3. Claude Code permission prompt → `y`
4. Numbered menu or multi-choice → classify with LLM judgment using option length, semantic distinctness, surrounding context, and reversibility; no keyword list. Treat uncertainty as Strategic, then use the decision table below.
5. Open-ended answer determinable from visible context → send that answer.
6. Cannot determine keystrokes → use the decision table's hard-exclusion row.

### Non-Blocking Strategic Handling

Strategic handling MUST NOT block the loop: decide out-of-band, continue with the next monitored change in the same tick, and detect any asynchronous user resolution on a later tick.

| Classification | Action | Notify | Watchdog |
|----------------|--------|--------|----------|
| Routine menu (tool/permission, binary-framed, synonymous options) | Send first/default (`1`) after the re-capture guard | No | None |
| Strategic + defensible recommendation | Auto-pick the recommendation after the re-capture guard; PR review is the reversal point | Yes | None; already resolved |
| Strategic + no defensible default | Leave open and keep ticking | Yes | 30m idle auto-default |
| Cannot determine keystrokes (rule 6) | Leave open for the user; never guess | Escalate | Hard-excluded from auto-pick and auto-default |

### Notification Send

The notification is a single out-of-band send when the operator auto-picks or
leaves open a Strategic prompt. Use the default `rk notify` command and gate in
`_cli-external.md` § rk (run-kit).

**When `rk` is absent** (operator running where run-kit isn't installed), fall back to the first available **documented alternative**, configurable via the §8 `Notify channel` setting:

- **ntfy.sh** — `curl -d "{change}: {summary} ({repo})" ntfy.sh/<high-entropy-topic>`. No account, curl-from-shell, cross-repo aggregator, mobile push. **High-entropy topic REQUIRED** — public topics are world-readable to anyone who knows the name (the topic name is the only secret), so use a long random topic (e.g. `op-9f3a2c7e-strat`) and never put secrets in the body. The strongest no-run-kit fallback.
- **Discord webhook** — `curl -H 'Content-Type: application/json' -d '{"content":"…"}' <webhook>`. No account, one webhook = one channel, indefinite searchable history, mobile push.
- **`PushNotification`** (built-in Claude Code harness tool) — zero infra, no topic secret to leak, headless-safe; a *personal* push to the user's Claude apps, not a shared searchable feed. Good "just ping me" fallback.
- **Slack MCP** (`mcp__claude_ai_Slack__slack_send_message`) — searchable channel feed, mobile push; caveat: an interactively-authed MCP may be **absent in headless/cron** runs, so it cannot be a headless default.

**All notify sends fail silently** (the fallback path matches `rk notify`'s contract per `_preamble.md` § Run-Kit (rk) Reference). A notification that cannot be delivered (server unreachable, channel down, no subscriptions, `curl`/tool missing) MUST NOT crash or stall the loop — the operator logs one line and keeps ticking.

### Sending Auto-Answers

Deliver text answers via `rk mux send <pane> "<text>" --answer` when rk is installed (`command -v rk`-gated) — the answer-mode gate permits `waiting` (the auto-answer's primary target) and `idle`, still refuses `active`, and validates pane existence (full contract is tool-owned via `rk skill`; the usage summary lives in `_cli-agents.md` § Pre-Send Validation). Key-name answers (bare Enter, arrows, `C-c`) ride `rk mux send --key` on the same path. When rk is absent, the answer is raw `tmux send-keys` (keys and literal text alike) behind the same gate — never an error.

Before the send: run the §3 pre-send gate (`_cli-agents.md` § Pre-Send Validation — pane exists; state read per its step 2, expecting `waiting` or the idle fallback), then re-capture the terminal (the 20-line capture whose mechanics live in `_cli-agents.md` § Peek — `rk`-gated with raw-tmux fallback). If output changed since detection, abort — agent is no longer waiting. `fab pane questions` output is detection input only — it never replaces the pre-send gate or this re-capture-before-send guard (the batch capture is older than the just-in-time one, so the guard matters more, not less). If the answer appears to land but the agent does not resume: on the rk path the send's delivery verification is built in — a probe failure surfaces as staged text + a stderr warning + exit 1, so re-capture and decide; never blind-resend. On the rk-absent raw path, apply the delivery probe (`_cli-agents.md` § Delivery Probe) instead of re-sending blind.

### Idle Auto-Default on Strategic Escalations

- **Timer:** only the left-open Strategic/no-default row starts a hardcoded 30m real-time timer at its log line; no state-file field, override, or env setting exposes it. The background timer never blocks ticks and fires on the first later tick crossing the threshold.
- **Reset:** any terminal display change — agent output, user keystroke, or redraw — resets the pane-idle clock; §4 supplies sub-minute observation.
- **Answer/scope:** send a visibly stated default (`(default: 2)`, `Press enter for 2`, `[2]`), else `1`. Auto-picked Strategic prompts and rule-6 cannot-determine escalations are hard-excluded regardless of idle duration.

### Logging

- Auto-answer (routine): `"{change}: auto-answered '{summary}' → {answer}"`
- Auto-pick strategic (defensible recommendation): `"{change}: auto-picked strategic '{summary}' → {answer} · notified"`
- Left-open strategic (no defensible default): `"{change}: strategic '{summary}' left open · notified. Please respond."`
- Escalation (rule 6 — cannot determine keystrokes): `"{change}: can't determine answer for '{summary}'. Please respond."`
- Auto-default (after 30m idle on a left-open strategic prompt): `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"`
- Notification send failure (fail-silent — logged, loop continues): `"{change}: notify failed ({channel}). Continuing."`

---

## 6. Coordination Patterns

The operator understands the full fab pipeline and command vocabulary. It infers the right action from current state rather than following named playbooks.

### Pipeline Reference

```
intake → apply → review → hydrate → ship → review-pr
```

**Setup commands**: `/fab-new` (create + activate change), `/fab-draft` (create without activating), `/fab-switch` (activate existing change), `/git-branch` (align branch)

**Pipeline commands**: `/fab-proceed` (auto-detect state, run the needed prefix steps — `/fab-new`, `/fab-switch`, `/git-branch` — then `/fab-fff`), `/fab-continue` (one stage), `/fab-fff` (full pipeline), `/fab-ff` (fast-forward to hydrate), `/git-pr` (commit, push, create PR)

**Maintenance**: rebase onto `origin/{default_branch}` (resolved per Dependency Resolution step 0), merge PR (`gh pr merge`), `/fab-archive`

### Spawning an Agent

Every spawn flow is **repo-targeted and session-targeted**: the operator first establishes **which repo** the work targets (the existing change's repo, the `target_repo` of a watch, or the repo the user names) and **which tmux session** the new agent window must land in, then runs every step against those — not against the operator's own repo or its ambient session.

The spawn sequence is:

1. **Establish target repo** — determine the absolute main-worktree root the work targets. For an already-tracked change, use its `repo` (monitored entry or `branch_map`). For a watch spawn, use the watch's `target_repo` (§7). For a fresh user request, use the repo the user names (default: the repo the operator was launched in).
2. **Establish target session** — determine the tmux session the new agent window must land in, via the evidence-ordered inference below; it is passed explicitly at step 7. The operator MUST pass `-t '<session>:'` (shell-escaped — step 7 owns the escaping rule) on every `new-window` — **the ambient session is never an implicit target** (the operator may run in its own dedicated session, where an untargeted `new-window` silently misplaces the window; the exact mirror of step 3's "never rely on the operator's CWD"):

   **Exclusion rule (applied before any evidence is weighed)** — `_rk-*`-prefixed sessions and the operator's own session MUST NOT be spawn targets. The `_rk-*` prefix is run-kit's reserved-infrastructure namespace (`_rk-ctl` the control anchor, `_rk-pin-*` board pin-sessions, `_rk-operator` the operator session); fab adopts the prefix as a naming convention the way it consumes `@rk_pane_agent_state`. This strengthens the ambient-session prohibition above — the operator's own session is excluded by rule, not merely never defaulted to.

   Within the remaining candidate sessions, decide from the evidence already at hand, strongest first:

   a. **Monitored agents for the target repo (strongest)** — the session holding existing monitored agents for the target repo, re-verified from the current tick snapshot / `fab pane map --all-sessions` (§1 Re-derive state). When that repo's monitored agents span **multiple** sessions, the session holding the most of them wins; break ties with the most recently enrolled entry's session. When none match the target repo, the session holding any monitored agent decides only if exactly one candidate session results. Never trust the persisted `session` field alone — it is context, not identity (§1, §4).
   b. **Pane-map repo affinity** — the session holding panes whose worktrees belong to the target repo, read from the same snapshot; no monitoring required, so this is what decides a cold start (the monitored set is always empty there). Same majority rule as (a); a tie no other signal separates falls to the genuinely-torn path below (pane-map evidence carries no enrollment recency to break ties with).
   c. **The §8 setting** — the "Spawn target session" setting, when set (by the user, or auto-set by an earlier announced inference this session).
   d. **Structural dominance** — exactly one plausible candidate session remains after the exclusion rule. Attached state and window count are supporting signals for the announcement at any tier, never silent deciders on their own.

   **Default-and-announce.** An evidence-backed decision at any tier proceeds without asking — a wrong landing is cheap to correct (`move-window`; step 7's `-P -F` print confirms where the window landed). Announce the chosen session and its deciding evidence in the spawn output, and auto-set it as the §8 "Spawn target session" setting so later spawns stay consistent; the user overrides at any time with "spawn into session {name}".

   **Genuinely torn — the only ask.** When the tiers produce no decision (zero candidates, or two-plus plausible candidates with no repo affinity separating them): attended, ask the user once and keep the answer as the §8 setting for the rest of the operator session; on an **unattended** spawn (a watch or autopilot tick), escalate via the §5 notification path instead. Never decide without evidence, never fall back to ambient — an evidence-backed inference is a derivation, not a guess.
3. **Create worktree** — run the repo-targeted, probe-and-route procedure in `_cli-external.md` § wt; never rely on the operator's CWD
4. **Activate the change pointer (existence-guarded)** — in the **just-created worktree's directory**, set that worktree's own `.fab-status.yaml` so the worktree is self-describing after the pipeline completes (a bare `fab`/`/fab-*` later resolves the change without naming it). Run the switch **only when the change folder already exists** — `fab resolve --folder <change>` succeeds iff a non-archived change folder matches:

   ```sh
   # In the newly created worktree directory, only when the change already exists.
   # `fab resolve --folder <change>` succeeds iff a non-archived change folder matches.
   if fab resolve --folder "<change>" >/dev/null 2>&1; then
     # Fail-soft: swallow a switch failure and log one line, so a set -e context
     # does not abort the spawn (the pointer write is an ergonomic enhancement).
     fab change switch "<change>" \
       || echo "<change>: pointer activation failed (fab change switch); continuing." >&2
   fi
   ```

   **Guard:** switch only an already-existing change, from the just-created worktree CWD, and fail soft. Raw/backlog forms wait for `/fab-new` Step 10; the dedicated worktree owns its own pointer, and the embedded transient override preserves correctness if activation fails.
5. **Resolve dependencies** — if the change has a non-empty `depends_on` list, resolve it per repo: same-repo deps cherry-pick into the worktree, cross-repo deps are ordering-only barriers (see Dependency Resolution below)
6. **Read the target repo's session command** — compose it per `_cli-agents.md` § Spawn Composition, in the **role-addressed** form with the target repo named: `fab agent --print --repo <target-repo>`. The operator-specific rule: **always pass `--repo <target-repo>`** — do NOT use the operator's own `config.yaml`, since each repo may configure a different provider/session command. (The provider-addressed form documented there is for ad-hoc cross-provider sessions, not operator worker spawns, which must carry the target repo's `default`-role profile.)
7. **Open agent tab** — open the composed command per `_cli-agents.md` § Spawn Composition ("Open it in a pane", incl. the one-prompt/no-`&&`-chaining rule), targeted at step 2's session and with the operator's window-marker name. The command carries the interactive shell fallback owned there (`; exec "$SHELL"` — the why is the owner's, not restated here):

   ```sh
   tmux new-window -t '<session>:' -P -F '#{session_name} #{pane_id}' -n "»<wt>" -c <worktree-path> "<spawn_cmd> '<command>'; exec \"\$SHELL\""
   ```

   (where `<session>` is the target session from step 2, `<wt>` is the worktree name from step 3, and `<spawn_cmd>` is the target repo's command from step 6). **Shell-escape the session name before embedding it** — it can come from the natural-language §8 setting or an arbitrary tmux session name, so raw interpolation inside double quotes would let an embedded `$()`/backtick execute; the single-quoted `-t` keeps such text literal, and a name containing a single quote must itself be escaped, never interpolated raw. `-P -F` prints the landed `#{session_name}` and `#{pane_id}` — step 8's enrollment consumes both, and the printed session confirms where the window actually landed. A missing `-t` target errors loudly at spawn (tmux refuses an absent session); surface it per normal error handling — never silently retry against the ambient session.
8. **Enroll in monitored set** — unconditionally and silently via `fab operator enroll`, passing step 7's printed values as `--pane <pane-id> --session <session-name>` (plus repo, stage, branch, and dependencies — the `branch_map` pair rides the same command; contract in `_cli-fab.md` § fab operator); then apply §4 Enrollment's window prefix; never ask whether to monitor

Window markers (`»` / `›`) key on server-global pane IDs.

### Dependency Resolution

**Dependency satisfied.** A `depends_on` entry is satisfied when the dependency's **pipeline has completed** — its monitored entry has emitted (or would emit) a `completion` delta: `review-pr` done/skipped when its `stop_stage` is null, or at/past its `stop_stage` — **and**, for a same-repo dependency with a null `stop_stage`, its PR exists (`gh pr view <dep-branch> --json url` succeeds, so the branch is pushed and stable). Neither enrollment, a `branch_map` entry, nor the branch being minted is satisfaction — all three exist from the moment the dep's agent spawns. An unsatisfied dependency **holds the spawn** in both tiers, re-checked on each tick, logging `"{change}: waiting on dependency {dep} ({dep.repo}) to complete."`. Every consumer below (both tiers, the autopilot loop, watches) gates on this definition.

Dependency resolution is **two-tier**, split by repo. Each entry in `depends_on` is classified by comparing the dependency's `repo` (from its `branch_map` `{ branch, repo }` pair, or the dep's monitored entry) against **this change's** `repo`:

- **Same-repo dependency** (`dep.repo == change.repo`) → **cherry-pick** the dependency's code into the worktree, exactly as today. **In the `stacked-prs` merge mode the same-repo strategy changes** — the dependent's branch is created off the dependency's branch (no cherry-pick commit); see the `stacked-prs` note under Same-repo resolution below.
- **Cross-repo dependency** (`dep.repo != change.repo`) → **ordering-only barrier** in every mode: the operator waits until the dependency is satisfied per **Dependency satisfied** above, then spawns the dependent agent. **No code is merged.**

> **REQUIRED caveat — cross-repo deps give the dependent agent NO code.** An ordering-only cross-repo dependency is a pure *sequencing* constraint: the dependent worktree receives nothing from the dependency. This is correct only for **logical** dependencies (e.g., "don't start the frontend change until the API change merges to its repo's main"), never for **code-level** dependencies. Cross-repo branches share no common default-branch base to cherry-pick across, so there is no sound way to make the dependency's code available — do not expect cross-repo `depends_on` to do so. For code sharing across repos, the dependency must merge and be consumed as a normal upstream artifact (package, vendored copy), outside the operator's scope.

**Same-repo resolution.** For the same-repo subset of `depends_on`, before opening the agent tab:

0. **Fetch and resolve the base** — in the target worktree, refresh the remote and resolve the repo's **actual default branch** (never assume `main`):

   ```bash
   git fetch origin
   default_branch=$(git symbolic-ref --short refs/remotes/origin/HEAD 2>/dev/null | sed 's|^origin/||')
   [ -n "$default_branch" ] || default_branch=$(gh repo view --json defaultBranchRef -q .defaultBranchRef.name 2>/dev/null)
   # Literal fallback when both commands fail: probe the just-fetched refs — main when origin/main exists, else master
   [ -n "$default_branch" ] || default_branch=$(git rev-parse --verify -q origin/main >/dev/null && echo main || echo master)
   ```

   `origin/{default_branch}` is the cherry-pick base in step 3 below. Fetching first prevents a stale base even on correctly-defaulted repos; resolving the name makes autopilot usable on repos whose default branch isn't `main`.

0.5. **Readiness gate** — for each same-repo change ID still in the monitored set, check it is satisfied per **Dependency satisfied** above. If any is not, hold the spawn (no branch lookup, no cherry-pick) and let the loop re-check on subsequent ticks. A dep that has left the monitored set (present only in `branch_map`) was removed on its own `completion` and passes this gate. The `stacked-prs` variant below inherits this gate.

1. **Resolve same-repo dependency branches** — For each same-repo change ID, look up its branch:
   - First from the monitored entry's `branch` field (if the dep is still active).
   - Otherwise from `branch_map` (the `{ branch, repo }` pair, if the dep has left the monitored set).

   Build a mapping `dep_change_id -> dep_branch` for the same-repo subset. If any same-repo dependency branch is not found in either location: log `"{change}: dependency {dep} branch not found. Escalating."`, escalate to the user, and do **not** spawn the agent.

2. **Prune redundant deps across the same-repo subset** — Using the resolved `dep_change_id -> dep_branch` mapping, remove dependencies whose branches are ancestors of other same-repo dependency branches:
   - If dep A's branch is an ancestor of dep B's branch (both same-repo deps in `depends_on`), drop A from the effective dependency set.
   - Check via: `git merge-base --is-ancestor <A-branch> <B-branch>`.

   Pruning is scoped to the **same-repo subset only** — `git merge-base --is-ancestor` is meaningless across repos with no shared history. It runs *across that subset* before any cherry-picks, to prevent duplicate cherry-picks in chains where B's branch already carries A's content transitively.

3. **For each remaining (pruned) same-repo dependency**, in the target worktree:

   a. **Check if already present** — run:
      ```bash
      git merge-base --is-ancestor <dep-branch> HEAD
      ```
      If the dep branch is already an ancestor of `HEAD`, skip this dependency's cherry-pick.

   b. **Cherry-pick** — if not already present, in the worktree directory (using the `{default_branch}` resolved in step 0):
      ```bash
      git cherry-pick --no-commit origin/{default_branch}..<dep-branch> && \
      git commit -m "operator: cherry-pick <dep-change> dependency"
      ```
      This cherry-picks all commits unique to the dependency branch since it diverged from `origin/{default_branch}`, stages them without individual commits, and squashes into a single operator commit.

   c. **On conflict** — abort immediately, do not spawn:
      ```bash
      git cherry-pick --abort
      ```
      Log: `"{change}: cherry-pick conflict with dependency {dep-change}. Escalating."`
      Escalate to user. Do not proceed without the dependency content. Bounded retry: 0 (§3).

**Cross-repo resolution.** For each cross-repo dependency, do not cherry-pick. Instead, before spawning, verify the dependency is satisfied per **Dependency satisfied** above. If it is not, hold the spawn and let the loop re-check on subsequent ticks; spawn once every cross-repo barrier clears, logging the wait with the shared line from that definition.

**Same-repo resolution (`stacked-prs` mode).** Steps 1–3 are skipped for same-repo dependencies — the dependent's branch is created off its nearest same-repo predecessor's *branch* at the §6 spawn sequence's worktree/branch step instead of off `origin/{default_branch}` (the probe-and-route per `_cli-external.md` § wt: existing dep branch → `wt create --checkout <dep-branch>` route). The squashed `"operator: cherry-pick"` commit does not exist for same-repo deps in this mode. After `/git-pr` creates the dependent's PR, the operator retargets its base to the dependency's branch: `gh pr edit <pr> --base <dep-branch>` (`/git-pr` itself is unchanged and mode-unaware). The merge-all choreography for the stack lives under Ordered Merge below. Dependency-branch drift after a dependent PR exists (a dep's review-pr rework moving its branch) is out of scope — the same exposure exists in the cherry-pick model; conflicts surface at merge-all and escalate.

**Why `origin/{default_branch}` as base (same-repo only)**: Each same-repo dependency branch carries its full transitive same-repo dependency content. When the operator spawned dep B, it cherry-picked dep A into B's worktree first. B's branch therefore contains A's commits. So `origin/{default_branch}..<B-branch>` gives the complete transitive closure within the repo — no need to chase transitive same-repo deps manually. This is why only direct/leaf same-repo dependencies need cherry-picking. (Cross-repo deps carry no such transitive content — they are ordering-only.)

### Dependency Declaration

Dependencies are declared through three conversational paths, all of which coexist:

1. **Explicit**: "cd34 depends on ab12" — operator records it through enrollment: `fab operator enroll cd34 … --depends-on ab12` (at spawn this is step 8; mid-flight it re-enrolls, which replaces the entry wholesale — carry the current stage/agent along)
2. **Autopilot queue (implicit)**: resolve ordering per § Autopilot → Queue ordering
3. **`--base` flag (explicit)**: autopilot `--base <prev-change>` explicitly sets `depends_on: [<prev-change-id>]` for the subsequent change (matches path 2's pick when the previous entry is same-repo; available for ad-hoc overrides)

### Working a Change

> **Pipeline-first routing (§1):** all three work paths below MUST go through the fab pipeline (`/fab-new` then a pipeline command for new work; the appropriate stage for already-intaked changes) — never raw implementation instructions to agent panes.

Every form runs §6's target-repo + target-session → worktree → guarded activation → dependencies → target-repo session command → tab → enrollment sequence:

1. **Existing change:** use the monitored/`branch_map` repo and embed `/fab-fff <change>` as the single prompt per `_cli-agents.md` § Spawn Composition; the transient override targets the pipeline and spawn step 4 activates the pointer.
2. **Raw text** (for example, "fix login after password reset"): use the named repo (default operator launch repo) and embed `/fab-new <shell_escaped_description>`. Shell-escape the raw description; never insert it unescaped. The existence guard skips activation until `/fab-new` creates and activates the change at Step 10.
3. **Backlog ID or Linear issue:** resolve it first (optional `idea` lookup per `_cli-external.md` § Delegation and binary gate), then embed `/fab-new <id>`. The existence guard skips activation and `/fab-new` owns it.

On completion (all three): PR ready, optionally archive. Both raw text and backlog paths use `/fab-new` to generate a proper intake with traceability. `/fab-new` captures the raw input in the intake's Origin section — the user just says "fix [description]" and the operator does the rest.

### Autopilot

User provides a queue of changes. Confirmation prompt reflects the active mode:
- **Default (`cherry-pick-ladder`):** "Confirm upfront (creates PRs — merge after review)."
- **`merge-auto`:** "Confirm upfront (merges PRs on completion)."
- **`stacked-prs`:** "Confirm upfront (creates stacked PRs — merge after review)."

A queue **may span repos**, with mixed dependency semantics: implicit `--base` chaining (and explicit `depends_on`) cherry-picks **within a repo** and **degrades to an ordering-only barrier across repo boundaries** (per Dependency Resolution above; the nearest-same-repo-predecessor rule is defined in Queue ordering below). Worked example — a chain `ab12 → cd34 → ef56` where `cd34` lives in a different repo: `cd34` gets `depends_on: [ab12]` (cross-repo — waits for `ab12` to be satisfied per § Dependency Resolution **Dependency satisfied**, no cherry-pick), and `ef56` (back in `ab12`'s repo) gets `depends_on: [ab12]` — its nearest same-repo predecessor — and cherry-picks from it; queue order still runs `ef56` after `cd34`.

Once the user confirms, persist the queue via `fab operator autopilot start --queue <id,id,...> [--mode <name>]` (the binary stores the mode and prints `mode: <name> (<source>)`; contracts in `_cli-fab.md` § fab operator autopilot); every later progression (completion or skip) is `fab operator autopilot advance [--skip]`, and the interrupts below ride `pause`/`resume`/`stop`.

Queue ordering:

| Strategy | Description |
|----------|-------------|
| User-provided | Run in the exact order given. Implicit `--base` chaining by default: every change after the first gets `depends_on: [<nearest-same-repo-predecessor>]` — the closest earlier queue entry in the same repo (cherry-picked); when no earlier entry shares the repo, the immediately previous entry (cross-repo → ordering-only). No explicit `--base` flag required. |
| Confidence-based | Sort by confidence score descending. Highest-confidence first (independent changes) |
| Hybrid | User provides constraints (partial order); operator sorts unconstrained by confidence |

**Merge modes** — three flat names. **Mode resolution (silent by default):** when the user's queue request names no mode — explicitly or via natural language — resolve it by the ladder explicit user instruction / `--mode` flag > config `autopilot.merge_mode` > built-in `cherry-pick-ladder` and proceed WITHOUT asking. `fab operator autopilot start` prints `mode: <name> (<source>)` where source is `flag` / `config` / `default` — that line is how the operator learns the resolved mode and its source (the operator never parses config files itself). State the resolved mode inside the **existing** upfront queue-confirmation line above (which already varies by mode), so the user vetoes in the same breath — no extra round-trip.

Pause and ask the mode question ONLY on one of exactly two misfits:

1. The resolved mode is `merge-auto` but the queue has same-repo `depends_on` entries — implicit chaining is disabled in that mode, so the declared dependency semantics contradict it.
2. The user's own message conflicts with the resolved mode (e.g. they say "merge as you go" while the resolved mode is a held mode like `cherry-pick-ladder` or `stacked-prs`).

No other condition triggers the mode question. When the operator DOES ask, the question MUST include the at-a-glance glyphs, the three compact box diagrams below, and a one-line tradeoff per mode (an invalid config value is the binary's own actionable `start` error, not a misfit — it never reaches a question).

At a glance: `▂▄▆` cherry-pick-ladder · `░▒▓█` merge-auto · `▄▀` stacked-prs. (The diagrams below are skill documentation — never emit them into the **status frame**, which stays fence-free per §4. That prohibition is status-frame-only: a mode question is an ordinary conversational message, where the fenced diagrams render fine and are REQUIRED.)

- **`cherry-pick-ladder`** (default) — PRs are created but not merged until the user explicitly requests merging; implicit `--base` chaining is active (per Queue ordering, "User-provided").

  ```
                      ┌───┐
              ┌───┐   │ C │
      ┌───┐   │ B │   ├╌╌╌┤
      │ A │   ├╌╌╌┤   │ b'│
      │   │   │ a'│   │ a'│
  ────┴───┴───┴───┴───┴───┴──▶ main
       PR1     PR2     PR3
  ```

  Every PR stands on main; each successive diff is taller because it carries cherry-picked copies of its predecessors (`a'`, `b'`) below the dotted line. All PRs held; merged base-first on "merge all".

- **`merge-auto`** — merge-as-you-go: **arm** each PR on completion (§6 Auto-Merge Choreography — a one-PR sequence position; all five rules apply) instead of merging and foreground CI-waiting; once the merge is verified on a later tick, `git fetch origin` and rebase the next change onto `origin/{default_branch}` (the default branch resolved per Dependency Resolution step 0 — never a hardcoded `origin/main`). Implicit `--base` chaining is disabled in this mode — each change rebases onto `origin/{default_branch}` independently. Natural language equivalents: "merge as you go", "merge on complete", "merge each when done".

  ```
      ┌───┐          ┌───┐          ┌───┐
      │ A │          │ B │          │ C │
  ────┴─▼─┴●─────────┴─▼─┴●─────────┴─▼─┴●──▶ main
         merged         merged         merged
  ```

  Nothing coexists and nothing is held: the operator arms each PR the moment it lands and GitHub merges it into main when checks pass (▼ into ●), main advances, and the next change starts from the advanced line — no batch review, no re-stacking.

- **`stacked-prs`** — `cherry-pick-ladder` merge timing (PRs created up front, merged only on explicit user request) with true stacked-PR topology for same-repo chains: the dependent's branch is created off its dependency's *branch* (no cherry-pick commit) and its PR targets the dependency's branch, so each PR diff shows only its own delta. Mechanics: same-repo resolution in Dependency Resolution below; merge-all choreography in Ordered Merge. Natural language equivalents: "stacked PRs", "stack the PRs".

  ```
                      ┌───┐
              ┌───┐   │ C │  PR3 · base: B
      ┌───┐   │ B │   └───┘
      │ A │   └───┘  PR2 · base: A
  ────┴───┴──────────────────▶ main
      PR1 · base: main
  ```

  Uniform height: every diff shows only its own delta. The diagonal is load-bearing — each PR's base is the previous PR's branch, so merging a bottom box means re-seating the ones above it (the Ordered Merge retarget + rebase steps).

The operator works each change through the pipeline. Pre-send validation (§3) applies to any command sent to an existing pane; the initial pipeline command itself is **embedded at spawn** (§6 step 7) — the single dispatch point:

1. **Gate** — check confidence score **before anything spawns**. If below threshold, flag and wait — no worktree, no tab, no dispatch for a below-threshold change
2. **Spawn** — run the §6 spawn sequence steps 1–3 (establish the change's target repo and target session, create worktree in the repo; `--reuse` for respawns)
3. **Resolve dependencies + open tab + enroll** — §6 spawn sequence steps 4–8 (existence-guarded pointer activation, same-repo cherry-pick / cross-repo ordering-only barriers per Dependency Resolution). Step 7's `<command>` is the change's pipeline command — `/fab-fff <change>` (or the appropriate command for its current stage) — so the dispatch happens **once, at spawn**; do NOT send the command again after the tab opens
4. **Monitor** — normal tick detection handles progress
5. **Record** — when the current change is satisfied per § Dependency Resolution **Dependency satisfied** (its `completion` delta observed **and** its PR URL collected), run `fab operator autopilot advance` (the binary moves `current` to `completed` and promotes the next entry) and collect the PR URL. The `{ branch, repo }` pair is already in `branch_map` — `enroll` recorded it at spawn
6. **Spawn next** — only after item 5's satisfaction check; repeat from item 1 using § Queue ordering and § Dependency Resolution; embed its command at spawn
7. **Report** — `"ab12: PR ready. 1 of 3 complete. Starting cd34."`
8. **(After all complete) Summary** — list all PR links with per-repo dependency annotations and per-repo merge order suggestion (see Queue Completion Summary below)

In `merge-auto` mode, steps 5–8 arm the just-shipped PR (§6 Auto-Merge Choreography) instead of foreground CI-waiting, and **defer the autopilot `advance` and the next change's spawn to the tick that verifies the merge** — spawning earlier would start the next agent from a stale `origin/{default_branch}`. On the verified-merge tick: run `git fetch origin`, rebase the next change onto `origin/{default_branch}` (resolved per Dependency Resolution step 0), report the merge, and spawn.

Autopilot-driven changes display `▶` in the status frame (§4). Queue progress is visible from the list — entries with `▶` and health `✅` are complete; the current entry shows health `🟢` while active or `🟡` while waiting.

#### Queue Completion Summary

When all changes in a `cherry-pick-ladder` or `stacked-prs` autopilot queue complete, the operator displays a completion summary. When the queue spans repos, each PR is **annotated with its repo**, and the suggested merge order respects **each repo's own dependency chain** (a per-repo PR sequence):

```
Queue complete. 3 PRs ready for review:
1. ab12: <PR-URL-1> (~/code/foo, base)
2. cd34: <PR-URL-2> (~/code/bar, ordering-only after ab12)
3. ef56: <PR-URL-3> (~/code/foo, depends on ab12)
Merge per-repo: foo 1→3, bar 2 (after foo:1 reaches main). Or ask me to merge all.
```

For a single-item queue: `"ab12: PR ready. Queue complete."`

#### Ordered Merge

When the user says "merge all" or "merge the queue" after a `cherry-pick-ladder` or `stacked-prs` queue completes, the operator merges PRs respecting **per-repo PR sequences** — within each repo, base-first in dependency order; across repos, cross-repo ordering barriers are honored (a cross-repo dependent's PR is merged only after its barrier dependency reaches its target repo's main). **The CI gate between merges depends on the mode**: in `cherry-pick-ladder`, merge-all runs the **Auto-Merge Choreography** below — arm each PR via GitHub auto-merge and verify the merge on later ticks, a passive tick check instead of a foreground wait. In `stacked-prs` — or whenever arming is unavailable (Auto-Merge Choreography rule 2) — the operator merges each PR itself and foreground-waits for CI to pass before proceeding to the next in that repo's sequence:

1. Merge `~/code/foo` PR 1 (base) — CI gate per mode (arm + tick-verify, or foreground CI wait)
2. Merge `~/code/bar` PR 2 (its cross-repo barrier `foo:1` is now on main) — CI gate per mode
3. Merge `~/code/foo` PR 3 — CI gate per mode

Report each merge with its repo: `"ab12: merged (foo 1/2)"`, `"cd34: merged (bar 1/1)"`, `"ef56: merged (foo 2/2)"`.

**`stacked-prs` merge-all adds two steps per merge**, because each PR in a same-repo chain is based on its dependency's branch:

1. **Verify base retarget** — after a chain's base PR merges, GitHub auto-retargets the dependent PR's base onto the default branch when the merged base branch is deleted. Rely on this, and retarget explicitly (`gh pr edit <pr> --base {default_branch}` — a plain branch name, never a remote ref) when the branch was not deleted.
2. **Rebase the next branch after a squash merge** — after a squash merge, the next branch in the chain still carries the dependency's original commits, which the default branch now contains only as a squashed commit. Before that next PR is clean/mergeable, rebase it onto the default branch, dropping the already-merged dependency commits, and force-push:

   ```bash
   git fetch origin && git rebase --onto origin/{default_branch} <merged-dep-branch> <next-branch> && git push --force-with-lease
   ```

   `{default_branch}` is resolved per Dependency Resolution step 0 — never a hardcoded `origin/main`. A conflict in this rebase **halts and escalates** (never silently skips), consistent with the cherry-pick-conflict policy.

**CI failure during ordered merge (halt-dependents-only)**: If CI fails on a PR, the operator halts **that repo's merge sub-sequence** AND **any repo whose queued items carry a cross-repo `depends_on` into the failed chain — transitively**. In an armed sequence, the halt first disarms the halted sequences' remaining armed PRs (Auto-Merge Choreography rule 5); independent sub-sequences keep theirs and continue. "Dependent" is determined over the cross-repo `depends_on` graph: a repo halts if any of its queued items depends (directly, or via another already-halted item) on a PR in the failed chain. **Truly independent repos' sub-sequences continue merging.** The operator does not abandon the queue; it isolates the blast radius to the failure's dependency cone. On completion it reports which sub-sequences halted vs. completed and escalates the failure to the user:

```
ab12: CI failed (~/code/foo). Halted: foo sub-sequence; bar (cross-repo dep into foo). Completed: baz sub-sequence (2 PRs merged). Fix foo and retry.
```

Autopilot state (queue, current, completed, mode) persists in the operator state file — written by the `fab operator autopilot` verbs, never hand-edited; on queue exhaustion the binary retains `queue`/`completed`/`mode` with `current: null, state: null` so the summary below can still read them, and `fab operator autopilot stop` clears the block after the summary renders.

**Failures**: review exhausted → skip. Rebase conflict mid-queue → skip (`merge-auto` only; does not apply in `cherry-pick-ladder` since there are no rebase steps). Rebase conflict during a `stacked-prs` merge-all → escalate (never skip). Cherry-pick conflict → escalate (do not skip). Pane dies → 1 respawn (`--reuse`), then skip. Stage timeout (>30m) → flag. Total timeout (>2h) → flag.

**Interrupts**: "stop after current", "skip <change>", "pause", "resume" — acknowledged immediately, and persisted through the matching verb: `fab operator autopilot stop` once the current change lands (or immediately to abandon the queue), `advance --skip` (drop current without recording it completed), `pause`, `resume`.

#### Auto-Merge Choreography

The CI gate for `cherry-pick-ladder` merge-all (and `merge-auto`'s per-PR merge on completion) — the modes where PRs target main. Instead of merging and foreground-waiting for CI — the longest operator-busy stretches in the coordination lifecycle — the operator **arms** each PR with GitHub auto-merge (`gh pr merge --auto --squash` — the method flag is explicit and REQUIRED: flagless `--auto` may prompt or take the repo's default, unsafe on an unattended tick; a user-directed method maps to `--merge`/`--rebase`) and lets GitHub merge it when checks pass, verifying on later ticks. Arming is part of the user's confirmed merge-all — the "merge all" confirmation is the §3 Destructive-tier confirm for the whole sequence, so no per-PR re-confirmation is asked.

**`stacked-prs` is excluded and keeps the manual merge-all above**: its inter-merge choreography (retarget-verify, `rebase --onto`, force-push) is operator-sequenced anyway, and an armed stacked PR can merge into its dependency's *branch* (destroying the stack silently) or fire on stale-green checks after GitHub's no-re-CI base retarget.

All five rules are MUSTs:

1. **Sequential arming.** At most one armed PR per repo-sequence. Arm PR_n only after PR_{n-1}'s merge is **verified** — a merge event on the PR's timeline, never an assumption. Never arm a PR whose base is another PR's branch.
2. **Arming-failure shapes.** A draft PR MUST be readied first with `gh pr ready` (fab's `/git-pr` creates drafts, so this is every autopilot PR). An "already clean" rejection (the repo has no required checks, so auto-merge has nothing to wait for) → merge directly. Auto-merge disabled on the repo → fall back to the foreground CI-wait choreography above for the sequence.
3. **Stall rule.** The per-tick check on a still-unmerged armed PR inspects two things. A **failed required check** (`gh pr checks`) is Ordered Merge's CI failure — disarm per rule 5 and apply the halt-dependents-only policy (a failed check never makes the PR `CONFLICTING`; auto-merge just silently never fires). Unmerged after 3 consecutive ticks with no failed check → check `gh pr view --json mergeable` (the field is `mergeable` — `gh pr view --json` exposes no `mergeableState` field); `CONFLICTING` → disarm and escalate. Both shapes are event-less — the tick MUST poll.
4. **Persisted sequence.** Starting a merge sequence MUST write a `kind: coordination` note (`fab operator note add --kind coordination`) recording, **per repo-sequence**, the sequence, current position, and armed PR (a multi-repo merge-all runs one armed PR per repo-sequence — rule 1 — so the note's prose carries one line per sequence) — an armed PR **outlives the operator** (it survives compaction, `/clear`, crash, and abandonment), so the sequence must not live only in conversation. Update the note as the sequence advances (`fab operator note update`) and resolve it at sequence end (`fab operator note resolve`). A restarted operator re-orients from the note (§2 Init) and resumes verification/arming.
5. **Disarm on halt.** Any halt or escalation — CI failure, stall, conflict — MUST run `gh pr merge --disable-auto` on the remaining armed PRs of the **halted sequences**: the failing repo's sub-sequence plus its transitive cross-repo dependent cone, matching the halt-dependents-only policy (which assumes unstarted merges stay unstarted — armed auto-merge violates that without the disarm). Independent sub-sequences keep their armed PRs and continue. A user "stop" is global and disarms every armed PR.

**Per tick while a merge sequence is in progress** (an open merge-sequence `coordination` note): check **each** armed PR (one per repo-sequence) — merged (timeline event) → report, advance that sequence's position in the note, and arm its next PR per rule 1 (readying a draft per rule 2); unmerged → run rule 3's checks and count toward its stall threshold. This check rides the normal tick (§4 Tick Behavior step 4), so merge-all consumes no foreground attention between arms — and a merge sequence in progress is by itself a loop run-condition (§4): at merge-all time the autopilot queue is exhausted and the monitored set is typically empty, so without this condition the loop might not even be running to do the tick-verify work.

---

## 7. Watches

Watches are standing instructions to monitor an external source and take action when new items appear. Users create watches conversationally: "watch Linear project DEV for new issues, spawn agents, stop at intake."

### Schema

Each watch in the operator state file has the fields below (reference documentation of what the binary maintains — watches are created and mutated only through the `fab operator watch` verbs; contracts in `_cli-fab.md` § fab operator watch):

| Field | Description |
|-------|-------------|
| `enabled` | `true` or `false` — paused watches retain config but skip tick evaluation |
| `source` | `linear` or `slack` — determines which MCP tool to query |
| `query` | Source-specific API filter (project, status, assignee, channel) — passed to MCP |
| `target_repo` | Absolute main-worktree root the watch's spawned changes land in. Required for a spawning watch — the spawn sequence (§6) uses it as the target repo. A watch with no `target_repo` cannot spawn |
| `stop_stage` | How far to go: `intake`, `apply`, `hydrate`, or `null` (full pipeline) |
| `known` | Already-handled item IDs — appended by `fab operator watch seen`; the binary enforces the 200-entry cap (oldest pruned first) |
| `completed` | Items that reached `stop_stage` — lets users query "what did this watch produce?" |
| `last_checked` | ISO timestamp of last successful query |
| `last_error` | Last error message, or `null`. Shown in status frame when set |
| `instructions` | Free-form natural language — trigger conditions, concurrency limits, label filters, anything else |

Structured fields handle machine-readable concerns; `instructions` handles everything the operator evaluates as an LLM. Concurrency limits in `instructions` are enforced by counting monitored entries where `spawned_by` matches the watch name.

### Tick Behavior

On each tick (step 3), for each enabled watch:

1. **Query source** — Linear via MCP (`mcp__claude_ai_Linear__list_issues`), Slack via MCP (`mcp__claude_ai_Slack__slack_read_channel`), using `query` as the API filter. On failure: `fab operator watch checked <name> --error "<msg>"`, skip this watch for this tick. After 3 consecutive failures: `fab operator watch toggle <name> --off`, alert user.
2. **Deduplicate** — skip items in `known` **plus** `completed` lists (an item that reached `stop_stage` moves from `known` to `completed` but may still match the query — it MUST NOT be respawned). On success: `fab operator watch checked <name>` (sets `last_checked`, clears `last_error`).
3. **Evaluate instructions** — apply trigger conditions, label filters, concurrency limits (count monitored entries with `spawned_by: <watch-name>`), and any other criteria from `instructions`
4. **Act** — for each item that passes:
   - Run the §6 spawn sequence with the watch's `target_repo` as the target repo, sending the appropriate initial command (e.g., `/fab-new DEV-123`)
   - Enroll via `fab operator enroll` with `repo` (= `target_repo`), `session`, `stop_stage`, and `spawned_by` from the watch
   - `fab operator watch seen <name> <item-id>` (only after successful spawn — the binary appends idempotently and enforces the 200-cap)
5. **Report** — `"Watch linear-bugs: DEV-1024 — Fix auth redirect (72m old). Spawning."`

When a watch-spawned agent completes (its `completion` delta — at/past the watch's `stop_stage`, or `review-pr` done/skipped when it is null), `fab operator watch complete <name> <item-id>` (moves the item from `known` to `completed`) and report: `"Watch linear-bugs: DEV-1024 completed intake."`

### Conversational Management

Every utterance maps to a `fab operator watch` verb (contracts in `_cli-fab.md` § fab operator watch) — the operator composes flags, never YAML:

- "Watch Linear project DEV for bugs older than 1 hour, **spawn into ~/code/foo**, stop at intake" → `watch add <name> --source linear --target-repo ~/code/foo --stop-stage intake --query '<json>' --instructions '…'`
- "Pause the Linear watch" / "Resume the Linear watch" → `watch toggle <name> --off` / `--on`
- "Stop watching Linear" → `watch rm <name>`
- "Spawn the Linear watch's changes into ~/code/bar instead" → `watch update <name> --target-repo ~/code/bar`
- "What are you watching?" → read via `fab operator state`; list active watches with their `target_repo`, instructions, and completed items
- "What did linear-bugs produce?" → lists `completed` items (from `fab operator state`)
- "Test watch linear-bugs" → dry-run: query, deduplicate, evaluate instructions, report what *would* happen without spawning and without running any mutation verb
- "Change the Linear watch to go through full pipeline" → `watch update <name> --stop-stage ""` (clears to null)
- "Also limit to 2 concurrent agents" → `watch update <name> --instructions '<merged text>'` (the operator appends to the existing instructions and passes the merged text)

---

## 8. Configuration

### One Operator Per Server

The isolation unit is the **tmux server**. There is exactly **one operator per tmux server** — it spans every session and every repo on that server, coordinating all of them through a single server-keyed state file (§4, §9). This matches the server-wide singleton already enforced by the `operator` window (`fab operator` switches to the existing window rather than creating a second one).

- **Multiple sessions, same server** share one operator and one state file. The operator addresses their agents by the `(session, repo, pane)` tuple (§1) — where `session` scopes the addressing/display, never the identity: the pane ID is the join key, and a session can change mid-lifetime (§1, §4); there is no per-session or per-repo operator.
- **A second operator means a second tmux server** — start one on a separate socket (`tmux -L <label>`). Its state file is keyed by that socket, so the two operators never collide. There is no `--name` dimension; the server boundary is the only isolation knob. Sends on a non-default socket carry the matching flag: `rk mux -L <label> send` (or `tmux -L <label> send-keys …` on the rk-absent raw path).

### Settings

| Setting | Default | Override via natural language |
|---------|---------|------------------------------|
| Loop interval | 3m | "check every {N}m" |
| Stuck threshold | 15m | "flag agents stuck for more than {N}m" |
| Waiting/menu heartbeat | 90s | "tighten to {N}s when an agent is on a menu" |
| Spawn target session | inferred (§6 step 2 evidence tiers; auto-set on each announced inference) | "spawn into session {name}" |
| Notify channel | `rk` (run-kit Web Push; auto-fallback when `rk` absent) | "notify via ntfy topic {topic}" / "notify via discord {url}" / "notify via push" |

These settings are session-scoped and reset on compaction, `/clear`, or session restart (§4 Post-Compaction Reload); they are not operator-state-file fields. The **strategic auto-default threshold is hardcoded at 30m** (§5) — there is deliberately **no** setting for it.

---

## 9. Key Properties

| Property | Value |
|----------|-------|
| Requires active change? | No |
| Runs `fab preflight`? | No |
| Read-only? | No — sends commands, auto-answers, mutates the operator state file (only via `fab operator` subcommands) |
| Idempotent? | Yes — state re-derived every tick |
| Advances stage? | No |
| Outputs `Next:` line? | No — ends with ready signal |
| Loads change artifacts? | No — coordination context only |
| Requires tmux? | Yes — hard stop without it |
| Requires a git repo? | No — `fab operator` opens its window in the repo root inside a repo, else `os.Getwd()` (neutral parent dir). Errors only if both fail |
| Requires a `fab/` project? | No — session command comes from the project's `providers.claude.interactive_command` when `fab/` is resolvable, else `spawn.DefaultSpawnCommand` (the template `claude --permission-mode bypassPermissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}`). No project `providers`/`agent:` block is read on a `fab/`-less launch |
| Coordinating-agent model | Operator role — `fab operator` resolves the `operator` role (`agent.ResolveRole`; a Tier-1 role, so the `agent.session` knob picks its provider), reads that provider's `interactive_command`, injects the profile via `spawn.WithProfile` (**substitutes** into a `{model}`/`{effort}` template — the built-in claude default is templated — or **appends** `--model`/`--effort` to a plain command carrying no placeholder); falls back to the built-in operator profile + built-in claude provider on any failure (incl. no resolvable `fab/` project) |
| Uses `/loop`? | Yes — adaptive heartbeat: `3m` normally, tightens to `90s` (§8) when any monitored agent is `waiting` (`@rk_pane_agent_state`) or menu-waiting (capture fallback), relaxes back to `3m`; one loop at a time; runs while any tracked state remains — monitored set, autopilot queue, watches, or an in-progress merge sequence (§4); quiet ticks render the one-line compact frame (§4 Status Frame Format); loop prompt is the bare `operator tick` — never a slash command (§4 Loop Prompt) |
| Uses the operator state file? | Yes — monitored set + autopilot queue + branch map + notes persistence in the server-keyed path (§2 Init step 1); reads via `fab operator state`, every mutation through a `fab operator` verb — never a hand-write (§4 doctrine) |
| Multi-repo / multi-session? | Yes — one operator per tmux server spans all its sessions and repos via the `(session, repo, pane)` addressing tuple |
