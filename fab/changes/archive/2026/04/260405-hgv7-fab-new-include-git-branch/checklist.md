# Quality Checklist: fab-new Include Git Branch

**Change**: 260405-hgv7-fab-new-include-git-branch
**Generated**: 2026-04-05
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Step 11 present: `src/kit/skills/fab-new.md` contains Step 11 after Step 10, with all 5 branching cases documented
- [ ] CHK-002 Non-fatal behavior: Step 11 skips with warning when not in a git repo; reports error but continues when git operation fails
- [ ] CHK-003 Output updated: `fab-new.md` Output section includes `Branch: {name} (...)` line after `Activated: {name}`
- [ ] CHK-004 Error table updated: Two new rows added — "Not in a git repo (Step 11)" and "git checkout / git branch failure (Step 11)"
- [ ] CHK-005 Frontmatter description updated: `description` field reads `"Start a new change — creates the intake, activates it, and creates the git branch."`
- [ ] CHK-006 No `allowed-tools` added: `fab-new.md` frontmatter has NO `allowed-tools: Bash(git:*)` (would break existing Bash steps)
- [ ] CHK-007 Constitution updated: `fab/project/constitution.md` Additional Constraints contains the `src/kit/` canonical source rule, placed after the existing `src/kit/skills/*.md` → `SPEC-*.md` rule

## Behavioral Correctness

- [ ] CHK-008 Branch name convention: Step 11 uses the change folder name as-is (no prefix) per `_naming.md`
- [ ] CHK-009 Upstream check: Step 11 checks `git config branch.{current}.remote` to distinguish local-only from pushed branches
- [ ] CHK-010 Already-active no-op: Step 11 reports `(already active)` without running any git command when already on the target branch

## Scenario Coverage

- [ ] CHK-011 main/master scenario: Step 11 runs `git checkout -b "{name}"` when on `main` or `master`
- [ ] CHK-012 Rename scenario: Step 11 runs `git branch -m "{name}"` when on a local-only branch
- [ ] CHK-013 Create-leaving-intact scenario: Step 11 runs `git checkout -b "{name}"` when on a pushed branch, leaving old branch intact
- [ ] CHK-014 Checkout scenario: Step 11 runs `git checkout "{name}"` when target branch exists but isn't current

## Edge Cases & Error Handling

- [ ] CHK-015 Not-in-git-repo: Step 11 handles `git rev-parse --is-inside-work-tree` failure gracefully, with warning message
- [ ] CHK-016 Git failure recovery: Error message instructs user to run `/git-branch` manually

## Spec and Memory Updates

- [ ] CHK-017 SPEC-fab-new.md flow: `docs/specs/skills/SPEC-fab-new.md` flow diagram includes Step 10 (Activate Change) and Step 11 (Create Git Branch)
- [ ] CHK-018 SPEC-fab-new.md tools: Tools table includes `Bash` rows for `git branch --show-current`, `git rev-parse`, and the git checkout/branch commands
- [ ] CHK-019 SPEC-fab-new.md summary: Summary description updated to mention activation and git branch creation
- [ ] CHK-020 planning-skills.md: Stale "never activates changes" text removed; git-branch step described; output format updated

## Documentation Accuracy

- [ ] CHK-021 Cross-references consistent: Constitution rule and context.md are consistent (context.md already mentions `src/kit/skills/*.md`; constitution extends to all kit content)
- [ ] CHK-022 No conflicting documentation: No other memory or spec file contradicts the updated fab-new behavior

## Code Quality

- [ ] CHK-023 Pattern consistency: Step 11 follows the same numbered-step format and code-block style as Steps 1–10 in `fab-new.md`
- [ ] CHK-024 No unnecessary duplication: Git logic in Step 11 references the `git-branch` skill as the source of truth (same logic, not a divergent copy)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`
