# Tasks: Delegate fab-switch Name Resolution to Shell

**Change**: 260216-jmy4-DEV-1044-switch-shell-name-resolution
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Update Argument Flow and Context Loading in `fab/.kit/skills/fab-switch.md` — replace in-prompt matching instructions with shell delegation via `resolve-change.sh`. Update Context Loading to note `resolve-change.sh` dependency. Rewrite Argument Flow to: (1) call `source fab/.kit/scripts/lib/resolve-change.sh && resolve_change "fab" "<change-name>"` via Bash, (2) on exit 0 use `$RESOLVED_CHANGE_NAME`, (3) on exit 1 with "Multiple changes match" parse stderr for comma-separated names and present options, (4) on exit 1 with "No change matches" list all available changes. Keep No Argument Flow unchanged (LLM-driven listing).

- [x] T002 Mirror all body content changes to `.claude/agents/fab-switch.md` — copy the entire body (everything below YAML frontmatter) from the updated skill file. Preserve the agent-specific frontmatter (`name: fab-switch`, `model: haiku`).

## Phase 2: Verification

- [x] T003 Verify file sync — diff body content of `fab/.kit/skills/fab-switch.md` and `.claude/agents/fab-switch.md` (ignoring YAML frontmatter). Confirm identical content.

---

## Execution Order

- T001 blocks T002 (skill is the source of truth)
- T002 blocks T003 (both files must be updated before verification)
