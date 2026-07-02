# _generation

## Summary

Shared artifact generation procedures. The two **forward** procedures are used by five skills across two consumer groups: `/fab-new`, `/fab-draft`, and `/fab-continue` (its intake-`active` regeneration row) follow the **Intake Generation Procedure**; `/fab-continue`, `/fab-ff`, and `/fab-fff` follow the **Plan Generation Procedure** (invoked at apply entry, before any task executes) — `/fab-continue` belongs to both groups. Two **-from-Diff** procedures are the adoption variants used by `/fab-adopt` only: **Intake-from-Diff** reconstructs `intake.md` from a fixed existing branch diff + PR body (Origin = `adopted from {PR/branch}`, Affected Memory inferred from touched `docs/memory/` domains), and **Plan-from-Diff** writes a deliberately thin `plan.md` — plain-language `## Requirements` (the only part hydrate reads), all-`[x]` `## Tasks`/`## Acceptance` stubs, and **no** R#/T#/A# scaffolding or GIVEN/WHEN/THEN (the apply↔review traceability loop never runs for an adopted change). `/fab-adopt` runs both diff procedures in **one main-session pass** (same agent, reads the diff once). Each skill references these procedures instead of inlining them, so generation behavior is authoritative in one location. Orchestration (stage guards, the human-confirmation checkpoint, question handling, design decisions, resumability) stays in each consuming skill's own file.

This is an internal partial (`user-invocable: false`) — never invoked directly. Skills load it via `helpers: [_generation]` frontmatter.

**Prose optimization** (260620-skop): a `## Contents` TOC added to `_generation.md` (structural check, file >100 lines); no prose trimmed and no behavioral change (Flow unchanged).

## Flow

```
Consumer skill reads _generation.md (via helpers: declaration)
│
├─ Intake Generation Procedure (fab-new, fab-draft,
│                                fab-continue's intake regeneration)
│  ├─ Read: $(fab kit-path)/templates/intake.md
│  ├─ Fill metadata ({CHANGE_NAME}, {YYMMDD-XXXX-slug}, {DATE})
│  ├─ Write every section substantively (Origin, Why, What Changes,
│  │  Affected Memory, Impact, Open Questions) — the intake is a
│  │  STATE TRANSFER document: the downstream apply-entry agent has
│  │  no shared context beyond this file + always-loaded layers, so
│  │  design decisions are reproduced verbatim, never summarized
│  ├─ Append ## Assumptions per the SRAD framework (_srad.md,
│  │  loaded via helpers: by all consumers of this procedure)
│  │  (intake artifacts record all four grades; section always
│  │   present — "0 assumptions." footer when empty, the
│  │   omit-when-zero rule is displayed-output-only)
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
   ├─ ## Assumptions: persist the graded SRAD rows decided inline
   │  during the walk (explicit step — 3 grades, Scores required,
   │  footer; ALWAYS present in the artifact, "0 assumptions."
   │  footer when empty; omit-when-zero is displayed-output-only)
   └─ Write: fab/changes/{name}/plan.md
      (fab status refresh recomputes .status.yaml plan.* counters,
       self-healed at advance/finish/preflight — no manual
       fab status set-acceptance needed at generation time)

── Adoption variants (fab-adopt only — one main-session pass, diff read once) ──

├─ Intake-from-Diff Procedure
│  Inputs: git diff {base}...HEAD, --name-only, PR body/title (or branch)
│  ├─ Origin = "adopted from {PR url or branch}"
│  ├─ Why / What Changes synthesised from the diff + PR body (descriptive)
│  ├─ Affected Memory inferred from which docs/memory/ domains the paths touch
│  ├─ Impact from changed paths; apply SRAD + fab score
│  └─ Write: fab/changes/{name}/intake.md
│     (human-confirmation checkpoint is fab-adopt orchestration, not this proc)
│
└─ Plan-from-Diff Procedure (deliberately MINIMAL — sole consumer is hydrate)
   ├─ Header note: "Adopted change — code authored off-pipeline. Apply was
   │  skipped; this plan is reverse-engineered from the branch diff to feed hydrate."
   ├─ ## Requirements — plain-language restatement of intake's What-Changes,
   │  grouped by area (NO R# IDs, no RFC-2119 ceremony, no GIVEN/WHEN/THEN)
   ├─ ## Tasks — single all-[x] stub (no T# / phases / [P] / <!-- R# -->)
   ├─ ## Acceptance — single all-[x] stub (no A# / R#)
   ├─ ## Assumptions — present ("0 assumptions." when none; diff-reading
   │  assumptions live on the intake, not duplicated here)
   └─ Write: fab/changes/{name}/plan.md
      (parser contract = only the three heading literals — confirmed against
       templates/plan.md; omitting R#/T#/A# is correct, the loop never runs)
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
| *(none — writes only)* | `fab status refresh`, self-healed at the transition seams (`advance`/`finish`/`preflight`), recomputes `.status.yaml` from the artifacts on disk (`plan.*` counters from `plan.md`; `change_type` + intake score from `intake.md`) |
