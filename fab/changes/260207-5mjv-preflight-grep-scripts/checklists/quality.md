# Quality Checklist: Add fab-preflight.sh and update skills to consume it

**Change**: 260207-5mjv-preflight-grep-scripts
**Generated**: 2026-02-07
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Structured YAML Output: fab-preflight.sh outputs all specified fields (name, change_dir, stage, branch, progress, checklist) as valid YAML to stdout
- [x] CHK-002 Project Initialization Validation: script checks config.yaml and constitution.md exist, exits 1 with correct stderr message if missing
- [x] CHK-003 fab/current Validation: script checks fab/current exists and is non-empty (including whitespace-only), exits 1 with correct stderr message
- [x] CHK-004 Change Directory Validation: script checks change directory exists, exits 1 with correct stderr message including relative path
- [x] CHK-005 .status.yaml Validation: script checks .status.yaml exists in change dir, exits 1 with correct stderr message including change name
- [x] CHK-006 Validation Order: checks run in specified order (init → current → change dir → .status.yaml), stopping at first failure
- [x] CHK-007 _context.md Change Context: Section 2 references fab-preflight.sh with Bash execution, exit code check, and stdout parsing instructions
- [x] CHK-008 _context.md Always Load: Section 1 notes preflight covers init check, skills don't need separate existence checks
- [x] CHK-009 Skill Updates: all 6 skills (ff, apply, review, archive, continue, clarify) have preflight directive in pre-flight section
- [x] CHK-010 Skills Remove Redundant Checks: no remaining inline config/constitution existence checks in updated skills' pre-flight sections

## Behavioral Correctness

- [x] CHK-011 Relative Paths: change_dir in output is relative to fab/ (e.g., `changes/260207-...`), not absolute
- [x] CHK-012 Missing Branch Handling: when .status.yaml has no branch field, output includes `branch: ""`
- [x] CHK-013 Stage-specific Checks Preserved: each skill still validates its own stage preconditions (e.g., apply checks tasks done) using preflight output fields

## Scenario Coverage

- [x] CHK-014 Normal Active Change: preflight outputs complete YAML with exit 0 for a valid active change
- [x] CHK-015 Missing config.yaml: preflight exits 1 with init message
- [x] CHK-016 Missing fab/current: preflight exits 1 with "no active change" message
- [x] CHK-017 Missing change directory: preflight exits 1 with directory-not-found message
- [x] CHK-018 Missing .status.yaml: preflight exits 1 with corrupted message
- [x] CHK-019 Called from different working directory: preflight resolves paths correctly regardless of cwd

## Edge Cases & Error Handling

- [x] CHK-020 Whitespace-only fab/current: treated as empty, exits 1
- [x] CHK-021 Read-only / Idempotent: script modifies no files, produces identical output on repeated runs

## Documentation Accuracy

- [x] CHK-022 Existing _context.md inline steps preserved: 4-step sequence remains as documentation of what the script validates
- [x] CHK-023 Context loading sections untouched: skill artifact-loading sections (proposal.md, spec.md, etc.) not modified

## Cross References

- [x] CHK-024 Exempt skills not modified: init, switch, status, hydrate, help, new are unchanged
- [x] CHK-025 Consistent preflight pattern: all 6 updated skills use the same preflight directive structure

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab:archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
