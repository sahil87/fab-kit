---
name: fab-fff
description: "Full pipeline — implementation, sub-agent review, hydrate, ship, and PR review — gated on the single intake confidence gate, with the one-time light/full lane fork on plan task count (--light/--full override; light lane runs everything but review inline) and autonomous rework with bounded retry."
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

Steps 1–3 use `_pipeline.md` § Stage Dispatch Procedure and the current canon at `_preamble.md` § CLI-Adapter Dispatch. The fff-only delta is that Steps 4–5 dispatch full `/git-pr` and `/git-pr-review` behaviors through the native model/effort seams; those skills manage their own stage transitions, so their prompts do not carry the block-contract transition prohibition.

**Light lane** (`_pipeline.md` § Light Lane owns the mechanics): Steps 4–5 run inline in the orchestrator's context, and the Step 5 synchronous-poll directive is moot there. In the full lane Steps 4–5 dispatch exactly as written below.

> **`{name}`** — the change's **folder name** from the preflight YAML (`name` field). Steps 4–5 pass `{name}`, never the 4-char `{id}`: git-pr classifies any argument matching one of the 7 PR type words as a `<type>`, and a 4-char id can collide with `feat`, `docs`, or `test` — a folder name (`{YYMMDD}-{XXXX}-{slug}`) never matches a type token.

### Step 3.5: Link Linear Issue (optional)

*(Skip if `progress.ship` is `done` — the PR title has already shipped.)*

Read `fab-issue.md` and run the `/fab-issue` behavior for `{name}` **inline in this orchestrator's context, in both lanes** — no dispatch, no `fab resolve-agent`. It is an optional linking action, not a pipeline stage: it carries no `.status.yaml` progress entry and fires no transition, so a gate skip or deferral never blocks Step 4 (Ship). The gate chain, three-branch outcome, and promptless deferral are owned by `fab-issue.md` — all gates skip gracefully with a one-line report (an unconfigured project sees zero behavior change), and the autonomous carve-out applies in this promptless context. This step runs before ship so `/git-pr` picks up the linked ID in the PR title.

### Step 4: Ship

*(Skip if `progress.ship` is `done`.)* *(**Light lane**: run `/git-pr {name}` inline per the Behavior note above.)*

Run `fab resolve-agent ship --alias`, surface `model=/effort=`, and dispatch via both seams; then dispatch `/git-pr` as subagent — the prompt instructs it to invoke `/git-pr {name}` (the **explicit change argument**, using the folder name per the `{name}` note above: git-pr resolves it as a transient override, so the subagent targets this pipeline's change rather than self-resolving the active one, and its branch-matches-change guard verifies the checked-out branch before mutating anything). The subagent commits, pushes, and creates a GitHub PR. Handles `fab status` integration internally (start/finish ship stage). Returns PR URL or error.

**If git-pr fails**: STOP with the error from git-pr. The ship stage remains `active` for user retry.

On success: `progress.ship` becomes `done`, `progress.review-pr` auto-activates.

### Step 5: Review-PR

*(Skip if `progress.review-pr` is `done`.)* *(**Light lane**: run `/git-pr-review {name}` inline per the Behavior note above; the synchronous-poll directive below is moot inline.)*

Run `fab resolve-agent review-pr --alias`, surface `model=/effort=`, and dispatch via both seams; then dispatch `/git-pr-review` as subagent — the prompt instructs it to invoke `/git-pr-review {name}` (the **explicit change argument**, same transient-override + branch-guard contract as Step 4). The subagent detects existing reviews, triages comments, applies fixes, and pushes. If no reviews exist, it requests a Copilot review and polls up to 10 minutes — see the timeout outcome below. Handles `fab status` integration internally (start/finish/fail review-pr stage). Returns completion status.

> **Synchronous-poll directive (bake into the dispatch prompt).** The review-pr dispatch prompt MUST instruct the `/git-pr-review` subagent to **complete the Copilot poll synchronously and not yield mid-poll**: if it requests a Copilot review and enters the 30s × 20 (10-minute) poll, it MUST stay in that poll loop within the single invocation — never yielding, returning, or handing back control while the poll is pending — until a review appears or all 20 attempts are exhausted. The poll **stays inside `/git-pr-review`** (the subagent owns request + poll + triage synchronously); the wait is NOT relocated to this orchestrator. Carry the rationale in the prompt — see `git-pr-review.md` Step 2 Phase 2's synchronous-poll discipline note (it mirrors that note into the dispatch seam).

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
