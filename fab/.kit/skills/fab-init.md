---
name: fab-init
description: "Bootstrap fab/ directory structure, or manage config/constitution/validation. Safe to re-run."
model_tier: fast
---

# /fab-init [subcommand]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.
> **Exception**: `/fab-init` skips the "Always Load" context layer (config and constitution don't exist yet on first run). Load them only if they already exist (re-run scenario).

---

## Arguments

- **No arguments** — full structural bootstrap (default behavior)
- **`config [section]`** — create or update `fab/config.yaml` interactively. Optional `[section]` skips the menu and edits that section directly. Valid sections: `project`, `context`, `source_paths`, `stages`, `rules`, `checklist`, `git`, `naming`, `code_quality`.
- **`constitution`** — create or amend `fab/constitution.md` with semantic versioning
- **`validate`** — validate structural correctness of `fab/config.yaml` and `fab/constitution.md`

Any unrecognized argument triggers: "Did you mean /docs-hydrate-memory? /fab-init no longer accepts source arguments."

---

## Pre-flight Check

Before doing anything else, verify the kit exists:

1. Check that `fab/.kit/` directory exists
2. Check that `fab/.kit/VERSION` file exists and is readable

**If either check fails, STOP immediately.** Output: `fab/.kit/ not found. Copy the kit directory into fab/.kit/ first — see the Getting Started guide.` Do NOT create any files.

### Argument Classification

| First argument | Action |
|----------------|--------|
| *(none)* | Proceed to **Bootstrap Behavior** |
| `config` | Proceed to **Config Behavior** (pass remaining args as section argument) |
| `constitution` | Proceed to **Constitution Behavior** |
| `validate` | Proceed to **Validate Behavior** |
| *(anything else)* | Output redirect message and STOP |

---

## Bootstrap Behavior

When invoked with no arguments, perform the full structural bootstrap. `/fab-init` delegates directory/symlink/skeleton creation to `fab/.kit/scripts/lib/sync-workspace.sh` (step 1f) while handling interactive config/constitution generation itself.

### Phase 1: Structural Bootstrap

Each step is **idempotent** — skip if the artifact already exists and is valid. On re-run, verify and repair rather than recreate.

#### 1a. `fab/config.yaml`

If missing: execute **Config Behavior** (below) in create mode.
If exists: report "config.yaml already exists — skipping".

#### 1b. `fab/constitution.md`

If missing: execute **Constitution Behavior** (below) in create mode.
If exists: report "constitution.md already exists — skipping".

#### 1c. `docs/memory/index.md`

If missing, create `docs/memory/` directory and `docs/memory/index.md`:

```markdown
# Memory Index

<!-- This index is maintained by /fab-archive when changes are completed. -->
<!-- Each domain gets a row linking to its memory files. -->

| Domain | Description | Memory Files |
|--------|-------------|--------------|
```

If exists: skip.

#### 1d. `docs/specs/index.md`

If missing, create `docs/specs/` directory and `docs/specs/index.md`:

```markdown
# Specifications Index

> **Specs are pre-implementation artifacts** — what you *planned*. They capture conceptual design
> intent, high-level decisions, and the "why" behind features. Specs are human-curated,
> flat in structure, and deliberately size-controlled for quick reading.
>
> Contrast with [`docs/memory/index.md`](../memory/index.md): memory files are *post-implementation* —
> what actually happened. Memory files are the authoritative source of truth for system behavior,
> maintained by `/fab-continue` (hydrate) and `/fab-archive`.
>
> **Ownership**: Specs are written and maintained by humans. No automated tooling creates or
> enforces structure here — organize files however makes sense for your project.

| Spec | Description |
|------|-------------|
```

If exists: skip.

#### 1e. `fab/VERSION`

Handled by `lib/sync-workspace.sh` (step 1f). The scaffold script creates `fab/VERSION` with version logic based on project state:

- **New project** (no `fab/config.yaml`): copies `fab/.kit/VERSION` value (engine version)
- **Existing project** (has `fab/config.yaml`, no `fab/VERSION`): writes `0.1.0` (base version, run `/fab-update` to migrate)
- **Already exists**: preserves existing `fab/VERSION` — no overwrite

On bootstrap output:
- New project: `Created: fab/VERSION ({engine_version})`
- Existing project: `Created: fab/VERSION (0.1.0 — existing project, run /fab-update to migrate)`
- Re-run: `fab/VERSION` reported as part of scaffold output (no modification)

#### 1f. `fab/changes/`

If missing: create `fab/changes/`, `fab/changes/archive/`, and `fab/changes/.gitkeep`.
If exists: ensure `fab/changes/archive/` exists, then skip.

#### 1g. `.claude/skills/` Symlinks

Run `fab/.kit/scripts/lib/sync-workspace.sh` to create or repair all skill symlinks, directories, and `fab/VERSION`. The script discovers skills by globbing `fab/.kit/skills/fab-*.md` and creates:

```
.claude/skills/fab-{name}/SKILL.md → ../../../fab/.kit/skills/fab-{name}.md
```

If the script cannot execute, perform the equivalent manually:
1. For each `fab-*.md` in `fab/.kit/skills/`, create the symlink (skip `_context.md`)
2. If symlink resolves correctly, skip; if broken, remove and recreate
3. Use **relative paths** — never absolute
4. Do NOT modify existing content in `.claude/skills/`

Report how many symlinks were created, repaired, or already valid.

#### 1h. `.gitignore` — append `fab/current`

Read `.gitignore` (create if missing). If `fab/current` is not listed, append it.

### Bootstrap Output

```
Found fab/.kit/ (v{VERSION}). Initializing project...
{config.yaml prompts and creation}
{constitution.md generation}
Created: fab/config.yaml
Created: fab/constitution.md
Created: fab/VERSION ({version})
Created: docs/memory/index.md
Created: docs/specs/index.md
Created: fab/changes/
Created: 11 symlinks in .claude/skills/
Updated: .gitignore (added fab/current)
fab/ initialized successfully.

Next: /fab-new <description> or /docs-hydrate-memory <sources>
```

On re-run, report each artifact as OK/repaired instead of Created, ending with `fab/ structure verified.`

---

## Config Behavior

Create a new `fab/config.yaml` interactively or update specific sections. Preserves YAML comments via targeted string replacement. Validates after each edit.

**Context loading**: Loads `fab/config.yaml` only (the file being edited). Does NOT load constitution, memory, or specs.

### Config Arguments

- **`[section]`** *(optional)* — section to edit directly, skipping the menu. Valid values: `project`, `context`, `source_paths`, `stages`, `rules`, `checklist`, `git`, `naming`.

### Config Pre-flight

- **Update mode**: `fab/config.yaml` must exist. If missing (direct invocation): STOP with `fab/config.yaml not found. Run /fab-init to create it.`
- **Create mode** (from bootstrap): `fab/config.yaml` does not exist.

### Config Create Mode

When `fab/config.yaml` does not exist:

1. Read the project's README, package.json, or other root-level files for context
2. Ask the user: project name, description, tech stack/conventions, source paths
3. Generate `fab/config.yaml`:

```yaml
# fab/config.yaml

project:
  name: "{PROJECT_NAME}"
  description: "{PROJECT_DESCRIPTION}"

context: |
  {TECH_STACK_AND_CONVENTIONS}

naming:
  format: "{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}"

git:
  enabled: true
  branch_prefix: ""

stages:
  - id: intake
    generates: intake.md
    required: true
  - id: spec
    generates: spec.md
    requires: [intake]
    required: true
  - id: tasks
    generates: tasks.md
    requires: [spec]
    required: true
    auto_checklist: true
  - id: apply
    requires: [tasks]
  - id: review
    requires: [apply]
  - id: hydrate
    requires: [review]

source_paths:
  - {SOURCE_PATHS}

checklist:
  extra_categories: []

rules:
  spec:
    - Use GIVEN/WHEN/THEN for scenarios
    - "Mark ambiguities with [NEEDS CLARIFICATION]"

# code_quality — Optional coding standards consumed during apply and review.
#   Projects opt in by uncommenting. All fields are independently optional.
#
# code_quality:
#   # principles — Positive coding standards to follow during implementation.
#   principles:
#     - "Readability and maintainability over cleverness"
#     - "Follow existing project patterns unless there's compelling reason to deviate"
#     - "Prefer composition over inheritance"
#
#   # anti_patterns — Patterns to avoid. Flagged during review.
#   anti_patterns:
#     - "God functions (>50 lines without clear reason)"
#     - "Duplicating existing utilities instead of reusing them"
#     - "Magic strings or numbers without named constants"
#
#   # test_strategy — How tests relate to implementation.
#   # Values: test-alongside (default) | test-after | tdd
#   test_strategy: "test-alongside"
```

4. Output: `Created fab/config.yaml`

### Config Update Mode — Menu Flow

When invoked without a section argument:

1. Display the section menu:

```
fab/config.yaml sections:
1. project     — name and description
2. context     — tech stack and conventions
3. source_paths — implementation code directories
4. stages      — pipeline stage definitions
5. rules       — per-stage generation rules
6. checklist   — extra quality categories
7. git         — branch integration settings
8. naming      — change folder naming format
9. code_quality — coding standards for apply/review
10. Done

Which section to update? (1-10)
```

2. Process selection → **Edit Section Flow**
3. After editing: "Update another section? (1-9 or 'done')"
4. Loop until Done

When invoked with a section argument: validate against valid sections (error if invalid), go directly to **Edit Section Flow**, then offer to update another section.

### Config Edit Section Flow

1. **Display current value** of the section
2. **Accept new value** — inline for simple values, block for multi-line
3. **Apply via string replacement** — targeted match, NOT full YAML rewrite (preserves comments)
4. **Validate** — YAML parseable, required fields present (`project.name`, `project.description`, `stages`), stage `requires` references valid
5. Pass → confirm: `Updated {section}.` Fail → report error, offer revert.

If no changes made, output: `No changes made. config.yaml unchanged.`

### Config Output

Show `Created fab/config.yaml` (create mode), `{N} sections updated in fab/config.yaml` (update mode), or `No changes made` (no-op). Next steps: `/fab-new` after create, `/fab-init validate` after update.

### Config Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing (update mode, direct invocation) | Abort with creation guidance |
| Invalid section argument | Output valid section names |
| YAML parse failure after edit | Report error, offer revert |
| Missing required field after edit | Report which field, offer revert |
| Broken stage reference after edit | Report which stage and reference, offer revert |
| String replacement target not found | Warn about manual reformatting, fall back to section insert |

---

## Constitution Behavior

Create a new project constitution or amend an existing one with semantic versioning and structural preservation.

**Context loading**: Loads `fab/config.yaml` and `fab/constitution.md` (if it exists). Does NOT load memory or specs.

### Constitution Pre-flight

1. `fab/config.yaml` must exist. If missing (direct invocation): STOP with `fab/config.yaml not found. Run /fab-init first.`
2. Read `fab/config.yaml` for project context
3. Check whether `fab/constitution.md` exists → determines mode

### Constitution Create Mode

When `fab/constitution.md` does not exist:

1. Read project context from `fab/config.yaml` + README, existing docs, codebase structure
2. Generate `fab/constitution.md`:

```markdown
# {Project Name} Constitution

## Core Principles

### I. {Principle Name}
{Description using MUST/SHALL/SHOULD keywords. Include rationale.}

### II. {Principle Name}
{Description}

<!-- Generate 3-7 principles based on the project's actual patterns, tech stack, and constraints -->

## Additional Constraints
<!-- Project-specific: security, performance, testing, etc. -->

## Governance

**Version**: 1.0.0 | **Ratified**: {TODAY'S DATE} | **Last Amended**: {TODAY'S DATE}
```

3. Output: `Created fab/constitution.md (version 1.0.0) with {N} principles.`

### Constitution Update Mode

When `fab/constitution.md` already exists:

1. Read and display current content, read version from Governance
2. Present amendment menu:

```
Current constitution: version {X.Y.Z}, {N} principles

What would you like to change?
1. Add a new principle
2. Modify an existing principle
3. Remove a principle
4. Add or modify a constraint
5. Update governance metadata
6. Done — no changes
```

3. Process selection:
   - **Add**: Ask for name/description, insert at next Roman numeral. Bump: MINOR.
   - **Modify**: Show numbered list, accept new text. Ask: "(1) fundamental change or (2) wording clarification?" Bump: MAJOR or PATCH.
   - **Remove**: Show numbered list, re-number remaining. Bump: MAJOR.
   - **Add/modify constraint**: Show section, accept edits. Bump: MINOR (add) or PATCH (modify).
   - **Update governance**: Allow metadata edits. Bump: PATCH.
   - **Done**: Proceed to version bump.

4. After each action: "Any other changes? (yes/no)" — loop or proceed.

5. **Version bump**: Apply highest-severity bump across all amendments (MAJOR > MINOR > PATCH). Update Governance: increment version, set "Last Amended" to today.

6. **Structural preservation**: Verify heading hierarchy, sequential Roman numerals, Governance format. Re-number if needed.

7. Write updated file. If no changes: `No changes made. Constitution unchanged at version {X.Y.Z}.`

### Constitution Output

Show `Created fab/constitution.md (version 1.0.0) with {N} principles.` (create) or amendment summary with `Version: {old} → {new}` (update). Next steps: `/fab-init validate` or `/fab-new`.

### Constitution Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing (direct invocation) | Abort with guidance |
| `fab/constitution.md` malformed (update mode) | Warn: "Structure appears non-standard. Proceeding with best-effort parsing." |
| Governance section missing version | Warn and start from 1.0.0 |
| Roman numeral parsing fails | Warn and proceed with sequential numbering from I |

---

## Validate Behavior

Validate structural correctness of `fab/config.yaml` and `fab/constitution.md`. Reports issues with actionable fix suggestions.

**Context loading**: Reads both files for validation. Does NOT load memory or specs.

### Validate Pre-flight

No preflight script — this behavior validates the files preflight would normally check, so it handles missing files gracefully.

### Validate Step 1: Discover Files

Check existence of `fab/config.yaml` and `fab/constitution.md`. Missing files are reported but do not block validation of the other file.

### Validate Step 2: Validate `config.yaml`

If exists, run these 8 checks in order (if check 1 fails, skip 2-8):

| # | Check | Pass criteria | Failure message | Fix suggestion |
|---|-------|---------------|-----------------|----------------|
| 1 | YAML parseable | Valid YAML | "FAIL: not valid YAML: {error}" | "Fix the syntax error at the indicated location" |
| 2 | Required top-level keys | `project`, `context`, `stages`, `source_paths` present | "FAIL: Missing required key '{key}'" | "Add `{key}:` as a top-level section" |
| 3 | `project.name` non-empty | String, length > 0 | "FAIL: project.name missing or empty" | "Add `name: \"your-project\"` under `project:`" |
| 4 | `project.description` non-empty | String, length > 0 | "FAIL: project.description missing or empty" | "Add `description: \"...\"` under `project:`" |
| 5 | `stages` non-empty list | Array with ≥1 entry | "FAIL: stages list empty" | "Add at least the default stages" |
| 6 | Stage `id` fields present | Every stage has an `id` string | "FAIL: Stage at index {N} missing `id`" | "Add `id: {suggested}` to the stage entry" |
| 7 | Stage `requires` valid | Every `requires` references existing stage ID | "FAIL: Stage '{id}' requires non-existent '{ref}'" | "Valid stage IDs are: {list}" |
| 8 | No circular dependencies | No cycles in dependency graph | "FAIL: Circular dependency: {cycle}" | "Remove one `requires` entry to break the cycle" |

Also check: stage IDs are unique ("FAIL: Duplicate stage ID '{id}'").

If file missing: `config.yaml: not found — run /fab-init or /fab-init config to create it`

### Validate Step 3: Validate `constitution.md`

If exists, run these 6 checks:

| # | Check | Pass criteria | Failure message | Fix suggestion |
|---|-------|---------------|-----------------|----------------|
| 1 | Non-empty | Has content | "FAIL: empty" | "Run /fab-init constitution to generate content" |
| 2 | Level-1 heading | Contains `# ... Constitution` | "FAIL: Missing level-1 heading with 'Constitution'" | "Add `# {Name} Constitution` as first heading" |
| 3 | Core Principles section | Contains `## Core Principles` | "FAIL: Missing `## Core Principles`" | "Add section with at least one principle" |
| 4 | Roman numeral headings | At least one `### I.` etc. | "FAIL: No Roman numeral headings found" | "Number principles: `### I. {Name}`, `### II. {Name}`" |
| 5 | Governance section | Contains `## Governance` | "FAIL: Missing `## Governance`" | "Add Governance with version, dates" |
| 6 | Version format | `MAJOR.MINOR.PATCH` in Governance | "FAIL: No version found" | "Add `**Version**: 1.0.0`" |

If a check depends on a failed earlier check, mark as "skipped" with reason.

If file missing: `constitution.md: not found — run /fab-init or /fab-init constitution to create it`

### Validate Step 4: Combined Report

```
config.yaml checks:
  ✓ YAML parseable
  ✓ Required top-level keys present
  ✗ project.name is missing or empty
    → Add `name: "your-project"` under the `project:` section
  ...

config.yaml:      {passed}/{total} checks passed {✓ or ✗}
constitution.md:  {passed}/{total} checks passed {✓ or ✗}

{All passed → "All validation checks passed." | Failures → "{N} issue(s) found. Fix and re-run /fab-init validate."}
{Both missing → "No files to validate. Run /fab-init to bootstrap the project."}
```

---

## Idempotency

All paths are safe to re-run. Structural artifacts are created once (skipped on re-run). Symlinks are verified/repaired every run. Config/constitution edits are no-ops when unchanged. Validate is read-only.

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No — project-level tool |
| Idempotent? | Yes |
| Modifies `fab/config.yaml`? | Yes (bootstrap creates, config subcommand updates) |
| Modifies `fab/constitution.md`? | Yes (bootstrap creates, constitution subcommand updates) |

---

## Next Steps Reference

- After bootstrap: `/fab-new <description>` or `/docs-hydrate-memory <sources>`
- After config create: `/fab-new <description>`
- After config/constitution update: `/fab-init validate`
- After constitution create: `/fab-init validate` or `/fab-new <description>`
- After validate (pass): `/fab-new` or `/fab-init config` or `/fab-init constitution`
- After validate (fail): Fix issues, re-run `/fab-init validate`
