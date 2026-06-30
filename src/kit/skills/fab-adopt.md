---
name: fab-adopt
description: "Adopt a completed off-pipeline change (a feature branch with an OPEN or not-yet-created PR, authored without fab) into the Fab pipeline — reconstruct intake + plan from the diff, run review/hydrate/ship/review-pr for real, with apply marked skipped."
helpers: [_srad, _generation, _review, _pipeline]
---

# /fab-adopt [<slug>]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Contents

- Purpose
- Arguments
- Behavior
- Output
- Error Handling
- Key Properties

---

## Purpose

Bring a **completed-but-off-pipeline** change into the Fab pipeline. The trigger is mid-flight adoption (scenario B): a feature branch whose code was authored **without** fab, with an **OPEN PR or no PR yet** — typically after a reviewer points out "you skipped fab-kit and might have missed a few checks."

Of the six stages, exactly one — **apply** — cannot meaningfully re-run on an adopted change (the code already exists; there is nothing to generate). Every other stage *can* run for real, just *late*: intake reconstructed from the diff, review run on the diff before merge, hydrate writing memory before merge, ship retrofitting the PR decoration, review-pr resuming normally. So `/fab-adopt` is **not** a parallel "fake pipeline" — it is the *real* pipeline entered late, with apply skipped.

`/fab-adopt` is a thin orchestrator: it reuses existing skills/procedures as sub-agents (the `/fab-proceed` / `/fab-ff` pattern) and introduces only what is genuinely new (the diff→intake and thin diff→plan procedures in `_generation.md`, and `_review.md`'s `outward-only` mode).

**Scope** — `/fab-adopt` covers scenario B only. A **MERGED** PR (scenario A — retroactive backfill of already-merged work) is explicitly out of scope and STOPs at Step 0.

---

## Arguments

- **`<slug>`** *(optional)* — the folder-name suffix for the reconstructed change. If omitted, derive it from the PR title (when a PR exists) or the branch name. Same slug semantics as `/fab-new`.

---

## Behavior

### Step 0 — Guards & diff base

Reuse `/git-pr`'s guard idioms verbatim. Run these checks **before any mutation**; STOP (no `fab change new`, no status mutation) on any failure.

```bash
branch=$(git branch --show-current)
default_branch=$(git symbolic-ref --short refs/remotes/origin/HEAD 2>/dev/null | sed 's|^origin/||')
[ -n "$default_branch" ] || default_branch=$(gh repo view --json defaultBranchRef -q .defaultBranchRef.name 2>/dev/null)
[ -n "$default_branch" ] || default_branch=$(git rev-parse --verify -q refs/remotes/origin/main >/dev/null && echo main || echo master)
gh pr view --json number,state,url 2>/dev/null || echo "NO_PR"
```

1. **Detached HEAD / default-branch guard** (reuse `/git-pr`'s messages):
   - `branch` empty (detached HEAD) → STOP: `Cannot ship from a detached HEAD — check out a branch first (run /git-branch).`
   - `branch` is the default branch (or literal `main`/`master`) → STOP: `Cannot adopt from the default branch ({default_branch}).`
2. **PR-state guard**: from `gh pr view`:
   - `state == MERGED` → STOP: `PR is already merged — that's retroactive backfill (scenario A), out of scope for /fab-adopt. Adopt operates on in-flight (open or not-yet-created) PRs.`
   - `OPEN` and `none` (no PR) both **proceed**. Capture `{pr_state}`, `{pr_url}`, `{pr_number}` for later steps.
3. **Collision guard**: if a fab change already maps to this branch (`fab change resolve "$(git branch --show-current)" 2>/dev/null` succeeds), STOP: `Branch '{branch}' already maps to fab change '{name}' — it is already in the pipeline. Run /fab-continue (or /fab-fff) to advance it.`
4. **Resolve the diff base and capture the diff**:
   ```bash
   base=$(git merge-base HEAD "origin/$default_branch")
   git diff "$base"...HEAD            # the adopted diff
   git diff --name-only "$base"...HEAD # the changed file list
   ```
   If the diff is **empty** (no changed files) → STOP: `No diff against {default_branch} — nothing to adopt.`

### Steps 1+2 — ONE main-session generation pass

These run in the **main session** (the same agent, NOT a dispatched sub-agent) reading the diff + PR body once — both artifacts merely *describe one fixed existing diff*, so a context boundary between them would only invite drift and waste.

1. **Create + activate the change**: `fab change new --slug {slug}` against the current branch, then activate it. The change branch already exists, so `/fab-new`'s Step 11 row 1 ("already active") or row 2 ("checked out") applies — do NOT recreate or rename the branch.
2. **Reconstruct `intake.md`** via the **Intake-from-Diff Procedure** (`_generation.md`), passing the diff, the changed-file list, and the PR body/title (or branch name). Apply SRAD and run `fab score`.
3. **Human-confirmation checkpoint**: present the reconstructed intent (Origin / Why / What Changes / Affected Memory) + the SRAD assumptions for the user to confirm or correct. This *is* the late deliberation the bypass skipped (it mirrors `/fab-new`'s interactive intake moment). On confirm:
   ```bash
   fab status advance {name} intake
   fab status finish {name} intake   # auto-activates apply
   ```
4. **Write a deliberately MINIMAL `plan.md`** via the **Plan-from-Diff Procedure** (`_generation.md`), from the *same* understanding (no re-read of the diff): plain-language `## Requirements` (the only part hydrate reads — concentrate effort here), an all-`[x]` `## Tasks` stub, an all-`[x]` `## Acceptance` stub, no R#/T#/A# scaffolding, plus the "Adopted change …" header note.

### Step 2 (state) — apply → skipped, review → active

Compose the honest state from **existing** transitions only — no Go change:

```bash
fab status skip {name} apply             # apply → skipped; cascades review/hydrate/ship/review-pr → skipped
fab status reset {name} review {driver}  # review skipped → active; cascades hydrate/ship/review-pr back to pending
fab status set-summary {name} "adopted off-pipeline change; apply skipped"
```

`{driver}` is `fab-adopt`. Net state: `apply: skipped`, `review: active`, `hydrate/ship/review-pr: pending`. (Verified transition composition: `skip` is `{pending,active}→skipped` with forward cascade; `reset` accepts `skipped→active` and cascades its downstream back to `pending`.)

### Step 3 — Review (dispatched, `mode: outward-only`)

Resolve the review model: run `fab resolve-agent review --alias`, surface the resolved `model=/effort=`, and apply both halves (model via the Agent `model` param; effort via the imperative prompt instruction) per `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution.

Dispatch `/fab-continue` Review Behavior as a sub-agent (Agent tool, `general-purpose`), passing **`mode: outward-only`** (the `_review.md` parameter). The prompt MUST include the standard subagent context files and **"do NOT run `fab status` commands; return results only"** — this orchestrator owns the verdict transition. Outward review reads `git diff {base}...HEAD` natively, so no file-set prompt hack is needed; inward preconditions (`plan.md` tasks all `[x]`) are skipped in `outward-only` mode.

**Verdict** (owned here):
- **Pass** (no must-fix, including zero findings — outward-only with no available external reviewer passes best-effort): `fab status finish {name} review {driver}`.
- **Fail**: auto-rework per `_pipeline.md` § Auto-Rework Loop budget when run autonomously; when run interactively, **hand the findings back** to the user rather than auto-editing a hand-authored branch (the default for adopted code is hand-back — see Key Properties). On exhaustion, stop per the bracket's exhaustion rule.

### Step 4 — Hydrate (dispatched, verbatim)

Reuse `_pipeline.md` Step 3 unchanged: resolve the hydrate model, dispatch `/fab-continue` Hydrate Behavior as a sub-agent (same prompt contract — do NOT run `fab status`; return results only). This is the permanent-loss recovery — `docs/memory/` finally reflects what shipped. On success: `fab status finish {name} hydrate {driver}`.

### Step 5 — Ship (retrofit Meta onto the existing PR)

Dispatch `/git-pr {name}` (pass the **folder name**, not a bare id). Because the PR is OPEN (or `none`), `/git-pr` takes its existing-PR path (or creates the PR fresh when `pr_state == none`). Its **Step 3d body-retrofit** injects the `## Meta` block when the open PR's body lacks one (idempotent — gated on body-lacks-`## Meta`). `/git-pr` runs `finish ship` itself (best-effort), which auto-activates review-pr.

### Step 6 — Land in review-pr

After ship, `review-pr` is active. Print `Next: /git-pr-review`. The normal pipeline tail resumes from here.

**Honest-state summary** the skill prints at the end:

```
Adopted {name}.

  intake    ✓ ran (reconstructed from the diff)
  apply     — skipped (code authored off-pipeline)
  review    ✓ ran (outward-only)
  hydrate   ✓ ran (docs/memory/ updated)
  ship      ✓ ran (## Meta retrofitted onto the PR)
  review-pr → active

Only apply is skipped; every other stage genuinely ran (just late, after the code was written).
```

---

## Output

```
/fab-adopt — adopting {branch} into the pipeline

--- Reconstruct ---
{intake reconstructed from diff; human-confirmation checkpoint}

--- State ---
apply → skipped, review → active

--- Review (outward-only) ---
{review output}

--- Hydrate ---
{hydrate output}

--- Ship ---
{git-pr output, incl. Meta retrofit}

{honest-state summary}

Next: /git-pr-review
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Detached HEAD | STOP: "Cannot ship from a detached HEAD — check out a branch first (run /git-branch)." |
| On default branch | STOP: "Cannot adopt from the default branch ({default_branch})." |
| PR already MERGED | STOP: scenario-A out-of-scope message (Step 0.2) |
| Branch already maps to a fab change | STOP: point at /fab-continue (already in the pipeline) |
| Empty diff against default branch | STOP: "No diff against {default_branch} — nothing to adopt." |
| `fab change new` failure | Surface stderr and STOP |
| Review fails (autonomous) | Auto-rework per `_pipeline.md` budget; on exhaustion stop with the per-cycle summary |
| Review fails (interactive) | Hand findings back to the user (default for hand-authored code); do not auto-edit |
| Hydrate fails | Surface the sub-agent's failure and STOP (review remains done; re-runnable) |
| `/git-pr` fails | Surface its error and STOP (review/hydrate already done; re-runnable) |

---

## Key Properties

| Property | Value |
|----------|-------|
| Idempotent? | Partially — Step 0 guards STOP cleanly before any mutation; the collision guard makes a re-run after the change is created route to `/fab-continue` rather than re-create. The dispatched stages (review/hydrate/ship) are themselves resumable/idempotent, and the Step 5 Meta retrofit is gated on body-lacks-`## Meta` |
| Advances stage? | Yes — reconstructs intake, marks apply `skipped`, then runs review → hydrate → ship → review-pr via existing transitions |
| Modifies `.fab-status.yaml`? | Yes — activates the reconstructed change (Step 1) |
| Modifies git state? | Indirectly — Step 5's `/git-pr` commits/pushes the reconstructed `fab/` artifacts and retrofits the PR body; `/fab-adopt` makes no commit itself |
| Go change? | None — state composed from existing `skip`/`reset` transitions; Meta retrofit reuses `fab pr-meta` + `gh pr edit` |

---

Next: {derive at runtime per `_preamble.md` § Lookup Procedure — after ship the state is review-pr (active): `/git-pr-review`}
