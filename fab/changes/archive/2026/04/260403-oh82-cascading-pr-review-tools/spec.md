# Spec: Cascading PR Review Tools

**Change**: 260403-oh82-cascading-pr-review-tools
**Created**: 2026-04-03
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Replacing the existing comment-processing behavior of `/git-pr-review` — the cascade is additive
- Supporting review tools beyond Copilot, Codex, and Claude in this change
- Automatically re-invoking `/git-pr-review` after a Copilot review lands — the user triggers the second pass manually

## Skill: Review Request Cascade

### Requirement: Cascade Order

The skill SHALL attempt review tools in this fixed order: Copilot, Codex, Claude. When a tool succeeds, the cascade MUST stop — no further tools are attempted. When a tool is unavailable or fails, the cascade SHALL fall through to the next tool.

#### Scenario: Full cascade fallthrough
- **GIVEN** Copilot is not available (no Enterprise/Business subscription), Codex CLI is not installed, and Claude CLI is installed
- **WHEN** the review request cascade runs
- **THEN** Copilot is attempted and fails, Codex is attempted and fails, Claude runs the review locally

#### Scenario: First tool succeeds
- **GIVEN** Copilot is available on the repository
- **WHEN** the review request cascade runs
- **THEN** Copilot review is requested and the cascade stops without attempting Codex or Claude

#### Scenario: All tools unavailable
- **GIVEN** Copilot is not available, Codex is not installed, and Claude is not installed
- **WHEN** the review request cascade runs
- **THEN** the skill prints a message indicating no review tools are available and stops

### Requirement: Copilot Reviewer (Remote)

The skill SHALL request a Copilot review via `gh pr edit {number} --add-reviewer copilot`. Detection is by attempting the command and checking the exit code. On success, the skill SHALL print "Copilot review requested" and exit — the user re-invokes `/git-pr-review` later to process Copilot's comments.

#### Scenario: Copilot available
- **GIVEN** the repository has Copilot code review enabled
- **WHEN** Copilot is attempted
- **THEN** `gh pr edit {number} --add-reviewer copilot` succeeds
- **AND** the skill prints "Copilot review requested — re-run /git-pr-review later to process comments"
- **AND** the cascade stops (no Codex or Claude attempted)

#### Scenario: Copilot unavailable
- **GIVEN** the repository does not have Copilot code review
- **WHEN** Copilot is attempted
- **THEN** `gh pr edit {number} --add-reviewer copilot` fails (non-zero exit)
- **AND** the cascade falls through to Codex

### Requirement: Codex Reviewer (Local)

The skill SHALL detect Codex availability via `command -v codex`. When available, it SHALL run Codex with the enriched review prompt (see Context Enrichment). Output SHALL be posted as a PR review comment via `gh api` and also printed to the terminal.

#### Scenario: Codex installed and runs
- **GIVEN** `codex` is on PATH
- **WHEN** Codex is attempted
- **THEN** Codex runs with the enriched diff as input
- **AND** the review output is posted as a PR comment
- **AND** the review output is printed to the terminal

#### Scenario: Codex not installed
- **GIVEN** `codex` is not on PATH
- **WHEN** Codex is attempted
- **THEN** detection fails immediately (`command -v codex` returns non-zero)
- **AND** the cascade falls through to Claude

### Requirement: Claude Reviewer (Local)

The skill SHALL detect Claude availability via `command -v claude`. When available, it SHALL run Claude with the enriched review prompt via `claude -p "..."`. Output SHALL be posted as a PR review comment via `gh api` and also printed to the terminal.

#### Scenario: Claude installed and runs
- **GIVEN** `claude` is on PATH
- **WHEN** Claude is attempted
- **THEN** Claude runs with the enriched diff as input
- **AND** the review output is posted as a PR comment
- **AND** the review output is printed to the terminal

#### Scenario: Claude not installed
- **GIVEN** `claude` is not on PATH
- **WHEN** Claude is attempted
- **THEN** detection fails immediately (`command -v claude` returns non-zero)
- **AND** since Claude is the last tool, the cascade reports no tools available

## Skill: Context Enrichment

### Requirement: Enriched Review Prompt

When running Codex or Claude locally, the skill SHALL construct an enriched prompt containing: the diff (`git diff main...HEAD`), the list of changed files (`git diff --name-only main...HEAD`), test results (run test suite and capture output, best-effort), and the PR description (`gh pr view --json body -q .body`). Each section SHALL be clearly labeled in the prompt.

#### Scenario: All context available
- **GIVEN** a PR exists with a description, and tests are configured
- **WHEN** the enriched prompt is constructed
- **THEN** the prompt includes the diff, file list, test output, and PR description as labeled sections

#### Scenario: No test suite or no PR description
- **GIVEN** the project has no test command or the PR has an empty body
- **WHEN** the enriched prompt is constructed
- **THEN** the missing sections are omitted and the prompt is still valid with the available context

### Requirement: Local Review Output Posting

When a local tool (Codex or Claude) produces review output, the skill SHALL attempt to post it as a PR comment via `gh api repos/{owner}/{repo}/issues/{number}/comments -f body="..."`. Posting is best-effort — if it fails, the output is still printed to the terminal and the skill does not abort.

#### Scenario: Posting succeeds
- **GIVEN** Codex or Claude produced review output
- **WHEN** the output is posted as a PR comment
- **THEN** the comment appears on the PR and the output is also printed to the terminal

#### Scenario: Posting fails
- **GIVEN** Codex or Claude produced review output but the `gh api` call fails
- **WHEN** the posting attempt fails
- **THEN** the error is logged, the output is printed to the terminal, and the skill continues without aborting

## Configuration: Review Tools

### Requirement: Per-Tool Enable/Disable

The skill SHALL read `review_tools` from `fab/project/config.yaml`. Each tool (copilot, codex, claude) MAY be set to `true` (try) or `false` (skip). When the `review_tools` key is absent, all tools SHALL default to `true`.

```yaml
review_tools:
  copilot: true
  codex: true
  claude: true
```

#### Scenario: Tool disabled in config
- **GIVEN** `review_tools.codex` is set to `false` in config.yaml
- **WHEN** the cascade runs
- **THEN** Codex is skipped entirely (no detection attempted) and the cascade proceeds to Claude

#### Scenario: Config key absent
- **GIVEN** `review_tools` key does not exist in config.yaml
- **WHEN** the cascade runs
- **THEN** all three tools are attempted in order (default behavior)

#### Scenario: All tools disabled
- **GIVEN** `review_tools.copilot`, `review_tools.codex`, and `review_tools.claude` are all `false`
- **WHEN** the cascade runs
- **THEN** the skill prints a message indicating all review tools are disabled and stops

### Requirement: `--tool` Flag

The skill SHALL accept a `--tool <name>` flag that forces a specific reviewer, bypassing the cascade. Valid values: `copilot`, `codex`, `claude`. When `--tool` is provided, only that tool is attempted — no fallthrough.

#### Scenario: Force specific tool
- **GIVEN** the user invokes `/git-pr-review --tool claude`
- **WHEN** the skill processes the flag
- **THEN** only Claude is attempted, regardless of Copilot or Codex availability or config settings

#### Scenario: Invalid tool name
- **GIVEN** the user invokes `/git-pr-review --tool invalid`
- **WHEN** the skill processes the flag
- **THEN** the skill prints an error listing valid tool names and stops

## Migration: Add review_tools Config

### Requirement: Config Migration

A migration file SHALL be created at `src/kit/migrations/1.1.0-to-1.2.0.md` that adds the `review_tools` block to existing `fab/project/config.yaml` files. The migration MUST be idempotent — if `review_tools` already exists, it skips without modification.

#### Scenario: Fresh migration
- **GIVEN** `config.yaml` exists without a `review_tools` key
- **WHEN** the migration runs
- **THEN** the default `review_tools` block (all true) is appended to `config.yaml`

#### Scenario: Already migrated
- **GIVEN** `config.yaml` already has a `review_tools` key
- **WHEN** the migration runs
- **THEN** no changes are made

## Skill: Integration with Existing Behavior

### Requirement: Cascade Placement in Skill Flow

The review request cascade SHALL run as a new phase in Step 2 of `/git-pr-review`. When Step 2 Phase 1 finds no existing reviews with comments, the cascade runs before the skill stops. If existing reviews with comments are found, the cascade is skipped and the skill proceeds to its existing comment-processing flow (Steps 3-6).

#### Scenario: No existing reviews — cascade triggers
- **GIVEN** the PR has no submitted reviews with inline comments
- **WHEN** Step 2 Phase 1 completes
- **THEN** the review request cascade runs (Copilot → Codex → Claude)

#### Scenario: Existing reviews — cascade skipped
- **GIVEN** the PR has submitted reviews with inline comments
- **WHEN** Step 2 Phase 1 completes
- **THEN** the cascade is skipped and the skill proceeds to Step 3 (fetch and process comments)

### Requirement: Spec Update for SPEC-git-pr-review

The existing `docs/specs/skills/SPEC-git-pr-review.md` SHALL be updated to reflect the new cascade behavior added by this change.

#### Scenario: Spec reflects cascade
- **GIVEN** the change is implemented
- **WHEN** the spec file is reviewed
- **THEN** it includes the cascade flow (Copilot → Codex → Claude), config-based enable/disable, and the `--tool` flag

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Cascade order: Copilot → Codex → Claude | Confirmed from intake #1 — user explicitly specified | S:95 R:85 A:90 D:95 |
| 2 | Certain | Copilot is remote (GitHub API), Codex and Claude are local (CLI) | Confirmed from intake #2 — user confirmed modalities | S:90 R:80 A:90 D:95 |
| 3 | Certain | Availability detection: attempt-and-fallback for Copilot, command check for local tools | Confirmed from intake #3 — user agreed on pattern | S:85 R:90 A:85 D:90 |
| 4 | Confident | Cascade extends `/git-pr-review` Step 2 rather than being a separate skill or step | Upgraded from intake #4 — natural fit: Step 2 already routes on review presence, cascade adds a fallback path when no reviews exist | S:75 R:75 A:85 D:75 |
| 5 | Confident | Context enrichment includes diff, file list, test results, PR body | Confirmed from intake #5 — user mentioned "enrich context with file tree and test results" | S:75 R:80 A:70 D:65 |
| 6 | Confident | Config under `review_tools` key with per-tool boolean flags | Confirmed from intake #6 — user specified config structure | S:80 R:75 A:75 D:70 |
| 7 | Certain | Local review output posted as PR comments (best-effort) and printed to terminal | Confirmed from intake #7 — user said "if possible yes" | S:85 R:80 A:85 D:90 |
| 8 | Certain | `--tool` flag to force a specific reviewer | Confirmed from intake #8 — user confirmed | S:90 R:85 A:85 D:90 |
| 9 | Certain | Copilot request and comment processing are split into separate runs | Confirmed from intake #9 — user said "split is better" | S:85 R:75 A:80 D:90 |
| 10 | Certain | Migration at `src/kit/migrations/1.1.0-to-1.2.0.md` | Confirmed from intake #10 — user flagged migration requirement. Version number is a Confident guess based on current version 1.1.0 | S:85 R:70 A:80 D:85 |
| 11 | Confident | Local tools post to issues comments endpoint (`/issues/{n}/comments`) not review comments | PR-level comments are simpler for tool output (no line-level annotation needed); review comments require `commit_id` and `path` | S:60 R:80 A:75 D:65 |

11 assumptions (7 certain, 4 confident, 0 tentative, 0 unresolved).
