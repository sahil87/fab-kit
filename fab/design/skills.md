# Fab Skills Reference

> Detailed behavior for each `/fab:*` skill. For a quick overview, see the [Quick Reference](README.md#quick-reference).

---

## Terminology: "spec" vs "docs"

Fab uses two distinct terms to avoid confusion:

| Term | Location | Meaning |
|------|----------|---------|
| **Centralized docs** | `fab/docs/` | Source-of-truth documentation for the system. Contains both requirements (what) and durable design decisions (why). Updated by `/fab-hydrate` (from external sources) and `/fab-continue` (archive) (from change artifacts). |
| **spec.md** | `fab/changes/{name}/spec.md` | Change-level specification. Describes the requirements relevant to this change. |

The stage named "spec" refers to the *activity* of writing the specification — its output is `spec.md` in the change folder.

---

## Context Loading Convention

Every skill that generates or validates artifacts MUST load relevant context before proceeding. This ensures agents produce accurate, grounded output rather than hallucinating requirements or ignoring existing patterns.

**Always loaded** (by every skill except `/fab-init`, `/fab-switch`, `/fab-status`, `/fab-hydrate`):
- `fab/config.yaml` — project configuration, tech stack, conventions
- `fab/constitution.md` — project principles and constraints
- `fab/docs/index.md` — documentation landscape (which domains and docs exist)

**Change context** (loaded by skills operating on an active change):
- `.status.yaml` — current stage, progress
- All completed artifacts in the active change folder (e.g., `brief.md`, `spec.md`)

**Centralized doc lookup** (loaded by skills operating on an active change):
- Read the brief's "Affected Docs" section to identify relevant domains
- Read domain indexes (`fab/docs/{domain}/index.md`) for each relevant domain
- Read the specific centralized doc(s) referenced by the Affected Docs entries
- If a referenced doc doesn't exist yet (listed under New Docs), note this and proceed — it will be created by `/fab-continue` (archive)
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
| `/fab-init` | initialized | `Next: /fab-new <description> or /fab-hydrate <sources>` |
| `/fab-hydrate` | docs hydrated | `Next: /fab-new <description> or /fab-hydrate <more-sources>` |
| `/fab-new` | brief done | `Next: /fab-continue or /fab-ff (fast-forward all planning)` |
| `/fab-continue` → spec | spec done | `Next: /fab-continue (tasks) or /fab-ff (fast-forward) or /fab-clarify (refine spec)` |
| `/fab-continue` → tasks | tasks done | `Next: /fab-continue (apply)` |
| `/fab-ff` | tasks done | `Next: /fab-continue (apply)` |
| `/fab-clarify` | same stage | `Next: /fab-clarify (refine further) or /fab-continue or /fab-ff` |
| `/fab-continue` → apply | apply done | `Next: /fab-continue (review)` |
| `/fab-continue` → review (pass) | review done | `Next: /fab-continue (archive)` |
| `/fab-continue` → review (fail) | review failed | *(contextual — see [Review Behavior](#review-behavior-via-fab-continue) for fix options)* |
| `/fab-continue` → archive | archived | `Next: /fab-new <description> (start next change)` |

---

## `/fab-init`

**Purpose**: Bootstrap `fab/` in an existing project. Safe to run repeatedly — structural artifacts are created once and symlinks are repaired if broken.

**Prerequisite**: `fab/.kit/` must exist. If missing, abort with: *"fab/.kit/ not found. Copy the kit directory into fab/.kit/ first — see the Getting Started guide."*

**Arguments**: None. If arguments are provided, abort with: *"Did you mean /fab-hydrate? /fab-init no longer accepts source arguments."*

**Creates** (first run only — skipped if already present):
- `fab/config.yaml` — project configuration (prompts for name, tech stack, conventions)
- `fab/constitution.md` — project principles and constraints (generated from conversation or existing docs)
- `fab/docs/index.md` — initial docs index
- `fab/changes/` — empty, ready for change folders
- `.claude/skills/` — symlinks pointing into `fab/.kit/skills/`

**Examples**:
```
# First run — full bootstrap
/fab-init
→ "Found fab/.kit/ (v0.1.0). Initializing project..."
→ "What's the project name?"
→ "Describe the tech stack and conventions..."
→ "fab/ initialized with config, constitution, and empty docs."
→ "Next: /fab-new <description> or /fab-hydrate <sources>"

# Re-run — structural health check
/fab-init
→ "fab/ already initialized. Verified structure, repaired 1 missing symlink."

# Arguments are redirected
/fab-init https://notion.so/myteam/API-Spec-abc123
→ "Did you mean /fab-hydrate? /fab-init no longer accepts source arguments."
```

**Behavior**:

1. **Pre-flight check**: Verify `fab/.kit/` exists (abort with guidance if not). If arguments are provided, abort with redirect to `/fab-hydrate`.
2. **Structural bootstrap** (idempotent — each step skips if artifact already exists):
   a. `fab/config.yaml` — if missing, prompt for project name, description, tech stack and generate
   b. `fab/constitution.md` — if missing, generate from project context (README, existing docs, conversation)
   c. `fab/docs/index.md` — if missing, create empty index
   d. `fab/changes/` — if missing, create empty directory
   e. `.claude/skills/` symlinks — create missing ones, repair broken ones
   f. `.gitignore` — append `fab/current` if not already present

---

## `/fab-hydrate [sources...]`

**Purpose**: Ingest external documentation into `fab/docs/` with domain mapping and index maintenance. Safe to run repeatedly — content is merged into existing docs without duplication.

**Prerequisite**: `fab/docs/` must exist (run `/fab-init` first). If missing, abort with: *"fab/docs/ not found. Run /fab-init first to create the docs directory."*

**Arguments**:
- `[sources...]` *(required)* — one or more URLs or local paths containing documentation to ingest. Supported source types:
  - **Notion URLs** — pages or databases (fetched via Notion MCP or API)
  - **Linear URLs** — issues or projects (fetched via Linear MCP or API)
  - **Local files/directories** — markdown, text, or directories of docs (read from filesystem)

**Creates/Updates**:
- `fab/docs/{domain}/{topic}.md` — centralized doc files (created or merged)
- `fab/docs/{domain}/index.md` — domain indexes (created or updated)
- `fab/docs/index.md` — top-level index (updated with new domains/docs)

**Examples**:
```
# Hydrate docs from a Notion page
/fab-hydrate https://notion.so/myteam/API-Spec-abc123
→ "Fetched: API Spec (Notion)"
→ "Created: fab/docs/api/endpoints.md, fab/docs/api/authentication.md"
→ "Updated: fab/docs/index.md"

# Ingest local legacy docs
/fab-hydrate ./legacy-docs/payments/
→ "Fetched: 3 files from ./legacy-docs/payments/"
→ "Created: fab/docs/payments/checkout.md, fab/docs/payments/refunds.md"

# Multiple sources at once
/fab-hydrate https://notion.so/myteam/Auth-xyz ./legacy-docs/payments/
→ "Fetched: Auth Design (Notion), 3 files from ./legacy-docs/payments/"
→ "Created: fab/docs/auth/oauth.md, fab/docs/payments/checkout.md"
→ "Updated: fab/docs/index.md"
```

**Behavior**:

1. **Pre-flight check**: Verify `fab/docs/` and `fab/docs/index.md` exist (abort with guidance if not). If no sources are provided, abort with usage message.
2. **Fetch/read** each source:
   - Notion URLs → fetch page content via Notion MCP or API
   - Linear URLs → fetch issue/project content via Linear MCP or API
   - Local paths → read files; if directory, read all markdown files recursively
3. **Analyze** fetched content to identify domains and topics
4. **Create or merge** docs — for each identified topic, either create a new doc in `fab/docs/{domain}/` or merge into an existing doc. Follow the [Centralized Doc Format](TEMPLATES.md#centralized-doc-format-fabdocs) and [Hydration Rules](TEMPLATES.md#hydration-rules).
5. **Update domain indexes** — create or update `fab/docs/{domain}/index.md` for each affected domain
6. **Update top-level index** — update `fab/docs/index.md` with new domains and expanded doc lists
7. **Report** what was created and updated

---

## `/fab-new <description> [--branch <name>]`

**Purpose**: Start a new change from a natural language description.

**Context**: config, constitution, `fab/docs/index.md` (to understand existing doc landscape)

**Creates**:
- Change folder named `{YYMMDD}-{XXXX}-{slug}`
- `.status.yaml` manifest
- `brief.md` from template (with clarifying questions if ambiguous)

**Arguments**:
- `<description>` — natural language description of the change (required)
- `--branch <name>` — explicit branch name to use (optional). Skips the branch prompt and uses this name directly. Useful for Linear-linked branches, team conventions, or pre-existing branches.

**Examples**:
```
/fab-new Add OAuth2 support for Google and GitHub sign-in
→ Created fab/changes/260115-a7k2-add-oauth/
→ Branch: 260115-a7k2-add-oauth (created)

/fab-new --branch feature/dev-907-oauth Add OAuth2 support
→ Created fab/changes/260115-a7k2-add-oauth/
→ Branch: feature/dev-907-oauth (adopted)
```

**Behavior**:
1. Generate folder name: today's date (`YYMMDD`) + 4 random alphanumeric chars + 2-6 word slug from description
2. Create `fab/changes/{name}/`
3. Write change name to `fab/current` (sets this as the active change)
4. **Branch integration** (if `git.enabled` in config and inside a git repo):
   - If `--branch <name>` was provided → use that name directly (create if it doesn't exist, adopt if it does)
   - Else if on `main`/`master` → offer to create a new branch named `{prefix}{change-name}`
   - Else if on a feature branch → offer to adopt it (record current branch name as-is)
   - If user declines → skip, no `branch:` field in `.status.yaml`
   - Record chosen branch name in `.status.yaml` as `branch:`
5. Initialize `.status.yaml` with `progress.spec: active` (and `branch:` if set)
6. Generate `brief.md` using template (loading `fab/constitution.md` and `fab/config.yaml` as context)
7. Perform gap analysis — check whether the change is already covered by existing mechanisms
8. Use SRAD-driven adaptive questioning (no fixed cap) to resolve ambiguities conversationally
9. Mark brief complete once the user is satisfied

---

## `/fab-continue [<stage>]`

**Purpose**: Create the next artifact in sequence — or, when called with a stage argument, reset to that stage and regenerate from there.

**Arguments**:
- `<stage>` *(optional)* — target stage to reset to (`spec` or `tasks`). Used after `/fab-continue` (review) identifies issues upstream. When provided, resets `.status.yaml` to this stage and regenerates artifacts from that point forward.

**Context** (varies by target stage):
- **Spec stage**: config, constitution, `brief.md`, target centralized doc(s) from `fab/docs/`
- **Tasks stage**: above + completed `spec.md`

**Examples**:
```
/fab-continue
→ "Stage: brief (done). Next: Create spec.md."

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
1. **Guard**: target stage must be `spec` or `tasks`. Cannot reset to `brief` (use `/fab-new`) or `apply`/`review`/`archive`.
2. Reset `.status.yaml` stage to the target. Mark all stages from target onward as `pending`.
3. Regenerate the target stage's artifact in place (update, not recreate from scratch — preserve what's still valid).
4. Downstream artifacts are invalidated: tasks are reset to `- [ ]`, checklist is regenerated.
5. Update `.status.yaml` and report what was reset.

---

## `/fab-ff` (Fast Forward)

**Purpose**: Fast-forward through remaining planning stages in one pass. Requires an active change with a completed brief (run `/fab-new` first).

**Context**: config, constitution, `brief.md`, target centralized doc(s) from `fab/docs/` (all loaded upfront since ff traverses all planning stages)

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
1. Read `fab/current` to resolve the active change; verify brief is complete
2. **Frontload questions** — scan the brief for ambiguities across *all* planning stages (spec, tasks). Collect everything that needs user input into a single batch of questions. Ask once, then proceed without further interruption. The goal: one Q&A round, then heads-down generation.
3. Generate `spec.md` (incorporating answers from step 2)
4. Produce task breakdown (referencing spec and brief)
5. Auto-generate quality checklist
6. Update status to `tasks: done`

---

## `/fab-clarify`

**Purpose**: Deepen and refine the current stage artifact without advancing to the next stage.

**Context** (varies by current stage):
- **Spec**: config, constitution, `brief.md`, target centralized doc(s) from `fab/docs/`
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
- Brief scope needs sharpening before moving to spec

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

**Context**: config, constitution, `tasks.md`, `checklists/quality.md`, `spec.md`, target centralized doc(s) from `fab/docs/`, relevant source code (files touched by the change)

**Example**:
```
/fab-continue
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

- **Fix code** → `/fab-continue` (apply)
  Implementation bug. The agent identifies which tasks need rework, unchecks them in `tasks.md` (marks `- [ ]` again with a `<!-- rework: reason -->` comment), and re-runs `/fab-continue` which picks up the unchecked items.

- **Revise tasks** → edit `tasks.md`, then `/fab-continue` (apply)
  Missing or wrong tasks. The agent adds/modifies tasks in `tasks.md` (new tasks get the next sequential ID). Completed tasks that are unaffected stay `[x]`. Only new or revised tasks are executed.

- **Revise spec** → `/fab-continue spec`
  Requirements were wrong or incomplete. Resets to spec stage, updates `spec.md` in place. Tasks are subsequently regenerated. All downstream artifacts are reset.

The `.status.yaml` stage is reset to the chosen re-entry point. The general rule: **artifacts at and after the re-entry point are regenerated or updated; artifacts before it are preserved.**

---

## Archive Behavior (via `/fab-continue`)

**Purpose**: Complete the change and hydrate into centralized docs.

**Context**: `spec.md`, target centralized doc(s) from `fab/docs/`, `fab/docs/index.md` and relevant domain indexes

**Example**:
```
/fab-continue
→ "Archived to fab/changes/archive/260115-a7k2-add-oauth/"
→ "Hydrated docs: fab/docs/auth/authentication.md"
```

**Behavior**:
1. **Final validation** — review must pass (all tasks `[x]`, all checklist items `[x]` including N/A items)
2. **Concurrent change check** — scan `fab/changes/` for other active changes whose specs reference the same centralized doc files. If found, warn the user: *"Change {name} also modifies {doc}. After this archive, that change's spec was written against a now-stale base. Re-review with `/fab-continue` (review) after switching to it."*
3. **Hydrate into `fab/docs/`**:
   The agent reads `spec.md` and the current centralized doc, then rewrites the centralized doc to incorporate the changes:
   - **From spec.md** → integrate new/changed requirements and scenarios into the Requirements section. Remove requirements that the spec explicitly deprecates.
   The agent compares against the existing doc to determine what's new vs changed vs removed — no explicit delta markers needed. Minimize edits to unchanged sections to prevent drift over successive archives.
4. **Update status** to `archive: done` in `.status.yaml`
5. **Move change folder** to `archive/` (no rename — date is already in the folder name)
6. **Clear pointer** — delete `fab/current` (no active change)

**Order of operations**: Steps 3–6 are ordered to fail safely. Status is updated *before* the folder move, so if the move is interrupted, the change is marked archived but still in `changes/` — the agent can detect and complete the move on next invocation. The pointer is cleared last so that mid-archive, `/fab-status` still reports the active change rather than "no active change" with a half-hydrated spec.

**Recovery**: Hydration modifies centralized docs in-place. If the merge goes wrong (garbled text, incorrect removals), the only recovery is `git checkout` on the affected doc files. Commit (or at least review the diff) before pushing after an archive.

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
Stage:  spec (1/5)

Progress:
  ◉ spec        active
  ○ tasks       pending
  ○ apply       pending
  ○ review      pending
  ○ archive     pending

Checklist: not yet generated (created at tasks stage)

Next: Complete spec.md, then /fab-continue
```

---

## `/fab-hydrate-design [domain]`

**Purpose**: Identify structural gaps between `fab/docs/` and `fab/design/` and propose concise additions back to design specs with interactive confirmation.

**Context**: `fab/docs/index.md`, `fab/design/index.md`, all doc files, all design spec files

**Arguments**:
- `[domain]` *(optional)* — scope to a single doc domain. Scans all domains if omitted.

**Example**:
```
/fab-hydrate-design
→ "Found 5 structural gaps (showing top 3):"
→ Gap 1: Preflight Script — Source: preflight.md, Target: architecture.md
→ Shows exact markdown preview, asks: "Add this? (yes / no / done)"
```

**Behavior**:
1. Read all doc files to build a topic inventory (headings + summaries)
2. Read all spec files to build a coverage inventory (headings + inline mentions)
3. Cross-reference at section level — a gap is a doc topic with no spec coverage at all
4. Rank by impact (core behaviors > supporting concepts > implementation detail)
5. Present top 3 with exact markdown previews
6. Per-gap interactive confirm: yes (write), no (skip), done (stop)
7. Only confirmed additions are written to spec files

**Key properties**: No active change required. No git operations. Idempotent. Specs modified only with user confirmation.
