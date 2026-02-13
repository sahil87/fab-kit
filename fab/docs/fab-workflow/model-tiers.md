# Model Tiers

**Domain**: fab-workflow

## Overview

Fab uses a two-tier model classification system to match skill complexity with appropriate AI models. Simple skills that delegate to shell scripts run on cheaper, faster models; complex skills that generate artifacts or require deep reasoning use the platform's most capable model. The tier system is provider-agnostic — canonical skill files declare a generic tier, and the deployment script translates it to provider-specific model identifiers.

## Requirements

### Tier Naming Scheme

Two tiers:

- **`fast`** — skills that delegate to shell scripts, display formatted output, or perform simple lookups without deep reasoning or artifact generation
- **`capable`** (implicit default) — skills that require multi-step reasoning, artifact generation, SRAD evaluation, code analysis, or review

A skill that omits the `model_tier` field is treated as `capable`. Only skills explicitly marked `model_tier: fast` use a cheaper/faster model.

### Tier Selection Criteria

| Criterion | fast | capable |
|-----------|------|---------|
| Delegates to shell script | Yes | — |
| Generates markdown artifacts | — | Yes |
| Applies SRAD framework | — | Yes |
| Reads/modifies source code | — | Yes |
| Requires multi-step reasoning | — | Yes |
| Simple state lookup or display | Yes | — |

A skill matching ANY `capable` criterion is classified as `capable`, regardless of other characteristics.

### Skill Classification Audit

**Fast tier** (`model_tier: fast`):

| Skill | Rationale |
|-------|-----------|
| `fab-help` | Delegates to `fab-help.sh` |
| `fab-status` | Delegates to `fab-status.sh` |
| `fab-switch` | State lookup, branch operations, no artifact generation |
| `fab-init` | Structural bootstrap, delegates to `fab-setup.sh` |

**Capable tier** (no `model_tier` field):

| Skill | Rationale |
|-------|-----------|
| `fab-new` | Brief generation, SRAD evaluation |
| `fab-hydrate` | Content analysis, doc generation/merging |
| `fab-continue` | Artifact generation (spec/tasks), code implementation, review validation, documentation hydration |
| `fab-ff` | Multi-stage pipeline orchestration |
| `fab-fff` | Full pipeline with confidence gating |
| `fab-clarify` | Gap resolution, deep reasoning |
| `fab-hydrate-design` | Structural gap analysis, spec modification |
| `internal-consistency-check` | Cross-layer drift detection |
| `internal-retrospect` | Retrospective analysis |

Shared partials (`_context.md`, `_generation.md`) are not deployable and have no tier.

### Mapping File

`fab/.kit/model-tiers.yaml` defines the translation from tier names to provider-specific model identifiers:

```yaml
tiers:
  fast:
    claude: haiku
  capable:
    claude: null    # null = platform default
```

### Per-Project Override

`fab/config.yaml` can optionally override `.kit/` defaults:

```yaml
model_tiers:
  fast:
    claude: sonnet   # use sonnet instead of haiku
```

Config entries replace (not merge with) the corresponding `.kit/` entries. If no `model_tiers:` section exists, defaults apply.

### Deployment: Dual Strategy

`fab-setup.sh` deploys fast-tier skills to both skill and agent directories:

- **Skill directory** (`.claude/skills/`): Symlink as usual — for user invocation via `/fab-help`
- **Agent directory** (`.claude/agents/`): Generated file with translated `model:` field — for pipeline invocation via Task tool

Capable skills get symlinks only (no agent files). This is because Claude Code skills don't support `model:` in frontmatter; only agents do.

### Adding a New Provider

1. Add the platform key under each tier in `fab/.kit/model-tiers.yaml`
2. Update `fab-setup.sh` to generate agent files for the new platform
3. Add the symlink/agent creation call in the appropriate section

### Selecting a Tier for New Skills

When creating a new fab skill, ask:

1. Does it generate markdown artifacts? → **capable**
2. Does it apply the SRAD framework? → **capable**
3. Does it read or modify source code? → **capable**
4. Does it require multi-step reasoning? → **capable**
5. Does it mainly delegate to a shell script? → **fast**
6. Is it a simple state lookup or display? → **fast**

If in doubt, use **capable** (the default — just omit `model_tier`).

## Design Decisions

### Two Tiers, Not Three
**Decision**: Only `fast` and `capable`. No middle tier.
**Why**: The gap between script delegation and artifact generation is clear-cut. A middle tier would create ambiguous classification decisions.
**Rejected**: Three tiers (fast/standard/capable) — adds complexity without practical benefit.
*Introduced by*: 260212-k8m3-skill-model-tiers

### Omission = Capable
**Decision**: Only `fast` is explicitly tagged. No `model_tier: capable` needed.
**Why**: 12 of 16 skills are capable. Tagging only the 4 exceptions reduces noise.
**Rejected**: Explicit `model_tier: capable` on all skills.
*Introduced by*: 260212-k8m3-skill-model-tiers

### Dual Deployment for Fast-Tier
**Decision**: Fast-tier skills get both a skill symlink and a generated agent file.
**Why**: Claude Code skills don't support `model:` — only agents do. Skill symlinks preserve user invocation; agent files enable model selection in pipelines.
**Rejected**: Generated skill files (model: ignored in skill context). Agent-only (breaks user invocation).
*Introduced by*: 260212-k8m3-skill-model-tiers

### Defaults in .kit/, Overridable via config.yaml
**Decision**: `model-tiers.yaml` provides defaults; `config.yaml` can override per-project.
**Why**: Most users won't customize, but power users may need project-specific model selection.
**Rejected**: Fixed .kit/-only mapping — forces forking for any customization.
*Introduced by*: 260212-k8m3-skill-model-tiers

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260212-k8m3-skill-model-tiers | 2026-02-12 | Initial creation — two-tier system, skill classification audit, mapping file, dual deployment, config override |
