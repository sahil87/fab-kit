# Intake: Restructure config.yaml

**Change**: 260218-bb93-restructure-config-yaml
**Created**: 2026-02-18
**Status**: Draft

## Origin

> Restructure config.yaml: 1) Extract code_quality section to fab/code-quality.md, 2) Extract context: section to fab/context.md, 3) Delete stages: from config.yaml (dead code — workflow.yaml is source of truth), 4) Merge model-tiers.yaml into config.yaml scaffold as model_tiers: section with sensible defaults, delete standalone model-tiers.yaml, update fab-sync.sh to read from config.yaml only. Update all skill references, scaffold template, and fab-setup.md accordingly.

Discussion-driven change. User identified that config.yaml has accumulated sections that are either dead code, prose masquerading as configuration, or duplicated across files. Each extraction/deletion was discussed and agreed upon before creating this change.

## Why

`fab/config.yaml` has grown beyond its purpose as a configuration file:

1. **`context:`** is free-form prose (tech stack, conventions) jammed into YAML. It's awkward to author and review in YAML multi-line strings. Moving to markdown lets users write naturally and keeps config.yaml focused on structured settings.

2. **`code_quality:`** (commented out in scaffold, active in some projects) contains principles, anti-patterns, and strategy — essentially prose guidelines. Same problem as context.

3. **`stages:`** is dead code. No script or skill reads it from config.yaml. `fab/.kit/schemas/workflow.yaml` is the authoritative source, and `stageman.sh` reads from there exclusively. The config.yaml copy is a strictly-less-detailed subset that adds confusion about which is canonical.

4. **`model-tiers.yaml`** is a separate 21-line file with a kit-default → project-override pattern via `fab-sync.sh`. The override pattern adds complexity for little benefit — users should just configure model tiers directly in config.yaml. Collapsing to a single source simplifies the mental model.

## What Changes

### 1. Extract `context:` to `fab/context.md`

- Create new file `fab/context.md` as a companion to `fab/constitution.md`
- Skills already load config.yaml + constitution.md as a pair; context.md becomes a third always-load file
- Remove `context:` key from `fab/config.yaml` and `fab/.kit/scaffold/config.yaml`
- Add `fab/context.md` pointer comment in config.yaml header alongside existing constitution.md reference
- Add `fab/.kit/scaffold/context.md` template for `/fab-setup` bootstrap
- Update `_context.md` Layer 1 to include `fab/context.md` in the always-load list
- Update all skill references that mention loading context from config.yaml

### 2. Extract `code_quality:` to `fab/code-quality.md`

- Create new file pattern `fab/code-quality.md` as another companion file
- Move `code_quality.principles`, `code_quality.anti_patterns`, and `code_quality.test_strategy` content into markdown format
- Remove `code_quality:` section from `fab/config.yaml` and scaffold
- Add pointer comment in config.yaml header
- Add `fab/.kit/scaffold/code-quality.md` template
- Update `_context.md` to include `fab/code-quality.md` in the always-load list (optional file — no error if missing)
- Update skills that reference `code_quality` from config.yaml:
  - `fab-continue.md` (apply stage, review stage)
  - `_generation.md` (checklist generation)
  - `fab-setup.md` (config section list)

### 3. Delete `stages:` from config.yaml

- Remove the entire `stages:` block from `fab/config.yaml` and `fab/.kit/scaffold/config.yaml`
- Remove `stages` from the valid sections list in `fab-setup.md` line 19
- Remove any "Consumed by" comments referencing stages in config.yaml
- No replacement needed — `fab/.kit/schemas/workflow.yaml` already serves this role completely
- Verify no other consumers exist (confirmed: `stageman.sh` reads from `WORKFLOW_SCHEMA` only)

### 4. Merge `model-tiers.yaml` into config.yaml

- Add `model_tiers:` section to `fab/config.yaml` and scaffold with the current defaults:
  ```yaml
  model_tiers:
    fast:
      claude: haiku
    capable:
      claude: null    # null = use platform default
  ```
- Delete `fab/.kit/model-tiers.yaml`
- Update `fab-sync.sh`:
  - Remove the pre-flight check for `model-tiers.yaml` existence (lines 23-26)
  - Change `yaml_value "$kit_dir/model-tiers.yaml" ...` to read from `"$fab_dir/config.yaml"` directly
  - Remove the override logic (lines 328-333) — config.yaml is now the single source
  - Add fallback: if config.yaml has no `model_tiers` section, use hardcoded defaults in the script
- Update `fab-setup.md` to include `model_tiers` in the valid config sections list
- Update memory file `docs/memory/fab-workflow/model-tiers.md` (if it references the standalone file)

### 5. Update references across skills and docs

- `_context.md`: Update Layer 1 description and file list
- `fab-setup.md`: Update valid sections list, bootstrap flow, config subcommand
- `fab-continue.md`: Update code_quality references to point to code-quality.md
- `_generation.md`: Update checklist generation to read from code-quality.md
- `fab-new.md`: Update pre-flight to check for context.md
- `fab-help.md`: Update if it references config.yaml structure
- Config.yaml header comment: Update companion files list

## Affected Memory

- `fab-workflow/configuration`: (modify) Document new file layout: config.yaml + context.md + constitution.md + code-quality.md
- `fab-workflow/model-tiers`: (modify) Update to reflect model_tiers living in config.yaml instead of standalone file
- `fab-workflow/context-loading`: (modify) Update always-load list to include context.md and code-quality.md

## Impact

- **All skills**: Context loading changes (new files to read)
- **fab-sync.sh**: Model tier resolution logic changes
- **fab-setup.md**: Bootstrap and config subcommand changes
- **scaffold/**: New templates (context.md, code-quality.md), updated config.yaml template
- **Existing projects**: Need migration path — projects with `context:` in config.yaml and standalone model-tiers.yaml need guidance (migration in fab-setup.md)

## Open Questions

- Should `/fab-setup migrations` handle the extraction automatically (move context: content to context.md, etc.), or is this a manual upgrade with docs?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fab/context.md` as the filename for extracted context | Mirrors `fab/constitution.md` naming pattern; user agreed in discussion | S:90 R:90 A:95 D:90 |
| 2 | Certain | `fab/code-quality.md` as the filename for extracted code_quality | Natural name; follows same companion-file pattern | S:85 R:85 A:90 D:85 |
| 3 | Certain | Delete stages: entirely (no move) | Confirmed zero consumers; workflow.yaml is complete superset | S:95 R:95 A:95 D:95 |
| 4 | Certain | model_tiers goes into config.yaml (not a separate section file) | User explicitly chose this; simplifies mental model | S:95 R:85 A:90 D:90 |
| 5 | Confident | context.md and code-quality.md are optional files (no error if missing) | constitution.md is required; these are supplementary. New projects get them from scaffold, but existing projects shouldn't break without them | S:70 R:90 A:80 D:75 |
| 6 | Confident | fab-sync.sh uses hardcoded defaults as fallback when config.yaml has no model_tiers | Maintains "just works" behavior for projects that don't customize tiers | S:75 R:85 A:80 D:70 |
| 7 | Tentative | Migration is handled by /fab-setup migrations rather than manual | Fits existing migration pattern, but scope unclear until we see how many projects are affected | S:60 R:70 A:55 D:50 |
<!-- assumed: migration via fab-setup — fits existing pattern but implementation scope depends on how many fields need migrating -->

7 assumptions (4 certain, 2 confident, 1 tentative, 0 unresolved). Run /fab-clarify to review.
