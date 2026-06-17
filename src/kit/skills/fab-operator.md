---
name: fab-operator
description: "Use when coordinating multiple fab agents across tmux panes — multi-agent monitoring, auto-answering prompts, routing commands, driving autopilot queues, and dependency-aware agent spawning."
helpers: [_cli-fab, _cli-external]
---

# /fab-operator

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

Multi-agent coordination layer. Runs in a dedicated tmux pane, observes agents across all sessions on its tmux server via `fab pane map --all-sessions`, routes commands via `tmux send-keys`, monitors progress via `/loop`. Spans multiple repos and sessions on one server. The loop is the heart of the operator.

Start via `fab operator` (singleton tmux tab named `operator`). The launcher requires **neither a git repo nor a resolvable `fab/` project** — matching the per-server, cross-repo singleton model, whose natural launch point is a neutral parent directory (e.g. `~/code`) with no `fab/` project. The degraded behavior is exact:
- **Window cwd**: the repo root when launched inside a git repo, else `os.Getwd()` (the current directory). It errors only if both resolutions fail.
- **Spawn command**: `agent.spawn_command` from the project's `fab/project/config.yaml` when a `fab/` project is resolvable, else the built-in default `spawn.DefaultSpawnCommand` (`claude --dangerously-skip-permissions`). When launched `fab/`-less, **no project `agent.spawn_command`/`agent.tiers` is read** — there is no project to customize from.
- **Doing-tier model**: the launcher resolves the **doing-tier** `{model, effort}` (via `fab resolve-agent apply`) and appends `--model`/`--effort` to the spawn command, so the coordinating agent runs on a deliberately-chosen model rather than whatever the spawn command happened to specify. With no resolvable `fab/` project (or any other failure) the doing tier resolves to its built-in default `{claude-opus-4-8, high}` — so a `fab/`-less launch composes a fully-defaulted command (default spawn command + doing default).

---

## 1. Principles

**Coordinate, don't execute.** The operator routes instructions to the right agent — it never implements work directly. If ambiguous, ask. Exception: operational maintenance (merge PR, archive, delete worktree) is executed directly by the operator since these are coordination-level actions, not pipeline work.

**Multi-repo aware.** The operator spans multiple repos and multiple tmux sessions on a **single tmux server** — one operator per server (§8). Every agent is addressed as a `(session, repo, pane)` tuple: the **pane ID is the primary key** (server-global and stable), with `repo` (the agent's absolute main-worktree root) and `session` (its tmux session name) layered on as dimensions, not replacements. Every monitored entry, every `branch_map` entry, and every watch is repo-qualified. State lives in one server-keyed file, not per-repo (§4, §9).

**Spawn-in-worktree.** The operator's own pane is reserved for coordination state — pane maps, autopilot queue, operator state file bookkeeping (see §4). All pipeline work (`/fab-new`, `/fab-proceed`, `/fab-fff`, `/fab-ff`, `/fab-continue`, `/git-branch`, `/git-pr`) MUST run in a freshly spawned agent tab in its own worktree — never in the operator pane itself. The first action for any new request is `wt create --non-interactive`, then spawn the agent tab (see §6). Even a one-liner change gets its own worktree.

**Automate the routine.** The operator exists to take work off the user's hands. Auto-answer prompts, nudge stuck agents, rebase stale PRs, spawn agents from backlog — act on the user's behalf for routine operational decisions. The PR review stage is the safety net. Never ask whether to monitor a spawned agent — if the operator spawned it, monitor it.

**Not a lifecycle enforcer.** Individual agents self-govern via their own pipeline skills. The operator does not validate stage transitions or enforce pipeline rules. If an agent is at an unexpected stage, report it factually.

**Context discipline.** The operator never reads change artifacts (intakes, specs, plans). Its context window is reserved for coordination state — pane maps, stage snapshots, the operator state file. This keeps long-running sessions lean.

**State re-derivation.** Before every action, re-query live state via `fab pane map --all-sessions` (so every session on the server is seen, not just the operator's own). Panes die, stages advance, agents finish — stale state leads to wrong actions. Never rely on conversation memory for pane, repo, session, or stage values.

**Self-manage context.** The operator is long-lived. When context approaches capacity, run `/clear` and restart the loop. Continuity is maintained via the operator state file — the monitored set and autopilot queue survive a clear. After clearing, re-read context files, re-read the operator state file, and resume.

**Pipeline-first routing.** The operator MUST route all new work through `/fab-new` (to generate intake) then a pipeline command (`/fab-fff`, `/fab-ff`, or `/fab-continue`). The operator MUST NOT dispatch raw inline implementation instructions (e.g., "fix the login bug by changing line 42 in auth.ts") directly to agent panes. The operator MUST NOT send `/fab-continue` to skip intake for new work — `/fab-new` is always the entry point. Exception: operational maintenance commands (see "Coordinate, don't execute" above) are coordination-level actions and remain direct.

---

## 2. Startup

### Context Loading

Load only `fab/project/config.yaml`, `fab/project/constitution.md`, and `fab/project/context.md` (optional — skip gracefully if missing). The operator is a listed exception to the `_preamble.md` §1 always-load layer: code-quality, code-review, and the doc indexes serve artifact generation and review, which the operator never does (§1 Context discipline) — and a long-lived session re-pays any loaded file after every `/clear`. Do not run preflight. Do not load change artifacts.

Helpers declared in frontmatter: `_cli-fab` (fab command reference) and `_cli-external` (wt, idea, tmux, /loop reference). Naming conventions are inlined in `_preamble.md` § Naming Conventions — already loaded.

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

### Init

1. Read the server-keyed operator state file (`$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`, fallback `~/.local/state/...`; the binary derives the path via `fab operator tick-start` — the operator does not compute it). If missing, it is created with empty `monitored: {}`, `autopilot: null`, and `branch_map: {}`. Old repo-rooted `.fab-operator.yaml` files are not read or migrated
2. Restore monitored set, autopilot queue, and branch_map from the file (supports `/clear` recovery)
3. Run `fab pane map --all-sessions` and display the output (all sessions on this server, not just the operator's own)
4. If any tracked items exist (monitored changes, active autopilot, or watches), start the loop at the cadence the current state warrants (§4 Adaptive cadence): `/loop 90s "operator tick"` if any restored agent is already menu-waiting, else `/loop 3m "operator tick"`
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

Before sending keys to any pane:

1. **Verify pane exists** — refresh pane map. If gone: "Pane for {change} is gone." Do not send.
2. **Check agent is idle** — if busy: "{change} is active. Sending may corrupt its work. Send anyway?" Only on explicit confirmation.
3. **Check change is active** — if the target change isn't the active change in that tab, send `/fab-switch <change>` first.
4. **Check branch alignment** — if the tab's git branch doesn't match the change folder name, send `/git-branch` to align it.

### Branch Fallback

When `fab resolve` fails during a **user-initiated** action (not monitoring ticks):

1. Scan branches: `git for-each-ref --format='%(refname:short)' refs/heads/ refs/remotes/ | grep -iF "<query>"`
2. **Single match, read-only**: read `.status.yaml` via `git show <branch>:fab/changes/<folder>/.status.yaml`
3. **Single match, action**: create a worktree and proceed (`wt create --non-interactive --worktree-name <name> <branch>`)
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

**Adaptive cadence.** The heartbeat interval is **not fixed** — it adapts to whether any monitored agent is waiting on an interactive menu:

- **Normal cadence: `3m`** (the default). Used when no monitored agent is input-waiting.
- **Tightened cadence: `90s`** (§8, overridable). The moment a tick detects **any** monitored agent sitting on an interactive menu (input-waiting per §5 Question Detection), the operator tightens the heartbeat to bound worst-case detection/pickup latency. When a later tick finds no monitored agent menu-waiting, it relaxes back to `3m`.
- **One-loop invariant preserved.** Adapting cadence means **re-establishing the single loop at the new interval** (e.g. restart `/loop 90s "operator tick"`), never running two loops concurrently (`_cli-external.md` § /loop — "one loop at a time"). The operator changes the interval of *the* loop; it does not add a second.
- **Autopilot composes unchanged.** When an autopilot queue is driving, autopilot's own cadence (default `2m`, `_cli-external.md`) governs the loop; the menu-tightening applies to the monitoring loop's `3m`/`90s` band, not autopilot's `2m`.

### Operator State File

Persistent state, read on startup and every tick, written after every state change. The term **operator state file** used throughout this skill refers to this file.

The operator state file is **server-keyed**, not repo-rooted: it lives at `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (falling back to `~/.local/state/fab/operator/<server-slug>.yaml` when `XDG_STATE_HOME` is unset), keyed by the tmux socket path so the one operator per server owns one file spanning every repo it coordinates (see §8, §9). The binary derives this path (`fab operator tick-start` reads/writes it); the operator never needs to compute it. Old repo-rooted `.fab-operator.yaml` files from before the server-keyed model are **not migrated** — they are abandoned in place (the monitored set is re-derivable from live `»`-prefixed panes).

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

**Enrollment**: operator sends a command to a change, user requests monitoring, or operator triggers an automatic action (including autopilot and watch spawns). Read-only actions do not enroll. On enrollment, the change's `{ branch, repo }` pair is also recorded in the top-level `branch_map`.

After writing the monitored entry to the server-keyed state file (§4), the operator MUST prefix `»` (U+00BB) to the target tmux window's name via the `fab pane window-name ensure-prefix` primitive. The primitive enforces the idempotent literal-prefix check internally, so the rename applies to every enrollment path without the caller needing to guard:

```sh
fab pane window-name ensure-prefix <pane> »
```

Windows that already carry `»` (operator-spawned windows from §6, `/clear`-restored entries, re-enrolled changes) no-op through the primitive's guard. A non-zero exit — pane vanished between refresh and rename (exit 2) or any other tmux error (exit 3, including tmux not running / socket unreachable) — causes the operator to log one line and continue. Enrollment itself is already durable from the preceding server-keyed state file write:

```
{change}: window rename skipped ({error}).
```

**Removal**: change reaches its stop stage (or a terminal stage if `stop_stage` is null), pane dies, user explicitly stops. The `branch_map` entry is **not** removed — it persists for downstream dependency resolution. On every removal path, the operator MUST swap the active-monitoring `»` prefix for the done-marker `›` (U+203A, SINGLE RIGHT-POINTING ANGLE QUOTATION MARK) via the `replace-prefix` primitive:

```sh
fab pane window-name replace-prefix <pane> » ›
```

The primitive's literal-prefix guard protects user-renamed windows (if the user renamed the window mid-monitoring so it no longer starts with `»`, the call no-ops). Exit 2 (pane missing — window is gone anyway) is treated as successful removal; other non-zero exits log `"{change}: window rename skipped ({error})."` and the operator continues. This keeps the tab bar an accurate at-a-glance map of what is currently tracked (`»` active) vs. operator-touched (`›` trail).

**Stop stage**: when `stop_stage` is set on a monitored entry, the operator treats that stage as the terminal stage for that change. On reaching it, the operator reports completion and removes the change — it does not push the agent further. Default is `null` (full pipeline: hydrate/ship/review-pr are terminal).

### Branch Map

The top-level `branch_map` persists change ID → `{ branch, repo }` mappings. Entries are added when changes are enrolled in the monitored set. Entries persist after changes leave the monitored set (merged, archived, pane died) — this is necessary so downstream changes can still look up dependency branches for cherry-picking. The `repo` is required to disambiguate a dependency's branch across repos and to decide same-repo (cherry-pick) vs. cross-repo (ordering-only) resolution per §6. Entries persist until the user explicitly clears them — the server-keyed state file survives operator sessions, so there is no session-end expiry.

### Tick Behavior

On each tick:

1. **Snapshot** — run `fab operator tick-start` (increments `tick_count`, writes `last_tick_at`, outputs `tick: N` and `now: HH:MM`). Parse stdout for the tick number and current time. Then run `fab pane map --all-sessions --json` (the `--all-sessions` flag is required so the operator sees agents in **every** session on its server, not just its own; `--json` exposes the per-row `repo` field — the agent's absolute main-worktree root, or `null` when the pane is not in a git repo — and the nullable per-row `display_state` field — `active|ready|done|failed|pending|skipped`, `null` under the same conditions as `stage` — an honest attention-state axis alongside `stage` (`failed` is reachable since the DisplayStage failed tier shipped with 260612-dkn3)) and read the server-keyed state file. **Group the rows first by `repo`, then by `session`** within each repo. Compute status for all tracked items: stage advances, completions, review failures, pane deaths, and watch statuses from the last persisted check (`last_checked` / `last_error` / last counts). Output the status frame — see **Status Frame Format** below.

2. **Auto-nudge** — for each idle agent, run question detection (§5). (No post-intake `/git-branch` nudge — `/fab-new` Step 11 creates or renames the branch inline; only a detected branch/change mismatch warrants a `/git-branch` send, per §3 pre-send validation item 4.)
3. **Watches** — for each watch, query the source, compare against `known` + `completed` (§7 step 2's dedupe rule), spawn on new matches (§7).
4. **Autopilot dispatch** — if an autopilot queue is active, run the next autopilot action (§6). Autopilot-driven changes are visible in the frame via `▶`.
5. **Removals** — remove completed changes (reached stop stage or terminal stage) and dead panes from the monitored set.
6. **Persist** — write updated state to the operator state file
7. **Loop lifecycle** — if monitored set is empty, no autopilot, and no watches, stop the loop. Otherwise **adapt the cadence** (§4 Adaptive cadence): if any monitored agent was detected menu-waiting this tick (step 2) and the loop is not already at the tightened interval, re-establish the single loop at `90s` (§8); if no monitored agent is menu-waiting and the loop is tightened, relax it back to `3m`. Re-establishing the loop replaces the interval of the one loop — it never starts a second (`_cli-external.md` § /loop). Autopilot's own cadence governs when a queue is driving (§6).

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

**Header line**: `🛰️ **Operator** · {HH:MM} · tick #{N} · **{N} tracked**`. The 🛰️ emoji and bold give it prominence. `N tracked` is the total count of all entries (changes + watches) — no per-type or per-repo counts.

**Repo-section anchor**: `📂 **{repo-path}** · {session}` — one per repo, with the repo's change table beneath it. The 📂 emoji is the section landmark the eye jumps to. The session label drops the literal word "session:". A repo whose main-worktree root could not be resolved (`null` in the `repo` JSON field) renders under a `📂 **(unresolved repo)**` anchor rather than being dropped.

**Change table** columns (consistent across all repo sections):

| Column | Content |
|--------|---------|
| (autopilot) | `▶` if autopilot-driven, blank otherwise. Center-aligned, header-less |
| ID | Change ID (4-char) in a `code span` |
| Health | Health emoji — universal position across all types |
| Stage | Stage text (e.g. `apply → review`), with the `⚠️` stuck marker trailing when applicable |
| PR | Full PR URL from the `pr_url` JSON field when present (ship/review-pr stages); blank otherwise |

**Watches table** columns: `Watch` (name in `code span`), `Target` (the watch's `target_repo`), `Health` (emoji), `Status` (counts + relative timestamp). Watches render after all repo sections.

**Ordering**: Repo sections first (repos sorted by path, sessions sorted by name within a repo, changes sorted by enrollment time within a session), then the Watches section (watches sorted alphabetically by name).

**Health emoji** (geometric glyphs like `●◌✗` render monochrome and are NOT used):

| State | Change | Watch | Emoji |
|-------|--------|-------|:-----:|
| active / healthy | active | last query ok, no new items | 🟢 |
| idle / new-items | idle | has new unprocessed items | 🟡 |
| stuck / errored | >15m idle at non-terminal | `last_error` set | 🔴 |
| complete | reached terminal/stop stage | — | ✅ |
| paused | — | `enabled: false` | ⚪ |

**Markdown styling**: emoji carry the health color; **bold** marks the header title, `N tracked`, and repo-path anchors; `code spans` mark change/watch IDs and watch names; the PR cell holds a **full URL as plain text** (selectable/copyable in any terminal, including a plain xterm — markdown `[#N](url)` link syntax is deliberately NOT used because xterm shows only the `#N` display text, not a copyable URL). The autopilot `▶` is a plain monochrome glyph in its own column.

**Stuck marker**: `⚠️` trails the Stage cell text on any change row whose idle duration has exceeded the stuck threshold (§8, default 15m) at a non-terminal stage — the same condition that shows the 🔴 health emoji. It is a redundant inline flag drawing the eye to rows needing manual investigation; rows below the threshold carry no marker.

**Autopilot marker**: `▶` marks changes driven by the autopilot queue. Non-autopilot changes (manually enrolled or watch-spawned) show blank. Queue state is readable from the list — which entries have `▶`, which are complete.

**Watch timestamps**: Relative format (`{N}m ago`) matching the idle duration format: `{N}s ago` (< 60s), `{N}m ago` (60s–59m), `{N}h ago` (>= 60m). Floor division.

### Idle Message

Between ticks, the operator displays an idle message with the current time and next-tick time:

```
Waiting for next tick. Time: 08:26 · next tick: 08:29
```

Run `fab operator time --interval {interval}` (where `{interval}` is the **currently active** loop interval — `3m` normally, `90s` when the cadence is tightened per §4 Adaptive cadence) to get the `now:` and `next:` values to fill in the message. A tightened cadence therefore shows the nearer next-tick time. This lets the user gauge staleness at a glance without scrolling to the last tick frame.

---

## 5. Auto-Nudge

The operator auto-answers routine prompts from monitored agents. Each idle agent is checked every tick.

### Question Detection

1. **Capture**: `tmux capture-pane -t <pane> -p -S -20`
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
4. Numbered menu:
   - Classify the prompt as **Routine** or **Strategic** using LLM judgment over the terminal capture. Signals: option text length, semantic distinctness of options, surrounding agent context, reversibility of the choice. No hardcoded keyword list.
     - **Routine** (tool/permission prompts, binary-framed menus, synonymous-option menus) → `1` (first/default).
     - **Strategic** (multi-option choices representing materially different directions — scope, PR split, pipeline shape, commit organization, spec/approach decisions) → **non-blocking** handling, split by whether a defensible recommendation exists (see **Non-Blocking Strategic Handling** below). A Strategic classification **never** ends the operator's turn.
   - On classification uncertainty, treat as Strategic. False-negative strategic commits the queue to an unchosen direction; false-positive strategic costs at most a notification (and, if auto-picked, a reversal at PR review — §1).
5. Open-ended, answer determinable from visible context → send that answer
6. Cannot determine keystrokes → escalate to user (left open). Rule-6 escalations are **excluded** from auto-pick and from the 30m idle auto-default — the operator does not know the correct keystrokes, so sending a guess would emit nonsense into the pane.

### Non-Blocking Strategic Handling

A Strategic classification (rule 4 above) **MUST NOT block the loop**. The operator handles the prompt out-of-band within the current tick and proceeds to the next monitored change in the **same** tick — one strategic question on one change no longer freezes the queue. Two branches:

- **Strategic + defensible recommendation** → **auto-pick-and-notify.** The operator picks its recommended option (LLM judgment over the capture — the same signals rule 4 lists: option text, distinctness, surrounding context, reversibility), sends it (after the **Sending Auto-Answers** re-capture guard below), fires a notification (see **Notification Send**), and keeps ticking. The PR review stage is the reversal point (§1 "The PR review stage is the safety net").
- **Strategic + no defensible default** → **leave open and notify.** The operator leaves the prompt open for the user, fires a notification, and keeps ticking. The 30m **Idle Auto-Default** (below) remains the backstop for these left-open prompts.

In both branches the operator **continues ticking**. The user answers asynchronously — by responding to the notification's guidance or by typing directly into the agent's pane — and the operator **picks up the resolution on a later tick** via its normal re-capture/re-detection (**Sending Auto-Answers** already re-captures before any send). No new pickup mechanism is added.

### Notification Send

The notification is a single out-of-band shell send the operator runs when it auto-picks or leaves open a Strategic prompt. The **default channel is `rk notify`** — a run-kit external contract: `rk notify <message> [--title string]` (run-kit Web Push, released in `rk v2.3.2`; full command reference in `_cli-external.md` § rk (run-kit)). The send is gated on `command -v rk` and runs fail-silent per `_preamble.md` § Run-Kit (rk) Reference (which documents the gate and the fail-silent discipline; the `notify` subcommand itself is documented in `_cli-external.md` § rk):

```sh
command -v rk >/dev/null 2>&1 && rk notify "{change}: {summary} ({repo})" --title "Operator: strategic question"
```

`rk notify` delivers a real background mobile/desktop Web Push to every subscribed device and is **fail-silent by contract** (exits 0 / prints nothing on any error — server unreachable, no subscriptions — so it can never stall the loop).

**When `rk` is absent** (operator running where run-kit isn't installed), fall back to the first available **documented alternative**, configurable via the §8 `Notify channel` setting:

- **ntfy.sh** — `curl -d "{change}: {summary} ({repo})" ntfy.sh/<high-entropy-topic>`. No account, curl-from-shell, cross-repo aggregator, mobile push. **High-entropy topic REQUIRED** — public topics are world-readable to anyone who knows the name (the topic name is the only secret), so use a long random topic (e.g. `op-9f3a2c7e-strat`) and never put secrets in the body. The strongest no-run-kit fallback.
- **Discord webhook** — `curl -H 'Content-Type: application/json' -d '{"content":"…"}' <webhook>`. No account, one webhook = one channel, indefinite searchable history, mobile push.
- **`PushNotification`** (built-in Claude Code harness tool) — zero infra, no topic secret to leak, headless-safe; a *personal* push to the user's Claude apps, not a shared searchable feed. Good "just ping me" fallback.
- **Slack MCP** (`mcp__claude_ai_Slack__slack_send_message`) — searchable channel feed, mobile push; caveat: an interactively-authed MCP may be **absent in headless/cron** runs, so it cannot be a headless default.

**All notify sends fail silently.** A notification that cannot be delivered (`rk`/run-kit server unreachable, channel down, no subscriptions, `curl`/tool missing) MUST NOT crash or stall the loop — the operator logs one line and keeps ticking. `rk notify` is already fail-silent by contract; the fallback path matches it. This mirrors the `_preamble.md` § Run-Kit (rk) Reference "fail silently" discipline.

### Sending Auto-Answers

Before `tmux send-keys`: verify pane exists and agent is still idle (§3 steps 1-2), then re-capture the terminal. If output changed since detection, abort — agent is no longer waiting.

### Idle Auto-Default on Strategic Escalations

This is the watchdog for a **left-open** Strategic prompt — the no-defensible-default branch of **Non-Blocking Strategic Handling** above. (Auto-picked Strategic prompts are already resolved, so the watchdog has nothing to act on for them.) When rule 4 leaves a prompt open as **Strategic**, the operator starts a per-prompt idle timer measured in real time from the moment the left-open log line is written. If the prompt remains idle for 30 minutes, the operator auto-answers the prompt and logs using the distinct `auto-defaulted` format (§5 Logging). The timer runs in the background — it does **not** block the loop; the operator keeps ticking and fires the auto-default on whatever later tick crosses the 30-minute mark.

**Threshold**: 30 minutes, hardcoded. No operator-state-file field, no per-change override, no environment variable exposes this value. The §4 operator state file schema is unchanged.

**Idle clock reset**: the idle timer resets on any terminal-state change in the pane — new content appended by the agent, user keystrokes that alter the prompt display, or the prompt's own redraw. The timer is a watchdog on pane-idle-ness, not on escalation-open-ness. Tick cadence already provides sub-minute resolution via §4 Tick Behavior — no new polling infrastructure is required.

**Answer selection** (in priority order):

1. If the prompt text visibly states a default (e.g., `(default: 2)`, `Press enter for 2`, `[2]`), send that stated default.
2. Otherwise, send `1`.

This matches rule 4's existing "first/default" semantics for routine menus.

**Scope (hard exclusion)**: the idle auto-default applies ONLY to **left-open** Strategic prompts from rule 4's no-defensible-default branch. It does NOT apply to auto-picked Strategic prompts (already resolved) and MUST NOT apply to rule 6 escalations ("cannot determine keystrokes") — the operator does not know what the correct keystrokes are, so sending `1` or the stated default would emit nonsense into the pane. Rule-6 escalations remain open pending user action regardless of idle duration.

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
2. **Create worktree** — run `wt create --non-interactive --worktree-name <wt> [<branch>]` **with the target repo as the working directory** (so the worktree lands under `$(dirname <target-repo>)/<repo-name>.worktrees/`, not the operator's repo). The operator never relies on its own CWD for spawning.
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

   - **Existence guard is mandatory.** When the change folder/intake does not exist yet (the raw-text and backlog entry forms, before `/fab-new` runs inside the spawned agent), the operator MUST NOT attempt a switch — there is nothing to switch to, and `/fab-new` creates and then activates the change itself once the agent runs (activation at fab-new Step 10).
   - **Scoped to the new worktree — no cross-tab collision.** The switch runs with the just-created worktree as CWD, so it writes *that worktree's* `.fab-status.yaml` — never the operator's own checkout or any other worktree. Each operator worktree is a dedicated, single-change checkout that owns its own per-worktree pointer file, so there is zero cross-tab collision risk (the very concern the transient-override path protects against — parallel tabs targeting different changes via one shared pointer — does not arise within a single dedicated worktree).
   - **Fail-soft.** A `fab change switch` failure is non-fatal to the spawn — log one line and continue opening the agent tab. The transient `<change>` override on the embedded pipeline command still makes the pipeline resolve correctly even if the pointer write failed; the activation is an ergonomic enhancement, not a correctness prerequisite.
4. **Resolve dependencies** — if the change has a non-empty `depends_on` list, resolve it per repo: same-repo deps cherry-pick into the worktree, cross-repo deps are ordering-only barriers (see Dependency Resolution below)
5. **Read the target repo's spawn command** — run `fab spawn-command --repo <target-repo>` to read **that repo's** `agent.spawn_command` (default: `claude --dangerously-skip-permissions`). Do NOT use the operator's own `config.yaml` — each repo may configure a different spawn command.
6. **Open agent tab** — `tmux new-window -n "»<wt>" -c <worktree-path> "<spawn_cmd> '<command>'"` (where `<wt>` is the worktree name from step 2 and `<spawn_cmd>` is the target repo's command from step 5)
7. **Enroll in monitored set** — unconditionally and silently record pane, **repo** (the target repo from step 1), **session** (the tmux session the new window landed in), stage, branch, depends_on in the state file; add `{ branch, repo }` to `branch_map`. MUST NOT prompt the user about whether to monitor. (Enrollment calls `fab pane window-name ensure-prefix <pane> »` per §4; the `»<wt>` name produced in step 6 already satisfies the primitive's idempotent prefix check, so no duplicate rename occurs.)

Window markers (`»` / `›`) are **unchanged** by the multi-repo model — they key on server-global pane IDs, which are unique across every repo and session on the server.

> **Auto-enroll is mandatory.** Every spawned agent MUST be enrolled in the monitored set immediately as part of the spawn sequence. The operator MUST NOT ask the user whether to monitor a spawned agent — this decision is already made by the act of spawning. If the operator spawned it, it is monitored. No exceptions.

### Dependency Resolution

Dependency resolution is **two-tier**, split by repo. Each entry in `depends_on` is classified by comparing the dependency's `repo` (from its `branch_map` `{ branch, repo }` pair, or the dep's monitored entry) against **this change's** `repo`:

- **Same-repo dependency** (`dep.repo == change.repo`) → **cherry-pick** the dependency's code into the worktree, exactly as today.
- **Cross-repo dependency** (`dep.repo != change.repo`) → **ordering-only barrier**: the operator waits until the dependency reaches its `stop_stage` (a terminal stage when `stop_stage` is null), then spawns the dependent agent. **No code is merged.**

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

**Why `origin/{default_branch}` as base (same-repo only)**: Each same-repo dependency branch carries its full transitive same-repo dependency content. When the operator spawned dep B, it cherry-picked dep A into B's worktree first. B's branch therefore contains A's commits. So `origin/{default_branch}..<B-branch>` gives the complete transitive closure within the repo — no need to chase transitive same-repo deps manually. This is why only direct/leaf same-repo dependencies need cherry-picking. (Cross-repo deps carry no such transitive content — they are ordering-only.)

### Dependency Declaration

Dependencies are declared through three conversational paths, all of which coexist:

1. **Explicit**: "cd34 depends on ab12" — operator sets `depends_on: [ab12]` on the monitored entry
2. **Autopilot queue (implicit)**: user-provided ordering implies `--base` chaining by default — every change after the first automatically gets `depends_on: [<nearest-same-repo-predecessor>]`: the closest earlier queue entry in the **same repo** (cherry-picked), falling back to the immediately previous entry when no earlier entry shares the repo (cross-repo → ordering-only barrier, no code)
3. **`--base` flag (explicit)**: autopilot `--base <prev-change>` explicitly sets `depends_on: [<prev-change-id>]` for the subsequent change (matches path 2's pick when the previous entry is same-repo; available for ad-hoc overrides)

### Working a Change

> **Pipeline-first routing (§1):** All three work paths below MUST go through the fab pipeline. For *new* work, this means `/fab-new` followed by a pipeline command; for already-intaked changes, start from the appropriate pipeline command stage instead of repeating `/fab-new`. The operator MUST NOT send raw implementation instructions directly to agent panes. See the "Pipeline-first routing" principle in §1.

The operator accepts work in three forms. Each runs the §6 spawn sequence above (establish target repo → `wt create` in it → existence-guarded pointer activation → resolve dependencies → `fab spawn-command --repo <target-repo>` → open agent tab → enroll with `repo` + `session`); only the entry-form specifics below differ:

| Entry form | Target repo / pre-step | Initial command (sent via the spawn sequence's agent tab) |
|------------|------------------------|-----------------------------------------------------------|
| **Existing change** (already has intake or further) | The change's `repo` (monitored entry or `branch_map`) | `/fab-fff <change>` — embed exactly **one** slash command in the spawn (do NOT send a `&&`-joined string like `/fab-switch <change> && /fab-fff`). The embedded command is delivered as a single prompt to the spawned agent, where `&&` is not a shell operator and Claude reads one leading `/command` per prompt — so the `&& …` tail is swallowed into `/fab-switch`'s argument rather than running as a second command. (Two sequential slash commands *are* achievable via separate Enter-terminated sends, but the operator deliberately avoids that here: a slash-command switch is a full agent round-trip for a one-line symlink write, and re-adding a post-spawn send would regress the single-dispatch-at-spawn property. The synchronous `fab change switch` CLI verb in spawn-sequence step 3 does the same write directly — see below.) The change-name override targets the change directly, no `/fab-switch` needed; the worktree's branch already matches (created by `wt create … <change-folder-name>`); `/fab-fff` picks up from the change's current stage. The spawn sequence's existence-guarded step 3 **also activates the pointer** (`fab change switch <change>` in the new worktree) so the finished worktree is self-describing — but it is still the transient `<change>` override on `/fab-fff`, not the pointer, that targets the pipeline (the activation is an ergonomic add-on; the override remains the load-bearing mechanism) |
| **Raw text** (e.g., "fix login after password reset") | The repo the user names; default the operator's launch repo | `/fab-new <shell_escaped_description>` — the raw description safely shell-escaped for inclusion in a single-quoted shell argument (do NOT insert unescaped raw text directly). No operator pointer-switch at spawn — the change folder doesn't exist yet, so §6 step 3's existence guard skips it; `/fab-new` creates and then activates the change inside the spawned agent (activation at fab-new Step 10) |
| **Backlog ID or Linear issue** (structured) | Pre-step: look up the idea (`idea show <id>`) or resolve the Linear issue first | `/fab-new <id>` — same as raw text: no operator pointer-switch at spawn (§6 step 3's guard skips the not-yet-existing folder); `/fab-new` owns activation inside the spawned agent |

On completion (all three): PR ready, optionally archive. Both raw text and backlog paths use `/fab-new` to generate a proper intake with traceability. `/fab-new` captures the raw input in the intake's Origin section — the user just says "fix [description]" and the operator does the rest.

### Autopilot

User provides a queue of changes. Confirmation prompt reflects the active mode:
- **Default (stack-then-review):** "Confirm upfront (creates PRs — merge after review)."
- **`--merge-on-complete`:** "Confirm upfront (merges PRs on completion)."

A queue **may span repos**. The dependency semantics are mixed: implicit `--base` chaining (and explicit `depends_on`) cherry-picks **within a repo** and **degrades to an ordering-only barrier across repo boundaries** (per Dependency Resolution above). Implicit chaining picks each change's **nearest same-repo predecessor** — not blindly the previous queue entry, which would silently break same-repo stacking whenever a cross-repo entry sits in between (a cross-repo dep contributes no code). So a chain `ab12 → cd34 → ef56` where `cd34` lives in a different repo means: `cd34` gets `depends_on: [ab12]` (cross-repo — waits for `ab12` to reach its stop/terminal stage, no code), and `ef56` (back in `ab12`'s repo) gets `depends_on: [ab12]` — its nearest same-repo predecessor — and cherry-picks from it; queue order still runs `ef56` after `cd34`.

Queue ordering:

| Strategy | Description |
|----------|-------------|
| User-provided | Run in the exact order given. Implicit `--base` chaining by default: every change after the first gets `depends_on: [<nearest-same-repo-predecessor>]` — the closest earlier queue entry in the same repo (cherry-picked); when no earlier entry shares the repo, the immediately previous entry (cross-repo → ordering-only). No explicit `--base` flag required. |
| Confidence-based | Sort by confidence score descending. Highest-confidence first (independent changes) |
| Hybrid | User provides constraints (partial order); operator sorts unconstrained by confidence |

**`--merge-on-complete`** — opt-in flag that reverts to the previous merge-as-you-go behavior: merge each PR on completion, then `git fetch origin` and rebase the next change onto `origin/{default_branch}` (the default branch resolved per Dependency Resolution step 0 — never a hardcoded `origin/main`). Implicit `--base` chaining is disabled under this flag — each change rebases onto `origin/{default_branch}` independently instead of stacking on the previous change's branch. Natural language equivalents: "merge as you go", "merge on complete", "merge each when done". Without this flag, the default is stack-then-review: PRs are created but not merged until the user explicitly requests merging, and implicit `--base` chaining is active (every change after the first gets `depends_on: [<nearest-same-repo-predecessor>]`, falling back to the immediately previous entry across repo boundaries).

The operator works each change through the pipeline. Pre-send validation (§3) applies to any command sent to an existing pane; the initial pipeline command itself is **embedded at spawn** (§6 step 6) — the single dispatch point:

1. **Gate** — check confidence score **before anything spawns**. If below threshold, flag and wait — no worktree, no tab, no dispatch for a below-threshold change
2. **Spawn** — run the §6 spawn sequence steps 1–2 (establish the change's target repo, create worktree in it; `--reuse` for respawns)
3. **Resolve dependencies + open tab + enroll** — §6 spawn sequence steps 3–7 (existence-guarded pointer activation, same-repo cherry-pick / cross-repo ordering-only barriers per Dependency Resolution). Step 6's `<command>` is the change's pipeline command — `/fab-fff <change>` (or the appropriate command for its current stage) — so the dispatch happens **once, at spawn**; do NOT send the command again after the tab opens
4. **Monitor** — normal tick detection handles progress
5. **Record** — on completion, record `{ branch, repo }` in `branch_map`, collect PR URL
6. **Spawn next** — repeat from item 1 for the next queue entry (with implicit `depends_on: [<nearest-same-repo-predecessor>]` per Queue ordering above; deps resolved per repo — cherry-pick same-repo, barrier cross-repo); its command is likewise embedded at spawn
7. **Report** — `"ab12: PR ready. 1 of 3 complete. Starting cd34."`
8. **(After all complete) Summary** — list all PR links with per-repo dependency annotations and per-repo merge order suggestion (see Queue Completion Summary below)

When `--merge-on-complete` is active, steps 5–8 revert to the previous merge-as-you-go behavior: merge PR on completion, `git fetch origin`, rebase next change onto `origin/{default_branch}` (resolved per Dependency Resolution step 0), report merge.

Autopilot-driven changes display `▶` in the status frame (§4). Queue progress is visible from the list — entries with `▶` that show ✓ (green) are complete, the one showing ● (green) / ◌ (yellow) is current.

#### Queue Completion Summary

When all changes in a stack-then-review autopilot queue complete, the operator displays a completion summary. When the queue spans repos, each PR is **annotated with its repo**, and the suggested merge order respects **each repo's own dependency chain** (a per-repo PR sequence):

```
Queue complete. 3 PRs ready for review:
1. ab12: <PR-URL-1> (~/code/foo, base)
2. cd34: <PR-URL-2> (~/code/bar, ordering-only after ab12)
3. ef56: <PR-URL-3> (~/code/foo, depends on ab12)
Merge per-repo: foo 1→3, bar 2 (after foo:1 reaches main). Or ask me to merge all.
```

For a single-item queue: `"ab12: PR ready. Queue complete."`

#### Ordered Merge

When the user says "merge all" or "merge the queue" after a stack-then-review queue completes, the operator merges PRs respecting **per-repo PR sequences** — within each repo, base-first in dependency order; across repos, cross-repo ordering barriers are honored (a cross-repo dependent's PR is merged only after its barrier dependency reaches its target repo's main). It waits for CI to pass on each PR before proceeding to the next in that repo's sequence:

1. Merge `~/code/foo` PR 1 (base) — wait for CI pass
2. Merge `~/code/bar` PR 2 (its cross-repo barrier `foo:1` is now on main) — wait for CI pass
3. Merge `~/code/foo` PR 3 — wait for CI pass

Report each merge with its repo: `"ab12: merged (foo 1/2)"`, `"cd34: merged (bar 1/1)"`, `"ef56: merged (foo 2/2)"`.

**CI failure during ordered merge (halt-dependents-only)**: If CI fails on a PR, the operator halts **that repo's merge sub-sequence** AND **any repo whose queued items carry a cross-repo `depends_on` into the failed chain — transitively**. "Dependent" is determined over the cross-repo `depends_on` graph: a repo halts if any of its queued items depends (directly, or via another already-halted item) on a PR in the failed chain. **Truly independent repos' sub-sequences continue merging.** The operator does not abandon the queue; it isolates the blast radius to the failure's dependency cone. On completion it reports which sub-sequences halted vs. completed and escalates the failure to the user:

```
ab12: CI failed (~/code/foo). Halted: foo sub-sequence; bar (cross-repo dep into foo). Completed: baz sub-sequence (2 PRs merged). Fix foo and retry.
```

Autopilot state (queue, current, completed) persists in the operator state file.

**Failures**: review exhausted → skip. Rebase conflict → skip (`--merge-on-complete` only; does not apply in default stack-then-review mode since there are no rebase steps). Cherry-pick conflict → escalate (do not skip). Pane dies → 1 respawn (`--reuse`), then skip. Stage timeout (>30m) → flag. Total timeout (>2h) → flag.

**Interrupts**: "stop after current", "skip <change>", "pause", "resume" — acknowledged immediately.

---

## 7. Watches

Watches are standing instructions to monitor an external source and take action when new items appear. Users create watches conversationally: "watch Linear project DEV for new issues, spawn agents, stop at intake."

### Schema

Each watch in the operator state file has:

| Field | Description |
|-------|-------------|
| `enabled` | `true` or `false` — paused watches retain config but skip tick evaluation |
| `source` | `linear` or `slack` — determines which MCP tool to query |
| `query` | Source-specific API filter (project, status, assignee, channel) — passed to MCP |
| `target_repo` | Absolute main-worktree root the watch's spawned changes land in. Required for a spawning watch — the spawn sequence (§6) uses it as the target repo. A watch with no `target_repo` cannot spawn |
| `stop_stage` | How far to go: `intake`, `apply`, `hydrate`, or `null` (full pipeline) |
| `known` | Already-handled item IDs — managed automatically, capped at 200 (oldest pruned first) |
| `completed` | Items that reached `stop_stage` — lets users query "what did this watch produce?" |
| `last_checked` | ISO timestamp of last successful query |
| `last_error` | Last error message, or `null`. Shown in status frame when set |
| `instructions` | Free-form natural language — trigger conditions, concurrency limits, label filters, anything else |

Structured fields handle machine-readable concerns; `instructions` handles everything the operator evaluates as an LLM. Concurrency limits in `instructions` are enforced by counting monitored entries where `spawned_by` matches the watch name.

### Tick Behavior

On each tick (step 3), for each enabled watch:

1. **Query source** — Linear via MCP (`mcp__claude_ai_Linear__list_issues`), Slack via MCP (`mcp__claude_ai_Slack__slack_read_channel`), using `query` as the API filter. On failure: set `last_error`, skip this watch for this tick. After 3 consecutive failures: disable the watch, alert user.
2. **Deduplicate** — skip items in `known` **plus** `completed` lists (an item that reached `stop_stage` moves from `known` to `completed` but may still match the query — it MUST NOT be respawned). Update `last_checked`.
3. **Evaluate instructions** — apply trigger conditions, label filters, concurrency limits (count monitored entries with `spawned_by: <watch-name>`), and any other criteria from `instructions`
4. **Act** — for each item that passes:
   - Run the §6 spawn sequence with the watch's `target_repo` as the target repo, sending the appropriate initial command (e.g., `/fab-new DEV-123`)
   - Enroll in monitored set with `repo` (= `target_repo`), `session`, `stop_stage`, and `spawned_by` from the watch
   - Add item ID to `known` (only after successful spawn)
   - Prune `known` if over 200 entries (drop oldest)
5. **Report** — `"Watch linear-bugs: DEV-1024 — Fix auth redirect (72m old). Spawning."`

When a watch-spawned agent reaches its `stop_stage`, move the item ID from `known` to `completed` and report: `"Watch linear-bugs: DEV-1024 completed intake."`

### Conversational Management

- "Watch Linear project DEV for bugs older than 1 hour, **spawn into ~/code/foo**, stop at intake" → creates watch with `target_repo: ~/code/foo`
- "Pause the Linear watch" / "Resume the Linear watch" → toggles `enabled`
- "Stop watching Linear" → removes watch
- "Spawn the Linear watch's changes into ~/code/bar instead" → updates `target_repo`
- "What are you watching?" → lists active watches with their `target_repo`, instructions, and completed items
- "What did linear-bugs produce?" → lists `completed` items
- "Test watch linear-bugs" → dry-run: query, deduplicate, evaluate instructions, report what *would* happen without spawning or updating state
- "Change the Linear watch to go through full pipeline" → updates `stop_stage` to null
- "Also limit to 2 concurrent agents" → appends to `instructions`

---

## 8. Configuration

### One Operator Per Server

The isolation unit is the **tmux server**. There is exactly **one operator per tmux server** — it spans every session and every repo on that server, coordinating all of them through a single server-keyed state file (§4, §9). This matches the server-wide singleton already enforced by the `operator` window (`fab operator` switches to the existing window rather than creating a second one).

- **Multiple sessions, same server** share one operator and one state file. The operator addresses their agents by the `(session, repo, pane)` tuple (§1); there is no per-session or per-repo operator.
- **A second operator means a second tmux server** — start one on a separate socket (`tmux -L <label>`). Its state file is keyed by that socket, so the two operators never collide. There is no `--name` dimension; the server boundary is the only isolation knob.

### Settings

| Setting | Default | Override via natural language |
|---------|---------|------------------------------|
| Loop interval | 3m | "check every {N}m" |
| Stuck threshold | 15m | "flag agents stuck for more than {N}m" |
| Menu-detected heartbeat | 90s | "tighten to {N}s when an agent is on a menu" |
| Notify channel | `rk` (run-kit Web Push; auto-fallback when `rk` absent) | "notify via ntfy topic {topic}" / "notify via discord {url}" / "notify via push" |

Session-scoped — resets on `/clear` or session restart. The §4 operator state-file schema is **unchanged** (these are session settings, consistent with the loop-interval / stuck-threshold rows). The **strategic auto-default threshold stays hardcoded at 30m** (§5) — there is deliberately **no** setting for it.

---

## 9. Key Properties

| Property | Value |
|----------|-------|
| Requires active change? | No |
| Runs preflight? | No |
| Read-only? | No — sends commands, auto-answers, writes the operator state file |
| Idempotent? | Yes — state re-derived every tick |
| Advances stage? | No |
| Outputs `Next:` line? | No — ends with ready signal |
| Loads change artifacts? | No — coordination context only |
| Requires tmux? | Yes — hard stop without it |
| Requires a git repo? | No — `fab operator` opens its window in the repo root inside a repo, else `os.Getwd()` (neutral parent dir). Errors only if both fail |
| Requires a `fab/` project? | No — spawn command comes from the project's `agent.spawn_command` when `fab/` is resolvable, else `spawn.DefaultSpawnCommand` (`claude --dangerously-skip-permissions`). No project `agent.spawn_command`/`agent.tiers` is read on a `fab/`-less launch |
| Coordinating-agent model | Doing tier — `fab operator` resolves `fab resolve-agent apply` (canonical doing-tier stage), appends `--model`/`--effort`; falls back to the built-in `{claude-opus-4-8, high}` on any failure (incl. no resolvable `fab/` project) |
| Uses `/loop`? | Yes — adaptive heartbeat: `3m` normally, tightens to `90s` (§8) when any monitored agent is menu-waiting, relaxes back to `3m`; one loop at a time |
| Uses the operator state file? | Yes — monitored set + autopilot queue + branch map persistence. **Server-keyed**, not repo-rooted: `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state/fab/operator/<server-slug>.yaml`), keyed by the tmux socket path. The binary derives the path; old repo-rooted files are not migrated |
| Multi-repo / multi-session? | Yes — one operator per tmux server spans all its sessions and repos via the `(session, repo, pane)` addressing tuple |
