# Intake: Tolerate run-kit's D5 JSON envelope on every `rk … --json` read

**Change**: 260912-zx5k-rk-json-envelope-tolerance
**Created**: 2026-09-12

## Origin

Synthesized from the 2026-09-12 working session (promptless dispatch via `/fab-proceed`, questioning mode `promptless-defer`). The user's framing:

> fab tolerates run-kit's D5 JSON envelope on every `rk … --json` read.
>
> run-kit's CLI read verbs now emit every `--json` document inside the standard D5 envelope — success `{"ok":true,"result":<the former bare document>}`, failure `{"ok":false,"error":{"code":…,"message":…}}` (plus an `Error:` line on stderr). Introduced by run-kit PR #949 ("CLI JSON read verbs — the D5 envelope …"), refined in #951/#953. fab still unmarshals these outputs as bare JSON arrays, so three fab read sites silently fail. […] One shared helper, not three inline fixes […] Backward compatible […] Fail-silent posture of each call site is unchanged. […] A fourth site to VERIFY rather than assume: `internal/pane/gate_rk.go`.

The trigger was an observed live symptom: during today's pipeline every `fab dispatch ready` printed `Warning: rk gate delegation returned an unparsable report for %169; falling back to the raw-tmux classifier`, and the operator-tick cron entry never converged on the schedule fab now derives.

**Cross-repo pairing.** This change is the fab-kit half of a run-kit contract change. run-kit's `260911-ehm2-cli-json-read-verbs` shipped the `{"ok":true,"result":…}` / `{"ok":false,"error":{…}}` envelope on every `rk … --json` verb (PR #949, refined in #951/#953), and run-kit's backlog row `[towb]` (2026-09-11) is the follow-up it filed against fab-kit:

> `[towb]` 2026-09-11: [fab-kit follow-up] Make fab's `rk … --json` readers envelope-aware: `pane_map.go` `parseRKPanes`, `fab-operator.md`'s `rk mux sessions --json` step, and any `cron list` reader — accept both `{"ok":true,"result":<doc>}` and the bare document, and read `error.message` on `ok:false`. (follow-up from run-kit `260911-ehm2-cli-json-read-verbs`, which shipped the `{"ok","result"}` / `{"ok":false,"error"}` envelope on every `rk … --json` verb)

`[towb]` is therefore the scope authority alongside the session: it names a doc site the session's list did not — `fab-operator.md`'s `rk mux sessions --json` step — which is why § D sweeps **every** `rk … --json` mention in the kit skills rather than only the two verbs the Go code reads.

Decisions carried in from that session are recorded as Certain/Confident rows in `## Assumptions`. The grounding below was verified live in this worktree against the installed `rk` (run-kit v3.19.52) and the Go sources — nothing here is inferred from the description alone.

## Why

### 1. The problem — three silent read failures, one of them load-bearing

`rk` v3.19.52 wraps every `--json` read verb in the D5 envelope. Verified verbatim on this machine:

```console
$ rk --version
run-kit version v3.19.52

$ rk cron list --json | head -c 200
{
  "ok": true,
  "result": [
    {
      "id": "hk6c",
      "name": "operator tick",
      "schedule": {
        "kind": "backoff",
        "min": "1m0s",
        "max": "30m0s"
      },
…

$ rk mux panes --json | head -c 180
{
  "ok": true,
  "result": [
    {
      "session": "_rk-operator",
      "session_id": "$10",
      "window_index": 1,
…

$ rk mux panes --json -L nosuchsrv; echo "exit=$?"
Error: list sessions: exit status 1: error connecting to /tmp/tmux-1001/nosuchsrv (No such file or directory)   # stderr
{
  "ok": false,
  "error": {
    "code": "operational",
    "message": "list sessions: exit status 1: error connecting to /tmp/tmux-1001/nosuchsrv (No such file or directory)"
  }
}
exit=1
```

run-kit's own help text now states the contract outright — `rk cron list --help`: *"`--json` emits the same records as a JSON array inside the standard `{"ok":true,"result":…}` envelope"*; `rk mux panes --help`: *"`--json` emits the machine-readable array, wrapped in the standard `{"ok":true,"result":…}` envelope."*

fab unmarshals all three of these into a bare Go slice, so each `json.Unmarshal` now fails with `cannot unmarshal object into Go value of type []…`:

| # | Site | Call | Failure mode today |
|---|------|------|--------------------|
| 1 | `src/go/fab/cmd/fab/operator_clock.go:116` — `resolveOperatorCronRow()` | `rk cron list --json` → `[]operatorCronRow` | `ok=false` ⇒ **the entire clock side effect is a no-op**: tracked-predicate mute/unmute, the `rk cron edit` schedule reconcile, and `/fab-operator` §2 Init's unmute-while-tracked all silently do nothing |
| 2 | `src/go/fab/cmd/fab/operator_clock.go:382` — `operatorPaneEpoch()` | `rk mux panes --json` → `[]rkPaneRow` | returns `false` (no epoch) ⇒ a derived schedule uses `--every` where it should use `--idle-every` |
| 3 | `src/go/fab/cmd/fab/pane_map.go:265` — `parseRKPanes()` | `rk mux panes --json` → `[]rkPaneRow` | `discoverPanesViaRK` returns `ok=false` ⇒ **silent fallback to fab's own `tmux list-panes` enumeration** |

**Live confirmation of #1** — the operator-tick entry on this machine still reads `backoff 1m→30m` / `deliver: immediate` / `muted: true`, although change `tnmm` (merged today) makes fab derive `3m→24m` with `skip-if-busy`. The reconcile has never converged, which is exactly the signature of a read that always returns `ok=false`.

**Live confirmation of #3** — `fab pane map --all-sessions` run from this worktree exits 0 and prints a table, but the first row is `_rk-ctl  %1  1  sahil …`. `rk mux panes` excludes internal sessions (`_rk-pin-*` pin-sessions and the `_rk-ctl` anchor) by contract, so the presence of `_rk-ctl` proves the delegated path failed and the raw-tmux fallback produced the table. The observed effect is therefore **degradation, not an error**: `fab pane map` and the operator's `fab operator tick-start --diff` snapshot keep working, but they lose (a) rk's filtered row set, (b) rk's *reconciled* agent state, and (c) the `has_agent` tri-state — which on the fallback path is always `null`, so the operator tick's `agent_exited` detection falls back to the process-tree walk on **every** pane instead of trusting rk's answer.

### 2. What happens if we don't fix it

The operator's clock stays frozen at whatever cadence the entry was last given by hand, and every `track` mutation that should mute/unmute silently doesn't. This is a *fail-silent* posture working exactly as designed and therefore invisible: no error, no exit code, no log line. The next change in this line (backlog `[bjrk]`, seeding the operator-tick entry from fab) builds directly on these reads, so it would be built on a dead seam.

### 3. Why this approach

The three sites already share an identical shape ("run an rk read verb, unmarshal stdout into a slice, degrade silently on failure"), and a fourth rk JSON read is a plausible near-term addition. Three inline `bytes.HasPrefix` sniffs would be the duplication anti-pattern `fab/project/code-quality.md` names outright. One pure, table-tested unwrap function keeps the sniff in one place and makes "every rk JSON read in fab goes through it" a reviewable, greppable rule.

Sniffing the shape (rather than probing an rk version or gating on a capability flag) is the compatibility mechanism: a bare array from a pre-D5 rk passes through untouched, so fab keeps working against both. A version gate would need maintaining, and a bottle can predate a same-version source change — the same reasoning `gate_rk.go`'s `rkSentinelProbe` already applies to the readiness classifier.

### 4. The fourth site — VERIFIED, and it is a different defect

`src/go/fab/internal/pane/gate_rk.go` runs `rk mux await --ready <pane> --timeout N` **without** `--json` and reads the first stdout token via `rkReportToken`. It is therefore **not** the envelope problem. It is a *report-word drift*, and the evidence is in run-kit's own help:

```console
$ rk mux await --help
… It reports `ready %N (state)`, `ready %N (echo)`, `parked %N`, or `narrow %N (WxH)` …
  The probe is gated on geometry: below the 80x20 readiness floor (either dimension) a bordered
  composer reflows or is not drawn at all, so the pane classifies `narrow %N (WxH)` instead of
  probing — exit 0, the geometry and remedy on stderr …
```

`narrow` is a **new fifth report word**. `probeRK` in `src/go/fab/internal/pane/gate.go:308-337` maps exactly four (`ready`, `parked`, `running`, `gone`); anything else falls to `default:` → `warnRKFallback(…)` with `runErr == nil`, which emits precisely the observed string:

> `Warning: rk gate delegation returned an unparsable report for %169; falling back to the raw-tmux classifier`

Geometry makes this reachable in normal operation: with `dispatch.column_width: 45`, a worker pane carved in the operator's own 173-column window is ≈77 columns — **below rk's 80-column floor**. A live `rk mux await --ready $TMUX_PANE --timeout 1` in this (332-column) pane reports `ready %168 (state)`, which maps fine; the warning only appears on panes under the floor, which is why it cannot be reproduced against the currently-open wide panes.

It is a run-kit output-contract drift that fab does not tolerate, so it is **in scope** — but as a distinct one-line mapping fix, not as a consumer of the JSON helper.

## What Changes

### A. New shared helper — `unwrapRkJSON`

New file `src/go/fab/cmd/fab/rk_json.go` (package `main`), with `src/go/fab/cmd/fab/rk_json_test.go` alongside.

```go
// unwrapRkJSON normalizes an `rk … --json` document to the bare payload,
// accepting BOTH of run-kit's shapes …
func unwrapRkJSON(data []byte) ([]byte, error)
```

Behavior, exhaustively:

| Input | Result |
|-------|--------|
| A bare JSON **array** (pre-D5 rk) | returned unchanged |
| A bare JSON **object** with no `ok` key | returned unchanged |
| `{"ok":true,"result":<doc>}` | the raw bytes of `result` |
| `{"ok":true}` with `result` absent or `null` | error (a malformed envelope, not a payload) |
| `{"ok":false,"error":{"code":C,"message":M}}` | error whose text carries `C` and `M` |
| Malformed JSON | error (the underlying decode error) |
| Empty / whitespace-only input | error |

Implementation sketch — decode once into a probe struct so the `ok` key's **presence** (not its value) is the discriminator:

```go
var probe struct {
    OK     *bool           `json:"ok"`
    Result json.RawMessage `json:"result"`
    Error  *struct {
        Code    string `json:"code"`
        Message string `json:"message"`
    } `json:"error"`
}
```

A leading `[` short-circuits to pass-through before the object decode. `OK == nil` means "not an envelope" ⇒ pass through unchanged.

**Package choice**: `cmd/fab` (package `main`). `operator_clock.go` and `pane_map.go` are in the *same package*, so one unexported function reaches all three JSON call sites with zero duplication and no new package. `internal/pane` was considered (it already hosts rk seams) but is not required: `gate_rk.go` needs no JSON unwrapping at all (§ D). If a future `internal/*` consumer needs it, promoting the file is a mechanical move.

### B. Route the three reads through it

Each site gains one step between the runner and the unmarshal, and **each keeps its existing degrade branch verbatim** — an unwrap error takes exactly the path the old unmarshal error took:

```go
// operator_clock.go resolveOperatorCronRow()
out, err := rkCronRunner("cron", "list", "--json")
if err != nil {
    return operatorCronRow{}, false
}
payload, err := unwrapRkJSON([]byte(out))
if err != nil {
    return operatorCronRow{}, false          // unchanged silent no-op
}
var rows []operatorCronRow
if err := json.Unmarshal(payload, &rows); err != nil {
    return operatorCronRow{}, false
}
```

- `operator_clock.go:116` `resolveOperatorCronRow()` — degrade branch: `return operatorCronRow{}, false`
- `operator_clock.go:382` `operatorPaneEpoch()` — degrade branch: `return false`
- `pane_map.go:265` `parseRKPanes()` — degrade branch: `return nil, err`, which `discoverPanesViaRK` turns into the silent raw-tmux fallback

No new error surface, no new exit code, no new warning, no `--json` flag change, no argv change.

Note on reachability: both runners surface a non-zero rk exit as a Go error (`exec.Command(…).Output()` for panes, `pane.RunCmdContext` for cron), and rk exits **1** on an `ok:false` envelope (verified above). So the `ok:false` branch is **defensive today** — the call sites bail on the exec error first. It is implemented anyway because the helper is a general contract and rk could emit `ok:false` on a zero exit for a partial result.

### C. `narrow` — teach `probeRK` run-kit's fifth report word

`src/go/fab/internal/pane/gate.go` `probeRK` gains a `narrow` case. Proposed mapping (see Assumption 8): treat `narrow` as **rk declining to classify**, i.e. fall through to the raw-tmux arm **without** the `warnRKFallback` warning — `handled=false, err=nil`, silently. Rationale: fab's own raw-tmux classifier has no geometry floor (fab's own floor is `dispatch.min_cols: 50`), so it can classify the pane rk refused to probe. Mapping `narrow` to `parked` instead would stall the gate on a pane fab can read, and burn two judgment rounds sending keystrokes at a problem keystrokes cannot solve.

Practical effect: the spurious per-process warning disappears and the gate's behavior on narrow panes is unchanged (it already falls through today, just noisily).

### D. Docs — the CLI-consumer contract

Constitution V applies throughout: these are deployed files, so they restate the rule and cite no fab-kit-only path.

- `src/kit/skills/_cli-fab-operator.md` (the clock paragraph, ~line 170) — the sentence "compared against the structured `schedule`/`deliver` fields of the resolved `rk cron list --json` entry … unparseable output is a silent no-op" gains: fab accepts **both** the bare-array form and the `{"ok":true,"result":[…]}` envelope form; an `{"ok":false,…}` envelope is treated as unparseable (the same silent no-op).
- `src/kit/skills/_cli-fab-pane.md` (~line 36, the `map` delegation paragraph) — the failure list "rk absent, a pre-3.17.18 rk, non-zero exit, unparseable JSON, …" gains the same both-shapes statement for `rk mux panes --json`.
- `src/kit/skills/_cli-fab-pane.md` (§ `fab pane ready`, ~lines 94 and 222) — the rk-arm report map gains `narrow %N (WxH)` → falls through to the raw-tmux arm (silently, not a warning).
- `src/kit/skills/fab-operator.md` §2 Init step 4 (~line 113) — "read it and select the row whose `target` is `role:operator`" gains a short clause that in the envelope form the rows live under `result`.

**Full `rk … --json` sweep of the kit skills.** The doc sweep is defined by a grep for every `rk … --json` mention, not by the two verbs the Go code reads — an agent following a skill step parses these documents by hand and needs the same both-shapes statement. The complete set in this worktree:

| File:line | Read | Sweep action |
|-----------|------|--------------|
| `_cli-fab-operator.md:170` | `rk cron list --json`, `rk mux panes --json` | both-shapes clause (the clock paragraph, above) |
| `_cli-fab-operator.md:140` | `rk mux panes --json` (`has_agent` tri-state in tick detection) | both-shapes clause on the row source |
| `_cli-fab-pane.md:36`, `:57` | `rk mux panes --json` (`map` delegation, identity-key contract) | both-shapes clause |
| `fab-operator.md:113` | `rk cron list --json` (§2 Init step 4) | rows live under `result` |
| `fab-operator.md:305`, `:359` | `rk cron list --json` (per-tick cadence read, Status Frame Format) | rows live under `result` — the agent renders `schedule_summary`/`deliver` from this read by hand |
| `fab-operator.md:479` | `rk mux sessions --json`, `rk mux panes --json` (spawn target-session selection) | rows live under `result` — named explicitly by run-kit backlog `[towb]` |
| `fab-operator.md:816`, `:821` | `rk cron list --json` (the §2 rk Gate reference row, the cadence summary row) | same clause where a field is read; `:77`'s gate command discards stdout (`>/dev/null`) and needs no clause |
| `_cli-agents.md:141` | `rk mux capture --json` | both-shapes clause — note this one's `result` is an **object**, not an array (verified: `{"ok":true,"result":{"pane":…,"lines":…,"content":…}}`) |

Verified live that the envelope covers these verbs too:

```console
$ rk mux sessions --json | head -c 90
{
  "ok": true,
  "result": [
    {
      "name": "fab_kit",
      "role": "user",
…

$ rk mux capture --json %168 --lines 2 | head -c 70
{
  "ok": true,
  "result": {
    "pane": "%168",
…
```

Prefer one restated sentence per file (or per paragraph where several reads sit together) over eight near-duplicate clauses — `fab/project/code-quality.md`'s owner-or-pointer convention: the deployed skill is the owner here, so state it where an agent reads a field, not everywhere the verb is named.

### E. Memory — present truth

FKF style: no narration of the transition, rationale relocated into a `## Design Decisions` entry.

- `docs/memory/runtime/operator.md` — the clock paragraph (~line 132) states that the `rk cron list --json` read accepts both the bare array and the `{ok,result}` envelope.
- `docs/memory/runtime/pane-commands.md` — the rk enumeration paragraph (~line 11) gets the same statement for `rk mux panes --json`; the readiness-gate paragraphs (~lines 143, 229) record `narrow` as a known rk report word that falls through to the raw arm.

Sweep note: `docs/memory/runtime/dispatch.md` and `docs/memory/runtime/agent-primitives.md` both describe the `fab dispatch ready` rk-arm delegation; grep for the report-word list before finishing apply (`fab/project/code-quality.md` § Sibling Sweeps).

### F. Tests

Table-driven, per `fab/project/code-quality.md` § Test Strategy (test-alongside).

1. `src/go/fab/cmd/fab/rk_json_test.go` — one table over `unwrapRkJSON`: bare array, bare object, `ok:true` with an array `result`, `ok:true` with an object `result`, `ok:true` with `result` absent, `ok:false` with `error.code`/`error.message` (assert both appear in the error text), malformed JSON, empty input, whitespace-only input.
2. `src/go/fab/cmd/fab/operator_clock_test.go` — an envelope-shaped fixture beside `cronListBackoffJSON` fed through `stubRkCron`, asserting the same row is resolved from both shapes; and an envelope-shaped `rk mux panes` fixture in the `operatorPaneEpoch` table (`stubEpoch`).
3. `src/go/fab/cmd/fab/pane_map_test.go` — an envelope-shaped fixture in the `parseRKPanes` table, asserting identical `paneEntry` output to the bare-array fixture.
4. `src/go/fab/internal/pane/` gate tests — a `narrow %169 (77x40)` stdout case asserting `handled=false`, `err=nil`, and **no** warning emitted (the existing warn-capture seam `rkWarn` is already a test var).

## Affected Memory

- `runtime/operator`: (modify) clock paragraph — the `rk cron list --json` read accepts both the bare-array and `{ok,result}` envelope shapes; add a Design Decision for the shape-sniff-over-version-gate choice
- `runtime/pane-commands`: (modify) rk enumeration paragraph — same both-shapes statement for `rk mux panes --json`; readiness-gate paragraphs — `narrow` as a known rk report word that falls through to the raw-tmux arm without a warning

## Impact

**Code**

- `src/go/fab/cmd/fab/rk_json.go` (new) + `rk_json_test.go` (new)
- `src/go/fab/cmd/fab/operator_clock.go` — `resolveOperatorCronRow()`, `operatorPaneEpoch()`
- `src/go/fab/cmd/fab/pane_map.go` — `parseRKPanes()`
- `src/go/fab/internal/pane/gate.go` — `probeRK()` (the `narrow` case)
- `src/go/fab/cmd/fab/operator_clock_test.go`, `pane_map_test.go`, and the `internal/pane` gate test file

**Docs (deployed kit content)**

- `src/kit/skills/_cli-fab-operator.md`, `src/kit/skills/_cli-fab-pane.md`, `src/kit/skills/fab-operator.md`, `src/kit/skills/_cli-agents.md` — the full `rk … --json` grep set (§ D)

**Memory**

- `docs/memory/runtime/operator.md`, `docs/memory/runtime/pane-commands.md`

**Behavior surfaces restored (not newly created)**

- The operator clock's mute/unmute and `rk cron edit` schedule reconcile begin converging again
- `fab pane map` / `fab operator tick-start --diff` regain rk's filtered row set, reconciled agent state, and the `has_agent` tri-state
- `fab dispatch ready` stops emitting the spurious unparsable-report warning on sub-80-column panes

**Dependencies** — none added. run-kit stays an optional, LookPath-gated, fail-silent dependency of the two `cmd/fab` sites and the gate's rk arm.

**Other `json.Unmarshal` sites audited** (a full sweep of `src/go/fab`, non-test): `operator_track.go:316` and `:661` decode **user-supplied** `--scope` / `--json` flag values; `operator_track.go:361` decodes `gh repo view --json nameWithOwner` (GitHub, not rk); `operator_tick_start.go:807` decodes a **tracked shell probe's** stdout against a generic "must be a JSON object" contract; `internal/predicate`, `internal/log`, `internal/configref`, `operator_state.go` are all fab-internal marshalling. **No other rk-output parse exists.**

## Open Questions

- A tracked shell probe configured to run `rk … --json` (`fab operator track add --kind shell`) now receives the envelope. `operator_tick_start.go:807` accepts it — it *is* a JSON object — but the item's `done_when` predicate paths must now be written against `result.…`. Should the generic probe contract special-case rk envelopes, or is rewriting the predicate path the user's job? Treated as out of scope here (see Assumption 9); no live tracked item is known to be affected. <!-- assumed: tracked shell probes running `rk … --json` stay out of scope — unwrapping inside a provider-agnostic probe contract would silently change semantics for non-rk probes -->
- Should `unwrapRkJSON`'s pass-through of a bare object that happens to carry its own `ok` field be guarded further? No current rk read verb returns such a document (`cron list` and `mux panes` are arrays), so the ambiguity is theoretical; noted so a future rk object-returning read verb is checked against it.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | One shared pure helper `unwrapRkJSON(data []byte) ([]byte, error)` rather than three inline shape sniffs | Explicitly decided in the dispatching session; also the direct reading of `fab/project/code-quality.md`'s "duplicating existing utilities" anti-pattern | S:95 R:85 A:90 D:90 |
| 2 | Certain | Backward compatible by **shape sniff** — a bare array/object passes through; no rk version gate and no capability probe | Explicitly decided in the session; matches the existing `rkSentinelProbe` precedent (probe the binary, never the version string) | S:95 R:80 A:90 D:95 |
| 3 | Certain | Each call site's fail-silent posture is unchanged — an `ok:false` envelope or an unwrap error takes the same degrade branch the unmarshal error takes today; no new error surface or exit code | Explicitly decided in the session; the three degrade branches already exist and are documented in the deployed CLI partials | S:95 R:85 A:95 D:95 |
| 4 | Certain | The helper lives in `src/go/fab/cmd/fab/rk_json.go`, package `main` | Delegated to me with "prefer an existing internal package if one fits". Verified: all three JSON call sites are in package `main` (`operator_clock.go`, `pane_map.go`), and `gate_rk.go` needs no JSON unwrapping — so no cross-package seam exists to serve. Promoting to `internal/pane` later is a mechanical move | S:70 R:90 A:90 D:85 |
| 5 | Certain | Change type `fix`; LIGHT lane expected (≤5 tasks) | Stated in the dispatch; `fab status refresh`'s keyword inference independently lands on `fix` | S:90 R:90 A:90 D:90 |
| 6 | Certain | `gate_rk.go` is IN SCOPE, but as a report-word drift, not an envelope problem: `rk mux await --ready` gained a fifth report word `narrow %N (WxH)` (80x20 geometry floor) that `probeRK`'s four-case switch does not map | Verified from `rk mux await --help` on v3.19.52 and from `gate.go:308-337`; the `default:` branch emits the exact warning string observed today. The dispatch said to include a text-shape drift in scope | S:80 R:75 A:85 D:80 |
| 7 | Certain | Docs scope is defined by a **grep for every `rk … --json` mention in `src/kit/skills/`** — four deployed files (`_cli-fab-operator.md`, `_cli-fab-pane.md`, `fab-operator.md`, `_cli-agents.md`, covering `cron list` / `mux panes` / `mux sessions` / `mux capture`) plus two memory files (`runtime/operator.md`, `runtime/pane-commands.md`), all citing no fab-kit-only paths | Enumerated in the dispatch, widened by run-kit backlog `[towb]` (which names the `rk mux sessions --json` step the session's list missed), and each target line located in this worktree by grep (§ D table). Constitution V + `fab/project/code-review.md`'s must-fix rule fix the citation form | S:85 R:75 A:90 D:85 |
| 8 | Confident | `narrow` maps to a SILENT fall-through to the raw-tmux arm (`handled=false, err=nil`, no warning) rather than to `parked` or to today's noisy fail-open | Not discussed in the session — my call. The codebase answers it: fab's raw classifier has no geometry floor, so it can classify the pane rk declined to probe; mapping to `parked` would stall the gate and spend two judgment rounds sending keystrokes at a geometry problem. One switch case + one test, trivially reversible | S:25 R:70 A:70 D:60 |
| 9 | Tentative | Tracked shell probes that run `rk … --json` are OUT of scope — `operator_tick_start.go`'s generic probe contract is left alone and a user's `done_when` path is theirs to point at `result.…` | Would-be question, defaulted rather than deferred: unwrapping rk-shaped envelopes inside a provider-agnostic probe contract would silently change semantics for non-rk probes, and no live tracked item is known to be affected. Recorded as an Open Question so `/fab-clarify` can reverse it cheaply | S:20 R:55 A:45 D:35 |
| 10 | Certain | Non-goals: no run-kit change; no argv change and no `--json` added to `mux await`; no `.fab-dispatch` mechanics touched; seeding the operator-tick entry from fab stays backlog `[bjrk]` (the next change) | Explicitly stated in the dispatch | S:95 R:90 A:95 D:90 |

10 assumptions (8 certain, 1 confident, 1 tentative, 0 unresolved).
