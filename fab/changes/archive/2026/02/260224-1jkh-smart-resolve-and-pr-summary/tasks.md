# Tasks: Smart Change Resolution & PR Summary Generation

**Change**: 260224-1jkh-smart-resolve-and-pr-summary
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add single-change guessing fallback to `cmd_resolve()` in `fab/.kit/scripts/lib/changeman.sh` — insert after the existing `fab/current` check fails (line ~128), enumerate non-archive folders with `.status.yaml`, return single candidate with stderr note `(resolved from single active change)`, exit 1 with `No active change.` for 0 candidates, exit 1 with `No active change (multiple changes exist — use /fab-switch).` for 2+ candidates
- [x] T002 [P] Update Step 3c in `.claude/skills/git-pr/SKILL.md` — add intake resolution logic before `gh pr create`: resolve active change via changeman, read `intake.md`, derive PR title from H1 (strip `Intake: ` prefix), generate body with Summary (from `## Why`), Changes (from `## What Changes` subheadings), Context (links to intake + optional spec), fall back to `gh pr create --fill` on failure

## Phase 2: Verification

- [x] T003 Verify changeman resolve guessing by running `bash fab/.kit/scripts/lib/changeman.sh resolve` with `fab/current` absent and a single active change present — confirm stdout and stderr output match spec scenarios

---

## Execution Order

- T001 and T002 are independent (`[P]`), can execute in parallel
- T003 depends on T001
