# Proposal: Add "generate" mode to fab-hydrate

**Change**: 260207-k5od-hydrate-generate-mode
**Created**: 2026-02-07
**Status**: Draft

## Why

`/fab:hydrate` currently only handles one direction: ingesting external sources (Notion, Linear, local files) into `fab/docs/`. But many projects have significant undocumented behavior living only in source code — APIs, module boundaries, architectural patterns, conventions. There's no way to bootstrap documentation from what already exists in the codebase. A generate mode would let users point hydrate at their own code to produce docs, closing the gap between "code reality" and "documented knowledge." For large codebases, interactive scoping prevents overwhelming the user with a wall of generated docs they didn't ask for.

## What Changes

- **Unified argument-driven mode selection**: No flags. The type of argument determines behavior:
  - **URLs** (Notion, Linear) → ingest mode (existing behavior)
  - **Markdown files** (paths ending in `.md`) → ingest mode (existing behavior)
  - **Folders** (directory paths) → generate mode (scan code, produce docs)
  - **No arguments** → generate mode (scan from project root)
- **Codebase analysis engine**: Scans source code to identify undocumented areas — public APIs, modules, architectural patterns, configuration, conventions — by analyzing code structure and comparing against existing `fab/docs/`
- **Gap report with interactive scoping**: Scans from the target path (or project root), presents a prioritized list of discovered documentation gaps, lets the user batch-select which to document. Prevents overwhelming output for large codebases.
- **Doc generation from code**: Produces structured docs (Overview, Requirements, Design Decisions format) from code analysis, written to `fab/docs/` with proper domain mapping and index maintenance
- **Existing ingest behavior unchanged**: URLs and markdown file paths work identically to today

## Affected Docs

### New Docs
- `fab-workflow/hydrate-generate`: Requirements and behavior for the generate mode — scanning, gap detection, interactive scoping, doc generation

### Modified Docs
- `fab-workflow/hydrate`: Update argument handling to describe unified mode selection (URLs/md files → ingest, folders/no-args → generate); add cross-reference to hydrate-generate doc

### Removed Docs
(none)

## Impact

- **`fab/.kit/skills/fab-hydrate.md`**: Primary file modified — add generate mode, update argument parsing to route by type
- **`fab/docs/` index maintenance**: Generate mode reuses the same index update logic as ingest mode
- **No new dependencies**: Analysis is done by the AI agent reading code; no external tools required (Constitution I: Pure Prompt Play)
- **No breaking changes**: Existing `/fab:hydrate <url>` and `/fab:hydrate file.md` invocations work identically

## Open Questions

- [DEFERRED] Should generated docs include a `[GENERATED]` marker to distinguish them from manually-written or ingested docs? Could help with future re-generation, but adds visual noise.
- [DEFERRED] Should the gap report persist to a file (e.g., `fab/docs/.gaps.md`) for later reference, or is it purely interactive?
- [DEFERRED] Edge case: `/fab:hydrate ./legacy-docs/` (folder of markdown to ingest, not scan). Workaround: pass individual files or globs. Could add heuristic later if this becomes friction.
