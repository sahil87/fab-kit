# Intake: Operator `pane` kind replaces `fab-change` — join on pane ID plus pid fingerprint, change as an observed field; plus three agreed skill slims

**Change**: 260911-1159-operator-pane-kind-pid-fingerprint
**Created**: 2026-09-11

## Origin

Conversational — synthesized from a `/fab-discuss` session on 2026-09-11 and dispatched promptless via `/fab-proceed` (create-new). Every repo fact below was verified during that session and re-verified at intake (file/line references are to the tree at commit `c3c6a5d8`, release v2.25.1).

**The incident.** The running operator refused to track pane `%222` — a live agent in worktree `tireless-perch` — because no active fab change existed there. Its words: *"doesn't fit the fab-change kind (its pane probe joins on a change's completion predicate)"*. The user wants the operator to track **any** agent session in a pane and watch idle/waiting/stuck, whether or not a fab change exists yet.

**Root cause (verified).** `diffPaneItems` in `src/go/fab/cmd/fab/operator_tick_start.go` (line 506) uses the change id as the pane item's identity:

```go
if row.changeID != it.ID {
    out.Deltas = append(out.Deltas, tickDelta{Kind: "pane_mismatch", ID: it.ID, Pane: paneID, Found: toNullable(row.changeID)})
    continue
}
```

For a `fab-change` item the item id IS the change id, so a pane with no change mismatches every tick. This contradicts `fab-operator.md` §1 ("the pane ID is the join key … `session` is a display/context dimension") — session and change are context, the pane is the identity.

**Same defect from the spawn side** — backlog `[lm49]` (`fab/backlog.md` line 47, 2026-09-09): raw-text spawns run `/fab-new "<text>"` before any change id exists, yet §6 spawn step 8 says `fab operator track add <id> --kind fab-change --pane …` unconditionally. No id exists for a raw-text spawn, so the tracking path is unspecified for the first minutes of every raw spawn. This change closes `lm49`.

**`pane_mismatch` conflates two events today**: a recycled `%N` pane id after a tmux server restart (the case the delta was designed for — see the comment at `operator_tick_start.go:502`), and a change appearing or switching in a pane that is legitimately watched.

**Decisions reached in the discussion** (all recorded as Certain rows in `## Assumptions`):

- Rename the kind `fab-change` → `pane`; remove `fab-change` at `track add` outright, **no alias** (precedent: 4a8m removed `enroll`/`watch`/`autopilot`/`note` verbs with no aliases).
- Join on `scope.pane` **plus** a `scope.pane_pid` fingerprint (the pane's shell pid). `pane_start_time` was rejected — empty on tmux 3.7c.
- Per-tick pids come from **one** batched `tmux list-panes -a -F '#{pane_id} #{pane_pid}'` call. `rk mux panes --json` does **not** carry a pid (verified live: rows carry `session, session_id, window_index, window_id, window_name, window_active, pane, pane_index, pane_active, command, cwd, agent_state, agent_state_duration, has_agent`).
- The change becomes an **observed** field `scope.change`, maintained by the baseline writer like `scope.stage`; a change appearing/switching is a consumed-on-read `changed` delta, never `pane_mismatch`.
- Folding `pending` into `task` is **out of scope** (user decision, revisit later).
- Three skill slims agreed (§5 idle auto-default, §6 spawn step 2, §4 Clock history bullets) — and **only** those three; §6 Queues / merge modes / Ordered Merge / Auto-Merge Choreography stay inline and untouched (user confirmed all three merge modes are widely used and the choreography must survive compaction inline).
- Change type pinned to `feat`.

Interaction mode: promptless dispatch — no questions were asked; would-be questions are recorded as deferred rows in `## Assumptions` and listed under `## Open Questions`.

## Why

**Problem.** The operator's core tracked kind is named and joined after a *pipeline change*, but what the operator actually watches is an *agent in a pane*. Two consequences follow directly:

1. **The LLM refuses legitimate work.** The kind name told the running operator that a pane without a change "does not fit", so it declined to track a live agent the user pointed at. The user's request ("track %222") has exactly one correct verb and zero questions; the kind's name and its join rule made the skill hallucinate a gate.
2. **The binary emits a false alarm every tick.** `row.changeID != it.ID` fires `pane_mismatch` for any pane whose resolved change differs from the item id — including a pane with no change yet (raw-text spawn before `/fab-new` completes), a pane whose change was just created, and a pane whose agent switched changes. The level-triggered delta re-emits until `track rm`, so the operator is told to *remove* an item it should be watching. Meanwhile the delta's real purpose — detecting a **recycled** `%N` after a tmux server restart — is served only by accident (the recycled pane happens to resolve to a different change).

**Consequence of not fixing.** Every raw-text spawn (`/fab-new "<text>"`) is untracked or mis-tracked for its first minutes (`lm49`); any agent session that is not a fab change (a `/fab-discuss` session, an ad-hoc agent, a worktree the user opened by hand) cannot be watched for idle/waiting/stuck at all; and the operator keeps a confusing dual meaning for `pane_mismatch`.

**Why this approach.** The §1 principle already says the pane id is the join key. Making the join *honest* — pane id plus a pid fingerprint that only a recycled pane can fail — lets `pane_mismatch` mean exactly one thing (recycled pane), and lets the change be what it is: an observed attribute of the pane, diffed and reported like the stage. `pane_pid` is already the recycle discriminator in `fab dispatch` (the `PaneWorkerAlive` identity check in `internal/dispatch`; `pane.GetPanePID` at `src/go/fab/internal/pane/pane.go:404`), so the operator adopts an existing, proven fingerprint rather than inventing one. A batched `tmux list-panes -a` keeps the per-tick cost at one subprocess regardless of fleet size, and works on rk-less servers because it is plain tmux.

**Alternatives rejected** (recorded here so apply does not re-open them):

| Alternative | Why rejected |
|---|---|
| Keep the `fab-change` name and only relax the join | The kind *name* itself drove the LLM to refuse tracking change-less panes; a rename is the fix for the skill-side half of the bug |
| Name the kind `agent` | Collides with the existing `agent` probe mode (linear/slack items) |
| `pane_start_time` as the recycle fingerprint | Empty on tmux 3.7c; `pane_pid` is available via the existing helper and is what dispatch already uses |
| One `tmux display-message -t <pane> -p '#{pane_pid}'` per tracked item per tick | N subprocesses per tick; one batched `list-panes -a` is O(1) |
| Fold `pending` into `task` | Deferred by the user — out of scope, revisit later |
| Move queue/merge choreography into an on-demand helper | Rejected by the user — modes widely used; must survive compaction inline |
| Any run-kit change | None required — the pid comes from tmux directly. (run-kit's separate envelope fix is backlog `[peui]` in run-kit and unrelated) |

## What Changes

Five surfaces: Go (`fab operator`), a migration file, the operator skill, the CLI reference partial, and memory (hydrate). Constitution Additional Constraints: a `fab` CLI change MUST update `src/kit/skills/_cli-fab-operator.md` and ship tests. Canonical skill source is `src/kit/skills/`; `.agents/skills/` and `.claude/skills/` are deployed copies and are never edited (Constitution V). Deployed kit content MUST NOT cite `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*` paths (Principle V, 1.8.0 — the Go guard test fails on it). Run `gofmt` on every touched `.go` file before ship (a prior change failed CI on unformatted worker-written Go).

### 1. Go — kind rename `fab-change` → `pane`

Files: `src/go/fab/cmd/fab/operator_track_types.go`, `operator_track.go`, `operator_tick_start.go`, `operator_migrate.go`.

**1a. Constants and tables** (`operator_track_types.go` lines 46, 69–70, 80):

```go
const (
    kindPane     = "pane"      // was kindFabChange = "fab-change"
    kindGitHubPR = "github-pr"
    …
)

var trackKinds = map[string]trackKind{
    kindPane:     {defaultProbe: probePane},
    …
}

const trackKindNames = "pane, github-pr, linear, slack, shell, task, note"
```

Rename every `kindFabChange` use site (verified set: `operator_track.go` lines 89, 160, 205, 812, 857; `operator_tick_start.go` lines 907, 942; `operator_migrate.go` lines 243, 360; tests). `fab-change` is **removed outright** at `track add` — `--kind fab-change` exits non-zero through the existing unknown-kind path (`unknown --kind "fab-change" (valid: pane, github-pr, linear, slack, shell, task, note)`); no alias, no deprecation message. Same for `track list --kind fab-change`.

**1b. Sugar flags** (`operator_track.go` lines 42–67, 89–101, 244): the sugar list becomes `--pane --repo --session --branch --stage --agent --stop-stage --spawned-by --change`; the error text becomes `--<flag> applies only to kind pane`; the `--kind` help string lists `pane | github-pr | …`; the `--mode` help says "merge mode for a chained pane item". The pinned scope keys for `--kind pane` grow from nine to **eleven**, all null until set:

```
pane, pane_pid, change, repo, session, branch, stage, agent, stop_stage, spawned_by, merge_mode
```

`--change <id>` writes `scope.change` (a non-empty string; existence of the change is **not** validated — the change may not exist yet). `fabChangeScope()` in `operator_migrate.go:205` becomes `paneScope()` seeding the eleven keys.

**1c. `track list`, `state`, `items:` rows, `tickKindOrder`** render `kind: pane`; `pane` keeps ordering slot 0 (`operator_tick_start.go:942`). `tickItemState`'s `pending` check (`operator_tick_start.go:907`) and `track list`'s equivalents (`operator_track.go:812, 857`) test `it.Kind == kindPane`.

### 2. Go — pid fingerprint at `track add` / `update`

**2a. Recording.** Whenever a verb sets or changes `scope.pane` to a non-empty value — `track add --pane %N`, `track add --scope '{"pane":"%N"}'`, `track update --scope '{"pane":"%N"}'` — the same mutation records the pane's shell pid in `scope.pane_pid` (integer). Lookup uses the existing helper `pane.GetPanePID(paneID, server)` (`src/go/fab/internal/pane/pane.go:404`) with the default server (the operator runs inside its own tmux; `$TMUX` derivation addresses the socket). On **any** lookup failure (tmux unqueryable, pane absent, unparseable pid) store `null` and proceed — a null fingerprint means "join on pane id alone", exactly today's behavior. Clearing `scope.pane` (explicitly-empty value) also nulls `pane_pid`.

```yaml
scope:
  pane: "%222"
  pane_pid: 48213      # shell pid recorded when pane was set; null ⇒ pane-id-only join
  change: null         # observed; see §3
  repo: /home/user/code/fab-kit
  session: work
  …
```

**2b. Per-tick pid snapshot.** `tick-start --diff` fetches pids for the whole server with **one** batched call per tick, only when at least one tracked pane item has a non-null `scope.pane` (the existing `anyPaneItems` gate):

```sh
tmux list-panes -a -F '#{pane_id} #{pane_pid}'
```

Parsed into `map[paneID]pid`. The helper lives in `internal/pane` beside `GetPanePID` (e.g. `ListPanePIDs(server string) (map[string]int, error)`), argv-only, no shell string, bounded by the same short timeout posture the pane primitives use. A failed batch call yields an empty map and is **not** an error — every fingerprint comparison then degrades to "unreadable ⇒ not a mismatch" (the `PaneWorkerAlive` rule dispatch already applies: an unreadable pid is not a mismatch).

**2c. New `pane_mismatch` rule** (`diffPaneItems`, replacing the `row.changeID != it.ID` check):

```go
// pane_mismatch: level-triggered — the recycled-pane case ONLY. tmux
// recycles %N across server restarts while the socket-keyed state file
// survives. A recorded fingerprint that differs from the pane's current
// shell pid proves the pane is not the one we tracked. A null recorded
// fingerprint, or an unreadable current pid, joins on the pane id alone.
if rec, ok := scopeInt(it.Scope, "pane_pid"); ok && rec != 0 {
    if cur, found := pids[paneID]; found && cur != rec {
        out.Deltas = append(out.Deltas, tickDelta{
            Kind: "pane_mismatch", ID: it.ID, Pane: paneID,
            Found: toNullable(row.changeID),   // observed change id or null — unchanged field
        })
        continue
    }
}
```

Evaluation order per item stays `pane_death` → `pane_mismatch` → `agent_exited` → clean join. A mismatched pane still gets no baseline write, no stage/change diff, and no `candidates:` row. The observed change id **never** participates in the mismatch decision.

### 3. Go — `scope.change` as an observed field

**3a. Baseline.** On a clean join the baseline writer sets `scope.change` ← the snapshot row's `changeID` (`paneRow.changeID`, `pane_map.go:75`) when resolved (non-empty), exactly as it sets `scope.stage`. An unresolved snapshot change (empty — no `.fab-status.yaml` in the pane's cwd) fabricates no delta and leaves the baseline alone (mirrors the em-dash rule for stage). Consequence: `scope.change` is sticky — once observed it is not nulled if the pointer later dangles.

**3b. `changed` delta.** When the observed change differs from the baseline (null → id, or id → other id), emit a consumed-on-read `changed` delta and consume it with the same-write baseline update:

```yaml
- kind: changed
  id: tireless-perch
  fields: { change: { from: null, to: "4a8m" } }
```

The existing `changed` contract holds: the operator reports the field delta and runs `then` only if it names a changed reaction. Same tick, a `stage_advance`/`review_fail` may also emit for the stage baseline — they are independent.

**3c. `branch_map` write at first observation.** When a change is first observed for an item (baseline null → id) and `branch_map[<change>]` is absent, the same mutation writes `branch_map[<change>] = {branch, repo}` with `repo` = `scope.repo` and `branch` resolved from the pane's cwd. The snapshot row does **not** carry a branch or cwd today (verified: `paneRow` has `worktree` = basename + `/`, `repo`, `changeID`, no `cwd`/`branch`), so apply carries the raw enumeration `cwd` onto `paneRow` snapshot-internally (like `command`/`hasAgent` — never rendered) and runs `git -C <cwd> branch --show-current` once, at that tick only. An empty result (detached HEAD) or git failure skips the write; the next tick where the change is still observed and the entry is still absent retries. `track add` with `--branch`+`--repo` keeps writing `branch_map` in the add mutation, keyed by `--change` when given, else by the item id (today's behavior — for known-change spawns id == change).

**3d. Item id convention** (skill prose, binary-agnostic): the change id when known at spawn; the worktree name for raw-text spawns (known at spawn step 3). The id is **not** renamed when the change later appears — dependents reference item ids, and `branch_map` is keyed by change, which is why no rename is needed.

### 4. Go — done predicate

With `done_when` null on a `pane` item:

| `scope.change` | Done when |
|---|---|
| non-null | the existing built-in `tickCompleted(stopStage, stage, displayState)` — `review-pr` done/skipped, or at/past `scope.stop_stage` (`operator_tick_start.go:422`) |
| null | **never on its own** — the item leaves the list only via `track rm` or a dependent's/its own `then` |

`done_at` persistence is unchanged. `--stop-stage` on a change-less item is accepted and inert until a change is observed.

### 5. Go — `items:` pane rows, `pending`, unchanged surfaces

- **Pane rows** gain a present-keyed `change` field (null when none) after `pane`: `id, kind, state, pane, change, repo, session, stage, display_state, agent_state, idle_duration, pr_url, checked_at: null, next`. `stage`, `display_state`, `pr_url` are null when no change is observed (`putPaneRowFields`, `operator_tick_start.go`). Unjoined rows carry baseline identity incl. `change` from scope.
- **`pending`** (pane item with null `scope.pane` and deps satisfied → spawn this tick) stays exactly as is — only the kind constant changes.
- **`candidates:` rows gain `state_duration`** (clarified #21): rk's `agent_state_duration` verbatim for both `waiting` and `idle` rows (`null` when rk reports none); `idle_duration` keeps its idle-only meaning and is unchanged. This is the field §5's 30m auto-default keys on (8c). Everything else about candidates — the waiting-first ordering, the exclusion of active/unknown/mismatched/exited panes — is unchanged.
- **Unchanged**: idle/stuck detection, `agent_exited`, `pane_death`, `fleet_summary` — they ride `agent_state`/`idle_duration`/`has_agent`, not the change. The quiet/full decision and `last_full_at` are untouched.

### 6. Go — legacy conversion (`operator_migrate.go`) and the migration file

**6a. Conversion rule.** Extend the existing Legacy conversion (which today fires only when `tracked` is absent) with a second, idempotent pass that fires on **any** verb's read-modify-write when a `tracked`-present file holds an item with `kind: fab-change`: rewrite `kind: pane`, seed `scope.change` = the item's id (a `fab-change` item's id was the change id by construction — without this, converted items would never complete), seed `scope.pane_pid: null` (no live fingerprint ⇒ pane-id-only join, today's behavior). Same atomic write, same tolerant-read/typed-write contract; the read verbs `state` and `track list` convert-and-save then read, as today. `convertMonitoredEntry` and `convertAutopilotQueue` (the ≤2.24 path) emit `kind: pane` with `scope.change` seeded directly, so a ≤2.24 file converts in one pass.

**6b. Migration file** `src/kit/migrations/2.25.1-to-2.26.0.md`, in the style of `2.24.9-to-2.25.0.md`: Summary (the kind rename, `fab-change` removed at `track add` with no alias, `--change` sugar, `pane_pid`/`change` scope keys, the binary-owned first-touch conversion — **this migration performs no file edits**), Pre-check (`fab operator track list` exits 0), Changes (None), Verification (`fab operator state` shows `kind: pane` and no `kind: fab-change`; idempotent second run). FROM the released `2.25.1` TO `2.26.0` (a removed kind name at `track add` plus a new sugar flag ⇒ minor, per the catalog's feature-migration convention).

### 7. Tests (`src/go/fab/cmd/fab/operator_*_test.go`)

Using the existing seams (`tickSnapshotRows` stub via `stubSnapshot`, `seedDiffState`, `snapRow*` builders in `operator_tick_diff_test.go`; `trackAddArgs` in `operator_track_test.go`) plus a new injectable seam for the batched pid map (package-level var, the `tickSnapshotRows`/`rkPanesRunner` precedent):

- fingerprint **match** ⇒ clean join; **mismatch** ⇒ `pane_mismatch` with `found:` = observed change or null, no baseline write, no candidate; **null recorded pid** ⇒ join regardless of current pid; **unreadable current pid** ⇒ join.
- change **appears** (baseline null, snapshot `4a8m`) ⇒ `changed` with `fields.change {from: null, to: 4a8m}`, baseline updated, `branch_map[4a8m]` written from `scope.repo` + the stubbed branch; change **switches** ⇒ `changed` from/to; unresolved snapshot change ⇒ no delta, baseline untouched.
- done predicate: change non-null at `review-pr` done ⇒ `done` + `done_at`; change null at any stage ⇒ never done.
- `items:` pane row carries `change` (null and non-null), null `stage`/`display_state`/`pr_url` when no change.
- `candidates:` rows carry `state_duration` for both a `waiting` and an `idle` pane (from the snapshot's `agent_state_duration`), `null` when absent; `idle_duration` still null for `waiting`.
- legacy: `tracked`-present file with `kind: fab-change` ⇒ `kind: pane`, `scope.change` = id, `pane_pid: null`, idempotent; ≤2.24 `monitored`/`autopilot` conversion emits `kind: pane` with `scope.change` seeded.
- `track add --kind pane` sugar incl. `--change`; `--pane` records `pane_pid` (stub `GetPanePID` seam) and null on failure; `track update --scope '{"pane":…}'` records it too; `--kind fab-change` ⇒ unknown-kind error; `--pane` on kind `task` ⇒ `--pane applies only to kind pane`.
- rename every existing `fab-change`/`kindFabChange` test reference (13 in `operator_track_test.go`, 4 in `operator_tick_diff_test.go`, 5 in `operator_migrate_test.go`, 6 in `operator_clock_test.go`, 1 in `operator_track_types_test.go`).

Constitution VII: tests conform to this spec; no implementation edits to satisfy fixtures.

### 8. Skill — `src/kit/skills/fab-operator.md`

Edit **only** the sites below; §6 Queues, Queue Completion Summary, Ordered Merge, Auto-Merge Choreography are not restructured or moved (their `fab-change` spellings are renamed in place by the sweep, nothing else).

**8a. §4 Tracked Items.**
- Schema block: `kind: pane`, comment `# pane | github-pr | …`; `id` comment "pane items: the change id when known, else the worktree name; other kinds: a slug unique in the list"; `done_when: null # null on pane = the built-in predicate when scope.change is set; never done alone when null`; `scope.pane_pid: 48213 # shell pid fingerprint recorded when pane was set; null ⇒ pane-id-only join`; `scope.change: r3m7 # observed (baseline for change diffs); null until a change appears`; `branch_map` comment "change id → { branch, repo }; written by `track add` (branch+repo) or at the tick a change is first observed".
- Kinds table row: `pane` — probe `pane`: the binary snapshots the pane every tick and joins on `scope.pane` plus the `pane_pid` fingerprint — done when: built-in when `scope.change` is set (`review-pr` done/skipped, or at/past `scope.stop_stage`); never on its own when null — notes: "The operator's core kind (§6): **any agent session in a pane**; change and stage are observed, not required. With branch+repo set, `track add` also writes `branch_map`".
- Lifecycle: `pending` = "pane item with `scope.pane` null and deps satisfied"; `live` = "pane item whose pane is present **and** whose fingerprint matches (or is null)"; `done` wording "or the pane built-in fired".
- Tick Behavior step 2: `pane_mismatch` = "recycled pane only (fingerprint differs)"; `changed` bullet gains "— including `change` appearing or switching on a pane item (report; a raw-text spawn's change shows up this way)"; step 2's separate line "`pending` pane items…"; step 4 Ack "for pane items also clear the window…".
- Status Frame: Kind cell `pane · <repo basename>`; State cell for pane items with no change: `🟢 active · no change`, `🟡 idle 12m · no change`; with a change: the stage text, prefixed by the change id when it differs from the item id (`🟢 4a8m · apply`); example table rows use `pane · foo`; Ordering row lists `pane` first.

**8b. §4 The Clock — slim (c).** Delete the four-bullet block "**How the union predicate covers the retired loop behaviors**" (lines 185–190: `wake_on` ≻ tightened cadence; mute/unmute ≻ stop-when-empty; derived schedule ≻ fixed heartbeat; `target: role=operator` + `if_absent` ≻ session-bound liveness). Keep the entry-shape YAML and the **Ownership** paragraph verbatim. The rationale already lives in memory (present truth), so no deployed content needs to carry it.

**8c. §5 Idle Auto-Default on Strategic Escalations — slim (a).** Replace the three bullets (Timer / Reset / Answer-scope) with one rule derived from the tick's candidate row, keeping the hardcoded 30m, the answer-selection rule, and the hard exclusions:

> A left-open Strategic prompt (§ Logging "left open") gets the auto-default on the first tick whose `candidates:` row for that pane shows a `state_duration` of **30 minutes or more** (hardcoded — no setting, no state-file field). The LLM runs no timers: run-kit's agent-state epoch already resets on any activity in the pane, so the row's duration is the idle clock. Answer: the visibly stated default (`(default: 2)`, `Press enter for 2`, `[2]`), else `1`. Hard-excluded regardless of duration: auto-picked Strategic prompts and rule-6 cannot-determine escalations.

<!-- clarified: 2026-09-11 — user accepted: `candidates:` rows gain `state_duration` (rk `agent_state_duration` verbatim for waiting AND idle); `idle_duration` stays idle-only; §5 keys on `state_duration ≥ 30m`. See Assumptions #21. -->

Mirror the trim in §8 Settings' last sentence only if its wording references the timer (it references only the hardcoded 30m — keep).

**8d. §6 "The fab-change Kind" → "The pane Kind and Spawn Rules".** Rewrite the section: the `pane` item is the operator's core kind — **any agent session in a tmux pane**; a fab change, when present, is observed (`scope.change`/`stage`) and drives the built-in completion, but is never required to track. **Pipeline-first** and **spawn in a worktree** bind **spawns**, not tracking. Tracking an existing pane the user points at needs no gate and no kind question — "track %222" maps to exactly one verb, zero questions:

```sh
fab operator track add <slug> --kind pane --pane %222 --repo <repo> --session <session>
# <slug> = the change id when the pane already has one (fab pane map --all-sessions resolves it),
#          else the pane's window / worktree name
```

Keep the `--stop-stage hydrate` paragraph (parked `/fab-ff` runs).

**8e. §6 Spawning an Agent step 2 — slim (b).** Compress the Exclusion / Majority / Default-and-announce / Genuinely-torn sub-blocks to one short rule:

> **Target session** — candidates are `rk mux sessions --json` rows with `role: "user"` minus the operator's own session; pick the candidate holding the most panes whose `cwd` is under the target repo (from the tick's `rk mux panes --json` snapshot, or `fab pane map --all-sessions` on demand). Tie → the §8 "Spawn target session" setting; still torn → ask once when attended, notify via §5 when unattended. Announce the choice and auto-set §8. Never trust a persisted `scope.session` alone; the ambient session is never an implicit target.

Keep step 7's `=<session>` pin and never-ambient rule verbatim.

**8f. §6 spawn step 8.** Id = the change id when known, else the worktree name from step 3; raw-text spawns are tracked at spawn and their change appears later as a `changed` delta (the `lm49` fix). New form:

```sh
# new item (raw-text or fresh known-change spawn):
fab operator track add <id> --kind pane --pane <pane-id> --session <session> --repo <repo> [--change <change-id> --branch <branch>] [--stage <stage>] [--stop-stage <stage>] [--spawned-by <item-id>] [--depends-on <id,…>]
# an item that already exists as `pending` (a queued change spawned by tick step 2):
fab operator track update <id> --scope '{"pane":"<pane-id>","session":"<session>"}'
```

(`track add` refuses a duplicate id, so the pending case must go through `update`; the fingerprint is recorded on either path.) "Never ask whether to track" stays.

**8g. §6 Dependency Resolution.** One clause: `branch_map` is keyed by the dependency's change id — its `scope.change`, which equals its id for known-change spawns. No other edit.

**8h. §7 Conversational Map.** Add a row: "Track pane %222" · "watch the agent in tireless-perch" → `fab operator track add tireless-perch --kind pane --pane %222 --repo /home/x/code/fab-kit --session work`.

**8i. Sweep.** Every remaining `fab-change` spelling in `fab-operator.md` (27 occurrences — §2 wt Gate "fab-change spawn path", §6 Queues `--kind fab-change`, frame examples, Lifecycle, Ordering…) is renamed to `pane` or reworded ("a pane item with a change"). Grep `fab-change` repo-wide under `src/kit/` before finishing apply — the deployed set must read zero.

### 9. CLI reference — `src/kit/skills/_cli-fab-operator.md`

- **§ fab operator tick-start**: the `--diff` intro joins pane items "on `scope.pane`, fingerprinted by `scope.pane_pid`"; the deltas example shows `pane_mismatch` with the recycled-pane comment and a `changed` example on `change`; `items:` pane row example gains `change: r3m7` after `pane`; **Row field sets** adds `change`; the `candidates:` example and prose gain `state_duration` (rk `agent_state_duration` verbatim for `waiting` and `idle`; `idle_duration` stays idle-only) — the §5 30m auto-default's input; **Detection semantics** rewrites `pane_mismatch`: fires when the recorded `scope.pane_pid` is non-null AND the pane's current shell pid (one batched `tmux list-panes -a -F '#{pane_id} #{pane_pid}'` per tick) differs; a null recorded pid or an unreadable current pid joins on the pane id alone; the observed change never participates; `found:` = observed change id or null; **Baseline writer** adds `scope.change` ← snapshot change when resolved, the `changed` delta on appear/switch, and the first-observation `branch_map[<change>]` write (branch from `git -C <cwd> branch --show-current`, skipped when empty); **built-in completion** paragraph gains "applies only when `scope.change` is non-null; a change-less pane item is never done on its own"; **Item states** `pending` = "pane item with null `scope.pane` …".
- **§ fab operator track**: synopsis sugar list `[pane sugar: --pane --repo --session --branch --stage --agent --stop-stage --spawned-by --change]`; Kinds table `pane` row (probe `pane`, done_when null — built-in when `scope.change` set, never alone when null; notes: `--pane` records `scope.pane_pid` in the same mutation, null on lookup failure; `--change` pre-declares the expected change; branch+repo write `branch_map[<change or id>]`); `add` bullet: eleven pinned keys, `--<flag> applies only to kind pane`, "chained pane item"; `update` bullet: a `--scope` carrying `pane` re-records `pane_pid`; `rm` bullet "a pane item's `branch_map` entry is retained".
- **Legacy conversion** paragraph: add the `fab-change`→`pane` rule (any verb, `tracked`-present file, `scope.change` seeded from the id, `pane_pid: null`, idempotent) and name the 2.25.1 → 2.26.0 migration file as the walkthrough.
- **State path / contract** paragraph: note that `pane_pid` and `change` are new keys **inside** the owned `scope` mapping of `tracked` items — a typed write within the owned section, additive under the tolerant-read contract; the file path and slug rule (the run-kit cross-repo contract) are unchanged.
- **§ fab operator branch-map rm**: "Entries are written by `track add` (branch+repo) or at first change observation".

### 10. Backlog

At ship, mark `fab/backlog.md` line 47: `- [x] [lm49] … — SHIPPED via 260911-1159-operator-pane-kind-pid-fingerprint, PR #<n>` (the standard suffix).

### 11. Not in scope

- Folding `pending` into `task` (user decision — revisit later).
- §6 Queues, merge modes, Ordered Merge, Auto-Merge Choreography — no restructuring, no relocation.
- Any run-kit change; any change to `fab pane map` JSON (no `pane_pid` there); auto-discovery/auto-tracking of untracked agent panes; a `--change` sugar on `track update` (`--scope '{"change":…}'` suffices).
- `docs/specs/*` edits (human-curated, Constitution VI) — memory carries the present truth at hydrate.

## Affected Memory

- `runtime/operator`: (modify) tracked-item model — the `pane` kind (any agent session in a pane), the pane-id + `pane_pid` join and the recycled-pane-only `pane_mismatch`, `scope.change` as an observed field with the `changed` delta and first-observation `branch_map` write, the done predicate split on `scope.change`, item-id convention (change id else worktree name), the three skill slims (idle auto-default from the candidate row, spawn target-session short rule, Clock history block deleted), the `lm49` closure; Design Decisions: "Pane Is the Identity; Change Is Observed" (rejected: keep `fab-change` name / `agent` name / `pane_start_time`), "`pane_mismatch` Means Recycled Pane Only".
- `distribution/migrations`: (modify) catalog entry `2.25.1-to-2.26.0` — the binary-owned `fab-change`→`pane` conversion, no file edits, version-slot rationale.
- `runtime/pane-commands`: (modify) only if the batched pid helper is exported from `internal/pane` (planned: `ListPanePIDs`) — one line beside `GetPanePID` in the helper list.

## Impact

- **Go**: `src/go/fab/cmd/fab/operator_track_types.go`, `operator_track.go`, `operator_tick_start.go`, `operator_migrate.go`, `pane_map.go` (snapshot-internal `cwd` on `paneRow`), `src/go/fab/internal/pane/pane.go` (batched pid helper); tests in `operator_track_test.go`, `operator_tick_diff_test.go`, `operator_migrate_test.go`, `operator_clock_test.go`, `operator_track_types_test.go`, plus `internal/pane` tests for the parser. Scope the test run to `./src/go/fab/cmd/fab/ ./src/go/fab/internal/pane/` first.
- **Kit**: `src/kit/skills/fab-operator.md`, `src/kit/skills/_cli-fab-operator.md`, `src/kit/skills/_cli-external.md` (line 132 pointer "the fab-change kind" → "the pane kind"), new `src/kit/migrations/2.25.1-to-2.26.0.md`.
- **Memory**: the three files above at hydrate; the `runtime/index.md` description regenerates via `fab docs-index` if the operator description changes.
- **State-file compatibility**: additive keys inside `scope`; existing files convert on first touch; the path/slug cross-repo contract with run-kit is untouched. rk-less servers work (pid fetch is plain tmux); an unqueryable tmux degrades to pane-id-only join.
- **User-visible**: `--kind fab-change` stops working at `track add` (hard error, migration file documents it); the operator can now track any pane; `pane_mismatch` only on genuine recycles.
- **Lane**: full (Go + tests + migration + two skills + three memory files ≈ 15–20 tasks).

## Open Questions

None outstanding. The slim (a) premise question (candidate `idle_duration` is null for `waiting` panes, so the agreed 30m rule could not fire) was resolved 2026-09-11: `candidates:` rows gain `state_duration` (rk `agent_state_duration` verbatim for `waiting` and `idle`), `idle_duration` stays idle-only, and §5 keys on `state_duration ≥ 30m` — see Assumptions #21 and Clarifications.

## Clarifications

### Session 2026-09-11

**Q (Assumptions #21, Tentative):** Slim (a) keyed the 30m auto-default on the candidate row's `idle_duration`, which is null for `waiting` panes — a left-open Strategic prompt is `waiting`, so the rule as agreed could not fire. Add `state_duration` to `candidates:` rows and key §5 on it?
**A:** Accepted the recommended option — `candidates:` rows gain `state_duration` (rk `agent_state_duration` verbatim for `waiting` and `idle`); `idle_duration` stays idle-only; §5 keys on `state_duration ≥ 30m`. Rejected: populating `idle_duration` for waiting rows (breaks its documented meaning); keeping the LLM-side timer prose. Deviation noted: A/D re-scored on the user decision (R unchanged), per the qvek precedent.

### Session 2026-09-11 (bulk confirm)

User delegated the Confident rows to the agent's recommendation; all confirmed as written.

| # | Action | Detail |
|---|--------|--------|
| 12 | Confirmed | — |
| 13 | Confirmed | — |
| 14 | Confirmed | — |
| 15 | Confirmed | — |
| 16 | Confirmed | — |
| 17 | Confirmed | — |
| 18 | Confirmed | — |
| 19 | Confirmed | — |
| 20 | Confirmed | — |
| 28 | Confirmed | — |
| 29 | Confirmed | — |
| 32 | Confirmed | — |
| 33 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Kind `fab-change` renamed to `pane`; `fab-change` removed outright at `track add`/`track list --kind` (unknown-kind error), no alias, no deprecation text | Discussed — user decision; 4a8m precedent removed verbs with no aliases | S:95 R:70 A:90 D:95 |
| 2 | Certain | Recycle fingerprint is `scope.pane_pid` (the pane's shell pid, `pane.GetPanePID`); recorded when the pane is set; null on lookup failure ⇒ pane-id-only join | Discussed — `pane_start_time` rejected (empty on tmux 3.7c); dispatch already uses `pane_pid` as its recycle discriminator | S:95 R:80 A:90 D:90 |
| 3 | Certain | Per-tick pids via ONE batched `tmux list-panes -a -F '#{pane_id} #{pane_pid}'`, gated on `anyPaneItems`; `rk mux panes --json` carries no pid | Discussed — rk field set verified live; one subprocess per tick regardless of fleet size | S:95 R:85 A:90 D:90 |
| 4 | Certain | `pane_mismatch` fires only when recorded pid is non-null and differs from the observed pid; `found:` stays the observed change id or null; evaluation order unchanged | Discussed — the delta's single meaning is the recycled pane | S:95 R:80 A:90 D:95 |
| 5 | Certain | `scope.change` is an observed, baseline-maintained field like `scope.stage`; appear/switch emits consumed-on-read `changed` with `fields: {change: {from, to}}`, never `pane_mismatch` | Discussed — user decision | S:95 R:80 A:90 D:90 |
| 6 | Certain | `track add --change <id>` sugar pre-declares the expected change (sets the baseline; existence not validated); item id = change id when known, worktree name for raw-text spawns | Discussed — the change may not exist yet at spawn (lm49) | S:90 R:85 A:90 D:90 |
| 7 | Certain | `done_when` null: `scope.change` non-null ⇒ existing `tickCompleted`; null ⇒ never done on its own; `done_at` unchanged | Discussed — user decision | S:95 R:85 A:90 D:95 |
| 8 | Certain | `pending` (null pane, deps satisfied ⇒ spawn) unchanged; folding `pending` into `task` is OUT OF SCOPE | Discussed — explicit user deferral 2026-09-11 | S:95 R:90 A:95 D:95 |
| 9 | Certain | `candidates:`, idle/stuck, `agent_exited`, `pane_death`, `fleet_summary`, quiet/full decision unchanged | Discussed — they ride agent_state/has_agent, not the change | S:95 R:90 A:95 D:95 |
| 10 | Certain | `items:` pane rows gain present-keyed `change` (after `pane`); `stage`/`display_state`/`pr_url` null when no change | Discussed — user decision | S:90 R:90 A:90 D:90 |
| 11 | Certain | Legacy conversion extended: any verb, `tracked`-present file, `kind: fab-change` ⇒ `kind: pane`, same atomic write, idempotent; migration file `2.25.1-to-2.26.0.md` in the no-file-edits style of `2.24.9-to-2.25.0.md` | Discussed — binary-owned state file; context.md § Migrations requires the file | S:90 R:85 A:90 D:85 |
| 12 | Certain | Converted legacy `fab-change` items seed `scope.change` = item id and `scope.pane_pid` = null | Clarified — user confirmed | S:95 R:85 A:85 D:75 |
| 13 | Certain | `convertMonitoredEntry`/`convertAutopilotQueue` (≤2.24 path) emit `kind: pane` with `scope.change` seeded directly — one conversion pass, not fab-change-then-rewrite | Clarified — user confirmed | S:95 R:90 A:85 D:80 |
| 14 | Certain | Snapshot row carries no branch/cwd today (verified); apply carries the enumeration `cwd` onto `paneRow` snapshot-internally and resolves the branch via `git -C <cwd> branch --show-current` only at the tick a change is first observed and `branch_map[<change>]` is absent; empty/detached ⇒ skip and retry next tick | Clarified — user confirmed | S:95 R:85 A:85 D:75 |
| 15 | Certain | `branch_map` key at `track add` = `--change` when given else item id (today's behavior); at first observation = the observed change id; §6 Dependency Resolution gains one clause: the key is the dep's `scope.change` (equal to its id for known-change spawns) | Clarified — user confirmed | S:95 R:80 A:80 D:70 |
| 16 | Certain | `pane_pid` is recorded whenever `scope.pane` is set/changed by any verb (`add` sugar, `add`/`update --scope` with a `pane` key); clearing the pane nulls it; an unreadable current pid at tick is not a mismatch | Clarified — user confirmed | S:95 R:85 A:85 D:75 |
| 17 | Certain | §6 step 8 documents two forms: `track add` for a new item, `track update <id> --scope '{"pane":…,"session":…}'` for an item already `pending` (add refuses duplicate ids) | Clarified — user confirmed | S:95 R:90 A:80 D:70 |
| 18 | Confident | `scope.change` is sticky: an unresolved snapshot change fabricates no delta and leaves the baseline (mirrors the em-dash stage rule); disappearance is not reported | Clarified — user confirmed | S:95 R:85 A:75 D:60 |
| 19 | Certain | The item id is NOT renamed when a change appears on a worktree-named item | Clarified — user confirmed | S:95 R:80 A:80 D:65 |
| 20 | Confident | Frame State cell for a pane item with a change prefixes the change id when it differs from the item id (`🟢 4a8m · apply`); no change ⇒ `· no change` suffix | Clarified — user confirmed | S:95 R:90 A:70 D:55 |
| 21 | Confident | Slim (a) implementation: `candidates:` rows gain `state_duration` (rk `agent_state_duration` verbatim for waiting AND idle); `idle_duration` stays idle-only; §5 keys on `state_duration ≥ 30m` | Clarified — user confirmed the recommended option (add `state_duration` to `candidates:` rows; §5 keys on it). A/D re-scored on the user decision; R unchanged | S:95 R:35 A:90 D:85 |
| 22 | Certain | Slim (b): §6 step 2 compressed to the short candidate/majority/tie rule; `=<session>` pin and never-ambient rule in step 7 kept | Discussed — user agreed the exact rule | S:95 R:90 A:90 D:90 |
| 23 | Certain | Slim (c): delete the four-bullet "How the union predicate covers the retired loop behaviors" block; keep entry YAML + Ownership | Discussed — pure history already in memory | S:95 R:95 A:95 D:95 |
| 24 | Certain | Non-goals: §6 Queues/merge modes/Ordered Merge/Auto-Merge untouched (rename-in-place only); no run-kit change; no `fab pane map` JSON change; no auto-discovery; no `update --change` sugar | Discussed — user confirmed choreography stays inline; pid comes from tmux | S:95 R:90 A:95 D:95 |
| 25 | Certain | `change_type` = `feat`, pinned via `fab status set-change-type` | Description mandates the pin; refresh would infer `fix` from intake keywords | S:90 R:95 A:95 D:95 |
| 26 | Certain | Sweep class: every `fab-change` in `src/kit/skills/*.md` (fab-operator 27, _cli-fab-operator 10, _cli-external 1) renamed/reworded; historical records (`2.24.9-to-2.25.0.md`, log.md, findings) stay; memory at hydrate | Repo-wide grep at intake; code-quality.md Sibling Sweeps | S:85 R:90 A:90 D:85 |
| 27 | Certain | Constitution V (no fab-kit-only paths in deployed skills), CLI-ref update, `gofmt` before ship, tests conform to spec (VII) | Constitution + code-quality.md + code-review.md rules | S:95 R:95 A:95 D:95 |
| 28 | Certain | Batched pid fetch is an exported `internal/pane` helper beside `GetPanePID` (e.g. `ListPanePIDs(server) (map[string]int, error)`), injectable seam in the tick for tests; documented in `runtime/pane-commands.md` | Clarified — user confirmed | S:95 R:90 A:85 D:75 |
| 29 | Certain | `docs/specs/operator.md` / `config.md` untouched — memory carries present truth at hydrate | Clarified — user confirmed | S:95 R:90 A:75 D:65 |
| 30 | Certain | Full lane | Go + tests + migration + two skills + three memory files ≈ 15–20 tasks | S:85 R:95 A:90 D:90 |
| 31 | Certain | Backlog `lm49` marked `[x] … — SHIPPED via 260911-1159-operator-pane-kind-pid-fingerprint, PR #<n>` at ship | Description mandates; standard suffix verified in `fab/backlog.md` | S:95 R:95 A:95 D:95 |
| 32 | Certain | `pane` kind pins eleven scope keys (`pane, pane_pid, change, repo, session, branch, stage, agent, stop_stage, spawned_by, merge_mode`), null until set; `--change` validates only non-empty | Clarified — user confirmed | S:95 R:85 A:85 D:80 |
| 33 | Certain | Migration version slot FROM released `2.25.1` TO `2.26.0` | Clarified — user confirmed | S:95 R:80 A:85 D:80 |

33 assumptions (19 certain, 13 confident, 1 tentative, 0 unresolved).
