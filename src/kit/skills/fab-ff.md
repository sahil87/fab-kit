---
name: fab-ff
description: "Fast-forward through hydrate — confidence-gated pipeline from intake through hydrate, with sub-agent review, auto-rework loop, and stop on exhaustion."
helpers: [_generation, _review, _srad, _pipeline]
---

# /fab-ff [<change-name>] [--force]

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

The bracket defines everything else: pre-flight (intake prerequisite + intake gate), context loading, resumability, Steps 1–3 (apply → review → hydrate), the auto-rework loop with its per-cycle choreography, and the exhaustion stop.

Stage dispatch uses `_pipeline.md` § Stage Dispatch Procedure and the current canon at `_preamble.md` § CLI-Adapter Dispatch.

---

## Output

Use `_pipeline.md` § Driver Framing with header
`/fab-ff — confidence {score} of 5.0, gate passed.`

---

## Error Handling

See `_pipeline.md` § Shared Error Handling (with `{driver}` = `fab-ff`). `/fab-ff` adds no driver-specific rows.
