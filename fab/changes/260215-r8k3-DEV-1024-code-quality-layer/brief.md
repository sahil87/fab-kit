# Brief: Add Code Quality Layer

**Change**: 260215-r8k3-DEV-1024-code-quality-layer
**Created**: 2026-02-15
**Status**: Draft

## Origin

> Add a code quality layer to fab-kit so the pipeline produces high-quality, maintainable code when developers use it in their projects. Currently the quality model is "good spec → good code" — the pipeline enforces planning discipline and validates spec compliance at review, but doesn't guide *how well* code is written during apply, doesn't assess code quality during review, and doesn't learn implementation patterns across changes.

## Why

Fab-kit enforces *what* to build (via specs, tasks, checklists) but not *how well* to build it. The apply stage has one line of guidance ("implement per spec/constitution/patterns"), review only checks spec compliance and test passage, and the checklist is entirely spec-derived. This means:

- Developers get correct code (matches spec) but not necessarily maintainable code
- No mechanism for projects to encode their coding standards into the pipeline
- "Follow existing patterns" is aspirational without actually reading and extracting those patterns
- The same quality mistakes can repeat across changes because patterns aren't captured in memory

## What Changes

### 1. New `code_quality` config section in `config.yaml`

Optional, commented-out by default (matching the `conventions` pattern). Projects opt in. Consumed during apply and review.

```yaml
# code_quality:
#   # principles — Positive coding standards to follow during implementation.
#   principles:
#     - "Readability and maintainability over cleverness"
#     - "Follow existing project patterns unless there's compelling reason to deviate"
#     - "Prefer composition over inheritance"
#
#   # anti_patterns — Patterns to avoid. Flagged during review.
#   anti_patterns:
#     - "God functions (>50 lines without clear reason)"
#     - "Duplicating existing utilities instead of reusing them"
#     - "Magic strings or numbers without named constants"
#
#   # test_strategy — How tests relate to implementation.
#   # Values: test-alongside (default) | test-after | tdd
#   test_strategy: "test-alongside"
```

### 2. Expanded Apply Behavior in `fab-continue.md`

**Pattern Extraction** — new section between Preconditions and Task Execution. Before executing the first unchecked task, read existing source files in the areas the change will touch and extract:

1. **Naming conventions** — variable/function/class naming style
2. **Error handling** — how the codebase handles errors (exceptions, Result types, error codes)
3. **Structure** — typical function length, module boundaries, import organization
4. **Reusable utilities** — existing helpers or shared modules that new code should use instead of reimplementing

Hold these as context for all subsequent tasks. If `config.yaml` defines `code_quality`, load its `principles` and `anti_patterns` as additional constraints. Skip when resuming mid-apply.

**Per-task guidance** — expand the current single-line step 4 ("read source, implement per spec/constitution/patterns, run tests, fix failures, mark [x]") to:

- Read source files relevant to this task
- Implement per spec, constitution, and extracted patterns
- Prefer reusing existing utilities over creating new ones
- Keep functions focused — if implementation exceeds the codebase's typical function size, consider extracting
- Write tests per `config.yaml` `code_quality.test_strategy` (default: test-alongside)
- Run tests, fix failures
- Mark `[x]` immediately

### 3. Expanded Review Behavior in `fab-continue.md`

Add step 6 (code quality check) after the existing step 5 (memory drift check). For each file modified during apply:

- Naming conventions consistent with surrounding code?
- Functions focused and appropriately sized?
- Error handling consistent with codebase style?
- Existing utilities reused where applicable?
- If `config.yaml` defines `code_quality.principles`, check each
- If `config.yaml` defines `code_quality.anti_patterns`, check for violations

Code quality issues are review failures with specific file:line references (same rework flow as spec mismatches).

### 4. Code Quality checklist category in `checklist.md` template

Always included (unlike Security which requires security surface). Derived from implementation diff and config, not from spec. Two baseline items when no `code_quality` config exists:

```markdown
## Code Quality
- [ ] CHK-{NNN} Pattern consistency: New code follows naming and structural patterns of surrounding code
- [ ] CHK-{NNN} No unnecessary duplication: Existing utilities reused where applicable
```

When `code_quality` config exists: one item per relevant `principle`, one per relevant `anti_pattern` that applies to the change's scope.

### 5. Updated Checklist Generation Procedure in `_generation.md`

Add `code_quality` as a derivation source in step 4, alongside the existing spec-derived sources. If no `code_quality` section, include the two baseline items above.

### 6. Expanded Source Code Loading in `_context.md`

Add two stage-specific steps after the existing 3 steps:

- **Apply stage**: Also read neighboring files in the same directories to extract pattern context (naming, error handling, structure). See Pattern Extraction in `/fab-continue`
- **Review stage**: Re-read modified files to validate consistency with surrounding code

### 7. Optional pattern capture in Hydrate in `fab-continue.md`

Add step 5 to Hydrate Behavior: if the change introduced non-obvious implementation patterns that future changes should follow (e.g., a new error handling approach, a reusable abstraction), note them in the relevant memory file's Design Decisions section. Skip for implementations that follow existing patterns.

### 8. Updated `fab-init.md`

- Add `code_quality` to valid sections lists
- Add menu item 9 ("code_quality — coding standards for apply/review"), bump Done to 10
- Add commented-out `code_quality` block to Config Create Mode template

## Affected Memory

- `fab-workflow/configuration`: (modify) Document new `code_quality` config section schema
- `fab-workflow/execution-skills`: (modify) Add Pattern Extraction to Apply, code quality check to Review, pattern capture to Hydrate. Update "Checklist Tests Implementation Fidelity" design decision
- `fab-workflow/templates`: (modify) Add Code Quality to checklist categories list

## Impact

- **Skills**: `fab-continue.md` (Apply Behavior, Review Behavior, Hydrate Behavior), `_generation.md` (Checklist Generation Procedure), `_context.md` (Source Code Loading)
- **Templates**: `checklist.md` (new Code Quality section)
- **Config**: `config.yaml` (new section), `fab-init.md` (valid sections, menu, template)
- **Memory**: 3 files updated
- **Cascade**: fab-ff.md and fab-fff.md delegate to fab-continue behaviors — changes cascade automatically, no direct edits needed

## Open Questions

None — the change was thoroughly discussed and all design decisions were resolved during the discussion phase.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Changes go into kit skills/templates (not project-specific files) | This is a framework improvement for all fab-kit users, not a project-specific change | S:95 R:90 A:95 D:95 |
| 2 | Certain | `code_quality` config is optional/commented-out by default | Matches existing pattern (conventions section), projects opt in | S:90 R:95 A:90 D:90 |
| 3 | Certain | fab-ff/fff don't need direct changes | They delegate to fab-continue's Apply/Review/Hydrate behaviors | S:95 R:90 A:95 D:95 |
| 4 | Certain | Code quality checklist items are always included (not opt-in) | Unlike Security which requires security surface, code quality applies to all changes | S:85 R:85 A:85 D:90 |
| 5 | Confident | Pattern Extraction runs before first task, not before each task | Per-task extraction would be redundant; patterns for the change area are stable | S:80 R:85 A:70 D:75 |
| 6 | Confident | Pattern capture in hydrate is optional, not mandatory | Most implementations follow existing patterns — only capture genuinely new patterns | S:75 R:90 A:70 D:70 |
| 7 | Confident | Baseline code quality items (when no config) are: pattern consistency + utility reuse | These are universal and non-controversial quality dimensions | S:70 R:85 A:70 D:65 |
| 8 | Confident | test_strategy options are: test-alongside, test-after, tdd | Covers the main approaches without over-complicating | S:75 R:85 A:65 D:70 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
