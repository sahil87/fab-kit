# Intake: Scaffold/Config Truth + SRAD Scoring Coherence

**Change**: 260612-c5tr-scaffold-config-truth-srad-coherence
**Created**: 2026-06-12

## Origin

One-shot invocation: `/fab-new c5tr` (cold start — no prior conversation context). The 4-char ID resolves to backlog entry `[c5tr]` dated 2026-06-12 — **batch 4/5 of the skills-audit 2026-06-12**, covering report §2 Themes 5 (scaffold/config rot) and 6 (SRAD scoring incoherence).

> **Source caveat**: the `[c5tr]` backlog entry and the audit report (`docs/specs/findings/skills-review-2026-06-12.md`) are **uncommitted working-tree changes in the main repo** at intake time. This worktree (branch `c5tr`, forked at v2.1.6 / `1431a9c3`, even with `main`) contains neither. The report must be committed (to main or this branch) before apply can cite it from here. All file:line references below are vs `1431a9c3`.

Raw backlog entry:

> [c5tr] 2026-06-12: Skills-audit batch 4/5 — scaffold/config truth + SRAD scoring coherence. PARALLEL: wave 1 — safe alongside k4ge and g8st (overlaps are same-file-different-section only: _preamble.md:40 vs k4ge's :230-232, fab-new.md:182-189 vs g8st's :154-160, shared SPEC mirrors; second-to-merge rebases). GOAL: what ships to new projects and what the scoring layer computes both match the 6-stage reality. ACTIONS (report §2 Themes 5+6): SCAFFOLD — src/kit/scaffold/fab/project/config.yaml:31-38 still seeds stage_directives.spec (must-fix): new projects never run the relocating migration (fab init stamps fab_version past it), so the zombie key is permanent; move/cull the directives — the relocated "Mark ambiguities with [NEEDS CLARIFICATION]" apply-directive contradicts src/kit/templates/plan.md:36 (markers are intake-only post-1.10.0); this repo's own fab/project/config.yaml:19-21 carries the same residue, clean it too. Decide wire-or-remove (ask user): stage_directives (scaffold promises it, fab-setup.md:167 edits it, three migrations preserve it, ZERO readers in skills or Go) and stage_hooks (LIVE in config.go:18 + status.go:603-621 — pre blocks start, post runs after save — documented NOWHERE, incl. the failing-post-hook re-run trap where the stage is already done); model_tiers likewise has zero consumers. Retire-or-regenerate src/kit/schemas/workflow.yaml — still defines the 7-stage pipeline with spec (must-fix); nothing consumes it yet user-flow.md:201 calls it source of truth (recommend retire + repoint to the Go state machine). Restore the one-line semver-comparison rule fab-setup's three-way branch needs (#393 f080 dedup deleted it; fab-setup.md:303 vs :294); fix the fab_version create-mode fallback (:95/:153) and Next-Steps drift (:430-436). Scaffold code-review.md:44 names the removed "revise spec" escalation; the "Max cycles: 3" knob is consumed by nothing (_pipeline hard-codes 3) — wire or annotate. _preamble.md:40 always-load descriptor still advertises removed naming + dead model tiers. SRAD — half-open grade bands ≥85/≥60/≥30/else (closed integer bands at _srad.md:30 leave composites like 59.85/84.5 ungraded; boundary worth 0.7 score/row); ONE Critical-Rule number (_srad.md:47 <25 override vs :30 0-39 Low band); re-dimension Worked Example 3 (Certain grade mathematically unreachable, cap 84.75<85, _srad.md:71); fix srad.md:54-118 Assumptions-table contract (Scores "optional", Certain/Unresolved rows omitted — emits tables fab score cannot fully parse, deflating gate inputs) + Example 1 arithmetic (:246-248); fab-clarify: evaluate bulk-confirm BEFORE the Step 1.5 zero-gaps early exit (must-fix; :55 vs :61-73 — a below-gate Confident-only intake currently dead-ends at "artifact looks solid") + S→95 upgrade labels rows Certain below the 85 threshold (:110-118) + audit-trail asymmetry (:122/:153); fab-new.md:182-189 output ordering violates _srad's Assumptions-block-last SHALL; _generation.md:75-78 plan walk never emits the ## Assumptions section it depends on (add the explicit step; reconcile omit-when-zero vs scaffolded templates); _srad.md:51-57 autonomy table covers 4 of the 6 declaring skills. CONSTRAINTS: removals need migrations in src/kit/migrations/; Go changes need tests + _cli-fab updates; SPEC mirrors incl. docs/specs/srad.md. REPORT: docs/specs/findings/skills-review-2026-06-12.md §2 Themes 5+6.

## Why

**Theme 5 — what ships to new projects lies about the system.** `fab init` stamps `fab_version` past all migrations, so a freshly scaffolded project never runs the 1.9.7→1.10.0 relocation — the zombie `stage_directives.spec` key (with its dead GIVEN/WHEN/THEN defaults and the `[NEEDS CLARIFICATION]` directive that contradicts `src/kit/templates/plan.md:36`, markers being intake-only post-1.10.0) is **permanent** in every new project. Meanwhile the config surface is dishonest in both directions: `stage_directives` and `model_tiers` are promised/preserved but have zero readers in skills or Go, while `stage_hooks` is live Go behavior (`config.go:18`, `status.go:603-621` — pre blocks `start`, post runs after save) documented nowhere, including a re-run trap (a failing post-hook leaves the stage `done`, so the documented re-run hits done→done). `src/kit/schemas/workflow.yaml` still defines the retired 7-stage pipeline with `spec`, yet `docs/specs/user-flow.md:201` calls it source of truth. And #393's f080 dedup deleted the semver-comparison rule `fab-setup`'s three-way version branch depends on — a refactor-introduced regression.

**Theme 6 — the scoring rubric behind the pipeline's single gate is internally inconsistent.** All human judgment is frontloaded to intake; the flat 3.0 gate is the only check before autonomous execution. Holes in the rubric move real scores across that gate: closed integer bands leave continuous composites (59.85, 84.5) ungraded (~0.7 score/row at stake); the Critical Rule has two competing numeric definitions (<25 at `_srad.md:30` vs the 0–39 Low band read of `:47`); Worked Example 3's Certain grade is mathematically unreachable (max composite 84.75 < 85); `docs/specs/srad.md:54-118` contradicts `_srad` (Scores "optional", Certain/Unresolved rows omitted) producing tables `fab score` cannot fully parse — deflating gate inputs; and `fab-clarify`'s zero-gaps early exit makes bulk-confirm unreachable in its primary scenario, dead-ending a below-gate Confident-only intake at "artifact looks solid".

**If unfixed**: every new project is seeded with self-contradicting config, and the one gate that decides whether a change runs unattended is computed from a rubric that grades the same intake differently depending on which line the agent pattern-matched. Both themes are correctness debt in the layer users can't see drift in.

**Why this approach**: batch 4/5 of the curated audit rollout — same proven shape as the merged 06-11 batches (#390–#393). The report suggested two separate changes (`scaffold-config-truth`, `srad-scoring-coherence`); the backlog entry deliberately combines them into one wave-1 batch.

## What Changes

### 1. Scaffold de-zombification (Theme 5, must-fix)

`src/kit/scaffold/fab/project/config.yaml:31-38` currently ships:

```yaml
stage_directives:
  intake: []
  spec:
    - Use GIVEN/WHEN/THEN for scenarios
    - "Mark ambiguities with [NEEDS CLARIFICATION]"
  apply: []
  review: []
  hydrate: []
```

**Resolved (asked at intake): `stage_directives` is removed.** The whole block (scaffold lines 29–38, comment included) leaves the scaffold; fab-setup's stage_directives editor (`fab-setup.md:167`) goes with it; a new migration in `src/kit/migrations/` drops the key from populated user configs. The directive-relocation sub-question dissolved with the removal — no directive survives anywhere. (For reference: `Use GIVEN/WHEN/THEN for scenarios` was already redundant with `_generation.md` step 3, which mandates GIVEN/WHEN/THEN scenarios; `Mark ambiguities with [NEEDS CLARIFICATION]` is illegal outside intake post-1.10.0.)

This repo's own `fab/project/config.yaml:18-26` carries the same residue (relocated under `apply:`, including the contradictory marker directive) — remove that block in the same pass (direct edit or via the migration).

### 2. Wire-or-remove: `stage_directives`, `model_tiers`, `stage_hooks` (Theme 5, user decision)

- **`stage_directives`** — scaffold promises it, `fab-setup.md:167` edits it, three migrations preserve it, **zero readers** in skills or Go. Wire = `_generation`/`_review` consume `stage_directives.{stage}` (adds prompt surface to every generation); remove = scaffold + fab-setup menu + migration.
- **`model_tiers`** — zero consumers; not even in the current scaffold, but still advertised by `_preamble.md:40`'s always-load descriptor.
- **`stage_hooks`** — **live** Go behavior (`config.go:18`, `status.go:603-621`): pre-hook failure blocks `fab status start`; post-hook runs after save. Documented nowhere. Document = `_cli-fab.md` section incl. the failing-post-hook re-run trap; remove = Go change (tests + `_cli-fab` update per constitution).

**Resolved (asked at intake): remove dead, document live.** `stage_directives` and `model_tiers` are removed everywhere they appear (scaffold, the `fab-setup.md:167` editor, the `_preamble.md:40` descriptor, plus a new migration dropping the key from populated user configs — the three existing migrations that *preserve* `stage_directives` stay untouched; only the new migration handles the drop). `stage_hooks` stays live and gets documented in `_cli-fab.md`: pre-hook failure blocks `fab status start`, post-hook runs after save, **including the failing-post-hook re-run trap** (stage already `done`; the documented re-run hits done→done). **No Go changes** in this disposition.

### 3. Retire `src/kit/schemas/workflow.yaml` (Theme 5, must-fix)

The schema still defines the 7-stage pipeline including `spec`. Nothing consumes it; it has drifted a full pipeline generation unnoticed. **Retire** it (recommended in both the backlog entry and report Structural bet 5) rather than regenerate: delete the file, repoint `docs/specs/user-flow.md:201`'s "source of truth" claim at the Go state machine (`src/go/fab/internal/status`), and update `docs/memory/pipeline/schemas.md` which documents it. Tradeoff accepted: loses the one declarative schema artifact.

### 4. fab-setup repairs (Theme 5)

- **Restore the one-line semver-comparison rule** the three-way version branch needs (`fab-setup.md:303` references what `:294`'s context lost) — deleted by #393's f080 triplication-dedup. Regression fix, not new behavior.
- **`fab_version` create-mode fallback** (`:95`/`:153`): the "guarantee" that `fab_version` exists has no fallback when config is created fresh in that path.
- **Next Steps drift** (`:430-436`): the lines no longer match the `_preamble` State Table they claim to derive from.

### 5. Scaffold `code-review.md` truth (Theme 5)

- `src/kit/scaffold/fab/project/code-review.md:44` escalation names the removed "revise spec" path — reword to the post-1.10.0 vocabulary (e.g., "revise tasks" / "revise requirements"). This repo's local `fab/project/code-review.md:44` has the same line — clean both.
- The **"Max cycles: 3"** Rework Budget knob is consumed by nothing (`_pipeline` hard-codes 3). **Resolved (asked at intake): wire it** — `_pipeline` reads `code-review.md`'s Rework Budget when the file exists, defaults to 3 otherwise.

### 6. `_preamble.md:40` always-load descriptor (Theme 5)

The config.yaml line in the Always Load list still advertises "naming conventions, model tiers" — removed/dead features. Rewrite the descriptor to the real config surface. **Wave-1 overlap**: k4ge touches `_preamble.md:230-232` — different section, second-to-merge rebases.

### 7. SRAD band math (`_srad.md`, Theme 6)

- `:30` — replace closed integer bands `Certain (85–100), Confident (60–84), Tentative (30–59), Unresolved (0–29)` with **half-open thresholds: ≥85 Certain, ≥60 Confident, ≥30 Tentative, else Unresolved**. Composites are continuous (weighted mean) — 59.85 and 84.5 currently match no band.
- `:47` vs `:30` — **one Critical-Rule number**: standardize on the `R < 25 AND A < 25` override already in the aggregation line; rewrite `:47`'s prose ("low Reversibility AND low Agent Competence") to cite `< 25` explicitly instead of implying the 0–39 Low band.
- `:71` — Worked Example 3 claims Certain, but with S: Low the composite caps at `0.25×39 + 0.30×100 + 0.25×100 + 0.20×100 = 84.75 < 85`. Re-dimension the example (raise S or change the expected grade) so its arithmetic reaches the grade it teaches.
- `:51-57` — the Skill-Specific Autonomy Levels table covers 4 columns (fab-new, fab-continue, fab-fff, fab-ff) but `_srad` is declared by **6** skills — add fab-draft and fab-clarify columns (or a covering note).

### 8. `docs/specs/srad.md` contract + arithmetic (Theme 6, must-fix)

- `:54-118` — the spec's Assumptions-table contract says the Scores column is "optional" and shows Certain/Unresolved rows omitted. `_srad` (canonical) requires Scores on every row and all four grades recorded in intake artifacts; `fab score` parses accordingly. Align the spec to `_srad` so it stops teaching agents to emit tables that deflate gate inputs.
- `:246-248` — fix Example 1's two wrong composite values (row-1 arithmetic verified correct; grade outcomes mostly unaffected per report §5).

### 9. fab-clarify escape-valve fixes (Theme 6, must-fix)

- **Bulk-confirm before early exit** (`:55` vs `:61-73`): Step 1.5's zero-gaps early exit ("artifact looks solid") fires before bulk-confirm is evaluated, making bulk-confirm unreachable in its primary scenario — a marker-free, Confident-only intake sitting **below** the 3.0 gate. Reorder: evaluate the bulk-confirm trigger before the zero-gaps exit.
- **S→95 upgrade mislabeling** (`:110-118`): the confirm flow sets S to 95 and labels rows Certain, but the resulting composite can sit below the 85 threshold. Label by recomputed composite, not by fiat.
- **Audit-trail asymmetry** (`:122`/`:153`): placement/append rules differ between the two paths that write the audit trail — unify.

### 10. fab-new output ordering + plan-walk Assumptions step (Theme 6)

- `fab-new.md:182-189` — the Output template orders blocks in violation of `_srad`'s SHALL that the Assumptions summary is the **final content block immediately before `Next:`**. Reorder the template. **Wave-1 overlap**: g8st touches `fab-new.md:154-160` — different section, second-to-merge rebases.
- `_generation.md:75-78` — the Plan Generation walk consumes SRAD assumptions ("resolved inline as a graded SRAD assumption recorded in the plan's `## Assumptions` section") but no step ever **emits** that section. Add the explicit emission step to the walk. **Resolved (asked at intake)**: reconcile by scoping `_srad`'s omit-when-zero rule to the **displayed output summary only** — artifacts (intake.md / plan.md) always carry the `## Assumptions` section, with a "0 assumptions" footer when empty, keeping `fab score` parsing uniform.

## Affected Memory

- `pipeline/planning-skills`: (modify) — `_srad` band math/Critical Rule/autonomy table, `_generation` plan-walk Assumptions step, fab-new output ordering
- `pipeline/clarify`: (modify) — bulk-confirm-before-early-exit reorder, S→95 relabeling, audit-trail symmetry
- `pipeline/schemas`: (modify) — workflow.yaml retirement; repoint schema authority at the Go state machine
- `pipeline/change-lifecycle`: (modify) — `stage_hooks` documentation (pre blocks start, post runs after save, failing-post-hook re-run trap)
- `distribution/setup`: (modify) — semver rule restoration, fab_version fallback, Next Steps fix, stage_directives editor removal
- `distribution/kit-architecture`: (modify) — scaffold contents change (stage_directives block removed, code-review.md escalation fix)
- `distribution/migrations`: (modify) — new stage_directives/model_tiers removal migration
- `_shared/configuration`: (modify) — stage_directives schema block, "model tiers" mention, fab-setup menu numbering
- `pipeline/execution-skills`: (modify) — "up to 3 cycles" → `{max_cycles}` knob (code-review.md Rework Budget, default 3)

## Impact

- **Kit sources** (`src/kit/` canonical): `scaffold/fab/project/config.yaml`, `scaffold/fab/project/code-review.md`, `schemas/workflow.yaml` (delete), `skills/_srad.md`, `skills/_generation.md`, `skills/_preamble.md`, `skills/fab-new.md`, `skills/fab-clarify.md`, `skills/fab-setup.md`, `skills/_pipeline.md` (Max-cycles wiring), `skills/_cli-fab.md` (stage_hooks documentation), `migrations/` (new removal migration)
- **Go**: none — the resolved disposition keeps `stage_hooks` live (`config.go:18`, `status.go:603-621` untouched) and `stage_directives`/`model_tiers` have no Go readers to remove
- **Docs**: `docs/specs/srad.md`, `docs/specs/user-flow.md:201`, SPEC mirrors for every touched skill (`docs/specs/skills/SPEC-*.md`), affected memory files at hydrate
- **This repo's project files**: `fab/project/config.yaml:19-21`, `fab/project/code-review.md:44`
- **Parallel wave 1**: runs alongside k4ge and g8st; overlaps are same-file-different-section only (`_preamble.md`, `fab-new.md`, shared SPEC mirrors); whichever merges second rebases. Not in conflict with w7dp (that collision is g8st's).
- **External dependency**: the audit report is uncommitted in the main repo — commit it (main or this branch) before apply needs to cite finding details beyond what this intake captures.

## Open Questions

None — the wire-or-remove cluster, the omit-when-zero direction, and the Max-cycles knob were all asked and resolved at intake (Assumptions rows 2, 5, 6).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope = backlog entry `[c5tr]` actions verbatim (audit Themes 5+6), as one combined change | Entry is a curated near-spec with file:line targets; combining the report's two suggested slugs into batch 4/5 was the user's explicit batching decision | S:95 R:75 A:90 D:90 |
| 2 | Confident | Remove `stage_directives` + `model_tiers` (scaffold, fab-setup editor, _preamble descriptor, + removal migration); keep `stage_hooks` and document it in `_cli-fab.md` incl. the re-run trap. No Go changes | Asked — user chose remove-dead/document-live. Confident not Certain: removal ships a migration to user projects, so reversibility stays moderate even with the decision made | S:95 R:50 A:100 D:100 |
| 3 | Confident | `workflow.yaml` is retired, not regenerated; `user-flow.md:201` repoints to the Go state machine | Recommended by both the backlog entry and report Structural bet 5; zero consumers; regeneration would re-create a drift-prone artifact | S:85 R:65 A:80 D:80 |
| 4 | Confident | Critical Rule standardizes on the `R<25 AND A<25` override; `:47` prose cites it | The `<25` form is the operative aggregation-line rule; the 0–39 reading comes from loose prose, not a definition | S:70 R:90 A:75 D:70 |
| 5 | Certain | Omit-when-zero scoped to the displayed output summary only; artifacts always carry `## Assumptions` (with a "0 assumptions" footer when empty) | Asked — user confirmed always-emit-in-artifacts; keeps `fab score` parsing uniform | S:95 R:80 A:100 D:100 |
| 6 | Certain | "Max cycles: 3" knob gets wired — `_pipeline` reads code-review.md Rework Budget when present, defaults to 3 | Asked — user confirmed wire over annotate/remove; consistent with the audit's honor-documented-contracts theme | S:95 R:70 A:100 D:100 |
| 7 | Certain | Constitution compliance: removals ship migrations; every touched skill updates its SPEC mirror (incl. `docs/specs/srad.md`); `src/kit/` is canonical — never edit `.claude/skills/` | Restated as hard CONSTRAINTS in the backlog entry; mandated by constitution Additional Constraints | S:95 R:80 A:95 D:95 |
| 8 | Confident | Proceed in parallel wave 1 despite k4ge/g8st same-file overlaps; second-to-merge rebases | Entry explicitly declares wave-1 safety and the rebase rule; overlaps verified section-disjoint | S:85 R:70 A:80 D:85 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
