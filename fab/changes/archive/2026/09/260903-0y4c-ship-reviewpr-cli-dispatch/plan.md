# Plan: Ship/Review-PR CLI-Adapter Dispatch Wiring

**Change**: 260903-0y4c-ship-reviewpr-cli-dispatch
**Intake**: [intake.md](intake.md)

## Requirements

### Dispatch Sites: fab-fff Steps 4–5

- **R1** — `src/kit/skills/fab-fff.md` Step 4 (Ship) MUST branch on the `dispatch:` key of the
  already-required `fab agent ship -o yaml` resolution. Absent ⇒ today's native-subagent path
  verbatim (two seams, `/git-pr {name}` prompt, same `{name}` contract). Present ⇒ the CLI adapter
  per `_preamble.md` § CLI-Adapter Dispatch, referenced (not restated): branch on `dispatch.rung`
  (`headless` ⇒ `start` with the stage prompt on stdin; `pane` ⇒ `open` → readiness gate →
  `deliver`), then blocking `fab dispatch wait`, state handling, and reap at done-read.
  - GIVEN `agent.workers: claude` (dispatch: absent), WHEN fab-fff Step 4 runs, THEN behavior is
    byte-equivalent to today's native dispatch.
  - GIVEN a resolution with `dispatch: {rung: pane}`, WHEN Step 4 runs, THEN `/git-pr` executes in
    a pane worker via open → gate → deliver, and no native subagent is spawned.
- **R2** — `src/kit/skills/fab-fff.md` Step 5 (Review-PR) MUST gain the identical branch, with the
  synchronous-poll directive baked into the dispatched prompt **on all arms** (retained verbatim;
  it MAY note that a pane worker cannot yield, making the directive moot-by-construction there).
  - GIVEN `dispatch:` present at Step 5, WHEN the review-pr worker runs in a pane, THEN the
    Copilot poll completes inside the pane worker and the orchestrator observes via `fab dispatch wait`.
- **R3** — fab-fff's Behavior note (the "fff-only delta" paragraph) MUST be rewritten: the delta is
  that Steps 4–5 dispatch full `/git-pr` / `/git-pr-review` behaviors with **self-managed
  transitions** (their prompts do not carry the block-contract transition prohibition); the
  dispatch *arm* is no longer part of the delta — Steps 4–5 use the same two-branch rule as
  Steps 1–3.
- **R4** — fab-fff Error Handling (and the Step 4/5 failure prose) MUST map CLI-arm observations
  onto the existing outcomes, adding no new recovery rules:

  | Observed | Action |
  |----------|--------|
  | `done` + ship result `status: failure` | Existing row: STOP with the reported `reason`; stage stays `active` for user retry |
  | `done` + review-pr result `outcome: failure` | Existing row: STOP with the reported `reason`; stage stays `active` for user retry |
  | `done` + review-pr result `outcome: timeout` | Existing timeout row: stage deliberately left `active`, report the exact pending message, stop |
  | `done` + review-pr result `outcome: no-reviews` | Successful no-op — stage finished `done` by the worker |
  | `failed` / `failed (no-result)` / `orphaned` | Canon recovery table (`_preamble.md` § CLI-Adapter Dispatch) — pointer, not restatement |

### Dispatch Sites: /fab-continue ship and review-pr rows

- **R5** — `src/kit/skills/fab-continue.md`'s `ship`/`active` dispatch row MUST replace "apply the
  two seams — mirroring `/fab-fff` Step 4" with the same two-branch rule (still mirroring Step 4,
  which now branches). The only-if-still-active `finish` guard after the behavior returns is
  unchanged and applies on both arms.
- **R6** — The `review-pr`/`active` and `review-pr`/`failed` rows MUST gain the same branch (the
  failed row's `start`-based recovery and the timeout/fail only-if-still-active guards unchanged).
  - GIVEN `progress.review-pr == failed` and a `dispatch:`-present resolution, WHEN the row
    re-executes `/git-pr-review {name}` via the CLI adapter, THEN the worker's own Step 0
    `fab status start` performs the failed→active recovery exactly as on the native arm.

### Contract: dispatched ship/review-pr prompts (self-managing workers)

- **R7** — `_preamble.md` § Dispatch-Prompt Obligations MUST be extended (it is the owner; sites
  point): dispatched ship/review-pr prompts carry obligations 1–3 (result file, standard context
  files, terminal `fab status refresh`) but NOT the transition prohibition — these workers
  self-manage their **own** stage's `fab status` start/finish/fail exactly as the standalone
  skills do; the orchestrator still owns sequencing and never runs a transition for a stage whose
  worker owns it (the existing only-if-still-active guards are the seam). Minimal result schemas
  are added beside the existing three:

  ```yaml
  # ship (mirrors "returns PR URL or error")
  stage: ship
  status: success            # success | failure — worker/infra outcome
  pr_url: "https://github.com/{owner}/{repo}/pull/{N}"
  summary: "committed, pushed, draft PR created"
  # on failure only:
  reason: "push rejected: …"
  ```

  ```yaml
  # review-pr (mirrors git-pr-review's four-class Step 6 outcome)
  stage: review-pr
  status: success            # worker/infra outcome
  outcome: success           # success | failure | no-reviews | timeout — the Step 6 outcome class
  summary: "3 comments triaged: 2 fixed, 1 deferred"
  # on outcome: failure only:
  reason: "no PR found on this branch"
  ```

  The `status` vs `outcome` split mirrors the existing `status` vs `verdict` split for review:
  `outcome: timeout` and `outcome: failure` are dispatch-state `done` (result present), never
  dispatch-state `failed`.
- **R8** — `docs/specs/harness-adapters.md` § Dispatch-prompt obligations MUST state the
  stage-scoped carve-out: the block-contract transition prohibition is scoped to
  `/fab-continue`-behavior workers (apply/review/hydrate); ship/review-pr workers self-manage
  their own stage's transitions on every adapter. The adapter-coverage prose ("Every post-intake
  stage is executed by dispatching a worker…") MUST remain accurate — adjust only if it now
  under- or over-claims.
- **R9** — `docs/specs/stage-models.md` § Skill wiring's ship/review-pr bullet (the "only the
  model/effort seam is added" paragraph, currently lines ~711–717) MUST be rewritten to the
  two-branch rule; the "closes the caller asymmetry" history and the fast-multi-referent rationale
  stay.

### Non-Goals

- No Go/runtime change — `fab dispatch` is already stage-generic (`stageRoles` carries `ship` and
  `review-pr`; verified live). No `_cli-fab.md` change (no CLI surface change), no tests triggered
  by the CLI⇒docs+tests rule.
- Light lane untouched: ship/review-pr remain inline there (`_pipeline.md` § Light Lane); only the
  full-lane/standalone dispatched forms change.
- No change to `/git-pr` / `/git-pr-review` core behavior (their internal steps, guards, and
  transitions are untouched; they gain at most a dispatched-form note).
- Worker Continuation stays apply-only; ship/review-pr workers are never named or continued, and
  their panes reap immediately at done-read (the existing "every other stage" row).
- The config-side workaround (`agent.profiles.fast.provider: claude`) is documented nowhere new —
  orthogonal and untouched.

### Design Decisions

- **Decision**: Ship/review-pr dispatch sites branch on `dispatch:` presence, exactly like
  Steps 1–3. **Why**: makes the resolver's surfaced answer actionable at every site; zero behavior
  change when native resolves; fixes the fast-role inversion (session-model inherit on non-native
  workers providers). **Rejected**: cross-provider fallback to claude's fills (the resolver never
  substitutes providers — verbatim pass-through is a design invariant); config-only per-machine
  override (treats the symptom at one site on one machine). *Introduced by 0y4c.*
- **Decision**: Dispatched ship/review-pr workers keep self-managed transitions. **Why**: matches
  the native-subagent form and the standalone skills byte-for-byte; the only-if-still-active
  guards already reconcile orchestrator and worker. **Rejected**: orchestrator-owned transitions
  (would fork `/git-pr`'s behavior by caller and require a larger refactor of its Step 0/4/6
  internals). *Introduced by 0y4c.*
- **Decision**: The review-pr result file carries the four-class `outcome`
  (success | failure | no-reviews | timeout) alongside `status`, plus `reason` on
  `outcome: failure`. **Why**: the timeout is an
  outcome, not an infra failure — it must map to the existing leave-active + pending-message path;
  reusing git-pr-review's own Step 6 class set adds no new taxonomy. **Rejected**: overloading
  `status: failure` for timeout (would route a healthy outcome into the recovery/restart budget).
  *Introduced by 0y4c.*

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rewrite `src/kit/skills/fab-fff.md` Step 4 (Ship): two-branch dispatch on `dispatch:` presence — native branch verbatim today's text; CLI branch pointing at `_preamble.md` § CLI-Adapter Dispatch (rung branch, wait, state table, done-read reap) with the ship result schema named <!-- R1 -->
- [x] T002 Rewrite `src/kit/skills/fab-fff.md` Step 5 (Review-PR): same branch; synchronous-poll directive retained in the dispatched prompt on all arms with the moot-on-pane note <!-- R2 -->
- [x] T003 Rewrite fab-fff.md's Behavior note ("fff-only delta") and extend Error Handling with the R4 mapping rows <!-- R3, R4 -->
- [x] T004 Rewrite `src/kit/skills/fab-continue.md` ship/`active`, review-pr/`active`, review-pr/`failed` rows to the two-branch rule (guards unchanged) <!-- R5, R6 -->
- [x] T005 Extend `src/kit/skills/_preamble.md` § Dispatch-Prompt Obligations: ship/review-pr result schemas + the self-managed-transitions carve-out (owner text); verify the reap table's "every other stage" row needs no wording change <!-- R7 -->
- [x] T006 [P] Amend `docs/specs/harness-adapters.md` § Dispatch-prompt obligations (stage-scoped carve-out) + verify the adapter-coverage prose <!-- R8 -->
- [x] T007 [P] Rewrite `docs/specs/stage-models.md` § Skill wiring ship/review-pr bullet to the two-branch rule <!-- R9 -->

### Phase 3: Integration & Edge Cases

- [x] T008 Sweep the sibling class: `src/kit/skills/_pipeline.md` light-lane contrast wording ("dispatch exactly as written below"), `docs/specs/skills.md` fab-fff/fab-continue sections, `docs/specs/glossary.md`, `src/kit/skills/git-pr.md` / `git-pr-review.md` (add a dispatched-form note only if they claim native-only), and a repo-wide grep for "both seams" / "native model/effort seams" / "only the model/effort seam" — update every occurrence describing ship/review-pr dispatch <!-- R1-R9 -->
- [x] T009 Run `fab sync` so deployed `.claude/skills/` copies match the edited sources; verify no drift-guard test references the rewritten spec passages (`go test ./src/go/fab/... -run 'Drift|Spec'` scoped first) <!-- R1-R9 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: fab-fff Step 4 branches on `dispatch:` presence; native branch text is behavior-identical to today's; CLI branch references (never restates) the canon procedure
- [x] A-002 R2: fab-fff Step 5 branches identically; synchronous-poll directive present in the dispatched prompt text on all arms
- [x] A-003 R3: the Behavior note's delta is self-managed transitions only, not the dispatch arm
- [x] A-004 R4: Error Handling maps ship `status: failure` / review-pr `outcome: failure` / `outcome: timeout` / `outcome: no-reviews` / infra states to existing outcomes with no new recovery rules
- [x] A-005 R5, R6: all three fab-continue rows branch; only-if-still-active guards and the failed-row `start` recovery unchanged
- [x] A-006 R7: `_preamble.md` obligations carry ship/review-pr schemas + the carve-out, stated once (owner), with sites pointing

### Behavioral Correctness

- [x] A-007 R1, R5: with `dispatch:` absent, every rewritten site prescribes today's native path verbatim (zero blast radius for claude workers)
- [x] A-008 R7: `outcome: timeout`/`failure` are documented as dispatch-state `done`, never `failed`
- [x] A-009 R8, R9: both specs updated; no spec still claims ship/review-pr are native-seams-only

### Scenario Coverage

- [x] A-010 R1: pane-rung scenario — Step 4 text walks open → gate → deliver → wait → done-read → reap for ship
- [x] A-011 R6: review-pr failed-row scenario — CLI-arm re-execution relies on the worker's own `fab status start` recovery

### Edge Cases & Error Handling

- [x] A-012 R4: `failed (no-result)` on ship/review-pr routes to the canon escalate-never-done rule via pointer
- [x] A-013 R7: a steered pane worker still owes result file + refresh (contract-neutral steering holds for self-managing stages)

### Code Quality

- [x] A-014 Owner-or-pointer: no rewritten site restates the CLI-Adapter Dispatch procedure or the carve-out it points at (code-quality.md anti-pattern 4)
- [x] A-015 Sibling sweep completed up front: "both seams"-class grep clean across `src/kit/skills/` + `docs/specs/`
- [x] A-016 Edits under `src/kit/skills/` only (never `.claude/skills/`), `fab sync` run after

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Ship result schema carries `pr_url` + failure `reason`; review-pr carries four-class `outcome` + failure `reason` | Mirrors each skill's existing return contract ("Returns PR URL or error"; Step 6's outcome classes) — no new taxonomy | S:70 R:75 A:80 D:75 |
| 2 | Confident | git-pr.md / git-pr-review.md themselves need at most a dispatched-form note (obligations ride the dispatch prompt, not the skill files) | The prompt composer (fab-fff/fab-continue) owns the obligations; the skills' standalone behavior is unchanged | S:65 R:80 A:75 D:70 |
| 3 | Certain | `_pipeline.md` § Stage Dispatch Procedure itself is NOT extended to ship/review-pr (fab-fff Steps 4–5 stay driver-local, mirroring today's structure) | The bracket's procedure is scoped to apply/review/hydrate by design; fab-fff owns its Steps 4–5 text — structure preserved | S:80 R:85 A:90 D:85 |
| 4 | Certain | `_preamble.md`'s reap table needs no wording change — ship/review-pr fall under the "every other stage" immediate-reap row literally | The row is exhaustive by complement (everything but apply); verified at § CLI-Adapter Dispatch step 3 during T005 | S:85 R:90 A:95 D:90 |
| 5 | Confident | `git-pr.md` needed no dispatched-form note; `git-pr-review.md`'s synchronous-poll note was extended to name both dispatched forms and the pane arm's mootness | git-pr.md is grep-clean of dispatch-form claims; git-pr-review.md's "dispatched subagent" phrasing implied native-only, so the note now covers the CLI-adapter arms | S:70 R:80 A:75 D:70 |
| 6 | Certain | `_pipeline.md`, `fab-ff.md`, `docs/specs/skills.md`, and `docs/specs/glossary.md` needed no edits — none claim ship/review-pr dispatch is native-seams-only | Sweep grep found only lane-locus claims ("dispatched in the full lane, inline in the light lane") that stay true under both arms; remaining "two seams"-class prose lives in `docs/memory/` (hydrate's scope — including `_shared/context-loading.md`, which the intake's Affected Memory list did not name) | S:80 R:85 A:85 D:85 |

3 assumptions (1 certain, 2 confident, 0 tentative, 0 unresolved) planned; 3 added at apply (2 certain, 1 confident).

## Deletion Candidates

- None — this change adds new dispatch branches without making existing code or prose redundant beyond the in-place rewrites it already performs
