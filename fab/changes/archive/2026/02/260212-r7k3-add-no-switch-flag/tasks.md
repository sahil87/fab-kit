# Tasks: Add --no-switch Flag to fab-new

**Change**: 260212-r7k3-add-no-switch-flag
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Implementation

- [x] T001 Add `--no-switch` flag to the Arguments section of `fab/.kit/skills/fab-new.md` — document it as an optional flag that skips Step 8
- [x] T002 Add conditional logic to Step 8 in `fab/.kit/skills/fab-new.md` — when `--no-switch` is present, skip the internal `/fab-switch` invocation entirely (no `fab/current` write, no branch integration)
- [x] T003 Add `--no-switch` output example to the Output section of `fab/.kit/skills/fab-new.md` — show the variant without a `Branch:` line and with the contextual Next line: `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-ff`

---

## Execution Order

- T001 → T002 → T003 (sequential — each section builds on the previous context in the same file)
