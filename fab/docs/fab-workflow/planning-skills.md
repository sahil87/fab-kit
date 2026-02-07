# Planning Skills

**Domain**: fab-workflow

## Overview

The planning skills (`/fab-new`, `/fab-continue`, `/fab-ff`, `/fab-clarify`) handle the first four stages of the Fab workflow: proposal, specs, plan, and tasks. They produce the artifacts that define *what* changes and *how*, before any code is written.

## Requirements

### `/fab-new <description> [--branch <name>]`

`/fab-new` starts a new change from a natural language description. It creates the change folder, initializes status tracking, and generates a proposal.

#### Folder Name Generation

The agent SHALL generate a folder name in the format `{YYMMDD}-{XXXX}-{slug}` where YYMMDD is today's date, XXXX is 4 random lowercase alphanumeric chars, and slug is 2-4 words extracted from the description. All components MUST be lowercase.

#### Change Initialization

The skill SHALL:
1. Create `fab/changes/{name}/`
2. Write the change name to `fab/current` (sets active change)
3. Initialize `.status.yaml` with `stage: proposal`
4. Generate `proposal.md` from the template, loading `fab/constitution.md` and `fab/config.yaml` as context
5. Ask clarifying questions if intent is ambiguous
6. Mark proposal complete once the user is satisfied

#### Branch Integration

When `git.enabled` in config and the project is a git repo:
- If `--branch <name>` is provided → use that name directly (create if new, adopt if existing)
- If on `main`/`master` → offer to create a branch named `{prefix}{change-name}`
- If on a feature branch → offer to adopt it (record current branch name as-is)
- If user declines → skip; no `branch:` field in `.status.yaml`

The `--branch` flag is useful for Linear-linked branches, team conventions, or pre-existing branches.

#### Context

Loads: config, constitution, `fab/docs/index.md` (to understand the existing doc landscape).

### `/fab-continue [<stage>]`

`/fab-continue` advances to the next planning stage and generates its artifact. When called with a stage argument, it resets to that stage and regenerates from there.

#### Normal Forward Flow (no argument)

1. Read `.status.yaml` to determine current stage
2. Identify next artifact to create
3. Load relevant template + context (including `fab/constitution.md` for principles)
4. Generate artifact (with clarification/research as needed)
5. Auto-generate checklist when creating tasks
6. Update `.status.yaml`

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

`/fab-ff` fast-forwards through all remaining planning stages in one pass to reach implementation quickly. It requires an active change with a completed proposal. Supports two modes: default (with clarify checkpoints) and full-auto (`--auto`).

#### Frontloaded Questions

The skill SHALL scan the proposal for ambiguities across *all* planning stages (specs, plan, tasks), collect everything that needs user input into a single batch, and ask once. The goal: one Q&A round, then heads-down generation.

#### Interleaved Auto-Clarify

The default `/fab-ff` pipeline interleaves auto-clarify between stage generations: `spec → auto-clarify → plan-decision → auto-clarify → tasks → auto-clarify`. This catches gaps before they compound downstream.

- If auto-clarify finds **blocking issues** (cannot resolve autonomously), the pipeline **bails** — stops, reports the issues, and suggests `Run /fab-clarify to resolve these, then /fab-ff to resume.`
- The pipeline is **resumable** — re-running `/fab-ff` after a bail skips stages already marked `done` and continues from the first incomplete stage.

#### Full-Auto Mode (`--auto`)

`/fab-ff --auto` runs the same interleaved pipeline but never stops for blockers. Instead, it makes best-guess decisions, marks them with `<!-- auto-guess: {description} -->` markers in the artifact, and warns the user in output listing all guesses made. These markers are detectable by `/fab-review` and resolvable by `/fab-clarify` suggest mode.

#### Generation Flow

1. Read `fab/current` to resolve the active change; verify proposal is complete
2. Frontload questions (single batch)
3. Generate `spec.md` (incorporating answers) → run auto-clarify on spec
4. Evaluate whether a plan is warranted. Unlike `/fab-continue`, `/fab-ff` does **not** confirm with the user before skipping — it decides autonomously to maintain fast-forward flow → run auto-clarify on plan (if generated)
5. Produce task breakdown (referencing plan if it exists, otherwise referencing spec and proposal directly) → run auto-clarify on tasks
6. Auto-generate quality checklist
7. Update status to `tasks: done`

#### When to Use

- **Default `/fab-ff`**: Changes needing quality gates — auto-clarify catches issues between stages
- **`/fab-ff --auto`**: Quick changes with high agent trust — never bails, marks guesses for later review
- Both: Clear requirements upfront, want to reach implementation quickly

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

### Proposal-First with Capped Blocking Questions
**Decision**: Every change starts with a proposal that may include up to 3 [BLOCKING] questions. The agent makes informed guesses for ambiguities beyond the cap.
**Why**: Prevents question-paralysis while still catching showstopper ambiguities. Forces the agent to be opinionated rather than deferring everything to the user.
**Rejected**: Unlimited clarification rounds — too many back-and-forth exchanges before any artifact is produced.
*Source*: doc/fab-spec/TEMPLATES.md, doc/fab-spec/README.md

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
**Decision**: `/fab-ff` interleaves auto-clarify between stage generations (`spec → auto-clarify → plan → auto-clarify → tasks → auto-clarify`). Default mode bails on blocking issues; `--auto` mode guesses and marks them.
**Why**: Gaps in one stage compound downstream. Catching them between stages prevents tasks built on unverified assumptions. The bail/guess split gives users control vs speed.
**Rejected**: No clarify in ff (gaps compound). Full user-interactive clarify in ff (defeats fast-forward flow).
*Introduced by*: 260207-m3qf-clarify-dual-modes

### Reset via `/fab-continue <stage>`
**Decision**: Reset to an earlier planning stage by passing the stage name as an argument to `/fab-continue`. Downstream artifacts are invalidated and regenerated.
**Why**: Provides a clean re-entry point after `/fab-review` identifies upstream issues. Reuses the existing skill rather than adding a separate `/fab-reset` command.
**Rejected**: Separate reset skill — unnecessary proliferation of skills for a rare operation.
*Source*: doc/fab-spec/SKILLS.md

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab:xxx` colon format to `/fab-xxx` hyphen format |
| 260207-m3qf-clarify-dual-modes | 2026-02-07 | Updated `/fab-clarify` to dual-mode (suggest + auto), `/fab-ff` with interleaved auto-clarify and `--auto` flag |
| — | 2026-02-07 | Generated from doc/fab-spec/ (README.md, SKILLS.md, TEMPLATES.md) |
