# Quality Checklist: Operator 7 — Dependency-Aware Agent Spawning

**Change**: 260324-prtv-operator7-dep-aware-spawning
**Generated**: 2026-03-24
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 operator7 skill file: `fab/.kit/skills/fab-operator7.md` exists with all operator6 content plus additions
- [x] CHK-002 Schema additions: `.fab-operator.yaml` example in §4 includes `depends_on`, `branch`, and `branch_map`
- [x] CHK-003 Pre-spawn dependency resolution: §6 "Spawning an Agent" has 4-step sequence with resolve dependencies step
- [x] CHK-004 Cherry-pick command: uses `git cherry-pick --no-commit origin/main..<dep-branch> && git commit`
- [x] CHK-005 Conflict handling: abort, log, escalate, do not spawn — documented in §6
- [x] CHK-006 Branch lookup: documented from monitored entry `branch` field and `branch_map`
- [x] CHK-007 Redundant dep pruning: `git merge-base --is-ancestor` documented
- [x] CHK-008 Already-present check: `git merge-base --is-ancestor <dep-branch> HEAD` documented
- [x] CHK-009 Working a Change flows: dependency resolution inserted as step 3 in structured flow
- [x] CHK-010 Autopilot dispatch: dependency resolution inserted as step 2
- [x] CHK-011 --base implies depends_on: documented in autopilot section
- [x] CHK-012 Three declaration paths: explicit, queue, --base all documented
- [x] CHK-013 Bounded retries: cherry-pick conflict row added to §3 table
- [x] CHK-014 Idle message timestamp: `Time: HH:MM · next tick: HH:MM` format in §4
- [x] CHK-015 Launcher script: `fab-operator7.sh` exists, references `/fab-operator7`
- [x] CHK-016 Deployed copy: `.claude/skills/fab-operator7.md` exists after sync

## Behavioral Correctness
- [x] CHK-017 operator6 untouched: `fab/.kit/skills/fab-operator6.md` is NOT modified
- [x] CHK-018 operator7 frontmatter: name and description reference operator7, not operator6
- [x] CHK-019 Launcher references: `fab-operator7.sh` invokes `/fab-operator7`, not `/fab-operator6`

## Scenario Coverage
- [x] CHK-020 Agent spawned with dependency: spec scenario covered in §6 documentation
- [x] CHK-021 Agent spawned without dependencies: no-op behavior documented
- [x] CHK-022 Chain with transitive deps: pruning behavior documented
- [x] CHK-023 Missing dependency branch: escalation behavior documented
- [x] CHK-024 Conflict during cherry-pick: abort + escalation documented

## Code Quality
- [x] CHK-025 Pattern consistency: operator7 follows same structural patterns as operator6 (section numbering, formatting, YAML examples)
- [x] CHK-026 No unnecessary duplication: new content is additive, not duplicating existing operator6 sections

## Documentation Accuracy
- [x] CHK-027 All YAML examples are syntactically valid
- [x] CHK-028 Git commands in documentation are correct (`cherry-pick --no-commit`, `merge-base --is-ancestor`)

## Cross References
- [x] CHK-029 §6 dependency resolution references §3 bounded retries for conflict handling
- [x] CHK-030 §6 dependency resolution references §4 `.fab-operator.yaml` schema for field definitions

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
