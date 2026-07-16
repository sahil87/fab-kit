---
type: memory
# description: a curated ONE-LINE index-row summary — a routing signal, not a summary of record.
# Cap: 500 characters (FKF §3.2). Keep it a single line; detail (requirements, design decisions,
# history) belongs in the BODY sections below, NEVER in the description. `fab memory-index` emits an
# advisory warning over the cap.
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
     change history lives in the per-folder generated `log.md` (§6). Strip these
     guidance comments on fill. -->

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
