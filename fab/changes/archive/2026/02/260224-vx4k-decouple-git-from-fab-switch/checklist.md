# Quality Checklist: Decouple Git from Fab Switch

**Change**: 260224-vx4k-decouple-git-from-fab-switch
**Generated**: 2026-02-24
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 fab-switch git-free: `/fab-switch` writes only `fab/current`, executes zero git commands
- [x] CHK-002 changeman.sh git-free: `cmd_switch()` contains no `git` commands, no config reading for `git.enabled`/`git.branch_prefix`
- [x] CHK-003 /git-branch create: Running `/git-branch` when branch doesn't exist creates and checks out `{prefix}{change-name}`
- [x] CHK-004 /git-branch checkout: Running `/git-branch` when branch exists checks it out
- [x] CHK-005 /git-branch already active: Running `/git-branch` when already on correct branch shows "already active"
- [x] CHK-006 /git-branch respects git.enabled: Reports disabled status and stops when `git.enabled: false`
- [x] CHK-007 /git-branch context prompts: On main → auto-create; on feature → prompt 3 options; on wt/* → prompt 3 options
- [x] CHK-008 /git-pr main guard enhanced: When on main with active change, suggests `/git-branch`
- [x] CHK-009 /git-pr mismatch nudge: When branch doesn't match active change, shows non-blocking note then proceeds

## Behavioral Correctness

- [x] CHK-010 fab-switch hint: Output includes `/git-branch` tip when `git.enabled: true`, omits when false
- [x] CHK-011 fab-switch output format: No `Branch:` line in changeman.sh switch output
- [x] CHK-012 /git-branch no fab mutation: `/git-branch` never writes `fab/current` or `.status.yaml`
- [x] CHK-013 /git-pr nudge match logic: Prefix-stripped branch compared against change name; exact or substring match = no nudge

## Removal Verification

- [x] CHK-014 --branch flag removed: `/fab-switch --branch` produces hard error (not in arguments list)
- [x] CHK-015 --no-branch-change flag removed: `/fab-switch --no-branch-change` produces hard error (not in arguments list)
- [x] CHK-016 --blank --branch removed: No reference to `--branch` in `--blank` flow
- [x] CHK-017 dispatch.sh cleaned: `--no-branch-change` no longer appears in dispatch.sh send-keys call (line 217)

## Scenario Coverage

- [x] CHK-018 Switch on main: `fab/current` written, git branch unchanged — changeman.sh has no git ops
- [x] CHK-019 Switch with no git repo: Works without error or git warning — no git commands to fail
- [x] CHK-020 /git-branch explicit change: Resolves change name via `changeman.sh resolve` when argument provided (Step 3)
- [x] CHK-021 /git-branch prefix: Branch name correctly includes `git.branch_prefix` (Step 4)
- [x] CHK-022 /git-pr no active change: No nudge shown when `fab/current` doesn't exist — "skip this step silently"

## Edge Cases & Error Handling

- [x] CHK-023 /git-branch checkout conflict: Reports git error on conflicting uncommitted changes, no fab state modified (Error Handling table)
- [x] CHK-024 /git-branch no git repo: Shows appropriate error, no crash (Step 2)
- [x] CHK-025 /git-pr no active change on main: Shows standard branch guard without /git-branch suggestion

## Code Quality

- [x] CHK-026 Pattern consistency: New skill file follows existing skill frontmatter (name, description, model_tier, allowed-tools) and section structure (Arguments, Behavior, Output, Error Handling, Key Properties)
- [x] CHK-027 No unnecessary duplication: Uses `changeman.sh resolve` for change name resolution, not reimplemented

## Documentation Accuracy

- [x] CHK-028 architecture.md updated: Git Integration section reflects `/git-branch` as the branch management command, `/fab-switch` no longer handles branches
- [x] CHK-029 skills.md updated: `/git-branch` skill entry added with purpose, example, behavior, and key properties

## Cross References

- [x] CHK-030 **N/A**: Memory files (`change-lifecycle.md`, `execution-skills.md`) will be updated at hydrate stage — not expected to be current yet

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
