# Plan: Operator Generic Tracked Items — one tracked-item model, probe loop, derived schedule, one-table frame

**Change**: 260911-4a8m-operator-generic-tracked-items
**Intake**: `intake.md`

> Read `intake.md` in full before any task. Its `## What Changes` section (B1–B5) carries the exact schemas, verb signatures, tick document, frame table, and rules the requirements below reference by name; the requirements pin the contract, the intake carries the worked examples. Where the two disagree, this plan wins and the disagreement is a review finding.

## Requirements

### Operator State: Tracked Items

#### R1: One `tracked` list replaces the four owned sections
The operator state file (`$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`, path unchanged) SHALL carry one owned section `tracked:` — an ordered list of items with the fields `id`, `kind`, `probe`, `check_every`, `done_when`, `then`, `depends_on`, `scope`, `last`, `seen` (linear/slack only), `text` (note only), `checked_at`, `unchanged`, `failures`, `paused`, `added_at`, `updated_at` — exactly as the intake B1 YAML shows. The sections `monitored`, `watches`, `autopilot`, `notes`, `notes_seq` SHALL no longer be written or read except by the legacy conversion (R5). `branch_map`, `tick_count`, `last_tick_at` stay. Kinds are `fab-change | github-pr | linear | slack | shell | task | note`; probe modes are `pane | shell | agent | none`. The binary MUST fill each kind's defaults (intake B1 kinds table) at `track add` and MUST reject an unknown kind or a probe mode the kind does not allow.

- **GIVEN** an empty state file
- **WHEN** `fab operator track add pr-913 --kind github-pr --scope '{"repo":"/x","pr":913}'` runs
- **THEN** the file holds one item with `probe.mode: shell`, `probe.argv` = `[gh, pr, view, "913", --repo, <owner/repo derived or as given>, --json, "state,mergedAt,mergeable"]`, `probe.fields: [state, mergedAt, mergeable]`, `done_when: 'state == "MERGED"'`, `check_every: 5m`, `failures: 0`, `paused: false`, RFC3339 UTC `added_at`/`updated_at`, and no `monitored`/`watches`/`autopilot`/`notes` keys are created

- **GIVEN** a `fab-change` item added with `--pane %3 --repo /x --session work --branch b --stage apply`
- **WHEN** the file is read back
- **THEN** `scope` carries `pane`, `repo`, `session`, `branch`, `stage`, `agent`, `stop_stage`, `spawned_by`, `merge_mode` and `branch_map[<id>] = {branch: b, repo: /x}` was written in the same mutation

#### R2: `done_when` predicate grammar
`done_when` SHALL be parsed and evaluated by the binary with this grammar and nothing more: one or more clauses joined by ` and `; a clause is `<path> <op> <literal>` with `<path>` a field name or dotted path (leading `.` accepted and stripped), `<op>` ∈ {`==`, `!=`}, `<literal>` a JSON scalar (double-quoted string, number, `true`, `false`, `null`). A path absent from `last` compares as `null`. Malformed predicates MUST be rejected at `track add`/`update` with exit non-zero naming the offending clause. `or`, `<`/`>`, regex, functions MUST be rejected.

- **GIVEN** `last: {state: MERGED, checks: {conclusion: success}}`
- **WHEN** `state == "MERGED" and checks.conclusion == "success"` is evaluated
- **THEN** the result is true; `state != "MERGED"` is false; `missing == null` is true; `state == MERGED` (unquoted) is a parse error

#### R3: `check_every` rules
`check_every` SHALL be a Go duration with floor `1m` (below → exit non-zero), default `5m` for `shell`/`agent` items when omitted, and forced `null` for `pane`/`none` items.

- **GIVEN** `track add x --kind shell --argv true --fields a --check-every 30s`
- **WHEN** it runs
- **THEN** it exits non-zero with a one-line floor error and writes nothing

#### R4: `track` verbs
The binary SHALL expose `fab operator track add|update|observe|rm|list|clock` with the signatures in intake B1 (add/update/observe/rm/list) and B3 (`clock --every|--idle-every <dur> --for <dur> | --off`). The verbs `enroll`, `update` (monitored), `remove`, `watch *`, `autopilot *`, `note *` SHALL be removed from the binary with no aliases. `fab operator state`, `tick-start`, `time`, `branch-map rm` keep their signatures; `state` prints an `OPEN NOTES` header from `kind: note` items. Validation: duplicate id, unknown id, malformed `--done-when`, `--check-every` below floor, `shell` probe without `--fields`, `--depends-on` naming an unknown id, `clock` without `--for` (unless `--off`) ⇒ exit non-zero, one-line error, no write. `observe` stores the declared `fields` (or the whole object when `fields` is empty) into `last`, sets `checked_at`, resets `failures`, appends `--seen` ids (200 cap, oldest pruned), evaluates `done_when`; `observe --error <msg>` increments `failures` instead (3 ⇒ `paused: true`). `update --pause|--resume` toggles `paused` (`--resume` zeroes `failures`). `list` prints one line per item (`id · kind · state · checked-age · next`) or the items array with `--json`.

- **GIVEN** a `linear` item with `seen: [A]`
- **WHEN** `track observe linear-bugs --json '{"new":["B"]}' --seen B` runs
- **THEN** `last == {new: [B]}`, `seen == [A, B]`, `checked_at` is now, `failures == 0`

- **GIVEN** any `track` mutation
- **WHEN** the tracked predicate (R6) flips
- **THEN** exactly one `rk cron mute <id>` or `rk cron mute <id> --off` is issued, then the schedule reconcile (R12) runs — both fail-silent

#### R5: Legacy state-file conversion
On the first read-modify-write by any `fab operator` verb (including `tick-start`), a file with any of `monitored`/`watches`/`autopilot`/`notes` present and `tracked` absent SHALL be converted in the same atomic write per intake B1 Migration (monitored → `fab-change` items; watches → `linear`/`slack` agent items with `seen = known ∪ completed`, `then = instructions`, `check_every: 5m`; autopilot queue entries not in `completed` → pane-less `fab-change` items chained by `depends_on` with `scope.merge_mode = autopilot.mode`; open notes → `kind: note` items `n<N>`; resolved notes dropped; legacy keys deleted). When `autopilot.state == running` the verb MUST exit non-zero with the intake's refusal message and write nothing. A migration file `src/kit/migrations/2.24.9-to-2.25.0.md` SHALL document the conversion and refusal; its Changes section performs no file edit.

- **GIVEN** a v10 state file with two monitored entries, one watch, an exhausted autopilot block, and one open + one resolved note
- **WHEN** `fab operator state` runs
- **THEN** the file is rewritten with four `tracked` items (2 fab-change, 1 linear, 1 note, 0 from the exhausted queue) and no legacy keys; a second run changes nothing

- **GIVEN** a state file with `autopilot.state: running`
- **WHEN** any `fab operator` verb runs
- **THEN** it exits non-zero with `operator state file has a running autopilot queue — finish or stop it (fab operator autopilot stop on fab ≤2.24) before upgrading` and the file is byte-identical

#### R6: Tracked predicate and branch map
The clock's tracked predicate SHALL be "any item whose state is not `done`" (a list of only done items is untracked). `branch_map` keeps its semantics: written by `track add --kind fab-change`, retained by `track rm`, cleared only by `branch-map rm`.

- **GIVEN** one `github-pr` item whose `done_when` just became true
- **WHEN** `tick-start --diff` runs
- **THEN** the reconcile mutes the entry (the state is untracked) and the `done` delta still emits until `track rm`

### Operator Tick: Probe Loop

#### R7: Shell probe runner
`tick-start --diff` SHALL run every due `shell` probe (`paused: false`, `now - checked_at ≥ check_every`, or `checked_at` null): argv executed directly (never a shell string), 10 s timeout per probe, stderr discarded, stdout parsed as a JSON object (anything else is a probe error). Probes run sequentially under a 60 s per-tick budget; items not reached emit `probe_error` with `error: "skipped (tick budget)"` and `failures` untouched. Only `probe.fields` (top-level or dotted) are extracted, compared with `last`, and stored back. Success resets `failures` to 0; failure increments it; the third consecutive failure sets `paused: true`. `unchanged` is 0 on a field delta and +1 otherwise. The runner MUST sit behind an injectable seam (the `rkCronRunner` precedent) so tests need no live `gh`.

- **GIVEN** an item with `last: {state: OPEN}` whose probe now prints `{"state":"MERGED","mergedAt":"…","extra":1}`
- **WHEN** the tick runs
- **THEN** `last == {state: MERGED, mergedAt: …}` (no `extra`), `unchanged == 0`, a `changed` delta carries `fields: {state: {from: OPEN, to: MERGED}}`, and `done` emits because `done_when` is true

- **GIVEN** an item whose probe exits 1 on three consecutive ticks
- **WHEN** the third tick runs
- **THEN** `probe_error` carries `failures: 3, paused: true` and the item is not probed on the fourth tick

#### R8: Delta kinds and delivery classes
The tick document's `deltas:` SHALL carry `changed`, `done`, `stale`, `probe_error` for probed items and `pane_death`, `pane_mismatch`, `agent_exited`, `stage_advance`, `review_fail` for pane items — every delta keyed by `id` (the former `change` key), with the kind-specific fields in intake B2. `done`, `stale`, `probe_error`, `pane_death`, `pane_mismatch`, `agent_exited` are level-triggered (re-emitted until `track rm`, `track observe`, or `track update --resume`); `changed`, `stage_advance`, `review_fail` are consumed-on-read. `stale` fires for an `agent` item with `now - checked_at > 2 × check_every` (or `checked_at` null and `added_at` older than 2×). A `done` delta carries the item's `then` verbatim (null when none). The `fab-change` built-in completion predicate (review-pr done/skipped, or at/past `stop_stage`) emits `done`, replacing `completion`.

- **GIVEN** an `agent` item with `check_every: 5m` and `checked_at` 11 minutes ago
- **WHEN** the tick runs
- **THEN** `stale` emits with `age: 11m` and the item appears in `needs_check:`; after `track observe` the next tick emits no `stale`

#### R9: Tick document blocks
After the `tick:`/`now:` header the document SHALL emit, in order, `deltas:`, `candidates:` (unchanged shape, pane items only, `id` key), `needs_check:` (due `agent` items: `id`, `kind`, `age`, `instruction`), then `items:` (one row per item ordered kind → `scope.repo` → id, with the per-kind fields in intake B2 and a `state` and `next` column) **or**, on a quiet tick under `--quiet`, `fleet_summary:` with its five keys where `tracked` counts all not-done items and the four state counts cover pane items (`tracked ≥ waiting + idle + active + unknown`). A non-empty `deltas:` or `needs_check:`, or a tick count multiple of 10, forces the full document. An empty tracked list skips the pane snapshot and emits `[]` blocks (or the zero summary).

- **GIVEN** three pane items (active, idle, waiting) and two shell items, no deltas, tick 47, `--quiet`
- **WHEN** the tick runs
- **THEN** `fleet_summary: {tracked: 5, waiting: 1, idle: 1, active: 1, unknown: 0}` replaces `items:`

#### R10: Item state derivation
Each item's `state` SHALL be derived by the binary as `held` (any `depends_on` item not done — a `depends_on` id missing from the list emits `probe_error` with `error: "unknown dependency <id>"`), `pending` (`fab-change` with null `scope.pane` and deps satisfied), `live` (pane present and change-matched), `watching` (shell/agent, `done_when` false), `stale` (R8), `paused`, `done`. `next` is the item's `then`, or `spawn` for pending, or `held: <dep-id>` for held.

- **GIVEN** `ef56` (fab-change, pane null, `depends_on: [k8ds]`) and `k8ds` live
- **WHEN** the tick runs
- **THEN** `ef56` renders `state: held`, `next: "held: k8ds"`; once `k8ds` is done, `ef56` renders `state: pending`, `next: spawn`

#### R11: `has_agent` tri-state liveness
`agent_exited` SHALL use `has_agent` from `rk mux panes --json` (passed through `fab pane map`'s pane row): `false` ⇒ exited, `true` ⇒ alive, `null` ⇒ today's process-tree walk decides. The walk code stays; it is reached only for `null`.

- **GIVEN** a pane row with `has_agent: false` and a live-looking `agent_state`
- **WHEN** the tick runs
- **THEN** `agent_exited` emits without a process-tree walk; with `has_agent: null` and a shell foreground command the walk runs as today

### Operator Clock: Derived Schedule

#### R12: Schedule derived from the tracked set, applied via `rk cron edit` only on change
After every `track` mutation and at the end of `tick-start --diff`, the binary SHALL derive the schedule per intake B3's table (pane/none only ⇒ `--backoff --min 1m --max 30m --deliver immediate`; any shell/agent item ⇒ `--idle-every min(check_every) --deliver skip-if-busy`, or `--every min(check_every)` when the operator pane's `rk mux panes --json` row has null `agent_state`), compare it with the structured `schedule`/`deliver` fields of the resolved operator row from `rk cron list --json`, and issue exactly one `rk cron edit <id> …` only when they differ. Every rk call is `exec.LookPath`-gated, argv-only, 5 s timeout, fail-silent. A muted or leased entry is still edited. The `operatorCronRow` struct SHALL gain `schedule {kind, min, max, every}` and `deliver`.

- **GIVEN** the live row `schedule: {kind: backoff, min: 1m0s, max: 30m0s}, deliver: immediate` and a tracked set of one shell item at `2m` plus one at `5m`
- **WHEN** the reconcile runs on an operator pane with `agent_state` set
- **THEN** `rk cron edit <id> --idle-every 2m --deliver skip-if-busy` is issued once; a second reconcile with the row now reading `{kind: idle-every, every: 2m0s}, skip-if-busy` issues nothing

#### R13: Bounded cadence override
`fab operator track clock (--every|--idle-every) <dur> --for <dur>` SHALL write top-level `clock_override: {schedule: {kind, every}, deliver: skip-if-busy, until: <RFC3339>}`; the reconcile applies the override instead of the derived value until `until` passes, then reverts. `--for` is required (else exit non-zero); `track clock --off` deletes the override.

- **GIVEN** `clock_override` with `until` in the past
- **WHEN** the reconcile runs
- **THEN** the override is removed from the file and the derived schedule is applied

### Skill: `fab-operator.md` Rewrite

#### R14: §4 rewritten around items — four-step tick, Tracked Items section, notes as a kind
`src/kit/skills/fab-operator.md` §4 SHALL replace the seven-step Tick Behavior with the four steps in intake B4 (snapshot → act on deltas → answer waiting agents → ack), replace Operator State File / Monitored Set / Notes with one **Tracked Items** section (schema reference block, kinds table, lifecycle table, the never-hand-write rule), delete the four note kinds and the routing-doctrine table (keep the one-sentence "process lessons are not operator state → `idea`" rule), delete the **Idle Message** section, and re-point Post-Compaction Reload's step list to the four steps. The Clock section's mute/unmute prose SHALL name the `track` verbs and the derived schedule (R12/R13) and the `track clock --for` override.

- **GIVEN** the rewritten skill
- **WHEN** grepping it for `monitored:`, `watches:`, `autopilot:`, `notes_seq`, `Idle Message`, `phase_plan`, `routing doctrine`, `fleet:`
- **THEN** there are zero hits (the only permitted mention of the old sections is one migration sentence pointing at the migration file)

#### R15: One-table status frame
§4 Status Frame Format SHALL specify the single table `| | ID | Kind | State | Checked | Next |` with the per-column rules in intake B4's table (▶ = has `then` or pending; ⏸ held naming the dep; Kind ` · <repo basename>`; State health emoji + text per kind; Checked `live` / age / ` ⚠` past `check_every` / ` 🔴` past 2× / note age; Next ≤ 5 words; ordering kind → repo → id; removed items render once with ✅), the header/compact line with `{schedule_summary} · {deliver}` copied from `rk cron list --json`, and keep the no-fence rule, emoji-as-colour, italic footnote, two-shape rule, `⏏ shell`. Per-repo `📂` anchors, the Watches table, and the Health column are deleted.

- **GIVEN** the rewritten frame section
- **WHEN** an operator renders a tick with the intake B4 example data
- **THEN** the output is one table matching the intake example row-for-row

#### R16: Question detection via `rk mux capture --classify`
§5 Question Detection SHALL name `rk mux capture <pane> --lines 40 --classify --json` as the detection step after the `waiting` signal (class + matched line feed the unchanged answer model); `fab pane questions` SHALL not be referenced anywhere in the operator skill.

- **GIVEN** the rewritten §5
- **WHEN** grepping the skill for `pane questions`
- **THEN** zero hits

#### R17: Spawn via `rk tab new`, marks via `rk tab mark/note`, majority-rule session inference
§6 Spawning an Agent SHALL: collapse step 2 to the majority rule over `rk mux sessions --json` `role: user` rows (minus the operator's session) by `cwd`-under-target-repo pane count, ties → §8 setting → the one ask; replace step 7 with `rk tab new --session =<session> --cwd <worktree> --name <wt> --ready --json -- <argv…>` (argv, no `; exec` composition, `ready:` verdict handled per `_cli-agents.md` § Await); mark with `rk tab mark @<window_id> auto` + `rk tab note @<window_id> "<id> · <stage>"`, `blocked` while waiting, `--off` + `"✓ <id> done"` at removal; replace step 8 with `track add --kind fab-change …`; move "pipeline-first" and "spawn in a worktree" from §1 into the fab-change kind paragraph; gate `wt` on a fab-change spawn. `fab pane window-name` and `»`/`›` SHALL not be referenced in the skill.

- **GIVEN** the rewritten §6
- **WHEN** grepping the skill for `window-name`, `»`, `›`, `new-window`, `-P -F`, `exec "$SHELL"`
- **THEN** zero hits

#### R18: Choreography re-pointed at items; §7 becomes the Linear/Slack kinds section
§6 Choreography SHALL define "Dependency satisfied" once against items (dep `done`; fab-change same-repo null-`stop_stage` also needs its PR), express a queue as N `track add --kind fab-change` calls chained by `depends_on` (nearest same-repo predecessor / immediate predecessor) with `--mode` on the first, keep the confirmation line, the two misfit conditions, the three mode diagrams, Queue ordering, Queue Completion Summary, Ordered Merge (stacked-prs steps, halt-dependents-only), and rewrite Auto-Merge Choreography rule 4 as a chain of `github-pr` items (`then: "arm next: gh pr merge --auto --squash <next>"`, stall rule on `unchanged ≥ 3`, disarm over the halted sequences' items). The `coordination` note disappears. Target ≈180 lines for §6 Choreography (Dependency Resolution through Auto-Merge). §7 SHALL become "Linear and Slack items" (probe agent, `seen` dedupe, `then` prose, auto-pause after 3 `observe --error`) with the conversational map of utterances → `track` verbs from intake B4.

- **GIVEN** the rewritten §6/§7
- **WHEN** grepping for `coordination`, `autopilot start`, `watch add`, `note add`
- **THEN** zero hits, and every merge-sequence rule references `github-pr` items

### CLI Reference and Helpers

#### R19: `_cli-fab-operator.md` and sibling helpers match the binary
`src/kit/skills/_cli-fab-operator.md` § fab operator SHALL document `track add|update|observe|rm|list|clock`, the R7–R11 tick document, the shared state-verb mechanics (owned section now `tracked` + `branch_map` + `clock_override`), the clock side effect + schedule reconcile paragraph, and the legacy conversion; the `enroll / update / remove`, `note`, `watch`, `autopilot` subsections SHALL be deleted. `_cli-agents.md` § Spawn Composition SHALL point the open step at `rk tab new` argv form and § Pre-Send Validation / § Peek at `rk mux capture --classify`; `_cli-external.md` § wt § Operator Spawning Rules SHALL keep only the probe-and-route recipe. `_cli-fab-pane.md` is untouched (the `questions`/`window-name` verbs stay). No deployed file cites `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*`.

- **GIVEN** the rewritten helpers
- **WHEN** the fab-kit deployed-path guard test and `fab sync` run
- **THEN** both pass and `.agents/skills/` mirrors `src/kit/skills/`

### Specs

#### R20: Specs updated
`docs/specs/operator.md` SHALL gain a version-history row `v11` (generic tracked items, probe loop, derived schedule, one-table frame); `docs/specs/skills.md`'s `/fab-operator` entry SHALL describe the four-step tick and the `track` verb family. The Constitution is untouched. (Memory files listed in the intake's Affected Memory are hydrate's, not apply's.)

- **GIVEN** the edited specs
- **WHEN** `fab docs-index docs/specs --check` runs
- **THEN** exit 0

### Non-Goals

- Deleting the `fab pane questions` / `fab pane window-name` Go verbs — follow-up backlog item.
- Any rk-side probe or wake mechanism; any change to run-kit.
- Parallel shell probes; `or` / comparison operators in `done_when`.
- Changing the state-file path or slug rule (cross-repo contract with run-kit).
- Memory (`docs/memory/`) edits at apply — hydrate owns them.

### Design Decisions

#### Probes Run in fab at tick-start, Never as an rk Wake
**Decision**: `tick-start --diff` runs every mechanical probe and compares declared fields against `last`.
**Why**: a raw-output fingerprint wakes on noise; the LLM declares the fields that matter at `track add`.
**Rejected**: rk `wake_on: probe-change` — user decision, not reopened.
*Introduced by*: 260911-4a8m-operator-generic-tracked-items

#### Binary Derives the Cron Schedule From the Tracked Set
**Decision**: the schedule is a pure function of the items' `check_every` values, applied through `rk cron edit` only on change; user overrides are bounded by `--for`.
**Why**: a per-tick LLM cadence judgment is conversation-only state, the same failure class as the #913 incident.
**Rejected**: LLM-issued `rk cron edit` per tick; unbounded overrides.
*Introduced by*: 260911-4a8m-operator-generic-tracked-items

#### `done_when` Is Equality-Only
**Decision**: ` and `-joined `==`/`!=` clauses over JSON scalars.
**Why**: table-testable, hard for an LLM to get wrong, covers every kind's built-in predicate.
**Rejected**: jq expressions, regex, comparison operators — extensible later if a kind needs them.
*Introduced by*: 260911-4a8m-operator-generic-tracked-items

#### Legacy State Converts on First Touch
**Decision**: any `fab operator` verb converts a legacy-shaped file in its own atomic write, refusing on a running autopilot queue.
**Why**: the operator may run with no `fab/` project, so `/fab-setup migrations` cannot be the only trigger; the file is binary-owned.
**Rejected**: a separate `track migrate` verb with every other verb refusing until it runs.
*Introduced by*: 260911-4a8m-operator-generic-tracked-items

#### Choreography Stays Inline in the Operator Skill
**Decision**: dependency resolution, merge modes, ordered merge, auto-merge remain in `fab-operator.md` §6, re-pointed at items.
**Why**: it is the operator's main job and must survive a compaction reload with a merge sequence armed.
**Rejected**: an on-demand choreography helper.
*Introduced by*: 260911-4a8m-operator-generic-tracked-items

### Deprecated Requirements

#### `fab operator enroll / update / remove`, `watch *`, `autopilot *`, `note *`
**Reason**: five schemas with separate lifecycles are what made non-fab items invisible to the tick.
**Migration**: `fab operator track add|update|observe|rm|list|clock`; legacy files auto-convert (R5).

#### Merge sequence persisted as a `kind: coordination` note
**Reason**: prose state the tick never probes.
**Migration**: a chain of `github-pr` items with `depends_on` and `then` (R18).

#### `»` / `›` window-name markers and the `fab pane questions` sweep in the operator skill
**Reason**: run-kit owns window marks (`rk tab mark/note`) and prompt classification (`rk mux capture --classify`).
**Migration**: R16, R17. The Go verbs stay for now (Non-Goals).

#### Idle Message and per-repo `📂` frame anchors
**Reason**: the one-table frame with a Checked column is the freshness surface; nothing renders between ticks.
**Migration**: R15.

## Tasks

### Phase 1: Setup

- [x] T001 Create `src/go/fab/internal/predicate/predicate.go` + `predicate_test.go`: `Parse(string) (Predicate, error)` and `(Predicate).Eval(map[string]interface{}) bool` implementing R2 (dotted paths, ` and `, `==`/`!=`, JSON scalars, absent path = null); table tests for valid/invalid inputs including the unquoted-string and `or` rejections <!-- R2 -->
- [x] T002 [P] Add `trackedItem`/`probeSpec` typed structs, the kinds-defaults table, `check_every` floor/default logic, and item validation in a new `src/go/fab/cmd/fab/operator_track_types.go` (+ tests); extend `emptyOperatorState()` in `operator_state.go` to `{tracked: [], branch_map: {}}`; keep `operatorSection` tolerant-read/typed-write for the `tracked` section <!-- R1, R3 -->
- [x] T003 [P] Implement legacy detection and conversion in `src/go/fab/cmd/fab/operator_migrate.go` (+ tests covering the R5 scenarios incl. the running-autopilot refusal and idempotence), wired into `loadOperatorState`/`mutateOperatorStateClock` so every verb converts on first touch <!-- R5 -->

### Phase 2: Core Implementation

- [x] T004 Implement `src/go/fab/cmd/fab/operator_track.go` (+ `operator_track_test.go`): `track add|update|rm|list` per R4 incl. fab-change flag sugar (`--pane --repo --session --branch --stage --agent --stop-stage --spawned-by --depends-on --mode`) writing `scope` + `branch_map`, `--scope` JSON merge, `--pause/--resume`, all validation errors; register under `operator.go`; delete `operator_monitored.go`, `operator_watch.go`, `operator_autopilot.go`, `operator_note.go` and their `_test.go` files; port `state`'s OPEN NOTES header to `kind: note` items <!-- R1, R4 -->
- [x] T005 Implement `track observe` (fields/whole-object store, `--seen` 200-cap, `--error` failures/paused, `done_when` eval) and `track clock` (`clock_override` write/`--off`, `--for` required) in `operator_track.go` (+ tests) <!-- R4, R13 -->
- [x] T006 Replace `operatorTracked` in `src/go/fab/cmd/fab/operator_clock.go` with the "any item not done" predicate over `tracked`; extend `operatorCronRow` with `schedule {kind,min,max,every}` and `deliver`; add `deriveOperatorSchedule(items, override, epoch bool)` and `reconcileOperatorSchedule()` issuing one `rk cron edit` only on difference; epoch detection via the operator pane row's `agent_state` from `rk mux panes --json` (reuse the pane-map runner seam); expire stale `clock_override`; hook the reconcile into `mutateOperatorStateClock` and the tick reconcile; tests via the `rkCronRunner` stub with fixture rows (`operator_clock_test.go`) <!-- R6, R12, R13 -->
- [x] T007 Rework `src/go/fab/cmd/fab/operator_tick_start.go`: item-keyed pane join (`scope.pane`), shell probe runner behind an injectable `probeRunner` seam (argv, 10 s timeout, 60 s budget, JSON object, fields extraction, `last`/`unchanged`/`failures`/`paused` bookkeeping), due-set computation, `done_when` evaluation via `internal/predicate`, `held`/`pending`/`live`/`watching`/`stale`/`paused`/`done` derivation, `needs_check:` block <!-- R7, R8, R10 -->
- [x] T008 In `operator_tick_start.go` emit the new document: `deltas` (`changed` with per-field from/to, `done` with `then`, `stale` with `age`, `probe_error` with `error`/`failures`/`paused`, pane deltas keyed by `id`), `candidates` (id key), `needs_check`, `items` (per-kind fields, ordering kind → repo → id, `state`, `next`) or `fleet_summary` with the `tracked ≥ sum` rule and the full-document triggers; pass `has_agent` through `pane_map.go`'s pane row and gate the process-tree walk on `null`; rewrite `operator_tick_diff_test.go` and `operator_test.go` against the new shapes <!-- R8, R9, R10, R11 -->
- [x] T009 Run `gofmt -l`, `go vet ./...`, and `go test ./cmd/fab/... ./internal/predicate/...` from `src/go/fab`; fix failures; then run the full `go test ./...` for `src/go/fab` and the fab-kit deployed-path guard test in `src/go/fab-kit/cmd/fab/` <!-- R1, R4, R7, R12, R19 -->

### Phase 3: Integration & Skill Rewrite

- [x] T010 Rewrite `src/kit/skills/_cli-fab-operator.md` § fab operator per R19: `track` verb contracts, the R7–R11 tick document (copy the YAML shape from intake B2), shared state-verb mechanics naming `tracked`/`branch_map`/`clock_override`, clock side effect + schedule reconcile + override paragraph, legacy conversion note; delete the enroll/update/remove, note, watch, autopilot subsections; update the file's Contents list <!-- R19 -->
- [x] T011 Rewrite `src/kit/skills/fab-operator.md` §1 (move pipeline-first / spawn-in-worktree into the fab-change kind), §2 (wt gate scoped to fab-change spawns; Init reads `track list`), §4 (Clock prose → track verbs + derived schedule + `track clock --for`; Post-Compaction Reload step list; Tracked Items section; four-step Tick Behavior; one-table Status Frame Format with the intake B4 example; delete Idle Message, note kinds, routing doctrine, `📂` anchors, Watches table); update Contents and the frontmatter description <!-- R14, R15 -->
- [x] T012 Rewrite `src/kit/skills/fab-operator.md` §5 Question Detection (`rk mux capture --classify`), §6 Spawning (majority-rule session inference, `rk tab new` argv step, `rk tab mark/note`, `track add` enrolment, fab-change kind paragraph), §6 Choreography (Dependency satisfied on items, chain-as-queue, github-pr merge sequence replacing rule 4, ≈180 lines), §7 → Linear and Slack items + conversational map; §8 Settings drop anything referencing removed surfaces <!-- R16, R17, R18 -->
- [x] T013 [P] Re-point `src/kit/skills/_cli-agents.md` (§ Spawn Composition → `rk tab new` argv; § Pre-Send Validation / § Peek → `rk mux capture --classify`) and `src/kit/skills/_cli-external.md` § wt § Operator Spawning Rules (keep probe-and-route only); write `src/kit/migrations/2.24.9-to-2.25.0.md` (Summary / Pre-check `fab operator state` succeeds / Changes = none, binary converts / Verification incl. the running-queue refusal); run `fab sync` <!-- R5, R19 -->
- [x] T014 Sibling sweep: grep `src/kit docs/specs` for `monitored|watches:|autopilot|notes_seq|coordination note|enroll|fab pane questions|window-name|»|›|Idle Message|fleet:|completion delta|phase_plan` and reconcile every hit in deployed skills and specs (memory hits are hydrate's — list them in the result summary, do not edit `docs/memory/`) <!-- R14, R16, R17, R18, R19 -->

### Phase 4: Polish

- [x] T015 Update `docs/specs/operator.md` (v11 version-history row + the intro sentence "evolved through eleven iterations") and `docs/specs/skills.md` `/fab-operator` entry (four-step tick, `track` verbs, helper list unchanged); run `fab docs-index docs/specs --check` <!-- R20 -->

## Execution Order

- T001 blocks T005, T007 (predicate evaluation)
- T002 blocks T003, T004 (types)
- T004 blocks T005 (observe/clock live in the same file)
- T006 and T007 are independent of each other; both block T008
- T009 gates Phase 3 (docs describe a green binary)
- T010 blocks T011/T012 (the skill points at the reference contracts)
- T014 runs after T010–T013; T015 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `track add` writes a `tracked` item with the kind's defaults and never creates `monitored`/`watches`/`autopilot`/`notes` keys; fab-change adds also write `branch_map`
- [x] A-002 R2: `internal/predicate` parses and evaluates the pinned grammar and rejects `or`, comparisons, unquoted strings, functions
- [x] A-003 R3: `check_every` floor 1m rejected, default 5m for shell/agent, null for pane/none
- [x] A-004 R4: all six `track` verbs exist with the documented flags and validation; the six legacy verb families are gone from `fab operator --help`
- [x] A-005 R5: a legacy v10 file converts on first touch (idempotent) and a running autopilot queue is refused with the exact message
- [x] A-006 R6: the tracked predicate is "any item not done"; `branch_map` survives `track rm`
- [x] A-007 R7: shell probes run argv-only with 10 s timeout under a 60 s tick budget, extract only declared fields, and maintain `last`/`unchanged`/`failures`/`paused`
- [x] A-008 R8: all nine delta kinds emit with their pinned fields and delivery classes
- [x] A-009 R9: the tick document block order and `items:`/`fleet_summary:` selection match the requirement
- [x] A-010 R10: `held`/`pending`/`live`/`watching`/`stale`/`paused`/`done` and `next` derive as specified
- [x] A-011 R11: `has_agent` false/true short-circuit; `null` reaches the process-tree walk
- [x] A-012 R12: derived schedule computed per the table and applied via one `rk cron edit` only when the structured row differs
- [x] A-013 R13: `track clock --for` writes/expires `clock_override`; `--for` required; `--off` clears
- [x] A-014 R14: §4 four-step tick, Tracked Items section, Idle Message deleted, notes as a kind
- [x] A-015 R15: one-table frame spec with the six columns and the Checked/⚠/🔴 rules; example matches intake B4
- [x] A-016 R16: §5 detection via `rk mux capture --classify`; no `pane questions` in the skill
- [x] A-017 R17: §6 spawn via `rk tab new` argv, marks via `rk tab mark/note`, majority-rule session inference; no `window-name`/`»`/`›`/`new-window`
- [x] A-018 R18: choreography re-pointed at items; merge sequence is a `github-pr` chain; §7 is the Linear/Slack kinds section with the conversational map
- [x] A-019 R19: `_cli-fab-operator.md` documents exactly the shipped verbs and tick document; `_cli-agents.md`/`_cli-external.md` re-pointed; `_cli-fab-pane.md` untouched
- [x] A-020 R20: `docs/specs/operator.md` v11 row and `docs/specs/skills.md` operator entry updated; specs index check passes

### Behavioral Correctness

- [x] A-021 R8: `done` is level-triggered (re-emits until `track rm`); `changed` is consumed-on-read (emits once)
- [x] A-022 R9: `fleet_summary.tracked ≥ waiting + idle + active + unknown` with mixed pane and shell items; a non-empty `needs_check:` forces the full document
- [x] A-023 R12: a lease or mute does not suppress the schedule edit; an equal derived value issues no rk call
- [x] A-024 R5: conversion maps `watches.known ∪ completed` → `seen`, autopilot queue → chained pane-less fab-change items with `scope.merge_mode`, resolved notes dropped

### Removal Verification

- [x] A-025 R4: `operator_monitored.go`, `operator_watch.go`, `operator_autopilot.go`, `operator_note.go` and their tests are deleted; no dead `monitoredEntry`/`watchEntry`/`autopilotState`/`noteEntry` code remains outside the conversion path
- [x] A-026 R14: grep of `fab-operator.md` for `monitored:|watches:|autopilot:|notes_seq|Idle Message|phase_plan|fleet:` returns zero hits
- [x] A-027 R18: grep of `fab-operator.md` for `coordination|autopilot start|watch add|note add` returns zero hits

### Scenario Coverage

- [x] A-028 R7: a `github-pr` item whose probe flips `state` OPEN→MERGED emits `changed` and `done` (with `then`) in the same tick — covered by a test with the probe runner stubbed
- [x] A-029 R8: an `agent` item 11 minutes past a 5m cadence emits `stale` and appears in `needs_check:`; `observe` clears it — covered by test
- [x] A-030 R10: a `depends_on` chain renders `held` → `pending` as the dependency completes — covered by test
- [x] A-031 R12: backoff↔idle-every transitions verified against fixture `rk cron list --json` rows in `operator_clock_test.go`

### Edge Cases & Error Handling

- [x] A-032 R7: probe stdout that is not a JSON object, a timeout, and a non-zero exit each count as one failure; the third pauses the item; over-budget items are skipped without touching `failures`
- [x] A-033 R4: every validation error (duplicate id, unknown id, malformed predicate, floor, missing `--fields`, unknown `--depends-on`, `clock` without `--for`) exits non-zero with a one-line message and writes nothing
- [x] A-034 R5: a state file with `tracked` already present and a stray legacy key is left as-is (no double conversion)
- [x] A-035 R11: a pane row lacking `has_agent` entirely (older rk) behaves as `null`

### Code Quality

- [x] A-036 Pattern consistency: new verbs follow the cobra + `mutateOperatorState` + `operatorSection` patterns and the `rkCronRunner`-style injectable seams of the existing operator files
- [x] A-037 No unnecessary duplication: the pane snapshot path, RFC3339 timestamp source, and rk argv runner are reused, not re-implemented
- [x] A-038 No god functions: `runOperatorTickStartDiff` and the new verbs are split into focused helpers (probe run, diff, derive state, emit) each well under 50 lines where practical
- [x] A-039 Named constants for the 10 s probe timeout, 60 s tick budget, 3-failure pause cap, 200-entry `seen` cap, 1m floor, 5m default, 2× stale multiplier
- [x] A-040 Canonical sources only: every skill edit is under `src/kit/skills/`; deployed copies regenerated by `fab sync`
- [x] A-041 Migration shipped: `src/kit/migrations/2.24.9-to-2.25.0.md` exists in the documented format
- [x] A-042 CLI reference + tests updated together: `_cli-fab-operator.md` matches every new/changed/removed `fab operator` signature and Go tests cover them
- [x] A-043 Owner-or-pointer: the operator skill points at `_cli-fab-operator.md` for verb contracts and does not restate them; no deployed file cites `docs/specs/*`, `docs/memory/*`, `docs/site/*`, `src/go/*`
- [x] A-044 Sibling sweep performed: T014's grep shows no stale claim in `src/kit` or `docs/specs`; remaining `docs/memory` hits are listed for hydrate
- [x] A-045 Tests run scoped then widened: `go test ./cmd/fab/... ./internal/predicate/...` then `go test ./...` in `src/go/fab` are green; `gofmt -l` prints nothing

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Line targets (`fab-operator.md` ≈480, §6 Choreography ≈180) are targets, not acceptance gates — completeness of R14–R18 is what review checks.

## Deletion Candidates

- `src/go/fab/cmd/fab/pane_questions.go` (`fab pane questions` verb) + `src/kit/skills/_cli-fab-pane.md` § questions — the operator skill no longer consumes it (detection rides `rk mux capture --classify`); retention is deliberate (dispatch-facing, intake Assumption 11 / Non-Goals) with removal declared as a follow-up backlog item
- `src/go/fab/cmd/fab/pane_window_name.go` (`fab pane window-name` verbs) + `src/kit/skills/_cli-fab-pane.md` § window-name — the operator skill no longer consumes them (window marks ride `rk tab mark`/`rk tab note`); same deliberate retention / follow-up backlog item

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | `done_when` evaluator lives in a new `src/go/fab/internal/predicate` package rather than inside `cmd/fab` | Reusable, table-testable in isolation; matches the `internal/<concern>` layout | S:80 R:90 A:85 D:85 |
| 2 | Confident | Migration file is named `2.24.9-to-2.25.0.md` (minor bump: new verbs, removed verbs) | Range is fixed by the release author; renaming at release is a one-line change | S:70 R:95 A:80 D:80 |
| 3 | Certain | Memory (`docs/memory/`) edits are hydrate's; apply touches specs and skills only | Pipeline contract: hydrate owns memory | S:90 R:95 A:100 D:100 |
| 4 | Confident | Shell probes and rk calls sit behind injectable runner seams so tests need no live `gh`/`rk` | Existing `rkCronRunner`/pane-runner precedent | S:85 R:90 A:95 D:90 |
| 5 | Confident | `github-pr` default probe derives `--repo <owner/repo>` from `scope.repo` via `gh repo view --json nameWithOwner` at add time when not given in scope; failure leaves argv without `--repo` (gh infers from cwd) | Keeps `track add pr-N` terse; the LLM may always pass explicit argv | S:60 R:85 A:75 D:70 |
| 6 | Confident | `agent_state` non-null on the operator pane row is the epoch signal for `--idle-every` vs `--every` | Intake row 13, user-confirmed | S:95 R:85 A:75 D:80 |
| 7 | Certain | The `fab pane questions` / `window-name` Go verbs stay; only skill references are removed | Intake row 11 (corrected rationale: scope bound, follow-up deletion) | S:90 R:90 A:95 D:90 |
| 8 | Confident | Legacy `monitored.stage`/`agent` become `scope.stage`/`scope.agent` and the pane baseline diff keys on them | Byte-compatible carry of today's baseline semantics into the kind's scope | S:80 R:80 A:90 D:85 |

8 assumptions (2 certain, 6 confident, 0 tentative).
