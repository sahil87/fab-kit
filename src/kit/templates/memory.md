---
type: memory
# description: a curated ONE-LINE index-row summary — a routing signal, not a summary of record.
# Cap: 500 characters (FKF §3.2). Keep it a single line; detail (requirements, design decisions,
# prose) belongs in the BODY sections below, NEVER in the description. NO change-ids in the
# description (no `— xu0k` suffix, no `(d9rs)` citation) — provenance citations belong in the body.
# `fab memory-index` warns (advisory) at 501–1000 chars and BLOCKS `--check` past 1000 chars
# (gross over-cap) or on any change-id in the description (FKF §3.2, enforced).
description: "{One-line summary used by the generated domain-index row — ≤500 chars.}"
---
# {File Name}

**Domain**: {domain}

<!-- Conventional memory-file shape. The full heading/scenario rules are the FKF
     contract — see `$(fab kit-path)/reference/fkf.md` §3.3 (Body). Headings are
     SHOULD-use-where-they-apply, NOT MUST-have: a file is conforming without any
     particular section, so a small reference-pointer file may legitimately omit a
     GIVEN/WHEN/THEN scenario. This template scaffolds the full shape; fill the
     sections the content warrants and delete the rest. NO `## Changelog` section —
     change history lives in the per-folder generated `log.md` (§6).

     BODY STYLE — state current truth in present tense (FKF §3.3). Describe what IS,
     not how it came to be: NO transition narration ("renamed X→Y in {id}", "this
     inverts {id}'s claim", "was `old.value`"), and NEVER describe superseded behavior
     — the previous state lives in `log.md`, git history, and archived change folders.
     Provenance in the body is citation-only: a trailing `(change-id)` and the
     `*Introduced by*: {change-name}` field on a Design Decision (kept below).

     HEADINGS CARRY NO CHANGE-IDS — a heading names its topic (`## Dispatch States`),
     never a change (`### Dispatch States (xu0k)`); change-ids stay citation-only in
     body text.

     NO OPERATIONAL TODOs — follow-up work items (TODOs, "still needs X", next-step
     checklists) belong in the backlog (`fab/backlog.md`) or the change folder, never a
     memory body. A body states what IS, not what remains to be done.

     RATIONALE → DESIGN DECISIONS — any why / rejected alternative / constraint goes into
     a `## Design Decisions` entry in the four-field shape (Decision / Why / Rejected /
     *Introduced by*, kept below), never inline narration. Rationale is NOT narration —
     Why/Rejected stay durable present-tense intent. The changelog-bullet shape
     (`- **{change-id} — retired X**`) is BANNED inside Design Decisions — that is change
     history (`log.md`'s job, §6), not a decision; a DD heading is a decision title.

     Strip these guidance comments on fill. -->

## Overview
<!-- 1-2 sentences describing what this file covers. -->

## Requirements
### Requirement: {Name}
{RFC 2119 text: MUST / SHALL / SHOULD / MAY}

#### Scenario: {Name}
- **GIVEN** {precondition}
- **WHEN** {action}
- **THEN** {expected outcome}

## Design Decisions
### {Decision Title}
**Decision**: {chosen approach}
**Why**: {rationale}
**Rejected**: {alternative and why it was worse}
*Introduced by*: {change-name}

<!-- Cross-links to other memory files use the bundle-relative form (FKF §7),
     resolved from `docs/memory/`:
       See [migrations](/distribution/migrations.md).
     Links OUT of the bundle (source, specs, URLs) stay repo-relative/absolute-URL. -->
