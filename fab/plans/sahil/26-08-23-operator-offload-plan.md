# Operator offload plan — narrative state + load reduction

> Plan doc — written 2026-08-23 from a /fab-discuss brainstorm over
> `26-08-23-operator-narrative-state.md`, then **revised the same day after a
> four-agent review** (Go feasibility: sound-with-changes · doctrine coherence:
> coherent-with-changes · failure red-team: needs-hardening · scope: adjust).
> All reviewer must-fixes are integrated below. Goal: give the operator an owned
> surface for cross-cutting narrative state, and push every mechanical piece of
> its tick into the `fab` binary, so the operator's queue stays free and a
> restarted operator re-orients from one file read. Visual (pre-review) version:
> `~/.agent/diagrams/operator-offload-plan.html`.
> Code anchors verified 2026-08-23 against `src/go/fab/cmd/fab/operator_*.go`.

## The problem, both halves

1. **No owned narrative surface** (the companion doc's complaint): phase-plan
   progress, peer-session agreements, corrections, and dependency waits that
   outlive `monitored` have no schema home, so the operator hand-wrote a
   `plan_queue:` key via `yq` — a never-hand-write doctrine violation, with no
   `resolved` flag, no pruning, no live-vs-stale distinction. After `/clear`,
   re-orientation means re-deriving hours of context.
2. **The tick itself is expensive**: every tick the LLM re-runs a full
   `fab pane map --all-sessions --json`, diffs stage/state per entry in context,
   does one `rk mux capture` round-trip per waiting/idle pane, and applies the
   turn-boundary/blank guards plus the indicator patterns itself. All of that is
   mechanical.

Both are the existing Full Mediation doctrine ("agents never compute or
hand-write what the binary can own") not yet applied to the note and the tick.

## Phase A — `fab operator note` (narrative state)

An owned verb family over a new `notes:` section in the server-keyed state file.
Envelope structured, body free prose — the split that already works for watches
(mechanical fields + LLM-evaluated `instructions`).

Schema (per note): `id` (binary-assigned `n<N>` from a **persisted top-level
`notes_seq` counter — ids are never reused after prune**; an old binary's
tolerant read carries the unknown key through), `kind`
(`dependency_wait | phase_plan | coordination | correction`), `text` (free
prose, **binary-enforced cap ~500 chars** — notes re-enter operator context on
every state read, so unbounded prose defeats the plan's own goal), `refs`
(optional change IDs / repos), `created_at`, `updated_at`, `resolved`,
`resolved_at`.

Verbs:

- `fab operator note add --kind <k> [--ref <r>]... <text>` — prints the id;
  unknown kind / over-cap text exits **1** (matches the operator-verb enum
  convention — invalid `--source`/`--stage` exit 1; exit 2 is the pane-family
  pane-missing code)
- `fab operator note resolve <id>` — idempotent; prunes **resolved** notes past
  a cap of 50 oldest-first (the `knownCap = 200` pattern,
  `operator_state.go:72`); **open notes never auto-expire** (notes are
  decisions, not a dedupe cache)
- `fab operator note update <id> <text>` — evolving phase-plan notes update in
  place rather than accreting near-duplicates
- `fab operator note list [--open|--all] [--json]` — `--open` default; renders
  each note's **age from `updated_at`** and flags stale ones (e.g. `⚠ 21d`,
  display-only); warns above 25 open notes (the "do I even need a note?" nudge
  made mechanical)
- `fab operator state` — **modified**: prints an OPEN NOTES header block
  (id · kind · age · first line) before the dump, **human output only** —
  absent from `--json`, and comment-prefixed in default mode so stdout stays
  parseable YAML for yq consumers. Default output **excludes resolved notes**
  (`--all` includes). Skeleton gains `notes: []` (`emptyOperatorState()`,
  `operator_state.go:170`; the owned-sections doc comment at
  `operator_state.go:20` gains the fifth section).

**Doctrine line (goes in the skill's new § Notes):**

| Content | Home |
|---|---|
| Passive narrative read on restart/orientation — phase progress + holds, peer scoping agreements, report-back promises, corrections to earlier conclusions, **merge-gate dependency waits** (checked by operator judgment per tick — no git/GitHub watch `source` exists today) | **note** |
| Standing concerns a watch source can actually express today (`linear`/`slack` queries with `instructions`) | **watch** — if a git-source watch ships later, merge-gate waits migrate to it as its own change |
| Anything still true for a different operator next month — process lessons | **not operator state** — route via an `idea` backlog entry → a fab change into docs/memory (the operator has no memory write path: 3-file context load, may run with no `fab/` project at all); deliberately **no `lesson` kind** — its absence is the guard against notes degrading into a reflexive scratchpad |

Skill § Notes also updates §2 Init step 2's restart-restore list ("monitored
set, autopilot queue, and branch_map") to include **notes** — surviving restart
is Phase A's whole point.

Answers to the companion doc's open questions: do **not** teach `depends_on` to
survive removal (fixes one case by complicating two schemas — a surviving wait
is a note per the table above); consumers are both the restarted operator and
skimming humans (via the `state` header); no auto-expiry for open notes,
bounded history for resolved.

## Phase B — tick offload into the binary

### B1 · `fab operator tick-start --diff`

The binary runs the pane-map snapshot internally (same `main` package —
`panemap.go`'s row collection is extracted into a `collectPaneRows`-style
helper behind a stubbable package-var seam, the `rkPanesRunner` /
`operatorStatePathOverride` precedent), compares against the `monitored`
baseline, and emits events plus a candidates list. **Two delivery classes —
this is the load-bearing design point** (red-team + doctrine reviews: a
baseline that advances at print time is at-most-once, and the lost side is the
action-demanding side; today's skill self-heals because every tick recomputes
from the full map):

- **Level-triggered, re-emitted every tick until the skill acts** —
  `completion` and `pane_death`. Both are stateless predicates over the current
  snapshot, needing no baseline: completion = stage terminal (or ==
  `stop_stage`) — note a stage-diff **cannot** detect completion at all, since
  a change completing at its terminal stage never changes its stage string,
  only `display_state` flips (`status.go` DisplayStage); pane_death = pane
  absent from the snapshot. The skill's `fab operator remove` is the natural
  ack — the entry disappears, the event stops. A crash between diff and action
  loses nothing; the queue can never stall on a consumed completion.
- **Consumed-on-read (baseline-diffed)** — `stage_advance` and `review_fail`.
  A lost one costs a missed report only; tolerable.
- **`pane_mismatch`** (new, level-triggered): each entry's change ID is
  cross-checked against the snapshot row's `change` field (`panemap.go:593`) —
  tmux recycles `%N` pane IDs across server restarts while the socket-keyed
  state file survives, so a recycled pane must never be diffed or swept as if
  it were the old agent. Treated like pane_death: report + remove, never a
  candidate. (Aligns with the pane-identity-keying contract, #612.)

Output also carries:

- `candidates:` — waiting-first then idle, each row with `pane`, `change`,
  `agent_state`, **`idle_duration`** (stuck detection needs it; without it the
  skill is back to the full map). Population is monitored agents only —
  discovery stays here, not in `pane questions`, because the sweep population
  is operator state the pane family shouldn't read, and the snapshot is
  already in hand. On rk-less servers all panes read `—` unknown → candidates
  empty → identical to today's §5 policy.
- `fleet:` — a compact full-fleet block (per entry: repo, session, stage,
  display_state, agent_state, idle_duration, pr_url) — **the status frame's
  data source**. Both the doctrine and scope reviews flagged this as the
  plan's biggest hole: the frame renders every tick from pane-map data
  (idle durations, PR URLs) that deltas can't carry, so without `fleet:` the
  skill either keeps fetching the full map (offload evaporates) or the frame
  dies. This also settles C3: with the binary owning frame data,
  `state --frame` becomes nearly free (see C3).

Baseline update happens in the same atomic mutation (`mutateOperatorState`,
`operator_state.go:127`); the stale "owned sections untouched here" comment in
`operator_tick_start.go:30` is corrected. A quiet fleet returns
`deltas: [] candidates: []` plus the fleet block — a cheap no-op tick. Flagless
`tick-start` is unchanged.

### B2 · `fab pane questions [--all-sessions] [--panes <id>...] [--json]`

One command sweeps the candidate panes (typically the `--panes` list from
B1's `candidates:`): capture last 20 lines (via a **new injectable capture
seam** — `pane.Capture` execs tmux directly today with no test seam; the
guard/indicator scanner is a pure function so the table-driven tests need no
seam at all), apply the mechanical guards (Claude turn-boundary `^\s*>\s*$`,
blank capture) and the question-indicator patterns, return only matching panes
with snippet + matched indicator + `agent_state` as read at capture time, plus
a `skipped:` list with reasons (`turn_boundary`, `blank_capture`,
`no_indicator`, `capture_failed`, `state_changed` — a pane no longer
waiting/idle at capture time is skipped, closing half the
candidates-to-capture race).

**Indicator class 4 is not mechanical** (three reviewers independently): the
skill's "Claude Code permission/tool approval prompts" class is prose, not a
regex. Binary-side it is covered by concrete proxies (the `[Y/n]`-family,
`Allow?/Approve?/Confirm?/Proceed?`, `Do you want to…`, numbered-option
patterns — indicator classes 2/3/5/7 overlap it almost entirely); novel prompt
shapes remain operator judgment via an on-demand `--panes` sweep or manual
capture. The skill text states this split.

**The LLM keeps answer-model rules 4–6 whole** — Routine/Strategic
classification, answer selection, escalation — and the §5 rewrite **explicitly
retains both delivery guards unchanged**: the §3 pre-send gate and
re-capture-before-send (fresh capture compared against the returned snippet;
any change aborts). `questions` output is detection input only, never a
license to send blind — the batch capture is older than today's just-in-time
capture, so the guard matters more, not less.

Placement note: this is the multi-tab analogue of today's N per-pane
`rk mux capture` calls, but it lives in **fab, not rk** — the indicator
patterns are fab-operator policy and the pane family already enumerates panes +
`@rk_agent_state` server-wide. rk stays the per-pane primitives tool. This is
a **deliberate carve-out from the cli-layering Part 7 fence** ("skill-facing
guidance rides rk's substrate twins — never the fab pane verbs"): `questions`
is a policy-bearing sweep, not raw peek. The hydrate re-scopes
`pane-commands.md`'s "never the fab pane verbs" to raw peek with `questions`
as the named exception, and updates `_cli-agents.md`'s § Peek/layering text.

Result: an idle-fleet tick = two commands (`tick-start --diff`,
`pane questions`), near-zero LLM context. The rewired tick **drops the
per-tick `fab operator state` read** — the fleet block and deltas carry what
the frame and monitoring need; the watch pass reads state on its own step as
today.

**Version-skew fallback (one line in §4):** if `tick-start --diff` or
`fab pane questions` errors as unknown flag/command (new skill against an old
installed binary — bottle lag is a recorded recurring lesson), fall back to
the flagless tick (full pane map + per-pane capture) for the session and
report the mismatch once.

## Phase C2 — auto-merge, re-specced (promoted into the queue)

The scope review promoted C2 (cheapest large win — it kills the longest
operator-busy stretches); the red-team showed the original one-liner was
dangerous. Re-specced, and **scoped to `cherry-pick-ladder`** (and trivially
`merge-auto`), where PRs target main; **stacked-prs keeps today's manual
merge-all** — its inter-merge choreography (retarget-verify, rebase --onto,
force-push) is operator-sequenced anyway, and an armed stacked PR can merge
into its dependency's *branch* (destroying the stack silently) or fire on
stale-green checks after GitHub's no-re-CI base retarget. Rules, all MUSTs in
the skill text:

1. **Sequential arming**: at most one armed PR per repo-sequence. Arm PR_n
   only after PR_{n-1}'s merge is **verified** (timeline event, not
   assumption). Never arm a PR whose base is another PR's branch.
2. **Arming-failure shapes**: draft PR → `gh pr ready` first (fab's `/git-pr`
   creates drafts, so this is every autopilot PR); "already clean" rejection
   (no required checks) → merge directly; auto-merge disabled on the repo →
   today's CI-wait choreography.
3. **Stall rule**: an armed PR unmerged after N ticks → check
   `mergeableState`; `CONFLICTING` → escalate (auto-merge fails silently —
   there is no event to observe).
4. **Persisted sequence**: merge-all state is conversational today and an
   armed PR **outlives the operator** (survives /clear, crash, abandonment) —
   so starting a merge sequence writes a `kind: coordination` note (sequence,
   position, armed PR), making **C2 depend on Phase A**; the note resolves at
   sequence end. The loop run-condition gains "merge sequence in progress"
   (at merge-all time the autopilot queue is exhausted and the monitored set
   typically empty — today the loop may not even be running to do the
   "verify on later ticks" work).
5. **Disarm on halt**: any halt/escalation runs `gh pr merge --disable-auto`
   on the remaining armed PRs — halt-dependents-only assumes unstarted merges
   stay unstarted, which armed auto-merge violates.

Honest sizing: the win is real for cherry-pick-ladder (foreground CI-watch →
passive tick check); it was always marginal for stacked-prs, hence the scope
cut.

## Phase C — remaining follow-ups

- **C1 · Event wake** *(run-kit backlog — file the `idea` entry with this spec
  attached)*: extend `rk mux await` (single-target today) to multi-target
  any-of. The backlog entry MUST carry the red-team's five requirements:
  (a) exclusion set / arm only against not-currently-waiting panes (a left-open
  Strategic prompt legitimately sits `waiting` for up to 30m — level-triggered
  re-arm would busy-loop); (b) poll-after-arm — one immediate state sweep after
  arming closes the edge-trigger gap end-to-end; (c) the operator's §2 Init
  re-arms the background await after `/clear` (resumed sessions lose background
  children — recorded lesson); (d) kill+re-arm on every enroll/remove;
  (e) min-inter-wake debounce against waiting↔active flap. Accepted trade,
  stated plainly: the fallback heartbeat relaxes to ~10m, so pane-death
  detection latency worsens from ≤3m to ~10m (interacts with autopilot's
  1-respawn policy).
- **C3 · Self-serve status**: `fab operator state --frame` renders the frame
  from the state file's fleet baseline (cheap once B1's `fleet:` fields are
  baselined). MUST print staleness from `last_tick_at` ("as of 6m ago") — at a
  10m heartbeat the frame is up to 10 minutes old on a surface whose point is
  at-a-glance trust.
- **C4 · Routine answers below the operator** *(deferred)*: revisit only if
  routine volume still dominates after B2 — classification is the operator's
  core value.

## Edge cases

- Legacy hand-written `plan_queue:` key **survives** — `loadOperatorState` is a
  tolerant whole-file read and unknown top-level keys round-trip (verified:
  `operator_tick_start.go` comment). One-time manual conversion to notes, then
  delete the key; no migration code.
- Crash/`/clear` between `--diff` and acting: level-triggered events
  (completion, pane_death, pane_mismatch) re-emit next tick — nothing lost;
  consumed-on-read events lose at most one report.
- `--diff` with empty monitored set → empty deltas/candidates + empty fleet,
  tick still increments (no-op tick is a first-class result).
- Unknown-state (`—`) panes excluded from `candidates:` by default (matches the
  §5 waiting+idle sweep policy); capture-based fallback stays an operator
  judgment call.
- Enrollment between diffs enters the baseline via `enroll`; the next `--diff`
  emits no synthetic event.
- Delta + question on the same pane, same tick: independent outputs; skill acts
  deltas-first (a completion removes the entry and skips its answer).
- `note resolve` idempotent; duplicate `note add` allowed (dedupe is judgment,
  aided by the list warning).
- Concurrency: same guarantee as every verb — atomic temp+rename
  (`saveOperatorState`); last-writer-wins acceptable under one operator per
  server.
- Server-scoping asymmetry (accepted): `fab pane questions` inherits the pane
  family's `--server` flag; `tick-start` keys off the current socket — an
  operator drives `--diff` only for its own server, which is the model.

## Tests

- `operator_note_test.go` (new): ids from persisted `notes_seq` — **never
  reused after prune**; timestamps; kind + text-cap validation (exit 1);
  resolve idempotency + unknown-id exit 1; resolved-cap prune; open notes never
  pruned; list shapes + staleness flag; `state` header human-mode-only (absent
  from `--json`, stdout stays parseable); skeleton `notes: []`; unknown
  top-level keys survive every verb.
- tick-start `--diff` (extend `operator_test.go` or new file — today's
  tick-start tests live in `operator_test.go:245ff`): all event kinds from
  seeded baseline + the named snapshot seam; **completion/pane_death/
  pane_mismatch re-emit until `remove`**; completion detected via
  display-state/terminal-stage predicate (stage-diff provably can't);
  baseline updated in same write; candidate ordering + `idle_duration`
  presence; `fleet:` block shape; empty monitored ⇒ empty outputs; flagless
  path byte-identical.
- `pane_questions_test.go` (new): both guards; table-driven indicator classes
  over the pure scanner function; bottom-most indicator wins; dead pane ⇒
  `capture_failed` skip; state-changed ⇒ `state_changed` skip; `--json` schema
  stable.

## File map

| File | Kind | Change |
|---|---|---|
| `src/go/fab/cmd/fab/operator_note.go` | new | verb family, follows `operator_watch.go` structure; `notes_seq` counter |
| `src/go/fab/cmd/fab/operator_state.go` | modify | skeleton `notes: []`; owned-sections comment; open-notes header (human mode only); resolved excluded by default |
| `src/go/fab/cmd/fab/operator_tick_start.go` | modify | `--diff`; corrected owned-sections comment |
| `src/go/fab/cmd/fab/panemap.go` | modify | extract `collectPaneRows` + snapshot seam var (same file, small) |
| `src/go/fab/cmd/fab/pane_questions.go` | new | sweep + guards + indicator patterns (pure scanner fn + capture seam) |
| `src/kit/skills/fab-operator.md` | skill | § Notes (verbs, doctrine table, §2 restore-list + Init re-arm additions); §4 tick rewired to deltas + fleet-sourced frame + skew fallback line; §5 detection via `pane questions` with guards explicitly retained; C2 merge rules |
| `src/kit/skills/_cli-fab.md` | skill | **constitution-mandated** signatures for all new/changed commands |
| `src/kit/skills/_cli-agents.md` | skill | § Peek / layering text updated for the `questions` carve-out |
| `docs/memory/runtime/operator.md` | hydrate | rewrite BOTH Design Constraints bullets ("hardcoded patterns", "pane-map only") and the two "operator's own detection policy" sentences; extend the Full Mediation decision's scope (first detection-policy offload); new decisions: level-triggered deltas, notes-vs-watches line, C2 sequential arming |
| `docs/memory/runtime/pane-commands.md` | hydrate | re-scope the Part 7 "never the fab pane verbs" fence to raw peek, `questions` as named exception |

## Critical couplings & sequencing

- **`--diff` is the authoritative baseline writer.** The skill must stop doing
  its own last-known bookkeeping via `fab operator update` on the diff path
  (today: tick step 6, `fab-operator.md:241`), or the next diff under-reports.
  The flag and the skill §4 rewrite ship in the **same change** — coupling also
  gives the flag a consumer (acceptance) and satisfies the constitution's
  simultaneous `_cli-fab.md` rule. (`fab operator update` stays for
  non-baseline field edits.)
- Everything is additive/back-compatible: old state files load fine, flagless
  `tick-start` and per-pane capture keep working; the one skew direction that
  bites (new skill + old binary) is covered by the §4 fallback line.
- `fab pane questions` spawns one tmux capture per candidate — same process
  count as today's per-pane rk calls, in one invocation; fine at 5–15 panes.

Ship as **one same-repo ordered queue** (all four changes touch
`fab-operator.md` + `_cli-fab.md`; A and B1 both touch `operator_state.go` —
cherry-pick-ladder, not parallel spawns):

1. **Change 1 (Phase A)** — `operator_note.go` + state header + skill § Notes +
   `_cli-fab.md`. Self-contained; immediately kills the yq hand-writing.
2. **Change 2 (C2)** — auto-merge choreography per the re-spec above
   (skill-only + depends on A's `coordination` note). Cheapest large win.
3. **Change 3 (Phase B1)** — `tick-start --diff` (+ `fleet:` frame source) +
   skill §4 rewrite (coupled, see above). Biggest per-tick win.
4. **Change 4 (Phase B2)** — `fab pane questions` + skill §5 rewire.
5. **Follow-ups** — C1 → `idea` entry in the run-kit repo carrying the 5-point
   spec; C3 rides on B1's fleet baseline as a small later change; C4 deferred.
