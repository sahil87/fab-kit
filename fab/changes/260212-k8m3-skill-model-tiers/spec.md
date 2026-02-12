# Spec: Provider-Agnostic Model Tiers for Fab Skills

**Change**: 260212-k8m3-skill-model-tiers
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/model-tiers.md` (new), `fab/docs/fab-workflow/kit-architecture.md`, `fab/docs/fab-workflow/templates.md`

## Non-Goals

- Runtime model routing — this change does not implement a runtime mechanism that dynamically selects models. It defines metadata and deployment-time translation only.
- Multi-provider deployment testing — only the Anthropic mapping needs to be functional. Other providers are designed for but not tested.
- Changing skill invocation UX — users continue to invoke skills via `/fab-*` commands as before.

## Model Tier System: Tier Definition

### Requirement: Two-Tier Model Classification

The system SHALL define exactly two model tiers:

| Tier | Intent | Typical provider mapping |
|------|--------|------------------------|
| `fast` | Cheap, low-latency operations with minimal reasoning | Haiku-class models |
| `capable` | Complex reasoning, artifact generation, SRAD analysis | Sonnet/Opus-class models |

Skills that do not specify a tier SHALL default to `capable`.

#### Scenario: Skill with explicit fast tier
- **GIVEN** a skill file with `model: fast` in its YAML frontmatter
- **WHEN** the deployment script processes this skill
- **THEN** the skill is deployed with the provider-specific model for the `fast` tier

#### Scenario: Skill with no model field
- **GIVEN** a skill file with no `model:` field in its YAML frontmatter
- **WHEN** the deployment script processes this skill
- **THEN** the skill is deployed with the provider-specific model for the `capable` tier (the default)

#### Scenario: Skill with explicit capable tier
- **GIVEN** a skill file with `model: capable` in its YAML frontmatter
- **WHEN** the deployment script processes this skill
- **THEN** the skill is deployed identically to a skill with no `model:` field

### Requirement: Tier Assignment Criteria

The tier selection criteria SHALL be documented and follow these guidelines:

| Criterion | `fast` | `capable` |
|-----------|--------|-----------|
| Primary action | Script execution, file matching, mechanical I/O | Artifact generation, reasoning, analysis |
| Context loading | None or minimal (config only) | Full (constitution, docs, design, specs) |
| SRAD involvement | None | Active (evaluates decisions, grades assumptions) |
| User interaction | Simple prompts (pick from list) | Conversational, clarifying questions |
| Output type | Pre-formatted script output, status display | Generated prose, specifications, code |

#### Scenario: Categorizing a script-delegation skill
- **GIVEN** a skill whose behavior is "run a bash script and display output"
- **WHEN** evaluating its tier
- **THEN** it SHOULD be assigned `fast` because no LLM reasoning is required beyond following instructions

#### Scenario: Categorizing an artifact-generation skill
- **GIVEN** a skill that generates spec.md, tasks.md, or performs SRAD analysis
- **WHEN** evaluating its tier
- **THEN** it SHOULD be assigned `capable` because it requires deep reasoning over project context

## Model Tier System: Skill Frontmatter

### Requirement: Model Field in Skill Frontmatter

Skills in `fab/.kit/skills/` that are assigned the `fast` tier MUST include a `model: fast` field in their YAML frontmatter. Skills assigned the `capable` tier SHOULD omit the `model:` field entirely (relying on the default).

<!-- assumed: capable tier omits frontmatter field — reduces noise in the majority of skills, "no field = capable" is the simplest convention -->

#### Scenario: Fast skill frontmatter
- **GIVEN** `fab/.kit/skills/fab-help.md`
- **WHEN** the `model: fast` field is added to its frontmatter
- **THEN** the frontmatter reads:
  ```yaml
  ---
  name: fab-help
  description: "Show the fab workflow overview..."
  model: fast
  ---
  ```

#### Scenario: Capable skill frontmatter unchanged
- **GIVEN** `fab/.kit/skills/fab-new.md` (a capable-tier skill)
- **WHEN** reviewed for frontmatter changes
- **THEN** no `model:` field is added — the absence of the field indicates `capable`

## Model Tier System: Tier-to-Provider Mapping

### Requirement: Central Mapping File

A mapping file SHALL exist at `fab/.kit/model-tiers.yaml` that maps generic tier names to provider-specific model identifiers.

The file SHALL have the following structure:

```yaml
# fab/.kit/model-tiers.yaml
#
# Maps generic model tiers to provider-specific model identifiers.
# Read by fab-setup.sh during deployment to agent directories.

tiers:
  fast:
    description: "Cheap, low-latency — script execution, mechanical operations"
    providers:
      anthropic: haiku
      # openai: gpt-4o-mini
      # google: gemini-flash
  capable:
    description: "Full reasoning — artifact generation, SRAD, code review"
    providers:
      anthropic: null  # use platform default (user's chosen model)
      # openai: null
      # google: null
```

<!-- assumed: mapping file in .kit/ as YAML — consistent with config.yaml convention, machine-readable for fab-setup.sh -->

#### Scenario: Reading the mapping for Anthropic fast tier
- **GIVEN** the mapping file exists at `fab/.kit/model-tiers.yaml`
- **WHEN** `fab-setup.sh` looks up the `fast` tier for the `anthropic` provider
- **THEN** it gets the value `haiku`

#### Scenario: Reading the mapping for a capable tier
- **GIVEN** the mapping file exists
- **WHEN** `fab-setup.sh` looks up the `capable` tier for the `anthropic` provider
- **THEN** it gets `null`, meaning no model override (use platform default)

### Requirement: Extensibility for Future Providers

The mapping file SHOULD include commented-out entries for non-Anthropic providers as extension points. Active (uncommented) entries MUST exist only for providers that have been tested.

#### Scenario: Adding a new provider
- **GIVEN** a user wants to add OpenAI support
- **WHEN** they uncomment and fill in the `openai` entries
- **THEN** `fab-setup.sh` can deploy with OpenAI model identifiers (provided it supports that agent platform)

## Deployment: Setup Script Integration

### Requirement: Deployment-Time Tier Translation

`fab-setup.sh` MUST read the `model:` frontmatter from each skill file and the tier mapping from `fab/.kit/model-tiers.yaml`, then apply the provider-specific model when creating agent integrations.

For agent platforms that support per-skill model specification, the deployment MUST set the model. For platforms that do not, the `model:` field serves as documentation metadata only.
<!-- assumed: deployment translates at setup time — preserves symlink-based updates where possible, consistent with existing fab-setup.sh responsibility -->

#### Scenario: Deploying a fast skill to Claude Code
- **GIVEN** `fab-help.md` has `model: fast` in frontmatter
- **AND** the Anthropic mapping for `fast` is `haiku`
- **WHEN** `fab-setup.sh` creates the Claude Code integration
- **THEN** the deployed skill includes the model `haiku` in a way Claude Code can consume
- **AND** the skill remains invocable via `/fab-help`

#### Scenario: Deploying a capable skill to Claude Code
- **GIVEN** `fab-new.md` has no `model:` field (defaults to `capable`)
- **AND** the Anthropic mapping for `capable` is `null`
- **WHEN** `fab-setup.sh` creates the Claude Code integration
- **THEN** no model override is applied — the skill uses the session's default model

#### Scenario: Preserving update semantics
- **GIVEN** `fab-setup.sh` has been run with tier translation
- **WHEN** `fab-update.sh` replaces `fab/.kit/` with a new version
- **AND** `fab-setup.sh` is re-run
- **THEN** all tier translations are correctly reapplied from the updated skill files and mapping

## Skill Audit: Tier Assignments

### Requirement: Complete Skill Audit

Every skill file in `fab/.kit/skills/` (excluding partials `_context.md` and `_generation.md`) SHALL be reviewed and assigned a tier. The following assignments SHALL apply:

#### `fast` Tier Skills

| Skill | Rationale |
|-------|-----------|
| `fab-help` | Delegates to `fab-help.sh`. No context loading, no reasoning. |
| `fab-status` | Delegates to `fab-status.sh`. No context loading, no reasoning. |
| `fab-switch` | Folder matching, file write, git operations. Deterministic branching logic, no artifact generation. |

#### `capable` Tier Skills (default — no `model:` field)

| Skill | Rationale |
|-------|-----------|
| `fab-new` | SRAD analysis, brief generation, conversational interaction. |
| `fab-continue` | Spec/tasks generation, SRAD analysis, full context loading. |
| `fab-ff` | Multi-stage artifact generation, cumulative SRAD. |
| `fab-fff` | Full pipeline orchestration, confidence gating. |
| `fab-clarify` | Ambiguity resolution, deep reasoning over assumptions. |
| `fab-apply` | Code implementation from task specs. |
| `fab-review` | Implementation validation against specs and checklists. |
| `fab-archive` | Doc hydration, semantic merging into centralized docs. |
| `fab-hydrate` | Document ingestion, generation, domain analysis. |
| `fab-backfill` | Gap analysis between docs and specs, structural comparison. |
| `fab-init` | Interactive config/constitution generation, project understanding. |
| `internal-consistency-check` | Cross-document validation, drift detection. |
| `internal-retrospect` | Session analysis, pattern recognition. |

#### Scenario: Verifying all skills are audited
- **GIVEN** the skill audit table above
- **WHEN** compared against the list of `*.md` files in `fab/.kit/skills/` (excluding `_context.md` and `_generation.md`)
- **THEN** every skill file appears in exactly one tier table (fast or capable)

## Documentation: Model Tiers Doc

### Requirement: New Centralized Doc

A new centralized doc SHALL be created at `fab/docs/fab-workflow/model-tiers.md` documenting:

1. The tier system (tier names, descriptions, default)
2. Tier assignment criteria (the decision matrix)
3. The complete skill-to-tier audit table
4. The mapping file format and location
5. How to add support for new providers
6. How deployment translates tiers

#### Scenario: Future skill author selecting a tier
- **GIVEN** a developer is creating a new skill `fab-foo.md`
- **WHEN** they consult `fab/docs/fab-workflow/model-tiers.md`
- **THEN** the criteria table helps them determine whether to use `fast` or `capable`
- **AND** they know to add `model: fast` to frontmatter if applicable, or omit it for `capable`

### Requirement: Updated Kit Architecture Doc

`fab/docs/fab-workflow/kit-architecture.md` SHALL be updated to:

1. Reference the model tier system in the directory structure listing (add `model-tiers.yaml` to the `.kit/` tree)
2. Document how `fab-setup.sh` handles tier translation during deployment

#### Scenario: Kit architecture reflects model tiers
- **GIVEN** the updated `kit-architecture.md`
- **WHEN** a reader looks at the `.kit/` directory structure
- **THEN** `model-tiers.yaml` appears in the listing with a description

### Requirement: Updated Templates Doc

`fab/docs/fab-workflow/templates.md` SHALL be updated to document the `model:` frontmatter field as an optional field in skill files.

#### Scenario: Templates doc includes model field
- **GIVEN** the updated `templates.md`
- **WHEN** a reader looks at the skill frontmatter documentation
- **THEN** the `model:` field is listed with its accepted values (`fast`, or omit for `capable`)

## Design Decisions

1. **Two tiers, not three**: Use `fast`/`capable` instead of `fast`/`standard`/`capable` or a numeric scale.
   - *Why*: The current skill landscape splits cleanly into "runs a script" vs "generates artifacts." A middle tier would be empty today and invite bikeshedding. Easy to split later if needed.
   - *Rejected*: Three-tier (`fast`/`standard`/`capable`) — no skills currently need a middle tier, and the distinction between standard and capable would be subjective.

2. **Capable as implicit default**: Skills without a `model:` field are `capable` rather than requiring explicit annotation on every file.
   - *Why*: ~85% of skills (13/16) are capable-tier. Annotating only the minority reduces noise and makes `fast` opt-in (safe default is the expensive-but-reliable model).
   - *Rejected*: Explicit `model: capable` on every file — verbose, no information gain over absence.

3. **Central mapping file, not per-skill provider mappings**: Provider-specific models are in one YAML file, not scattered across skill frontmatter.
   - *Why*: Changing providers means editing one file. Skills stay provider-agnostic. Consistent with `.kit/` portability principle.
   - *Rejected*: Per-skill `providers:` blocks in frontmatter — couples skills to providers, violates portability.

4. **Deployment-time translation, not runtime**: `fab-setup.sh` bakes in the provider model during deployment rather than resolving at invocation time.
   - *Why*: No runtime dependency on the mapping file. Simpler for agent platforms that read static config. Compatible with the existing symlink-based deployment.
   - *Rejected*: Runtime resolution — requires agent platforms to understand generic tiers (none do today), adds a dependency on the mapping file at invocation time.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | "fast"/"capable" tier names | Carried from brief — clearer intent than "low"/"high", aligns with common LLM tier terminology |
| 2 | Confident | Capable as implicit default (no `model:` field) | 13/16 skills are capable; annotating only the fast minority reduces noise |
| 3 | Confident | Mapping file at `fab/.kit/model-tiers.yaml` | YAML consistent with `config.yaml`, `.kit/` location follows portability principle |
| 4 | Confident | `fab-switch` is `fast` tier | Deterministic matching/branching logic with no artifact generation, despite being 315 lines of instructions |
| 5 | Tentative | Deployment-time translation mechanism in `fab-setup.sh` | Preserves symlinks where possible, but exact mechanism (generated wrappers, metadata files, or partial copies) deferred to implementation |

5 assumptions made (4 confident, 1 tentative). Run /fab-clarify to review.
