# Intake: Split Stage Display from Routing

**Change**: 260218-95xn-split-stage-display-from-routing
**Created**: 2026-02-18
**Status**: Draft

## Origin

> User observed that `fab-status` shows "Stage: spec (2/6)" when `intake: done` and `spec: pending`, but shows "Stage: intake (1/6)" when `intake: active` — yet `/fab-continue` produces the same artifact (spec.md) in both cases. The display conflates "where you are" with "what's next," creating a misleading UX. Discussion converged on splitting the two concepts: a **display stage** (where you ARE) and a **next action** (what `/fab-continue` will produce).

## Why

The single `get_current_stage` function in `stageman.sh` serves two incompatible purposes:

1. **Routing** — determining what `/fab-continue` should produce next (correctly returns the first pending stage after the last done)
2. **Display** — telling the user where they are in the pipeline (misleading when the last done stage differs from the next pending stage)

When `intake: done` and `spec: pending`, the user sees "Stage: spec (2/6)" even though they haven't started spec yet. This is confusing because:
- It implies spec work is in progress when it hasn't begun
- It's inconsistent with the `intake: active` case, which correctly shows "Stage: intake (1/6)"
- The `active` state has real semantic meaning (in-progress, possibly interrupted, what `fab-clarify` targets), and the display should reflect it

Without this fix, users will continue to see misleading stage information, especially after `/fab-new` (which leaves intake as `done`, not `active`) and after any interrupted pipeline run.

## What Changes

### 1. New stageman function: `get_display_stage`

Add a new function to `stageman.sh` that returns the **last active or last done stage** — representing "where you are" rather than "what's next." Logic:

1. If any stage is `active`, return it (you're in the middle of that stage)
2. Otherwise, return the last `done` stage (you completed it, nothing active yet)
3. If nothing is done or active, return the first stage (fresh change)

### 2. New stageman CLI command: `display-stage`

Expose `get_display_stage` as `stageman.sh display-stage <file>`, parallel to the existing `current-stage` command.

### 3. Update preflight.sh output

Add a `display_stage` field to the YAML output from `preflight.sh`, alongside the existing `stage` field. Skills can then use `display_stage` for presentation and `stage` for routing.

### 4. Update changeman.sh switch output

The `switch` subcommand currently shows `Stage: $stage ($snum/6)`. Update to show the display stage with its state qualifier, plus the next action:

```
Stage:  intake (1/6) — done
Next:   spec (via /fab-continue)
```

### 5. Update fab-status skill display

The `fab-status.md` skill renders the stage progress block. Update to:
- Show the display stage (with state qualifier) as the primary "Stage:" line
- Show the next action as a separate "Next:" line
- The progress table (with `✓ ● ○` symbols) remains unchanged — it already conveys the full picture

### 6. Update fab-switch skill display

The `fab-switch.md` skill shows stage info after switching. Update to use the same two-line format as fab-status.

## Affected Memory

- `fab-workflow/preflight`: (modify) Document new `display_stage` field in preflight output
- `fab-workflow/change-lifecycle`: (modify) Document display-stage vs current-stage distinction

## Impact

- **stageman.sh** — new function + CLI command
- **preflight.sh** — new output field
- **changeman.sh** — updated switch display format
- **fab-status.md** — updated display format
- **fab-switch.md** — updated display format
- No changes to stage transitions, routing logic, or `/fab-continue` behavior

## Open Questions

- None — design was discussed and agreed in conversation before intake.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Keep `get_current_stage` unchanged for routing | Existing routing logic is correct; only the display is misleading | S:95 R:90 A:95 D:95 |
| 2 | Certain | New function returns last active-or-done stage | Directly follows from the discussion; "where you are" = last stage you touched | S:90 R:85 A:90 D:90 |
| 3 | Certain | Add `display_stage` to preflight output | Skills need both values; adding a field is non-breaking | S:85 R:90 A:90 D:90 |
| 4 | Confident | Use "— done" / "— active" as state qualifier | Matches the two meaningful states; "pending" never appears as display stage | S:80 R:90 A:70 D:75 |
| 5 | Confident | Show "Next: {stage} (via /fab-continue)" format | Clear, actionable; mirrors existing "Next:" convention from _context.md state table | S:75 R:85 A:75 D:70 |

5 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).
