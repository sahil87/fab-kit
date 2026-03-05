# Spec: Add fab-discuss Skill

**Change**: 260220-9ogw-add-fab-discuss
**Created**: 2026-02-20
**Affected memory**: `docs/memory/fab-workflow/context-loading.md`

## Non-Goals

- Loading change-specific artifacts (intake, spec, tasks) — `fab-discuss` loads only the always-load layer, not change context
- Advancing or modifying pipeline state — fully read-only, no `.status.yaml` writes
- Replacing `/fab-help` — `fab-discuss` orients to the *project*, not the *workflow*

## Fab Workflow: fab-discuss Skill

### Requirement: Always-Load Context

`fab-discuss` SHALL load the standard 7-file always-load layer defined in `_context.md` §1:

1. `fab/project/config.yaml`
2. `fab/project/constitution.md`
3. `fab/project/context.md` *(optional)*
4. `fab/project/code-quality.md` *(optional)*
5. `fab/project/code-review.md` *(optional)*
6. `docs/memory/index.md`
7. `docs/specs/index.md`

The skill MUST gracefully skip optional files (3–5) when they do not exist, without error.

#### Scenario: All 7 files present
- **GIVEN** a fab-kit repo with all 7 always-load files present
- **WHEN** the user invokes `/fab-discuss`
- **THEN** the skill reads all 7 files
- **AND** displays an orientation summary listing what was loaded

#### Scenario: Optional files missing
- **GIVEN** a fab-kit repo where `context.md`, `code-quality.md`, and `code-review.md` do not exist
- **WHEN** the user invokes `/fab-discuss`
- **THEN** the skill reads the 4 required files without error
- **AND** the orientation summary notes which optional files were not found

### Requirement: No Active Change Required

`fab-discuss` SHALL NOT require an active change (`fab/current`) and SHALL NOT run the preflight script. The skill MUST work in repos with no changes directory, no `fab/current`, or an empty `fab/current`.

#### Scenario: No active change
- **GIVEN** a fab-kit repo with no `fab/current` file
- **WHEN** the user invokes `/fab-discuss`
- **THEN** the skill loads the always-load layer and presents the orientation summary
- **AND** the active change section shows "No active change"

#### Scenario: Active change exists
- **GIVEN** a fab-kit repo with `fab/current` pointing to a valid change
- **WHEN** the user invokes `/fab-discuss`
- **THEN** the skill reads `fab/current` and the change's `.status.yaml`
- **AND** the orientation summary includes the change name and current stage
- **AND** the skill does NOT deep-load the change's artifacts (intake, spec, tasks)
<!-- assumed: Light-touch active change mention — useful orientation without heavy context loading. Agreed in origin discussion. -->

### Requirement: Orientation Summary Output

`fab-discuss` SHALL output a structured orientation summary containing:

1. **Project identity** — name and description from `config.yaml`
2. **Memory domains** — list of domains from `docs/memory/index.md` with file counts
3. **Specs landscape** — list of specs from `docs/specs/index.md`
4. **Active change** (if any) — name and stage from `fab/current` + `.status.yaml`
5. **Ready signal** — "Ready to discuss. What would you like to explore?"

The summary MUST NOT include a `Next:` pipeline command. `fab-discuss` is a session entry point, not a pipeline stage.
<!-- assumed: No state table integration — fab-discuss is not part of the pipeline, so no Next: line. -->

#### Scenario: Orientation output structure
- **GIVEN** a fab-kit repo with memory domain "fab-workflow" and specs "overview", "skills"
- **WHEN** the user invokes `/fab-discuss`
- **THEN** the output includes the project name and description
- **AND** lists "fab-workflow" as an available memory domain
- **AND** lists "overview" and "skills" as available specs
- **AND** ends with "Ready to discuss. What would you like to explore?"

### Requirement: Read-Only and Idempotent

`fab-discuss` SHALL NOT modify any files. It MUST be safe to invoke repeatedly without side effects. It SHALL NOT write to `fab/current`, `.status.yaml`, or any other file.

#### Scenario: Repeated invocation
- **GIVEN** a fab-kit repo in any state
- **WHEN** the user invokes `/fab-discuss` twice in succession
- **THEN** both invocations produce identical output
- **AND** no files are created, modified, or deleted

### Requirement: Skill Frontmatter

The skill file SHALL include the following frontmatter:

```yaml
---
name: fab-discuss
description: "Prime the agent with project context for a discussion session — loads the always-load layer and orients to the repo landscape."
model_tier: capable
---
```

#### Scenario: Skill discovery
- **GIVEN** the skill file exists at `fab/.kit/skills/fab-discuss.md`
- **WHEN** `/fab-help` scans skill frontmatter
- **THEN** `fab-discuss` appears in the "Start & Navigate" group with its description

## Fab Workflow: fab-help Integration

### Requirement: Group Assignment

`fab-help.sh` SHALL include `fab-discuss` in the `skill_to_group` mapping under `"Start & Navigate"`:

```bash
[fab-discuss]="Start & Navigate"
```

No other changes to `fab-help.sh` are needed — the script auto-discovers descriptions from frontmatter.

#### Scenario: Help output includes fab-discuss
- **GIVEN** `fab-discuss` is mapped to "Start & Navigate" in `fab-help.sh`
- **WHEN** the user runs `/fab-help`
- **THEN** `fab-discuss` appears under the "Start & Navigate" section

## Documentation: Specs Updates

### Requirement: skills.md Section

`docs/specs/skills.md` SHALL include a new `## /fab-discuss` section documenting:
- Purpose: prime agent with project context for discussion
- Context: same as always-load (the full 7-file layer)
- Key properties: no active change required, read-only, idempotent
- What it outputs: orientation summary + ready signal

Additionally, the **Context Loading Convention** section in `docs/specs/skills.md` SHALL be updated to reflect all 7 files (currently stale, listing only 3).

#### Scenario: skills.md accuracy
- **GIVEN** the updated `docs/specs/skills.md`
- **WHEN** an agent reads the context loading section
- **THEN** all 7 always-load files are listed, matching `_context.md` §1

### Requirement: overview.md Quick Reference

`docs/specs/overview.md` SHALL include a `/fab-discuss` row in the Quick Reference table:

```markdown
| `/fab-discuss` | Prime agent with project context for discussion | — (read-only) |
```

#### Scenario: Quick reference completeness
- **GIVEN** the updated Quick Reference table in `docs/specs/overview.md`
- **WHEN** a user looks up `/fab-discuss`
- **THEN** the row shows its purpose and indicates read-only

### Requirement: user-flow.md Diagram

`docs/specs/user-flow.md` SHALL add `/fab-discuss` to the Setup & Utilities diagram (Section 3A, "Utility (anytime)" subgraph) alongside `/fab-status` and `/fab-help`.

#### Scenario: Diagram includes fab-discuss
- **GIVEN** the updated Mermaid diagram in Section 3A
- **WHEN** rendered
- **THEN** `/fab-discuss` appears in the "Utility (anytime)" subgraph

## Documentation: Memory Update

### Requirement: context-loading.md Update

`docs/memory/fab-workflow/context-loading.md` SHALL be updated in the **Exception Skills** section to clarify `fab-discuss`'s relationship to the always-load layer:

- `fab-discuss` is the *only* skill whose entire purpose is to surface the always-load layer — it loads all 7 files and presents them as its primary output, rather than as a preamble to something else
- It is NOT an exception skill (it loads the full always-load layer), but it is a special case worth noting

A changelog entry SHALL be added.

#### Scenario: Memory accuracy
- **GIVEN** the updated `docs/memory/fab-workflow/context-loading.md`
- **WHEN** an agent reads the Exception Skills section
- **THEN** `fab-discuss` is mentioned as a special case that surfaces the always-load layer as its primary output

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Load exactly the standard 7-file always-load layer | Confirmed from intake #1 — explicitly agreed in conversation, paths are fixed constants | S:95 R:90 A:95 D:95 |
| 2 | Certain | Do not require an active change | Confirmed from intake #2 — discussion sessions don't need a change | S:90 R:90 A:95 D:90 |
| 3 | Certain | Skill frontmatter uses `model_tier: capable` | Upgraded from intake #3 — discussion requires reasoning and synthesis; no simpler tier fits | S:75 R:90 A:85 D:85 |
| 4 | Certain | Add to "Start & Navigate" group in fab-help.sh | Upgraded from intake #4 — it's a session-entry/orientation skill alongside fab-status and fab-help | S:70 R:90 A:90 D:85 |
| 5 | Confident | Fix stale context-loading section in skills.md | Confirmed from intake #5 — currently wrong (lists 3 files vs 7), low-risk fix | S:80 R:85 A:85 D:80 |
| 6 | Confident | Mention active change name/stage in output (light touch) | Upgraded from intake #6 — useful orientation that costs little; reads fab/current + .status.yaml only | S:60 R:85 A:75 D:70 |
| 7 | Confident | Do not add fab-discuss to state table Next: suggestions | Upgraded from intake #7 — it's a session entry point, not a pipeline step; state table covers pipeline only | S:65 R:80 A:80 D:70 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
