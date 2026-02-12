# Tasks: Fix created_by Format to Use GitHub ID

**Change**: 260212-0r8e-fix-created-by-github
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Implementation

- [x] T001 Update `created_by` instructions in `fab/.kit/skills/fab-new.md` — change Step 4 (Initialize `.status.yaml`) to use `gh api user --jq .login` as primary source with fallback chain: `gh` → `git config user.name` → `"unknown"`. Update the YAML template block (line ~133) and the explanation text (line ~155).

## Phase 2: Documentation

- [x] T002 Update `fab/docs/fab-workflow/planning-skills.md` — modify the `/fab-new` Change Initialization section (line ~52) to reflect the new `created_by` behavior: primary source is `gh api user --jq .login`, fallback to `git config user.name`, then `"unknown"`. Add a changelog entry for this change.

---

## Execution Order

- T001 blocks T002 (doc update should reflect final implementation)
