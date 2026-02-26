# Intake: Drop Fast Model Tier

**Change**: 260226-85rg-drop-fast-model-tier
**Created**: 2026-02-26
**Status**: Draft

## Origin

> fab-switch causes a compacting of the conversation mid-conversation. My guess is this happens because of the model tier switch from a high context limit to a lower context limit. If it's causing such an issue, may be we should remove the model tier 'fast' from these commands.

One-shot input. The user observed conversation compaction when invoking `/fab-switch` during a conversation and attributed it to the model tier switch from the current session model (e.g., Opus with 1M context) to Haiku (smaller context window). This is the same class of problem that led to `git-pr` being moved from fast to capable tier in change `260222-s101`.

## Why

When a skill declares `model_tier: fast`, Claude Code switches to a smaller model (Haiku) for that skill invocation. If the conversation has accumulated significant context, the smaller model's context window cannot hold it all, forcing a compaction. This degrades the user experience — the conversation loses context mid-flow, and subsequent interactions may lack awareness of earlier discussion.

This problem compounds for skills invoked mid-pipeline (like `fab-switch`, `fab-status`) where the conversation already contains loaded project context, change artifacts, and prior discussion. The cost savings from using Haiku for these lightweight skills are negligible compared to the disruption of forced compaction.

The `git-pr` skill was already migrated away from fast tier for the same reason (context limit hits late in pipeline sessions). This change extends that fix to all remaining fast-tier skills.

## What Changes

### Remove `model_tier: fast` from all skill frontmatter

Delete the `model_tier: fast` line from the YAML frontmatter of all five affected skills:

- `fab/.kit/skills/fab-switch.md` — line 4
- `fab/.kit/skills/fab-help.md` — line 4
- `fab/.kit/skills/fab-status.md` — line 4
- `fab/.kit/skills/fab-setup.md` — line 4
- `fab/.kit/skills/git-branch.md` — line 4

After removal, these skills become implicitly `capable` (the default when `model_tier` is absent).

### Remove `model_tiers` from `config.yaml`

With no skills using the fast tier, the `model_tiers:` section in `fab/project/config.yaml` becomes dead config. Remove it:

```yaml
# DELETE this block:
model_tiers:
  fast:
    claude: haiku
  capable:
    claude: null
```

### Remove model tier resolution from sync script

The `sync/2-sync-workspace.sh` script currently reads `model_tiers.fast.claude` from config and performs sed substitution (`model_tier: fast` → `model: {resolved}`) during skill deployment. With no fast-tier skills, this logic becomes dead code:

- Remove the tier resolution logic (reading config, sed substitution)
- All skills are now plain copies with no templating needed

### Remove `model_tiers` from scaffold config

The scaffold config at `fab/.kit/scaffold/config.yaml` includes `model_tiers:` for new projects. Remove it so new projects don't ship with unused configuration.

### Update memory and specs

The `docs/memory/fab-workflow/model-tiers.md` file documents the two-tier system extensively. It needs to be updated to reflect that the fast tier has been eliminated — all skills now run on the capable (default) tier.

## Affected Memory

- `fab-workflow/model-tiers`: (modify) Remove fast tier, update skill classification audit, update design decisions to note elimination of fast tier
- `fab-workflow/kit-architecture`: (modify) Remove any references to model tier deployment mechanics

## Impact

- **Skills**: 5 skill files modified (frontmatter only)
- **Config**: `fab/project/config.yaml` loses `model_tiers:` section
- **Scaffold**: `fab/.kit/scaffold/config.yaml` loses `model_tiers:` section
- **Sync script**: `sync/2-sync-workspace.sh` loses tier resolution logic
- **Memory**: 1-2 memory files updated
- **User impact**: All skills run on the session's current model — no more mid-conversation context switches or compaction

## Open Questions

- None — the precedent from `git-pr` and the clear user report make the direction unambiguous.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove fast tier from all 5 skills, not just fab-switch | User said "these commands" (plural) and the problem applies equally to all fast-tier skills invoked mid-conversation | S:80 R:90 A:85 D:90 |
| 2 | Certain | Remove `model_tiers` config section entirely | No fast-tier consumers remain; dead config violates simplicity | S:75 R:95 A:90 D:95 |
| 3 | Certain | Remove tier resolution from sync script | Dead code with no fast-tier skills to template | S:75 R:90 A:90 D:95 |
| 4 | Confident | Remove scaffold `model_tiers` too | New projects shouldn't ship with unused config, but user didn't mention scaffold explicitly | S:60 R:90 A:80 D:85 |
| 5 | Confident | The root cause is context window mismatch triggering compaction | User's hypothesis is plausible and matches the git-pr precedent; not independently verified | S:70 R:85 A:70 D:80 |

5 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).
