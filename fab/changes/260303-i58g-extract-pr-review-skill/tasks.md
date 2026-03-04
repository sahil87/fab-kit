# Tasks: Extract PR Review Skill

**Change**: 260303-i58g-extract-pr-review-skill
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create skill directory and scaffold `.claude/skills/git-review/SKILL.md` with frontmatter (name, description, allowed-tools)

## Phase 2: Core Implementation

- [x] T002 Implement PR Resolution section in `.claude/skills/git-review/SKILL.md` — detect PR via `gh pr view`, extract number/url/owner/repo, handle missing gh and no-PR cases
- [x] T003 Implement Review Detection and Routing section — check for existing reviews with comments via `GET /pulls/{number}/reviews` + `GET /pulls/{number}/comments`, route to comment processing if found; otherwise attempt Copilot request via POST, handle failure gracefully
- [x] T004 Implement Copilot Polling section — 30s interval, 12 attempts, capture review ID on arrival, timeout message on expiry
- [x] T005 Implement Comment Fetching section — Path A (all unresolved comments across reviewers) and Path B (specific Copilot review comments), skip resolved threads where possible
- [x] T006 Implement Comment Triage section — classify actionable vs informational, print triage summary, stop if all informational
- [x] T007 Implement Fix Application section — read file, understand issue, apply targeted fix per comment body and line reference
- [x] T008 Implement Commit and Push section — check for modifications, stage specific files, generate reviewer-aware commit message, push, handle failures with `git reset`

## Phase 3: Integration & Edge Cases

- [x] T009 Add Rules section to `.claude/skills/git-review/SKILL.md` — autonomous behavior, fail-fast, idempotent, targeted fixes only
- [x] T010 Remove Step 6 (Auto-Fix Copilot Review) from `.claude/skills/git-pr/SKILL.md` — delete Step 6 section, remove "Step 6 (Copilot fix) is best-effort" from Rules, remove any other Step 6 or `/git-pr-fix` references
- [x] T011 Delete `.claude/skills/git-pr-fix/SKILL.md` (the entire skill file)

---

## Execution Order

- T002–T008 are sequential within Phase 2 (each section builds on the prior in the skill file)
- T009 depends on T002–T008 (rules summarize the full behavior)
- T010 and T011 are independent of each other and of T009
