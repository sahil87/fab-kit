# Intake: Ship/Review-PR CLI-Adapter Dispatch Wiring

**Change**: 260903-0y4c-ship-reviewpr-cli-dispatch
**Created**: 2026-09-03

## Origin

> Wire the ship and review-pr dispatch sites (fab-fff Steps 4–5 and /fab-continue's ship/review-pr rows) through the CLI-adapter dispatch branch so a resolved dispatch: mapping is honored — pane/headless git-pr and git-pr-review workers with self-managed transitions plus result-file and refresh obligations — instead of silently inheriting the session model when agent.workers is a non-native provider

Conversational (`/fab-discuss` session, 2026-09-03). The user observed a live ship-stage worker
(`ship-7ajq`, run-kit `glazed-gopher` worktree) running **Fable 5 at high effort** — ~90.7k tokens
of the session's most expensive model on commit/push/PR mechanics — and asked why it wasn't on the
fast tier. Root-cause investigation traced the full resolution chain (below); the user chose the
**design fix** over the config-side workaround and approved this intake.

Key decisions from the discussion:

- Wire **both** fab-fff Steps 4–5 **and** `/fab-continue`'s ship/review-pr rows through the
  `dispatch:`-presence branch (fixing only fab-fff would leave the standalone path with the gap).
- **Keep self-managed transitions** — `/git-pr` and `/git-pr-review` continue to run their own
  `fab status` start/finish/fail; the block-contract transition prohibition is NOT imposed on these
  workers. The dispatched form gains the other two prompt obligations (result file + terminal refresh).
- Define the **failure mapping** explicitly (dispatch states → the existing retry story).
- **Light lane untouched** — ship/review-pr stay inline there.
- The config-side escape hatch (`agent.profiles.fast.provider: claude`) remains available and
  orthogonal; it is NOT part of this change.

## Why

**The problem.** `fab agent ship -o yaml` / `fab agent review-pr -o yaml` can return an actionable
`dispatch:` mapping (e.g. `{rung: pane, command: kimi --auto}`), and the skill contract requires the
dispatch site to *surface* that YAML — but fab-fff Steps 4–5 then ignore it. Line 46 of
`src/kit/skills/fab-fff.md` makes this explicit: "The fff-only delta is that Steps 4–5 dispatch full
`/git-pr` and `/git-pr-review` behaviors through the native model/effort seams." The native arm can
only honor Claude resolutions; handed a non-Claude one, the seam contract degrades to
"empty `model_alias` ⇒ omit the parameter ⇒ inherit the session model."

**Observed consequence (2026-09-03, live).** With system config `agent.workers: kimi` +
`dispatch.mode: pane`: apply, review, and hydrate ran as kimi pane workers
(`.fab-dispatch/7ajq/` holds their records), but ship — resolved to
`provider: kimi, model: "", effort: "", dispatch: {rung: pane, command: kimi --auto}` (kimi
deliberately ships zero fills, `defaults.yaml:159`) — was dispatched as a native subagent and
inherited the session model: **Fable 5 at high effort on the pipeline's cheapest-intent stage**.
The `fast` role exists precisely to run ship cheap; this silently inverts it. review-pr
(role `doing`, `agent.go:304`) has the identical failure shape under a non-native workers provider.

**If we don't fix it:** every non-claude `agent.workers` setup silently pays session-model prices
for ship and review-pr, the "surface the resolved YAML" rule stays decorative at these two sites,
and the pipeline's adapter story is inconsistent (four stages honor `dispatch:`, two don't).

**Why this approach over the alternative.** The config override
(`agent.profiles.fast.provider: claude`) treats the symptom per-machine and only for ship. The
design fix makes the resolver's answer mean the same thing at every dispatch site — native when
`dispatch:` is absent (zero behavior change for claude-workers setups), CLI adapter when present —
which is the existing, canonical branch rule (`_preamble.md` § CLI-Adapter Dispatch). Bonus: on the
pane arm, `/git-pr-review`'s 10-minute Copilot poll runs inside a pane worker that *cannot yield*,
making the synchronous-poll directive moot-by-construction there (the mid-poll yield was a real
incident — PR #610 run).

## What Changes

### 1. fab-fff Steps 4–5 — branch on `dispatch:` presence

`src/kit/skills/fab-fff.md` Step 4 (Ship) and Step 5 (Review-PR) currently read
"…and dispatch via both seams; then dispatch `/git-pr` [`/git-pr-review`] as subagent…".
Each step becomes a two-branch dispatch, exactly parallel to `_pipeline.md` § Stage Dispatch
Procedure's rule:

- **`dispatch:` absent** — today's behavior verbatim: native subagent through the two model/effort
  seams, same prompt, same `{name}` argument contract, same synchronous-poll directive (Step 5).
- **`dispatch:` present** — CLI adapter per `_preamble.md` § CLI-Adapter Dispatch: `rung: headless`
  ⇒ `fab dispatch start <change> <stage>` with the stage prompt on stdin; `rung: pane` ⇒
  `open` → readiness gate → `deliver`; then blocking `fab dispatch wait`, state handling, and reap
  at done-read (ship and review-pr fall under the existing "every other stage" immediate-reap row —
  no continuation, no naming; § Worker Continuation stays apply-only).

The Behavior note (fab-fff.md line 46) is rewritten: the fff-only delta becomes "Steps 4–5 dispatch
the full `/git-pr` / `/git-pr-review` behaviors (self-managed transitions)"; the *arm* is no longer
part of the delta.

### 2. /fab-continue ship and review-pr rows — same branch

`src/kit/skills/fab-continue.md`'s ship and review-pr dispatch rows (they delegate to `/git-pr` /
`/git-pr-review` and currently resolve "through the two seams" per `docs/specs/stage-models.md`
§ Skill wiring) gain the identical `dispatch:`-presence branch. Exact current wording to be read at
apply; the target semantics are the same two branches as change area 1.

### 3. Dispatch-prompt contract for self-managing stage workers

The dispatched ship/review-pr prompt (identical content on every adapter, per the
delivery-mechanism-varies rule) carries:

- **Obligation 1** — write `.fab-dispatch/{id}/{stage}-result.yaml`. Minimal schemas, mirroring the
  existing apply/review/hydrate shapes (exact fields settled at plan):

  ```yaml
  # ship
  stage: ship
  status: success            # success | failure — worker/infra outcome
  pr_url: "https://github.com/…/pull/N"
  summary: "committed, pushed, draft PR created"
  ```

  ```yaml
  # review-pr
  stage: review-pr
  status: success
  outcome: success           # success | failure | no-reviews | timeout — git-pr-review's Step 6
                             # outcome classes; the Copilot-poll timeout is an OUTCOME, not a failure
  summary: "2 findings fixed, review resolved"
  # on outcome: failure only:
  reason: "no PR found on this branch"
  ```

- **Obligation 2** — the standard subagent context files (already required of every dispatch).
- **Obligation 3** — terminal `fab status refresh <change>`.
- **The delta from other stages**: NO transition prohibition. These workers self-manage
  `fab status` start/finish/fail for their own stage, exactly as the native subagent form does
  today ("Handles fab status integration internally"). `docs/specs/harness-adapters.md`
  § Dispatch-prompt obligations gains a stage-scoped carve-out stating this (ship/review-pr
  workers own their single stage's transitions; the orchestrator still owns sequencing and never
  runs finish for these stages — unchanged, it never did). `_preamble.md`'s block-contract
  carve-out text gains the matching pointer/statement per the owner-or-pointer rule.

- **Step 5's synchronous-poll directive is retained verbatim in the dispatched prompt on all
  arms** — still load-bearing on the native arm; harmlessly moot on the pane arm (a pane worker
  cannot yield). Optionally noted as such where the directive is defined.

### 4. Failure mapping

Error-handling rows in fab-fff.md (and the fab-continue rows) map dispatch states onto the existing
retry story:

| Observed | Meaning | Action |
|----------|---------|--------|
| `done` + ship result `status: failure` | git-pr reported failure | Existing path: STOP with the reported error; the stage remains `active` for user retry |
| `done` + review-pr result `outcome: failure` | git-pr-review reported failure | Existing path: STOP with the result's reported `reason`; the stage remains `active` for user retry |
| `done` + review-pr result `outcome: timeout` | Copilot poll exhausted | Existing pending-message outcome (fab-fff Error Handling), verbatim |
| `failed` / `failed (no-result)` / `orphaned` | Worker/infra states | Canon recovery table (`_preamble.md` § CLI-Adapter Dispatch): one-restart budget, logs surfacing, escalation — no new rules |

### 5. Spec updates

- `docs/specs/stage-models.md` § Skill wiring: the ship/review-pr seam paragraphs (currently
  "through the two seams (empty ⇒ omit)") are rewritten to state the two-branch rule; the
  fast-role rationale table stays.
- `docs/specs/harness-adapters.md`: the adapter-coverage statement ("Every post-intake stage is
  executed by dispatching a worker…") is confirmed/adjusted, and § Dispatch-prompt obligations
  gains the self-managed-transitions carve-out for ship/review-pr (change area 3).

### 6. Sibling sweeps (up front, per code-quality.md)

- `fab-ff.md` twin — no ship/review-pr steps (ends at hydrate), but grep for restatements of the
  Steps 4–5 arm claim.
- Aggregate specs: `docs/specs/skills.md` (fab-fff/fab-continue sections), `docs/specs/glossary.md`.
- `_pipeline.md` § Light Lane wording that contrasts inline vs dispatched Steps 4–5 (the contrast
  survives; the "dispatch exactly as written below" phrasing may need touching).
- Repo-wide grep for "both seams" / "native model/effort seams" claims about ship/review-pr.

### Non-changes (explicit)

- **No Go/runtime change.** `fab dispatch` is stage-generic: `stageRoles` carries `ship` and
  `review-pr` (`src/go/fab/internal/agent/agent.go:304`), and a live `fab agent ship -o yaml`
  returned a well-formed pane mapping. Skill prose + specs + memory only.
- **Light lane untouched** — Steps 4–5 remain inline in the orchestrator's context there.
- **No change to `/git-pr` / `/git-pr-review` core behavior** — they keep self-managing
  transitions; only the dispatch seam around them changes. (Whether their skill files need a small
  dispatched-form note — result file + refresh — is settled at plan.)
- **Worker Continuation stays apply-only** — ship/review-pr workers are never named or continued.

## Affected Memory

- `pipeline/execution-skills`: (modify) fab-fff Steps 4–5 and fab-continue ship/review-pr rows now
  branch on `dispatch:` presence; /git-pr and /git-pr-review gain a dispatched form with
  result-file + refresh obligations and self-managed transitions
- `runtime/dispatch`: (modify) dispatch stage coverage now includes ship/review-pr workers; the
  self-managed-transitions carve-out; reap-at-done-read applies to them
- `runtime/providers-and-profiles`: (modify) consumers section — the fast role's ship referent and
  review-pr's doing referent now execute the resolved dispatch mapping instead of native-only

## Impact

- **Skill sources**: `src/kit/skills/fab-fff.md` (Steps 4–5, Behavior note, Error Handling),
  `src/kit/skills/fab-continue.md` (ship/review-pr rows), `src/kit/skills/_preamble.md`
  (block-contract carve-out pointer/statement; possibly the reap table's stage wording),
  possibly `src/kit/skills/git-pr.md` / `git-pr-review.md` (dispatched-form note),
  `src/kit/skills/_pipeline.md` (light-lane contrast wording).
- **Specs**: `docs/specs/stage-models.md`, `docs/specs/harness-adapters.md`, aggregate sweeps in
  `docs/specs/skills.md` + `docs/specs/glossary.md`.
- **Memory**: three files above at hydrate.
- **Go**: none. **Tests**: none (no CLI surface change) — the constitution's CLI⇒docs+tests rule is
  not triggered.
- **Behavior blast radius**: zero for `agent.workers: claude` (dispatch: absent ⇒ native branch is
  today's path verbatim). Non-native workers setups change from "silent session-model inherit" to
  "honored pane/headless dispatch".

## Open Questions

- None blocking. Plan-level details deliberately deferred to apply (decide-and-record): exact
  result-yaml field set for ship/review-pr, exact current `/fab-continue` row wording, and where
  the carve-out text lives vs points (owner-or-pointer audit).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope covers both fab-fff Steps 4–5 AND /fab-continue's ship/review-pr rows | Discussed — user endorsed; fixing one site leaves the standalone path broken | S:90 R:70 A:90 D:90 |
| 2 | Confident | Dispatched ship/review-pr workers keep self-managed transitions; no block-contract transition prohibition; harness-adapters gains a stage-scoped carve-out | Discussed — matches today's native-subagent form ("Handles fab status integration internally"); alternative (orchestrator-owned transitions) rejected as a larger git-pr refactor | S:80 R:55 A:75 D:75 |
| 3 | Confident | Dispatched form gains obligations 1+3 (result file, terminal refresh); minimal schemas mirror existing stage shapes, exact fields settled at plan | Discussed — the result file is the pane arm's sole completion signal, non-negotiable; field detail is reversible plan work | S:75 R:70 A:80 D:80 |
| 4 | Certain | Light lane untouched — ship/review-pr stay inline there | Discussed — inline stages are by-design no-dispatch (`_pipeline.md` § Light Lane) | S:85 R:85 A:90 D:95 |
| 5 | Confident | Failure mapping: ship result `status: failure` / review-pr result `outcome: failure` ⇒ existing STOP-with-error path (stage stays active); infra states per canon recovery table; review-pr timeout is an outcome, not a failure | Direction discussed; row detail derived from existing fab-fff Error Handling + `_preamble.md` recovery canon | S:70 R:65 A:75 D:70 |
| 6 | Certain | No Go/runtime change — `fab dispatch` is stage-generic | Verified in-session: stageRoles carries ship/review-pr (agent.go:304); live `fab agent ship -o yaml` returned a pane mapping; no stage allowlist in dispatch | S:80 R:90 A:95 D:90 |
| 7 | Certain | Reap timing: ship/review-pr panes reap immediately at done-read; never named/continued | The canon's "every other stage" reap row literally covers them; Worker Continuation is apply-scoped by design | S:70 R:80 A:85 D:85 |
| 8 | Confident | Synchronous-poll directive retained verbatim in the dispatched review-pr prompt on all arms | Still load-bearing on the native arm; harmlessly moot on the pane arm — removing it per-arm would fork the prompt content rule | S:60 R:75 A:60 D:55 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
