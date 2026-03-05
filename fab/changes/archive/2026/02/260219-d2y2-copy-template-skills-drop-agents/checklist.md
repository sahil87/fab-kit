# Quality Checklist: Copy-Template Skills, Drop Agents

**Change**: 260219-d2y2-copy-template-skills-drop-agents
**Generated**: 2026-02-19
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Copy-with-template mode: Claude Code skills deployed as regular files (not symlinks) with `model_tier:` → `model:` substitution for fast-tier skills
- [x] CHK-002 Agent file generation removal: Section 4 deleted from `2-sync-workspace.sh`, no `.claude/agents/` files created during sync
- [x] CHK-003 Transitional agent cleanup: `.claude/agents/` files matching known skill names are removed on sync
- [x] CHK-004 Idempotency: repeated `2-sync-workspace.sh` runs count all skills as "already valid" with 0 repaired

## Behavioral Correctness

- [x] CHK-005 Fast-tier skills: deployed SKILL.md contains `model: haiku` (or configured model) — not `model_tier: fast`
- [x] CHK-006 Capable-tier skills: deployed SKILL.md is an exact verbatim copy of source — no `model:` line injected

## Removal Verification

- [x] CHK-007 Section 4 code: entire agent generation block removed from `2-sync-workspace.sh` — no "Agents:" output line
- [x] CHK-008 Dual deployment: no code path creates files in `.claude/agents/` during normal sync

## Scenario Coverage

- [x] CHK-009 Fast-tier skill deployed with model substitution (spec scenario 1)
- [x] CHK-010 Capable-tier skill deployed verbatim (spec scenario 2)
- [x] CHK-011 Config missing `model_tiers` section — falls back to `haiku` (spec scenario 3)
- [x] CHK-012 Stale agent files removed on first sync after upgrade (spec scenario 4)
- [x] CHK-013 User-created agent files preserved during cleanup (spec scenario 5)
- [x] CHK-014 No agents directory exists — no error during cleanup (spec scenario 6)

## Edge Cases & Error Handling

- [x] CHK-015 Missing `config.yaml` doesn't break model resolution — fallback to `haiku` works
- [x] CHK-016 Skills without frontmatter (internal `_*.md` files already excluded) don't cause errors in copy-with-template path

## Code Quality

- [x] CHK-017 Pattern consistency: new code follows naming and structural patterns of surrounding `2-sync-workspace.sh` code
- [x] CHK-018 No unnecessary duplication: reuses existing `yaml_value` function for model resolution
- [x] CHK-019 Readability: function comment updated to document dual-purpose 5th parameter
- [x] CHK-020 No god functions: modified `sync_agent_skills` is 80 lines (exceeds 50-line guideline but has clear structural reason — three deployment modes with idempotency)

## Documentation Accuracy

- [x] CHK-021 **N/A**: memory updates happen during hydrate phase (next step)
- [x] CHK-022 **N/A**: memory updates happen during hydrate phase (next step)

## Cross References

- [x] CHK-023 All sync output messages reflect copy mode (not symlink) for Claude Code
- [x] CHK-024 `sync_agent_skills` function comment and header comment reflect the changes

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
