---
name: _generation
description: "Artifact generation procedures — Intake Generation (used by fab-new, fab-draft, and fab-continue's intake regeneration), Plan Generation (used by fab-continue, fab-ff, fab-fff at apply entry), and the diff-based Intake-from-Diff + Plan-from-Diff procedures (used by fab-adopt to reconstruct artifacts from an existing branch diff)."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Artifact Generation Procedures

> This file defines the shared artifact generation logic used by five skills: `/fab-new`,
> `/fab-draft`, and `/fab-continue` (its intake-`active` regeneration row) follow the
> **Intake Generation Procedure**; `/fab-continue`, `/fab-ff`, and `/fab-fff` follow the
> **Plan Generation Procedure** (at apply entry) — `/fab-continue` belongs to both consumer
> groups. The two **-from-Diff** procedures (Intake-from-Diff, Plan-from-Diff) are the
> adoption variants used by `/fab-adopt` only — they reconstruct both artifacts from a fixed
> existing branch diff rather than from a forward design. Each skill references these procedures
> instead of inlining them, ensuring generation behavior is authoritative in one location.
>
> **Orchestration** (stage guards, question handling, design decisions, resumability)
> remains in each skill's own file. This partial covers only the mechanics of producing each artifact.

## Contents

- Intake Generation Procedure
- Plan Generation Procedure
- Intake-from-Diff Procedure
- Plan-from-Diff Procedure

---

## Intake Generation Procedure

> **Generation rule**: The intake is a state transfer document — the downstream apply-entry agent
> (which co-generates `plan.md`) has NO shared context beyond this file and the always-loaded
> config/constitution/memory. Every section must contain enough concrete detail (examples, code blocks,
> specific values, exact behavior descriptions) for an agent with no conversation history to generate a
> complete plan (requirements + tasks + acceptance). If a design decision was discussed with specific
> values — include them verbatim. Do not summarize or abstract.

1. Read the template from `$(fab kit-path)/templates/intake.md`
2. Fill in metadata fields:
   - `{CHANGE_NAME}`: Human-readable name from the description
   - `{YYMMDD-XXXX-slug}`: The change folder name
   - `{DATE}`: Today's date
3. For each section (Origin, Why, What Changes, Affected Memory, Impact, Open Questions):
   - Write substantively — no placeholder text, no single-sentence descriptions
   - Include concrete examples: code blocks, YAML snippets, specific file paths, exact behavior
   - The "What Changes" section should be the most detailed — use subsections per change area
   - If a design includes specific values (config structure, template content, validation questions), reproduce them in full
4. Append `## Assumptions` section per the SRAD framework (`_srad.md`, loaded via `helpers:` by all consumers of this procedure). The section is always present in the artifact — `_srad.md`'s omit-when-zero rule applies to the displayed output summary only; when no assumptions were made, keep the section with the footer `0 assumptions.` and no table rows
5. Write the completed intake to `fab/changes/{name}/intake.md`

---

## Plan Generation Procedure

> Merges requirement generation (formerly the standalone Spec Generation Procedure / `spec.md`),
> tasks generation, and acceptance generation into a single walk. The procedure derives
> `## Requirements` from the intake, then — in the same pass — emits an imperative Task entry and a
> declarative Acceptance entry per requirement. **One skill call, one context window** co-generating
> all three sections is the strongest alignment guarantee: the same agent that writes a requirement
> immediately consumes it. No mid-change ID rewrites: newly generated plans use `R#` for requirements,
> `T{NNN}` for tasks, and `A-{NNN}` for acceptance items; in-flight migrations preserve legacy
> `CHK-NNN` IDs verbatim (handled by the migration, not this procedure).

> **Invocation**: This procedure is invoked from `/fab-continue` Apply Behavior at apply
> entry, before any task is executed. It is not a planning-stage step. There is no `spec` stage and
> no separate `spec.md` artifact — the canonical artifact flow is `intake.md → plan.md → code`.

1. Read the template from `$(fab kit-path)/templates/plan.md`
2. Fill in metadata fields:
   - `{CHANGE_NAME}`: From the intake (the human-readable name)
   - `{YYMMDD-XXXX-slug}`: The change folder name
   - Keep the `Intake` link pointing at `intake.md`
3. **Generate `## Requirements` from the intake-derived design** (absorbs the former Spec
   Generation Procedure):
   - For each domain/topic affected by this change, create a `### {Domain}: {Topic}` section with
     RFC 2119 requirement statements (MUST, SHALL, SHOULD, MAY), each with a stable `R#` ID and at
     least one GIVEN/WHEN/THEN scenario.
   - Include a `### Non-Goals` subsection (optional) for meaningful scope exclusions; a
     `### Design Decisions` subsection (optional) for architectural choices — each entry in the
     **four-field shape** (**Decision** / **Why** / **Rejected** / *Introduced by*) matching the
     memory `## Design Decisions` entry shape (FKF §3.3), so hydrate's pattern capture can lift a
     plan DD entry into memory DD without reshaping; and a `### Deprecated Requirements` subsection
     if the change removes existing requirements.
   - **No `[NEEDS CLARIFICATION]` markers.** Those are an intake-only construct (a human still needs
     to decide). An under-specified requirement encountered here is resolved inline as a graded SRAD
     assumption (Certain/Confident/Tentative) recorded in the plan's `## Assumptions` section — not
     as a marker. (Apply does not *clarify*; it *decides and records*.)
   - **Legacy `spec.md` ingestion (one-release back-compat)**: if a leftover `spec.md` exists in the
     change folder AND `plan.md` does not yet have a `## Requirements` section, fold the spec.md
     requirement body (domain sections, scenarios, Non-Goals/Design Decisions/Deprecated
     Requirements) into `## Requirements` instead of regenerating from scratch. Annotate the section
     `<!-- migrated from spec.md -->`. Do not move spec.md's `## Assumptions` table.
4. **Walk the `## Requirements` just generated.** For each requirement, emit two entries — a Task and
   an Acceptance — paired by the same logical work item:
   - The Task entry goes under `## Tasks` and describes *what to implement, in which file*
   - The Acceptance entry goes under `## Acceptance` and describes *what must be true for
     review to pass* (a declarative outcome, not a step)
   - **Traceability is REQUIRED** (not optional): each `## Tasks` item MUST carry a `<!-- R# -->`
     trace annotation naming the requirement it implements, and each **requirement-derived**
     `## Acceptance` item MUST name the requirement it accepts (e.g., `A-001 R2: {outcome}`).
     Non-requirement-derived categories (Code Quality, `checklist.extra_categories`) are exempt —
     see step 6. The chain `R# → T# → test → A#` is what lets the autonomous (apply ↔ review)
     loop localize a failing acceptance item back to its requirement and converge.
5. **Tasks subsection** (`## Tasks`):
   - Group by phase. Phases execute sequentially; within a phase, `[P]`-marked tasks may run
     in parallel:
     - **Phase 1: Setup** — scaffolding, dependencies, configuration
     - **Phase 2: Core Implementation** — primary functionality, ordered by dependency
     - **Phase 3: Integration & Edge Cases** — wiring, error states, validation
     - **Phase 4: Polish** — documentation, cleanup (only if warranted)
   - Each task follows the format: `- [ ] T{NNN} [{markers}] {description with file paths} <!-- R# -->`
   - IDs are sequential, three-digit: T001, T002, ...
   - Mark parallelizable tasks with `[P]`
   - Include exact file paths in descriptions
   - Each task should be completable in one focused session
   - Each task MUST carry a `<!-- R# -->` trace annotation naming the requirement it implements
   - Include an `## Execution Order` section after `## Tasks` only for non-obvious
     dependencies between tasks; omit when ordering is self-evident
6. **Acceptance subsection** (`## Acceptance`):
   - Populate items derived from:
     - `## Requirements` — every requirement gets at least one item under **Functional Completeness**
     - Changed requirements → **Behavioral Correctness** items
     - Deprecated requirements → **Removal Verification** items
     - Key scenarios from `## Requirements` → **Scenario Coverage** items
     - Edge cases identified in `## Requirements` → **Edge Cases & Error Handling** items
     - `fab/project/code-quality.md` → **Code Quality** items. If
       `fab/project/code-quality.md` exists: one item per relevant principle from
       `## Principles`, one per relevant anti-pattern from `## Anti-Patterns` that applies to
       the change's scope, plus the two baseline items. If no `fab/project/code-quality.md`:
       include the two baseline items only (pattern consistency, no unnecessary duplication)
     - Security-relevant changes → **Security** items (only if applicable)
     - Additional categories from `fab/project/config.yaml` `checklist.extra_categories` (if
       any)
   - **Requirement-derived categories** (Functional Completeness, Behavioral Correctness,
     Removal Verification, Scenario Coverage, Edge Cases & Error Handling, Security) follow the
     format: `- [ ] A-{NNN} R#: {declarative outcome}` — naming the requirement it accepts
     (REQUIRED)
   - **Non-requirement-derived categories** (Code Quality, `checklist.extra_categories`) carry
     no R# reference: `- [ ] A-{NNN}: {declarative outcome}` (an optional label may precede the
     colon, e.g. `A-007 Pattern consistency: ...`) — these accept project-wide standards, not a
     specific requirement
   - IDs are sequential, three-digit, zero-padded: A-001, A-002, ...
7. **Assumptions section** (`## Assumptions`): persist every graded SRAD assumption made during
   steps 3–6 (each under-specified point decided inline) as a row of the template's trailing
   `## Assumptions` table, per `_srad.md` § Assumptions Summary Block: three grades only
   (Certain/Confident/Tentative — Unresolved is intake-only), the `Scores` column required on
   every row, and the footer `{N} assumptions ({Ce} certain, {Co} confident, {T} tentative).`
   The section is ALWAYS present in the artifact — if zero assumptions were made, keep the
   section with no table rows and the footer `0 assumptions.` (`_srad.md`'s omit-when-zero rule
   applies to the displayed output summary only, never to artifacts).
8. Write the completed plan to `fab/changes/{name}/plan.md`. `fab status refresh` recomputes
   `.status.yaml` `plan.generated`, `plan.task_count`, `plan.acceptance_count`, and
   `plan.acceptance_completed`, self-healed at the next `advance`/`finish`/`preflight`; no
   manual `fab status set-acceptance` calls are required at generation time. Skills that wish
   to assert the counts explicitly MAY call `fab status set-acceptance <change> <field> <value>`
   (valid fields: `generated`, `task_count`, `acceptance_count`, `acceptance_completed`).

---

## Intake-from-Diff Procedure

> **Adoption variant** (used by `/fab-adopt` only). Where the Intake Generation Procedure
> reconstructs a *forward* design from a conversation, this procedure reverse-engineers the
> intake from a **fixed existing branch diff** plus the PR body — the code already exists, so
> the artifact *describes* it rather than *plans* it. Both this procedure and the Plan-from-Diff
> Procedure run in **one main-session pass** by the same agent reading the diff once (no dispatched
> sub-agent, no re-read between the two artifacts — a context boundary would only invite drift).

**Inputs** (resolved by `/fab-adopt` Step 0, passed into this procedure): the diff against the
default-branch merge-base (`git diff {base}...HEAD`), the changed file list
(`git diff --name-only {base}...HEAD`), and the PR body / title when a PR exists (else the branch
name).

1. Read the template from `$(fab kit-path)/templates/intake.md` and fill the metadata fields
   (`{CHANGE_NAME}`, `{YYMMDD-XXXX-slug}`, `{DATE}`) as in the Intake Generation Procedure.
2. Fill each section **from the diff + PR body**, not from a design conversation:
   - **Origin**: `adopted from {PR url or, if no PR, the branch name}` — state plainly that the
     code was authored off-pipeline and is being brought in via `/fab-adopt`.
   - **Why**: synthesise the problem/rationale from the PR body (its description, linked issues)
     and what the diff actually does. When the PR body is thin, infer conservatively from the diff
     and record the inference as a graded SRAD assumption.
   - **What Changes**: synthesise per-area subsections from the changed files — group the diff by
     directory/domain and describe what each area changed. This is descriptive (what the code
     does), not prescriptive.
   - **Affected Memory**: infer from which `docs/memory/` domains the changed paths map to (the
     same domain mapping the Memory File Lookup uses). This list feeds hydrate — be accurate.
   - **Impact**: derive from the changed paths (files touched, rough scale). Do not fabricate test
     coverage the diff does not contain.
3. Apply SRAD per `_srad.md` and append the `## Assumptions` section (every inference made while
   reading the diff is a graded row). Run `fab score` on the result.
4. Write the completed intake to `fab/changes/{name}/intake.md`.

> **Human-confirmation checkpoint** (owned by `/fab-adopt`, noted here for context): the
> reconstructed intent + SRAD assumptions are presented for the user to confirm/correct before the
> pipeline advances. This is the late deliberation the bypass skipped. The checkpoint itself is
> orchestration and lives in `fab-adopt.md`, not in this procedure.

---

## Plan-from-Diff Procedure

> **Adoption variant** (used by `/fab-adopt` only). Apply was **skipped** for an adopted change —
> the code already exists and the apply↔review traceability loop never runs — so this plan is
> deliberately **minimal**. It exists for **one consumer**: hydrate, which reads `## Requirements`
> to write accurate memory. Effort concentrates entirely on plain-language requirements; the
> `## Tasks` / `## Acceptance` sections are all-`[x]` stubs that carry no traceability scaffolding.
> Runs in the same main-session pass as the Intake-from-Diff Procedure (no re-read of the diff).

1. Read the template from `$(fab kit-path)/templates/plan.md` and fill the metadata fields
   (`{CHANGE_NAME}`, `{YYMMDD-XXXX-slug}`, keep the `Intake` link).
2. Carry a header note immediately below the metadata, verbatim intent:
   `> Adopted change — code authored off-pipeline. Apply was skipped; this plan is reverse-engineered from the branch diff to feed hydrate.`
3. **`## Requirements`** — a **plain-language** restatement of the intake's What-Changes, grouped
   by change area. This is the only part hydrate reads, so it is the only part that needs care.
   Largely lifts the intake's What-Changes into plan form. **Do NOT** emit `R#` IDs, RFC-2119
   ceremony, or GIVEN/WHEN/THEN scenarios — plain prose per area is correct and sufficient.
4. **`## Tasks`** — a single all-`[x]` stub, e.g.:
   `- [x] Adopted: implementation authored outside the pipeline (see {PR or branch}).`
   No `T{NNN}` IDs, no phases, no `[P]` markers, no `<!-- R# -->` traces.
5. **`## Acceptance`** — a single all-`[x]` stub, e.g.:
   `- [x] Adopted: code already authored and (where applicable) PR-reviewed; a diff-only review runs in this pipeline.`
   No `A-{NNN}` IDs, no `R#` references.
6. **`## Assumptions`** — present per the artifact rule (footer `0 assumptions.` when none; the
   diff-reading assumptions are recorded on the **intake**, not duplicated here).
7. Write the completed plan to `fab/changes/{name}/plan.md`.

> **Parser-contract guarantee**: the only stable contract the pipeline parses is the three heading
> literals `## Requirements`, `## Tasks`, and `## Acceptance` (confirmed against
> `templates/plan.md`). The thin plan keeps all three headings, so every downstream reader
> (hydrate, `fab status refresh`'s counters) parses it without special-casing. The R#/T#/A# IDs and
> GIVEN/WHEN/THEN scenarios are presentational scaffolding the apply↔review loop needs — and that
> loop never runs for an adopted change, so omitting them is correct, not a gap.
