# Tasks: Add fab-preflight.sh and update skills to consume it

**Change**: 260207-5mjv-preflight-grep-scripts
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Core Script

- [x] T001 Create `fab/.kit/scripts/fab-preflight.sh` — validation logic (config.yaml, constitution.md, fab/current, change dir, .status.yaml) with ordered checks and stderr error messages per spec. Exits non-zero on first failure.
- [x] T002 Add YAML output to `fab/.kit/scripts/fab-preflight.sh` — on successful validation, parse `.status.yaml` and emit structured YAML to stdout (name, change_dir relative to fab/, stage, branch, progress map, checklist block). Handle missing branch field (output empty string).

## Phase 2: _context.md Update

- [x] T003 Update `fab/.kit/skills/_context.md` Section 2 ("Change Context") — add preflight directive (run script via Bash, check exit code, parse stdout YAML). Keep existing 4-step inline sequence as documentation of what the script validates internally.
- [x] T004 Update `fab/.kit/skills/_context.md` Section 1 ("Always Load") — note that `fab-preflight.sh` covers the init check (config.yaml and constitution.md existence), so skills running preflight don't need separate existence checks.

## Phase 3: Skill Updates

- [x] T005 [P] Update pre-flight section in `fab/.kit/skills/fab-ff.md` — replace inline validation steps with preflight directive. Preserve stage-specific checks (proposal must be done) using preflight output fields.
- [x] T006 [P] Update pre-flight section in `fab/.kit/skills/fab-apply.md` — replace inline validation with preflight directive. Preserve stage-specific check (tasks must be done) using preflight progress field.
- [x] T007 [P] Update pre-flight section in `fab/.kit/skills/fab-review.md` — replace inline validation with preflight directive. Preserve stage-specific check (apply must be done) using preflight progress field.
- [x] T008 [P] Update pre-flight section in `fab/.kit/skills/fab-archive.md` — replace inline validation with preflight directive. Preserve stage-specific check (review must be done) using preflight progress field.
- [x] T009 [P] Update pre-flight section in `fab/.kit/skills/fab-continue.md` — replace inline validation with preflight directive. Preserve stage guard logic using preflight stage/progress fields.
- [x] T010 [P] Update pre-flight section in `fab/.kit/skills/fab-clarify.md` — replace inline validation with preflight directive. Remove redundant config/constitution existence checks. Preserve stage guard logic.

## Phase 4: Verification

- [x] T011 Run `fab/.kit/scripts/fab-preflight.sh` against the current active change and verify output matches the expected YAML format from the spec's example output scenario.
- [x] T012 Verify all 6 updated skill files have consistent preflight directive pattern — no remaining inline `fab/current` reads, no remaining inline `.status.yaml` existence checks, no remaining inline config/constitution existence checks in pre-flight sections.

---

## Execution Order

- T001 blocks T002 (validation before output)
- T001+T002 block T003, T004 (script must exist before documenting it)
- T003+T004 block T005–T010 (_context.md updated before skills reference it)
- T005–T010 are independent [P] (each skill is a separate file)
- T005–T010 block T011, T012 (verification after all changes)
