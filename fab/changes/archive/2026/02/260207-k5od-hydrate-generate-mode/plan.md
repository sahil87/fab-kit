# Plan: Add "generate" mode to fab-hydrate

**Change**: 260207-k5od-hydrate-generate-mode
**Created**: 2026-02-07
**Proposal**: `proposal.md`
**Spec**: `spec.md`

## Summary

Add a generate mode to `/fab:hydrate` that scans the codebase for undocumented areas, presents an interactive gap report, and generates structured docs into `fab/docs/`. Mode is determined by argument type — no flags needed.

## Goals / Non-Goals

**Goals:**
- Argument routing: no-args and folder paths → generate, URLs and `.md` files → ingest
- Codebase scanning that identifies modules, APIs, patterns, config, conventions
- Gap comparison against existing `fab/docs/` to avoid re-documenting known areas
- Interactive scoping UI for prioritized batch selection
- Doc output in centralized doc format with index maintenance
- Idempotent re-runs

**Non-Goals:**
- AST parsing or language-specific analysis — scan uses file structure and content heuristics, not parsers
- Automatic re-generation on code changes (no watch mode)
- Generating docs for external dependencies or third-party APIs
- Modifying the ingest mode behavior beyond removing the no-args error

## Technical Context

- **Relevant stack**: Markdown skill files (Constitution I: Pure Prompt Play), AI agent as executor
- **Key dependencies**: Glob, Grep, Read tools for codebase scanning; AskUserQuestion for interactive scoping
- **Constraints**: No external tools or binaries. All analysis is done by the agent reading files. Must be idempotent (Constitution III).

## Decisions

1. **Scan strategy: structural heuristics, not AST parsing**
   - *Why*: Constitution I (Pure Prompt Play) forbids system dependencies. The agent reads files with Glob/Grep/Read — parsing code structure from directory layout, exports, entry points, and naming conventions is sufficient and language-agnostic.
   - *Rejected*: Language-specific AST parsers (tree-sitter, etc.) — would require binary dependencies and per-language configuration.

2. **Gap detection: multi-signal heuristics by category** <!-- clarified: expanded beyond directory-only matching to cover all 5 spec categories -->
   - *Why*: Different gap categories require different detection signals. Directory-to-domain comparison is the backbone, but not sufficient alone for APIs, patterns, config, and conventions.
   - *How*:
     1. **Modules**: Enumerate top-level source directories (excluding standard ignores). For each, check if a matching domain exists in `fab/docs/index.md`. Unmatched → Module gap.
     2. **APIs**: Grep for route definitions, endpoint handlers, CLI command registrations, exported public interfaces (language-dependent patterns like `app.get(`, `@route`, `export function`, `def command`). Cross-reference against existing docs. Undocumented endpoints/exports → API gap.
     3. **Patterns**: Look for recurring structural patterns across the codebase — middleware chains, plugin directories, event handler registrations, factory functions, decorator usage. If a pattern appears 3+ times and has no doc, flag it.
     4. **Configuration**: Glob for config files (`.env*`, `*.config.*`, `config/`, `settings.*`), environment variable references (`process.env`, `os.environ`, `ENV[]`). Undocumented config → Configuration gap.
     5. **Conventions**: Analyze file naming patterns, directory structure conventions, common prefixes/suffixes. These are lowest priority and only flagged when the pattern is clear and consistent.
     6. For each gap, check `fab/docs/` domains and their doc entries — anything already covered is excluded or deprioritized.
   - *Rejected*: Content-based similarity matching between code and docs — too slow, too fragile. Pure directory-only matching — misses APIs, patterns, and config that don't map 1:1 to directories.

3. **Interactive scoping: two-step — display report, then ask selection strategy** <!-- clarified: AskUserQuestion limited to 2-4 options, can't list individual gaps -->
   - *Why*: AskUserQuestion supports max 4 options, so listing individual gaps as options won't work for larger reports. Instead: display the full gap report as formatted text (numbered), then ask the user *how* they want to select.
   - *Flow*:
     1. Present gap report as formatted text (numbered, grouped by category, with priorities)
     2. AskUserQuestion with selection strategy options: "All", "All High priority", "Select by number" (user types gap numbers in Other), "Select by category"
     3. If "Select by number" or "Select by category": user provides specifics via the Other text input
     4. Agent parses the selection and processes the chosen gaps
   - *Rejected*: Listing individual gaps as AskUserQuestion options — tool limited to 2-4 options. Multi-step wizard — too many round-trips.
   - *Edge case*: For 1-3 gaps, skip AskUserQuestion entirely — just confirm and proceed.

4. **Doc generation: one doc per gap, not one doc per file**
   - *Why*: A "module" gap (e.g., `src/auth/`) should produce one doc covering the module, not individual docs for every file in the module. This matches how humans think about documentation (by domain, not by file). The agent reads all files in the gap scope and synthesizes.
   - *Rejected*: Per-file doc generation — would produce dozens of small docs that fragment the knowledge. Hard to navigate.

5. **`[INFERRED]` markers on uncertain behaviors**
   - *Why*: Generated docs are the agent's best understanding of code behavior, not verified specs. Marking inferences gives the user clear signals about what to verify. Markers are inline, not in a separate section — keeps them close to the relevant requirement.
   - *Rejected*: No markers (trust the agent) — too risky for production docs. Separate "uncertainties" section — disconnects the marker from the content.

6. **Argument classification: detect at parse time, reject mixed modes** <!-- clarified: replaced fs.stat with agent-executable detection -->
   - *Why*: Simpler than trying to merge ingest and generate in one pass. The two modes have fundamentally different pipelines (fetch URL vs. scan directory). Rejecting mixed args is an explicit, clear error rather than undefined behavior.
   - *Classification rules*: URL pattern match (contains `://` or known domains) → ingest. Ends with `.md` → ingest. Otherwise, treat as folder path → generate. No args → generate. The agent verifies folder existence via Glob or Bash `ls` — if the path doesn't exist, report an error.

## Risks / Trade-offs

1. **Scan quality varies by codebase structure** — well-organized codebases (clear directories, good naming) will produce better gap reports than flat/chaotic structures. Mitigation: the interactive scoping lets users correct the agent's prioritization.

2. **Generated docs may be shallow** — the agent can only infer what's visible from code structure and content, not deep domain knowledge. Mitigation: `[INFERRED]` markers flag uncertainty; docs are a starting point, not the final word.

3. **Large codebases may produce overwhelming gap reports** — even with interactive scoping, 50+ gaps is hard to parse. Mitigation: group by category, sort by priority, and cap the displayed list at a reasonable number (e.g., top 20) with an option to see all.

4. **Folder-of-markdown ambiguity** — `/fab:hydrate ./legacy-docs/` (a folder of `.md` files meant for ingest) will trigger generate mode. Mitigation: document the workaround (pass individual files or globs). Revisit if users hit this in practice.

5. **Zero gaps found** — if the scan finds nothing undocumented (small project, or everything is already documented), generate mode should report cleanly and exit without presenting the selection UI. Not an error — just "No documentation gaps found." <!-- clarified: added missing zero-gaps scenario -->

## File Changes

### New Files
- `fab/docs/fab-workflow/hydrate-generate.md`: Centralized doc for generate mode (created by `/fab:archive` hydration, not manually)

### Modified Files
- `fab/.kit/skills/fab-hydrate.md`: Add argument routing logic, generate mode behavior (scanning, gap report, interactive scoping, doc generation), update purpose/description
- `fab/docs/fab-workflow/hydrate.md`: Add reference to generate mode, update overview to describe both modes (done by `/fab:archive` hydration)
- `fab/docs/fab-workflow/index.md`: Add entry for `hydrate-generate` doc (done by `/fab:archive` hydration)
- `fab/docs/index.md`: Update fab-workflow domain doc list (done by `/fab:archive` hydration)

### Deleted Files
(none)
