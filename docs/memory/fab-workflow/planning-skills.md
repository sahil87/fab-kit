# Planning Skills

**Domain**: fab-workflow

## Overview

The planning skills (`/fab-new`, `/fab-continue`, `/fab-clarify`) handle the first three stages of the 6-stage Fab pipeline: intake, spec, and tasks. They produce the artifacts that define *what* changes and *how*, before any code is written.

`/fab-fff` and `/fab-ff` are also documented here because their planning behavior originated as planning skills. `/fab-fff` is the **full-pipeline command** (intake → hydrate, no confidence gate, frontloaded questions, interactive rework). `/fab-ff` is the **fast-forward-from-spec command** (spec → hydrate, confidence-gated, no frontloaded questions, bail on failure). See sections below for details.

## Shared Generation Partial

The artifact generation logic (spec, tasks, checklist) is defined in a single shared partial: `fab/.kit/skills/_generation.md`. Both `/fab-continue` and `/fab-ff` reference this partial for the mechanics of producing each artifact, rather than inlining the generation steps.

The partial contains three procedures:
- **Spec Generation Procedure** — template loading, metadata, RFC 2119 requirements, GIVEN/WHEN/THEN scenarios, Assumptions section (reads intake assumptions as starting point, confirms/upgrades/overrides each)
- **Tasks Generation Procedure** — template loading, metadata, phased task breakdown, task format, execution order
- **Checklist Generation Procedure** — template loading, category population, sequential CHK IDs, `.status.yaml` updates via `lib/stageman.sh set-checklist` CLI commands

All pipeline skills (`/fab-new`, `/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-clarify`) call `lib/stageman.sh log-command` after preflight to record the invocation in `.history.jsonl`. All stage transitions pass a `driver` parameter identifying the invoking skill (e.g., `fab-continue`, `fab-ff`).

Each skill retains its own orchestration logic (stage guards, question handling, auto-clarify, resumability). Only the generation mechanics are shared.

## Requirements

### `/fab-new <description>`

`/fab-new` starts a new change from a natural language description. It is adaptive: clear inputs get a quick intake, vague inputs trigger conversational exploration. It creates the change folder, initializes status tracking, and generates an intake (with Origin section). The change is NOT activated — no write to `fab/current`, no branch integration. The user activates via `/fab-switch` after creation. Output is always a single artifact: `intake.md`.

#### Slug Generation and Change Creation

The agent generates a 2-6 word slug (lowercase, hyphen-joined, no articles/prepositions) from the description. If a Linear issue ID was parsed, it prefixes the slug (e.g., `DEV-988-add-oauth`). The folder name format `{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}` is constructed by `lib/changeman.sh`, which handles date generation, random/provided 4-char ID, collision detection, directory creation, `created_by` detection, `.status.yaml` initialization, and stageman integration. The skill calls `changeman.sh new --slug <slug> [--change-id <4char>] [--log-args <description>]` as a single operation and captures the folder name from stdout.

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
2. Call `lib/changeman.sh new` with `--slug`, optional `--change-id` (backlog ID), and `--log-args` (description). The script handles: directory creation, `created_by` detection (`gh api user` → `git config user.name` → `"unknown"`, silent fallback), `.status.yaml` initialization from template via `sed`, and stageman integration (`set-state intake active fab-new`, `log-command`)
3. Generate `intake.md` from the template (including Origin section), loading `fab/project/constitution.md` and `fab/project/config.yaml` as context

`/fab-new` never activates changes — this reduces disruption when capturing change ideas. The user activates via `/fab-switch` after creation. Branch integration is delegated to `/fab-switch`, which provides consistent branch handling.

#### Change Type Inference

After generating `intake.md`, `/fab-new` infers the `change_type` from the intake content using keyword matching (case-insensitive, first match wins): fix/bug/broken/regression → `fix`, refactor/restructure/consolidate/split/rename → `refactor`, docs/document/readme/guide → `docs`, test/spec/coverage → `test`, ci/pipeline/deploy/build → `ci`, chore/cleanup/maintenance/housekeeping → `chore`, otherwise → `feat`. The inferred type is written to `.status.yaml` via `stageman.sh set-change-type`.

#### Indicative Confidence

After generating `intake.md` and inferring the change type, `/fab-new` computes and displays an indicative confidence score using the coverage-weighted formula with intake-stage `expected_min` thresholds. This is display-only — NOT written to `.status.yaml`. The authoritative score is computed at the spec stage by `calc-score.sh`. Output format: `Indicative confidence: {score} / 5.0 ({N} decisions, cover: {cover})`.

#### Intake-Only Output

`/fab-new` produces a single artifact: `intake.md`. It does not generate `spec.md` or any other downstream artifacts. The intake includes an **Origin** section recording how the change was initiated (description text, conversational vs. one-shot mode, key decisions from the conversation).

#### Context

Loads: config, constitution, `docs/memory/index.md` (to understand the existing memory landscape).

### `/fab-continue [<change-name>] [<stage>]`

`/fab-continue` advances to the next pipeline stage — planning, implementation, review, or hydrate — and either generates the artifact or executes the stage's behavior. When called with a stage argument, it resets to that stage. When called with a change-name argument, it targets that change instead of the active one in `fab/current` (transient — `fab/current` is not modified). Both arguments can coexist; stage names are disambiguated first (fixed set of 6), all other arguments are treated as change-name overrides. The pipeline flows intake → spec → tasks → apply → review → hydrate.

#### Normal Forward Flow (no argument)

1. Read `.status.yaml` to determine current stage (the stage with `active` in the progress map)
2. **Stage guard**: Check `progress.{stage}` value from preflight output:
   - For planning stages (intake, spec, tasks): if `progress.{stage} == 'done'` AND stage is `tasks`, transition to apply. If `progress.{stage} == 'active'`, allow generation to resume. If `progress.{stage} == 'pending'`, allow generation to start.
   - For execution stages (apply, review, hydrate): dispatch to the stage's behavior (apply executes tasks, review validates implementation, hydrate completes the change).
3. Identify next artifact to create
4. Load relevant template + context (including `fab/project/constitution.md` for principles)
5. Generate artifact using the shared generation procedures from `_generation.md` (with clarification/research as needed)
6. Run `lib/calc-score.sh` (spec stage only — computes confidence from spec Assumptions table)
7. Auto-generate checklist when creating tasks (using `_generation.md` Checklist Generation Procedure)
8. Update `.status.yaml`

#### Reset Behavior (with stage argument)

When called as `/fab-continue <stage>` (e.g., `/fab-continue spec`):
1. Target stage can be any of the 6 stages: `intake`, `spec`, `tasks`, `apply`, `review`, `hydrate`
2. Reset `.status.yaml` progress: set target stage to `active`; mark all stages after target as `pending`
3. Regenerate the target stage's artifact in place (update, not recreate from scratch — preserve what's still valid)
4. Downstream artifacts are invalidated: tasks reset to `- [ ]`, checklist regenerated
5. Update `.status.yaml` and report what was reset

Reset is primarily used after review identifies issues upstream.

#### Context (varies by target stage)

- **Spec**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`
- **Tasks**: above + completed `spec.md`

### `/fab-fff [<change-name>]` (Full Pipeline)

`/fab-fff` runs the entire Fab pipeline in a single invocation: planning (spec, tasks) → apply → review → hydrate. No confidence gate. It frontloads questions, interleaves auto-clarify between planning stages, and autonomously reworks on review failure with bounded retry (3 cycles max, escalation after 2 consecutive fix-code failures). Accepts an optional change-name argument to target a specific change instead of the active one in `fab/current`.

#### Minimum Prerequisite

`intake.md` must exist (spec pending or later). `/fab-fff` is callable from any stage at or after intake — it picks up from the current stage and runs forward, skipping stages already `done`.

#### Frontloaded Questions

The skill SHALL scan the intake for ambiguities across *all* remaining planning stages, collect Unresolved decisions into a single batch, and ask once. The goal: one Q&A round, then heads-down generation.

#### Interleaved Auto-Clarify

The `/fab-fff` pipeline interleaves auto-clarify between planning stage generations: `spec → auto-clarify → tasks → auto-clarify`. Each auto-clarify invocation uses the `[AUTO-MODE]` prefix defined in the Skill Invocation Protocol (`_preamble.md`) to signal `/fab-clarify` to operate autonomously. This catches gaps before they compound downstream.

- If auto-clarify finds **blocking issues** (cannot resolve autonomously), the pipeline **bails** — stops, reports the issues, and suggests `Run /fab-clarify to resolve these, then /fab-fff to resume.`
- The pipeline is **resumable** — re-running `/fab-fff` after a bail skips stages already marked `done` and continues from the first incomplete stage.

#### Pipeline Flow

1. Read `fab/current` to resolve the active change; verify intake exists
2. Frontload questions (single batch)
3. Generate `spec.md` (incorporating answers) → run auto-clarify on spec
4. Produce task breakdown (referencing spec and intake) → run auto-clarify on tasks
5. Auto-generate quality checklist
6. Execute tasks via apply behavior
7. Validate implementation via review behavior — on failure, autonomously selects rework path (fix code, revise tasks, revise spec) and retries (max 3 cycles)
8. Hydrate into memory files

#### Autonomous Review Rework

On review failure, `/fab-fff` autonomously selects the rework path based on failure analysis (test failures → fix code, missing functionality → revise tasks, spec drift → revise spec). Maximum 3 rework cycles. Escalation rule: after 2 consecutive "fix code" failures, the agent must escalate to "revise tasks" or "revise spec." After 3 failed cycles, bails with a per-cycle summary and suggests `/fab-continue` for manual rework.

#### When to Use

- Want the full pipeline from intake through hydrate in one command
- Clear requirements upfront, want to reach hydrate quickly with safety nets
- Changes needing quality gates — auto-clarify catches issues between planning stages

#### Context

Loads all planning context upfront: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`.

### `/fab-ff [<change-name>]` (Fast-Forward from Spec, Gated)

`/fab-ff` runs the pipeline from the current stage through hydrate, starting from spec or later. Gated on confidence score via `calc-score.sh --check-gate` (dynamic per-type thresholds: bugfix=2.0, feature/refactor=3.0, architecture=4.0; default 3.0). Minimal auto-clarify (tasks only). Interactive rework on review failure — presents the user with the same 3 rework options as `/fab-continue`. Accepts an optional change-name argument.

#### Minimum Prerequisite

Spec must be `active` or later (spec.md and confidence score already exist). If spec is `pending`, the skill aborts with a message suggesting `/fab-continue` or `/fab-fff`.

#### Confidence Gate

Before proceeding, `/fab-ff` runs `calc-score.sh --check-gate <change_dir>` to check the confidence score against a **dynamic threshold** that varies by change type: bugfix=2.0, feature/refactor=3.0, architecture=4.0 (default: 3.0 when `change_type` is absent). If the score is below the threshold, the skill aborts with a message suggesting `/fab-clarify` to raise confidence.

#### No Frontloaded Questions

Since the spec already exists when `/fab-ff` is invoked, there is no frontloaded question batch. Tasks generation proceeds directly.

#### Minimal Auto-Clarify

`/fab-ff` interleaves auto-clarify only if tasks generation occurs (tasks not yet `done`). If the pipeline starts at apply or later, no auto-clarify runs. Auto-clarify uses `[AUTO-MODE]` prefix and bails on blocking issues.

#### Pipeline Flow

1. Read `fab/current` to resolve the active change; verify spec exists and confidence gate passes
2. Generate `tasks.md` if needed → run auto-clarify on tasks
3. Auto-generate quality checklist
4. Execute tasks via apply behavior
5. Validate implementation via review behavior — on failure, presents interactive rework menu (fix code, revise tasks, revise spec)
6. Hydrate into memory files

#### Interactive Review Rework

On review failure, `/fab-ff` presents the user with the same 3 rework options as `/fab-continue`: fix code (uncheck affected tasks), revise tasks, or revise spec (reset downstream). The user chooses, the pipeline re-runs from the chosen point, and review runs again. No retry cap — the user is in the loop.

#### Resumability

`/fab-ff` is resumable — re-invoking skips stages already marked `done` and continues from the first incomplete stage.

#### Confidence Recomputation

`/fab-ff` does NOT recompute the confidence score during execution. The gate check uses the score from the last manual step (`/fab-continue` at spec stage, or `/fab-clarify`).

#### When to Use

- Spec is already done and confidence is high — want to fast-forward the rest
- After raising confidence via `/fab-clarify` to meet the threshold
- Want autonomous execution with clear abort on unexpected issues

#### Context

Loads all planning context upfront: config, constitution, `intake.md`, `spec.md`, target memory file(s) from `docs/memory/`.

### `/fab-clarify [<change-name>]`

`/fab-clarify` deepens and refines the current stage artifact without advancing to the next stage. It operates in two modes depending on call context: **suggest mode** (user invocation) and **auto mode** (internal `fab-ff` call). It is idempotent and non-advancing. Accepts an optional change-name argument to target a specific change instead of the active one in `fab/current`. See [clarify.md](clarify.md) for the detailed dual-mode specification.

#### Suggest Mode (User Invocation)

When the user invokes `/fab-clarify` directly:

1. Read `.status.yaml` to determine current stage
2. Stage MUST be `intake`, `spec`, or `tasks`. Each stage scans its corresponding artifact(s) using per-artifact taxonomy. If `apply` or later, suggest `/fab-continue` instead
3. Load current artifact + relevant context
4. Perform a **stage-scoped taxonomy scan** for gaps, ambiguities, and `[NEEDS CLARIFICATION]` markers (categories vary by stage)
5. Present structured questions **one at a time** (max 5 per invocation), each with a recommendation and options table or suggested answer
6. **Immediately update the artifact** after each user answer (incremental, not batched)
7. User may terminate early with "done"/"good"/"no more"
8. Append audit trail under `## Clarifications > ### Session {date}` with `Q:` / `A:` entries
9. Display coverage summary (Resolved / Clear / Deferred / Outstanding)
10. Do NOT advance the stage

#### Auto Mode (Internal fab-ff Call)

When called internally by `fab-ff` between stage generations:

1. Perform the same taxonomy scan autonomously — no user interaction
2. Resolve gaps using available context; classify remaining gaps as blocking or non-blocking
3. Return machine-readable result: `{resolved: N, blocking: N, non_blocking: N}`
4. `fab-ff` uses the result to decide whether to continue or bail

#### Key Property

Calling `/fab-clarify` multiple times is safe — it refines further each time. It never transitions to the next stage. Use `/fab-continue` when satisfied.

#### Context (varies by current stage)

- **Spec**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`
- **Tasks**: above + `spec.md`, `tasks.md`

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

### Full-Pipeline Frontloads All Questions
**Decision**: `/fab-fff` collects questions across all planning stages into a single batch before generating any artifacts.
**Why**: Maintains the full-pipeline promise — one Q&A round, then heads-down generation. Multiple interruptions would defeat the purpose. `/fab-ff` does not frontload questions since the spec already exists when it's invoked.
**Rejected**: Per-stage questions during fff (defeats full-pipeline flow). No questions at all (too risky for ambiguous proposals).
*Source*: doc/fab-spec/SKILLS.md; *Updated by*: 260215-237b-DEV-1027-redefine-ff-fff-scope (moved from fab-ff to fab-fff)

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

### Pipeline Skills Interleave Auto-Clarify
**Decision**: `/fab-fff` interleaves auto-clarify between stage generations (`spec → auto-clarify → tasks → auto-clarify`). `/fab-ff` does minimal auto-clarify (tasks only, since spec is already done). Both bail on blocking issues that cannot be resolved autonomously.
**Why**: Gaps in one stage compound downstream. Catching them between stages prevents tasks built on unverified assumptions.
**Rejected**: No clarify in pipeline (gaps compound). Full user-interactive clarify in pipeline (defeats fast-forward flow). Full-auto mode with `<!-- auto-guess -->` markers (defers interaction rather than eliminating it).
*Introduced by*: 260207-m3qf-clarify-dual-modes; *Updated by*: 260208-k3m7-add-fab-fff (removed `--auto` mode); 260215-237b-DEV-1027-redefine-ff-fff-scope (split behavior between fab-fff and fab-ff)

### /fab-new as Single Adaptive Entry Point
**Decision**: Consolidate `/fab-new` and `/fab-discuss` into a single `/fab-new` that adapts via SRAD scoring.
**Why**: Three overlapping entry paths created confusion.
**Rejected**: Keeping both skills with clearer differentiation.
*Introduced by*: 260212-v5p2-simplify-stages-entry-paths

### Stage Rename: "brief" → "intake"
**Decision**: The first pipeline stage is named `intake` (not `brief`). The artifact is `intake.md`. The Intake Generation Procedure lives in `_generation.md` alongside spec/tasks/checklist procedures, with an explicit generation rule emphasizing the intake is a state transfer document.
**Why**: "Brief" in English means short/concise, which triggered summarization instincts in LLMs — agents consistently produced thin briefs despite template instructions. "Intake" signals thorough initial collection in professional contexts (legal, medical, project management). The generation rule in `_generation.md` provides defense in depth alongside the name change.
**Rejected**: `handoff` — could describe any stage boundary. `charter` — too formal. Generation rule only without rename — doesn't fix the misleading name.
*Introduced by*: 260215-v4n7-DEV-1025-rename-brief-to-intake

### Shared Generation Partial
**Decision**: Extract duplicated artifact generation logic from `/fab-continue` and `/fab-ff` into a shared `_generation.md` partial. Both skills reference the partial for spec, tasks, checklist, and intake generation mechanics; each retains its own orchestration logic.
**Why**: Generation steps were nearly identical in both skills, requiring every fix or behavior change to be applied in two places. Centralizing eliminates drift and makes generation behavior authoritative in one location.
**Rejected**: Keeping inline duplication — inevitable drift between the two copies.
*Introduced by*: 260210-wpay-extract-shared-generation-logic

### Unified Command: `/fab-continue` Absorbs Execution Stages
**Decision**: `/fab-continue` handles all 6 pipeline stages (intake → spec → tasks → apply → review → hydrate). Apply, review, and hydrate behaviors are described as dedicated sections within `fab-continue.md`, not extracted into a shared partial. `/fab-archive` exists as a standalone housekeeping skill (not a pipeline stage) for post-hydrate cleanup.
**Why**: Reduces developer command surface from 4+ commands to 2 (`/fab-continue` + `/fab-clarify`). Execution stages are orchestration-heavy with distinct flows (task execution, validation with rework, memory hydration) — inlining keeps each stage's behavior in one readable location.
**Rejected**: Keeping standalone `/fab-apply`, `/fab-review` — command fragmentation. Extracting to `_execution.md` partial — low reuse value since only fab-continue calls these.
*Introduced by*: 260212-a4bd-unify-fab-continue

### `/fab-ff` and `/fab-fff` Keep Behavioral Descriptions
**Decision**: `/fab-ff` and `/fab-fff` describe execution behavior inline within their own orchestration context, rather than literally invoking `/fab-continue` as a sub-skill.
**Why**: These skills have fundamentally different orchestration: frontloaded questions, auto-clarify interleaving, bail behavior, resumability across all stages. Literal sub-skill invocation would add complexity (nested preflight checks, status conflicts) without benefit.
**Rejected**: Literal `/fab-continue` invocation from fab-ff/fff — orchestration mismatch, nested state management issues.
*Introduced by*: 260212-a4bd-unify-fab-continue

### Scope Differentiation: fab-fff (Full Pipeline) vs fab-ff (From-Spec, Gated)
**Decision**: `/fab-fff` is the full pipeline command (intake → hydrate) with no confidence gate, frontloaded questions, interleaved auto-clarify, and autonomous rework on review failure (3-cycle retry cap, escalation after 2 consecutive fix-code failures). `/fab-ff` is the fast-forward-from-spec command (spec → hydrate) gated on confidence score, with no frontloaded questions, minimal auto-clarify, and interactive rework on review failure (same 3 options as `/fab-continue`, no retry cap).
**Why**: The naming intuition: `fff` (three f's) = "maximum autonomy, don't bother me" — autonomous rework is the natural completion of that contract. `ff` (two f's) = "fast but I'm watching" — interactive rework lets the user steer correction when the confidence gate passed but review still failed. The confidence gate on `/fab-ff` means the user asserted trust in spec quality; when failure occurs despite high confidence, user steering is more valuable than autonomous guessing.
**Rejected**: Previous design had it backwards: `/fab-fff` (interactive rework) interrupted the user in the most autonomous command, `/fab-ff` (bail) gave the user the least control over recovery. Both autonomous — makes `/fab-ff` too similar to `/fab-fff`. Both interactive — same issue.
*Introduced by*: 260215-237b-DEV-1027-redefine-ff-fff-scope; *Updated by*: 260216-knmw-DEV-1030-swap-ff-fff-review-rework (swapped review failure behavior)

### Reset via `/fab-continue <stage>`
**Decision**: Reset to any pipeline stage by passing the stage name as an argument to `/fab-continue`. For planning stages, downstream artifacts are invalidated and regenerated. For execution stages, the stage behavior is re-run without resetting task checkboxes.
**Why**: Provides a clean re-entry point after review identifies upstream issues. Reuses the existing skill rather than adding a separate `/fab-reset` command. Covers all 6 stages (intake, spec, tasks, apply, review, hydrate).
**Rejected**: Separate reset skill — unnecessary proliferation of skills for a rare operation.
*Source*: doc/fab-spec/SKILLS.md; *Updated by*: 260212-a4bd-unify-fab-continue (extended to all 6 stages)

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260226-i9av-add-ready-state-to-stages | 2026-02-26 | `/fab-continue` gains state-based dispatch: `active` → generate artifact, `ready` → advance to next stage. `/fab-ff` redefined: starts from intake (was: spec-only), 3 safety gates (intake indicative >= 3.0, spec per-type threshold, review 3-cycle stop). `/fab-fff` unchanged except contrast text. `/fab-clarify` accepts `ready` state. `_preamble.md` State Table adds `/fab-ff` to intake row, state derivation includes `ready`. Dual gate thresholds documented (intake fixed 3.0, spec dynamic per-type). |
| 260226-tnr8-coverage-scoring-change-types | 2026-02-26 | `/fab-new` gains change type inference (keyword heuristic → `stageman.sh set-change-type`) and indicative confidence display (coverage-weighted formula, display-only, not persisted). Coverage-weighted confidence formula added to `_preamble.md` §Confidence Scoring. Gate thresholds updated from 4-type (`bugfix`/`feature`/`refactor`/`architecture`) to 7-type taxonomy (`feat`/`fix`/`refactor`/`docs`/`test`/`ci`/`chore`). |
| 260221-5tj7-rename-context-to-preamble | 2026-02-21 | Renamed shared skill preamble from `_context.md` to `_preamble.md`. Updated all references in Shared Generation Partial section, SRAD design decision, and mode selection references. |
| 260216-7ltw-DEV-1038-standardize-state-keyed-suggestions | 2026-02-16 | Replaced skill-keyed suggestion lookup with state-keyed table in `_preamble.md`. Removed `--switch` flag and natural language switching detection from `/fab-new` — change is never activated by `/fab-new`. All skills now derive `Next:` lines from canonical state table. Extended `/fab-clarify` stage guard to include `intake`. |
| 260216-knmw-DEV-1030-swap-ff-fff-review-rework | 2026-02-16 | Swapped review failure behavior: `/fab-ff` now presents interactive rework menu (3 options, no retry cap); `/fab-fff` now uses autonomous rework (agent selects path, 3-cycle retry cap, escalation after 2 consecutive fix-code). Updated overview paragraphs, pipeline flow steps, rework sections, and Scope Differentiation design decision. |
| 260215-237b-DEV-1027-redefine-ff-fff-scope | 2026-02-16 | Redefined `/fab-ff` and `/fab-fff` scope. `/fab-fff` is now the full pipeline command (intake → hydrate, no gate, frontloaded questions, interactive rework). `/fab-ff` is now the fast-forward-from-spec command (spec → hydrate, confidence-gated, no frontloaded questions, bail on failure). Updated overview, requirement sections, and design decisions. Added Scope Differentiation design decision. |
| 260215-9yjx-DEV-1022-create-changeman-script | 2026-02-15 | Refactored `/fab-new`: "Folder Name Generation" → "Slug Generation and Change Creation" (delegated to `lib/changeman.sh`). Change Initialization steps consolidated — steps 1-2 of old init (mkdir, .status.yaml, created_by, stageman calls) replaced by single `changeman.sh new` call. Skill now focuses on AI tasks (slug generation, gap analysis, intake writing). Error table simplified. |
| 260215-v4n7-DEV-1025-rename-brief-to-intake | 2026-02-15 | Renamed `brief` → `intake` throughout. Added Intake Generation Procedure to `_generation.md`. Updated `/fab-new` to reference procedure instead of inlining. Renamed "Brief-First" design decision to "Intake-First" |
| 260215-w3n8-naming-linear-id-drop-conventions | 2026-02-15 | Updated `/fab-new` folder name generation format to `{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}` with optional uppercase Linear issue ID |
| 260214-m3w7-formalize-assumptions-scoring | 2026-02-14 | Formalized Assumptions tables: all four SRAD grades recorded (not just Confident/Tentative), Scores column required, Unresolved rows include status context. `calc-score.sh` reads only spec.md (not intake+spec), fixed AWK cols[6], removed has_scores detection and Certain carry-forward, parses Unresolved grade. Spec generation reads intake assumptions as starting point (confirm/upgrade/override). Templates include formalized `## Assumptions` sections. Summary line uses 4-grade format. |
| 260214-r7k3-stageman-yq-metrics | 2026-02-14 | All skill prompts now call `log-command` after preflight and pass `driver` on all `set-state`/`transition` calls. `/fab-new` calls `set-state intake active fab-new`. `/fab-clarify` calls `log-command` after preflight. `/fab-ff` and `/fab-fff` pass driver on all transitions. Added shared generation partial note about `log-command` and driver conventions |
| 260212-f9m3-enhance-srad-fuzzy | 2026-02-14 | SRAD framework updated to fuzzy 0–100 dimension scoring with weighted mean aggregation; `/fab-fff` confidence gate now uses dynamic per-type thresholds (bugfix=2.0, feature/refactor=3.0, architecture=4.0) via `calc-score.sh --check-gate`; optional Scores column in Assumptions tables for per-dimension data |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_stageman.sh` → `lib/stageman.sh` and `_calc-score.sh` → `lib/calc-score.sh` in all references; updated shared generation partial `lib/stageman.sh set-checklist` references |
| 260214-w3r8-stageman-write-api | 2026-02-14 | Skill prompts (`fab-continue.md`, `fab-ff.md`, `fab-fff.md`, `_generation.md`) now reference `lib/stageman.sh` CLI commands for all `.status.yaml` mutations instead of ad-hoc editing |
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
| 260210-zr1f-discuss-auto-activate-when-no-current | 2026-02-10 | `/fab-discuss` conditionally offers activation when `fab/current` is empty; updated proposal output, key differences table |
| 260209-r4w8-archive-index-longer-slugs | 2026-02-09 | Expanded slug word count from 2-4 to 2-6 words in `/fab-new` folder name generation |
| 260208-q8v3-branch-to-switch | 2026-02-09 | Moved branch integration from `/fab-new` to `/fab-switch`, removed `--branch` flag from `/fab-new`, `/fab-new` now calls `/fab-switch` internally |
| 260208-lgd7-fab-discuss-command | 2026-02-08 | Added `/fab-discuss` conversational intake skill, `/fab-new` confidence scoring, context-driven mode selection design decisions |
| 260208-k3m7-add-fab-fff | 2026-02-08 | Added `/fab-fff` full pipeline skill, confidence recomputation in `/fab-continue`, removed `/fab-ff --auto` mode, updated design decisions |
| 260207-09sj-autonomy-framework | 2026-02-08 | Added SRAD autonomy framework, confidence grades, assumptions summaries, branch auto-create on main, soft gate on fab-apply |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260207-m3qf-clarify-dual-modes | 2026-02-07 | Updated `/fab-clarify` to dual-mode (suggest + auto), `/fab-ff` with interleaved auto-clarify and `--auto` flag |
| — | 2026-02-07 | Generated from doc/fab-spec/ (README.md, SKILLS.md, TEMPLATES.md) |
