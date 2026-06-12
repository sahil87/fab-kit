# Intake: Skills-Review Batch 1 — Surgical Correctness + Idempotency Fixes

**Change**: 260611-9u91-skills-correctness-idempotency-fixes
**Created**: 2026-06-11

## Origin

> 9u91

Backlog item `[9u91]` (fab/backlog.md, 2026-06-11): *"Skills-review batch 1/4 — surgical correctness + idempotency fixes (skills + 3 small Go fixes). GOAL: fix verified bugs/contradictions that change agent behavior; mostly one-line skill edits."* The entry enumerates 16 finding IDs from the multi-agent skills review report at `docs/specs/findings/skills-review-2026-06-11.md` (134 confirmed findings, every one adversarially verified; line numbers vs commit ae79e04c).

Interaction mode: conversational. The backlog flagged three decisions as "ask user"; all three were asked via structured questions at intake and the user confirmed the recommended option for each:

1. **f006 ownership** → orchestrator owns all `fab status` transitions; subagents return results only.
2. **f012 pass rule** → deterministic: "no must-fix findings (including zero findings) → review passes."
3. **g1-2 re-run semantics** → backlog-ID collision routes to resume (`/fab-switch` + `/fab-continue`); NL re-run documented as intentionally creating a new change. No Go change to `change.go`.

## Why

1. **The pain point**: The skills review verified 16 bugs/contradictions in this batch that change agent behavior at runtime. Skill markdown *is* the implementation (Constitution I), so contradictory or wrong instructions are production bugs: fab-continue's dispatch table prescribes CLI calls that hard-error on the main path (f005 — `start` only accepts `pending`, status.go:38); orchestrators and their subagents are both told to run the same stage transitions, causing guaranteed double-finish errors (f006); the operator's watch dedup hole respawns agents in a loop (f018); the canonical `fab init` → `/fab-setup` flow silently skips project configuration on every new install (f024); every `/fab-help` prints the retired pre-1.10.0 spec/tasks pipeline that the CLI itself rejects (f014); `/git-branch` can silently rename another change's branch away (f100); and five idempotency gaps (g1-2/g1-3/g1-5/g2-2/g1-7) violate Constitution III (skills MUST be safe to re-run).
2. **If we don't fix it**: agents follow whichever of two contradictory lines they read last — burning bounded auto-rework cycles, stranding changes in dead-end states (review `failed` has no documented recovery), duplicating memory Changelog entries on hydrate re-run, and showing users a six-stage pipeline that doesn't exist.
3. **Why this approach**: every edit follows the adversarially-verified recommendation from the report — surgical, mostly one-line changes with the blast radius already mapped by verifiers. The larger staleness sweep, context diet, and twins refactor are deliberately split into backlog batches 2–4 (`uliv`, `zc9m`, `szxd`) so this batch stays reviewable and behavior-focused.

## What Changes

All skill edits land in `src/kit/skills/` (canonical source — `.claude/skills/` is deployed by `fab sync` and never edited directly). Every touched skill gets its `docs/specs/skills/SPEC-*.md` mirror updated; Go changes get test updates (both constitution constraints).

### 1. fab-continue dispatch table — invalid transitions (f005)

`src/kit/skills/fab-continue.md` Step 1:
- Lines 43 and 49: change "finish intake → start apply → execute apply" to **"finish intake (auto-activates apply) → execute apply"** — `finish` auto-activates the next stage; `start` only accepts `pending` and hard-errors (status.go:38). `_preamble.md:256` already says "never call start after finish".
- Line 52 (review-fail row): change `start <change> apply` to **`reset <change> apply`** — matching line 150's Verdict section and both orchestrators.

### 2. Status-transition ownership — orchestrator owns (f006, decided)

Single owner for `fab status` transitions when `/fab-ff`/`/fab-fff` dispatch fab-continue's behavior sections as subagents:
- `src/kit/skills/fab-ff.md` and `fab-fff.md`: subagent dispatch prompts gain an explicit instruction — **"do NOT run `fab status` commands; return results only"**. The orchestrator runs all transitions (finish/fail/reset) itself.
- Resolve the hydrate driver inversion the same way (today fab-ff.md:101 has the subagent run finish while fab-continue.md:175 names driver `fab-continue`).
- `src/kit/skills/fab-continue.md` Apply/Review/Hydrate behavior sections gain a rule: **"when invoked as a subagent, skip §Verdict / the finish step — the orchestrator owns transitions"**.
- The ship re-finish (fab-continue.md:54 re-finishing what git-pr.md:236 already finished) is covered by the same when-subagent rule / guarded phrasing.

This matches `_review.md:16-18` ("verdict transitions remain in each orchestrator's own file").

### 3. Deterministic review pass rule (f012, decided)

`src/kit/skills/_review.md` Findings Merge step 4: replace the hedged "review **may pass**" with the deterministic rule:

> **No must-fix findings (including zero findings) → review passes.**

should-fix and nice-to-have findings are reported but never block. `SPEC-_review.md:62` already states this form; this makes the skill match its spec. Mirror at `docs/specs/skills.md:504`.

### 4. Go: fab-help renders the retired pipeline (f014 + g4-1)

`src/go/fab/.../fabhelp.go`:
- Lines 100–104: replace `"Planning stages: spec → tasks"` / `"Execution stages: apply → review → hydrate"` with the canonical six-stage pipeline: **`intake → apply → review → hydrate → ship → review-pr`** (constitution.md:34).
- Lines 23–41 (`skillToGroupMap`): add the four unmapped skills — `fab-proceed`, `fab-operator`, `git-branch`, `git-pr-review` — grouped per the existing map's semantics.
- Update `fabhelp_test.go` accordingly (constitution: CLI changes need test updates).

### 5. Go: actionable "No active change." errors (f124)

`src/go/fab/internal/resolve/resolve.go` (lines 139, 159–160): append guidance to the bare `No active change.` errors, e.g.:

> `No active change. Run /fab-new <description> to start one, or /fab-switch to activate an existing one.`

This honors `_preamble.md:55`'s promise that preflight stderr "contains the specific error and suggested fix" (the multi-candidate variant at resolve.go:161 already includes a `/fab-switch` hint). Update tests; update the affected `_cli-fab.md` Common Error Messages rows for the changed strings only — the full table regeneration is batch 2 (f052).

### 6. git-pr-review: single exit point + timeout outcome (f015/f016)

`src/kit/skills/git-pr-review.md`:
- Replace each terminal **STOP** in Steps 1, 2, and 4 with "go to Step 6 with outcome {success | failure | no-reviews}"; state in Step 6 that it is the single exit point for all terminal paths after Step 0.
- Add a **fourth outcome class** to Step 6: *Copilot review requested but timed out (10 min)* → **leave the review-pr stage active — no finish, no fail** — and keep the "Re-run /git-pr-review to process when ready" message. (Today the timeout path can be classed as "no reviews" → finish → stage `done` with the review still pending, and `start` can't reactivate a done stage, status.go:48.)

### 7. fab-operator: watch dedup hole (f018)

`src/kit/skills/fab-operator.md` §7 Tick Behavior step 2: deduplicate spawns against **`known` PLUS `completed`**. Today moving an item ID from `known` to `completed` at stop_stage re-enables spawning — a Linear issue still matching the watch query is re-detected and respawned in a loop.

### 8. fab-setup: bootstrap trigger never fires for fab-init configs (f024)

`src/kit/skills/fab-setup.md` step 1a (line 75): broaden the trigger from "missing or raw template" to **"missing OR raw template OR missing required fields `project.name`/`project.description`"** — `fab init` writes a `fab_version`-only config.yaml before sync's copy-if-absent runs, so the current trigger is always false on the canonical install path and project config is silently never collected. Per the verifier: also update the pre-flight reference at line 181, and ensure Config Create Mode **preserves an existing `fab_version` key** (the scaffold template lacks it; config.go:69 errors without it).

### 9. git-branch: rename guard (f100)

`src/kit/skills/git-branch.md` Step 4: rename the current local-only branch **only when its name does not match another change folder** under `fab/changes/` (e.g., `fab change resolve <current-branch>` fails to match); otherwise **create a new branch** (`git checkout -b`). Today any non-main upstream-less branch gets renamed — hijacking another change's unpushed branch after `/fab-switch`. (Known accepted caveat from the verifier: the checkout -b fallback inherits the old change's HEAD.)

### 10. ff/fff: intake-finish condition misses `ready` (f069 + g1-6)

`src/kit/skills/fab-ff.md:52` and `fab-fff.md:52`: change "finish intake first **if still active**" to **"if `progress.intake` is not `done`, finish intake"** — `/fab-new` leaves intake at `ready` (the normal path), and a literal agent matching `active` skips the finish, leaving apply `pending` so the later finish-apply errors. (`finish` accepts both active and ready, status.go:40.)

### 11. fab-status: unsatisfiable ANSI mandate (f047)

`src/kit/skills/fab-status.md:53`: replace the "highlighted in yellow (terminal `\e[33m...\e[0m`)" MUST with the surviving channels — **warning emoji (⚠️) prefix + bold** on the over-threshold Impact line, mirroring fab-operator's health-emoji convention (fab-operator.md:200 empirically verified ANSI SGR is stripped by the render path).

### 12. Idempotency sub-batch (Constitution III)

Kept in this change (backlog allows an optional split; not exercised).

- **g1-2 — fab-new/fab-draft re-run semantics (decided)**: in Step 3 of both skills, detect an existing non-archived change for the detected backlog/Linear ID and **route to resume** — point the user to `/fab-switch {name}` + `/fab-continue` (whose intake-active row regenerates a missing intake) — instead of surfacing the `Change ID already in use` error. Map the `fab change new` collision failure row (fab-new.md:221, fab-draft.md:151) to that recovery guidance. **Document explicitly that a natural-language re-run creates a new change each run.** `change.go:45-47`'s error stays unchanged as the safety net.
- **g1-3 — hydrate merge-without-duplication**: `fab-continue.md` Hydrate step 4 — before appending, check target memory files for an existing entry referencing this change (by change name) and **update in place**; same contract as `docs-hydrate-memory.md:19/176` and `_review.md:75`'s "replaced in place (not duplicated)". Extend the line-209 Key Properties idempotency claim to cover hydrate.
- **g1-5 — review-failed resume guard**: add to fab-continue Step 1 and fab-ff/fab-fff Resumability: **"if `progress.review` is `failed`, run `fab status start <change> review` first"** — the failed→active transition exists exactly for this (status.go:48) but no skill invokes it; today an interruption between `fail review` and `reset apply` is a dead end.
- **g2-2 — generic fab-command failure rule**: add one sentence to `_preamble.md` § Common fab Commands "Key behaviors": **any fab command not explicitly marked best-effort (`2>/dev/null || true`) that exits non-zero → STOP and surface stderr** — deferring to explicit per-skill handling where a skill intentionally branches on non-zero exit (fab-proceed.md:38, fab-discuss.md:40, git-pr.md:182, fab-archive.md:155 do so by design, per the verifier).
- **g1-7 — idempotency declarations**: add the standard Key Properties "Idempotent?" declaration to `fab-new.md` and `fab-draft.md` (stating the g1-2 semantics) and to `git-pr.md` (re-run after ship is a no-op via its lines 117–125 path; the contract exists, it's just unstated). These three files currently have no Key Properties section — add the section, not just a row.

### Non-Goals

- **f019** (review-failed dispatch row presenting the rework menu) — batch 4 (`szxd`), which also restructures the ff/fff bracket.
- **g3-4** (change-type inference vs PostToolUse hook alignment) — batch 2 (`uliv`).
- **f052** (full Common Error Messages table regeneration) — batch 2; only rows whose strings f124 changes are touched here.
- All other batch 2–4 findings (staleness sweep, `_preamble` context diet, twins refactor).
- No `change.go` behavior change for g1-2 — resume routing is skill-level only.

## Affected Memory

- `pipeline/execution-skills`: (modify) fab-continue dispatch fixes, orchestrator-owns-transitions rule, deterministic review pass rule, hydrate merge-without-duplication
- `pipeline/planning-skills`: (modify) fab-new/fab-draft re-run semantics and idempotency declarations
- `pipeline/change-lifecycle`: (modify) git-branch rename guard; fab-status over-threshold highlight channel (emoji/bold)
- `runtime/operator`: (modify) watch dedup against known + completed
- `distribution/setup`: (modify) broadened bootstrap step 1a trigger; fab_version preservation in Config Create Mode
- `_shared/context-loading`: (modify) generic non-best-effort fab-command failure rule in the preamble's Common fab Commands

## Impact

- **Skills (14 files in `src/kit/skills/`)**: fab-continue.md, fab-ff.md, fab-fff.md, _review.md, _preamble.md, _cli-fab.md, git-pr-review.md, git-pr.md, git-branch.md, fab-operator.md, fab-setup.md, fab-status.md, fab-new.md, fab-draft.md
- **Go (`src/go/fab/`)**: fabhelp.go + fabhelp_test.go; internal/resolve/resolve.go + its tests
- **Spec mirrors**: `docs/specs/skills/SPEC-*.md` for every touched skill, plus the flagged lines in `docs/specs/skills.md` (e.g., :298 dispatch wording, :504 pass rule)
- **Line-number drift**: report line numbers are vs commit ae79e04c; HEAD has advanced (f8ba3629) — re-locate each edit by content, not line number, at apply time
- No runtime/user-data migrations: no `.status.yaml`/config schema changes, so no `src/kit/migrations/` file is needed

## Open Questions

- None — the three decisions the backlog flagged (f006 ownership, f012 pass rule, g1-2 re-run semantics) were asked and resolved at intake.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly the 16 finding IDs enumerated in backlog [9u91]; batch 2–4 findings excluded even where they touch the same files | Backlog ACTIONS list is explicit and the sibling batches (uliv/zc9m/szxd) own the rest | S:95 R:85 A:95 D:90 |
| 2 | Certain | f006: orchestrator owns all fab status transitions; subagent prompts say "do NOT run fab status; return results only"; same rule resolves the hydrate inversion and ship re-finish | Asked — user confirmed the recommended option (which included the hydrate/ship consequences); matches _review.md:16-18 | S:95 R:70 A:90 D:90 |
| 3 | Certain | f012: "no must-fix findings (including zero findings) → review passes" | Asked — user confirmed; SPEC-_review.md:62 already states this form | S:95 R:85 A:90 D:95 |
| 4 | Certain | g1-2: backlog-ID collision routes to resume (/fab-switch + /fab-continue); NL re-run = new change, documented; change.go untouched | Asked — user confirmed the recommended option | S:95 R:75 A:90 D:85 |
| 5 | Certain | Every skill edit updates its SPEC-*.md mirror; Go changes update tests; src/kit is canonical (never .claude/skills) | Constitution constraints, restated in the backlog CONSTRAINTS clause | S:95 R:90 A:95 D:95 |
| 6 | Certain | change_type = fix | Keyword rule 1 (intake contains "fix"/"bug"); backlog GOAL is "fix verified bugs" | S:90 R:95 A:95 D:90 |
| 7 | Certain | g2-2 rule worded per backlog with a defer-to-explicit-per-skill-handling carve-out | Rule text is verbatim in the backlog; the carve-out is verifier-mandated (4 skills intentionally branch on non-zero exits) | S:90 R:90 A:90 D:85 |
| 8 | Certain | f024 includes preserving an existing fab_version key in Config Create Mode | Report recommendation states it explicitly; verifier confirms config.go:69 errors without it — load-bearing, not optional | S:85 R:80 A:95 D:90 |
| 9 | Confident | Idempotency sub-batch stays in this change (no second change split) | Backlog says "may split" — optional; single cohesive batch of one-line edits, easy to split later if review balloons | S:80 R:90 A:85 D:80 |
| 10 | Confident | f014: group assignments for fab-proceed/fab-operator/git-branch/git-pr-review inferred from existing skillToGroupMap semantics | Backlog says add them but not which groups; existing map gives the pattern; trivially reversible | S:60 R:90 A:75 D:70 |
| 11 | Confident | f047: over-threshold channel is ⚠️ emoji prefix + bold | Backlog prescribes "emoji/bold"; exact emoji choice mirrors operator health-emoji convention | S:75 R:95 A:85 D:75 |
| 12 | Confident | f100: guard mechanism is `fab change resolve <current-branch>` failing to match; checkout -b fallback accepted with inherited-HEAD caveat | Report recommendation gives the mechanism as an example ("e.g."), not a mandate | S:85 R:85 A:85 D:80 |
| 13 | Confident | f124: update only the _cli-fab.md error rows whose strings change; defer full table regeneration to batch 2 (f052) | Scoping choice to avoid overlapping batch 2; constitution requires _cli-fab updates for CLI changes | S:75 R:90 A:85 D:80 |

13 assumptions (8 certain, 5 confident, 0 tentative, 0 unresolved).
