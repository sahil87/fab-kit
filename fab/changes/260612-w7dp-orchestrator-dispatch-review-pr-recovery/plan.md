# Plan: Orchestrator Dispatch Targets + Review-PR Failure Recovery

**Change**: 260612-w7dp-orchestrator-dispatch-review-pr-recovery
**Intake**: `intake.md`

## Requirements

### Ship Pipeline: Explicit Change Targets + Branch Guard

#### R1: `/git-pr` accepts an explicit `<change>` argument
`git-pr.md` SHALL accept an optional explicit `<change>` argument. When provided, Step 0 MUST resolve it via `fab change resolve <change>` (transient override — `.fab-status.yaml` untouched) and use that change for ALL status transitions and artifact paths. An explicit argument that fails to resolve MUST STOP with an error (a named-but-unresolvable target is a caller error, never a silent `{has_fab} = false` degradation). Arguments matching one of the 7 valid PR types remain type arguments; any other argument is classified as a change reference.

- **GIVEN** `/git-pr w7dp` is invoked while a different change is active
- **WHEN** Step 0 resolves the change context
- **THEN** `{name}` resolves to the w7dp change (transiently) and all of Steps 0a/0b/1/3c/4a–4c operate on it
- **AND** `.fab-status.yaml` is not modified

#### R2: `/git-pr-review` accepts an explicit `<change>` argument
`git-pr-review.md` SHALL accept the same optional explicit `<change>` argument with identical resolution semantics (transient override; explicit-arg resolution failure STOPs; argless failure degrades silently as today).

- **GIVEN** `/git-pr-review w7dp` is invoked
- **WHEN** Step 0 resolves `{name}`
- **THEN** all `fab status` commands and Step 6.5 paths use the w7dp change, not the active one

#### R3: Branch-matches-change guard in both skills
Both skills MUST verify, **before any commit/push/status mutation** (including `fab status start`), that the current branch matches the resolved change (exact equality with the change folder name, or the folder name as a substring of the branch — the same two-form match the former git-pr Step 1b nudge used). On mismatch they MUST STOP with a mismatch report and guidance (`/git-branch`, `/fab-switch`, or passing the intended change) — no autonomous checkout. An empty branch (detached HEAD) is handled by each skill's detached-HEAD path, not a mismatch message.

- **GIVEN** change `X` resolves (explicitly or as the active change) and the current branch belongs to change `Y`
- **WHEN** either skill runs
- **THEN** it STOPs before `fab status start`, before any commit, and before any push, reporting the mismatch with recovery guidance

#### R4: `/fab-fff` Steps 4–5 dispatch with the explicit change argument
`fab-fff.md` Steps 4–5 MUST instruct the subagents to invoke `/git-pr {name}` and `/git-pr-review {name}` (the explicit change argument — the folder name, which unlike the 4-char `{id}` can never collide with the 7 PR-type tokens), so the dispatched skills resolve the orchestrator's target instead of self-resolving the active change. <!-- cycle-1 rework refined {id} → {name} (T020(d)); R4 wording aligned cycle 3 -->

- **GIVEN** `/fab-fff <change-name>` runs with a non-active change override
- **WHEN** the bracket reaches Steps 4–5
- **THEN** ship and review-pr operate on the override change (and the R3 guard STOPs if the checked-out branch belongs to another change)

### Recovery Routes: Executable `/fab-clarify` Replacement

#### R5: All five `/fab-clarify intake` recovery pointers replaced with an executable route
The five sites (`_pipeline.md` stop text, `fab-continue.md` Arguments line, Verdict "Revise requirements" row, Reset Flow step 1, Error Handling row) MUST route intake-deepening recovery through `/fab-continue intake` (Reset Flow: reset to intake, regenerate, advance to ready) followed by `/fab-clarify`. **Override-awareness**: a site whose surrounding flow may be driving a non-active change (the `_pipeline.md` exhaustion stop — the orchestrators advertise the `<change-name>` override) MUST pass the change to BOTH commands (`/fab-continue <change> intake`, then `/fab-clarify <change>`); sites inside fab-continue's own interactive flows, where the active change is implied, use the argless forms. Where plan regeneration is the actual intent, the text MUST state explicitly that `plan.md` must be deleted before re-running `/fab-continue` (the documented force-regeneration mechanism). No site may retain the unexecutable `/fab-clarify intake` form or the false claim that requirements regenerate automatically. <!-- revised in rework cycle 2: the original "argless /fab-clarify" prescription resolved the wrong change under the very override scenario the stop text names -->

- **GIVEN** an exhausted ff/fff rework loop (possibly running under a change-name override) printed the stop summary
- **WHEN** the user follows the alternative recovery guidance
- **THEN** every command in the guidance is executable as written AND resolves the same change the run was driving (`/fab-continue <change> intake`, then `/fab-clarify <change>`), and plan regeneration is only promised together with the delete-`plan.md` instruction

### Operator: Autopilot Dispatch

#### R6: Single dispatch point for autopilot commands
`fab-operator.md` MUST have exactly one dispatch point for a spawned change's initial command: §6 spawn-sequence step 5 embeds `<command>` in the `tmux new-window` invocation, and the autopilot per-change loop MUST NOT send the command a second time. The loop's separate "Dispatch" item is removed (reworded as command-embedded-at-spawn), and the confidence Gate moves before the tab opens so nothing is dispatched for a below-threshold change.

- **GIVEN** autopilot works a queue entry
- **WHEN** the per-change loop runs
- **THEN** the pipeline command reaches the agent pane exactly once (embedded at spawn), and the gate was evaluated before the tab opened

#### R7: Parseable entry-form command
The Working-a-Change entry-form table's Existing-change initial command MUST be the single parseable `/fab-fff <change>` (the change-name override) — `&&`-chained slash commands have no chaining semantics and MUST NOT appear.

- **GIVEN** the operator spawns an agent for an existing change
- **WHEN** the initial command is sent via the spawn sequence
- **THEN** it is `/fab-fff <change>` — one command, no `&&` chain, no `/fab-switch` pre-step

#### R8: Queue-chaining reconciled to nearest-same-repo-predecessor
The implicit `--base` chaining rule MUST be **nearest-same-repo-predecessor**: every change after the first gets `depends_on:` naming the closest earlier queue entry in the same repo (cherry-picked); when no earlier entry shares the repo, the immediately previous entry (cross-repo → ordering-only barrier, no code). Rule text, the worked example, the Dependency Declaration paths, the `--merge-on-complete` paragraph, and the autopilot loop's spawn-next item MUST all state the same semantics.

- **GIVEN** queue `ab12 → cd34 → ef56` where `cd34` lives in a different repo
- **WHEN** dependencies are assigned
- **THEN** `cd34` gets `depends_on: [ab12]` (ordering-only) and `ef56` gets `depends_on: [ab12]` — its nearest same-repo predecessor — preserving the same-repo stack

### `/fab-continue`: Failure Recovery + Idempotent Reset

#### R9: `review-pr`/`failed` dispatch row
`fab-continue.md` Step 1 MUST gain a `review-pr`/`failed` dispatch row keyed off `progress.review-pr == failed` (the same progress-map guard mechanism as the `review`/`failed` row). The route is to re-execute `/git-pr-review` behavior — its Step 0 runs `fab status start <change> review-pr`, and the CLI's review-pr `start` transition accepts `failed → active`. The row MUST NOT route through `reset` (reset From-set `{done, ready, skipped}` excludes `failed` — the CLI would error), and a failed PR review MUST NOT fall through to the "all done / Change is complete" row.

- **GIVEN** a change with `progress.review-pr == failed`
- **WHEN** `/fab-continue` is invoked
- **THEN** it re-executes `/git-pr-review` behavior (whose Step 0 start recovers `failed → active`) instead of reporting the change complete

#### R10: Reset Flow idempotency
When the Reset Flow's target stage is already `active`, the skill MUST skip the `fab status reset` call (`active` is not in the From-set — the call would error) and proceed directly to the execute step, making a re-run of a reset a state-wise no-op (Constitution Principle III).

- **GIVEN** `/fab-continue apply` is re-run after an earlier reset already left apply `active`
- **WHEN** the Reset Flow reaches the reset step
- **THEN** no `fab status reset` is issued and execution proceeds without error

#### R11: Reset From-set documentation aligned to Go
`fab-continue.md`'s event-command list MUST document reset as `done/ready/skipped → active`, matching `status.go` (`reset` From includes `skipped` since k4ge).

- **GIVEN** a reader checks the Step 4 event-command list
- **WHEN** they compare against `status.go`
- **THEN** the documented From-set matches `{done, ready, skipped}`

#### R12: `intake.md`-missing error points to an executable command
The apply-entry error MUST point to `/fab-continue intake` (whose Reset Flow regenerates the intake) instead of plain `/fab-continue` (which re-enters apply and loops back into the same error).

- **GIVEN** apply entry finds no `intake.md`
- **WHEN** the error is shown
- **THEN** it reads "Run /fab-continue intake to regenerate the intake first." (or equivalent naming the intake reset)

### State Enumerations: Alignment to `status.go` AllowedStates

#### R13: Ship and review-pr dispatch rows drop `ready`
`fab-continue.md`'s ship and review-pr dispatch rows MUST cite `active` only — `ready` is not in either stage's AllowedStates (`ship: {pending, active, done, skipped}`, `review-pr: {pending, active, done, failed, skipped}`).

- **GIVEN** the Step 1 dispatch table
- **WHEN** the ship/review-pr rows are read
- **THEN** neither lists `ready` as a reachable state

#### R14: `/fab-switch` `{state}` qualifier enumerates all display states
`fab-switch.md`'s Output section MUST enumerate the `{state}` qualifier as the six states preflight's `display_state` derivation can emit: `active`, `failed`, `ready`, `done`, `skipped`, `pending` (verified against `status.go` `DisplayStage` — Tier 2 emits `failed`, Tier 4 emits `skipped`).

- **GIVEN** a freshly switched draft (intake `ready`) or a change with a parked failure
- **WHEN** the Output documentation is compared with what `fab change switch` prints
- **THEN** the printed state appears in the documented enumeration

#### R15: `/fab-status` legend gains a skipped glyph
`fab-status.md`'s progress-table legend MUST include a `skipped` glyph: `⏭` — the glyph `status.go` `ProgressLine` already prints for skipped stages (consistency with the Go renderer).

- **GIVEN** a change with a skipped stage
- **WHEN** the progress table is rendered per the legend
- **THEN** the skipped stage has a documented glyph (`⏭`) instead of falling outside the legend

### Dispatch Inputs: Defined Contracts

#### R16: `/fab-proceed` fab-new dispatch defines a defer-and-surface contract
`fab-proceed.md`'s fab-new Dispatch MUST define behavior for SRAD Unresolved decisions under the promptless dispatch: the dispatch prompt instructs the subagent to ask no questions; would-be-asked Unresolved decisions are recorded in the intake's `## Assumptions` table as Unresolved rows with Rationale `Deferred — promptless dispatch` and returned in the subagent result; `/fab-proceed` surfaces them to the user (zero-prompt — informational lines) before delegating to `/fab-fff`. The intake gate is the structural backstop (an Unresolved row scores 0.0).

- **GIVEN** the synthesized description leaves an Unresolved decision
- **WHEN** the fab-new subagent runs under `/fab-proceed`
- **THEN** no question is asked; the decision lands in the intake's Assumptions table as `Deferred — promptless dispatch`, `/fab-proceed` surfaces it, and a deferral-heavy intake fails the `/fab-fff` gate normally

#### R17: `_review.md` defines the `change_type` input
`_review.md`'s inward sub-agent prompt contract MUST define where `change_type` comes from: the dispatching orchestrator reads it from the change's `.status.yaml` and passes the value in the sub-agent prompt (Steps 7–8 key their skip condition on it). `fab-continue.md`'s Review Behavior (the dispatching call site) MUST reference supplying it.

- **GIVEN** the inward sub-agent evaluates the parsimony-pass skip condition
- **WHEN** it needs `change_type`
- **THEN** the value was supplied in its prompt per the documented contract — not left undefined

#### R18: `/fab-new` collision pre-check anchored on exact ID equality
`fab-new.md` Step 3's backlog-ID pre-check MUST resolve then compare the canonical 4-char ID (`fab resolve --id {id}`) for **equality** with `{id}` — only an exact ID match routes to resume. A substring hit inside another change's slug (which resolves with a different canonical ID) MUST NOT route to resume.

- **GIVEN** backlog ID `w7dp` and an existing unrelated change whose slug contains `w7dp`
- **WHEN** the pre-check runs
- **THEN** `fab resolve --id w7dp` returns that change's different ID, the equality test fails, and creation proceeds (the CLI `Change ID already in use` error remains the safety net)

### Doc-Truth: Stale Chains, False Claims, Competing Routes

#### R19: `/git-branch` dropped from `/fab-new`-prefixed `/fab-proceed` rows only
`fab-proceed.md`'s dispatch table MUST drop `/git-branch` from the two `/fab-new`-prefixed rows (fab-new Step 11 creates/checks out the branch inline since #322); `/fab-switch`-prefixed rows and the branch-mismatch row keep it (switching activates a change but creates no branch). The git-branch Dispatch section's "runs when" claim MUST match the updated table.

- **GIVEN** the dispatch table selects the `/fab-new` path
- **WHEN** the prefix steps run
- **THEN** no `/git-branch` subagent is dispatched after `/fab-new` (the branch already exists from fab-new Step 11)

#### R20: `_cli-external.md` worktree flow matches inline branch creation
The "New change (from backlog)" flow MUST NOT claim the operator sends `/git-branch` after the intake advances — fab-new Step 11 renames the worktree's disposable branch inline (rename guard: the `wt create` branch resolves to no change). Operator-side restatements of the same stale claim in `fab-operator.md` (tick auto-nudge step, pipeline-commands line) are updated in lockstep to avoid an intra-change contradiction.

- **GIVEN** an operator spawns a backlog-sourced new change
- **WHEN** the documented flow is followed
- **THEN** no step instructs sending `/git-branch` post-intake; the branch alignment is attributed to fab-new Step 11

#### R21: `{driver}` claim scoped to the commands that take the driver
`fab-ff.md` and `fab-fff.md`'s `{driver}` parameter rows MUST NOT claim the driver is "passed to every `fab status` event command" — `_pipeline.md`'s fail/recovery commands are deliberately driver-less. The claim is scoped to the commands the bracket shows it on.

- **GIVEN** the bracket's `fab status fail <change> review` (no driver)
- **WHEN** compared against the wrappers' parameter rows
- **THEN** the rows' wording admits the driver-less commands instead of contradicting them

#### R22: Disjoint rework decision heuristics
`_pipeline.md`'s decision heuristics MUST be disjoint: code-fails-a-correct-requirement failures route to **Fix code**; the-requirement-itself-is-wrong/drifted failures route to **Revise requirements**. Each failure description appears exactly once across the three bullets.

- **GIVEN** a failed review citing a requirements mismatch
- **WHEN** the agent applies the heuristics
- **THEN** exactly one bullet matches, depending on whether the code or the requirement is wrong

### Constitution: SPEC Mirrors

#### R23: Every touched skill file has its SPEC mirror updated in the same change
Each touched `src/kit/skills/*.md` with an existing `docs/specs/skills/SPEC-*.md` MUST have that mirror updated to reflect the change. (`_cli-external.md` has no SPEC mirror — internal CLI references carry none — so none is created.)

- **GIVEN** the change's diff
- **WHEN** the touched skill list is compared with the touched SPEC list
- **THEN** every skill with a mirror has a corresponding mirror edit

### Non-Goals

- No Go code changes — gate exit codes, AllowedStates enforcement, reset From-set, and the metrics cascade landed in k4ge (#395); g8st (#401) guards are the baseline these edits extend
- No migrations, no `.status.yaml` schema changes — markdown only; `fab sync` redeploys
- No `[AUTO-MODE]` adoption or interactive relay for `/fab-proceed`'s fab-new dispatch (defer-and-surface chosen)
- No memory-file (`docs/memory/`) edits — hydrate's responsibility, not apply's

### Design Decisions

1. **Guard match rule reuses the nudge's two-form match**: exact folder-name equality OR folder name as branch substring — *Why*: one matching contract in the pipeline (the rule the Step 1b nudge already established); substring tolerates prefixed branch names while still catching every wrong-change case — *Rejected*: strict equality only (breaks legitimate prefixed-branch setups for no safety gain).
2. **Guard placement: Step 0 of both skills**, before the `fab status start` calls — *Why*: the intake requires the guard before any status mutation; `git-pr` Step 0a and `git-pr-review` Step 0's start are the first mutations — *Rejected*: guard at git-pr Step 2 (Step 0a would already have mutated the override change's status).
3. **Operator single-dispatch keeps command-embedded-at-spawn** (§6 step 5) and drops the loop's separate Dispatch item, moving Gate before the tab opens — *Why*: all three entry forms and the Watches flow already send the initial command "via the spawn sequence's agent tab"; a post-spawn send would need ready-state polling — *Rejected*: bare tab + `fab pane send` dispatch (adds an idle-detection dependency to every spawn).

## Tasks

### Phase 1: Ship-Dispatch Hardening

- [x] T001 `src/kit/skills/git-pr.md`: add optional explicit `<change>` argument (Step 0 transient resolution, hard STOP on explicit-arg resolution failure; argument classification: 7 valid types → type, otherwise change reference; update title line + Step 0b chain step 1); add Step 0 branch-matches-change guard item (two-form match, skip on empty branch — Step 2's detached-HEAD guard owns that, STOP with `/git-branch`//`/fab-switch`/pass-change guidance, before Step 0a); remove the superseded Step 1b mismatch nudge <!-- R1, R3 -->
- [x] T002 [P] `src/kit/skills/git-pr-review.md`: add optional explicit `<change>` argument to title + intro + Step 0 (transient resolution; hard STOP on explicit-arg failure); add the branch-matches-change guard in Step 0 before `fab status start` (two-form match; detached-HEAD STOP folded in) <!-- R2, R3 -->
- [x] T003 [P] `src/kit/skills/fab-fff.md`: Steps 4–5 dispatch prompts invoke `/git-pr {id}` and `/git-pr-review {id}` explicitly (explicit change argument; note the transient override + branch guard) <!-- R4 -->

### Phase 2: Recovery Routes + Pipeline Truth

- [x] T004 `src/kit/skills/fab-continue.md`: add the `review-pr`/`failed` dispatch row (progress-map guard like the review row; route = re-execute `/git-pr-review` behavior, never `reset`); extend the Step 1 failure-state guard paragraph to cover `progress.review-pr == failed`; drop `ready` from the ship and review-pr dispatch rows <!-- R9, R13 -->
- [x] T005 `src/kit/skills/fab-continue.md`: Reset Flow step 3 idempotency (skip `fab status reset` when target already `active`); Step 4 reset line → `done/ready/skipped → active`; Error Handling `intake.md`-missing row → `/fab-continue intake`; replace the `/fab-clarify intake` pointers at the Arguments line, Verdict "Revise requirements" row, Reset Flow step 1, and Error Handling reset-target row with the `/fab-continue intake` → `/fab-clarify` route (+ delete-`plan.md` note where plan regeneration is the intent) <!-- R5, R10, R11, R12 -->
- [x] T006 [P] `src/kit/skills/_pipeline.md`: replace the stop-text `/fab-clarify intake` route (site :108) with the executable sequence incl. the delete-`plan.md` note; make the three decision-heuristic bullets disjoint (fix-code = code fails a correct requirement; revise-requirements = the requirement itself is wrong/drifted) <!-- R5, R22 -->
- [x] T007 [P] `src/kit/skills/fab-ff.md` + `src/kit/skills/fab-fff.md` + `src/kit/skills/_pipeline.md`: reword the `{driver}` parameter rows AND the bracket partial's own `{driver}` parameter description (`_pipeline.md:15-16`) to scope the claim to the commands that actually take the driver (fail/recovery commands are deliberately driver-less) <!-- R21 --> <!-- rework: cycle 1 — the unscoped claim survives verbatim inside _pipeline.md:15-16, contradicting that file's own Behavior note (:45), its driver-less fail commands (:83/:98), and the already-updated SPEC-_pipeline.md:10 mirror -->

### Phase 3: Operator + Dispatch Inputs + Enumerations

- [x] T008 `src/kit/skills/fab-operator.md`: autopilot per-change loop → single dispatch point (Gate first, command embedded at spawn via §6 step 5, separate Dispatch item removed, renumber + fix the `--merge-on-complete` step range); entry-form Existing-change command → `/fab-fff <change>`; queue-chaining rule/worked example/Dependency Declaration paths 2–3/`--merge-on-complete` tail/spawn-next item → nearest-same-repo-predecessor; lockstep consistency edits for the stale post-intake `/git-branch` claims (tick auto-nudge step :200, §6 pipeline-commands line :377, §2 routing example :47) <!-- R6, R7, R8, R20 -->
- [x] T009 [P] `src/kit/skills/fab-proceed.md`: fab-new Dispatch defer-and-surface contract (no-questions prompt, `Deferred — promptless dispatch` rows, surfaced before `/fab-fff`); drop `/git-branch` from the two `/fab-new`-prefixed dispatch-table rows; update the git-branch Dispatch "runs when" claim <!-- R16, R19 -->
- [x] T010 [P] `src/kit/skills/_review.md` + `src/kit/skills/fab-continue.md`: define the inward sub-agent's `change_type` input (dispatcher reads `.status.yaml`, passes it in the prompt) in `_review.md`'s context contract; reference supplying it at fab-continue's Review Behavior call site <!-- R17 -->
- [x] T011 [P] `src/kit/skills/fab-new.md`: backlog-ID collision pre-check → `fab resolve --id {id}` exact-equality compare (substring slug hits no longer route to resume) <!-- R18 -->
- [x] T012 [P] `src/kit/skills/fab-switch.md`: `{state}` qualifier enumeration → the six display states (`active`, `failed`, `ready`, `done`, `skipped`, `pending`) <!-- R14 -->
- [x] T013 [P] `src/kit/skills/fab-status.md`: progress-table legend gains `⏭ skipped` (matches `status.go` `ProgressLine`) <!-- R15 -->
- [x] T014 [P] `src/kit/skills/_cli-external.md`: rewrite "New change (from backlog)" step 3 — no post-intake `/git-branch` send; branch aligned inline by fab-new Step 11 <!-- R20 -->

### Phase 4: SPEC Mirrors + Verification

- [x] T015 `docs/specs/skills/SPEC-git-pr.md`, `SPEC-git-pr-review.md`, `SPEC-fab-fff.md`: mirror the explicit `<change>` argument, the Step 0 branch-matches-change guard, the removed Step 1b nudge, and the explicit-arg Steps 4–5 dispatch <!-- R23 -->
- [x] T016 `docs/specs/skills/SPEC-fab-continue.md`, `SPEC-_pipeline.md`, `SPEC-fab-ff.md`: mirror the review-pr/failed row, idempotent reset, executable recovery route, scoped `{driver}` claim, and disjoint heuristics <!-- R23 -->
- [x] T017 `docs/specs/skills/SPEC-fab-operator.md`, `SPEC-fab-proceed.md`, `SPEC-_review.md`, `SPEC-fab-new.md`, `SPEC-fab-switch.md`, `SPEC-fab-status.md`: mirror single-dispatch autopilot + `/fab-fff <change>` entry form + nearest-same-repo chaining; defer-and-surface + dropped `/git-branch` rows; `change_type` prompt input; exact-ID collision check; state enumeration; skipped glyph <!-- R23 -->
- [x] T018 Verification sweep: `git status --porcelain` shows only `src/kit/skills/*.md`, `docs/specs/skills/SPEC-*.md`, and this change's `fab/changes/...` artifacts; no `.claude/skills/` file modified; grep confirms zero remaining `/fab-clarify intake` pointers, `&& /fab-proceed` chains, and `depends_on: [<prev-change-id>]` rule text in `src/kit/skills/` <!-- R23 -->

### Phase 5: Rework (cycle 1)

- [x] T019 `docs/specs/skills.md` + `docs/specs/glossary.md`: eradicate the unexecutable `/fab-clarify intake` route from the spec aggregates — skills.md:301 and :534 replace it with the executable `/fab-continue intake` → `/fab-clarify` route (:534 also drops the false auto-regeneration claim); skills.md:325 re-quotes the current fab-continue.md error text verbatim; glossary.md:121 gets the same executable route <!-- R5 -->
- [x] T020 Should-fix batch (review cycle 1, clear/low-effort — each with its SPEC mirror updated in lockstep): (a) `src/kit/skills/fab-continue.md:55` delete the stale "preflight's derived stage/state never yields a `failed` tier" parenthetical (deletion candidate — `DisplayStage` emits a failed tier since dkn3); (b) `fab-continue.md:57-59` ship/review-pr rows (incl. the new failed row) pass the resolved change explicitly to `/git-pr`/`/git-pr-review` via the new explicit-arg contract; (c) `fab-continue.md:194` extend the reset carve-out to non-resettable states — target `failed` routes via the failed dispatch rows (`start` owns failed→active, review/review-pr only), target `pending` errors with guidance; (d) `src/kit/skills/fab-fff.md:43,:53` dispatch `{name}` (folder name — never a valid type token) instead of `{id}` and define the parameter, removing the type-word collision; (e) `src/kit/skills/git-pr.md:255` and `src/kit/skills/git-pr-review.md:235` reword residual "active change" phrasing on the override path to the resolved change; (f) `src/kit/skills/fab-proceed.md:139` + `docs/specs/skills/SPEC-fab-proceed.md:105` correct "deferral-heavy" — a single Deferred row zeroes the score and fails the gate; (g) `src/kit/skills/git-pr.md` order the detached-HEAD STOP before Step 0a's `fab status start` mutation (verify-before-mutate parity with git-pr-review); (h) optional one-liners: `fab-continue.md:83` add the review/review-pr-only qualifier to the `start` event line; `_pipeline.md:108` stop guidance names the change in the `/fab-continue` route <!-- R1, R4, R5, R9, R16, R23 -->

### Phase 6: Rework (cycle 2)

- [x] T021 `src/kit/skills/_pipeline.md` + `docs/specs/skills/SPEC-_pipeline.md`: override-aware clarify route per revised R5 — the exhaustion-stop guidance passes the change to BOTH commands (`/fab-continue <change> intake`, then `/fab-clarify <change>`; clarify accepts the `<change-name>` override per fab-clarify.md:28); also name the change in the stop template's `Run /fab-continue for manual rework options.` line (`_pipeline.md:104`) for the same reason <!-- R5 -->
- [x] T022 Should-fix batch (review cycle 2, clear/low-effort): (a) `docs/specs/skills.md:752` `## /git-pr [type]` section gains the `[<change>]` argument + type-vs-change classification + hard-STOP semantics, and `:787` `## /git-pr-review` gains `[<change>]` — aligned with the updated skill sources; (b) `docs/specs/architecture.md:317` replace the removed-nudge claim with the branch-matches-change hard-STOP guard behavior; (c) optional one-liner: `src/kit/skills/git-pr-review.md:11` adopt git-pr's value-based argument classification wording so `--tool`'s value cannot be misread as a change reference <!-- R1, R2, R3, R23 -->

### Phase 7: Rework (cycle 3)

- [x] T023 `docs/specs/architecture.md` § Git Integration (:294-323): reconcile the section with the branch-matches-change hard guard this change introduced — (a) :296 "strictly informational / Fab never couples its state to git state" → state that ship-time skills now enforce branch↔change correspondence (guard STOPs before any mutation); (b) :302 multi-branch-per-change claim → qualify with the ship-path reality (shipping requires the branch to match; `/git-branch` aligns it); (c) :314 "Adopt current branch" row → note that an adopted foreign-named branch must be aligned via `/git-branch` before `/git-pr` (the guard rejects non-matching names; no autonomous checkout); (d) :323 "no rename" claim → keep storage semantics but cross-reference the ship-time guard. Do NOT invent an intentional-mismatch escape hatch — the intake chose hard STOP; the escape-hatch question is surfaced to the user as a follow-up <!-- R3 -->
- [x] T024 Should-fix batch (review cycle 3, clear/low-effort): (a) override-unaware sibling guidance in files this change already touches names the change — `_pipeline.md:32` gate-fail guidance (`/fab-clarify <change>`), `:62`/`:132` `re-run /{driver} <change>` where the override may be driving, `fab-fff.md:61,:95,:105-107` re-run/retry lines, `git-pr-review.md:112` + Step 6 item 4 re-run guidance (name the change when one was passed); (b) `src/kit/skills/_srad.md` Critical Rule gains the promptless-dispatch carve-out cross-referencing fab-proceed's defer-and-surface contract (and fab-proceed.md:139 references the carve-out back) — resolves the contradictory MUSTs a subagent loading both would receive; (c) `docs/specs/skills/SPEC-fab-continue.md:7` fix the "rows … key on `active` only" self-contradiction (the failed row keys on `failed`); (d) `docs/specs/skills.md:~326` reset description: target → `active`, downstream → `pending`, plus the non-resettable-state carve-outs; (e) `fab-continue.md:55` replace the soft "reached from the progress map, not the dispatch key" clause with DisplayStage-accurate wording (the failed tier IS derivable since dkn3); (f) `fab-continue.md` + `SPEC-fab-continue.md:7` reword the argless-forms rationale to "the change reference of the current invocation is implied (active or override)" so it stays true under fab-continue's own `[change-name]` override; SPEC mirrors in lockstep for every touched file <!-- R3, R5, R16, R23 -->

### Phase 8: Rework (cycle 4 — manual, must-fix only)

- [x] T025 `src/kit/skills/git-pr-review.md` Step 5 push-failure recovery (:178): apply the :112 conditional change-naming treatment — when an explicit `<change>` was passed in Step 0, the recovery's re-run command names it (`… then re-run /git-pr-review <change>.`; argless resolves the active change instead); update `docs/specs/skills/SPEC-git-pr-review.md`'s recovery paraphrase (:82) in lockstep <!-- R2, R5 --> <!-- rework: cycle 4 (manual, user-selected must-fix-only) — A-044 residual; cycle-4 should-fixes deliberately deferred per user choice -->
- [x] T026 `docs/specs/skills/SPEC-git-pr-review.md` Step 6.5 gate mirror drift: `:105` flow gate and `:143` Gate row say "active change" where the skill's gates (:217/:228/:243) say "a change was resolved in Step 0 (active or explicit)" — read literally the SPEC excluded the primary explicit-dispatch path; `:143` also gains the `timeout` skip the skill (:228) and flow gate (:105) already state <!-- R2 --> <!-- rework: cycle 5 (manual, must-fix only) — A-034 mirror-drift residual found by outward review -->

## Execution Order

- T001 → T003 (fab-fff's dispatch text references the argument T001/T002 introduce)
- T004 → T005 (same file — sequential edits to fab-continue.md)
- T005 and T010 both touch fab-continue.md — run T010 after T005
- T015–T017 after all Phase 1–3 tasks (mirrors reflect final skill text); T018 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `git-pr.md` documents the optional `<change>` argument with transient resolution, type-vs-change argument classification, and a hard STOP on explicit-arg resolution failure
- [x] A-002 R2: `git-pr-review.md` documents the optional `<change>` argument with the same semantics
- [x] A-003 R3: both skills specify the branch-matches-change guard before any commit/push/status mutation, with mismatch STOP + `/git-branch`//`/fab-switch` guidance and no autonomous checkout
- [x] A-004 R4: `fab-fff.md` Steps 4–5 pass `{name}` (collision-proof folder name; refined from `{id}` in cycle 1) as the explicit argument to `/git-pr` and `/git-pr-review`
- [x] A-005 R5: all five recovery sites route through `/fab-continue intake` + `/fab-clarify`, with the change name passed to BOTH commands at override-capable sites (`_pipeline.md` stop text) and argless forms only where the active change is implied; zero `/fab-clarify intake` occurrences remain in `src/kit/skills/` <!-- rework: cycle 2 — re-verify against the revised R5 override-awareness rule --> <!-- verified cycle 3: _pipeline.md:109 names the change in both commands; fab-continue.md:24/:164/:192/:213 use argless forms (active change implied); grep over src/kit/skills/ returns zero occurrences -->
- [x] A-006 R6: `fab-operator.md` has exactly one dispatch point for the initial command (spawn-embedded) and the autopilot loop no longer contains a second send
- [x] A-007 R7: the entry-form table's Existing-change command is `/fab-fff <change>` with no `&&` chain
- [x] A-008 R8: rule text, worked example, Dependency Declaration, `--merge-on-complete`, and spawn-next all state nearest-same-repo-predecessor chaining
- [x] A-009 R9: `fab-continue.md` has a `review-pr`/`failed` row keyed off the progress map routing to `/git-pr-review` re-execution (no `reset`)
- [x] A-010 R10: Reset Flow documents the skip-when-already-active no-op path
- [x] A-011 R11: the reset event line reads `done/ready/skipped → active`
- [x] A-012 R12: the `intake.md`-missing error names `/fab-continue intake`
- [x] A-013 R13: ship and review-pr dispatch rows list `active` only
- [x] A-014 R14: `fab-switch.md` enumerates all six display states for `{state}`
- [x] A-015 R15: `fab-status.md` legend includes `⏭` for skipped
- [x] A-016 R16: `fab-proceed.md` defines the defer-and-surface contract (no questions; `Deferred — promptless dispatch` rows; surfaced before `/fab-fff`)
- [x] A-017 R17: `_review.md` defines the `change_type` prompt input and fab-continue's call site references supplying it
- [x] A-018 R18: `fab-new.md`'s backlog-ID pre-check compares `fab resolve --id` output for equality with `{id}`
- [x] A-019 R19: `/git-branch` removed from exactly the two `/fab-new`-prefixed rows; switch rows + mismatch row keep it; the git-branch Dispatch claim matches
- [x] A-020 R20: `_cli-external.md` (and the operator's lockstep sites) no longer claim a post-intake `/git-branch` send
- [x] A-021 R21: `fab-ff.md`/`fab-fff.md` `{driver}` rows no longer claim "every `fab status` event command"
- [x] A-022 R22: the three heuristics bullets are disjoint — each failure description appears exactly once

### Behavioral Correctness

- [x] A-023 R3: the guard's placement precedes `fab status start` in both skills (mutation-free STOP path)
- [x] A-024 R9: the new row's transition relies only on `start: failed → active` (valid per `status.go` review-pr stageTransitions); no documented command violates AllowedStates

### Scenario Coverage

- [x] A-025 R8: the `ab12 → cd34 → ef56` worked example's assigned `depends_on` values are stated and consistent with the rule text
- [x] A-026 R5: the exhaustion-stop guidance in `_pipeline.md` is executable end-to-end as written

### Edge Cases & Error Handling

- [x] A-027 R1: an explicit `<change>` that fails to resolve produces a STOP (not silent `{has_fab}=false`) in both skills
- [x] A-028 R3: a detached HEAD (empty branch) routes to the detached-HEAD handling, not a confusing empty-name mismatch report
- [x] A-029 R18: a backlog ID that substring-matches another change's slug does not route to resume (CLI collision error remains the safety net)

### Code Quality

- [x] A-030 Pattern consistency: edits match each file's existing conventions — table-driven dispatch rows, RFC-2119 keywords, step-numbered flows, the per-file voice
- [x] A-031 No unnecessary duplication: shared contracts stated once and referenced (guard match rule defined in each skill's own Step 0 but mirroring the established two-form match; chaining rule stated in the queue table and referenced elsewhere)
- [x] A-032 Readability: replacement wording is direct and avoids cleverness; no god-paragraphs introduced

### Documentation Accuracy

- [x] A-033: every behavioral claim added matches `status.go` ground truth (AllowedStates, transitions, From-sets) and existing skill cross-file behavior (fab-new Step 11, wt create branch naming)
- [x] A-034: SPEC mirrors accurately restate the updated skill behavior (no mirror drift introduced) <!-- rework: cycle 5 — SPEC-git-pr-review.md:105/:143 Step 6.5 gate said "active change" vs the skill's "(active or explicit)"; fixed by T026 --> <!-- verified re-review cycle 5: SPEC:105 flow gate and :143 Gate row now state "(active or explicit)" + the timeout skip, matching the skill's gates at :217/:228/:237/:243; the SPEC's only remaining "active change" phrasings (:59, :84) are the correct argless-fallback semantics mirroring skill :112/:182, not gate statements; skill file unmodified this cycle (:213-255 byte-identical to the cycle-4 verified text; mtimes confirm only SPEC + plan touched) -->

### Cross References

- [x] A-035: cross-file pointers updated in lockstep — fab-fff ↔ git-pr/git-pr-review argument contract, fab-proceed ↔ fab-new Step 11, _cli-external ↔ fab-operator, _review ↔ fab-continue call site
- [x] A-036: no file outside the intended set (`src/kit/skills/*.md`, `docs/specs/skills/SPEC-*.md`, this change's artifacts) is modified; `.claude/skills/` untouched

### Rework (cycle 1)

- [x] A-037 R21: no unscoped "passed to every `fab status` event command" claim remains anywhere in `src/kit/skills/` — `_pipeline.md`'s own `{driver}` parameter description matches its Behavior note and the SPEC-_pipeline.md mirror
- [x] A-038 R5: `/fab-clarify intake` occurrences in `docs/specs/skills.md` and `docs/specs/glossary.md` are eliminated or strictly historical annotations; skills.md's quoted reset-target error text matches fab-continue.md verbatim
- [x] A-039 R4: every `/git-pr`//`git-pr-review` dispatch from fab-fff AND fab-continue passes the resolved change explicitly; fab-fff dispatches the collision-proof `{name}` and defines it
- [x] A-040 R1/R3: git-pr's detached-HEAD STOP precedes any status mutation (parity with git-pr-review); residual "active change" phrasings on the override path corrected; fab-proceed's gate-math claim is accurate

### Rework (cycle 2)

- [x] A-041 R5: the `_pipeline.md` exhaustion-stop guidance resolves the driven change in the override scenario — both the `/fab-continue <change> intake` and `/fab-clarify <change>` commands name the change, and the SPEC-_pipeline mirror states the same route <!-- verified cycle 3: _pipeline.md:106 stop template names the change in the rework-menu line, :109 names it in both commands (fab-clarify.md:26 confirms the <change-name> override); SPEC-_pipeline.md:28 mirrors the full override-aware route -->
- [x] A-042 R1/R2: `docs/specs/skills.md`'s `/git-pr` and `/git-pr-review` sections carry the `[<change>]` argument with the new classification/STOP semantics, and `docs/specs/architecture.md` no longer describes the removed mismatch nudge <!-- verified cycle 3: skills.md:752 `## /git-pr [<change>] [type]` + :757 classification/hard-STOP bullet + :781 branch-guard Key property; :789 `## /git-pr-review [<change>] [--tool <name>]` + :794 argument bullet; architecture.md:317 describes the hard guard, zero nudge references remain -->

### Rework (cycle 3)

- [x] A-043 R3: `docs/specs/architecture.md` § Git Integration is internally consistent with the hard guard — no surviving claim of uncoupled git state, unqualified multi-branch flows, or guard-free adopt-branch shipping; the documented ship path for a foreign-named branch is `/git-branch` alignment <!-- verified re-review cycle 3: :296 splits state-bookkeeping (informational) from the ship-path exception (guard STOPs pre-mutation); :298 retitled "(During Development)"; :302 multi-branch claim carries the ship-time qualifier; :314 Adopt row routes foreign names via /git-branch (no autonomous checkout); :317/:323 state the two-form match with pass/STOP examples; no intentional-mismatch escape hatch introduced -->.
- [x] A-044 R5: no override-capable recovery/re-run guidance in the files this change touches is argless — `_pipeline.md` gate-fail and re-run lines, `fab-fff.md` retry lines, and `git-pr-review.md` re-run guidance name the change where one may be driving; `_srad.md`'s Critical Rule and fab-proceed's defer-and-surface contract cross-reference each other instead of issuing contradictory MUSTs <!-- verified re-review cycle 4 (T025): git-pr-review.md:182 — the note after the push-failure fenced block now gives the :112 conditional change-naming treatment ("When an explicit <change> was passed in Step 0, include it in the recovery's re-run command (… then re-run /git-pr-review <change>.) — an argless re-run would resolve the active change instead"); SPEC-git-pr-review.md:81-85 mirrors it in the Step 5 push-fails branch. All other clauses re-verified still met: _pipeline.md:32/:62/:106/:109/:132 and fab-fff.md:61/:95/:105-107 name the change; git-pr-review.md:112 + Step 6 item 4 (:222) name it; _srad.md:49 carve-out ↔ fab-proceed.md:139 cross-reference each other. The argless advisories at git-pr-review.md:100/:114 ("Run /git-pr-review when reviews are added") remain nice-to-have per the cycle-3 judgment (post-finish future-run advisories, not recovery guidance); cycle-4 should-fixes deliberately deferred per user choice -->.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — independently re-verified at the cycle-3 re-review. The cycle-1 candidates remain resolved: `_pipeline.md:15-17`'s `{driver}` definition carries the scoped claim (consistent with the Behavior note at `_pipeline.md:46`, both wrappers' parameter rows at `fab-ff.md:32`/`fab-fff.md:32`, and `SPEC-_pipeline.md:10`), and "never yields a `failed` tier" greps to zero hits in `src/kit/skills/` and all of `docs/specs/` (its last live occurrence is `docs/memory/pipeline/execution-skills.md:20` — hydrate's territory). Everything this change superseded was deleted in-change: git-pr's Step 1b mismatch nudge (removal annotated in `SPEC-git-pr.md`), the five unexecutable `/fab-clarify intake` pointers (the only remaining `docs/specs/` occurrences are historical "replaced the unexecutable…" annotations in `SPEC-_pipeline.md:28`/`SPEC-fab-continue.md:7` plus the immutable findings report), the `&&`-chained operator entry command, and the autopilot loop's duplicate Dispatch item (loop renumbered 9→8). The cycle-3 batch (architecture.md § Git Integration reconciliation, `_srad.md` promptless-dispatch carve-out, override-aware re-run lines, SPEC/aggregate lockstep) replaced claims in place and created no new redundancy; `fab-operator.md:465`'s surviving `depends_on: [<prev-change-id>]` is the explicit `--base` flag's own semantics, not the implicit rule R8 rewrote — not redundant. Stale memory passages (`runtime/operator.md:87/:95/:160/:223/:225/:349/:415`, `pipeline/execution-skills.md:20/:28/:30`, `pipeline/planning-skills.md:163`, `pipeline/change-lifecycle.md:174/:176`) are hydrate's responsibility per the Affected Memory list, not deletion candidates.

## Assumptions

<!-- Three grades only (Certain/Confident/Tentative) — apply decides-and-records. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Branch guard match rule = the former Step 1b two-form match (exact folder-name equality OR folder name as branch substring), not strict equality | Keeps one matching contract in the pipeline; substring still catches every wrong-change case while tolerating prefixed branches; intake left exact wording to apply | S:60 R:85 A:80 D:70 |
| 2 | Confident | Guard lives in Step 0 of both skills (before `fab status start`); git-pr's Step 1b nudge is removed as superseded; empty-branch (detached HEAD) defers to git-pr Step 2's existing STOP and gets a folded-in STOP in git-pr-review | Intake requires the guard "before any commit/push/status mutation" — Step 0a/Step 0 starts are the first mutations; a dead nudge behind a hard guard is noise | S:65 R:80 A:80 D:75 |
| 3 | Confident | Explicit `<change>` argument that fails to resolve → hard STOP (argless failure keeps today's silent degradation) | A named-but-unresolvable dispatch target is the exact wrong-change failure class this change exists to close; silent fallback would reintroduce it | S:60 R:85 A:80 D:80 |
| 4 | Confident | Operator single-dispatch = command-embedded-at-spawn (§6 step 5 kept); autopilot Gate moved before the tab opens; loop renumbered 9→8 items with the Dispatch item folded into the open-tab item | Entry forms + Watches already send the initial command via the spawn tab; a bare-tab + post-spawn send adds idle-detection coupling the audit didn't ask for; gating after dispatch would be ineffective | S:60 R:75 A:75 D:65 |
| 5 | Certain | `/fab-status` skipped glyph = `⏭` | `status.go` `ProgressLine` already prints `⏭` for skipped — the intake said to match an existing renderer | S:80 R:90 A:95 D:90 |
| 6 | Certain | `/fab-switch` `{state}` enumeration = `active`/`failed`/`ready`/`done`/`skipped`/`pending` | Verified against `status.go` `DisplayStage` tiers (failed = Tier 2, skipped via Tier 4) as the intake instructed | S:80 R:90 A:95 D:90 |
| 7 | Confident | Recovery-route wording per site: error rows say "use /fab-continue apply to re-run apply (delete plan.md first to force regeneration), or /fab-continue intake then /fab-clarify"; prose sites spell out reset-then-clarify with the delete-`plan.md` note | Intake fixed the route and left per-site wording to apply; wording keeps each site's existing form (error message vs. prose) | S:60 R:85 A:80 D:70 |
| 8 | Confident | Lockstep consistency edits slightly beyond the intake's site list: fab-operator.md's three stale `/git-branch`-after-intake claims (:47, :200, :377) and the new `review-pr`/`failed` row avoids restating the review-row's now-stale "never yields a failed tier" parenthetical (DisplayStage gained a failed tier in dkn3) | Fixing `_cli-external.md:63` while leaving the operator instructing the same stale send would create a contradiction between two files this change touches; restating a known-false claim in new text would be a doc-truth regression | S:55 R:85 A:75 D:70 |
| 9 | Certain | `_cli-external.md` gets no SPEC mirror | No `SPEC-_cli-external.md` exists; the repo carries no mirrors for the `_cli-*` reference partials — the constitution constraint applies to mirrors that exist | S:75 R:90 A:90 D:90 |
| 10 | Confident | Queue-chaining fallback when no same-repo predecessor exists = the immediately previous entry as an ordering-only barrier | Matches the worked example (`cd34` waits on `ab12`, no code) — the only reading under which the example and the two-tier semantics are both coherent | S:60 R:75 A:75 D:70 |
| 11 | Confident | (cycle 1) Reset-target `pending` error text: `Stage '{stage}' has not run yet — nothing to reset. Run /fab-continue to advance to it.`; `failed`-target routes via the Step 1 failed dispatch rows rather than a new inline recovery | T020(c) delegated the wording ("errors with guidance"); routing failed to the existing rows keeps one recovery surface per state instead of duplicating it inside the Reset Flow | S:60 R:90 A:80 D:75 |
| 12 | Confident | (cycle 1) `/fab-clarify intake` eradication scope excludes `docs/specs/findings/` (the immutable review report documenting the defect); remaining SPEC-mirror occurrences are strictly historical "replaced/was unexecutable" annotations | A findings report quoting the defect is evidence, not a live pointer; rewriting it would falsify the audit record | S:65 R:90 A:85 D:80 |
| 13 | Confident | (cycle 2) Adjacent stale claims fixed in the T022 sentences being edited: architecture.md:317's "/fab-new does not handle branches" (false since #322 — same lockstep rationale as row 8) corrected alongside the nudge claim; skills.md's `/git-pr` Key properties gained the branch-guard STOP line so the aggregate's hard-STOP picture is complete; SPEC-fab-continue's "argless /fab-clarify" sentence gained the active-change-implied scope qualifier | Leaving a known-false claim inside the exact sentence cluster under edit would be a doc-truth regression; the qualifier prevents the next review from misreading fab-continue's correct argless usage as the override defect | S:60 R:90 A:80 D:75 |
| 14 | Confident | (cycle 3) § Git Integration reframed as one voice: "Why Decoupled" retitled "Why Decoupled (During Development)", state-bookkeeping-vs-ship-path split stated once in the intro and echoed (not restated) at the bullets/Adopt row/Branch Naming; no intentional-mismatch escape hatch introduced per the task's explicit instruction | T023 delegated the reconciliation wording; a single intro-level frame keeps the four fixes from reading as patches, and the escape-hatch question is the user's separate follow-up | S:65 R:90 A:80 D:75 |

14 assumptions (3 certain, 11 confident, 0 tentative).
