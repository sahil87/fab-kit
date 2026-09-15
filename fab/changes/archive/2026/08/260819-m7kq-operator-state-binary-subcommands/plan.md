# Plan: Operator State Mutations Behind Binary Subcommands

**Change**: 260819-m7kq-operator-state-binary-subcommands
**Intake**: `intake.md`

## Requirements

### CLI: Shared state IO

#### R1: Typed, tolerant state-file IO helper
The operator state-file IO SHALL be extracted into a shared helper (suggested: `src/go/fab/cmd/fab/operator_state.go`) used by every `fab operator` subcommand, including the existing `tick-start`. It SHALL define typed structs for the four owned sections (`monitored` entries, `autopilot`, `branch_map` values, `watches` entries) matching the schema documented in `fab-operator.md` §4 byte-compatibly. Reads SHALL preserve unknown **top-level** keys (the tick-start posture); mutations SHALL re-marshal the owned section being mutated from its typed struct (so free-form fields inside an owned section cannot be introduced and do not survive a mutation of that section). Writes SHALL go through `atomicfile.WriteFile`; the path SHALL come from `StatePath("")` honoring the `operatorStatePathOverride` test seam. All timestamps are computed by the binary (RFC3339 UTC); no subcommand accepts a timestamp flag.

- **GIVEN** a state file containing an unknown top-level key and an invented free-form field inside a `watches` entry
- **WHEN** any watch mutation subcommand runs
- **THEN** the unknown top-level key survives, the invented in-section field is dropped, and the file is written atomically

#### R2: `fab operator enroll`
`fab operator enroll <change-id> --pane <id> --repo <path> --session <name> --branch <branch> [--stage <stage>] [--agent <state>] [--stop-stage <stage>] [--spawned-by <watch>] [--depends-on <id,...>]` SHALL create (or wholesale-replace) the monitored entry with `enrolled_at` and `last_transition` set to now, AND record the `branch_map` entry `{ branch, repo }`. `--pane`, `--repo`, `--session`, and `--branch` are required. Stage-valued flags SHALL be validated against the six stage names.

- **GIVEN** no monitored entry for `r3m7`
- **WHEN** `fab operator enroll r3m7 --pane %3 --repo /home/u/foo --session work --branch 260819-r3m7-x --stage apply`
- **THEN** the entry exists with binary-set timestamps, defaults (`stop_stage: null`, `spawned_by: null`, `depends_on: []`), and `branch_map.r3m7 == { branch: 260819-r3m7-x, repo: /home/u/foo }`

#### R3: `fab operator update`
`fab operator update <change-id> [--stage <stage>] [--agent <state>] [--stop-stage <stage>]` SHALL mutate only the passed fields of an existing monitored entry, touching `last_transition` if and only if `--stage` changes the stored value. An unknown change-id SHALL exit non-zero with a clear error. `--agent` passes through verbatim (fab is a consumer of run-kit's `@rk_agent_state` convention and does not enumerate its states).

- **GIVEN** entry `r3m7` at `stage: apply`
- **WHEN** `fab operator update r3m7 --agent idle` then `fab operator update r3m7 --stage review`
- **THEN** the first call leaves `last_transition` unchanged and the second updates it

#### R4: `fab operator remove`
`fab operator remove <change-id>` SHALL delete the monitored entry and SHALL retain the `branch_map` entry. Unknown id → non-zero error.

- **GIVEN** enrolled `r3m7`
- **WHEN** `fab operator remove r3m7`
- **THEN** `monitored.r3m7` is gone and `branch_map.r3m7` survives

### CLI: Watch verbs

#### R5: Watch CRUD — `watch add` / `rm` / `toggle` / `update`
`fab operator watch add <name> --source <linear|slack> --target-repo <path> [--query <json>] [--stop-stage <stage>] [--instructions <text>]` SHALL create the watch with `enabled: true`, empty `known`/`completed`, null `last_checked`/`last_error`; `--query` takes a JSON object string stored as the YAML `query` map (invalid JSON → non-zero error). `watch rm <name>` deletes; `watch toggle <name> [--on|--off]` flips (or forces) `enabled`; `watch update <name>` mutates any of `--target-repo`/`--stop-stage`/`--instructions`/`--query`. Duplicate `add` and unknown names on `rm`/`toggle`/`update` → non-zero error.

- **GIVEN** no watch `linear-bugs`
- **WHEN** `fab operator watch add linear-bugs --source linear --target-repo /home/u/foo --query '{"project":"DEV","status":["Backlog","Todo"]}'`
- **THEN** the watch exists with the nested query map and `enabled: true`

#### R6: Watch tick bookkeeping — `watch checked` / `seen` / `complete`
`watch checked <name> [--error <msg>]` SHALL set `last_checked` to now and set `last_error` to the message (or clear it to null when the flag is absent). `watch seen <name> <item-id>` SHALL append to `known` (idempotent — no duplicates) and enforce the 200-entry cap, pruning oldest first, in the binary. `watch complete <name> <item-id>` SHALL move the item from `known` to `completed` (item absent from `known` → still added to `completed`, so a late completion is never lost).

- **GIVEN** a watch whose `known` holds 200 items
- **WHEN** `fab operator watch seen <name> NEW-1`
- **THEN** `known` still holds 200 items, the oldest was pruned, and `NEW-1` is last

### CLI: Autopilot and branch map

#### R7: Autopilot lifecycle verbs
`fab operator autopilot start --queue <id,...>` SHALL set `{ queue, current: <first>, completed: [], state: running }`; `pause`/`resume` SHALL flip `state` between `paused`/`running`; `advance [--skip]` SHALL append `current` to `completed` (unless `--skip`) and promote the next queue entry, setting `current: null, state: null` on exhaustion while retaining `queue`/`completed` for the completion summary; `stop` SHALL clear the whole block to `autopilot: null`. Verbs other than `start` SHALL error non-zero when no queue is active.

- **GIVEN** `autopilot start --queue ab12,cd34`
- **WHEN** `advance` runs twice
- **THEN** after the first: `current: cd34, completed: [ab12]`; after the second: `current: null, state: null, completed: [ab12, cd34]` with `queue` retained

#### R8: `fab operator branch-map rm`
`fab operator branch-map rm <change-id>` SHALL remove one entry; `fab operator branch-map rm --all` SHALL clear the map. Unknown id → non-zero error.

- **GIVEN** `branch_map` entries for `ab12` and `cd34`
- **WHEN** `fab operator branch-map rm ab12`
- **THEN** only `cd34` remains

#### R9: `fab operator state` read verb
`fab operator state` SHALL print the state file verbatim as YAML; `--json` SHALL print the JSON conversion. When the file is missing it SHALL create and persist the empty skeleton (`monitored: {}`, `autopilot: null`, `branch_map: {}`, `watches: {}`) — replacing the skill's hand-managed "create if missing" init step — then print it.

- **GIVEN** no state file for this server
- **WHEN** `fab operator state --json`
- **THEN** the skeleton is persisted to the server-keyed path and printed as JSON

### Kit skills: documentation

#### R10: `_cli-fab.md` § fab operator documents the full verb surface
`src/kit/skills/_cli-fab.md` § fab operator SHALL document every new subcommand (signature + behavior + binary-owned invariants: timestamps, 200-cap pruning, branch_map retention on remove, skeleton-on-missing) alongside `tick-start`/`time`, pointing at (not restating) the shared state-path/atomic-write mechanics already documented under tick-start.

- **GIVEN** the updated `_cli-fab.md`
- **WHEN** an agent looks up any state mutation
- **THEN** it finds the owning verb and its contract, with no state-path prose duplicated

#### R11: `fab-operator.md` state-touching sections route through the verbs
`src/kit/skills/fab-operator.md` SHALL be updated so no instruction tells the model to write the state file: §2 Init reads via `fab operator state`; §4 gains the doctrine line (the file is mutated only through `fab operator` subcommands — the operator never hand-writes it, mirroring `fab score`) and its schema block stays as reference of what the binary maintains; §4 enrollment/removal name `enroll`/`remove` (window-prefix `fab pane window-name` calls unchanged); §4 Tick Behavior step 6 "Persist" is removed/reworded (persistence rides each verb; per-tick observed changes ride `update`); §6 spawn step 7 and Autopilot name `enroll` and the autopilot verbs; §7 names `checked`/`seen`/`complete` and maps the Conversational Management utterances to `add`/`rm`/`toggle`/`update`; §9 Key Properties updated. A repo-wide sweep SHALL catch stale hand-write claims in the sibling class (aggregate specs `skills.md`/`glossary.md`/`architecture.md`, other `src/kit/skills/*.md`).

- **GIVEN** the updated skill sources
- **WHEN** grepping `src/kit/ docs/specs/` for instructions to write/persist the operator state file directly
- **THEN** every state-touching site names a `fab operator` verb (or points at `_cli-fab.md`), and no section instructs a whole-file write

### Non-Goals

- Binary-rendered status frame / per-provider loop dictionary — the unshipped sibling change; frame emission stays model-side
- Provider-neutrality prose sweep of fab-operator.md — backlog `[tdqd]`
- Schema changes or a migration — the file shape is byte-compatible; machine-local XDG state
- Locking/flock — the operator is the single writer (one loop per server); tick-start's lock-free atomic-write posture is kept

### Design Decisions

#### Full mediation of state writes
**Decision**: Every state-file mutation — including per-tick observed-field updates and watch bookkeeping — goes through a `fab operator` verb; the agent never writes the YAML.
**Why**: "Schema cannot drift" holds only if no hand-write path remains; the motivating incident was a tick-time watch write. "Run a command with flags" is the instruction class weak models follow reliably; whole-file YAML regeneration is the class they fail.
**Rejected**: Mediating only structural ops (leaves the highest-frequency writes hand-edited); a post-hoc validator (`lint` detects, doesn't prevent); schema-validated whole-file `write` on stdin (still makes weak models compose YAML; rejection loops stall the loop).
*Introduced by*: 260819-m7kq-operator-state-binary-subcommands

#### Tolerant-read / typed-write IO posture
**Decision**: Unknown top-level keys are preserved on read-modify-write (tick-start precedent); the four owned sections are re-marshaled from typed structs on mutation, scrubbing in-section drift.
**Why**: Strict decode would wedge the operator on any pre-existing hand-drifted file; tolerant top-level + typed sections prevents new drift while degrading gracefully on old files.
**Rejected**: Strict `KnownFields` decode (turns legacy drift into a hard loop failure).
*Introduced by*: 260819-m7kq-operator-state-binary-subcommands

#### Window markers stay separate primitives
**Decision**: `enroll`/`remove` touch only the state file; the `»`/`›` renames remain the operator's separate `fab pane window-name` calls.
**Why**: Composable primitives match the existing pattern; rename-failure handling (log-and-continue) is operator policy, and enrollment durability must not depend on tmux reachability.
**Rejected**: Folding the rename into the verbs (couples a state write to a tmux side-effect with different failure semantics).
*Introduced by*: 260819-m7kq-operator-state-binary-subcommands

## Tasks

### Phase 1: Setup

- [x] T001 Extract shared operator state IO into `src/go/fab/cmd/fab/operator_state.go`: typed structs for monitored/autopilot/branch_map/watches, `loadOperatorState`/`saveOperatorState` (tolerant top-level, atomicfile, `StatePath("")` + `operatorStatePathOverride`), binary-computed timestamps; rewire `operator_tick_start.go` onto it with unchanged behavior/output <!-- R1 -->

### Phase 2: Core Implementation

- [x] T002 [P] `fab operator enroll` + `update` + `remove` in `src/go/fab/cmd/fab/operator_monitored.go` (incl. branch_map write on enroll, retention on remove, last_transition-on-stage-change, stage-name validation, non-zero unknown-id errors); register in `operatorCmd()` <!-- R2 -->
- [x] T003 [P] `fab operator watch add/rm/toggle/update` in `src/go/fab/cmd/fab/operator_watch.go` (JSON `--query` parsing, duplicate/unknown-name errors); register <!-- R5 -->
- [x] T004 `fab operator watch checked/seen/complete` in `operator_watch.go` (200-cap oldest-first pruning, idempotent seen, known→completed move) <!-- R6 -->
- [x] T005 [P] `fab operator autopilot start/pause/resume/advance/stop` in `src/go/fab/cmd/fab/operator_autopilot.go` (exhaustion semantics per R7, no-active-queue errors); register <!-- R7 -->
- [x] T006 [P] `fab operator branch-map rm <id>|--all` and `fab operator state [--json]` (skeleton-on-missing persist) in `operator_state.go` or siblings; register <!-- R9 -->

### Phase 3: Integration & Edge Cases

- [x] T007 [P] Tests for shared IO + monitored family in `src/go/fab/cmd/fab/operator_monitored_test.go`: enroll defaults/timestamps/branch_map, wholesale re-enroll, update last_transition rule, remove retention, unknown-top-level-key preservation, in-section drift scrub <!-- R2 -->
- [x] T008 [P] Tests for watch family (`operator_watch_test.go`): CRUD round-trip, toggle forms, query JSON (valid nested + invalid → error), checked set/clear, seen cap-200 pruning + idempotence, complete move + absent-from-known case <!-- R6 -->
- [x] T009 [P] Tests for autopilot + branch-map + state (`operator_autopilot_test.go` etc.): start/pause/resume/advance/--skip/exhaustion-retains-completed/stop-clears, rm one/--all, state skeleton persist + --json <!-- R7 -->
- [x] T010 Run `go test ./src/go/fab/cmd/fab/` (widen to `./src/go/...` if shared packages were touched) and fix failures <!-- R1 -->

### Phase 4: Polish

- [x] T011 Document the full verb surface in `src/kit/skills/_cli-fab.md` § fab operator (signatures, invariants, pointer to shared mechanics under tick-start) <!-- R10 -->
- [x] T012 Rewrite `src/kit/skills/fab-operator.md` state-touching sections (§2 Init, §4 doctrine line + enrollment/removal + tick step 6, §6 step 7 + autopilot, §7 bookkeeping + conversational table, §9) to name the verbs <!-- R11 -->
- [x] T013 Sibling sweep: grep `src/kit/ docs/specs/` for stale "write/persist the operator state file" claims (incl. `skills.md`, `glossary.md`, `architecture.md`, `_cli-agents.md`, `_cli-external.md`) and route each through a verb name or `_cli-fab.md` pointer <!-- R11 -->

## Execution Order

- T001 blocks all of Phase 2 (shared IO)
- T004 depends on T003 (same file, watch structs)
- Phase 3 depends on Phase 2; T007–T009 are mutually parallel
- T012/T013 after T011 (skill prose points at the documented contract)

## Acceptance

### Functional Completeness

- [x] A-001 R1: Shared typed IO helper exists; tick-start rewired with byte-identical stdout and unchanged field behavior
- [x] A-002 R2: `enroll` creates entry + branch_map with binary timestamps and documented defaults
- [x] A-003 R3: `update` mutates only passed fields; `last_transition` touched only on stage change
- [x] A-004 R4: `remove` deletes entry, retains branch_map
- [x] A-005 R5: watch `add`/`rm`/`toggle`/`update` round-trip with validation errors on duplicate/unknown/invalid-JSON
- [x] A-006 R6: `checked`/`seen`/`complete` implement set/clear, cap-200 oldest-first, known→completed move
- [x] A-007 R7: autopilot verbs implement the documented lifecycle incl. exhaustion and stop semantics
- [x] A-008 R8: `branch-map rm` removes one entry or all
- [x] A-009 R9: `state` prints YAML/JSON and persists the skeleton when missing

### Behavioral Correctness

- [x] A-010 R1: A mutation preserves unknown top-level keys and scrubs invented fields from the mutated owned section
- [x] A-011 R7: `advance` on exhaustion retains `queue`/`completed` (summary still readable); only `stop` clears the block

### Scenario Coverage

- [x] A-012 R2: Monitored-family tests cover the R2–R4 scenarios and pass
- [x] A-013 R6: Watch-family tests cover the R5–R6 scenarios (incl. cap pruning) and pass
- [x] A-014 R7: Autopilot/branch-map/state tests cover the R7–R9 scenarios and pass

### Edge Cases & Error Handling

- [x] A-015 R3: Unknown change-id / watch name / no-active-queue paths exit non-zero with clear one-line errors (fail-loud; the operator's log-and-continue policy stays skill-side)

### Code Quality

- [x] A-016 Pattern consistency: new files follow the existing `operator_*.go` cobra layout; no new packages; `atomicfile` reused, no duplicate IO paths
- [x] A-017 CLI ⇒ docs + tests: every new/changed signature documented in `_cli-fab.md` and covered by tests (constitution Additional Constraints)
- [x] A-018 Owner-or-pointer: `fab-operator.md` names verbs and points at `_cli-fab.md` for contracts — no flag tables restated in the skill
- [x] A-019 Canonical source only: skill edits land in `src/kit/skills/`, never `.claude/skills/`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant (tick-start's inline IO was folded into the shared `operator_state.go` helper, not left behind)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Re-enroll replaces the entry wholesale with a fresh `enrolled_at` | Skill documents re-enrollment after transient removal; a fresh timestamp is the honest reading and simplest contract | S:55 R:80 A:70 D:65 |
| 2 | Confident | `--agent` values pass through verbatim, no enum | fab is a consumer of run-kit's `@rk_agent_state` convention; enumerating it here would couple fab to rk's evolution (provider-neutral pass-through doctrine) | S:60 R:80 A:80 D:70 |
| 3 | Confident | Exhausted autopilot keeps `queue`/`completed` with `current: null, state: null`; only `stop` clears to `autopilot: null` | The queue-completion summary reads `completed` after the last change finishes; clearing on exhaustion would erase it before the summary renders | S:55 R:75 A:75 D:65 |
| 4 | Confident | `state` persists the skeleton on missing (write-on-first-read) | Matches §2 Init's documented "created if missing" behavior it replaces; keeps every later mutation free of missing-file branches | S:60 R:80 A:75 D:70 |
| 5 | Confident | Stage-valued flags validated against the six stage names; `--source` against `linear\|slack` | Validation is the change's purpose; both vocabularies are closed sets owned by fab | S:65 R:80 A:85 D:75 |
| 6 | Certain | Verbs live under the existing `operatorCmd()` in per-family `operator_*.go` files | Direct extension of the established file/registration pattern (`tick-start`, `time`) | S:80 R:85 A:90 D:85 |
| 7 | Confident | Mid-flight `depends_on` changes ride re-enroll (`enroll` replaces wholesale), not a new `update --depends-on` flag | R3 pins `update` to per-tick observed fields (stage/agent/stop-stage); `depends_on` is spawn-time data normally passed at enrollment, and §6 Dependency Declaration's explicit path is rare mid-flight. Keeps the shipped verb surface exactly as planned; a dedicated flag can be added later without breaking anything | S:55 R:75 A:75 D:60 |

6 assumptions (1 certain, 5 confident, 0 tentative).
