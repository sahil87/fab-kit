# Fab Skills Reference

> Detailed behavior for each `/fab:*` skill. For a quick overview, see the [Quick Reference](README.md#quick-reference).

---

## Terminology: "spec" vs "memory"

Fab uses two distinct terms to avoid confusion:

| Term | Location | Meaning |
|------|----------|---------|
| **Memory files** | `docs/memory/` | Source-of-truth documentation for the system. Contains both requirements (what) and durable design decisions (why). Updated by `/docs-hydrate-memory` (from external sources) and `/fab-continue` (hydrate) (from change artifacts). |
| **spec.md** | `fab/changes/{name}/spec.md` | Change-level specification. Describes the requirements relevant to this change. |

The stage named "spec" refers to the *activity* of writing the specification — its output is `spec.md` in the change folder.

---

## Context Loading Convention

Every skill that generates or validates artifacts MUST load relevant context before proceeding. This ensures agents produce accurate, grounded output rather than hallucinating requirements or ignoring existing patterns.

**Always loaded** (by every skill except `/fab-init`, `/fab-switch`, `/fab-status`, `/docs-hydrate-memory`):
- `fab/config.yaml` — project configuration, tech stack, conventions
- `fab/constitution.md` — project principles and constraints
- `docs/memory/index.md` — memory landscape (which domains and memory files exist)

**Change context** (loaded by skills operating on an active change):
- `.status.yaml` — current stage, progress
- All completed artifacts in the active change folder (e.g., `intake.md`, `spec.md`)

**Memory file lookup** (loaded by skills operating on an active change):
- Read the intake's "Affected Memory" section to identify relevant domains
- Read domain indexes (`docs/memory/{domain}/index.md`) for each relevant domain
- Read the specific memory file(s) referenced by the Affected Memory entries
- If a referenced file doesn't exist yet (listed under New Files), note this and proceed — it will be created by `/fab-continue` (hydrate)
- This grounds all artifact generation (spec, tasks, reviews) in the real current state, not assumptions

**Source code** (loaded during implementation and review):
- Read relevant source files referenced in the task descriptions
- Scope to files actually touched by the change — don't load the entire codebase

Each skill section below lists its specific context requirements under a **Context** field.

---

## Next Steps Convention

Every skill MUST end its output with a `Next:` line suggesting the available follow-up commands. This keeps the user oriented in the workflow without needing to memorize the stage graph.

**Format**: `Next: /fab-command` or `Next: /fab-commandA or /fab-commandB (description)`

**Lookup table**:

| After | Stage reached | Next line |
|-------|---------------|-----------|
| `/fab-init` | initialized | `Next: /fab-new <description> or /docs-hydrate-memory <sources>` |
| `/docs-hydrate-memory` | memory hydrated | `Next: /fab-new <description> or /docs-hydrate-memory <more-sources>` |
| `/fab-new` | intake done | `Next: /fab-continue or /fab-ff (fast-forward all planning)` |
| `/fab-continue` → spec | spec done | `Next: /fab-continue (tasks) or /fab-ff (fast-forward) or /fab-clarify (refine spec)` |
| `/fab-continue` → tasks | tasks done | `Next: /fab-continue (apply)` |
| `/fab-ff` | tasks done | `Next: /fab-continue (apply)` |
| `/fab-clarify` | same stage | `Next: /fab-clarify (refine further) or /fab-continue or /fab-ff` |
| `/fab-continue` → apply | apply done | `Next: /fab-continue (review)` |
| `/fab-continue` → review (pass) | review done | `Next: /fab-continue (hydrate)` |
| `/fab-continue` → review (fail) | review failed | *(contextual — see [Review Behavior](#review-behavior-via-fab-continue) for fix options)* |
| `/fab-continue` → hydrate | hydrated | `Next: /fab-archive` |

---

## `/fab-init`

**Purpose**: Bootstrap `fab/` in an existing project. Safe to run repeatedly — structural artifacts are created once and symlinks are repaired if broken.

**Prerequisite**: `fab/.kit/` must exist. If missing, abort with: *"fab/.kit/ not found. Copy the kit directory into fab/.kit/ first — see the Getting Started guide."*

**Arguments**: None. If arguments are provided, abort with: *"Did you mean /docs-hydrate-memory? /fab-init no longer accepts source arguments."*

**Creates** (first run only — skipped if already present):
- `fab/config.yaml` — project configuration (prompts for name, tech stack, conventions)
- `fab/constitution.md` — project principles and constraints (generated from conversation or existing documentation)
- `docs/memory/index.md` — initial memory index
- `fab/changes/` — empty, ready for change folders
- `.claude/skills/` — symlinks pointing into `fab/.kit/skills/`

**Examples**:
```
# First run — full bootstrap
/fab-init
→ "Found fab/.kit/ (v0.1.0). Initializing project..."
→ "What's the project name?"
→ "Describe the tech stack and conventions..."
→ "fab/ initialized with config, constitution, and empty memory index."
→ "Next: /fab-new <description> or /docs-hydrate-memory <sources>"

# Re-run — structural health check
/fab-init
→ "fab/ already initialized. Verified structure, repaired 1 missing symlink."

# Arguments are redirected
/fab-init https://notion.so/myteam/API-Spec-abc123
→ "Did you mean /docs-hydrate-memory? /fab-init no longer accepts source arguments."
```

**Behavior**:

1. **Pre-flight check**: Verify `fab/.kit/` exists (abort with guidance if not). If arguments are provided, abort with redirect to `/docs-hydrate-memory`.
2. **Structural bootstrap** (idempotent — each step skips if artifact already exists):
   a. `fab/config.yaml` — if missing, prompt for project name, description, tech stack and generate
   b. `fab/constitution.md` — if missing, generate from project context (README, existing documentation, conversation)
   c. `docs/memory/index.md` — if missing, create empty index
   d. `fab/changes/` — if missing, create empty directory
   e. `.claude/skills/` symlinks — create missing ones, repair broken ones
   f. `.gitignore` — append `fab/current` if not already present

---

## `/docs-hydrate-memory [sources...]`

**Purpose**: Ingest external sources into `docs/memory/` with domain mapping and index maintenance. Safe to run repeatedly — content is merged into existing memory files without duplication.

**Prerequisite**: `docs/memory/` must exist (run `/fab-init` first). If missing, abort with: *"docs/memory/ not found. Run /fab-init first to create the memory directory."*

**Arguments**:
- `[sources...]` *(required)* — one or more URLs or local paths containing documentation to ingest. Supported source types:
  - **Notion URLs** — pages or databases (fetched via Notion MCP or API)
  - **Linear URLs** — issues or projects (fetched via Linear MCP or API)
  - **Local files/directories** — markdown, text, or directories of files (read from filesystem)

**Creates/Updates**:
- `docs/memory/{domain}/{topic}.md` — memory files (created or merged)
- `docs/memory/{domain}/index.md` — domain indexes (created or updated)
- `docs/memory/index.md` — top-level index (updated with new domains/files)

**Examples**:
```
# Hydrate memory from a Notion page
/docs-hydrate-memory https://notion.so/myteam/API-Spec-abc123
→ "Fetched: API Spec (Notion)"
→ "Created: docs/memory/api/endpoints.md, docs/memory/api/authentication.md"
→ "Updated: docs/memory/index.md"

# Ingest local legacy documentation
/docs-hydrate-memory ./legacy-docs/payments/
→ "Fetched: 3 files from ./legacy-docs/payments/"
→ "Created: docs/memory/payments/checkout.md, docs/memory/payments/refunds.md"

# Multiple sources at once
/docs-hydrate-memory https://notion.so/myteam/Auth-xyz ./legacy-docs/payments/
→ "Fetched: Auth Design (Notion), 3 files from ./legacy-docs/payments/"
→ "Created: docs/memory/auth/oauth.md, docs/memory/payments/checkout.md"
→ "Updated: docs/memory/index.md"
```

**Behavior**:

1. **Pre-flight check**: Verify `docs/memory/` and `docs/memory/index.md` exist (abort with guidance if not). If no sources are provided, abort with usage message.
2. **Fetch/read** each source:
   - Notion URLs → fetch page content via Notion MCP or API
   - Linear URLs → fetch issue/project content via Linear MCP or API
   - Local paths → read files; if directory, read all markdown files recursively
3. **Analyze** fetched content to identify domains and topics
4. **Create or merge** memory files — for each identified topic, either create a new file in `docs/memory/{domain}/` or merge into an existing file. Follow the [Memory File Format](templates.md#memory-file-format-fabmemory) and [Hydration Rules](templates.md#hydration-rules).
5. **Update domain indexes** — create or update `docs/memory/{domain}/index.md` for each affected domain
6. **Update top-level index** — update `docs/memory/index.md` with new domains and expanded file lists
7. **Report** what was created and updated

---

## `/fab-new <description> [--switch]`

**Purpose**: Start a new change from a natural language description.

**Context**: config, constitution, `docs/memory/index.md` (to understand existing memory landscape)

**Creates**:
- Change folder named `{YYMMDD}-{XXXX}-{slug}`
- `.status.yaml` manifest
- `intake.md` from template (with clarifying questions if ambiguous)

**Arguments**:
- `<description>` — natural language description of the change, Linear ticket ID (e.g., `DEV-988`), or backlog ID (e.g., `90g5`) (required)
- `--switch` — automatically switch to the new change after creation (calls `/fab-switch` internally to write `fab/current` and handle branch integration). Also detected from natural language (e.g., "and switch to it", "make it active").

**Examples**:
```
/fab-new Add OAuth2 support for Google and GitHub sign-in
→ Created fab/changes/260115-a7k2-add-oauth/

/fab-new --switch Add OAuth2 support
→ Created fab/changes/260115-a7k2-add-oauth/
→ Switched to 260115-a7k2-add-oauth (branch created)
```

**Behavior**:
1. Generate folder name: today's date (`YYMMDD`) + 4 random alphanumeric chars + 2-6 word slug from description
2. Create `fab/changes/{name}/`
3. Initialize `.status.yaml` with `progress.intake: active`
4. Generate `intake.md` using template (loading `fab/constitution.md` and `fab/config.yaml` as context)
5. Perform gap analysis — check whether the change is already covered by existing mechanisms
6. Use SRAD-driven adaptive questioning (no fixed cap) to resolve ambiguities conversationally
7. Leave intake as `active` — `/fab-continue` handles the intake → spec transition
8. **Switch** (if `--switch` flag or switching intent detected): call `/fab-switch` to write `fab/current` and handle branch integration. Default: skip this step.

---

## `/fab-continue [<stage>]`

**Purpose**: Create the next artifact in sequence — or, when called with a stage argument, reset to that stage and regenerate from there.

**Arguments**:
- `<stage>` *(optional)* — target stage to reset to (`spec` or `tasks`). Used after `/fab-continue` (review) identifies issues upstream. When provided, resets `.status.yaml` to this stage and regenerates artifacts from that point forward.

**Context** (varies by target stage):
- **Spec stage**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`
- **Tasks stage**: above + completed `spec.md`

**Examples**:
```
/fab-continue
→ "Stage: intake (done). Next: Create spec.md."

/fab-continue spec
→ "Resetting to spec stage. Regenerating spec.md..."
→ "Spec updated. Run /fab-continue to generate tasks, or /fab-ff to fast-forward."
```

**Behavior** (no argument — normal forward flow):
1. Read `.status.yaml` to determine current stage
2. Identify next artifact to create
3. Load relevant template + context (including `fab/constitution.md` for project principles)
4. Generate artifact (with clarification/research as needed)
5. Auto-generate checklist when creating tasks
7. Update `.status.yaml`

**Behavior** (with stage argument — reset and regenerate):
1. **Guard**: target stage must be `spec` or `tasks`. Cannot reset to `intake` (use `/fab-new`) or `apply`/`review`/`hydrate`.
2. Reset `.status.yaml` stage to the target. Mark all stages from target onward as `pending`.
3. Regenerate the target stage's artifact in place (update, not recreate from scratch — preserve what's still valid).
4. Downstream artifacts are invalidated: tasks are reset to `- [ ]`, checklist is regenerated.
5. Update `.status.yaml` and report what was reset.

---

## `/fab-ff` (Fast Forward)

**Purpose**: Fast-forward through remaining planning stages in one pass. Requires an active change with a completed intake (run `/fab-new` first).

**Context**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/` (all loaded upfront since ff traverses all planning stages)

**Flow**: spec → tasks (+ checklist)

**When to use**:
- Small, well-understood changes
- Clear requirements upfront
- Want to reach implementation quickly

**Example**:
```
/fab-new Add a logout button to the navbar that clears session
/fab-ff
```

**Behavior**:
1. Read `fab/current` to resolve the active change; verify intake is complete
2. **Frontload questions** — scan the intake for ambiguities across *all* planning stages (spec, tasks). Collect everything that needs user input into a single batch of questions. Ask once, then proceed without further interruption. The goal: one Q&A round, then heads-down generation.
3. Generate `spec.md` (incorporating answers from step 2)
4. Produce task breakdown (referencing spec and intake)
5. Auto-generate quality checklist
6. Update status to `tasks: done`

---

## `/fab-fff` (Full Autonomous Pipeline)

**Purpose**: Run the entire Fab pipeline from planning through hydrate in a single invocation. No confidence gate. Frontloads questions, interleaves auto-clarify between planning stages, and autonomously reworks on review failure (3-cycle retry cap, escalation after 2 consecutive fix-code failures).

**Prerequisite**: Active change with completed `intake.md`.

**Context**: Same as `/fab-ff` — all context loaded upfront (config, constitution, intake, memory index, affected memory files).

**Example**:
```
/fab-fff
→ --- Planning ---
→ ... (spec + tasks generated, with auto-clarify)
→ --- Implementation ---
→ ... (tasks executed)
→ --- Review ---
→ ... (validation passed)
→ --- Hydrate ---
→ ... (memory hydrated)
→ "Pipeline complete. Change hydrated."
```

**Behavior**:
1. **Frontload questions**: Scan intake for ambiguities across all planning stages. Collect Unresolved decisions into a single batch. Ask once, then proceed.
2. **Resumability**: Check `progress` map — skip any stage already marked `done` or `skipped`. Re-invoking after interruption picks up from the first incomplete stage.
3. **Step 1 — Planning**: Generate spec + tasks with checklist. Interleave auto-clarify between stages. Bails on blocking issues.
4. **Step 2 — Implementation**: Execute tasks in dependency order, run tests after each.
5. **Step 3 — Review**: Validate implementation. On failure, autonomously select rework path (fix code, revise tasks, revise spec) and retry (max 3 cycles, escalation after 2 consecutive fix-code). Bail with summary after 3 failed cycles.
6. **Step 4 — Hydrate**: Hydrate into memory.

**Key difference from `/fab-ff`**: `/fab-fff` is the full pipeline (intake → hydrate) with autonomous rework and no confidence gate. `/fab-ff` is fast-forward from spec (spec → hydrate) with confidence gate and interactive rework on review failure.

---

## `/fab-clarify`

**Purpose**: Deepen and refine the current stage artifact without advancing to the next stage.

**Context** (varies by current stage):
- **Spec**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`
- **Tasks**: above + `spec.md`, `tasks.md`

**Example**:
```
/fab-clarify
→ "Stage: spec (active). Reviewing spec.md for gaps..."
→ "Found 2 [NEEDS CLARIFICATION] markers. Resolving..."
→ "Added 3 missing scenarios to spec.md"
```

**When to use**:
- Current artifact has unresolved ambiguities or [NEEDS CLARIFICATION] markers
- You want deeper technical research before moving to tasks
- Task breakdown feels incomplete or wrong-grained
- Intake scope needs sharpening before moving to spec

**Behavior**:
1. Read `.status.yaml` to determine current stage
2. **Guard**: stage must be `spec` or `tasks`. If stage is `apply` or later, suggest `/fab-continue` (review) instead
3. Load the current stage's artifact + relevant context
4. Analyze the artifact for gaps, ambiguities, and opportunities to deepen:
   - **Spec**: [NEEDS CLARIFICATION] markers, missing scenarios, underspecified requirements
   - **Tasks**: Missing tasks, wrong granularity, unclear dependencies, missing file paths
5. Refine the artifact **in place** — edit the existing file, don't regenerate from scratch
6. Report what was clarified/refined
7. Do **not** advance the stage or update `.status.yaml` stage field

**Key property**: Idempotent and non-advancing. Calling `/fab-clarify` multiple times is safe — it refines further each time. It never transitions to the next stage. Use `/fab-continue` when satisfied.

---

## Apply Behavior (via `/fab-continue`)

**Purpose**: Execute tasks from `tasks.md`.

**Context**: config, constitution, `tasks.md`, `spec.md`, relevant source code (files referenced in tasks)

**Example**:
```
/fab-continue
→ "Starting implementation. 12 tasks remaining."
```

**Behavior**:
1. Parse `tasks.md` for unchecked items `- [ ]`
2. Execute tasks in dependency order
3. Respect parallel markers `[P]`
4. After completing each task, run relevant tests (e.g., the test file for the module just modified). Fix failures before moving on.
5. Mark each task `[x]` immediately upon completion (not batched at the end)
6. Update `.status.yaml` progress after each task

**Resumability**: `/fab-continue` (apply) is inherently resumable. If the agent is interrupted mid-run, re-invoking `/fab-continue` picks up from the first unchecked item. The markdown checklist *is* the progress state — no separate tracking needed.

---

## Review Behavior (via `/fab-continue`)

**Purpose**: Validate implementation against spec and checklists.

**Context**: config, constitution, `tasks.md`, `checklist.md`, `spec.md`, target memory file(s) from `docs/memory/`, relevant source code (files touched by the change)

**Example**:
```
/fab-continue
→ "✓ 12/12 tasks complete"
→ "✓ 10/12 checklist items passed"
→ "✗ 2 items need attention: [CHK-007, CHK-011]"
```

**Checks** (the agent performs all of these):
1. All tasks in `tasks.md` marked `[x]`

2. All checklist items in `checklist.md` verified and checked off — the agent re-reads each `CHK-*` item, inspects the relevant code/tests, and marks `[x]` or reports failure
3. Run tests affected by the change (scoped to modules touched, not the full suite)
4. Features match spec requirements (spot-check key scenarios from `spec.md`)
5. No memory drift detected (implementation doesn't contradict memory files)

**On failure**, the agent presents the options and the user chooses where to loop back:

- **Fix code** → `/fab-continue` (apply)
  Implementation bug. The agent identifies which tasks need rework, unchecks them in `tasks.md` (marks `- [ ]` again with a `<!-- rework: reason -->` comment), and re-runs `/fab-continue` which picks up the unchecked items.

- **Revise tasks** → edit `tasks.md`, then `/fab-continue` (apply)
  Missing or wrong tasks. The agent adds/modifies tasks in `tasks.md` (new tasks get the next sequential ID). Completed tasks that are unaffected stay `[x]`. Only new or revised tasks are executed.

- **Revise spec** → `/fab-continue spec`
  Requirements were wrong or incomplete. Resets to spec stage, updates `spec.md` in place. Tasks are subsequently regenerated. All downstream artifacts are reset.

The `.status.yaml` stage is reset to the chosen re-entry point. The general rule: **artifacts at and after the re-entry point are regenerated or updated; artifacts before it are preserved.**

---

## Hydrate Behavior (via `/fab-continue`)

**Purpose**: Validate review passed and hydrate change artifacts into memory files. The change folder remains in `fab/changes/` after hydrate — archiving is a separate step via `/fab-archive`.

**Context**: `spec.md`, `intake.md`, target memory file(s) from `docs/memory/`, `docs/memory/index.md` and relevant domain indexes

**Example**:
```
/fab-continue
→ "Hydrated memory: docs/memory/auth/authentication.md"
→ "Next: /fab-archive"
```

**Behavior**:
1. **Final validation** — review must pass (all tasks `[x]`, all checklist items `[x]` including N/A items)
2. **Concurrent change check** — scan `fab/changes/` for other active changes whose specs reference the same memory files. If found, warn the user: *"Change {name} also modifies {file}. After this hydrate, that change's spec was written against a now-stale base. Re-review with `/fab-continue` after switching to it."*
3. **Hydrate into `docs/memory/`**:
   The agent reads `spec.md` and the current memory file, then rewrites the memory file to incorporate the changes:
   - **From spec.md** → integrate new/changed requirements and scenarios into the Requirements section. Remove requirements that the spec explicitly deprecates. Extract durable design decisions into Design Decisions section.
   The agent compares against the existing memory file to determine what's new vs changed vs removed — no explicit delta markers needed. Minimize edits to unchanged sections to prevent drift.
4. **Update status** to `hydrate: done` in `.status.yaml`

**Recovery**: Hydration modifies memory files in-place. If the merge goes wrong (garbled text, incorrect removals), the only recovery is `git checkout` on the affected memory files. Commit (or at least review the diff) before pushing after hydrate.

---

## `/fab-archive [<change-name>]`

**Purpose**: Standalone housekeeping command — not a pipeline stage. Moves completed changes to the archive directory, updates the archive index, marks backlog items done, and clears the pointer.

**Prerequisite**: `hydrate: done` in `.status.yaml`. If hydrate is not done, stop with: *"Hydrate has not completed. Run /fab-continue to hydrate memory first."*

**Arguments**:
- `<change-name>` *(optional)* — target a specific change instead of `fab/current`

**Example**:
```
/fab-archive
→ "Archived to fab/changes/archive/260115-a7k2-add-oauth/"
→ "Next: /fab-new <description>"
```

**Behavior**:
1. **Move change folder** — `fab/changes/{name}/` → `fab/changes/archive/{name}/`. Create `archive/` if needed. No rename.
2. **Update archive index** — prepend entry to `fab/changes/archive/index.md` (create with backfill if missing). Format: `- **{folder-name}** — {1-2 sentence description}`. Most-recent-first.
3. **Mark backlog items done** — exact-ID check (always), then keyword scan with interactive confirmation.
4. **Clear pointer** — delete `fab/current` only if the archived change is the active one.

**Order of operations**: Steps 1–4 execute in this order for safety. Folder move first (recoverable if interrupted — re-run detects folder already in archive and completes remaining steps). Index after folder is in place. Backlog marking after index. Pointer last.

**Restore mode** (`/fab-archive restore <change-name> [--switch]`): Moves an archived change back to `fab/changes/`. Preserves all artifacts and `.status.yaml` without modification. Optionally activates via `--switch` flag.

---

## `/fab-switch <change-name>`

**Purpose**: Switch the active change when multiple changes exist.

**Example**:
```
/fab-switch fix-checkout
→ "fab/current now points to 260202-m3x1-fix-checkout-bug"
```

**Behavior**:
1. Match `change-name` against `fab/changes/` (supports partial/slug match)
2. **Ambiguous match** — if multiple changes match the input (e.g., `/fab-switch add` matches both `260115-a7k2-add-oauth` and `260202-m3x1-add-dark-mode`), list the matches and ask the user to pick one. Never guess.
3. **No match** — if nothing matches, list available changes and ask
4. Write the full change name to `fab/current`
5. Display the switched change's status summary

---

## `/fab-status`

**Purpose**: Show current change state at a glance.

**Example output**:
```
Change: 260115-a7k2-add-oauth
Branch: 260115-a7k2-add-oauth
Stage:  intake (1/6)

Progress:
  ◉ intake      active
  ○ spec        pending
  ○ tasks       pending
  ○ apply       pending
  ○ review      pending
  ○ hydrate     pending

Checklist: not yet generated (created at tasks stage)

Next: Complete intake.md, then /fab-continue
```

---

## `/docs-hydrate-specs [domain]`

**Purpose**: Identify structural gaps between `docs/memory/` and `docs/specs/` and propose concise additions back to specs with interactive confirmation.

**Context**: `docs/memory/index.md`, `docs/specs/index.md`, all memory files, all spec files

**Arguments**:
- `[domain]` *(optional)* — scope to a single memory domain. Scans all domains if omitted.

**Example**:
```
/docs-hydrate-specs
→ "Found 5 structural gaps (showing top 3):"
→ Gap 1: Preflight Script — Source: preflight.md, Target: architecture.md
→ Shows exact markdown preview, asks: "Add this? (yes / no / done)"
```

**Behavior**:
1. Read all memory files to build a topic inventory (headings + summaries)
2. Read all spec files to build a coverage inventory (headings + inline mentions)
3. Cross-reference at section level — a gap is a memory topic with no spec coverage at all
4. Rank by impact (core behaviors > supporting concepts > implementation detail)
5. Present top 3 with exact markdown previews
6. Per-gap interactive confirm: yes (write), no (skip), done (stop)
7. Only confirmed additions are written to spec files

**Key properties**: No active change required. No git operations. Idempotent. Specs modified only with user confirmation.

---

## `/docs-reorg-memory`

**Purpose**: Analyze memory files across all domains for themes and propose a reorganization plan. Read-only by default — files only moved/rewritten with explicit user approval.

**Context**: `docs/memory/index.md`, all domain indexes and memory files. Does NOT require `fab/current`, config, or constitution.

**Prerequisite**: `docs/memory/index.md` must exist and `docs/memory/` must contain at least one domain with `.md` files besides `index.md`.

**Behavior**:
1. Read all memory files — extract headings, section summaries, approximate line counts
2. Identify themes (up to 10) with cohesion assessment (concentrated / scattered)
3. Diagnose current structure — what works, pain points, missing connections
4. Propose reorganization with migration map and updated index previews
5. User confirmation — apply all, cherry-pick specific migrations, or skip

**Key properties**: No active change required. No git operations. Idempotent. Memory files modified only with explicit confirmation.

---

## `/docs-reorg-specs`

**Purpose**: Analyze spec files for themes and propose a reorganization plan. Read-only by default — files only moved/rewritten with explicit user approval.

**Context**: `docs/specs/index.md` and all spec files. Does NOT require `fab/current`, config, or constitution.

**Prerequisite**: `docs/specs/index.md` must exist and `docs/specs/` must contain at least one `.md` file besides `index.md`.

**Behavior**:
1. Read all spec files — extract headings, section summaries, approximate line counts
2. Identify themes (up to 10) with cohesion assessment (concentrated / scattered)
3. Diagnose current structure — what works, pain points, missing connections
4. Propose reorganization with migration map and updated index preview
5. User confirmation — apply all, cherry-pick specific migrations, or skip

**Key properties**: No active change required. No git operations. Idempotent. Spec files modified only with explicit confirmation.
