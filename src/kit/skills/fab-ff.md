---
name: fab-ff
description: "Fast-forward through hydrate — confidence-gated pipeline from intake through hydrate, with the one-time light/full lane fork on plan task count (--light/--full override; light lane runs task execution and hydrate inline), sub-agent review in both lanes, auto-rework loop, stop on exhaustion, and the PR Boundary Procedure (draft PR opened at apply exit; review-pass/hydrate boundaries push + sync Meta) so the run ends with an OPEN draft PR a later /git-pr finalizes. Not for micro changes: a single-spot edit with no memory/spec impact and no behavior-contract change — make it directly and commit, no fab (when unsure, use fab); a follow-up tweak to a change still in flight is not new work — amend that change."
helpers: [_generation, _review, _srad, _pipeline]
---

# /fab-ff [<change-name>] [--force] [--light|--full]

> Read the `_preamble` skill first (deployed to `.agents/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

Run `_pipeline.md` § Driver Framing through hydrate. The bracket's PR Boundary Procedure still runs — the draft PR opens at apply exit and the review-pass/hydrate boundaries push and sync its Meta block — so the run ends with an OPEN draft PR (with a current Meta block) that a later `/git-pr` (standalone or via `/fab-continue` / `/fab-fff`) finalizes.

---

## Arguments

See `_pipeline.md` § Driver Framing.

---

## Behavior

Execute the **shared pipeline bracket** (`_pipeline.md`, loaded via `helpers:`) with these parameters:

| Parameter | Value |
|-----------|-------|
| `{driver}` | `fab-ff` |
| `{terminal}` | `hydrate` — the pipeline ends after the bracket's Step 3; there is no ship step, but the PR Boundary Procedure ran at every boundary, leaving an OPEN draft PR for a later `/git-pr` to finalize |

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
