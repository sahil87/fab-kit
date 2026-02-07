# Proposal: Add `fab/specs/` Index and Clarify Specs vs Docs Distinction

**Change**: 260207-bb1q-add-specs-index
**Created**: 2026-02-07
**Status**: Complete

## Why

The fab workflow currently has `fab/docs/` as the centralized knowledge store but no parallel `fab/specs/` directory. Docs capture *what happened* (post-implementation truth), but there's no persistent home for *what was planned* (pre-implementation intent). This creates two problems:

1. **Specs are ephemeral** — they live only inside `fab/changes/` and get archived, making it hard to find the original design intent behind a feature
2. **No clear conceptual layer** — docs are detailed and implementation-accurate, but sometimes you want the higher-level "why and what" without the "exactly how"

Adding `fab/specs/` gives specs a permanent home alongside docs, with a clear separation of concerns.

## What Changes

- **Add `fab/specs/index.md`** — new centralized index for specification artifacts, parallel to `fab/docs/index.md`
- **Update `fab/docs/index.md` boilerplate** — add a clear header distinguishing docs from specs (docs = post-implementation truth)
- **Update `fab/.kit/skills/fab-init.md`** — add a step to create `fab/specs/index.md` during bootstrap (idempotent, like docs)
- **Update `fab/.kit/skills/_context.md`** — add `fab/specs/index.md` to the "Always Load" context layer so agents are aware of the specs landscape
- **Future consideration noted** — reverse-hydration (docs → specs) is a future concern, not in scope here. Specs are human-curated for now.

## Affected Docs

### New Docs
- `fab-workflow/specs-index`: Documentation for the new `fab/specs/` directory, its purpose, and its relationship to `fab/docs/`

### Modified Docs
- `fab-workflow/init`: Update to reflect the new `fab/specs/index.md` creation step
- `fab-workflow/context-loading`: Update to include `fab/specs/index.md` in the Always Load layer

### Removed Docs
(none)

## Impact

- **`fab/.kit/skills/fab-init.md`** — new step 1c-bis for `fab/specs/index.md`
- **`fab/.kit/skills/_context.md`** — add `fab/specs/index.md` to Always Load
- **`fab/.kit/templates/proposal.md`** — potentially add "Affected Specs" section (parallel to Affected Docs)
- **`fab/constitution.md`** — may need a new principle or amendment about specs vs docs distinction
- **Future commands** — this change surfaces the need for a reverse-hydration command but does NOT implement it

## Open Questions

- ~~[BLOCKING] Should `fab/specs/` mirror the domain structure of `fab/docs/`?~~ **Resolved**: No prescribed structure. Keep it flat, let humans organize as they see fit.
- [DEFERRED] Should the proposal template gain an "Affected Specs" section alongside "Affected Docs"? Or is one "Affected Artifacts" section sufficient?
- ~~[DEFERRED] What triggers hydration from docs back to specs?~~ **Resolved**: Out of scope. Specs are human-curated for now. Reverse-hydration (`fab-hydrate-specs`) is a future consideration when the pattern is better understood.
