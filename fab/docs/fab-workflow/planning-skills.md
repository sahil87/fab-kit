# Planning Skills

**Domain**: fab-workflow

## Overview

The planning skills (`/fab-new`, `/fab-continue`, `/fab-clarify`) handle the first three stages of the 6-stage Fab pipeline: brief, spec, and tasks. They produce the artifacts that define *what* changes and *how*, before any code is written.

`/fab-ff` is also documented here because its planning behavior (frontloaded questions, auto-clarify) originated as a planning skill. However, `/fab-ff` is now a **full-pipeline command** that continues through apply, review, and archive after planning completes. See the `/fab-ff` section below for details.

## Shared Generation Partial

The artifact generation logic (spec, tasks, checklist) is defined in a single shared partial: `fab/.kit/skills/_generation.md`. Both `/fab-continue` and `/fab-ff` reference this partial for the mechanics of producing each artifact, rather than inlining the generation steps.

The partial contains three procedures:
- **Spec Generation Procedure** — template loading, metadata, RFC 2119 requirements, GIVEN/WHEN/THEN scenarios, Assumptions section
- **Tasks Generation Procedure** — template loading, metadata, phased task breakdown, task format, execution order
- **Checklist Generation Procedure** — template loading, category population, sequential CHK IDs, `.status.yaml` updates

Each skill retains its own orchestration logic (stage guards, question handling, auto-clarify, resumability). Only the generation mechanics are shared.

## Requirements

### `/fab-new <description>`

`/fab-new` starts a new change from a natural language description. It is adaptive: clear inputs get a quick brief, vague inputs trigger conversational exploration. It creates the change folder, initializes status tracking, and generates a brief (with Origin section). By default, the change is NOT activated (no write to `fab/current`, no branch integration). Accepts an optional `--switch` flag or detects switching intent from natural language to call `/fab-switch` internally and activate the change. Output is always a single artifact: `brief.md`.

#### Folder Name Generation

The agent SHALL generate a folder name in the format `{YYMMDD}-{XXXX}-{slug}` where YYMMDD is today's date, XXXX is 4 random lowercase alphanumeric chars, and slug is 2-6 words extracted from the description. All components MUST be lowercase.

#### Adaptive Behavior (SRAD-Driven)

`/fab-new` adapts its interaction style based on the input clarity:

1. **Clear input** — SRAD scoring identifies few or no Unresolved decisions. The skill generates the brief with up to 3 targeted questions (highest blast radius), assumes all Confident/Tentative decisions, and completes quickly.
2. **Vague input** — SRAD scoring identifies many Unresolved decisions. The skill enters **conversational mode**: back-and-forth exploration with no fixed question cap, starting with the highest-impact decisions (lowest Reversibility + lowest Agent Competence). Each question builds on previous answers. The conversation ends when the confidence score reaches >= 3.0 and the user signals satisfaction, or the user terminates early.

#### Gap Analysis

Before committing to a brief, `/fab-new` evaluates whether the change is needed:

1. Checks for existing mechanisms in the current workflow, codebase, or docs
2. Evaluates scope — is the idea too broad (should be split) or too narrow (part of something larger)?
3. Considers alternatives — simpler approaches, extending existing skills

If an existing mechanism covers the idea, the skill presents its findings and lets the user decide whether to proceed. If no change folder is created, no `Next:` line is shown.

#### Change Initialization

The skill SHALL:
1. Create `fab/changes/{name}/`
2. Initialize `.status.yaml` with `created_by` set using a fallback chain: `gh api user --jq .login` (primary), then `git config user.name`, then `"unknown"`. No error or warning on fallback. `brief: active` as the initial progress state
3. Generate `brief.md` from the template (including Origin section), loading `fab/constitution.md` and `fab/config.yaml` as context
4. Mark brief complete once the user is satisfied
5. Conditionally call `/fab-switch` to activate the change (writes `fab/current`, performs branch integration) — only if the `--switch` flag was provided OR switching intent is detected in the description (phrases like "and switch to it", "make it active", "activate it")

Note: `/fab-new` does not activate changes by default — this reduces disruption when capturing change ideas. Branch integration is delegated to `/fab-switch`, which provides consistent branch handling.

#### `--switch` Flag and Natural Language Detection

By default, `/fab-new` does NOT activate the newly created change — `fab/current` is not modified and no branch is created or checked out. The output omits the `Branch:` line and suggests: `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff`.

To activate the change automatically, either:
1. **Use the `--switch` flag**: `/fab-new "description" --switch`
2. **Include switching intent in the description**: Phrases like "and switch to it", "make it active", "activate it", "switch to it", "set as active", or "and activate" will be detected (case-insensitive)

When switching occurs (via flag or detection), the output includes the `Branch:` line and uses: `Next: /fab-continue or /fab-ff (fast-forward all planning)`.

#### Brief-Only Output

`/fab-new` produces a single artifact: `brief.md`. It does not generate `spec.md` or any other downstream artifacts. The brief includes an **Origin** section recording how the change was initiated (description text, conversational vs. one-shot mode, key decisions from the conversation).

#### Confidence Scoring

After generating the brief, `/fab-new` computes the SRAD confidence score and writes the `confidence` block to `.status.yaml` with actual counts (overwriting the template defaults of zero counts and score 5.0). This ensures briefs created via `/fab-new` have a valid score for the `/fab-fff` gate.

#### Context

Loads: config, constitution, `fab/docs/index.md` (to understand the existing doc landscape).

### `/fab-continue [<stage>]`

`/fab-continue` advances to the next pipeline stage — planning, implementation, review, or archive — and either generates the artifact or executes the stage's behavior. When called with a stage argument, it resets to that stage. The pipeline flows brief → spec → tasks → apply → review → archive.

#### Normal Forward Flow (no argument)

1. Read `.status.yaml` to determine current stage (the stage with `active` in the progress map)
2. **Stage guard**: Check `progress.{stage}` value from preflight output:
   - For planning stages (brief, spec, tasks): if `progress.{stage} == 'done'` AND stage is `tasks`, transition to apply. If `progress.{stage} == 'active'`, allow generation to resume. If `progress.{stage} == 'pending'`, allow generation to start.
   - For execution stages (apply, review, archive): dispatch to the stage's behavior (apply executes tasks, review validates implementation, archive completes the change).
3. Identify next artifact to create
4. Load relevant template + context (including `fab/constitution.md` for principles)
5. Generate artifact using the shared generation procedures from `_generation.md` (with clarification/research as needed)
6. Recompute confidence score (re-count SRAD grades across all artifacts, apply formula, update `.status.yaml`)
7. Auto-generate checklist when creating tasks (using `_generation.md` Checklist Generation Procedure)
8. Update `.status.yaml`

#### Reset Behavior (with stage argument)

When called as `/fab-continue <stage>` (e.g., `/fab-continue spec`):
1. Target stage can be any of the 6 stages: `brief`, `spec`, `tasks`, `apply`, `review`, `archive`
2. Reset `.status.yaml` progress: set target stage to `active`; mark all stages after target as `pending`
3. Regenerate the target stage's artifact in place (update, not recreate from scratch — preserve what's still valid)
4. Downstream artifacts are invalidated: tasks reset to `- [ ]`, checklist regenerated
5. Update `.status.yaml` and report what was reset

Reset is primarily used after review identifies issues upstream.

#### Context (varies by target stage)

- **Spec**: config, constitution, `brief.md`, target centralized doc(s) from `fab/docs/`
- **Tasks**: above + completed `spec.md`

### `/fab-ff` (Fast Forward — Full Pipeline)

`/fab-ff` runs the entire Fab pipeline in a single invocation: planning (spec, tasks) → apply → review → archive. It frontloads questions, interleaves auto-clarify between planning stages, and stops for interactive resolution when blocking issues arise at any phase. No confidence gate.

#### Frontloaded Questions

The skill SHALL scan the brief for ambiguities across *all* remaining planning stages (brief, spec, tasks), collect everything that needs user input into a single batch, and ask once. The goal: one Q&A round, then heads-down generation.

#### Interleaved Auto-Clarify

The `/fab-ff` pipeline interleaves auto-clarify between planning stage generations: `spec → auto-clarify → tasks → auto-clarify`. Each auto-clarify invocation uses the `[AUTO-MODE]` prefix defined in the Skill Invocation Protocol (`_context.md`) to signal `/fab-clarify` to operate autonomously. This catches gaps before they compound downstream.

- If auto-clarify finds **blocking issues** (cannot resolve autonomously), the pipeline **bails** — stops, reports the issues, and suggests `Run /fab-clarify to resolve these, then /fab-ff to resume.`
- The pipeline is **resumable** — re-running `/fab-ff` after a bail skips stages already marked `done` and continues from the first incomplete stage.

#### Pipeline Flow

1. Read `fab/current` to resolve the active change; verify brief is complete
2. Frontload questions (single batch)
3. Generate `spec.md` (incorporating answers) → run auto-clarify on spec
4. Produce task breakdown (referencing spec and brief) → run auto-clarify on tasks
5. Auto-generate quality checklist
6. Execute tasks via apply behavior
7. Validate implementation via review behavior — on failure, presents interactive rework menu (fix code, revise tasks, revise spec)
8. Hydrate docs and archive the change

#### Interactive Review Failure

Unlike `/fab-fff` which bails immediately on review failure, `/fab-ff` presents an interactive rework menu on review failure. This is the key behavioral difference: `/fab-ff` is "fast but interactive" while `/fab-fff` is "fully autonomous."

#### When to Use

- Want the full pipeline in one command but with the ability to intervene
- Clear requirements upfront, want to reach archive quickly with safety nets
- Changes needing quality gates — auto-clarify catches issues between planning stages

### `/fab-fff` (Full Autonomous Pipeline)

`/fab-fff` runs the entire Fab pipeline autonomously in a single invocation, gated on confidence score >= 3.0. Unlike `/fab-ff`, which stops for interactive clarification, `/fab-fff` never stops for user input — it bails immediately on review failure and auto-clarifies without user interaction.

#### Confidence Gate

Before proceeding, `/fab-fff` reads `confidence.score` from `.status.yaml`. If the score is below 3.0, the skill aborts with a message suggesting `/fab-clarify` to raise confidence.

#### Pipeline Behavior

Each stage uses the same behavior as its standalone invocation. If planning bails on blocking issues or review fails, the pipeline stops immediately. Unlike `/fab-ff` (which offers interactive rework on review failure), `/fab-fff` bails with an actionable message and no interactive menu.

#### Resumability

`/fab-fff` is resumable — re-invoking skips stages already marked `done` and continues from the first incomplete stage.

#### Confidence Recomputation

`/fab-fff` does NOT recompute the confidence score during execution. The gate check uses the score from the last manual step (`/fab-new`, `/fab-continue`, or `/fab-clarify`).

#### When to Use

- High-confidence changes where you want full autonomy from planning to archive
- After raising confidence via `/fab-clarify` to meet the >= 3.0 threshold

#### Context

Loads all planning context upfront: config, constitution, `brief.md`, target centralized doc(s) from `fab/docs/`.

### `/fab-clarify`

`/fab-clarify` deepens and refines the current stage artifact without advancing to the next stage. It operates in two modes depending on call context: **suggest mode** (user invocation) and **auto mode** (internal `fab-ff` call). It is idempotent and non-advancing. See [clarify.md](clarify.md) for the detailed dual-mode specification.

#### Suggest Mode (User Invocation)

When the user invokes `/fab-clarify` directly:

1. Read `.status.yaml` to determine current stage
2. Stage MUST be `brief`, `spec`, or `tasks`. Each stage scans its corresponding artifact(s) using per-artifact taxonomy. If `apply` or later, suggest `/fab-continue` instead
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

- **Spec**: config, constitution, `brief.md`, target centralized doc(s) from `fab/docs/`
- **Tasks**: above + `spec.md`, `tasks.md`

## Design Decisions

### SRAD Autonomy Framework
**Decision**: All planning skills use the SRAD framework (Signal Strength, Reversibility, Agent Competence, Disambiguation Type) to evaluate decision points and assign confidence grades (Certain, Confident, Tentative, Unresolved). Each skill has a defined autonomy level and interruption budget.
**Why**: Replaces ad-hoc question selection with a principled, consistent framework. Ensures high-blast-radius decisions are always surfaced while low-value prompts are eliminated. The four-dimension scoring prevents both over-asking and silent high-risk assumptions.
**Rejected**: Ad-hoc question selection — inconsistent, no way to predict agent behavior. Full autonomy — too risky for Unresolved decisions with cascading consequences.
*Introduced by*: 260207-09sj-autonomy-framework

### Brief-First with SRAD-Based Questions
**Decision**: Every change starts with a brief. The agent applies SRAD scoring to identify up to 3 Unresolved decisions with the highest blast radius and asks those. All other decisions are assumed at their assessed confidence grade and surfaced in the Assumptions summary.
**Why**: Prevents question-paralysis while catching the decisions that actually matter. SRAD scoring replaces gut-feel question selection with a repeatable evaluation method.
**Rejected**: Unlimited clarification rounds — too many back-and-forth exchanges. Fixed 3-question cap without SRAD — may ask the wrong 3 questions.
*Source*: doc/fab-spec/TEMPLATES.md, doc/fab-spec/README.md; *Updated by*: 260207-09sj-autonomy-framework

### Fast-Forward Frontloads All Questions
**Decision**: `/fab-ff` collects questions across all planning stages into a single batch before generating any artifacts.
**Why**: Maintains the fast-forward promise — one Q&A round, then heads-down generation. Multiple interruptions would defeat the purpose.
**Rejected**: Per-stage questions during ff (defeats fast-forward flow). No questions at all (too risky for ambiguous proposals).
*Source*: doc/fab-spec/SKILLS.md

### Clarify is Non-Advancing
**Decision**: `/fab-clarify` never transitions to the next stage. It refines in place.
**Why**: Separates the concerns of "deepen the current work" from "move forward." The user explicitly chooses when to advance via `/fab-continue`.
**Rejected**: Auto-advancing after clarification — unclear when the user considers the artifact ready.
*Source*: doc/fab-spec/SKILLS.md

### Clarify Mode Selection by Call Context
**Decision**: `/fab-clarify` mode is determined by the `[AUTO-MODE]` prefix defined in the Skill Invocation Protocol (`_context.md`). When the prefix is present (e.g., `/fab-ff` invoking internally), `/fab-clarify` enters auto mode. When absent (user invocation), it enters suggest mode. No `--suggest`/`--auto` flags.
**Why**: Avoids a confusing flag pair with no clear use case for user-invoked auto mode. The explicit prefix protocol makes the contract testable rather than relying on implicit call-context interpretation.
**Rejected**: Flag-based mode selection — adds complexity, no user scenario requires it. Implicit call-context detection — unreliable, not testable.
*Introduced by*: 260207-m3qf-clarify-dual-modes; *Updated by*: 260210-nan4-define-auto-mode-signaling (explicit `[AUTO-MODE]` protocol)

### Fast-Forward Interleaves Auto-Clarify
**Decision**: `/fab-ff` interleaves auto-clarify between stage generations (`spec → auto-clarify → tasks → auto-clarify`). Bails on blocking issues that cannot be resolved autonomously.
**Why**: Gaps in one stage compound downstream. Catching them between stages prevents tasks built on unverified assumptions.
**Rejected**: No clarify in ff (gaps compound). Full user-interactive clarify in ff (defeats fast-forward flow). Full-auto mode with `<!-- auto-guess -->` markers (defers interaction rather than eliminating it — replaced by confidence-gated `/fab-fff`).
*Introduced by*: 260207-m3qf-clarify-dual-modes; *Updated by*: 260208-k3m7-add-fab-fff (removed `--auto` mode)

### /fab-new as Single Adaptive Entry Point
**Decision**: Consolidate `/fab-new` and `/fab-discuss` into a single `/fab-new` that adapts via SRAD scoring.
**Why**: Three overlapping entry paths created confusion.
**Rejected**: Keeping both skills with clearer differentiation.
*Introduced by*: 260212-v5p2-simplify-stages-entry-paths

### Shared Generation Partial
**Decision**: Extract duplicated artifact generation logic from `/fab-continue` and `/fab-ff` into a shared `_generation.md` partial. Both skills reference the partial for spec, tasks, and checklist generation mechanics; each retains its own orchestration logic.
**Why**: Generation steps were nearly identical in both skills, requiring every fix or behavior change to be applied in two places. Centralizing eliminates drift and makes generation behavior authoritative in one location.
**Rejected**: Keeping inline duplication — inevitable drift between the two copies.
*Introduced by*: 260210-wpay-extract-shared-generation-logic

### Unified Command: `/fab-continue` Absorbs Execution Stages
**Decision**: `/fab-continue` handles all 6 pipeline stages (brief → spec → tasks → apply → review → archive). Apply, review, and archive behaviors are described as dedicated sections within `fab-continue.md`, not extracted into a shared partial.
**Why**: Reduces developer command surface from 4+ commands to 2 (`/fab-continue` + `/fab-clarify`). Execution stages are orchestration-heavy with distinct flows (task execution, validation with rework, hydration with folder moves) — inlining keeps each stage's behavior in one readable location.
**Rejected**: Keeping standalone `/fab-apply`, `/fab-review`, `/fab-archive` — command fragmentation. Extracting to `_execution.md` partial — low reuse value since only fab-continue calls these.
*Introduced by*: 260212-a4bd-unify-fab-continue

### `/fab-ff` and `/fab-fff` Keep Behavioral Descriptions
**Decision**: `/fab-ff` and `/fab-fff` describe execution behavior inline within their own orchestration context, rather than literally invoking `/fab-continue` as a sub-skill.
**Why**: These skills have fundamentally different orchestration: frontloaded questions, auto-clarify interleaving, bail behavior, resumability across all stages. Literal sub-skill invocation would add complexity (nested preflight checks, status conflicts) without benefit.
**Rejected**: Literal `/fab-continue` invocation from fab-ff/fff — orchestration mismatch, nested state management issues.
*Introduced by*: 260212-a4bd-unify-fab-continue

### Reset via `/fab-continue <stage>`
**Decision**: Reset to any pipeline stage by passing the stage name as an argument to `/fab-continue`. For planning stages, downstream artifacts are invalidated and regenerated. For execution stages, the stage behavior is re-run without resetting task checkboxes.
**Why**: Provides a clean re-entry point after review identifies upstream issues. Reuses the existing skill rather than adding a separate `/fab-reset` command. Covers all 6 stages (brief, spec, tasks, apply, review, archive).
**Rejected**: Separate reset skill — unnecessary proliferation of skills for a rare operation.
*Source*: doc/fab-spec/SKILLS.md; *Updated by*: 260212-a4bd-unify-fab-continue (extended to all 6 stages)

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260212-a4bd-unify-fab-continue | 2026-02-12 | Unified `/fab-apply`, `/fab-review`, `/fab-archive` into `/fab-continue`. Updated stage guard, reset behavior, and cross-references to reflect unified command |
| 260212-ipoe-checklist-folder-location | 2026-02-12 | Updated checklist generation and validation paths from `checklists/quality.md` to `checklist.md` in `/fab-continue`, `/fab-ff`, and shared generation partial |
| 260212-bk1n-rework-fab-ff-archive | 2026-02-12 | Extended `/fab-ff` from planning-only to full pipeline (planning → apply → review → archive). Updated `/fab-fff` description and comparison table to reflect new differentiation. `/fab-ff` now offers interactive rework on review failure; `/fab-fff` remains fully autonomous with confidence gate |
| 260212-29xv-scoring-formula | 2026-02-12 | Increased Confident penalty from 0.1 to 0.3 in confidence formula; `/fab-clarify` now reclassifies resolved assumptions (Tentative/Confident → Certain) so scores increase after clarification |
| 260212-k7m3-fix-consistency-drift | 2026-02-12 | Clarified confidence score template default phrasing ("zero counts and score 5.0" instead of "all zeros") |
| 260212-0r8e-fix-created-by-github | 2026-02-12 | `/fab-new` now uses `gh api user --jq .login` as primary source for `created_by`, with `git config user.name` as fallback |
| — | 2026-02-12 | Reversed `/fab-new` default behavior: no longer auto-switches to new changes. Replaced `--no-switch` with `--switch` flag, added natural language switching detection. Default output now suggests `/fab-switch {name}` command |
| 260212-r7k3-add-no-switch-flag | 2026-02-12 | Added `--no-switch` flag to `/fab-new` — skips activation and branch integration when batching change captures |
| 260212-v5p2-simplify-stages-entry-paths | 2026-02-12 | Removed /fab-discuss section, rewrote /fab-new for adaptive SRAD-driven behavior with gap analysis and conversational mode |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | 6-stage pipeline (brief → spec → tasks), removed plan stage, /fab-discuss dual output, /fab-ff generates spec → tasks directly |
| 260211-endg-add-created-by-field | 2026-02-11 | `/fab-new` and `/fab-discuss` now populate `created_by` in `.status.yaml` from `git config user.name` at change creation |
| 260210-wpay-extract-shared-generation-logic | 2026-02-10 | Extracted shared generation logic (spec, tasks, checklist) into `_generation.md` partial; both `/fab-continue` and `/fab-ff` now reference it |
| 260210-nan4-define-auto-mode-signaling | 2026-02-10 | Defined explicit `[AUTO-MODE]` prefix protocol for skill-to-skill invocation in `_context.md`; updated `/fab-ff` auto-clarify invocations and "Clarify Mode Selection" design decision |
| 260210-0p4e-fix-stage-guard-progress-check | 2026-02-10 | `/fab-continue` stage guard now checks `progress.{stage}` value to distinguish done/active/pending states, allowing resumption of interrupted stage generations |
| 260210-zr1f-discuss-auto-activate-when-no-current | 2026-02-10 | `/fab-discuss` conditionally offers activation when `fab/current` is empty; updated proposal output, key differences table |
| 260209-r4w8-archive-index-longer-slugs | 2026-02-09 | Expanded slug word count from 2-4 to 2-6 words in `/fab-new` folder name generation |
| 260208-q8v3-branch-to-switch | 2026-02-09 | Moved branch integration from `/fab-new` to `/fab-switch`, removed `--branch` flag from `/fab-new`, `/fab-new` now calls `/fab-switch` internally |
| 260208-lgd7-fab-discuss-command | 2026-02-08 | Added `/fab-discuss` conversational brief skill, `/fab-new` confidence scoring, context-driven mode selection design decisions |
| 260208-k3m7-add-fab-fff | 2026-02-08 | Added `/fab-fff` full pipeline skill, confidence recomputation in `/fab-continue`, removed `/fab-ff --auto` mode, updated design decisions |
| 260207-09sj-autonomy-framework | 2026-02-08 | Added SRAD autonomy framework, confidence grades, assumptions summaries, branch auto-create on main, soft gate on fab-apply |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260207-m3qf-clarify-dual-modes | 2026-02-07 | Updated `/fab-clarify` to dual-mode (suggest + auto), `/fab-ff` with interleaved auto-clarify and `--auto` flag |
| — | 2026-02-07 | Generated from doc/fab-spec/ (README.md, SKILLS.md, TEMPLATES.md) |
