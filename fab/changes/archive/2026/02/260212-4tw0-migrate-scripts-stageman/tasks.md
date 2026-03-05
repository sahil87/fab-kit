# Tasks: Migrate Scripts to Use Stage Manager

**Change**: 260212-4tw0-migrate-scripts-stageman
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 Move `fab/.kit/schemas/MIGRATION.md` to `fab/changes/260212-4tw0-migrate-scripts-stageman/MIGRATION.md` (reference doc for implementation)

## Phase 2: Script Migration (Part A)

- [x] T002 [P] Migrate `fab/.kit/scripts/fab-status.sh` to use stageman.sh:
  - Source `stageman.sh` at top (after path resolution)
  - Replace hardcoded `for s in brief spec tasks apply review archive` loop (line 69) with `for s in $(get_all_stages)`
  - Replace hardcoded `case` stage-number mapping (lines 93-97) with `get_stage_number "$stage"`
  - Replace hardcoded `symbol()` function (lines 100-105) with `get_state_symbol`
  - Replace per-stage progress variables (`p_brief`, `p_spec`, etc., lines 60-65) with dynamic loop over `get_all_stages`
  - Replace hardcoded `/6` in stage display (line 153) with dynamic count from `get_all_stages`
  - Replace hardcoded progress_line calls (lines 156-161) with dynamic loop
  - Verify output is byte-identical before/after

- [x] T003 [P] Migrate `fab/.kit/scripts/fab-preflight.sh` to use stageman.sh:
  - Source `stageman.sh` at top (after path resolution)
  - Replace hardcoded progress extraction (lines 42-47) with dynamic loop over `get_all_stages`
  - Replace hardcoded stage derivation loop (line 58) with `get_current_stage "$status_file"` or equivalent dynamic loop
  - Add `validate_status_file "$status_file"` call after existence check (line 37), exit non-zero on failure
  - Replace hardcoded YAML output progress block (lines 86-92) with dynamic loop
  - Preserve migration shim for backward compatibility (lines 50-54)
  - Verify YAML output is identical before/after (except for new validation behavior)

## Phase 3: Documentation Deduplication (Part B)

- [x] T004 [P] Consolidate `src/stageman/` to single README.md:
  - Rewrite `src/stageman/README.md` with: overview, sources-of-truth links, API reference (content from SPEC.md), CLI interface, testing, changelog (content from CHANGELOG.md)
  - Delete `src/stageman/SUMMARY.md`
  - Delete `src/stageman/SPEC.md`
  - Delete `src/stageman/CHANGELOG.md`
  - Resulting directory: only `README.md`, `stageman.sh`, `test.sh`, `test-simple.sh`

- [x] T005 [P] Move and trim `fab/.kit/schemas/README.md` → `fab/docs/fab-workflow/schemas.md`:
  - Create `fab/docs/fab-workflow/schemas.md` with trimmed content: what workflow.yaml defines, design principles, how to reference from skills vs scripts, future enhancements
  - Remove stageman API function list (lines 48-70 of current README) and bash usage examples (lines 77-101)
  - Add link to `src/stageman/README.md` for API details
  - Delete `fab/.kit/schemas/README.md`
  - Result: `fab/.kit/schemas/` contains only `workflow.yaml`

- [x] T006 Fix all dangling references across codebase:
  - `README.md` (root, lines 118, 125-126): Update `SPEC.md` → `src/stageman/README.md`, `schemas/README.md` → `fab/docs/fab-workflow/schemas.md`
  - `fab/.kit/scripts/stageman.sh` (line 374): Update SEE ALSO paths to reflect new file locations
  - `fab/docs/fab-workflow/index.md`: Add `schemas` entry to the domain doc table
  - Verify with grep: `grep -r 'SPEC.md\|SUMMARY.md\|CHANGELOG.md\|schemas/README' --include='*.md' --include='*.sh'` returns no unexpected matches

## Phase 4: Validation

- [x] T007 Verify output parity for migrated scripts:
  - Run `fab-status.sh` and confirm output format is unchanged
  - Run `fab-preflight.sh` and confirm YAML output structure is unchanged
  - Run `stageman.sh --test` to confirm stageman functions still pass
  - Run `src/stageman/test-simple.sh` to confirm symlink-based tests pass

- [x] T008 Run broken link check:
  - `grep -r 'SPEC.md\|SUMMARY.md\|CHANGELOG.md\|schemas/README' --include='*.md' --include='*.sh'` from project root
  - Confirm no unexpected matches (only this change's own brief.md/spec.md are allowed)

---

## Execution Order

- T001 is a prerequisite (setup)
- T002 and T003 are independent [P] — can run in parallel
- T004 and T005 are independent [P] — can run in parallel
- T006 depends on T004 and T005 (needs to know final file locations)
- T007 depends on T002 and T003 (validates migrated scripts)
- T008 depends on T004, T005, and T006 (validates no broken links remain)
