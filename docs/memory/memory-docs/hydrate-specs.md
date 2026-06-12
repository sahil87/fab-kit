---
description: "`/docs-hydrate-specs` skill — structural gap detection between memory and specs, interactive propose-then-apply (incl. the no-target new-spec-file branch and aligned prompt/handler tokens — d9rs)"
---
# Hydrate Specs

**Domain**: memory-docs

## Overview

`/docs-hydrate-specs` detects structural gaps between `docs/memory/` and `docs/specs/` — topics that memory covers but specs don't mention at all — and proposes concise additions back to specs with interactive per-gap confirmation.

## Requirements

### Requirement: Section-Level Gap Detection

The skill SHALL cross-reference memory and specs at the section level (headings + inline mentions), not just file level. A memory topic is a structural gap only if no spec file mentions it at all — neither as a heading nor as an inline reference.

### Requirement: Top-3 Cap with Impact Ranking

Output SHALL be capped at 3 gaps, ranked by impact: core behavioral rules and key decisions rank highest; implementation details rank lowest. When more gaps exist, the summary notes the overflow count.

### Requirement: Exact Markdown Preview with Per-Gap Confirmation

Each gap SHALL show the exact markdown that would be inserted, the source memory file, and the target spec file. The user confirms (yes), rejects (no), or stops (done) for each gap. Only confirmed additions are written.

**Prompt/handler token alignment (d9rs)**: the Step 6 handler accepts exactly the tokens the Step 5 prompt offers — `yes` / `no` / `done` — with `skip rest` defined as an alias for `done`. Before d9rs the handler handled a "skip rest" token the prompt never offered.

### Requirement: No-Target Branch (New Spec File)

When no existing spec file is a suitable home for a gap, the proposal SHALL target a **new** spec file instead — `**Target**: docs/specs/{kebab-topic}.md (new file)` — with the preview showing the full proposed file content, matching sibling specs' tone. The same per-gap confirmation gates it; on `yes` the skill creates the proposed file and adds its row to `docs/specs/index.md` (the one index edit this skill makes). Specs stay human-curated (Constitution VI). Before d9rs, Step 5 had no branch for a gap with no suitable target.

### Requirement: No Active Change Required

The skill operates on project-level `docs/memory/` and `docs/specs/` directories. It does not require an active change (`.fab-status.yaml`), does not modify `.status.yaml`, and does not create git branches.

### Requirement: Pre-flight Checks

The skill SHALL verify `docs/memory/index.md` and `docs/specs/index.md` exist before proceeding. Missing indexes abort with guidance to run `/fab-setup`.

## Design Decisions

### Structural Gaps Only, Not Detail Enrichment
**Decision**: Only surface topics that memory covers but specs don't mention at all — no detail-level diffing.
**Why**: Memory files are intentionally verbose (machine-maintained). Specs are intentionally concise (human-curated). A detail-level diff would surface everything, defeating the purpose.
**Rejected**: Detail-level comparison — would generate too many false positives and bloat specs.
*Introduced by*: 260209-h3v7-fab-backfill

### Interactive Propose-Then-Apply Flow
**Decision**: Show exact markdown previews and confirm per-gap rather than batch-apply.
**Why**: Constitution principle VI says specs are human-curated and MUST NOT be auto-generated. Per-gap confirmation keeps humans in control of spec content and tone.
**Rejected**: Batch-apply with undo — too easy to accidentally bloat specs.
*Introduced by*: 260209-h3v7-fab-backfill

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260612-d9rs-docs-reality-sweep | 2026-06-12 | **No-target branch added** (skills-audit batch 5/5, Theme 8): when no existing spec file suits a gap, Step 5 proposes a new `docs/specs/{kebab-topic}.md` (full-content preview, sibling tone); on `yes` the file is created and its row added to `docs/specs/index.md` — still gated per-gap, specs stay human-curated. **Prompt/handler tokens aligned**: handler accepts exactly `yes`/`no`/`done` (the tokens Step 5 offers), `skip rest` demoted to a `done` alias (previously handled but never offered). |
| 260218-5isu-fix-docs-consistency-drift | 2026-02-18 | Replaced stale `/fab-init` → `/fab-setup` in pre-flight check guidance |
| 260214-m3v8-relocate-docs-dev-scripts | 2026-02-14 | Updated path references from `fab/memory/` and `fab/specs/` to `docs/memory/` and `docs/specs/` |
| 260209-h3v7-fab-backfill | 2026-02-09 | Initial creation — `/docs-hydrate-specs` skill for detecting and hydrating structural gaps from memory to specs |
| 260212-akhp-rename-fab-backfill | 2026-02-12 | Renamed from `/fab-backfill` to `/docs-hydrate-specs` for semantic consistency with `/fab-hydrate` |
