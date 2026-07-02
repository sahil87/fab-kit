# Intake: true_impact Recorded as All-Zeros — Recompute at Ship

**Change**: 260702-h65d-true-impact-recompute-at-ship
**Created**: 2026-07-02

## Origin

Bug report relayed by the user from another agent working in run-kit (fab 2.9.2), drafted via `/fab-draft` with the user's chosen fix framing ("recompute at ship"). The reporting agent's full handoff doc (`scratchpad/true-impact-zeros-handoff.md`) lives on another machine and is NOT available here — its substance is reproduced below and was independently re-verified against fab-kit source at main (v2.9.2) before this intake was written.

> **Bug: true_impact recorded as all-zeros for every change**
>
> Symptom: Every completed change's `.status.yaml` `true_impact` block reads 0/0/0 (raw/excluding/tests) regardless of change size. Confirmed across 4 run-kit changes; f4e5's real diff is 334/30/304 and 70a0's is 1842/426/1416 — both recorded as zeros.
>
> Root cause (confirmed by reproduction): `internal/status/true_impact.go` → `WriteTrueImpact()` runs `git diff --shortstat <merge-base>...HEAD` at apply-finish and hydrate-finish only. But the fab pipeline commits nothing until `/git-pr` (ship) — during apply/hydrate the code lives in the working tree and HEAD == merge-base. So: `base...HEAD` is an empty commit range → 0/0/0, and three-dot `git diff` ignores the working tree by design, so the uncommitted apply edits are invisible. The math is correct (it shares `impact.Compute()` with the working `fab impact` CLI) — it's purely a timing bug: the diff runs before the branch tip it needs to measure exists.
>
> Fix direction — recompute at ship: Add "ship" to `WriteTrueImpact`'s accepted stages so the block is recomputed after `/git-pr` commits+pushes. `computed_at_stage` already distinguishes writes, so the ship-time value cleanly supersedes any hydrate estimate.

**Verification performed at intake (fab-kit main = v2.9.2, plus run-kit ground truth):**

- **Root cause CONFIRMED.** `src/go/fab/internal/status/true_impact.go:22-24` gates on `stage != "apply" && stage != "hydrate"` → return nil. `status.Finish` (`internal/status/status.go:196`) calls `WriteTrueImpact` on every stage finish, so the gate is the only thing blocking a ship-time write. `impact.runShortstat` (`internal/impact/impact.go:121`) uses the three-dot form `base + "..." + head`, which ignores the working tree. Empty commit range → `--shortstat` prints nothing → parses as 0/0.
- **Symptom CONFIRMED in run-kit.** `fab/changes/260701-f4e5-*/`, `260701-vshd-*/`, and `260616-37ub-*/` `.status.yaml` files all carry all-zero `true_impact` blocks with `computed_at_stage: hydrate`.
- **One report claim DISPROVEN — the PR Impact section was NOT omitted.** The report asserted "`fab pr-meta` reads the persisted block (doesn't recompute)" and that the PR Impact table was silently dropped on every fab PR. At v2.9.2, `fab pr-meta` recomputes live via `impact.ComputeForRepo(fabRoot, base, "HEAD")` (`internal/prmeta/prmeta.go:490`) and `/git-pr` renders it at Step 3c, AFTER the Step 3a commit — so HEAD is the branch tip and the table is correct. Verified against run-kit PR #293 (f4e5): its body contains a correct Impact table (raw +319/−27, true +43/−25). **`fab pr-meta` needs no change** (the report reached the same conclusion via wrong reasoning).

## Why

1. **The pain point**: the persisted `.status.yaml` `true_impact` block — the durable record of a change's line-count impact — is all-zeros for every change that follows the standard pipeline, in every fab project. The standard pipeline commits nothing until `/git-pr` (ship), so both computation points (apply-finish, hydrate-finish) run when `HEAD == merge-base` and the three-dot diff is definitionally empty. This is a 100%-reproducible timing bug, not a math bug.
2. **The consequence if unfixed**: every consumer of the persisted block shows garbage — `fab change list --show-stats` (the `true_impact` net column via `impactColumn`, `internal/change/change.go:277`), `/fab-status`'s Impact line and its >100/>50 net warning thresholds (which can never trip at 0), and the refactor-growth soft warning. Historical impact stats across all fab projects are worthless zeros.
3. **Why this approach**: recompute at ship-finish — the earliest pipeline point where the branch tip exists (after `/git-pr` Step 3a commit / 3b push). No new call site is needed: `status.Finish` already invokes `WriteTrueImpact` with the stage name on every finish, and `/git-pr` Step 4b already runs `fab status finish {name} ship git-pr`. Widening the stage gate is the minimal, root-cause-level fix. Alternatives rejected: (a) making apply/hydrate computations include the working tree (two-dot or `git diff <base>` without HEAD) — changes the measurement's meaning, diverges from the PR diff, and still goes stale the moment more edits land; (b) reading `fab pr-meta`'s live computation back into the block — inverts the data flow and couples the block to PR creation succeeding.

## What Changes

### 1. Widen `WriteTrueImpact`'s stage gate to include `ship`

`src/go/fab/internal/status/true_impact.go`:

```go
// current (line 23)
if stage != "apply" && stage != "hydrate" {
    return nil
}
// becomes
if stage != "apply" && stage != "hydrate" && stage != "ship" {
    return nil
}
```

- No new call site: `status.Finish` (`status.go:196`) already passes the stage on every finish. `/git-pr` Step 4b (`fab status finish {name} ship git-pr`) runs after Step 3a commit + 3b push + Step 4 `gh pr create`, so at ship-finish `HEAD` is the branch tip and `merge-base(origin/main, HEAD)...HEAD` measures the true PR diff.
- `computed_at_stage: ship` supersedes the earlier apply/hydrate zeros in place (the block is a single overwrite-on-write value, not a history).
- `/git-pr` Step 4c then commits the `.status.yaml` update ("Update ship status and record PR URL") and pushes — so the corrected block lands on the PR branch with no extra choreography.
- Keep the apply-finish and hydrate-finish writes as-is: they are harmless in the standard flow (zeros until ship supersedes) and produce real values in non-standard flows where commits exist before ship (e.g., adopted off-pipeline changes, manual mid-apply commits).
- Update the function's doc comment (currently "Stage MUST be one of: apply, hydrate … per spec assumption #16") and the stale comment at `status.go:195` ("Compute and write true_impact for apply/hydrate finish").

### 2. Tests

Extend `src/go/fab/internal/status/true_impact_test.go`: ship-finish writes a non-zero block when commits exist on the branch (the repro scenario: apply/hydrate finish with clean-tree HEAD == merge-base → zeros; then commit; then ship finish → real counts, `computed_at_stage: ship` superseding the earlier write). Constitution: Go changes ship tests.

### 3. Documentation sweep (stale "apply/hydrate only" claims)

The stage-set claim is restated in a known sweep class — update every occurrence, not just the code:

- `docs/memory/pipeline/schemas.md` §"`.status.yaml` `true_impact` Block" — "at apply-finish and hydrate-finish" (~line 204) and "`status.Finish` invokes the helper for stages `apply` and `hydrate` only" (~line 236)
- `docs/specs/templates.md:73` — "written lazily by the apply-finish and hydrate-finish hooks"
- `docs/memory/_shared/configuration.md:78` — "the apply/hydrate `true_impact` write path"
- `src/kit/templates/status.yaml` — the `# true_impact: lazily created on first apply-finish` comment (mirrored at `docs/specs/templates.md:59`)
- Grep `apply-finish and hydrate-finish` / `apply/hydrate` repo-wide before finishing apply to catch any occurrence missed above

No `_cli-fab.md` change expected: `WriteTrueImpact` is internal (no command signature changes; `fab impact` and `fab pr-meta` are untouched). No skill-file behavior change expected: `/git-pr` already runs `fab status finish ship` — but if any skill/SPEC prose restates the stage set, it joins the sweep class (SPEC mirrors of any touched skill file must be updated per constitution).

## Affected Memory

- `pipeline/schemas.md`: (modify) `true_impact` block — computation stages now apply/hydrate/ship; ship-finish is the authoritative write in the standard pipeline (apply/hydrate writes are zeros until commits exist); note the run-kit all-zeros bug as the motivation
- `_shared/configuration.md`: (modify) one-phrase touch — "the apply/hydrate `true_impact` write path" gains ship

## Impact

- **Code**: `src/go/fab/internal/status/true_impact.go` (gate + doc comment), `src/go/fab/internal/status/status.go` (comment only), `src/go/fab/internal/status/true_impact_test.go`
- **Docs**: `docs/memory/pipeline/schemas.md`, `docs/memory/_shared/configuration.md`, `docs/specs/templates.md`, `src/kit/templates/status.yaml`
- **Not affected**: `internal/impact/` (math is correct), `internal/prmeta/` (recomputes live), `/git-pr` skill flow (already calls `fab status finish ship` post-commit), `fab pr-meta`/`fab impact` CLI signatures
- **Runtime surface**: every fab project's `.status.yaml` gains a correct ship-time block on its next `/git-pr`; historical all-zero blocks (run-kit's 4+ changes, fab-kit's own archive) are NOT backfilled — merged branches are gone, so the pre-fix data is unrecoverable and stays as-is
- **No migration**: no schema/field change — the block's shape is unchanged; only the write timing widens. `computed_at_stage: ship` is a new value for an existing free-string field

## Open Questions

- None blocking. (Whether `review-pr`-finish should also recompute — capturing PR-review rework commits — is recorded as a Tentative assumption below, deferred as a follow-up rather than widened into this fix.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix = add `ship` to `WriteTrueImpact`'s stage gate; recompute fires through the existing `status.Finish` → `/git-pr` Step 4b path, post-commit | User-chosen framing; call path verified in source (`status.go:196`, `git-pr.md` Steps 3a→4b ordering) | S:90 R:85 A:95 D:90 |
| 2 | Certain | `fab pr-meta` needs no change | Verified: recomputes live at `prmeta.go:490`; run-kit PR #293 carries a correct Impact table — the report's "reads the persisted block" claim is false at v2.9.2 | S:85 R:90 A:95 D:90 |
| 3 | Confident | Keep apply/hydrate-finish writes (ship supersedes via `computed_at_stage`) rather than removing them | Harmless in the standard flow; real values for early-commit flows (adopt, manual commits); report implied the same ("cleanly supersedes") | S:70 R:85 A:80 D:70 |
| 4 | Confident | No backfill/migration for historical all-zero blocks | Merged branches are deleted — the diff is unrecoverable post-merge; block self-corrects for every future ship; no schema change to migrate | S:65 R:80 A:80 D:70 |
| 5 | Confident | Embed the bug report verbatim in this intake instead of copying the handoff doc into `docs/findings/` | The handoff file lives on the reporter's machine and is not available here; Origin preserves its full substance plus verification deltas | S:65 R:90 A:80 D:75 |
| 6 | Tentative | Ship-only recompute; do NOT also add `review-pr`-finish (which would capture PR-review rework commits) | Reporter's chosen scope; ship-time value is close to final and strictly better than zeros; widening later is trivial (same one-line gate) <!-- assumed: ship-only stage widening — review-pr-finish recompute deferred as follow-up --> | S:55 R:85 A:60 D:45 |

6 assumptions (2 certain, 3 confident, 1 tentative, 0 unresolved).
