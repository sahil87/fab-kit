# Tasks: Fix git-pr-review Copilot reviewer login

**Change**: 260418-u1m1-copilot-reviewer-login
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

_(No setup required — string replacements only in existing files.)_

## Phase 2: Core Implementation

- [x] T001 [P] Replace the `--add-reviewer copilot` call at `src/kit/skills/git-pr-review.md:87` with `--add-reviewer copilot-pull-request-reviewer`.
- [x] T002 [P] Replace the poll filter at `src/kit/skills/git-pr-review.md:92` — change `.author.login == "copilot"` to `.author.login == "copilot-pull-request-reviewer"`.
- [x] T003 [P] In `docs/specs/skills/SPEC-git-pr-review.md` update the flow-diagram arrow (line ~46) `gh pr edit {n} --add-reviewer copilot` → `--add-reviewer copilot-pull-request-reviewer`.
- [x] T004 [P] In `docs/specs/skills/SPEC-git-pr-review.md` update the reviewer-table cell (line ~86) `Attempt \`gh pr edit --add-reviewer copilot\`` → `--add-reviewer copilot-pull-request-reviewer`.
- [x] T005 [P] In `docs/memory/fab-workflow/configuration.md` update the command example at line ~61 — change `gh pr edit --add-reviewer copilot` to `gh pr edit --add-reviewer copilot-pull-request-reviewer`. Leave the config-key name `review_tools.copilot` unchanged elsewhere in the bullet.
- [x] T006 [P] In `docs/memory/fab-workflow/execution-skills.md` update the Phase 2 narrative at line ~31 — change the `gh pr edit {number} --add-reviewer copilot` command reference to `--add-reviewer copilot-pull-request-reviewer`.
- [x] T007 [P] In `docs/memory/fab-workflow/execution-skills.md` update the decision-log entry at line ~473 — change the `gh pr edit {number} --add-reviewer copilot` command reference to `--add-reviewer copilot-pull-request-reviewer`.

## Phase 3: Integration & Edge Cases

- [x] T008 Verify completeness with `grep -rn "add-reviewer copilot\b" src/ docs/` — SHALL return zero matches (exit 1). If any match remains, fix it and re-run until clean.
- [x] T009 Verify consistency with `grep -rn "copilot-pull-request-reviewer" src/ docs/` — SHALL show the expected hits across the four files (at least 7 occurrences total: 2 in `git-pr-review.md`, 2 in `SPEC-git-pr-review.md`, 1 in `configuration.md`, 2 in `execution-skills.md`). Note: additional occurrences may appear in this change's own artifacts (`intake.md`, `spec.md`, `tasks.md`, `checklist.md`) — those are expected and not part of the count above.
- [x] T010 Re-read the modified Phase 2 block in `src/kit/skills/git-pr-review.md` end-to-end to confirm internal consistency — the add-reviewer call and the poll filter must both reference `copilot-pull-request-reviewer`.

## Phase 4: Polish

_(No polish tasks — documentation updates are part of Phase 2 since this change is documentation-heavy by nature.)_

---

## Execution Order

- T001-T007 are all `[P]` — they touch different files (or different, non-overlapping lines of the same file) and have no mutual dependencies. They can be applied in any order or in parallel.
- T008, T009, T010 are verification tasks — run after all Phase 2 tasks are complete.
- T008 gates correctness (no stray old references). T009 gates consistency (new references present where expected). T010 gates semantic correctness of the skill file itself.
