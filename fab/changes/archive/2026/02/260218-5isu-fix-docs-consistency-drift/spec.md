# Spec: Fix Documentation Consistency Drift

**Change**: 260218-5isu-fix-docs-consistency-drift
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/context-loading.md`, `docs/memory/fab-workflow/hydrate.md`, `docs/memory/fab-workflow/hydrate-specs.md`, `docs/memory/fab-workflow/specs-index.md`, `docs/memory/fab-workflow/model-tiers.md`, `docs/memory/fab-workflow/templates.md`, `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/migrations.md`

## Non-Goals

- Implementation changes — all code is already correct; only docs/specs/memory are stale
- Content rewrites beyond the specific corrections identified — no editorial improvements

## Specs: Stale `/fab-init` References

### Requirement: Replace `/fab-init` with `/fab-setup` in All Spec Files

All occurrences of `/fab-init` in `docs/specs/` SHALL be replaced with `/fab-setup`. The stale command name exists across 6 spec files: `glossary.md`, `architecture.md`, `overview.md`, `skills.md`, `user-flow.md`, and `templates.md`.

#### Scenario: Grep for `/fab-init` in specs after fix

- **GIVEN** all spec files in `docs/specs/` have been updated
- **WHEN** searching for the string `/fab-init` across `docs/specs/`
- **THEN** zero matches are found

#### Scenario: `/fab-setup` references are present

- **GIVEN** all `/fab-init` references have been replaced
- **WHEN** searching for `/fab-setup` across `docs/specs/`
- **THEN** matches exist in the files that previously referenced `/fab-init`

### Requirement: Rewrite `/fab-init` Section in `skills.md` as `/fab-setup`

The `docs/specs/skills.md` section currently documenting `/fab-init` (approximately lines 72-104) SHALL be replaced with a `/fab-setup` section documenting three subcommands:

- `/fab-setup config [section]` — create/update `fab/config.yaml`
- `/fab-setup constitution` — create/amend `fab/constitution.md`
- `/fab-setup migrations [file]` — run version migrations

The rewrite SHALL use `docs/memory/fab-workflow/setup.md` as the source of truth for current behavior.

#### Scenario: skills.md contains `/fab-setup` documentation

- **GIVEN** `docs/specs/skills.md` has been rewritten
- **WHEN** reading the `/fab-setup` section
- **THEN** three subcommands are documented: `config`, `constitution`, `migrations`
- **AND** no references to `/fab-init` remain in the section

### Requirement: Replace "briefs" with "intakes" in `architecture.md`

The string `"change artifacts (briefs, specs, tasks)"` in `docs/specs/architecture.md` SHALL be replaced with `"change artifacts (intakes, specs, tasks)"`.

#### Scenario: No "briefs" references remain

- **GIVEN** `docs/specs/architecture.md` has been updated
- **WHEN** searching for the word "briefs" in the file
- **THEN** zero matches are found

## Specs: `_init_scaffold.sh` Removal

### Requirement: Remove or Replace `_init_scaffold.sh` References in `architecture.md`

All references to `_init_scaffold.sh` in `docs/specs/architecture.md` SHALL be removed or replaced with documentation of the `fab/.kit/scaffold/` directory approach. The implementation now uses `fab-sync.sh` with scaffold files, not `_init_scaffold.sh`.

#### Scenario: No `_init_scaffold.sh` references remain

- **GIVEN** `docs/specs/architecture.md` has been updated
- **WHEN** searching for `_init_scaffold.sh` across `docs/specs/`
- **THEN** zero matches are found

## Specs: `/fab-update` Removal

### Requirement: Remove `/fab-update` from User Flow Diagram

The `/fab-update` node in the Mermaid diagram in `docs/specs/user-flow.md` SHALL be removed. `/fab-setup migrations` is a maintenance subcommand, not a diagram-level workflow node.

#### Scenario: No `/fab-update` node in diagram

- **GIVEN** `docs/specs/user-flow.md` has been updated
- **WHEN** reading the Mermaid diagram
- **THEN** no `UPDATE` or `/fab-update` node exists

### Requirement: Replace `/fab-update` with `/fab-setup migrations` in `user-flow.md` Prose

Any prose references to `/fab-update` in `docs/specs/user-flow.md` outside the diagram SHALL be replaced with `/fab-setup migrations`.

#### Scenario: No prose `/fab-update` references

- **GIVEN** `docs/specs/user-flow.md` has been updated
- **WHEN** searching for `/fab-update` in the file
- **THEN** zero matches are found

## Specs: Template Accuracy

### Requirement: Fix Stage Name in `.status.yaml` Template Spec

In `docs/specs/templates.md`, the `.status.yaml` documentation SHALL use `hydrate` instead of `archive` in the progress map. Both the keys list and any template example showing `archive: pending` SHALL be corrected to `hydrate: pending`.

#### Scenario: Template spec uses `hydrate` stage name

- **GIVEN** `docs/specs/templates.md` has been updated
- **WHEN** reading the `.status.yaml` template documentation
- **THEN** the progress map shows `hydrate:` not `archive:`
- **AND** the keys list includes `hydrate` not `archive`

### Requirement: Add Missing `.status.yaml` Fields to Template Spec

The `.status.yaml` section in `docs/specs/templates.md` SHALL document three fields present in the actual template (`fab/.kit/templates/status.yaml`) but missing from the spec:

1. `change_type: feature` — the change type classification
2. `confidence:` block with sub-fields `certain`, `confident`, `tentative`, `unresolved`, `score`
3. `stage_metrics: {}` — empty initial stage metrics

#### Scenario: Template spec includes all fields from actual template

- **GIVEN** `docs/specs/templates.md` has been updated
- **WHEN** comparing the documented `.status.yaml` structure against `fab/.kit/templates/status.yaml`
- **THEN** `change_type`, `confidence` block, and `stage_metrics` are all documented

## Memory: Stale `/fab-init` References

### Requirement: Replace `/fab-init` with `/fab-setup` in Memory Files

All occurrences of `/fab-init` in `docs/memory/fab-workflow/` SHALL be replaced with `/fab-setup`. Affected files: `context-loading.md`, `hydrate.md`, `hydrate-specs.md`, `specs-index.md`.

#### Scenario: Grep for `/fab-init` in memory after fix

- **GIVEN** all memory files have been updated
- **WHEN** searching for `/fab-init` across `docs/memory/`
- **THEN** zero matches are found

## Memory: Stale Script Path References

### Requirement: Replace `lib/sync-workspace.sh` with `fab-sync.sh` in Memory Files

All occurrences of `lib/sync-workspace.sh` in `docs/memory/fab-workflow/` SHALL be replaced with the correct current path. Affected files: `hydrate.md`, `model-tiers.md`, `templates.md`.

Note: Some references may have already been corrected in recent changes. The apply stage SHALL verify actual file content before making replacements — only replace instances that are still stale.

#### Scenario: No stale `lib/sync-workspace.sh` references

- **GIVEN** all memory files have been checked and updated
- **WHEN** searching for `lib/sync-workspace.sh` across `docs/memory/`
- **THEN** zero matches are found (or all remaining instances correctly reference the path in context)

## Memory: Stale `/fab-update` Reference

### Requirement: Replace `/fab-update` with `/fab-setup migrations` in `migrations.md`

The reference to `/fab-update` in `docs/memory/fab-workflow/migrations.md` SHALL be replaced with `/fab-setup migrations`.

#### Scenario: No `/fab-update` in migrations memory

- **GIVEN** `docs/memory/fab-workflow/migrations.md` has been updated
- **WHEN** searching for `/fab-update` in the file
- **THEN** zero matches are found (note: the `/fab-update` entry in the Deprecated Requirements section of `setup.md` is correct and intentional — it documents the deprecated command)

## Memory: Kit Architecture Accuracy

### Requirement: Remove `model-tiers.yaml` from Directory Listing

The `model-tiers.yaml` entry in the directory tree in `docs/memory/fab-workflow/kit-architecture.md` SHALL be removed. This file was deleted in v0.8.0 when model tier configuration was consolidated into `config.yaml`.

#### Scenario: No `model-tiers.yaml` in directory tree

- **GIVEN** `docs/memory/fab-workflow/kit-architecture.md` has been updated
- **WHEN** reading the `.kit/` directory tree
- **THEN** no `model-tiers.yaml` entry exists

### Requirement: Add `fab-fff.md` to Skills Listing

`fab-fff.md` SHALL be added to the skills listing in the `.kit/` directory tree in `docs/memory/fab-workflow/kit-architecture.md`, positioned after `fab-ff.md` in alphabetical order.

#### Scenario: `fab-fff.md` appears in directory tree

- **GIVEN** `docs/memory/fab-workflow/kit-architecture.md` has been updated
- **WHEN** reading the skills listing in the directory tree
- **THEN** `fab-fff.md` appears after `fab-ff.md`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | No implementation changes needed | Confirmed from intake #1 — consistency check verified implementation is correct; only docs/specs/memory are stale | S:95 R:95 A:95 D:95 |
| 2 | Certain | Use memory as source of truth for `/fab-setup` rewrite in skills.md | Confirmed from intake #2 — Constitution II mandates memory as authoritative post-implementation source | S:90 R:90 A:95 D:90 |
| 3 | Confident | Remove `/fab-update` from user-flow diagram entirely rather than replacing | Confirmed from intake #3 — `/fab-setup migrations` is a maintenance subcommand, not a diagram-level user flow | S:70 R:85 A:80 D:70 |
| 4 | Confident | Apply stage should verify actual content before replacing | Line numbers may have shifted since the consistency check; content matching is safer than line-offset edits | S:75 R:90 A:85 D:80 |
| 5 | Certain | Deprecated command references in Deprecated Requirements sections are intentional | Memory files like `setup.md` correctly document `/fab-init` and `/fab-update` as deprecated items — these are not stale | S:90 R:95 A:90 D:95 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
