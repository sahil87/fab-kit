# Intake: Fix git-pr-review Copilot reviewer login

**Change**: 260418-u1m1-copilot-reviewer-login
**Created**: 2026-04-18
**Status**: Draft

## Origin

One-shot bug report from the user. The `/git-pr-review` skill (and its spec/memory docs) hardcodes the reviewer login as `copilot`, which is not a valid GitHub user. The correct login for GitHub's Copilot PR reviewer bot is `copilot-pull-request-reviewer`.

> Fix git-pr-review skill to use correct Copilot bot login. `gh pr edit --add-reviewer copilot` fails because the correct GitHub login for the Copilot PR reviewer bot is `copilot-pull-request-reviewer`, not `copilot`.

The user noted the bug is non-fatal because the PR-creation hook already queues a Copilot review through another path, so reviews eventually appear. But the explicit add-reviewer call silently errors, masking the real review trigger and making the skill flow confusing to operators.

## Why

1. **Pain point**: `gh pr edit {n} --add-reviewer copilot` returns a non-zero exit because `copilot` is not a valid GitHub user/app login. This sends `/git-pr-review` Phase 2 down the failure branch ("No automated reviewer available. Run /git-pr-review when reviews are added.") even when Copilot review *is* available via the hook-queued path.
2. **Consequence if unfixed**: The skill's self-reported behavior diverges from reality — it says no reviewer was requested even when a Copilot review shows up moments later. This wastes operator attention on re-runs and masks the true review trigger (the PR-creation hook).
3. **Why this approach**: `copilot-pull-request-reviewer` is the canonical, documented GitHub login for the Copilot PR review bot. A literal string replacement in the one skill file, one spec file, and two memory files fully resolves the issue. No schema, config, or API changes needed.

## What Changes

### 1. `src/kit/skills/git-pr-review.md`

Replace the hardcoded `copilot` reviewer string in Phase 2:

- Line 87: `gh pr edit {number} --add-reviewer copilot` → `gh pr edit {number} --add-reviewer copilot-pull-request-reviewer`
- Line 92 (the polling check): `.reviews | map(select(.author.login == "copilot")) | length` → `.reviews | map(select(.author.login == "copilot-pull-request-reviewer")) | length`

Both references point to the same bot identity — the `--add-reviewer` call tags the bot, and the poll filters review events by the bot's `author.login`. If only the add-reviewer call is corrected, the poll would still never match incoming Copilot reviews and the skill would wait out the 20-attempt timeout. Both lines must change together.

The existing config key `review_tools.copilot` (used to enable/disable the reviewer) stays as `copilot` — that's an internal config name, not a GitHub login, and downstream migrations already encode it.

### 2. `docs/specs/skills/SPEC-git-pr-review.md`

Two literal occurrences of `copilot` as a GitHub-login reviewer string:

- Line 46 (flow diagram): `│     ├─ Bash: gh pr edit {n} --add-reviewer copilot` → `--add-reviewer copilot-pull-request-reviewer`
- Line 86 (reviewer table): `Attempt \`gh pr edit --add-reviewer copilot\`` → `--add-reviewer copilot-pull-request-reviewer`

### 3. `docs/memory/fab-workflow/configuration.md`

Line 61 documents the `review_tools.copilot` key and mentions the `gh pr edit --add-reviewer copilot` call. Update the command example only (leave the config key name alone):

- `gh pr edit --add-reviewer copilot` → `gh pr edit --add-reviewer copilot-pull-request-reviewer`

### 4. `docs/memory/fab-workflow/execution-skills.md`

Two occurrences:

- Line 31: the long narrative of Phase 2 — update the `gh pr edit {number} --add-reviewer copilot` command string
- Line 473: the decision log entry — update the same command string

Both are descriptions of the bot's login, not the internal config key, so `copilot-pull-request-reviewer` applies.

### 5. Verification

After edits, run `grep -rn "add-reviewer copilot\b" src/ docs/` and confirm no unqualified `copilot` reviewer references remain. Also confirm all updated lines match the GitHub canonical login `copilot-pull-request-reviewer`.

## Affected Memory

- `fab-workflow/configuration.md`: (modify) update the documented `gh pr edit --add-reviewer` command example to use `copilot-pull-request-reviewer`
- `fab-workflow/execution-skills.md`: (modify) update the Phase 2 narrative and the decision log entry to use `copilot-pull-request-reviewer`

No new memory files. No domain additions.

## Impact

- **Files**: `src/kit/skills/git-pr-review.md`, `docs/specs/skills/SPEC-git-pr-review.md`, `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/execution-skills.md` (4 files, ~5 line-level edits total)
- **APIs / schemas**: none
- **Config**: none (the `review_tools.copilot` key stays as-is)
- **Backlog / migrations**: none — no version bump needed, the kit regenerates deployed skills via `fab sync` on next run
- **Runtime risk**: zero — string replacement only, no logic change, no stage-transition change
- **Test surface**: no automated tests for this skill; verification is by running `/git-pr-review` against a live PR after release. Pre-ship validation is a grep check and a re-read of the updated phase-2 block for internal consistency.

## Open Questions

None — the bug, fix, and files are fully specified.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replacement login is `copilot-pull-request-reviewer` | User explicitly specified this as the canonical GitHub login for the Copilot PR reviewer bot | S:95 R:90 A:90 D:95 |
| 2 | Certain | Scope is `src/kit/skills/git-pr-review.md`, `docs/specs/skills/SPEC-git-pr-review.md`, and `docs/memory/fab-workflow/` | User enumerated these files explicitly in the change request; grep confirmed no other `add-reviewer copilot` references exist in src/ or docs/ | S:95 R:90 A:95 D:90 |
| 3 | Confident | The poll-check at `git-pr-review.md:92` (`.author.login == "copilot"`) must also change to `copilot-pull-request-reviewer` | The `--add-reviewer` call and the poll filter reference the same bot; if only add-reviewer is fixed, the poll will never match incoming reviews and will time out. User wrote "every `gh pr edit --add-reviewer copilot` (or equivalent) invocation" — the `(or equivalent)` phrase covers this | S:75 R:85 A:80 D:85 |
| 4 | Certain | The internal config key `review_tools.copilot` is NOT renamed | It is a local config name, not a GitHub login. Renaming it would require a migration and break existing user configs. User said "No config or schema changes." | S:95 R:85 A:95 D:95 |
| 5 | Certain | No backlog entry, no version bump, no migration | User explicitly stated "No backlog impact, no migration needed" and "No config or schema changes" | S:95 R:95 A:95 D:95 |
| 6 | Confident | Change type is `fix` | Described as a bug fix; keyword "Fix"/"bug" in the user's input; change has no new capability | S:95 R:90 A:90 D:90 |
| 7 | Confident | Verification is by grep after edits, not by automated tests | No unit tests cover the skill's string contents; grep is the closest objective check that the replacement is complete and consistent | S:70 R:80 A:85 D:80 |
| 8 | Confident | Deployed `.claude/skills/` copy regenerates via `fab sync` | User explicitly noted "the corresponding deployed copy regenerates via `fab sync`". This change only edits `src/kit/skills/`; the deployed artifact is downstream | S:90 R:85 A:85 D:85 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
