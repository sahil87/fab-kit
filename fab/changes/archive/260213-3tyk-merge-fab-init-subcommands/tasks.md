# Tasks: Merge fab-init Subcommands into Single Skill

**Change**: 260213-3tyk-merge-fab-init-subcommands
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Merge

- [x] T001 Rewrite `fab/.kit/skills/fab-init.md` — add subcommand routing (Arguments section classifying `config`, `constitution`, `validate` as subcommand keywords), then append Config Behavior section (full content from `fab/.kit/skills/fab-init-config.md`), Constitution Behavior section (full content from `fab/.kit/skills/fab-init-constitution.md`), and Validate Behavior section (full content from `fab/.kit/skills/fab-init-validate.md`). Update frontmatter description. Update bootstrap steps 1a/1b to reference internal sections instead of delegating to separate commands. Update Related Commands section and Next Steps references.

## Phase 2: Cleanup

- [x] T002 Delete variant skill files: `fab/.kit/skills/fab-init-config.md`, `fab/.kit/skills/fab-init-constitution.md`, `fab/.kit/skills/fab-init-validate.md`
- [x] T003 [P] Remove stale symlink directories `.claude/skills/fab-init-config/`, `.claude/skills/fab-init-constitution/`, `.claude/skills/fab-init-validate/`; agent files `.claude/agents/fab-init-config.md`, `.claude/agents/fab-init-constitution.md`, `.claude/agents/fab-init-validate.md`; multi-agent symlinks `.opencode/commands/fab-init-config.md`, `.opencode/commands/fab-init-constitution.md`, `.opencode/commands/fab-init-validate.md`; and `.agents/skills/fab-init-config/`, `.agents/skills/fab-init-constitution/`, `.agents/skills/fab-init-validate/`
- [x] T004 [P] Update cross-references in 6 doc files: replace `/fab-init-config` → `/fab-init config`, `/fab-init-constitution` → `/fab-init constitution`, `/fab-init-validate` → `/fab-init validate` across `fab/docs/fab-workflow/init.md` (6), `init-family.md` (8), `config-management.md` (11), `configuration.md` (3), `constitution-governance.md` (4), `index.md` (1) — 33 total occurrences

## Phase 3: Verification

- [x] T005 Run `fab/.kit/scripts/fab-setup.sh` and verify only `fab-init` symlink/agent is created for the init family (no variant entries). Confirm no broken symlinks.

---

## Execution Order

- T001 blocks T002 (don't delete source files until merge is verified)
- T003, T004 are independent of each other and can start after T002
- T005 runs last (verifies the final state)
