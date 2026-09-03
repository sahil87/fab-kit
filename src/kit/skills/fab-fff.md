---
name: fab-fff
description: "Full pipeline — implementation, sub-agent review, hydrate, ship, and PR review — gated on the single intake confidence gate, with the one-time light/full lane fork on plan task count (--light/--full override; light lane runs everything but review inline) and autonomous rework with bounded retry. Not for micro changes: a single-spot edit with no memory/spec impact and no behavior-contract change — make it directly and commit, no fab (when unsure, use fab); a follow-up tweak to a change still in flight is not new work — amend that change."
helpers: [_generation, _review, _srad, _pipeline]
---

# /fab-fff [<change-name>] [--force] [--light|--full]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Contents

- Purpose
- Arguments
- Behavior
- Output
- Error Handling

---

## Purpose

Run `_pipeline.md` § Driver Framing, then continue through ship and review-pr.

---

## Arguments

See `_pipeline.md` § Driver Framing.

---

## Behavior

Execute the **shared pipeline bracket** (`_pipeline.md`, loaded via `helpers:`) with these parameters:

| Parameter | Value |
|-----------|-------|
| `{driver}` | `fab-fff` |
| `{terminal}` | `review-pr` — after the bracket's Step 3 (hydrate), continue with Steps 3.5–5 below |

The bracket defines pre-flight (intake prerequisite + intake gate), context loading, resumability, Steps 1–3 (apply → review → hydrate) with the inline plan co-gen and one-time light/full lane fork, the auto-rework loop with its per-cycle choreography, and the exhaustion stop. The three steps below (3.5–5) are fff-only.

Steps 1–5 all branch on `dispatch:` key presence per `_preamble.md` § CLI-Adapter Dispatch — Steps 1–3 via `_pipeline.md` § Stage Dispatch Procedure, Steps 4–5 via their own two-branch text below. The fff-only delta is that Steps 4–5 dispatch full `/git-pr` and `/git-pr-review` behaviors with **self-managed stage transitions** — those skills manage their own `fab status` transitions, so their prompts do not carry the block-contract transition prohibition (the carve-out is owned by `_preamble.md` § Dispatch-Prompt Obligations).

**Light lane** (`_pipeline.md` § Light Lane owns the mechanics): Steps 4–5 run inline in the orchestrator's context, and the Step 5 synchronous-poll directive is moot there. In the full lane Steps 4–5 dispatch exactly as written below.

> **`{name}`** — the change's **folder name** from the preflight YAML (`name` field). Steps 4–5 pass `{name}`, never the 4-char `{id}`: git-pr classifies any argument matching one of the 7 PR type words as a `<type>`, and a 4-char id can collide with `feat`, `docs`, or `test` — a folder name (`{YYMMDD}-{XXXX}-{slug}`) never matches a type token.

### Step 3.5: Link Linear Issue (optional)

*(Skip if `progress.ship` is `done` — the PR title has already shipped.)*

Read `fab-issue.md` and run the `/fab-issue` behavior for `{name}` **inline in this orchestrator's context, in both lanes** — no dispatch, no `fab agent <stage> -o yaml` resolution. It is an optional linking action, not a pipeline stage: it carries no `.status.yaml` progress entry and fires no transition, so a gate skip or deferral never blocks Step 4 (Ship). The gate chain, three-branch outcome, and promptless deferral are owned by `fab-issue.md` — all gates skip gracefully with a one-line report (an unconfigured project sees zero behavior change), and the autonomous carve-out applies in this promptless context. This step runs before ship so `/git-pr` picks up the linked ID in the PR title.

### Step 4: Ship

*(Skip if `progress.ship` is `done`.)* *(**Light lane**: run `/git-pr {name}` inline per the Behavior note above.)*

Run `fab agent ship -o yaml`, surface the resolved YAML (at minimum `provider`/`model`/`model_alias`/`effort` and `dispatch:` presence), then branch on the `dispatch:` key — the same two-branch rule as Steps 1–3:

- **`dispatch:` absent** (native rung) — dispatch `/git-pr` as subagent through the two model/effort seams. The prompt instructs it to invoke `/git-pr {name}` (the **explicit change argument**, using the folder name per the `{name}` note above: git-pr resolves it as a transient override, so the subagent targets this pipeline's change rather than self-resolving the active one, and its branch-matches-change guard verifies the checked-out branch before mutating anything). The subagent commits, pushes, and creates a GitHub PR. Handles `fab status` integration internally (start/finish ship stage). Returns PR URL or error.
- **`dispatch:` present** — dispatch the same `/git-pr {name}` prompt through the CLI adapter per `_preamble.md` § CLI-Adapter Dispatch: branch on `dispatch.rung` (`headless` ⇒ `fab dispatch start <change> ship` with the stage prompt on stdin; `pane` ⇒ `open` → readiness gate → `deliver`), then blocking `fab dispatch wait`, state handling, and reap at done-read (ship falls under the "every other stage" immediate-reap row — never named, never continued). The worker writes `ship-result.yaml` (`status`, `pr_url`, `summary`; on failure `reason`) and self-manages the ship stage's `fab status` start/finish exactly as on the native arm — the self-managing-stages carve-out to the block contract is owned by `_preamble.md` § Dispatch-Prompt Obligations.

**If git-pr fails**: STOP with the error from git-pr. The ship stage remains `active` for user retry.

On success: `progress.ship` becomes `done`, `progress.review-pr` auto-activates.

### Step 5: Review-PR

*(Skip if `progress.review-pr` is `done`.)* *(**Light lane**: run `/git-pr-review {name}` inline per the Behavior note above; the synchronous-poll directive below is moot inline.)*

Run `fab agent review-pr -o yaml`, surface the resolved YAML (at minimum `provider`/`model`/`model_alias`/`effort` and `dispatch:` presence), then branch on the `dispatch:` key — the same two-branch rule as Step 4:

- **`dispatch:` absent** (native rung) — dispatch `/git-pr-review` as subagent through the two model/effort seams. The prompt instructs it to invoke `/git-pr-review {name}` (the **explicit change argument**, same transient-override + branch-guard contract as Step 4). The subagent detects existing reviews, triages comments, applies fixes, and pushes. If no reviews exist, it requests a Copilot review and polls up to 10 minutes — see the timeout outcome below. Handles `fab status` integration internally (start/finish/fail review-pr stage). Returns completion status.
- **`dispatch:` present** — dispatch the same `/git-pr-review {name}` prompt through the CLI adapter per `_preamble.md` § CLI-Adapter Dispatch (rung branch, blocking `fab dispatch wait`, state handling, done-read reap under the "every other stage" row — exactly as Step 4). The worker writes `review-pr-result.yaml` (`status`, `outcome` — `success | failure | no-reviews | timeout`, git-pr-review's own Step 6 classes — and `summary`) and self-manages the review-pr stage's start/finish/fail exactly as on the native arm (the same `_preamble.md` § Dispatch-Prompt Obligations carve-out Step 4 points at). The synchronous-poll directive below is baked into the dispatched prompt on **every arm**; on the pane arm it is moot by construction — a pane worker cannot yield.

> **Synchronous-poll directive (bake into the dispatch prompt on every arm).** The review-pr dispatch prompt MUST instruct the `/git-pr-review` subagent to **complete the Copilot poll synchronously and not yield mid-poll**: if it requests a Copilot review and enters the 30s × 20 (10-minute) poll, it MUST stay in that poll loop within the single invocation — never yielding, returning, or handing back control while the poll is pending — until a review appears or all 20 attempts are exhausted. The poll **stays inside `/git-pr-review`** (the subagent owns request + poll + triage synchronously); the wait is NOT relocated to this orchestrator. Carry the rationale in the prompt — see `git-pr-review.md` Step 2 Phase 2's synchronous-poll discipline note (it mirrors that note into the dispatch seam). Load-bearing on the native arm; moot by construction on the pane arm.

**If review-pr fails** (no PR found, processing error): STOP with the error.

**If no actionable reviews** (no automated reviewer available, or reviews with no inline comments to process): the stage completes as `done` — this is a successful no-op.

**If timeout** (Copilot review requested but not available within 10 minutes — git-pr-review's Step 6 timeout outcome): the subagent deliberately leaves `review-pr` `active` (no finish, no fail); report the pending message **instead of** `Pipeline complete.` and stop — see the Review-PR timeout row in Error Handling for the exact string.

On success: `progress.review-pr` becomes `done`.

---

## Output

Use `_pipeline.md` § Driver Framing with header
`/fab-fff — intake confidence {score} of 5.0, gate passed.` After Implementation,
render:

```
## Assumptions (cumulative)
{table with Artifact column — apply-recorded assumptions from plan.md}
```

After Hydrate, append the Step 3.5 Linear-link report (one line), then the Ship
and Review-PR output sections. On the Step 5 timeout
outcome, replace `Pipeline complete.` with the exact pending message in Error
Handling.

---

## Error Handling

Shared rows: see `_pipeline.md` § Shared Error Handling (with `{driver}` = `fab-fff`). fff-only rows:

| Condition | Action |
|-----------|--------|
| Ship fails | Stop with git-pr error. User retries /fab-fff <change> or /git-pr {name}. |
| Review-PR fails | Stop with git-pr-review error. User retries /fab-fff <change> or /git-pr-review {name}. |
| Review-PR timeout (Copilot review requested, not yet available) | Stage deliberately left `active`. Report `Review-PR pending (Copilot review requested, timed out waiting) — re-run /git-pr-review {name} when ready` and stop — no finish, no fail. |

CLI-arm rows (Step 4/5 dispatched via `_preamble.md` § CLI-Adapter Dispatch — the observations map onto the outcomes above; no new recovery rules):

| Observed | Action |
|----------|--------|
| `done` + result `status: failure` | The matching fail row above — STOP with the result's reported `reason`; the stage stays `active` for user retry |
| `done` + review-pr result `outcome: timeout` | The Review-PR timeout row above, verbatim — the timeout is an outcome, not a failure |
| `done` + review-pr result `outcome: no-reviews` | Successful no-op — the worker finished the stage `done` itself |
| `failed` / `failed (no-result)` / `orphaned` | Canon recovery table (`_preamble.md` § CLI-Adapter Dispatch) |
