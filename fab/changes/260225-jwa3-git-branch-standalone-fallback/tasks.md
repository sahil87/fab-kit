# Tasks: git-branch Standalone Fallback

**Change**: 260225-jwa3-git-branch-standalone-fallback
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Modify `fab/.kit/skills/git-branch.md` Step 3 to add standalone fallback path — when `changeman.sh resolve` fails and an explicit argument was provided, use the raw argument as a literal branch name instead of stopping. Print feedback message: `No matching change found — creating standalone branch '{name}'`

- [x] T002 Modify `fab/.kit/skills/git-branch.md` Step 4 to skip `git.branch_prefix` when in standalone fallback mode — the literal name bypasses prefix application

- [x] T003 Modify `fab/.kit/skills/git-branch.md` Step 5 to handle existing standalone branches — check if the literal branch already exists locally and switch to it (`git checkout`) instead of failing on `git checkout -b`

## Phase 2: Documentation

- [x] T004 Update `fab/.kit/skills/git-branch.md` Error Handling table — add row for "Change resolution fails with explicit argument" → "Fall back to standalone branch with literal name"

---

## Execution Order

- T001 → T002 → T003 (sequential — each modifies the same file, building on previous changes)
- T004 depends on T001-T003 (documents final behavior)
