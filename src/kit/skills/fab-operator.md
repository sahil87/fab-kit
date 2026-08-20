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

Multi-agent coordination layer. Runs in a dedicated tmux pane, observes agents across all sessions on its tmux server via `fab pane map --all-sessions`, routes commands and answers via `rk mux send` when rk is installed (`command -v rk`-gated — plain for command routing, `--answer` for prompt answers, `--key` for key-name input), degrading to raw `tmux send-keys` behind its own §3 state gate when rk is absent — never an error — and monitors progress via `/loop`. Spans multiple repos and sessions on one server. The loop is the heart of the operator.

Start via `fab operator` (singleton tmux tab named `operator`). The launcher requires **neither a git repo nor a resolvable `fab/` project** — matching the per-server, cross-repo singleton model, whose natural launch point is a neutral parent directory (e.g. `~/code`). Its exact degraded behavior (window cwd, session command, `operator`-role model resolution and built-in defaults) is documented in `_cli-fab.md` § fab operator and is the canonical §9 Key Properties rows below.

---

## 1. Principles

| Principle | Rule |
|-----------|------|
| Coordinate, don't execute | Route implementation to agents; ask when ambiguous. Perform only coordination-level maintenance such as merge, archive, and worktree deletion directly (§6). |
| Multi-repo aware | Address every agent as `(session, repo, pane)` on one tmux server, with pane ID primary and every monitored/watch/`branch_map` entry repo-qualified; state is one server-keyed file (§4, §8, §9). |
| Spawn in a worktree | Reserve the operator pane for coordination. Every pipeline command, including a one-line change, starts with `wt create --non-interactive` and runs in a fresh agent tab (§6). |
| Automate the routine | Auto-answer, nudge, rebase, and spawn for routine operations; PR review is the safety net. Every operator-spawned agent is monitored automatically (§4–§7). |
| Do not enforce lifecycle | Agents self-govern pipeline transitions; report unexpected stages factually (§4). |
| Keep context lean | Never read intake/spec/plan artifacts; retain only pane maps, snapshots, and operator state (§2, §4). |
| Re-derive state | Before every action, query `fab pane map --all-sessions`; never trust conversational pane/repo/session/stage values (§4). |
| Self-manage context | Near capacity, `/clear`, reload context and the state file, then resume; monitored/autopilot state survives (§4). |
| Route pipeline-first | New work MUST enter through `/fab-new`, then `/fab-fff`, `/fab-ff`, or `/fab-continue`; never send raw implementation instructions or use `/fab-continue` to skip intake. Coordination maintenance remains direct (§6). |

---

## 2. Startup

### Context Loading

Load only `fab/project/config.yaml`, `fab/project/constitution.md`, and `fab/project/context.md` (optional — skip gracefully if missing). The operator is a listed exception to the `_preamble.md` §1 always-load layer: code-quality, code-review, and the doc indexes serve artifact generation and review, which the operator never does (§1 Context discipline) — and a long-lived session re-pays any loaded file after every `/clear`. Do not run `fab preflight`. Do not load change artifacts.

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

Mark this tmux window as the operator for run-kit's dashboard (the `@rk_role` window option — rk owns the option contract, the pinned rendering, and the one-operator-per-server radio semantics; fab is only the producer):

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
2. Restore monitored set, autopilot queue, and branch_map from the file (supports `/clear` recovery)
3. Run `fab pane map --all-sessions` and display the output (all sessions on this server, not just the operator's own)
4. If any tracked items exist, start the single loop per §4 Adaptive cadence
5. Output: `Operator ready.` (+ `Loop active ({interval}).` if loop started)

---

## 3. Safety

### Confirmation Tiers

| Tier | Examples | Behavior |
|------|----------|----------|
| Read-only | Status check, pane map | No confirmation |
| Recoverable | Send `/fab-continue`, rebase | Announce before sending |
| Destructive | Merge PR, archive, delete worktree | Confirm before executing |

### Pre-Send Validation

Before sending keys to any pane, run the two-step gate in **`_cli-agents.md` § Pre-Send Validation** (pane exists via a refreshed pane map → agent state fits the send intent per the three-state `@rk_agent_state` read — the same mode-aware gate `rk mux send` enforces), then apply the operator's own policy on its outcome plus the two operator-specific checks:

1. **Pane gone** (gate step 1 fails) — report "Pane for {change} is gone." Do not send.
2. **Agent not `idle`** (gate step 2) — the operator does **not** silently proceed. If `active` or `waiting`: "{change} is {state}. Sending may corrupt its work / cut across a pending human answer. Send anyway?" — send only on explicit confirmation, and keep the confirmed send gated: with rk installed (`command -v rk`), a `waiting` target rides `rk mux send --answer`, an `active` target requires `rk mux send --force` (the deliberate skip-everything override); with rk absent, the confirmed send is raw `tmux send-keys` behind the operator's own state gate just performed. If unknown (`—`, no `@rk_agent_state` on the pane): the agent isn't instrumented; confirm before sending (plain send then warns-and-sends — no override needed). Only `idle` sends unattended. *(This routed-command confirm policy is unchanged by `--answer` — the flag swaps the mechanism after confirmation, not the ask; the auto-answer flow in §5 is the unattended `--answer` consumer.)*
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
| Send to busy agent | 0 | Warn, require explicit confirmation |
| Cherry-pick conflict | 0 | Abort, log, escalate. Do not spawn. |

---

## 4. The Loop

The loop is the operator's heartbeat — a `/loop "operator tick"` that runs as long as the monitored set is non-empty, an autopilot queue is active, or any watch is configured. When all three are empty, stop the loop. The loop starts when the first change is enrolled, an autopilot queue begins, or a watch is created. A user prompt can also restart it.

**Adaptive cadence.** The heartbeat interval is **not fixed** — it adapts to whether any monitored agent is `waiting` (blocked on a human):

- **Normal cadence: `3m`** (the default). Used when no monitored agent is `waiting` (or input-waiting).
- **Tightened cadence: `90s`** (§8, overridable). The moment a tick detects **any** monitored agent in the **`waiting`** Agent-column state (the pane's `@rk_agent_state` is `waiting` — the agent is blocked on a permission prompt / menu / elicitation), the operator tightens the heartbeat to bound worst-case detection/pickup latency. `waiting` is the primary, event-driven trigger; capture-based §5 menu detection is the fallback for uninstrumented panes (`—`). When a later tick finds no monitored agent `waiting` (or menu-waiting), it relaxes back to `3m`.
- **One-loop invariant.** Adapting cadence means **re-establishing the single loop at the new interval** (e.g. restart `/loop 90s "operator tick"`), never running two loops concurrently (`_cli-external.md` § /loop — "one loop at a time"). The operator changes the interval of *the* loop; it does not add a second.
- **Autopilot composition.** When an autopilot queue is driving, autopilot's own cadence (default `2m`, `_cli-external.md`) governs the loop; the menu-tightening applies to the monitoring loop's `3m`/`90s` band, not autopilot's `2m`.

### Operator State File

Persistent state, read on startup and every tick via `fab operator state`. The term **operator state file** used throughout this skill refers to the server-keyed file from §2 Init step 1 — one per tmux server spanning every repo it coordinates.

**The operator never hand-writes this file.** Every mutation goes through a `fab operator` subcommand (`enroll`/`update`/`remove`, the `watch` verbs, the `autopilot` verbs, `branch-map rm`) — agents state intent through flags; the binary owns the schema, the timestamps, the list-cap pruning, and the atomic write (same doctrine as `fab score`: agents never compute what the binary can own). The schema block below is *reference* documentation of what the binary maintains; the command contracts live in `_cli-fab.md` § fab operator.

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
    depends_on: []         # change IDs — same-repo deps cherry-pick, cross-repo deps are ordering-only (§6)
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
```

### Monitored Set

Each entry tracks: change ID, pane, **repo** (absolute main-worktree root), **session** (tmux session name), last-known stage, last-known agent state, stop_stage, spawned_by (watch name or null), depends_on (change IDs — same-repo cherry-pick, cross-repo ordering-only per §6), branch (this change's branch name), enrolled-at, last-transition-at. The pane ID is the server-global primary key; `repo` and `session` are the `(session, repo, pane)` addressing dimensions (§1).

**Enrollment**: operator sends a command to a change, user requests monitoring, or operator triggers an automatic action (including autopilot and watch spawns). Read-only actions do not enroll. Enrollment is `fab operator enroll <change-id> --pane … --repo … --session … --branch … [--stage …] [--agent …] [--stop-stage …] [--spawned-by …] [--depends-on …]` (contract in `_cli-fab.md` § fab operator) — one command writes both the monitored entry and the `{ branch, repo }` pair in the top-level `branch_map`.

After the enroll call, the operator MUST prefix `»` (U+00BB) to the target tmux window's name via the `fab pane window-name ensure-prefix` primitive. The primitive enforces the idempotent literal-prefix check internally, so the rename applies to every enrollment path without the caller needing to guard:

```sh
fab pane window-name ensure-prefix <pane> »
```

Windows that already carry `»` (operator-spawned windows from §6, `/clear`-restored entries, re-enrolled changes) no-op through the primitive's guard. A non-zero exit — pane vanished between refresh and rename (exit 2) or any other tmux error (exit 3, including tmux not running / socket unreachable) — causes the operator to log one line and continue. Enrollment itself is already durable from the preceding server-keyed state file write:

```
{change}: window rename skipped ({error}).
```

**Removal**: change reaches its stop stage (or a terminal stage if `stop_stage` is null), pane dies, user explicitly stops. Removal is `fab operator remove <change-id>` — the `branch_map` entry is **not** removed by it; it persists for downstream dependency resolution. On every removal path, the operator MUST swap the active-monitoring `»` prefix for the done-marker `›` (U+203A, SINGLE RIGHT-POINTING ANGLE QUOTATION MARK) via the `replace-prefix` primitive:

```sh
fab pane window-name replace-prefix <pane> » ›
```

The primitive's literal-prefix guard protects user-renamed windows (if the user renamed the window mid-monitoring so it no longer starts with `»`, the call no-ops). Exit 2 (pane missing — window is gone anyway) is treated as successful removal; other non-zero exits log `"{change}: window rename skipped ({error})."` and the operator continues. This keeps the tab bar an accurate at-a-glance map of what is currently tracked (`»` active) vs. operator-touched (`›` trail).

**Stop stage**: when `stop_stage` is set on a monitored entry, the operator treats that stage as the terminal stage for that change. On reaching it, the operator reports completion and removes the change — it does not push the agent further. Default is `null` (full pipeline: hydrate/ship/review-pr are terminal).

### Branch Map

The top-level `branch_map` persists change ID → `{ branch, repo }` mappings. Entries are added by `fab operator enroll` when changes are enrolled in the monitored set. Entries persist after changes leave the monitored set (merged, archived, pane died) — this is necessary so downstream changes can still look up dependency branches for cherry-picking. The `repo` is required to disambiguate a dependency's branch across repos and to decide same-repo (cherry-pick) vs. cross-repo (ordering-only) resolution per §6. Entries persist until the user explicitly clears them (`fab operator branch-map rm <change-id>` or `--all`) — the server-keyed state file survives operator sessions, so there is no session-end expiry.

### Tick Behavior

On each tick:

1. **Snapshot** — run `fab operator tick-start` (increments `tick_count`, writes `last_tick_at`, outputs `tick: N` and `now: HH:MM`). Parse stdout for the tick number and current time. Then run `fab pane map --all-sessions --json` (flag/field semantics — `--all-sessions`, the per-row `repo` and nullable `display_state` fields — in `_cli-fab.md` § fab pane map) and read the state via `fab operator state`. **Group the rows first by `repo`, then by `session`** within each repo. Compute status for all tracked items: stage advances, completions, review failures, pane deaths, and watch statuses from the last persisted check (`last_checked` / `last_error` / last counts). Output the status frame — see **Status Frame Format** below.

2. **Auto-nudge** — for each `waiting` agent (and each idle agent as fallback), run question detection (§5 — `waiting` is the primary signal). (No post-intake `/git-branch` nudge — `/fab-new` Step 11 creates or renames the branch inline; only a detected branch/change mismatch warrants a `/git-branch` send, per §3 pre-send validation item 4.)
3. **Watches** — for each watch, query the source, compare against `known` + `completed` (§7 step 2's dedupe rule), spawn on new matches (§7).
4. **Autopilot dispatch** — if an autopilot queue is active, run the next autopilot action (§6). Autopilot-driven changes are visible in the frame via `▶`.
5. **Removals** — remove completed changes (reached stop stage or terminal stage) and dead panes from the monitored set via `fab operator remove`.
6. **Observed-field updates** — per-tick changes to a monitored entry's `stage`/`agent`/`stop_stage` ride `fab operator update <change-id>` (the binary touches `last_transition` on a stage change). There is no whole-file persist step — every action above already persisted through its own verb.
7. **Loop lifecycle** — stop when no tracked state remains; otherwise apply §4 Adaptive cadence (autopilot uses §6's cadence)

Actions (nudges, removals, autopilot progress) render as an *italic* footnote line below the frame as they happen, `·`-separated, keeping them visually subordinate to the table frame:

```
*k8ds: auto-answered 'Allow Bash: npm test?' → y · Removed ab12 (complete), ef56 (pane gone) · Autopilot: cd34 → next ef56*
```

When the action log is long, the operator MAY split it across several italic lines rather than one — but each remains italic to stay subordinate to the frame.

### Status Frame Format

The frame is emitted as an assistant message that the agent harness renders as GitHub-flavored markdown in the terminal. **Render rule** (the binding constraint on every styling choice below): emit **bare markdown** — no code fence, no headings, no ANSI escapes (none of these survive the render path); the channels that DO render are **tables**, **emoji** (the only color channel), **bold** (`**…**`), *italic*, `code spans`, and plain URLs. The frame uses exactly these.

The frame is: a **header line**, one **repo section** per repo (an anchor line + a change table), then a **Watches** section (anchor line + table).

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
| Repo anchor | `📂 **{repo-path}** · {session}` | One per repo; omit `session:` label. Null roots render `📂 **(unresolved repo)**` |
| Change table | Headerless centered `▶`, `ID`, `Health`, `Stage`, `PR` | ID is a code span; Stage may trail `⚠️`; PR is the full `pr_url`, never markdown display text |
| Watches table | `Watch`, `Target`, `Health`, `Status` | Watch name is a code span; Target is `target_repo`; Status is counts + relative time |
| Ordering | Repo → session → change; then watches | Sort repos by path, sessions by name, changes by enrollment, watches by name |
| Styling | Emoji, bold, italic, code spans, plain URLs | Bold header/title/count/repo; health emoji is the color channel; action log stays italic |
| Stuck marker | `⚠️` after Stage | Same non-terminal >15m idle condition as 🔴 (§8) |
| Autopilot marker | `▶` or blank | Marks queue-driven changes; completion remains visible through ✅ |
| Watch timestamp | `{N}s ago` / `{N}m ago` / `{N}h ago` | Floor division at 60s and 60m |

**Health emoji** (geometric glyphs like `●◌✗` render monochrome and are NOT used):

| State | Change | Watch | Emoji |
|-------|--------|-------|:-----:|
| active / healthy | active | last query ok, no new items | 🟢 |
| waiting / idle / new-items | `waiting` (blocked on a human) or idle | has new unprocessed items | 🟡 |
| stuck / errored | >15m idle at non-terminal | `last_error` set | 🔴 |
| complete | reached terminal/stop stage | — | ✅ |
| paused | — | `enabled: false` | ⚪ |

### Idle Message

Between ticks, the operator displays an idle message with the current time and next-tick time:

```
Waiting for next tick. Time: 08:26 · next tick: 08:29
```

Run `fab operator time --interval {interval}` (where `{interval}` is the **currently active** loop interval — `3m` normally, `90s` when the cadence is tightened per §4 Adaptive cadence) to get the `now:` and `next:` values to fill in the message. A tightened cadence therefore shows the nearer next-tick time. This lets the user gauge staleness at a glance without scrolling to the last tick frame.

---

## 5. Auto-Nudge

The operator auto-answers routine prompts from monitored agents. The per-tick question-detection population (tick step 2) is each `waiting` agent (the primary signal — see below) plus, as a fallback, each idle agent. The capture-based patterns below **remain applicable** to `active`/unknown (`—`) panes — an uninstrumented harness, or a mid-turn prompt not yet flipped to `waiting` — but those panes are **not swept every tick**; the per-tick sweep is `waiting`+idle only.

**The `waiting` Agent-column state is the primary signal.** When a monitored pane's `@rk_agent_state` is `waiting`, the agent is blocked on a human (permission prompt / menu / elicitation) — this is event-driven and covers all instrumented harnesses (Claude/codex/copilot/gemini), so it is the first-class trigger for both the tightened cadence (§4) and question detection here. A `waiting` pane MUST be capture-scanned and run through the answer model, with each **idle** pane as the per-tick fallback (the population stated above).

### Question Detection

Capture and state-read mechanics (including the uninstrumented-pane state-writer caveat that makes capture the universal fallback) are in `_cli-agents.md` § Peek; the patterns and guards below are the operator's own question-detection policy over that capture.

1. **Capture**: `rk mux capture --raw -l 20 [-L <server>] <pane>` when rk is installed (`command -v rk`-gated); raw `tmux capture-pane -p -t <pane> | tail -20` when rk is absent — never an error (`-L <server>` only when the operator runs on a non-default tmux socket — the §9 second-operator case)
2. **Claude turn boundary guard**: `^\s*>\s*$` in last 2 lines → skip (normal human-turn boundary)
3. **Blank capture guard**: all blank → skip (treat as "cannot determine")
4. **Scan for indicators** (bottom-most match wins):
   - Lines ending with `?` (last non-empty line only, <120 chars, skip `#`/`//`/`*`/`>`/timestamp lines)
   - `[Y/n]`, `[y/N]`, `(y/n)`, `(yes/no)`
   - `Allow?`, `Approve?`, `Confirm?`, `Proceed?`
   - Claude Code permission/tool approval prompts
   - `Do you want to...`, `Should I...`, `Would you like...`
   - Lines ending with `:` (CLI input prompts)
   - Enumerated options (`[1-9]\)`)
   - `Press.*key`, `press.*enter`, `hit.*enter` (case-insensitive)
5. **No match** → stuck detection applies
6. **Match** → answer model

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

Before the send: run the §3 pre-send gate (`_cli-agents.md` § Pre-Send Validation — pane exists; state read per its step 2, expecting `waiting` or the idle fallback), then re-capture the terminal (the same rk-gated `rk mux capture --raw -l 20` capture as § Question Detection step 1, raw-tmux when rk is absent). If output changed since detection, abort — agent is no longer waiting. If the answer appears to land but the agent does not resume: on the rk path the send's delivery verification is built in — a probe failure surfaces as staged text + a stderr warning + exit 1, so re-capture and decide; never blind-resend. On the rk-absent raw path, apply the delivery probe (`_cli-agents.md` § Delivery Probe) instead of re-sending blind.

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

Every spawn flow is **repo-targeted**: the operator first establishes **which repo** the work targets (the existing change's repo, the `target_repo` of a watch, or the repo the user names), then runs every step against that repo — not against the operator's own repo.

The spawn sequence is:

1. **Establish target repo** — determine the absolute main-worktree root the work targets. For an already-tracked change, use its `repo` (monitored entry or `branch_map`). For a watch spawn, use the watch's `target_repo` (§7). For a fresh user request, use the repo the user names (default: the repo the operator was launched in).
2. **Create worktree** — run the repo-targeted, probe-and-route procedure in `_cli-external.md` § wt; never rely on the operator's CWD
3. **Activate the change pointer (existence-guarded)** — in the **just-created worktree's directory**, set that worktree's own `.fab-status.yaml` so the worktree is self-describing after the pipeline completes (a bare `fab`/`/fab-*` later resolves the change without naming it). Run the switch **only when the change folder already exists** — `fab resolve --folder <change>` succeeds iff a non-archived change folder matches:

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
4. **Resolve dependencies** — if the change has a non-empty `depends_on` list, resolve it per repo: same-repo deps cherry-pick into the worktree, cross-repo deps are ordering-only barriers (see Dependency Resolution below)
5. **Read the target repo's session command** — compose it per `_cli-agents.md` § Spawn Composition, in the **role-addressed** form with the target repo named: `fab agent --print --repo <target-repo>`. The operator-specific rule: **always pass `--repo <target-repo>`** — do NOT use the operator's own `config.yaml`, since each repo may configure a different provider/session command. (The provider-addressed form documented there is for ad-hoc cross-provider sessions, not operator worker spawns, which must carry the target repo's `default`-role profile.)
6. **Open agent tab** — open the composed command per `_cli-agents.md` § Spawn Composition ("Open it in a pane", incl. the one-prompt/no-`&&`-chaining rule), with the operator's window-marker name: `tmux new-window -n "»<wt>" -c <worktree-path> "<spawn_cmd> '<command>'"` (where `<wt>` is the worktree name from step 2 and `<spawn_cmd>` is the target repo's command from step 5)
7. **Enroll in monitored set** — unconditionally and silently via `fab operator enroll` (records pane, repo, session, stage, branch, and dependencies, plus the `branch_map` pair — contract in `_cli-fab.md` § fab operator); then apply §4 Enrollment's window prefix; never ask whether to monitor

Window markers (`»` / `›`) key on server-global pane IDs.

### Dependency Resolution

Dependency resolution is **two-tier**, split by repo. Each entry in `depends_on` is classified by comparing the dependency's `repo` (from its `branch_map` `{ branch, repo }` pair, or the dep's monitored entry) against **this change's** `repo`:

- **Same-repo dependency** (`dep.repo == change.repo`) → **cherry-pick** the dependency's code into the worktree, exactly as today. **In the `stacked-prs` merge mode the same-repo strategy changes** — the dependent's branch is created off the dependency's branch (no cherry-pick commit); see the `stacked-prs` note under Same-repo resolution below.
- **Cross-repo dependency** (`dep.repo != change.repo`) → **ordering-only barrier** in every mode: the operator waits until the dependency reaches its `stop_stage` (a terminal stage when `stop_stage` is null), then spawns the dependent agent. **No code is merged.**

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

**Cross-repo resolution.** For each cross-repo dependency, do not cherry-pick. Instead, before spawning, verify the dependency has reached its `stop_stage` (or terminal stage). If it has not, hold the spawn and let the loop re-check on subsequent ticks; spawn once every cross-repo barrier clears. Log the wait: `"{change}: waiting on cross-repo dependency {dep} (in {dep.repo}) to reach {stop_stage}."`

**Same-repo resolution (`stacked-prs` mode).** Steps 1–3 are skipped for same-repo dependencies — the dependent's branch is created off its nearest same-repo predecessor's *branch* at the §6 spawn sequence's worktree/branch step instead of off `origin/{default_branch}` (the probe-and-route per `_cli-external.md` § wt: existing dep branch → `wt create --checkout <dep-branch>` route). The squashed `"operator: cherry-pick"` commit does not exist for same-repo deps in this mode. After `/git-pr` creates the dependent's PR, the operator retargets its base to the dependency's branch: `gh pr edit <pr> --base <dep-branch>` (`/git-pr` itself is unchanged and mode-unaware). The merge-all choreography for the stack lives under Ordered Merge below. Dependency-branch drift after a dependent PR exists (a dep's review-pr rework moving its branch) is out of scope — the same exposure exists in the cherry-pick model; conflicts surface at merge-all and escalate.

**Why `origin/{default_branch}` as base (same-repo only)**: Each same-repo dependency branch carries its full transitive same-repo dependency content. When the operator spawned dep B, it cherry-picked dep A into B's worktree first. B's branch therefore contains A's commits. So `origin/{default_branch}..<B-branch>` gives the complete transitive closure within the repo — no need to chase transitive same-repo deps manually. This is why only direct/leaf same-repo dependencies need cherry-picking. (Cross-repo deps carry no such transitive content — they are ordering-only.)

### Dependency Declaration

Dependencies are declared through three conversational paths, all of which coexist:

1. **Explicit**: "cd34 depends on ab12" — operator records it through enrollment: `fab operator enroll cd34 … --depends-on ab12` (at spawn this is step 7; mid-flight it re-enrolls, which replaces the entry wholesale — carry the current stage/agent along)
2. **Autopilot queue (implicit)**: resolve ordering per § Autopilot → Queue ordering
3. **`--base` flag (explicit)**: autopilot `--base <prev-change>` explicitly sets `depends_on: [<prev-change-id>]` for the subsequent change (matches path 2's pick when the previous entry is same-repo; available for ad-hoc overrides)

### Working a Change

> **Pipeline-first routing (§1):** all three work paths below MUST go through the fab pipeline (`/fab-new` then a pipeline command for new work; the appropriate stage for already-intaked changes) — never raw implementation instructions to agent panes.

Every form runs §6's target-repo worktree → guarded activation → dependencies → target-repo session command → tab → enrollment sequence:

1. **Existing change:** use the monitored/`branch_map` repo and embed `/fab-fff <change>` as the single prompt per `_cli-agents.md` § Spawn Composition; the transient override targets the pipeline and spawn step 3 activates the pointer.
2. **Raw text** (for example, "fix login after password reset"): use the named repo (default operator launch repo) and embed `/fab-new <shell_escaped_description>`. Shell-escape the raw description; never insert it unescaped. The existence guard skips activation until `/fab-new` creates and activates the change at Step 10.
3. **Backlog ID or Linear issue:** resolve it first (optional `idea` lookup per `_cli-external.md` § Delegation and binary gate), then embed `/fab-new <id>`. The existence guard skips activation and `/fab-new` owns it.

On completion (all three): PR ready, optionally archive. Both raw text and backlog paths use `/fab-new` to generate a proper intake with traceability. `/fab-new` captures the raw input in the intake's Origin section — the user just says "fix [description]" and the operator does the rest.

### Autopilot

User provides a queue of changes. Confirmation prompt reflects the active mode:
- **Default (`cherry-pick-ladder`):** "Confirm upfront (creates PRs — merge after review)."
- **`merge-auto`:** "Confirm upfront (merges PRs on completion)."
- **`stacked-prs`:** "Confirm upfront (creates stacked PRs — merge after review)."

A queue **may span repos**, with mixed dependency semantics: implicit `--base` chaining (and explicit `depends_on`) cherry-picks **within a repo** and **degrades to an ordering-only barrier across repo boundaries** (per Dependency Resolution above; the nearest-same-repo-predecessor rule is defined in Queue ordering below). Worked example — a chain `ab12 → cd34 → ef56` where `cd34` lives in a different repo: `cd34` gets `depends_on: [ab12]` (cross-repo — waits for `ab12` to reach its stop/terminal stage, no code), and `ef56` (back in `ab12`'s repo) gets `depends_on: [ab12]` — its nearest same-repo predecessor — and cherry-picks from it; queue order still runs `ef56` after `cd34`.

Once the user confirms, persist the queue via `fab operator autopilot start --queue <id,id,...> [--mode <name>]` (the binary stores and prints the mode; contracts in `_cli-fab.md` § fab operator autopilot); every later progression (completion or skip) is `fab operator autopilot advance [--skip]`, and the interrupts below ride `pause`/`resume`/`stop`.

Queue ordering:

| Strategy | Description |
|----------|-------------|
| User-provided | Run in the exact order given. Implicit `--base` chaining by default: every change after the first gets `depends_on: [<nearest-same-repo-predecessor>]` — the closest earlier queue entry in the same repo (cherry-picked); when no earlier entry shares the repo, the immediately previous entry (cross-repo → ordering-only). No explicit `--base` flag required. |
| Confidence-based | Sort by confidence score descending. Highest-confidence first (independent changes) |
| Hybrid | User provides constraints (partial order); operator sorts unconstrained by confidence |

**Merge modes** — three flat names, selected at queue start via `fab operator autopilot start --mode <name>` (persisted in the autopilot state block, so the mode survives `/clear`) or natural language mapping onto them:

- **`cherry-pick-ladder`** (default) — PRs are created but not merged until the user explicitly requests merging; implicit `--base` chaining is active (per Queue ordering, "User-provided").
- **`merge-auto`** — merge-as-you-go: merge each PR on completion, then `git fetch origin` and rebase the next change onto `origin/{default_branch}` (the default branch resolved per Dependency Resolution step 0 — never a hardcoded `origin/main`). Implicit `--base` chaining is disabled in this mode — each change rebases onto `origin/{default_branch}` independently. Natural language equivalents: "merge as you go", "merge on complete", "merge each when done".
- **`stacked-prs`** — `cherry-pick-ladder` merge timing (PRs created up front, merged only on explicit user request) with true stacked-PR topology for same-repo chains: the dependent's branch is created off its dependency's *branch* (no cherry-pick commit) and its PR targets the dependency's branch, so each PR diff shows only its own delta. Mechanics: same-repo resolution in Dependency Resolution below; merge-all choreography in Ordered Merge. Natural language equivalents: "stacked PRs", "stack the PRs".

The operator works each change through the pipeline. Pre-send validation (§3) applies to any command sent to an existing pane; the initial pipeline command itself is **embedded at spawn** (§6 step 6) — the single dispatch point:

1. **Gate** — check confidence score **before anything spawns**. If below threshold, flag and wait — no worktree, no tab, no dispatch for a below-threshold change
2. **Spawn** — run the §6 spawn sequence steps 1–2 (establish the change's target repo, create worktree in it; `--reuse` for respawns)
3. **Resolve dependencies + open tab + enroll** — §6 spawn sequence steps 3–7 (existence-guarded pointer activation, same-repo cherry-pick / cross-repo ordering-only barriers per Dependency Resolution). Step 6's `<command>` is the change's pipeline command — `/fab-fff <change>` (or the appropriate command for its current stage) — so the dispatch happens **once, at spawn**; do NOT send the command again after the tab opens
4. **Monitor** — normal tick detection handles progress
5. **Record** — on completion, run `fab operator autopilot advance` (the binary moves `current` to `completed` and promotes the next entry) and collect the PR URL. The `{ branch, repo }` pair is already in `branch_map` — `enroll` recorded it at spawn
6. **Spawn next** — repeat from item 1 using § Queue ordering and § Dependency Resolution; embed its command at spawn
7. **Report** — `"ab12: PR ready. 1 of 3 complete. Starting cd34."`
8. **(After all complete) Summary** — list all PR links with per-repo dependency annotations and per-repo merge order suggestion (see Queue Completion Summary below)

In `merge-auto` mode, steps 5–8 merge the PR on completion, run `git fetch origin`, rebase the next change onto `origin/{default_branch}` (resolved per Dependency Resolution step 0), and report the merge.

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

When the user says "merge all" or "merge the queue" after a `cherry-pick-ladder` or `stacked-prs` queue completes, the operator merges PRs respecting **per-repo PR sequences** — within each repo, base-first in dependency order; across repos, cross-repo ordering barriers are honored (a cross-repo dependent's PR is merged only after its barrier dependency reaches its target repo's main). It waits for CI to pass on each PR before proceeding to the next in that repo's sequence:

1. Merge `~/code/foo` PR 1 (base) — wait for CI pass
2. Merge `~/code/bar` PR 2 (its cross-repo barrier `foo:1` is now on main) — wait for CI pass
3. Merge `~/code/foo` PR 3 — wait for CI pass

Report each merge with its repo: `"ab12: merged (foo 1/2)"`, `"cd34: merged (bar 1/1)"`, `"ef56: merged (foo 2/2)"`.

**`stacked-prs` merge-all adds two steps per merge**, because each PR in a same-repo chain is based on its dependency's branch:

1. **Verify base retarget** — after a chain's base PR merges, GitHub auto-retargets the dependent PR's base onto the default branch when the merged base branch is deleted. Rely on this, and retarget explicitly (`gh pr edit <pr> --base {default_branch}` — a plain branch name, never a remote ref) when the branch was not deleted.
2. **Rebase the next branch after a squash merge** — after a squash merge, the next branch in the chain still carries the dependency's original commits, which the default branch now contains only as a squashed commit. Before that next PR is clean/mergeable, rebase it onto the default branch, dropping the already-merged dependency commits, and force-push:

   ```bash
   git fetch origin && git rebase --onto origin/{default_branch} <merged-dep-branch> <next-branch> && git push --force-with-lease
   ```

   `{default_branch}` is resolved per Dependency Resolution step 0 — never a hardcoded `origin/main`. A conflict in this rebase **halts and escalates** (never silently skips), consistent with the cherry-pick-conflict policy.

**CI failure during ordered merge (halt-dependents-only)**: If CI fails on a PR, the operator halts **that repo's merge sub-sequence** AND **any repo whose queued items carry a cross-repo `depends_on` into the failed chain — transitively**. "Dependent" is determined over the cross-repo `depends_on` graph: a repo halts if any of its queued items depends (directly, or via another already-halted item) on a PR in the failed chain. **Truly independent repos' sub-sequences continue merging.** The operator does not abandon the queue; it isolates the blast radius to the failure's dependency cone. On completion it reports which sub-sequences halted vs. completed and escalates the failure to the user:

```
ab12: CI failed (~/code/foo). Halted: foo sub-sequence; bar (cross-repo dep into foo). Completed: baz sub-sequence (2 PRs merged). Fix foo and retry.
```

Autopilot state (queue, current, completed, mode) persists in the operator state file — written by the `fab operator autopilot` verbs, never hand-edited; on queue exhaustion the binary retains `queue`/`completed`/`mode` with `current: null, state: null` so the summary below can still read them, and `fab operator autopilot stop` clears the block after the summary renders.

**Failures**: review exhausted → skip. Rebase conflict mid-queue → skip (`merge-auto` only; does not apply in `cherry-pick-ladder` since there are no rebase steps). Rebase conflict during a `stacked-prs` merge-all → escalate (never skip). Cherry-pick conflict → escalate (do not skip). Pane dies → 1 respawn (`--reuse`), then skip. Stage timeout (>30m) → flag. Total timeout (>2h) → flag.

**Interrupts**: "stop after current", "skip <change>", "pause", "resume" — acknowledged immediately, and persisted through the matching verb: `fab operator autopilot stop` once the current change lands (or immediately to abandon the queue), `advance --skip` (drop current without recording it completed), `pause`, `resume`.

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

When a watch-spawned agent reaches its `stop_stage`, `fab operator watch complete <name> <item-id>` (moves the item from `known` to `completed`) and report: `"Watch linear-bugs: DEV-1024 completed intake."`

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

- **Multiple sessions, same server** share one operator and one state file. The operator addresses their agents by the `(session, repo, pane)` tuple (§1); there is no per-session or per-repo operator.
- **A second operator means a second tmux server** — start one on a separate socket (`tmux -L <label>`). Its state file is keyed by that socket, so the two operators never collide. There is no `--name` dimension; the server boundary is the only isolation knob. Sends on a non-default socket carry the matching flag: `rk mux -L <label> send` (or `tmux -L <label> send-keys …` on the rk-absent raw path).

### Settings

| Setting | Default | Override via natural language |
|---------|---------|------------------------------|
| Loop interval | 3m | "check every {N}m" |
| Stuck threshold | 15m | "flag agents stuck for more than {N}m" |
| Waiting/menu heartbeat | 90s | "tighten to {N}s when an agent is on a menu" |
| Notify channel | `rk` (run-kit Web Push; auto-fallback when `rk` absent) | "notify via ntfy topic {topic}" / "notify via discord {url}" / "notify via push" |

These settings are session-scoped and reset on `/clear` or session restart; they are not operator-state-file fields. The **strategic auto-default threshold is hardcoded at 30m** (§5) — there is deliberately **no** setting for it.

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
| Requires a `fab/` project? | No — session command comes from the project's `providers.claude.interactive_command` when `fab/` is resolvable, else `spawn.DefaultSpawnCommand` (the template `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}`). No project `providers`/`agent:` block is read on a `fab/`-less launch |
| Coordinating-agent model | Operator role — `fab operator` resolves the `operator` role (`agent.ResolveRole`; a Tier-1 role, so the `agent.session` knob picks its provider), reads that provider's `interactive_command`, injects the profile via `spawn.WithProfile` (**substitutes** into a `{model}`/`{effort}` template — the built-in claude default is templated — or **appends** `--model`/`--effort` to a plain command carrying no placeholder); falls back to the built-in operator profile + built-in claude provider on any failure (incl. no resolvable `fab/` project) |
| Uses `/loop`? | Yes — adaptive heartbeat: `3m` normally, tightens to `90s` (§8) when any monitored agent is `waiting` (`@rk_agent_state`) or menu-waiting (capture fallback), relaxes back to `3m`; one loop at a time |
| Uses the operator state file? | Yes — monitored set + autopilot queue + branch map persistence in the server-keyed path (§2 Init step 1); reads via `fab operator state`, every mutation through a `fab operator` verb — never a hand-write (§4 doctrine) |
| Multi-repo / multi-session? | Yes — one operator per tmux server spans all its sessions and repos via the `(session, repo, pane)` addressing tuple |
