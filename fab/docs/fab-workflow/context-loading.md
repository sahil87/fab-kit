# Context Loading

**Domain**: fab-workflow

## Overview

The context loading convention defines how fab skills load project context before execution. It is implemented in `fab/.kit/skills/_context.md` as a shared preamble read by all skills. The convention uses a layered approach: always-load essentials, change-specific artifacts, and selective domain doc loading.

## Requirements

### Always Load Layer

Every skill (except `/fab:init`, `/fab:switch`, `/fab:status`, `/fab:hydrate`) reads three files as baseline context:

1. `fab/config.yaml` — project configuration, tech stack, naming conventions
2. `fab/constitution.md` — project principles and constraints (MUST/SHOULD/MUST NOT rules)
3. `fab/docs/index.md` — documentation landscape (which domains and docs exist)

This gives the agent awareness of project configuration, constraints, and the documentation landscape before generating any artifact.

### Selective Domain Loading

When operating on an active change, skills selectively load relevant domain docs based on the change's scope:

1. Read the proposal's Affected Docs section (or spec's Affected docs metadata) to identify relevant domains
2. For each referenced domain, read `fab/docs/{domain}/index.md`
3. For each specific doc referenced, read `fab/docs/{domain}/{name}.md`
4. If a referenced domain or doc doesn't exist yet, note this and proceed without error (it will be created by archive)
5. Do not load unrelated domains — keeps context focused and efficient

This applies to all skills operating on an active change, not just spec-writing skills.

### Exception Skills

The following skills skip the standard context loading layers:
- `/fab:init` — bootstraps structure, doesn't need project docs
- `/fab:switch` — navigation only
- `/fab:status` — read-only status display
- `/fab:hydrate` — ingests docs, doesn't need to load them first

## Design Decisions

### Smart Loading for All Skills on Active Changes
**Decision**: Expanded "Centralized Doc Lookup" from spec-writing-only to all skills operating on an active change.
**Why**: Agents need domain awareness for planning, implementation, and review — not just spec writing.
**Rejected**: Per-skill opt-in — too much maintenance overhead and easy to miss new skills.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Always Load fab/docs/index.md
**Decision**: Added `fab/docs/index.md` to the "Always Load" layer alongside config.yaml and constitution.md.
**Why**: Gives every skill baseline awareness of the documentation landscape. The index is lightweight (a table of domains), so the context cost is minimal.
**Rejected**: Loading only when needed — would require each skill to independently decide, leading to inconsistency.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260207-q7m3-separate-hydrate-smart-context | 2026-02-07 | Added `fab/docs/index.md` to always-load, expanded selective loading to all skills on active changes |
