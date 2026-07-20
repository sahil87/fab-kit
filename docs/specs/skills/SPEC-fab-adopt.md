# fab-adopt

## Contents

- Summary
- Flow

## Summary

Adopts a **completed-but-off-pipeline** change (scenario B: a feature branch authored without fab, with an **OPEN** or **not-yet-created** PR) into the Fab pipeline. A **MERGED** PR (scenario A — retroactive backfill) is out of scope and STOPs at Step 0. The framing is honest: of the six stages, only **apply** cannot meaningfully re-run on an adopted change (the code already exists), so `/fab-adopt` enters the *real* pipeline late with `apply` marked **skipped** — intake/review/hydrate/ship/review-pr all genuinely run.

A thin orchestrator (the `/fab-proceed` / `/fab-ff` pattern): declares `helpers: [_srad, _generation, _review, _pipeline]` and reuses existing skills/procedures as sub-agents. It introduces only what is genuinely new — the **Intake-from-Diff** and **Plan-from-Diff** procedures in `_generation.md`, and `_review.md`'s **`diff-only`** mode.

**Key design decisions**:

- **Real pipeline entered late, not a fake parallel pipeline** — only apply is `skipped`; every other stage runs (just late, after the code was written). Honest state, not "mark all done" (Constitution II — memory must reflect what shipped).
- **Diff-only review** — the plan-conformance steps validate the implementation against `plan.md` requirements, but an adopted change has no forward requirements to conform to, so those steps are omitted from the single review agent's prompt via the general `mode: diff-only` parameter on `_review.md` (not an adopt-specific branch). An empty result (no available external reviewer) passes best-effort.
- **State via existing transitions** — `fab status skip {name} apply` (cascades downstream → skipped) then `fab status reset {name} review fab-adopt` (skipped → active, cascades downstream → pending) yields `apply=skipped, review=active`. **No Go transition added.**
- **Intake + thin plan in one main-session pass** — both artifacts describe one fixed existing diff, so the same agent reads the diff once and writes both (no dispatched-apply split that would invite drift). A human-confirmation checkpoint between them is the late deliberation the bypass skipped.
- **PR Meta retrofit** — Step 5's `/git-pr` injects `## Meta` onto the existing OPEN PR via its Step 3d body-retrofit path (gated on body-lacks-`## Meta`; reuses `fab pr-meta` + `gh pr edit --body-file -`). **No Go change.**

## Flow

```
User invokes /fab-adopt [<slug>]
│
├─ Step 0: Guards & diff base (reuse /git-pr guard idioms — STOP before any mutation)
│  ├─ Bash: git branch --show-current → detached HEAD or default branch → STOP
│  ├─ Bash: gh pr view --json number,state,url
│  │        → MERGED → STOP (scenario A, out of scope)
│  │        → OPEN / none → proceed (capture {pr_state}, {pr_url}, {pr_number})
│  ├─ Collision guard: fab resolve --folder "$(git branch --show-current)" --or-none
│  │        prints a folder name (≠ "(none)") → STOP
│  │        (already in the pipeline → /fab-continue; 260720-dow0)
│  └─ Bash: base=$(git merge-base HEAD origin/{default});
│           git diff {base}...HEAD + --name-only
│           empty diff → STOP (nothing to adopt)
│
├─ Steps 1+2: ONE main-session generation pass (same agent, NOT dispatched —
│  │          reads the diff + PR body once)
│  ├─ Bash: fab change new --slug {slug}; activate (branch already exists —
│  │        fab-new Step 11 row 1/2 "already active"/"checked out")
│  ├─ Reconstruct intake.md via Intake-from-Diff Procedure (_generation.md):
│  │        Origin = adopted from {PR or branch}; Why/What-Changes from diff + PR body;
│  │        Affected Memory inferred from touched docs/memory/ domains;
│  │        Impact from changed paths; apply SRAD + fab score
│  ├─ Human-confirmation checkpoint (confirm/correct reconstructed intent + SRAD)
│  │        → on confirm: fab status advance {name} intake; finish {name} intake
│  │          (auto-activates apply)
│  └─ Write MINIMAL plan.md via Plan-from-Diff Procedure (_generation.md):
│           plain-language ## Requirements (the only part hydrate reads),
│           all-[x] ## Tasks + ## Acceptance stubs, NO R#/T#/A# scaffolding,
│           "Adopted change…" header note
│
├─ Step 2 (state): apply → skipped, review → active (existing transitions, no Go change)
│  ├─ Bash: fab status skip {name} apply          (cascades downstream → skipped)
│  ├─ Bash: fab status reset {name} review fab-adopt (skipped → active, downstream → pending)
│  └─ Bash: fab status set-summary {name} "adopted off-pipeline change; apply skipped"
│
├─ Step 3: Review — dispatched, mode: diff-only (resolve + branch on dispatch= — native Agent tool / CLI adapter)
│  ├─ Bash: fab resolve-agent review --alias (surface model=/effort=/dispatch=;
│  │        branch on dispatch= — native or CLI adapter, 260702-aetz)
│  ├─ Dispatch /fab-continue Review Behavior, mode: diff-only
│  │        (prompt carries the block-contract carve-out: no fab status
│  │         TRANSITION commands, terminal fab status refresh required,
│  │         return results only; preconditions skipped in diff-only;
│  │         the single review agent reads git diff {base}...HEAD natively)
│  └─ Verdict (owned here):
│        pass (incl. zero findings, best-effort) → fab status finish {name} review fab-adopt
│        fail → auto-rework per _pipeline.md budget (autonomous) /
│               hand findings back (interactive default for hand-authored code)
│
├─ Step 4: Hydrate — dispatched, verbatim (_pipeline.md Step 3; resolve + branch on dispatch=)
│  └─ Dispatch /fab-continue Hydrate Behavior (same block-contract carve-out) → on success fab status finish {name} hydrate fab-adopt
│        (the permanent-loss recovery — docs/memory/ finally reflects what shipped)
│
├─ Step 5: Ship — dispatch /git-pr {name} (folder name, not bare id)
│  └─ OPEN PR → existing-PR path + Step 3d Meta retrofit (body-lacks-## Meta);
│     none → /git-pr creates the PR fresh; /git-pr runs finish ship itself
│     (auto-activates review-pr)
│
└─ Step 6: Land in review-pr → print honest-state summary + Next: /git-pr-review
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | `git` (branch/merge-base/diff), `gh pr view`, `fab change new`, `fab status skip/reset/advance/finish/set-summary`, `fab score`, `fab resolve-agent` |
| Read | The diff + PR body (Step 0/1), templates via `$(fab kit-path)` (through the `_generation.md` procedures) |
| Write | `intake.md` + `plan.md` in the reconstructed change folder (the one main-session pass) |
| Agent | Review (diff-only) + Hydrate sub-agents; Step 5 `/git-pr` (folder-name target) |

### Sub-agents

| Agent | When | Purpose |
|-------|------|---------|
| `/fab-continue` Review Behavior (`mode: diff-only`) | Step 3 | Diff-only review (plan-conformance steps omitted); verdict transition owned by `/fab-adopt` |
| `/fab-continue` Hydrate Behavior | Step 4 | Write `docs/memory/` from the thin plan's `## Requirements` — the permanent-loss recovery |
| `/git-pr {name}` | Step 5 | Commit/push the reconstructed `fab/` artifacts, retrofit the PR `## Meta` block (Step 3d), finish ship |

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| Intake finish | `fab status advance/finish {name} intake` | `/fab-adopt` (after the human-confirmation checkpoint) |
| State composition | `fab status skip {name} apply`; `fab status reset {name} review fab-adopt`; `fab status set-summary {name} …` | `/fab-adopt` Step 2 |
| Review verdict | `fab status finish/fail {name} review fab-adopt` | `/fab-adopt` (orchestrator owns it; sub-agent does not) |
| Hydrate finish | `fab status finish {name} hydrate fab-adopt` | `/fab-adopt` Step 4 |
| Ship finish | `fab status finish {name} ship git-pr` | `/git-pr` (best-effort, auto-activates review-pr) |
