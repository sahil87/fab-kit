# Model Tiers

**Domain**: fab-workflow

## Overview

All fab skills run on the session's default model — there is no model tier system. The former two-tier system (`fast` and `capable`) was eliminated because the `fast` tier caused mid-conversation context compaction in Claude Code when switching from a high-context model to Haiku's smaller context window.

## Requirements

### Single Tier: All Skills Use Platform Default

No skill file in `fab/.kit/skills/` declares a `model_tier` field. All skills run on whatever model the user's session is using. There is no `model_tiers:` section in `config.yaml`, no tier resolution logic in `2-sync-workspace.sh`, and no model substitution during skill deployment.

### Deployment: Plain Copy

`sync/2-sync-workspace.sh` deploys skills to platform-specific directories:

- **Claude Code** (`.claude/skills/`): Plain copies (byte-accurate)
- **OpenCode** (`.opencode/commands/`): Symlinks to canonical skill files in `.kit/skills/`
- **Codex** (`.agents/skills/`): Plain copies (Codex ignores symlinks)

All skills are deployed identically to all platforms — no per-skill model templating or substitution.

## Design Decisions

### ~~Two Tiers, Not Three~~ (Superseded)
**Decision**: ~~Only `fast` and `capable`. No middle tier.~~
**Superseded by**: Single tier — all skills use platform default. The fast tier caused context compaction.
*Introduced by*: 260212-k8m3-skill-model-tiers
*Superseded by*: 260226-85rg-drop-fast-model-tier

### ~~Omission = Capable~~ (Superseded)
**Decision**: ~~Only `fast` is explicitly tagged. No `model_tier: capable` needed.~~
**Superseded by**: No tier tagging at all — the `model_tier` frontmatter field no longer exists.
*Introduced by*: 260212-k8m3-skill-model-tiers
*Superseded by*: 260226-85rg-drop-fast-model-tier

### ~~Dual Deployment for Fast-Tier~~ (Superseded)
**Decision**: ~~Fast-tier skills get both a skill symlink and a generated agent file.~~
**Superseded by**: Copy-with-template (260219), then eliminated entirely (260226) when the fast tier was removed.
*Introduced by*: 260212-k8m3-skill-model-tiers
*Superseded by*: 260219-d2y2-copy-template-skills-drop-agents, 260226-85rg-drop-fast-model-tier

### ~~Single Source in config.yaml with Hardcoded Fallback~~ (Superseded)
**Decision**: ~~Model tier mappings live exclusively in `config.yaml`.~~
**Superseded by**: No tier mappings anywhere — `model_tiers:` section removed from config and scaffold.
*Introduced by*: 260218-bb93-restructure-config-yaml
*Superseded by*: 260226-85rg-drop-fast-model-tier

### Eliminate Fast Tier to Prevent Context Compaction
**Decision**: Remove the fast tier entirely — all skills use the session's default model.
**Why**: When Claude Code invoked a `model_tier: fast` skill, it switched to Haiku mid-conversation. Haiku's smaller context window forced conversation compaction, losing context and degrading the user experience. The cost savings from Haiku for lightweight skills were negligible compared to the disruption. The same problem had already led to `git-pr` being moved off the fast tier.
**Rejected**: Fix only the most-affected skill (`fab-switch`) — the problem applied equally to all fast-tier skills. Keep the infrastructure "in case we need it later" — violates YAGNI.
*Introduced by*: 260226-85rg-drop-fast-model-tier

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260226-85rg-drop-fast-model-tier | 2026-02-26 | Eliminated the fast tier entirely. Removed `model_tier: fast` from all 5 remaining skills, `model_tiers:` from config and scaffold, tier classification/resolution logic from sync script, `yaml_value` helper, and model-tier tests. All skills now use the session's default model. Added migration 0.21.0→0.22.0. Bumped kit to v0.22.0. |
| 260222-s101-wt-create-stderr-wt-list-flags | 2026-02-22 | Moved `git-pr` from fast to capable tier — removed `model_tier: fast` and preamble directive. Commit message generation needs reasoning; haiku's smaller context window caused limit hits when invoked late in pipeline sessions. |
| 260219-d2y2-copy-template-skills-drop-agents | 2026-02-19 | Replaced dual deployment (symlinks + agent files) with copy-with-template: Claude Code skills deployed as copies with `model_tier:` → `model:` substitution. Removed agent file generation. Added transitional agent cleanup. Marked Dual Deployment design decision as superseded |
| 260218-5isu-fix-docs-consistency-drift | 2026-02-18 | Replaced stale `fab-init` → `fab-setup` in skill classification and `lib/sync-workspace.sh` → correct paths (`fab-sync.sh`, `sync/2-sync-workspace.sh`) in deployment references |
| 260218-bb93-restructure-config-yaml | 2026-02-18 | Deleted `fab/.kit/model-tiers.yaml`, consolidated into `config.yaml` `model_tiers:` section with hardcoded `haiku` fallback. Updated mapping file section, provider instructions, and design decisions |
| 260216-gqpp-DEV-1040-code-review-loop | 2026-02-16 | Added review sub-agent classification as capable tier — spawned during pipeline execution by `/fab-continue`, `/fab-ff`, `/fab-fff` |
| 260215-v4n7-DEV-1025-rename-brief-to-intake | 2026-02-15 | Renamed `brief` stage/artifact to `intake` throughout — stage identifiers, artifact filenames, YAML keys, prose references |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_preflight.sh` → `lib/preflight.sh` and `_init_scaffold.sh` → `lib/sync-workspace.sh` in skill classification and deployment references |
| 260212-k8m3-skill-model-tiers | 2026-02-12 | Initial creation — two-tier system, skill classification audit, mapping file, dual deployment, config override |
