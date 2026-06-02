# Spec: Fix git-pr-review Copilot reviewer login

**Change**: 260418-u1m1-copilot-reviewer-login
**Created**: 2026-04-18
**Affected memory**: `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Renaming the internal config key `review_tools.copilot` — it is a fab-kit-local identifier, not a GitHub login. Renaming would require a migration and break existing user configs.
- Changing the `_review.md` outward sub-agent cascade — this spec covers only `/git-pr-review` Phase 2 (post-PR) and its documentation.
- Adding new reviewer tools or configuration options — scope is a literal string fix only.
- Modifying the deployed copy of the skill at `.claude/skills/git-pr-review/SKILL.md` — it regenerates from `src/` via `fab sync` at the user's next invocation.

## Skill: git-pr-review

### Requirement: Correct Copilot reviewer login in add-reviewer call

The `/git-pr-review` skill SHALL use `copilot-pull-request-reviewer` (not `copilot`) as the reviewer identifier when invoking `gh pr edit --add-reviewer` in Phase 2. This matches GitHub's canonical login for the Copilot Pull Request Reviewer bot.

#### Scenario: Phase 2 runs and request succeeds

- **GIVEN** a PR has no existing reviews with inline comments
- **AND** `review_tools.copilot` is `true` (or absent — defaults to true)
- **WHEN** `/git-pr-review` Phase 2 executes the add-reviewer call
- **THEN** the invoked command SHALL be `gh pr edit {number} --add-reviewer copilot-pull-request-reviewer`
- **AND** on exit 0, the skill SHALL print `Copilot review requested. Waiting up to 10 minutes...`
- **AND** the skill SHALL proceed to the polling loop

#### Scenario: Phase 2 runs and request fails

- **GIVEN** a PR has no existing reviews with inline comments
- **AND** the add-reviewer call is invoked with the corrected login
- **WHEN** the `gh pr edit` command returns a non-zero exit for any reason (e.g., permission denied, app not installed in the repo)
- **THEN** the skill SHALL print `No automated reviewer available. Run /git-pr-review when reviews are added.` and STOP (clean finish)

### Requirement: Poll check uses matching reviewer login

The `/git-pr-review` Phase 2 polling loop SHALL filter reviews by `author.login == "copilot-pull-request-reviewer"` so that it matches the author login of reviews posted by the bot that was just tagged. The `--add-reviewer` identity and the poll filter are two halves of the same handshake — they MUST be identical.

#### Scenario: Polling detects a Copilot review

- **GIVEN** Phase 2 successfully requested Copilot as a reviewer using `copilot-pull-request-reviewer`
- **AND** the Copilot bot has submitted a review on the PR
- **WHEN** the polling loop queries `gh pr view {number} --json reviews -q '.reviews | map(select(.author.login == "copilot-pull-request-reviewer")) | length'`
- **THEN** the returned count SHALL be greater than 0
- **AND** the skill SHALL proceed to Step 3 (Fetch Comments)

#### Scenario: Polling exhausts all attempts

- **GIVEN** Phase 2 successfully requested Copilot as a reviewer
- **AND** no Copilot-authored review appears on the PR
- **WHEN** the polling loop completes 20 attempts at 30-second intervals (10 minutes elapsed)
- **THEN** the skill SHALL print `Copilot review requested but not yet available. Re-run /git-pr-review to process when ready.` and STOP (clean finish)

### Requirement: Config key `review_tools.copilot` retains its name

The fab-kit config key `review_tools.copilot` SHALL remain named `copilot`. It is an internal toggle that enables/disables Phase 2's attempt to request a Copilot review — it is not a GitHub login and does not need to match the bot identifier.

#### Scenario: Config check with copilot disabled

- **GIVEN** `fab/project/config.yaml` contains `review_tools.copilot: false`
- **AND** no `--tool copilot` flag was provided
- **WHEN** `/git-pr-review` reaches Phase 2
- **THEN** the skill SHALL print `No automated reviewer available. Run /git-pr-review when reviews are added.` and STOP (clean finish) — without attempting the add-reviewer call

#### Scenario: Config key absent

- **GIVEN** `fab/project/config.yaml` omits the `review_tools` block entirely
- **WHEN** `/git-pr-review` reaches Phase 2
- **THEN** Copilot SHALL default to enabled and the add-reviewer call SHALL run with the corrected login

## Documentation: spec and memory alignment

### Requirement: SPEC-git-pr-review reflects corrected login

`docs/specs/skills/SPEC-git-pr-review.md` SHALL show the corrected `copilot-pull-request-reviewer` login in every reference to the `gh pr edit --add-reviewer` command, including the flow diagram and the reviewer table.

#### Scenario: Reader consults the spec

- **GIVEN** a developer reads `docs/specs/skills/SPEC-git-pr-review.md` to understand Phase 2 behavior
- **WHEN** the reader scans the flow diagram (line ~46) and the reviewer table (line ~86)
- **THEN** both SHALL show `gh pr edit --add-reviewer copilot-pull-request-reviewer` (not `copilot`)

### Requirement: Memory files reflect corrected login

`docs/memory/fab-workflow/configuration.md` and `docs/memory/fab-workflow/execution-skills.md` SHALL show `copilot-pull-request-reviewer` in every narrative mention of the `gh pr edit --add-reviewer` command. The internal config key `review_tools.copilot` SHALL remain spelled as `copilot` in these files.

#### Scenario: Reader consults memory for Phase 2 behavior

- **GIVEN** a developer or agent reads `docs/memory/fab-workflow/execution-skills.md` line 31 (Phase 2 narrative) or line 473 (decision log entry)
- **WHEN** they encounter the `gh pr edit {number} --add-reviewer ...` command
- **THEN** the command SHALL use `copilot-pull-request-reviewer`

#### Scenario: Reader consults configuration.md

- **GIVEN** a developer reads `docs/memory/fab-workflow/configuration.md` line ~61 (the `copilot` bullet)
- **WHEN** they encounter the embedded command example
- **THEN** the command SHALL use `gh pr edit --add-reviewer copilot-pull-request-reviewer`
- **AND** the surrounding prose describing the config key SHALL still reference the `copilot` key by its local name

### Requirement: No stray `copilot` reviewer references remain

After the change is applied, a repository-wide grep for `add-reviewer copilot\b` (word-boundary anchored — excludes `copilot-pull-request-reviewer`) SHALL return zero matches across `src/` and `docs/`.

#### Scenario: Post-change verification grep

- **GIVEN** all four files have been edited
- **WHEN** the verifier runs `grep -rn "add-reviewer copilot\b" src/ docs/`
- **THEN** the command SHALL return no matches (exit 1 from grep)

## Design Decisions

1. **Decision**: Change the poll filter's `author.login` check alongside the add-reviewer call.
   - *Why*: The two references (`--add-reviewer {login}` and `.author.login == {login}`) must refer to the same GitHub identity — otherwise Phase 2 tags the bot correctly but never recognizes the bot's review when it arrives, causing the poll to time out unnecessarily. The user's phrasing `"every ... invocation (or equivalent)"` covers the poll check by intent.
   - *Rejected*: Changing only the add-reviewer call — leaves Phase 2 functionally broken in the polling path even if the tag succeeds.

2. **Decision**: Keep the internal config key `review_tools.copilot` unchanged.
   - *Why*: It is a fab-kit-local config identifier. Renaming it would require a version-bumped migration file, user config rewrites, and would break existing installations that have not upgraded. The user explicitly said "No config or schema changes."
   - *Rejected*: Renaming to `review_tools.copilot-pull-request-reviewer` for consistency — too invasive for a bug fix; surface area change is disproportionate to the actual defect.

3. **Decision**: Do not edit the deployed skill copy at `.claude/skills/git-pr-review/SKILL.md`.
   - *Why*: The deployed copy is a build artifact regenerated by `fab sync`. Editing it would be wasted work — the next sync would stomp any direct edit. Users pick up fixed behavior on their next sync, which the PR-creation hook triggers automatically in fab-kit's own workflow.
   - *Rejected*: Also editing the deployed copy for immediate consistency — creates a two-place edit and implies to readers that the deployed copy is authoritative.

4. **Decision**: Verification is by post-edit grep, not by a new automated test.
   - *Why*: The skill is a Markdown file with instructions for the agent; it has no existing test harness. Adding one just for string matching is disproportionate. A grep check is objective, fast, and sufficient to confirm completeness.
   - *Rejected*: Adding a unit test — there is no place in this project where markdown string contents are validated by test code, and a ad-hoc test would rot.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replacement reviewer login is `copilot-pull-request-reviewer` | Confirmed from intake #1 — user explicitly specified the canonical GitHub login | S:95 R:90 A:90 D:95 |
| 2 | Certain | Scope is four files: `src/kit/skills/git-pr-review.md`, `docs/specs/skills/SPEC-git-pr-review.md`, `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/execution-skills.md` | Confirmed from intake #2 — grep verified no other `add-reviewer copilot` references exist in `src/` or `docs/` | S:95 R:90 A:95 D:95 |
| 3 | Certain | The Phase 2 poll check at `git-pr-review.md:92` (`.author.login == "copilot"`) MUST also change | Upgraded from intake Confident #3. The add-reviewer and poll filter reference the same bot identity — if only one is fixed, Phase 2's polling path silently times out instead of detecting the review. This is a functional correctness requirement, not an optional alignment | S:90 R:90 A:95 D:95 |
| 4 | Certain | The internal config key `review_tools.copilot` is NOT renamed | Confirmed from intake #4 — it is a fab-kit-local config name, not a GitHub login. User explicitly said "No config or schema changes" | S:95 R:85 A:95 D:95 |
| 5 | Certain | No backlog entry, no version bump, no migration | Confirmed from intake #5 — user stated "No backlog impact, no migration needed" | S:95 R:95 A:95 D:95 |
| 6 | Certain | Change type is `fix` | Confirmed from intake #6 — bug fix with no new capability | S:95 R:95 A:95 D:95 |
| 7 | Confident | Verification is by post-edit grep (`grep -rn "add-reviewer copilot\b"` returning zero) | No existing test harness covers markdown skill content; grep is the objective completeness check | S:80 R:85 A:85 D:80 |
| 8 | Confident | Deployed `.claude/skills/git-pr-review/SKILL.md` is NOT directly edited | User noted "the corresponding deployed copy regenerates via `fab sync`". Editing the deployed copy is wasted work and implies the wrong source of truth | S:90 R:85 A:85 D:85 |
| 9 | Confident | Both memory files need edits (not just one) | `configuration.md:61` has one command-string reference; `execution-skills.md:31,473` has two. All three narrative mentions concern the GitHub login, not the config key | S:90 R:85 A:90 D:90 |
| 10 | Confident | Flow-diagram and table rows in SPEC-git-pr-review both need the fix | Line 46 (flow diagram arrow) and line 86 (reviewer table cell) both embed the `gh pr edit --add-reviewer copilot` command string — both describe the same post-fix behavior | S:90 R:80 A:90 D:90 |

10 assumptions (6 certain, 4 confident, 0 tentative, 0 unresolved).
