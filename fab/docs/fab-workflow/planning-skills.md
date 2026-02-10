# Planning Skills

**Domain**: fab-workflow

## Overview

The planning skills (`/fab-new`, `/fab-discuss`, `/fab-continue`, `/fab-ff`, `/fab-clarify`) handle the first four stages of the Fab workflow: proposal, specs, plan, and tasks. They produce the artifacts that define *what* changes and *how*, before any code is written.

## Requirements

### `/fab-new <description>`

`/fab-new` starts a new change from a natural language description. It creates the change folder, initializes status tracking, generates a proposal, and calls `/fab-switch` internally to activate the change (including branch integration).

#### Folder Name Generation

The agent SHALL generate a folder name in the format `{YYMMDD}-{XXXX}-{slug}` where YYMMDD is today's date, XXXX is 4 random lowercase alphanumeric chars, and slug is 2-6 words extracted from the description. All components MUST be lowercase.

#### Change Initialization

The skill SHALL:
1. Create `fab/changes/{name}/`
2. Initialize `.status.yaml` with `stage: proposal`
3. Generate `proposal.md` from the template, loading `fab/constitution.md` and `fab/config.yaml` as context
4. Apply SRAD scoring to identify up to 3 Unresolved questions; assume all Confident/Tentative decisions
5. Mark proposal complete once the user is satisfied
6. Call `/fab-switch` internally to activate the change (writes `fab/current`, performs branch integration)

Note: `/fab-new` no longer handles branch integration directly — this is delegated to `/fab-switch`, which provides consistent branch handling for both `/fab-new` and `/fab-discuss` entry points.

#### Confidence Scoring

After generating the proposal, `/fab-new` computes the SRAD confidence score and writes the `confidence` block to `.status.yaml` with actual counts (overwriting the template defaults of all zeros). This ensures proposals created via `/fab-new` have a valid score for the `/fab-fff` gate.

#### Context

Loads: config, constitution, `fab/docs/index.md` (to understand the existing doc landscape).

### `/fab-discuss [description]`

`/fab-discuss` develops proposals through free-form conversation. Unlike `/fab-new` (one-shot capture with max 3 questions), `/fab-discuss` is a back-and-forth exploration — it helps figure out if a change is even needed, walks through clarifying questions, and outputs a solid `proposal.md` with a high confidence score.

#### Context-Driven Mode Selection

`/fab-discuss` determines its mode from the active change state (`fab/current`):

1. If `fab/current` exists and points to a valid change, `/fab-discuss` defaults to **Refine mode** — working on the active change's proposal
2. If the user's description is significantly different from the active change's scope, `/fab-discuss` confirms whether this is a new change or related to the current one
3. If `fab/current` does not exist or is empty, `/fab-discuss` enters **New change mode** — starts from scratch

#### Gap Analysis (New Change Mode Only)

Before committing to a proposal, `/fab-discuss` evaluates whether the change is needed:

1. Checks for existing mechanisms in the current workflow, codebase, or docs
2. Evaluates scope — is the idea too broad (should be split) or too narrow (part of something larger)?
3. Considers alternatives — simpler approaches, extending existing skills

If an existing mechanism covers the idea, the skill presents its findings and lets the user decide whether to proceed. If no change folder is created, no `Next:` line is shown.

#### Conversational Proposal Development

The skill develops the proposal through back-and-forth conversation with no fixed question cap. Each question builds on previous answers, starting with the highest-impact decisions (lowest Reversibility + lowest Agent Competence). SRAD grades are tracked for each decision point throughout the conversation.

#### Conversation Termination

The discussion ends when both conditions are met:
1. The confidence score is >= 3.0
2. The user signals satisfaction (e.g., "looks good", "done")

When the confidence score crosses 3.0, `/fab-discuss` proactively suggests wrapping up. The user may also end the discussion early at any time regardless of the current score.

#### Proposal Output

**New change mode**: Creates the change folder, `checklists/` subdirectory, `.status.yaml` (without `branch:` field — no git integration), and `proposal.md`. Sets `progress.proposal` to `done`. After displaying the summary, checks whether `fab/current` is empty — if so, offers to activate the new change via internal `/fab-switch` (writes `fab/current` and handles branch integration). If `fab/current` already points to another change, no offer is made.

**Refine mode**: Updates the existing `proposal.md` in place, recomputes the confidence score, and updates `.status.yaml`.

#### Key Differences from `/fab-new`

| Aspect | fab-discuss | fab-new |
|--------|-------------|---------|
| **Purpose** | Explore & develop proposal through conversation | Capture clear description as proposal |
| **Gap analysis** | Yes — "is this change even needed?" | No — assumes the change is needed |
| **Interaction style** | Free-form conversation, unlimited questions | One-shot generation, max 3 SRAD questions |
| **Sets active change** | Conditionally — offers when no active change | Yes (via internal `/fab-switch` call) |
| **Git integration** | Conditionally — via internal `/fab-switch` when activation accepted | Yes (via internal `/fab-switch` call) |
| **Confidence goal** | Drive score high for `/fab-fff` | Compute initial score |

#### Context

Loads: config, constitution, `fab/docs/index.md`, `fab/specs/index.md`. In refine mode, also loads the active change's `proposal.md` and `.status.yaml`.

### `/fab-continue [<stage>]`

`/fab-continue` advances to the next planning stage and generates its artifact. When called with a stage argument, it resets to that stage and regenerates from there.

#### Normal Forward Flow (no argument)

1. Read `.status.yaml` to determine current stage
2. **Stage guard**: Check both `stage` field and `progress.{stage}` value from preflight output:
   - For planning stages (proposal, specs, plan, tasks): if `progress.{stage} == 'done'` AND stage is `tasks`, block (planning complete). If `progress.{stage} == 'active'`, allow generation to resume (interrupted mid-way). If `progress.{stage} == 'pending'`, allow generation to start.
   - For apply/review/archive stages: block regardless of progress value (use stage-specific skill instead).
3. Identify next artifact to create
4. Load relevant template + context (including `fab/constitution.md` for principles)
5. Generate artifact (with clarification/research as needed)
6. Recompute confidence score (re-count SRAD grades across all artifacts, apply formula, update `.status.yaml`)
7. Auto-generate checklist when creating tasks
8. Update `.status.yaml`

#### Plan Decision

When transitioning from specs to plan, the agent SHALL evaluate whether a plan is warranted. If the change is small and the approach is obvious, propose skipping: "This change is straightforward — skip plan and go directly to tasks?" If user agrees, record `plan: skipped` in `.status.yaml` and proceed to tasks.

#### Reset Behavior (with stage argument)

When called as `/fab-continue <stage>` (e.g., `/fab-continue specs`):
1. Target stage MUST be `specs`, `plan`, or `tasks`. Cannot reset to `proposal` (use `/fab-new`) or `apply`/`review`/`archive`
2. Reset `.status.yaml` stage to the target; mark all stages from target onward as `pending`
3. Regenerate the target stage's artifact in place (update, not recreate from scratch — preserve what's still valid)
4. Downstream artifacts are invalidated: tasks reset to `- [ ]`, checklist regenerated. Plan regenerated if target is `specs`
5. Update `.status.yaml` and report what was reset

Reset is primarily used after `/fab-review` identifies issues upstream.

#### Context (varies by target stage)

- **Specs**: config, constitution, `proposal.md`, target centralized doc(s) from `fab/docs/`
- **Plan**: above + completed `spec.md`
- **Tasks**: above + `plan.md` (if not skipped)

### `/fab-ff` (Fast Forward)

`/fab-ff` fast-forwards through all remaining planning stages in one pass to reach implementation quickly. It requires an active change with a completed proposal. Frontloads questions, interleaves auto-clarify, and bails on blockers.

#### Frontloaded Questions

The skill SHALL scan the proposal for ambiguities across *all* planning stages (specs, plan, tasks), collect everything that needs user input into a single batch, and ask once. The goal: one Q&A round, then heads-down generation.

#### Interleaved Auto-Clarify

The `/fab-ff` pipeline interleaves auto-clarify between stage generations: `spec → auto-clarify → plan-decision → auto-clarify → tasks → auto-clarify`. This catches gaps before they compound downstream.

- If auto-clarify finds **blocking issues** (cannot resolve autonomously), the pipeline **bails** — stops, reports the issues, and suggests `Run /fab-clarify to resolve these, then /fab-ff to resume.`
- The pipeline is **resumable** — re-running `/fab-ff` after a bail skips stages already marked `done` and continues from the first incomplete stage.

#### Generation Flow

1. Read `fab/current` to resolve the active change; verify proposal is complete
2. Frontload questions (single batch)
3. Generate `spec.md` (incorporating answers) → run auto-clarify on spec
4. Evaluate whether a plan is warranted. Unlike `/fab-continue`, `/fab-ff` does **not** confirm with the user before skipping — it decides autonomously to maintain fast-forward flow → run auto-clarify on plan (if generated)
5. Produce task breakdown (referencing plan if it exists, otherwise referencing spec and proposal directly) → run auto-clarify on tasks
6. Auto-generate quality checklist
7. Update status to `tasks: done`

#### When to Use

- Clear requirements upfront, want to reach implementation quickly
- Changes needing quality gates — auto-clarify catches issues between stages

### `/fab-fff` (Full Pipeline)

`/fab-fff` chains the full pipeline in a single invocation: `/fab-ff` → `/fab-apply` → `/fab-review` → `/fab-archive`. Gated on confidence score >= 3.0.

#### Confidence Gate

Before proceeding, `/fab-fff` reads `confidence.score` from `.status.yaml`. If the score is below 3.0, the skill aborts with a message suggesting `/fab-clarify` to raise confidence.

#### Pipeline Behavior

Each stage uses the same behavior as its standalone invocation. If `/fab-ff` bails on blocking issues or `/fab-review` fails, the pipeline stops immediately. Unlike standalone `/fab-review`, the review failure does NOT offer an interactive rework menu — it bails with an actionable message.

#### Resumability

`/fab-fff` is resumable — re-invoking skips stages already marked `done` and continues from the first incomplete stage.

#### Confidence Recomputation

`/fab-fff` does NOT recompute the confidence score during execution. The gate check uses the score from the last manual step (`/fab-new`, `/fab-continue`, or `/fab-clarify`).

#### When to Use

- High-confidence changes where you want full autonomy from planning to archive
- After raising confidence via `/fab-clarify` to meet the >= 3.0 threshold

#### Context

Loads all planning context upfront: config, constitution, `proposal.md`, target centralized doc(s) from `fab/docs/`.

### `/fab-clarify`

`/fab-clarify` deepens and refines the current stage artifact without advancing to the next stage. It operates in two modes depending on call context: **suggest mode** (user invocation) and **auto mode** (internal `fab-ff` call). It is idempotent and non-advancing. See [clarify.md](clarify.md) for the detailed dual-mode specification.

#### Suggest Mode (User Invocation)

When the user invokes `/fab-clarify` directly:

1. Read `.status.yaml` to determine current stage
2. Stage MUST be `proposal`, `specs`, `plan`, or `tasks`. If `apply` or later, suggest `/fab-review` instead
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

- **Proposal**: config, constitution, `proposal.md`
- **Specs**: above + target centralized doc(s) from `fab/docs/`
- **Plan**: above + `spec.md`, `plan.md`
- **Tasks**: above + `plan.md` (if not skipped), `tasks.md`

## Design Decisions

### SRAD Autonomy Framework
**Decision**: All planning skills use the SRAD framework (Signal Strength, Reversibility, Agent Competence, Disambiguation Type) to evaluate decision points and assign confidence grades (Certain, Confident, Tentative, Unresolved). Each skill has a defined autonomy level and interruption budget.
**Why**: Replaces ad-hoc question selection with a principled, consistent framework. Ensures high-blast-radius decisions are always surfaced while low-value prompts are eliminated. The four-dimension scoring prevents both over-asking and silent high-risk assumptions.
**Rejected**: Ad-hoc question selection — inconsistent, no way to predict agent behavior. Full autonomy — too risky for Unresolved decisions with cascading consequences.
*Introduced by*: 260207-09sj-autonomy-framework

### Proposal-First with SRAD-Based Questions
**Decision**: Every change starts with a proposal. The agent applies SRAD scoring to identify up to 3 Unresolved decisions with the highest blast radius and asks those. All other decisions are assumed at their assessed confidence grade and surfaced in the Assumptions summary.
**Why**: Prevents question-paralysis while catching the decisions that actually matter. SRAD scoring replaces gut-feel question selection with a repeatable evaluation method.
**Rejected**: Unlimited clarification rounds — too many back-and-forth exchanges. Fixed 3-question cap without SRAD — may ask the wrong 3 questions.
*Source*: doc/fab-spec/TEMPLATES.md, doc/fab-spec/README.md; *Updated by*: 260207-09sj-autonomy-framework

### Plan Stage is Optional
**Decision**: `/fab-continue` may propose skipping the plan for straightforward changes. `/fab-ff` skips autonomously. Status records `plan: skipped`.
**Why**: Many changes are small and well-understood. Forcing a plan adds overhead without value. The spec and proposal provide enough context for task generation.
**Rejected**: Always requiring a plan — adds friction to simple changes. Never having a plan — misses architectural decisions for complex changes.
*Source*: doc/fab-spec/SKILLS.md

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
**Decision**: `/fab-clarify` mode is determined by how it is invoked (user = suggest mode, `fab-ff` internal = auto mode), not by `--suggest`/`--auto` flags.
**Why**: Avoids a confusing flag pair with no clear use case for user-invoked auto mode. The call context naturally maps to the right behavior.
**Rejected**: Flag-based mode selection — adds complexity, no user scenario requires it.
*Introduced by*: 260207-m3qf-clarify-dual-modes

### Fast-Forward Interleaves Auto-Clarify
**Decision**: `/fab-ff` interleaves auto-clarify between stage generations (`spec → auto-clarify → plan → auto-clarify → tasks → auto-clarify`). Bails on blocking issues that cannot be resolved autonomously.
**Why**: Gaps in one stage compound downstream. Catching them between stages prevents tasks built on unverified assumptions.
**Rejected**: No clarify in ff (gaps compound). Full user-interactive clarify in ff (defeats fast-forward flow). Full-auto mode with `<!-- auto-guess -->` markers (defers interaction rather than eliminating it — replaced by confidence-gated `/fab-fff`).
*Introduced by*: 260207-m3qf-clarify-dual-modes; *Updated by*: 260208-k3m7-add-fab-fff (removed `--auto` mode)

### Conversational Entry Point via `/fab-discuss`
**Decision**: Add `/fab-discuss` as a separate skill for conversational proposal development, distinct from `/fab-new` (one-shot capture) and `/fab-clarify` (structured artifact refinement).
**Why**: `/fab-new` is optimized for clear ideas (one-shot, max 3 questions). `/fab-clarify` refines existing artifacts. Neither serves the "I have a vague idea, let's figure it out" use case. `/fab-discuss` fills this gap with free-form conversation, gap analysis, and unlimited questions.
**Rejected**: Extending `/fab-new` with a conversation mode — would bloat a focused skill. Using only `/fab-clarify` — it's scoped to artifact gaps, not idea exploration.
*Introduced by*: 260208-lgd7-fab-discuss-command

### Context-Driven Mode Selection for `/fab-discuss`
**Decision**: `/fab-discuss` determines its mode (new vs. refine) from the active change state (`fab/current`), not from argument-based auto-detection.
**Why**: Natural UX — if you're working on something, discussing relates to it by default. If the description diverges significantly, the skill confirms. No special arguments needed.
**Rejected**: Argument-based detection (matching against `fab/changes/` folder names) — brittle, confusing syntax.
*Introduced by*: 260208-lgd7-fab-discuss-command

### Reset via `/fab-continue <stage>`
**Decision**: Reset to an earlier planning stage by passing the stage name as an argument to `/fab-continue`. Downstream artifacts are invalidated and regenerated.
**Why**: Provides a clean re-entry point after `/fab-review` identifies upstream issues. Reuses the existing skill rather than adding a separate `/fab-reset` command.
**Rejected**: Separate reset skill — unnecessary proliferation of skills for a rare operation.
*Source*: doc/fab-spec/SKILLS.md

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260210-0p4e-fix-stage-guard-progress-check | 2026-02-10 | `/fab-continue` stage guard now checks `progress.{stage}` value to distinguish done/active/pending states, allowing resumption of interrupted stage generations |
| 260210-zr1f-discuss-auto-activate-when-no-current | 2026-02-10 | `/fab-discuss` conditionally offers activation when `fab/current` is empty; updated proposal output, key differences table |
| 260209-r4w8-archive-index-longer-slugs | 2026-02-09 | Expanded slug word count from 2-4 to 2-6 words in `/fab-new` folder name generation |
| 260208-q8v3-branch-to-switch | 2026-02-09 | Moved branch integration from `/fab-new` to `/fab-switch`, removed `--branch` flag from `/fab-new`, `/fab-new` now calls `/fab-switch` internally |
| 260208-lgd7-fab-discuss-command | 2026-02-08 | Added `/fab-discuss` conversational proposal skill, `/fab-new` confidence scoring, context-driven mode selection design decisions |
| 260208-k3m7-add-fab-fff | 2026-02-08 | Added `/fab-fff` full pipeline skill, confidence recomputation in `/fab-continue`, removed `/fab-ff --auto` mode, updated design decisions |
| 260207-09sj-autonomy-framework | 2026-02-08 | Added SRAD autonomy framework, confidence grades, assumptions summaries, branch auto-create on main, soft gate on fab-apply |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab:xxx` colon format to `/fab-xxx` hyphen format |
| 260207-m3qf-clarify-dual-modes | 2026-02-07 | Updated `/fab-clarify` to dual-mode (suggest + auto), `/fab-ff` with interleaved auto-clarify and `--auto` flag |
| — | 2026-02-07 | Generated from doc/fab-spec/ (README.md, SKILLS.md, TEMPLATES.md) |
