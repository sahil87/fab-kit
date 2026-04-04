# Tasks: Cascading PR Review Tools

**Change**: 260403-oh82-cascading-pr-review-tools
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Add `review_tools` config block to `fab/project/config.yaml` with all tools enabled (`copilot: true`, `codex: true`, `claude: true`)

## Phase 2: Core Implementation

- [x] T002 Add review request cascade logic to `src/kit/skills/git-pr-review.md` — new Step 2 Phase 2 that runs when no existing reviews with comments are found. Implement cascade order: Copilot → Codex → Claude, with config-based enable/disable checks <!-- clarified: removed --tool flag from T002 scope — T005 owns flag parsing and cascade bypass -->
- [x] T003 Add context enrichment section to `src/kit/skills/git-pr-review.md` — construct enriched prompt with diff, file list, test results (best-effort), and PR description for local tools (Codex, Claude)
- [x] T004 Add local review output posting to `src/kit/skills/git-pr-review.md` — post Codex/Claude output as PR comment via `gh api repos/{owner}/{repo}/issues/{number}/comments`, best-effort with terminal fallback

## Phase 3: Integration & Edge Cases

- [x] T005 Add `--tool` flag documentation and argument parsing to the skill frontmatter and behavior section in `src/kit/skills/git-pr-review.md` — validate tool name, skip cascade when flag is set
- [x] T006 Handle edge cases in `src/kit/skills/git-pr-review.md` — all tools disabled in config, all tools unavailable, `--tool` with invalid name, missing `review_tools` config key (default all true)
- [x] T007 Create migration file `src/kit/migrations/1.1.0-to-1.2.0.md` — idempotent migration to add `review_tools` block to existing `config.yaml` files

## Phase 4: Polish

- [x] T008 Update `docs/specs/skills/SPEC-git-pr-review.md` to reflect the cascade flow, config, and `--tool` flag

---

## Execution Order

- T001 is independent setup
- T002 is the core cascade logic; T003 and T004 depend on T002 (they reference cascade context)
- T005 and T006 depend on T002 (they extend the cascade behavior)
- T007 is independent of all other tasks
- T008 depends on T002-T006 being complete (documents final behavior)
