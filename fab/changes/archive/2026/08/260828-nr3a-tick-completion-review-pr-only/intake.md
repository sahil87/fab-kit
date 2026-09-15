# Intake: Tick-Start Completion Fires Only at the Pipeline Terminus

**Change**: 260828-nr3a-tick-completion-review-pr-only
**Created**: 2026-08-28

## Origin

Backlog item `[nr3a]` (2026-08-28), one-shot `/fab-new nr3a`, no prior discussion in the conversation.

> bug: fab operator tick-start --diff emits spurious 'completion' delta for a change mid-pipeline at hydrate/ship (non-terminal stage, no stop_stage set). Repro: enroll a change with fab operator enroll (no --stop-stage), drive it through /fab-fff; once it reaches hydrate (stage 4/6) or ship (5/6) while display_state is 'active' (still running, .status.yaml shows e.g. hydrate: active, ship: pending, review-pr: pending), tick-start --diff still reports kind: completion for that stage every tick until the stage genuinely finishes. Only review-pr (terminal, stop_stage null) should ever emit completion. Observed live 2026-08-28 across 3 separate changes (run-kit b71j, 3o5d, 5jlp) on every hydrate/ship transition -- 100% reproducible, not flaky. Likely cause (unverified): the completion check probably treats any stage whose .status.yaml progress value is present/non-pending as done, rather than checking progress[stage] == done specifically -- needs comparing against actual stage completion state, not just presence/dispatch-started. Fix should live in the tick-start diff computation (cmd/fab or internal/operator, wherever tick-start's completion classification lives) so operators don't have to manually cross-check .status.yaml every tick to avoid prematurely running 'fab operator remove' on a change that's still mid-pipeline.

**Gap analysis (intake-time, verified in source)**: the backlog's "likely cause" is wrong in mechanism but right in effect. The classification lives in `src/go/fab/cmd/fab/operator_tick_start.go`:

```go
// tickTerminalStages is the completion set when an entry has no stop_stage —
// today's §4 step-2 terminal policy (a fully-done pipeline reads
// review-pr/done and is contained in it).
var tickTerminalStages = map[string]bool{"hydrate": true, "ship": true, "review-pr": true}

func tickCompleted(entry monitoredEntry, stage, displayState string) bool {
	if entry.StopStage == nil {
		return tickTerminalStages[stage]          // ← display_state never consulted
	}
	oi, oStop := stageOrderIndex(stage), stageOrderIndex(*entry.StopStage)
	if oi < 0 || oStop < 0 {
		return false
	}
	if oi > oStop {
		return true
	}
	return oi == oStop && (displayState == "done" || displayState == "skipped")
}
```

With `stop_stage: null` the predicate is a pure stage-set membership test — `hydrate/active` and `ship/active` are "complete" by construction. This is an inherited **policy**, not a parsing slip: the "terminal set {hydrate, ship, review-pr}" is written into `docs/memory/runtime/operator.md` (§ Removal triggers, § Tick behavior step 2, the removal-path Design Decision), `src/kit/skills/fab-operator.md` (§ Stop stage: "full pipeline: hydrate/ship/review-pr are terminal"; tick step 1 / step 5), and `src/kit/skills/_cli-fab.md` § fab operator tick-start "Detection semantics". The existing test `TestOperatorTickDiff_CompletionPredicateBranches` (`operator_tick_diff_test.go`) encodes the buggy behavior — row `s005` seeds `ship/active` and asserts a `completion` delta.

The policy predates `/fab-fff` being the operator's default spawn command: when `/fab-ff` (stops at hydrate) was the common terminus, treating hydrate as terminal made a parked ff run complete. Under `/fab-fff`, hydrate and ship are ordinary mid-pipeline stages and the stage-set test fires the moment the change *enters* them.

## Why

1. **Pain point**: every `tick-start --diff` tick reports `kind: completion` for a change that is still running hydrate or ship. The operator skill treats `completion` as the trigger to report the change done and `fab operator remove` it (level-triggered; `remove` is the ack). Following the skill literally removes a change from the monitored set 2 stages early — it then loses ship/review-pr monitoring, question auto-answering, stuck detection, and (in autopilot) the `advance` + merge choreography is triggered against a change whose PR does not yet exist.
2. **Consequence if unfixed**: operators must manually cross-check `.status.yaml` on every completion delta before acting (exactly the toil the diff path was built to remove), and any operator that trusts the contract removes changes prematurely. Observed 100% reproducible on 2026-08-28 across three run-kit changes on every hydrate/ship entry.
3. **Approach**: fix the predicate at its source — `tickCompleted`'s `stop_stage: null` branch becomes a *display-state* check at the true pipeline terminus (`review-pr` with `display_state` `done` or `skipped`), mirroring the already-correct at-the-stop branch. The `{hydrate, ship, review-pr}` terminal set is deleted, not patched (adding a `display_state` check on top of the set would still race: a finished hydrate auto-activates ship in the same sequencer step, so a `hydrate/done` snapshot is a transient window in a `/fab-fff` run, and the same "equality alone would race" reasoning already in the code comment applies). Callers that legitimately park earlier — `/fab-ff` runs stop at hydrate — express that through `stop_stage`, which is what the field exists for.

## What Changes

### 1. `tickCompleted` null-`stop_stage` branch (`src/go/fab/cmd/fab/operator_tick_start.go`)

Replace the terminal-set lookup with a terminus + display-state predicate:

```go
// tickTerminusStage is the pipeline terminus — the only stage at which an
// entry with no stop_stage completes. Completion is a display-state check
// at that stage (done/skipped), never bare stage membership: a change
// entering hydrate or ship under /fab-fff is mid-pipeline, not complete.
const tickTerminusStage = "review-pr"

func tickCompleted(entry monitoredEntry, stage, displayState string) bool {
	if entry.StopStage == nil {
		return stage == tickTerminusStage && (displayState == "done" || displayState == "skipped")
	}
	// stop_stage-set branch unchanged
	...
}
```

- `tickTerminalStages` (the map) is removed along with its comment; nothing else references it (verified by grep — `stageOrderIndex` has its own ordered slice).
- The `stop_stage`-set branch is byte-identical to today.
- The `done || skipped` acceptance duplicates the at-the-stop branch's test; extract a tiny `stageFinished(displayState string) bool` helper so both branches share it (the review-pr stage is `skipped` when a change ships with review-pr disabled, so `skipped` must count as finished at the terminus too).
- `review-pr/active` (e.g. waiting on a Copilot review) is **not** complete. `review-pr/pending` cannot occur as the resolved stage (the active stage is what the snapshot resolves), but it is not `done`/`skipped` either, so it is trivially excluded.

Resulting truth table for `stop_stage: null`:

| Snapshot (`stage`/`display_state`) | Today | After |
|---|---|---|
| `hydrate/active` | completion | — |
| `hydrate/done` (transient, `/fab-fff` between finish and ship start; or a parked `/fab-ff` run) | completion | — |
| `ship/active` | completion | — |
| `review-pr/active` | completion | — |
| `review-pr/done` | completion | completion |
| `review-pr/skipped` | completion | completion |
| `apply/active` | — | — |

### 2. Tests (`src/go/fab/cmd/fab/operator_tick_diff_test.go`)

`TestOperatorTickDiff_CompletionPredicateBranches`:
- Change `s005` from `ship/active ⇒ completion` to `ship/active ⇒ NO completion` (it becomes a regression guard for this bug).
- Add null-`stop_stage` rows: `hydrate/active` ⇒ no completion; `hydrate/done` ⇒ no completion (the `/fab-fff` transient); `review-pr/active` ⇒ no completion; `review-pr/done` ⇒ completion; `review-pr/skipped` ⇒ completion.
- `s006` (`apply/active` ⇒ none) and all `stop_stage`-set rows (`s001`–`s004`) stay as-is; the `stop_stage`-set branch is untouched.

Check `TestOperatorTickDiff_AllEventKinds` and `TestOperatorTickDiff_LevelTriggeredReEmitUntilRemove` — if either seeds its `completion` sample via `hydrate`/`ship` with a null `stop_stage`, move that fixture to `review-pr/done` (or give it a `stop_stage`) so the event-kind coverage keeps exercising `completion`.

### 3. Skill + CLI contract text (`src/kit/skills/`)

Reword every "terminal stage (hydrate, ship, review-pr)" statement to the new rule — completion with `stop_stage: null` = **`review-pr` reached `done`/`skipped`**:

- `_cli-fab.md` § fab operator tick-start, "Detection semantics" bullet: `with stop_stage: null the terminal set is {hydrate, ship, review-pr}` → `with stop_stage: null it fires only at the pipeline terminus — review-pr with display_state done/skipped; hydrate and ship are mid-pipeline and never complete an entry by themselves`.
- `fab-operator.md`:
  - § Stop stage: `Default is null (full pipeline: hydrate/ship/review-pr are terminal)` → `Default is null (full pipeline: complete when review-pr is done/skipped). Spawns that deliberately park earlier — e.g. a /fab-ff run, which stops after hydrate — MUST enroll with --stop-stage hydrate, otherwise the entry never completes and sits in the monitored set until the user stops it.`
  - § Removal (`or a terminal stage if stop_stage is null`), tick step 1 `completion` bullet (`terminal stage, or at/past the entry's stop_stage`), tick step 5 (`reached stop stage or terminal stage`), the status-frame legend row `complete | reached terminal/stop stage`, and the cross-repo barrier wording in § Dependency Resolution + § Cross-repo resolution (`a terminal stage when stop_stage is null`) → say "the pipeline terminus (review-pr done/skipped) when `stop_stage` is null".
- Sweep for `terminal stage` / `terminal set` / `hydrate, ship, review-pr` / `hydrate/ship/review-pr` phrase-class variants across `src/kit/skills/*.md` (contrastive phrasing included), so no stale claim survives — the sweep is the recurring miss-class for this repo.

No CLI signature changes; `fab operator tick-start --diff` output shape (`kind`, `change`, `pane`, `stage`, `display_state`) is unchanged — only *when* `completion` appears changes.

### 4. Memory (hydrate stage, `docs/memory/runtime/operator.md`)

Present-truth updates for the same three claim sites (§ Removal triggers, § Tick behavior step 2 "Pipeline completion", removal-path Design Decision) plus the cross-repo-barrier line, and a new Design Decision entry (four-field shape) recording: **Decision** — null-`stop_stage` completion is `review-pr` done/skipped only; **Why** — the `{hydrate, ship, review-pr}` set fired on stage *entry* under `/fab-fff` (100% reproducible premature removals, 2026-08-28); **Rejected** — keeping the set and adding a `display_state` check (still races the hydrate→ship / ship→review-pr auto-activation window); **Introduced by** — `260828-nr3a`.

### 5. Operator skill: one canonical definition of "dependency satisfied" (`src/kit/skills/fab-operator.md`)

Today the skill never defines when a `depends_on` entry is *satisfied*, and the two tiers drift:

- **Same-repo** (§ Same-repo resolution step 1) gates only on the dep's branch *resolving* — from the monitored entry or `branch_map`. `branch_map` is written by `fab operator enroll` at spawn, so a same-repo dep "resolves" the instant its agent starts, with no commits on the branch; an explicit `depends_on` on a still-running dep cherry-picks whatever is there. Autopilot masks this only through queue ordering (step 5 `completion` → step 6 spawn next) — the very signal this change shows firing two stages early.
- **Cross-repo** (§ Dependency Resolution bullet 2, § Cross-repo resolution) waits for "`stop_stage` (a terminal stage when null)" — the same ambiguous terminal wording area 3 removes.
- Nothing ties completion to **the PR existing**; autopilot step 5 "collect the PR URL" assumes it, and `stacked-prs`' `gh pr edit --base <dep-branch>` retarget needs the dep pushed.

Add a **Dependency satisfied** paragraph at the top of § Dependency Resolution and make every consumer point at it:

> **Dependency satisfied** — a `depends_on` entry is satisfied when the dependency's **pipeline has completed**: its monitored entry has emitted (or would emit) a `completion` delta — `review-pr` done/skipped when `stop_stage` is null, or at/past its `stop_stage` — **and**, for a same-repo dependency with a null `stop_stage`, its PR exists (`gh pr view <dep-branch> --json url` succeeds; the branch is therefore pushed and stable). Neither enrollment, a `branch_map` entry, nor the branch being minted is satisfaction — those exist from the moment the dep's agent spawns. An unsatisfied dependency **holds the spawn** in both tiers (re-checked each tick), logging `"{change}: waiting on dependency {dep} ({dep.repo}) to complete."`.

Consumers to reword to "satisfied per § Dependency satisfied":
- § Dependency Resolution cross-repo bullet + § Cross-repo resolution (replace the `stop_stage`/terminal-stage phrasing; keep the wait/hold/log behavior, swap the log line to the shared one above).
- § Same-repo resolution: insert a **step 0.5 readiness gate** before step 1 — if the dep is still in the monitored set and not satisfied, hold the spawn (do not proceed to branch lookup / cherry-pick). Step 1's "branch not found → escalate" stays for the genuinely-missing case. The `stacked-prs` variant (`wt create --checkout <dep-branch>`) inherits the same gate.
- § Autopilot per-change loop step 5/6: "on completion" → "when the current change is satisfied per § Dependency satisfied (its `completion` delta observed **and** PR URL collected)"; step 6 spawns only then. `merge-auto` already defers further (to the verified merge) — unchanged.
- § Watches cross-repo barrier line (§6 "wait until the dependency reaches its `stop_stage`") and the `depends_on` comment in the state-file example (`# change IDs — same-repo deps cherry-pick, cross-repo deps are ordering-only (§6)`) → append "(both gated on § Dependency satisfied)".

Memory: `runtime/operator` § Dependency resolution / cross-repo barrier Design Decision (currently "wait until the dependency reaches its `stop_stage` (terminal stage when `stop_stage` is null)") gets the same definition, and the new nr3a Design Decision entry records it as part of the fix (Rejected: "branch resolvable ⇒ satisfied", the implicit status quo).

## Affected Memory

- `runtime/operator`: (modify) completion/removal trigger is review-pr done/skipped (not the hydrate/ship/review-pr terminal set); parked `/fab-ff` spawns use `--stop-stage hydrate`; canonical "dependency satisfied" definition (pipeline completed + PR exists — never branch minted); new Design Decision entry

## Impact

- **Code**: `src/go/fab/cmd/fab/operator_tick_start.go` (predicate + removed map, ~10 lines), `src/go/fab/cmd/fab/operator_tick_diff_test.go` (fixture rewrite + new rows). Relevant test run: `go test ./src/go/fab/cmd/fab/ -run 'TestOperatorTickDiff|TestOperatorTickStart'`.
- **Skills**: `src/kit/skills/_cli-fab.md`, `src/kit/skills/fab-operator.md` (prose plus one added readiness-gate step in Same-repo resolution; no tool/sub-agent change).
- **Memory**: `docs/memory/runtime/operator.md`; `fab memory-index` regen.
- **Behavior contract**: operators running the deployed `/fab-operator` skill stop seeing `completion` for mid-pipeline changes; a change spawned with `/fab-ff` and enrolled with `stop_stage: null` will no longer auto-complete at hydrate — it must be enrolled with `--stop-stage hydrate` (skill text updated to say so). No autopilot code path changes: autopilot spawns run `/fab-fff` and consume the same `completion` delta.
- **Release**: PATCH (`fix`); no CLI signature change, `_cli-fab.md` semantics bullet updated per the constitution's CLI constraint.

## Open Questions

- None blocking. See Assumptions #3 (parked `/fab-ff` runs now require `--stop-stage hydrate`) and #5 (PR-exists is part of same-repo satisfaction).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Root cause is `tickCompleted`'s null-`stop_stage` branch returning bare `tickTerminalStages[stage]` (set `{hydrate, ship, review-pr}`), not a `.status.yaml` progress-presence check as the backlog guessed | Read directly from `operator_tick_start.go:84-87,243-245`; the existing test seeds `ship/active` and asserts completion | S:85 R:90 A:95 D:95 |
| 2 | Confident | Null-`stop_stage` completion = `review-pr` AND `display_state ∈ {done, skipped}` (not bare `stage == review-pr`) | Backlog says "only review-pr should ever emit completion"; requiring done/skipped mirrors the existing at-the-stop branch and avoids firing while review-pr awaits Copilot; `skipped` covers review-pr-disabled ships | S:70 R:85 A:80 D:70 |
| 3 | Confident | Delete the terminal set entirely rather than keep `{hydrate, ship}` with an added `display_state` check; parked `/fab-ff` spawns must enroll with `--stop-stage hydrate` (skill text updated) | A `hydrate/done` snapshot is a transient window in `/fab-fff` (finish auto-activates ship) — the code's own comment explains this race for the at-the-stop case; `stop_stage` is the field built for early parking. Reversible via `/fab-clarify` or a follow-up if a parked-ff heuristic proves needed | S:45 R:75 A:65 D:50 |
| 4 | Certain | Scope: code + test + `_cli-fab.md`/`fab-operator.md` prose + `operator.md` memory; no CLI signature or output-shape change; `change_type: fix`, PATCH release | Constitution requires `_cli-fab.md` updates for changed CLI semantics; output YAML shape untouched | S:80 R:90 A:90 D:90 |
| 5 | Confident | "Dependency satisfied" = pipeline completed (`completion` delta) AND, for same-repo null-`stop_stage` deps, PR exists — defined once and referenced by both tiers, autopilot step 5/6, and watches; same-repo gains a readiness gate that holds the spawn | User asked for exactly this clarification; skill today gates same-repo deps on branch resolvability only (`branch_map` is written at enroll = spawn); PR-exists is the cheapest observable proof the branch is pushed and stable for cherry-pick / stacked retarget. Skill-prose only, reversible | S:65 R:80 A:70 D:65 |

5 assumptions (2 certain, 3 confident, 0 tentative, 0 unresolved).
