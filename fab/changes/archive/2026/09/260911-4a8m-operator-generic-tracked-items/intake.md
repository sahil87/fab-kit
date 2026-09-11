# Intake: Operator Generic Tracked Items — one tracked-item model, probe loop, derived schedule, one-table frame

**Change**: 260911-4a8m-operator-generic-tracked-items
**Created**: 2026-09-11

## Origin

> Implement Part B ("Change B — generic tracked list, probe loop, derived schedule, one-table frame") of the plan at fab/plans/sahil/26-09-11-operator-generic-tracking.md. Read the full plan file first (Decisions already taken, Current state, and the B1-B5 subsections) before writing intake.md - do not restate it from memory. Scope is B1-B5 only (Go tracked-item schema + verbs + migration, tick-start polling, derived rk cron schedule via rk cron edit, the operator skill rewrite of Sections 4/5/6/7, and the docs sweep in B5). Run-kit dependencies (9aup rk cron edit/skip-if-busy/--idle-every, r5ao structured schedule/wake_on/respawn in rk cron list --json, wzve rk tab new, hzih rk tab mark/note, 7pek has_agent, a7g2 rk mux capture --classify) have been verified released on this machine (rk v3.19.46) - the plan's do-not-start gate is cleared. Change A (loading/dependency split) is a separate prerequisite change not yet done; if `_cli-fab-operator.md`/`_cli-fab-pane.md` do not exist yet, treat the file split in A as not-yet-done and scope this change to B only, still referencing `_cli-fab.md` as it exists today - flag the ordering conflict in intake as an assumption for confirmation rather than silently doing A's split inline.

**Interaction mode**: one-shot `/fab-new` from a written plan. The design conversation happened earlier in the `/fab-discuss` study `docs/wiki/operator-tick-anatomy.html` and is frozen in `fab/plans/sahil/26-09-11-operator-generic-tracking.md`; that plan's **Decisions already taken** table is treated as settled (not reopened here) and its **Open decisions (B)** list is resolved in this intake as graded assumptions. The user confirmed mid-intake: *"change A is done"* — verified: Change A (`260910-si4k`, PR #662) is merged as `eda41f7d` on `main`, and `src/kit/skills/_cli-fab-operator.md` (265 lines) and `_cli-fab-pane.md` (352 lines) exist. **The ordering conflict the prompt anticipated does not apply**; this intake references the split files.

**Facts verified at intake (2026-09-11, this machine, rk v3.19.46, fab 2.24.9)**:

- `rk cron edit <id>` accepts `--idle-every <dur>`, `--every <dur>`, `--backoff --min --max`, `--deliver immediate|when-idle|skip-if-busy`; a schedule/deliver change logs `rescheduled` and resets the anchor (9aup).
- `rk cron list --json` rows carry structured `schedule: {kind, min, max}` plus `schedule_summary`, `wake_on: {event, scope, debounce}`, `deliver`, `if_absent`, `respawn: [...]`, `muted`, `pinned`, `target: "role:operator"` (r5ao). The live operator entry today: `id: hk6c`, `backoff 1m→30m`, `deliver: immediate`.
- `rk tab new [--session =S] [--cwd DIR] [--name N] [--json] [--ready [--timeout SECS]] -- CMD [ARG…]` prints `{"session","window_id","pane_id"}` (+ `"ready": ready|parked|narrow|running|gone` with `--ready`); argv is single-quoted per token; the `; exec "${SHELL:-/bin/sh}"` tail is appended by rk (wzve).
- `rk tab mark [@N] manual|auto|blocked[:1|:2|:3] | --off` writes `@rk_win_marker`; `rk tab note [@N] <text> | --off` writes `@rk_win_note` (120-char cap, epoch-prefixed) (hzih).
- `rk mux panes --json` rows carry `has_agent: true|false|null` (null = unknown/uninstrumented) alongside `agent_state`, `agent_state_duration`, `command`, `cwd` (7pek).
- `rk mux capture <target> [--lines N] --classify [--json]` reports the pending-prompt class (yes/no, numbered menu, colon prompt, open question, press-key) or `none` with a reason (a7g2).
- `fab pane questions` and `fab pane window-name` have **no Go code consumer outside their own files** (`src/go/fab/internal/dispatch/dispatch.go` mentions `window-name` only in a comment); they leave the operator skill in this change, and the Go verbs plus their `_cli-fab-pane.md` entries **stay to bound scope** — deleting them is a follow-up backlog item, not part of B.
- `fab-operator.md` is **905 lines today** (the plan's "≈780 after A" was a target; A added the rk gate and clock sections while deleting fallbacks). B's ≈480 line target is a target, not an acceptance gate.

## Why

**The operator's tracked-thing model is fab-change-shaped, and the #913 incident was that model working as written.** The only thing the binary probes per tick is the set of tmux panes in `monitored`. Linear and Slack are reached only through a typed `watch` (the Go verb rejects any other `--source`), GitHub only while the operator itself has armed a merge, and notes never. Anything else the user hands the operator — "tell me when #913 merges", a non-fab branch, a deploy, a CI run, a ticket to merely watch — has no home except a prose note that §4 Notes explicitly labels "checked by operator judgment per tick". On quiet ticks the skill also forbids any output beyond the one-line frame, so nothing prompts that judgment. The operator reported a PR "queued behind #913" fifteen minutes after it had merged because nothing in the tick loop ever asked GitHub.

**If left as is**, every new long-running thing needs a new typed section (`monitored`, `watches`, `autopilot`, `notes` each have their own verbs, lifecycle, completion semantics, and clock-predicate clause; the merge sequence is a sixth pseudo-entity persisted as note prose), and every one of them is invisible to the mechanical tick unless someone writes Go for it. The frame has no freshness column and no note rows, so a stale item cannot be *seen* either.

**Why this approach**: collapse the five owned sections into one `tracked` list where every item declares *how* it is probed (`pane` | `shell` | `agent` | `none`), *when* it is done (`done_when`), *what happens* then (`then` prose), and *how often* to look (`check_every`). The binary runs every mechanical probe at `tick-start --diff` and emits field-scoped deltas; the LLM is told which `agent`-probed items are due (`needs_check:`) instead of being trusted to remember; the frame shows one table with a `Checked` age column so staleness is visible. The clock is derived from the tracked set by the binary and applied via `rk cron edit` — a per-tick LLM cadence judgment would be conversation-only state, the same failure class. The choreography (dependencies, merge modes, ordered merge, auto-merge) stays inline in the skill because it is the operator's main job and must survive a compaction reload with a merge sequence armed. Probes run in fab, never as an rk `wake_on: probe-change`, because a raw-output fingerprint wakes on noise; the guard is field-scoped comparison declared at `track add`.

## What Changes

### B1 — One tracked-item schema (Go `operator_state.go` + skill §4)

Replace the owned sections `monitored`, `watches`, `autopilot`, `notes` (+ `notes_seq`) and the merge-sequence-in-a-`coordination`-note with **one owned section** `tracked:`. `branch_map`, `tick_count`, `last_tick_at` stay. The binary owns the schema, timestamps, atomic write, and the tolerant-read/typed-write posture exactly as today (unknown top-level keys survive; an invented field inside `tracked` cannot survive a mutation).

```yaml
tick_count: 47
last_tick_at: "2026-09-11T16:33:00Z"
tracked:
  - id: r3m7                       # fab-change: the change ID; other kinds: LLM-chosen slug, unique in the list
    kind: fab-change               # fab-change | github-pr | linear | slack | shell | task | note
    probe: { mode: pane }          # pane items: the binary snapshots the pane each tick (check_every ignored)
    check_every: null
    done_when: null                # null for fab-change = the kind's built-in predicate (review-pr done/skipped, or at/past stop_stage)
    then: null                     # prose the LLM executes when done/changed fires; null = report only
    depends_on: []                 # item ids; unsatisfied → the item is `held` (see lifecycle)
    scope:                         # kind-specific metadata, opaque to the probe runner
      pane: "%3"                   # null until spawned (a queued fab-change item has no pane yet)
      repo: /home/user/code/foo
      session: work
      branch: 260324-r3m7-add-retry-logic
      stage: apply                 # baseline for stage_advance / review_fail diffs (was monitored.stage)
      agent: active                # baseline agent state (was monitored.agent)
      stop_stage: null
      spawned_by: null             # item id of the linear/slack item that spawned it, or null
      merge_mode: cherry-pick-ladder   # set when the item was queued via a chain (replaces autopilot.mode)
    last: {}                       # pane items: empty — the snapshot is the state
    checked_at: "2026-09-11T16:33:00Z"
    unchanged: 0                   # consecutive due probes with no field delta (stall counter, binary-owned)
    failures: 0                    # consecutive probe errors; 3 → paused: true
    paused: false
    added_at: "2026-09-11T16:20:00Z"
    updated_at: "2026-09-11T16:33:00Z"

  - id: pr-913
    kind: github-pr
    probe:
      mode: shell
      argv: [gh, pr, view, "913", --repo, sahil87/run-kit, --json, "state,mergedAt,mergeable"]
      fields: [state, mergedAt, mergeable]     # only these keys of the JSON output are compared and stored
    check_every: 2m
    done_when: 'state == "MERGED"'
    then: "spawn n34 in ~/code/hexokit via /fab-fff"
    depends_on: []
    scope: { repo: /home/user/code/hexokit, pr: 913 }
    last: { state: OPEN, mergedAt: null, mergeable: MERGEABLE }
    checked_at: "2026-09-11T16:33:00Z"
    unchanged: 4
    failures: 0
    paused: false
    added_at: …
    updated_at: …

  - id: linear-bugs
    kind: linear
    probe:
      mode: agent
      instruction: "mcp__claude_ai_Linear__list_issues project=DEV status in [Backlog,Todo] assignee=@me; report new issue ids not in seen"
    check_every: 5m
    done_when: null                # standing item — never done by itself; removed by the user
    then: "for each new id: spawn /fab-new <id> in ~/code/foo, stop at intake; max 2 concurrent (count items with scope.spawned_by == linear-bugs)"
    scope: { repo: /home/user/code/foo, stop_stage: intake, query: { project: DEV, status: [Backlog, Todo], assignee: "@me" } }
    seen: [DEV-988, DEV-992]       # bounded dedupe list, 200 cap oldest-pruned, binary-owned (was watches.known + completed)
    last: { new: [] }              # what the last observe reported
    checked_at: …
    …

  - id: n1
    kind: note
    probe: { mode: none }
    check_every: null
    done_when: null
    then: null
    text: "Phase 2 of 4 — hexokit n34 spawns after run-kit #913 merges"   # 500-char cap (was notes.text)
    scope: { refs: [pr-913, n34] }
    checked_at: null
    added_at: …
    updated_at: …
branch_map:                        # unchanged — written by `track add --kind fab-change`, read by choreography, cleared by `branch-map rm`
  ab12: { branch: 260324-ab12-fix-auth, repo: /home/user/code/foo }
```

**Kinds and their defaults** (the binary fills defaults at `track add`; the LLM may override any field the kind allows):

| Kind | Default probe | Default `done_when` | Default `then` | Required `scope` | Notes |
|---|---|---|---|---|---|
| `fab-change` | `pane` | built-in: `review-pr` done/skipped, or at/past `scope.stop_stage` | null (report + `track rm`) | `repo`, `branch`; `pane`/`session` null until spawned | Carries today's `monitored` semantics. `track add --kind fab-change <id>` also writes `branch_map[id] = {branch, repo}`. `merge_mode` resolved by the ladder flag > config `autopilot.merge_mode` > `cherry-pick-ladder`; binary prints `mode: <name> (<source>)` when a chain is added |
| `github-pr` | `shell`: `gh pr view <n> --repo <owner/repo> --json state,mergedAt,mergeable`, fields `[state, mergedAt, mergeable]` | `state == "MERGED"` | null | `repo`, `pr` | `then: "arm next"` prose is how a merge sequence chains (B4) |
| `linear` / `slack` | `agent` (LLM runs the MCP call named in `probe.instruction`) | null (standing) | LLM prose | `repo` (spawn target), `stop_stage`, `query` | Gains `seen` (200-cap). Replaces `watches` |
| `shell` | `shell` (argv + fields required at add) | required at add | optional | — | Generic: deploys, CI runs, any JSON-emitting command |
| `task` | `none` | null | prose | — | A held action with `depends_on` and `then` but nothing to probe (e.g. "when both PRs merge, tag v3.20") — the operator runs `then` when deps are done |
| `note` | `none` | null | null | `refs` optional | Replaces `notes`; `text` 500-char cap; rendered in the frame |

**Probe modes**:

- `pane` — binary snapshot as today (`rk mux panes --json` via the pane-map pipeline), joined on `scope.pane`; `check_every` is ignored (every tick).
- `shell` — binary runs `probe.argv` (an argv list, **never a shell string**; no per-item environment — the operator's environment is inherited), 10 s timeout, stderr swallowed, stdout parsed as JSON (an object; a non-JSON or non-object stdout is a probe error). Only `probe.fields` (top-level keys, or dotted paths `a.b` into nested objects) are extracted, compared against `last`, and stored back into `last`. Exit non-zero, timeout, or parse failure ⇒ `failures++`; a success resets `failures` to 0.
- `agent` — the LLM runs `probe.instruction` (MCP or anything else) when the tick lists the item under `needs_check:`, and records the result with `track observe`. The binary never runs it.
- `none` — never probed (`note`, `task`).

**`done_when` grammar** (pinned to kill the highest rework risk): one or more clauses joined by ` and `; each clause is `<path> <op> <literal>` where `<path>` is a field name or dotted path (a leading `.` is accepted and stripped), `<op>` ∈ `==`, `!=`, and `<literal>` is a JSON scalar (`"MERGED"`, `42`, `true`, `null`). No `or`, no comparison operators, no functions, no regex. A path absent from `last` compares as `null`. The predicate is evaluated by the binary after every successful probe or `observe`; it is validated at `track add`/`update` (a malformed predicate exits non-zero with the offending clause). Examples: `state == "MERGED"`; `conclusion == "success" and status == "completed"`; `ready != false`.

**`check_every`**: Go-duration string; **floor 1m** (a smaller value exits non-zero); **default 5m** for `shell`/`agent` items when omitted; must be null for `pane`/`none` items (the binary nulls it silently).

**Lifecycle / states** (binary-derived, emitted in the tick document and rendered in the frame):

| State | Condition |
|---|---|
| `held` | any `depends_on` item is not done (still in the list, or never existed → `probe_error`-class warning naming the missing id) |
| `pending` | `fab-change` with `scope.pane` null and deps satisfied — to be spawned by the LLM this tick |
| `live` | pane item whose pane is present and change-matched (state text = agent state + stage) |
| `watching` | `shell`/`agent` item probed on cadence, `done_when` false |
| `stale` | `agent` item with `now - checked_at > 2 × check_every` |
| `paused` | `failures` reached 3 (auto) or `track update --pause` |
| `done` | `done_when` true (or the fab-change built-in fired) — level-triggered until `track rm` |

**Verbs** (`src/go/fab/cmd/fab/operator_track.go`, replacing `operator_monitored.go`, `operator_watch.go`, `operator_autopilot.go`, `operator_note.go` — the old verbs are **removed outright**, no aliases; the operator skill is their only caller):

```
fab operator track add <id> --kind <kind> [--probe shell --argv <tok>... --fields <a,b>] [--probe agent --instruction <text>]
                          [--check-every <dur>] [--done-when <pred>] [--then <text>] [--depends-on <id,...>]
                          [--scope <json-object>] [--text <note-text>] [--mode <merge-mode>]
fab operator track update <id> [--check-every <dur>] [--done-when <pred>] [--then <text>] [--depends-on <id,...>]
                             [--scope <json-object, merged per key>] [--text <t>] [--pause|--resume]
fab operator track observe <id> --json '<object>' [--seen <item-id>]...    # agent-probe result: stores declared fields (or the whole object when fields is empty) into last, sets checked_at, resets failures, evaluates done_when, appends --seen ids (200-cap)
fab operator track rm <id>                                                  # the ack for done / removal; fab-change: branch_map entry retained
fab operator track list [--kind <k>] [--json]                               # human: one line per item id · kind · state · checked age · next; --json: the items array
fab operator branch-map rm <change-id> | --all                              # unchanged
fab operator state | tick-start | time                                      # unchanged signatures; state prints OPEN NOTES header from kind: note items
```

- `--scope` for `fab-change` accepts the existing enrol flags too (`--pane --repo --session --branch --stage --agent --stop-stage --spawned-by`) as sugar so the spawn step stays one command; the binary writes them into `scope` and writes `branch_map`.
- Duplicate id, unknown id, malformed `--done-when`, `--check-every` below floor, `--argv` with a `shell` probe missing `--fields`, or `--depends-on` naming an unknown id ⇒ exit non-zero, one-line error, no state written.
- **Clock predicate** becomes **"any item not `done`"** (a list with only `done` items counts as untracked). Every `track` verb keeps the before/after flip → `rk cron mute <id>` / `--off` behaviour from `operator_clock.go`, and additionally runs the B3 schedule reconcile.

**Migration** (binary-owned conversion + a markdown migration file):

- The binary detects a **legacy-shaped file** (any of `monitored`/`watches`/`autopilot`/`notes` present and `tracked` absent) on the first read-modify-write by any `fab operator` verb — including `tick-start` — and converts in the same atomic write: `monitored.<id>` → `fab-change` items (scope from the entry fields, `checked_at` = `last_transition`); `watches.<name>` → `linear`/`slack` items with `probe: agent`, `probe.instruction` composed from `source` + `query`, `scope.{repo: target_repo, stop_stage, query}`, `seen` = `known ∪ completed`, `then` = `instructions`, `check_every: 5m`; `autopilot.queue` entries not yet in `completed` → `fab-change` items (pane null → `pending`/`held`) chained by `depends_on` (nearest same-repo predecessor rule, cross-repo → immediate predecessor), `scope.merge_mode` = `autopilot.mode`; open `notes` → `kind: note` items with `id: n<N>`; resolved notes dropped. The legacy keys are deleted from the file after conversion. **Refusal**: when `autopilot.state == running`, the verb exits non-zero with `operator state file has a running autopilot queue — finish or stop it (fab operator autopilot stop on fab ≤2.24) before upgrading`, and writes nothing. <!-- clarified: auto-convert on first touch (refusing on a running queue) confirmed over a separate migrate verb — fab-clarify 2026-09-11 -->
- `src/kit/migrations/2.24.9-to-<release>.md` documents the conversion, the refusal, and the user action (finish/stop the queue before upgrading the binary); its Pre-check is `fab operator state` succeeding — it performs no file edits itself (the binary owns the file). Range fixed at release per `docs/memory/distribution/migrations.md` § Range-Based Applicability.

### B2 — `tick-start --diff` polls every mechanical probe (`operator_tick_start.go`)

`fab operator tick-start --diff [--quiet]` keeps its signature. Per tick, after the pane snapshot:

1. **Due set**: every `shell` item with `paused: false` and `now - checked_at ≥ check_every` (a null `checked_at` is always due). `pane` items are always due. `agent` items are never run — due ones are *listed*.
2. **Run** each due `shell` probe (argv, 10 s timeout, stderr swallowed, sequentially — a 20-item fleet at 10 s worst-case is 200 s; bound total wall time at 60 s and report the rest as `probe_error: skipped (tick budget)` without touching `failures`; parallel execution is a follow-up if fleets grow).
3. **Compare** declared `fields` against `last`; write `last`, `checked_at`, `unchanged` (0 on delta, +1 otherwise), `failures` in the same atomic mutation as the tick bookkeeping and the pane baseline.
4. **Evaluate** `done_when`, `depends_on`, staleness.

**Stdout document** — `tick:`/`now:` header lines, then YAML blocks in this pinned order: `deltas:`, `candidates:`, `needs_check:`, then `items:` **or** `fleet_summary:`:

```yaml
tick: 48
now: 16:36
deltas:
  - kind: changed              # changed | done | stale | probe_error | pane_death | pane_mismatch | agent_exited | stage_advance | review_fail
    id: pr-913
    fields: { state: { from: OPEN, to: MERGED } }        # changed only — one entry per differing declared field
  - kind: done
    id: pr-913
    then: "spawn n34 in ~/code/hexokit via /fab-fff"      # done only — the item's then, verbatim (null when none)
  - kind: stale
    id: linear-bugs
    age: 11m                                              # since checked_at
  - kind: probe_error
    id: deploy-prod
    error: "exit 1: gh: Not Found"
    failures: 3
    paused: true                                          # true on the tick that trips the cap
  - kind: pane_death            # pane items: today's shape, keyed by id (was change)
    id: r3m7
    pane: "%3"
  - kind: stage_advance
    id: k8ds
    pane: "%7"
    from: apply
    to: review
candidates:                     # unchanged shape — waiting-first then idle, pane items only
  - pane: "%7"
    id: k8ds
    agent_state: waiting
    idle_duration: null
needs_check:                    # due agent items — the LLM runs probe.instruction and records via track observe
  - id: linear-bugs
    kind: linear
    age: 11m
    instruction: "mcp__claude_ai_Linear__list_issues project=DEV …"
items:                          # was fleet: — one row per item, ordered kind → scope.repo → id; the frame's data source
  - id: r3m7
    kind: fab-change
    state: live                 # held | pending | live | watching | stale | paused | done
    pane: "%3"
    repo: /home/user/code/foo
    session: work
    stage: review-pr
    display_state: done
    agent_state: idle
    idle_duration: 8m
    pr_url: https://github.com/acme/foo/pull/412
    checked_at: null            # pane items: null (rendered `live`)
    next: null                  # the item's then, or "spawn" for pending, or "held: <dep-id>" for held
  - id: pr-913
    kind: github-pr
    state: done
    repo: /home/user/code/hexokit
    last: { state: MERGED, mergedAt: "2026-09-11T16:35:10Z", mergeable: UNKNOWN }
    checked_at: "2026-09-11T16:36:02Z"
    check_every: 2m
    unchanged: 0
    next: "spawn n34 in ~/code/hexokit via /fab-fff"
  - id: n1
    kind: note
    state: watching
    text: "Phase 2 of 4 — …"
    updated_at: …
```

- **Delivery classes** (unchanged doctrine): `done`, `stale`, `probe_error`, `pane_death`, `pane_mismatch`, `agent_exited` are **level-triggered** — re-emitted every tick until acked (`track rm` for done/pane deltas; `track observe` clears `stale`; `track update --resume` or `track rm` clears `probe_error`). `changed`, `stage_advance`, `review_fail` are **consumed-on-read**.
- `stale` = `agent` item with `now - checked_at > 2 × check_every` (or `checked_at` null and `added_at` older than 2×). `⚠` in the frame at `> check_every` is a rendering rule, not a delta.
- `probe_error` at `failures == 3` sets `paused: true`; the item stops being probed until `track update --resume` (which zeroes `failures`).
- `agent_exited` uses `has_agent` from `rk mux panes --json`: `false` ⇒ exited; `true` ⇒ alive; **`null` ⇒ fall back to today's process-tree walk** (observed: the operator's own pane reports `null`). The walk is kept for the `null` case only.
- `fleet_summary:` keeps its five keys on a quiet tick (`--quiet`, no deltas, tick not a multiple of 10); `tracked` counts **all** items not done, the four state counts cover pane items only, so the documented invariant becomes `tracked ≥ waiting + idle + active + unknown`. A `changed`/`done`/`stale`/`probe_error` delta or a non-empty `needs_check:` forces the full document.
- Empty tracked list: the snapshot subprocess is skipped and every block emits `[]` (or the zero `fleet_summary:`), as today.

### B3 — Derived schedule via `rk cron edit` (`operator_clock.go`)

`syncOperatorClock` grows a **schedule reconcile** that runs after every `track` mutation and at the end of `tick-start --diff`, right after the existing mute/unmute flip logic:

| Tracked set (items not done) | Derived schedule | Derived deliver |
|---|---|---|
| empty | *(muted — unchanged)* | — |
| only `pane`/`none` items | `--backoff --min 1m --max 30m` | `immediate` |
| any `shell`/`agent` item **and** the operator pane has an agent-state epoch | `--idle-every <min(check_every) over shell+agent items>` | `skip-if-busy` |
| any `shell`/`agent` item, no epoch | `--every <min(check_every)>` | `skip-if-busy` |

- **Epoch detection**: the `rk mux panes --json` row for the operator pane (the pane whose window carries `@rk_win_role=operator`, else the current `$TMUX_PANE`) has non-null `agent_state` (rk's `--idle-every` keys on the pane's idle epoch, produced by the same agent-state instrumentation).
- **Only on change**: the derived `{kind, min, max | every}` and `deliver` are compared against the structured `schedule`/`deliver` fields of the resolved entry (`target == "role:operator"`, `name == "operator tick"` tiebreak) from `rk cron list --json`; equal ⇒ no call. Otherwise exactly one `rk cron edit <id> <schedule flags> --deliver <policy>` — `exec.LookPath`-gated, argv-only, 5 s timeout, fail-silent (never changes the verb's exit, stdout, or saved state), same posture as the mute calls.
- **Bounded user override**: a lease (`rk cron mute <id> --for <dur>`) is untouched by the reconcile (a muted/leased entry is still edited — the schedule takes effect when the lease expires). A user who wants a different cadence for a while ("tick every 10 minutes for the next two hours") gets an **override with expiry**: `fab operator track clock --every 10m --for 2h` (a sixth `track` sub-verb) writes the top-level `clock_override: { schedule: {…}, deliver, until: <ts> }`; the reconcile applies it instead of the derived value until `until` passes, then reverts. An override with no `--for` is rejected (unbounded overrides are the failure class Mute and Lease already forbids); `track clock --off` clears it early.
- Cross-repo contract: fab is one more `rk cron edit` caller; nothing changes in run-kit. The `rescheduled` log line and anchor reset are rk's documented behaviour and are surfaced in the frame's cadence cell on the next tick.

### B4 — Skill rewrite (`src/kit/skills/fab-operator.md` §4, §5 detection, §6 spawn + choreography, §7; `_cli-fab-operator.md`, `_cli-agents.md`, `_cli-external.md`)

**§4 Tick Behavior → four steps**:

1. **Snapshot** — `fab operator tick-start --diff --quiet` (drop `--quiet` on a user status request). Render the frame from `items:`/`fleet_summary:` + the entry's live cadence (`rk cron list --json`, read once per tick, never carried from memory).
2. **Act on deltas** — `done`: run the item's `then` prose (verbatim from the delta), report; `changed`: report the field delta, run `then` only if it names a `changed` reaction; `stale`: run the item's `probe.instruction` now and `track observe`; `probe_error`: report (paused ⇒ 🔴 and ask the user); `pane_death`/`pane_mismatch`/`agent_exited`: report; `pending` fab-change items whose deps are done: run the §6 spawn sequence (confidence gate first). Act on deltas **before any answers**.
3. **Answer waiting agents** — for each `candidates:` row run `rk mux capture <pane> --lines 40 --classify --json`; a class other than `none` goes through the §5 answer model (unchanged), delivered via `rk mux send --answer`. `fab pane questions` leaves the skill (the verb stays in Go for dispatch).
4. **Ack** — `track rm <id>` for every `done`/`pane_death`/`pane_mismatch`/`agent_exited` item (fab-change items: `rk tab mark --off` + `rk tab note "✓ <id> done"` on the window); `track observe <id> --json …` for every `needs_check:` item the LLM checked this tick (with `--seen` for linear/slack new ids handled).

Steps 3–7 of today's list (Watches, Autopilot dispatch, Removals, Observed-field updates, Clock) are absorbed: watches are `needs_check:` rows, autopilot is `pending` items, removals are step 4, baseline writes are the binary's, the clock is derived (B3).

**§4 Operator State File / Monitored Set / Branch Map / Notes** → one **Tracked Items** section: the schema block above (reference), the kinds table, the lifecycle table, and the rule "the operator never hand-writes this file — every mutation is a `track` verb". The four note kinds and the routing-doctrine table are deleted; the one surviving routing rule ("anything still true for a different operator next month is not operator state — route via `idea`") stays as one sentence.

**§4 Status Frame Format** — one table over all items:

```
🛰️ **Operator** · 16:36 · tick #48 · **7 tracked** · idle-every 2m · skip-if-busy

| | ID | Kind | State | Checked | Next |
|:--:|---|---|---|---|---|
| | `r3m7` | fab-change · foo | 🟢 apply → review | live | |
| ▶ | `k8ds` | fab-change · bar | 🟡 waiting · review | live | spawn ef56 |
| ⏸ | `ef56` | fab-change · bar | held: k8ds | — | spawn after k8ds |
| ▶ | `pr-913` | github-pr · run-kit | ✅ MERGED | 12s | spawn n34 |
| ▶ | `pr-914` | github-pr · run-kit | OPEN · armed | 2m ⚠ | arm next |
| | `linear-bugs` | linear · foo | 0 new · 2 seen | 11m 🔴 | |
| | `n1` | note | Phase 2 of 4 — hexokit after #913 | 3h | |
```

| Element | Rule |
|---|---|
| Header | `🛰️ **Operator** · {HH:MM} · tick #{N} · **{tracked} tracked** · {schedule_summary} · {deliver}` — cadence copied from `rk cron list --json` fields |
| Compact frame (quiet tick) | `🛰️ **Operator** · {HH:MM} · tick #{N} · **{tracked} tracked** · {schedule_summary} · {deliver} · no change[ · {W} waiting]` — shape unchanged from today plus the deliver policy |
| `▶` | item has a `then` (or is a `pending` fab-change) |
| `⏸` | `held` — State cell names what it waits on: `held: <dep-id>` |
| Kind | the kind, then ` · <repo basename>` when `scope.repo` is set (one table has no repo anchors; the basename keeps the row narrow) |
| State | pane items: health emoji (🟢 active · 🟡 waiting/idle · 🔴 >15m idle non-terminal · ✅ done · `⏏ shell` exited) + stage text; shell/github-pr: the declared fields' current values, ✅ prefix when done; linear/slack: `{new} new · {seen} seen`; notes: the text's first line; paused: `⚪ paused` |
| Checked | `live` for pane items; age since `checked_at` (`12s`/`2m`/`3h`, floor division) for probed items; ` ⚠` appended past `check_every`; ` 🔴` past 2×; notes: age since `updated_at`; held items `—` |
| Next | the armed action in ≤ 5 words (`then` compressed by the LLM; `spawn <id>` for pending; `spawn after <dep>` for held) |
| Ordering | kind (fab-change, github-pr, shell, task, linear, slack, note) → `scope.repo` → id |
| Removed items | render once with ✅ on the tick they are acked, then vanish |
| Kept rules | no-fence rule, emoji as the only colour channel, italic action footnote, two-shape rule, `⏏ shell` marker |
| Deleted | the Idle Message section (nothing renders between ticks); per-repo `📂` anchors; the separate Watches table; the Health column (folded into State) |

**§5 Auto-Nudge** — the answer model, escalation, logging are unchanged; **Question Detection** re-points to `rk mux capture --classify` (class + matched line) as the primary signal after `waiting`, replacing the `fab pane questions` sweep description.

**§6 Spawning an Agent** — steps 1–6 keep their intent; steps 2 and 7–8 change:

- Step 2 session inference collapses to the **majority rule over `rk mux sessions --json`** `role: user` rows minus the operator's own session: the session holding the most panes whose `cwd` is under the target repo wins (from the same `rk mux panes --json` snapshot); ties → the §8 setting; still torn → the one ask. Tiers (a)–(d) and their prose are deleted.
- Step 7: `rk tab new --session =<session> --cwd <worktree> --name <wt> --ready --json -- <spawn argv…>` — argv tokens, never a composed shell string; rk appends the shell fallback, so the `; exec "$SHELL"` composition and the `-P -F` print leave the skill. `ready: parked|narrow|gone` is handled per `_cli-agents.md` § Await (gone ⇒ bounded retry). The `»<wt>` window name becomes plain `<wt>` plus `rk tab mark @<window_id> auto` and `rk tab note @<window_id> "<id> · <stage>"`; removal runs `rk tab mark --off` + `rk tab note "✓ <id> done"`. `fab pane window-name` leaves the skill.
- Step 8: `fab operator track add <id> --kind fab-change --pane <pane_id> --session <session> --repo … --branch … [--stage …] [--stop-stage …] [--spawned-by …] [--depends-on …]`.
- "Pipeline-first" and "spawn in a worktree" move from §1 Principles into the fab-change kind's paragraph; the `wt` gate (§2) runs only when a fab-change item is about to spawn.

**§6 Choreography** (target ≈180 lines from 235; diagrams kept): "Dependency satisfied" is defined once against items — a `depends_on` entry is satisfied when that item is `done` (fab-change: the built-in predicate **and**, for a same-repo dep with null `stop_stage`, its PR exists). Autopilot becomes "a chain": the user's queue is N `track add --kind fab-change` calls with `depends_on` = nearest same-repo predecessor (cross-repo → immediate predecessor) and `--mode` on the first (printed `mode: <name> (<source>)`); the confirmation line, the two misfit conditions, the three mode diagrams, Queue ordering, Queue Completion Summary, Ordered Merge (incl. stacked-prs retarget/rebase and halt-dependents-only) stay. **Auto-Merge Choreography rule 4** is rewritten: starting a merge sequence = one `github-pr` item per PR (`track add pr-<n> --kind github-pr --scope '{"repo":…,"pr":n}' --then "arm next: gh pr merge --auto --squash <next>" --depends-on pr-<n-1>`); the first is armed immediately; per tick a `done` on PR_n runs its `then`; the **stall rule** reads `unchanged ≥ 3` on an armed item to trigger `gh pr checks` + `mergeable` inspection; **disarm on halt** runs `gh pr merge --disable-auto` over the remaining armed items of the halted sequences. The `coordination` note and `notes_seq` disappear. `▶` marks chained items.

**§7 Watches → "Linear and Slack items"**: `probe: agent` with the MCP call named in `probe.instruction`; dedupe against `seen` (binary-capped at 200 via `observe --seen`); `then` prose for spawn/notify with the concurrency count over `scope.spawned_by`; auto-pause after 3 failed observes (the LLM reports a failure via `track observe <id> --error "<msg>"`, which increments `failures`). Conversational map — every utterance → a `track` verb:

- "Watch Linear project DEV for bugs older than 1 hour, spawn into ~/code/foo, stop at intake" → `track add linear-bugs --kind linear --instruction '…' --check-every 5m --scope '{"repo":"/home/x/code/foo","stop_stage":"intake","query":{…}}' --then '…'`
- "Tell me when run-kit #913 merges, then start n34 in hexokit" → `track add pr-913 --kind github-pr --scope '{"repo":"/home/x/code/hexokit","pr":913}' --check-every 2m --then 'spawn n34 in ~/code/hexokit via /fab-fff'`
- "Poll the prod deploy every 2 minutes until green" → `track add deploy-prod --kind shell --argv <cmd…> --fields status --check-every 2m --done-when 'status == "green"'`
- "Pause / resume the Linear watch" → `track update linear-bugs --pause` / `--resume`; "stop watching" → `track rm`; "hold the ticks for 30 minutes" → `rk cron mute <id> --for 30m` (unchanged); "tick every 10 minutes for the next two hours" → `track clock --every 10m --for 2h`.

**Helpers**: `_cli-fab-operator.md` § fab operator is rewritten around `track` (verb contracts, tick document, clock reconcile paragraph, migration note; `enroll/update/remove`, `note`, `watch`, `autopilot` subsections deleted); `_cli-agents.md` § Spawn Composition re-points the open step to `rk tab new` argv form, § Pre-Send Validation / Peek to `rk mux capture --classify`; `_cli-external.md` § wt § Operator Spawning Rules moves its choreography sentences into the fab-change kind paragraph and keeps the probe-and-route recipe; `_preamble.md` is untouched.

### B5 — Docs

- `docs/memory/runtime/operator.md` — rewritten around items: state model, tick document, lifecycle, derived clock, frame; Design Decisions gain entries for one-tracked-list, probes-in-fab-not-rk, derived-schedule, level-triggered `done`, `done_when` grammar; entries that describe the deleted surfaces (Notes vs Watches routing, `»`/`›` markers, coordination note) are rewritten as superseded-by-items.
- `docs/memory/runtime/agent-primitives.md` and `pane-commands.md` — `fab pane questions`/`window-name` prose loses its "operator sweep" framing (the verbs stay for dispatch); `has_agent` tri-state fallback documented.
- `docs/memory/distribution/migrations.md` — a `2.24.9-to-<release>` section (binary-owned state-file conversion, running-autopilot refusal).
- `docs/specs/operator.md` — version-history row **v11**: generic tracked items, probe loop, derived schedule, one-table frame.
- `docs/specs/skills.md` — the `/fab-operator` entry (§ Context, helper list unchanged; tick flow four steps).
- Constitution unaffected (no new MUST; "binary owns state" restated, not changed).

## Affected Memory

- `runtime/operator`: (modify) rewrite around tracked items — state model, tick document, lifecycle, derived clock reconcile, one-table frame, `track` verbs, migration; superseded design decisions marked
- `runtime/agent-primitives`: (modify) `has_agent` tri-state liveness (walk kept for `null`), question detection via `rk mux capture --classify`, `rk tab new` argv spawn
- `runtime/pane-commands`: (modify) `fab pane questions`/`window-name` no longer operator-consumed (dispatch-only); pane-map `has_agent` enrichment
- `distribution/migrations`: (modify) the `2.24.9-to-<release>` state-file conversion section
- `distribution/kit-architecture`: (modify) "Operator State File" pointer — sections list `tracked` + `branch_map`

## Impact

- **Go** (`src/go/fab/cmd/fab/`): new `operator_track.go` (+ tests) replacing `operator_monitored.go`, `operator_watch.go`, `operator_autopilot.go`, `operator_note.go` (+ their 1,569 lines of tests, rewritten against items); `operator_state.go` (typed `trackedItem`, legacy detection + conversion); `operator_tick_start.go` (probe runner, predicate evaluator, `needs_check`, `items:`; `operator_tick_diff_test.go` 948 lines re-based); `operator_clock.go` (schedule derive + `rk cron edit`, `clock_override`); `pane_map.go` (`has_agent` passthrough). A `done_when` parser/evaluator as a small internal package with table tests. Constitution constraint: `_cli-fab-operator.md` updated with every signature.
- **Skill**: `fab-operator.md` 905 → target ≈480 lines (choreography ≈180); `_cli-fab-operator.md` § fab operator rewritten; `_cli-agents.md` and `_cli-external.md` re-pointed. Deployed copies regenerate via `fab sync` — never edited directly.
- **Kit**: one migration file; `templates/`, `scaffold/` untouched.
- **Docs**: five memory files, two specs. Sibling sweep before finishing apply (`fab/project/code-quality.md` § Sibling Sweeps): grep repo-wide for `monitored|watches|autopilot|notes_seq|coordination note|enroll|fab pane questions|window-name|»|›|Idle Message|fleet:` across `src/kit`, `docs/memory`, `docs/specs` and reconcile every hit.
- **Behavioural contract changes**: the `fab operator enroll/update/remove/watch/autopilot/note` verbs are removed (the skill is their only caller — verified by grep; `docs/` mentions are sweep targets); the tick document renames `fleet:` → `items:` and re-keys pane deltas on `id`; the operator's cron entry schedule is now edited by fab (users who hand-tuned it will see it re-derived — the bounded `track clock --for` override is the sanctioned path).
- **Not affected**: `fab dispatch`, `fab pane` verbs (stay for dispatch), the state-file path/slug contract with run-kit, `rk operator` seeding, `_preamble.md`, `fab agent`.
- **Sizing**: ≈14 tasks, FULL lane. Highest rework risks — pinned above: `done_when` grammar, Checked/⚠/🔴 semantics, `fleet_summary` invariant, verb removal sweep.

## Open Questions

None outstanding. The three questions raised at intake were resolved in `/fab-clarify` on 2026-09-11 (see Clarifications): legacy state files auto-convert on first touch; the repo basename rides the Kind cell of the one-table frame; `rk tab mark` uses `auto` / `blocked` / `--off`.

## Clarifications

### Session 2026-09-11

| Q | Question | Answer |
|---|----------|--------|
| 1 | Legacy state-file migration: auto-convert on first touch vs explicit `track migrate` verb (row 9) | Auto-convert on the first read-modify-write by any `fab operator` verb, refusing with a one-line error while `autopilot.state == running`; the markdown migration file documents and edits nothing |
| 2 | Repo visibility in the one-table frame: basename in the Kind cell vs per-repo `📂` anchors (row 14) | Basename rides the Kind cell (`fab-change · foo`); no anchors, one table |
| 3 | `rk tab mark` mapping (row 15) | `auto` when tracked, `blocked` while a pane item is `waiting` on a human, `--off` + `rk tab note "✓ <id> done"` at removal |

### Session 2026-09-11 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 4 | Confirmed | — |
| 7 | Confirmed | — |
| 13 | Confirmed | — |
| 14 | Confirmed | Via Q2 |
| 15 | Confirmed | Via Q3 |
| 17 | Confirmed | — |
| 19 | Confirmed | — |
| 20 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Change A is merged (eda41f7d, PR #662); B references `_cli-fab-operator.md`/`_cli-fab-pane.md` and performs no split — the prompt's ordering-conflict flag does not apply | Verified on disk and in `git log`; user confirmed "change A is done" | S:95 R:90 A:100 D:100 |
| 2 | Certain | The plan's "Decisions already taken" table (probes in fab not rk; choreography inline; rk hard dependency; `--idle-every` + `skip-if-busy` clock kind; binary derives schedule via `rk cron edit`; one-table frame `▶ · ID · Kind · State · Checked · Next`) is settled and not reopened | Plan explicitly says do not reopen at intake; user restated scope as B1–B5 | S:95 R:70 A:95 D:100 |
| 3 | Certain | All six run-kit dependencies are released on this machine (rk v3.19.46) — each flag/field was exercised at intake | Observed `rk cron edit --help`, `rk cron list --json`, `rk tab new/mark/note --help`, `rk mux capture --classify`, `has_agent` in `rk mux panes --json` | S:95 R:85 A:100 D:100 |
| 4 | Confident | `done_when` grammar = ` and `-joined clauses of `<path> ==/!= <json-scalar>`; no `or`, comparisons, regex, functions; validated at add/update | Plan proposes jq-style equality/inequality only; keeps the evaluator table-testable and the LLM-composed predicate hard to get wrong; extensible later — Clarified — user confirmed (auto-resolve, recommended option) | S:95 R:70 A:80 D:75 |
| 5 | Certain | `check_every` floor 1m, default 5m for shell/agent items, forced null for pane/none | Plan proposals; rk's own `--idle-every` floor is 1m-class; 5m matches today's watch feel | S:80 R:90 A:85 D:85 |
| 6 | Certain | `shell` probes are argv-only, inherit the operator's environment, no per-item `env` | Plan proposal "no"; matches the existing argv-only rk-call posture in `operator_clock.go` | S:80 R:85 A:90 D:90 |
| 7 | Certain | `linear`/`slack` items keep a bounded `seen` list (200 cap, binary-pruned via `observe --seen`) instead of relying on `last` alone | A query returns a set; new-item detection needs the handled-id set, which `last` cannot carry without becoming that list; today's `known`+`completed` cap already proves the size — Clarified — user confirmed (auto-resolve, recommended option) | S:95 R:75 A:85 D:80 |
| 8 | Certain | Migration refuses while `autopilot.state == running`, with a one-line instruction to finish or stop the queue on the old binary | Plan proposal; a running queue's `current` has in-flight side effects the conversion cannot reproduce faithfully | S:80 R:80 A:85 D:85 |
| 9 | Confident | Legacy state file is auto-converted on the first read-modify-write by any `fab operator` verb (incl. `tick-start`), not via a separate `migrate` verb; the markdown migration file documents it and edits nothing | The operator may run with no `fab/` project, so `/fab-setup migrations` cannot be the sole trigger; auto-upgrade of a binary-owned file is conventional, but an explicit verb is a valid alternative the user may prefer — Clarified — user confirmed (auto-resolve, recommended option) | S:95 R:55 A:50 D:35 |
| 10 | Certain | Old verbs (`enroll/update/remove`, `watch *`, `autopilot *`, `note *`) are removed outright with no aliases | Repo-wide grep shows `fab-operator.md`/`_cli-fab-operator.md` as the only kit callers; docs hits are sweep targets; aliases would keep five schemas alive | S:75 R:70 A:90 D:85 |
| 11 | Certain | `fab pane questions` and `fab pane window-name` Go verbs and their `_cli-fab-pane.md` entries stay; only the operator skill stops using them; deletion is a follow-up | No Go consumer outside their own files (dispatch.go references `window-name` in a comment only); keeping them bounds B's scope and rework risk | S:90 R:90 A:95 D:90 |
| 12 | Certain | `agent_exited` uses `has_agent` tri-state: `false` ⇒ exited, `true` ⇒ alive, `null` ⇒ today's process-tree walk | Observed `has_agent: null` on uninstrumented panes at intake; plan says keep the walk only when 7pek is absent — `null` is the per-pane form of absent | S:80 R:85 A:90 D:85 |
| 13 | Certain | Epoch detection for `--idle-every` vs `--every` = the operator pane's `rk mux panes --json` row has non-null `agent_state` | rk's `--idle-every` keys on the pane's idle epoch, produced by the same agent-state instrumentation — Clarified — user confirmed (auto-resolve, recommended option) | S:95 R:85 A:75 D:80 |
| 14 | Confident | Health emoji folds into the State cell; the Checked cell carries `live`/age/`⚠`/`🔴`; repo basename rides the Kind cell as ` · <basename>` | Plan pins the six columns and drops per-repo anchors; the Kind cell is the least loaded place for scope — Clarified — user confirmed (auto-resolve, recommended option) | S:95 R:85 A:70 D:60 |
| 15 | Confident | `rk tab mark auto` on spawn/track, `blocked` while a pane item is `waiting`, `--off` + `rk tab note "✓ <id> done"` at removal | hzih's mark modes are manual / auto / blocked; `auto` reads as operator-driven, `blocked` as human-needed; cheap to change, so reversibility carries it — listed under Open Questions for a quick veto — Clarified — user confirmed (auto-resolve, recommended option) | S:95 R:85 A:50 D:40 |
| 16 | Certain | `fleet_summary:` keeps its five keys; `tracked` counts all not-done items, the four state counts cover pane items; invariant becomes `tracked ≥ sum` | Plan says the block keeps its shape; adding keys would ripple into the compact-frame rule | S:75 R:85 A:85 D:80 |
| 17 | Certain | Autopilot queue → `fab-change` items with `scope.pane` null (`pending`/`held`) chained by `depends_on`; `merge_mode` stored per item in `scope`, resolved by the existing ladder at the first `track add --mode` | Plan B1 ("autopilot queue → items chained by depends_on"); mode fixity per queue is preserved by stamping each chained item — Clarified — user confirmed (auto-resolve, recommended option) | S:95 R:70 A:85 D:80 |
| 18 | Certain | Merge sequence = chain of `github-pr` items with `depends_on` and `then: arm next`; stall rule reads the binary's `unchanged` counter; `coordination` note and rule 4 deleted | Plan B4; the `unchanged` counter replaces the LLM-counted "3 consecutive ticks" with binary state | S:85 R:70 A:85 D:85 |
| 19 | Certain | Bounded cadence override = `track clock --every <dur> --for <dur>` writing a top-level `clock_override` with expiry; unbounded overrides rejected; leases untouched by the reconcile | Plan: "user overrides are bounded (lease / edit-with-expiry)"; mirrors the Mute and Lease no-indefinite rule — Clarified — user confirmed (auto-resolve, recommended option) | S:95 R:80 A:80 D:75 |
| 20 | Certain | Shell probes run sequentially under a 60 s per-tick budget; over-budget items report `probe_error: skipped (tick budget)` without touching `failures` | Predictable and simple; 10 s timeout × small fleets fits; parallelism is a follow-up — Clarified — user confirmed (auto-resolve, recommended option) | S:95 R:90 A:80 D:75 |
| 21 | Certain | `fab-operator.md` line targets (≈480, choreography ≈180) are targets, not acceptance gates; today's file is 905 lines, not the plan's ≈780 | Measured at intake; the plan's own heading says "targets, not measurements" | S:90 R:95 A:100 D:95 |

21 assumptions (17 certain, 4 confident, 0 tentative, 0 unresolved).
