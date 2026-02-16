# Tasks: Consolidate Setup & Upgrade Flow

**Change**: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Script Relocation

- [x] T001 Move `fab/.kit/scripts/lib/sync-workspace.sh` → `fab/.kit/scripts/fab-sync.sh`. Update `SCRIPT_DIR`/path resolution: script is now one level up (`scripts/` not `scripts/lib/`), so `kit_dir` should be `$(dirname "$scripts_dir")` where `scripts_dir="$(cd "$(dirname "$0")" && pwd)"`. Update internal echo messages: `/fab-update` → `/fab-setup migrations` in VERSION creation output (line 95). Update header comment to reflect new path.
- [x] T002 Add stale artifact cleanup pass to `fab/.kit/scripts/fab-sync.sh`. After each `sync_agent_skills` call, scan target directories for entries not in the `skills[]` array and remove them. For Claude Code: remove `.claude/skills/{name}/` dirs where `{name}` is not in `skills[]`. For OpenCode: remove `.opencode/commands/{name}.md` files not in `skills[]`. For Codex: remove `.agents/skills/{name}/` dirs not in `skills[]`. For agent files: remove `.claude/agents/{name}.md` where `{name}` is not in `fast_skills[]` AND `{name}.md` exists in none of `.kit/skills/`. Preserve user-created agent files that don't match any skill name pattern.
- [x] T003 Rename test directory `src/lib/sync-workspace/` → `src/lib/fab-sync/`. Update symlink: `src/lib/fab-sync/fab-sync.sh` → `../../../fab/.kit/scripts/fab-sync.sh`. Update `SPEC-sync-workspace.md` → `SPEC-fab-sync.md` (update content: paths, script name, usage examples). Update `test.bats`: script reference variable (`SYNC_WORKSPACE` → `FAB_SYNC` or similar), header comment, any internal path references to `scripts/lib/sync-workspace.sh`. Add test(s) for stale cleanup behavior.

## Phase 2: Skill Swap

- [x] T004 Create `fab/.kit/skills/fab-setup.md`. Frontmatter: `name: fab-setup`, `description: "Set up a new project, manage config/constitution, or apply version migrations. Safe to re-run."`, `model_tier: fast`. Content: merge from `fab-init.md` (bootstrap, config, constitution behaviors) + `fab-update.md` (migrations behavior as new subcommand). Key changes from fab-init.md: (1) replace all `/fab-init` → `/fab-setup` in text; (2) remove Validate Behavior section entirely — add redirect message for `validate` argument; (3) update bootstrap to call `fab-sync.sh` instead of `lib/sync-workspace.sh`; (4) add context loading exceptions table per spec; (5) add `migrations [file]` subcommand section with full logic from fab-update.md (version comparison, discovery, application, failure handling, output formats); (6) update argument classification table; (7) update next steps references; (8) update unrecognized-argument message.
- [x] T005 [P] Delete `fab/.kit/skills/fab-init.md`
- [x] T006 [P] Delete `fab/.kit/skills/fab-update.md`

## Phase 3: Cross-Reference Updates

- [x] T007 [P] Update `fab/.kit/scripts/fab-upgrade.sh`: (1) header comment line 7: `lib/sync-workspace.sh` → `fab-sync.sh`; (2) line 96 echo: `lib/sync-workspace.sh` → `fab-sync.sh`; (3) line 97: `bash "$kit_dir/scripts/lib/sync-workspace.sh"` → `bash "$kit_dir/scripts/fab-sync.sh"`; (4) line 105: `/fab-update` → `/fab-setup migrations`; (5) line 109: `/fab-init` → `fab-sync.sh`, `/fab-update` → `/fab-setup`
- [x] T008 [P] Update `fab/.kit/scripts/fab-help.sh`: replace `/fab-init` entry with `/fab-setup` (description covering subcommands), remove `/fab-update` entry from Maintenance section, add `fab-sync.sh` as a script in Setup section, update description
- [x] T009 [P] Update `fab/.kit/scripts/lib/preflight.sh` line 15: `Run /fab-init first` → `Run /fab-setup first`. Update `fab/.kit/scripts/lib/changeman.sh` line 128 comment: `sync-workspace.sh` → `fab-sync.sh`
- [x] T010 [P] Update `fab/.kit/worktree-init-common/2-rerun-sync-workspace.sh` line 3: path `scripts/lib/sync-workspace.sh` → `scripts/fab-sync.sh`
- [x] T011 Update `fab/.kit/skills/_context.md`: (1) line 12 exception list: `/fab-init` → `/fab-setup`; (2) line 72 next steps table: replace `/fab-init` row with `/fab-setup` row (`initialized` → `/fab-new or /docs-hydrate-memory`)
- [x] T012 [P] Update skill files — `fab-new.md`: lines 15, 94 `/fab-init` → `/fab-setup`. `fab-status.md`: line 45 `/fab-update` → `/fab-setup migrations`. `fab-switch.md`: line 109 `/fab-init` → `/fab-setup`
- [x] T013 [P] Update skill files — `docs-hydrate-memory.md` line 28: `/fab-init` → `/fab-setup`. `docs-hydrate-specs.md` lines 28-29: `/fab-init` → `/fab-setup`. `docs-reorg-memory.md` line 101: `/fab-init` → `/fab-setup`. `docs-reorg-specs.md` line 101: `/fab-init` → `/fab-setup`

## Phase 4: Documentation & Config

- [x] T014 [P] Update `fab/.kit/model-tiers.yaml` line 4 comment: `lib/sync-workspace.sh` → `fab-sync.sh`. Update `fab/config.yaml` header comment line 8: `/fab:init` → `/fab-setup`
- [x] T015 Update `README.md`: (1) line 49: `scripts/lib/sync-workspace.sh` → `scripts/fab-sync.sh`; (2) line 56: `/fab-init` → `/fab-setup`; (3) line 119: `/fab-init` entry → `/fab-setup`; (4) line 149: `/fab-update` → `/fab-setup migrations`; (5) line 154: `scripts/lib/sync-workspace.sh` → `scripts/fab-sync.sh`; (6) update Getting Started flow to show two-entry-point architecture
- [x] T016 Run `fab/.kit/scripts/fab-sync.sh` to regenerate all symlinks and agent files. Verify: (1) `.claude/skills/fab-setup/` created with valid symlink; (2) `.claude/agents/fab-setup.md` generated; (3) `.claude/skills/fab-init/` removed (stale cleanup); (4) `.claude/skills/fab-update/` removed; (5) `.claude/agents/fab-init.md` removed; (6) OpenCode and Codex equivalents updated

---

## Execution Order

- T001 blocks T002 (stale cleanup requires the moved script)
- T001 blocks T003 (test infrastructure references the new path)
- T004 blocks T005, T006 (create new skill before deleting old ones)
- T005, T006 block T016 (skills must be deleted before sync can clean up stale artifacts)
- Phase 3 (T007-T013) can start after T001 (scripts reference new path)
- Phase 4 (T014-T016) can start after Phase 2 (skills swapped)
