# Intake: Add Code Review Scaffold & 5 Cs of Quality

**Change**: 260218-xkkc-add-code-review-5cs-quality
**Created**: 2026-02-18
**Status**: Draft

## Origin

> Add code-review.md scaffold and document the "5 Cs of Quality" model. Create fab/.kit/scaffold/code-review.md with review-specific policy (severity definitions, review scope, false positive handling, rework budget, project-specific review rules). Update README.md "Code Quality as a Guardrail" section to introduce the 5 Cs mental model (Constitution, Context, Code Quality, Code Review, Config). Update context-loading to include code-review.md as a 7th Always Load file (optional). Update /fab-setup config menu to include code-review.md editing.

One-shot input. The user identified a gap in the configuration surface: review-specific policy has no dedicated home. The existing 4 configuration files (constitution.md, context.md, code-quality.md, config.yaml) plus the proposed code-review.md form a "5 Cs" mental model worth documenting.

## Why

1. **Review policy is scattered** — Severity triage (must-fix/should-fix/nice-to-have), review scope, and rework budget are currently hardcoded in the `fab-continue.md` skill prompt. Projects can't customize review behavior without forking the skill.
2. **Author vs. critic separation** — `code-quality.md` guides the *writing* agent during apply. Review needs its own policy for the *reviewing* sub-agent — different cognitive mode, different concerns (severity mapping, scope boundaries, false positive handling).
3. **The 5 Cs model deserves documentation** — The five configuration files already exist (4 of 5) but there's no unifying narrative. Naming the pattern makes it discoverable and teachable.

## What Changes

### 1. New scaffold: `fab/.kit/scaffold/code-review.md`

A template with placeholder sections for projects to customize:

- **Severity Definitions** — What constitutes must-fix, should-fix, nice-to-have (project can override the defaults from the skill prompt)
- **Review Scope** — Boundaries: only changed files? Skip generated code? Vendor directories?
- **False Positive Policy** — How to suppress, annotate, or override findings
- **Rework Budget** — Max auto-rework cycles, escalation triggers
- **Project-Specific Review Rules** — e.g., "all public APIs need integration tests", "no new dependencies without justification"

### 2. Update `fab/.kit/scripts/sync/3-sync-workspace.sh`

Scaffold `fab/code-review.md` from the template during sync (same pattern as context.md and code-quality.md — create if missing, never overwrite).

### 3. Update `fab/.kit/skills/_context.md`

Add `fab/code-review.md` as a 7th file in the Always Load layer. Optional — no error if missing. The review stage in `fab-continue` should specifically reference it when dispatching the sub-agent.

### 4. Update `fab/.kit/skills/fab-setup.md`

Add `code-review.md` as item 10 in the config editing menu (alongside context.md and code-quality.md).

### 5. Update `fab/.kit/scaffold/config.yaml`

Add `fab/code-review.md` to the "Companion files" comment block at the top.

### 6. Update `README.md` — "Code Quality as a Guardrail" section

Introduce the 5 Cs mental model. Show the relationship between the five files with a table or diagram. Each C answers a different question:

| C | File | Question |
|---|------|----------|
| Constitution | `fab/constitution.md` | What are our non-negotiable principles? |
| Context | `fab/context.md` | What are we working with? |
| Code Quality | `fab/code-quality.md` | How should code look when we write it? |
| Code Review | `fab/code-review.md` | What should we look for when we validate? |
| Config | `fab/config.yaml` | What are the project's factual settings? |

### 7. Update `docs/memory/fab-workflow/configuration.md`

Add code-review.md to the configuration relationship section and document its purpose.

## Affected Memory

- `fab-workflow/configuration`: (modify) Add code-review.md to configuration relationships, lifecycle management menu, and design decisions
- `fab-workflow/context-loading`: (modify) Add code-review.md as 7th Always Load file

## Impact

- **Scaffold**: `fab/.kit/scaffold/code-review.md` (new file)
- **Skills**: `_context.md`, `fab-setup.md`, `fab-continue.md` (review stage references)
- **Scripts**: `sync/3-sync-workspace.sh` (scaffold new file on sync)
- **Config scaffold**: `config.yaml` comment header
- **README.md**: "Code Quality as a Guardrail" section expansion
- **Memory**: 2 files modified during hydrate

## Open Questions

- Should the scaffold include default severity definitions (matching what's currently in the skill prompt) or just section headings with guidance comments?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | code-review.md is optional — skills proceed without error if missing | Same pattern as context.md and code-quality.md; backward compatible | S:95 R:90 A:95 D:95 |
| 2 | Certain | Scaffold lives at fab/.kit/scaffold/code-review.md | All scaffolds follow this pattern | S:95 R:90 A:95 D:95 |
| 3 | Confident | fab-sync.sh creates fab/code-review.md from scaffold if missing | Same mechanism as context.md and code-quality.md — confirmed in 3-sync-workspace.sh | S:80 R:85 A:85 D:80 |
| 4 | Confident | Scaffold includes populated defaults (not just empty sections) | User said "fill code-review.md using a scaffold" — implies useful starting content, not just headings | S:75 R:90 A:80 D:70 |
| 5 | Confident | README section is an expansion of the existing "Code Quality as a Guardrail" heading, not a new section | User selected that heading and said "add the above focus on quality to README.md also (under Code Quality as a Guardrail)" | S:85 R:90 A:85 D:75 |
| 6 | Certain | Memory files updated during hydrate, not as part of this change's apply | Standard fab workflow — memory is post-implementation | S:95 R:90 A:95 D:95 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
