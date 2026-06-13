---
name: fab-ff
description: "Fast-forward through hydrate — confidence-gated pipeline from intake through hydrate, with sub-agent review, auto-rework loop, and stop on exhaustion."
helpers: [_generation, _review, _srad, _pipeline]
---

# /fab-ff [<change-name>] [--force]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

Fast-forward through hydrate: apply → review → hydrate (everything after intake, stopping before the PR stages). Gated on the single intake confidence gate (flat 3.0, all types), checked before the bracket; review failures get a bounded auto-rework loop (`{max_cycles}` cycles — the `Max cycles:` knob in `fab/project/code-review.md` § Rework Budget, default 3) and then stop. On any stop, the user can intervene then re-run. Resumable — re-running picks up from the first incomplete stage. No `/fab-clarify` runs inside the bracket — clarification is intake-only.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of the active one resolved via `.fab-status.yaml`. Resolution per `_preamble.md` (Change-name override).
- **`--force`** *(optional)* — bypass the intake confidence gate. All other behavior (rework loop, etc.) is unchanged. Output header includes "(force mode -- gate bypassed)".

---

## Behavior

Execute the **shared pipeline bracket** (`_pipeline.md`, loaded via `helpers:`) with these parameters:

| Parameter | Value |
|-----------|-------|
| `{driver}` | `fab-ff` — passed to the `fab status` event commands the bracket shows it on (the fail/recovery commands are deliberately driver-less — see `_pipeline.md`'s Behavior note) and used in re-run guidance |
| `{terminal}` | `hydrate` — the pipeline ends after the bracket's Step 3; there are no ship/review-pr steps |

The bracket defines everything else: pre-flight (intake prerequisite + intake gate), context loading, resumability, Steps 1–3 (apply → review → hydrate), the auto-rework loop with its per-cycle choreography, and the exhaustion stop.

> **Per-stage model**: each stage dispatch in the bracket resolves the stage's profile first, surfaces the resolved `model=/effort=` (so a skipped or mis-resolved tier is visible, not silent), then dispatches through the two seams — model via the Agent tool's `model` param, resolved with `fab resolve-agent <stage> --alias` so the alias is Agent-tool-valid (empty ⇒ omit/inherit), and effort via an imperative instruction in the dispatch prompt (``Operate at `<effort>` reasoning effort for this task.``; empty effort ⇒ omit, since the Agent tool has no effort param) — see `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution.

---

## Output

```
/fab-ff — confidence {score} of 5.0, gate passed.

--- Implementation ---
{apply output (plan generation + task execution)}

--- Review ---
{review output}

--- Hydrate ---
{hydrate output}

Pipeline complete.

Next: {per state table}
```

Resuming shows `(resuming)...` header and `Skipping {stage} — already done.` for completed stages. Bail/failure stops at the relevant stage with `Next:` derived from the state reached per state table in `_preamble.md`.

---

## Error Handling

See `_pipeline.md` § Shared Error Handling (with `{driver}` = `fab-ff`). `/fab-ff` adds no driver-specific rows.
