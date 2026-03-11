# Quality Checklist: Fix Stale Shell-Script References After Go Binary Conversion

**Change**: 260311-i7it-fix-stale-shell-script-refs
**Generated**: 2026-03-11
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Accurate Package Architecture: `docs/specs/packages.md` describes wt as Go binary at `fab/.kit/bin/wt` with subcommands, idea as Go binary with shell fallback
- [ ] CHK-002 Accurate Packages Directory Structure: Only `idea/bin/idea` listed under `fab/.kit/packages/`, no `wt/` directory
- [ ] CHK-003 Correct Worktree Naming Source: naming.md Worktree section references `wt create` (`fab/.kit/bin/wt`)
- [ ] CHK-004 Correct Backlog Entry Encoding Source: naming.md Backlog Entry section references `idea` (`fab/.kit/bin/idea`)
- [ ] CHK-005 Remove Obsolete Memory File: `docs/memory/fab-workflow/kit-scripts.md` deleted
- [ ] CHK-006 Update Memory Index: `kit-scripts` entry removed from `docs/memory/fab-workflow/index.md`
- [ ] CHK-007 Cross-Reference in kit-architecture.md: Overview section references `_scripts.md` as canonical CLI reference
- [ ] CHK-008 Remove Stale lib/ Sections: No statusman.sh, logman.sh, calc-score.sh, changeman.sh, archiveman.sh subsections in kit-architecture.md
- [ ] CHK-009 Clarify env-packages.sh: Description notes wt is a binary in `$KIT_DIR/bin/`, packages iteration picks up only shell packages

## Behavioral Correctness
- [ ] CHK-010 Packages.md wt commands: Lists `wt create`, `wt list`, `wt open`, `wt delete`, `wt init` — NOT `wt-create`, `wt-list`, etc.
- [ ] CHK-011 Packages.md wt pr dropped: No reference to `wt pr` or `wt-pr`; notes replacement by `/git-pr`
- [ ] CHK-012 Packages.md PATH setup: Correctly describes `env-packages.sh` adding `$KIT_DIR/bin` first, then iterating `packages/*/bin`

## Removal Verification
- [ ] CHK-013 No shell script references for wt: No mention of `wt-create`, `wt-list`, `wt-open`, `wt-delete`, `wt-init`, `wt-pr` as executables across all changed files
- [ ] CHK-014 No `packages/wt/` references: No mention of `fab/.kit/packages/wt/` across all changed files
- [ ] CHK-015 No `lib/wt-common.sh` references: No mention of `lib/wt-common.sh` across all changed files
- [ ] CHK-016 No stale lib/ script sections in kit-architecture.md: Sections for deleted scripts fully removed, not just commented out

## Scenario Coverage
- [ ] CHK-017 Agent reads packages.md for wt: Gets Go binary architecture, not shell scripts
- [ ] CHK-018 Agent reads naming.md: Gets correct encoding locations for worktree and backlog entry conventions
- [ ] CHK-019 Agent reads kit-architecture.md: Finds cross-reference to _scripts.md, no stale lib/ sections

## Code Quality
- [ ] CHK-020 Pattern consistency: Rewritten content follows existing markdown style and structure of surrounding documentation
- [ ] CHK-021 No unnecessary duplication: Cross-references used where content already exists in _scripts.md or kit-architecture.md Go Binary section

## Documentation Accuracy
- [ ] CHK-022 All file paths in packages.md are verifiable against actual directory structure
- [ ] CHK-023 Command signatures in packages.md match actual Go binary behavior

## Cross References
- [ ] CHK-024 kit-architecture.md replacement note references _scripts.md correctly
- [ ] CHK-025 Memory index accurately reflects current file listing after kit-scripts.md deletion

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
