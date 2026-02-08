# Quality Checklist: Move Branch Integration from fab-new to fab-switch

**Change**: 260208-q8v3-branch-to-switch
**Generated**: 2026-02-09
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 fab-switch branch integration: fab-switch.md contains branch integration step with main/master auto-create, feature branch options, wt/* default, --branch flag
- [x] CHK-002 fab-new branch removal: fab-new.md no longer contains Step 4 (Git Integration) or --branch argument
- [x] CHK-003 fab-new internal switch call: fab-new.md includes Step 8 to call /fab-switch internally after proposal generation
- [x] CHK-004 fab-status live branch: fab-status.sh uses `git branch --show-current` instead of .status.yaml branch field
- [x] CHK-005 preflight branch removal: fab-preflight.sh no longer parses or emits `branch:` in YAML output
- [x] CHK-006 template branch removal: status.yaml template no longer contains `branch:` line

## Behavioral Correctness
- [x] CHK-007 fab-switch context loading: fab-switch.md context loading section now includes config.yaml for git settings
- [x] CHK-008 fab-status git-disabled behavior: fab-status.sh omits Branch line when git disabled or not in a repo (verified via show_branch flag)
- [x] CHK-009 branch_prefix in config: config.yaml retains `branch_prefix: ""` under `git:` section

## Removal Verification
- [x] CHK-010 No branch in .status.yaml schema: No skill writes `branch:` to .status.yaml — verified via grep
- [x] CHK-011 No --branch on fab-new: fab-new.md argument section does not mention --branch — verified via grep
- [x] CHK-012 No branch in preflight output: fab-preflight.sh stdout contains no branch line — verified via live run

## Scenario Coverage
- [x] CHK-013 Switch on main auto-creates branch: fab-switch.md Branch Integration section describes auto-create behavior on main/master
- [x] CHK-014 Switch on feature branch prompts: fab-switch.md describes Adopt/Create/Skip options with Adopt as default
- [x] CHK-015 Switch on wt/* defaults to Create: fab-switch.md specifies wt/* branch section with Create new branch as default
- [x] CHK-016 --branch flag skips prompt: fab-switch.md describes --branch bypassing interactive flow
- [x] CHK-017 Git disabled skips branch: fab-switch.md Branch Integration section starts with "Skip this step entirely if git.enabled is false"

## Edge Cases & Error Handling
- [x] CHK-018 Branch creation failure: fab-switch.md error handling table includes "Git branch creation fails" — switch still completes
- [x] CHK-019 Archived changes with branch field: Existing archived .status.yaml files are unaffected — no migration, no skill modifies archived files

## Documentation Accuracy
- [x] CHK-020 _context.md preflight fields: branch removed from preflight YAML fields list — now lists name, change_dir, stage, progress, checklist, confidence
- [x] CHK-021 _context.md autonomy table: fab-new interruption budget now says "Max 3 for unresolved questions" — verified via grep
- [x] CHK-022 fab-discuss branch note: fab-discuss.md no longer references branch field omission — verified via grep

## Cross References
- [x] CHK-023 All skill files: No remaining references to `branch` in preflight YAML parsing — verified via grep across all skill files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
