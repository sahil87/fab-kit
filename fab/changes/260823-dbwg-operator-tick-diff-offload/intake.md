# Intake: Operator Tick-Diff Offload (`fab operator tick-start --diff`)

**Change**: 260823-dbwg-operator-tick-diff-offload
**Created**: 2026-08-23

## Origin

Promptless dispatch (`/fab-proceed` create-new path, `{questioning-mode} = promptless-defer`). This change is **exactly Change 3 (Phase B1)** of the ordered queue in the authoritative plan `fab/plans/sahil/26-08-23-operator-offload-plan.md` (commit 4c4ba90d, written from a /fab-discuss brainstorm and revised the same day after a four-agent review — Go feasibility, doctrine coherence, failure red-team, scope). All design decisions below are **plan-settled post-review and treated as settled**, per the dispatch instruction.

> Change 3 (Phase B1): `fab operator tick-start --diff` — the binary takes over the mechanical
> half of the operator's tick — PLUS the coupled `src/kit/skills/fab-operator.md` §4 tick
> rewrite. These MUST ship in the same change (plan § Critical couplings).

Queue context: Change 1 (Phase A, `fab operator note` verbs — PR #615) and Change 2 (C2 auto-merge choreography — PR #616) are **already merged to main** and available as dependencies. Change 4 (Phase B2, `fab pane questions` + §5 rewire) is a LATER change and out of scope here.

All plan code anchors were re-verified on 2026-08-23 against **origin/main** (which carries the #615/#616 merges); drifted line numbers are corrected inline below — reference by content where noted.

## Why

1. **The tick is expensive and mechanical.** Every operator tick today, the LLM re-runs a full `fab pane map --all-sessions --json`, re-reads `fab operator state`, diffs stage/state per monitored entry in context, and hand-maintains last-known bookkeeping via `fab operator update`. All of that is mechanical work the binary can own — the existing Full Mediation doctrine ("agents never compute or hand-write what the binary can own", docs/memory/runtime/operator.md § Full Mediation of State Writes) not yet applied to the tick.
2. **If we don't fix it**: the operator's context window fills with per-tick pane-map dumps and state reads, its queue stays busy on bookkeeping instead of judgment, and a restarted operator pays the full re-derivation cost every tick. Per-tick LLM cost scales with fleet size instead of with events.
3. **Why this approach**: a binary-side diff with **two delivery classes** (level-triggered for action-demanding events, consumed-on-read for report-only events) was the four-agent review's load-bearing correction — a naive baseline-advancing diff is at-most-once delivery and the lost side is the action-demanding side, while today's skill self-heals only because every tick recomputes from the full map. The `fleet:` block closes the review's biggest hole: without it the skill keeps fetching the full map for the status frame and the offload evaporates.

The result: an idle-fleet tick becomes one cheap command (`tick-start --diff`) with near-zero LLM context.

## What Changes

### 1. `fab operator tick-start --diff` (src/go/fab/cmd/fab/operator_tick_start.go — modify)

Flagless `tick-start` stays **byte-identical/unchanged** (increments `tick_count`, writes `last_tick_at`, prints `tick: N\nnow: HH:MM`). With `--diff`, the binary additionally:

1. **Runs the pane-map snapshot internally** — same `main` package; `panemap.go`'s discovery+resolve row collection (the `discoverPanesViaRK` → `discoverPanes` fallback plus the `resolvePane` loop inside `runPaneMap`, panemap.go ~lines 104–135) is extracted into a `collectPaneRows`-style helper behind a **stubbable package-var seam** — the `rkPanesRunner` (panemap.go:215) / `operatorStatePathOverride` (operator_tick_start.go:13) precedent. `fab pane map`'s own behavior is unchanged (pure extraction).
2. **Compares against the `monitored` baseline** and emits `deltas:` in **two delivery classes** (the load-bearing design point):
   - **Level-triggered, re-emitted every tick until the skill acts** — `completion` and `pane_death`. Both are stateless predicates over the current snapshot, needing no baseline:
     - `completion` = stage terminal (or == the entry's `stop_stage`). A stage-diff **cannot** detect completion: a change completing at its terminal stage never changes its stage string; only `display_state` flips (see `status.DisplayStage`, src/go/fab/internal/status/status.go:544). Detection is a display-state/terminal-stage predicate.
     - `pane_death` = the entry's pane absent from the snapshot.
     - `fab operator remove` is the natural ack — the entry disappears, the event stops. A crash between diff and action loses nothing.
   - **Consumed-on-read (baseline-diffed)** — `stage_advance` and `review_fail`. A lost one costs a missed report only; tolerable.
   - **`pane_mismatch`** (new, level-triggered): each entry's change ID cross-checked against the snapshot row's `change` field (`paneRow.change`; JSON `Change` at panemap.go:593) — tmux recycles `%N` pane IDs across server restarts while the socket-keyed state file survives, so a recycled pane must never be diffed or swept as the old agent. Treated like pane_death: report + remove, never a candidate. Aligns with the pane-identity-keying contract (#612).
3. **Emits `candidates:`** — waiting-first then idle; each row: `pane`, `change`, `agent_state`, `idle_duration` (stuck detection needs it). Population is **monitored agents only** — discovery stays here, not in a pane verb (sweep population is operator state). Unknown-state (`—`) panes excluded by default; on rk-less servers all panes read `—` → candidates empty → identical to today's §5 policy.
4. **Emits `fleet:`** — a compact full-fleet block, per entry: `repo`, `session`, `stage`, `display_state`, `agent_state`, `idle_duration`, `pr_url` — **the status frame's data source**. Without it the skill keeps fetching the full map and the offload evaporates. (This also pre-baselines C3's `state --frame`, a follow-up, not this change.)
5. **Updates the baseline in the same atomic mutation** (`mutateOperatorState`, operator_state.go:127 — load tolerant → typed edit → `saveOperatorState` atomic temp+rename). `--diff` becomes the **authoritative baseline writer** for the monitored entries' observed fields. The stale comment at operator_tick_start.go:30–31 ("and the owned sections, untouched here") is corrected — with `--diff` the `monitored` section IS touched.
6. **Quiet fleet** → `deltas: [] candidates: []` plus the fleet block — a cheap no-op tick. **Empty monitored set** → empty deltas/candidates + empty fleet, tick still increments (no-op tick is first-class). Enrollment between diffs enters the baseline via `enroll`; the next `--diff` emits no synthetic event. Concurrency: atomic temp+rename as every verb; last-writer-wins under one operator per server. Server-scoping asymmetry accepted: `tick-start` keys off the current socket — an operator drives `--diff` only for its own server, which is the model.

`--diff` output remains a single stdout document the skill parses (retaining the `tick:`/`now:` lines, then the `deltas:`/`candidates:`/`fleet:` blocks) — exact key spelling is an apply-time decision within this shape.

### 2. Skill §4 tick rewrite (src/kit/skills/fab-operator.md — modify, COUPLED)

Ships in the **same change** (plan § Critical couplings): `--diff` becomes the authoritative baseline writer, so the skill must simultaneously stop doing its own last-known bookkeeping on the diff path, or the next diff under-reports. The coupling also gives the flag a consumer and satisfies the constitution's simultaneous `_cli-fab.md` rule.

Against origin/main (post-#615/#616; the plan's `fab-operator.md:241` anchor drifted — reference by content):

- **§4 Tick Behavior step 1 (Snapshot, main line ~266)**: rewired to `fab operator tick-start --diff` — the deltas replace the in-context stage/state diffing, and the **status frame is rendered from the `fleet:` block**. Drops the per-tick `fab pane map --all-sessions --json` call and drops the per-tick `fab operator state` read (fleet + deltas carry what the frame and monitoring need; the watch pass reads state on its own step as today).
- **§4 step 6 (Observed-field updates, main line ~272)**: the per-tick `fab operator update <change-id>` stage/agent bookkeeping is **removed from the diff path** — `--diff` owns the baseline write. `fab operator update` **stays** for non-baseline field edits (e.g. `stop_stage`).
- **Level-triggered semantics stated**: `completion`/`pane_death`/`pane_mismatch` re-emit every tick until acted on; `fab operator remove` is the ack (ties into today's step 5 Removals). `pane_mismatch` is report + remove, never a candidate.
- **Version-skew fallback, one line in §4**: if `tick-start --diff` errors as unknown flag (new skill vs old installed binary — bottle lag is a recorded recurring lesson), fall back to the flagless tick (full pane map + per-pane capture) for the session and report the mismatch once.
- **NOT touched**: §5 Auto-Nudge mechanics (per-pane `rk mux capture`, guards, indicator patterns, answer model) stay as today until Change 4 — this change must NOT wire in `fab pane questions`. §4's candidates list feeds §5's existing sweep population (it is computed to match today's waiting+idle policy exactly); the detection mechanics themselves are unchanged. § Notes and §6 Auto-Merge Choreography (C2) belong to their (already-merged) changes and are untouched.

### 3. `src/kit/skills/_cli-fab.md` (modify — constitution-mandated)

Update § fab operator tick-start (main line ~1218) with the `--diff` signature, the output contract (delivery classes, `candidates:`, `fleet:`, baseline-writer semantics, empty-set behavior), and the note that the flagless form is unchanged. Per the constitution's Additional Constraints, this ships simultaneously with the CLI change.

### 4. panemap.go (modify — small, same file)

Extract `collectPaneRows` + the snapshot seam var as described in §1. No behavior change to `fab pane map` (flagless path byte-identical; existing panemap tests keep passing).

### 5. Tests (extend src/go/fab/cmd/fab/operator_test.go or new file)

Today's tick-start tests start at operator_test.go:248 (`TestOperatorTickStart_IncrementsCount`; plan said :245ff — negligible drift). Per the plan's § Tests tick-start bullet:

- all event kinds from a seeded baseline + the named snapshot seam
- `completion`/`pane_death`/`pane_mismatch` **re-emit until `remove`**
- completion detected via display-state/terminal-stage predicate (stage-diff provably can't)
- baseline updated in the same write
- candidate ordering (waiting-first) + `idle_duration` presence
- `fleet:` block shape
- empty monitored ⇒ empty outputs (tick still increments)
- flagless path byte-identical

### Explicitly OUT of scope

- Phase A `fab operator note` verbs and C2 auto-merge choreography — already merged (#615, #616); dependencies, not deliverables.
- Phase B2 `fab pane questions` + skill §5 rewire — Change 4. No `pane questions` wiring anywhere in this change.
- C1 event wake (run-kit backlog), C3 `state --frame`, C4 — follow-ups.
- `docs/memory/runtime/pane-commands.md` — its Part-7-fence re-scope belongs to Change 4 (B2), not here.

## Affected Memory

- `runtime/operator.md`: (modify) Scoped to what B1 changes, checked against current main (Phase A + C2 lines already landed there — do not re-touch them): rewrite the **"Pane-map only"** Design Constraints bullet (the observation primitive is now the binary-internal snapshot via `tick-start --diff`; `fab pane map` remains the on-demand/manual surface); extend the **Full Mediation** decision's scope (first tick-mechanics offload — the binary now owns the per-tick diff and the observed-field baseline write); new design decision: **level-triggered vs consumed-on-read delta classes** (with the completion-can't-be-stage-diffed and pane_mismatch/recycled-pane-ID rationale); update the **§ Launcher `fab operator tick-start`** prose and the tick-lifecycle usage paragraph for `--diff`. Leave the "Hardcoded patterns" bullet and the two "operator's own detection policy" sentences to Change 4 (B2).

(`runtime/index.md` regenerates via `fab memory-index` at hydrate — not listed as a separate edit.)

## Impact

- **Code**: `src/go/fab/cmd/fab/operator_tick_start.go` (main change), `src/go/fab/cmd/fab/panemap.go` (small extraction), tests in `src/go/fab/cmd/fab/operator_test.go` or a new sibling `_test.go`. Go tests scope: `go test ./src/go/fab/cmd/fab/` (plus `./src/go/fab/internal/status/` if the terminal-stage predicate touches it).
- **Skills**: `src/kit/skills/fab-operator.md` (§4 only), `src/kit/skills/_cli-fab.md` (§ fab operator tick-start). Canonical sources only — never `.claude/skills/` (deployed copies).
- **Behavioral surface**: additive/back-compatible — old state files load fine (tolerant read), flagless `tick-start` unchanged, `fab pane map` unchanged. If review_fail diffing requires the baseline to carry last-known display state, the addition is an additive binary-owned field inside the owned `monitored` section (typed-write posture; no migration — the binary self-heals it on the first `--diff` write).
- **Base branch caveat**: this worktree (`racing-dugong`) is at 4c4ba90d, **behind origin/main** — #615 (notes) and #616 (C2) are on main but not in this checkout. The implementation branch must be based on (or rebased onto) updated main before apply, since the skill file and `operator_state.go` sections being edited here already changed in those merges.
- **Change type**: feat (new CLI capability + coupled skill rewire).
- **Sibling sweeps** (code-quality.md): the memory file documenting the skill's behavior (`runtime/operator.md`) is in the sweep class — covered above. Grep the old per-tick claims ("run `fab pane map --all-sessions --json` and read the state via `fab operator state`", step-6 `fab operator update` bookkeeping) repo-wide at apply to catch aggregate-spec restatements (e.g. `docs/specs/skills.md` §/fab-operator partial flow, glossary).

## Open Questions

- None — the plan is post-four-agent-review and the dispatch marks its decisions as settled; residual implementation choices are graded in Assumptions below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Two delivery classes: level-triggered `completion`/`pane_death`/`pane_mismatch` re-emitted until `fab operator remove`; consumed-on-read `stage_advance`/`review_fail` | Plan-settled post four-agent review (red-team + doctrine must-fix); dispatch marks settled | S:95 R:70 A:90 D:95 |
| 2 | Certain | Snapshot via extracted `collectPaneRows` helper + stubbable package-var seam in panemap.go (same `main` package; `rkPanesRunner`/`operatorStatePathOverride` precedent) | Plan-settled; anchors verified — extraction target is the discovery+resolve loop in `runPaneMap` | S:90 R:85 A:95 D:90 |
| 3 | Certain | Completion = terminal-stage-or-`stop_stage` predicate over display state, never a stage diff | Plan-settled; verified — `status.DisplayStage` (status.go:544) confirms only `display_state` flips at a terminal stage | S:95 R:75 A:95 D:95 |
| 4 | Certain | `pane_mismatch`: entry change ID vs snapshot `change` field; report + remove, never a candidate (recycled `%N` pane IDs, #612 contract) | Plan-settled; `paneRow.change`/`paneJSON.Change` anchor verified | S:90 R:75 A:90 D:90 |
| 5 | Certain | `candidates:` waiting-first then idle with `pane`/`change`/`agent_state`/`idle_duration`; monitored-only; unknown-state excluded by default | Plan-settled; matches today's §5 sweep policy on rk-less servers | S:90 R:80 A:90 D:90 |
| 6 | Certain | `fleet:` block (repo, session, stage, display_state, agent_state, idle_duration, pr_url) as the status frame's data source | Plan-settled (doctrine + scope reviews' biggest-hole fix) | S:90 R:75 A:90 D:90 |
| 7 | Certain | Baseline update in the same atomic mutation (`mutateOperatorState`); stale operator_tick_start.go:30 comment corrected; flagless path byte-identical; empty monitored ⇒ empty outputs with tick still incrementing | Plan-settled; anchors verified on main | S:90 R:80 A:90 D:90 |
| 8 | Certain | Coupled skill §4 rewrite ships in this change: tick rewired to `--diff` deltas + fleet-sourced frame, drops per-tick `fab operator state` read and step-6 `fab operator update` stage/agent bookkeeping; one-line version-skew fallback | Plan § Critical couplings — `--diff` is the authoritative baseline writer, or the next diff under-reports; constitution's simultaneous `_cli-fab.md` rule | S:95 R:70 A:90 D:95 |
| 9 | Certain | Scope fence: no `pane questions` wiring, §5 detection mechanics unchanged (Change 4); § Notes and §6 C2 untouched (merged in #615/#616) | Dispatch explicit; verified both landed on main | S:95 R:85 A:95 D:95 |
| 10 | Certain | Implementation branch bases on updated origin/main (includes #615/#616); racing-dugong checkout at 4c4ba90d is behind and needs fast-forward/rebase before apply | Verified via git: skill + operator_state.go sections this change edits already moved in those merges; mechanical git step with one obvious default | S:85 R:90 A:90 D:85 |
| 11 | Certain | Server scoping asymmetry accepted (`tick-start` keys off current socket); concurrency = atomic temp+rename, last-writer-wins under one operator per server | Plan § Edge cases, settled | S:90 R:80 A:90 D:90 |
| 12 | Confident | `--diff` stdout stays one parseable document: `tick:`/`now:` lines retained, followed by YAML `deltas:`/`candidates:`/`fleet:` blocks; exact key spelling decided at apply | Plan names the blocks and per-row fields but not the full serialization; skill is the sole consumer, easily revised, one consistent house style (operator verbs print YAML-ish stdout) | S:60 R:85 A:80 D:70 |
| 13 | Confident | `review_fail` consumed-on-read diffing carries last-known display state in the baseline as an additive binary-owned `monitored` field (typed-write posture; no migration) | Plan classifies review_fail as baseline-diffed but today's `monitoredEntry` stores only `stage`/`agent`; an additive owned-section field is the one back-compatible mechanism under the tolerant-read/typed-write posture | S:55 R:70 A:80 D:70 |
| 14 | Confident | §5's per-tick sweep population is fed by `--diff`'s `candidates:` (computed to match today's waiting+idle policy exactly); §5 capture/guard/answer mechanics untouched | Dispatch fences §5 mechanics to Change 4, but §4 no longer fetches the full map — candidates are the only population source that preserves the offload; identical policy by construction | S:65 R:80 A:80 D:75 |
| 15 | Confident | `fab operator update` remains for non-baseline edits (e.g. `stop_stage`); `--diff` owns stage/agent and touches `last_transition` on a stage change (preserving `update`'s semantics) | Plan-settled for the split; the `last_transition` detail is inference from `update`'s documented behavior | S:75 R:75 A:85 D:80 |

15 assumptions (11 certain, 4 confident, 0 tentative, 0 unresolved).
