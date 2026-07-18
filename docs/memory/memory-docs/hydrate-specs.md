---
type: memory
description: "`/docs-hydrate-specs` skill — structural gap detection between memory and specs, interactive propose-then-apply, incl. the no-target new-spec-file branch and aligned prompt/handler tokens"
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

**Prompt/handler token alignment (d9rs)**: the Step 6 handler accepts exactly the tokens the Step 5 prompt offers — `yes` / `no` / `done` — with `skip rest` defined as an alias for `done`.

### Requirement: No-Target Branch (New Spec File)

When no existing spec file is a suitable home for a gap, the proposal SHALL target a **new** spec file instead — `**Target**: docs/specs/{kebab-topic}.md (new file)` — with the preview showing the full proposed file content, matching sibling specs' tone. The same per-gap confirmation gates it; on `yes` the skill creates the proposed file and adds its row to `docs/specs/index.md` (the one index edit this skill makes). Specs stay human-curated (Constitution VI).

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
