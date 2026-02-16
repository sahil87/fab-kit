---
name: fab-setup
description: "Set up a new project, manage config/constitution, or apply version migrations. Safe to re-run."
model_tier: fast
---

# /fab-setup [subcommand]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.
> **Exception**: `/fab-setup` has subcommand-specific context loading:
> - **bare / config / constitution**: Skip the "Always Load" context layer if files don't exist (first-run). Load them only if they already exist (re-run scenario).
> - **migrations**: Load `fab/config.yaml` (MUST exist). Skip Change Context loading — migrations operate on project-level files, not a specific change.

---

## Arguments

- **No arguments** — full structural bootstrap (default behavior)
- **`config [section]`** — create or update `fab/config.yaml` interactively. Optional `[section]` skips the menu and edits that section directly. Valid sections: `project`, `context`, `source_paths`, `stages`, `rules`, `checklist`, `git`, `naming`, `code_quality`.
- **`constitution`** — create or amend `fab/constitution.md` with semantic versioning
- **`migrations [file]`** — apply version migrations to bring project files in sync with the installed kit version (absorbed from fab-update)
- **`validate`** — redirect message: "Validation is built into `/fab-setup config` and `/fab-setup constitution` — each validates after every edit."

Any unrecognized argument triggers: "Unknown subcommand: {arg}. Valid: config, constitution, migrations. Run `/fab-setup` with no arguments for full setup."

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
| `migrations` | Proceed to **Migrations Behavior** (pass remaining args as file argument) |
| `validate` | Output redirect message and STOP |
| *(anything else)* | Output unknown subcommand message and STOP |

---

## Bootstrap Behavior

When invoked with no arguments, perform the full structural bootstrap. `/fab-setup` delegates directory/symlink/skeleton creation to `fab/.kit/scripts/fab-sync.sh` (step 1f) while handling interactive config/constitution generation itself.

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

Handled by `fab-sync.sh` (step 1f). The scaffold script creates `fab/VERSION` with version logic based on project state:

- **New project** (no `fab/config.yaml`): copies `fab/.kit/VERSION` value (engine version)
- **Existing project** (has `fab/config.yaml`, no `fab/VERSION`): writes `0.1.0` (base version, run `/fab-setup migrations` to migrate)
- **Already exists**: preserves existing `fab/VERSION` — no overwrite

On bootstrap output:
- New project: `Created: fab/VERSION ({engine_version})`
- Existing project: `Created: fab/VERSION (0.1.0 — existing project, run /fab-setup migrations to migrate)`
- Re-run: `fab/VERSION` reported as part of scaffold output (no modification)

#### 1f. `fab/changes/`

If missing: create `fab/changes/`, `fab/changes/archive/`, and `fab/changes/.gitkeep`.
If exists: ensure `fab/changes/archive/` exists, then skip.

#### 1g. `.claude/skills/` Symlinks

Run `fab/.kit/scripts/fab-sync.sh` to create or repair all skill symlinks, directories, and `fab/VERSION`. The script discovers skills by globbing `fab/.kit/skills/fab-*.md` and creates:

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

- **Update mode**: `fab/config.yaml` must exist. If missing (direct invocation): STOP with `fab/config.yaml not found. Run /fab-setup to create it.`
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

2. Process selection -> **Edit Section Flow**
3. After editing: "Update another section? (1-9 or 'done')"
4. Loop until Done

When invoked with a section argument: validate against valid sections (error if invalid), go directly to **Edit Section Flow**, then offer to update another section.

### Config Edit Section Flow

1. **Display current value** of the section
2. **Accept new value** — inline for simple values, block for multi-line
3. **Apply via string replacement** — targeted match, NOT full YAML rewrite (preserves comments)
4. **Validate** — YAML parseable, required fields present (`project.name`, `project.description`, `stages`), stage `requires` references valid
5. Pass -> confirm: `Updated {section}.` Fail -> report error, offer revert.

If no changes made, output: `No changes made. config.yaml unchanged.`

### Config Output

Show `Created fab/config.yaml` (create mode), `{N} sections updated in fab/config.yaml` (update mode), or `No changes made` (no-op). Next steps: `/fab-new` after create.

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

1. `fab/config.yaml` must exist. If missing (direct invocation): STOP with `fab/config.yaml not found. Run /fab-setup first.`
2. Read `fab/config.yaml` for project context
3. Check whether `fab/constitution.md` exists -> determines mode

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

Show `Created fab/constitution.md (version 1.0.0) with {N} principles.` (create) or amendment summary with `Version: {old} -> {new}` (update). Next steps: `/fab-new`.

### Constitution Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing (direct invocation) | Abort with guidance |
| `fab/constitution.md` malformed (update mode) | Warn: "Structure appears non-standard. Proceeding with best-effort parsing." |
| Governance section missing version | Warn and start from 1.0.0 |
| Roman numeral parsing fails | Warn and proceed with sequential numbering from I |

---

## Migrations Behavior

Compare `fab/VERSION` (local project version) to `fab/.kit/VERSION` (engine version), discover applicable migration files in `fab/.kit/migrations/`, and apply them sequentially. Each migration is a markdown instruction file — the skill reads it and executes the steps as an LLM agent.

When `[file]` is provided, read and apply that specific migration file directly, bypassing version range discovery.

### Migrations Context Loading

1. Read `fab/VERSION` and `fab/.kit/VERSION`
2. Read `fab/config.yaml` (Always Load layer — MUST exist). Skip Change Context.
3. Scan `fab/.kit/migrations/` for migration files

### Migrations Pre-flight Checks

Before attempting any migration, verify:

1. **`fab/VERSION` exists** — if not: STOP with `fab/VERSION not found. Run fab-sync.sh to create it.`
2. **`fab/.kit/VERSION` exists** — if not: STOP with `fab/.kit/VERSION not found — kit may be corrupted.`
3. **`fab/config.yaml` exists** — if not: STOP with `fab/config.yaml not found. Run /fab-setup to create it.`
4. Read both version strings and parse as `MAJOR.MINOR.PATCH` integers

### Migrations Step 1: Compare Versions

- Read `fab/VERSION` -> `current`
- Read `fab/.kit/VERSION` -> `target`
- If `current` >= `target`: report and stop (see scenarios below)

### Migrations Step 2: Discover Migrations

1. Scan `fab/.kit/migrations/` for files matching `{FROM}-to-{TO}.md`
2. Parse FROM and TO as semver from each filename
3. **Validate non-overlapping ranges**: for every pair of migration files, check that their ranges do not overlap (`A.FROM < B.TO AND B.FROM < A.TO` means overlap). If overlap detected: STOP with error listing the conflicting files
4. Sort migrations by FROM ascending

### Migrations Step 3: Apply Migrations (Loop)

Execute the migration discovery algorithm:

1. Find the first migration where `FROM <= current < TO`
2. **If found**: apply it (see [Applying a Migration](#applying-a-migration)), set `current = TO`, repeat from (1)
3. **If not found but a migration exists with `FROM > current`**: skip to that FROM — log: `No migration needed for {current} -> {FROM}, skipping.` — repeat from (1)
4. **If not found and no later migrations exist**: set `fab/VERSION` to engine version, done

### Migrations Step 4: Finalize

- Write `fab/VERSION` with the engine version (should already match after migrations)
- Output completion summary

---

## Applying a Migration

For each migration file:

1. **Read** the migration file `fab/.kit/migrations/{FROM}-to-{TO}.md`
2. **Execute Pre-check** section: verify each condition. If any fails -> STOP, report which pre-check failed, do NOT proceed
3. **Execute Changes** section: apply each change in order. Read referenced files, make modifications, write results
4. **Execute Verification** section: validate each condition. If any fails -> STOP, report which verification step failed
5. **Update version**: write `TO` to `fab/VERSION`

---

## Migrations Output Format

### Successful Multi-Step Migration

```
Local version:  {current}
Engine version: {target}
Migrations found: {N}

[1/{N}] Applying {FROM} -> {TO}...
{migration output}
-> fab/VERSION updated to {TO}

[2/{N}] Applying {FROM} -> {TO}...
{migration output}
-> fab/VERSION updated to {TO}

All migrations complete. fab/VERSION: {original} -> {final}
```

### Migration with Gap Skip

```
Local version:  {current}
Engine version: {target}
Migrations found: {N}

No migration needed for {current} -> {FROM}, skipping.

[1/{N}] Applying {FROM} -> {TO}...
{migration output}
-> fab/VERSION updated to {TO}

All migrations complete. fab/VERSION: {original} -> {final}
```

### Versions Already Equal

```
Already up to date ({version}).
```

### Local Version Ahead

```
Local version (fab/VERSION) is ahead of engine version (fab/.kit/VERSION): {local} > {engine}.
This is unexpected — check your fab/.kit/ installation.
```

### No Migrations Exist

```
Local version:  {current}
Engine version: {target}
No migrations found. fab/VERSION updated to {target}.
```

### Overlapping Ranges

```
Overlapping migration ranges detected: {file1} and {file2}. Fix the migrations directory.
```

### Mid-Chain Failure

```
[{N}/{total}] Applying {FROM} -> {TO}...
{partial output}
FAIL: Migration failed at {Pre-check|Changes|Verification} step: {description}
fab/VERSION remains at {current_version}.
Fix the issue and re-run /fab-setup migrations to continue from {current_version}.
```

---

## Semver Comparison

To compare two semver strings, compare MAJOR, then MINOR, then PATCH as integers. `A >= B` means A.MAJOR > B.MAJOR, or (A.MAJOR == B.MAJOR and A.MINOR > B.MINOR), or (A.MAJOR == B.MAJOR and A.MINOR == B.MINOR and A.PATCH >= B.PATCH).

---

## Idempotency

All paths are safe to re-run. Structural artifacts are created once (skipped on re-run). Symlinks are verified/repaired every run. Config/constitution edits are no-ops when unchanged. Migrations apply only remaining steps.

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No — project-level tool |
| Idempotent? | Yes |
| Modifies `fab/config.yaml`? | Yes (bootstrap creates, config subcommand updates, migrations may modify) |
| Modifies `fab/constitution.md`? | Yes (bootstrap creates, constitution subcommand updates, migrations may modify) |
| Modifies `fab/VERSION`? | Yes (migrations) |
| Modifies `fab/.kit/`? | No — migrations only touch project-level files |
| Requires active change? | No |

---

## Next Steps Reference

- After bootstrap: `/fab-new <description>` or `/docs-hydrate-memory <sources>`
- After config create: `/fab-new <description>`
- After config/constitution update: (no further action needed — validation is automatic)
- After constitution create: `/fab-new <description>`
- After migrations: `/fab-new` or `/fab-status`
