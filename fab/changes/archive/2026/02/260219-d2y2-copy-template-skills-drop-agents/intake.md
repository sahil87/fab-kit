# Intake: Copy-Template Skills, Drop Agents

**Change**: 260219-d2y2-copy-template-skills-drop-agents
**Created**: 2026-02-19
**Status**: Draft

## Origin

> Replace skill symlinks with copy-with-template (like Codex) so model_tier gets resolved to model: in SKILL.md, and stop generating .claude/agents/ files for fast-tier skills

User-initiated after investigating why `.claude/agents/` contained four files (`fab-help.md`, `fab-setup.md`, `fab-status.md`, `fab-switch.md`) that were near-identical duplicates of the corresponding `.claude/skills/` symlinks. The only difference: agents had `model: haiku` while skills had `model_tier: fast`. Investigation confirmed Claude Code skills now support `model:` in frontmatter, invalidating the original rationale for the dual deployment strategy.

## Why

The current "Dual Deployment" design (documented in `docs/memory/fab-workflow/model-tiers.md`) was built on the premise that Claude Code skills don't support `model:` in frontmatter. That premise is no longer true — skills support `model:` natively.

This creates three problems:

1. **Double registration**: Fast-tier skills appear both as skills (via `/fab-status`) and as Task subagent types (via `.claude/agents/`). They compete for invocation and confuse the tool dispatch.
2. **Model override doesn't reach skills**: The symlinked SKILL.md files contain `model_tier: fast` which Claude Code ignores — so user-invoked skills always run on the default model, not the intended fast-tier model.
3. **Unnecessary complexity**: `2-sync-workspace.sh` has ~70 lines of agent generation + cleanup logic that can be deleted entirely.

## What Changes

### 1. Switch Claude Code skills from symlinks to copies with model templating

In `fab/.kit/sync/2-sync-workspace.sh`, change the Claude Code skill sync call from:

```bash
# Current: symlinks
sync_agent_skills "Claude Code" "$repo_root/.claude/skills" "directory" "symlink" "../../../"
```

To copy mode with model templating — same approach already used for Codex (line 383):

```bash
# New: copies with model_tier → model: substitution
sync_agent_skills "Claude Code" "$repo_root/.claude/skills" "directory" "copy"
```

But unlike Codex (which copies verbatim), Claude Code copies need the `model_tier:` → `model:` substitution applied. The `sync_agent_skills` function's copy path currently does a plain `cp`. It needs to:

1. For skills with `model_tier: fast` — resolve the model name from `config.yaml` (or fallback `haiku`) and `sed` the frontmatter: `model_tier: fast` → `model: haiku`
2. For skills without `model_tier` — copy verbatim (no `model:` line injected; capable skills use the platform default)

### 2. Remove agent file generation (Section 4)

Delete the entire "Section 4: Model tier agent files" block in `2-sync-workspace.sh` (lines ~386-458). This removes:
- The loop that generates `.claude/agents/<name>.md` files
- The stale agent cleanup logic
- The agent count output line

### 3. Update `sync_agent_skills` to support templated copies

The function needs a new mode or enhancement to the existing `copy` mode. Options:

- **Option A**: Add a `--template` flag / new mode `"copy-template"` that applies the sed substitution during copy
- **Option B**: Keep the `copy` mode but add model resolution logic that checks each file for `model_tier:` and substitutes if config provides a mapping

The simplest approach: after the `cp` in copy mode, apply the same `sed` substitution that Section 4 currently does, but only for skills that have a `model_tier:` field. This is already computed in `fast_skills[]`.

### 4. Update memory documentation

Update `docs/memory/fab-workflow/model-tiers.md`:
- **Dual Deployment design decision**: Mark as superseded — skills now support `model:`, so copy-with-template replaces the dual strategy
- **Deployment section**: Document the new single-deployment approach (copy-with-template for all platforms)

## Affected Memory

- `fab-workflow/model-tiers`: (modify) Update Dual Deployment design decision to superseded; update deployment section to reflect copy-with-template
- `fab-workflow/distribution`: (modify) Update references from "symlinks" to "copies" for Claude Code skills; update `fab-sync.sh` description

## Impact

- `fab/.kit/sync/2-sync-workspace.sh` — primary change target (sync function + section 4 removal)
- `.claude/skills/*/SKILL.md` — will become regular files instead of symlinks after next `fab-sync` run
- `.claude/agents/fab-{help,setup,status,switch}.md` — will be deleted (stale cleanup or manual removal)
- Downstream projects using `fab/.kit/` — will get the new behavior on next `fab-upgrade.sh` run; their `.claude/agents/` files will be cleaned up by the stale agent removal logic during the transition, then that logic itself gets removed

## Open Questions

- Should the stale agent cleanup logic be kept for one release cycle as a migration aid (clean up agents in downstream projects that still have them), or can we remove it immediately since `fab-sync` already runs on every `direnv allow`?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Claude Code skills support `model:` frontmatter | Confirmed via Claude Code documentation — `model` is a supported optional field in SKILL.md frontmatter | S:95 R:90 A:95 D:95 |
| 2 | Certain | Copy mode already exists in `sync_agent_skills` | Function at line 292-300 handles `mode=copy` for Codex | S:95 R:95 A:95 D:95 |
| 3 | Confident | `model_tier: fast` sed to `model: haiku` is sufficient | Same substitution Section 4 already does; just moving it into the copy path | S:85 R:85 A:80 D:80 |
| 4 | Confident | Capable skills need no `model:` line | Omission = platform default, which is the intended behavior for capable tier | S:80 R:90 A:85 D:85 |
| 5 | Tentative | Stale agent cleanup can be removed immediately | `fab-sync` runs on every `direnv allow`, so downstream projects will get cleanup on next sync — but projects that haven't upgraded yet will retain stale agents until they do | S:60 R:70 A:55 D:50 |

5 assumptions (2 certain, 2 confident, 1 tentative). Run /fab-clarify to review.
