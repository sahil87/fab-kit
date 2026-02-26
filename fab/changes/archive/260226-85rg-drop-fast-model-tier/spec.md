# Spec: Drop Fast Model Tier

**Change**: 260226-85rg-drop-fast-model-tier
**Created**: 2026-02-26
**Affected memory**: `docs/memory/fab-workflow/model-tiers.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Introducing a replacement mechanism (e.g., per-skill model hints) — the whole point is simplification
- Changing behavior of any skill beyond frontmatter removal — all skills keep their existing logic

## Skills: Remove Fast Tier Frontmatter

### Requirement: All skills SHALL omit `model_tier` from frontmatter

No skill file in `fab/.kit/skills/` SHALL contain a `model_tier:` field in its YAML frontmatter. With no fast tier, all skills are implicitly capable (the platform default model).

The following five skills currently have `model_tier: fast` and MUST have that line removed:

- `fab/.kit/skills/fab-switch.md`
- `fab/.kit/skills/fab-help.md`
- `fab/.kit/skills/fab-status.md`
- `fab/.kit/skills/fab-setup.md`
- `fab/.kit/skills/git-branch.md`

#### Scenario: Skill frontmatter after removal

- **GIVEN** `fab/.kit/skills/fab-switch.md` exists with frontmatter containing `model_tier: fast`
- **WHEN** the change is applied
- **THEN** the frontmatter contains only `name` and `description` fields
- **AND** no `model_tier` field appears anywhere in the file

#### Scenario: Capable skills are unchanged

- **GIVEN** `fab/.kit/skills/fab-continue.md` has no `model_tier` field
- **WHEN** the change is applied
- **THEN** the file is unmodified

## Config: Remove `model_tiers` Section

### Requirement: `config.yaml` SHALL NOT contain a `model_tiers` section

The `model_tiers:` block in `fab/project/config.yaml` SHALL be removed entirely. With no fast-tier consumers, the configuration is dead.

```yaml
# BEFORE (remove this entire block):
model_tiers:
  fast:
    claude: haiku
  capable:
    claude: null
```

#### Scenario: Config after removal

- **GIVEN** `fab/project/config.yaml` contains a `model_tiers:` section
- **WHEN** the change is applied
- **THEN** the `model_tiers:` section (including all nested keys) is absent from the file
- **AND** all other config sections (`project:`, `source_paths:`, `checklist:`, `stage_directives:`) are preserved

### Requirement: Scaffold config SHALL NOT contain `model_tiers`

`fab/.kit/scaffold/fab/project/config.yaml` SHALL have its `model_tiers:` block removed so new projects don't ship with unused configuration.

#### Scenario: Scaffold config for new projects

- **GIVEN** a new project is created via `/fab-setup`
- **WHEN** `fab/.kit/scaffold/fab/project/config.yaml` is used as the template
- **THEN** the generated `config.yaml` does not contain `model_tiers:`

## Sync Script: Remove Tier Resolution Logic

### Requirement: `2-sync-workspace.sh` SHALL NOT classify skills by model tier

Section 3b ("Classify skills by model tier") in `fab/.kit/sync/2-sync-workspace.sh` SHALL be removed entirely. This includes the `fast_skills` array, the frontmatter parsing loop, and the `model_tier` validation logic (lines 248-274).

#### Scenario: No fast_skills classification

- **GIVEN** `2-sync-workspace.sh` is executed
- **WHEN** skills are enumerated
- **THEN** no skill is classified into a `fast_skills` array
- **AND** no `model_tier` field is read from any skill frontmatter during sync

### Requirement: `2-sync-workspace.sh` SHALL NOT perform model tier substitution

Section 3c ("Resolve fast-tier model for Claude Code") SHALL be removed (lines 413-423). The Claude Code deployment call SHALL use plain copy mode without a sed expression.

```bash
# BEFORE:
if [ -n "$claude_fast_model" ]; then
  sync_agent_skills "Claude Code" "$repo_root/.claude/skills" "directory" "copy" \
    "s/^model_tier: .*/model: $claude_fast_model/"
else
  sync_agent_skills "Claude Code" "$repo_root/.claude/skills" "directory" "copy"
fi

# AFTER:
sync_agent_skills "Claude Code" "$repo_root/.claude/skills" "directory" "copy"
```

#### Scenario: Claude Code skills deployed as plain copies

- **GIVEN** `2-sync-workspace.sh` runs with no fast-tier skills
- **WHEN** skills are deployed to `.claude/skills/`
- **THEN** each skill is a byte-accurate copy of the canonical file
- **AND** no sed substitution is applied during copy

### Requirement: `yaml_value` helper MAY be removed

The `yaml_value` shell function (lines 38-52) is only used for reading `model_tiers` from config. It SHOULD be removed as dead code. However, if other callers are later added, it MAY be retained.
<!-- assumed: yaml_value is only used for model tier resolution — grep confirms no other callers in sync script -->

#### Scenario: yaml_value removal

- **GIVEN** `yaml_value` has no callers after tier resolution removal
- **WHEN** the change is applied
- **THEN** the `yaml_value` function is removed from the script

## Tests: Update Sync Workspace Tests

### Requirement: Model-tier test cases SHALL be removed or updated

The following tests in `src/lib/sync-workspace/test.bats` reference model-tier behavior and SHALL be removed:

- `fast-tier skill copy has model: instead of model_tier:` (line 299)
- `uses haiku fallback when config.yaml has no model_tiers` (line 380)
- `reads model_tiers from config.yaml when present` (line 389)

The test fixture `setup()` SHALL be updated to remove the `model_tier: fast` frontmatter from the `fab-status.md` test fixture (lines 70-78).

The test `capable-tier skill copy preserves content without model override` (line 308) SHALL be updated to verify that ALL skills are deployed as plain copies without model override (since the fast/capable distinction no longer exists).

#### Scenario: Test suite passes after changes

- **GIVEN** the model-tier logic has been removed from the sync script
- **WHEN** `bats src/lib/sync-workspace/test.bats` runs
- **THEN** all tests pass
- **AND** no test references `model_tier` or `model_tiers`

## Skill Optimize: Update Frontmatter Reference

### Requirement: `internal-skill-optimize.md` SHALL NOT reference `model_tier`

The line "Preserve frontmatter exactly — `name`, `description`, `model_tier` fields are untouched" SHALL be updated to remove the `model_tier` reference since the field no longer exists.

#### Scenario: Updated preservation list

- **GIVEN** `internal-skill-optimize.md` lists frontmatter fields to preserve
- **WHEN** the change is applied
- **THEN** the preservation list reads `` `name`, `description` `` (without `model_tier`)

## Deprecated Requirements

### Model Tier Classification
**Reason**: The fast tier caused mid-conversation context compaction when Claude Code switched from a high-context model to Haiku. All skills now run on the session's default model.
**Migration**: Remove `model_tier: fast` from skill frontmatter. No replacement needed — omitting the field means "use platform default."

### Model Tier Configuration
**Reason**: No fast-tier consumers remain; `model_tiers:` in config is dead config.
**Migration**: Remove `model_tiers:` section from `config.yaml` and scaffold config.

### Model Tier Deployment Substitution
**Reason**: No `model_tier:` → `model:` templating needed when all skills use the default.
**Migration**: Simplify Claude Code deployment to plain copy.

## Design Decisions

### 1. Remove entire fast tier, not just fab-switch
- *Why*: The compaction problem affects any fast-tier skill invoked mid-conversation. Fixing only `fab-switch` leaves the same bug in `fab-help`, `fab-status`, `fab-setup`, and `git-branch`. The cost savings from Haiku for these lightweight skills are negligible.
- *Rejected*: Fix only `fab-switch` — leaves identical bugs in other skills.

### 2. Remove infrastructure, not just frontmatter
- *Why*: With zero consumers of the fast tier, the config section, sync script logic, scaffold template, and tests are all dead code. Leaving dead infrastructure creates maintenance burden and confuses future contributors.
- *Rejected*: Keep infrastructure "in case we need it later" — violates YAGNI and the project's simplicity principle.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove fast tier from all 5 skills | Confirmed from intake #1 — problem applies equally to all fast-tier skills; git-pr precedent validates approach | S:80 R:90 A:85 D:90 |
| 2 | Certain | Remove `model_tiers` config section entirely | Confirmed from intake #2 — zero consumers, dead config | S:75 R:95 A:90 D:95 |
| 3 | Certain | Remove tier resolution from sync script (sections 3b, 3c) | Confirmed from intake #3 — dead code after frontmatter removal | S:80 R:90 A:90 D:95 |
| 4 | Certain | Remove scaffold `model_tiers` too | Upgraded from intake Confident #4 — scaffold config verified at `fab/.kit/scaffold/fab/project/config.yaml` | S:75 R:95 A:90 D:95 |
| 5 | Confident | Root cause is context window mismatch triggering compaction | Confirmed from intake #5 — matches git-pr changelog entry verbatim: "haiku's smaller context window caused limit hits" | S:70 R:85 A:70 D:80 |
| 6 | Certain | Remove `yaml_value` helper from sync script | Only caller is model-tier resolution; grep confirms no other usage in the file | S:80 R:95 A:95 D:95 |
| 7 | Certain | Remove model-tier tests from test.bats | Tests validate removed behavior; keeping them would cause failures | S:85 R:90 A:90 D:95 |
| 8 | Certain | Update `internal-skill-optimize.md` frontmatter reference | Field no longer exists; stale reference would mislead the optimization skill | S:75 R:95 A:90 D:95 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
