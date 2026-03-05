# Tasks: Rename /fab-backfill to /fab-hydrate-design

**Change**: 260212-akhp-rename-fab-backfill
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Renames

<!-- File and directory moves. T001 must complete before T002 (symlink target). T003 is independent. -->

- [x] T001 Rename kit skill file `fab/.kit/skills/fab-backfill.md` → `fab/.kit/skills/fab-hydrate-design.md`. Update frontmatter `name: fab-backfill` → `name: fab-hydrate-design`, heading `# /fab-backfill` → `# /fab-hydrate-design`, and all `/fab-backfill` references in prose.
- [x] T002 Rename Claude skill directory `.claude/skills/fab-backfill/` → `.claude/skills/fab-hydrate-design/`. Recreate `SKILL.md` symlink to point to `../../../fab/.kit/skills/fab-hydrate-design.md`. Remove old directory.
- [x] T003 [P] Rename centralized doc `fab/docs/fab-workflow/backfill.md` → `fab/docs/fab-workflow/hydrate-design.md`. Update heading `# Backfill` → `# Hydrate Design`, all `/fab-backfill` command references → `/fab-hydrate-design`. Preserve change folder names (e.g. `260209-h3v7-fab-backfill`) as historical identifiers.

## Phase 2: Reference Updates

<!-- All tasks are parallelizable — each touches independent files. -->

- [x] T004 [P] Update centralized doc indexes and cross-cutting docs: `fab/docs/fab-workflow/index.md` (entry + link), `fab/docs/index.md` (domain doc list), `fab/docs/fab-workflow/model-tiers.md`, `fab/docs/fab-workflow/design-index.md`. Replace all `backfill`/`fab-backfill` command-name references with `hydrate-design`/`fab-hydrate-design`.
- [x] T005 [P] Update design spec references: `fab/design/glossary.md`, `fab/design/skills.md`, `fab/design/user-flow.md`. Replace all `/fab-backfill` command references with `/fab-hydrate-design`.
- [x] T006 [P] Update project root, scripts, and active change references: `README.md`, `fab/backlog.md`, `fab/.kit/scripts/fab-help.sh`, `fab/changes/260212-h9k3-fab-init-family/brief.md`. Replace all `/fab-backfill` command references with `/fab-hydrate-design`.

## Phase 3: Verification

- [x] T007 Run `grep -r "fab-backfill" --include="*.md" --include="*.sh" --include="*.yaml"` excluding `fab/changes/archive/` and this change's own artifacts. Verify zero command-name hits remain.

---

## Execution Order

- T001 blocks T002 (symlink target must exist)
- T003 is independent, can run alongside T001
- T004, T005, T006 are fully parallel, can start after Phase 1
- T007 runs last (verifies all prior work)
