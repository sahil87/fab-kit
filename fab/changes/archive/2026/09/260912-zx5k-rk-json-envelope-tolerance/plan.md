# Plan: fab Tolerates run-kit's D5 JSON Envelope on Every `rk … --json` Read

**Change**: 260912-zx5k-rk-json-envelope-tolerance
**Intake**: `intake.md`

## Requirements

### Runtime: rk JSON consumption (Go)

#### R1: One shared unwrap helper accepts both rk output shapes
A pure function `unwrapRkJSON(data []byte) ([]byte, error)` in `src/go/fab/cmd/fab/rk_json.go` MUST return a bare JSON array or a bare object without an `ok` key unchanged; MUST return the raw `result` bytes of `{"ok":true,"result":…}`; MUST return an error for `{"ok":true}` with `result` absent or `null`, for `{"ok":false,"error":{code,message}}` (error text carries both `code` and `message`), for malformed JSON, and for empty or whitespace-only input. The `ok` key's presence, not its value, is the envelope discriminator.

- **GIVEN** `rk cron list --json` output from run-kit ≥ 3.19 (`{"ok":true,"result":[…]}`)
- **WHEN** the helper unwraps it
- **THEN** the returned bytes are exactly the inner array
- **AND** a pre-D5 bare array passes through byte-identical

#### R2: Every rk JSON read routes through the helper with its degrade branch unchanged
`resolveOperatorCronRow()` and `operatorPaneEpoch()` in `operator_clock.go`, and `parseRKPanes()` in `pane_map.go`, MUST unwrap before unmarshalling; an unwrap error MUST take exactly the branch the unmarshal error already takes (`ok=false` / `false` / `nil, err` → the silent raw-tmux fallback). No new error surface, exit code, warning, argv, or `--json` flag.

- **GIVEN** the live cron row `{"ok":true,"result":[{…"target":"role:operator"…}]}` and a pane-only tracked set
- **WHEN** the schedule reconcile runs
- **THEN** the `role:operator` row resolves and the reconcile issues its `rk cron edit` as designed
- **AND** an enveloped `rk mux panes --json` document yields the same `paneEntry` list as its bare-array twin

#### R3: The readiness gate maps rk's `narrow` report word silently
`probeRK` in `src/go/fab/internal/pane/gate.go` MUST treat a `narrow` first token (rk declining to classify a pane under its 80x20 floor, exit 0) as a silent fall-through to the raw-tmux arm: `handled=false, err=nil`, and NO `warnRKFallback` call. The four existing mappings and the `default:` warn-once fail-open are unchanged for every other token.

- **GIVEN** rk stdout `narrow %17 (77x40)` with a nil run error
- **WHEN** `Gate.Probe` runs on an rk-capable host
- **THEN** the raw-tmux arm classifies the pane and the warning sink is never invoked

### Kit: deployed CLI-consumer contract

#### R4: Kit skills state the both-shapes rule where an agent or fab reads a field
`src/kit/skills/_cli-fab-operator.md` (clock paragraph ~L170; detection paragraph ~L140), `_cli-fab-pane.md` (`map` delegation ~L36; the two rk-arm report maps ~L94 and ~L222 gain `narrow` → silent fall-through), `fab-operator.md` (§2 step 4 ~L113; per-tick cadence read ~L305/L359; spawn target-session step ~L479 for `rk mux sessions --json`; Key Properties cadence row ~L821), and `_cli-agents.md` (~L141, `rk mux capture --json` whose `result` is an object) MUST carry one restated sentence per file or paragraph: fab and the agent accept both the bare document and the `{"ok":true,"result":<doc>}` envelope; an `{"ok":false,…}` envelope is treated as unparseable (the same silent degrade). Deployed files MUST NOT cite fab-kit-only paths.

- **GIVEN** an agent following §2 Init step 4 against run-kit ≥ 3.19
- **WHEN** it reads `rk cron list --json`
- **THEN** the skill tells it the rows live under `result`

### Non-Goals
- Seeding the operator-tick entry from fab — backlog `[bjrk]`, the next change, built on this one
- Any run-kit change; adding `--json` to `mux await`; any argv change
- Unwrapping envelopes inside tracked shell probes (`operator_tick_start.go`'s provider-agnostic probe contract) — a user's `done_when` path points at `result.…` (intake Assumption 9)
- A version gate or capability probe — the shape sniff is the compatibility
- Mapping `narrow` to `parked` (would stall the gate and spend judgment rounds on a geometry problem)

### Design Decisions

#### rk JSON Reads Are Shape-Sniffed, Never Version-Gated
**Decision**: every `rk … --json` document fab reads passes through one pure helper that accepts the bare document and the `{ok,result}` / `{ok:false,error}` envelope, discriminating on the presence of `ok`; each call site keeps its existing fail-silent degrade branch. rk's `narrow` readiness report falls through to the raw-tmux arm silently.
**Why**: rk is substrate fab shells out to and its output contract moved under fab; a shape sniff keeps fab working against both old and new rk with no version compare (the `rkSentinelProbe` posture). Three inline sniffs would be the duplicated-utility anti-pattern. `narrow` is rk declining to classify, and fab's own classifier has no geometry floor, so falling through is the correct answer and the warning was noise.
**Rejected**: per-site inline unwrapping (three drifting copies); an rk version gate (bottle/source skew makes version strings lie); mapping `narrow` to `parked` (stalls the gate, burns keystroke rounds on a geometry problem); unwrapping inside the generic shell-probe contract (would change semantics for non-rk probes).
*Introduced by*: 260912-zx5k-rk-json-envelope-tolerance

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add `src/go/fab/cmd/fab/rk_json.go` (`unwrapRkJSON`, probe struct with `OK *bool`, `Result json.RawMessage`, `Error *{Code,Message}`; leading `[` short-circuit) and `rk_json_test.go` (table: bare array, bare object, ok:true array result, ok:true object result, ok:true missing result, ok:true null result, ok:false with code+message asserted in error text, malformed JSON, empty, whitespace). <!-- R1 -->
- [x] T002 Route the three reads through `unwrapRkJSON` keeping each degrade branch verbatim: `operator_clock.go` `resolveOperatorCronRow` (~L116) and `operatorPaneEpoch` (~L382), `pane_map.go` `parseRKPanes` (~L265). Tests: an enveloped twin of `cronListBackoffJSON` resolved through `stubRkCron` (same row, same no-edit outcome), an enveloped row in the `operatorPaneEpoch` table, an enveloped `rkPanesFixture` case in `pane_map_test.go` asserting identical entries to the bare fixture, and an `{"ok":false,…}` case at one site proving the degrade branch. <!-- R2 -->
- [x] T003 `src/go/fab/internal/pane/gate.go` `probeRK`: add `case runErr == nil && token == "narrow": return "", "", false, nil` before `default:` and extend the doc comment's report table; `gate_rk_test.go`: a `narrow %17 (77x40)` case asserting the raw arm answers, `rkWarn` is never called, and `rkWarned` stays false. `gofmt -l` clean; `go test ./internal/pane ./cmd/fab` from `src/go/fab`. <!-- R3 -->

### Phase 4: Polish

- [x] T004 [P] Kit skill sweep per R4 (rework cycle 1 added: `_cli-fab-operator.md` ~L140 detection paragraph; `rk mux capture --classify --json` report at `fab-operator.md` ~L382 and `_cli-agents.md` ~L146; `rk tab new --ready --json` report at `_cli-agents.md` ~L92 and `fab-operator.md` ~L503): `_cli-fab-operator.md` (~L140, ~L170), `_cli-fab-pane.md` (~L36, ~L94, ~L222), `fab-operator.md` (~L113, ~L305, ~L359, ~L479, ~L821), `_cli-agents.md` (~L141). One sentence per file/paragraph, no fab-kit paths. <!-- R4 -->
- [x] T005 [P] Verification sweep: grep `src/kit/skills` for every `rk .* --json` mention and confirm each reading site is covered or is stdout-discarded (`fab-operator.md:77`); grep `parked` report-word maps in `src/kit/skills` for a missing `narrow`; list the memory sites for hydrate (`docs/memory/runtime/operator.md` clock paragraph, `pane-commands.md` ~L11/L143/L229, `dispatch.md` ~L157–167 report table + the rk-arm design decision ~L911, `agent-primitives.md` if it lists report words). <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `unwrapRkJSON` exists in `cmd/fab/rk_json.go` with the ten-case table test green
- [x] A-002 R2: `resolveOperatorCronRow`, `operatorPaneEpoch`, and `parseRKPanes` each call `unwrapRkJSON` before `json.Unmarshal`
- [x] A-003 R3: `probeRK` has a `narrow` case returning `"", "", false, nil` with no warning
- [x] A-004 R4: every kit-skill site listed in T004 carries the both-shapes (or `narrow`) statement — verified cycle 2: `_cli-fab-operator.md:140` detection paragraph now carries "(rows read from the bare array or from the `result` array of run-kit's `{"ok":true,…}` envelope — either shape)"; all other sites verified by grep

### Behavioral Correctness

- [x] A-005 R2: the enveloped cron fixture resolves the same `role:operator` row as the bare fixture (test)
- [x] A-006 R2: the enveloped `rk mux panes` fixture yields entries identical to the bare fixture (test)
- [x] A-007 R2: an `{"ok":false,…}` document takes the pre-existing degrade branch at the tested site, with no new stderr output
- [x] A-008 R3: a `narrow` report leaves `rkWarned` false and the raw arm's classification is returned (test)

### Scenario Coverage

- [x] A-009 R1: pre-D5 bare array and bare object pass through byte-identical (test)
- [x] A-010 R2: the epoch probe returns `true` for an enveloped row with `agent_state: active` on the role-marked window (test)

### Edge Cases & Error Handling

- [x] A-011 R1: `{"ok":true}` with `result` absent or `null` is an error, not an empty payload
- [x] A-012 R1: empty and whitespace-only input are errors; the leading-`[` short-circuit tolerates leading whitespace
- [x] A-013 R3: `ready`/`parked`/`running`/`gone` mappings and the `default:` warn-once path are unchanged (existing tests still pass)

### Code Quality

- [x] A-014 Pattern consistency: the helper and test follow the package's seam/table conventions (`stubRkCron`, `rkPanesRunner`, `rkWarn`)
- [x] A-015 No unnecessary duplication: exactly one unwrap implementation; no per-site shape sniff
- [x] A-016 No magic strings: the `narrow` token is a named constant or documented literal in the same style as `"running"`
- [x] A-017 Canonical sources only: kit edits in `src/kit/skills/*.md`, never `.agents/` or `.claude/`
- [x] A-018 CLI-consumer contract ⇒ reference + tests: `_cli-fab-operator.md` and `_cli-fab-pane.md` change with the Go, tests ship alongside
- [x] A-019 Constitution V: no deployed file cites `docs/specs/`, `docs/memory/`, `docs/site/`, or `src/go/`
- [x] A-020 **N/A**: hydrate-owned, verified post-hydrate
- [x] A-021 Test strategy: `gofmt -l` clean on touched Go; `go test ./cmd/fab ./internal/pane` green before apply finishes

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant (the `default:` warn-once path in `probeRK` still serves every non-`narrow` unknown token, and all three unwrap call sites remain live)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Five tasks ⇒ LIGHT lane; memory edits are hydrate's (A-020), not an apply task | `_pipeline.md` fork rule; pipeline convention | S:90 R:95 A:90 D:90 |
| 2 | Confident | `narrow` is matched as a literal token string beside `"running"` rather than a new `Readiness` constant | It is not a fab report state — fab never prints it — so it does not belong in the `Readiness` set the skills branch on | S:70 R:90 A:85 D:75 |
| 3 | Confident | The `ok:false` degrade test lives at the cron site (`stubRkCron` already takes a JSON string) rather than at all three | One proof of the branch is enough; the helper's own table covers the error text | S:65 R:95 A:85 D:75 |

3 assumptions (1 certain, 2 confident, 0 tentative).
