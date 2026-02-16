# Quality Checklist: Consolidate Setup & Upgrade Flow

**Change**: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow
**Generated**: 2026-02-16
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Script relocation: `fab-sync.sh` exists at `fab/.kit/scripts/fab-sync.sh` and `lib/sync-workspace.sh` is gone
- [x] CHK-002 Path resolution: `fab-sync.sh` correctly resolves `kit_dir`, `fab_dir`, `repo_root` from its new location
- [x] CHK-003 Stale cleanup: `fab-sync.sh` removes orphaned skill symlinks, agent files, and copies for deleted skills
- [x] CHK-004 Skill creation: `fab/.kit/skills/fab-setup.md` exists with correct frontmatter (`name`, `description`, `model_tier: fast`)
- [x] CHK-005 Skill deletion: `fab/.kit/skills/fab-init.md` and `fab/.kit/skills/fab-update.md` are deleted
- [x] CHK-006 Migrations subcommand: `fab-setup.md` contains full migration logic (version comparison, discovery, sequential application, failure handling, `[file]` argument)
- [x] CHK-007 Context loading: `fab-setup.md` specifies per-subcommand context loading exceptions

## Behavioral Correctness

- [x] CHK-008 fab-sync.sh output: Identical functional behavior to old sync-workspace.sh (directories, VERSION, symlinks, agents, gitignore)
- [x] CHK-009 fab-setup.md bootstrap: Same flow as fab-init (pre-flight → sync → config → constitution)
- [x] CHK-010 fab-setup.md config/constitution: Behaviors preserved from fab-init (create mode, update mode, validation after edit)
- [x] CHK-011 Validate redirect: `/fab-setup validate` produces helpful redirect message (not an error)
- [x] CHK-012 Unrecognized args: `/fab-setup foobar` produces unknown subcommand message listing valid options

## Removal Verification

- [x] CHK-013 No `/fab-init` references: grep across `fab/.kit/` returns zero hits (excluding change artifacts)
- [x] CHK-014 No `/fab-update` references: grep across `fab/.kit/` returns zero hits (excluding change artifacts)
- [x] CHK-015 No `sync-workspace` references: grep across `fab/.kit/` returns zero hits (excluding change artifacts and `lib/` directory where the file was removed)
- [x] CHK-016 Stale symlinks gone: `.claude/skills/fab-init/`, `.claude/skills/fab-update/`, `.claude/agents/fab-init.md` removed after sync

## Scenario Coverage

- [x] CHK-017 Script relocation scenario: fab-sync.sh runs successfully from new location
- [x] CHK-018 Stale cleanup scenario: Running fab-sync.sh after skill deletion removes orphaned artifacts
- [x] CHK-019 Cross-reference completeness: All error messages, help text, and next-step suggestions reference new command names
- [x] CHK-020 Test suite passes: `src/lib/fab-sync/test.bats` passes against the relocated script

## Edge Cases & Error Handling

- [x] CHK-021 User-created agents preserved: fab-sync.sh stale cleanup does not remove `.claude/agents/` files that don't correspond to any `.kit/skills/*.md`
- [x] CHK-022 Preflight error updated: Missing config.yaml triggers `/fab-setup` guidance, not `/fab-init`
- [x] CHK-023 Upgrade drift message: fab-upgrade.sh references `/fab-setup migrations` for version drift

## Code Quality

- [x] CHK-024 Pattern consistency: New code in fab-sync.sh follows naming and structural patterns of existing sections
- [x] CHK-025 No unnecessary duplication: Stale cleanup logic reuses the existing `skills[]` array rather than re-scanning

## Documentation Accuracy

- [x] CHK-026 README.md: Getting Started section shows two-entry-point flow with correct commands
- [x] CHK-027 fab-help.sh: Command catalog shows `/fab-setup` and `fab-sync.sh`, not `/fab-init` or `/fab-update`

## Cross References

- [x] CHK-028 config.yaml header: References `/fab-setup`, not `/fab:init`
- [x] CHK-029 model-tiers.yaml: Comment references `fab-sync.sh`, not `sync-workspace.sh`
- [x] CHK-030 worktree-init-common: Calls `fab-sync.sh` at new path

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
