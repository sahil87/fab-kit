# Proposal: Add fab/specs/index.md to context loading in apply, review, and archive

**Change**: 260210-7wxx-add-specs-index-context-loading
**Created**: 2026-02-10
**Status**: Draft

## Why

`_context.md` defines an "Always Load" protocol that includes `fab/specs/index.md` as one of four files every skill must read before proceeding. However, `fab-apply.md`, `fab-review.md`, and `fab-archive.md` omit `fab/specs/index.md` from their Context Loading sections. This deviation means these skills operate without the specifications landscape context, which could lead to missed cross-references or inconsistencies with the pre-implementation design intent.

## What Changes

- Add `fab/specs/index.md` to the Context Loading section of `fab-apply.md`
- Add `fab/specs/index.md` to the Context Loading section of `fab-review.md`
- Add `fab/specs/index.md` to the Context Loading section of `fab-archive.md`
- Each addition includes a brief description consistent with `_context.md`: "specifications landscape (pre-implementation design intent, human-curated)"

## Affected Docs

### New Docs
- None

### Modified Docs
- `fab-workflow/execution-skills`: Update to note that all execution skills now load `fab/specs/index.md` per the always-load protocol

### Removed Docs
- None

## Impact

- `fab/.kit/skills/fab-apply.md` — Context Loading section gains one line
- `fab/.kit/skills/fab-review.md` — Context Loading section gains one line
- `fab/.kit/skills/fab-archive.md` — Context Loading section gains one line

## Open Questions

- None — this is a straightforward alignment fix. The `_context.md` protocol is unambiguous and the three skills clearly omit the required file.
