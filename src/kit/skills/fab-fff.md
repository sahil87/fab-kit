---
name: fab-fff
description: "Full pipeline — implementation, sub-agent review, hydrate, and ship — gated on the single intake confidence gate, with the one-time light/full lane fork on plan task count (--light/--full override; light lane runs everything but review inline) and autonomous rework with bounded retry. Not for micro changes: a single-spot edit with no memory/spec impact and no behavior-contract change — make it directly and commit, no fab (when unsure, use fab); a follow-up tweak to a change still in flight is not new work — amend that change."
helpers: [_generation, _review, _srad, _pipeline]
---

# /fab-fff [<change-name>] [--force] [--light|--full]

> Read the `_preamble` skill first (deployed to `.agents/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Contents

- Purpose
- Arguments
- Behavior
- Output
- Error Handling

---

## Purpose

Run `_pipeline.md` § Driver Framing, then continue through ship.

---

## Arguments

See `_pipeline.md` § Driver Framing.

---

## Behavior

Execute the **shared pipeline bracket** (`_pipeline.md`, loaded via `helpers:`) with these parameters:

| Parameter | Value |
|-----------|-------|
| `{driver}` | `fab-fff` |
| `{terminal}` | `ship` — after the bracket's Step 3 (hydrate), continue with Steps 3.5–4 below |

The bracket defines pre-flight (intake prerequisite + intake gate), context loading, resumability, Steps 1–3 (apply → review → hydrate) with the inline plan co-gen and one-time light/full lane fork, the auto-rework loop with its per-cycle choreography, the PR Boundary Procedure at the Steps 1–3 boundaries (draft PR opened at apply exit; review-pass/hydrate push + Meta sync), and the exhaustion stop. The two steps below (3.5–4) are fff-only.

Steps 1–4 all branch on `dispatch:` key presence per `_preamble.md` § CLI-Adapter Dispatch — Steps 1–3 via `_pipeline.md` § Stage Dispatch Procedure, Step 4 via its own two-branch text below. The fff-only delta is that Step 4 dispatches the full `/git-pr` behavior with **self-managed stage transitions** — that skill manages its own `fab status` transitions, so its prompt does not carry the block-contract transition prohibition (the carve-out is owned by `_preamble.md` § Dispatch-Prompt Obligations).

**Light lane** (`_pipeline.md` § Light Lane owns the mechanics): Step 4 runs inline in the orchestrator's context. In the full lane Step 4 dispatches exactly as written below.

> **`{name}`** — the change's **folder name** from the preflight YAML (`name` field). Step 4 passes `{name}`, never the 4-char `{id}`: git-pr classifies any argument matching one of the 7 PR type words as a `<type>`, and a 4-char id can collide with `feat`, `docs`, or `test` — a folder name (`{YYMMDD}-{XXXX}-{slug}`) never matches a type token.

### Step 3.5: Link Linear Issue (optional)

*(Skip if `progress.ship` is `done` — the PR title has already shipped.)*

Read `fab-issue.md` and run the `/fab-issue` behavior for `{name}` **inline in this orchestrator's context, in both lanes** — no dispatch, no `fab agent <stage> -o yaml` resolution. It is an optional linking action, not a pipeline stage: it carries no `.status.yaml` progress entry and fires no transition, so a gate skip or deferral never blocks Step 4 (Ship). The gate chain, three-branch outcome, and promptless deferral are owned by `fab-issue.md` — all gates skip gracefully with a one-line report (an unconfigured project sees zero behavior change), and the autonomous carve-out applies in this promptless context. This step runs before ship so `/git-pr` picks up the linked ID in the PR title.

### Step 4: Ship

*(Skip if `progress.ship` is `done`.)* *(**Light lane**: run `/git-pr {name}` inline per the Behavior note above.)*

Run `fab agent ship -o yaml`, surface the resolved YAML (at minimum `provider`/`model`/`model_alias`/`effort` and `dispatch:` presence), then branch on the `dispatch:` key — the same two-branch rule as Steps 1–3:

- **`dispatch:` absent** (native rung) — dispatch `/git-pr` as subagent through the two model/effort seams. The prompt instructs it to invoke `/git-pr {name}` (the **explicit change argument**, using the folder name per the `{name}` note above: git-pr resolves it as a transient override, so the subagent targets this pipeline's change rather than self-resolving the active one, and its branch-matches-change guard verifies the checked-out branch before mutating anything). The subagent commits, rebases onto the recorded base, pushes, syncs Meta, and **finalizes** the draft PR the bracket opened at apply exit (`_pipeline.md` § PR Boundary Procedure) — `/git-pr` on a branch with no PR still creates one, so a standalone or resumed run is unaffected. Handles `fab status` integration internally (start/finish ship stage). Returns PR URL or error.
- **`dispatch:` present** — dispatch the same `/git-pr {name}` prompt through the CLI adapter per `_preamble.md` § CLI-Adapter Dispatch: branch on `dispatch.rung` (`headless` ⇒ `fab dispatch start <change> ship` with the stage prompt on stdin; `pane` ⇒ `open` → readiness gate → `deliver`), then blocking `fab dispatch wait`, state handling, and reap at done-read (ship falls under the "every other stage" immediate-reap row — never named, never continued). The worker writes `ship-result.yaml` (`status`, `pr_url`, `summary`; on failure `reason`) and self-manages the ship stage's `fab status` start/finish exactly as on the native arm — the self-managing-stages carve-out to the block contract is owned by `_preamble.md` § Dispatch-Prompt Obligations.

**If git-pr fails**: STOP with the error from git-pr. The ship stage remains `active` for user retry.

On success: `progress.ship` becomes `done` — the pipeline is complete. (`review-pr` stays `pending`; PR triage is a manual `/git-pr-review` invocation, not a pipeline step.)

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
output section. Ship success ends the pipeline — report `Pipeline complete.`

---

## Error Handling

Shared rows: see `_pipeline.md` § Shared Error Handling (with `{driver}` = `fab-fff`). fff-only rows:

| Condition | Action |
|-----------|--------|
| Ship fails | Stop with git-pr error. User retries /fab-fff <change> or /git-pr {name}. |

CLI-arm rows (Step 4 dispatched via `_preamble.md` § CLI-Adapter Dispatch — the observations map onto the outcomes above; no new recovery rules):

| Observed | Action |
|----------|--------|
| `done` + ship result `status: failure` | The Ship fails row above — STOP with the result's reported `reason`; the stage stays `active` for user retry |
| `failed` / `failed (no-result)` / `orphaned` | Canon recovery table (`_preamble.md` § CLI-Adapter Dispatch) |
