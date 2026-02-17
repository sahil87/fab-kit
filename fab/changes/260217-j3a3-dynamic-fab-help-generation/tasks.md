# Tasks: Dynamic Fab Help Generation

**Change**: 260217-j3a3-dynamic-fab-help-generation
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/scripts/lib/frontmatter.sh` — extract `frontmatter_field()` from `fab/.kit/sync/3-sync-workspace.sh` into a new shared sourceable library (no shebang, no `set -euo pipefail`, just the function definition)

## Phase 2: Core Implementation

- [x] T002 Refactor `fab/.kit/sync/3-sync-workspace.sh` — replace the inline `frontmatter_field()` definition with `source "$kit_dir/scripts/lib/frontmatter.sh"`. Verify the source path resolves correctly from `$kit_dir` (which is `$(dirname "$sync_dir")`)
- [x] T003 Rewrite `fab/.kit/scripts/fab-help.sh` — replace the hardcoded `cat <<EOF` block with dynamic generation: source `frontmatter.sh`, scan `fab/.kit/skills/*.md`, exclude `_*` and `internal-*` prefixed files, extract `name` and `description` via `frontmatter_field()`, assign skills to groups via hardcoded mapping, compute dynamic column alignment, render version header + workflow diagram + grouped commands + "Typical Flow" footer. Include `fab-sync.sh` as a hardcoded entry in the "Setup" group. Unmapped skills go to "Other" group at end.

## Phase 3: Integration & Cleanup

- [x] T004 Delete `.claude/agents/fab-help.md` — remove the stale hand-authored agent file. Run `fab/.kit/scripts/fab-sync.sh` to regenerate the correct version via model-tier agent file logic (section 6 of `3-sync-workspace.sh`)
- [x] T005 Verify `fab/.kit/scripts/fab-help.sh` output — run the script and confirm: all 14 user-facing skills appear, no `_*`/`internal-*` files show, descriptions align, groups are correct, no "Other" group appears for current skills, `fab-sync.sh` appears in Setup

---

## Execution Order

- T001 blocks T002, T003 (both need the shared library)
- T002 and T003 are independent of each other after T001
- T004 depends on T003 (sync regenerates from the skill file, which fab-help.sh now reads)
- T005 depends on T003, T004
