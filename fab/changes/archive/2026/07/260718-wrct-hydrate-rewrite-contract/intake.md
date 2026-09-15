# Intake: Memory Writer Contract — Hydrate Rewrites, Never Appends

**Change**: 260718-wrct-hydrate-rewrite-contract
**Created**: 2026-07-18

## Origin

One-shot invocation: `/fab-new wrct` (backlog ID). Raw backlog entry (`fab/backlog.md`, Change B of the three-change wave from the cross-repo memory audit 2026-07-18):

> [wrct] 2026-07-18: Change B: memory writer contract — hydrate REWRITES, never appends (root-cause fix from the cross-repo memory audit 2026-07-18: every problem class traces to accretion — hydrate appends each change delta instead of consolidating to present truth; narration scales with change count while accuracy stays perfect, so the writers, not the facts, are the leak; run-kit minted its worst descriptions the same week it shipped changes). (1) WRITER RULES in /fab-continue hydrate behavior + \_generation + docs-hydrate-memory: when touching a section, REWRITE it to current truth — never append a delta paragraph below the old one; headings never carry change-ids (run-kit: 60 change-id-suffixed headings); after a body edit, re-check the description still routes in ≤500 chars; add a post-hydrate self-check (re-read touched files, strip any transition phrasing just introduced). (2) RATIONALE LANDS IN DESIGN DECISIONS: any why / rejected-alternative goes into a ## Design Decisions entry (Decision/Why/Rejected/Introduced-by), never inline narration; the changelog-bullet shape (`- **{change-id} — retired X**`) is banned inside that section. Evidence: missing/misused DD is the strongest rot predictor — the worst-accreted file in every audited repo lacks or misuses it (loom docwriter-system.md, run-kit ui-patterns.md, idea ci+release pipeline.md), while idea skill.md (proper DD entries) is a near-perfect exemplar. (3) FKF §3.3 ADDITIONS: operational TODOs/follow-ups belong in the backlog or change folder, never memory bodies (idea evidence: a gh-secret-delete follow-up embedded in memory); heading change-id ban. Update BOTH docs/specs/fkf.md AND src/kit/reference/fkf.md. Surfaces: skill sources + their SPEC-\*.md mirrors per constitution (sweep the whole mirror class). PARALLEL with [mxgu] (shared fkf.md merge seam only); [dsrx] cites the rules this change writes.

No prior conversation context — the backlog entry (authored from the audit findings) is the sole decision source.

## Why

1. **The pain point**: the cross-repo memory audit (2026-07-18, loom / run-kit / idea) found that every memory-rot problem class traces to **accretion by writers**, not to wrong facts — staleness was 0/18 spot-checks, yet narration density scales monotonically with change count. Hydrate-style writers append a delta paragraph per change instead of consolidating the touched section to present truth; run-kit minted its worst descriptions the same week it shipped changes, and carries 60 change-id-suffixed headings. Missing or misused `## Design Decisions` is the strongest rot predictor: the worst-accreted file in every audited repo lacks or misuses it (loom `docwriter-system.md`, run-kit `ui-patterns.md`, idea `ci+release pipeline.md`), while idea `skill.md` — which uses proper DD entries — is a near-perfect exemplar.
2. **The consequence of not fixing**: the sibling changes are treadmills without this one — [mxgu]'s index guards only *detect* debt and [dsrx]'s distill extensions only *drain* it; if writers keep minting narration at every hydrate, detection and drainage never converge. This change is the root-cause fix: stop the leak at the writers.
3. **Why this approach**: fab-kit's writers are pure prompt-play — the writer contract lives in skill markdown (`fab-continue` Hydrate Behavior, `docs-hydrate-memory`) and the FKF spec (§3.3). The core rewrite-not-append rule already shipped with the FKF work; the audit shows the *residual* leak classes (change-id headings, drifting descriptions after body edits, no final self-review, rationale written as inline narration instead of DD entries, embedded TODOs). Closing them is rule additions in the same normative homes the existing rules occupy — no new mechanism, no Go changes ([mxgu] owns enforcement signals).

## What Changes

Verified current state (gap analysis 2026-07-18): the rewrite-to-current-truth rule is already normative in `src/kit/skills/fab-continue.md` § Hydrate Behavior step 4 ("Merge as current truth"), `src/kit/skills/docs-hydrate-memory.md` ingest Step 3 item 4, and FKF §3.3 ("Body style: state current truth in present tense"). **None of the four new rule classes below exist anywhere today** (grep-verified: no heading change-id ban, no TODO-relocation rule, no post-hydrate self-check, no DD changelog-bullet ban). This change therefore *extends the existing writer contract in place* — it does not restructure the shipped rules.

### 1. Writer rules — `fab-continue` Hydrate Behavior + `docs-hydrate-memory`

Add to the current-truth merge bullets (`fab-continue.md` § Hydrate Behavior step 4; `docs-hydrate-memory.md` ingest Step 3 items 3–4 and the FKF-frontmatter paragraph below them, plus generate-mode Step 3 which references ingest Step 3):

- **Heading change-id ban**: a heading names its topic (`## Dispatch States`), never a change (`### Dispatch States (xu0k)` / `## xu0k — dispatch states`). Change-ids appear only as trailing citations in body text, never in heading text. (Evidence: run-kit's 60 change-id-suffixed headings.) Writers never *introduce* such headings; *draining* existing ones in other repos is [dsrx]'s distill extension, not this change.
- **Description re-check after body edit**: after any body edit, re-check the file's `description:` frontmatter still routes the file accurately — one line, ≤500 chars, change-id-free (FKF §3.2). Today's rules say "keep `description:` accurate, within the cap" at merge time; the explicit *post-body-edit trigger* is what's added (the audit found descriptions drifting to 33x/50x cap because nobody re-read them after body growth).
- **Post-hydrate self-check** (new step, not a bullet): after all memory writes and before returning/regenerating indexes, re-read every file touched this run and strip any transition phrasing just introduced — no "renamed/now/previously/no longer/was `old.value`" narration, no change-keyed delta paragraph left below an older paragraph on the same topic, no change-ids in headings, descriptions still route. Scoped to files touched this run (it is a self-review of this hydrate's own writes, not a corpus sweep). Lands as a new numbered step in `fab-continue.md` § Hydrate Behavior (between step 4 "Hydrate docs/memory/" and the return step) and an equivalent step in `docs-hydrate-memory.md` ingest/generate modes (before index regeneration). Backfill mode is exempt — it is a pure-frontmatter, body-preserving operation.

### 2. Rationale lands in Design Decisions

Add to the same writer surfaces AND FKF §3.3 (see area 3):

- Any *why*, rejected alternative, or constraint explanation goes into a `## Design Decisions` entry in the four-field shape (**Decision** / **Why** / **Rejected** / *Introduced by*), never as inline narration in Overview/Requirements prose. This gives "don't re-break this" content a durable, present-tense home (the audit's near-perfect exemplar, idea `skill.md`, does exactly this).
- The **changelog-bullet shape is banned inside `## Design Decisions`**: entries like `- **{change-id} — retired X**` are change history (that's `log.md`'s job, FKF §6), not design decisions. A DD entry heading is a decision *title*, never a change-id.
- `fab-continue.md` Hydrate step 6 (Pattern capture) already routes patterns into DD with citation-form provenance — align its wording with the four-field entry shape.

### 3. FKF §3.3 additions — BOTH `docs/specs/fkf.md` AND `src/kit/reference/fkf.md`

The two copies must never diverge — amend both identically (this is the sole merge seam shared with [mxgu], which amends §3.2/§4 of the same files; coordinate at merge, no ordering dependency). Additions to the normative "Body style" bullet list in §3.3:

- **No operational TODOs**: follow-up work items (TODOs, "still needs X", next-step checklists) are never memory-body content — they belong in the project backlog (`fab/backlog.md`) or the originating change folder. A memory body states what IS, not what remains to be done. (Evidence: a gh-secret-delete follow-up embedded in an idea memory body.)
- **Headings carry no change-ids**: heading text names the topic; provenance stays citation-only in body text (extends the existing "Provenance is citation-only" bullet).
- **Design Decisions entry shape**: rationale relocation + the changelog-bullet ban from area 2, added to the §3.3 conventional-structure guidance around the existing four-field DD scaffold.

### 4. `_generation.md` touchpoint

`_generation.md` contains **no memory-writing procedure** (verified: its only memory references are intake's Affected Memory guidance). Its one seam where the writer contract applies is the Plan Generation Procedure's `### Design Decisions` subsection (currently "summary + rationale + rejected alternatives"): align its entry shape with the four-field DD form so hydrate's pattern capture can lift plan DD entries into memory DD without reshaping. This is the minimal edit consistent with the backlog naming `_generation` as a surface.

### 5. Mirror + template sweep

Per constitution ("Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md` file") and the backlog's "sweep the whole mirror class":

- `docs/specs/skills/SPEC-fab-continue.md`, `SPEC-docs-hydrate-memory.md`, `SPEC-_generation.md` — mirror the skill edits.
- `src/kit/templates/memory.md` — its guidance comments already restate §3.3 body style; add the new rules (heading ban, no-TODOs, DD shape/bullet ban) so the template a writer reads at file-creation time carries the full contract.
- Sweep check at apply: `docs/specs/skills.md` (aggregate) and `docs/specs/templates.md` — update only where they restate the amended rules (grep for restatements of the merge/body-style rules, including user-facing string literals, per the class-sweep discipline).

**Not in scope**: enforcement/warnings (`fab memory-index` narration-density, size caps, blocking tiers — [mxgu]); distill/reorg drain of existing debt in other repos ([dsrx]); any Go code change.

## Affected Memory

- `pipeline/execution-skills`: (modify) hydrate writer-contract additions — heading change-id ban, description re-check trigger, post-hydrate self-check step, DD entry-shape alignment
- `memory-docs/hydrate`: (modify) same rule additions on the `/docs-hydrate-memory` ingest/generate paths (backfill exempt)
- `memory-docs/templates`: (modify) memory-file format additions — FKF §3.3 new bullets (no-TODOs, heading ban, DD shape) + `templates/memory.md` guidance-comment updates
- `pipeline/planning-skills`: (modify) plan `### Design Decisions` four-field entry-shape alignment in `_generation`

## Impact

- **Markdown-only change** — skill sources (`src/kit/skills/fab-continue.md`, `docs-hydrate-memory.md`, `_generation.md`), spec + kit-reference FKF copies (`docs/specs/fkf.md`, `src/kit/reference/fkf.md`), template (`src/kit/templates/memory.md`), SPEC mirrors (`docs/specs/skills/SPEC-*.md`). No Go code, no tests to modify; no `fab` CLI surface change (so no `_cli-fab.md` update).
- **Coordination**: runs PARALLEL with [mxgu] — the only overlap is that both amend the two `fkf.md` files (merge seam, not an ordering dependency). [dsrx] is downstream: its distill removal classes cite the §3.3 rules this change writes, so this change should land before [dsrx] starts.
- **Behavioral blast radius**: every future hydrate (pipeline and standalone) in every fab-kit-driven repo picks up the tightened contract on the next kit release + sync; no data migration, no state change.

## Open Questions

None — the backlog entry resolves scope, rule content, surfaces, and sibling boundaries explicitly; the one ambiguous surface (`_generation`) is resolved as a Confident assumption (row 4).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is writer rules only — no enforcement signals (that's [mxgu]) and no drain of existing debt (that's [dsrx]) | Backlog states the three-change split and each sibling's ownership explicitly | S:95 R:90 A:95 D:95 |
| 2 | Certain | Both `fkf.md` copies (`docs/specs/` + `src/kit/reference/`) receive identical §3.3 amendments | Backlog: "Update BOTH ... they must never diverge"; same rule already governs the [mxgu] seam | S:95 R:85 A:95 D:95 |
| 3 | Certain | Extend the shipped current-truth rules in place rather than restructure them — the rewrite-not-append core already exists at `fab-continue.md` § Hydrate step 4, `docs-hydrate-memory.md` Step 3, FKF §3.3 | Grep-verified during gap analysis; only the four new rule classes are absent | S:80 R:85 A:90 D:75 |
| 4 | Confident | `_generation.md`'s touchpoint is the Plan Generation `### Design Decisions` entry shape (align to the four-field DD form); it contains no memory-writing procedure to attach hydrate rules to | Verified by grep; the alternative (backlog mislabel, no `_generation` edit) is weaker — the plan-DD → memory-DD lift is a real seam and the minimal edit honoring the named surface | S:45 R:80 A:70 D:55 |
| 5 | Confident | Post-hydrate self-check lands as a new numbered step (after step 4 in `fab-continue` Hydrate Behavior; before index regen in `docs-hydrate-memory` ingest/generate), scoped to files touched this run; backfill mode exempt | Backlog specifies the check's content ("re-read touched files, strip any transition phrasing just introduced") but not placement; these are the procedure seams where a final pass fits, and backfill is body-preserving by contract | S:70 R:85 A:80 D:70 |
| 6 | Confident | `src/kit/templates/memory.md` guidance comments + the SPEC mirror class (`SPEC-fab-continue`, `SPEC-docs-hydrate-memory`, `SPEC-_generation`) are in-scope surfaces; `docs/specs/skills.md`/`templates.md` updated only where they restate amended rules | Constitution mandates the SPEC mirrors; the template already restates §3.3 and would contradict the new rules if skipped; class-sweep discipline (code-quality § Sibling & Mirror Sweeps) | S:55 R:90 A:85 D:75 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
