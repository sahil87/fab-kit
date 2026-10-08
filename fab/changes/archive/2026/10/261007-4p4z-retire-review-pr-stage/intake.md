# Intake: Retire `review-pr` as a pipeline stage — ship becomes terminal, `/git-pr-review` becomes a manual skill

**Change**: 261007-4p4z-retire-review-pr-stage
**Created**: 2026-10-08

## Origin

Synthesized from a design conversation about pipeline latency, immediately following change `261007-gyp9-early-draft-pr-live-meta` (PR #688, merged to `main` as `57ec1776`). That change opened the draft PR at apply exit, made `## Meta` marker-delimited and refreshable, and added `fab pr-sync`. Its intake and plan both name **this** change as the explicit follow-on and forbade pre-empting it:

> The `review-pr` stage, `/git-pr-review`, and the Copilot request/poll machinery are **explicitly out of scope** — a separate follow-on change. *(gyp9 intake, Origin)*
>
> The `review-pr` teardown is a deliberate non-goal here — a follow-on change owns it. Do not touch `/git-pr-review` or the Copilot machinery. *(gyp9 plan, Non-Goals)*

Interaction mode: conversational design session, then a promptless dispatch (`{questioning-mode} = promptless-defer`) — no interactive questions were asked during intake.

The user's decision, in their words:

> fab should not decide when a review gets requested or how long to wait, because that "depends too much on the user's flow matching what we think."

Decisions reached in that conversation and carried verbatim into this intake:

- **`ship` becomes the pipeline's terminal stage.** `fab status finish <change> ship` no longer auto-activates `review-pr`.
- **`review-pr` STAYS in the state machine** — in `AllowedStates`, `stageTransitions`, and `statusfile.StageOrder`. It simply sits `pending` forever unless a human invokes `/git-pr-review`.
- **`/git-pr-review` becomes manual triage-only.** It NEVER requests a review. No "manual request" escape hatch, no "fire and forget at ship" variant — both were explicitly rejected.
- **The 42% review-feedback hit rate being traded away is an accepted cost**, not an open question and not a risk to relitigate. The user was shown the number and decided anyway.

## Why

### The problem

The `review-pr` stage waits synchronously for a GitHub Copilot review that fab itself requests. Measured over this repo's own history (`.history.jsonl`, n = 308 changes, 233 completed `review-pr` cycles) during the design conversation:

| Metric | Value |
|--------|-------|
| `review-pr` stage duration, median | **8.5 min** |
| `review-pr` stage duration, p90 | **16.4 min** |
| Fraction of that spent idle-polling | **~90%** |
| Copilot review latency after request | ~4.5–6.5 min |

Roughly 90% of the stage is `git-pr-review.md` Step 2 Phase 2's 30s × 20 poll loop doing nothing but waiting. The stage's entire design — the synchronous-poll discipline note, the two-invocation timeout budget, the `review-gate-unavailable` reason, the operator's `no-reviewer` recovery row — is machinery built to manage that wait.

### The compounding cost on dependency chains (the user's stated primary motive)

`src/kit/skills/fab-operator.md` § Dependency satisfied (around line 531) satisfies a `depends_on` entry only when the dependency **item is `done`**, and a pane item with a null `stop_stage` is `done` only when its built-in predicate fires — `review-pr` done/skipped (`src/go/fab/cmd/fab/operator_tick_start.go`'s `tickTerminusStage = "review-pr"` + `tickCompleted`). So in a dependency chain of depth N, **every link pays the full Copilot poll before the next one spawns**. That is a flat ~40% added to the critical path of every multi-change plan, and it compounds linearly with chain depth.

### What happens if we don't fix it

Every change keeps paying a median 8.5 minutes of idle polling, and every multi-change plan keeps multiplying it by its chain depth. The machinery built to survive that wait keeps growing — the timeout budget, the fail-closed unwritable-history path, and the operator's `no-reviewer` re-send were all added to stop a *waiting* stage from stalling the pipeline. Deleting the wait deletes its whole support structure.

### Why this approach over alternatives

- **Make the review optional and manually triggered** (chosen): fab stops deciding *when* a review is requested and *how long* to wait. `/git-pr-review` keeps its full triage/fix/reply value and runs whenever the user actually wants it.
- **Keep the request, drop the wait ("fire and forget at ship")** (rejected by the user explicitly): still bakes fab's opinion about when a review should be requested into the pipeline, and leaves a review landing on a PR nobody is watching.
- **Shorten the poll window** (not chosen): Copilot lands at 4.5–6.5 min, so a shorter window mostly converts waits into timeouts — it trades idle time for the timeout budget's complexity, which is the thing we want to delete.
- **Remove `review-pr` from the state machine entirely** (rejected): would require `AllowedStates`/`stageTransitions`/`StageOrder` surgery, a `.status.yaml` migration, and would break every in-flight change mid-pipeline. Keeping the stage inert costs nothing.

### Accepted tradeoff — recorded, not re-opened

Measured during the same conversation: **215 of 509 merged PRs (42%)** carried a `fix: address review feedback` commit. Making the review optional means in practice it will rarely run, so that is the hit rate being traded away. **The user was shown this number and decided anyway**, for the reason in this section. This is an accepted cost. Do not re-surface it as an open question, a risk, or a reason to design a partial automatic request back in.

## What Changes

### 1. `ship` becomes the pipeline's terminal stage (Go)

`Finish` in `src/go/fab/internal/status/status.go` (the auto-activate block around line 180) currently reads:

```go
	// Auto-activate next pending stage
	nextStage := sf.NextStage(stage)
	if nextStage != "" {
		nextState := statusFile.GetProgress(nextStage)
		if nextState == "pending" {
			if err := statusFile.SetProgress(nextStage, "active"); err != nil {
				return err
			}
			applyMetricsSideEffect(statusFile, fabRoot, nextStage, "active", driver, "", "")
		}
	}
```

Guard it so that finishing `ship` does **not** auto-activate `review-pr`. `review-pr` then sits `pending` indefinitely unless invoked manually.

Note the deliberate asymmetry: `sf.NextStage("ship")` still returns `"review-pr"` (used by ordering, metrics, and display). Only the *auto-activation* is suppressed. Implement this as a terminal-stage notion rather than a bare `stage == "ship"` string compare wherever that is natural — see change area 7, which owns the shared decision.

Everything else in the state machine is untouched:

- `statusfile.StageOrder` keeps all six entries.
- `review-pr`'s `AllowedStates` (`{pending, active, done, failed, skipped}`) and its `stageTransitions` rows are unchanged.
- `src/kit/templates/status.yaml` keeps `review-pr: pending`.
- **No `.status.yaml` migration is required**, and no in-flight change breaks: a change already sitting at `review-pr: active` stays exactly where it is and `/git-pr-review` still finishes it.

`/git-pr-review` Step 0's existing `fab status start <change> review-pr git-pr-review` already handles `pending → active`, so a manual invocation works with **zero edits to that part of the skill**.

**Go tests**: `src/go/fab/internal/status/status_test.go`, `transitions_test.go`, `mutators_test.go`, and `src/go/fab/cmd/fab/refresh_selfheal_test.go` all encode `ship finish → review-pr active`. These must be updated. Run them from the module root: `cd src/go/fab && go test ./...` (the repo has two Go modules, `src/go/fab` and `src/go/fab-kit`; `go test ./src/go/...` from the repo root fails).

### 2. Delete the Copilot request and poll machinery from `src/kit/skills/git-pr-review.md`

The file is 335 lines today. Remove, in full:

| What | Where (current line refs) |
|------|---------------------------|
| **Step 2 Phase 2 entirely** — the `code-review.md` § Review Tools config check, the two-login request/poll table, the GraphQL trap note, the `gh pr edit --add-reviewer copilot-pull-request-reviewer` request, and the 30s × 20 (10-minute) poll | lines 95–167 |
| **The timeout budget** — the `fab log command … "timeout pr={number}"` marker, the `awk` history-marker counting over `.history.jsonl`, and the `timeouts = 0 / 1 / ≥2` branching including the fail-closed unwritable-`.history.jsonl` path | lines 127–163 |
| **The `review-gate-unavailable` reason** — every mention, including the `fab log command … "review-gate-unavailable pr={number}"` records | lines 146–163, 274, 279 |
| **The `timeout` outcome class** — Step 6's outcome table drops from 4 rows to 3 | line 275 |
| **The synchronous-poll discipline note** | line 114 |
| **The `--tool` flag** — the frontmatter/header mention, the argument-classification sentence, the flag description, and Step 1.5's parsing/validation | lines 7, 11, 13, 63–69 |
| **Idempotency paragraph's budget narrative** — rewritten; the re-run story is now simply "no reviews → report and exit" | line 281 |

After the deletion, Step 2 becomes **Phase 1 only**:

- Reviews exist **with inline comments** → Step 3 (fetch/triage/fix/reply) — **unchanged from today**.
- Reviews exist but **no inline comments** → today's message, outcome `no-reviews`.
- **No reviews at all** → print `No reviews on PR #{number}.` and go to Step 6 with outcome **`no-reviews`**. It NEVER requests one.

Step 6's outcome table becomes three classes:

| Outcome | Includes | Stage action | Commit status in Step 6.5? |
|---------|----------|--------------|----------------------------|
| **success** | Comments processed/pushed; no actionable comments | `fab status finish <change> review-pr git-pr-review 2>/dev/null \|\| true` | Yes |
| **failure** | `gh` missing; no PR; processing error | `fab status fail <change> review-pr git-pr-review 2>/dev/null \|\| true` | No |
| **no-reviews** | No reviews on the PR; reviews with no actionable inline comments | `fab status finish <change> review-pr git-pr-review 2>/dev/null \|\| true` (successful no-op) | Yes |

Everything from Step 3 onward — fetch, triage, the `fix`/`defer`/`skip`/informational dispositions, the Step 5 commit/push with its unpushed-commit re-run gate and FKF §5 generated-index rule, Step 5.5's reply dedup, Step 6.5's status commit, Phase Sub-State Tracking, the Rules, and the Disposition Reference — is **untouched**.

The frontmatter `description` must be rewritten (it currently advertises the request/poll/timeout behavior verbatim), as must the `# /git-pr-review` header line and its one-paragraph summary. The `allowed-tools` line stays as-is.

### 3. Remove the orchestrator wiring

**`src/kit/skills/fab-fff.md`**:

- **Delete Step 5 (Review-PR) entirely** — both dispatch branches (lines ~73–88), the synchronous-poll directive it mirrors (line ~80), and the `/git-pr-review` result-handling text.
- `{terminal}` becomes **`ship`** (line 42: `| `{terminal}` | `review-pr` — after the bracket's Step 3 (hydrate), continue with Steps 3.5–5 below |` → `ship`, Steps 3.5–4).
- Line 25's "then continue through ship and review-pr" → "then continue through ship".
- Line 46's "Steps 1–5 … Steps 4–5 dispatch full `/git-pr` and `/git-pr-review`" → Steps 1–4 / Step 4 dispatches `/git-pr`.
- Line 69: ``On success: `progress.ship` becomes `done`, `progress.review-pr` auto-activates.`` → ship `done` is the pipeline's terminal state.
- Line 104: the Review-PR output section and the Step 5 timeout branch.
- **Error Handling table**: delete both review-pr rows (lines 117–118) and both review-pr result rows (lines 125–127).
- The skill's frontmatter `description` ("Full pipeline — implementation, sub-agent review, hydrate, ship, and PR review") drops "and PR review".

**`src/kit/skills/_pipeline.md`**:

- Line 3 `description` — the bracket description's terminal-stage framing.
- Line 17 "Driver-specific ship/review-pr steps" → ship steps.
- Line 45 — ``{terminal}` to `hydrate` (`fab-ff`) or `review-pr` (`fab-fff`)` → `ship` (`fab-fff`).
- Line 133 — the `{terminal}: done` resumability check (mechanically correct once `{terminal}` is `ship`, but verify the wording).
- Line 161 (**Light Lane**) — "for `{terminal} = review-pr` only — ship and review-pr (the driver's Steps 4–5)" → `{terminal} = ship`, ship only (Step 4); and delete the trailing sentence about inline review-pr removing the yield-seam hazard, since the hazard no longer exists.
- Line 224 — "When `{terminal}` is `review-pr`, continue with the driver's own Steps 3.5–5" → `ship`, Steps 3.5–4.

**`src/kit/skills/fab-ff.md`** (sibling sweep — `fab-ff` ↔ `fab-fff` is a declared twin class in `fab/project/code-quality.md` § Sibling Sweeps): line 32's `{terminal}` row names `review-pr` only in contrast ("there are no ship/review-pr steps"); reword to "no ship step". `fab-ff` otherwise keeps `{terminal} = hydrate` and is unaffected.

**`src/kit/skills/fab-operator.md` § Review-PR Recovery** (around lines 768–791):

- **Delete the `no-reviewer` row** (line 772) — it exists entirely to unstick a first-timeout stall that can no longer happen — together with its escalation-message line (line 791's `no-reviewer` entry) and its share of the § Recovery spawn text (line 785 names `/git-pr-review <change>` as the `no-reviewer` invocation).
- **KEEP the `ci-failed` and `conflicting` rows.** They stay valid and become *more* useful now that CI runs from apply onward (post-#688). Line 768's detection-signal sentence must drop "`/git-pr-review`'s `timeout` outcome" from its list while keeping the `gh pr checks` and `mergeable == CONFLICTING` probes.
- Line 783's "the same class as `/git-pr-review`'s own fix commits" carve-out survives — `/git-pr-review` still exists and still makes those commits.
- The section heading and its opening framing ("A `review-pr` stage that cannot finish") need rewording — the surviving conditions are about a **shipped PR** that cannot merge, not a stage that cannot finish.

**`src/kit/skills/_preamble.md`**:

- § Dispatch-Prompt Obligations — the review-pr result schema's `outcome` comment loses `timeout` (`success | failure | no-reviews`), and the paragraph after the three YAML blocks loses its whole "review-pr `status` vs `outcome` split … first timeout maps to the orchestrator's leave-`active` + pending-message path; the second consecutive timeout exits as `no-reviews` (reason `review-gate-unavailable`)" passage.
- § Self-managing stages (ship/review-pr) — still true (`/git-pr-review` self-manages when run manually or via `/fab-continue`), but the `/fab-fff` Steps 4–5 reference becomes Step 4.
- § Next Steps Convention **State Table** — the `ship` row's default changes from `/git-pr-review` to `/fab-archive`, with `/git-pr-review` listed as an available (optional, manual) command. The `review-pr (pass)` / `review-pr (fail)` rows **stay** — the stage is still reachable manually.

**`src/kit/skills/fab-continue.md`**:

- The `ship`/`active` row (line 85) — strip the trailing "(auto-activates review-pr)".
- The `review-pr`/`active` row (line 86) — delete the "Timeout (Copilot review requested…)" sentence and its `review-gate-unavailable` clause. **Keep the row itself**: `/fab-continue` on an `active` review-pr stage is exactly the manual path that must keep working.
- The `review-pr`/`failed` row (line 87) is unchanged.
- Line 61 / 75 / 90's ship/review-pr delegation prose stays correct.

**`src/kit/skills/git-pr.md`** line 372 — "This marks `ship` as `done` and auto-activates `review-pr`" is now false; it marks `ship` done and the pipeline is complete.

**`src/kit/skills/fab-adopt.md`** *(not named in the originating brief — added here as a mechanical consequence)*:

- Frontmatter `description` (line 3) "run review/hydrate/ship/review-pr for real".
- Line 28's six-stage narrative ("ship refreshing the PR's Meta block, review-pr resuming normally").
- Line 117's "which auto-activates review-pr".
- **Step 6 — "Land in review-pr"** (lines 119–133): an adopted change now lands at `ship: done`, not `review-pr: active`.
- Line 162's closing `Next: /git-pr-review`.
- Line 189's "runs review → hydrate → ship → review-pr via existing transitions".
- Lines 94–99's `skip`/`reset` cascade comments mention `review-pr` as a cascaded stage — those remain **factually correct** (the cascade still covers it, since `StageOrder` is unchanged) and should NOT be rewritten.

**`src/kit/skills/_srad.md`** lines 65 and 67 — the `fab-fff` column's "extends through ship + review-pr" and "apply/review/hydrate/ship/review-pr output" → ship.

**`src/kit/scaffold/fab/project/code-review.md`** line 61 — the § Review Tools `copilot` entry ("the /git-pr-review Phase 2 Copilot request. Set to false to skip") has no consumer once Phase 2 is deleted. Remove the entry; if `copilot` is the section's only entry, remove the § Review Tools section with it. (fab-kit's own `fab/project/code-review.md` has no § Review Tools section, so nothing to sweep there.)

### 4. Drop `review-pr` from the rendered Pipeline line (Go)

`src/go/fab/internal/prmeta/prmeta.go` line ~32:

```go
// pipelineStages is the fixed pipeline order rendered in the **Pipeline** line.
var pipelineStages = []string{"intake", "apply", "review", "hydrate", "ship", "review-pr"}
```

Remove `"review-pr"`. Otherwise every PR body renders a trailing unticked stage that reads as perpetually unfinished — which is exactly the opposite of what the Meta block is for now that #688 made it live and refreshable on an early PR.

`src/go/fab/internal/prmeta/prmeta_test.go` holds **byte-stable render goldens** for the Meta block. They must be updated in the same change.

### 5. Documentation and memory sweep

`review-pr` / `git-pr-review` appears across roughly **115 files** (`grep -rln 'review-pr\|git-pr-review' src/ docs/` → ~59 under `src/`, ~56 under `docs/`). The plan MUST survey honestly and split them into two classes:

**Rewrite** — files describing `review-pr` as *current behavior*:

- `src/kit/skills/*.md` — the eleven files enumerated in change area 3, plus the mention-only files that remain correct (`_cli-agents.md`, `fab-switch.md`, `fab-discuss.md`, `fab-incognito.md`, `fab-clarify.md`, `internal-skill-optimize.md`) which should be **verified, not reflexively edited**.
- `src/kit/skills/_cli-fab.md` — § fab status's `finish` side-effect chain (line 147: `hydrate→ship`, `ship→review-pr`) is now wrong and MUST be corrected; the `advance`/`fail`/`all-stages`/target-state-validation rows naming `review-pr` stay correct. **Constitution Additional Constraints + `fab/project/code-review.md` § Project-Specific Review Rules both require the owning CLI reference partial to be updated for any Go command-signature or documented-behavior change, with Go tests in the same change.**
- `src/kit/skills/_cli-fab-operator.md` — lines 139 and 204 restate the pane built-in completion predicate (`review-pr` done/skipped) verbatim; they move in lockstep with change area 7. Line 109's example YAML and line 283's accepted-selector list stay correct.
- `src/kit/skills/_cli-fab-pane.md` line 59 — "`stage: "review-pr"` plus `display_state: "done"` distinguishes a parked shipped change" becomes `ship`.
- `src/go/fab/cmd/fab/fab_help.go` line 182 and `src/go/fab/cmd/fab/skill.md` lines 4 and 79 — the "six-stage pipeline" help strings and the `gh` dependency note.
- `src/go/fab/internal/setupcheck/probes.go` line 43 — `{"gh", Warn, "needed by the ship/review-pr stages"}`.
- `src/go/fab/internal/configref/configref.go` line 626 — the `agent.workers` description enumerating worker-role stages.
- `docs/specs/` — `overview.md`, `architecture.md`, `skills.md`, `glossary.md`, `user-flow.md`, `stage-models.md`, `harness-adapters.md`, `templates.md`, `operator.md`, `assembly-line.md`, `srad.md`, `hooks.md`, `fkf.md`, `superpowers-comparison.md`. **`fab/project/code-quality.md` § Sibling Sweeps declares the aggregate specs (`skills.md`, `glossary.md`, `architecture.md`, plus `overview.md`, `user-flow.md`, `stage-models.md`, `harness-adapters.md`) a sweep class — sweep the whole class up front, not reactively after review flags it.**
- `docs/site/` — `workflows.md`, `skill.md`, `install.md`, `merge-topologies.md`.
- `docs/memory/` — see Affected Memory below.
- `docs/img/stage-coverage.svg` — a generated-looking asset that renders the stage list; check whether it is hand-maintained before editing.

**Do NOT rewrite** — files that merely *mention* the stage historically:

- `src/kit/migrations/*.md` (6 files: `1.1.0-to-1.2.0`, `1.8.0-to-1.9.0`, `1.9.7-to-1.10.0`, `2.2.0-to-2.3.0`, `2.12.1-to-2.13.0`, `2.16.19-to-2.17.0`) — a migration is a dated historical instruction, never retro-edited.
- `src/kit/reference/fkf.md` and `docs/site/fkf.md` — the FKF standard is synced verbatim by `scripts/sync-fkf.sh` with a CI drift-guard test; touch only if the mention is a live behavioral claim, and if so change `docs/site/fkf.md` and re-sync, never the kit copy directly.
- `docs/findings/*` and `docs/specs/findings/*` — dated analyses.
- `docs/specs/srad-v1.md` — explicitly superseded.
- `docs/memory/**/log.md` and `log.seed.md` — generated/append-only history (FKF §5: never hand-merge or retro-edit a generated index/log).
- `docs/wiki/operator-tick-anatomy.html` — a dated wiki snapshot; verify before touching.
- `fab/changes/**` and `fab/changes/archive/**` — completed change artifacts.
- `src/go/**/*_test.go` fixtures that assert *historical* status files — distinguish these from the tests that assert current behavior (change areas 1, 4, 7), which MUST be updated.

### 6. Backlog reconciliation

- **`[vb4l]`** (`fab-operator`: review-pr recovery playbook when the review gate is degraded/unavailable — generalize beyond Copilot) becomes **obsolete**: there is no review gate for fab to find degraded. Mark it accordingly at archive time rather than leaving it to rot.
- **`[hf6x]`** (on a review-pr-stage CI failure, ask the authoring agent to diagnose/fix before escalating) **stays open and stays valid** — it is the `ci-failed` recovery row, which this change deliberately keeps.
- **New follow-up**: the bare `git push --force-with-lease` at `src/kit/skills/fab-operator.md:733` and `:774` (autopilot rebase arms). After a `git fetch`, the bare flag takes the just-fetched tip as its lease, so it accepts a stranger's push and overwrites it — the explicit pre-fetch lease-OID capture that `git-pr.md` Step 3a-ter uses (`lease_oid=$(git rev-parse -q --verify '@{u}')` **before** the fetch) is the correct form. This is a **real latent bug but PRE-EXISTING and OUT OF SCOPE here**. File it as a backlog item; do not fix it in this change.

### 7. The operator's completion predicate stops hardcoding `review-pr` (option (b) — user-decided)

The operator's built-in completion predicate for a pane item with a null `stop_stage` currently fires on `review-pr` done/skipped. **The user decided on 2026-10-08, asked directly with the tradeoff stated, to take the minimal option (b): swap the hardcoded stage name from `review-pr` to `ship`.** Option (a) — a `terminal_stage` field in `.status.yaml` with a derived `done` predicate surfaced as `fab status is-done` / a `done:` key — was presented and **rejected**. Do not build it.

The tradeoff the user accepted, recorded so a later reader does not mistake it for an oversight: **(b) reproduces the same hardcoded-stage-name coupling one stage over, so a future pipeline reshape would again be operator surgery; the gain is a materially smaller change with no schema migration.**

What this concretely means for the plan:

- **NO** new `terminal_stage` field in `.status.yaml`.
- **NO** derived `done` predicate, **no** `fab status is-done`, **no** `done:` key in `fab status --json`.
- **NO** `src/kit/migrations/` file for this change — nothing restructures user data any more.
- **`stop_stage` stays exactly as it is today.** It does NOT fold into anything; its semantics (past the stop in stage order, or at the stop with display_state done/skipped) are unchanged.

Sites to change — survey for all of them, these are the ones found so far:

| File | Site |
|------|------|
| `src/go/fab/cmd/fab/operator_tick_start.go` | line 166 — `const tickTerminusStage = "review-pr"` → `"ship"`. Its doc comment (lines 160–165) explains the terminus with a "a change entering hydrate or ship under /fab-fff is mid-pipeline, not complete" example that must be reworded. Line 459's `tickCompleted` doc comment ("stop_stage null: AT the terminus (review-pr)") moves with it |
| `src/kit/skills/fab-operator.md` | line ~281 — the `pane` kind row's built-in predicate (`review-pr` done/skipped, or at/past `scope.stop_stage`) |
| `src/kit/skills/fab-operator.md` | line ~531 — § Dependency satisfied: "its built-in predicate fired (`review-pr` done/skipped when its `stop_stage` is null …)" |
| `src/kit/skills/fab-operator.md` | line ~812 — the spawned-pane-completes line: "its `done` delta — at/past the item's `stop_stage`, or `review-pr` done/skipped when it is null" |
| `src/kit/skills/_cli-fab-operator.md` | line ~139 — § Pane built-in completion's full restatement of the predicate |
| `src/kit/skills/_cli-fab-operator.md` | line ~204 — the kind table's `pane` row restatement |

`src/go/fab/cmd/fab/operator_tick_diff_test.go` asserts the terminus behavior and must be updated in the same change (`cd src/go/fab && go test ./cmd/fab/...`).

Note that change area 1's `Finish` guard shares this decision's posture: with option (a) rejected, the guard is a plain terminal-stage check against `ship` — no status-file field is read to derive it.

`review-pr` remains a valid `--stage` / `--stop-stage` value (`StageOrder` is unchanged), so an operator item can still be deliberately parked at `review-pr` via an explicit `stop_stage`.

### 8. Constitution wording

`fab/project/constitution.md` § Additional Constraints currently reads:

> The core pipeline is six stages (`intake → apply → review → hydrate → ship → review-pr`). All human judgment is frontloaded to intake (the sole confidence gate); everything after intake runs unattended unless review-rework exhausts or PR feedback arrives.

The six-stage enumeration stays **literally true** (`review-pr` remains in `StageOrder`), but the trailing clause's "unless … PR feedback arrives" now describes a path the pipeline never enters on its own. Reword the trailing clause to name `ship` as the terminal stage and `review-pr` as a manually-invoked stage.
<!-- assumed: a wording-only constitution edit with a dated governance comment and NO version bump — the six-stage enumeration and every MUST rule are unchanged, matching the udwv / jjg0 / t513 / yd9s / si4k precedents recorded in the governance comment block. A plan could instead treat this as a narrowed normative rule warranting a minor bump, or leave the sentence alone. -->

## Affected Memory

- `pipeline/execution-skills`: (modify) — the `/git-pr-review` behavior description (request/poll/timeout budget/`review-gate-unavailable`), `/fab-fff` Steps 4–5 → Step 4, the `_pipeline.md` bracket's `{terminal}`, and the Light Lane's inline ship/review-pr note. 40 current matches — the densest file in the sweep.
- `pipeline/change-lifecycle`: (modify) — the stage lifecycle and state machine; `ship finish` no longer auto-activates `review-pr`; `ship` is terminal while `review-pr` stays a reachable manual stage. 18 matches.
- `pipeline/planning-skills`: (modify) — the `/fab-ff` ↔ `/fab-fff` autonomy posture table and terminal-stage framing. 16 matches.
- `pipeline/schemas`: (modify) — the review-pr dispatch result schema's outcome classes (4 → 3) and the `.status.yaml` progress map's terminal semantics. 12 matches.
- `runtime/operator`: (modify) — the pane built-in completion predicate, § Dependency satisfied, and the Review-PR Recovery playbook (`no-reviewer` row removed; `ci-failed` + `conflicting` kept). 23 matches.
- `runtime/dispatch`: (modify) — the review-pr dispatch-prompt obligations and the dropped `timeout` outcome. 7 matches.
- `_shared/configuration`: (modify) — the `review_tools` / `copilot` knob documentation, now removed from the scaffold. 6 matches.
- `_shared/context-loading`: (modify) — the Next-Steps state table rows and the self-managing-stages text. 5 matches.
- `runtime/providers-and-profiles`: (modify) — the stage→role mapping restatement (`review-pr` → `doing`). The mapping itself is **unchanged**; verify the surrounding prose does not assert review-pr is auto-reached. 7 matches.
- `memory-docs/templates`: (modify) — verify only; `templates/status.yaml` keeps `review-pr: pending`. 2 matches.
- `pipeline/clarify`, `pipeline/preflight`, `pipeline/issue-linking`, `runtime/pane-commands`, `memory-docs/hydrate`: (modify) — single-mention each; verify, edit only if the mention asserts current auto-reached behavior.
- `pipeline/index`, `runtime/index`, `docs/memory/index.md`: (modify) — domain description lines enumerate the six-stage pipeline.

`docs/memory/**/log.md` and `log.seed.md` are generated/append-only and are **not** in this list.

## Impact

**Go** (`src/go/fab`, module root `cd src/go/fab`):

| File | Change |
|------|--------|
| `internal/status/status.go` | `Finish` auto-activation guard (change area 1) |
| `internal/prmeta/prmeta.go` | `pipelineStages` drops `review-pr` (change area 4) |
| `internal/prmeta/prmeta_test.go` | Byte-stable render goldens |
| `cmd/fab/operator_tick_start.go` | `tickTerminusStage` (change area 7, option (b)) |
| `cmd/fab/fab_help.go`, `cmd/fab/skill.md` | Help-string pipeline enumeration |
| `internal/setupcheck/probes.go` | `gh` probe description |
| `internal/configref/configref.go` | `agent.workers` description |
| `internal/status/status_test.go`, `transitions_test.go`, `mutators_test.go` | ship-finish transition assertions |
| `cmd/fab/refresh_selfheal_test.go`, `cmd/fab/operator_tick_diff_test.go`, `cmd/fab/status_json_test.go`, `internal/statusfile/golden_test.go` | Transition / terminus / golden assertions — audit each for current-behavior vs. historical-fixture |

Untouched by design: `internal/statusfile/statusfile.go` (`StageOrder`, `NextStage`, `AllowedStates`), `internal/change`, `internal/archive`, `internal/score`.

**Kit skills** (`src/kit/skills/`, canonical — Constitution V; never edit `.agents/skills/` or `.claude/skills/`): `git-pr-review.md` (largest delta), `fab-fff.md`, `_pipeline.md`, `fab-operator.md`, `_preamble.md`, `fab-continue.md`, `fab-adopt.md`, `git-pr.md`, `_srad.md`, `fab-ff.md`, `_cli-fab.md`, `_cli-fab-operator.md`, `_cli-fab-pane.md`. Mention-only / verify: `_cli-agents.md`, `fab-switch.md`, `fab-discuss.md`, `fab-incognito.md`, `fab-clarify.md`, `internal-skill-optimize.md`.

**Kit non-skill**: `src/kit/scaffold/fab/project/code-review.md`. Untouched: `src/kit/templates/status.yaml`, `src/kit/migrations/*`, `src/kit/reference/fkf.md`.

**Docs**: ~14 `docs/specs/` files, 4 `docs/site/` files, ~12 `docs/memory/` files, `docs/img/stage-coverage.svg`.

**Project files**: `fab/project/constitution.md` (change area 8), `fab/backlog.md` (change area 6).

**Constraints the plan must honor**:

1. **Constitution V** — canonical kit sources are `src/kit/`; deployed copies under `.agents/skills/` and `.claude/skills/` are never edited.
2. **Constitution V citation rule** — deployed `src/kit/**` content must not cite fab-kit-only paths (`docs/specs/*`, `docs/memory/*`, `docs/site/*`, `src/go/*`) as an authority. A Go guard test in `src/go/fab-kit/cmd/fab/` fails on this. In particular, the new `git-pr-review.md` and `fab-operator.md` prose must not point at `src/go/.../operator_tick_start.go` for the terminus rule — restate it.
3. **Constitution III (idempotency)** — re-running `/git-pr-review` on a PR with no reviews must be a clean, repeatable no-op.
4. **CLI reference + tests** — any `fab` command-signature change updates `_cli-fab.md` / `_cli-fab-pane.md` / `_cli-fab-operator.md` and ships Go test updates.
5. **Migrations** — none are expected: with option (a) rejected (change area 7), nothing in this change restructures user data. If the plan nonetheless finds a `.status.yaml`/config/archive-layout restructuring it must ship as a `src/kit/migrations/` file, never an ad-hoc script.
6. **Sibling Sweeps** — sweep `fab-ff` ↔ `fab-fff`, the aggregate specs, and the memory files documenting the skills **up front**.
7. **Two Go modules** — run tests from `src/go/fab` and `src/go/fab-kit` separately; `go test ./src/go/...` from the repo root fails.

**Explicitly NOT in scope**:

- Re-adding any automatic Copilot request anywhere, including a "fire and forget at ship" variant. Rejected by the user.
- Removing `review-pr` from `AllowedStates` / `stageTransitions` / `StageOrder`, or any `.status.yaml` migration for the stage itself.
- Option (a) from the operator-predicate decision: a `terminal_stage` field, a derived `done` predicate, `fab status is-done`, a `done:` key in `fab status --json`, or folding `stop_stage` into any of them. User-rejected 2026-10-08.
- Anything from change #688 (early PR open, `## Meta` markers, `fab pr-sync`) — shipped and settled.
- Fixing the bare `git push --force-with-lease` at `fab-operator.md:733` / `:774` — pre-existing, backlog it (change area 6).

## Open Questions

None. The one design decision that was open at the time of writing — the operator's completion predicate (change area 7) — was put to the user directly with its tradeoff stated and decided on 2026-10-08 in favour of the minimal option (b). It is recorded as a Certain assumption below and is not re-opened here. The Constitution-wording treatment (change area 8) is settled by the j6cs precedent (see assumption 12).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `review-pr` stays in `AllowedStates`, `stageTransitions`, and `statusfile.StageOrder`; only `Finish`'s auto-activation of the stage after `ship` is suppressed | User's explicit decision with its reason stated: no Go state-machine surgery, no `.status.yaml` migration, no breakage for in-flight changes. `/git-pr-review` Step 0's existing `fab status start` already covers `pending → active` | S:95 R:80 A:90 D:95 |
| 2 | Certain | `/git-pr-review` NEVER requests a review — no manual-request escape hatch, no fire-and-forget-at-ship variant | User's explicit decision, stated twice in the originating brief and repeated as an out-of-scope item. Reversible by a follow-up change if ever wanted | S:95 R:70 A:90 D:95 |
| 3 | Certain | The 42% (215/509) review-feedback hit rate being traded away is an ACCEPTED tradeoff — recorded, never re-surfaced as an open question or risk | The user was shown the measured number and decided anyway, with the reason in § Why. The brief instructs explicitly: record as accepted, not as a risk to relitigate | S:95 R:60 A:85 D:95 |
| 4 | Certain | Historical mentions are not rewritten: `src/kit/migrations/*` (6 files), `src/kit/reference/fkf.md`, `docs/findings/*`, `docs/specs/findings/*`, `docs/specs/srad-v1.md`, `docs/memory/**/log.md` + `log.seed.md`, archived change folders, and historical-fixture test data | Stated explicitly in the brief, and independently required by FKF §5 (never retro-edit a generated index/log) and by the dated-instruction nature of migrations | S:80 R:85 A:90 D:85 |
| 5 | Certain | The pre-existing bare `git push --force-with-lease` at `fab-operator.md:733` / `:774` is backlogged as a follow-up, not fixed here | Stated explicitly in the brief as out of scope. The correct form (pre-fetch lease OID) is already documented in `git-pr.md` Step 3a-ter, so the follow-up is well-specified | S:95 R:85 A:90 D:95 |
| 6 | Confident | `/fab-adopt` is in scope (its Step 6 "Land in review-pr", its `Next: /git-pr-review` closing line, its description, and three prose claims), even though the originating brief did not name it | A mechanical consequence of ship-terminal: an adopted change can no longer land at `review-pr: active`. Its `skip`/`reset` cascade comments stay correct and are deliberately left alone | S:55 R:80 A:85 D:80 |
| 7 | Confident | `_preamble.md` § Next Steps State Table: the `ship` row's default becomes `/fab-archive` with `/git-pr-review` listed as an available manual command; the `review-pr (pass)` / `(fail)` rows stay | Follows directly from the terminal-stage change plus the decision that review-pr stays manually reachable. Low-cost to revise if the preferred default is different | S:60 R:85 A:85 D:75 |
| 8 | Confident | The now-dead `copilot` entry in `src/kit/scaffold/fab/project/code-review.md` § Review Tools is removed rather than left inert, together with its `_shared/configuration` documentation | Nothing reads the knob once Step 2 Phase 2 is deleted; a config key with no consumer is a trap for the next reader. Trivially restorable | S:60 R:80 A:85 D:80 |
| 9 | Confident | `review-pr` stays a valid value for `--stage` / `--stop-stage` operator validation and for `fab agent <stage>` selection; only the terminus constant moves to `ship` | `StageOrder` is unchanged, so every name-validation surface stays correct by construction. Only `tickTerminusStage` and the four prose restatements encode the terminus | S:55 R:80 A:85 D:80 |
| 10 | Certain | Backlog `[vb4l]` (generalized review-gate-degraded recovery playbook) is marked obsolete at archive time; `[hf6x]` (CI-failure fix request) stays open | `[vb4l]` presupposes a review gate fab manages — deleted by this change. `[hf6x]` maps to the `ci-failed` recovery row, which this change deliberately keeps. Both are reversible one-line backlog edits | S:65 R:90 A:80 D:80 |
| 11 | Certain | Operator completion predicate takes option (b): swap the hardcoded `review-pr` to `ship` in `tickTerminusStage` plus the five prose restatements. No `terminal_stage` field, no derived `done` predicate, no `fab status is-done`, no `done:` JSON key, no `src/kit/migrations/` file; `stop_stage` is unchanged | User-decided 2026-10-08, asked directly with the tradeoff stated. Accepted tradeoff: (b) reproduces the same hardcoded-stage-name coupling one stage over, so a future pipeline reshape would again be operator surgery; the gain is a materially smaller change with no schema migration | S:95 R:75 A:90 D:95 |
| 12 | Certain | `fab/project/constitution.md` § Additional Constraints gets a wording-only edit to its trailing clause (naming `ship` terminal and `review-pr` manual) with a dated governance comment and NO version bump | The governing precedent is **j6cs** (260601), which merged the `spec` stage into `apply` — a 7→6 **pipeline-shape** change, strictly larger than this one — and recorded exactly this treatment: dated governance comment, "no new normative MUST-rule was added (the existing constraints already cover it)", no version bump. The udwv / jjg0 / t513 / yd9s / si4k precedents agree but are only pure-wording cases; j6cs is the on-point one. The six-stage enumeration also stays literally true — `review-pr` remains a stage in `AllowedStates`, `stageTransitions`, and `StageOrder`; only its auto-activation goes | S:85 R:80 A:85 D:85 |

12 assumptions (7 certain, 4 confident, 1 tentative, 0 unresolved).
