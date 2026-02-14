# Hydrate: Generate Mode

**Domain**: fab-workflow

## Overview

`/docs-hydrate-memory` supports a generate mode that scans the codebase for undocumented areas, presents an interactive gap report, and generates structured documentation into `fab/memory/`. Generate mode is triggered when no arguments are provided (scans project root) or when folder paths are passed as arguments (scans those folders). It complements ingest mode, which handles URLs and `.md` files.

## Requirements

### Unified Argument Routing

`/docs-hydrate-memory` SHALL determine its operating mode from the type of arguments provided, with no flags or subcommands:

| Argument type | Detection rule | Mode |
|---|---|---|
| No arguments | Argument list is empty | Generate (scan from project root) |
| URL | Matches `notion.so`, `notion.site`, `linear.app`, or `http(s)://` | Ingest |
| Markdown file | Path ends with `.md` | Ingest |
| Folder | Path resolves to an existing directory | Generate |

When multiple arguments are provided, they MUST all resolve to the same mode. Mixed-mode invocations SHALL be rejected with: "Cannot mix ingest sources (URLs, .md files) with generate targets (folders). Run separately."

### No-Args Replaces Usage Error

When `/docs-hydrate-memory` is invoked with no arguments, it SHALL enter generate mode instead of displaying a usage error. The previous "Usage: /fab-hydrate ..." abort behavior is removed.

### Codebase Gap Detection

In generate mode, the skill SHALL scan source code to identify undocumented areas by comparing codebase structure against existing `fab/memory/`. The scan MUST identify:

- **Modules**: Top-level directories and packages with distinct responsibilities
- **APIs**: Exported functions, classes, endpoints, CLI commands
- **Patterns**: Recurring architectural patterns (middleware chains, plugin systems, event buses, etc.)
- **Configuration**: Config files, environment variables, feature flags
- **Conventions**: Naming patterns, file organization, coding standards evident from code

The scan SHALL use file system exploration (Glob, Grep, Read) and SHALL NOT require external tools or dependencies (Constitution I: Pure Prompt Play).

### Scan Scope

When folder paths are provided, the scan SHALL be limited to those paths. When no arguments are provided, the scan SHALL start from the project root. The scan SHOULD respect common ignore patterns (`.git/`, `node_modules/`, `vendor/`, `__pycache__/`, `dist/`, `build/`).

### Gap Report Presentation

After scanning, the skill SHALL present a categorized, prioritized gap report. Each gap MUST include:

- **Category**: Module, API, Pattern, Configuration, or Convention
- **Name**: Human-readable identifier
- **Location**: File paths or directory paths involved
- **Priority**: High (core functionality, public API), Medium (internal modules, patterns), Low (utilities, config)

The report SHALL be grouped by category and sorted by priority within each category.

### Interactive Selection

After presenting the gap report, the skill SHALL offer batch selection via AskUserQuestion with four strategy options:

1. **All** — document everything found
2. **All High priority** — document only High priority gaps
3. **Select by number** — user types gap numbers
4. **Select by category** — user types category names

If only 1-3 gaps are found, the skill MAY skip the interactive prompt and proceed with a brief confirmation. If zero gaps are found, the skill SHALL report "No documentation gaps found" and exit cleanly.

### Structured Memory Output

For each selected gap, the skill SHALL generate a memory file in `fab/memory/{domain}/{topic}.md` following the memory file format (Overview, Requirements with RFC 2119 keywords, Design Decisions, Changelog). Generated files SHALL synthesize one file per gap (not per source file). When behavior is ambiguous, files SHOULD include `[INFERRED]` markers inline with explanations.

### Index Maintenance

Generate mode SHALL reuse the same index maintenance logic as ingest mode:

1. Create or update `fab/memory/{domain}/index.md` for each domain touched
2. Update `fab/memory/index.md` with new domains and file lists
3. All links SHALL be relative
4. Existing entries SHALL NOT be removed

### Idempotent Generation

Generate mode SHALL be safe to re-run:

- Existing generated files SHALL be updated (merged), not overwritten
- Manually-added content SHALL be preserved
- New gaps discovered since last run SHALL appear in the gap report
- Previously documented areas SHALL NOT appear as gaps (or appear with lower priority)

## Design Decisions

### Scan Strategy: Structural Heuristics, Not AST Parsing
**Decision**: Use file system exploration (Glob, Grep, Read) with structural heuristics for gap detection. Analyze directory layout, exports, entry points, and naming conventions.
**Why**: Constitution I (Pure Prompt Play) forbids system dependencies. Structural heuristics are language-agnostic and require no external tooling.
**Rejected**: Language-specific AST parsers (tree-sitter, etc.) — would require binary dependencies and per-language configuration.
*Introduced by*: 260207-k5od-hydrate-generate-mode

### Gap Detection: Multi-Signal Heuristics by Category
**Decision**: Use different detection signals per category (directory matching for modules, grep patterns for APIs, occurrence counting for patterns, glob patterns for config, naming analysis for conventions).
**Why**: Different gap categories require different detection signals. Directory-to-domain comparison alone misses APIs, patterns, and config that don't map 1:1 to directories.
**Rejected**: Content-based similarity matching (too slow, too fragile). Pure directory-only matching (misses non-directory-mapped gaps).
*Introduced by*: 260207-k5od-hydrate-generate-mode

### Interactive Scoping: Two-Step Display Then Select
**Decision**: Display the full gap report as formatted text (numbered), then use AskUserQuestion with 4 strategy options. For 1-3 gaps, skip the selection UI and confirm directly.
**Why**: AskUserQuestion supports max 4 options, so individual gaps can't be listed as options. The two-step approach (display report, then ask strategy) handles any number of gaps.
**Rejected**: Listing individual gaps as AskUserQuestion options (tool limited to 4). Multi-step wizard (too many round-trips).
*Introduced by*: 260207-k5od-hydrate-generate-mode

### Memory Generation: One File Per Gap
**Decision**: Synthesize one memory file per gap (e.g., one file for an entire module), not one file per source file.
**Why**: Matches how humans think about documentation — by domain, not by file. Prevents fragmentation into dozens of small files.
**Rejected**: Per-source-file generation — would fragment knowledge and be hard to navigate.
*Introduced by*: 260207-k5od-hydrate-generate-mode

### `[INFERRED]` Markers on Uncertain Behaviors
**Decision**: Mark ambiguous or uncertain behaviors with `[INFERRED]` tags inline, close to the relevant requirement, with explanations suggesting verification.
**Why**: Generated files are the agent's best understanding, not verified specs. Inline markers give clear signals about what to verify.
**Rejected**: No markers (too risky). Separate "uncertainties" section (disconnects marker from content).
*Introduced by*: 260207-k5od-hydrate-generate-mode

### Argument Classification: Detect at Parse Time, Reject Mixed Modes
**Decision**: Classify arguments at parse time (URL pattern → ingest, `.md` extension → ingest, folder → generate, no args → generate). Reject mixed-mode invocations with a clear error.
**Why**: The two modes have fundamentally different pipelines. Rejecting mixed args is explicit and clear rather than undefined.
**Rejected**: Merging ingest and generate in one pass — too complex, unclear semantics.
*Introduced by*: 260207-k5od-hydrate-generate-mode

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260207-sawf-fix-command-format | 2026-02-07 | Fixed command references from `/fab-xxx` colon format to `/fab-xxx` hyphen format |
| 260207-k5od-hydrate-generate-mode | 2026-02-07 | Created — generate mode requirements and design decisions for `/docs-hydrate-memory` |
