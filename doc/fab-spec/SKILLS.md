# Fab Skills Reference

> Detailed behavior for each `/fab:*` skill. For a quick overview, see the [Quick Reference](README.md#quick-reference).

---

## Terminology: "spec" vs "docs"

Fab uses two distinct terms to avoid confusion:

| Term | Location | Meaning |
|------|----------|---------|
| **Centralized docs** | `fab/docs/` | Source-of-truth documentation for the system. Contains both requirements (what) and durable design decisions (why). Updated only by `/fab:archive` hydration. |
| **spec.md** | `fab/changes/{name}/spec.md` | Change-level specification. Describes the requirements relevant to this change. |

The stage named "specs" refers to the *activity* of writing the specification — its output is `spec.md` in the change folder.

---

## Context Loading Convention

Every skill that generates or validates artifacts MUST load relevant context before proceeding. This ensures agents produce accurate, grounded output rather than hallucinating requirements or ignoring existing patterns.

**Always loaded** (by every skill except `/fab:init`, `/fab:switch`, `/fab:status`):
- `fab/config.yaml` — project configuration, tech stack, conventions
- `fab/constitution.md` — project principles and constraints

**Change context** (loaded by skills operating on an active change):
- `.status.yaml` — current stage, progress
- All completed artifacts in the active change folder (e.g., `proposal.md`, `spec.md`, `plan.md`)

**Centralized doc lookup** (loaded when writing or validating specs):
- Read `fab/docs/index.md` to understand the doc landscape
- Read the specific centralized doc(s) referenced by the proposal's "Affected Docs" section
- This ensures specs are written against the *actual* current state, not assumptions

**Source code** (loaded during implementation and review):
- Read relevant source files referenced in the plan's "File Changes" section or the task descriptions
- Scope to files actually touched by the change — don't load the entire codebase

Each skill section below lists its specific context requirements under a **Context** field.

---

## `/fab:init`

**Purpose**: Bootstrap `fab/` in an existing project.

**Creates**:
- `fab/.kit/` — engine directory (templates, skills, scripts)
- `fab/config.yaml` — project configuration (prompts for name, tech stack, conventions)
- `fab/constitution.md` — project principles and constraints (generated from conversation or existing docs)
- `fab/docs/index.md` — initial docs index (populated by `/fab:archive` hydration)
- `fab/changes/` — empty, ready for change folders
- `.claude/skills/` — symlinks pointing into `fab/.kit/skills/`

**Example**:
```
/fab:init
→ "What's the project name?"
→ "Describe the tech stack and conventions..."
→ "fab/ initialized with config, templates, and empty docs."
```

**Behavior**:
1. Check if `fab/` already exists (abort if so, suggest manual edits)
2. Prompt for project name, description, tech stack
3. Create `fab/.kit/` with default templates, skills, and scripts
4. Generate `fab/config.yaml` from responses
5. Generate `fab/constitution.md` from project context (README, existing docs, conversation)
6. Create symlinks in `.claude/skills/` pointing to `fab/.kit/skills/`
7. Optionally scaffold initial docs from existing code or documentation

---

## `/fab:new <description> [--branch <name>]`

**Purpose**: Start a new change from a natural language description.

**Context**: config, constitution, `fab/docs/index.md` (to understand existing doc landscape)

**Creates**:
- Change folder named `{YYMMDD}-{XXXX}-{slug}`
- `.status.yaml` manifest
- `proposal.md` from template (with clarifying questions if ambiguous)

**Arguments**:
- `<description>` — natural language description of the change (required)
- `--branch <name>` — explicit branch name to use (optional). Skips the branch prompt and uses this name directly. Useful for Linear-linked branches, team conventions, or pre-existing branches.

**Examples**:
```
/fab:new Add OAuth2 support for Google and GitHub sign-in
→ Created fab/changes/260115-a7k2-add-oauth/
→ Branch: 260115-a7k2-add-oauth (created)

/fab:new --branch feature/dev-907-oauth Add OAuth2 support
→ Created fab/changes/260115-a7k2-add-oauth/
→ Branch: feature/dev-907-oauth (adopted)
```

**Behavior**:
1. Generate folder name: today's date (`YYMMDD`) + 4 random alphanumeric chars + 2-4 word slug from description
2. Create `fab/changes/{name}/`
3. Write change name to `fab/current` (sets this as the active change)
4. **Branch integration** (if `git.enabled` in config and inside a git repo):
   - If `--branch <name>` was provided → use that name directly (create if it doesn't exist, adopt if it does)
   - Else if on `main`/`master` → offer to create a new branch named `{prefix}{change-name}`
   - Else if on a feature branch → offer to adopt it (record current branch name as-is)
   - If user declines → skip, no `branch:` field in `.status.yaml`
   - Record chosen branch name in `.status.yaml` as `branch:`
5. Initialize `.status.yaml` with stage: proposal (and `branch:` if set)
6. Generate `proposal.md` using template (loading `fab/constitution.md` and `fab/config.yaml` as context)
7. Ask clarifying questions if intent is ambiguous
8. Mark proposal complete once the user is satisfied

---

## `/fab:continue [<stage>]`

**Purpose**: Create the next artifact in sequence — or, when called with a stage argument, reset to that stage and regenerate from there.

**Arguments**:
- `<stage>` *(optional)* — target stage to reset to (`specs`, `plan`, or `tasks`). Used after `/fab:review` identifies issues upstream. When provided, resets `.status.yaml` to this stage and regenerates artifacts from that point forward.

**Context** (varies by target stage):
- **Specs stage**: config, constitution, `proposal.md`, target centralized doc(s) from `fab/docs/`
- **Plan stage**: above + completed `spec.md`
- **Tasks stage**: above + `plan.md` (if not skipped)

**Examples**:
```
/fab:continue
→ "Stage: proposal (done). Next: Create spec.md."

/fab:continue plan
→ "Resetting to plan stage. Regenerating plan.md..."
→ "Plan updated. Run /fab:continue to generate tasks, or /fab:ff to fast-forward."
```

**Behavior** (no argument — normal forward flow):
1. Read `.status.yaml` to determine current stage
2. Identify next artifact to create
3. Load relevant template + context (including `fab/constitution.md` for project principles)
4. Generate artifact (with clarification/research as needed)
5. **Plan decision** (when transitioning from specs to plan): evaluate whether a plan is warranted. If the change is small and the approach is obvious, propose skipping to the user: *"This change is straightforward — skip plan and go directly to tasks?"* If the user agrees, record `plan: skipped` in `.status.yaml` and proceed to tasks. If the user wants a plan, generate it normally.
6. Auto-generate checklist when creating tasks
7. Update `.status.yaml`

**Behavior** (with stage argument — reset and regenerate):
1. **Guard**: target stage must be `specs`, `plan`, or `tasks`. Cannot reset to `proposal` (use `/fab:new`) or `apply`/`review`/`archive`.
2. Reset `.status.yaml` stage to the target. Mark all stages from target onward as `pending`.
3. Regenerate the target stage's artifact in place (update, not recreate from scratch — preserve what's still valid).
4. Downstream artifacts are invalidated: tasks are reset to `- [ ]`, checklist is regenerated. Plan is regenerated if the target is `specs`.
5. Update `.status.yaml` and report what was reset.

---

## `/fab:ff` (Fast Forward)

**Purpose**: Fast-forward through remaining planning stages in one pass. Requires an active change with a completed proposal (run `/fab:new` first).

**Context**: config, constitution, `proposal.md`, target centralized doc(s) from `fab/docs/` (all loaded upfront since ff traverses all planning stages)

**Flow**: specs → plan (if warranted) → tasks (+ checklist)

**When to use**:
- Small, well-understood changes
- Clear requirements upfront
- Want to reach implementation quickly

**Example**:
```
/fab:new Add a logout button to the navbar that clears session
/fab:ff
```

**Behavior**:
1. Read `fab/current` to resolve the active change; verify proposal is complete
2. **Frontload questions** — scan the proposal for ambiguities across *all* planning stages (specs, plan, tasks). Collect everything that needs user input into a single batch of questions. Ask once, then proceed without further interruption. The goal: one Q&A round, then heads-down generation.
3. Generate `spec.md` (incorporating answers from step 2)
4. Evaluate whether a plan is warranted (see "Plan decision" below). If yes, draft plan with inline research. If no, skip directly to tasks
5. Produce task breakdown (referencing plan if it exists, otherwise referencing spec and proposal directly)
6. Auto-generate quality checklist
7. Update status to `tasks: done`

**Plan decision**: The agent skips `plan.md` when the change is small and the implementation approach is obvious — e.g., single-file changes, straightforward CRUD, or well-known patterns. When skipped, `.status.yaml` records `plan: skipped`. Unlike `/fab:continue`, `/fab:ff` does **not** confirm with the user before skipping — it decides autonomously to maintain the fast-forward flow.

---

## `/fab:clarify`

**Purpose**: Deepen and refine the current stage artifact without advancing to the next stage.

**Context** (varies by current stage):
- **Proposal**: config, constitution, `proposal.md`
- **Specs**: above + target centralized doc(s) from `fab/docs/`
- **Plan**: above + `spec.md`, `plan.md`
- **Tasks**: above + `plan.md` (if not skipped), `tasks.md`

**Example**:
```
/fab:clarify
→ "Stage: specs (active). Reviewing spec.md for gaps..."
→ "Found 2 [NEEDS CLARIFICATION] markers. Resolving..."
→ "Added 3 missing scenarios to spec.md"
```

**When to use**:
- Current artifact has unresolved ambiguities or [NEEDS CLARIFICATION] markers
- You want deeper technical research before committing to a plan
- Task breakdown feels incomplete or wrong-grained
- Proposal scope needs sharpening before moving to specs

**Behavior**:
1. Read `.status.yaml` to determine current stage
2. **Guard**: stage must be `proposal`, `specs`, `plan`, or `tasks`. If stage is `apply` or later, suggest `/fab:review` instead
3. Load the current stage's artifact + relevant context
4. Analyze the artifact for gaps, ambiguities, and opportunities to deepen:
   - **Proposal**: Unresolved [BLOCKING] questions, vague scope, missing impact analysis
   - **Specs**: [NEEDS CLARIFICATION] markers, missing scenarios, underspecified requirements
   - **Plan**: Untested assumptions, missing research, weak decision rationale
   - **Tasks**: Missing tasks, wrong granularity, unclear dependencies, missing file paths
5. Refine the artifact **in place** — edit the existing file, don't regenerate from scratch
6. Report what was clarified/refined
7. Do **not** advance the stage or update `.status.yaml` stage field

**Key property**: Idempotent and non-advancing. Calling `/fab:clarify` multiple times is safe — it refines further each time. It never transitions to the next stage. Use `/fab:continue` when satisfied.

---

## `/fab:apply`

**Purpose**: Execute tasks from `tasks.md`.

**Context**: config, constitution, `tasks.md`, `spec.md`, `plan.md` (if exists), relevant source code (files referenced in tasks)

**Example**:
```
/fab:apply
→ "Starting implementation. 12 tasks remaining."
```

**Behavior**:
1. Parse `tasks.md` for unchecked items `- [ ]`
2. Execute tasks in dependency order
3. Respect parallel markers `[P]`
4. After completing each task, run relevant tests (e.g., the test file for the module just modified). Fix failures before moving on.
5. Mark each task `[x]` immediately upon completion (not batched at the end)
6. Update `.status.yaml` progress after each task

**Resumability**: `/fab:apply` is inherently resumable. If the agent is interrupted mid-run, re-invoking `/fab:apply` picks up from the first unchecked item. The markdown checklist *is* the progress state — no separate tracking needed.

---

## `/fab:review`

**Purpose**: Validate implementation against specs and checklists.

**Context**: config, constitution, `tasks.md`, `checklists/quality.md`, `spec.md`, target centralized doc(s) from `fab/docs/`, relevant source code (files touched by the change)

**Example**:
```
/fab:review
→ "✓ 12/12 tasks complete"
→ "✓ 10/12 checklist items passed"
→ "✗ 2 items need attention: [CHK-007, CHK-011]"
```

**Checks** (the agent performs all of these):
1. All tasks in `tasks.md` marked `[x]`
2. All checklist items in `checklists/quality.md` verified and checked off — the agent re-reads each `CHK-*` item, inspects the relevant code/tests, and marks `[x]` or reports failure
3. Run tests affected by the change (scoped to modules touched, not the full suite)
4. Features match spec requirements (spot-check key scenarios from `spec.md`)
5. No doc drift detected (implementation doesn't contradict centralized docs)

**On failure**, the agent presents the options and the user chooses where to loop back:

- **Fix code** → `/fab:apply`
  Implementation bug. The agent identifies which tasks need rework, unchecks them in `tasks.md` (marks `- [ ]` again with a `<!-- rework: reason -->` comment), and re-runs `/fab:apply` which picks up the unchecked items.

- **Revise tasks** → edit `tasks.md`, then `/fab:apply`
  Missing or wrong tasks. The agent adds/modifies tasks in `tasks.md` (new tasks get the next sequential ID). Completed tasks that are unaffected stay `[x]`. Only new or revised tasks are executed.

- **Revise plan** → `/fab:continue plan`
  Architecture was wrong. Resets to plan stage, updates `plan.md` in place, and invalidates downstream artifacts — `tasks.md` is reset to `- [ ]` and the checklist is regenerated.

- **Revise specs** → `/fab:continue specs`
  Requirements were wrong or incomplete. Resets to specs stage, updates `spec.md` in place. Plan (if it exists) and tasks are subsequently regenerated. All downstream artifacts are reset.

The `.status.yaml` stage is reset to the chosen re-entry point. The general rule: **artifacts at and after the re-entry point are regenerated or updated; artifacts before it are preserved.**

---

## `/fab:archive`

**Purpose**: Complete the change and hydrate into centralized docs.

**Context**: `spec.md`, `plan.md` (if exists), target centralized doc(s) from `fab/docs/`, `fab/docs/index.md` and relevant domain indexes

**Example**:
```
/fab:archive
→ "Archived to fab/changes/archive/260115-a7k2-add-oauth/"
→ "Hydrated docs: fab/docs/auth/authentication.md"
```

**Behavior**:
1. **Final validation** — review must pass (all tasks `[x]`, all checklist items `[x]` including N/A items)
2. **Concurrent change check** — scan `fab/changes/` for other active changes whose specs reference the same centralized doc files. If found, warn the user: *"Change {name} also modifies {doc}. After this archive, that change's spec was written against a now-stale base. Re-review with `/fab:review` after switching to it."*
3. **Hydrate into `fab/docs/`**:
   The agent reads `spec.md`, `plan.md` (if it exists), and the current centralized doc, then rewrites the centralized doc to incorporate the changes:
   - **From spec.md** → integrate new/changed requirements and scenarios into the Requirements section. Remove requirements that the spec explicitly deprecates.
   - **From plan.md** → extract durable design decisions (from the Decisions section) into the Design Decisions section of the centralized doc. Skip tactical details (file paths, setup steps, library install commands).
   The agent compares against the existing doc to determine what's new vs changed vs removed — no explicit delta markers needed. Minimize edits to unchanged sections to prevent drift over successive archives.
4. **Update status** to `archive: done` in `.status.yaml`
5. **Move change folder** to `archive/` (no rename — date is already in the folder name)
6. **Clear pointer** — delete `fab/current` (no active change)

**Order of operations**: Steps 3–6 are ordered to fail safely. Status is updated *before* the folder move, so if the move is interrupted, the change is marked archived but still in `changes/` — the agent can detect and complete the move on next invocation. The pointer is cleared last so that mid-archive, `/fab:status` still reports the active change rather than "no active change" with a half-hydrated spec.

**Recovery**: Hydration modifies centralized docs in-place. If the merge goes wrong (garbled text, incorrect removals), the only recovery is `git checkout` on the affected doc files. Commit (or at least review the diff) before pushing after an archive.

---

## `/fab:switch <change-name>`

**Purpose**: Switch the active change when multiple changes exist.

**Example**:
```
/fab:switch fix-checkout
→ "fab/current now points to 260202-m3x1-fix-checkout-bug"
```

**Behavior**:
1. Match `change-name` against `fab/changes/` (supports partial/slug match)
2. **Ambiguous match** — if multiple changes match the input (e.g., `/fab:switch add` matches both `260115-a7k2-add-oauth` and `260202-m3x1-add-dark-mode`), list the matches and ask the user to pick one. Never guess.
3. **No match** — if nothing matches, list available changes and ask
4. Write the full change name to `fab/current`
5. Display the switched change's status summary

---

## `/fab:status`

**Purpose**: Show current change state at a glance.

**Example output**:
```
Change: 260115-a7k2-add-oauth
Branch: 260115-a7k2-add-oauth
Stage:  plan (3/7)

Progress:
  ✓ proposal    done
  ✓ specs       done
  ◉ plan        active
  ○ tasks       pending
  ○ apply       pending
  ○ review      pending
  ○ archive     pending

Checklist: not yet generated (created at tasks stage)

Next: Complete plan.md, then /fab:continue
```
