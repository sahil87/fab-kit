# Plan: Gate Delegation to rk mux await --ready

**Change**: 260904-b4j7-gate-delegation-rk-await-ready
**Intake**: `intake.md`

## Requirements

### Runtime: Two-Arm Readiness Classification

#### R1: `Gate.Probe` delegates classification to a sentinel-capable rk, fail-open to the raw-tmux arm
`Gate.Probe` (`src/go/fab/internal/pane/gate.go`) SHALL classify in two arms behind the unchanged takeover precondition. The **takeover precondition runs first and fab-side on every probe**: read `#{pane_current_command}`; while `IsShellCommand` matches, report `booting` (with fab's own capture snippet) and invoke NOTHING — neither the sentinel nor rk. Past the precondition, the **rk arm** runs when a sentinel-capable rk is on PATH (R3): invoke `rk mux await --ready <pane> [-L <server>] --timeout <bounded>` and map its outcome per R2. The **raw-tmux arm** — today's sentinel/echo/stability classifier, byte-identical — runs when rk is absent, not sentinel-capable, or the rk invocation fails unexpectedly (fail-open, one stderr warning per process at most).

- **GIVEN** a pane whose foreground command is still a shell
- **WHEN** `Probe` runs with rk installed
- **THEN** the report is `booting`, nothing is typed, and rk is never invoked
- **AND** GIVEN a non-shell foreground and no `rk` on PATH, the raw-tmux classifier runs exactly as today

#### R2: rk outcome mapping onto the frozen `Readiness` contract
The rk arm SHALL branch on the **first token of rk's stdout report line** and map:

| rk outcome | fab result |
|------------|-----------|
| `ready` (either `(state)` or `(echo)`, exit 0) | `ReadyReady`, empty snippet |
| `parked` (exit 0) | `ReadyParked` + snippet |
| `running` (timeout, exit 0) | `ReadyBooting` + snippet |
| `gone` (exit 1) | an error on the existing dead-pane path (not a classification) |
| any other exit, empty/unparseable stdout | fail-open: fall through to the raw-tmux arm for this probe, stderr warning |

Snippets for non-`ready` results SHALL come from **fab's own `Capture` + `Snippet` path** (one extra capture), NOT from rk's stderr — this keeps the documented `--- last 20 lines ---` output form and the `--json` `snippet` field byte-compatible regardless of rk's stderr formatting; rk's stderr is surfaced only in the fail-open warning. The rk timeout is a bounded unexported constant (mechanics, not policy — the `settleDelay` precedent; no flag, no config field). Sentinel-scope safety is preserved by construction: `fab dispatch ready`/`deliver` refuse mid-stage workers before `Probe` runs, so rk's sentinel is only ever typed into pre-delivery panes.

- **GIVEN** rk reports `parked %5` with a screen snippet on stderr
- **WHEN** the rk arm maps the outcome
- **THEN** `Probe` returns `ReadyParked` with fab's own trailing-20-line capture snippet
- **AND** GIVEN rk exits 1 printing `gone %5`, `Probe` returns an error naming the dead pane
- **AND** GIVEN rk exits 127 or prints an unrecognized first token, the raw-tmux arm classifies this probe and a warning names the rk failure

#### R3: Sentinel-capability probe, cached per process
A helper SHALL answer "does the installed rk have Part B's sentinel `--ready`?" by probing the binary, never the version string: `exec.LookPath("rk")`, then `rk mux await --help` must mention `parked` (present in v3.19.9's `--ready` flag text, absent before Part B). The answer SHALL be computed at most once per process (cached), and the runner/probe seams SHALL be package-level injectable vars for tmux-free, rk-free tests (the `rkPanesRunner` precedent in `pane_map.go`).

- **GIVEN** an rk whose `mux await --help` output lacks the token `parked`
- **WHEN** the capability probe runs
- **THEN** the gate uses the raw-tmux arm for the whole process, probing rk's help at most once

#### R4: Verb contracts and consumers are unchanged
`fab pane ready`, `fab dispatch ready`, `fab pane deliver`, and `fab dispatch deliver` SHALL keep their exact report words (`ready`/`booting`/`parked`), non-`ready` report shape (`pane:`/`server:`/snippet), exit codes (0 for all three classifications; pane-family 2/3; deliver's 1-on-exhaustion), and `--json` shapes. `Gate.Deliver`'s readiness precondition inherits the delegation by calling `Probe` — no separate wiring. The delivery typing choreography itself SHALL NOT delegate to `rk mux send` (Non-Goals). No new flag, config field, or state file is introduced.

- **GIVEN** an rk-equipped machine and an rk-less machine
- **WHEN** `fab pane ready %5 --json` runs on each
- **THEN** both emit the same `{"state","pane","server","snippet"}` shape with the same possible `state` values and exit 0

### Docs: Specs, Kit Skills, Sweeps

#### R5: Documentation reflects the two-arm gate everywhere it is owned
`docs/specs/harness-adapters.md` (§3's readiness-gate bullet and the `fab dispatch ready` row/language) SHALL be amended to the two-arm shape: classification delegates to `rk mux await --ready` when a sentinel-capable rk is on PATH (fail-open, capability-probed), the raw-tmux classifier is the rk-less arm, and the takeover precondition plus the no-dialog-table / answers-nothing / non-`ready`-carries-snippet MUSTs bind both arms. `src/kit/skills/_cli-fab.md` § fab pane ready, § fab dispatch ready, and the § fab dispatch overview gate sentence SHALL document the delegation, the capability probe, and the fail-open rule. `src/kit/skills/_preamble.md` § The pane readiness gate gains at most a one-line rk-delegation note (wiring, budget, judgment rounds unchanged; owner-or-pointer — mechanics stay owned by `_cli-fab.md`/the spec). A repo-wide sibling sweep of the classifier phrase class ("types a sentinel", "echo-and-stability", "FAB-READY-PROBE", "two screen-stability captures") SHALL update every restatement in `src/kit/skills/`, `docs/specs/`, and `docs/memory/runtime/` (the three Affected Memory files' mechanical claims; hydrate does the structured pass).

- **GIVEN** the delegation has landed
- **WHEN** grepping the repo for the classifier phrase class
- **THEN** no doc claims the sentinel/echo classifier is unconditionally fab-typed; each owner describes the two-arm gate and non-owners point at it

#### R6: Tests ship with the change
`gate_test.go`'s existing raw-arm tests SHALL pass unchanged (the fallback must remain byte-identical). New table tests SHALL cover: the R2 mapping (each rk outcome → Readiness/error), each fail-open trigger (rk absent, capability probe negative, non-zero exit, unparseable output), capability-probe parsing (help text with/without `parked`), the takeover precondition short-circuiting rk, and argv construction (`-L` threading, timeout flag). If help strings change, `help_dump_test.go` fixtures SHALL be regenerated. Affected Go packages' tests run green before apply completes.

- **GIVEN** the new delegation code
- **WHEN** `go test ./src/go/fab/internal/pane/... ./src/go/fab/cmd/fab/...` runs
- **THEN** all tests pass, including the untouched raw-arm tables

#### R7: Toolkit standards audit
The change SHALL re-fetch the live shll standards (`shll standards`, then each standard governing CLI surface/help output) and verify the changed help text and CLI behavior against them before ship (constitution § Toolkit Standards; conformance is re-checked live, never trusted from memory).

- **GIVEN** the changed `fab pane ready`/`fab dispatch ready` help text (if any)
- **WHEN** the governing standards are re-read
- **THEN** the surface conforms or is corrected, and the audit outcome is recorded in the apply notes

### Non-Goals

- No delegation of the delivery typing choreography to `rk mux send` (a separate gap; `Gate.Deliver`'s engine is untouched beyond its `Probe` call).
- No new CLI flags, config fields, `.fab-dispatch/` schema fields, or migrations.
- No change to rk itself (the missing takeover guard upstream is a run-kit follow-up idea, out of scope here).
- No blocking-await rewrite of the skill-side gate loop (`_preamble.md` wiring, budgets, 5-consecutive-booting allowance stay as-is).

### Design Decisions

#### Takeover Precondition Stays Fab-Side, Ahead of Both Arms
**Decision**: The `#{pane_current_command}` shell-foreground check runs before either arm; rk is never invoked while a shell owns the pane.
**Why**: run-kit's `inject.AwaitReady` has no takeover guard by design (terminals-are-one-standard: a cooked-shell echo is a valid `ready (echo)` for rk's send use case), but for a dispatch pane that exact signal is the 57mp false-ready. The precondition is also not part of the classification the agent-messaging spec delegates (state-present / sentinel echo / parked).
**Rejected**: Full delegation including the precondition (regresses 57mp); pushing the guard upstream into rk (a run-kit change, out of scope — noted as a follow-up idea).
*Introduced by*: 260904-b4j7-gate-delegation-rk-await-ready

#### Help-Text Capability Probe, Not a Version Compare
**Decision**: Sentinel capability is detected by `rk mux await --help` mentioning `parked`, cached per process.
**Why**: `--ready` predates Part B (capture-settle semantics with the `ready (settled)` false-fire hazard), so presence of the flag is not capability; and a version string can lie (a bottle can predate a same-version source change — verify the binary).
**Rejected**: `rk --version >= 3.19.9` compare (version-skew hazard); probing by invoking `--ready` against a scratch pane (side-effectful, slow).
*Introduced by*: 260904-b4j7-gate-delegation-rk-await-ready

#### Snippets Stay Fab-Captured on the rk Arm
**Decision**: Non-`ready` snippets come from fab's own `Capture`+`Snippet` path on both arms; rk's stderr is used only in fail-open warnings.
**Why**: The `--- last 20 lines ---` text form and the `--json` `snippet` field are published contract; deriving them from rk's stderr would couple fab's output bytes to rk's stderr formatting across rk versions.
**Rejected**: Passing rk's stderr snippet through verbatim (contract coupling); dropping snippets on the rk arm (judgment rounds need the screen).
*Introduced by*: 260904-b4j7-gate-delegation-rk-await-ready

#### Probe Posture Kept: Bounded rk Timeout, `running` → `booting`
**Decision**: The rk call rides an unexported ~20s timeout constant; a timeout return maps to `booting`, so `fab … ready` keeps its probe semantics and the skill-side loop is unrewired.
**Why**: rk's await blocks through boot churn, which lets one probe call absorb most boots; mapping the bound's expiry to `booting` preserves every documented report and the gate wiring's re-probe/allowance rules.
**Rejected**: Blocking indefinitely inside `ready` (changes the verb's latency contract and stalls `deliver`'s precondition); exposing the timeout as a flag/config (mechanics, not policy — the gate-timings precedent).
*Introduced by*: 260904-b4j7-gate-delegation-rk-await-ready

## Tasks

### Phase 1: Setup

- [x] T001 Add the rk delegation seams in a new `src/go/fab/internal/pane/gate_rk.go`: `exec.LookPath` gate, `rk mux await --help` sentinel-capability probe (token `parked`), per-process cache, injectable package-level runner vars (`rkPanesRunner` precedent), and the argv builder (`mux await --ready <pane> [-L <server>] --timeout <secs>`) with the unexported timeout constant <!-- R3 -->

### Phase 2: Core Implementation

- [x] T002 Wire the rk arm into `Gate.Probe` in `src/go/fab/internal/pane/gate.go`: after the unchanged shell-foreground precondition, branch to the rk runner when capable; parse the first stdout token; map per R2 (`ready`→ready, `parked`→parked+own-capture snippet, `running`→booting+own-capture snippet, `gone`→dead-pane error, else fail-open to the raw arm with a stderr warning) <!-- R1 --> <!-- rework: rk `gone` maps to PaneNotFoundError but `runPaneReady` sends every Probe error to exit 3 — route the Probe error branch in cmd/fab/pane_ready.go through paneValidationExitCode(err) so rk-gone exits 2 per the documented dead-pane contract; also gate the `gone` case on runErr != nil (rk contract: gone exits 1) -->
- [x] T003 Unit tests in `src/go/fab/internal/pane/gate_rk_test.go` (+ `gate_test.go` additions): R2 mapping table, each fail-open trigger, capability-probe parsing, precondition-short-circuits-rk, argv construction; confirm existing raw-arm tests pass untouched <!-- R6 --> <!-- rework: add a cmd-layer test pinning rk-gone → exit 2 in `fab pane ready` -->
- [x] T004 Run `go test ./...` under `src/go/fab/` scoped to `internal/pane` and `cmd/fab` first, widen on green; `gofmt` check <!-- R6 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Amend `docs/specs/harness-adapters.md` §3: the readiness-gate bullet and `fab dispatch ready` row to the two-arm shape (delegation MUSTs, fail-open, precondition binds both arms), via the spec's explicit-amendment convention <!-- R5 -->
- [x] T006 Update `src/kit/skills/_cli-fab.md` § fab pane ready, § fab dispatch ready, and the § fab dispatch overview gate sentence; add the one-line note in `src/kit/skills/_preamble.md` § The pane readiness gate <!-- R5 -->
- [x] T007 Update help strings in `src/go/fab/cmd/fab/pane_ready.go` / `dispatch_ready.go` to name the two-arm gate; regenerate `help_dump_test.go` fixtures if text changed <!-- R5 -->
- [x] T008 Sibling sweep: grep the classifier phrase class ("types a sentinel", "echo-and-stability", "FAB-READY-PROBE", "two screen-stability captures", "stability captures") across `src/kit/skills/`, `docs/specs/`, `docs/memory/runtime/` and fix every stale unconditional-classifier claim (mechanical-claim pass on `pane-commands.md`/`dispatch.md`/`agent-primitives.md`; hydrate does the structured pass) <!-- R5 --> <!-- rework: _cli-agents.md:113 both restates the two-arm mechanics AND points at the owner — trim to the shell-foreground context plus the pointer, drop the restated mechanics (owner-or-pointer, A-011) -->

### Phase 4: Polish

- [x] T009 Standards audit: run `shll standards`, read the standards governing CLI surface/help output, verify the changed surface, record the outcome <!-- R7 --> <!-- rework: the audit ran but its outcome is not recorded — add the audit-outcome line under plan.md ## Notes (R7's THEN clause) -->

## Execution Order

- T001 blocks T002; T002 blocks T003–T004
- T005–T008 depend on T002's final shape; T007 depends on T004 (fixtures regenerate against a compiling binary)
- T009 last, after help text settles

## Acceptance

### Functional Completeness

- [x] A-001 R1: `Gate.Probe` runs the rk arm behind the unchanged takeover precondition and the raw-tmux arm otherwise; rk is never invoked while a shell owns the pane
- [x] A-002 R2: every rk outcome maps per the table (ready/parked/running/gone/other) with fab-captured snippets and a bounded internal timeout
- [x] A-003 R3: the capability probe requires `parked` in `rk mux await --help`, caches per process, and is injectable for tests
- [x] A-004 R5: harness-adapters.md, `_cli-fab.md` (both ready sections + overview), and `_preamble.md` describe the two-arm gate; the phrase-class sweep left no stale unconditional-classifier claim

### Behavioral Correctness

- [x] A-005 R4: report words, non-`ready` report shape, exit codes, and `--json` shapes of all four verbs are byte-compatible on both arms; `Gate.Deliver` inherits via `Probe` with no separate wiring

### Scenario Coverage

- [x] A-006 R6: table tests cover the mapping, all fail-open triggers, capability parsing, precondition short-circuit, and argv construction; raw-arm tests pass unmodified
- [x] A-007 R6: affected package tests and gofmt are green; help-dump fixtures match the shipped help text (no golden fixtures exist — `help_dump_test.go` walks a synthetic cobra tree, and `cmd/fab` tests pass against the changed help strings)

### Edge Cases & Error Handling

- [x] A-008 R2: rk `gone` surfaces as the dead-pane error path (never a classification, never fallback); an unexpected rk failure warns on stderr and classifies via the raw arm in the same probe call

### Code Quality

- [x] A-009 Pattern consistency: the delegation follows the `rkPanesRunner` injectable-seam and fail-open precedent; constants unexported per the gate-timings pattern
- [x] A-010 No unnecessary duplication: one classifier home in `internal/pane`; no second echo-verify or capability probe copy
- [x] A-011 Owner-or-pointer: no skill file both states and points at the gate's mechanics; restatements removed, pointers kept
- [x] A-012 CLI ⇒ docs + tests: the Go behavior change ships with `_cli-fab.md` updates and test updates in the same change

### Security

- [x] A-013 R1: no credentials or wall-answering logic added; login walls still escalate (judgment rounds untouched)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- **Standards-audit outcome (R7, apply)**: re-fetched live via `shll standards`; governing standards checked were `help-dump` (the mechanical help contract) and `principles` (the ten toolkit CLI principles). Conformance verified: `fab help-dump` on a fresh build of the changed tree exits 0 with an empty stderr and a conformant envelope (`{tool, version, schema_version, root}`, no `captured_at`, `completion`/`help`/hidden nodes filtered, `version` from the binary); the two reworded Long help texts keep the layered shape principle №3 requires. The delegation itself is the principles' intended shape: capability is probed from rk's advertised `--help` text rather than assumed (№7 compose-don't-reinvent), the raw-tmux arm is fail-open degradation, never an error (№8), classification stays on stdout with the warn-once diagnostic on stderr (№2), and rk's `gone` maps to the documented dead-pane exit-2 path (№4). No corrections were needed.

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant (the raw-tmux classifier stays as the rk-less fallback arm; no file, symbol, or branch becomes unused)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | New seams live in a new `gate_rk.go` beside `gate.go` rather than inside it | Mirrors the package's file-per-concern layout (create.go/gate.go/pane.go); keeps the raw classifier diff-clean | S:60 R:90 A:85 D:80 |
| 2 | Confident | The fail-open stderr warning is emitted at most once per process (alongside the cached capability answer) | A per-probe warning would spam the gate loop's re-probes; the map-enumeration delegation warns silently — but the gate is a correctness path, so one warning beats silence | S:50 R:85 A:70 D:65 |
| 3 | Tentative | rk timeout constant 20s (`rkReadyTimeout`) | Carried from intake assumption 7; tunable single constant | S:40 R:90 A:60 D:55 |
| 4 | Confident | Snippet for the rk arm's `booting`/`parked` uses one extra fab capture per non-ready probe | Byte-stable output contract outweighs one subprocess on a non-hot path | S:55 R:85 A:80 D:75 |

4 assumptions (0 certain, 3 confident, 1 tentative).
