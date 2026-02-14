# Context Loading

**Domain**: fab-workflow

## Overview

The context loading convention defines how fab skills load project context before execution. It is implemented in `fab/.kit/skills/_context.md` as a shared preamble read by all skills. The convention uses a layered approach: always-load essentials, change-specific artifacts, and selective domain memory loading.

## Requirements

### Always Load Layer

Every skill (except `/fab-init`, `/fab-switch`, `/fab-status`, `/docs-hydrate-memory`) reads four files as baseline context:

1. `fab/config.yaml` — project configuration, tech stack, naming conventions
2. `fab/constitution.md` — project principles and constraints (MUST/SHOULD/MUST NOT rules)
3. `docs/memory/index.md` — documentation landscape (which domains and memory files exist)
4. `docs/specs/index.md` — specifications landscape (pre-implementation design intent, human-curated)

This gives the agent awareness of project configuration, constraints, the documentation landscape, and the specifications landscape before generating any artifact.

### Preflight Script for Change Context

Skills that operate on an active change resolve the change context by running `fab/.kit/scripts/_preflight.sh [change-name]` via Bash. The script accepts an optional first positional argument as a change name override. When provided, the script resolves the change using case-insensitive substring matching against folder names in `fab/changes/` (excluding `archive/`) instead of reading `fab/current`. The override is transient — `fab/current` is never modified. When no argument is provided, the script falls back to reading `fab/current` (backward compatible).

The matching supports full folder names, partial slug matches, and 4-char random IDs (e.g., `r3m7`). Exact match takes priority; single partial match resolves directly; multiple matches or no match produce a non-zero exit with a descriptive error.

The script validates project initialization, the change directory, and `.status.yaml`, then outputs structured YAML with name, stage, branch, progress, and checklist fields. On non-zero exit, the agent stops and surfaces the stderr error message. On success, the agent uses the stdout YAML instead of re-reading `.status.yaml`.

Since the preflight script validates `config.yaml` and `constitution.md` existence, skills using preflight don't need separate existence checks for these files — they only need to read them for content.

The existing 4-step inline validation sequence (check current, check directory, check .status.yaml, check config/constitution) remains documented in `_context.md` as reference for what the script validates internally.

### Selective Domain Loading

When operating on an active change, skills selectively load relevant memory files based on the change's scope:

1. Read the brief's Affected Memory section (or spec's Affected memory metadata) to identify relevant domains
2. For each referenced domain, read `docs/memory/{domain}/index.md`
3. For each specific file referenced, read `docs/memory/{domain}/{name}.md`
4. If a referenced domain or file doesn't exist yet, note this and proceed without error (it will be created during hydrate)
5. Do not load unrelated domains — keeps context focused and efficient

This applies to all skills operating on an active change, not just spec-writing skills.

### SRAD Protocol

The shared context preamble (`_context.md`) includes the SRAD autonomy framework, which all planning skills reference during artifact generation. The framework defines:
- **SRAD scoring table** — four dimensions for evaluating decision points
- **Confidence grades** — Certain, Confident, Tentative, Unresolved with corresponding artifact markers
- **Worked examples** — demonstrating how dimensions interact to produce grades
- **Artifact markers** — `<!-- assumed: ... -->` for Tentative, `<!-- clarified: ... -->` for resolved assumptions
- **Assumptions Summary Block** — standard format for output and artifact persistence

This protocol is loaded as part of the "Always Load" layer via `_context.md` and does not require separate file loading.

### Exception Skills

The following skills skip the standard context loading layers:
- `/fab-init` — bootstraps structure, doesn't need project memory
- `/fab-switch` — navigation only
- `/fab-status` — read-only status display
- `/docs-hydrate-memory` — ingests sources, doesn't need to load them first

## Design Decisions

### Smart Loading for All Skills on Active Changes
**Decision**: Expanded "Memory Lookup" from spec-writing-only to all skills operating on an active change.
**Why**: Agents need domain awareness for planning, implementation, and review — not just spec writing.
**Rejected**: Per-skill opt-in — too much maintenance overhead and easy to miss new skills.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Always Load docs/specs/index.md
**Decision**: Added `docs/specs/index.md` to the "Always Load" layer as a 4th baseline file.
**Why**: Gives every skill awareness of the specifications landscape (pre-implementation design intent) alongside the documentation landscape. The index is lightweight and human-curated, so context cost is minimal.
**Rejected**: Loading design index only when relevant — same inconsistency risk as with memory/index.md.
*Introduced by*: 260207-bb1q-add-specs-index

### Always Load docs/memory/index.md
**Decision**: Added `docs/memory/index.md` to the "Always Load" layer alongside config.yaml and constitution.md.
**Why**: Gives every skill baseline awareness of the documentation landscape. The index is lightweight (a table of domains), so the context cost is minimal.
**Rejected**: Loading only when needed — would require each skill to independently decide, leading to inconsistency.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260214-m3v8-relocate-docs-dev-scripts | 2026-02-14 | Updated always-load paths to `docs/memory/index.md` and `docs/specs/index.md`; updated selective domain loading to `docs/memory/{domain}/` |
| 260213-w4k9-explicit-change-targeting | 2026-02-13 | Preflight script now accepts optional `$1` change-name override with case-insensitive substring matching; `fab/current` is not modified when override is used |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Updated docs/specs/index.md reference, removed plan.md from artifact loading |
| 260208-k3m7-add-fab-fff | 2026-02-08 | Fixed stale `auto-guess` marker reference in SRAD Protocol section — replaced with `clarified` marker per updated `_context.md` |
| 260207-09sj-autonomy-framework | 2026-02-08 | Added SRAD protocol section — framework is loaded via `_context.md` as part of Always Load layer |
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260207-5mjv-preflight-grep-scripts | 2026-02-07 | Added preflight script integration — Change Context layer now uses `_preflight.sh` for validation and state resolution |
| 260207-bb1q-add-specs-index | 2026-02-07 | Added `docs/specs/index.md` as 4th file in Always Load layer |
| 260207-q7m3-separate-hydrate-smart-context | 2026-02-07 | Added `docs/memory/index.md` to always-load, expanded selective loading to all skills on active changes |
