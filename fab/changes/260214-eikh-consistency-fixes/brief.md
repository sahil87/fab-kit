# Brief: Consistency Fixes from 260214 Audit

**Change**: 260214-eikh-consistency-fixes
**Created**: 2026-02-14
**Status**: Draft

## Origin

> Backlog items cf01–cf15 under "Consistency Fixes (from 260214 audit)" in fab/backlog.md. These are documentation/spec drift corrections identified during an audit — no behavioral code changes.

## Why

Specs and memory files have drifted from the implemented reality after several rounds of renames (archive→hydrate stage, fab-hydrate→docs-hydrate-memory, fab-apply/fab-review→fab-continue, fab-init sub-skills removed). Stale references confuse agents that consult specs for context, and orphaned entries in memory index point to non-existent files. Fixing these in a single pass restores specs-to-implementation alignment.

## What Changes

### Fix (actively wrong references)

- **cf01**: Replace "archive" stage references with "hydrate" in overview.md, glossary.md, skills.md, templates.md — stage 6 is "hydrate", `/fab-archive` is a separate housekeeping command
- **cf02**: Rename `/fab-hydrate` → `/docs-hydrate-memory` across all spec files (~15+ occurrences in overview.md, skills.md, glossary.md, architecture.md)
- **cf03**: Update workflow.yaml lines 90 and 100 to reference `fab-continue` instead of `fab-apply`/`fab-review`
- **cf04**: Fix `SKILLS.md` → `skills.md` and `TEMPLATES.md` → `templates.md` case in cross-references (architecture.md:400, skills.md:199)
- **cf05**: Fix symlink reference `fab-hydrate.md` → `docs-hydrate-memory.md` in architecture.md:362

### Add (missing coverage)

- **cf06**: Document `/fab-archive` as a standalone skill — currently the skills.md section (lines 443-468) titles it "Archive Behavior (via `/fab-continue`)" which conflates hydrate-stage behavior with the separate archive command
- **cf07**: Add behavioral spec sections for `/docs-reorg-memory` and `/docs-reorg-specs` — both are user-facing skills with no spec coverage
- **cf08**: Document batch scripts (batch-archive-change.sh, batch-new-backlog.sh, batch-switch-change.sh) in architecture.md overview
- **cf09**: Add internal skills (internal-consistency-check.md, internal-retrospect.md, internal-skill-optimize.md) to kit-architecture.md directory listing

### Remove (stale/orphaned)

- **cf11**: Remove `/fab-init-config`, `/fab-init-constitution`, `/fab-init-validate` sub-skill documentation (skills.md:119-150, architecture.md:21-23) — only monolithic `/fab-init` exists
- **cf12**: Remove `hydrate-design` and `design-index` from memory/index.md:14 — these files don't exist
- **cf13**: Clean up contradictory changelog entries about fab-status.sh/stageman.sh in execution-skills.md and kit-architecture.md (may already be resolved — verify during apply)

### Rename (terminology alignment)

- **cf14**: Add `docs-reorg-*` skills to glossary.md as "user-facing" (currently absent entirely)
- **cf15**: Expand "Hydration" glossary definition to cover dual-mode behavior (ingest + generate) per memory/hydrate.md

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Add internal skills to directory listing (cf09)
- `fab-workflow/execution-skills`: (modify) Verify/clean changelog entries (cf13)

## Impact

- **Spec files**: overview.md, glossary.md, skills.md, templates.md, architecture.md — bulk text corrections
- **Schema**: workflow.yaml — command references updated
- **Memory**: kit-architecture.md, execution-skills.md — minor corrections
- **No code changes**: All fixes are documentation/spec/schema only
- **No behavioral impact**: No skill files, shell scripts, or templates are modified

## Open Questions

None — all items are well-scoped mechanical corrections from a completed audit.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | cf13 may already be resolved — will verify during apply and skip if clean | Subagent scan found no contradictions currently; changelog may have been fixed in a prior change |
| 2 | Confident | cf14 reframed as "add to glossary" rather than "reclassify" since docs-reorg-* is entirely absent from glossary | Scan confirmed zero glossary entries for these skills |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.
