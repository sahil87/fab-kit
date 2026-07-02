# Intake: Adopt an off-pipeline change into the Fab pipeline

**Change**: 260630-t54n-adopt-off-pipeline-change
**Created**: 2026-06-30

## Origin

> Many times, users discuss with AI agents to create changes, and also create a git PR. The fab pipeline gets skipped, either by mistake or deliberately. Now after the change is completed, there should a way to "bring it" into the fab pipeline. What key parts would have gotten missed by bypassing the fab pipeline? How to ensure those get followed?

Conversational design session (`/fab-discuss` → exploratory). The user steered the scope across several turns; the resulting design diverges from the normal pipeline in deliberate, reasoned ways:

- **Scenario narrowed to (B), not (A).** The user explicitly de-prioritised retroactive backfill of an *already-merged* PR (scenario A) and chose **mid-flight adoption** (scenario B): a feature branch with an **OPEN PR (or no PR yet)** whose code was authored without fab. The motivating trigger is *social* — "a PR review team points out: hey, you forgot to use fab-kit and might have missed a few checks."
- **Generic-user lens, not fab-kit-repo lens.** An early proposal leaned on fab-kit's own SPEC-mirror sweep discipline; the user corrected this — the skill must serve *any* project using fab-kit and lean on *their* `code-quality.md` / `code-review.md` / `review_tools` config, not fab-kit-specific mirror classes.
- **Review = outward-only.** After weighing the loss (inward review checks conformance against requirements — the most valuable check for a bypassed change), the user chose **outward-only** review for adopted changes, implemented as a **general `mode` parameter** in `_review.md` (not an adopt-specific branch).
- **PR Meta decoration is in scope.** The user wants the git PR's `## Meta` decoration added retrospectively to the existing open PR.
- **Honest state.** Stages that can't truly re-run are marked truthfully; the user chose "mark skipped + note" over "mark done."
- **Intake + plan written by the same main-session agent.** The user confirmed both artifacts are generated in one pass by the same agent reading the diff once (no dispatched-apply split), because both merely *describe one fixed existing diff* — a context boundary between them would only invite drift and waste.

Two open verification items raised during the discussion were resolved against the Go source *before* this intake was written (see Assumptions #8, #9) — both land as **skill-layer-only** changes; no Go binary change is required.

## Why

**Problem.** The Fab pipeline's value is not the PR ceremony — `/git-pr` is a thin shipping step. The value lives in four artifacts an ad-hoc PR never produces: the **intake + confidence decision record** (the deliberated "why"), the **plan's requirements** (the conformance baseline), the **review pass** (correctness/quality/cross-reference checks against the project's own config), and — with permanent consequences — **hydrate's update to `docs/memory/`**. Constitution II makes memory the source of truth; a change shipped outside the pipeline never updates memory, so the docs silently drift from what actually shipped, and the change is invisible to the generated `log.md` history.

**Consequence if unfixed.** Teams that adopt fab-kit but occasionally bypass it (deliberately or by mistake) accumulate undocumented changes. There is today **no supported path to bring a completed-but-off-pipeline change back in**. The only options are to re-do the work through the pipeline (wasteful — the code already exists and may be reviewed) or to leave memory permanently stale. A reviewer's "you skipped fab-kit" comment has no actionable remedy.

**Why this approach.** Of the six stages, exactly one — **apply** — cannot meaningfully re-run on an adopted change (the code already exists; there is nothing to generate). Every other stage *can* run for real, just *late*: intake reconstructed from the diff, review run on the diff before merge, hydrate writing memory before merge, ship retrofitting the PR decoration, review-pr resuming normally. So adoption is **not** a parallel "fake pipeline" — it is the *real* pipeline entered late, with apply skipped. This is the honest, minimal framing and it dictates the whole design: build a thin orchestrator that reuses existing skills as sub-agents (the `/fab-proceed`/`/fab-ff` pattern), introduces only what is genuinely new (diff→intake, thin diff→plan, an outward-only review mode), and closes two small skill-layer gaps.

## What Changes

### 1. New skill `/fab-adopt` — `src/kit/skills/fab-adopt.md` (+ `docs/specs/skills/SPEC-fab-adopt.md` mirror)

A thin orchestrator skill. Arguments: optional `<slug>` (derived from branch name / PR title if omitted). It reads the `_preamble` always-load layer and declares helpers `[_srad, _generation, _review, _pipeline]`.

**Step 0 — Guards & diff base** (reuse `/git-pr`'s guard idioms verbatim):
- `git branch --show-current` — STOP on **detached HEAD** or the **default branch** (with `/git-pr`'s messages).
- `gh pr view --json number,state,url` — if `state == MERGED`, STOP: *"PR is already merged — that's retroactive backfill (scenario A), out of scope for /fab-adopt. Adopt operates on in-flight (open or not-yet-created) PRs."* `OPEN` and `none` both proceed.
- **Collision guard**: if a fab change already maps to this branch (`fab change resolve "$(git branch --show-current)"` succeeds), STOP and point at `/fab-continue` — it is already in the pipeline.
- Resolve the diff base: `base=$(git merge-base HEAD origin/{default})`; capture `git diff {base}...HEAD` and `git diff --name-only {base}...HEAD`. STOP if the diff is empty (nothing to adopt).

**Steps 1+2 — ONE main-session generation pass** (same agent, not dispatched — reads the diff + PR body once):
1. `fab change new --slug {slug}` against the current branch; activate it (the change branch already exists — `/fab-new`'s Step 11 row 1/2 "already active"/"checked out" applies).
2. **Reconstruct `intake.md`** via the new **Intake-from-Diff Procedure** (added to `_generation.md`): Origin = `adopted from {PR or branch}`; Why/What-Changes synthesised from the diff + PR body; **Affected Memory** inferred from which `docs/memory/` domains the diff touches; Impact from the changed paths. Apply SRAD and `fab score`.
3. **Human confirmation checkpoint** — present the reconstructed intent + SRAD assumptions for the user to confirm/correct. This *is* the late deliberation the bypass skipped (mirrors `/fab-new`'s interactive intake moment). On confirm: `fab status advance {name} intake` → `finish {name} intake` (auto-activates apply).
4. **Write a deliberately MINIMAL `plan.md`** via the new **Plan-from-Diff Procedure** (added to `_generation.md`), from the *same* understanding (no re-read of the diff):
   - `## Requirements` — **plain-language** restatement of the intake's What-Changes, grouped by change area. **This is the only part hydrate reads to write accurate memory, so effort concentrates here.** Largely lifts the intake's What-Changes into plan form.
   - `## Tasks` — a single all-`[x]` stub (e.g. `- [x] Adopted: implementation authored outside the pipeline (see {PR}).`).
   - `## Acceptance` — a single all-`[x]` stub.
   - **NO** `R#`/`T{NNN}`/`A-{NNN}` traceability IDs, GIVEN/WHEN/THEN scenarios, phases, or `[P]` markers — the apply↔review traceability loop never runs for adopted changes, so that scaffolding is dead weight. (The stable parser contract is only the three heading literals `## Requirements`/`## Tasks`/`## Acceptance`, confirmed against `templates/plan.md`.)
   - Carry a header note: *"Adopted change — code authored off-pipeline. Apply was skipped; this plan is reverse-engineered from the branch diff to feed hydrate."*

**Step 2 (state) — apply → skipped, review → active** (the resolved transition path, Assumption #8):
```bash
fab status skip {name} apply          # apply → skipped; cascades review/hydrate/ship/review-pr → skipped
fab status reset {name} review {driver} # review skipped → active; cascades hydrate/ship/review-pr back to pending
```
Net state: `apply: skipped`, `review: active`, `hydrate/ship/review-pr: pending`. Honest, uses only existing transitions, **no Go change**. Record the off-pipeline fact via `fab status set-summary {name} "adopted off-pipeline change; apply skipped"`.

**Step 3 — Review, dispatched, `mode: outward-only`** (the new `_review.md` parameter, §2 below): dispatch `/fab-continue` Review Behavior as a sub-agent (per the standard dispatch contract + `fab resolve-agent review --alias`), passing `mode: outward-only`. Outward review already uses `git diff {base}...HEAD` natively, so no file-set prompt hack is needed. The orchestrator owns the verdict transition (pass → `finish review`; fail → auto-rework per `_pipeline.md` budget, or hand back).

**Step 4 — Hydrate, dispatched, verbatim**: reuse `_pipeline.md` Step 3 unchanged. This is the permanent-loss recovery — `docs/memory/` finally reflects what shipped. On success `finish {name} hydrate`.

**Step 5 — Ship (retrofit Meta onto the existing OPEN PR)**: dispatch `/git-pr {name}`. Because the PR is OPEN, `/git-pr` takes its existing-PR path — which today only *records* the URL. The new `/git-pr` body-retrofit path (§3 below) injects the `## Meta` block when the open PR's body lacks one.

**Step 6 — Land in review-pr**: `finish ship` auto-activates `review-pr`; output `Next: /git-pr-review`. Normal tail resumes.

**Honest-state summary the skill prints**: *intake, review, hydrate, ship, review-pr all genuinely ran (just late, after the code was written). Only apply is `skipped`.*

### 2. New `mode` parameter in `src/kit/skills/_review.md` (+ `SPEC-_review.md` mirror)

Add a **Review Mode** concept to `_review.md` — the single authority for review dispatch, so `/fab-continue`, `/fab-ff`, `/fab-fff`, and `/fab-adopt` all inherit it.

- A `mode` parameter passed in the review dispatch: `full` (default — inward + outward) and `outward-only` (outward sub-agent only). **Do not** add a speculative `inward-only` value — no caller exists today (parsimony; the repo's own review enforces this).
- **Precondition gating**: the inward Preconditions (`plan.md` MUST exist with `## Tasks`/`## Acceptance`; all tasks `[x]`) are checked **only** in `full` mode. In `outward-only` they are skipped — there is nothing for inward to validate.
- **Parallel Dispatch**: dispatch only the sub-agent(s) selected by `mode`.
- **Findings Merge / pass-fail**: unchanged — "any must-fix → fail" works with a single source. An empty outward result (all `review_tools` disabled/unavailable) → zero findings → **passes** (best-effort; adoption must not hard-block when no external reviewer is available).
- Default is `full` (param omitted) so all existing callers are unaffected.

### 3. `/git-pr` body-retrofit path — `src/kit/skills/git-pr.md` (+ `SPEC-git-pr.md` mirror)

Close the gap that `/git-pr` injects `## Meta` only on PR **create** (Step 3c), never when an open PR already exists. Add to the existing-OPEN-PR path: if the PR body lacks a `## Meta` block, render it via `fab pr-meta {name} --type {type} --issues "{issues}"` and apply with `gh pr edit --body-file -` (stdin). Gated on body-lacks-`## Meta` for idempotency (a second run is a no-op). **No Go change** — `prmeta.Render`/`fab pr-meta` already exist and are reused.

### 4. Aggregate spec & doc sweeps

Per the project's Sibling & Mirror Sweep discipline, update every place that enumerates skills or stages:
- `docs/specs/skills.md` — add `/fab-adopt` behavior.
- `docs/specs/glossary.md` — add "adopt" terminology.
- `docs/specs/overview.md` and `docs/specs/user-flow.md` — note the adoption entry-point into the pipeline.
- `_preamble.md` Next-Steps **State Table** — adoption is an alternate entry; assess whether a row/note is warranted.
- `fab-help.md` — list the new command.

## Affected Memory

- `pipeline/execution-skills.md`: (modify) document `/fab-adopt` — the adoption orchestrator, its skip-apply/reset-review state path, and the `/git-pr` body-retrofit path
- `pipeline/planning-skills.md`: (modify) note the Intake-from-Diff and Plan-from-Diff generation procedures added to `_generation.md`, and that adopt generates both artifacts in one main-session pass
- `pipeline/change-lifecycle.md`: (modify) record adoption as an alternate pipeline entry-point and the `apply: skipped` honest-state pattern
- `pipeline/schemas.md`: (modify) document the `skip apply` + `reset review` transition composition that yields `apply=skipped, review=active` (no new transition added)

## Impact

- **Skill files**: `src/kit/skills/fab-adopt.md` (new), `_review.md` (mode param), `_generation.md` (two new procedures), `git-pr.md` (body-retrofit path), `fab-help.md`.
- **Spec mirrors** (constitution-required): `SPEC-fab-adopt.md` (new), `SPEC-_review.md`, `SPEC-_generation.md`, `SPEC-git-pr.md`; aggregate specs `skills.md`, `glossary.md`, `overview.md`, `user-flow.md`.
- **Go binary**: **none required.** Both open items resolved to skill-layer changes (state composition uses existing `skip`/`reset` transitions; Meta-retrofit reuses existing `fab pr-meta`/`prmeta.Render` + `gh pr edit`).
- **Tests**: no Go test changes expected (no Go change). Skill changes carry SPEC-mirror updates per the constitution.
- **External tools used**: `git`, `gh` (incl. `gh pr edit --body-file -`), `fab pr-meta`, `fab status skip/reset/finish/set-summary`, `fab resolve-agent`.

## Open Questions

- Should `/fab-adopt` offer a `--no-pr` mode for branches with no PR yet (reconstruct + review + hydrate, then create the PR fresh via `/git-pr`'s normal create path)? The current design handles `pr_state == none` by letting Step 5's `/git-pr` create the PR — likely sufficient, but worth confirming during apply.
- Should the auto-rework loop apply to adopted-change review at all, given the code is already written and the author may not want autonomous edits to a hand-authored branch? (Leaning: yes for `/fab-adopt` when run autonomously, but the interactive default should hand findings back rather than auto-edit. Resolve at apply.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scenario is (B) mid-flight adoption of an OPEN/not-yet-created PR; MERGED PRs (scenario A) are explicitly out of scope and STOP | User stated this directly and twice in the discussion | S:95 R:90 A:90 D:95 |
| 2 | Certain | Review for adopted changes is **outward-only**, via a general `mode` param on `_review.md` (not an adopt-specific branch) | User chose both the outward-only behavior and the general-mode placement explicitly | S:95 R:85 A:90 D:95 |
| 3 | Certain | Only **apply** is marked `skipped`; intake/review/hydrate/ship/review-pr genuinely run; honest "skipped + note" state (not "mark done") | User chose "mark skipped + note" explicitly | S:95 R:90 A:90 D:95 |
| 4 | Certain | Intake + plan are generated by the **same main-session agent in one pass** reading the diff once — not via the dispatched-apply path | User confirmed this and the drift/waste rationale explicitly | S:90 R:85 A:90 D:90 |
| 5 | Confident | The thin `plan.md` carries plain-language `## Requirements` (for hydrate) + all-`[x]` `## Tasks`/`## Acceptance` stubs; no R#/T#/A# scaffolding | Derived from verified facts: hydrate is the only consumer needing requirements; the parser contract is just the three headings (confirmed against templates/plan.md); apply↔review loop never runs | S:85 R:80 A:85 D:80 |
| 6 | Confident | `/fab-adopt` is a thin orchestrator reusing `_pipeline.md`/`_generation.md`/`_review.md`/existing skills as sub-agents (the `/fab-proceed`/`/fab-ff` pattern) | Matches the established orchestrator pattern and the constitution's single-authority discipline | S:85 R:80 A:85 D:80 |
| 7 | Confident | Outward-only with zero findings (no external reviewer available) **passes** (best-effort), not blocks | Matches `_review.md`'s existing graceful-no-op contract for an empty outward cascade; adoption shouldn't hard-block on tool availability | S:85 R:80 A:80 D:80 |
| 8 | Certain | apply→skipped + review→active is achieved by `skip apply` (cascades all downstream → skipped) then `reset review` (skipped→active, cascades its downstream → pending) — **no Go change** | Verified against `src/go/fab/internal/status/status.go`: `skip` From={pending,active}→skipped with forward cascade (L246–278); `reset` From includes `skipped`→active (L41); `start` does NOT accept skipped→active | S:95 R:90 A:90 D:95 |
| 9 | Certain | Meta-retrofit onto an open PR is skill-layer only: `fab pr-meta` + `gh pr edit --body-file -`, gated on body lacking `## Meta` — **no Go change** | Verified: `prmeta.Render` (L93) + `fab pr-meta` CLI already exist; `git-pr.md` injects Meta only on create (L279); `gh pr edit --body`/`--body-file` confirmed available | S:95 R:90 A:90 D:95 |
| 10 | Tentative | Aggregate sweep targets are skills.md, glossary.md, overview.md, user-flow.md, _preamble State Table, fab-help.md | Standard sweep class for a new skill; exact set confirmed during apply by grepping the enumerations | S:75 R:70 A:75 D:65 |

10 assumptions (6 certain, 3 confident, 1 tentative, 0 unresolved).
