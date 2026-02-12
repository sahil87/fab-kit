# Proposal: Provider-Agnostic Model Tiers for Fab Skills

**Change**: 260212-k8m3-skill-model-tiers
**Created**: 2026-02-12
**Status**: Draft

## Why

Fab skills currently either omit model specifications (defaulting to the user's chosen model, typically Opus) or hard-code Anthropic-specific model names (like `haiku`). This creates two problems: (1) expensive/slow operations when a cheaper/faster model would suffice, and (2) skills are not portable across AI providers. A provider-agnostic model tier system allows skills to declare their complexity level while remaining provider-neutral.

## What Changes

1. **Audit all skills** in `fab/.kit/skills/` and categorize each by appropriate model tier based on task complexity, reasoning requirements, and cost sensitivity
2. **Define a provider-agnostic tier naming scheme** (e.g., `fast`/`capable` or `low`/`high`) that abstracts over provider-specific model names
   <!-- assumed: "fast"/"capable" naming — clearer intent than "low"/"high", aligns with common LLM tier terminology -->
3. **Create a tier-to-model mapping system** that translates generic tier names to provider-specific model identifiers when skills are deployed to agent directories (`.agents/`, `.claude/`, `.opencode/`, `.codex/`)
4. **Update skill frontmatter** to include `model: {tier}` field where appropriate, using the generic tier name
5. **Document the tier selection criteria** so future skill authors know when to use each tier

## Affected Docs

### New Docs
- `fab-workflow/model-tiers`: Documents the tier system, tier-to-provider mapping, and criteria for selecting tiers

### Modified Docs
- `fab-workflow/kit-architecture`: Add section on model tier system and skill deployment
- `fab-workflow/templates`: Document the `model:` frontmatter field for skills

### Removed Docs
None

## Impact

- All 18 skill files in `fab/.kit/skills/` will be reviewed and potentially updated with `model:` frontmatter
- Deployment/copy scripts (if they exist) need to handle tier-to-model translation
- Future skill authors need clear guidance on tier selection

## Open Questions

- [DEFERRED] Should the tier-to-model mapping be configurable per-project (in `fab/config.yaml`), or is a fixed mapping in `.kit/` sufficient? <!-- assumed: fixed mapping in .kit/ — most users won't need to customize this, keeps config simpler -->

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Use "fast"/"capable" tier names | Clearer intent than "low"/"high", aligns with common LLM tier terminology (fast = Haiku/GPT-3.5, capable = Sonnet/GPT-4) |
| 2 | Confident | Tier selection criteria: task complexity, reasoning needs, cost sensitivity | We already identified fab-help as a "fast" candidate (simple bash execution), can generalize this |
| 3 | Confident | Mapping lives in `.kit/` (not per-project config) | Follows fab architecture principle — `.kit/` is portable, project config is minimal |
| 4 | Confident | Start with Anthropic mapping, design for extensibility | Current primary use case, clear extension path for other providers |
| 5 | Tentative | Fixed mapping vs. configurable per-project | Most users won't customize this, but power users might want provider-specific overrides |

5 assumptions made (4 confident, 1 tentative). Run /fab-clarify to review.
