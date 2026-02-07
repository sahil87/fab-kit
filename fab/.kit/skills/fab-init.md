---
name: fab-init
description: "Bootstrap fab/ directory and optionally hydrate docs from external sources. Safe to re-run."
---

# /fab:init [sources...]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.
> **Exception**: `/fab:init` skips the "Always Load" context layer (config and constitution don't exist yet on first run). Load them only if they already exist (re-run scenario).

---

## Purpose

Bootstrap `fab/` in an existing project and optionally hydrate `fab/docs/` from external documentation sources. Safe to run repeatedly — structural artifacts are created once (skipped if they already exist), symlinks are repaired if broken, and sources are ingested additively.

---

## Pre-flight Check

Before doing anything else, verify the kit exists:

1. Check that `fab/.kit/` directory exists
2. Check that `fab/.kit/VERSION` file exists and is readable

**If either check fails, STOP immediately.** Output this message and do nothing else:

> `fab/.kit/ not found. Copy the kit directory into fab/.kit/ first — see the Getting Started guide.`

Do NOT create partial structure. Do NOT create `fab/config.yaml`, `fab/constitution.md`, or any other file. The kit must be in place before init can run.

---

## Arguments

- **`[sources...]`** *(optional)* — one or more URLs or local paths containing documentation to ingest into `fab/docs/`. Supported source types:
  - **Notion URLs** — pages or databases (fetch via Notion MCP or API)
  - **Linear URLs** — issues or projects (fetch via Linear MCP or API)
  - **Local files/directories** — markdown, text, or directories of docs (read from filesystem; if directory, read all markdown files recursively)

If no sources are provided, only the structural bootstrap runs.

---

## Behavior

### Phase 1: Structural Bootstrap

Each step is **idempotent** — skip if the artifact already exists and is valid. On re-run, verify and repair rather than recreate.

#### 1a. `fab/config.yaml`

If `fab/config.yaml` does **not** exist:

1. Read the project's README, package.json, or other root-level files to gather context
2. Ask the user:
   - **Project name** — short identifier (e.g., `my-app`)
   - **Description** — one-line summary of the project
   - **Tech stack and conventions** — languages, frameworks, API style, testing approach, etc.
3. Generate `fab/config.yaml` with this structure:

```yaml
# fab/config.yaml

project:
  name: "{PROJECT_NAME}"
  description: "{PROJECT_DESCRIPTION}"

context: |
  {TECH_STACK_AND_CONVENTIONS}

naming:
  format: "{YYMMDD}-{XXXX}-{slug}"

git:
  enabled: true
  branch_prefix: ""

stages:
  - id: proposal
    generates: proposal.md
    required: true
  - id: specs
    generates: spec.md
    requires: [proposal]
    required: true
  - id: plan
    generates: plan.md
    requires: [specs]
    required: false
  - id: tasks
    generates: tasks.md
    requires: [specs]
    required: true
    auto_checklist: true
  - id: apply
    requires: [tasks]
  - id: review
    requires: [apply]
  - id: archive
    requires: [review]

checklist:
  extra_categories: []

rules:
  plan: []
  specs:
    - Use GIVEN/WHEN/THEN for scenarios
    - "Mark ambiguities with [NEEDS CLARIFICATION]"
```

If `fab/config.yaml` **already exists**: report "config.yaml already exists — skipping" and move on.

#### 1b. `fab/constitution.md`

If `fab/constitution.md` does **not** exist:

1. Load `fab/config.yaml` (just created or already present)
2. Examine the project context: README, existing documentation, codebase structure, and conversation history
3. Generate `fab/constitution.md` with principles derived from the project's actual patterns and constraints. Use this structure:

```markdown
# {Project Name} Constitution

## Core Principles

### I. {Principle Name}
{Description using MUST/SHALL/SHOULD keywords. Include rationale.}

### II. {Principle Name}
{Description}

<!-- Add 3-7 principles based on the project's actual patterns, tech stack, and constraints -->

## Additional Constraints
<!-- Project-specific: security, performance, testing, etc. -->

## Governance

**Version**: 1.0.0 | **Ratified**: {TODAY'S DATE} | **Last Amended**: {TODAY'S DATE}
```

If `fab/constitution.md` **already exists**: report "constitution.md already exists — skipping" and move on.

#### 1c. `fab/docs/index.md`

If `fab/docs/index.md` does **not** exist:

1. Create `fab/docs/` directory if needed
2. Create `fab/docs/index.md` with an empty index:

```markdown
# Documentation Index

<!-- This index is maintained by /fab:archive when changes are completed. -->
<!-- Each domain gets a row linking to its docs. -->

| Domain | Description | Docs |
|--------|-------------|------|
```

If `fab/docs/index.md` **already exists**: report "docs/index.md already exists — skipping" and move on.

#### 1d. `fab/changes/`

If `fab/changes/` directory does **not** exist:

1. Create `fab/changes/` directory
2. Create `fab/changes/.gitkeep` to ensure git tracks the empty directory

If `fab/changes/` **already exists**: report "changes/ already exists — skipping" and move on.

#### 1e. `.claude/skills/` Symlinks

Create or repair symlinks in `.claude/skills/` pointing into `fab/.kit/skills/`. There are **10 skill symlinks** (all skills except `_context.md`, which is internal to the kit):

| Symlink | Target |
|---------|--------|
| `.claude/skills/fab-init.md` | `../../fab/.kit/skills/fab-init.md` |
| `.claude/skills/fab-new.md` | `../../fab/.kit/skills/fab-new.md` |
| `.claude/skills/fab-continue.md` | `../../fab/.kit/skills/fab-continue.md` |
| `.claude/skills/fab-ff.md` | `../../fab/.kit/skills/fab-ff.md` |
| `.claude/skills/fab-clarify.md` | `../../fab/.kit/skills/fab-clarify.md` |
| `.claude/skills/fab-apply.md` | `../../fab/.kit/skills/fab-apply.md` |
| `.claude/skills/fab-review.md` | `../../fab/.kit/skills/fab-review.md` |
| `.claude/skills/fab-archive.md` | `../../fab/.kit/skills/fab-archive.md` |
| `.claude/skills/fab-switch.md` | `../../fab/.kit/skills/fab-switch.md` |
| `.claude/skills/fab-status.md` | `../../fab/.kit/skills/fab-status.md` |

For each symlink:

1. If `.claude/skills/` directory does not exist, create it
2. If the symlink already exists and resolves correctly (`test -e` passes), skip it
3. If the symlink exists but is broken (dangling), remove it and recreate
4. If the symlink does not exist, create it using: `ln -s ../../fab/.kit/skills/fab-{name}.md .claude/skills/fab-{name}.md`

**Important**: Use relative paths (`../../fab/.kit/skills/`) so symlinks work after cloning the repo. Do NOT use absolute paths.

**Important**: Do NOT modify or remove any existing content in `.claude/skills/` (e.g., `commit/`, `dev-browser/`, `prd/`).

Report how many symlinks were created, repaired, or already valid.

#### 1f. `.gitignore` — append `fab/current`

1. Read `.gitignore` at the project root (create it if it doesn't exist)
2. Check if `fab/current` is already listed (exact line match or with trailing whitespace/comment)
3. If not present, append `fab/current` on a new line
4. If already present, skip

---

### Phase 2: Source Hydration

**Only runs when `[sources...]` arguments are provided.** If no sources were given, skip this phase entirely.

For each source provided:

#### 2a. Fetch/Read Source Content

- **Notion URL** (contains `notion.so` or `notion.site`): Fetch page or database content via Notion MCP tool or API. Extract the page title and body content.
- **Linear URL** (contains `linear.app`): Fetch issue or project content via Linear MCP tool or API. Extract title, description, and relevant details.
- **Local file path**: Read the file content directly. If it's a directory, recursively read all `.md` files within it.

Report each source as it's fetched: `Fetched: {title or filename} ({source type})`

#### 2b. Analyze and Map to Domains

For each fetched source:

1. Analyze the content to identify **domains** (logical topic areas, e.g., `auth`, `payments`, `api`, `infrastructure`)
2. Identify individual **topics** within each domain (e.g., `authentication`, `authorization` within `auth`)
3. Map each topic to a target file: `fab/docs/{domain}/{topic}.md`

#### 2c. Create or Merge Documentation

For each identified topic:

1. If `fab/docs/{domain}/` directory does not exist, create it
2. If `fab/docs/{domain}/index.md` does not exist, create a domain index
3. If the target doc file does not exist, create it with the analyzed content, structured as a centralized doc (with Requirements sections, changelog, etc.)
4. If the target doc file already exists, **merge** the new content into the existing doc — add new requirements, update existing ones, do not remove existing content

#### 2d. Update `fab/docs/index.md`

After all sources are processed:

1. Read the current `fab/docs/index.md`
2. Add rows for any new domains that were created
3. Update descriptions if domains were expanded with new docs
4. Do not remove existing entries

Report what was created and updated:
```
Created: fab/docs/{domain}/{topic}.md
Updated: fab/docs/index.md
```

---

## Output

### First Run (fresh bootstrap)

```
Found fab/.kit/ (v{VERSION}). Initializing project...
{config.yaml prompts and creation}
{constitution.md generation}
Created: fab/config.yaml
Created: fab/constitution.md
Created: fab/docs/index.md
Created: fab/changes/
Created: 10 symlinks in .claude/skills/
Updated: .gitignore (added fab/current)
fab/ initialized successfully.
```

### Re-run (structural health check)

```
Found fab/.kit/ (v{VERSION}). Verifying structure...
config.yaml — OK
constitution.md — OK
docs/index.md — OK
changes/ — OK
Symlinks: 10/10 valid (repaired 1)
.gitignore: fab/current present
fab/ structure verified.
```

### With Sources

```
Found fab/.kit/ (v{VERSION}). Verifying structure...
{structural check output}
Hydrating docs from 2 sources...
Fetched: API Spec (Notion)
Fetched: 3 files from ./legacy-docs/payments/
Created: fab/docs/api/endpoints.md
Created: fab/docs/api/authentication.md
Created: fab/docs/payments/checkout.md
Created: fab/docs/payments/refunds.md
Updated: fab/docs/index.md
```

---

## Idempotency Guarantee

This skill is safe to run any number of times:

- **Config and constitution**: Created once, never overwritten on re-run
- **Docs index**: Created once, only updated (not replaced) during hydration
- **Changes directory**: Created once, never touched on re-run
- **Symlinks**: Verified and repaired on every run — broken symlinks are fixed, valid ones are left alone
- **`.gitignore`**: Entry is appended only if not already present
- **Source hydration**: Merges into existing docs, does not overwrite or delete

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/.kit/` missing | Abort immediately with guidance message. Do NOT create any files. |
| `fab/.kit/VERSION` unreadable | Abort with: "fab/.kit/VERSION not found or unreadable — kit may be corrupted." |
| Source URL unreachable | Report the error for that source, continue with remaining sources |
| Source content unreadable | Report the error, skip that source, continue |
| Symlink target missing | Report which skill file is missing in `fab/.kit/skills/` — do NOT create a broken symlink |

---

Next: `/fab:new <description>`
