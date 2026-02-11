# Backfill

**Domain**: fab-workflow

## Overview

`/fab-backfill` detects structural gaps between `fab/docs/` and `fab/design/` — topics that docs cover but design doesn't mention at all — and proposes concise additions back to design with interactive per-gap confirmation.

## Requirements

### Requirement: Section-Level Gap Detection

The skill SHALL cross-reference docs and design at the section level (headings + inline mentions), not just file level. A doc topic is a structural gap only if no design file mentions it at all — neither as a heading nor as an inline reference.

### Requirement: Top-3 Cap with Impact Ranking

Output SHALL be capped at 3 gaps, ranked by impact: core behavioral rules and design decisions rank highest; implementation details rank lowest. When more gaps exist, the summary notes the overflow count.

### Requirement: Exact Markdown Preview with Per-Gap Confirmation

Each gap SHALL show the exact markdown that would be inserted, the source doc, and the target design file. The user confirms (yes), rejects (no), or stops (done) for each gap. Only confirmed additions are written.

### Requirement: No Active Change Required

The skill operates on project-level `fab/docs/` and `fab/design/` directories. It does not require `fab/current`, does not modify `.status.yaml`, and does not create git branches.

### Requirement: Pre-flight Checks

The skill SHALL verify `fab/docs/index.md` and `fab/design/index.md` exist before proceeding. Missing indexes abort with guidance to run `/fab-init`.

## Design Decisions

### Structural Gaps Only, Not Detail Enrichment
**Decision**: Only surface topics that docs cover but design doesn't mention at all — no detail-level diffing.
**Why**: Docs are intentionally verbose (machine-maintained). Design is intentionally concise (human-curated). A detail-level diff would surface everything, defeating the purpose.
**Rejected**: Detail-level comparison — would generate too many false positives and bloat design.
*Introduced by*: 260209-h3v7-fab-backfill

### Interactive Propose-Then-Apply Flow
**Decision**: Show exact markdown previews and confirm per-gap rather than batch-apply.
**Why**: Constitution principle VI says design is human-curated and MUST NOT be auto-generated. Per-gap confirmation keeps humans in control of design content and tone.
**Rejected**: Batch-apply with undo — too easy to accidentally bloat design.
*Introduced by*: 260209-h3v7-fab-backfill

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260209-h3v7-fab-backfill | 2026-02-09 | Initial creation — `/fab-backfill` skill for detecting and backfilling structural gaps from docs to design |
