# Brief: Remove Dead fab-help Agent File

**Change**: 260213-v8r3-remove-dead-fab-help-agent
**Created**: 2026-02-13
**Status**: Draft

## Origin

> Remove the dead fab-help agent file. The .claude/agents/fab-help.md agent is never spawned by any skill or pipeline -- it's dead code created by the model-tiers change. The skill at fab/.kit/skills/fab-help.md and the script at fab/.kit/scripts/fab-help.sh cover all actual usage. Delete the agent file and update any references.

## Why

The `.claude/agents/fab-help.md` agent file was created by the model-tiers change (260212-k8m3) to enable pipeline invocation via the Task tool. A grep for `subagent_type.*fab-help` across the entire codebase confirms no skill or agent ever spawns it. The skill + script pair (`fab/.kit/skills/fab-help.md` + `fab/.kit/scripts/fab-help.sh`) covers all actual usage. Keeping the dead agent file adds maintenance burden (it must be kept in sync with the skill) for zero benefit.

## What Changes

- **Delete** `.claude/agents/fab-help.md` -- the unused agent definition
- **Update** `fab/docs/fab-workflow/kit-architecture.md` -- remove the line referencing `.claude/agents/fab-help.md` from the directory tree listing

## Affected Docs

### New Docs

(none)

### Modified Docs

- `fab-workflow/kit-architecture`: Remove `.claude/agents/fab-help.md` from the directory tree listing (line 104)

### Removed Docs

(none)

## Impact

- **`.claude/agents/`** -- one fewer agent file; no behavioral change since nothing invoked it
- **No runtime impact** -- `/fab-help` continues to work via the skill path as before
- **Docs** -- kit-architecture.md tree listing becomes more accurate

## Open Questions

(none -- scope is narrow and fully confirmed by codebase analysis)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Only fab-help agent needs removal (not other fast-tier agents) | User's description specifically targets fab-help; other agents may have similar dead-code status but that's a separate change |
| 2 | Confident | kit-architecture.md is the only doc needing update | Grep for `agents/fab-help` in fab/docs/ found only one match; design/ had none |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.
