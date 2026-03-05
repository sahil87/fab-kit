# Intake: Rename _context.md to _preamble.md

**Change**: 260221-5tj7-rename-context-to-preamble
**Created**: 2026-02-21
**Status**: Draft

## Origin

> User observed that `fab/project/context.md` (project context) and `fab/.kit/skills/_context.md` (shared skill preamble) have confusingly similar names. After discussion, agreed on `_preamble.md` as the replacement — it matches the file's own heading ("Shared Context Preamble") and avoids the overloaded "context" term.

## Why

The term "context" is heavily overloaded in this project: `context.md` (project context), `_context.md` (skill preamble), "context loading" (a section within `_context.md`), and "context layers" (the loading tiers). When someone mentions "the context file," it's ambiguous which file they mean. This causes confusion during skill development and documentation review.

The file `_context.md` already describes itself as a "Shared Context Preamble" in its heading. Renaming it to `_preamble.md` disambiguates it from project context while accurately reflecting its role: a preamble that every skill reads before proceeding.

## What Changes

### File rename

Rename `fab/.kit/skills/_context.md` to `fab/.kit/skills/_preamble.md`. No content changes to the file itself.

### Reference updates in live skill files

Every skill file in `fab/.kit/skills/` that contains the instruction line:

```
Read and follow the instructions in fab/.kit/skills/_context.md before proceeding.
```

must be updated to:

```
Read and follow the instructions in fab/.kit/skills/_preamble.md before proceeding.
```

Affected skill files (15):
- `fab-new.md`, `fab-continue.md`, `fab-ff.md`, `fab-fff.md`
- `fab-clarify.md`, `fab-switch.md`, `fab-setup.md`, `fab-status.md`
- `fab-archive.md`, `fab-discuss.md`
- `docs-hydrate-memory.md`, `docs-hydrate-specs.md`
- `_generation.md`, `internal-retrospect.md`, `internal-skill-optimize.md`

Additionally, `_context.md` (now `_preamble.md`) references itself in the opening blockquote — that self-reference must also be updated.

### Reference updates in scaffold

The scaffold template at `fab/.kit/scaffold/fab/project/context.md` is unrelated (it's a project context template, not a skill preamble reference) and requires no changes.

### Reference updates in docs

Memory and spec files that reference `_context.md` by path need updating. Key live files include:
- `docs/memory/fab-workflow/context-loading.md`
- `docs/memory/fab-workflow/kit-architecture.md`
- `docs/memory/fab-workflow/planning-skills.md`
- `docs/memory/fab-workflow/clarify.md`
- `docs/memory/fab-workflow/execution-skills.md`
- `docs/memory/fab-workflow/model-tiers.md`
- `docs/memory/fab-workflow/change-lifecycle.md`
- `docs/memory/fab-workflow/specs-index.md`
- `docs/specs/skills.md`
- `docs/specs/glossary.md`

### Archived changes — no updates

Archived change artifacts (`fab/changes/archive/`) are historical records and SHALL NOT be modified. References to `_context.md` in archived files remain as-is.

## Affected Memory

- `fab-workflow/context-loading`: (modify) Update references from `_context.md` to `_preamble.md`
- `fab-workflow/kit-architecture`: (modify) Update references from `_context.md` to `_preamble.md`
- `fab-workflow/planning-skills`: (modify) Update references from `_context.md` to `_preamble.md`
- `fab-workflow/clarify`: (modify) Update references from `_context.md` to `_preamble.md`
- `fab-workflow/execution-skills`: (modify) Update references from `_context.md` to `_preamble.md`

## Impact

- **Skill files**: All 15+ skill files need a one-line reference update
- **Memory/spec docs**: ~10 documentation files need path reference updates
- **Scaffold**: No impact (different file)
- **Scripts**: No impact (`_context.md` is not referenced by shell scripts — only by markdown skill files read by agents)
- **Backward compatibility**: None needed — `_context.md` is consumed by agents at runtime, not by compiled code. Once renamed and references updated, the old name has no lingering effect.

## Open Questions

(none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New name is `_preamble.md` | Explicitly agreed in conversation — matches the file's own heading | S:95 R:90 A:95 D:95 |
| 2 | Certain | Archived changes are not updated | Constitution principle: archives are historical records | S:90 R:95 A:90 D:95 |
| 3 | Certain | No content changes to the file itself | Scope is rename + reference updates only, per user description | S:90 R:95 A:90 D:95 |
| 4 | Confident | Scaffold `context.md` needs no changes | It's a project context template, unrelated to the skill preamble | S:80 R:90 A:85 D:85 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
