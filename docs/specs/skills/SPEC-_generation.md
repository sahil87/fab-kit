# _generation

## Summary

Shared artifact generation procedures used by five skills, split across two disjoint consumer groups: `/fab-new` and `/fab-draft` follow the **Intake Generation Procedure**; `/fab-continue`, `/fab-ff`, and `/fab-fff` follow the **Plan Generation Procedure** (invoked at apply entry, before any task executes). Each skill references these procedures instead of inlining them, so generation behavior is authoritative in one location. Orchestration (stage guards, question handling, resumability) stays in each consuming skill's own file.

This is an internal partial (`user-invocable: false`) — never invoked directly. Skills load it via `helpers: [_generation]` frontmatter.

## Flow

```
Consumer skill reads _generation.md (via helpers: declaration)
│
├─ Intake Generation Procedure (fab-new, fab-draft)
│  ├─ Read: $(fab kit-path)/templates/intake.md
│  ├─ Fill metadata ({CHANGE_NAME}, {YYMMDD-XXXX-slug}, {DATE})
│  ├─ Write every section substantively (Origin, Why, What Changes,
│  │  Affected Memory, Impact, Open Questions) — the intake is a
│  │  STATE TRANSFER document: the downstream apply-entry agent has
│  │  no shared context beyond this file + always-loaded layers, so
│  │  design decisions are reproduced verbatim, never summarized
│  ├─ Append ## Assumptions per the SRAD framework (_srad.md,
│  │  loaded via helpers: by all consumers of this procedure)
│  │  (intake artifacts record all four grades)
│  └─ Write: fab/changes/{name}/intake.md
│
└─ Plan Generation Procedure (fab-continue, fab-ff, fab-fff @ apply entry)
   ├─ Read: $(fab kit-path)/templates/plan.md
   ├─ Generate ## Requirements from the intake-derived design
   │  ├─ ### {Domain}: {Topic} sections, RFC-2119 statements,
   │  │  stable R# IDs, ≥1 GIVEN/WHEN/THEN scenario each
   │  ├─ Optional: ### Non-Goals / ### Design Decisions /
   │  │  ### Deprecated Requirements
   │  ├─ NO [NEEDS CLARIFICATION] markers — under-specified points
   │  │  become graded SRAD ## Assumptions rows
   │  │  (Certain/Confident/Tentative only; Unresolved is intake-only)
   │  └─ Legacy spec.md ingestion (one-release back-compat): fold a
   │     leftover spec.md into ## Requirements, annotate
   │     <!-- migrated from spec.md -->
   ├─ Walk ## Requirements: emit a Task + an Acceptance entry per
   │  requirement (paired by work item)
   │  └─ Traceability REQUIRED: R# → T# → test → A#
   │     ├─ each ## Tasks item carries <!-- R# -->
   │     └─ each requirement-derived ## Acceptance item names its R#;
   │        Code Quality + checklist.extra_categories items are
   │        exempt (A-{NNN}: {outcome}, no R#)
   ├─ ## Tasks: phases 1-4 (Setup / Core / Integration / Polish),
   │  [P] parallel markers, exact file paths, T{NNN} IDs;
   │  ## Execution Order only for non-obvious dependencies
   ├─ ## Acceptance: categories derived from requirements
   │  (Functional Completeness, Behavioral Correctness, Removal
   │  Verification, Scenario Coverage, Edge Cases & Error Handling,
   │  Security) plus Code Quality (baseline 2 items, expanded by
   │  fab/project/code-quality.md) and checklist.extra_categories
   └─ Write: fab/changes/{name}/plan.md
      (PostToolUse hook updates .status.yaml plan.* counters —
       no manual fab status set-acceptance needed at generation time)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Templates (`intake.md`, `plan.md` via `$(fab kit-path)`), intake.md, memory files |
| Write | `intake.md` or `plan.md` in the change folder |

### Sub-agents

None — procedures run inside the consuming skill's context (one skill call, one context window co-generates `## Requirements` + `## Tasks` + `## Acceptance`, the alignment guarantee).

### Bookkeeping commands (hook candidates)

| Command | Trigger |
|---------|---------|
| *(none — writes only)* | The `plan.md`/`intake.md` Write fires the PostToolUse artifact hook, which updates `.status.yaml` (`plan.*` counters on plan writes; `change_type` + intake score on intake writes) |
