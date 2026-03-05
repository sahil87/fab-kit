# Tasks: Naming Linear ID & Drop Conventions

**Change**: 260215-w3n8-naming-linear-id-drop-conventions
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Config Updates

- [x] T001 Update `fab/config.yaml` naming section — change `format` value to `"{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}"`, update inline comments to document the optional `{ISSUE}` component (uppercase Linear issue ID), add examples showing both with and without Linear ID
- [x] T002 Delete `fab/config.yaml` conventions section — remove the entire commented-out `conventions:` block (comments and commented-out YAML, approximately lines 74-90)

## Phase 2: Skill Prompt Updates

- [x] T003 [P] Update `fab/.kit/skills/fab-new.md` Step 1 (Generate Folder Name) — change format string to `{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}`, document that when a Linear issue ID is parsed in Step 0, it is inserted as the uppercase `{ISSUE}` component between `{XXXX}` and `{slug}`
- [x] T004 [P] Update `fab/.kit/skills/fab-init.md` config template — change the `naming` section in the Create Mode template to use `format: "{YYMMDD}-{XXXX}-[{ISSUE}-]{slug}"`

---

## Execution Order

- T001 and T002 are independent (both edit `config.yaml` but different sections)
- T003 and T004 are independent of each other and of T001/T002 (different files)
