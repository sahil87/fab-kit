# Intake: Rename "brief" to "intake" + Add Intake Generation Rule

**Change**: 260215-v4n7-DEV-1025-rename-brief-to-intake
**Created**: 2026-02-15
**Status**: Draft

## Origin

> During the code quality layer discussion, we discovered that Claude consistently treats `brief.md` as a summary document rather than the detailed state transfer document it's designed to be. The root cause: "brief" in English means "short/concise," which triggers summarization instincts in LLMs. The template's `STATE TRANSFER` comments weren't enough to override this. Two fixes: rename the stage/artifact from "brief" to "intake" (which in professional contexts means "thorough initial collection of all relevant information"), and add an explicit generation rule in `_generation.md` to reinforce the expected detail level.

## Why

The `brief.md` artifact is the **sole context for downstream pipeline stages**. Spec, tasks, and checklist generation all depend on it. When the brief is thin/abstract, every downstream artifact suffers — the spec-stage agent has to reinvent decisions that were already made. The word "brief" actively works against this: it tells the agent to be concise when it should be thorough.

"Intake" solves this at the naming level:
- In legal, medical, and project management contexts, "intake" means "the thorough initial collection of all relevant information"
- Nobody abbreviates an intake form — you fill it out completely
- It signals "this is the start" (unlike "handoff" which could be any stage boundary)

The generation rule in `_generation.md` is defense in depth — catches agents that still compress despite good naming.

## What Changes

### 1. Rename stage: `brief` → `intake` across the pipeline

**Scope**: ~180 occurrences across ~40 files. Systematic find-and-replace with context-aware substitutions:

**Stage identifiers** (exact token replacement):
- `.status.yaml` template and all status references: `brief: active` → `intake: active`
- `config.yaml` stages: `id: brief` → `id: intake`, `generates: brief.md` → `generates: intake.md`, `requires: [brief]` → `requires: [intake]`
- All skill files referencing the stage name: `brief` stage → `intake` stage

**Artifact references** (filename replacement):
- Template: `fab/.kit/templates/brief.md` → `fab/.kit/templates/intake.md`
- Generated artifact: `brief.md` → `intake.md` in all skill instructions
- Cross-references in spec.md, tasks.md templates that reference `brief.md`

**Skill files** (~19 files in `fab/.kit/skills/`):
- `fab-new.md`: "Generate `brief.md`" → "Generate `intake.md`", all step references
- `fab-continue.md`: stage dispatch table, context loading references, "brief: done" transitions
- `fab-ff.md`, `fab-fff.md`: stage references
- `fab-clarify.md`: brief references in scanning/gap analysis
- `fab-init.md`: valid sections, bootstrap output
- `fab-archive.md`: artifact references
- `fab-switch.md`: artifact loading
- `_context.md`: context loading layers, memory file lookup from brief's Affected Memory
- `_generation.md`: spec generation reading brief, tasks generation referencing brief
- Other skills with incidental references: `docs-reorg-specs.md`, `docs-reorg-memory.md`, `docs-hydrate-specs.md`, `internal-skill-optimize.md`

**Schema files**:
- `fab/.kit/schemas/workflow.yaml`: stage enum and validation references

**Documentation** (~18 files in `docs/`):
- `docs/specs/`: architecture.md, skills.md, templates.md, user-flow.md, overview.md, glossary.md, srad.md, index.md
- `docs/memory/fab-workflow/`: planning-skills.md (26 occurrences), templates.md, change-lifecycle.md, configuration.md, kit-architecture.md, clarify.md, index.md, context-loading.md, execution-skills.md, hydrate-generate.md

**Config**:
- `fab/config.yaml`: stage definition (`id: brief` → `id: intake`, `generates: brief.md` → `generates: intake.md`, `requires: [brief]` → `requires: [intake]`)

**NOT renamed**:
- The `fab-new` skill name stays — it creates changes, not just intakes
- Existing change folders in `fab/changes/` — their `brief.md` files are historical artifacts, not renamed

### 2. Add Intake Generation Procedure in `_generation.md`

Currently `_generation.md` has procedures for Spec, Tasks, and Checklist — but NOT for the intake (brief). The intake generation instructions live only in `fab-new.md` Step 5. Add a new `## Intake Generation Procedure` section in `_generation.md` that:

1. References the template from `fab/.kit/templates/intake.md`
2. Includes an explicit generation rule:

```markdown
## Intake Generation Procedure

> **Generation rule**: The intake is a state transfer document — downstream agents (spec, tasks, checklist)
> have NO shared context beyond this file and the always-loaded config/constitution/memory. Every section
> must contain enough concrete detail (examples, code blocks, specific values, exact behavior descriptions)
> for an agent with no conversation history to generate a complete spec. If a design decision was discussed
> with specific values — include them verbatim. Do not summarize or abstract.

1. Read the template from `fab/.kit/templates/intake.md`
2. Fill in metadata fields:
   - `{CHANGE_NAME}`: Human-readable name from the description
   - `{YYMMDD-XXXX-slug}`: The change folder name
   - `{DATE}`: Today's date
3. For each section (Origin, Why, What Changes, Affected Memory, Impact, Open Questions):
   - Write substantively — no placeholder text, no single-sentence descriptions
   - Include concrete examples: code blocks, YAML snippets, specific file paths, exact behavior
   - The "What Changes" section should be the most detailed — use subsections per change area
   - If a design includes specific values (config structure, template content, validation questions), reproduce them in full
4. Append `## Assumptions` section per `_context.md` SRAD framework
5. Write the completed intake to `fab/changes/{name}/intake.md`
```

Then update `fab-new.md` Step 5 to reference this procedure: "Generate `intake.md` per the **Intake Generation Procedure** (`_generation.md`)" — same pattern as how fab-continue references spec/tasks/checklist procedures.

### 3. Update intake template with stronger structural cues

Rename `fab/.kit/templates/brief.md` → `fab/.kit/templates/intake.md` and strengthen the template comments to reinforce detail expectations. For example, the "What Changes" section comment should say:

```markdown
## What Changes

<!-- Be specific about new capabilities, modifications, or removals.
     Use subsections (### per change area) for multi-part changes.
     Include concrete examples: code blocks, config snippets, exact behavior.
     This section is the primary input for spec generation — if a design decision
     was made with specific values, include them here. -->
```

Similar strengthening for Origin and Why sections.

## Affected Memory

- `fab-workflow/planning-skills`: (modify) Rename brief → intake throughout, update fab-new description
- `fab-workflow/templates`: (modify) Rename brief.md → intake.md, update template description
- `fab-workflow/change-lifecycle`: (modify) Rename brief stage → intake stage
- `fab-workflow/configuration`: (modify) Update stage references
- `fab-workflow/kit-architecture`: (modify) Update file structure references
- `fab-workflow/clarify`: (modify) Update brief references
- `fab-workflow/context-loading`: (modify) Update context layer references
- `fab-workflow/execution-skills`: (modify) Update brief references
- `fab-workflow/index`: (modify) Update domain description if it mentions brief

## Impact

- **Templates**: `brief.md` → `intake.md` (rename + content update)
- **Skills**: All 19 skill files with `brief` references
- **Schema**: `workflow.yaml` stage enum
- **Config**: `config.yaml` stages section
- **Specs**: 8 spec files
- **Memory**: 10 memory files
- **Cascade**: All existing change folders keep their `brief.md` files unchanged (historical)
- **Ordering**: This change should be applied BEFORE the code quality layer change (260215-r8k3), since both touch the same skill files. The code quality layer brief already uses "brief" terminology that would need updating.

## Open Questions

None — the rename scope and generation rule placement were explicitly discussed and agreed.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename is `brief` → `intake` (not handoff, charter, etc.) | Explicitly discussed and chosen — "intake" signals thorough collection and start-of-pipeline | S:95 R:80 A:95 D:95 |
| 2 | Certain | Generation rule goes in `_generation.md` as new Intake Generation Procedure | User explicitly corrected: not config.yaml rules, but `_generation.md` — matching the pattern of Spec/Tasks/Checklist procedures | S:95 R:85 A:90 D:95 |
| 3 | Certain | Existing change folders keep their `brief.md` files | Historical artifacts — renaming completed changes would be destructive and pointless | S:90 R:95 A:90 D:95 |
| 4 | Certain | `fab-new` skill name stays unchanged | The skill creates changes, not just intakes — renaming it would be misleading | S:85 R:90 A:90 D:90 |
| 5 | Certain | This change is separate from and ordered before the code quality layer | Different concerns (naming clarity vs. feature addition), explicitly discussed | S:95 R:85 A:95 D:95 |
| 6 | Confident | `fab-new.md` Step 5 should reference `_generation.md` procedure instead of inlining | Matches how fab-continue references spec/tasks/checklist procedures — single source of truth | S:75 R:85 A:80 D:70 |
| 7 | Confident | Template comments should be strengthened alongside the rename | The rename fixes naming instinct but template structure reinforces expected detail level | S:70 R:90 A:70 D:70 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
