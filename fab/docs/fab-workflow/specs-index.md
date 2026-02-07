# Specs Index

**Domain**: fab-workflow

## Overview

`fab/specs/index.md` is the centralized index for pre-implementation specification artifacts. It complements `fab/docs/index.md` (post-implementation truth) by providing a persistent home for conceptual design intent — what was planned, the "why" behind features.

## Requirements

### Specs vs Docs Distinction

Specs are pre-implementation artifacts — what you planned. They capture conceptual design intent, high-level decisions, and the "why" behind features. Docs are post-implementation artifacts — what actually happened, the authoritative source of truth for system behavior.

- `fab/specs/index.md` boilerplate clearly states specs are pre-implementation / planning artifacts
- `fab/docs/index.md` boilerplate clearly states docs are post-implementation / authoritative truth
- Both index files cross-reference each other with relative links

### Flat Structure

The specs index does not prescribe a domain-based directory hierarchy. Specs may be organized by the human in any structure they choose. The index simply lists what exists.

### Human-Curated Ownership

Specs are written and maintained by humans. No automated tooling creates or enforces structure in `fab/specs/`. No automated hydration from docs into specs exists (reverse-hydration is a future consideration).

### Bootstrap Integration

`/fab-init` creates `fab/specs/index.md` as step 1d during structural bootstrap (after docs/index.md at step 1c). The creation is idempotent — if the file already exists, init skips it with a status message.

### Context Loading Integration

`fab/specs/index.md` is included in the "Always Load" context layer in `_context.md`, alongside `config.yaml`, `constitution.md`, and `fab/docs/index.md`. This gives every skill baseline awareness of the specifications landscape.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260207-bb1q-add-specs-index | 2026-02-07 | Initial creation — added `fab/specs/` directory, specs index with boilerplate, bootstrap and context loading integration |
