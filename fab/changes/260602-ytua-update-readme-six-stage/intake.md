# Intake: Update README to the 6-stage pipeline

**Change**: 260602-ytua-update-readme-six-stage
**Created**: 2026-06-02
**Status**: Draft

## Origin

Initiated from a `/fab-discuss` session followed by `/fab-new`. The user's raw request:

> Because of the recent change in fab-kit workflow, the README.md and other corresponding markdown files linked from there have gotten outdated. Create a task to update them.

Interaction mode: conversational. Before creating this change, the agent investigated the
actual staleness rather than assuming a blanket sweep. Key findings and decisions from that
investigation:

- The "recent change" is the **spec→apply merge** (constitution v1.3.0, commit `260601-j6cs`):
  the pipeline went from **7 stages** (`intake → spec → apply → review → hydrate → ship →
  review-pr`) to **6 stages** (`intake → apply → review → hydrate → ship → review-pr`). The
  standalone `spec` stage and `spec.md` artifact were removed; requirement capture is now
  co-generated into `plan.md`'s `## Requirements` section at apply entry. SRAD confidence
  scoring was frontloaded to intake (the single gate, flat 3.0).
- **README.md is heavily stale** — 11 markers still describe the 7-stage model.
- **The README-linked spec docs are already current** — `overview.md` ("4 core stages plus 2
  PR-side stages… no separate `spec.md`"), `glossary.md` ("formerly `spec.md`"), and the rest
  (`skills.md`, `user-flow.md`, `assembly-line.md`, `companions.md`, `srad.md`, `operator.md`)
  plus `CONTRIBUTING.md` show **0 stale markers**. They were updated as part of the merge change.
- **Decision (user-confirmed): scope is "README + audit linked docs."** Update README.md, and
  re-verify every README-linked spec file + CONTRIBUTING.md, fixing any lingering staleness found
  during the audit.
- **Decision: `docs/memory/*` is out of scope.** Those files are not linked from the README and
  are post-implementation memory that legitimately records the historical spec stage and the
  merge migration — rewriting them would corrupt accurate history (Constitution II).

## Why

1. **Problem**: The README is the project's front door (and the canonical onboarding doc, linked
   from both `docs/memory/index.md` and `docs/specs/index.md`). It still teaches a 7-stage
   pipeline with a separate `spec` stage and a `spec.md` artifact that no longer exist. A new user
   following the Quick Start will run `/fab-continue` expecting a "Planning - generates spec.md"
   step that does not occur, and will look for a `spec.md` file the pipeline never creates.
2. **Consequence if unfixed**: Onboarding friction and eroded trust. The README's diagrams,
   tables, and command walkthrough actively contradict the shipped behavior and the already-updated
   spec docs it links to — an internal inconsistency that makes the whole doc set look unmaintained.
3. **Why this approach**: Targeted correction of README.md plus a verification audit of its linked
   docs. The spec docs are owned by humans (Constitution VI) and already current; the audit confirms
   that and catches anything the merge change missed, without a risky blanket find-replace across
   `docs/memory/`.

## What Changes

### 1. README.md — pipeline stage count and narrative (7 → 6)

Every "7 stages" / "seven stages" reference becomes "6 stages". The standalone **Spec** stage is
removed; its responsibility (requirement capture) folds into **Apply**.

Specific edits:

- **Intro paragraph (line ~7)**: "a 7-stage pipeline (intake → spec → apply → review → hydrate →
  review-PR)" → "a 6-stage pipeline (intake → apply → review → hydrate → ship → review-PR)".
  (Also fixes the current list, which omits `ship` while claiming 7.)
- **Contents nav (line ~13)** and **section heading "## The 7 Stages" (line ~15)** → "The 6 Stages",
  with the anchor `#the-7-stages` → `#the-6-stages` updated everywhere it's referenced.
- **Intro sentence under the heading (line ~17)**: "moves through seven stages" → "moves through
  six stages".

### 2. README.md — the stage mermaid diagram (lines ~19–46)

Remove the `S["2 SPEC"]` node and the `B --> S` / `S --> A` edges. Renumber: Apply becomes stage 2,
Review 3, Hydrate 4, Ship 5, Review-PR 6. The "Planning" subgraph currently contains only Intake +
Spec — after removing Spec it holds just Intake; fold Intake into a restructured grouping (e.g.,
Intake stands alone or joins the Execution group). Connect `B["1 INTAKE"] --> A["2 APPLY"]` directly.

### 3. README.md — the stage table (lines ~48–57)

Drop the `| 2 | **Spec** | Define requirements | spec.md |` row. Renumber rows 3–7 to 2–6. Update
the **Apply** row to reflect that it now generates `plan.md` (with `## Requirements`, `## Tasks`,
`## Acceptance`) directly from `intake.md` — not "from spec". Suggested Apply row:

> | 2 | **Apply** | Co-generate `plan.md` (requirements + tasks + acceptance) from intake, then execute the tasks | `plan.md` + code changes |

### 4. README.md — change-folder layout (lines ~64–70)

Remove the `├── spec.md          # Requirements (generated)` line. Update the `plan.md` comment to
note it now carries requirements too:

> `├── plan.md          # Requirements + tasks + acceptance (generated at apply entry)`

### 5. README.md — Quick Start "first change" walkthrough (lines ~220–242)

The current flow shows four `/fab-continue` calls with comments "Planning - generates spec.md
(structured requirements)" then three execution steps. Reduce to the 6-stage reality:

- Remove the "Planning - generates spec.md" `/fab-continue` step.
- Relabel the first post-`/fab-new` `/fab-continue` as the **Apply** step: "generates plan.md
  (requirements + tasks + acceptance) and implements the code".
- Keep the subsequent Review and Hydrate `/fab-continue` steps.
- Verify the comment about `/fab-ff` ("skips intermediate planning stages") still reads correctly
  with no separate planning stage — reword to reflect the single intake gate if needed.

### 6. README.md — "Shared Memory" hydrate diagram (lines ~319–326)

The ASCII diagram shows `spec.md ─hydrate→ docs/memory/`. Update the source box from `spec.md` to
`plan.md` (or `change` artifacts), since hydrate now draws from the completed change's `plan.md`,
not a `spec.md`. Also fix the prose at line ~330: "Design decisions from `spec.md` merge into
memory" → "from `plan.md`".

### 7. README.md — "Code Quality as a Guardrail" diagram + prose (lines ~339–365)

- The ASCII pipeline diagram shows `intake → spec → apply ⇄ review → hydrate`. Update to
  `intake → apply ⇄ review → hydrate` (the sub-agent review loop is between apply and review).
- Prose at line ~353: "The pipeline requires intake and spec before any code is written" → "requires
  intake before any code is written" (or "intake and a generated plan").
- **Review loopback table (lines ~357–363)**: remove the `| Requirements were wrong | Must-fix | →
  spec | Updates spec, regenerates plan |` row, since there is no spec stage to loop back to. A
  wrong-requirements finding now loops back to **apply** (which regenerates `plan.md`'s
  `## Requirements`). Either drop the row or rewrite its target to `→ apply`.

### 8. README.md — Stage Coverage section (lines ~466–603)

This section has both a large mermaid `block-beta` diagram and a "Quick reference" coverage table,
both of which include a `spec` row.

- **Coverage table (lines ~592–603)**: remove the `| spec |` row. The remaining stage rows
  (context, intake, change active, branch name, apply, review, hydrate, ship, review-pr) are
  correct. Verify the ✅ marks per command still hold (e.g., `/fab-continue` covers apply, not spec).
- **`block-beta` diagram (lines ~479–588)**: remove the `row_spec` row label and every
  `*_spec` cell (`cont_spec`, `ff_spec`, `fff_spec`, `proceed_spec`) plus their `style` lines and
  the `new_branch --> *_spec` edges (re-point those edges to the `*_apply` cells). Adjust `columns`
  / `space` counts so the grid still aligns. This is the most mechanically involved edit — the
  diagram is hand-laid-out.

### 9. README.md — prose mentions of "spec" as a stage

Sweep the remaining prose for stage-sense "spec" references and correct them, e.g. line ~311
("Each change has its own spec, plan, and status") → "its own intake, plan, and status". Preserve
references to **design specs** (`docs/specs/`), which are a distinct, still-valid concept — do not
rewrite those.

### 10. Audit of README-linked docs (verification pass)

For each file linked from README.md, audit **and fix**:

- `docs/specs/overview.md`, `docs/specs/glossary.md`, `docs/specs/skills.md`,
  `docs/specs/user-flow.md`, `docs/specs/assembly-line.md`, `docs/specs/companions.md`,
  `docs/specs/srad.md`, `docs/specs/operator.md`
- `CONTRIBUTING.md`

Two classes of edit:

1. **Genuinely stale references** — any prose/diagram/table that describes the **current** model as
   7-stage or as having a `spec` stage / `spec.md` artifact. Fix in place. Preliminary grep shows 0
   such markers in these files (the merge change already updated them), so few or no edits expected
   here.
2. **`spec.md` reference cleanup (new-reader clarity)** — even where the docs are factually current,
   remove `spec.md` mentions that could confuse a reader who never saw the old artifact. Specifically:
   - `docs/specs/overview.md:69` — drop the "(one pass — **no separate `spec.md`**)" aside; the
     surrounding text already describes the one-pass `plan.md` generation correctly without it.
   - `docs/specs/glossary.md:22` — **keep** the single "formerly `spec.md`" bridge in the `plan.md`
     entry. The glossary is exactly where a returning user looks up "where did spec.md go?", so this
     one pointer stays. This is the *only* surviving `spec.md` reference across the doc set.
   - README.md — the change-folder layout, quick-start, and diagrams lose all `spec.md` mentions
     (covered by items 1–9 above); no bridging note needed there since the glossary carries it.

Because these are human-owned design specs (Constitution VI), edits are limited to factual
staleness corrections and the targeted `spec.md`-reference cleanup above — not restructuring.

## Affected Memory

No memory files are created, modified, or removed by this change. README.md and `docs/specs/*` are
documentation, not memory. `docs/memory/*` is explicitly out of scope (see Origin — those files
record historical/migration state that must not be rewritten). Hydrate for this change should be a
no-op on memory.

## Impact

- **Files touched**: `README.md` (substantial — narrative, 3 diagrams, 2 tables, change-folder
  layout, quick-start). Linked docs touched only if the audit finds genuine staleness (expected:
  none).
- **No code, CLI, or skill changes** — pure documentation. `src/`, the `fab` binary, and
  `src/kit/skills/*` are untouched.
- **Risk areas**: the two mermaid diagrams (stage flow and the `block-beta` coverage grid) are
  hand-laid-out; removing the spec node/row requires re-balancing `space`/`columns` counts so the
  rendering stays aligned. Anchor links to `#the-7-stages` must be updated in lockstep with the
  heading rename to avoid broken in-page links.
- **Verification**: render the README mermaid diagrams (or validate syntax) after editing; grep the
  final README for residual `7[ -]stage|seven stage|spec\.md|2 SPEC` markers (target: 0 in README).
  Across the whole doc set, exactly **one** intentional `spec.md` reference should survive — the
  "formerly `spec.md`" bridge in `docs/specs/glossary.md` (assumption #6).

## Open Questions

None — scope, exclusions, and the specific stale sections were resolved during the pre-creation
investigation and confirmed with the user.

## Clarifications

### Session 2026-06-02 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 3 | Changed | Scope is "audit **and fix**" the README-linked specs + CONTRIBUTING.md, not verify-only |
| 4 | Confirmed | `docs/memory/*` stays out of scope |
| 6 | Changed | After explanation: remove confusing `spec.md` mentions but keep one "formerly `spec.md`" bridge in `glossary.md` |
| 7 | Confirmed | Mermaid: Intake stands alone; drop the single-node "Planning" subgraph |

(#5 not raised — remains Confident.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The "recent workflow change" is the spec→apply merge (7→6 stages, no `spec.md`), per constitution v1.3.0 / commit `260601-j6cs`. | Verified directly in `constitution.md` governance note, `config.yaml`, and the already-updated `docs/specs/overview.md`. | S:98 R:90 A:95 D:95 |
| 2 | Certain | Change type is `docs`. | Request is to update README/markdown docs; no code or behavior changes. Matches the `docs` keyword heuristic. | S:95 R:85 A:95 D:90 |
| 3 | Certain | Scope = README.md + audit **and fix** of all README-linked spec files + CONTRIBUTING.md (fix any genuine staleness found, not just verify). | Clarified — user changed to "audit and fix the linked specs + CONTRIBUTING.md". | S:95 R:75 A:85 D:90 |
| 4 | Certain | `docs/memory/*` is excluded from scope. | Clarified — user confirmed. Not linked from README; post-implementation memory legitimately records the historical spec stage + migration (Constitution II). | S:95 R:65 A:85 D:85 |
| 5 | Confident | The README-linked spec docs + CONTRIBUTING.md are already current; the audit-and-fix pass is expected to make few or no edits beyond the spec.md-reference cleanup in #6. | Grep shows 0 stale-behavior markers; `overview.md`/`glossary.md` already describe the 6-stage model. | S:90 R:80 A:85 D:80 |
| 6 | Certain | Remove `spec.md` references that could confuse new readers (e.g. `overview.md`'s "no separate `spec.md`"), but keep **one** bridging note in `glossary.md` ("formerly `spec.md`") so returning users can map the old artifact to `plan.md`. Genuinely stale 7-stage/spec-stage descriptions are always fixed. | Clarified — user changed: a single glossary bridge stays; other spec.md mentions are scrubbed for new-reader clarity. | S:95 R:80 A:85 D:80 |
| 7 | Certain | In the README stage mermaid diagram, after removing the Spec node, Intake stands alone — drop the now-redundant single-node "Planning" subgraph and flow Intake directly into the Apply/Execution grouping. | Clarified — user confirmed the recommended layout. | S:95 R:80 A:60 D:50 |

7 assumptions (6 certain, 1 confident, 0 tentative, 0 unresolved).
