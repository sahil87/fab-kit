# Tasks: Allow idea shorthand

**Change**: 260326-p4ki-allow-idea-shorthand
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add `RunE` and `Args: cobra.ArbitraryArgs` to root command in `src/go/idea/cmd/main.go` — delegate to `resolveFile()` + `idea.Add()` with empty `customID`/`customDate` when `len(args) > 0`, join args with space, validate non-empty after trim

## Phase 2: Tests

- [x] T002 Add test for bare shorthand in `src/go/idea/internal/idea/idea_test.go` or a new `src/go/idea/cmd/main_test.go` — verify `idea "text"` adds an idea, `idea ""` errors, and existing subcommands still work

## Phase 3: Documentation

- [x] T003 [P] Update `fab/.kit/skills/_cli-external.md` — add shorthand row to idea subcommand table: `(bare)` | `idea <text>` | `Shorthand for idea add <text>`
- [x] T004 [P] Update `docs/specs/packages.md` — add shorthand mention to idea commands table and/or common workflows section

---

## Execution Order

- T001 blocks T002
- T003 and T004 are independent, can run alongside T001-T002
