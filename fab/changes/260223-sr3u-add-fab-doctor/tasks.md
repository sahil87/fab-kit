# Tasks: Add fab-doctor.sh

**Change**: 260223-sr3u-add-fab-doctor
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/scripts/fab-doctor.sh` with shebang, `set -euo pipefail`, header comment, and failure counter variable. Make executable.

## Phase 2: Core Implementation

- [x] T002 Implement the 7 tool checks in `fab/.kit/scripts/fab-doctor.sh`: header output, per-tool `command -v` checks with version extraction (git, bash, yq, jq, gh, bats, direnv), success/failure formatting with `✓`/`✗`, fix hints, failure counter, and summary line. Include yq v4+ version gate and direnv shell hook detection via interactive subshell.
- [x] T003 Rewrite `fab/.kit/sync/1-prerequisites.sh` to delegate to `fab-doctor.sh` via `exec`

## Phase 3: Integration & Edge Cases

- [x] T004 Modify `fab/.kit/scripts/fab-upgrade.sh` version drift output — restructure lines 99-113 so "Update complete" prints first, then the `⚠` warning as the final line (or omit if versions match)
- [x] T005 Add doctor gate to `fab/.kit/skills/fab-setup.md` — insert a step before 1a that runs `fab/.kit/scripts/fab-doctor.sh` and stops bootstrap if it exits non-zero (bare invocation only, not subcommands)

---

## Execution Order

- T001 blocks T002
- T002 blocks T003
- T004 and T005 are independent of T001-T003 and each other
