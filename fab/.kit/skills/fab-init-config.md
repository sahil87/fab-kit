---
name: fab-init-config
description: "Create or update config.yaml interactively. Preserves comments."
model_tier: fast
---

# /fab-init-config [section]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.
> **Context loading**: This skill loads `fab/config.yaml` (the file being edited). It does NOT load `fab/constitution.md`, `fab/docs/index.md`, or `fab/design/index.md`.

---

## Purpose

Create a new `fab/config.yaml` interactively or update specific sections of an existing one. Preserves YAML comments and formatting through targeted string replacement. Validates structural correctness after each edit.

---

## Arguments

- **`[section]`** *(optional)* — name of the config section to edit directly, skipping the menu. Valid values: `project`, `context`, `source_paths`, `stages`, `rules`, `checklist`, `git`, `naming`.

If no argument is provided, the full section menu is displayed.

---

## Pre-flight Check

### Update Mode

1. Check that `fab/config.yaml` exists
   - If missing, **STOP**: `fab/config.yaml not found. Run /fab-init to create it.`
2. Read the current `fab/config.yaml` content

### Create Mode (invoked by `/fab-init` delegation)

When invoked during `/fab-init` and `fab/config.yaml` does not exist, operate in create mode (see Create Mode section below).

---

## Behavior

### Mode Selection

| `fab/config.yaml` exists? | Argument? | Mode |
|---------------------------|-----------|------|
| No | Any | **Create mode** — generate new config |
| Yes | None | **Update mode** — show section menu |
| Yes | Valid section | **Update mode** — edit that section directly |
| Yes | Invalid section | **Error** — show valid section names |

### Create Mode

When `fab/config.yaml` does not exist (typically invoked via `/fab-init` delegation):

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
  - id: archive
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

### Update Mode — Menu Flow

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

### Update Mode — Argument Flow

When invoked with a section argument (e.g., `/fab-init-config context`):

1. Validate the argument against the list of valid sections
2. If invalid, output: `Unknown section '{arg}'. Valid sections: project, context, source_paths, stages, rules, checklist, git, naming`
3. If valid, go directly to **Edit Section Flow** for that section
4. After editing, ask: **"Update another section?"** — if yes, show the menu; if no, exit

### Edit Section Flow

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

### No-Op Handling

If the user selects "Done" without making any changes, output:

```
No changes made. config.yaml unchanged.
```

---

## Output

### Create Mode

```
Created fab/config.yaml

Next: /fab-init-constitution or /fab-new <description>
```

### Update Mode — Changes Applied

```
Updated context.
Updated source_paths.

2 sections updated in fab/config.yaml.

Next: /fab-init-validate (verify structure)
```

### Update Mode — No Changes

```
No changes made. config.yaml unchanged.
```

### Validation Failure

```
Validation failed: Stage 'spec' requires non-existent stage 'planning'

Revert this change? (yes/no)
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing (update mode) | Abort: "fab/config.yaml not found. Run /fab-init to create it." |
| Invalid section argument | Output valid section names, do not proceed |
| YAML parse failure after edit | Report error, offer revert |
| Missing required field after edit | Report which field, offer revert |
| Broken stage reference after edit | Report which stage and reference, offer revert |
| String replacement target not found | Warn: "Could not locate {section} in config.yaml — file may have been manually reformatted. Attempting full rewrite for this section." Fall back to inserting the section. |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — project-level tool, not tied to the change pipeline |
| Idempotent? | **Yes** — no changes = no-op; same edit applied twice produces same result |
| Modifies `fab/config.yaml`? | **Yes** — creates or updates |
| Modifies `.status.yaml`? | **No** |
| Modifies source code? | **No** |
| Requires config? | **No** — this skill *creates* the config (create mode) or *modifies* it (update mode) |

---

## Next Steps Reference

After create mode: `Next: /fab-init-constitution or /fab-new <description>`

After update mode: `Next: /fab-init-validate (verify structure)`
