# Intake: Orchestrator Dispatch Targets + Review-PR Failure Recovery

**Change**: 260612-w7dp-orchestrator-dispatch-review-pr-recovery
**Created**: 2026-06-12

## Origin

One-shot invocation: `/fab-new w7dp` (backlog ID). Backlog entry `fab/backlog.md` `[w7dp]`:

> Skills-audit batch 2/5 — orchestrator dispatch targets + review-pr failure recovery. DEPENDS: wave 2 — after k4ge merges (skill text references the fixed exit-code contract) AND after g8st merges (collides with it on git-pr.md Step 0/guard block, git-pr-review.md failure paths, fab-operator.md §6). GOAL: every inter-skill hand-off names a verified target; no dead-end states. ACTIONS (report §2 Themes 2+4): [full action list — see What Changes below, which reproduces every action with re-verified HEAD anchors]. CONSTRAINTS: SPEC mirror per touched skill; src/kit is canonical. REPORT: docs/specs/findings/skills-review-2026-06-12.md §2 Themes 2+4.

Source report: `docs/specs/findings/skills-review-2026-06-12.md` §2 Theme 2 (`ship-dispatch-target-guard`) + Theme 4 (`review-pr-failed-recovery`). The report's line numbers are vs commit `1431a9c3`; PRs #394–#401 have since merged (including the colliding k4ge and g8st changes). **Every finding below was re-located and re-verified against current HEAD at intake time** — anchors cited here are current. Dependency check passed: both #395 (k4ge) and #401 (g8st) are in this branch's history.

## Why

1. **The pain point.** The orchestration layer (fab-fff, fab-proceed, fab-operator, _pipeline) hands off between skills via strings and substring resolution with no target verification, and the post-PR failure state has no exit:
   - fab-fff Steps 4–5 pass `change: {id}` to `/git-pr` and `/git-pr-review`, but both skills self-resolve only the ACTIVE change (argless `fab change resolve`, `git-pr.md:19-22`, `git-pr-review.md:19`). Under fab-fff's advertised `<change-name>` override, Steps 1–3 work the override change while ship/review-pr mutate the active change's status and push whatever branch is checked out (**must-fix** — wrong-change PR).
   - Five recovery pointers invoke `/fab-clarify intake` — unexecutable: clarify lost its target-artifact argument in 1.10.0, so `intake` parses as a change-name substring; clarify is stage-guard-blocked post-intake; and the promised requirements regeneration never happens (plan.md is preserved on reset) (**must-fix**).
   - `review-pr: failed` is a dead end: fab-continue's dispatch table has no row for it, so preflight at that state matches "all done" and reports a FAILED PR review as complete (3 convergent findings).
   - Reset re-runs error when the target stage is already `active` — a Constitution Principle III (idempotency) violation; state enumerations across fab-continue/fab-switch/fab-status disagree with `status.go` `AllowedStates`.
2. **If we don't fix it:** autonomous pipelines ship the wrong branch under the override, exhausted rework loops hand users a command that errors, failed PR reviews silently read as complete, operator autopilot double-dispatches commands and sends unparseable `&&` chains, and re-running recovery steps errors instead of no-opping.
3. **Why this approach:** these are skill-text contract fixes (markdown), batched as audit wave 2. The Go-side groundwork already landed in k4ge (`--check-gate` non-zero exit, AllowedStates-enforced transitions, reset From includes `skipped`, iterations-preserving reset cascade) — this change aligns the skill layer to those contracts and closes the dispatch/recovery seams. Themes 1 (k4ge) and 3 (g8st) are merged; the Theme 4 Go metrics-cascade fix also landed in k4ge, so **this change touches no Go code**.

## What Changes

All edits go to `src/kit/skills/*.md` (canonical) with a `docs/specs/skills/SPEC-*.md` mirror update per touched skill. Never edit `.claude/skills/` directly (gitignored deploy copies).

### 1. `/git-pr` + `/git-pr-review`: explicit change argument + branch-matches-change guard (must-fix)

- `git-pr.md` Step 0 (HEAD :19-22) resolves via argless `fab change resolve 2>/dev/null`; `git-pr-review.md` Step 0 (HEAD :19) likewise. `fab-fff.md` Steps 4–5 (HEAD :43, :53) already dispatch with `change: {id}` — the skills ignore it.
- Add an optional explicit `<change>` argument to both skills: when provided, resolve via `fab change resolve <change>` (transient override, `.fab-status.yaml` untouched) and use that change for ALL status transitions and artifact paths. fab-fff's dispatch prompts pass the id through.
- Add a **branch-matches-change guard** to both: before any commit/push/status mutation, verify `git branch --show-current` equals the resolved change's folder name; on mismatch STOP with the mismatch report and guidance (`/git-branch` or `/fab-switch`) — no autonomous checkout. (Theme 2 offered "or a fab-fff Step-4 precondition STOP" as the alternative; the backlog entry selects the explicit-arg + guard option.)
- Files: `git-pr.md`, `git-pr-review.md`, `fab-fff.md` (+ SPEC mirrors). Coordinate with g8st's merged Step 0 unified-resolution block — extend it, don't duplicate it.

### 2. Replace the unexecutable `/fab-clarify intake` recovery (5 sites)

Sites at HEAD: `_pipeline.md:108`, `fab-continue.md:24`, `:163`, `:191`, `:208`.
Replacement design: route recovery through an executable sequence — `/fab-continue intake` (Reset Flow: reset to intake, regenerate, advance to ready) followed by `/fab-clarify` (no argument — operates on the active change's intake); where plan regeneration is the actual intent, state explicitly that `plan.md` must be deleted before re-running `/fab-continue` (the documented force-regeneration mechanism, `fab-continue.md:196`). This is the only executable route: clarify is stage-guard-blocked post-intake, so "deepen the intake" necessarily means reset-to-intake first. Exact per-site wording decided at apply — sites differ (prose recovery vs error-message pointer).

### 3. `fab-operator` autopilot dispatch fixes

- **Double-dispatch** (#393 f049 regression): §6 spawn sequence step 5 (HEAD :391) embeds `'<command>'` in the `tmux new-window` invocation, while the autopilot per-change loop (HEAD :503-506, items 1–4) re-runs §6 steps then separately Gates and Dispatches `/fab-fff` — the command fires twice. Fix to a single dispatch point: either the spawn embeds the command (and autopilot's Dispatch item is dropped/reworded as "command already sent at spawn"), or the spawn opens a bare tab and Dispatch sends — pick one and make §6 + autopilot consistent.
- **Unparseable chain** (HEAD :475): entry-form table's Existing-change initial command is `/fab-switch <change> && /fab-proceed` — `&&` has no slash-command chaining semantics. Replace with the single parseable `/fab-fff <change>` (fab-fff takes the change-name override; no switch needed).
- **Queue-chaining contradiction** (HEAD :483 rule, :480 worked example, :511 autopilot item 7): the rule says strict queue-previous (`depends_on: [<prev-change-id>]` for every change after the first) but the worked example (`ab12 → cd34 → ef56`, cd34 cross-repo) says ef56 "cherry-picks from its same-repo predecessor" — strict queue-previous would give ef56 `depends_on: [cd34]` (cross-repo → ordering-only, NO code), losing the stack. Resolution: **nearest-same-repo-predecessor** (rule text updated to match the worked example) — the two-tier dependency design itself dictates this: cross-repo predecessors contribute no code, so strict queue-previous silently breaks same-repo stacking. Align rule + examples + autopilot item 7.
- Files: `fab-operator.md` (+ SPEC-fab-operator.md).

### 4. `fab-continue`: review-pr/failed dispatch row + reset idempotency + pointer fixes

- **Add the missing `review-pr`/`failed` dispatch row** to Step 1 (HEAD :42-58): keyed off `progress.review-pr == failed` (like the existing `review`/`failed` guard at :42/:55 — preflight's derived stage/state never yields a failed tier). Route: re-execute `/git-pr-review` behavior — its Step 0 already runs `fab status start <change> review-pr` and `status.go` stageTransitions allow `start: failed → active` for review-pr. Do NOT route through `reset` (reset From = `{done, ready, skipped}` — excludes `failed`, would error at the CLI).
- **Reset Flow idempotency** (HEAD :189-200): when the reset target stage is already `active`, skip the `fab status reset` call (it would error — `active` is not in reset From) and proceed directly to the execute step. Re-running a reset is then a no-op state-wise (Constitution III).
- **Reset From-set doc** (HEAD :85): `done/ready → active` → align to Go: `done/ready/skipped → active` (k4ge already shipped `skipped` in the From-set).
- **intake.md-missing pointer loop** (HEAD :204): "No intake.md found. Run /fab-continue to generate the intake first." — plain `/fab-continue` re-enters apply and hits the same error. Point to `/fab-continue intake` (the reset target whose flow regenerates the intake).
- Files: `fab-continue.md` (+ SPEC-fab-continue.md).

### 5. State-enumeration alignment to `status.go` AllowedStates

Ground truth (`src/go/fab/internal/status/status.go:18-28`): ValidStates = `{pending, active, ready, done, failed, skipped}`; AllowedStates per stage — `ship: {pending, active, done, skipped}`, `review-pr: {pending, active, done, failed, skipped}` (neither allows `ready`); `intake: {active, ready, done}`.

- `fab-continue.md:57-58` — ship and review-pr dispatch rows cite `active`/`ready`; `ready` is unreachable for both stages — drop it.
- `fab-switch.md:95` — `{state}` qualifier enumerates only `done`/`active`/`pending`; add `ready` (the standard state of every freshly switched draft) and `skipped` (and `failed` if the display_state contract surfaces it — verify against preflight's display_state derivation at apply).
- `fab-status.md:46` — progress-table legend has `✓ done, ● active, ◷ ready, ○ pending, ✗ failed` but no `skipped` glyph — add one (glyph choice at apply; check whether any Go renderer or other skill already prints one to stay consistent).
- Files: `fab-continue.md`, `fab-switch.md`, `fab-status.md` (+ SPEC mirrors).

### 6. Define the underspecified dispatch inputs

- `fab-proceed.md` fab-new dispatch (HEAD :133-139): no defined behavior when SRAD yields Unresolved (Critical Rule says MUST ask, but the subagent context is promptless and no `[AUTO-MODE]` prefix is sent). Define the contract as **defer-and-surface**: the dispatch prompt instructs the subagent to ask no questions — would-be-asked Unresolved decisions are recorded as `Deferred — promptless dispatch` rows in the intake's Assumptions table and returned in the subagent result; fab-proceed surfaces them to the user before delegating to `/fab-fff` (whose intake gate catches the resulting low confidence — an Unresolved row scores 0.0, so the gate is the structural backstop). Alternatives (interactive relay, adopting `[AUTO-MODE]`) add protocol surface the audit didn't ask for.
- `_review.md:53` — the parsimony-pass skip condition keys on `change_type`, but the inward sub-agent's prompt contract never supplies it. Define the input: the dispatching skill reads `change_type` from the change's `.status.yaml` and passes it in the sub-agent prompt (update `_review.md`'s prompt contract and the dispatching call sites).
- `fab-new.md:50` — the backlog-ID collision pre-check `fab change resolve {id}` is substring-based: a 4-char ID matching inside another change's slug false-positives and silently routes to resume (skipping creation). Anchor on the ID token: resolve then compare the canonical 4-char ID (`fab resolve --id`) for equality with `{id}` — only an exact ID match routes to resume.
- Files: `fab-proceed.md`, `_review.md`, `fab-new.md` (+ SPEC mirrors).

### 7. Stale chains, false claims, and competing rework routes

- `fab-proceed.md:88-92` — dispatch-table rows chain `/git-branch` after `/fab-new`: a stale no-op since #322 (fab-new Step 11 creates/renames the branch inline). Drop `/git-branch` from the `/fab-new`-prefixed rows only; `/fab-switch`-prefixed rows keep it (switching activates a change but creates no branch). Update the git-branch Dispatch section's "runs when" claim (:149) to match.
- `_cli-external.md:59-63` — "New change (from backlog)" worktree flow step 3 claims the operator sends `/git-branch` after the intake advances; fab-new Step 11 already renamed/created the branch inline. Rewrite the claim to match the inline behavior.
- `fab-ff.md:32` / `fab-fff.md:32` — `{driver}` row claims it is "passed to every `fab status` event command", contradicting `_pipeline.md`'s deliberately driver-less fail/recovery commands (history-shape divergence is intentional). Reword to scope the claim to the commands that actually take the driver.
- `_pipeline.md:87-92` — the decision heuristics route "requirements mismatches" to **Fix code** (item 1) AND "requirements mismatch" to **Revise requirements** (item 3). Make the bullets disjoint: code-fails-a-correct-requirement → Fix code; the-requirement-itself-is-wrong/drifted → Revise requirements — rewrite both bullets so each failure description appears exactly once.
- Files: `fab-proceed.md`, `_cli-external.md`, `fab-ff.md`, `fab-fff.md`, `_pipeline.md` (+ SPEC mirrors).

## Affected Memory

- `pipeline/execution-skills`: (modify) git-pr/git-pr-review explicit change argument + branch-matches-change guard; fab-continue review-pr/failed dispatch row, idempotent reset, aligned pointers; _pipeline recovery route + disjoint rework heuristics
- `pipeline/planning-skills`: (modify) fab-new ID-anchored collision pre-check; fab-proceed defer-and-surface fab-new dispatch contract + dropped /git-branch chain
- `pipeline/change-lifecycle`: (modify) /fab-switch state-qualifier enumeration + /fab-status skipped glyph
- `runtime/operator`: (modify) autopilot single-dispatch, single parseable entry-form command, reconciled queue-chaining semantics

## Impact

- **Skill sources** (13): `src/kit/skills/git-pr.md`, `git-pr-review.md`, `fab-fff.md`, `fab-ff.md`, `_pipeline.md`, `fab-continue.md`, `fab-operator.md`, `fab-proceed.md`, `_review.md`, `fab-new.md`, `fab-switch.md`, `fab-status.md`, `_cli-external.md`
- **SPEC mirrors**: the corresponding `docs/specs/skills/SPEC-*.md` per touched skill (constitution constraint)
- **No Go changes**: gate exit codes, AllowedStates enforcement, reset From-set, and the iterations-preserving reset cascade all landed in k4ge (#395); the g8st (#401) git-state guards are the baseline these edits extend
- **Markdown-only** — no migrations, no schema changes; `fab sync` redeploys
- **Collision care**: g8st rewrote git-pr.md Step 0 / git-pr-review.md failure paths / fab-operator.md §6 — all edits in those areas extend the merged g8st text (verified anchors above are post-g8st)

## Open Questions

- None — the backlog entry and the findings report fully specify the work; the remaining design choices are graded Tentative below (reversible markdown decisions, resolvable via `/fab-clarify` or at apply).

## Assumptions

<!-- Decisions stated verbatim in the backlog entry (explicit change argument + branch guard,
     review-pr/failed → re-execute /git-pr-review, SPEC mirror per touched skill, /fab-fff <change>
     as the operator entry-form command) are requirements, not assumptions — they live in
     What Changes and are not graded here. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is Themes 2+4 skill text only; no Go changes — gate exit, AllowedStates enforcement, reset From `skipped`, and the metrics cascade already landed in k4ge | Verified at HEAD: `status.go:41` reset From `{done,ready,skipped}`; k4ge/g8st commits in branch history | S:90 R:90 A:95 D:90 |
| 2 | Certain | Report line numbers (vs 1431a9c3) are superseded; the re-verified HEAD anchors recorded in What Changes are the working references for apply | All 9 action clusters re-located and confirmed live post-#394–#401 during intake | S:85 R:95 A:95 D:90 |
| 3 | Certain | fab-new collision pre-check anchored by exact 4-char ID equality (`fab resolve --id` compare), not substring resolution | Backlog instructs "anchor on the [id] token"; `fab resolve --id` is the one documented exact-ID query — only the CLI form was left to choose | S:85 R:85 A:85 D:85 |
| 4 | Confident | _review.md change_type input: dispatching skill reads `.status.yaml` `change_type` and passes it in the sub-agent prompt contract | One obvious mechanism; matches existing prompt-contract style | S:60 R:80 A:80 D:80 |
| 5 | Confident | /git-branch dropped only from /fab-new-prefixed fab-proceed rows; /fab-switch rows keep it (switch activates but creates no branch) | Backlog gives the drop; the keep-for-switch scoping follows from #322 covering only fab-new's inline creation | S:75 R:85 A:85 D:80 |
| 6 | Confident | {driver} reword + disjoint rework heuristics are doc-truth fixes; _pipeline's driver-less fail/recovery commands stay as designed | Report classifies the divergence "history-shape" (deliberate); fix the claim, not the behavior | S:70 R:85 A:80 D:80 |
| 7 | Confident | `/fab-clarify intake` replacement: `/fab-continue intake` then argless `/fab-clarify`, with explicit plan.md-deletion note where plan regeneration is the intent; per-site wording at apply | Constraint-determined: clarify is stage-guard-blocked post-intake, so reset-to-intake-first is the only executable route to "deepen the intake" | S:55 R:80 A:75 D:65 |
| 8 | Confident | Queue-chaining reconciled to nearest-same-repo-predecessor (rule text updated; worked example kept) | Backlog delegates the pick; the two-tier dependency design admits only this semantics without silently breaking same-repo stacking (cross-repo deps carry no code) — the example already encodes it | S:60 R:70 A:65 D:60 |
| 9 | Confident | fab-proceed promptless fab-new dispatch: defer-and-surface contract (no questions; Unresolved recorded as `Deferred — promptless dispatch`, surfaced before /fab-fff) | Keeps fab-proceed zero-prompt; an Unresolved row scores 0.0 so the existing intake gate is the structural backstop — interactive relay or [AUTO-MODE] adoption adds protocol surface the audit didn't ask for | S:50 R:75 A:65 D:55 |

9 assumptions (3 certain, 6 confident, 0 tentative, 0 unresolved).
