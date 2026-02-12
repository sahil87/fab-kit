# Spec: Provider-Agnostic Model Tiers for Fab Skills

**Change**: 260212-k8m3-skill-model-tiers
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/model-tiers.md` (new), `fab/docs/fab-workflow/kit-architecture.md`, `fab/docs/fab-workflow/templates.md`

## Non-Goals

- Runtime pipeline orchestration changes — skills invoked via `[AUTO-MODE]` continue running in the calling context, not as subagents with separate model selection
- Per-invocation model overrides — users cannot override the tier for a single skill run
- Third-party provider mappings — only Anthropic (Claude Code) mapping is implemented initially; the system is designed for extensibility but other providers are out of scope

## Fab-Workflow: Tier Naming Scheme

### Requirement: Two-Tier Classification

The tier system SHALL define exactly two tiers:

- **`fast`** — for skills that delegate to shell scripts, display formatted output, or perform simple lookups without deep reasoning or artifact generation
- **`capable`** — for skills that require multi-step reasoning, artifact generation, SRAD evaluation, code analysis, or review

A skill that omits the `model_tier` field SHALL be treated as `capable` (the default). Only skills explicitly marked `model_tier: fast` use a cheaper/faster model.

#### Scenario: Skill with no model_tier field
- **GIVEN** a skill file with frontmatter containing only `name` and `description`
- **WHEN** the skill is deployed to an agent directory
- **THEN** it SHALL use the platform's default/most-capable model (no `model:` override injected)

#### Scenario: Skill with model_tier: fast
- **GIVEN** a skill file with `model_tier: fast` in its frontmatter
- **WHEN** the skill is deployed to an agent directory
- **THEN** the deployment script SHALL translate the tier to the platform's fast model identifier

### Requirement: Tier Selection Criteria

The tier selection criteria SHALL be documented and applied consistently:

| Criterion | fast | capable |
|-----------|------|---------|
| Delegates to shell script | Yes | — |
| Generates markdown artifacts | — | Yes |
| Applies SRAD framework | — | Yes |
| Reads/modifies source code | — | Yes |
| Requires multi-step reasoning | — | Yes |
| Simple state lookup or display | Yes | — |

A skill that matches ANY `capable` criterion SHALL be classified as `capable`, regardless of other characteristics.

#### Scenario: Classifying a script-delegation skill
- **GIVEN** a skill like `fab-help` that only runs a shell script and displays output
- **WHEN** the tier criteria are applied
- **THEN** it SHALL be classified as `fast` (matches only `fast` criteria)

#### Scenario: Classifying an artifact-generating skill
- **GIVEN** a skill like `fab-continue` that generates spec.md or tasks.md
- **WHEN** the tier criteria are applied
- **THEN** it SHALL be classified as `capable` (matches artifact generation criterion)

## Fab-Workflow: Skill Classification

### Requirement: Audit and Tag All Skills

Every deployable skill in `fab/.kit/skills/` SHALL be classified and tagged. The following classification SHALL be applied:

**Fast tier** (`model_tier: fast`):
- `fab-help` — delegates to `fab-help.sh`
- `fab-status` — delegates to `fab-status.sh`
- `fab-switch` — state lookup, branch operations, no artifact generation
- `fab-init` — structural bootstrap, delegates to `fab-setup.sh`

**Capable tier** (no `model_tier` field — implicit default):
- `fab-new` — brief generation, SRAD evaluation, NLP parsing
- `fab-hydrate` — content analysis, doc generation/merging
- `fab-continue` — artifact generation (spec/tasks), SRAD evaluation
- `fab-ff` — multi-stage pipeline orchestration, artifact generation
- `fab-fff` — full pipeline orchestration with confidence gating
- `fab-clarify` — interactive/autonomous gap resolution, deep reasoning
- `fab-apply` — code implementation, test execution
- `fab-review` — multi-dimensional validation, code inspection
- `fab-archive` — documentation hydration, conflict detection
- `fab-backfill` — structural gap analysis, spec modification
- `internal-consistency-check` — cross-layer drift detection
- `internal-retrospect` — retrospective analysis, meta-reasoning

Shared partials (`_context.md`, `_generation.md`) are NOT deployable skills and SHALL NOT receive tier tags.

#### Scenario: All skills classified
- **GIVEN** the 16 deployable skill files in `fab/.kit/skills/`
- **WHEN** the audit is complete
- **THEN** 4 skills SHALL have `model_tier: fast` in frontmatter
- **AND** 12 skills SHALL have no `model_tier` field (implicit capable)

## Fab-Workflow: Frontmatter Format

### Requirement: model_tier Field in Skill Frontmatter

Skills that require a non-default model tier SHALL include `model_tier:` in their YAML frontmatter block. The field uses the provider-agnostic tier name.

```yaml
---
name: fab-help
description: "Show the fab workflow overview and a quick summary of all available commands."
model_tier: fast
---
```

The `model_tier` field SHALL be placed after `description` in the frontmatter block.

#### Scenario: Adding model_tier to a fast skill
- **GIVEN** `fab-help.md` with existing `name` and `description` frontmatter
- **WHEN** the tier tag is applied
- **THEN** the frontmatter SHALL include `model_tier: fast` after the `description` field
- **AND** no other frontmatter fields SHALL be modified

#### Scenario: Capable skill has no model_tier
- **GIVEN** `fab-continue.md` classified as capable
- **WHEN** the tier tagging is complete
- **THEN** the frontmatter SHALL NOT contain a `model_tier` field

## Fab-Workflow: Tier-to-Model Mapping

### Requirement: Mapping File

A mapping file SHALL exist at `fab/.kit/model-tiers.yaml` defining the translation from generic tier names to provider-specific model identifiers.

```yaml
# fab/.kit/model-tiers.yaml
#
# Maps provider-agnostic tier names to provider-specific model identifiers.
# Used by fab-setup.sh when deploying skills to agent directories.

tiers:
  fast:
    claude: haiku
    # opencode: <TBD>
    # codex: <TBD>
  capable:
    claude: null    # null = use platform default (no model: override)
    # opencode: <TBD>
    # codex: <TBD>
```

The `capable` tier SHALL map to `null` (meaning: do not inject a `model:` field, let the platform use its default).

### Requirement: Per-Project Override via config.yaml

`fab/config.yaml` MAY contain an optional `model_tiers:` section that overrides the defaults from `.kit/model-tiers.yaml`. When present, per-tier entries in `config.yaml` take precedence over `.kit/` defaults.

```yaml
# fab/config.yaml (optional section)
model_tiers:
  fast:
    claude: sonnet   # override: use sonnet instead of haiku for fast-tier
```

`fab-setup.sh` SHALL merge the two sources: load `.kit/model-tiers.yaml` as the base, then overlay any entries from `config.yaml`'s `model_tiers:` section. Per-tier, per-platform entries in `config.yaml` replace (not deep-merge) the corresponding `.kit/` entry.

If `config.yaml` has no `model_tiers:` section, the `.kit/` defaults are used as-is.

#### Scenario: config.yaml overrides fast tier for Claude
- **GIVEN** `.kit/model-tiers.yaml` with `tiers.fast.claude: haiku`
- **AND** `config.yaml` with `model_tiers.fast.claude: sonnet`
- **WHEN** `fab-setup.sh` deploys a `model_tier: fast` skill for Claude Code
- **THEN** the deployed agent file SHALL contain `model: sonnet` (config.yaml wins)

#### Scenario: config.yaml has no model_tiers section
- **GIVEN** `config.yaml` with no `model_tiers:` key
- **WHEN** `fab-setup.sh` deploys a `model_tier: fast` skill for Claude Code
- **THEN** the `.kit/model-tiers.yaml` default SHALL be used (`model: haiku`)

#### Scenario: Looking up the Claude model for "fast" tier
- **GIVEN** the mapping file with `tiers.fast.claude: haiku`
- **WHEN** `fab-setup.sh` deploys a `model_tier: fast` skill for Claude Code
- **THEN** the deployed skill SHALL include `model: haiku` in its frontmatter

#### Scenario: Looking up the Claude model for "capable" tier
- **GIVEN** the mapping file with `tiers.capable.claude: null`
- **WHEN** `fab-setup.sh` deploys a skill with no `model_tier` for Claude Code
- **THEN** the deployed skill SHALL NOT contain a `model:` field

### Requirement: Dual Deployment for Fast-Tier Skills

`fab-setup.sh` SHALL deploy fast-tier skills to **both** skill and agent directories, giving them user invocation (via skill) and model-optimized pipeline invocation (via agent).

For skills with `model_tier: fast`:
- **Skill directory** (`.claude/skills/`): Create a symlink as usual (for user invocation via `/fab-help`). The `model_tier` field is inert metadata in the skill context.
- **Agent directory** (`.claude/agents/`): Generate an agent file with the translated `model:` field (e.g., `model: haiku`). This enables pipeline skills to spawn fast-tier operations via the Task tool with cost-appropriate model selection.
- The generated agent file SHALL be self-contained (full content, not a symlink) and SHALL replace `model_tier` with the provider-specific `model:` field.

For skills without `model_tier` (capable/default):
- **Skill directory**: Create a symlink as usual (no change from current behavior)
- **Agent directory**: No agent file generated (capable skills run in the main conversation context)
<!-- clarified: Dual deployment (skill symlink + agent file) for fast-tier instead of generated-file-only — skill symlinks preserve user invocation; agent files provide working model selection for pipeline use -->

#### Scenario: Deploying a fast-tier skill to Claude Code
- **GIVEN** `fab-help.md` with `model_tier: fast` in frontmatter
- **AND** the mapping file with `tiers.fast.claude: haiku`
- **WHEN** `fab-setup.sh` runs
- **THEN** `.claude/skills/fab-help/SKILL.md` SHALL be a symlink to the canonical file
- **AND** `.claude/agents/fab-help.md` SHALL be a generated file
- **AND** the agent file's frontmatter SHALL contain `model: haiku` (not `model_tier: fast`)

#### Scenario: Deploying a capable skill to Claude Code
- **GIVEN** `fab-continue.md` with no `model_tier` field
- **WHEN** `fab-setup.sh` runs
- **THEN** `.claude/skills/fab-continue/SKILL.md` SHALL remain a symlink to the canonical file
- **AND** no `.claude/agents/fab-continue.md` file SHALL be created

#### Scenario: Updating .kit/ with fast-tier agent files deployed
- **GIVEN** fast-tier skills with agent files in `.claude/agents/`
- **WHEN** `fab-update.sh` replaces `.kit/` and re-runs `fab-setup.sh`
- **THEN** the agent files SHALL be regenerated with the updated content
- **AND** skill symlinks SHALL be repaired as before

#### Scenario: Pipeline skill spawns a fast-tier agent
- **GIVEN** a pipeline skill (e.g., `fab-fff`) needs to invoke `fab-help`
- **AND** `.claude/agents/fab-help.md` exists with `model: haiku`
- **WHEN** the pipeline skill uses the Task tool with `subagent_type: fab-help`
- **THEN** the invocation SHALL use the haiku model as specified in the agent file

### Requirement: Deployment Error Handling

`fab-setup.sh` SHALL validate the tier system strictly. Since `model-tiers.yaml` ships inside `.kit/`, its absence or corruption indicates a broken kit — not a user configuration error.

#### Scenario: Missing mapping file
- **GIVEN** `fab/.kit/model-tiers.yaml` does not exist
- **WHEN** `fab-setup.sh` runs
- **THEN** the script SHALL exit non-zero
- **AND** print to stderr: `ERROR: fab/.kit/model-tiers.yaml not found — kit may be corrupted.`

#### Scenario: Unrecognized model_tier value
- **GIVEN** a skill file with `model_tier: medium` (not a recognized tier)
- **WHEN** `fab-setup.sh` reads its frontmatter
- **THEN** the script SHALL exit non-zero
- **AND** print to stderr: `ERROR: Unrecognized model_tier "medium" in {skill}.md. Valid values: fast`

#### Scenario: model_tier present but no mapping for current platform
- **GIVEN** a skill with `model_tier: fast`
- **AND** the mapping file has no entry for the current platform under `tiers.fast`
- **WHEN** `fab-setup.sh` runs
- **THEN** the script SHALL exit non-zero
- **AND** print to stderr: `ERROR: No mapping for tier "fast" on platform "{platform}" in model-tiers.yaml`

#### Scenario: Skill frontmatter unparseable
- **GIVEN** a skill file with malformed YAML frontmatter
- **WHEN** `fab-setup.sh` attempts to read `model_tier`
- **THEN** the script SHALL exit non-zero
- **AND** print to stderr: `ERROR: Cannot parse frontmatter in {skill}.md`

## Fab-Workflow: Documentation

### Requirement: New Model Tiers Doc

A new centralized doc SHALL be created at `fab/docs/fab-workflow/model-tiers.md` documenting:

1. The tier naming scheme (fast, capable) and their intent
2. The tier selection criteria table
3. The full skill classification audit results
4. The mapping file format and location
5. How to add a new provider mapping
6. How future skill authors should select a tier

#### Scenario: New skill author selects a tier
- **GIVEN** a developer creating a new fab skill
- **WHEN** they consult `fab/docs/fab-workflow/model-tiers.md`
- **THEN** they SHALL find clear criteria and examples for choosing between `fast` and `capable`

### Requirement: Update Kit Architecture Doc

`fab/docs/fab-workflow/kit-architecture.md` SHALL be updated to:

1. Add `model-tiers.yaml` to the `.kit/` directory structure listing
2. Document the dual deployment strategy (skill symlinks for all skills, plus agent files for fast-tier)
3. Note the model tier system in the "Agent Integration via Symlinks" section

#### Scenario: Directory structure reflects model-tiers.yaml
- **GIVEN** the updated kit-architecture doc
- **WHEN** a reader views the `.kit/` directory structure
- **THEN** `model-tiers.yaml` SHALL appear in the listing

### Requirement: Update Templates Doc

`fab/docs/fab-workflow/templates.md` SHALL be updated to document the `model_tier` frontmatter field for skills, including valid values and default behavior.

#### Scenario: Templates doc describes model_tier
- **GIVEN** the updated templates doc
- **WHEN** a reader looks up skill frontmatter fields
- **THEN** the `model_tier` field SHALL be documented with valid values (`fast`) and default behavior (omission = capable)

## Design Decisions

1. **Two tiers, not three**: Only `fast` and `capable`. No middle tier.
   - *Why*: The gap between "simple script delegation" and "artifact generation with reasoning" is clear-cut. A middle tier would create ambiguous classification decisions. If a skill needs any reasoning, it needs the capable model.
   - *Rejected*: Three tiers (fast/standard/capable) — adds classification complexity without practical benefit; most skills clearly fall into one of two buckets.

2. **Omission = capable (implicit default)**: Only `fast` is explicitly tagged. No `model_tier: capable` needed.
   - *Why*: 12 of 16 skills are capable. Tagging only the 4 exceptions reduces frontmatter noise and follows the convention of annotating only deviations from the default.
   - *Rejected*: Explicit `model_tier: capable` on all skills — adds noise to 12 files without information gain.

3. **Dual deployment for fast-tier (skill symlink + agent file)**: Fast-tier skills get both a symlink in `.claude/skills/` and a generated agent file in `.claude/agents/`.
   - *Why*: Claude Code skills don't support `model:` in frontmatter — only agents do. Skill symlinks preserve user invocation (`/fab-help`); agent files enable model-optimized invocation via the Task tool in pipeline operations. All symlinks are preserved; agent files are additive.
   - *Rejected*: Generated skill files replacing symlinks — `model:` is ignored in skill context, so the complexity gains nothing. Agent-only deployment — breaks user invocation via `/skill-name`.

4. **`model_tier` field name (not `model`)**: The canonical frontmatter uses `model_tier`, not `model`.
   - *Why*: `model:` is a platform-specific field (Claude Code uses it to select haiku/sonnet/opus). Using `model_tier:` makes it clear this is a generic tier, not a provider model name. Avoids confusion when the same file appears in `.kit/` (with `model_tier`) and `.claude/` (with `model`).
   - *Rejected*: `model:` with tier names directly — ambiguous; platforms would try to interpret `fast` as a model name and fail.

5. **Defaults in `.kit/`, overridable via `config.yaml`**: `fab/.kit/model-tiers.yaml` provides sensible defaults; `fab/config.yaml` can override per-project.
   - *Why*: Most users won't customize, but power users may need project-specific model selection (e.g., `fast` → `sonnet` instead of `haiku` for projects where Haiku is too limited). The override keeps `.kit/` portable while allowing per-project tuning.
   - *Rejected*: Fixed `.kit/`-only mapping — forces forking `.kit/` for any customization, which defeats portability.
   <!-- clarified: config.yaml override — .kit/ provides defaults, config.yaml can override per-project -->

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | 4 fast skills: fab-help, fab-status, fab-switch, fab-init | Audit shows these delegate to scripts or do simple lookups; no artifact generation or SRAD |
| 2 | Confident | `model_tier` as field name (not `model` or `tier`) | Avoids collision with platform-specific `model:` field; clear intent |
| 3 | Confident | Anthropic-only mapping initially | Current primary platform; other providers are out of scope per Non-Goals |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-12

- **Q**: The spec deploys fast-tier skills to `.claude/skills/` with `model: haiku` injected, but Claude Code skills don't support `model:` (only agents do). How should the spec handle this?
  **A**: Dual deployment — fast-tier skills get both a skill symlink (for user invocation) and a generated agent file in `.claude/agents/` (for model-optimized pipeline invocation).
- **Q**: How should `fab-setup.sh` handle deployment errors (missing mapping file, invalid tier values, parse failures)?
  **A**: Strict validation — exit non-zero with descriptive error. Since `model-tiers.yaml` ships with `.kit/`, its absence indicates kit corruption.
- **Q**: Should the tier-to-model mapping be fixed in `.kit/` or overridable via `config.yaml`?
  **A**: config.yaml override — `.kit/model-tiers.yaml` provides defaults, `config.yaml` can override per-project via optional `model_tiers:` section.
- **Q**: Should `fab-switch` remain classified as `fast` despite interactive prompts and branch logic?
  **A**: Accepted recommendation: keep as `fast` — procedural logic, no artifact generation or SRAD.
