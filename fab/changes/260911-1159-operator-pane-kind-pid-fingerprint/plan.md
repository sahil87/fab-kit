# Plan: Operator `pane` kind replaces `fab-change` — join on pane ID plus pid fingerprint, change as an observed field; plus three agreed skill slims

**Change**: 260911-1159-operator-pane-kind-pid-fingerprint
**Intake**: `intake.md`

## Requirements

### Operator Track: Kind Rename `fab-change` → `pane`

#### R1: The tracked-item kind `fab-change` SHALL be renamed to `pane` across the binary
Every constant, table, and use site (`operator_track_types.go`, `operator_track.go`, `operator_tick_start.go`, `operator_migrate.go`, tests) uses `kindPane = "pane"`; `trackKinds` maps `pane → {defaultProbe: probePane}`; `trackKindNames` = `"pane, github-pr, linear, slack, shell, task, note"`; `pane` keeps `tickKindOrder` slot 0. `fab-change` is removed outright at `track add` and `track list --kind` — it exits non-zero through the existing unknown-kind path (`unknown --kind "fab-change" (valid: pane, github-pr, linear, slack, shell, task, note)`); no alias, no deprecation message.

- **GIVEN** a user runs `fab operator track add x --kind fab-change --pane %5`
- **WHEN** the kind is validated
- **THEN** the command exits non-zero with the unknown-kind error listing `pane` first
- **AND** `fab operator track list --kind fab-change` fails the same way

#### R2: The `pane` kind's sugar flags and pinned scope keys SHALL be updated
The sugar list becomes `--pane --repo --session --branch --stage --agent --stop-stage --spawned-by --change`; the error text becomes `--<flag> applies only to kind pane`; the `--kind` help lists `pane | github-pr | …`; the `--mode` help says "merge mode for a chained pane item". The pinned scope keys for `--kind pane` are eleven, all null until set: `pane, pane_pid, change, repo, session, branch, stage, agent, stop_stage, spawned_by, merge_mode`. `--change <id>` writes `scope.change` (non-empty string; existence NOT validated). `fabChangeScope()` becomes `paneScope()` seeding the eleven keys.

- **GIVEN** `fab operator track add tireless-perch --kind pane --pane %222 --repo /x --session work --change 4a8m`
- **WHEN** the item is created
- **THEN** its scope carries all eleven keys with `pane`, `pane_pid` (see R3), `change: 4a8m`, `repo`, `session` set and the rest null
- **AND** `--pane` on kind `task` errors `--pane applies only to kind pane`

### Operator Track: Pane PID Fingerprint

#### R3: Setting `scope.pane` SHALL record the pane's shell pid in `scope.pane_pid`
Whenever any verb sets or changes `scope.pane` to a non-empty value (`track add --pane %N`, `track add --scope '{"pane":"%N"}'`, `track update --scope '{"pane":"%N"}'`), the same mutation records the pane's shell pid (integer) via `pane.GetPanePID(paneID, server)` with the default server. On ANY lookup failure (tmux unqueryable, pane absent, unparseable pid) `pane_pid` is stored null and the verb proceeds. Clearing `scope.pane` (explicitly-empty value) also nulls `pane_pid`.

- **GIVEN** a live pane `%222` whose shell pid is 48213
- **WHEN** `track add x --kind pane --pane %222` runs
- **THEN** the item's scope has `pane: "%222"` and `pane_pid: 48213`
- **AND** when tmux cannot be queried, `pane_pid` is null and the add succeeds

#### R4: The tick SHALL fetch per-pane pids with one batched call
`tick-start --diff` fetches pids for the whole server with one `tmux list-panes -a -F '#{pane_id} #{pane_pid}'` call per tick, only when at least one tracked pane item has a non-null `scope.pane` (the existing `anyPaneItems` gate), parsed into `map[paneID]pid`. The helper SHALL be an exported `internal/pane` function `ListPanePIDs(server string) (map[string]int, error)` beside `GetPanePID` — argv-only, no shell string, bounded timeout — with an injectable package-level seam in the tick (the `tickSnapshotRows`/`rkPanesRunner` precedent). A failed batch call yields an empty map and is NOT an error.

- **GIVEN** three tracked pane items and a live tmux server
- **WHEN** a diff tick runs
- **THEN** exactly one `list-panes` subprocess runs and the pid map covers every pane on the server
- **AND** a failing batch call degrades every fingerprint comparison to "unreadable ⇒ not a mismatch"

#### R5: `pane_mismatch` SHALL mean recycled-pane only
`diffPaneItems` replaces the `row.changeID != it.ID` check: `pane_mismatch` fires only when the recorded `scope.pane_pid` is non-null AND the pane's current shell pid (from the batched map) differs. A null recorded fingerprint or an unreadable current pid joins on the pane id alone. `found:` stays the observed change id or null. Evaluation order per item stays `pane_death` → `pane_mismatch` → `agent_exited` → clean join; a mismatched pane gets no baseline write, no stage/change diff, no `candidates:` row. The observed change id never participates in the mismatch decision.

- **GIVEN** an item with `pane: %5, pane_pid: 100` and a current pane `%5` whose pid is 200
- **WHEN** the diff tick runs
- **THEN** a `pane_mismatch` delta emits with `found:` = observed change or null, and no baseline write occurs
- **AND** with `pane_pid: null` the same pane joins cleanly regardless of its pid

### Operator Tick: Change as an Observed Field

#### R6: `scope.change` SHALL be baseline-maintained like `scope.stage`
On a clean join the baseline writer sets `scope.change` ← the snapshot row's `changeID` when resolved (non-empty); an unresolved snapshot change fabricates no delta and leaves the baseline alone (sticky — mirrors the em-dash stage rule). When the observed change differs from the baseline (null → id, or id → other id), the tick SHALL emit a consumed-on-read `changed` delta (`fields: {change: {from, to}}`) and consume it with the same-write baseline update. The existing `changed` contract holds (operator reports; `then` runs only if it names a changed reaction); `stage_advance`/`review_fail` may emit independently in the same tick.

- **GIVEN** a pane item with baseline `change: null` whose snapshot resolves change `4a8m`
- **WHEN** the diff tick runs
- **THEN** a `changed` delta with `fields.change {from: null, to: "4a8m"}` emits and the baseline stores `change: 4a8m`
- **AND** a later tick with an unresolved snapshot change emits nothing and keeps `change: 4a8m`

#### R7: A first-observed change SHALL write `branch_map[<change>]`
When a change is first observed for an item (baseline null → id) and `branch_map[<change>]` is absent, the same mutation writes `branch_map[<change>] = {branch, repo}` with `repo` = `scope.repo` and `branch` resolved via `git -C <cwd> branch --show-current` once, at that tick only. `paneRow` SHALL carry the raw enumeration `cwd` snapshot-internally (like `command`/`hasAgent` — never rendered). An empty result (detached HEAD) or git failure skips the write; the next tick where the change is still observed and the entry still absent retries. `track add` with `--branch`+`--repo` keeps writing `branch_map` keyed by `--change` when given, else by the item id.

- **GIVEN** a raw-spawn item `tireless-perch` with `scope.repo: /x` and a snapshot row whose `cwd` is `/x/wt` on branch `feat-y`
- **WHEN** change `4a8m` is first observed and `branch_map` has no `4a8m` entry
- **THEN** `branch_map["4a8m"] = {branch: "feat-y", repo: "/x"}` is written in the same mutation
- **AND** a detached HEAD skips the write and retries on a later tick

#### R8: The `pane` done predicate SHALL split on `scope.change`
With `done_when` null on a `pane` item: `scope.change` non-null ⇒ the existing built-in `tickCompleted(stopStage, stage, displayState)`; `scope.change` null ⇒ never done on its own (the item leaves only via `track rm` or a `then`). `done_at` persistence unchanged. `--stop-stage` on a change-less item is accepted and inert until a change is observed.

- **GIVEN** a pane item with `done_when: null`
- **WHEN** `scope.change` is null and the tick runs at any stage
- **THEN** the item is never marked done
- **AND** once `scope.change` is non-null and the change reaches `review-pr` done/skipped (or `stop_stage`), the item completes with `done_at` set

#### R9: `items:` pane rows SHALL carry a present-keyed `change` field
Pane rows render `id, kind, state, pane, change, repo, session, stage, display_state, agent_state, idle_duration, pr_url, checked_at: null, next` — `change` after `pane`, null when none. `stage`, `display_state`, `pr_url` are null when no change is observed. Unjoined rows carry baseline identity incl. `change` from scope.

- **GIVEN** a joined pane item with no observed change
- **WHEN** `tick-start --diff` renders `items:`
- **THEN** the row shows `change: null, stage: null, display_state: null, pr_url: null` and the pane's live `agent_state`/`idle_duration`

#### R10: `candidates:` rows SHALL gain `state_duration`
`candidates:` rows gain `state_duration`: rk's `agent_state_duration` verbatim for both `waiting` and `idle` rows (`null` when rk reports none); `idle_duration` keeps its idle-only meaning, unchanged. Waiting-first ordering and the exclusion of active/unknown/mismatched/exited panes are unchanged.

- **GIVEN** a waiting pane with `agent_state_duration: 32m` and an idle pane with `agent_state_duration: 12m`
- **WHEN** candidates render
- **THEN** both rows carry `state_duration` (`32m`/`12m`), the waiting row keeps `idle_duration: null`, and an rk row with no duration yields `state_duration: null`

### Operator Migrate: Legacy Conversion

#### R11: Stored `kind: fab-change` items SHALL convert to `pane` idempotently
The existing Legacy conversion gains a second, idempotent pass firing on ANY verb's read-modify-write when a `tracked`-present file holds an item with `kind: fab-change`: rewrite `kind: pane`, seed `scope.change` = the item's id, seed `scope.pane_pid: null`. Same atomic write, same tolerant-read/typed-write contract; `state` and `track list` convert-and-save then read. `convertMonitoredEntry` and `convertAutopilotQueue` (≤2.24 path) emit `kind: pane` with `scope.change` seeded directly. The change SHALL ship migration file `src/kit/migrations/2.25.1-to-2.26.0.md` in the no-file-edits style of `2.24.9-to-2.25.0.md` (Summary / Pre-check / Changes: None / Verification).

- **GIVEN** a state file with a `tracked` list containing `kind: fab-change`, id `r3m7`
- **WHEN** any operator verb touches the file
- **THEN** the item reads `kind: pane` with `scope.change: r3m7` and `scope.pane_pid: null`, and a second run is a no-op
- **AND** a ≤2.24 `monitored` file converts in one pass to `kind: pane` with `scope.change` seeded

### Kit Skills: fab-operator.md

#### R12: `src/kit/skills/fab-operator.md` SHALL be updated per intake §8a–8i
Only the named sites: §4 schema block / kinds table / lifecycle / tick-behavior / status-frame updates (8a); §4 Clock four-bullet history block deleted, entry YAML + Ownership kept (8b); §5 idle auto-default replaced by the single `state_duration ≥ 30m` candidate-row rule (8c); §6 "The fab-change Kind" rewritten as "The pane Kind and Spawn Rules" — tracking any pane needs no gate (8d); §6 spawn step 2 compressed to the candidate/majority/tie rule, step 7 pin kept (8e); §6 spawn step 8 two forms (`track add` new / `track update` for existing `pending`) (8f); §6 Dependency Resolution one `branch_map` clause (8g); §7 Conversational Map new row (8h); every remaining `fab-change` spelling renamed/reworded so `grep -rn 'fab-change' src/kit/` returns zero (8i). §6 Queues / merge modes / Ordered Merge / Auto-Merge Choreography are renamed in place only, never restructured.

- **GIVEN** the edited `fab-operator.md`
- **WHEN** a user says "track %222"
- **THEN** the skill maps it to exactly one verb (`track add <slug> --kind pane --pane %222 …`) with zero questions
- **AND** `grep -c 'fab-change' src/kit/skills/fab-operator.md` is 0

### Kit Skills: CLI Reference Partial

#### R13: `src/kit/skills/_cli-fab-operator.md` and `_cli-external.md` SHALL reflect every CLI delta
`_cli-fab-operator.md`: tick-start `--diff` join/fingerprint prose, deltas examples (`pane_mismatch` recycled-pane comment, `changed` on `change`), `items:` row + Row field sets gain `change`, candidates `state_duration` prose + example, Detection semantics rewrite of `pane_mismatch`, Baseline writer additions (`scope.change`, `changed` delta, first-observation `branch_map`), built-in completion gated on non-null `scope.change`, Item states `pending` wording; track synopsis sugar list + Kinds table `pane` row + add/update/rm bullets; Legacy conversion paragraph gains the `fab-change`→`pane` rule naming migration 2.25.1→2.26.0; State path/contract paragraph notes the additive `pane_pid`/`change` keys; branch-map rm entry-writer note. `_cli-external.md` (~line 132): "the fab-change kind" → "the pane kind". No `src/kit/**` file cites `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*` (Constitution V; the Go guard test stays green).

- **GIVEN** a reader of `_cli-fab-operator.md` § fab operator track
- **WHEN** they look up the `pane` kind
- **THEN** the sugar list, eleven pinned keys, `--change` semantics, `pane_pid` recording, and the done-predicate split are all documented
- **AND** the deployed-content citation guard test passes

### Backlog

#### R14: `fab/backlog.md` `[lm49]` SHALL be marked shipped
The `[lm49]` line's leading `- [ ]` becomes `- [x]` with ` — SHIPPED via 260911-1159-operator-pane-kind-pid-fingerprint` appended (no PR number — added at archive).

- **GIVEN** `fab/backlog.md` line 47
- **WHEN** apply finishes
- **THEN** the line reads `- [x] [lm49] … — SHIPPED via 260911-1159-operator-pane-kind-pid-fingerprint`

### Non-Goals

- Folding `pending` into `task` — user deferral, revisit later
- Restructuring/moving `fab-operator.md` §6 Queues, merge modes, Ordered Merge, Auto-Merge Choreography — rename-in-place only
- Any run-kit change; any `fab pane map` JSON change; auto-discovery of untracked panes; a `--change` sugar on `track update`
- Any `docs/specs/*` or `docs/memory/*` edit (hydrate owns memory)
- Any change to the operator state-file path or slug rule

### Design Decisions

#### Pane Is the Identity; Change Is Observed
**Decision**: The operator's core tracked kind is `pane` — join on `scope.pane` plus a `scope.pane_pid` fingerprint; the change is an observed, baseline-maintained field (`scope.change`) diffed via the consumed-on-read `changed` delta.
**Why**: The kind name and change-keyed join made the LLM refuse to track change-less panes and made the binary emit a false `pane_mismatch` every tick; the pane is what the operator actually watches.
**Rejected**: Keep the `fab-change` name and only relax the join (the name itself drove the refusal); name the kind `agent` (collides with the existing `agent` probe mode); `pane_start_time` fingerprint (empty on tmux 3.7c); per-item `display-message` pid lookups (N subprocesses per tick vs. one batched `list-panes -a`).
*Introduced by*: 260911-1159-operator-pane-kind-pid-fingerprint

#### `pane_mismatch` Means Recycled Pane Only
**Decision**: `pane_mismatch` fires only when a non-null recorded `pane_pid` differs from the pane's current shell pid; null recorded pid or unreadable current pid joins on pane id alone.
**Why**: tmux recycles `%N` across server restarts while the socket-keyed state file survives; the pid fingerprint (already dispatch's recycle discriminator via `PaneWorkerAlive`) is the proof that a pane is not the one tracked. The dual meaning (recycle vs. change appearing) was the false-alarm source.
**Rejected**: Keeping the `row.changeID != it.ID` check (misfires for raw-text spawns, fresh changes, and change switches); treating an unreadable pid as a mismatch (violates the existing `PaneWorkerAlive` unreadable-is-not-a-mismatch rule).
*Introduced by*: 260911-1159-operator-pane-kind-pid-fingerprint

## Tasks

### Phase 1: Setup

- [x] T001 Add exported `ListPanePIDs(server string) (map[string]int, error)` to `src/go/fab/internal/pane/pane.go` beside `GetPanePID` — argv-only `tmux list-panes -a -F '#{pane_id} #{pane_pid}'`, bounded timeout, parser unit tests in `src/go/fab/internal/pane/` (stubbed runner seam per existing pane tests) <!-- R4 -->

### Phase 2: Core Implementation

- [x] T002 Rename the kind in `src/go/fab/cmd/fab/operator_track_types.go`: `kindFabChange = "fab-change"` → `kindPane = "pane"` (const, `trackKinds` table, `trackKindNames` string); `fab-change` falls through the existing unknown-kind error path <!-- R1 -->
- [x] T003 Update `src/go/fab/cmd/fab/operator_track.go`: sugar list gains `--change`; error text `--<flag> applies only to kind pane`; `--kind`/`--mode` help strings; eleven pinned scope keys (`pane, pane_pid, change, repo, session, branch, stage, agent, stop_stage, spawned_by, merge_mode`); `--change` writes `scope.change` (non-empty only, existence not validated); `track add`/`track update` record `pane_pid` via `pane.GetPanePID` (injectable seam) whenever `scope.pane` is set/changed, null on failure or on pane clear; rename `kindFabChange` use sites (lines ~89, 160, 205, 812, 857) <!-- R1 R2 R3 -->
- [x] T004 Update `src/go/fab/cmd/fab/operator_tick_start.go`: injectable batched-pid seam (package-level var per `tickSnapshotRows` precedent) called once per tick under the `anyPaneItems` gate; rewrite `diffPaneItems` `pane_mismatch` to the fingerprint rule (non-null recorded pid AND differing current pid; `found:` = observed change or null; order `pane_death` → `pane_mismatch` → `agent_exited` → clean join; mismatched pane gets no baseline write/diff/candidate); rename `kindFabChange` sites (lines ~907, 942) <!-- R4 R5 R1 -->
- [x] T005 Baseline writer in `operator_tick_start.go`: set `scope.change` ← snapshot `changeID` when resolved on clean join; emit consumed-on-read `changed` delta (`fields: {change: {from, to}}`) on appear/switch with same-write baseline update; unresolved snapshot change leaves baseline untouched (sticky) <!-- R6 -->
- [x] T006 `src/go/fab/cmd/fab/pane_map.go`: carry enumeration `cwd` onto `paneRow` snapshot-internally (never rendered); in the tick, on first change observation with `branch_map[<change>]` absent, write `{branch, repo}` (`repo` = `scope.repo`, branch via `git -C <cwd> branch --show-current`, injectable seam, skip empty/detached/failure, retry next tick) <!-- R7 -->
- [x] T007 Done predicate in `operator_tick_start.go`: `done_when` null + `scope.change` null ⇒ never done; `scope.change` non-null ⇒ existing `tickCompleted`; `done_at` persistence unchanged <!-- R8 -->
- [x] T008 Row rendering in `operator_tick_start.go`: `items:` pane rows gain present-keyed `change` after `pane` (null when none; `stage`/`display_state`/`pr_url` null when unobserved; unjoined rows carry baseline identity incl. `change`); `candidates:` rows gain `state_duration` (rk `agent_state_duration` verbatim for waiting and idle, null when absent; `idle_duration` unchanged) <!-- R9 R10 -->
- [x] T009 `src/go/fab/cmd/fab/operator_migrate.go`: `fabChangeScope()` → `paneScope()` seeding eleven keys; second idempotent conversion pass (any verb's read-modify-write, `tracked`-present file, `kind: fab-change` ⇒ `kind: pane` + `scope.change` = id + `scope.pane_pid: null`); `convertMonitoredEntry`/`convertAutopilotQueue` emit `kind: pane` with `scope.change` seeded directly <!-- R11 R2 -->

### Phase 3: Tests

- [x] T010 Rename existing `fab-change`/`kindFabChange` test references (13 in `operator_track_test.go`, 4 in `operator_tick_diff_test.go`, 5 in `operator_migrate_test.go`, 6 in `operator_clock_test.go`, 1 in `operator_track_types_test.go`) <!-- R1 -->
- [x] T011 Fingerprint tests in `operator_tick_diff_test.go` (stubbed pid-map seam): match ⇒ clean join; mismatch ⇒ `pane_mismatch` with `found:` observed/null, no baseline write, no candidate; null recorded pid ⇒ join regardless; unreadable current pid ⇒ join <!-- R4 R5 -->
- [x] T012 Change-observation tests: appear (baseline null, snapshot `4a8m`) ⇒ `changed` `{from: null, to: 4a8m}` + baseline update + `branch_map[4a8m]` from `scope.repo` + stubbed branch; switch ⇒ `changed` from/to; unresolved snapshot change ⇒ no delta, baseline untouched; detached/empty branch ⇒ no `branch_map` write, retried next tick <!-- R6 R7 -->
- [x] T013 Done-predicate + rendering tests: change non-null at `review-pr` done ⇒ `done` + `done_at`; change null at any stage ⇒ never done; `items:` pane row carries `change` (null/non-null) and null `stage`/`display_state`/`pr_url` when unobserved; `candidates:` `state_duration` for a waiting and an idle pane, null when absent, `idle_duration` null for waiting <!-- R8 R9 R10 -->
- [x] T014 Migrate + track-verb tests: `tracked`-present `kind: fab-change` ⇒ `kind: pane`, `scope.change` = id, `pane_pid: null`, idempotent; ≤2.24 `monitored`/`autopilot` conversion emits `kind: pane` with `scope.change` seeded; `track add --kind pane` sugar incl. `--change`; `--pane` records `pane_pid` (stubbed seam) and null on failure; `track update --scope '{"pane":…}'` records it too; pane clear nulls `pane_pid`; `--kind fab-change` ⇒ unknown-kind error; `--pane` on kind `task` ⇒ `--pane applies only to kind pane` <!-- R11 R2 R3 R1 -->

### Phase 4: Polish

- [x] T015 [P] Edit `src/kit/skills/fab-operator.md` per intake §8a–8d: §4 schema block/kinds table/lifecycle/tick behavior/status frame; §4 Clock four-bullet history block deleted (entry YAML + Ownership kept); §5 idle auto-default replaced by the single `state_duration ≥ 30m` candidate-row rule; §6 "The fab-change Kind" → "The pane Kind and Spawn Rules" rewrite <!-- R12 -->
- [x] T016 [P] Edit `src/kit/skills/fab-operator.md` per intake §8e–8i: §6 spawn step 2 compression (step 7 pin kept); spawn step 8 two forms; §6 Dependency Resolution `branch_map` clause; §7 Conversational Map row; rename/reword every remaining `fab-change` spelling in the file <!-- R12 -->
- [x] T017 [P] Edit `src/kit/skills/_cli-fab-operator.md` per intake §9 (tick-start join/fingerprint/deltas/items/candidates/detection/baseline/completion/item-states; track synopsis/Kinds/add/update/rm; Legacy conversion paragraph naming migration 2.25.1→2.26.0; state path/contract additive-keys note; branch-map rm note) and `src/kit/skills/_cli-external.md` (~line 132 "the fab-change kind" → "the pane kind") <!-- R13 -->
- [x] T018 [P] Create `src/kit/migrations/2.25.1-to-2.26.0.md` in the no-file-edits style of `2.24.9-to-2.25.0.md`: Summary (kind rename, no alias, `--change` sugar, `pane_pid`/`change` keys, binary-owned first-touch conversion, this migration performs no file edits), Pre-check (`fab operator track list` exits 0), Changes (None), Verification (`fab operator state` shows `kind: pane`, no `kind: fab-change`, idempotent second run) <!-- R11 -->
- [x] T019 [P] Mark `fab/backlog.md` `[lm49]` line `- [x]` + ` — SHIPPED via 260911-1159-operator-pane-kind-pid-fingerprint` (no PR number) <!-- R14 -->
- [x] T020 Final sweeps: `grep -rn 'fab-change' src/kit/` returns zero; `grep -rn 'kindFabChange\|"fab-change"' src/go/fab --include='*.go'` returns only intentional legacy-conversion sites; `gofmt -l ./src/go/fab` clean; scoped `go test ./cmd/fab/ ./internal/pane/` then full `go test ./...` green <!-- R1 R12 R13 -->

## Execution Order

- T001 (pane helper) blocks T004 (tick seam) and T011
- T002 blocks T003, T004, T009, T010 (constant rename ripples)
- T003 blocks T014 (verb tests); T004–T008 share `operator_tick_start.go` — run sequentially
- T010–T014 follow their implementation tasks; T020 is last
- T015–T019 (kit/backlog) are independent of Go and of each other ([P])

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `kindPane = "pane"` everywhere; `--kind fab-change` fails at `track add` and `track list --kind` with the unknown-kind error naming `pane` first; no alias
- [ ] A-002 R2: `pane` kind has the nine-plus-two sugar flags incl. `--change`, eleven pinned scope keys, and the updated error/help texts
- [ ] A-003 R3: setting `scope.pane` via any verb records `pane_pid`; failure stores null and proceeds; clearing the pane nulls `pane_pid`
- [ ] A-004 R4: one batched `list-panes` call per tick under the `anyPaneItems` gate; `ListPanePIDs` exported from `internal/pane`; batch failure degrades to empty map, not an error
- [ ] A-005 R5: `pane_mismatch` fires only on a differing non-null fingerprint; `found:` = observed change or null; evaluation order and mismatched-pane exclusions preserved
- [ ] A-006 R6: baseline writer maintains `scope.change`; appear/switch emits consumed-on-read `changed` with `{from, to}`; unresolved snapshot change is sticky-silent
- [ ] A-007 R7: first-observation `branch_map[<change>]` write from `scope.repo` + `git -C <cwd> branch --show-current`; empty/detached skips and retries; `paneRow` `cwd` never rendered
- [ ] A-008 R8: `done_when` null + `scope.change` null ⇒ never done; non-null ⇒ built-in `tickCompleted` with unchanged `done_at` persistence
- [ ] A-009 R9: `items:` pane rows carry `change` after `pane`; `stage`/`display_state`/`pr_url` null when no change observed
- [ ] A-010 R10: `candidates:` rows carry `state_duration` for waiting and idle (null when absent); `idle_duration` stays idle-only
- [ ] A-011 R11: `tracked`-present `kind: fab-change` converts idempotently to `kind: pane` + `scope.change` = id + `pane_pid: null` on any verb; ≤2.24 emitters produce `kind: pane` with `scope.change` seeded; migration file `2.25.1-to-2.26.0.md` exists in the no-edits style
- [ ] A-012 R12: `fab-operator.md` reflects all of §8a–8i; `grep -c 'fab-change' src/kit/skills/fab-operator.md` is 0
- [ ] A-013 R13: `_cli-fab-operator.md` documents every flag/key/delta/conversion delta; `_cli-external.md` pointer renamed; no `src/kit/**` file cites fab-kit-only paths
- [ ] A-014 R14: `fab/backlog.md` `[lm49]` line is `- [x]` with the SHIPPED suffix and no PR number

### Behavioral Correctness

- [ ] A-015 R5: a recycled `%N` (same pane id, different shell pid) emits `pane_mismatch` every tick until `track rm`; a change appearing on a watched pane emits `changed`, never `pane_mismatch`
- [ ] A-016 R6: a raw-text spawn tracked by worktree name completes via the built-in predicate once its change appears — without any item-id rename
- [ ] A-017 R8: `--stop-stage` on a change-less pane item is accepted and inert until a change is observed

### Removal Verification

- [ ] A-018 R1: no `kindFabChange` identifier remains; `"fab-change"` string literals in Go survive only at intentional legacy-conversion sites

### Scenario Coverage

- [ ] A-019 R5 R6 R8 R10: the intake §7 test matrix exists and passes — fingerprint (4 cases), change observation (3 cases), done predicate (2 cases), items/candidates rendering, legacy conversion, track-verb sugar/errors
- [ ] A-020 R12: the intake incident scenario works on paper: "track %222" maps to one verb with zero questions per the rewritten §6 and the new §7 row

### Edge Cases & Error Handling

- [ ] A-021 R3 R4: tmux unqueryable / pane absent / unparseable pid ⇒ null fingerprint at write time, empty pid map at tick time, never an error
- [ ] A-022 R7: detached HEAD or git failure at first observation skips the `branch_map` write and retries next tick while the entry stays absent
- [ ] A-023 R11: a second conversion run is a no-op (idempotent); read verbs convert-and-save then read

### Code Quality

- [ ] A-024: Readability — new Go follows surrounding naming/structure; no god functions; no magic strings without named constants
- [ ] A-025: No duplicated utilities — pid lookup reuses `pane.GetPanePID`; the batched helper lives beside it in `internal/pane`; test seams follow the `tickSnapshotRows`/`rkPanesRunner` precedent
- [ ] A-026 Canonical source only: no edit under `.agents/skills/` or `.claude/skills/`; all kit edits in `src/kit/`
- [ ] A-027 CLI ⇒ reference + tests: every changed/new `fab operator` flag, key, delta rule, and conversion rule appears in `src/kit/skills/_cli-fab-operator.md`, with test updates alongside
- [ ] A-028 Migration for user-data restructuring: the state-file change ships `src/kit/migrations/2.25.1-to-2.26.0.md`, not an ad-hoc script
- [ ] A-029 Pattern consistency: new code follows naming and structural patterns of surrounding code
- [ ] A-030 No unnecessary duplication: existing utilities reused where applicable
- [ ] A-031: `gofmt -l ./src/go/fab` is empty; `go test ./...` passes from `src/go/fab`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Injectable seams: batched pid map and `GetPanePID` use package-level vars per the `tickSnapshotRows`/`rkPanesRunner` precedent; branch resolution gets its own stubbable seam | Intake §7 mandates the pattern; existing tests establish it | S:95 R:90 A:90 D:90 |
| 2 | Confident | T015/T016 split `fab-operator.md` edits into two sequential tasks on one file (8a–8d, then 8e–8i) to keep each edit session focused; the [P] marker refers to independence from the Go and other kit tasks, not from each other | One file, nine edit sites — ordering within the file avoids stale-context edit failures | S:90 R:85 A:85 D:70 |
| 3 | Certain | The final verification sweep (T020) is its own task after all Go and kit work, running scoped tests then the full suite plus both greps and gofmt | Dispatch prompt §3 makes these binding completion gates | S:95 R:95 A:95 D:95 |
| 4 | Confident | The `fab-change` sweep's zero target applies to `src/kit/skills/**` (the operator-facing deployed set — 38 occurrences). Migrations legitimately name the retired kind: the historical `2.24.9-to-2.25.0.md` stays untouched (intake Assumption 26, user-confirmed), and the new `2.25.1-to-2.26.0.md` must name `fab-change` to document the rename (intake §6b). The dispatch's literal `grep -rn 'fab-change' src/kit/` therefore returns migration-file lines only, reported as such | §8i's "deployed set must read zero" vs. Assumption 26's "historical records stay" vs. §6b's Summary content — irreconcilable read literally; resolved toward the design authority's explicit carve-outs | S:90 R:80 A:85 D:65 |

4 assumptions (2 certain, 2 confident, 0 tentative).
