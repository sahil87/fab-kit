---
description: "Operator coordination skill (`/fab-operator`, superseding the historical operator4) — multi-repo / multi-session coordination on one tmux server via the `(session, repo, pane)` addressing tuple, server-keyed state file, multi-agent monitoring, auto-answer model with strategic escalation, repo-targeted spawning, two-tier dependency resolution, repo-spanning autopilot, and tmux tab-naming. Historical operator4 context plus the operator design-decision lineage."
---
# Operator

**Domain**: runtime

## Overview

The operator is a standalone, long-lived coordination skill (`/fab-operator`) — NOT a pipeline stage. It runs in a dedicated tmux pane, coordinates agents **across multiple repos and multiple tmux sessions on a single tmux server**, observes them via `fab pane map --all-sessions`, routes commands via `tmux send-keys`, monitors progress via `/loop`, and auto-answers idle agent prompts (escalating strategic ones to the user). Every agent is addressed by the `(session, repo, pane)` tuple: the **pane ID is the primary key** (server-global, stable), with `repo` (the agent's absolute main-worktree root) and `session` (its tmux session name) as added dimensions. There is **one operator per tmux server**, owning **one server-keyed state file** that spans every repo it coordinates. This file documents the operator's behavior, its tick lifecycle and state model, and the design-decision lineage from the historical `/fab-operator4` through the current multi-repo `/fab-operator`. For the `.fab-runtime.yaml` agent schema and `fab pane` primitives the operator builds on, see [runtime-agents.md](runtime-agents.md) and [pane-commands.md](pane-commands.md). For the Go-side mechanism (server-keyed state path derivation, the `repo` JSON field, `fab spawn-command --repo`), see [kit-architecture.md](../distribution/kit-architecture.md) → "Operator State File" and [pane-commands.md](pane-commands.md).

## Requirements

### `/fab-operator4` (Standalone Coordination Skill) — *superseded by `/fab-operator`*

> **Note**: Operator4 has been superseded by `/fab-operator` (v7). The skill file and launcher script have been removed. This section is preserved as historical context for the design decisions that evolved into the current operator. See `fab-operator.md` for authoritative behavior.

`/fab-operator4` was a standalone, self-contained coordination skill — NOT a pipeline stage. It ran as a long-lived Claude session in a dedicated tmux pane, observing agents via `fab pane-map`, routing commands via `tmux send-keys`, monitoring progress via `/loop`, and auto-answering idle agent prompts.

Operator4 was the first standalone operator skill. Previous iterations (operator1, operator2, operator3) were removed — their behavior was fully inlined into operator4 as a standalone file.

#### Principles

**Coordinate, don't execute.** The operator routes user instructions to the right agent — it never implements work directly. If the target is ambiguous, ask.

**Not a lifecycle enforcer.** Individual agents self-govern via their own pipeline skills. The operator does not validate stage transitions or enforce pipeline rules.

**Context discipline.** The operator never reads change artifacts (intakes, specs, tasks). Its context window is reserved for coordination state — pane maps, stage snapshots, monitoring state.

**State re-derivation.** Before every action, re-query live state via `fab pane-map` (or `wt list` + `fab change list` outside tmux). Panes die, stages advance, agents finish — stale state leads to wrong actions.

#### Context Loading

The operator loads the always-load layer (`_preamble.md` §1) plus `$(fab kit-path)/skills/_cli-external.md` (external tool reference for `wt`, `tmux`, and `/loop` — loaded only by operator, not by pipeline skills). It does NOT run preflight. It does NOT load change-specific artifacts.

#### Orientation

On invocation, runs `fab pane-map` and displays the output, then signals readiness. Outside tmux (`$TMUX` unset), falls back to `wt list` + `fab change list` for status queries only — monitoring is disabled.

#### Safety Model

| Tier | Examples | Behavior |
|------|----------|----------|
| Read-only | Status check, pane map | No confirmation |
| Recoverable | Send `/fab-continue`, rebase | Announce before sending |
| Destructive | Merge PR, archive, delete worktree, autopilot | Confirm before executing |

**Pre-send validation**: Before sending keys to any pane, the operator MUST (1) verify the pane exists via refreshed pane map (dead panes fail silently), (2) check the agent is idle via the Agent column. If busy, warn and require explicit confirmation.

**Bounded retries**: Every automatic action has a bounded retry count. Unbounded retries compound errors.

| Situation | Max retries | Escalation |
|-----------|-------------|------------|
| Stuck agent nudge | 1 | "Appears stuck at {stage}. Manual investigation recommended." |
| Rebase conflict | 0 | Immediately flag to user |
| Pane death (non-autopilot) | 0 | Report pane gone. No respawn outside autopilot |
| Send to busy agent | 0 | Warn user, require explicit confirmation |

#### Monitoring System (historical operator4 framing — see "Multi-Repo Monitoring Model" below for the current model)

Operator4 maintained a monitored set persisted to a repo-rooted `.fab-operator.yaml` (mirrored in conversation context for the active tick), each entry tracking: change ID, pane, last-known stage, last-known agent state, enrolled-at timestamp, last-transition-at timestamp. The enrollment/removal mechanics (window-name prefix on enroll, done-marker swap on removal) below are unchanged by the multi-repo reframe — only the addressing model and state-file location moved (see the next section).

**Enrollment triggers**: operator sends a command to it, user requests monitoring, operator triggers an automatic action toward it (including autopilot and watch spawns). Read-only actions do not enroll. **Spawned agents are always auto-enrolled** — the operator MUST NOT ask the user whether to monitor a spawned agent. This constraint is reinforced in both the §1 principles and the "Spawning an Agent" procedural subsection to ensure proximity-based LLM adherence.

**Enrollment also applies the `»` window-name prefix**: after writing the monitored entry to the state file, the operator invokes `fab pane window-name ensure-prefix <pane> »` (U+00BB). The primitive enforces the literal-prefix idempotent guard internally: operator-spawned windows (already `»<wt>` from the spawn step), `/clear`-restored entries, and re-enrollment after transient removal all no-op through the guard. A non-zero exit — pane vanished between refresh and rename (exit 2) or any other tmux error (exit 3, including tmux not running / socket unreachable) — is logged as `"{change}: window rename skipped ({error})."` and does not roll back the enrollment. Window markers (`»` / `›`) are **unchanged by the multi-repo model** — they key on server-global pane IDs, which are unique across every repo and session on the server.

**Removal triggers**: change reaches a terminal stage (hydrate, ship, review-pr), pane dies, user explicitly stops monitoring. On every removal path the operator invokes `fab pane window-name replace-prefix <pane> » ›` (U+203A, single right guillemet — the done-marker), swapping the active-monitoring `»` prefix for the trail-preserved `›`. The primitive's literal-prefix guard protects user-renamed windows: if the user renamed the window mid-monitoring so it no longer starts with `»`, the swap silently no-ops without clobbering the user's name. Exit 2 (pane missing — the window is gone anyway) is treated as a successful removal; other non-zero exits log `"{change}: window rename skipped ({error})."`. This keeps the tab bar an accurate at-a-glance map: `»` for currently-tracked, `›` for operator-touched-but-done, and untouched names for windows the operator never marked.

#### Multi-Repo Monitoring Model (current `/fab-operator`)

> Introduced by `260607-oy0k-operator-multi-repo-skill`; consumes the Go primitives shipped by `260607-h3jk`. This is the authoritative monitored-set model; the operator4 framing above is historical context.

**One operator per tmux server.** The isolation unit is the tmux server — a single operator spans every session and every repo on that server. A second operator means a second tmux server (`tmux -L <label>`). "Multiple sessions, same server" share one operator and one state file. There is no `--name` dimension; the server boundary is the only isolation knob. This matches the server-wide singleton already enforced by the `operator` window (`fab operator` switches to the existing window rather than creating a second one).

**Server-keyed state file.** The monitored set, autopilot queue, `branch_map`, and watches persist in **one server-keyed file** — `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state/fab/operator/<server-slug>.yaml`), keyed by the tmux socket path so the one operator-per-server owns one file across all the repos it coordinates. The binary derives this path (`fab operator tick-start` reads/writes it via `StatePath()`); the operator never computes it. See [kit-architecture.md](../distribution/kit-architecture.md) → "Operator State File" for the derivation mechanism. Old repo-rooted `.fab-operator.yaml` files from before the server-keyed model are **not migrated** — they are abandoned in place (the monitored set is re-derivable from live `»`-prefixed panes).

**`(session, repo, pane)` addressing.** Every monitored agent, `branch_map` value, and watch is repo-qualified. The pane ID remains the primary key (server-global, stable); `repo` (absolute main-worktree root) and `session` (tmux session name) are added dimensions, not replacements. Schema additions over the operator4 entry:

- Each `monitored` entry gains `repo` (absolute main-worktree root) and `session` (tmux session name).
- The `branch_map` value becomes `{ branch, repo }` (was a bare branch string). The `repo` is required to disambiguate a dependency's branch across repos and to choose same-repo (cherry-pick) vs. cross-repo (ordering-only) dependency resolution.
- Each `watches` entry gains `target_repo` — the repo a watch's spawned changes land in. A watch with no `target_repo` cannot spawn.

#### Multi-Repo Coordination: Spawning, Dependencies, Autopilot (current `/fab-operator`)

> Introduced by `260607-oy0k-operator-multi-repo-skill`. Supersedes the single-repo spawning/autopilot prose in the historical operator4 sections below.

**Repo-targeted spawning.** Every spawn flow first establishes **which repo** the work targets (an existing change's `repo`, a watch's `target_repo`, or the repo the user names — defaulting to the operator's launch repo), then runs each step against that repo, not the operator's own:

1. Run `wt create --non-interactive` **with the target repo as the working directory**, so the worktree lands under `$(dirname <target-repo>)/<repo-name>.worktrees/` rather than the operator's repo.
2. Read **that repo's** `agent.spawn_command` via `fab spawn-command --repo <target-repo>` (see [kit-architecture.md](../distribution/kit-architecture.md)) — never the operator's own `config.yaml`, since each repo may configure a different spawn command.
3. Open the agent tab and enroll with `repo` and `session` recorded, plus `{ branch, repo }` added to `branch_map`.

All three work paths (existing change, raw text, backlog/Linear) and watch-driven spawns use this same repo-targeted sequence.

**Two-tier dependency resolution.** Each `depends_on` entry is classified by comparing the dependency's `repo` (from its `branch_map` `{ branch, repo }` pair, or its monitored entry) against this change's `repo`:

- **Same-repo dependency** → **cherry-pick** as today: `git cherry-pick --no-commit origin/main..<dep-branch>` into the worktree.
- **Cross-repo dependency** → **ordering-only barrier**: wait until the dependency reaches its `stop_stage` (terminal stage when `stop_stage` is null), then spawn. **No code is merged.**

> **REQUIRED caveat — cross-repo deps give the dependent agent NO code.** A cross-repo `depends_on` is a pure *sequencing* constraint; the dependent worktree receives nothing from the dependency. This is correct only for **logical** dependencies ("don't start the frontend change until the API change merges"), never for **code-level** ones. Cross-repo branches share no common `origin/main` base to cherry-pick across, so there is no sound way to make the dependency's code available. For code sharing across repos, the dependency must merge and be consumed as a normal upstream artifact (package, vendored copy), outside the operator's scope.

Ancestor-pruning (`git merge-base --is-ancestor`) is scoped to the **same-repo subset** of the dependency set — it is meaningless across repos with no shared history. The `origin/main..<dep-branch>` transitive-closure argument (only direct/leaf deps need cherry-picking) holds only within a repo; cross-repo deps carry no such transitive content.

**Repo-spanning autopilot.** An autopilot queue **may span repos** with mixed dependency semantics: implicit `--base`/`depends_on` chaining cherry-picks **within** a repo and **degrades to an ordering-only barrier across** repo boundaries. Ordered merge tracks **per-repo PR sequences** — within each repo, base-first in dependency order; across repos, cross-repo barriers are honored (a cross-repo dependent's PR merges only after its barrier dependency reaches its target repo's main). The queue-completion summary annotates each PR with its repo and suggests a per-repo merge order.

**CI-failure = halt-dependents-only.** During ordered merge, a CI failure halts the failing repo's merge sub-sequence AND any repo whose queued items carry a cross-repo `depends_on` into the failed chain — **transitively** over the cross-repo `depends_on` graph (a repo halts if any of its queued items depends, directly or via another already-halted item, on a PR in the failed chain). **Truly independent repos' sub-sequences continue merging.** The operator isolates the blast radius to the failure's dependency cone, reports which sub-sequences halted vs. completed, and escalates the failure to the user.

**`/loop` lifecycle**: Start when first change enrolled (no loop running) — `/loop 5m "check monitored agents"`. Stop when monitored set empty. One-loop invariant: at most one active `/loop` at any time.

**Tick snapshot is server-wide.** The tick's snapshot step uses `fab pane map --all-sessions --json` (not bare `fab pane map`), so the operator sees agents in **every** session on its server, not just its own. `--json` exposes the per-row `repo` field (the agent's absolute main-worktree root, em dash when the pane is not in a git repo — see [pane-commands.md](pane-commands.md)). Rows are grouped first by `repo`, then by `session`. The health glyphs, autopilot `▶` marker, and `⚠` stuck marker are unchanged — only the grouping changes.

**Repo-section status frame.** The status frame renders **repo-section headers** — one header line per repo (noting the session) with the repo's change rows indented beneath — rather than per-row `repo`/`session` columns (chosen for scannability). Watches render after all repo sections in a flat list, each annotated with its `target_repo` (`→ ~/code/foo`). A pane whose main-worktree root could not be resolved renders under an `(unresolved repo)` header rather than being dropped. Example:

```
── Operator ── 17:32 ── tick #47 ── 7 tracked ──

  ~/code/foo (session: work)
    [change]  r3m7   ▶ ● apply → review
    [change]  ab12     ✓ hydrate
  ~/code/bar (session: side)
    [change]  k8ds   ▶ ◌ review · idle 8m
  [watch]   linear-bugs  → ~/code/foo   ● 2 known · 1 completed · 3m ago
```

**Monitoring tick** (on each `/loop` tick or "any updates?"):

1. **Stage advance detection** — compare current stage to last-known. Report transitions, update baseline.
2. **Pipeline completion detection** — stage is hydrate, ship, or review-pr. Report and remove from monitored set.
3. **Review failure detection** — stage went from review back to apply. Report rework.
4. **Pane death detection** — change no longer in pane map. Report and remove from monitored set.
5. **Auto-nudge** — for each idle agent, run question detection and answer model (see below). If a monitored agent was spawned for a new change from backlog and the tick detects the change has advanced past intake, send `/git-branch` to that agent's pane (aligns branch name with newly created change folder).
6. **Stuck detection** — for agents NOT detected as input-waiting in step 5, check idle duration. If idle at non-terminal stage for >15m, report as potentially stuck. Advisory only — an agent waiting for input is not stuck.

After processing all changes: if the monitored set is empty, stop the loop and report "All monitored changes complete."

#### Auto-Nudge

The operator acts as a proxy for the user on routine operational questions.

**Question detection** — for each idle monitored agent:

1. Capture: `tmux capture-pane -t <pane> -p -S -20` (wide window compensates for line wrapping)
2. Claude turn boundary guard: if `^\s*>\s*$` appears in last 2 lines, skip (normal human-turn boundary)
3. Blank capture guard: if output is entirely blank/whitespace, skip (treat as "cannot determine")
4. Scan for question indicators: lines ending with `?` (tightened — last non-empty line only, <120 chars, skip comment/log prefixes), `[Y/n]`/`[y/N]`/`(y/n)`/`(yes/no)`, `Allow?`/`Approve?`/`Confirm?`/`Proceed?`, Claude Code permission prompts, `Do you want to...`/`Should I...`/`Would you like...`, lines ending with `:`/`:\s*$`, enumerated options (`[1-9]\)` patterns), `Press.*key`/`press.*enter`/`hit.*enter` (case-insensitive)
5. No match → normal idle behavior (stuck detection applies)
6. Match found → proceed to answer model. Bottom-most (most recent) indicator evaluated when multiple match.

**Answer model** — most detected questions are auto-answered. Rule 4 (numbered menus) classifies the prompt before answering: Routine prompts auto-answer, Strategic prompts escalate. Rule 6 escalates when the operator cannot determine what keystrokes to send. Evaluate in order:

1. Binary yes/no or confirmation prompt → `y`
2. `[Y/n]` or `[y/N]` prompt → `y`
3. Claude Code permission/approval prompt → `y`
4. Numbered menu or multi-choice → classify as **Routine** or **Strategic** using LLM judgment over the terminal capture, weighing four signals: option text length, semantic distinctness of options, surrounding agent context, and reversibility of the choice. No hardcoded keyword list, no agent-side sentinel/marker protocol. **Routine** (tool/permission prompts, binary-framed menus, synonymous-option menus) → `1` (first/default option). **Strategic** (multi-option choices representing materially different directions — scope, PR split, pipeline shape, commit organization, spec/approach) → escalate to user. On classification uncertainty, treat as Strategic and escalate (asymmetric cost: false-negative strategic commits the queue to an unchosen direction; false-positive strategic is recovered by the 30m idle auto-default below).
5. Open-ended question where a concrete answer is determinable from visible terminal context → send that answer
6. Question where the operator cannot determine what keystrokes to send → escalate

No cooldown or retry limit — each question is evaluated independently. Worktree isolation and human PR merge provide the safety gate for auto-answered prompts.

**Idle Auto-Default on Strategic Escalations**: When rule 4 escalates a prompt as Strategic, the operator runs a per-prompt real-time idle timer from the moment the escalation log line is written. If the prompt remains idle for 30 minutes, the operator auto-answers and logs using the distinct `auto-defaulted` format (see Logging below).

- **Threshold**: 30 minutes, hardcoded. No `.fab-operator.yaml` field, no per-change override, no environment variable exposes this value — the `.fab-operator.yaml` schema is unchanged.
- **Idle clock reset**: the timer resets on any terminal-state change in the pane — new content appended by the agent, user keystrokes that alter the prompt display, or the prompt's own redraw. The timer watches pane-idle-ness, not escalation-open-ness. Tick cadence already provides sub-minute resolution — no new polling infrastructure is required.
- **Answer selection** (priority order): (1) if the prompt visibly states a default (e.g., `(default: 2)`, `Press enter for 2`, `[2]`), send that stated default; (2) otherwise, send `1`. This matches rule 4's existing "first/default" semantics for routine menus.
- **Scope (hard exclusion)**: applies ONLY to rule 4 Strategic escalations. Rule 6 ("cannot determine keystrokes") escalations MUST NOT trigger the idle auto-default — the operator does not know what the correct keystrokes are, so sending `1` or the stated default would emit nonsense into the pane. Rule-6 escalations remain open pending user action regardless of idle duration.

**Re-capture before send**: Before sending an auto-answer via `tmux send-keys`, MUST re-capture the terminal. If output changed since initial capture, abort — the agent is no longer waiting. Eliminates the race condition between detection and send.

**Logging**: Every auto-answer: `"{change}: auto-answered '{summary}' → {answer}"`. Escalated (rule 6 or rule 4 Strategic): `"{change}: can't determine answer for '{summary}'. Please respond."`. Auto-default after 30m idle on a Strategic escalation (distinct from `auto-answered` for grep-based after-action review): `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"`.

#### Modes of Operation

Every mode follows the same rhythm: interpret user intent → refresh state → validate preconditions → execute → report → enroll in monitoring (if work dispatched).

| Mode | Description |
|------|-------------|
| **Broadcast** | Send command to all idle agents. Filter pane map, announce targets, send to each, enroll all |
| **Sequenced rebase** | "When X finishes, rebase Y on main." Enroll trigger change. When monitoring detects target stage, send rebase, enroll target |
| **Merge PRs** | Merge completed PRs at ship/review-pr stage. Retrieve URLs, confirm (destructive), merge from operator's shell |
| **Spawn agent** | New worktree + agent from backlog idea. Look up idea, create worktree, open tmux tab with Claude session running `/fab-new` |
| **Status dashboard** | Concise summary of all agents: change name, tab, stage, agent state. Include monitored set if active |
| **Unstick agent** | Nudge a stuck agent with `/fab-continue`. Verify idle first. If second nudge for same agent, warn. Send only on explicit insistence |
| **Notification** | "Tell me when X finishes." Enroll in monitoring. Loop handles notification automatically |
| **Autopilot** | Drive a queue of changes through the full pipeline. See below |

#### Autopilot

> The autopilot mechanics below describe the operator4 single-repo framing. Under the current `/fab-operator`, an autopilot queue may span repos with mixed dependency semantics, ordered merge tracks per-repo PR sequences, and CI-failure halts dependents only (not the whole sequence) — see "Multi-Repo Coordination" above for the authoritative behavior.

Drives a queue of changes through the full pipeline — spawning agents, monitoring progress, and collecting PRs for review. The default mode is **stack-then-review**: all queued changes build on each other via implicit `depends_on` chaining, PRs are created but NOT merged until the user explicitly requests merging. Confirm queue before starting (destructive tier). Default confirmation: "Confirm upfront (creates PRs — merge after review)."

**Queue ordering**: User-provided (exact order given), confidence-based (descending score), or hybrid (partial user constraints, confidence tiebreaker). User-provided ordering implies implicit `--base` chaining — each queued change after the first gets `depends_on: [<prev-change-id>]` automatically.

**Per-change loop (stack-then-review, default)**: Spawn worktree (`--reuse` for respawns, `--base <prev-change>` for user-provided ordering) → resolve dependencies (cherry-pick `depends_on` entries into worktree) → open agent tab with `/fab-switch <change>` → gate check confidence (if >= gate, send `/fab-fff`; if < gate, flag to user) → monitor → on completion, record branch in `branch_map`, collect PR URL → dispatch next change (with implicit `depends_on`) → report `"ab12: PR ready. 1 of 3 complete. Starting cd34."`.

**Queue completion summary**: When all changes in a stack-then-review queue complete, the operator displays a summary with all PR links and suggested merge order (base-first). The user can merge individually, or ask the operator to merge all in dependency order. When merging in order, the operator merges each PR sequentially, waiting for CI to pass before proceeding to the next. CI failure halts the merge sequence.

**`--merge-on-complete` opt-in**: Reverts to the previous merge-as-you-go behavior — merge each PR on completion, rebase next change onto `origin/main`. Confirmation text changes to "Confirm upfront (merges PRs on completion)." Natural language equivalent: "merge as you go".

**Failure matrix**:

| Failure | Action | Resume? |
|---------|--------|---------|
| Confidence below gate | Flag to user: run `/fab-fff` or skip | Wait for user input |
| Review fails (rework exhausted) | Flag, skip to next | Yes |
| Cherry-pick conflict (stack-then-review) | Escalate, do not spawn | No — queue halts, wait for user input |
| Rebase conflict (merge-on-complete) | Flag, skip to next | Yes |
| Agent pane dies | 1 respawn attempt, then flag and skip | Yes |
| Stage timeout (>30 min same stage) | Flag regardless of retry state | Yes |
| Total timeout (>2 hr per change) | Flag for review | Yes |

**Interruptibility**: `"stop after current"` (finish active, halt queue), `"skip <change>"`, `"pause"` (stop new commands, running agents continue), `"resume"`. Interrupts acknowledged immediately.

**Resumability**: If the operator session restarts, state is reconstructable from `fab pane-map`. Resume from first non-completed change.

#### Configuration

| Setting | Default | Override |
|---------|---------|----------|
| Monitoring interval | 5m | "check every {N}m" |
| Stuck threshold | 15m | "flag agents stuck for more than {N} minutes" |
| Autopilot tick interval | 2m | "autopilot check every {N}m" |

All settings are session-scoped — they reset when the operator session restarts.

#### Design Constraints

- **Pane-map only**: Uses `fab pane-map` as its sole observation primitive — no `fab runtime is-idle`
- **No change artifacts**: Never reads intakes, specs, or tasks — context window reserved for coordination state
- **No persistent audit trail for v1**: Per-answer logging is inline only — no file-backed log
- **Hardcoded patterns**: Question indicator patterns embedded in skill file, not configurable via config.yaml

#### Launcher

The operator is launched via `fab operator` — a `fab-go` subcommand (source: `src/go/fab/cmd/fab/operator.go`). It creates a singleton tmux window named "operator" running the configured `agent.spawn_command` (via `internal/spawn/`) with `'/fab-operator'`. If the window already exists, switches to it. Requires an active tmux session. Previous shell launcher scripts (`fab-operator4.sh`, `fab-operator5.sh`, `fab-operator.sh`) have been removed.

`fab operator` is a parent command with two subcommands:

- **`fab operator tick-start`** — Called at the start of each operator tick (step 1 of tick behavior). Resolves repo root via `gitRepoRoot()`, reads `.fab-operator.yaml` into `map[string]interface{}` using `gopkg.in/yaml.v3` (absent file treated as empty), increments `tick_count` by 1, writes `last_tick_at` as an RFC3339 UTC timestamp (`time.RFC3339`), writes the updated map back preserving all other fields (monitored set, autopilot queue, branch_map, watches). Outputs `tick: N\nnow: HH:MM` to stdout using local time. Write failure → stderr error + exit 1. No flags.

- **`fab operator time`** — Pure clock query with no file I/O or side effects. Always outputs `now: HH:MM` (local 24-hour time). With `--interval <duration>` (Go duration string, e.g. `3m`), also outputs `next: HH:MM` = now + interval. Invalid duration string → stderr error + exit 1.

**Usage in tick lifecycle**: The agent invokes `fab operator tick-start` at step 1 of each tick and parses its stdout for the tick count (`tick: N`) and current time (`now: HH:MM`). Between ticks (idle message), the agent runs `fab operator time --interval {interval}` to obtain both `now:` and `next:` values for the idle message line `Waiting for next tick. Time: HH:MM · next tick: HH:MM`. Separation of concerns: `tick-start` has side effects (writes YAML state), `time` is a pure query (no writes).

## Design Decisions

### Auto-Answer Model with Strategic Escalation (rule 4 classification)
**Decision**: Detected questions are auto-answered by default via a numbered decision list (items 1-6, evaluated in priority order). Rule 4 (numbered menus) further classifies the prompt as Routine or Strategic before answering. Routine prompts (tool/permission, binary-framed, synonymous-option menus) auto-answer `1`. Strategic prompts (multi-option choices representing materially different directions — scope, PR split, pipeline shape, commit organization, spec/approach) escalate to the user. Classification is LLM-judged over four signals in the terminal capture (option text length, semantic distinctness, surrounding agent context, reversibility); no hardcoded keyword list, no agent-side sentinel protocol. Classification uncertainty MUST escalate (asymmetric cost structure: silently committing the queue to an unchosen branch of work is more expensive than an extra user nudge).
**Why**: Worktree isolation and human PR merge are sufficient safety gates for routine operational prompts, but not for prompts that commit the queue to a direction the user never inspected (scope, PR split, spec/approach). A pure all-auto-answer model traded correctness for throughput in the exact scenarios where correctness matters most. Principle-based LLM classification adapts to novel prompt text without maintaining a keyword list or coupling the operator to every skill's surface area.
**Rejected**: Pure all-auto-answer (original model) — loses correctness on strategic prompts. Hardcoded keyword list — brittle, fails on novel prompts, high-maintenance. Agent-side `[STRATEGIC]` sentinel protocol — couples the operator to every skill and fails on Claude Code native + third-party prompts the operator cannot modify.
*Introduced by*: 260314-007n-redesign-operator-auto-nudge (original model); 260422-hin2-operator-strategic-menu-escalation (Strategic classification + escalate-on-uncertainty)

### 30-Minute Idle Auto-Default on Strategic Escalations
**Decision**: When rule 4 escalates a prompt as Strategic, the operator starts a per-prompt real-time idle timer from the escalation log time. If the prompt remains idle for 30 minutes (no terminal-state change in the pane), the operator auto-answers — sending the prompt's stated default if visible (e.g., `(default: 2)`, `Press enter for 2`, `[2]`), otherwise option `1`. The auto-default logs with a distinct format — `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"` — so after-action review tooling can distinguish confidently-auto-answered decisions from decisions taken because the user never returned. The idle clock resets on any terminal-state change in the pane (new agent output, user keystrokes, prompt redraw). The threshold is hardcoded 30 minutes — no `.fab-operator.yaml` field, no per-change override, no environment variable exposes it.
**Why**: Strategic escalations stall forward progress if the user is asleep, in meetings, or otherwise away from the terminal. Headless autopilot runs (overnight, multi-hour) become unreliable when every strategic escalation halts the pipeline. A 30-minute idle watchdog trades oversight for throughput in exactly the scenario where the alternative is zero throughput. The grep-distinct log format preserves auditability — `auto-defaulted` entries are recoverable separately from `auto-answered` entries.
**Rejected**: Configurable threshold (`.fab-operator.yaml`, per-change override, environment variable) — added surface area for marginal benefit; one threshold serves the single motivating "user is asleep / in a meeting" scenario well. Shorter threshold — risks auto-defaulting while the user is mid-reply. Longer threshold — defeats the feature. Uniform auto-default across all escalation types — conflates rule 4 Strategic (known-good default `1`) with rule 6 "cannot determine keystrokes" (auto-defaulting `1` would emit nonsense into the pane). Reusing the `auto-answered` log line — muddies audit trails by conflating confidently-answered and fell-back decisions.
*Introduced by*: 260422-hin2-operator-strategic-menu-escalation

### Re-Capture Before Send
**Decision**: The operator re-captures terminal output immediately before sending an auto-answer. If the output changed, the send is aborted.
**Why**: Eliminates the race condition between idle check and send. Single-tick grace period was rejected — it adds latency without fully solving the race.
**Rejected**: Single-tick grace period — delays answers by one full monitoring cycle and doesn't guarantee safety.
*Introduced by*: 260314-007n-redesign-operator-auto-nudge

### Claude Turn Boundary Guard
**Decision**: If a Claude Code `>` prompt cursor (`^\s*>\s*$`) appears in the last 2 lines of captured output, question detection is skipped.
**Why**: Claude's output often contains question-like phrasing ("Would you like me to...?") that triggers detection. The `>` cursor indicates the agent is at a normal human-turn boundary, not a blocking prompt.
**Rejected**: Excluding all question-mark lines from Claude — too broad, would miss genuine blocking prompts from Claude.
*Introduced by*: 260314-007n-redesign-operator-auto-nudge

### Operator Uses /fab-fff for Autopilot
**Decision**: Operator4 uses `/fab-fff` instead of `/fab-ff` for autopilot gate checks and pipeline invocations.
**Why**: `/fab-fff` is the more autonomous pipeline variant, fitting for operator-driven autopilot where human interaction is minimized.
**Rejected**: Keeping `/fab-ff` — its interactive fallback on review failure conflicts with the operator's autonomous mode.
*Introduced by*: 260314-007n-redesign-operator-auto-nudge

### Standalone Operator Over Inheritance Chain
**Decision**: Operator4 is a fully self-contained skill file. Previous iterations (operator1, operator2, operator3) were deleted — their behavior is inlined into operator4. The skill file loads `_cli-external.md` (operator-only) for external tool references (`wt`, `tmux`, `/loop`).
**Why**: Understanding the operator previously required reading 4 files in sequence (operator1 -> 2 -> 3 -> 4), mentally applying overrides. The standalone version is readable from a single file plus standard `_` files. Dead operator files in the skills directory risked ghost triggers via sync.
**Rejected**: Keeping operator1/2/3 as archived files — git history preserves them; dead files risk agents loading them. Extracting a shared base — adds indirection for a single-consumer pattern.
*Introduced by*: 260315-a2b2-standalone-operator4-rewrite

### Use Case Registry Over Single-Purpose Monitoring
**Decision**: The operator uses a use case registry instead of single-purpose monitoring — named, toggleable concerns checked on each `/loop` tick. The loop is the operator's heartbeat (runs while any use case is enabled), not tied to the monitored set.
**Why**: Real workflows have multiple concurrent monitoring concerns (change progress, Linear inbox, PR staleness) that all need periodic attention. A registry model lets users toggle concerns without operator restarts. Three built-in use cases (fixed set, not user-extensible).
**Rejected**: CLI-level branch resolution (`fab resolve --search-branches`) — fab operates on change folders, not git branches; branch awareness belongs in the operator skill.
*Introduced by*: 260317-yrgo-operator5-branch-fallback

### Branch Fallback in Operator, Not CLI
**Decision**: Branch fallback resolution lives in the operator skill (user-initiated only), not in the `fab` CLI. When `fab resolve` fails, the operator scans branch names as a fallback before reporting failure.
**Why**: `fab` is orthogonal to git — it operates on change folders (filesystem/YAML). Branch name scanning is a coordination concern (finding where a change lives), not a CLI concern. The operator already has the context to decide between read-only (`git show`) and action (worktree creation) responses.
**Rejected**: `fab resolve --search-branches`, `--branch` output mode, automatic fallback in CLI — all rejected because they couple the CLI to git branch semantics.
*Introduced by*: 260317-yrgo-operator5-branch-fallback

### Dependency-Aware Agent Spawning (operator7)
**Decision**: `/fab-operator` (v7) adds pre-spawn dependency resolution to the operator. When spawning an agent for a change with `depends_on` entries, the operator cherry-picks dependency content into the worktree before opening the agent tab. Uses `git cherry-pick --no-commit origin/main..<dep-branch> && git commit -m "operator: cherry-pick <dep> dependency"`. On conflict: abort, escalate, do not spawn.
**Why**: Without dependency awareness, agents working on dependent changes start from a baseline missing the dependency code, causing build failures, spec divergence, and manual intervention. This defeats the operator's "automate the routine" principle.
**Rejected**: `git merge --squash` — rejected for unattended sessions where merge machinery introduces risk. Transitive dependency resolution — rejected because leaf dependency branches already carry transitive content via the operator's own cherry-picking when those deps were spawned; `origin/main..<dep-branch>` gives the complete transitive closure.
*Introduced by*: 260324-prtv-operator7-dep-aware-spawning

### Operator7 Schema Additions
**Decision**: `.fab-operator.yaml` gains three new fields: `depends_on` (list of change IDs per monitored entry), `branch` (change's branch name per monitored entry), and `branch_map` (top-level map persisting change ID → branch name after changes leave the monitored set). Redundant deps are pruned via `git merge-base --is-ancestor` before cherry-picking. The `--base` autopilot flag implies `depends_on`.
**Why**: Branch names must persist after dependencies complete (merged/archived) so downstream changes can still cherry-pick from them. Redundant dep pruning prevents duplicate cherry-picks in chains (B's branch already contains A's content).
*Introduced by*: 260324-prtv-operator7-dep-aware-spawning

### Operator7 Direct fab-new for Raw Text Spawns
**Decision**: When spawning agents from raw text descriptions, the operator passes the description directly to `/fab-new` instead of creating an intermediate backlog entry via `idea add`. The "From raw text" spawn path now follows the same structure as "From backlog ID": worktree → resolve deps → spawn with `/fab-new <description>` → enroll → completion.
**Why**: The `idea add` step created orphaned backlog entries in `fab/backlog.md` that served no further purpose — the intake's Origin section already captures the raw input for traceability. `/fab-new` natively accepts natural language descriptions, making the backlog indirection redundant overhead.
**Rejected**: Keeping `idea add` for backlog traceability — the intake artifact is the real record of a change's origin, not the backlog entry.
*Introduced by*: 260326-13ro-operator7-direct-fab-new-spawn

### Pipeline-First Routing Principle (operator7)
**Decision**: `/fab-operator` (v7) §1 Principles gains a "Pipeline-first routing" principle requiring the operator to route all new work through `/fab-new` then a pipeline command (`/fab-fff`, `/fab-ff`, `/fab-continue`). The operator MUST NOT dispatch raw inline implementation instructions to agent panes and MUST NOT use `/fab-continue` to skip intake for new work. Operational maintenance commands (merge PR, archive, delete worktree, rebase, `/git-branch`, `/fab-switch`) are exempt. A reinforcing blockquote in §6 "Working a Change" references the §1 principle.
**Why**: Without an explicit prohibition, an operator (especially after `/clear` or under time pressure) could shortcut by sending freeform implementation instructions directly to an agent pane — bypassing intake generation, confidence scoring, and the full pipeline. This violates the fab workflow's core value: specification-driven development with traceability (Constitution §II).
*Introduced by*: 260326-u3un-operator-enforce-pipeline-routing

### Stack-Then-Review Autopilot Default (operator7)
**Decision**: The autopilot queue defaults to **stack-then-review** mode. All queued changes after the first implicitly get `depends_on: [<prev-change-id>]` (equivalent to implicit `--base` chaining). PRs are created but not merged until the user reviews and explicitly requests merging. The previous merge-as-you-go behavior is preserved via `--merge-on-complete` opt-in flag. Queue completion produces a summary with all PR links and suggested merge order (base-first). Ordered merge waits for CI on each PR before proceeding to next.
**Why**: The previous merge-as-you-go default caused two problems: (1) rebase conflicts when rebasing dependent changes onto freshly-merged `origin/main` re-linearized commits that cherry-pick resolution had already handled, and (2) no opportunity for holistic review of the full change set before any code merged to `main`. Stack-then-review gives the user full review control over the entire queue.
**Rejected**: Keeping merge-as-you-go as default — too many rebase conflicts and no review control. Available as opt-in for users who want it.
*Introduced by*: 260327-gwg9-operator-base-chaining-default

### Standardized Tmux Tab Naming (operator7)
**Decision**: All agent tab names in `/fab-operator` use `»<wt>` format (right guillemet + worktree name, no space). Replaces the previous `fab-<id>` naming which was unreliable for new changes where the change ID doesn't exist at spawn time. The worktree name is always available at spawn time and unique across panes, making it a consistent identifier for all three spawn paths (existing change, raw text, backlog). Originally used `⚡` (zap emoji) as prefix, but switched to `»` (U+00BB) because the emoji's double-width rendering caused tmux tab bar misalignment and console output formatting issues.
**Why**: The `fab-<id>` format had two issues: (1) for new changes, the ID doesn't exist until `fab-new` runs inside the spawned agent, and (2) the raw-text path already used `fab-<wt>` as a workaround, creating inconsistency. The `»` prefix makes agent tabs visually distinct from other tmux windows while being single-width for consistent terminal rendering.
**Rejected**: Keeping `fab-<id>` with worktree fallback — adds conditional logic without benefit since worktree name is always available. `⚡` emoji — double-width rendering breaks tmux tab alignment.
*Introduced by*: 260328-iqt8-standardize-tmux-tab-naming

### `»` Prefix Extends to Enrolled Windows
**Decision**: The `»` convention applies to **every** monitored window, not just operator-spawned ones. On enrollment the operator invokes `fab pane window-name ensure-prefix <pane> »` (U+00BB). The primitive's literal-prefix idempotent check makes the step a no-op when the name already starts with `»`, covering operator-spawned, `/clear`-restored, and re-enrolled entries. The rename runs after the monitored entry is durably written to `.fab-operator.yaml`; a non-zero primitive exit logs one line and leaves the enrollment intact.
**Why**: With the original decision, the `»` prefix was only half-enforced — windows enrolled via direct command dispatch, user request ("watch this pane"), autopilot spawns, and watch spawns kept their original names, so the monitored set split visually into two indistinguishable populations (prefixed vs unprefixed). Extending the convention to every enrollment path makes the tab bar an accurate at-a-glance map of what the operator is currently tracking. Rename-after-YAML-write ordering guarantees that a partial failure leaves a tracked entry without the cosmetic prefix, never a prefix without a tracked entry.
**Rejected**: (1) Inventing a second signal (status-bar marker, pane title) — violates parity with the existing convention, adds surface area. (2) A generic "already-marked" regex (`^[»⚡…]`) — would silently absorb legacy or user-chosen markers; the literal `»` check keeps naming sovereignty with the user.
*Introduced by*: 260422-jyyg-operator-prefix-enrolled-windows. *Amended by*: 260423-rxu3-window-prefix-primitives (extracted the inline tmux shell into `fab pane window-name ensure-prefix`).

### Done-Marker Swap on Removal
**Decision**: On every removal path (change reaches its `stop_stage` or a terminal stage when `stop_stage` is null, pane dies, user explicitly stops monitoring), the operator invokes `fab pane window-name replace-prefix <pane> » ›` to swap the active-monitoring `»` prefix for the done-marker `›` (U+203A, single right guillemet). The primitive's literal-prefix guard silently no-ops on user-renamed windows (if the user renamed the window mid-monitoring so it no longer starts with `»`, the swap is skipped). Exit 2 (pane missing) is treated as successful removal; other non-zero exits log `"{change}: window rename skipped ({error})."` and the operator continues.
**Why**: Leaving `»` on removed windows made the tab bar lie about what the operator was currently tracking. The entire purpose of the prefix is at-a-glance coordination — an honest signal means "`»` = tracking now, `›` = operator touched but done, untouched = untouched." The done-marker `›` was chosen over `✓` because `✓` already appears in the operator status frame as the stage-done signal (`● apply → review ✓`); reusing it on window names would create a semantic collision. `›` preserves the guillemet visual family (`»` → `›`, double → single), is single-width BMP (consistent with the 260328 / 260416 decisions), and reads as "was-active, now trail-preserved." The literal-prefix guard inside `replace-prefix` replaces the prior "no restore on removal" rule: no state-storage (`original_name`) is needed, because the guard protects user-renamed windows inherently — if the current name no longer starts with `»`, the swap doesn't match and doesn't fire.
**Rejected**: (1) Leave `»` on removal (260422-jyyg's original rule) — self-defeating signal staleness; tab bar lies about what is tracked. (2) Restore the original name — requires storing `original_name` with a "user-renamed-mid-monitoring" ambiguity that the guard-based approach side-steps. (3) `✓` as the done-marker — collides with the stage-done signal in the operator status frame. (4) Per-project config option for the done-marker character — speculative, no current demand; the character is a skill constant in `src/kit/skills/fab-operator.md`.
*Introduced by*: 260423-rxu3-window-prefix-primitives

### Multi-Repo / Multi-Session Isolation = Tmux Server (one operator per server)
**Decision**: The operator's isolation unit is the **tmux server** — exactly one operator per server, spanning every session and repo on it, owning one **server-keyed** state file at `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (fallback `~/.local/state/...`), keyed by the tmux socket path (the binary derives it via `StatePath()` — see [kit-architecture.md](../distribution/kit-architecture.md)). A second operator means a second tmux server (`tmux -L <label>`). There is **no `--name` dimension** — the server boundary is the only isolation knob, matching the server-wide singleton already enforced by the `operator` window. Old repo-rooted `.fab-operator.yaml` files are **not migrated** — abandoned in place (the monitored set is re-derivable from live `»`-prefixed panes). Every monitored entry, `branch_map` value (`{ branch, repo }`), and watch (`target_repo`) is repo-qualified under the `(session, repo, pane)` addressing tuple, with pane ID as the server-global primary key.
**Why**: A single operator coordinating multiple repos/sessions is the central value of the operator (one pane of glass). A repo-rooted state file is single-repo-only; a fixed global path would force a machine-wide singleton. Keying the file by the tmux socket scopes one owner across all repos on a server while still allowing a second server to host an independent operator. No migration is acceptable because operators don't survive a binary upgrade anyway.
**Rejected**: Per-repo operators (loses the single pane of glass). A `--name` operator dimension (redundant with the server boundary). Repo-rooted `.fab-operator.yaml` (single-repo only). A fixed global state path (machine-wide singleton). Migrating old state files (unnecessary — monitored set is re-derivable).
*Introduced by*: 260607-oy0k-operator-multi-repo-skill

### Cross-Repo Dependencies = Ordering-Only (no code merge)
**Decision**: `depends_on` resolution is two-tier, split by repo. A **same-repo** dependency cherry-picks as today (`git cherry-pick --no-commit origin/main..<dep-branch>`). A **cross-repo** dependency is an **ordering-only barrier**: the operator waits until the dependency reaches its `stop_stage`, then spawns — **no code is merged**. The skill states the REQUIRED caveat that a cross-repo dependency gives the dependent agent no code (pure logical sequencing), correct only for logical deps ("don't start the frontend until the API merges"), never code-level ones. Ancestor-pruning (`git merge-base --is-ancestor`) is scoped to the same-repo subset only.
**Why**: Cross-repo branches share no common `origin/main` base, so there is no sound cross-repo cherry-pick; logical sequencing is the only sound cross-repo semantic. Forbidding cross-repo deps would be too restrictive for real multi-repo workflows.
**Rejected**: Forbid cross-repo deps (too restrictive). Full cross-repo code merge (unsound — no shared base). Cross-repo ancestor-pruning (meaningless across repos with no shared history).
*Introduced by*: 260607-oy0k-operator-multi-repo-skill

### Repo-Spanning Autopilot CI-Failure = Halt-Dependents-Only
**Decision**: An autopilot queue may span repos with mixed dependency semantics (within-repo cherry-pick chaining degrades to cross-repo ordering-only barriers); ordered merge tracks per-repo PR sequences. On a CI failure during ordered merge, the operator halts the failing repo's merge sub-sequence AND any repo whose queued items carry a cross-repo `depends_on` into the failed chain — **transitively** over the cross-repo `depends_on` graph. Truly independent repos' sub-sequences **continue merging**. The completion summary reports halted vs. completed sub-sequences and escalates to the user.
**Why**: Maximizes independent-repo throughput while still respecting cross-repo ordering barriers — the failure's blast radius is isolated to its dependency cone rather than throttling the whole queue.
**Rejected**: Halt-all (conservative, throttles independent repos — the earlier lean, superseded during clarify). Halt-only-failing-repo (ignores cross-repo ordering barriers, would merge a dependent ahead of its failed barrier).
*Introduced by*: 260607-oy0k-operator-multi-repo-skill


## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260608-memory-domain-restructure | 2026-06-08 | Created `runtime/operator.md` by extracting the operator coordination content from `pipeline/execution-skills.md` (the `/fab-operator4` historical section, Monitoring System, tick lifecycle via `fab operator tick-start`/`time`, and the operator design-decision lineage from "Auto-Answer Model with Strategic Escalation" through "Done-Marker Swap on Removal"). Part of the memory-domain restructure that replaced the single `fab-workflow/` pseudo-domain with `pipeline/`, `memory-docs/`, `distribution/`, `runtime/`, and `_shared/`. The per-change history of these design decisions is preserved in `execution-skills.md`'s changelog (where the content lived when those changes shipped); this row records the extraction only. No operator behavior changed. |
| 260607-oy0k-operator-multi-repo-skill | 2026-06-08 | Re-framed `/fab-operator` for **multi-repo / multi-session coordination on one tmux server**. Addressing moved to the `(session, repo, pane)` tuple (pane ID stays primary key; `repo` + `session` are added dimensions). Resolved the h3jk deferral note and rewrote the repo-rooted "Monitoring System" prose into a server-keyed model: one operator per server, one server-keyed state file `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml` (no migration of old repo-rooted files), `monitored` entries gain `repo`+`session`, `branch_map` value becomes `{ branch, repo }`, watches gain `target_repo`. Tick snapshots via `fab pane map --all-sessions --json` grouped by repo→session with repo-section-header status frame. Added repo-targeted spawning (`wt create` in the target repo dir, `fab spawn-command --repo`), two-tier dependency resolution (same-repo cherry-pick / cross-repo ordering-only with the no-code caveat; ancestor-pruning scoped to same-repo subset), and repo-spanning autopilot with per-repo PR sequences and halt-dependents-only CI semantics (transitive over the cross-repo `depends_on` graph). Added three Design Decisions (tmux-server isolation / one-operator-per-server, cross-repo deps ordering-only, halt-dependents-only CI). Skill+specs change only; consumes change 1's (`260607-h3jk`) Go primitives, whose mechanism docs live in [kit-architecture.md](../distribution/kit-architecture.md) and [pane-commands.md](pane-commands.md). |
