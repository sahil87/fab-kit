# Tasks: Expand fab-init Command Family

**Change**: 260212-h9k3-fab-init-family
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Skill Files

- [x] T001 [P] Create `fab/.kit/skills/fab-init-constitution.md` — dual-mode (create/update), semantic versioning, multiple amendments per session with highest-precedence bump, governance audit trail, structural preservation with re-numbering, idempotency. Follow existing skill frontmatter pattern (name, description, model_tier). Include all requirements and scenarios from spec section "Init Family: fab-init-constitution".
- [x] T002 [P] Create `fab/.kit/skills/fab-init-config.md` — section-based interactive updates with optional section argument, menu for 8 config sections, YAML validation after edit, comment/formatting preservation via string replacement, idempotency. Include all requirements and scenarios from spec section "Init Family: fab-init-config".
- [x] T003 [P] Create `fab/.kit/skills/fab-init-validate.md` — config structural validation (8 checks), constitution structural validation (6 checks), combined report, actionable fix suggestions, clear exit status. Include all requirements and scenarios from spec section "Init Family: fab-init-validate".

## Phase 2: Integration

- [x] T004 Update `fab/.kit/skills/fab-init.md` — (a) modify steps 1a and 1b to delegate to `/fab-init-config` and `/fab-init-constitution` respectively instead of containing independent generation logic; (b) add "Related Commands" section after Error Handling table listing the three new family commands with one-line descriptions.
- [x] T005 Run `fab/.kit/scripts/fab-setup.sh` to create symlinks for the three new skill files in `.claude/skills/`.

## Phase 3: Verification

- [x] T006 Verify all new skill files have correct frontmatter (name, description, model_tier), follow the `_context.md` preamble convention, and are structurally consistent with existing skills like `fab-init.md` and `fab-switch.md`.

---

## Execution Order

- T001, T002, T003 are independent — can execute in parallel
- T004 depends on T001, T002, T003 (needs to know the final skill names and descriptions for the Related Commands section)
- T005 depends on T001, T002, T003 (files must exist before fab-setup.sh can create symlinks)
- T006 depends on T001, T002, T003, T004
