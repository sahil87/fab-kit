---
name: fab-fff
description: "Full pipeline — implementation, sub-agent review, hydrate, ship, and PR review — gated on the single intake confidence gate, with autonomous rework with bounded retry."
helpers: [_generation, _review, _srad, _pipeline]
---

# /fab-fff [<change-name>] [--force]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

Run the entire automated Fab pipeline — apply → review → hydrate → ship → review-pr — in a single invocation (everything after intake). Gated on the single intake confidence gate (flat 3.0, all types), checked before the bracket; review failures get a bounded auto-rework loop (`{max_cycles}` cycles — the `Max cycles:` knob in `fab/project/code-review.md` § Rework Budget, default 3) and then stop. No `/fab-clarify` runs inside the bracket — clarification is intake-only. Resumable — re-running picks up from the first incomplete stage. The difference from `/fab-ff` is scope only: `/fab-fff` extends through ship and review-pr; `/fab-ff` stops at hydrate. Both have the identical single intake gate.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of the active one resolved via `.fab-status.yaml`. Resolution per `_preamble.md` (Change-name override).
- **`--force`** *(optional)* — bypass the intake confidence gate. All other behavior (rework loop, etc.) is unchanged. Output header includes "(force mode -- gate bypassed)".

---

## Behavior

Execute the **shared pipeline bracket** (`_pipeline.md`, loaded via `helpers:`) with these parameters:

| Parameter | Value |
|-----------|-------|
| `{driver}` | `fab-fff` — passed to the `fab status` event commands the bracket shows it on (the fail/recovery commands are deliberately driver-less — see `_pipeline.md`'s Behavior note) and used in re-run guidance |
| `{terminal}` | `review-pr` — after the bracket's Step 3 (hydrate), continue with Steps 4–5 below |

The bracket defines pre-flight (intake prerequisite + intake gate), context loading, resumability, Steps 1–3 (apply → review → hydrate), the auto-rework loop with its per-cycle choreography, and the exhaustion stop. The two steps below are fff-only.

> **Per-stage model**: every stage dispatch (the bracket's Steps 1–3 and the fff-only Steps 4–5 below) resolves `fab resolve-agent <stage>` first, surfaces the resolved `model=/effort=` (so a skipped or mis-resolved tier is visible, not silent), then dispatches through the two seams — model via the Agent tool's `model` param (empty ⇒ omit/inherit) and effort via an imperative instruction in the dispatch prompt (``Operate at `<effort>` reasoning effort for this task.``; empty effort ⇒ omit, since the Agent tool has no effort param) — see `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution.

> **Dispatch exception (ship and review-pr)**: unlike the bracket's `/fab-continue`-behavior subagents, `/git-pr` and `/git-pr-review` manage their own stage transitions internally — their subagent prompts do NOT carry the "do not run `fab status`" instruction.

> **`{name}`** — the change's **folder name** from the preflight YAML (`name` field). Steps 4–5 pass `{name}`, never the 4-char `{id}`: git-pr classifies any argument matching one of the 7 PR type words as a `<type>`, and a 4-char id can collide with `feat`, `docs`, or `test` — a folder name (`{YYMMDD}-{XXXX}-{slug}`) never matches a type token.

### Step 4: Ship

*(Skip if `progress.ship` is `done`.)*

Resolve the ship model: run `fab resolve-agent ship`, surface the resolved `model=/effort=`, and apply both halves to the dispatch below — model via the Agent `model` param (empty ⇒ omit/inherit), effort via the imperative prompt instruction ``Operate at `<effort>` reasoning effort for this task.`` (empty effort ⇒ omit) — per `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution. Dispatch `/git-pr` as subagent — the prompt instructs it to invoke `/git-pr {name}` (the **explicit change argument**, using the folder name per the `{name}` note above: git-pr resolves it as a transient override, so the subagent targets this pipeline's change rather than self-resolving the active one, and its branch-matches-change guard verifies the checked-out branch before mutating anything). The subagent commits, pushes, and creates a GitHub PR. Handles `fab status` integration internally (start/finish ship stage). Returns PR URL or error.

**If git-pr fails**: STOP with the error from git-pr. The ship stage remains `active` for user retry.

On success: `progress.ship` becomes `done`, `progress.review-pr` auto-activates.

### Step 5: Review-PR

*(Skip if `progress.review-pr` is `done`.)*

Resolve the review-pr model: run `fab resolve-agent review-pr`, surface the resolved `model=/effort=`, and apply both halves to the dispatch below — model via the Agent `model` param (empty ⇒ omit/inherit), effort via the imperative prompt instruction ``Operate at `<effort>` reasoning effort for this task.`` (empty effort ⇒ omit) — per `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution. Dispatch `/git-pr-review` as subagent — the prompt instructs it to invoke `/git-pr-review {name}` (the **explicit change argument**, same transient-override + branch-guard contract as Step 4). The subagent detects existing reviews, triages comments, applies fixes, and pushes. If no reviews exist, it requests a Copilot review and polls up to 10 minutes — see the timeout outcome below. Handles `fab status` integration internally (start/finish/fail review-pr stage). Returns completion status.

**If review-pr fails** (no PR found, processing error): STOP with the error.

**If no actionable reviews** (no automated reviewer available, or reviews with no inline comments to process): the stage completes as `done` — this is a successful no-op.

**If timeout** (Copilot review requested but not available within 10 minutes — git-pr-review's Step 6 timeout outcome): the subagent deliberately leaves `review-pr` `active` (no finish, no fail). Report `Review-PR pending (Copilot review requested, timed out waiting) — re-run /git-pr-review {name} when ready` **instead of** `Pipeline complete.` and stop.

On success: `progress.review-pr` becomes `done`.

---

## Output

```
/fab-fff — intake confidence {score} of 5.0, gate passed.

--- Implementation ---
{apply output (plan generation — incl. ## Requirements — + task execution)}

## Assumptions (cumulative)
{table with Artifact column — apply-recorded assumptions from plan.md}

--- Review ---
{review output}

--- Hydrate ---
{hydrate output}

--- Ship ---
{git-pr output}

--- Review-PR ---
{git-pr-review output}

Pipeline complete.

Next: {per state table}
```

Resuming shows `(resuming)...` header and `Skipping {stage} — already done.` for completed stages. Bail/failure stops at the relevant stage with `Next:` derived from the state reached per state table in `_preamble.md`. On the Step 5 timeout outcome, the closing line is `Review-PR pending (Copilot review requested, timed out waiting) — re-run /git-pr-review {name} when ready` instead of `Pipeline complete.`

---

## Error Handling

Shared rows: see `_pipeline.md` § Shared Error Handling (with `{driver}` = `fab-fff`). fff-only rows:

| Condition | Action |
|-----------|--------|
| Ship fails | Stop with git-pr error. User retries /fab-fff <change> or /git-pr {name}. |
| Review-PR fails | Stop with git-pr-review error. User retries /fab-fff <change> or /git-pr-review {name}. |
| Review-PR timeout (Copilot review requested, not yet available) | Stage deliberately left `active`. Report `Review-PR pending (Copilot review requested, timed out waiting) — re-run /git-pr-review {name} when ready` and stop — no finish, no fail. |
