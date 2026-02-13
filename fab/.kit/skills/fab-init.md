---
name: fab-init
description: "Bootstrap fab/ directory structure, or manage config/constitution/validation. Safe to re-run."
model_tier: fast
---

# /fab-init [subcommand]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.
> **Exception**: `/fab-init` skips the "Always Load" context layer (config and constitution don't exist yet on first run). Load them only if they already exist (re-run scenario).

---

## Purpose

Bootstrap `fab/` in an existing project, or manage its configuration artifacts. Safe to run repeatedly — structural artifacts are created once (skipped if they already exist) and symlinks are repaired if broken.

---

## Arguments

- **No arguments** — full structural bootstrap (default behavior)
- **`config [section]`** — create or update `fab/config.yaml` interactively. Optional `[section]` argument skips the menu and edits that section directly. Valid sections: `project`, `context`, `source_paths`, `stages`, `rules`, `checklist`, `git`, `naming`.
- **`constitution`** — create or amend `fab/constitution.md` with semantic versioning
- **`validate`** — validate structural correctness of `fab/config.yaml` and `fab/constitution.md`

Any argument that is not a recognized subcommand (`config`, `constitution`, `validate`) is treated as the old source-hydration interface and triggers a redirect message.

---

## Pre-flight Check

Before doing anything else, verify the kit exists:

1. Check that `fab/.kit/` directory exists
2. Check that `fab/.kit/VERSION` file exists and is readable

**If either check fails, STOP immediately.** Output this message and do nothing else:

> `fab/.kit/ not found. Copy the kit directory into fab/.kit/ first — see the Getting Started guide.`

Do NOT create partial structure. Do NOT create `fab/config.yaml`, `fab/constitution.md`, or any other file. The kit must be in place before init can run.

### Argument Classification

After the kit check passes, classify the first argument:

| First argument | Action |
|----------------|--------|
| *(none)* | Proceed to **Bootstrap Behavior** |
| `config` | Proceed to **Config Behavior** (pass remaining args as section argument) |
| `constitution` | Proceed to **Constitution Behavior** |
| `validate` | Proceed to **Validate Behavior** |
| *(anything else)* | Output: "Did you mean /fab-hydrate? /fab-init no longer accepts source arguments." and STOP |

---

## Bootstrap Behavior

When invoked with no arguments, perform the full structural bootstrap.

### Delegation Pattern

`/fab-init` delegates structural setup to `fab/.kit/scripts/_fab-scaffold.sh` (invoked in step 1f) and only adds interactive/configuration artifacts on top. This separation keeps the script automatable for CI and bootstrap workflows while the skill handles project-specific configuration that requires user input.

| Responsibility | Owner | Why |
|---|---|---|
| Directories, skeleton files, symlinks, .gitignore, .envrc | `_fab-scaffold.sh` | Scriptable, automatable, no user input needed |
| `config.yaml` (interactive) | Config Behavior section (below) | Single source of truth for config generation and updates |
| `constitution.md` (interactive) | Constitution Behavior section (below) | Single source of truth for constitution generation and amendments |
| Invoking `_fab-scaffold.sh` | `/fab-init` (step 1f) | Ensures structural setup runs as part of init |

Steps 1c–1e below have idempotent guards (`if not exists`) so they gracefully skip when `_fab-scaffold.sh` has already created the structural artifacts.

### Phase 1: Structural Bootstrap

Each step is **idempotent** — skip if the artifact already exists and is valid. On re-run, verify and repair rather than recreate.

#### 1a. `fab/config.yaml`

If `fab/config.yaml` does **not** exist:

Execute the **Config Behavior** (below) in create mode. This ensures a single source of truth for config generation logic.

If `fab/config.yaml` **already exists**: report "config.yaml already exists — skipping" and move on.

#### 1b. `fab/constitution.md`

If `fab/constitution.md` does **not** exist:

Execute the **Constitution Behavior** (below) in create mode. This ensures a single source of truth for constitution generation logic.

If `fab/constitution.md` **already exists**: report "constitution.md already exists — skipping" and move on.

#### 1c. `fab/memory/index.md`

If `fab/memory/index.md` does **not** exist:

1. Create `fab/memory/` directory if needed
2. Create `fab/memory/index.md` with an empty index:

```markdown
# Memory Index

<!-- This index is maintained by /fab-archive when changes are completed. -->
<!-- Each domain gets a row linking to its memory files. -->

| Domain | Description | Memory Files |
|--------|-------------|--------------|
```

If `fab/memory/index.md` **already exists**: report "memory/index.md already exists — skipping" and move on.

#### 1d. `fab/specs/index.md`

If `fab/specs/index.md` does **not** exist:

1. Create `fab/specs/` directory if needed
2. Create `fab/specs/index.md` with an empty index:

```markdown
# Specifications Index

> **Specs are pre-implementation artifacts** — what you *planned*. They capture conceptual design
> intent, high-level decisions, and the "why" behind features. Specs are human-curated,
> flat in structure, and deliberately size-controlled for quick reading.
>
> Contrast with [`fab/memory/index.md`](../memory/index.md): memory files are *post-implementation* —
> what actually happened. Memory files are the authoritative source of truth for system behavior,
> maintained by `/fab-continue` (hydrate) and `/fab-archive`.
>
> **Ownership**: Specs are written and maintained by humans. No automated tooling creates or
> enforces structure here — organize files however makes sense for your project.

| Spec | Description |
|------|-------------|
```

If `fab/specs/index.md` **already exists**: report "specs/index.md already exists — skipping" and move on.

#### 1e. `fab/changes/`

If `fab/changes/` directory does **not** exist:

1. Create `fab/changes/` directory
2. Create `fab/changes/archive/` subdirectory (pre-created so archive behavior doesn't need a separate `mkdir`)
3. Create `fab/changes/.gitkeep` to ensure git tracks the empty directory

If `fab/changes/` **already exists**: ensure `fab/changes/archive/` exists (create if missing), then report "changes/ already exists — skipping" and move on.

#### 1f. `.claude/skills/` Symlinks

Run `fab/.kit/scripts/_fab-scaffold.sh` to create or repair all skill symlinks and directories. This script is the **single source of truth** for the structural bootstrap — it handles directories, symlinks, memory index, and `.gitignore`.

The script discovers skills dynamically by globbing `fab/.kit/skills/fab-*.md` — no hardcoded list to maintain. Each discovered skill gets a subdirectory symlink:

```
.claude/skills/fab-{name}/SKILL.md → ../../../fab/.kit/skills/fab-{name}.md
```

If the script cannot be executed (e.g., Windows without bash), perform the equivalent manually:

1. For each `fab-*.md` file in `fab/.kit/skills/`, create `.claude/skills/fab-{name}/SKILL.md` as a relative symlink to `../../../fab/.kit/skills/fab-{name}.md`
2. Skip `_context.md` (internal, not a skill)
3. If a symlink already exists and resolves correctly (`test -e` passes), skip it
4. If a symlink is broken (dangling), remove and recreate it

**Important**: Use relative paths so symlinks work after cloning the repo. Do NOT use absolute paths.

**Important**: Do NOT modify or remove any existing content in `.claude/skills/` (e.g., `commit/`, `dev-browser/`, `prd/`).

Report how many symlinks were created, repaired, or already valid.

#### 1g. `.gitignore` — append `fab/current`

1. Read `.gitignore` at the project root (create it if it doesn't exist)
2. Check if `fab/current` is already listed (exact line match or with trailing whitespace/comment)
3. If not present, append `fab/current` on a new line
4. If already present, skip

### Bootstrap Output

#### First Run (fresh bootstrap)

```
Found fab/.kit/ (v{VERSION}). Initializing project...
{config.yaml prompts and creation}
{constitution.md generation}
Created: fab/config.yaml
Created: fab/constitution.md
Created: fab/memory/index.md
Created: fab/specs/index.md
Created: fab/changes/
Created: 11 symlinks in .claude/skills/
Updated: .gitignore (added fab/current)
fab/ initialized successfully.

Next: /fab-new <description> or /fab-hydrate <sources>
```

#### Re-run (structural health check)

```
Found fab/.kit/ (v{VERSION}). Verifying structure...
config.yaml — OK
constitution.md — OK
memory/index.md — OK
specs/index.md — OK
changes/ — OK
Symlinks: 11/11 valid (repaired 1)
.gitignore: fab/current present
fab/ structure verified.
```

---

## Config Behavior

Create a new `fab/config.yaml` interactively or update specific sections of an existing one. Preserves YAML comments and formatting through targeted string replacement. Validates structural correctness after each edit.

**Context loading**: This behavior loads `fab/config.yaml` (the file being edited). It does NOT load `fab/constitution.md`, `fab/memory/index.md`, or `fab/specs/index.md`.

### Config Arguments

- **`[section]`** *(optional)* — name of the config section to edit directly, skipping the menu. Valid values: `project`, `context`, `source_paths`, `stages`, `rules`, `checklist`, `git`, `naming`.

If no section argument is provided, the full section menu is displayed.

### Config Pre-flight

#### Update Mode

1. Check that `fab/config.yaml` exists
   - If missing and invoked directly (not from bootstrap): **STOP**: `fab/config.yaml not found. Run /fab-init to create it.`
2. Read the current `fab/config.yaml` content

#### Create Mode (invoked from bootstrap)

When invoked during bootstrap and `fab/config.yaml` does not exist, operate in create mode (see below).

### Config Mode Selection

| `fab/config.yaml` exists? | Argument? | Mode |
|---------------------------|-----------|------|
| No | Any | **Create mode** — generate new config |
| Yes | None | **Update mode** — show section menu |
| Yes | Valid section | **Update mode** — edit that section directly |
| Yes | Invalid section | **Error** — show valid section names |

### Config Create Mode

When `fab/config.yaml` does not exist (typically invoked from bootstrap):

1. Read the project's README, package.json, or other root-level files to gather context
2. Ask the user:
   - **Project name** — short identifier (e.g., `my-app`)
   - **Description** — one-line summary
   - **Tech stack and conventions** — languages, frameworks, API style, testing approach
   - **Source paths** — which directories contain implementation code (e.g., `src/`, `lib/`)
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
  - id: brief
    generates: brief.md
    required: true
  - id: spec
    generates: spec.md
    requires: [brief]
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
9. Done

Which section to update? (1-9)
```

2. Process the user's selection → go to **Edit Section Flow**
3. After editing, return to the menu: **"Update another section? (1-9 or 'done')"**
4. Loop until the user selects "Done"

### Config Update Mode — Argument Flow

When invoked with a section argument (e.g., `/fab-init config context`):

1. Validate the argument against the list of valid sections
2. If invalid, output: `Unknown section '{arg}'. Valid sections: project, context, source_paths, stages, rules, checklist, git, naming`
3. If valid, go directly to **Edit Section Flow** for that section
4. After editing, ask: **"Update another section?"** — if yes, show the menu; if no, exit

### Config Edit Section Flow

For the selected section:

1. **Display current value**: Show the current content of that section from `fab/config.yaml`
2. **Accept new value**: Ask the user for the updated content. For simple values (project name, branch prefix), accept inline. For multi-line values (context, rules), accept a block.
3. **Apply via string replacement**: Locate the section in the file and replace it using targeted string matching. Do NOT parse-and-rewrite the entire YAML — this preserves comments and formatting in other sections.
4. **Validate**: After the edit, validate the resulting file:
   - Parse as YAML — must be valid
   - Required fields present: `project.name`, `project.description`, `stages`
   - Stage `requires` references point to existing stage IDs
5. **If validation passes**: Confirm the edit: `Updated {section}.`
6. **If validation fails**:
   - Report: `Validation failed: {error details}`
   - Offer: `Revert this change? (yes/no)`
   - If yes, restore the previous content
   - If no, keep the invalid content (user takes responsibility)

### Config No-Op Handling

If the user selects "Done" without making any changes, output:

```
No changes made. config.yaml unchanged.
```

### Config Output

#### Create Mode

```
Created fab/config.yaml

Next: /fab-new <description>
```

#### Update Mode — Changes Applied

```
Updated context.
Updated source_paths.

2 sections updated in fab/config.yaml.

Next: /fab-init validate (verify structure)
```

#### Update Mode — No Changes

```
No changes made. config.yaml unchanged.
```

#### Validation Failure

```
Validation failed: Stage 'spec' requires non-existent stage 'planning'

Revert this change? (yes/no)
```

### Config Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing (update mode, direct invocation) | Abort: "fab/config.yaml not found. Run /fab-init to create it." |
| Invalid section argument | Output valid section names, do not proceed |
| YAML parse failure after edit | Report error, offer revert |
| Missing required field after edit | Report which field, offer revert |
| Broken stage reference after edit | Report which stage and reference, offer revert |
| String replacement target not found | Warn: "Could not locate {section} in config.yaml — file may have been manually reformatted. Attempting full rewrite for this section." Fall back to inserting the section. |

---

## Constitution Behavior

Create a new project constitution or amend an existing one. Manages the governance lifecycle of `fab/constitution.md` with semantic versioning, structural preservation, and audit trail output.

**Context loading**: This behavior loads `fab/config.yaml` (required for project context) and `fab/constitution.md` (if it exists). It does NOT load `fab/memory/index.md` or `fab/specs/index.md`.

### Constitution Pre-flight

1. Check that `fab/config.yaml` exists
   - If missing and invoked directly (not from bootstrap): **STOP**: `fab/config.yaml not found. Run /fab-init first.`
2. Read `fab/config.yaml` for project context
3. Check whether `fab/constitution.md` exists — this determines the mode

### Constitution Mode Selection

| `fab/constitution.md` exists? | Mode |
|-------------------------------|------|
| No | **Create mode** — generate a new constitution |
| Yes | **Update mode** — guided amendment |

### Constitution Create Mode

When `fab/constitution.md` does not exist:

1. Read project context from `fab/config.yaml` (project name, description, context/tech stack)
2. Examine additional project context: README, existing documentation, codebase structure, and conversation history
3. Generate `fab/constitution.md` with this structure:

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

4. Output confirmation:

```
Created fab/constitution.md (version 1.0.0) with {N} principles.
```

### Constitution Update Mode

When `fab/constitution.md` already exists:

1. Read and display the current constitution content
2. Read the current version from the Governance section
3. Present the amendment menu:

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

4. Process the user's selection:
   - **Add principle**: Ask for the principle name and description. Insert at the next Roman numeral position. Record bump level: MINOR.
   - **Modify principle**: Show numbered list of principles, ask which to modify, accept new text. Record bump level: MAJOR if meaning changes, PATCH if clarification only. Ask the user which: "Is this a (1) fundamental change or (2) wording clarification?"
   - **Remove principle**: Show numbered list, ask which to remove. Re-number remaining principles with sequential Roman numerals. Record bump level: MAJOR.
   - **Add/modify constraint**: Show the Additional Constraints section, accept edits. Record bump level: MINOR for additions, PATCH for modifications.
   - **Update governance**: Allow editing the Ratified date or other governance metadata. Record bump level: PATCH.
   - **Done**: Proceed to version bump (if any changes were made).

5. After each action, ask: **"Any other changes? (yes/no)"**
   - If yes, return to the amendment menu (step 3)
   - If no, proceed to step 6

6. **Apply version bump**: Determine the final version based on the highest-severity bump encountered across all amendments in this session:
   - **MAJOR** takes precedence over MINOR and PATCH
   - **MINOR** takes precedence over PATCH
   - Update the Governance section: increment the appropriate version component, update "Last Amended" to today's date

7. **Structural preservation**: After all edits:
   - Verify heading hierarchy is intact (h1 > h2 > h3)
   - Verify Roman numeral numbering is sequential (I, II, III, ...)
   - Verify Governance section format is correct
   - Re-number principles if any were added or removed

8. Write the updated `fab/constitution.md`

9. **Output amendment summary**:

```
Amended fab/constitution.md:
- Added: III. {Principle Name} (MINOR)
- Removed: V. {Old Principle Name} (MAJOR)
- Modified: I. {Principle Name} (PATCH — clarification)

Version: {old} → {new}
```

### Constitution No-Op Handling

If the user selects "Done" without making any changes (or answers "no" to the first "Any other changes?" prompt after selecting "Done" from the menu), output:

```
No changes made. Constitution unchanged at version {X.Y.Z}.
```

Do NOT modify the file or bump the version.

### Constitution Output

#### Create Mode

```
Created fab/constitution.md (version 1.0.0) with {N} principles.

Next: /fab-init validate (verify structure) or /fab-new <description>
```

#### Update Mode — Changes Applied

```
Amended fab/constitution.md:
- Added: VII. {Principle Name} (MINOR)

Version: 1.2.0 → 1.3.0

Next: /fab-init validate (verify structure)
```

#### Update Mode — No Changes

```
No changes made. Constitution unchanged at version 1.2.0.
```

### Constitution Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing (direct invocation) | Abort: "fab/config.yaml not found. Run /fab-init first." |
| `fab/constitution.md` malformed (update mode) | Warn: "Constitution structure appears non-standard. Proceeding with best-effort parsing." |
| Governance section missing version | Warn: "No version found in Governance section. Starting from 1.0.0." |
| Roman numeral parsing fails | Warn and proceed with sequential numbering from I |

---

## Validate Behavior

Validate the structural correctness of `fab/config.yaml` and `fab/constitution.md`. Reports issues with actionable fix suggestions. Useful after manual edits, before commits, or as a health check.

**Context loading**: This behavior reads `fab/config.yaml` and `fab/constitution.md` for validation. It does NOT load `fab/memory/index.md` or `fab/specs/index.md`.

### Validate Pre-flight

No preflight script needed — this behavior validates the files that preflight would normally check, so it must handle missing files gracefully.

### Validate Step 1: Discover Files

Check for the existence of both files:
- `fab/config.yaml`
- `fab/constitution.md`

Track which files exist. Missing files are reported but do not block validation of other files.

### Validate Step 2: Validate `config.yaml`

If `fab/config.yaml` exists, run all 8 structural checks in order:

| # | Check | Pass criteria | Failure message | Fix suggestion |
|---|-------|---------------|-----------------|----------------|
| 1 | YAML parseable | File parses as valid YAML | "FAIL: config.yaml is not valid YAML: {parse error}" | "Fix the YAML syntax error at the indicated location" |
| 2 | Required top-level keys | `project`, `context`, `stages`, `source_paths` all present | "FAIL: Missing required key '{key}'" | "Add `{key}:` as a top-level section" |
| 3 | `project.name` non-empty | String, length > 0 | "FAIL: project.name is missing or empty" | "Add `name: \"your-project\"` under the `project:` section" |
| 4 | `project.description` non-empty | String, length > 0 | "FAIL: project.description is missing or empty" | "Add `description: \"...\"` under the `project:` section" |
| 5 | `stages` non-empty list | Array with at least 1 entry | "FAIL: stages list is empty" | "Add at least the default stages (brief, spec, tasks, apply, review, hydrate)" |
| 6 | Stage `id` fields present | Every stage entry has an `id` string | "FAIL: Stage at index {N} is missing `id` field" | "Add `id: {suggested_id}` to the stage entry" |
| 7 | Stage `requires` valid | Every `requires` entry references an existing stage ID | "FAIL: Stage '{id}' requires non-existent stage '{ref}'" | "Check the `requires` list — valid stage IDs are: {list}" |
| 8 | No circular dependencies | No cycles in the stage dependency graph | "FAIL: Circular dependency detected: {cycle path}" | "Remove one of the `requires` entries to break the cycle" |

**Additional check (derived from check 6):**
- Stage IDs are unique: "FAIL: Duplicate stage ID '{id}'" / "Rename one of the duplicate stages"

If check 1 fails (YAML not parseable), **skip checks 2-8** — they depend on parsed content.

If `fab/config.yaml` does not exist:
```
config.yaml: not found — run /fab-init or /fab-init config to create it
```

### Validate Step 3: Validate `constitution.md`

If `fab/constitution.md` exists, run all 6 structural checks:

| # | Check | Pass criteria | Failure message | Fix suggestion |
|---|-------|---------------|-----------------|----------------|
| 1 | Non-empty | File has content (not just whitespace) | "FAIL: constitution.md is empty" | "Run /fab-init constitution to generate content" |
| 2 | Level-1 heading | Contains `# ... Constitution` (case-insensitive match on "Constitution") | "FAIL: Missing level-1 heading with 'Constitution'" | "Add `# {Project Name} Constitution` as the first heading" |
| 3 | Core Principles section | Contains `## Core Principles` heading | "FAIL: Missing `## Core Principles` section" | "Add a `## Core Principles` section with at least one principle" |
| 4 | Roman numeral headings | At least one `### I.` or `### II.` etc. under Core Principles | "FAIL: No Roman numeral principle headings found (expected `### I.`, `### II.`, etc.)" | "Number each principle with Roman numerals: `### I. {Name}`, `### II. {Name}`, etc." |
| 5 | Governance section | Contains `## Governance` heading | "FAIL: Missing `## Governance` section" | "Add a Governance section with version, ratified date, and last amended date" |
| 6 | Version format | Governance section contains a version matching `MAJOR.MINOR.PATCH` pattern (e.g., `1.0.0`, `2.3.1`) | "FAIL: No version in MAJOR.MINOR.PATCH format found in Governance section" | "Add `**Version**: 1.0.0` to the Governance section" |

If `fab/constitution.md` does not exist:
```
constitution.md: not found — run /fab-init or /fab-init constitution to create it
```

### Validate Step 4: Combined Report

Present results for both files:

```
config.yaml:      {passed}/{total} checks passed {✓ or ✗}
constitution.md:  {passed}/{total} checks passed {✓ or ✗}
```

If all checks pass:
```
All validation checks passed.
```

If any checks fail:
```
{N} issue(s) found. Fix the issues above and re-run /fab-init validate.
```

### Validate Output

#### All Checks Pass

```
config.yaml checks:
  ✓ YAML parseable
  ✓ Required top-level keys present
  ✓ project.name non-empty
  ✓ project.description non-empty
  ✓ stages non-empty list
  ✓ Stage id fields present
  ✓ Stage requires references valid
  ✓ No circular dependencies

constitution.md checks:
  ✓ Non-empty
  ✓ Level-1 heading with "Constitution"
  ✓ Core Principles section
  ✓ Roman numeral headings
  ✓ Governance section
  ✓ Version in MAJOR.MINOR.PATCH format

config.yaml:      8/8 checks passed ✓
constitution.md:  6/6 checks passed ✓

All validation checks passed.
```

#### With Failures

```
config.yaml checks:
  ✓ YAML parseable
  ✓ Required top-level keys present
  ✗ project.name is missing or empty
    → Add `name: "your-project"` under the `project:` section
  ✓ project.description non-empty
  ✓ stages non-empty list
  ✓ Stage id fields present
  ✓ Stage requires references valid
  ✓ No circular dependencies

constitution.md checks:
  ✓ Non-empty
  ✓ Level-1 heading with "Constitution"
  ✓ Core Principles section
  ✓ Roman numeral headings
  ✗ Missing ## Governance section
    → Add a Governance section with version, ratified date, and last amended date
  — Version format (skipped — no Governance section)

config.yaml:      7/8 checks passed ✗
constitution.md:  4/6 checks passed ✗

2 issue(s) found. Fix the issues above and re-run /fab-init validate.
```

#### One File Missing

```
config.yaml checks:
  ✓ YAML parseable
  ...

config.yaml:      8/8 checks passed ✓
constitution.md:  not found — run /fab-init or /fab-init constitution to create it

1 issue(s) found. Fix the issues above and re-run /fab-init validate.
```

#### Both Files Missing

```
config.yaml:      not found — run /fab-init or /fab-init config to create it
constitution.md:  not found — run /fab-init or /fab-init constitution to create it

No files to validate. Run /fab-init to bootstrap the project.
```

### Validate Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing | Report as missing, suggest creation, continue to validate constitution |
| `fab/constitution.md` missing | Report as missing, suggest creation, continue to validate config |
| Both files missing | Report both, suggest `/fab-init` |
| YAML parse failure | Report parse error, skip remaining config checks |
| Governance section exists but version not found | Fail check 6, suggest adding version |
| Constitution check depends on failed earlier check | Mark as "skipped" with reason |

---

## Idempotency Guarantee

This skill is safe to run any number of times:

- **Config and constitution**: Created once, never overwritten on re-run (bootstrap path)
- **Memory index**: Created once, never touched on re-run
- **Specs index**: Created once, never touched on re-run
- **Changes directory**: Created once, never touched on re-run
- **Symlinks**: Verified and repaired on every run — broken symlinks are fixed, valid ones are left alone
- **`.gitignore`**: Entry is appended only if not already present
- **Config updates**: Same edit applied twice produces same result
- **Constitution amendments**: No changes = no-op; version not bumped
- **Validate**: Read-only, no modifications

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/.kit/` missing | Abort immediately with guidance message. Do NOT create any files. |
| `fab/.kit/VERSION` unreadable | Abort with: "fab/.kit/VERSION not found or unreadable — kit may be corrupted." |
| Unrecognized argument | Abort with: "Did you mean /fab-hydrate? /fab-init no longer accepts source arguments." |
| Symlink target missing | Report which skill file is missing in `fab/.kit/skills/` — do NOT create a broken symlink |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — project-level tool, not tied to the change pipeline |
| Idempotent? | **Yes** — all paths are safe to re-run |
| Modifies `fab/config.yaml`? | **Yes** — bootstrap creates, config subcommand creates or updates |
| Modifies `fab/constitution.md`? | **Yes** — bootstrap creates, constitution subcommand creates or updates |
| Modifies `.status.yaml`? | **No** |
| Modifies source code? | **No** |

---

## Next Steps Reference

After bootstrap: `Next: /fab-new <description> or /fab-hydrate <sources>`

After config create: `Next: /fab-new <description>`

After config update: `Next: /fab-init validate (verify structure)`

After constitution create: `Next: /fab-init validate (verify structure) or /fab-new <description>`

After constitution update: `Next: /fab-init validate (verify structure)`

After validate (all pass): `Next: /fab-new <description> or /fab-init config (update) or /fab-init constitution (amend)`

After validate (failures): `Next: Fix the reported issues and re-run /fab-init validate`
