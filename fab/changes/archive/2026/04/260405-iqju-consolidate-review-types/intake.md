# Intake: Consolidate Review Types

**Change**: 260405-iqju-consolidate-review-types
**Created**: 2026-04-05
**Status**: Draft

## Origin

> Consolidate the two review types in the fab pipeline:
>
> Type 1 (inward review) — validates against spec, tasks, checklist. Already in the right place (apply ↔ review loop in fab-continue). No change.
>
> Type 2 (outward review) — holistic diff review with full repo access. Currently implemented as a one-shot `claude -p "..."` call inside `git-pr-review`'s Phase 2 cascade. Problems: (a) it fires too late — after ship/PR creation; (b) it uses a one-shot `-p` call with pasted diff text instead of a proper sub-agent that can explore the repo.

This change was initiated as a synthesized description from a conversation. The three distinct sub-changes were specified explicitly with file-level precision.

## Why

The current two-reviewer model has a structural timing problem: the holistic outward diff review runs only after the change has been shipped (PR created). By that point, rework means amending commits, force-pushing, or creating follow-up PRs — all of which are noisier and more disruptive than catching issues pre-ship.

Additionally, the outward review uses a one-shot `claude -p "..."` invocation with a pasted diff. This prevents the reviewer from exploring the broader codebase for context (e.g., checking how similar patterns are implemented elsewhere, validating interface contracts, reading referenced memory files). A proper sub-agent dispatch with full tool access produces qualitatively better findings.

Finally, `git-pr-review` Phase 2 currently holds three review mechanisms (Codex, Claude, Copilot) that conceptually belong to different lifecycle positions: Codex and Claude are pre-ship diff reviewers (now moving to `fab-continue`), while Copilot is a post-PR reviewer that operates on GitHub infrastructure. Separating these makes each skill's responsibility clear.

If we don't fix this: outward review continues to catch issues too late, one-shot reviews miss context-dependent bugs, and `git-pr-review` carries dead weight (Codex/Claude calls that no longer have a purpose post-consolidation).

## What Changes

### 1. `fab-continue` — Outward Review Sub-Agent

During the review stage, `fab-continue` currently dispatches one inward sub-agent (validates against spec/tasks/checklist). After this change, it dispatches two sub-agents in parallel:

**Inward sub-agent** (unchanged): validates implementation against spec requirements, tasks checklist, and project quality rules. Returns must-fix/should-fix/nice-to-have findings.

**Outward sub-agent** (new): performs a holistic diff review with full repo access.

- Dispatched via the Agent tool (`subagent_type: "general-purpose"`)
- Given: the diff of all changed files, the list of changed file paths
- Allowed to: read any file in the repo, explore neighboring code, check memory and spec files for context
- Uses a Codex → Claude cascade: attempt Codex first (via `codex` CLI or equivalent); if Codex is unavailable or fails, fall back to Claude as the reviewer
- Returns the same structured severity format: must-fix / should-fix / nice-to-have
- Both sub-agents' findings are merged and feed the same rework loop

The outward sub-agent prompt instructs it to look for: interface contract violations, inconsistencies with documented patterns (memory files), missing cross-references, behavioral regressions not caught by the inward reviewer, and structural issues that only become visible with full repo context.

Rework loop behavior is unchanged: must-fix findings trigger rework; the loop is bounded by the rework budget in `fab/project/code-review.md` (default max cycles: 3).

### 2. `git-pr-review` — Phase 2 Becomes Copilot-Only

Phase 2 of `git-pr-review` currently attempts a cascade: Codex → Claude → Copilot. After this change, Phase 2 is Copilot-only:

```
Phase 2: Automated Reviewer
  1. Attempt: gh pr edit {number} --add-reviewer copilot
  2. If success:
       - Wait/poll up to 10 minutes for the Copilot review to appear
         (poll interval: 30s; check via gh pr view --json reviews)
       - Once review appears: process it via Step 3+ (existing logic)
  3. If failure (Copilot not available on this repo / command fails):
       - Output: "No automated reviewer available. Run /git-pr-review when reviews are added."
       - Clean finish (no error, no rework)
```

Phase 1 (human/bot review comment processing) is completely unchanged.

The Codex and Claude tool invocations are removed from Phase 2. They are no longer called in `git-pr-review`.

### 3. `config.yaml` `review_tools` — Remove `codex` and `claude` Keys

Current schema:
```yaml
review_tools:
    claude: true
    codex: true
    copilot: true
```

After this change:
```yaml
review_tools:
    copilot: true/false
```

The `codex` and `claude` keys are removed. They had controlled whether those tools were used in `git-pr-review` Phase 2 — that phase no longer uses them. The outward sub-agent in `fab-continue` always attempts Codex→Claude cascade regardless of config (the cascade is a fallback mechanism, not a user preference).

This is a breaking schema change. A migration file is required if existing projects have `review_tools.codex` or `review_tools.claude` set. The migration strips those keys and leaves `review_tools.copilot` intact.

### 4. `_review.md` — Shared Review Skill File

Extract the Review Behavior from `fab-continue.md` into a new shared skill file `src/kit/skills/_review.md`, following the same pattern as `_generation.md` (which holds Spec/Tasks/Checklist generation procedures).

- `_review.md` defines both sub-agent dispatches: inward (spec/tasks/checklist validation) and outward (full-repo holistic review)
- `fab-continue.md` references `_review.md` instead of inlining Review Behavior — same pattern as how it references `_generation.md` for planning stages
- `fab-ff.md` and `fab-fff.md` already say "Dispatch /fab-continue as subagent — Review Behavior" — they just need the pointer updated to note `_review.md` is the authoritative source
- This makes the review stage as easy to update as the generation stage: one file, one place

The outward sub-agent is always on — no config flag. The Codex→Claude cascade gracefully no-ops if neither tool is available, so there is no harm in always attempting.

### 5. Spec Files and Memory Updates

- Update `docs/specs/skills/SPEC-fab-continue.md` to document the dual sub-agent dispatch and `_review.md` delegation
- Update `docs/specs/skills/SPEC-git-pr-review.md` to document the Copilot-only Phase 2
- Update `docs/specs/skills/SPEC-fab-ff.md` and `SPEC-fab-fff.md` to reference `_review.md` as the review behavior source
- After implementation, hydrate memory: `docs/memory/fab-workflow/execution-skills.md` and `docs/memory/fab-workflow/configuration.md`

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document the new dual sub-agent dispatch in the review stage, `_review.md` extraction, and Copilot-only Phase 2 in git-pr-review
- `fab-workflow/configuration`: (modify) Document the simplified `review_tools` schema (copilot-only key)

## Impact

**Skill source files** (canonical sources — always edit these, not deployed copies):
- `src/kit/skills/_review.md` — new file: extracted Review Behavior (inward + outward sub-agents)
- `src/kit/skills/fab-continue.md` — update: delegate Review Behavior to `_review.md`
- `src/kit/skills/git-pr-review.md` — update: strip Codex/Claude from Phase 2, Copilot-only with poll/wait
- `src/kit/skills/fab-ff.md` — update: reference `_review.md` as review behavior source
- `src/kit/skills/fab-fff.md` — update: same as fab-ff

**Spec files** (human-curated, update manually):
- `docs/specs/skills/SPEC-fab-continue.md`
- `docs/specs/skills/SPEC-git-pr-review.md`
- `docs/specs/skills/SPEC-fab-ff.md`
- `docs/specs/skills/SPEC-fab-fff.md`

**Config schema**:
- `fab/project/config.yaml` — update the dev repo's own config (remove `codex` and `claude` keys)
- Migration file in `src/kit/migrations/` — direct `yq` key-removal of `review_tools.codex` and `review_tools.claude`

**Memory files** (hydrate after implementation):
- `docs/memory/fab-workflow/execution-skills.md`
- `docs/memory/fab-workflow/configuration.md`

No changes to the `fab` Go binary, no changes to CLI command signatures, no changes to `.status.yaml` schema.

## Open Questions

None — all decisions resolved via clarification.

## Clarifications

### Session 2026-04-05 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 5 | Confirmed | Both sub-agents run in parallel |
| 6 | Confirmed | Migration is direct yq key-removal after example review |
| 7 | Changed | Extract to `_review.md` shared skill (like `_generation.md`); fab-ff/fab-fff/fab-continue all reference it |
| 8 | Confirmed | Outward sub-agent always on — no config flag |
| 9 | Confirmed | Direct key-removal migration, no dry-run needed |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Inward review sub-agent behavior is unchanged | Explicitly stated in the description ("No change") | S:95 R:95 A:95 D:95 |
| 2 | Certain | Outward sub-agent uses Agent tool (general-purpose), not Skill tool | Consistent with preamble orchestrator pattern; description specifies "dispatched via Agent tool" | S:90 R:80 A:90 D:90 |
| 3 | Certain | git-pr-review Phase 1 (human/bot reviews) is unchanged | Explicitly stated in the description | S:95 R:95 A:95 D:95 |
| 4 | Certain | Canonical skill sources are in src/kit/skills/, not .claude/skills/ | context.md states: "When modifying a skill, always update the source at src/kit/skills/" | S:95 R:95 A:95 D:95 |
| 5 | Certain | Both sub-agents run in parallel during the review stage | Clarified — user confirmed | S:95 R:70 A:80 D:75 |
| 6 | Certain | Migration is direct yq key-removal of review_tools.codex and review_tools.claude | Clarified — user confirmed after example review | S:95 R:65 A:85 D:80 |
| 7 | Certain | Review Behavior extracted to _review.md shared skill; fab-continue/fab-ff/fab-fff all reference it | Clarified — user changed to _review.md extraction pattern (like _generation.md) | S:95 R:70 A:65 D:70 |
| 8 | Certain | Outward sub-agent is always on (no config flag to disable) | Clarified — user confirmed; cascade gracefully no-ops if tools absent | S:95 R:60 A:70 D:55 |
| 9 | Certain | Migration is direct key-removal (not dry-run-first) | Clarified — user confirmed; trivial operation does not warrant dry-run overhead | S:95 R:55 A:60 D:55 |

9 assumptions (9 certain, 0 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
