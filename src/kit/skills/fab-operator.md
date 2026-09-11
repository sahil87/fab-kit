---
name: fab-operator
description: "Use when coordinating multiple fab agents across tmux panes — multi-agent monitoring over one generic tracked-item list (agent panes, GitHub PRs, Linear/Slack queries, shell probes), auto-answering prompts, routing commands, and dependency-aware agent spawning."
helpers: [_cli-fab-operator, _cli-fab-pane, _cli-agents, _cli-external]
---

# /fab-operator

> Read the `_preamble` skill first (deployed to `.agents/skills/` via `fab sync`). Then follow its instructions before proceeding.

## Contents

- 1. Principles
- 2. Startup
- 3. Safety
- 4. The Clock
  - Mute and Lease
  - Tracked Items
  - Tick Behavior
  - Status Frame Format
- 5. Auto-Nudge
- 6. Coordination Patterns
- 7. Linear and Slack Items
- 8. Configuration
- 9. Key Properties

Multi-agent orchestration layer. Runs in a dedicated tmux pane, observes agents across all sessions on its tmux server (per tick via `fab operator tick-start --diff --quiet`, on demand via `fab pane map --all-sessions`), routes commands and answers via `rk mux send` (plain for command routing, `--answer` for prompt answers, `--key` for key-name input), and takes its cadence from run-kit's operator-tick cron entry, whose `operator tick` deliveries are the heartbeat (§4). Spans multiple repos and sessions on one server.

Start via `fab operator` (singleton tmux tab named `operator`). When a capable run-kit is on PATH, the bare command delegates the entire launch to `rk operator`; the binary's built-in launcher is its fallback — its behavior (window cwd, session command, `operator`-role model resolution and built-in defaults) is documented in `_cli-fab-operator.md` § fab operator, the canonical source for the §9 Key Properties rows below.

---

## 1. Principles

| Principle | Rule |
|-----------|------|
| Coordinate, don't execute | Every task, bug report, or idea the user hands the operator **is a work request** and enters through §6 Working a Change (a fresh report takes the raw-text form — a `/fab-new` spawn in a fresh worktree; a report naming a live tracked item is a send to that item's agent). Reading code to reproduce or diagnose, or editing files, in the operator pane **is** executing and is prohibited. The operator reads whatever plan, roadmap, intake, or task document the tracked work needs, and keeps handing the next unit of work to an agent until the plan is done. Direct actions are exactly this maintenance allowlist: merge PR, archive, worktree deletion, rebase/cherry-pick for dependency resolution (§6), `fab operator track` verbs (§4), and pane sends/answers/nudges (§5). The only thing the operator asks about a work request is **which repo** (plus the target-session tie-break in §6 Spawning an Agent step 2) — never whether to spawn; that ask-set is scoped to the work request and does not relax the §3 pre-send and destructive-action confirmations or the §6 queue confirmation, which still apply. |
| Multi-repo aware | Address every agent as `(session, repo, pane)` on one tmux server, with pane ID primary and every tracked item and `branch_map` entry repo-qualified; state is one server-keyed file (§4, §8, §9). `session` is a **display/context dimension, never a join key** — a tracked agent's session can change mid-lifetime (`move-window` relocation), so correlation rides the pane ID (§ fab pane map's identity-key contract in `_cli-fab-pane.md`). |
| Automate the routine | Auto-answer, nudge, rebase, and spawn for routine operations; PR review is the safety net. Every operator-spawned agent is tracked automatically (§4–§7). |
| Do not enforce lifecycle | Agents self-govern pipeline transitions; report unexpected stages factually (§4). |
| Re-derive state | Before every action, query `fab pane map --all-sessions`; never trust conversational pane/repo/session/stage values (§4). |
| Survive compaction | The agent cannot `/clear` itself. When a tick fires and §4 Tick Behavior is no longer in context (harness auto-compaction, or a session resumed from a summary), run `/fab-operator` **once** to reload, re-run §2 Init, then resume lean `operator tick` firings; tracked items and `branch_map` survive in the server-keyed state file (§4 Post-Compaction Reload). |

---

## 2. Startup

### Context Loading

Load only `fab/project/config.yaml`, `fab/project/constitution.md`, and `fab/project/context.md` (optional — skip gracefully if missing). The operator is a listed exception to the `_preamble.md` §1 always-load layer: code-quality, code-review, and the doc indexes serve artifact generation and review, which the operator never does — it neither authors nor reviews artifacts (§1 Coordinate, don't execute) — and a long-lived session re-pays any loaded file after every reload (compaction, `/clear`, or restart — §4 Post-Compaction Reload). Do not run `fab preflight`.

Helpers declared in frontmatter: `_cli-fab-operator` (operator/agent CLI reference — the `track` verb contracts and the tick document), `_cli-fab-pane` (pane/dispatch CLI reference), `_cli-agents` (the generic agent-CLI interaction procedures — spawn composition, pre-send validation, delivery probe, peek, await — plus the per-provider grammar/discovery dictionary), and `_cli-external` (wt, idea, tmux reference). Naming conventions are inlined in `_preamble.md` § Naming Conventions — already loaded.

The split between `_cli-agents` and this file is **agent primitives vs. operator orchestration**: `_cli-agents` owns *how* to talk to an agent CLI (the mechanics any session could reuse); this file owns *when and whether* to (confirmation tiers, retry budgets, repo targeting, tracking, dependency resolution, queues).

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

### rk Gate

The operator's clock, role mark, send gate, spawn readiness, and notifications are run-kit. Probe once here — no later call site is individually gated:

```bash
command -v rk >/dev/null 2>&1 && rk cron list --json >/dev/null 2>&1
```

If either half fails (rk absent, or an installed rk predating `rk cron` — the capability probe), STOP:

```
Error: the operator requires run-kit — brew install sahil87/tap/run-kit
```

This is the operator's deliberate exception to `_preamble.md` § Run-Kit (rk) Reference's fail-silent rule: that rule protects skills for which rk is an optional enhancement; for the operator rk is the substrate, so absence is a startup error, not a degradation.

### Role Mark

Mark this tmux window as the operator for run-kit's dashboard (the `@rk_win_role` window option — rk owns the option contract, the pinned rendering, and the one-operator-per-server radio semantics; fab is only the producer):

```bash
rk role operator >/dev/null 2>&1 || true
```

Fail-silent by contract (`_preamble.md` § Run-Kit (rk) Reference): a role-mark failure is never startup-blocking. Idempotent: a restarted operator re-marks the same window harmlessly. There is no unmark step — the operator has no clean exit hook, and staleness and radio conflicts are rk's to resolve.

### wt Gate

`wt` gates the **agent spawn path only** (§6 step 3) — tracking a GitHub PR, a Linear query, or a shell probe needs no worktree, so startup does not probe it. When a pane item is about to spawn, probe `command -v wt >/dev/null 2>&1` once per operator session; on failure, STOP the spawn:

```
Error: wt is required for operator spawning — install it via: brew install sahil87/tap/wt
```

`wt` ships as a standalone formula (not a `fab-kit` Homebrew dependency), so it may legitimately be absent — an operator that never spawns an agent never needs it.

### Init

1. Run `fab operator state` to read (or create, on first run) the server-keyed operator state file — the binary derives the path and persists the empty skeleton when missing; the operator never computes the path or hand-creates the file (`_cli-fab-operator.md` § fab operator state). Old repo-rooted `.fab-operator.yaml` files are not read or migrated
2. Read the tracked items via `fab operator track list --json` and note the `branch_map` from the state dump (this is what makes §4 Post-Compaction Reload lossless)
3. Run `fab pane map --all-sessions` and display the output (all sessions on this server, not just the operator's own)
4. Verify the clock — the rk Gate already proved `rk cron list --json` works: read it and select the row whose `target` is `role:operator` — the operator-tick entry (the one `rk operator` seeds — §4 The Clock) — and read its `schedule_summary`, `deliver`, `muted`, and `muted_until` fields (`muted` is the effective state — an indefinite mute or a live lease; `muted_until`, unix seconds, is present only while a lease is live). When the entry is `muted` while `fab operator track list` (step 2) shows tracked work — any item whose state is not `done` — issue `rk cron mute <id> --off` to unmute it (§4 Mute and Lease). A missing operator-tick entry after the gate passed STOPs: `Error: no operator-tick cron entry on this server — run rk operator to seed it`
5. Output the ready line **with the clock status** — exactly one template, rendered from step 4's fields:

   ```
   Operator ready. Clock: rk cron "operator tick" · {schedule_summary} · {deliver}[ · muted[ until HH:MM]]
   ```

   (`{schedule_summary}` and `{deliver}` are the JSON values verbatim; append ` · muted` when `muted` is true; append ` until HH:MM` — local time — when `muted_until` is present. Renders today as `Operator ready. Clock: rk cron "operator tick" · backoff 1m→30m · immediate`, or `… · immediate · muted until 14:30`. A missing entry STOPs per step 4 — there is no `Clock: none` form.)

---

## 3. Safety

### Confirmation Tiers

| Tier | Examples | Behavior |
|------|----------|----------|
| Read-only | Status check, pane map | No confirmation |
| Recoverable | Send `/fab-continue`, rebase | Announce before sending |
| Destructive | Merge PR, archive, delete worktree | Confirm before executing |

### Pre-Send Validation

**Skill routing:** every explicit skill send in this file uses `_cli-agents.md` § Skill Prompts. Slash-form names in the routing vocabulary identify skills; they are not literal wire payloads. Render for the receiver before sending.

Before sending keys to any pane, run the two-step gate in **`_cli-agents.md` § Pre-Send Validation** (pane exists via a refreshed pane map → agent state fits the send intent per the three-state `@rk_pane_agent_state` read — the same mode-aware gate `rk mux send` enforces), then apply the operator's own policy on its outcome plus the two operator-specific checks:

1. **Pane gone** (gate step 1 fails) — report "Pane for {change} is gone." Do not send.
2. **Agent not `idle`** (gate step 2) — the operator does **not** silently proceed. If `active` or `waiting`: "{change} is {state}. Sending may corrupt its work / cut across a pending human answer. Send anyway?" — send only on explicit confirmation, and keep the confirmed send gated: a `waiting` target rides `rk mux send --answer`, an `active` target requires `rk mux send --force` (the deliberate skip-everything override). If unknown (`—`, no `@rk_pane_agent_state` on the pane): the agent isn't instrumented; confirm before sending (plain send then warns-and-sends — no override needed). Only `idle` sends unattended. *(This routed-command confirm policy is unchanged by `--answer` — the flag swaps the mechanism after confirmation, not the ask; the auto-answer flow in §5 is the unattended `--answer` consumer.)*
3. **Check change is active** — if the target change isn't the active change in that tab, send the rendered `fab-switch` invocation with `<change>` first.
4. **Check branch alignment** — if the tab's git branch doesn't match the change folder name, send the rendered `git-branch` invocation to align it.

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
| Pane death | 0 | Report gone. Respawn only for a queue-driven change (1 attempt) |
| Agent exited (pane survives as a shell) | 0 | Report gone (pane kept, cwd intact). Respawn only for a queue-driven change (1 attempt): kill the leftover shell pane first (`rk mux kill` — an uninstrumented/idle pane passes its gate), then spawn per §6 |
| Send to busy agent | 0 | Warn, require explicit confirmation |
| Cherry-pick conflict | 0 | Abort, log, escalate. Do not spawn. |

---

## 4. The Clock

The operator's cadence is **not owned by this skill** — it is run-kit substrate the skill documents and verifies, exactly like `@rk_pane_agent_state`. The clock is the **operator-tick cron entry** seeded idempotently by `rk operator` (one per tmux server); run-kit's operator-cron spec is the entry's design authority. Ticks arrive as the bare text `operator tick` delivered into the operator pane, for **every** provider (Claude, codex, gemini, …) — no provider-specific in-session clock is the primary cadence.

The seeded entry's shape (reference summary — schema and semantics are owned by the cron spec, not restated here):

```yaml
schedule: { kind: backoff, min: 60s, max: 30m }
wake_on: { event: agent-state-change, scope: server, debounce: 10s }
target: { kind: role, role: operator }
payload: "operator tick"
deliver: immediate
if_absent: respawn
respawn: ["rk", "operator", "-L", "{server}"]   # caller-supplied argv; {server} substituted by rk at fire time
pinned: true
```

**Ownership.** `rk operator` seeds the entry idempotently at launch. Every `fab operator track` verb (and `tick-start --diff`) then manages it as a side effect: the **tracked predicate** — any item whose state is not `done` — flips the entry between muted and live via `rk cron mute <id>` / `--off`, and the **schedule reconcile** derives the cadence from the tracked set (backoff `1m`→`30m` while only pane/none items are tracked; `--idle-every`/`--every <min check_every>` once any shell/agent item exists, deliver `skip-if-busy`) and applies it with exactly one `rk cron edit` when — and only when — the live entry differs. The derive table, epoch detection, and fail-silent posture are owned by `_cli-fab-operator.md` § fab operator (the shared **Clock side effect** paragraph); `rk cron add`/`rm` stay the user's.

### Mute and Lease

- The skill **MAY** use `rk cron mute <id> --for <dur>` for a user-requested bounded quiet window (e.g. "hold the ticks for 30 minutes" → `--for 30m`); the lease auto-expires — no unmute call is needed.
- The skill **MUST** never leave an indefinite mute behind while work is tracked — the `track` verbs enforce the tracked-set half (their `--off` clears any standing mute or lease); this rule covers a manual `rk cron mute <id>` a user or the operator typed.
- `rk cron mute <id> --off` clears both a mute and a lease.
- The lease is a bounded snooze, not a heartbeat — nothing renews it since the in-session loop is retired.
- A bounded **cadence override** (a different cadence for a while, not silence) is `fab operator track clock --every 10m --for 2h` (or `--idle-every <dur>`); `--for` is required — unbounded overrides are rejected — and the reconcile applies the override until it expires, then reverts to the derived schedule. `fab operator track clock --off` clears it early. Contract: `_cli-fab-operator.md` § fab operator track.

### Tick Payload

A tick's text **MUST be the bare text `operator tick`** — never `/fab-operator` or any other slash command. The rule binds the cron entry's `payload` and any manually typed tick. Reason: a slash command macro-expands its full source into the turn on **every** firing — this file alone is thousands of tokens, so a `/fab-operator` payload re-pays the whole skill each tick and exhausts the context window in a matter of ticks. The tick procedure (§4 Tick Behavior) is already in context; the payload only needs to *name* it.

Recovery when this procedure is no longer in context: § Post-Compaction Reload.

### Post-Compaction Reload

**Trigger** — a tick (`operator tick`) arrives and §4 Tick Behavior is not in context: the agent cannot see the numbered Snapshot → Act on deltas → Answer waiting agents → Ack list. Typical causes: harness auto-compaction of a long session, a fresh session resumed from a conversation summary, a user `/clear`. Cron-delivered ticks keep arriving regardless of session health, so this trigger is guaranteed to fire eventually.

**Procedure**:

1. Run `/fab-operator` exactly **once** — this reloads the skill body and its helpers and re-runs §2 Startup including Init (state file re-read via `fab operator state` + `fab operator track list --json`, `fab pane map --all-sessions`, clock verification per §2 Init step 4).
2. Treat the tick that triggered the reload as consumed — the next tick's `fab operator tick-start --diff` re-emits every level-triggered delta (§4 Tick Behavior step 1), so nothing durable is lost.
3. Continue with bare `operator tick` firings. **Never** put `/fab-operator` into a tick payload as a way to "stay reloaded" — that is the failure mode this procedure replaces.

**Durable state** — tracked items and `branch_map` live in the server-keyed operator state file and survive compaction, `/clear`, crash, and restart; only §8 session-scoped settings and in-conversation context are lost.

**`/clear` is a user action.** A user may `/clear` a bloated operator; it lands on this same procedure (the next tick, or the user's next message, finds no procedure in context). The skill never instructs the agent to `/clear` — the agent-side mechanism is *compaction → one-shot `/fab-operator` reload*.

### Tracked Items

One ordered `tracked` list is the operator's entire durable work set: every thing the operator watches, probes, spawns, or waits on is an **item** with a `kind`, a `probe`, a `done_when`, and a `then`. Persistent state, read on startup (§2 Init) and diffed every tick by `fab operator tick-start --diff` (§4 Tick Behavior). The term **operator state file** used throughout this skill refers to the server-keyed file from §2 Init step 1 — one per tmux server spanning every repo it coordinates.

**The operator never hand-writes this file — every mutation is a `track` verb** (`fab operator track add|update|observe|rm|list|clock`): the operator states intent through flags; the binary owns the schema, the timestamps, the caps, and the atomic write (same doctrine as `fab score`: agents never compute what the binary can own). The verb contracts, the `done_when` grammar, and the probe-runner mechanics live in `_cli-fab-operator.md` § fab operator track; the schema block below is *reference* documentation of what the binary maintains. A legacy state file from fab ≤2.24 (the old monitored / watches / autopilot / notes sections) is converted by the binary on first touch — the user-facing walkthrough is the 2.24.9 → 2.25.0 migration file under `$(fab kit-path)/migrations/`.

```yaml
tick_count: 47
last_tick_at: "2026-09-11T16:33:00Z"
last_full_at: "2026-09-11T16:30:00Z"   # written on every full tick document; the 10m periodic-refresh clock
tracked:
  - id: r3m7                    # pane items: the change id when known, else the worktree name; other kinds: a slug unique in the list
    kind: pane                  # pane | github-pr | …
    probe: { mode: pane }       # pane | shell | agent | none — how the item is checked
    check_every: null           # cadence for shell/agent probes; null for pane/none items
    done_when: null             # null on pane = the built-in predicate when scope.change is set; never done alone when null
    then: null                  # prose the operator runs when done/changed fires; null = report only
    depends_on: []              # item ids; unsatisfied deps hold the item
    scope:                      # kind-specific metadata, opaque to the probe runner
      pane: "%3"                # null until spawned (a queued item has no pane yet)
      pane_pid: 48213           # shell pid fingerprint recorded when pane was set; null ⇒ pane-id-only join
      change: r3m7              # observed (baseline for change diffs); null until a change appears
      repo: /home/user/code/foo
      session: work
      branch: 260324-r3m7-add-retry-logic
      stage: apply              # baseline for stage diffs
      stop_stage: null
      spawned_by: null          # id of the linear/slack item that spawned it
      merge_mode: null          # set when the item was queued via a chain
    last: {}                    # the last probe's declared fields (pane items: empty — the snapshot is the state)
    checked_at: "2026-09-11T16:33:00Z"
    unchanged: 0                # consecutive probes with no field delta — the stall counter, binary-owned
    failures: 0                 # consecutive probe errors; 3 → paused
    paused: false
    done_at: null               # pane items: set by the binary when the built-in completion fires — durable across pane death
  - id: pr-913
    kind: github-pr
    probe: { mode: shell, argv: [gh, pr, view, "913", --json, "state,mergedAt,mergeable"], fields: [state, mergedAt, mergeable] }
    check_every: 2m
    done_when: 'state == "MERGED"'
    then: "spawn n34 in ~/code/hexokit via /fab-fff"
    scope: { repo: /home/user/code/hexokit, pr: 913 }
    last: { state: OPEN, mergedAt: null, mergeable: MERGEABLE }
  - id: linear-bugs
    kind: linear
    probe: { mode: agent, instruction: "mcp__claude_ai_Linear__list_issues project=DEV …" }   # the operator runs it; the binary never does
    check_every: 5m
    then: "for each new id: spawn /fab-new <id> in ~/code/foo, stop at intake"
    scope: { repo: /home/user/code/foo, stop_stage: intake }
    seen: [DEV-988, DEV-992]    # bounded dedupe list (200 cap, oldest pruned), fed by track observe --seen
  - id: n1
    kind: note
    probe: { mode: none }
    text: "Phase 2 of 4 — hexokit n34 spawns after run-kit #913 merges"   # 500-char cap
branch_map:                     # change id → { branch, repo }; written by track add (branch+repo) or at the tick a change is first observed
  ab12: { branch: 260324-ab12-fix-auth, repo: /home/user/code/foo }
```

**Kinds** (the binary fills each kind's defaults at `track add`; flags override):

| Kind | Probe | Done when | Notes |
|---|---|---|---|
| `pane` | `pane` — the binary snapshots the pane every tick and joins on `scope.pane` plus the `pane_pid` fingerprint | built-in when `scope.change` is set (`review-pr` done/skipped, or at/past `scope.stop_stage`); never on its own when null | The operator's core kind (§6): **any agent session in a pane**; change and stage are observed, not required. With branch+repo set, `track add` also writes `branch_map` |
| `github-pr` | `shell`: `gh pr view <n> --json state,mergedAt,mergeable` (default argv/fields filled by the binary) | `state == "MERGED"` | `then: "arm next: …"` prose is how a merge sequence chains (§6 Auto-Merge Choreography) |
| `linear` / `slack` | `agent` — the operator runs `probe.instruction` when the tick lists the item under `needs_check:` | standing — removed by the user | Carries the `seen` dedupe list; §7 |
| `shell` | `shell` — argv + fields required at add | required at add | Generic: deploys, CI runs, any JSON-emitting command |
| `task` | `none` | deps done | A held action with `depends_on` and `then` but nothing to probe (e.g. "when both PRs merge, tag v3.20") |
| `note` | `none` | — | Free prose (500-char cap), rendered in the frame and the `state` OPEN NOTES header |

**Lifecycle** (binary-derived, emitted in the tick document and rendered in the frame):

| State | Condition |
|---|---|
| `held` | any `depends_on` item is not done |
| `pending` | pane item with `scope.pane` null and deps satisfied — spawned this tick (§6) |
| `live` | pane item whose pane is present **and** whose fingerprint matches (or is null) |
| `watching` | shell/agent item probed on cadence, `done_when` false |
| `stale` | agent item with `now - checked_at > 2 × check_every` |
| `paused` | `failures` reached 3 (auto) or `track update --pause` |
| `done` | `done_when` true, or the pane built-in fired (persisted as `done_at`, so it survives the pane vanishing) — level-triggered until `track rm`; removing a done item drops it from every dependent's `depends_on` |

The top-level **`branch_map`** persists change ID → `{ branch, repo }` beyond an item's removal — downstream dependency resolution looks up dependency branches there (§6). Entries are written by `track add` (when branch+repo are set) or at the tick a change is first observed, retained by `track rm`, and cleared only by `fab operator branch-map rm <change-id>` / `--all`.

Anything still true for a different operator next month is not operator state — route it via an `idea` backlog entry (`_cli-external.md` § idea), never a note item.

### Tick Behavior

On each tick:

1. **Snapshot** — run `fab operator tick-start --diff --quiet`: one command increments `tick_count`, snapshots the panes, runs every due shell probe, evaluates `done_when`/`depends_on`/staleness, and writes every baseline back in the same atomic mutation (the full contract — the tick document's block order, the delta kinds, and the two delivery classes — lives in `_cli-fab-operator.md` § fab operator tick-start). Drop `--quiet` only when the user asks for status ("status", "any updates?", "show the fleet") — the binary's built-in time-based full document — a full `items:` document whenever the last one is 10 minutes or older — is the periodic full refresh, so no skill-side counter or timer is kept. Stdout is one document: the `tick: N` / `now: HH:MM` header lines, then `deltas:`, `candidates:`, `needs_check:`, then `items:` — or, on a quiet tick, `fleet_summary:` **in place of** `items:`. Render the frame from `items:`/`fleet_summary:` plus the entry's live cadence from a once-per-tick `rk cron list --json` read (the `target: role:operator` row — never carried from a previous tick or composed from memory) — see **Status Frame Format** below.
2. **Act on deltas** — before any answers (a `done` item's `then` may spawn the work another item waits on). Per delta kind:
   - `done` — run the item's `then` prose verbatim (the delta carries it), then report;
   - `changed` — report the field delta — including `change` appearing or switching on a pane item (report; a raw-text spawn's change shows up this way); run `then` only if it names a changed reaction;
   - `stale` — run the item's `probe.instruction` now and record the result via `fab operator track observe`;
   - `probe_error` — report; an item paused at the 3-failure cap renders 🔴 and asks the user;
   - `pane_death` / `pane_mismatch` / `agent_exited` — report (`pane_mismatch` means a recycled pane only — the fingerprint differs; detection semantics are the binary's — `_cli-fab-operator.md` § fab operator tick-start; an exited pane is never a candidate, so the §5 sweep must never type into it);
   - `stage_advance` / `review_fail` — report in the frame.

   Separately: `pending` pane items (deps done, no pane yet) run the §6 spawn sequence this tick — the confidence gate first.
3. **Answer waiting agents** — for each `candidates:` row run `rk mux capture <pane> --lines 40 --classify --json`; a class other than `none` goes through the §5 answer model, delivered via `rk mux send --answer` (§5).
4. **Ack** — `fab operator track rm <id>` for every `done` / `pane_death` / `pane_mismatch` / `agent_exited` item (for pane items also clear the window: `rk tab mark @<window_id> --off` and `rk tab note @<window_id> "✓ <id> done"`); `fab operator track observe <id> --json …` for every `needs_check:` item checked this tick, with `--seen` for the linear/slack ids handled (§7). The level-triggered deltas re-emit until acked, so a crash between diff and action loses nothing.

There is no whole-file persist step — every action above already persisted through its own verb, and the per-tick baseline writes are the binary's.

Actions (nudges, answers, removals, spawns) render as an *italic* footnote line below the frame as they happen, `·`-separated, keeping them visually subordinate to the table frame:

```
*k8ds: auto-answered 'Allow Bash: npm test?' → y · Removed ab12 (done), ef56 (pane gone) · Spawned cd34 → %7*
```

When the action log is long, the operator MAY split it across several italic lines rather than one — but each remains italic to stay subordinate to the frame.

### Status Frame Format

The frame is emitted as an assistant message that the agent harness renders as GitHub-flavored markdown in the terminal. **Render rule** (the binding constraint on every styling choice below): emit **bare markdown** — no code fence, no headings, no ANSI escapes (none of these survive the render path); the channels that DO render are **tables**, **emoji** (the only color channel), **bold** (`**…**`), *italic*, `code spans`, and plain URLs. The frame uses exactly these.

The frame has **two shapes**, chosen by which key the tick document carries (tick step 1):

- **Full frame** (`items:` present — a delta tick, a non-empty `needs_check:`, a tick 10 minutes or more after the last full document, or a user status request run without `--quiet`): a **header line** plus **one table over all items**.
- **Compact frame** (`fleet_summary:` present — a quiet tick): exactly **ONE line**, no table.

On either shape, the *italic* action-footnote line still renders whenever an action happened. Nothing renders between ticks — the frame (plus its footnote) is the only per-tick output: no restating the tick document, no echoing `candidates:`, no per-candidate "no question detected" lines.

> **Runtime no-fence rule (agent-critical)**: do NOT wrap the frame in a ` ``` ` code fence. The fenced block below is for *documentation* (so this skill file shows the literal source). At runtime the operator must emit the header and table directly into its message body — a fenced frame renders as literal text (the table would not lay out and the emoji/bold would not style).

Example (this is the literal markdown the operator emits, shown fenced here only to display the source):

```
🛰️ **Operator** · 16:36 · tick #48 · **7 tracked** · idle-every 2m · skip-if-busy

| | ID | Kind | State | Checked | Next |
|:--:|---|---|---|---|---|
| | `r3m7` | pane · foo | 🟢 apply → review | live | |
| ▶ | `k8ds` | pane · bar | 🟡 waiting · review | live | spawn ef56 |
| ⏸ | `ef56` | pane · bar | held: k8ds | — | spawn after k8ds |
| ▶ | `pr-913` | github-pr · run-kit | ✅ MERGED | 12s | spawn n34 |
| ▶ | `pr-914` | github-pr · run-kit | OPEN · armed | 2m ⚠ | arm next |
| | `linear-bugs` | linear · foo | 0 new · 2 seen | 11m 🔴 | |
| | `n1` | note | Phase 2 of 4 — hexokit after #913 | 3h | |
```

| Element | Rule |
|---|---|
| Header | `🛰️ **Operator** · {HH:MM} · tick #{N} · **{tracked} tracked** · {schedule_summary} · {deliver}` — the cadence cells are copied verbatim from the per-tick `rk cron list --json` read (tick step 1), never from memory |
| Compact frame (quiet tick) | `🛰️ **Operator** · {HH:MM} · tick #{N} · **{tracked} tracked** · {schedule_summary} · {deliver} · no change[ · {W} waiting]` — rendered from `fleet_summary:` plus the same cron read |
| `▶` | the item has a `then` (or is a `pending` pane item) |
| `⏸` | `held` — the State cell names what it waits on: `held: <dep-id>` |
| Kind | the kind, then ` · <repo basename>` when `scope.repo` is set (one table has no repo anchors; the basename keeps the row narrow) |
| State | pane items: health emoji (🟢 active · 🟡 waiting/idle · 🔴 >15m idle non-terminal · ✅ done · `⏏ shell` exited) + stage text — with no change observed, `· no change` instead of a stage (`🟢 active · no change`, `🟡 idle 12m · no change`); with a change, the stage text prefixed by the change id when it differs from the item id (`🟢 4a8m · apply`); shell/github-pr: the declared fields' current values, ✅ prefix when done; linear/slack: `{new} new · {seen} seen`; notes: the text's first line; paused: `⚪ paused` |
| Checked | `live` for pane items; age since `checked_at` (`12s`/`2m`/`3h`, floor division) for probed items, with ` ⚠` appended past `check_every` and ` 🔴` past 2×; notes: age since `updated_at`; held items `—` |
| Next | the armed action in ≤ 5 words (`then` compressed by the operator; `spawn <id>` for pending; `spawn after <dep>` for held) |
| Ordering | kind (pane, github-pr, shell, task, linear, slack, note) → `scope.repo` → id |
| Removed items | render once with ✅ on the tick they are acked, then vanish |

Health emoji are geometric-glyph-free (mono glyphs like `●◌✗` render monochrome and are NOT used); emoji is the only colour channel, the action log stays italic, and an `agent_exited` row renders `⏏ shell` in its State cell so it never reads as a live agent.

---

## 5. Auto-Nudge

The operator auto-answers routine prompts from tracked pane agents. The per-tick question-detection population (tick step 3) is the tick's `candidates:` block — each `waiting` agent first (the primary signal — see below), then each idle agent. An on-demand classify capture on an `active`/unknown (`—`) pane — an uninstrumented harness, or a mid-turn prompt not yet flipped to `waiting` — remains operator judgment, but those panes are **not swept every tick**; the per-tick sweep is `waiting`+idle only.

**The `waiting` agent-state value is the primary signal.** When a tracked pane's `@rk_pane_agent_state` is `waiting`, the agent is blocked on a human (permission prompt / menu / elicitation) — this is event-driven and covers all instrumented harnesses (Claude/codex/copilot/gemini), so it is the first-class trigger for question detection here; it also feeds the clock — a `waiting` flip fires a tick within seconds via the entry's `wake_on` (§4 The Clock). A `waiting` pane MUST be capture-classified and run through the answer model, with each **idle** pane as the per-tick fallback (the population stated above).

### Question Detection

Detection is a per-pane classification, not a manual read: for each pane in the tick's `candidates:` block, run `rk mux capture <pane> --lines 40 --classify --json`. The report carries a pending-prompt **class** — yes/no, numbered menu, colon prompt, open question, press-key, or `none` with a reason — plus the **matched line**, and the two together feed the answer model below (the classification contract is tool-owned via `rk skill`; the usage summary lives in `_cli-agents.md` § Peek). Claude Code permission/tool-approval prompts are **not** mechanized as their own class — in practice they are covered by the yes/no and numbered-menu classes; novel prompt shapes remain operator judgment via plain capture (`_cli-agents.md` § Peek).

1. **Classify**: `rk mux capture <pane> --lines 40 --classify --json` per candidate
2. **Class `none`** → stuck detection applies
3. **Any other class** → answer model

### Answer Model

Evaluate in order:

1. Binary yes/no or confirmation → `y`
2. `[Y/n]` or `[y/N]` → `y`
3. Claude Code permission prompt → `y`
4. Numbered menu or multi-choice → classify with LLM judgment using option length, semantic distinctness, surrounding context, and reversibility; no keyword list. Treat uncertainty as Strategic, then use the decision table below.
5. Open-ended answer determinable from visible context → send that answer.
6. Cannot determine keystrokes → use the decision table's hard-exclusion row.

### Non-Blocking Strategic Handling

Strategic handling MUST NOT block the tick: decide out-of-band, continue with the next tracked item in the same tick, and detect any asynchronous user resolution on a later tick.

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

`rk notify` fails silently by contract — a notification that cannot be delivered MUST NOT crash or stall the operator; it logs one line and keeps ticking.

### Sending Auto-Answers

Deliver text answers via `rk mux send <pane> "<text>" --answer` — the answer-mode gate permits `waiting` (the auto-answer's primary target) and `idle`, still refuses `active`, and validates pane existence (full contract is tool-owned via `rk skill`; the usage summary lives in `_cli-agents.md` § Pre-Send Validation). Key-name answers (bare Enter, arrows, `C-c`) ride `rk mux send --key` on the same path.

Before the send: run the §3 pre-send gate (`_cli-agents.md` § Pre-Send Validation — pane exists; state read per its step 2, expecting `waiting` or the idle fallback), then re-capture the terminal (the 20-line capture whose mechanics live in `_cli-agents.md` § Peek). If output changed since detection, abort — agent is no longer waiting. The classify capture is detection input only — it never replaces the pre-send gate or this re-capture-before-send guard (the classification capture is older than the just-in-time one, so the guard matters more, not less). If the answer appears to land but the agent does not resume: the send's delivery verification is built in — a probe failure surfaces as staged text + a stderr warning + exit 1, so re-capture and decide; never blind-resend.

### Idle Auto-Default on Strategic Escalations

A left-open Strategic prompt (§ Logging "left open") gets the auto-default on the first tick whose `candidates:` row for that pane shows a `state_duration` of **30 minutes or more** (hardcoded — no setting, no state-file field). The LLM runs no timers: run-kit's agent-state epoch already resets on any activity in the pane, so the row's duration is the idle clock. Answer: the visibly stated default (`(default: 2)`, `Press enter for 2`, `[2]`), else `1`. Hard-excluded regardless of duration: auto-picked Strategic prompts and rule-6 cannot-determine escalations.

### Logging

- Auto-answer (routine): `"{change}: auto-answered '{summary}' → {answer}"`
- Auto-pick strategic (defensible recommendation): `"{change}: auto-picked strategic '{summary}' → {answer} · notified"`
- Left-open strategic (no defensible default): `"{change}: strategic '{summary}' left open · notified. Please respond."`
- Escalation (rule 6 — cannot determine keystrokes): `"{change}: can't determine answer for '{summary}'. Please respond."`
- Auto-default (after 30m idle on a left-open strategic prompt): `"{change}: auto-defaulted after 30m idle: '{summary}' → {answer}"`
- Notification send failure (fail-silent — logged, ticking continues): `"{change}: notify failed ({channel}). Continuing."`

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

### The pane Kind and Spawn Rules

The `pane` item is the operator's core kind — **any agent session in a tmux pane**. A fab change, when present, is observed (`scope.change`/`stage`) and drives the built-in completion, but is never required to track. **Pipeline-first** and **spawn in a worktree** bind **spawns**, not tracking. Tracking an existing pane the user points at needs no gate and no kind question — "track %222" maps to exactly one verb, zero questions:

```sh
fab operator track add <slug> --kind pane --pane %222 --repo <repo> --session <session>
# <slug> = the change id when the pane already has one (fab pane map --all-sessions resolves it),
#          else the pane's window / worktree name
```

Two principles govern every spawn of new work (they bind spawns only; a GitHub PR, a Linear query, or an existing pane the user points at needs neither):

- **Pipeline-first** — new work MUST enter through `/fab-new`, then `/fab-fff`, `/fab-ff`, or `/fab-continue`; never send raw implementation instructions or use `/fab-continue` to skip intake. Direct actions are exactly the §1 maintenance allowlist.
- **Spawn in a worktree** — reserve the operator pane for orchestration. Every pipeline command, including a one-line change, starts with `wt create --non-interactive` (§2 wt Gate) and runs in a fresh agent tab.

A spawn that deliberately parks early — e.g. a `/fab-ff` run, which stops after hydrate — MUST be tracked with `--stop-stage hydrate`; otherwise the item never completes and sits in the list until the user stops it.

### Spawning an Agent

Every spawn flow is **repo-targeted and session-targeted**: the operator first establishes **which repo** the work targets (the existing item's `scope.repo`, or the repo the user names) and **which tmux session** the new agent window must land in, then runs every step against those — not against the operator's own repo or its ambient session.

The spawn sequence is:

1. **Establish target repo** — determine the absolute main-worktree root the work targets. For an already-tracked item, use its `scope.repo` (or the `branch_map` pair after removal). For a linear/slack-driven spawn, use the spawning item's `scope.repo` (§7). For a fresh user request, use the repo the user names (default: the repo the operator was launched in).
2. **Establish target session** — determine the tmux session the new agent window must land in; it is passed explicitly at step 7. Candidates are `rk mux sessions --json` rows with `role: "user"` minus the operator's own session; pick the candidate holding the most panes whose `cwd` is under the target repo (from the tick's `rk mux panes --json` snapshot, or `fab pane map --all-sessions` on demand). Tie → the §8 "Spawn target session" setting; still torn → ask once when attended, notify via §5 when unattended. Announce the choice and auto-set §8. Never trust a persisted `scope.session` alone; the ambient session is never an implicit target.
3. **Create worktree** — run the repo-targeted, probe-and-route procedure in `_cli-external.md` § wt (gated by the §2 wt Gate on the first agent spawn of the session); never rely on the operator's CWD
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
5. **Resolve dependencies** — if the item has a non-empty `depends_on` list, resolve it per repo: same-repo deps cherry-pick into the worktree, cross-repo deps are ordering-only barriers (see Dependency Resolution below)
6. **Read the target repo's session command** — resolve it per `_cli-agents.md` § Spawn Composition, in the **role-addressed** form with the target repo named, as one structured query: `fab agent default -o yaml --repo <target-repo>`. Read both `command` (the session command `--print` would emit) and `skill_prefix` from that single document per `_cli-agents.md` § Skill Prompts — never a separate `--print` call. The operator-specific rule: **always pass `--repo <target-repo>`** — do NOT use the operator's own `config.yaml`, since each repo may configure a different provider/session command. (The provider-addressed form documented there is for ad-hoc cross-provider sessions, not operator worker spawns, which must carry the target repo's `default`-role profile.)
7. **Open agent tab** — compose the selected skill and arguments per `_cli-agents.md` § Skill Prompts, then open the tab per `_cli-agents.md` § Spawn Composition's raw form: the session command and the prompt ride as **argv tokens after `--`**, never a composed shell string (rk appends the interactive shell fallback itself; the argv form needs no shell escaping):

   ```sh
   rk tab new --session =<session> --cwd <worktree> --name <wt> --ready --json -- <spawn-argv…> "<skill-prompt>"
   ```

   (`=<session>` pins the target session from step 2 — there is no ambient-session fallback, and an unknown session errors loudly at spawn: surface it, never silently retry against the ambient session. `<wt>` is the worktree name from step 3.) The `--json` report carries `window_id` and `pane_id` — the marks below consume the window id, step 8 the pane id. Read the `ready:` verdict per `_cli-agents.md` § Await: `parked`/`narrow` are ordinary judgment rounds (answer the wall the snippet shows, probe again); `gone` gets one bounded retry, then escalate.

   **Window marks** — right after the spawn, mark the window: `rk tab mark @<window_id> auto` and `rk tab note @<window_id> "<id> · <stage>"`. While the pane item is `waiting` on a human, the mark flips to `rk tab mark @<window_id> blocked`; at removal (tick step 4) the operator runs `rk tab mark @<window_id> --off` and `rk tab note @<window_id> "✓ <id> done"`. The mark/note contracts are tool-owned (`rk skill`).
8. **Track the item** — unconditionally and silently, one command (the `branch_map` pair rides the `track add`; contract in `_cli-fab-operator.md` § fab operator track). The id is the change id when known, else the worktree name from step 3; raw-text spawns are tracked at spawn and their change appears later as a `changed` delta:

   ```sh
   # new item (raw-text or fresh known-change spawn):
   fab operator track add <id> --kind pane --pane <pane-id> --session <session> --repo <repo> [--change <change-id> --branch <branch>] [--stage <stage>] [--stop-stage <stage>] [--spawned-by <item-id>] [--depends-on <id,…>]
   # an item that already exists as `pending` (a queued change spawned by tick step 2):
   fab operator track update <id> --scope '{"pane":"<pane-id>","session":"<session>"}'
   ```

   (`track add` refuses a duplicate id, so the pending case must go through `update`; the fingerprint is recorded on either path.) Never ask whether to track.

### Dependency Resolution

**Dependency satisfied.** A `depends_on` entry is satisfied when the dependency **item is `done`** — for a pane item with a change, its built-in predicate fired (`review-pr` done/skipped when its `stop_stage` is null, or at/past its `stop_stage`) — **and**, for a same-repo dependency with a null `stop_stage`, its PR exists (`gh pr view <dep-branch> --json url` succeeds, so the branch is pushed and stable). Neither tracking, a `branch_map` entry, nor the branch being minted is satisfaction — all three exist from the moment the dep's agent spawns. An unsatisfied dependency **holds the item** (state `held`), re-derived by the binary each tick, logging `"{change}: waiting on dependency {dep} ({dep.repo}) to complete."` — when the last dep completes, the item flips to `pending` and the tick's step 2 spawns it. Every consumer below (the spawn sequence, queues, linear/slack spawns) gates on this definition.

Dependency resolution is **two-tier**, split by repo. Each entry in `depends_on` is classified by comparing the dependency's `repo` (from its item's `scope.repo`, or its `branch_map` `{ branch, repo }` pair after the item is gone) against **this change's** `repo`:

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

   `origin/{default_branch}` is the cherry-pick base in step 3 below. Fetching first prevents a stale base even on correctly-defaulted repos; resolving the name makes queue chaining usable on repos whose default branch isn't `main`.

0.5. **Readiness gate** — for each same-repo change ID still in the tracked list, check it is satisfied per **Dependency satisfied** above. If any is not, the item stays `held` (no branch lookup, no cherry-pick) and subsequent ticks re-check. A dep that has left the tracked list (present only in `branch_map`) was removed on its own `done` and passes this gate. The `stacked-prs` variant below inherits this gate.

1. **Resolve same-repo dependency branches** — For each same-repo change ID, look up its branch:
   - First from the tracked item's `scope.branch` (if the dep is still tracked).
   - Otherwise from `branch_map` (the `{ branch, repo }` pair, if the dep has left the tracked list) — `branch_map` is keyed by the dependency's change id: its `scope.change`, which equals its id for known-change spawns.

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

**Cross-repo resolution.** For each cross-repo dependency, do not cherry-pick. Instead, before spawning, verify the dependency is satisfied per **Dependency satisfied** above. If it is not, the item stays `held` and subsequent ticks re-check; spawn once every cross-repo barrier clears, logging the wait with the shared line from that definition.

**Same-repo resolution (`stacked-prs` mode).** Steps 1–3 are skipped for same-repo dependencies — the dependent's branch is created off its nearest same-repo predecessor's *branch* at the §6 spawn sequence's worktree/branch step instead of off `origin/{default_branch}` (the probe-and-route per `_cli-external.md` § wt: existing dep branch → `wt create --checkout <dep-branch>` route). The squashed `"operator: cherry-pick"` commit does not exist for same-repo deps in this mode. After `/git-pr` creates the dependent's PR, the operator retargets its base to the dependency's branch: `gh pr edit <pr> --base <dep-branch>` (`/git-pr` itself is unchanged and mode-unaware). The merge-all choreography for the stack lives under Ordered Merge below. Dependency-branch drift after a dependent PR exists (a dep's review-pr rework moving its branch) is out of scope — the same exposure exists in the cherry-pick model; conflicts surface at merge-all and escalate.

**Why `origin/{default_branch}` as base (same-repo only)**: Each same-repo dependency branch carries its full transitive same-repo dependency content. When the operator spawned dep B, it cherry-picked dep A into B's worktree first. B's branch therefore contains A's commits. So `origin/{default_branch}..<B-branch>` gives the complete transitive closure within the repo — no need to chase transitive same-repo deps manually. This is why only direct/leaf same-repo dependencies need cherry-picking. (Cross-repo deps carry no such transitive content — they are ordering-only.)

### Dependency Declaration

Dependencies are declared through two conversational paths, which coexist:

1. **Explicit**: "cd34 depends on ab12" — recorded at spawn (step 8's `--depends-on ab12`), or mid-flight via `fab operator track update cd34 --depends-on ab12` (the flag replaces the list — carry the existing deps along)
2. **Queue chaining (implicit)**: resolve ordering per § Queues → Queue ordering

### Working a Change

> **Pipeline-first routing (§6 The pane Kind and Spawn Rules):** all three work paths below MUST go through the fab pipeline (`/fab-new` then a pipeline command for new work; the appropriate stage for already-intaked changes) — never raw implementation instructions to agent panes.

Every form runs §6's target-repo + target-session → worktree → guarded activation → dependencies → target-repo session command → tab → track sequence:

1. **Existing change:** use the item's `scope.repo` (or `branch_map`) and select skill `fab-fff` with argument `<change>` and render it per `_cli-agents.md` § Skill Prompts; the transient override targets the pipeline and spawn step 4 activates the pointer.
2. **Raw text** (for example, "fix login after password reset"): use the named repo (default operator launch repo) and select skill `fab-new` with the raw description as its argument string; render it per `_cli-agents.md` § Skill Prompts. The existence guard skips activation until `/fab-new` creates and activates the change at Step 10.
3. **Backlog ID or Linear issue:** resolve it first (optional `idea` lookup per `_cli-external.md` § Delegation and binary gate), then select skill `fab-new` with argument `<id>` through the same rendering procedure. The existence guard skips activation and `/fab-new` owns it.

On completion (all three): PR ready, optionally archive. Both raw text and backlog paths use `/fab-new` to generate a proper intake with traceability. `/fab-new` captures the raw input in the intake's Origin section — the user just says "fix [description]" and the operator does the rest.

### Queues

User provides a queue of changes. Confirmation prompt reflects the active mode:
- **Default (`cherry-pick-ladder`):** "Confirm upfront (creates PRs — merge after review)."
- **`merge-auto`:** "Confirm upfront (merges PRs on completion)."
- **`stacked-prs`:** "Confirm upfront (creates stacked PRs — merge after review)."

A queue **may span repos**, with mixed dependency semantics: implicit chaining (and explicit `depends_on`) cherry-picks **within a repo** and **degrades to an ordering-only barrier across repo boundaries** (per Dependency Resolution above; the nearest-same-repo-predecessor rule is defined in Queue ordering below). Worked example — a chain `ab12 → cd34 → ef56` where `cd34` lives in a different repo: `cd34` gets `depends_on: [ab12]` (cross-repo — waits for `ab12` to be satisfied per § Dependency Resolution **Dependency satisfied**, no cherry-pick), and `ef56` (back in `ab12`'s repo) gets `depends_on: [ab12]` — its nearest same-repo predecessor — and cherry-picks from it; queue order still runs `ef56` after `cd34`.

Once the user confirms, persist the queue as **N `fab operator track add --kind pane` calls** — the first carries `--mode <name>` (the binary resolves the mode by the ladder below and prints `mode: <name> (<source>)`; contract in `_cli-fab-operator.md` § fab operator track), and every change after the first carries `--depends-on` per Queue ordering. Chained items are added with no pane: they sit `held` (or `pending` once their deps are done) until a tick's step 2 spawns them. There is no queue state beyond the items themselves — skipping an entry is `fab operator track rm <id>`, pausing the unspawned entries is `fab operator track update <id> --pause`.

Queue ordering:

| Strategy | Description |
|----------|-------------|
| User-provided | Run in the exact order given. Implicit chaining by default: every change after the first gets `depends_on: [<nearest-same-repo-predecessor>]` — the closest earlier queue entry in the same repo (cherry-picked); when no earlier entry shares the repo, the immediately previous entry (cross-repo → ordering-only). |
| Confidence-based | Sort by confidence score descending. Highest-confidence first (independent changes) |
| Hybrid | User provides constraints (partial order); operator sorts unconstrained by confidence |

**Merge modes** — three flat names. **Mode resolution (silent by default):** when the user's queue request names no mode — explicitly or via natural language — resolve it by the ladder explicit user instruction / `--mode` flag > config `autopilot.merge_mode` (the key keeps its historical name) > built-in `cherry-pick-ladder` and proceed WITHOUT asking. `track add --mode` prints `mode: <name> (<source>)` where source is `flag` / `config` / `default` — that line is how the operator learns the resolved mode and its source (the operator never parses config files itself). State the resolved mode inside the **existing** upfront queue-confirmation line above (which already varies by mode), so the user vetoes in the same breath — no extra round-trip.

Pause and ask the mode question ONLY on one of exactly two misfits:

1. The resolved mode is `merge-auto` but the queue has same-repo `depends_on` entries — implicit chaining is disabled in that mode, so the declared dependency semantics contradict it.
2. The user's own message conflicts with the resolved mode (e.g. they say "merge as you go" while the resolved mode is a held mode like `cherry-pick-ladder` or `stacked-prs`).

No other condition triggers the mode question. When the operator DOES ask, the question MUST include the at-a-glance glyphs, the three compact box diagrams below, and a one-line tradeoff per mode (an invalid config value is the binary's own actionable `track add` error, not a misfit — it never reaches a question).

At a glance: `▂▄▆` cherry-pick-ladder · `░▒▓█` merge-auto · `▄▀` stacked-prs. (The diagrams below are skill documentation — never emit them into the **status frame**, which stays fence-free per §4. That prohibition is status-frame-only: a mode question is an ordinary conversational message, where the fenced diagrams render fine and are REQUIRED.)

- **`cherry-pick-ladder`** (default) — PRs are created but not merged until the user explicitly requests merging; implicit chaining is active (per Queue ordering, "User-provided").

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

- **`merge-auto`** — merge-as-you-go: **arm** each PR on completion (§6 Auto-Merge Choreography — a one-PR merge sequence; all five rules apply) instead of merging and foreground CI-waiting; once the merge is verified on a later tick (the `github-pr` item's `done` delta runs its `then`), `git fetch origin` and rebase the next change onto `origin/{default_branch}` (the default branch resolved per Dependency Resolution step 0 — never a hardcoded `origin/main`). Implicit chaining is disabled in this mode — each change rebases onto `origin/{default_branch}` independently. Natural language equivalents: "merge as you go", "merge on complete", "merge each when done".

  ```
      ┌───┐          ┌───┐          ┌───┐
      │ A │          │ B │          │ C │
  ────┴─▼─┴●─────────┴─▼─┴●─────────┴─▼─┴●──▶ main
         merged         merged         merged
  ```

  Nothing coexists and nothing is held: the operator arms each PR the moment it lands and GitHub merges it into main when checks pass (▼ into ●), main advances, and the next change starts from the advanced line — no batch review, no re-stacking.

- **`stacked-prs`** — `cherry-pick-ladder` merge timing (PRs created up front, merged only on explicit user request) with true stacked-PR topology for same-repo chains: the dependent's branch is created off its dependency's *branch* (no cherry-pick commit) and its PR targets the dependency's branch, so each PR diff shows only its own delta. Mechanics: same-repo resolution in Dependency Resolution above; merge-all choreography in Ordered Merge. Natural language equivalents: "stacked PRs", "stack the PRs".

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
3. **Resolve dependencies + open tab + track** — §6 spawn sequence steps 4–8 (existence-guarded pointer activation, same-repo cherry-pick / cross-repo ordering-only barriers per Dependency Resolution). Step 7's skill selection is `fab-fff` with argument `<change>` (or the appropriate skill for its current stage), rendered through the shared procedure — so the dispatch happens **once, at spawn**; do NOT send the command again after the tab opens
4. **Monitor** — normal tick detection handles progress
5. **Record** — when the current change is satisfied per § Dependency Resolution **Dependency satisfied** (its `done` delta observed **and** its PR URL collected), collect the PR URL. The `{ branch, repo }` pair is already in `branch_map` — `track add` recorded it at spawn
6. **Spawn next** — the next chained item flips `held` → `pending` on its own when its deps complete; the tick's step 2 spawns it (confidence gate first), embedding its command at spawn
7. **Report** — `"ab12: PR ready. 1 of 3 complete. Starting cd34."`
8. **(After all complete) Summary** — list all PR links with per-repo dependency annotations and per-repo merge order suggestion (see Queue Completion Summary below)

In `merge-auto` mode, steps 5–8 arm the just-shipped PR (§6 Auto-Merge Choreography — one `github-pr` item whose `then` fetches, rebases, and spawns the next change) and **defer the next change's spawn to the tick that verifies the merge** — spawning earlier would start the next agent from a stale `origin/{default_branch}`. On the verified-merge tick: run `git fetch origin`, rebase the next change onto `origin/{default_branch}` (resolved per Dependency Resolution step 0), report the merge, and spawn.

Queue-driven items display `▶` in the status frame while pending (§4); completion remains visible through ✅.

#### Queue Completion Summary

When all changes in a `cherry-pick-ladder` or `stacked-prs` queue complete, the operator displays a completion summary. When the queue spans repos, each PR is **annotated with its repo**, and the suggested merge order respects **each repo's own dependency chain** (a per-repo PR sequence):

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

**CI failure during ordered merge (halt-dependents-only)**: If CI fails on a PR, the operator halts **that repo's merge sub-sequence** AND **any repo whose queued items carry a cross-repo `depends_on` into the failed chain — transitively**. In an armed sequence, the halt first disarms the halted sequences' remaining armed `github-pr` items (Auto-Merge Choreography rule 5); independent sub-sequences keep theirs and continue. "Dependent" is determined over the cross-repo `depends_on` graph: a repo halts if any of its queued items depends (directly, or via another already-halted item) on a PR in the failed chain. **Truly independent repos' sub-sequences continue merging.** The operator does not abandon the queue; it isolates the blast radius to the failure's dependency cone. On completion it reports which sub-sequences halted vs. completed and escalates the failure to the user:

```
ab12: CI failed (~/code/foo). Halted: foo sub-sequence; bar (cross-repo dep into foo). Completed: baz sub-sequence (2 PRs merged). Fix foo and retry.
```

Queue state IS the tracked items — there is no separate queue block in the state file; a restarted operator re-orients from `fab operator track list` and resumes.

**Failures**: review exhausted → skip (`track rm`). Rebase conflict mid-queue → skip (`merge-auto` only; does not apply in `cherry-pick-ladder` since there are no rebase steps). Rebase conflict during a `stacked-prs` merge-all → escalate (never skip). Cherry-pick conflict → escalate (do not skip). Pane dies → 1 respawn (`--reuse`), then skip. Stage timeout (>30m) → flag. Total timeout (>2h) → flag.

**Interrupts**: "stop after current", "skip <change>", "pause", "resume" — acknowledged immediately, and persisted through the `track` verbs: `fab operator track rm` on the not-yet-spawned items once the current change lands (or immediately to abandon the queue), `track rm <id>` to skip an entry, `track update <id> --pause` / `--resume` on the unspawned items.

#### Auto-Merge Choreography

The CI gate for `cherry-pick-ladder` merge-all (and `merge-auto`'s per-PR merge on completion) — the modes where PRs target main. Instead of merging and foreground-waiting for CI — the longest operator-busy stretches in the orchestration lifecycle — the operator **arms** each PR with GitHub auto-merge (`gh pr merge --auto --squash` — the method flag is explicit and REQUIRED: flagless `--auto` may prompt or take the repo's default, unsafe on an unattended tick; a user-directed method maps to `--merge`/`--rebase`) and lets GitHub merge it when checks pass, verifying on later ticks. Arming is part of the user's confirmed merge-all — the "merge all" confirmation is the §3 Destructive-tier confirm for the whole sequence, so no per-PR re-confirmation is asked.

**`stacked-prs` is excluded and keeps the manual merge-all above**: its inter-merge choreography (retarget-verify, `rebase --onto`, force-push) is operator-sequenced anyway, and an armed stacked PR can merge into its dependency's *branch* (destroying the stack silently) or fire on stale-green checks after GitHub's no-re-CI base retarget.

All five rules are MUSTs:

1. **Sequential arming.** At most one armed PR per repo-sequence — one armed `github-pr` item. Arm PR_n only after PR_{n-1}'s merge is **verified** — its item's `done` delta, a merge event on the PR's timeline, never an assumption. Never arm a PR whose base is another PR's branch.
2. **Arming-failure shapes.** A draft PR MUST be readied first with `gh pr ready` (fab's `/git-pr` creates drafts, so this is every queued PR). An "already clean" rejection (the repo has no required checks, so auto-merge has nothing to wait for) → merge directly. Auto-merge disabled on the repo → fall back to the foreground CI-wait choreography above for the sequence.
3. **Stall rule.** The tick's mechanical probe of a still-unmerged armed `github-pr` item feeds two inspections. A **failed required check** (`gh pr checks`) is Ordered Merge's CI failure — disarm per rule 5 and apply the halt-dependents-only policy (a failed check never makes the PR `CONFLICTING`; auto-merge just silently never fires). `unchanged ≥ 3` on the item (the binary-owned consecutive-no-delta counter) with no failed check → check `gh pr view --json mergeable` (the field is `mergeable` — `gh pr view --json` exposes no `mergeableState` field); `CONFLICTING` → disarm and escalate. Both shapes are event-less — the tick's probe IS the poll.
4. **The sequence is a chain of `github-pr` items.** Starting a merge sequence MUST add one item per PR: `fab operator track add pr-<n> --kind github-pr --scope '{"repo":…,"pr":n}' --then "arm next: gh pr merge --auto --squash <next>" --depends-on pr-<n-1>` — the first item takes no `--depends-on` and is armed immediately. An armed PR **outlives the operator** (it survives compaction, `/clear`, crash, and abandonment), and the items ARE the sequence state: a restarted operator re-orients from `fab operator track list` and resumes verifying/arming. Per tick, a `done` delta on `pr-<n>` runs its `then` verbatim — arming PR_{n+1} (readying a draft per rule 2) — and each armed item's `unchanged` counter feeds rule 3's stall threshold; on halt, rule 5's disarm runs over the remaining armed items, which are then removed via `track rm`.
5. **Disarm on halt.** Any halt or escalation — CI failure, stall, conflict — MUST run `gh pr merge --disable-auto` on the remaining armed PRs of the **halted sequences** (one `github-pr` item each): the failing repo's sub-sequence plus its transitive cross-repo dependent cone, matching the halt-dependents-only policy (which assumes unstarted merges stay unstarted — armed auto-merge violates that without the disarm). Independent sub-sequences keep their armed items and continue. A user "stop" is global and disarms every armed PR.

**Per tick while a merge sequence is in progress**: the tick probes each armed `github-pr` item mechanically (§4 Tick Behavior) — merged → the item's `done` delta fires its `then` (arm the next PR per rule 1); unmerged → rule 3's checks and the `unchanged` stall counter advance. Merge-all consumes no foreground attention between arms, and the armed items keep the clock live like any other tracked item (§4 The Clock) — there is no gap and no run-kit follow-up.

---

## 7. Linear and Slack Items

Linear and Slack items are standing queries over an external source, probed by the operator itself (`probe: agent`) — the binary never runs them; it lists due items under the tick's `needs_check:` block, and the operator runs the instruction and records the result. Users create them conversationally: "watch Linear project DEV for new issues, spawn agents, stop at intake."

On each tick, for each `needs_check:` row (tick step 2's `stale` handling and step 4's ack ride the same flow):

1. **Run the instruction** — `probe.instruction` names the MCP call (Linear via `mcp__claude_ai_Linear__list_issues`, Slack via `mcp__claude_ai_Slack__slack_read_channel`), with `scope.query` as the API filter. On failure: `fab operator track observe <id> --error "<msg>"` and skip this item for the tick — the third consecutive failed observe auto-pauses the item (`paused: true`); alert the user then.
2. **Deduplicate** — skip ids in the item's `seen` list (binary-capped at 200, oldest pruned — an item that reached `stop_stage` stays in `seen` and MUST NOT be respawned).
3. **Evaluate `then`** — the item's `then` prose carries trigger conditions, label filters, spawn/notify actions, and concurrency limits; a concurrency limit counts tracked items where `scope.spawned_by == <id>`.
4. **Act** — for each source item that passes: run the §6 spawn sequence with the item's `scope.repo` as the target repo, rendering the appropriate initial skill invocation (e.g., `fab-new` with argument `DEV-123`), tracked with `--stop-stage <scope.stop_stage>` and `--spawned-by <id>`; then record the observe — `fab operator track observe <id> --json '{"new":[…]}' --seen <item-id>…` (`--seen` appends idempotently, only after a successful spawn or a deliberate skip).
5. **Report** — `"linear-bugs: DEV-1024 — Fix auth redirect (72m old). Spawning."`

When a spawned pane item completes (its `done` delta — at/past the item's `stop_stage`, or `review-pr` done/skipped when it is null), report: `"linear-bugs: DEV-1024 completed intake."`

### Conversational Map

Every utterance maps to a `track` verb (contracts in `_cli-fab-operator.md` § fab operator track) — the operator composes flags, never YAML:

| Utterance | Verb |
|---|---|
| "Track pane %222" · "watch the agent in tireless-perch" | `fab operator track add tireless-perch --kind pane --pane %222 --repo /home/x/code/fab-kit --session work` |
| "Watch Linear project DEV for bugs older than 1 hour, spawn into ~/code/foo, stop at intake" | `fab operator track add linear-bugs --kind linear --instruction '…' --check-every 5m --scope '{"repo":"/home/x/code/foo","stop_stage":"intake","query":{…}}' --then '…'` |
| "Tell me when run-kit #913 merges, then start n34 in hexokit" | `fab operator track add pr-913 --kind github-pr --scope '{"repo":"/home/x/code/hexokit","pr":913}' --check-every 2m --then 'spawn n34 in ~/code/hexokit via /fab-fff'` |
| "Poll the prod deploy every 2 minutes until green" | `fab operator track add deploy-prod --kind shell --argv <cmd…> --fields status --check-every 2m --done-when 'status == "green"'` |
| "Pause / resume the Linear watch" · "stop watching" | `fab operator track update linear-bugs --pause` / `--resume` · `fab operator track rm linear-bugs` |
| "Hold the ticks for 30 minutes" · "tick every 10 minutes for the next two hours" | `rk cron mute <id> --for 30m` (§4 Mute and Lease) · `fab operator track clock --every 10m --for 2h` |

---

## 8. Configuration

### One Operator Per Server

The isolation unit is the **tmux server**. There is exactly **one operator per tmux server** — it spans every session and every repo on that server, orchestrating all of them through a single server-keyed state file (§4, §9). This matches the server-wide singleton already enforced by the `operator` window (`fab operator` switches to the existing window rather than creating a second one).

- **Multiple sessions, same server** share one operator and one state file. The operator addresses their agents by the `(session, repo, pane)` tuple (§1) — where `session` scopes the addressing/display, never the identity: the pane ID is the join key, and a session can change mid-lifetime (§1, §4); there is no per-session or per-repo operator.
- **A second operator means a second tmux server** — start one on a separate socket (`tmux -L <label>`). Its state file is keyed by that socket, so the two operators never collide. There is no `--name` dimension; the server boundary is the only isolation knob. Sends on a non-default socket carry the matching flag: `rk mux -L <label> send`.

### Settings

| Setting | Default | Override via natural language |
|---------|---------|------------------------------|
| Stuck threshold | 15m | "flag agents stuck for more than {N}m" |
| Spawn target session | inferred (§6 step 2 majority rule; auto-set on each announced inference) | "spawn into session {name}" |

Cadence is not a session setting — the binary derives it from the tracked set and applies it via `rk cron edit` (§4 The Clock); a bounded user override is `fab operator track clock --for` (§4 Mute and Lease).

These settings are session-scoped and reset on compaction, `/clear`, or session restart (§4 Post-Compaction Reload); they are not operator-state-file fields. The **strategic auto-default threshold is hardcoded at 30m** (§5) — there is deliberately **no** setting for it.

---

## 9. Key Properties

| Property | Value |
|----------|-------|
| Requires active change? | No |
| Runs `fab preflight`? | No |
| Read-only? | No — sends commands, auto-answers, mutates the operator state file (only via `fab operator track` verbs) |
| Idempotent? | Yes — state re-derived every tick |
| Advances stage? | No |
| Outputs `Next:` line? | No — ends with ready signal |
| Reads change/plan artifacts? | As the work needs — any plan, roadmap, intake, or task document required to drive tracked work (§1); none are startup always-loads (§2) |
| Requires tmux? | Yes — hard stop without it |
| Requires run-kit? | Yes — hard stop without it (§2 rk Gate: `command -v rk` + `rk cron list --json`) |
| Launcher delegation? | Yes — bare `fab operator` hands the launch to `rk operator` when a capable rk is on PATH (probe, pass-through, and failure semantics owned by `_cli-fab-operator.md` § fab operator); the rows below describe the binary's built-in launcher, whose behavior is owned by the CLI reference |
| Requires a git repo? | No — `fab operator` opens its window in the repo root inside a repo, else `os.Getwd()` (neutral parent dir). Errors only if both fail |
| Requires a `fab/` project? | No — session command comes from the project's `providers.claude.interactive_command` when `fab/` is resolvable, else `spawn.DefaultSpawnCommand` (the template `claude --permission-mode bypassPermissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}`). No project `providers`/`agent:` block is read on a `fab/`-less launch |
| Coordinating-agent model | Operator role — `fab operator` resolves the `operator` role (`agent.ResolveRole`; a Tier-1 role, so the `agent.session` knob picks its provider), reads that provider's `interactive_command`, injects the profile via `spawn.WithProfile` (**substitutes** into a `{model}`/`{effort}` template — the built-in claude default is templated — or **appends** `--model`/`--effort` to a plain command carrying no placeholder); falls back to the built-in operator profile + built-in claude provider on any failure (incl. no resolvable `fab/` project) |
| Cadence | rk cron operator-tick entry — union predicate (schedule + `wake_on: agent-state-change`), seeded by `rk operator`; the `track` verbs mute/unmute it via the tracked predicate and re-derive its schedule from the tracked set via `rk cron edit` only on change; lease = bounded snooze, bounded cadence override = `track clock --for` (§4 The Clock, §4 Mute and Lease); live cadence rendered from `rk cron list --json` (§2 Init step 5, §4 Status Frame Format); quiet ticks render the one-line compact frame (§4 Status Frame Format); tick payload is the bare `operator tick` — never a slash command (§4 Tick Payload) |
| Uses the operator state file? | Yes — the tracked-item list + branch map in the server-keyed path (§2 Init step 1); reads via `fab operator state` / `fab operator track list`, every mutation through a `fab operator track` verb — never a hand-write (§4 Tracked Items) |
| Multi-repo / multi-session? | Yes — one operator per tmux server spans all its sessions and repos via the `(session, repo, pane)` addressing tuple |
