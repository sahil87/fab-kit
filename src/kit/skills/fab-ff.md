---
name: fab-ff
description: "Fast-forward through hydrate — confidence-gated pipeline from intake through hydrate, with the one-time light/full lane fork on plan task count (--light/--full override; light lane runs task execution and hydrate inline), sub-agent review in both lanes, auto-rework loop, and stop on exhaustion."
helpers: [_generation, _review, _srad, _pipeline]
---

# /fab-ff [<change-name>] [--force] [--light|--full]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

Run `_pipeline.md` § Driver Framing through hydrate, stopping before PR stages.

---

## Arguments

See `_pipeline.md` § Driver Framing.

---

## Behavior

Execute the **shared pipeline bracket** (`_pipeline.md`, loaded via `helpers:`) with these parameters:

| Parameter | Value |
|-----------|-------|
| `{driver}` | `fab-ff` |
| `{terminal}` | `hydrate` — the pipeline ends after the bracket's Step 3; there are no ship/review-pr steps |

The bracket defines everything else: pre-flight (intake prerequisite + intake gate), context loading, resumability, Steps 1–3 (apply → review → hydrate) with the inline plan co-gen and one-time light/full lane fork, the auto-rework loop with its per-cycle choreography, and the exhaustion stop.

The lane mechanics (fork, inline loci, rework locus, review-always-dispatched) are owned by `_pipeline.md` § Light Lane — in the **light lane** this driver's run covers task execution and hydrate inline and ends there (its terminal).

Stage dispatch (full lane, and review in both lanes) uses `_pipeline.md` § Stage Dispatch Procedure and the current canon at `_preamble.md` § CLI-Adapter Dispatch.

---

## Output

Use `_pipeline.md` § Driver Framing with header
`/fab-ff — confidence {score} of 5.0, gate passed.`

---

## Error Handling

See `_pipeline.md` § Shared Error Handling (with `{driver}` = `fab-ff`). `/fab-ff` adds no driver-specific rows.
