# Planning Skills

**Domain**: fab-workflow

## Overview

The planning skills (`/fab-new`, `/fab-continue`, `/fab-clarify`) handle the first two stages of the 7-stage Fab pipeline: intake and spec. They produce the planning artifacts (`intake.md`, `spec.md`) that define *what* changes and *how*, before any code is written. The implementation plan (`plan.md`) is generated at apply entry — see [execution-skills.md](execution-skills.md) — not by a planning skill.

`/fab-fff` and `/fab-ff` are also documented here because their planning behavior originated as planning skills. `/fab-fff` is the **full-pipeline command** (intake → review-pr, confidence-gated, no frontloaded questions, autonomous rework). `/fab-ff` is the **fast-forward command** (intake → hydrate, confidence-gated, autonomous rework). See sections below for details.

## Shared Generation Partial

The artifact generation logic (spec and plan) is defined in a single shared partial: `$(fab kit-path)/skills/_generation.md`. `/fab-continue` references this partial for spec generation; the apply skill (within `/fab-continue`) references it for plan generation.

The partial contains two procedures:
- **Spec Generation Procedure** — template loading, metadata, RFC 2119 requirements, GIVEN/WHEN/THEN scenarios, Assumptions section (reads intake assumptions as starting point, confirms/upgrades/overrides each)
- **Plan Generation Procedure** — template loading (`plan.md`), single walk over spec requirements emitting paired Task + Acceptance entries per requirement, sequential `T-NNN` task IDs and `A-NNN` acceptance IDs, optional cross-linking. The procedure runs at apply entry, not as a separate planning stage. Counts (`task_count`, `acceptance_count`, `acceptance_completed`) flow into `.status.yaml` `plan:` block via the PostToolUse `artifactBookkeeping` hook on every `plan.md` write; skills MAY also call `fab status set-acceptance` for explicit updates (e.g., review marking acceptance items complete).

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

#### Indicative Confidence

After generating `intake.md` and inferring the change type, `/fab-new` persists an indicative confidence score by calling `calc-score.sh --stage intake <change>` in normal mode (not `--check-gate`). This writes the score to `.status.yaml` with `confidence.indicative: true`, making it visible to all consumers (`/fab-switch`, `/fab-status`, `changeman.sh list`) without recomputation. The authoritative spec-stage score overwrites it (clearing `indicative: true`) when `calc-score.sh` runs at the spec stage. Output format: `Indicative confidence: {score} / 5.0 ({N} decisions)`.

#### Output

`/fab-new` produces `intake.md` as its primary artifact. It does not generate `spec.md` or any other downstream artifacts. The intake includes an **Origin** section recording how the change was initiated (description text, conversational vs. one-shot mode, key decisions from the conversation). After the intake, the output includes `Activated: {name}` (Step 10) and `Branch: {name} (created|created, leaving {old_branch} intact|checked out|renamed from {old_branch}|already active)` (Step 11).

#### Context

Loads: config, constitution, `docs/memory/index.md` (to understand the existing memory landscape).

### `/fab-continue [<change-name>] [<stage>]`

`/fab-continue` advances to the next pipeline stage — planning, implementation, review, or hydrate — and either generates the artifact or executes the stage's behavior. When called with a stage argument, it resets to that stage. When called with a change-name argument, it targets that change instead of the active one in `.fab-status.yaml` (transient — `.fab-status.yaml` is not modified). Both arguments can coexist; stage names are disambiguated first (fixed set of 7: `intake`, `spec`, `apply`, `review`, `hydrate`, `ship`, `review-pr`), all other arguments are treated as change-name overrides. The pipeline flows intake → spec → apply → review → hydrate → ship → review-pr.

A passed `tasks` stage argument SHALL error immediately with `"tasks" stage was removed — use /fab-continue apply (regenerates plan.md and re-runs) or /fab-clarify spec.` No alias window — the migration ensures no in-flight `.status.yaml` carries a `tasks` key after upgrade.

#### Normal Forward Flow (no argument)

1. Read `.status.yaml` to determine current stage and state
2. **Consolidated planning dispatch**: For planning stages, `/fab-continue` handles a full cycle in one invocation:
   - **spec `ready`**: Finish spec (`done`) → start apply (`active`) → execute apply (apply's entry sub-step generates `plan.md`, then task execution begins)
   - **`active` state** (backward compat for interrupted generations): Generate the artifact, advance to `ready`
   - For execution stages (apply, review, hydrate, ship, review-pr): dispatch to the stage's behavior
3. Load relevant template + context (including `fab/project/constitution.md` for principles)
4. Generate artifact using the shared generation procedures from `_generation.md` (with clarification/research as needed)
5. Run `fab score` (spec stage only — computes confidence from spec Assumptions table)
6. Update `.status.yaml`

#### Reset Behavior (with stage argument)

When called as `/fab-continue <stage>` (e.g., `/fab-continue spec`):
1. Target stage can be any of the 7 stages: `intake`, `spec`, `apply`, `review`, `hydrate`, `ship`, `review-pr`. `tasks` is rejected with the strict-error message above.
2. Reset `.status.yaml` progress: set target stage to `active`; mark all stages after target as `pending`
3. Regenerate the target stage's artifact in place (update, not recreate from scratch — preserve what's still valid)
4. Downstream artifacts are invalidated where applicable: e.g., spec reset → apply pending → `plan.md` regenerated on next apply entry
5. Advance the target stage to `ready` for planning resets (not `done` — preserves `/fab-clarify` opportunity)

`fab status reset apply` modifies `.status.yaml` state only — `plan.md` persists on disk per the existing artifact-file convention. The apply skill's plan-generation sub-step is skipped on the next `/fab-continue` (idempotent on `plan.md` presence) and execution resumes from the first unchecked task. To force plan regeneration, the user MUST delete `plan.md` before re-running `/fab-continue`.

Reset is primarily used after review identifies issues upstream.

#### Context (varies by target stage)

- **Spec**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`
- **Apply** (plan generation sub-step): above + completed `spec.md`

### `/fab-fff [<change-name>]` (Full Pipeline)

`/fab-fff` runs the entire Fab pipeline in a single invocation: planning (spec) → apply (which generates `plan.md` at entry, then executes tasks) → review → hydrate → ship → review-pr. Confidence-gated with identical gates to `/fab-ff` (intake gate + spec gate). Interleaves auto-clarify between planning stages, and autonomously reworks on review failure with bounded retry (3 cycles max, escalation after 2 consecutive fix-code failures). Accepts an optional change-name argument to target a specific change instead of the active one in `.fab-status.yaml`. Accepts `--force` to bypass all confidence gates.

#### Minimum Prerequisite

`intake.md` must exist (spec pending or later). `/fab-fff` is callable from any stage at or after intake — it picks up from the current stage and runs forward, skipping stages already `done`. Confidence gates (intake gate + spec gate) must pass unless `--force` is used.

#### Interleaved Auto-Clarify

The `/fab-fff` pipeline interleaves auto-clarify between planning stages and at apply entry: `spec → auto-clarify spec → apply (plan generation) → auto-clarify plan → task execution`. Each auto-clarify invocation uses the `[AUTO-MODE]` prefix defined in the Skill Invocation Protocol (`_preamble.md`) to signal `/fab-clarify` to operate autonomously. This catches gaps before they compound downstream.

- If auto-clarify finds **blocking issues** (cannot resolve autonomously), the pipeline **bails** — stops, reports the issues, and suggests `Run /fab-clarify to resolve these, then /fab-fff to resume.`
- The pipeline is **resumable** — re-running `/fab-fff` after a bail skips stages already marked `done` and continues from the first incomplete stage.

#### Pipeline Flow

1. Resolve the active change (via `.fab-status.yaml` symlink); verify intake exists
2. Intake gate check (skip if `--force`)
3. Generate `spec.md` → spec gate check (skip if `--force`) → run auto-clarify on spec
4. Apply: plan generation sub-step writes `plan.md` (`## Tasks` + `## Acceptance` populated in one pass) → optional auto-clarify on `plan` → execute tasks
5. Validate implementation via review behavior — on failure, autonomously selects rework path (fix code, revise tasks, revise spec) and retries (max 3 cycles)
6. Hydrate into memory files
7. Ship (dispatch `/git-pr`)
8. Review-PR (dispatch `/git-pr-review`)

#### Autonomous Review Rework

On review failure, `/fab-fff` autonomously selects the rework path based on failure analysis (test failures → fix code, missing functionality → revise tasks, spec drift → revise spec). Maximum 3 rework cycles. Escalation rule: after 2 consecutive "fix code" failures, the agent must escalate to "revise tasks" or "revise spec." After 3 failed cycles, bails with a per-cycle summary and suggests `/fab-continue` for manual rework.

#### When to Use

- Want the full pipeline from intake through PR review in one command
- Clear requirements upfront, want to reach completion quickly with safety nets
- Changes needing quality gates — confidence gates and auto-clarify catch issues between planning stages

#### Context

Loads all planning context upfront: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`.

### `/fab-ff [<change-name>]` (Fast-Forward, Gated)

`/fab-ff` runs the pipeline from intake through hydrate. Gated on confidence score with identical gates to `/fab-fff`: intake gate (indicative >= 3.0) and spec gate (dynamic per-type thresholds). Interleaves auto-clarify between planning stages. Autonomous rework on review failure (3-cycle cap, escalation after 2 consecutive fix-code). Accepts an optional change-name argument. Accepts `--force` to bypass all confidence gates.

#### Minimum Prerequisite

`intake.md` must exist. `/fab-ff` is callable from any stage at or after intake — it picks up from the current stage and runs forward through hydrate, skipping stages already `done`. Confidence gates (intake gate + spec gate) must pass unless `--force` is used.

#### Confidence Gates

`/fab-ff` has two confidence gates identical to `/fab-fff`: (1) intake gate — indicative confidence >= 3.0 via `fab score --check-gate --stage intake`, and (2) spec gate — confidence >= per-type dynamic threshold via `fab score --check-gate`. Both gates are skipped when `--force` is passed.

#### Interleaved Auto-Clarify

`/fab-ff` interleaves auto-clarify between planning stages and at apply entry: `spec → auto-clarify spec → apply (plan generation) → auto-clarify plan → task execution`. Auto-clarify uses `[AUTO-MODE]` prefix and bails on blocking issues.

#### Pipeline Flow

1. Resolve the active change (via `.fab-status.yaml` symlink); verify intake exists
2. Intake gate check (skip if `--force`)
3. Generate `spec.md` → spec gate check (skip if `--force`) → run auto-clarify on spec
4. Apply: plan generation sub-step writes `plan.md` → optional auto-clarify on `plan` → execute tasks
5. Validate implementation via review behavior — on failure, autonomous rework (3-cycle cap, escalation rule)
6. Hydrate into memory files

#### Autonomous Review Rework

On review failure, `/fab-ff` autonomously selects the rework path based on failure analysis (same behavior as `/fab-fff`). Maximum 3 rework cycles. Escalation rule: after 2 consecutive "fix code" failures, the agent must escalate. After 3 failed cycles, stops with a per-cycle summary.

#### Resumability

`/fab-ff` is resumable — re-invoking skips stages already marked `done` and continues from the first incomplete stage.

#### Confidence Recomputation

`/fab-ff` does NOT recompute the confidence score during execution. The gate check uses the score from the last manual step (`/fab-continue` at spec stage, or `/fab-clarify`).

#### When to Use

- Small, well-understood changes that don't need ship/review-pr
- Want to reach hydrate quickly with safety nets (confidence gates + auto-rework)
- After raising confidence via `/fab-clarify` to meet the threshold

#### Context

Loads all planning context upfront: config, constitution, `intake.md`, `spec.md`, target memory file(s) from `docs/memory/`.

### `/fab-clarify [<change-name>]`

`/fab-clarify` deepens and refines the current stage artifact without advancing to the next stage. It operates in two modes depending on call context: **suggest mode** (user invocation) and **auto mode** (internal `fab-ff` call). It is idempotent and non-advancing. Accepts an optional change-name argument to target a specific change instead of the active one in `.fab-status.yaml`. See [clarify.md](clarify.md) for the detailed dual-mode specification.

#### Suggest Mode (User Invocation)

When the user invokes `/fab-clarify` directly:

1. Read `.status.yaml` to determine current stage
2. Stage MUST be `intake` or `spec` (planning stages), or `apply`/later with `plan.md` present (for `plan` target). Each stage scans its corresponding artifact(s) using per-artifact taxonomy. `/fab-clarify tasks` errors with `"tasks" target was removed — use plan (post-apply-entry) or spec (pre-apply).`
3. Load current artifact + relevant context
4. **Bulk confirm check** (Step 1.5): Parse the `## Assumptions` table. If `confident >= 3` AND `confident > tentative + unresolved`, display all Confident assumptions as a numbered list for conversational bulk response (confirm/change/explain). After resolution, proceed to step 5. See [clarify.md](clarify.md) for full details.
5. Perform a **stage-scoped taxonomy scan** for gaps, ambiguities, and `[NEEDS CLARIFICATION]` markers (categories vary by stage)
6. Present structured questions **one at a time** (max 5 per invocation), each with a recommendation and options table or suggested answer
7. **Immediately update the artifact** after each user answer (incremental, not batched)
8. User may terminate early with "done"/"good"/"no more"
9. Append audit trail under `## Clarifications > ### Session {date}` with `Q:` / `A:` entries
10. Display coverage summary (Resolved / Clear / Deferred / Outstanding)
11. Do NOT advance the stage

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
- **Plan** (post-apply-entry): above + `spec.md`, `plan.md`

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
**Decision**: Neither `/fab-ff` nor `/fab-fff` frontloads questions. Both proceed directly to spec generation, relying on confidence gates to block changes that are too ambiguous.
**Why**: Frontloaded questions interrupted the autonomous pipeline flow. Confidence gates provide a better signal — if the intake has too many unresolved decisions, the gate blocks and the user resolves via `/fab-clarify` before retrying. This is cleaner than a mid-pipeline Q&A round.
**Rejected**: Previous design had `/fab-fff` frontloading questions — this interrupted the autonomous flow unnecessarily.
*Source*: doc/fab-spec/SKILLS.md; *Updated by*: 260215-237b-DEV-1027-redefine-ff-fff-scope (moved from fab-ff to fab-fff); 260314-q5p9-redesign-ff-fff-scopes (dropped frontloaded questions entirely)

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
**Decision**: Both `/fab-ff` and `/fab-fff` interleave auto-clarify between planning stages and at apply entry (`spec → auto-clarify spec → apply plan-gen → optional auto-clarify plan → task execution`). Both bail on blocking issues that cannot be resolved autonomously.
**Why**: Gaps in one stage compound downstream. Catching them between stages prevents tasks built on unverified assumptions. With the `tasks` stage removed in qszh, the natural auto-clarify checkpoint moved from "after tasks generation" to "after plan generation at apply entry" — same intent, single artifact.
**Rejected**: No clarify in pipeline (gaps compound). Full user-interactive clarify in pipeline (defeats fast-forward flow). Full-auto mode with `<!-- auto-guess -->` markers (defers interaction rather than eliminating it).
*Introduced by*: 260207-m3qf-clarify-dual-modes; *Updated by*: 260208-k3m7-add-fab-fff (removed `--auto` mode); 260215-237b-DEV-1027-redefine-ff-fff-scope (split behavior between fab-fff and fab-ff); 260314-q5p9-redesign-ff-fff-scopes (unified: both skills now interleave identically); 260423-qszh-merge-tasks-checklist (auto-clarify tasks → auto-clarify plan; checkpoint moves into apply entry)

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
**Decision**: Artifact generation logic lives in a single shared `_generation.md` partial. After qszh, the partial defines two procedures: **Spec Generation Procedure** (invoked by planning stage) and **Plan Generation Procedure** (invoked by apply at entry). Each consumer retains its own orchestration logic.
**Why**: Generation steps were nearly identical across skills, requiring every fix or behavior change to be applied in two places. Centralizing eliminates drift and makes generation behavior authoritative in one location. With qszh, the prior Tasks Generation Procedure + Checklist Generation Procedure pair collapsed into one Plan Generation Procedure that walks spec requirements once and emits paired Task + Acceptance entries — eliminating the silent-drift failure mode where a requirement showed up in `tasks.md` but was missed from `checklist.md`.
**Rejected**: Keeping inline duplication — inevitable drift between the copies. Keeping two procedures (Tasks + Checklist) — re-introduces the drift the merger eliminated.
*Introduced by*: 260210-wpay-extract-shared-generation-logic; *Updated by*: 260423-qszh-merge-tasks-checklist (Tasks + Checklist procedures → unified Plan Generation Procedure)

### Plan Generation Lives at Apply Entry, Not in a Separate Stage (qszh)
**Decision**: The `tasks` stage is removed from the pipeline. Plan generation (writing `plan.md` with `## Tasks` + `## Acceptance`) is an entry sub-step of the apply skill, not a stage gate. Pipeline goes from 8 stages (intake → spec → tasks → apply → review → hydrate → ship → review-pr) to 7 stages (intake → spec → apply → review → hydrate → ship → review-pr). `progress.tasks` is dropped from `.status.yaml` entirely — no rename to `progress.plan`, since with no separate stage there is no key to populate.
**Why**: The `tasks` stage was a no-decision gate. `/fab-continue` advanced spec → tasks → apply back-to-back; users never stopped at tasks. Every change paid the wall-time, token, and `.status.yaml` cost of a transition that had no decision content. Folding generation into apply makes drift between Tasks and Acceptance mechanically impossible (single skill call, single context window, single LLM, single file). Spec stage and its decision-rich roles (`/fab-clarify spec` workhorse, per-type `fab score` spec gate, review's behavioral reference) are preserved — the collapse stops at `tasks`.
**Rejected**: (a) Keep two files, add cross-check step — adds ceremony, not less. (b) Drop checklist, review reads tasks directly — loses the imperative-vs-declarative framing review depends on. (c) Merge artifacts only, keep `tasks` stage — fixes drift but keeps the no-decision gate; xvaz workaround remains relevant. (d) Drop both spec AND tasks — loses `/fab-clarify spec`, breaks per-type spec gate, weakens review. (e) Rename `tasks` → `plan` stage — same gate, new name. (f) Rename `apply` → `execute`/`implement` — semantic gain doesn't justify migration churn across state table, `.status.yaml`, all skills, muscle memory.
*Introduced by*: 260423-qszh-merge-tasks-checklist; *Supersedes*: the proposal in `260423-xvaz-skip-tasks-simple-types` (per-type skip policy for the tasks stage) — that draft becomes obsolete by construction once qszh ships, since there is no separate stage to skip. Simple changes naturally produce a tiny `plan.md` and execute in seconds; no skip policy needed. The xvaz folder will be archived by a separate user-initiated `/fab-archive 260423-xvaz...` action.

### Strict-Error Stance for Legacy `tasks` References (qszh)
**Decision**: All `tasks` stage references and the legacy `set-checklist` CLI command error immediately with a helpful pointer message — no alias window, no phased deprecation. `fab status start|advance|finish|reset|skip|fail <change> tasks` returns exit 1 with `"tasks" stage was removed — run ... apply instead. plan.md is now generated at apply entry.` `fab status set-checklist` returns exit 1 with `"set-checklist" is now "set-acceptance" — run fab status set-acceptance instead.` `/fab-clarify tasks` errors with a similar pointer.
**Why**: The 1.8.0-to-1.9.0 migration rewrites every in-flight `.status.yaml` so no live change carries a `tasks` key after upgrade. With no live `tasks` state to support, an alias adds maintenance burden for zero user benefit. Strict errors with pointer messages are self-documenting and steer users toward the new workflow immediately.
**Rejected**: Phased deprecation (alias for one release, error in next) — no in-flight `.status.yaml` carries `tasks` after migration, so phasing buys nothing. Silent renaming — leaves users uncertain whether a command did what they expected.
*Introduced by*: 260423-qszh-merge-tasks-checklist

### Unified Command: `/fab-continue` Absorbs Execution Stages
**Decision**: `/fab-continue` handles all 7 pipeline stages (intake → spec → apply → review → hydrate → ship → review-pr). Apply, review, hydrate, ship, and review-pr behaviors are described as dedicated sections within `fab-continue.md`, not extracted into a shared partial. `/fab-archive` exists as a standalone housekeeping skill (not a pipeline stage) for post-hydrate cleanup. The apply behavior includes a Plan Generation sub-step at entry (writes `plan.md`); see [execution-skills.md](execution-skills.md).
**Why**: Reduces developer command surface from 4+ commands to 2 (`/fab-continue` + `/fab-clarify`). Execution stages are orchestration-heavy with distinct flows (plan generation + task execution, validation with rework, memory hydration) — inlining keeps each stage's behavior in one readable location.
**Rejected**: Keeping standalone `/fab-apply`, `/fab-review` — command fragmentation. Extracting to `_execution.md` partial — low reuse value since only fab-continue calls these. Splitting plan generation into a `/fab-plan` skill — adds a command surface for what is mechanically a single autonomous step at apply entry.
*Introduced by*: 260212-a4bd-unify-fab-continue; *Updated by*: 260303-he6t-extend-pipeline-through-pr (added ship + review-pr); 260423-qszh-merge-tasks-checklist (dropped tasks stage; plan generation folded into apply entry — 7 stages)

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
**Decision**: Reset to any pipeline stage by passing the stage name as an argument to `/fab-continue`. For planning stages, downstream artifacts are invalidated and regenerated. For execution stages, the stage behavior is re-run without resetting task checkboxes. `tasks` is rejected with a strict-error pointer to `apply` or `/fab-clarify spec` — the `tasks` stage was removed in qszh.
**Why**: Provides a clean re-entry point after review identifies upstream issues. Reuses the existing skill rather than adding a separate `/fab-reset` command. Covers all 7 stages (intake, spec, apply, review, hydrate, ship, review-pr). `fab status reset apply` preserves `plan.md` on disk; the apply entry sub-step skips regeneration when the file exists, so users who want a fresh plan must delete `plan.md` before re-running.
**Rejected**: Separate reset skill — unnecessary proliferation of skills for a rare operation. Auto-deleting `plan.md` on apply reset — violates the existing artifact-file convention (reset modifies `.status.yaml` state only; artifact files persist) and Constitution III idempotency.
*Source*: doc/fab-spec/SKILLS.md; *Updated by*: 260212-a4bd-unify-fab-continue (extended to all 6 stages); 260303-he6t-extend-pipeline-through-pr (extended to ship + review-pr — 8 stages); 260423-qszh-merge-tasks-checklist (dropped tasks stage — 7 stages; documented `reset apply` plan.md preservation)

## Changelog

| Change | Date | Summary |
|--------|------|---------|
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
