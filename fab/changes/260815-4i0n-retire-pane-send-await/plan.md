# Plan: Retire fab pane send/await (cli-layering Part 5)

**Change**: 260815-4i0n-retire-pane-send-await
**Intake**: `intake.md`

## Requirements

### CLI Surface: verb deletion

#### R1: `fab pane send` and `fab pane await` are removed from the CLI
The `fab` binary SHALL no longer expose `fab pane send` or `fab pane await`. The cobra commands, their flags, and their help text are deleted; the `fab pane` family `Use`/`Long` string and registration list name only the surviving verbs (`map`, `capture`, `process`, `window-name`, `open`, `ready`, `deliver`, `kill`).

- **GIVEN** the built binary
- **WHEN** `fab pane send %1 hi` or `fab pane await %1` runs
- **THEN** cobra reports an unknown command (no send/await handler exists)
- **AND** `fab pane --help` lists neither verb

#### R2: Internal builders and the deliver stack are untouched
The deletion SHALL NOT remove or alter `internal/pane`'s send builders (`SendLiteralArgs`, `SendKeyArgs`, `SendText`, `SendKey`), the gate/deliver choreography (`gate.go`), pane creation (`create.go`), or any `fab pane open/ready/deliver/kill/map/capture/process/window-name` or `fab dispatch` behavior — dispatch delivery (the pane arm) keeps working rk-less.

- **GIVEN** the post-deletion tree
- **WHEN** `go build ./... && go test ./...` runs under `src/go/fab`
- **THEN** all surviving packages build and pass
- **AND** `internal/pane/gate.go` still consumes `SendKey`/`SendText` via `tmuxPaneIO`

#### R3: The orphaned `Await` loop is deleted as dead code
`internal/pane/await.go` and `await_test.go` SHALL be deleted: their only non-test consumer is the deleted `pane_await.go` (`fab dispatch wait` runs its own loop in `internal/dispatch`).

- **GIVEN** the post-deletion tree
- **WHEN** `grep -rn "pane\.Await\|AwaitReport\|AwaitTick\|AwaitIdle\|AwaitFile\|AwaitGone\|AwaitRunning" src/go/fab --include="*.go"` runs
- **THEN** no occurrences remain

#### R4: Shared test helpers survive the test-file deletions
The tmux socket helpers `tmuxSocketDir`/`tmuxSocketPathLen` and their guard test `TestTmuxSocketDirLengthGuard` (currently defined in `pane_send_test.go`, consumed by ~12 sibling test files) SHALL be relocated to a neutral shared test file in the same package (`cmd/fab/tmuxsocket_test.go`) before `pane_send_test.go` is deleted. `pane_exitcode_test.go` drops only its two await rows; `panemap_test.go`'s `TestMapSendAgentAgreement_*` tests are kept (they pin the map-vs-`ResolvePaneContext` reader agreement, which `capture` still exercises) with names/comments reworded from `send` to `capture`.

- **GIVEN** the post-deletion tree
- **WHEN** `go test ./cmd/fab/...` runs
- **THEN** every surviving test compiles and passes with no reference to the deleted verbs

### Skill Guidance: migrate to rk mux with fail-open fallback

#### R5: `_cli-fab.md` reference sections deleted and family text swept
`src/kit/skills/_cli-fab.md` SHALL drop the `### send` and `### await` sections, remove `send`/`await` from the `## fab pane` family line and the pane-family exit-code paragraph (including await's two extra codes), and rewrite `§ agent state` to name `map`/`capture` as the readers (dropping the send warn-and-send clause). Every other cross-reference to the deleted verbs in the file is reworded to the surviving surface.

- **GIVEN** the edited file
- **WHEN** `grep -n "pane send\|pane await" src/kit/skills/_cli-fab.md` runs
- **THEN** no occurrences remain

#### R6: `_cli-agents.md` procedures repointed to rk mux, rk-absent path preserved
`src/kit/skills/_cli-agents.md` SHALL migrate: § Scope Boundary's `_cli-fab.md` list drops `fab pane send`; § Pre-Send Validation step 2 names `rk mux send` as the gated binary (same matrix: idle sends; waiting refuses unless `--answer`; active always refuses including under `--answer`; unknown warns and sends; `--force` skips — with probe-verified delivery built in and `--key <name>` covering key-name input), gated on `command -v rk`; § Delivery Probe notes `rk mux send` mechanizes the probe for message sends (probe failure = staged text + stderr + exit 1; never blind-resend) with the manual sentinel recipe as the raw-tmux fallback; § Await names `rk mux await` (`--until`/`--file`/`--timeout`; report words `idle`/`waiting`/`file`/`running`/`gone`, `gone` = exit 1) and the composed `rk mux send --await` as the preferred wait when rk is present. The rk-absent degraded path MUST be stated everywhere rk is recommended: the caller keeps the two-step policy gate itself (pane exists via `fab pane map`; state via `fab pane map`/`fab pane capture --json`), delivers via raw `tmux send-keys` + the manual delivery probe, and awaits via the poll-capture loop — never an error.

- **GIVEN** an agent following § Pre-Send Validation on a host without rk
- **WHEN** it reaches the send step
- **THEN** the guidance routes it through its own state read and raw `tmux send-keys` + manual probe with no rk invocation and no error path
- **AND** with rk installed the same section routes text sends through `rk mux send` (plain/`--answer`/`--force`/`--key`)

#### R7: `fab-operator.md` send paths migrated
`src/kit/skills/fab-operator.md` SHALL migrate its three send-path sites: the header routing paragraph (route via `rk mux send`, gated, with the raw-tmux degraded path); §3 Pre-Send Validation item 2 (confirmed sends: `waiting` → `rk mux send --answer`, `active` → `rk mux send --force`; rk-absent → confirm then raw send-keys behind the operator's own state gate); §5 Sending Auto-Answers (deliver via `rk mux send <pane> "<text>" --answer`, noting rk's built-in delivery verification and its staged-text failure mode; rk-absent fallback: re-capture guard + raw send-keys + manual delivery probe). Key-name input guidance becomes `rk mux send --key` when rk is present, raw `tmux send-keys` otherwise.

- **GIVEN** the edited skill
- **WHEN** the operator auto-answers a `waiting` agent with rk installed
- **THEN** the documented send is `rk mux send <pane> "<text>" --answer`
- **AND** with rk absent the documented flow is re-capture guard → raw `tmux send-keys` → manual delivery probe, without error

#### R8: `_cli-external.md` tmux note + rk section updated
`src/kit/skills/_cli-external.md` SHALL rewrite the § tmux "Send keys" bullet (prefer `rk mux send`, `command -v rk`-gated and fail-open; raw `tmux send-keys` when rk is absent) and add a fab-owned pointer under § rk (run-kit) for agent messaging: usage owned by `_cli-agents.md` § Pre-Send Validation / § Await and `fab-operator.md` §3/§5; the verbs' contract is tool-owned (`rk skill`). The frontmatter `description` is updated to match.

- **GIVEN** the edited file
- **WHEN** `grep -n "fab pane send" src/kit/skills/_cli-external.md` runs
- **THEN** no occurrences remain and the rk section carries the agent-messaging pointer

#### R9: Repo-wide guidance sweep
A sweep SHALL update every remaining skill-facing/spec reference to the deleted verbs: `docs/specs/skills.md` `/fab-operator` section (purpose, flow, and Tools lines), plus a repo-wide grep over `src/kit/` and `docs/specs/` (excluding `docs/specs/findings/` — dated historical reports stay) for the tokens `pane send`/`pane await` and the contrastive phrases ("prefer `fab pane send` over raw `tmux send-keys`", "the one remaining raw path", "the gated binary"). `docs/memory/` is deferred to hydrate.

- **GIVEN** the completed apply
- **WHEN** `grep -rn "pane send\|pane await" src/kit/ docs/specs/ --include="*.md"` runs (excluding `docs/specs/findings/`)
- **THEN** no skill-facing occurrences remain

### Non-Goals

- `fab pane capture`/`kill`/`process` guidance — Part 7 (gated on this part + Part 6 releasing).
- Any rk-side change — Part 1 shipped `rk mux send`/`await`; fab consumes the released contract.
- Deprecation aliases or stubs for the deleted verbs.
- A migration file — no user data is restructured.
- `docs/memory/` edits during apply — hydrate owns them.

### Design Decisions

#### Delete the internal Await loop with the verb
**Decision**: `internal/pane/await.go` + `await_test.go` go with `pane_await.go` rather than staying as "builders".
**Why**: its only non-test consumer is the deleted CLI verb; `fab dispatch wait` has its own record-keyed loop; keeping it would be dead code the "builders stay" rule was never meant to protect.
**Rejected**: keeping it for a hypothetical Part 6/7 consumer — the plan gives fab no future await consumer (rk mux await owns the record-free wait).
*Introduced by*: 260815-4i0n-retire-pane-send-await

#### Gate-matrix guidance stays fab-owned, contract delegated
**Decision**: fab skills keep a summarized gate matrix + flag mapping for `rk mux send`/`await` (fab-owned usage), pointing at `rk skill` for the tool-owned contract.
**Why**: `_cli-external.md` § Reference Model — fab-owned choreography lives in fab skills so the operator can act without a live rk lookup; full contracts delegate to the tool to avoid staleness.
**Rejected**: full delegation (operator's pre-send policy would lose its in-file matrix); full restatement (drifts against rk's release cadence, violates owner-or-pointer).
*Introduced by*: 260815-4i0n-retire-pane-send-await

#### Fail-open fallback is the caller's own gate + raw tmux
**Decision**: with rk absent, guidance degrades to the caller's two-step state read (`fab pane map`/`capture`) + raw `tmux send-keys` + manual delivery probe / poll-await — never an error, never a fab-binary gate.
**Why**: delegation rule 2 (capability-probed, fail-open); the binary gate is deleted, and the operator/agent procedures already perform the policy gate before every send.
**Rejected**: keeping `fab pane send` as the fallback binary (defeats the retirement; two gate implementations is the liability being removed).
*Introduced by*: 260815-4i0n-retire-pane-send-await

### Deprecated Requirements

#### `fab pane send` gated send (runtime/pane-commands)
**Reason**: substrate verb retired per cli-layering Part 5; rk owns the gate matrix (`rk mux send`, run-kit 260815-a5vf).
**Migration**: `rk mux send` (plain/`--answer`/`--force`/`--no-enter`/`--key`/`-L`), raw `tmux send-keys` when rk is absent.

#### `fab pane await` record-free wait (runtime/pane-commands)
**Reason**: same retirement; rk owns the record-free wait.
**Migration**: `rk mux await` / `rk mux send --await` (report `gone` = exit 1 under rk vs fab's exit 2); poll-capture loop when rk is absent. `fab dispatch wait` (record-keyed) is unaffected.

## Tasks

### Phase 1: Setup

- [x] T001 Relocate `tmuxSocketPathLen`, `tmuxSocketDir`, and `TestTmuxSocketDirLengthGuard` from `src/go/fab/cmd/fab/pane_send_test.go` into a new `src/go/fab/cmd/fab/tmuxsocket_test.go` (same package, verbatim move) <!-- R4 -->

### Phase 2: Core Implementation

- [x] T002 Delete `src/go/fab/cmd/fab/pane_send.go` and the remainder of `pane_send_test.go`; delete `src/go/fab/cmd/fab/pane_await.go` + `pane_await_test.go` <!-- R1 -->
- [x] T003 Edit `src/go/fab/cmd/fab/pane.go`: drop `paneSendCmd()`/`paneAwaitCmd()` registrations and update the family `Long` string to the surviving verbs <!-- R1 -->
- [x] T004 Delete `src/go/fab/internal/pane/await.go` + `await_test.go`; verify no `Await*` consumer remains (grep per R3) <!-- R3 -->
- [x] T005 Sweep surviving Go tests/comments: drop the two await rows in `cmd/fab/pane_exitcode_test.go`; reword `panemap_test.go` `TestMapSendAgentAgreement_*` names/comments to capture; fix stale `pane send` comment references in `internal/pane/pane.go`, `internal/pane/gate.go`, `cmd/fab/panemap.go`, `cmd/fab/memory_index.go` <!-- R4 --> <!-- rework: cycle 1 — pane.go sweep incomplete: stale `pane send` comments survive at internal/pane/pane.go:272 ("so send/map/capture agree") and :459 ("send/capture validate pane existence separately"); reword to the surviving readers (map/capture at :272, capture at :459) -->
- [x] T006 Run `go build ./... && go vet ./... && go test ./...` under `src/go/fab` (scope to `./cmd/fab/...` and `./internal/pane/...` first, then the full module) <!-- R2 -->

### Phase 3: Integration & Edge Cases (skill guidance)

- [x] T007 [P] `src/kit/skills/_cli-fab.md`: delete `### send`/`### await` sections; update family line, exit-code paragraph, `§ agent state`, and every other in-file cross-reference <!-- R5 -->
- [x] T008 [P] `src/kit/skills/_cli-agents.md`: migrate § Scope Boundary, § Pre-Send Validation step 2, § Delivery Probe, § Await to `rk mux send`/`await` with the `command -v rk` gate and the raw-tmux fail-open path; add the flag mapping and `--key` carve-out closure; update frontmatter description if it names the retired verbs <!-- R6 -->
- [x] T009 [P] `src/kit/skills/fab-operator.md`: migrate header paragraph, §3 Pre-Send Validation item 2, §5 Sending Auto-Answers to `rk mux send` (`--answer`/`--force`) + degraded path <!-- R7 -->
- [x] T010 [P] `src/kit/skills/_cli-external.md`: rewrite § tmux "Send keys" bullet; add the agent-messaging fab-owned pointer under § rk; update frontmatter description <!-- R8 -->
- [x] T011 `docs/specs/skills.md` `/fab-operator` section: purpose line, flow line, Tools line repointed to `rk mux send` + fallback phrasing <!-- R9 -->
- [x] T012 Repo-wide sweep: `grep -rn "pane send\|pane await" src/kit/ docs/specs/` (excluding `docs/specs/findings/`) plus the contrastive-phrase greps ("the gated binary", "prefer `fab pane send`", "the one remaining raw path", "remain raw `tmux send-keys`"); fix every skill-facing hit <!-- R9 -->

### Phase 4: Polish

- [x] T013 Run `fab sync` to refresh deployed skill copies, then re-run the R9 sweep against `.claude/skills/` as a deployment smoke check (no manual edits there) <!-- R9 -->

## Execution Order

- T001 blocks T002 (helpers must move before the file deletion).
- T002–T004 block T005–T006.
- T007–T010 are parallel; T011–T012 after them (sweep validates the edits); T013 last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab pane --help` lists neither `send` nor `await`; both invocations fail as unknown commands
- [x] A-002 R5: `_cli-fab.md` documents only the surviving pane verbs; family line + exit-code paragraph + § agent state consistent
- [x] A-003 R6: `_cli-agents.md` names `rk mux send`/`rk mux await` (gated) with the complete rk-absent degraded path
- [x] A-004 R7: `fab-operator.md` all three send sites migrated (header, §3 item 2, §5)
- [x] A-005 R8: `_cli-external.md` tmux bullet + rk section updated

### Behavioral Correctness

- [x] A-006 R6: Every rk recommendation in the touched skills is `command -v rk`-gated and states a working non-rk path (never an error) — delegation rule 2 honored verbatim
- [x] A-007 R6: The documented gate matrix matches shipped rk v3.16.24 (`--answer` permits waiting, refuses active; unknown warns and sends; `gone` exit 1 noted where await codes are described)

### Removal Verification

- [x] A-008 R1: No `paneSendCmd`/`paneAwaitCmd`/`idleGate`/`runPaneAwait` symbol remains in `cmd/fab`
- [x] A-009 R3: No `pane.Await`/`AwaitReport`/`AwaitTick` consumer remains anywhere in the module
- [x] A-010 R9: `grep -rn "pane send|pane await"` over `src/kit/` + `docs/specs/` (minus `findings/`) returns zero skill-facing hits

### Scenario Coverage

- [x] A-011 R2: `go build ./... && go test ./...` pass under `src/go/fab`; `internal/pane` gate/deliver tests untouched and green
- [x] A-012 R4: `tmuxSocketDir` consumers (dispatch/pane test files) compile and pass after the helper relocation

### Edge Cases & Error Handling

- [x] A-013 R6: Key-name input guidance covers both arms: `rk mux send --key` when rk present; raw `tmux send-keys` when absent
- [x] A-014 R7: The operator's `-L <server>` need is covered (`rk mux -L` documented where the operator sends cross-server)

### Code Quality

- [x] A-015 Pattern consistency: edits match surrounding file idiom (tables, § references, owner-or-pointer discipline)
- [x] A-016 No unnecessary duplication: rk contract facts are summarized once per owning section and pointed at elsewhere — no new owned-rule restatements
- [x] A-017 Canonical source only: zero edits under `.claude/skills/` (T013 uses `fab sync`)
- [x] A-018 CLI ⇒ docs + tests: the Go surface change lands with `_cli-fab.md` updates and test updates in the same change
- [x] A-019 Sibling sweep up front: aggregate specs (`skills.md`) and contrastive phrases swept before review, not after

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change is itself the deletion; review confirmed no further redundancy: `internal/pane` builders `SendLiteral`/`SendKey` (+ their `*Args` argv builders) remain consumed by `gate.go`'s deliver choreography (`PaneIO.SendLiteral`/`SendKey`), `paneValidationExitCode` still serves the five surviving validated verbs, and the relocated `tmuxsocket_test.go` helpers plus `strPtr` (`panemap_test.go:1370`) retain live consumers.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Relocated tmux socket helpers land in a new `tmuxsocket_test.go` rather than an existing test file | Neutral home avoids implying ownership by any one verb's tests; verbatim move keeps the diff reviewable | S:70 R:90 A:90 D:80 |
| 2 | Confident | `TestMapSendAgentAgreement_*` tests are kept and reworded (not deleted) | They pin the map-vs-ResolvePaneContext agreement, which `fab pane capture` still exercises; deleting them would lose real coverage | S:70 R:85 A:90 D:80 |
| 3 | Certain | Go comment mentions of `pane send` in surviving files are swept as part of T005 | Behavior-claim sweeps include comments per the recurring-lessons discipline | S:85 R:95 A:95 D:90 |

3 assumptions (1 certain, 2 confident, 0 tentative).
