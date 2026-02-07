# Tasks: Add "generate" mode to fab-hydrate

**Change**: 260207-k5od-hydrate-generate-mode
**Plan**: `plan.md`
**Spec**: `spec.md`

## Phase 1: Setup

- [x] T001 Restructure `fab/.kit/skills/fab-hydrate.md` — update frontmatter description to reflect dual-mode ("Hydrate docs from external sources or generate from codebase analysis"), update heading from `# /fab:hydrate [sources...]` to `# /fab:hydrate [sources...|folders...]`, update Purpose section to describe both modes

## Phase 2: Core Implementation

- [x] T002 Replace the Arguments section in `fab/.kit/skills/fab-hydrate.md` — remove the "no sources = abort" behavior, add argument classification logic (URL → ingest, `.md` path → ingest, folder → generate, no args → generate), add mixed-mode rejection error, add the routing table from spec
- [x] T003 Add Generate Mode Step 1: Codebase Scanning section to `fab/.kit/skills/fab-hydrate.md` — instructions for the agent to scan using Glob/Grep/Read, enumerate directories (excluding `.git/`, `node_modules/`, `vendor/`, `__pycache__/`, `dist/`, `build/`), detect gaps by category (Modules: directory-to-domain comparison against `fab/docs/index.md`; APIs: grep for route/export patterns; Patterns: recurring structural patterns 3+; Configuration: glob for config files + env var references; Conventions: naming/structure analysis), cross-reference each against existing `fab/docs/` entries
- [x] T004 Add Generate Mode Step 2: Gap Report & Interactive Scoping section to `fab/.kit/skills/fab-hydrate.md` — format numbered gap report grouped by category with priorities (High/Medium/Low), present report as formatted text, then use AskUserQuestion with 4 strategy options ("All", "All High priority", "Select by number", "Select by category"), handle user's selection via Other text input for number/category choices, handle 1-3 gaps case (skip selection UI, just confirm), handle zero gaps case ("No documentation gaps found")
- [x] T005 Add Generate Mode Step 3: Doc Generation section to `fab/.kit/skills/fab-hydrate.md` — for each selected gap: read all source files in scope, synthesize into one doc per gap (not per file), write to `fab/docs/{domain}/{topic}.md` in centralized doc format (Overview, Requirements with RFC 2119 keywords, Design Decisions, Changelog with "Generated from code analysis" entry), add `[INFERRED]` markers on uncertain behaviors with explanation
- [x] T006 Add Generate Mode Step 4: Index Maintenance section to `fab/.kit/skills/fab-hydrate.md` — reuse same logic as ingest mode Steps 4-5 (create/update domain indexes, update top-level index, relative links, no deletions), note this is shared between both modes

## Phase 3: Integration & Edge Cases

- [x] T007 Add Generate Mode Output section to `fab/.kit/skills/fab-hydrate.md` — output examples for: successful generation (scan → gap report → selection → docs created), zero gaps found, re-generation (idempotent merge), scoped scan (folder argument)
- [x] T008 Update Error Handling table in `fab/.kit/skills/fab-hydrate.md` — replace "No sources provided → abort with usage message" with "No sources provided → enter generate mode", add: "Mixed-mode arguments → reject with error message", add: "Folder path doesn't exist → report error, abort", add: "Zero gaps found → report and exit cleanly"
- [x] T009 Update Idempotency Guarantee section in `fab/.kit/skills/fab-hydrate.md` — add generate-mode idempotency rules (re-scan merges into existing docs, manual edits preserved, new gaps appear in report, previously documented areas excluded/deprioritized)
- [x] T010 Update the Next line at bottom of `fab/.kit/skills/fab-hydrate.md` — change from `/fab:new <description>` or `/fab:hydrate <more-sources>` to also mention `/fab:hydrate` (no args, generate mode)

---

## Execution Order

- T001 blocks T002 (restructured heading/purpose needed before argument section rewrite)
- T002 blocks T003-T006 (argument routing determines which mode runs)
- T003 blocks T004 (gap report depends on scan results)
- T004 blocks T005 (doc generation depends on selection)
- T005 blocks T006 (index maintenance runs after doc generation)
- T007-T010 can run after T006 (output/error/idempotency sections are additive)
