# Intake: Cascading PR Review Tools

**Change**: 260403-oh82-cascading-pr-review-tools
**Created**: 2026-04-03
**Status**: Draft

## Origin

> Add cascading code review to git-pr-review skill: Try Copilot first (gh pr edit --add-reviewer copilot), then Codex CLI (codex), then Claude CLI (claude) for PR code reviews. Each tool should be detected for availability and fall through to the next if unavailable. Copilot posts comments on the PR remotely; Codex and Claude run locally on the diff. The cascade should be configurable and the review should enrich context with file tree and test results where possible.

**Interaction mode**: Conversational — user discussed the problem (Copilot availability uncertainty), explored tool options, and agreed on cascade order before requesting the change.

**Key decisions from conversation**:
- Cascade order: Copilot → Codex → Claude (user-specified)
- Copilot is remote (GitHub API), Codex and Claude are local (CLI, diff-based)
- Primary motivation: users may not have Copilot Enterprise/Business
- Each tool detected for availability before attempting

## Why

The current `/git-pr-review` skill only *processes* existing review comments — it cannot *request* reviews from automated tools. Users need a way to get automated code reviews on their PRs, but the available tools vary by account type:

- **GitHub Copilot** requires Enterprise/Business subscription — many users don't have it
- **OpenAI Codex CLI** is a separate tool that may or may not be installed
- **Claude CLI** (Claude Code) is likely available since the user is already in the fab workflow

Without this change, users must manually figure out which review tool is available and invoke it themselves. A cascading fallback ensures every user gets automated review regardless of their tool availability.

## What Changes

### New "Request Review" phase in `/git-pr-review`

Add a new step (before the existing Step 0) or a separate invocation mode that **requests** an automated review on the current PR. The cascade:

1. **Copilot** (remote): `gh pr edit {number} --add-reviewer copilot`
   - Detection: attempt the command, check exit code
   - On success: Copilot posts review comments directly on the PR (async)
   - Skill exits with "Copilot review requested" — user re-invokes `/git-pr-review` later to process the comments (split into request + process)

2. **Codex** (local): `codex "Review this PR diff for bugs, edge cases, and improvements"`
   - Detection: `command -v codex`
   - Input: `git diff main...HEAD` piped in, enriched with file tree and test results
   - Output: posted as PR review comment via `gh api` if possible, also printed to terminal

3. **Claude** (local): `claude -p "Review this diff..."`
   - Detection: `command -v claude`
   - Input: same enriched diff as Codex
   - Output: posted as PR review comment via `gh api` if possible, also printed to terminal

### Context enrichment for local tools

When running Codex or Claude locally, enrich the review prompt with:
- The diff itself (`git diff main...HEAD`)
- File tree of changed files (`git diff --name-only main...HEAD`)
- Test results if available (run test suite, capture output)
- PR description if available (`gh pr view --json body -q .body`)

### Configurability via `fab/project/config.yaml`

Each reviewer can be enabled/disabled individually. Example config:

```yaml
review_tools:
  copilot: true    # try GitHub Copilot (remote)
  codex: true      # try OpenAI Codex CLI (local)
  claude: true     # try Claude CLI (local)
```

Setting a tool to `false` skips it in the cascade. All default to `true` (try everything). If the config key is absent, all tools are tried. This lets users permanently disable tools they don't have access to (e.g., `copilot: false` for non-Enterprise accounts) rather than failing and falling through every time.

### `--tool` flag

A `--tool <name>` flag forces a specific reviewer, bypassing the cascade. Example: `/git-pr-review --tool claude` skips Copilot and Codex, goes straight to Claude. Valid values: `copilot`, `codex`, `claude`.

### Migration for existing users

A migration file (`src/kit/migrations/1.1.0-to-X.X.X.md`) adds the `review_tools` block to existing `fab/project/config.yaml` files. The migration:

1. Check if `review_tools` key already exists in `config.yaml` — if yes, skip
2. Append the default `review_tools` block (all tools enabled):
   ```yaml
   review_tools:
     copilot: true
     codex: true
     claude: true
   ```
3. Print confirmation message

This ensures existing projects pick up the new config without manual editing.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document the new review-request capability in `/git-pr-review`

## Impact

- **`src/kit/skills/git-pr-review.md`**: Primary change — add cascade logic
- **`fab/project/config.yaml`**: New `review_tools` config key
- **`src/kit/migrations/1.1.0-to-X.X.X.md`**: Migration to add `review_tools` block to existing user `config.yaml` files
- **`docs/specs/skills/SPEC-git-pr-review.md`**: Spec update if it exists
- No changes to the `fab` Go binary — this is pure skill logic

## Open Questions

None — all resolved in conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Cascade order: Copilot → Codex → Claude | Discussed — user explicitly specified this order | S:95 R:85 A:90 D:95 |
| 2 | Certain | Copilot is remote (GitHub API), Codex and Claude are local (CLI) | Discussed — user confirmed these modalities | S:90 R:80 A:90 D:95 |
| 3 | Certain | Availability detection via command check / API attempt | Discussed — user agreed on try-and-fallback pattern | S:85 R:90 A:85 D:90 |
| 4 | Confident | This extends `/git-pr-review` rather than being a new skill | Natural fit — same skill handles PR review lifecycle | S:70 R:75 A:80 D:70 |
| 5 | Confident | Context enrichment includes diff, file tree, test results, PR body | Discussed — user mentioned "enrich context with file tree and test results" | S:75 R:80 A:70 D:65 |
| 6 | Confident | Configuration lives in `fab/project/config.yaml` under a `review_tools` key with per-tool on/off | Discussed — user confirmed config-based enable/disable for each reviewer | S:80 R:75 A:75 D:70 |
| 7 | Certain | Local review output posted as PR comments if possible, also printed to terminal | Clarified — user said "if possible yes" to posting as PR comments | S:85 R:80 A:85 D:90 |
| 8 | Certain | `--tool` flag to force a specific reviewer (e.g., `/git-pr-review --tool claude`) | Clarified — user confirmed "sure" | S:90 R:85 A:85 D:90 |
| 9 | Certain | Copilot request and comment processing are split into separate runs | Clarified — user said "split is better" | S:85 R:75 A:80 D:90 |
| 10 | Certain | Migration needed to add `review_tools` to existing user config.yaml | Discussed — user flagged this requirement | S:90 R:70 A:85 D:90 |

10 assumptions (7 certain, 3 confident, 0 tentative, 0 unresolved).
