---
description: "`/fab-new`, `/fab-continue`, `/fab-ff`, `/fab-clarify` — the planning stage (intake only) and the shared `_generation.md` partial (Intake + unified Plan procedures); requirement capture + plan generation live at apply entry (spec stage removed in j6cs)"
---
# Planning Skills

**Domain**: pipeline

## Overview

The planning skills (`/fab-new`, `/fab-clarify`) handle the single planning stage of the 6-stage Fab pipeline: **intake**. They produce the only pre-code planning artifact (`intake.md`), which defines *what* changes and *why*. Intake is also the sole confidence gate — all human judgment is frontloaded here. Requirement capture (the former `spec.md`) and the implementation plan are co-generated into a single `plan.md` at apply entry — see [execution-skills.md](execution-skills.md) — not by a planning skill. The `spec` stage and `spec.md` artifact were removed in j6cs.

`/fab-fff` and `/fab-ff` are also documented here because their planning behavior originated as planning skills. `/fab-fff` is the **full-pipeline command** (intake → review-pr, intake-gated, no frontloaded questions, autonomous rework). `/fab-ff` is the **fast-forward command** (intake → hydrate, intake-gated, autonomous rework). See sections below for details.

## Shared Generation Partial

Artifact generation is defined in a single shared partial: `$(fab kit-path)/skills/_generation.md`. As of j6cs the standalone Spec Generation Procedure is **deleted** — requirement generation is folded into the Plan Generation Procedure invoked by the apply skill (within `/fab-continue`) at apply entry.

The partial contains one procedure:
- **Plan Generation Procedure** — template loading (`plan.md`), one walk that emits `## Requirements` (RFC-2119 statements with stable `R#` IDs + GIVEN/WHEN/THEN scenarios, generated FIRST from the intake-derived design), then paired Task + Acceptance entries per requirement with sequential `T-NNN`/`A-NNN` IDs. It reads `intake.md` as input (not a separate `spec.md`), and includes a one-release legacy `spec.md` ingestion path (fold a leftover `spec.md` into `## Requirements` if present and `plan.md` lacks them). Trace annotations are **REQUIRED** (j6cs): each `## Tasks` item carries a `<!-- R# -->` annotation; each `## Acceptance` item names its `R#`. No `[NEEDS CLARIFICATION]` markers are emitted into `plan.md` — under-specified points become graded SRAD assumptions in `## Assumptions` instead. The procedure runs at apply entry, not as a separate planning stage. Counts (`task_count`, `acceptance_count`, `acceptance_completed`) flow into `.status.yaml` `plan:` block via the PostToolUse `artifactBookkeeping` hook on every `plan.md` write; skills MAY also call `fab status set-acceptance` for explicit updates (e.g., review marking acceptance items complete).

Command invocations are auto-logged via `preflight.sh --driver <skill-name>` — skills no longer call `log-command` manually. All event commands (`start`, `advance`, `finish`, `reset`, `fail`) accept an optional `driver` parameter; skills always pass it to identify the invoking skill (e.g., `fab-continue`, `fab-ff`).

**Hook-backed bookkeeping**: Bookkeeping commands (confidence scoring, change type inference, plan metadata) are supplemented by a PostToolUse hook (`on-artifact-write.sh`) that fires on Write and Edit events. The hook is a **reliability layer** — it catches bookkeeping the agent forgets. For `plan.md`, the hook performs section-bounded parsing: counts `- [ ]` + `- [x]` items between `## Tasks` and the next `##` heading for `task_count`; same between `## Acceptance` and the next `##` heading for `acceptance_count`; counts `- [x]` in `## Acceptance` for `acceptance_completed`. Missing sections leave the corresponding fields untouched (defensive: avoid overwriting valid values with zero on a malformed in-progress write). Skills keep their existing bookkeeping instructions unchanged for agent-agnostic portability (non-Claude-Code agents rely on skill instructions only). All bookkeeping commands are idempotent, so both the hook and the skill running the same command produces no conflict.

Each skill retains its own orchestration logic (stage guards, question handling, auto-clarify, resumability). Only the generation mechanics are shared.

## Requirements

### `/fab-new <description>`

`/fab-new` starts a new change from a natural language description. It is adaptive: clear inputs get a quick intake, vague inputs trigger conversational exploration. It creates the change folder, initializes status tracking, generates an intake (with Origin section), advances intake to `ready`, activates the change, and creates the matching git branch. Output includes `intake.md` plus activation and branch creation output.

#### Slug Generation and Change Creation

The agent generates a 2-6 word slug (lowercase, hyphen-joined, no articles/prepositions) from the description. If a Linear issue ID was parsed, it prefixes the slug (e.g., `DEV-988-add-oauth`). The folder name format `{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}` is constructed by `lib/changeman.sh`, which handles date generation, random/provided 4-char ID, collision detection, directory creation, `created_by` detection, `.status.yaml` initialization, and statusman integration. The skill calls `changeman.sh new --slug <slug> [--change-id <4char>] [--log-args <description>]` as a single operation and captures the folder name from stdout.

#### Adaptive Behavior (SRAD-Driven)

`/fab-new` adapts its interaction style based on the input clarity:

1. **Clear input** — SRAD scoring identifies few or no Unresolved decisions. The skill generates the intake with up to 3 targeted questions (highest blast radius), assumes all Confident/Tentative decisions, and completes quickly.
2. **Vague input** — SRAD scoring identifies many Unresolved decisions. The skill enters **conversational mode**: back-and-forth exploration with no fixed question cap, starting with the highest-impact decisions (lowest Reversibility + lowest Agent Competence). Each question builds on previous answers. The conversation ends when the confidence score reaches >= 3.0 and the user signals satisfaction, or the user terminates early.

#### Gap Analysis

Before committing to an intake, `/fab-new` evaluates whether the change is needed:

1. Checks for existing mechanisms in the current workflow, codebase, or memory
2. Evaluates scope — is the idea too broad (should be split) or too narrow (part of something larger)?
3. Considers alternatives — simpler approaches, extending existing skills

If an existing mechanism covers the idea, the skill presents its findings and lets the user decide whether to proceed. If no change folder is created, no `Next:` line is shown.

#### Change Initialization

The skill SHALL:
1. Generate the slug (AI task: word selection, article removal, issue ID prefixing)
2. Call `lib/changeman.sh new` with `--slug`, optional `--change-id` (backlog ID), and `--log-args` (description). The script handles: directory creation, `created_by` detection (`gh api user` → `git config user.name` → `"unknown"`, silent fallback), `.status.yaml` initialization from template via `sed`, and statusman integration (`start intake fab-new`, `logman.sh command` via `--log-args`)
3. Generate `intake.md` from the template (including Origin section), loading `fab/project/constitution.md` and `fab/project/config.yaml` as context

After generating the intake, `/fab-new` advances intake to `ready` — signaling the artifact exists and is open for `/fab-clarify` refinement. It then auto-activates the change via `fab change switch` (Step 10) and creates the matching git branch inline (Step 11). Branch creation applies the same 5-case logic as the standalone `/git-branch` skill: already active (no-op), target exists (checkout), on main/master (create), on local-only branch (rename), on pushed branch (create leaving old intact). The git step is non-fatal — if not in a git repo, it warns and skips; if a git operation fails, it reports the error and the change remains activated. For create-without-activate behavior, use `/fab-draft` instead.

#### Change Type Inference

After generating `intake.md`, `/fab-new` infers the `change_type` from the intake content using keyword matching (case-insensitive, first match wins): fix/bug/broken/regression → `fix`, refactor/restructure/consolidate/split/rename → `refactor`, docs/document/readme/guide → `docs`, test/spec/coverage → `test`, ci/pipeline/deploy/build → `ci`, chore/cleanup/maintenance/housekeeping → `chore`, otherwise → `feat`. The inferred type is written to `.status.yaml` via `statusman.sh set-change-type`.

#### Confidence

After generating `intake.md` and inferring the change type, `/fab-new` persists the confidence score by calling `fab score --stage intake <change>` in normal mode (not `--check-gate`). This writes the score to `.status.yaml`, making it visible to all consumers (`/fab-switch`, `/fab-status`, `fab change list`) without recomputation. As of j6cs `intake.md` is the **sole, authoritative** scoring source — there is no separate spec-stage score and no `confidence.indicative` flag (the flag was retired; `fab score` never writes it). Output format: `Confidence: {score} / 5.0 ({N} decisions)`.

#### Output

`/fab-new` produces `intake.md` as its primary artifact. It does not generate `plan.md` or any other downstream artifacts (the `spec.md` artifact no longer exists). The intake includes an **Origin** section recording how the change was initiated (description text, conversational vs. one-shot mode, key decisions from the conversation). After the intake, the output includes `Activated: {name}` (Step 10) and `Branch: {name} (created|created, leaving {old_branch} intact|checked out|renamed from {old_branch}|already active)` (Step 11). Step 7 (the score persist step) is titled "Confidence" — the former `indicative: true` persistence and spec-stage-overwrite behavior were removed in j6cs.

#### Context

Loads: config, constitution, `docs/memory/index.md` (to understand the existing memory landscape).

### `/fab-continue [<change-name>] [<stage>]`

`/fab-continue` advances to the next pipeline stage — planning, implementation, review, or hydrate — and either generates the artifact or executes the stage's behavior. When called with a stage argument, it resets to that stage. When called with a change-name argument, it targets that change instead of the active one in `.fab-status.yaml` (transient — `.fab-status.yaml` is not modified). Both arguments can coexist; stage names are disambiguated first (fixed set of 6: `intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`), all other arguments are treated as change-name overrides. The pipeline flows intake → apply → review → hydrate → ship → review-pr.

A passed `tasks` or `spec` stage argument SHALL error immediately. `tasks` → `"tasks" stage was removed — run "fab status <event> <change> apply" instead. plan.md is now generated at apply entry.` `spec` → `"spec" stage was removed — spec.md is now generated at apply entry. Use "apply".` No alias window — the migration ensures no in-flight `.status.yaml` carries a `tasks` or `spec` key after upgrade.

#### Normal Forward Flow (no argument)

1. Read `.status.yaml` to determine current stage and state
2. **Consolidated dispatch**:
   - **intake `ready`**: Finish intake (`done`) → start apply (`active`) → execute apply (apply's entry sub-step generates the unified `plan.md` with `## Requirements` + `## Tasks` + `## Acceptance`, then task execution begins)
   - **intake `active`** (backward compat for interrupted generations): Generate `intake.md`, advance to `ready`
   - For execution stages (apply, review, hydrate, ship, review-pr): dispatch to the stage's behavior
3. Load relevant template + context (including `fab/project/constitution.md` for principles)
4. Generate the artifact using the shared Plan Generation Procedure from `_generation.md` (apply entry)
5. Update `.status.yaml`

There is no per-stage scoring step in `/fab-continue` — scoring happens only at intake (via `/fab-new` and `/fab-clarify`). The former "Spec stage only — run `fab score`" step was removed in j6cs.

#### Reset Behavior (with stage argument)

When called as `/fab-continue <stage>` (e.g., `/fab-continue apply`):
1. Target stage can be any of the 6 stages: `intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`. `tasks` and `spec` are rejected with the strict-error messages above.
2. Reset `.status.yaml` progress: set target stage to `active`; mark all stages after target as `pending`
3. For planning resets (intake), regenerate the artifact in place (update, not recreate — preserve what's still valid)
4. Downstream artifacts are invalidated where applicable: e.g., intake reset → apply pending → `plan.md` regenerated on next apply entry
5. Advance the target stage to `ready` for planning resets (not `done` — preserves `/fab-clarify` opportunity)

`fab status reset apply` modifies `.status.yaml` state only — `plan.md` persists on disk per the existing artifact-file convention. The apply skill's plan-generation sub-step is skipped on the next `/fab-continue` (idempotent on `plan.md` presence) and execution resumes from the first unchecked task. To force plan regeneration (including a fresh `## Requirements`), the user MUST delete `plan.md` before re-running `/fab-continue`.

Reset is primarily used after review identifies issues upstream — including requirement-level rework, which now edits `plan.md`'s `## Requirements` section and re-runs apply rather than resetting to a removed spec stage.

#### Context (varies by target stage)

- **Intake**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`
- **Apply** (plan generation sub-step): above + `intake.md` as the requirement-generation input

### `/fab-fff [<change-name>]` (Full Pipeline)

`/fab-fff` runs the entire Fab pipeline in a single invocation: apply (which generates the unified `plan.md` at entry — `## Requirements` + `## Tasks` + `## Acceptance` — then executes tasks) → review → hydrate → ship → review-pr. Gated on a **single intake confidence gate** (flat 3.0). The standalone spec step, the spec gate, and BOTH `/fab-clarify [AUTO-MODE]` invocations were removed in j6cs. Autonomously reworks on review failure with bounded retry (3 cycles max, escalation after 2 consecutive fix-code failures). Accepts an optional change-name argument to target a specific change instead of the active one in `.fab-status.yaml`. Accepts `--force` to bypass the gate.

#### Minimum Prerequisite

`intake.md` must exist (apply pending or later). `/fab-fff` is callable from any stage at or after intake — it picks up from the current stage and runs forward, skipping stages already `done`. The intake gate must pass unless `--force` is used.

#### No Auto-Clarify

j6cs removed both auto-clarify checkpoints. The intake gate is the only "bounce" guard: a change whose intake scores below 3.0 cannot enter apply (non-`--force`), and the SRAD Critical Rule (Unresolved must be asked/bailed) applies at intake-time skills only. Inside apply, under-specified requirements are resolved inline as graded SRAD assumptions in `plan.md`'s `## Assumptions` — not via a clarify subagent. The pipeline is **resumable** — re-running `/fab-fff` skips stages already marked `done` and continues from the first incomplete stage.

#### Pipeline Flow

1. Resolve the active change (via `.fab-status.yaml` symlink); verify intake exists
2. Intake gate check (skip if `--force`)
3. Apply: plan generation sub-step writes the unified `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance` populated in one pass) → execute tasks
4. Validate implementation via review behavior — on failure, autonomously selects rework path (fix code, revise plan, revise requirements) and retries (max 3 cycles)
5. Hydrate into memory files
6. Ship (dispatch `/git-pr`)
7. Review-PR (dispatch `/git-pr-review`)

#### Autonomous Review Rework

On review failure, `/fab-fff` autonomously selects the rework path based on failure analysis (test failures → fix code, missing functionality → revise plan, requirement drift → revise `plan.md` `## Requirements`). The deepest tier "Revise requirements" replaced the former "Revise spec → reset to spec stage" (j6cs). Maximum 3 rework cycles. Escalation rule: after 2 consecutive "fix code" failures, the agent must escalate to "revise plan" or "revise requirements." After 3 failed cycles, bails with a per-cycle summary and suggests `/fab-continue` for manual rework.

#### When to Use

- Want the full pipeline from intake through PR review in one command
- Clear requirements upfront, want to reach completion quickly with safety nets
- Changes needing a quality gate — the single intake gate blocks too-ambiguous changes before apply

#### Context

Loads all planning context upfront: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`.

### `/fab-ff [<change-name>]` (Fast-Forward, Gated)

`/fab-ff` runs the pipeline from intake through hydrate. Gated on a single intake confidence gate identical to `/fab-fff` (flat 3.0). The spec gate and both auto-clarify checkpoints were removed in j6cs. Autonomous rework on review failure (3-cycle cap, escalation after 2 consecutive fix-code). Accepts an optional change-name argument. Accepts `--force` to bypass the gate.

#### Minimum Prerequisite

`intake.md` must exist. `/fab-ff` is callable from any stage at or after intake — it picks up from the current stage and runs forward through hydrate, skipping stages already `done`. The intake gate must pass unless `--force` is used.

#### Confidence Gate

`/fab-ff` has a single confidence gate identical to `/fab-fff`: the intake gate — confidence >= 3.0 (flat for all types) via `fab score --check-gate --stage intake`. The gate is skipped when `--force` is passed.

#### No Auto-Clarify

j6cs removed both auto-clarify checkpoints; the intake gate is the only guard. Inside apply, under-specified requirements become inline graded SRAD assumptions in `plan.md`'s `## Assumptions`.

#### Pipeline Flow

1. Resolve the active change (via `.fab-status.yaml` symlink); verify intake exists
2. Intake gate check (skip if `--force`)
3. Apply: plan generation sub-step writes the unified `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance`) → execute tasks
4. Validate implementation via review behavior — on failure, autonomous rework (3-cycle cap, escalation rule)
5. Hydrate into memory files

#### Autonomous Review Rework

On review failure, `/fab-ff` autonomously selects the rework path based on failure analysis (same behavior as `/fab-fff`). Maximum 3 rework cycles. Escalation rule: after 2 consecutive "fix code" failures, the agent must escalate. After 3 failed cycles, stops with a per-cycle summary.

#### Resumability

`/fab-ff` is resumable — re-invoking skips stages already marked `done` and continues from the first incomplete stage.

#### Confidence Recomputation

`/fab-ff` does NOT recompute the confidence score during execution. The gate check uses the persisted intake score from the last manual step (`/fab-new` or `/fab-clarify`).

#### When to Use

- Small, well-understood changes that don't need ship/review-pr
- Want to reach hydrate quickly with safety nets (intake gate + auto-rework)
- After raising confidence via `/fab-clarify` to meet the threshold

#### Context

Loads all planning context upfront: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`.

### `/fab-clarify [<change-name>]`

`/fab-clarify` deepens and refines the **intake** artifact without advancing. As of j6cs it is **intake-only**: the former `spec` and `plan` targets were removed (the spec stage is gone; under-specified requirements at apply become inline SRAD assumptions in `plan.md`, not clarify sessions). It operates in two modes depending on call context: **suggest mode** (user invocation) and **auto mode** (machine-readable result; no orchestrator currently invokes it). It is idempotent and non-advancing. Accepts an optional change-name argument to target a specific change instead of the active one in `.fab-status.yaml`. See [clarify.md](clarify.md) for the detailed dual-mode specification.

#### Suggest Mode (User Invocation)

When the user invokes `/fab-clarify` directly:

1. Read `.status.yaml` to determine current stage
2. Stage MUST be `intake` (`progress.intake` in `{active, ready, done}`). At apply or later, `/fab-clarify` STOPs with a pointer to `/fab-continue` for rework or editing `plan.md`'s `## Requirements` directly. A passed `spec`/`plan`/`tasks` argument is treated as a change-name (those targets no longer exist).
3. Load `intake.md` + relevant context
4. **Bulk confirm check** (Step 1.5): Parse the `## Assumptions` table. If `confident >= 3` AND `confident > tentative + unresolved`, display all Confident assumptions as a numbered list for conversational bulk response (confirm/change/explain). After resolution, proceed to step 5. See [clarify.md](clarify.md) for full details.
5. Perform a **stage-scoped taxonomy scan** for gaps, ambiguities, and `[NEEDS CLARIFICATION]` markers (categories vary by stage)
6. Present structured questions **one at a time** (max 5 per invocation), each with a recommendation and options table or suggested answer
7. **Immediately update the artifact** after each user answer (incremental, not batched)
8. User may terminate early with "done"/"good"/"no more"
9. Append audit trail under `## Clarifications > ### Session {date}` with `Q:` / `A:` entries
10. Display coverage summary (Resolved / Clear / Deferred / Outstanding)
11. Do NOT advance the stage

#### Auto Mode (Machine-Readable)

Auto mode (the `[AUTO-MODE]` prefix) operates on `intake.md` only and returns `{resolved: N, blocking: N, non_blocking: N}`. As of j6cs no orchestrator invokes it — `/fab-ff`/`/fab-fff` dropped their auto-clarify steps — but the mode is retained for future use:

1. Perform the intake taxonomy scan autonomously — no user interaction
2. Resolve gaps using available context; classify remaining gaps as blocking or non-blocking
3. Return the machine-readable result and recompute the intake score

#### Key Property

Calling `/fab-clarify` multiple times is safe — it refines the intake further each time. It never transitions to the next stage. Use `/fab-continue` when satisfied.

#### Context

- **Intake**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`

## Design Decisions

### SRAD Autonomy Framework
**Decision**: All planning skills use the SRAD framework (Signal Strength, Reversibility, Agent Competence, Disambiguation Type) to evaluate decision points and assign confidence grades (Certain, Confident, Tentative, Unresolved). All four grades are recorded in every Assumptions table with a required Scores column (`S:nn R:nn A:nn D:nn`). Unresolved rows include status context in Rationale. Each skill has a defined autonomy level and interruption budget. Dimensions are evaluated on a continuous 0–100 scale, aggregated via weighted mean (w_S=0.25, w_R=0.30, w_A=0.25, w_D=0.20), and mapped to grades via trapezoidal thresholds (Certain: 85–100, Confident: 60–84, Tentative: 30–59, Unresolved: 0–29). A Critical Rule override forces Unresolved when R < 25 AND A < 25.
**Why**: Replaces ad-hoc question selection with a principled, consistent framework. Ensures high-blast-radius decisions are always surfaced while low-value prompts are eliminated. The four-dimension scoring prevents both over-asking and silent high-risk assumptions. The R-biased weighting (0.30) encodes the Critical Rule's intent at the formula level.
**Rejected**: Ad-hoc question selection — inconsistent, no way to predict agent behavior. Full autonomy — too risky for Unresolved decisions with cascading consequences. Binary high/low dimension classification — lost nuance in the mid-range.
*Introduced by*: 260207-09sj-autonomy-framework; *Updated by*: 260212-f9m3-enhance-srad-fuzzy (fuzzy 0–100 dimensions, weighted mean aggregation, dynamic gate thresholds)

### Intake-First with SRAD-Based Questions
**Decision**: Every change starts with an intake. The agent applies SRAD scoring to identify up to 3 Unresolved decisions with the highest blast radius and asks those. All other decisions are assumed at their assessed confidence grade and surfaced in the Assumptions summary.
**Why**: Prevents question-paralysis while catching the decisions that actually matter. SRAD scoring replaces gut-feel question selection with a repeatable evaluation method.
**Rejected**: Unlimited clarification rounds — too many back-and-forth exchanges. Fixed 3-question cap without SRAD — may ask the wrong 3 questions.
*Source*: doc/fab-spec/TEMPLATES.md, doc/fab-spec/README.md; *Updated by*: 260207-09sj-autonomy-framework

### No Frontloaded Questions in Pipeline Skills
**Decision**: Neither `/fab-ff` nor `/fab-fff` frontloads questions. Both proceed directly into apply, relying on the single intake confidence gate to block changes that are too ambiguous.
**Why**: Frontloaded questions interrupted the autonomous pipeline flow. The intake gate provides a better signal — if the intake has too many unresolved decisions, the gate blocks and the user resolves via `/fab-clarify` before retrying. This is cleaner than a mid-pipeline Q&A round.
**Rejected**: Previous design had `/fab-fff` frontloading questions — this interrupted the autonomous flow unnecessarily.
*Source*: doc/fab-spec/SKILLS.md; *Updated by*: 260215-237b-DEV-1027-redefine-ff-fff-scope (moved from fab-ff to fab-fff); 260314-q5p9-redesign-ff-fff-scopes (dropped frontloaded questions entirely); 260601-j6cs-merge-spec-into-apply (single intake gate after spec stage removal)

### Clarify is Non-Advancing
**Decision**: `/fab-clarify` never transitions to the next stage. It refines in place.
**Why**: Separates the concerns of "deepen the current work" from "move forward." The user explicitly chooses when to advance via `/fab-continue`.
**Rejected**: Auto-advancing after clarification — unclear when the user considers the artifact ready.
*Source*: doc/fab-spec/SKILLS.md

### Clarify Mode Selection by Call Context
**Decision**: `/fab-clarify` mode is determined by the `[AUTO-MODE]` prefix defined in the Skill Invocation Protocol (`_preamble.md`). When the prefix is present (e.g., `/fab-ff` invoking internally), `/fab-clarify` enters auto mode. When absent (user invocation), it enters suggest mode. No `--suggest`/`--auto` flags.
**Why**: Avoids a confusing flag pair with no clear use case for user-invoked auto mode. The explicit prefix protocol makes the contract testable rather than relying on implicit call-context interpretation.
**Rejected**: Flag-based mode selection — adds complexity, no user scenario requires it. Implicit call-context detection — unreliable, not testable.
*Introduced by*: 260207-m3qf-clarify-dual-modes; *Updated by*: 260210-nan4-define-auto-mode-signaling (explicit `[AUTO-MODE]` protocol)

### Pipeline Skills Drop Auto-Clarify (j6cs)
**Decision**: Neither `/fab-ff` nor `/fab-fff` invokes `/fab-clarify` automatically. Both auto-clarify checkpoints (the former post-spec and on-plan invocations) were removed when the spec stage was merged into apply. The single intake confidence gate is the only "bounce" guard.
**Why**: With one manual stage (intake), all human judgment is frontloaded there and gated once at flat 3.0. There is no mid-pipeline stage boundary left for an auto-clarify to sit on. Inside apply, under-specified requirements are resolved inline as graded SRAD assumptions in `plan.md`'s `## Assumptions` — the apply agent doesn't need a separate clarify subagent. The independent assumption re-grade the old spec stage performed is accepted as lost, compensated by the flat-3.0 gate (≥ every old gate), the ~1% spec-rework loom evidence, and requirement-correctness still being caught at review.
**Rejected**: Keeping an apply-entry auto-clarify (re-adds the ceremony the merge removes). A runtime "apply detects Unresolved → reset to intake" mechanism (unbuilt, redundant with the existing gate). 
**History (pre-j6cs)**: Both skills previously interleaved auto-clarify between planning stages (`spec → auto-clarify spec → apply plan-gen → optional auto-clarify plan → task execution`), bailing on blocking issues.
*Introduced by*: 260207-m3qf-clarify-dual-modes; *Updated by*: 260208-k3m7-add-fab-fff; 260215-237b-DEV-1027-redefine-ff-fff-scope; 260314-q5p9-redesign-ff-fff-scopes; 260423-qszh-merge-tasks-checklist (auto-clarify tasks → auto-clarify plan); 260601-j6cs-merge-spec-into-apply (both auto-clarify checkpoints removed)

### /fab-new as Single Adaptive Entry Point
**Decision**: Consolidate `/fab-new` and `/fab-discuss` into a single `/fab-new` that adapts via SRAD scoring.
**Why**: Three overlapping entry paths created confusion.
**Rejected**: Keeping both skills with clearer differentiation.
*Introduced by*: 260212-v5p2-simplify-stages-entry-paths

### Stage Rename: "brief" → "intake"
**Decision**: The first pipeline stage is named `intake` (not `brief`). The artifact is `intake.md`. The Intake Generation Procedure lives in `_generation.md` (which, after j6cs, contains only this and the unified Plan Generation Procedure), with an explicit generation rule emphasizing the intake is a state transfer document.
**Why**: "Brief" in English means short/concise, which triggered summarization instincts in LLMs — agents consistently produced thin briefs despite template instructions. "Intake" signals thorough initial collection in professional contexts (legal, medical, project management). The generation rule in `_generation.md` provides defense in depth alongside the name change.
**Rejected**: `handoff` — could describe any stage boundary. `charter` — too formal. Generation rule only without rename — doesn't fix the misleading name.
*Introduced by*: 260215-v4n7-DEV-1025-rename-brief-to-intake

### Shared Generation Partial
**Decision**: Artifact generation logic lives in a single shared `_generation.md` partial. After j6cs, the partial defines two procedures: the **Intake Generation Procedure** and the unified **Plan Generation Procedure** (invoked by apply at entry, emitting `## Requirements` + `## Tasks` + `## Acceptance` in one walk). The standalone Spec Generation Procedure was deleted — requirement generation folded into plan generation. Each consumer retains its own orchestration logic.
**Why**: Generation steps were nearly identical across skills, requiring every fix or behavior change to be applied in two places. Centralizing eliminates drift and makes generation behavior authoritative in one location. With qszh the Tasks + Checklist procedures collapsed into one Plan Generation Procedure; with j6cs the Spec Generation Procedure folded into it too, so a single pass over the intake-derived design emits requirements, tasks, and acceptance — the strongest possible alignment guarantee (single skill call, single context window).
**Rejected**: Keeping inline duplication — inevitable drift between the copies. Keeping a separate Spec Generation Procedure and `spec.md` — reintroduces the seam the merge removes and leaves an unread file (nothing reads `spec.md` programmatically once the gate moves to intake).
*Introduced by*: 260210-wpay-extract-shared-generation-logic; *Updated by*: 260423-qszh-merge-tasks-checklist (Tasks + Checklist procedures → unified Plan Generation Procedure); 260601-j6cs-merge-spec-into-apply (Spec Generation Procedure folded into Plan Generation; `## Requirements` co-generated at apply entry)

### Plan Generation Lives at Apply Entry, Not in a Separate Stage (qszh)
**Decision**: The `tasks` stage is removed from the pipeline. Plan generation (writing `plan.md` with `## Tasks` + `## Acceptance`) is an entry sub-step of the apply skill, not a stage gate. Pipeline goes from 8 stages (intake → spec → tasks → apply → review → hydrate → ship → review-pr) to 7 stages (intake → spec → apply → review → hydrate → ship → review-pr). `progress.tasks` is dropped from `.status.yaml` entirely — no rename to `progress.plan`, since with no separate stage there is no key to populate.
**Why**: The `tasks` stage was a no-decision gate. `/fab-continue` advanced spec → tasks → apply back-to-back; users never stopped at tasks. Every change paid the wall-time, token, and `.status.yaml` cost of a transition that had no decision content. Folding generation into apply makes drift between Tasks and Acceptance mechanically impossible (single skill call, single context window, single LLM, single file). The qszh collapse stopped at `tasks` and preserved the spec stage; j6cs later folded the spec stage in too (see "Spec Stage Merged into Apply" below), so apply entry now co-generates `## Requirements` alongside `## Tasks` + `## Acceptance`.
**Rejected**: (a) Keep two files, add cross-check step — adds ceremony, not less. (b) Drop checklist, review reads tasks directly — loses the imperative-vs-declarative framing review depends on. (c) Merge artifacts only, keep `tasks` stage — fixes drift but keeps the no-decision gate; xvaz workaround remains relevant. (d) Drop both spec AND tasks — loses `/fab-clarify spec`, breaks per-type spec gate, weakens review. (e) Rename `tasks` → `plan` stage — same gate, new name. (f) Rename `apply` → `execute`/`implement` — semantic gain doesn't justify migration churn across state table, `.status.yaml`, all skills, muscle memory.
*Introduced by*: 260423-qszh-merge-tasks-checklist; *Supersedes*: the proposal in `260423-xvaz-skip-tasks-simple-types` (per-type skip policy for the tasks stage) — that draft becomes obsolete by construction once qszh ships, since there is no separate stage to skip. Simple changes naturally produce a tiny `plan.md` and execute in seconds; no skip policy needed. The xvaz folder will be archived by a separate user-initiated `/fab-archive 260423-xvaz...` action.

### Spec Stage Merged into Apply; `plan.md` Carries `## Requirements` (j6cs)
**Decision**: The `spec` stage is removed from the pipeline (7 → 6 stages: intake → apply → review → hydrate → ship → review-pr). The requirement discipline that lived in a separate `spec.md` artifact (RFC-2119 + GIVEN/WHEN/THEN) is absorbed into `plan.md` as a `## Requirements` section, co-generated at apply entry alongside `## Tasks` + `## Acceptance`. The `spec.md` template is removed; `spec.md` no longer exists as an artifact. The canonical artifact set becomes `intake.md → plan.md → code`. Any `fab status` event targeting `spec` hard-errors (mirroring the `tasks` branch). SRAD confidence scoring is frontloaded to intake as the **single** confidence gate (flat 3.0 for all types); the per-type spec gate and the `confidence.indicative` flag are retired.
**Why**: Empirical loom evidence (spec median ~2 min, ~1% rework, 32% of clarify) showed the spec stage was a near-pass-through. Once the confidence gate moves to intake, nothing reads `spec.md` programmatically (verified: score/hook/artifact-matcher/git-pr all removed or repointed) — a separate file would be generated, never machine-read, dead weight. One-pass co-generation of requirements + tasks + acceptance is the strongest alignment guarantee. The intake gate IS the bounce-back valve: a failing intake never reaches `done`, so gate-checking orchestrators cannot enter apply — no new runtime "reset to intake" mechanism is needed.
**Rejected**: Keeping `spec.md` as a separate hidden apply-entry artifact (reintroduces the seam, leaves an unread file). Per-type single gate at 2.0/3.0 (would relax the entry bar for 5 of 7 types — flat 3.0 keeps every type ≥ both old gates). A runtime apply-side "detect Unresolved → reset to intake" mechanism (unbuilt, redundant with the gate). Adding an independent assumption re-grade at apply entry (re-adds ceremony; deferred unless review evidence warrants).
*Introduced by*: 260601-j6cs-merge-spec-into-apply

### Strict-Error Stance for Legacy `tasks` References (qszh)
**Decision**: All `tasks` stage references and the legacy `set-checklist` CLI command error immediately with a helpful pointer message — no alias window, no phased deprecation. `fab status start|advance|finish|reset|skip|fail <change> tasks` returns exit 1 with `"tasks" stage was removed — run ... apply instead. plan.md is now generated at apply entry.` `fab status set-checklist` returns exit 1 with `"set-checklist" is now "set-acceptance" — run fab status set-acceptance instead.` `/fab-clarify tasks` errors with a similar pointer.
**Why**: The 1.8.0-to-1.9.0 migration rewrites every in-flight `.status.yaml` so no live change carries a `tasks` key after upgrade. With no live `tasks` state to support, an alias adds maintenance burden for zero user benefit. Strict errors with pointer messages are self-documenting and steer users toward the new workflow immediately.
**Rejected**: Phased deprecation (alias for one release, error in next) — no in-flight `.status.yaml` carries `tasks` after migration, so phasing buys nothing. Silent renaming — leaves users uncertain whether a command did what they expected.
*Introduced by*: 260423-qszh-merge-tasks-checklist

### Unified Command: `/fab-continue` Absorbs Execution Stages
**Decision**: `/fab-continue` handles all 6 pipeline stages (intake → apply → review → hydrate → ship → review-pr). Apply, review, hydrate, ship, and review-pr behaviors are described as dedicated sections within `fab-continue.md`, not extracted into a shared partial. `/fab-archive` exists as a standalone housekeeping skill (not a pipeline stage) for post-hydrate cleanup. The apply behavior includes a Plan Generation sub-step at entry (writes the unified `plan.md`); see [execution-skills.md](execution-skills.md).
**Why**: Reduces developer command surface to 2 (`/fab-continue` + `/fab-clarify`). Execution stages are orchestration-heavy with distinct flows (plan generation + task execution, validation with rework, memory hydration) — inlining keeps each stage's behavior in one readable location.
**Rejected**: Keeping standalone `/fab-apply`, `/fab-review` — command fragmentation. Extracting to `_execution.md` partial — low reuse value since only fab-continue calls these. Splitting plan generation into a `/fab-plan` skill — adds a command surface for what is mechanically a single autonomous step at apply entry.
*Introduced by*: 260212-a4bd-unify-fab-continue; *Updated by*: 260303-he6t-extend-pipeline-through-pr (added ship + review-pr); 260423-qszh-merge-tasks-checklist (dropped tasks stage; plan generation folded into apply entry — 7 stages); 260601-j6cs-merge-spec-into-apply (dropped spec stage — 6 stages)

### `/fab-ff` and `/fab-fff` Keep Behavioral Descriptions
**Decision**: `/fab-ff` and `/fab-fff` describe execution behavior inline within their own orchestration context, rather than literally invoking `/fab-continue` as a sub-skill.
**Why**: These skills have fundamentally different orchestration: frontloaded questions, auto-clarify interleaving, bail behavior, resumability across all stages. Literal sub-skill invocation would add complexity (nested preflight checks, status conflicts) without benefit.
**Rejected**: Literal `/fab-continue` invocation from fab-ff/fff — orchestration mismatch, nested state management issues.
*Introduced by*: 260212-a4bd-unify-fab-continue

### Scope Differentiation: fab-fff (Full Pipeline) vs fab-ff (Fast-Forward)
**Decision**: The difference between `/fab-ff` and `/fab-fff` is scope only. `/fab-ff` runs intake → hydrate; `/fab-fff` extends through ship → review-pr. Both have identical confidence gates (intake + spec), identical auto-clarify, identical autonomous rework (3-cycle cap, escalation rule), and accept `--force` to bypass gates. No frontloaded questions in either skill.
**Why**: The naming intuition: `ff` (fast-forward) = "get me to hydrate quickly." `fff` (fast-forward-further) = "go all the way through PR review." Scope is the only axis of differentiation — behavior (gates, rework, auto-clarify) is identical. This simplifies the mental model: choose ff or fff based on how far you want to go, not based on behavioral differences.
**Rejected**: Previous design differentiated on behavior (gates, frontloaded questions, rework style) — too many axes of variation, confusing mental model.
*Introduced by*: 260215-237b-DEV-1027-redefine-ff-fff-scope; *Updated by*: 260216-knmw-DEV-1030-swap-ff-fff-review-rework (swapped review failure behavior); 260314-q5p9-redesign-ff-fff-scopes (scope-only differentiation, identical gates on both, no frontloaded questions)

### Reset via `/fab-continue <stage>`
**Decision**: Reset to any pipeline stage by passing the stage name as an argument to `/fab-continue`. For the intake planning stage, the artifact is invalidated and regenerated. For execution stages, the stage behavior is re-run without resetting task checkboxes. `tasks` and `spec` are rejected with strict-error pointers to `apply` — both stages were removed (qszh and j6cs respectively).
**Why**: Provides a clean re-entry point after review identifies upstream issues. Reuses the existing skill rather than adding a separate `/fab-reset` command. Covers all 6 stages (intake, apply, review, hydrate, ship, review-pr). `fab status reset apply` preserves `plan.md` on disk; the apply entry sub-step skips regeneration when the file exists, so users who want a fresh plan (including a fresh `## Requirements`) must delete `plan.md` before re-running.
**Rejected**: Separate reset skill — unnecessary proliferation of skills for a rare operation. Auto-deleting `plan.md` on apply reset — violates the existing artifact-file convention (reset modifies `.status.yaml` state only; artifact files persist) and Constitution III idempotency.
*Source*: doc/fab-spec/SKILLS.md; *Updated by*: 260212-a4bd-unify-fab-continue (extended to all 6 stages); 260303-he6t-extend-pipeline-through-pr (extended to ship + review-pr — 8 stages); 260423-qszh-merge-tasks-checklist (dropped tasks stage — 7 stages; documented `reset apply` plan.md preservation); 260601-j6cs-merge-spec-into-apply (dropped spec stage — 6 stages; `spec` reset target rejected)

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260601-j6cs-merge-spec-into-apply | 2026-06-01 | Dropped `spec` stage from the pipeline (7 → 6 stages). Planning is now **intake only**: rewrote Overview ("the single planning stage: intake"), and folded the standalone Spec Generation Procedure into the unified Plan Generation Procedure (one walk → `## Requirements` + `## Tasks` + `## Acceptance`, reads `intake.md`, REQUIRED trace annotations, no `[NEEDS CLARIFICATION]`, legacy `spec.md` ingestion path). `/fab-new`: renamed "Indicative Confidence" → "Confidence"; dropped `indicative: true` persistence and spec-stage-overwrite; output `Confidence: {score} / 5.0`. `/fab-continue`: dispatch table dropped `spec` rows (intake-ready → starts apply); removed the spec-only scoring step; `spec`/`tasks` reset targets rejected with strict-error pointers; reset/context sections updated. `/fab-fff` + `/fab-ff`: removed the standalone spec step, the spec gate, and BOTH auto-clarify checkpoints; single intake gate at flat 3.0; "Revise spec" rework tier → "Revise requirements". `/fab-clarify`: intake-only summary (spec/plan targets removed; auto-mode retained but no orchestrator calls it). Design decisions: added "Spec Stage Merged into Apply; plan.md Carries ## Requirements"; renamed/updated "Pipeline Skills Drop Auto-Clarify", "Shared Generation Partial", "No Frontloaded Questions", "Unified Command" (6 stages), "Reset via /fab-continue" (6 stages). |
| 260423-qszh-merge-tasks-checklist | 2026-05-06 | Dropped `tasks` stage from the pipeline (8 → 7 stages). Updated Overview to "first two planning stages (intake, spec)" and noted that `plan.md` generation lives at apply entry, not in a planning skill. Replaced Tasks Generation Procedure + Checklist Generation Procedure with a unified Plan Generation Procedure in the Shared Generation Partial section. `/fab-continue` description: 7-stage pipeline; legacy `tasks` argument errors with strict-error pointer to `apply` or `/fab-clarify spec`; `fab status reset apply` preserves `plan.md` on disk (delete to force regen). `/fab-fff` and `/fab-ff` pipeline flow rewritten: spec → apply (plan-gen at entry) → execute → review → hydrate (+ ship/review-pr for fff). Auto-clarify checkpoint moves from "after tasks" to "after plan generation at apply entry" (`auto-clarify plan`). `/fab-clarify` accepts `intake`, `spec`, and `plan` (post-apply-entry); `tasks` errors with pointer. PostToolUse hook bookkeeping uses heading-bounded section parse for `plan.md`. Added two design decisions: "Plan Generation Lives at Apply Entry, Not in a Separate Stage" (with rationale, rejected alternatives, and a note that `260423-xvaz-skip-tasks-simple-types` becomes obsolete and should be user-archived) and "Strict-Error Stance for Legacy tasks References". Updated Shared Generation Partial, Pipeline Skills Interleave Auto-Clarify, Unified Command, and Reset design decisions. |
| 260405-hgv7-fab-new-include-git-branch | 2026-04-05 | `/fab-new` now auto-activates changes (Step 10: `fab change switch`) and creates the matching git branch inline (Step 11). Branch creation uses 5-case logic (already active, target exists, on main/master, local-only branch rename, pushed branch). Git step is non-fatal. Updated Change Initialization and Output sections. Removed stale "never activates" text. |
| 260402-gnx5-relocate-kit-to-system-cache | 2026-04-02 | Updated shared generation partial reference: `$(fab kit-path)/skills/_generation.md` now resolves from system cache. Template loading in spec/tasks/checklist generation procedures uses `$(fab kit-path)/templates/` instead of `fab/.kit/templates/`. Hook-backed bookkeeping references inline `fab hook <subcommand>` commands. |
| 260314-q5p9-redesign-ff-fff-scopes | 2026-03-14 | Redesigned `/fab-ff` and `/fab-fff` scope differentiation. `/fab-ff` now runs intake → hydrate (was: spec → review-pr). `/fab-fff` now runs intake → review-pr with identical confidence gates (was: no gates, frontloaded questions). Both have identical behavior (gates, auto-clarify, autonomous rework). Both accept `--force` to bypass gates. Frontloaded questions removed from `/fab-fff`. Updated Overview, requirements, pipeline flows, design decisions. |
| 260306-6bba-redesign-hooks-strategy | 2026-03-06 | Added hook-backed bookkeeping note: PostToolUse hook (`on-artifact-write.sh`) supplements skill-instructed bookkeeping as a reliability layer. Skills keep instructions unchanged for agent-agnostic portability; hooks catch what the agent forgets. All commands idempotent. |
| 260305-8ooz-persist-indicative-confidence | 2026-03-05 | `/fab-new` Step 7 now persists indicative confidence via `calc-score.sh --stage intake` (normal mode) instead of inline display-only computation. Score written to `.status.yaml` with `indicative: true`. `_preamble.md` Confidence Scoring section updated to document indicative flag, persistence, and uniform consumer reads. |
| 260303-6b7c-update-underscore-skill-references | 2026-03-04 | Standardized top-of-file `_preamble.md` references in all skill files — removed `./` prefix from `./$(fab kit-path)/skills/_preamble.md`, now `$(fab kit-path)/skills/_preamble.md`. Updated `_preamble.md` self-reference (line 12). Inline shorthand references (`_preamble.md` §2, `_generation.md`) unchanged. |
| 260302-c7is-fab-clarify-bulk-confirm | 2026-03-02 | Added bulk confirm mode (Step 1.5) to `/fab-clarify` suggest mode — detects Confident-dominant confidence drag, presents numbered list for conversational bulk confirmation. Updated suggest mode steps (now 11 steps, bulk confirm at step 4). Documented in `_preamble.md` Confidence Scoring section. |
| 260227-ijql-streamline-planning-dispatch | 2026-02-27 | Consolidated planning dispatch: `/fab-new` leaves intake as `ready` (Step 9 added). `/fab-continue` finishes previous `ready` stage + generates next artifact + advances to `ready` in one invocation. Single-dispatch rule removed. Reset flow uses `advance` (not `finish`) to preserve `/fab-clarify` checkpoint. |
| 260226-6boq-event-driven-statusman | 2026-02-26 | Replaced `set-state`/`transition` references with event commands (`start`, `advance`, `finish`, `reset`, `fail`). Driver parameter now optional (skills always pass it). Updated `changeman.sh` integration (`start intake fab-new`). |
| 260226-i9av-add-ready-state-to-stages | 2026-02-26 | `/fab-continue` gains state-based dispatch: `active` → generate artifact, `ready` → advance to next stage. `/fab-ff` redefined: starts from intake (was: spec-only), 3 safety gates (intake indicative >= 3.0, spec per-type threshold, review 3-cycle stop). `/fab-fff` unchanged except contrast text. `/fab-clarify` accepts `ready` state. `_preamble.md` State Table adds `/fab-ff` to intake row, state derivation includes `ready`. Dual gate thresholds documented (intake fixed 3.0, spec dynamic per-type). |
| 260226-tnr8-coverage-scoring-change-types | 2026-02-26 | `/fab-new` gains change type inference (keyword heuristic → `statusman.sh set-change-type`) and indicative confidence display (coverage-weighted formula, display-only, not persisted). Coverage-weighted confidence formula added to `_preamble.md` §Confidence Scoring. Gate thresholds updated from 4-type (`bugfix`/`feature`/`refactor`/`architecture`) to 7-type taxonomy (`feat`/`fix`/`refactor`/`docs`/`test`/`ci`/`chore`). |
| 260221-5tj7-rename-context-to-preamble | 2026-02-21 | Renamed shared skill preamble from `_context.md` to `_preamble.md`. Updated all references in Shared Generation Partial section, SRAD design decision, and mode selection references. |
| 260216-7ltw-DEV-1038-standardize-state-keyed-suggestions | 2026-02-16 | Replaced skill-keyed suggestion lookup with state-keyed table in `_preamble.md`. Removed `--switch` flag and natural language switching detection from `/fab-new` — change is never activated by `/fab-new`. All skills now derive `Next:` lines from canonical state table. Extended `/fab-clarify` stage guard to include `intake`. |
| 260216-knmw-DEV-1030-swap-ff-fff-review-rework | 2026-02-16 | Swapped review failure behavior: `/fab-ff` now presents interactive rework menu (3 options, no retry cap); `/fab-fff` now uses autonomous rework (agent selects path, 3-cycle retry cap, escalation after 2 consecutive fix-code). Updated overview paragraphs, pipeline flow steps, rework sections, and Scope Differentiation design decision. |
| 260215-237b-DEV-1027-redefine-ff-fff-scope | 2026-02-16 | Redefined `/fab-ff` and `/fab-fff` scope. `/fab-fff` is now the full pipeline command (intake → hydrate, no gate, frontloaded questions, interactive rework). `/fab-ff` is now the fast-forward-from-spec command (spec → hydrate, confidence-gated, no frontloaded questions, bail on failure). Updated overview, requirement sections, and design decisions. Added Scope Differentiation design decision. |
| 260215-9yjx-DEV-1022-create-changeman-script | 2026-02-15 | Refactored `/fab-new`: "Folder Name Generation" → "Slug Generation and Change Creation" (delegated to `lib/changeman.sh`). Change Initialization steps consolidated — steps 1-2 of old init (mkdir, .status.yaml, created_by, statusman calls) replaced by single `changeman.sh new` call. Skill now focuses on AI tasks (slug generation, gap analysis, intake writing). Error table simplified. |
| 260215-v4n7-DEV-1025-rename-brief-to-intake | 2026-02-15 | Renamed `brief` → `intake` throughout. Added Intake Generation Procedure to `_generation.md`. Updated `/fab-new` to reference procedure instead of inlining. Renamed "Brief-First" design decision to "Intake-First" |
| 260215-w3n8-naming-linear-id-drop-conventions | 2026-02-15 | Updated `/fab-new` folder name generation format to `{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}` with optional uppercase Linear issue ID |
| 260214-m3w7-formalize-assumptions-scoring | 2026-02-14 | Formalized Assumptions tables: all four SRAD grades recorded (not just Confident/Tentative), Scores column required, Unresolved rows include status context. `calc-score.sh` reads only spec.md (not intake+spec), fixed AWK cols[6], removed has_scores detection and Certain carry-forward, parses Unresolved grade. Spec generation reads intake assumptions as starting point (confirm/upgrade/override). Templates include formalized `## Assumptions` sections. Summary line uses 4-grade format. |
| 260214-r7k3-statusman-yq-metrics | 2026-02-14 | All skill prompts now call `log-command` after preflight and pass `driver` on all `set-state`/`transition` calls. `/fab-new` calls `set-state intake active fab-new`. `/fab-clarify` calls `log-command` after preflight. `/fab-ff` and `/fab-fff` pass driver on all transitions. Added shared generation partial note about `log-command` and driver conventions |
| 260212-f9m3-enhance-srad-fuzzy | 2026-02-14 | SRAD framework updated to fuzzy 0–100 dimension scoring with weighted mean aggregation; `/fab-fff` confidence gate now uses dynamic per-type thresholds (bugfix=2.0, feature/refactor=3.0, architecture=4.0) via `calc-score.sh --check-gate`; optional Scores column in Assumptions tables for per-dimension data |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_statusman.sh` → `lib/statusman.sh` and `_calc-score.sh` → `lib/calc-score.sh` in all references; updated shared generation partial `lib/statusman.sh set-checklist` references |
| 260214-w3r8-statusman-write-api | 2026-02-14 | Skill prompts (`fab-continue.md`, `fab-ff.md`, `fab-fff.md`, `_generation.md`) now reference `lib/statusman.sh` CLI commands for all `.status.yaml` mutations instead of ad-hoc editing |
| 260214-lptw-score-init-display | 2026-02-14 | Changed `/fab-fff` confidence gate and output header display format from `{score}` to `{score} of 5.0`. Updated `_preamble.md` template description from "score 5.0" to "score 0.0". |
| 260213-w8p3-extract-fab-score | 2026-02-14 | Extracted confidence scoring into `lib/calc-score.sh` script. Removed inline scoring from `/fab-new` (Step 7 deleted), `/fab-continue` (Step 3b replaced with script invocation at spec stage only), `/fab-clarify` (Step 7 replaced with script invocation in suggest mode). Updated `/fab-fff` confidence recomputation note. |
| 260213-jc0u-split-archive-hydrate | 2026-02-13 | Updated all pipeline references from `archive` to `hydrate` as terminal stage. Updated `/fab-continue` and `/fab-ff`/`/fab-fff` descriptions. Updated unified command design decision to reflect `/fab-archive` as standalone housekeeping skill. |
| 260213-w4k9-explicit-change-targeting | 2026-02-13 | All workflow skills (`/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-clarify`) now accept optional `[change-name]` argument for targeting non-active changes. `/fab-continue` disambiguates stage names vs change names. Preflight handles matching centrally |
| 260212-r7xp-fix-fab-new-intake-stage | 2026-02-12 | `/fab-new` no longer marks intake complete — removed Step 8 ("Mark Intake Complete"), renumbered Step 9 → Step 8. Intake stays `active` after `/fab-new`; `/fab-continue` handles the intake → spec transition. Updated Change Initialization list and `_preamble.md` Next Steps table |
| 260212-a4bd-unify-fab-continue | 2026-02-12 | Unified `/fab-apply`, `/fab-review`, `/fab-archive` into `/fab-continue`. Updated stage guard, reset behavior, and cross-references to reflect unified command |
| 260212-ipoe-checklist-folder-location | 2026-02-12 | Updated checklist generation and validation paths from `checklists/quality.md` to `checklist.md` in `/fab-continue`, `/fab-ff`, and shared generation partial |
| 260212-bk1n-rework-fab-ff-archive | 2026-02-12 | Extended `/fab-ff` from planning-only to full pipeline (planning → apply → review → archive). Updated `/fab-fff` description and comparison table to reflect new differentiation. `/fab-ff` now offers interactive rework on review failure; `/fab-fff` remains fully autonomous with confidence gate |
| 260212-29xv-scoring-formula | 2026-02-12 | Increased Confident penalty from 0.1 to 0.3 in confidence formula; `/fab-clarify` now reclassifies resolved assumptions (Tentative/Confident → Certain) so scores increase after clarification |
| 260212-k7m3-fix-consistency-drift | 2026-02-12 | Clarified confidence score template default phrasing ("zero counts and score 5.0" instead of "all zeros") |
| 260212-0r8e-fix-created-by-github | 2026-02-12 | `/fab-new` now uses `gh api user --jq .login` as primary source for `created_by`, with `git config user.name` as fallback |
| — | 2026-02-12 | Reversed `/fab-new` default behavior: no longer auto-switches to new changes. Replaced `--no-switch` with `--switch` flag, added natural language switching detection. Default output now suggests `/fab-switch {name}` command |
| 260212-r7k3-add-no-switch-flag | 2026-02-12 | Added `--no-switch` flag to `/fab-new` — skips activation and branch integration when batching change captures |
| 260212-v5p2-simplify-stages-entry-paths | 2026-02-12 | Removed /fab-discuss section, rewrote /fab-new for adaptive SRAD-driven behavior with gap analysis and conversational mode |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | 6-stage pipeline (intake → spec → tasks), removed plan stage, /fab-discuss dual output, /fab-ff generates spec → tasks directly |
| 260211-endg-add-created-by-field | 2026-02-11 | `/fab-new` and `/fab-discuss` now populate `created_by` in `.status.yaml` from `git config user.name` at change creation |
| 260210-wpay-extract-shared-generation-logic | 2026-02-10 | Extracted shared generation logic (spec, tasks, checklist) into `_generation.md` partial; both `/fab-continue` and `/fab-ff` now reference it |
| 260210-nan4-define-auto-mode-signaling | 2026-02-10 | Defined explicit `[AUTO-MODE]` prefix protocol for skill-to-skill invocation in `_preamble.md`; updated `/fab-ff` auto-clarify invocations and "Clarify Mode Selection" design decision |
| 260210-0p4e-fix-stage-guard-progress-check | 2026-02-10 | `/fab-continue` stage guard now checks `progress.{stage}` value to distinguish done/active/pending states, allowing resumption of interrupted stage generations |
| 260210-zr1f-discuss-auto-activate-when-no-current | 2026-02-10 | `/fab-discuss` conditionally offers activation when no active change; updated proposal output, key differences table |
| 260209-r4w8-archive-index-longer-slugs | 2026-02-09 | Expanded slug word count from 2-4 to 2-6 words in `/fab-new` folder name generation |
| 260208-q8v3-branch-to-switch | 2026-02-09 | Moved branch integration from `/fab-new` to `/fab-switch`, removed `--branch` flag from `/fab-new`, `/fab-new` now calls `/fab-switch` internally |
| 260208-lgd7-fab-discuss-command | 2026-02-08 | Added `/fab-discuss` conversational intake skill, `/fab-new` confidence scoring, context-driven mode selection design decisions |
| 260208-k3m7-add-fab-fff | 2026-02-08 | Added `/fab-fff` full pipeline skill, confidence recomputation in `/fab-continue`, removed `/fab-ff --auto` mode, updated design decisions |
| 260207-09sj-autonomy-framework | 2026-02-08 | Added SRAD autonomy framework, confidence grades, assumptions summaries, branch auto-create on main, soft gate on fab-apply |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260207-m3qf-clarify-dual-modes | 2026-02-07 | Updated `/fab-clarify` to dual-mode (suggest + auto), `/fab-ff` with interleaved auto-clarify and `--auto` flag |
| — | 2026-02-07 | Generated from doc/fab-spec/ (README.md, SKILLS.md, TEMPLATES.md) |
