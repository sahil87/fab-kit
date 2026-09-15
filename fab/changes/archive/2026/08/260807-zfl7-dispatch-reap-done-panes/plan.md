# Plan: fab dispatch pane-mode done-worker reaping

**Change**: 260807-zfl7-dispatch-reap-done-panes
**Intake**: `intake.md`

## Requirements

### Dispatch runtime: the reap decision

#### R1: Pure reap guard in `internal/dispatch`
The package SHALL expose a **pure, table-testable** decision function (no I/O, matching the
`SelectMode` / `SelectPaneShape` / `DerivePaneState` precedent) that maps the three guard inputs —
whether the record is pane-mode, the derived dispatch state, and the resolved `dispatch.reap_done`
knob — onto a verdict naming either "reap" or the single reason the reap is skipped. Every skip
reason MUST be a named constant, so the command layer never composes a reason string from raw
booleans.

- **GIVEN** a pane-mode record whose derived state is `done` and `reap_done` resolves `true`
- **WHEN** the guard is evaluated
- **THEN** the verdict is "reap"
- **AND** for a headless record, a non-`done` state, or `reap_done` `false`, the verdict is the
  corresponding named skip reason

#### R2: `fab dispatch reap <change> <stage>` subcommand
The `fab dispatch` family SHALL grow from seven subcommands to **eight** — `start`, `restart`,
`status`, `wait`, `logs`, `kill`, **`reap`**, `clean` — with `reap` registered on the `dispatch.go`
parent and its cobra wiring in `cmd/fab/dispatch_reap.go`. It takes exactly two positional
arguments and **no flags** (no `--json`, no `--server`: the socket comes from the record).
It SHALL kill the worker pane via `KillPane(rec.Pane, rec.Server)` only when the R1 guard returns
"reap", and otherwise perform a **no-op with a one-line report naming the reason**, exiting 0.

- **GIVEN** a pane dispatch whose `{stage}-result.yaml` is present and `dispatch.reap_done` unset
- **WHEN** `fab dispatch reap <change> <stage>` runs
- **THEN** the worker's tmux pane is killed on the record's own socket and the command prints a
  success line naming the pane, exiting 0

#### R3: Reap is not kill — no-op shapes and exit codes
`reap` MUST never terminate a `running`, `orphaned`, `failed`, or `failed (no-result)` dispatch, and
MUST NOT touch a headless dispatch. Every no-op SHALL exit **0** with a one-line reason; only **real
errors** — no dispatch record for the pair, or an unresolvable change — exit non-zero, reusing the
family's existing no-dispatch message surface. An already-gone pane SHALL be a benign
already-gone report (mirroring `kill`'s idempotence).

- **GIVEN** a pane dispatch that is still `running`
- **WHEN** `reap` runs
- **THEN** nothing is killed, the report names the state, and the exit code is 0
- **AND** the same holds for a headless record, for `reap_done: false`, and for a pane already gone

#### R4: Reap changes no state and cleans no state files
`reap` SHALL kill the pane **only**. The dispatch record (`{stage}.yaml`), the result file
(`{stage}-result.yaml`), the prompt file, and the log MUST remain untouched, so a reaped dispatch
still derives `done` forever (`DerivePaneState` gives result presence precedence over pane
liveness). No new dispatch state string, no record-schema field, and no migration are introduced.

- **GIVEN** a reaped `done` pane dispatch
- **WHEN** `fab dispatch status <change> <stage>` runs afterwards
- **THEN** it still prints `done`, and `.fab-dispatch/{id}/` still holds every file it held before

#### R5: Reaping is shape-blind and leaves siblings intact
Because `KillPane` is pane-ID keyed, reap SHALL cover both pane shapes with no shape branch:
killing a **split**-shape worker's pane leaves the dispatching agent's pane, its window, and any
sibling worker pane intact; killing the only pane of a **new-window**-shape worker takes the window
with it (plain tmux semantics).

- **GIVEN** a dispatcher pane with two stacked worker panes, one of them `done`
- **WHEN** the `done` worker is reaped
- **THEN** only that worker's pane disappears; the dispatcher pane and the sibling worker survive

### Configuration: the `dispatch.reap_done` knob

#### R6: Config struct field and accessor
`internal/config.DispatchConfig` SHALL gain `ReapDone *bool` (`yaml:"reap_done"`) — a **pointer**,
not a plain bool, because a default-**true** bool cannot ride the Go zero value the way the
default-false `Watchable` does and an explicit `false` MUST be distinguishable from absent. The
canonical default SHALL live in one exported Go symbol, and a nil-receiver-safe accessor
`GetDispatchReapDone() bool` SHALL return that default when the config or the field is nil and the
pointed value otherwise, following the `GetDispatchWatchable`/`GetDispatchColumnWidth` pattern.

- **GIVEN** a config with no `dispatch:` block, or a `dispatch:` block with no `reap_done` key
- **WHEN** `GetDispatchReapDone()` is called (including on a nil `*Config`)
- **THEN** it returns `true`
- **AND** an explicit `reap_done: false` returns `false`, including when it arrives from the system
  layer `~/.fab-kit/config.yaml` (scope `both`, not pruned)

#### R7: Registry row under the shared `dispatch:` segment
`internal/configref`'s ordered `[]Field` registry SHALL gain a `dispatch.reap_done` row
(`Default` sourced from the canonical Go symbol, `Scope: both`, `Advertise: true`, non-empty
`Description`), placed with the other `dispatch` rows. Per the "several registry rows under one
YAML block share a single segment" rule the row's own `Segment` MUST be **empty**; the existing
`dispatch.watchable` segment SHALL be extended to document and scaffold `reap_done`, so the fence
still renders exactly **one** commented `# dispatch:` parent. No `internal/configscope` change is
needed (the `dispatch` parent is already `ScopeBoth`) and no migration is needed (additive field
with a built-in default under the presence=intent fence model).

- **GIVEN** an un-overridden project config
- **WHEN** `fab config upgrade` regenerates the managed fence
- **THEN** the fence documents `# dispatch.reap_done` and scaffolds `#   reap_done: true` inside the
  single commented `# dispatch:` block
- **AND** `fab config reference --json` carries a `dispatch.reap_done` row with a non-null `default`
  and the JSON/YAML key-parity guard still passes

### Skill wiring and documentation

#### R8: `_preamble.md` § CLI-Adapter Dispatch wiring
`src/kit/skills/_preamble.md` SHALL wire the call **unconditionally and dumbly** into the CLI-adapter
procedure's step 3 `done` bullet: after reading `{stage}-result.yaml`, run
`fab dispatch reap <change> <stage>` — no mode check, no config check in the skill, because the
headless no-op and the knob/state guards live in Go where the config cascade is readable. Step 4's
"No cleanup after `done`" claim MUST be **narrowed, not deleted**, so pane hygiene and
`.fab-dispatch/` state cleanup cannot read as contradicting. The Recovery-policy verb-set sentence
SHALL gain **reap** with the reap-is-not-kill distinction stated. The pane-mode subsection's `done`
handling MUST still read correctly given the inherited call.

- **GIVEN** an orchestrator following § CLI-Adapter Dispatch on a CLI dispatch that reached `done`
- **WHEN** it reads the result file
- **THEN** the documented next action is `fab dispatch reap <change> <stage>`, with the state-cleanup
  posture (archive-time deletion + explicit `fab dispatch clean`, no auto-GC) explicitly unchanged

#### R9: `_cli-fab.md` § fab dispatch reference
`src/kit/skills/_cli-fab.md` SHALL update the family enumeration to
`<start|restart|status|wait|logs|kill|reap|clean>`, add a `### reap` subsection documenting the
three-condition guard, the no-op table, idempotence and exit codes, and cross-reference `reap` from
`### kill` so the recovery-verb/hygiene-verb distinction is explicit at both sites. The
`fab resolve-agent` config-surface notes SHALL name the new key where they enumerate `dispatch.*`.

- **GIVEN** an agent looking up the dispatch family in `_cli-fab.md`
- **WHEN** it reads § fab dispatch
- **THEN** eight subcommands are enumerated and `reap`'s guard, no-op table, and exit contract are
  documented

#### R10: SPEC mirrors updated in the same change
Per the constitution's Additional Constraints, `docs/specs/skills/SPEC-_preamble.md` and
`docs/specs/skills/SPEC-_cli-fab.md` SHALL be updated in this change to mirror the two skill edits.

- **GIVEN** the two `src/kit/skills/*.md` edits
- **WHEN** the change is reviewed
- **THEN** both SPEC mirrors describe the reap wiring and the eighth subcommand

#### R11: `harness-adapters.md` contract statements
`docs/specs/harness-adapters.md` SHALL record that a pane worker's lifecycle may end in an
**optional, orchestrator-invoked reap** (§ 3 Interactive-pane adapter), add `reap` to the pipeline
verb line in § Recovery is orchestrator policy over these states with the reap-is-not-kill
distinction, and narrow any cleanup/no-GC statement that would otherwise over-claim — while keeping
the two deterministic `.fab-dispatch/` cleanup moments and the no-auto-GC posture unchanged.

- **GIVEN** a reader of the cross-adapter contract
- **WHEN** they read the pane adapter and the recovery/cleanup sections
- **THEN** reap is described as pane hygiene composed over the existing states, adding no state,
  no protocol surface, and no automatic state-dir GC

#### R12: Config-surface specs and sibling enumerations
`docs/specs/config.md` SHALL update the non-null-default enumeration and the "two `dispatch` rows
are the convention's boundary cases" claim to **three**, the `both` scope list, and the advertise
list. `docs/specs/glossary.md` SHALL gain a `dispatch.reap_done` entry beside its existing
`dispatch.watchable` / `dispatch.column_width` entries, and `docs/specs/architecture.md`'s
`config.yaml` shape block SHALL show the third key. `docs/site/skill.md` (the canonical skill
bundle) SHALL enumerate the eighth subcommand and be re-synced into the embedded copy so the
drift-guard test passes.

- **GIVEN** the sibling/mirror sweep discipline
- **WHEN** the change lands
- **THEN** every enumeration of the `dispatch.*` config keys or the `fab dispatch` subcommand family
  in `docs/` and `src/` names `reap_done` / `reap`

#### R13: Tests ship in the same change
Per the constitution (Test Integrity) and `code-quality.md` (test-alongside), every Go change SHALL
ship its tests: the pure guard's mode × state × knob table, the cobra wiring's error and no-op
reports and exit codes, a live-tmux integration test following the established private-socket
discipline (verified socket, hard-fail if `$TMUX` is set while the server is created), the config
accessor's nil/unset/true/false/system-layer cases, and the registry/fence guards.

- **GIVEN** the change's Go surface
- **WHEN** `go test ./...` runs in `src/go/fab`
- **THEN** the new tests pass alongside the existing suite, with the tmux integration test skipping
  cleanly where tmux is unavailable

### Non-Goals

- **Part 2 of backlog [zfl7]** (richer pane names/colors for stage workers) — explicitly split out
  and still deferred; the backlog entry is amended at ship, not here.
- **Zoom / shrink-in-place / break-pane parking** — rejected for v1; parking either keeps a stub in
  the column or re-clutters the window list the two-tier hierarchy just cleaned up.
- **Any change to the Recovery policy** (peek-on-suspicion, the single restart budget, escalation,
  classification (c) never-kill) — untouched.
- **Any change to `.fab-dispatch/` cleanup** — archive-time deletion + explicit `fab dispatch clean`
  remain the only two cleanup moments; no automatic GC is introduced.
- **Any resolver change** — `fab resolve-agent`'s output is byte-identical.
- **Any migration** — the field is additive with a built-in default under the presence=intent fence.

### Design Decisions

#### The whole guard lives in Go, the skill wiring stays dumb
**Decision**: `fab dispatch reap` owns all three guard conditions (pane-mode record, derived state
`done`, `dispatch.reap_done` true); the skill calls it unconditionally after reading a `done` result.
**Why**: the three-layer config cascade (project > system `~/.fab-kit/config.yaml` > defaults) is only
readable from Go — a skill reading `fab/project/config.yaml` directly would miss the system layer,
which is exactly where a machine-wide `both`-scope preference lives. A dumb call site also keeps the
wiring identical across adapters and modes.
**Rejected**: a skill-side conditional (`if pane && done && knob`) — three chances to drift from the
runtime, and a wrong config read on the layer that matters most.
*Introduced by*: 260807-zfl7-dispatch-reap-done-panes

#### `ReapDone` is `*bool`, not `bool`
**Decision**: model the field as a pointer with `nil` = unset = `true`.
**Why**: the default is **true**, so the Go zero value means the opposite of the default — a plain
`bool` would make an absent key and an explicit `reap_done: false` indistinguishable, silently
disabling reaping for every project that never sets the key.
**Rejected**: a plain `bool` with an inverted key name (`keep_done_panes`) — it would fit the zero
value but advertise the non-default posture as the key's subject, and diverge from its two siblings.
*Introduced by*: 260807-zfl7-dispatch-reap-done-panes

#### Reap is a distinct verb from kill
**Decision**: add `reap` rather than teaching `kill` a `--if-done` flag.
**Why**: `kill` is a **recovery** verb valid in any state, already in the pipeline's sanctioned verb
set with a documented never-kill-a-live-worker-awaiting-input rule; `reap` is **hygiene**, fires only
on `done`, and is policy-gated. Folding them would put a config-gated no-op inside a verb whose
contract is "terminate this now".
**Rejected**: `kill --if-done` — one verb with two contracts, and a config knob silently modulating a
recovery command.
*Introduced by*: 260807-zfl7-dispatch-reap-done-panes

## Tasks

### Phase 1: Setup

- [x] T001 Add `ReapDone *bool` to `DispatchConfig`, the canonical `DefaultDispatchReapDone = true` symbol, and the nil-safe `GetDispatchReapDone()` accessor in `src/go/fab/internal/config/config.go` (extend the `DispatchConfig` doc comment with the pointer rationale) <!-- R6 -->
- [x] T002 Add the `dispatch.reap_done` registry row (empty `Segment`, `Default` from the config symbol, `ScopeBoth`, `Advertise: true`) and extend `dispatchSegment()` to document + scaffold `reap_done` in `src/go/fab/internal/configref/configref.go`; update the package doc's boundary-case note from two dispatch rows to three <!-- R7 -->

### Phase 2: Core Implementation

- [x] T003 Add the pure reap guard (`ReapVerdict` constants + decision function) in a new `src/go/fab/internal/dispatch/reap.go`, composing nothing but its three inputs <!-- R1 -->
- [x] T004 Add `cmd/fab/dispatch_reap.go` — cobra wiring that resolves the dispatch dir, loads the record via the shared `loadDispatchRecord`, derives the pane state, resolves `dispatch.reap_done` through `config.Load`, applies the guard, and either calls `dispatch.KillPane` or prints the named no-op reason (all exit 0) <!-- R2 R3 R4 R5 -->
- [x] T005 Register `dispatchReapCmd()` on the parent in `src/go/fab/cmd/fab/dispatch.go` and update its `Short`/`Long`/doc-comment enumerations from seven to eight subcommands <!-- R2 -->

### Phase 3: Integration & Edge Cases

- [x] T006 [P] Add `GetDispatchReapDone` tests to `src/go/fab/internal/config/config_test.go` — nil config, absent `dispatch:` block, absent key, explicit `true`, explicit `false`, and a system-layer `reap_done` honored through the cascade <!-- R6 R13 -->
- [x] T007 [P] Add the guard's table test in `src/go/fab/internal/dispatch/reap_test.go` covering the full mode × state × knob matrix (including every non-`done` pane state and every headless state) <!-- R1 R13 -->
- [x] T008 Add `src/go/fab/cmd/fab/dispatch_reap_test.go` — missing-record error (non-zero exit, family message surface), headless no-op, non-`done` pane no-op, knob-`false` no-op, already-gone pane no-op, and a live-tmux integration case (private verified socket) that reaps a `done` split worker and asserts the dispatcher + sibling panes survive, `status` still reads `done`, and `.fab-dispatch/` files are untouched <!-- R2 R3 R4 R5 R13 -->
- [x] T009 [P] Update `src/go/fab/cmd/fab/config_test.go` — add `dispatch.reap_done` to the non-null-default map, to the scope-assignment map, and add a registry-row test asserting default/scope/advertise/empty-Segment and the shared segment's third key <!-- R7 R13 -->
- [x] T010 [P] Add `TestRender_FenceAdvertisesDispatchReapDone` to `src/go/fab/internal/configupgrade/configupgrade_test.go`, asserting the scaffold line, the still-exactly-one `# dispatch:` parent, and the overridden-block suppression <!-- R7 R13 -->

### Phase 4: Documentation Sweep

- [x] T011 Update `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch — the step-3 `done` bullet gains the unconditional `fab dispatch reap` call, step 4's cleanup claim is narrowed, the Recovery-policy verb set gains `reap` with the reap-is-not-kill distinction, and the pane-mode subsection's `done` handling is verified to still read correctly <!-- R8 -->
- [x] T012 Update `src/kit/skills/_cli-fab.md` § fab dispatch — family enumeration to eight, a new `### reap` subsection (guard, no-op table, exit codes, idempotence, no flags), a kill↔reap cross-reference in `### kill`, and the `dispatch.*` key enumeration in the `fab resolve-agent` notes <!-- R9 -->
- [x] T013 [P] Update the SPEC mirrors `docs/specs/skills/SPEC-_preamble.md` and `docs/specs/skills/SPEC-_cli-fab.md` for the two skill edits <!-- R10 -->
- [x] T014 [P] Update `docs/specs/harness-adapters.md` — § 3 pane adapter's done-worker lifecycle, the § Recovery pipeline-verb line, and the § Cleanup narrowing <!-- R11 -->
- [x] T015 [P] Update `docs/specs/config.md` (non-null-default enumeration, three-boundary-case claim, `both` scope list, advertise list), `docs/specs/glossary.md` (new `dispatch.reap_done` entry beside its siblings), and `docs/specs/architecture.md` (the `dispatch:` config-shape block) <!-- R12 -->
- [x] T016 [P] Update the canonical `docs/site/skill.md` dispatch line to eight subcommands and run `scripts/sync-skill.sh` so the embedded `src/go/fab/cmd/fab/skill.md` matches (drift-guard test) <!-- R12 -->
- [x] T017 Aggregate-spec + sibling sweep: grep the repo for every `fab dispatch` subcommand-family enumeration and every `dispatch.*` config-key enumeration outside `fab/changes/` and `docs/memory/` (hydrate owns memory) and confirm each has been updated <!-- R12 -->
- [x] T018 Run the affected Go packages' tests (`internal/config`, `internal/configref`, `internal/configupgrade`, `internal/dispatch`, `cmd/fab`), then the full `go test ./...` in `src/go/fab` <!-- R13 -->

## Execution Order

- T001 blocks T002 (the registry row sources its default from the config symbol) and T004
- T003 blocks T004; T004 blocks T005 and T008
- T006–T010 depend on their respective Phase 1–2 tasks but are independent of each other
- T011–T016 are independent of each other; T017 runs after all of them
- T018 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `internal/dispatch` exposes a pure, I/O-free reap guard over (pane-mode, state, knob) with named skip-reason constants
- [x] A-002 R2: `fab dispatch reap <change> <stage>` exists, is registered on the `dispatch` parent, takes two args and no flags, and kills the recorded pane on the record's own socket when all three conditions hold
- [x] A-003 R3: every no-op shape (headless, non-`done`, knob false, pane already gone) exits 0 with a one-line reason; only a missing record or unresolvable change exits non-zero
- [x] A-004 R4: reap kills the pane only — record, result, prompt, and log files survive, and `status` still reports `done` afterwards
- [x] A-005 R5: reaping a split-shape worker leaves the dispatcher's pane, its window, and sibling worker panes intact
- [x] A-006 R6: `DispatchConfig.ReapDone` is `*bool` with a nil-safe accessor returning `true` when unset
- [x] A-007 R7: the registry carries a `dispatch.reap_done` row with an empty `Segment`, and the shared `dispatch.watchable` segment documents and scaffolds all three keys under one commented `# dispatch:` parent
- [x] A-008 R8: `_preamble.md` § CLI-Adapter Dispatch wires the unconditional reap call into the step-3 `done` bullet
- [x] A-009 R9: `_cli-fab.md` § fab dispatch enumerates eight subcommands and documents `### reap`
- [x] A-010 R10: both SPEC mirrors are updated in this change
- [x] A-011 R11: `harness-adapters.md` records reap in the pane-adapter lifecycle and the pipeline verb line
- [x] A-012 R12: `config.md`, `glossary.md`, `architecture.md`, and the synced skill bundle all name the new key/subcommand

### Behavioral Correctness

- [x] A-013 R4: a reaped dispatch's derived state is still `done` because `DerivePaneState` gives result presence precedence over pane liveness — no state machine changed, no record-schema field added, no migration shipped
- [x] A-014 R6: an explicit `reap_done: false` is distinguishable from an absent key, including when it arrives from the system layer (`scope: both`, not pruned)
- [x] A-015 R8: step 4's `.fab-dispatch/` cleanup posture is **narrowed, not deleted** — the two deterministic cleanup moments and the no-auto-GC statement still hold and cannot be read as contradicting the reap call
- [x] A-016 R3: the Recovery policy is untouched — reap never fires on `running`/`orphaned`/`failed`/`failed (no-result)`, and peek-on-suspicion, the single restart budget, escalation, and classification (c) are unchanged

### Scenario Coverage

- [x] A-017 R13: the guard's mode × state × knob matrix is table-tested, including every non-`done` pane state and the headless states
- [x] A-018 R13: the cobra wiring's error path, all four no-op shapes, and the success path are covered by tests asserting both output and exit code
- [x] A-019 R13: a live-tmux integration test reaps a `done` pane worker under the established private-socket discipline and asserts sibling/dispatcher survival plus post-reap `status`
- [x] A-020 R13: the config accessor's nil/absent-block/absent-key/true/false/system-layer cases are covered
- [x] A-021 R7: the registry lint, the `--json` key-parity guard, and the fence-advertisement guard all pass with the seventeenth row present

### Edge Cases & Error Handling

- [x] A-022 R3: reaping a pane that was already killed by hand (or whose tmux server died) is a benign already-gone report, mirroring `kill`'s idempotence
- [x] A-023 R2: a `--server`-started dispatch is reaped on the right socket with no flag, because the record's persisted `server` is used
- [x] A-024 R3: `reap` on a change/stage pair with no dispatch record exits non-zero with the family's actionable message

### Code Quality

- [x] A-025 Pattern consistency: the new guard follows the pure-function precedent (`SelectMode`/`DerivePaneState`) and the cobra wiring follows `dispatch_kill.go`'s shape, reusing `resolveDispatchDir`/`loadDispatchRecord`/`KillPane` rather than reimplementing them <!-- review: the three named helpers are reused; a separate should-fix covers the inlined copy of observeDispatch's pane-state derivation -->
- [x] A-026 No unnecessary duplication: the reap default lives in exactly one Go symbol, read by both the accessor and the registry row; the `dispatch:` documentation lives in exactly one shared segment
- [x] A-027 Canonical source only: no file under `.claude/skills/` was edited — the skill changes are in `src/kit/skills/`
- [x] A-028 SPEC-mirror sync: every edited `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` mirror updated in the same change
- [x] A-029 CLI ⇒ docs + tests: the new command signature is documented in `_cli-fab.md` and covered by tests
- [x] A-030 No migration for an additive default: the field is additive under the presence=intent fence model, so no `src/kit/migrations/` file is added (and none is needed)
- [x] A-031 Markdown-only artifacts: all documentation edits are standard CommonMark

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- Memory updates (`runtime/dispatch`, `_shared/configuration`, `_shared/context-loading`,
  `distribution/kit-architecture`) are **hydrate's** job, not apply's — the intake's Affected Memory
  list feeds that stage.
- The backlog [zfl7] amendment (part 1 covered, part 2 residual) is a **ship-time** step.

## Deletion Candidates

- None — this change adds a new verb (`fab dispatch reap`), a new config field, and their
  documentation. It composes existing symbols (`Load`/`IsPane`/`ResultPresent`/`DerivePaneState`/
  `PaneAlive`/`KillPane`) without superseding any of them, and it deliberately leaves `kill` intact
  as the recovery verb (the reap-is-not-kill split is the change's own design decision), so no
  existing file, function, branch, or config key became redundant or unused.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The pure guard lives in a new `internal/dispatch/reap.go` (not appended to `dispatch.go` or `pane_mode.go`) | The package already splits by concern (core/pane/wait); a new verb's decision deserves its own file with its own doc comment, matching `wait.go`'s precedent | S:75 R:90 A:90 D:85 |
| 2 | Confident | Guard evaluation order is mode → state → knob, so the never-kill-a-live-worker property is stated before the policy check | Matches the intake's no-op table order and makes the "reap is NOT kill" invariant independent of config; only the reported reason depends on the order | S:65 R:90 A:85 D:75 |
| 3 | Confident | The command reuses `loadDispatchRecord` (the `status`/`wait` loader, whose message carries the `run \`fab dispatch start\` first` hint) rather than duplicating `kill`'s bare variant | Reuse over duplication; the intake called for mirroring `status`/`kill`'s message surface and the shared loader is the single-sourced one | S:60 R:85 A:85 D:70 |
| 4 | Confident | The live-tmux integration test lives in `cmd/fab/dispatch_reap_test.go`, not `internal/dispatch` | Every existing real-tmux test with the private-socket discipline is in `cmd/fab` (`dispatch_start_test.go`); `internal/dispatch` tests stub tmux. Following the established location keeps the discipline single-sourced | S:80 R:85 A:85 D:80 |
| 5 | Confident | `docs/site/skill.md` (+ `scripts/sync-skill.sh`) and `docs/specs/glossary.md` / `docs/specs/architecture.md` are added to the intake's sweep list | Found by the up-front sibling/mirror grep: both enumerate the dispatch family / the `dispatch.*` config keys, and the skill bundle has a byte-drift guard test that would fail otherwise | S:85 R:80 A:90 D:80 |
| 6 | Certain | `docs/specs/config.md` carries no literal "16 rows" count to update — only enumerations (non-null defaults, the two-boundary-cases claim, the scope and advertise lists) | Verified by grep: the spec describes the registry by enumeration, not by count, so the intake's "16 → 17" note resolves to updating those enumerations | S:90 R:90 A:90 D:90 |
| 7 | Confident | The command reads config via `resolve.FabRoot()` + `config.Load(fabRoot)` in a small local helper, accepting a second `FabRoot` walk alongside `resolveDispatchDir`'s | Keeps `resolveDispatchDir`'s signature (and its three existing call sites) untouched; the walk is a cheap upward directory search and the helper documents why the cascade read must happen in Go | S:55 R:85 A:80 D:70 |

7 assumptions (2 certain, 5 confident, 0 tentative).
