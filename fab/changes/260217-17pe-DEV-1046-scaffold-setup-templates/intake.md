# Intake: Scaffold Setup Templates

**Change**: 260217-17pe-DEV-1046-scaffold-setup-templates
**Created**: 2026-02-17
**Status**: Draft

## Origin

> During installation investigation, noticed that `fab/config.yaml` created by `/fab-setup` uses a template hardcoded inline in the skill prose (fab-setup.md lines 192-259) rather than reading from a scaffold file. Same pattern for `fab/constitution.md` (lines 337-356). Additionally, `docs/memory/index.md` and `docs/specs/index.md` have templates duplicated in both `fab/.kit/scaffold/` files AND inline in fab-setup.md (lines 72-104), creating two sources of truth that can diverge.

One-shot change from direct investigation of the setup flow.

## Why

1. **Drift risk**: The config.yaml "template" lives only as inline YAML in a skill's markdown prose. When the config schema evolves (new sections, renamed keys, changed defaults), the inline template can go stale while the actual schema expectations in other skills advance. This already happened — the current config.yaml in this repo differs from what the inline template would produce.

2. **Duplication**: `memory-index.md` and `specs-index.md` each have two copies of their template content — one in `fab/.kit/scaffold/` (used by `fab-sync.sh`) and one inline in `fab-setup.md`. Changes to one won't propagate to the other.

3. **Principle alignment**: Constitution Principle V (Portability) says project-specific config belongs in `fab/config.yaml`, not in `.kit/`. But the corollary is that `.kit/` should own its own templates as files, not as prose embedded in skills. This also aligns with Principle I (Pure Prompt Play) — keeping templates as discrete files rather than buried in skill prose makes them inspectable and diffable.

## What Changes

### 1. New scaffold file: `fab/.kit/scaffold/config.yaml`

Create a default config.yaml template in the scaffold directory. This is the canonical starting point that `/fab-setup` reads and customizes interactively. Contains all sections with placeholder values and full inline comments:

```yaml
# fab/config.yaml
#
# Project configuration for the Fab workflow...
# (same header comments as current config.yaml)

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
  # ... (full default stages)

source_paths:
  - {SOURCE_PATHS}

checklist:
  extra_categories: []

rules:
  spec:
    - Use GIVEN/WHEN/THEN for scenarios
    - "Mark ambiguities with [NEEDS CLARIFICATION]"

# code_quality: ...  (commented-out section)
```

### 2. New scaffold file: `fab/.kit/scaffold/constitution.md`

Create a skeleton constitution template:

```markdown
# {Project Name} Constitution

## Core Principles

### I. {Principle Name}
{Description using MUST/SHALL/SHOULD keywords. Include rationale.}

## Additional Constraints

## Governance

**Version**: 1.0.0 | **Ratified**: {DATE} | **Last Amended**: {DATE}
```

### 3. Update `fab/.kit/skills/fab-setup.md` — Config Create Mode

Replace the inline YAML template (lines ~192-259) with an instruction to:
1. Read `fab/.kit/scaffold/config.yaml` as the starting template
2. Substitute placeholders with user-provided values
3. Write the result to `fab/config.yaml`

### 4. Update `fab/.kit/skills/fab-setup.md` — Constitution Create Mode

Replace the inline markdown template (lines ~337-356) with an instruction to:
1. Read `fab/.kit/scaffold/constitution.md` as the starting skeleton
2. Generate principles based on project context, filling in the skeleton
3. Write the result to `fab/constitution.md`

### 5. Update `fab/.kit/skills/fab-setup.md` — Memory/Specs Index References

Replace the inline templates for `docs/memory/index.md` (lines ~72-80) and `docs/specs/index.md` (lines ~86-104) with instructions to read from the existing scaffold files:
- "Copy from `fab/.kit/scaffold/memory-index.md`" (instead of inline template)
- "Copy from `fab/.kit/scaffold/specs-index.md`" (instead of inline template)

This aligns the skill prose with what `fab-sync.sh` already does — both now point to the same source file.

## Affected Memory

- `fab-workflow/setup`: (modify) Document that config.yaml and constitution.md templates now live in scaffold directory
- `fab-workflow/kit-architecture`: (modify) Update scaffold directory inventory to include new files

## Impact

- **`fab/.kit/scaffold/`** — 2 new files (config.yaml, constitution.md)
- **`fab/.kit/skills/fab-setup.md`** — 4 sections modified (config create, constitution create, memory index, specs index)
- **No behavioral change** — the generated output for users remains identical; only the source of truth for templates changes
- **No migration needed** — existing projects already have their config.yaml and constitution.md; this only affects new project setup

## Open Questions

- None — scope is well-defined from the investigation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scaffold files use placeholder syntax like `{PROJECT_NAME}` | Consistent with existing template conventions in `fab/.kit/templates/` | S:90 R:90 A:95 D:90 |
| 2 | Certain | fab-sync.sh is not modified | It already copies from scaffold files; no change needed there | S:95 R:95 A:95 D:95 |
| 3 | Confident | Constitution scaffold is a minimal skeleton, not a full example | fab-setup generates principles dynamically from project context — the scaffold just provides structure | S:80 R:85 A:80 D:70 |
| 4 | Certain | No migration file needed | This change only affects new project setup, not existing projects | S:90 R:95 A:90 D:95 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
