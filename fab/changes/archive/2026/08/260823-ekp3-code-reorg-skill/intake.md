# Intake: code-reorg skill

**Change**: 260823-ekp3-code-reorg-skill
**Created**: 2026-08-23

## Origin

Conversational — designed during a `/fab-discuss` session (2026-08-23), queued via `/fab-draft`, then reviewed by a 4-lens agent panel and amended per its findings plus a user scope decision.

> I want to create a skill the thinks about the folder structure of a repo and suggests improvement. Not functionality, just the folder structure, file names, and consolidations that come about as a result.

Key decisions reached in the discussion and review cycle:

- **Framing**: folder structure is a *prediction interface* — a structure is good when a reader can predict where a thing lives from its name and what a file contains from its path. Proposals must fix concrete prediction failures, never re-taxonomize for taste.
- **Name**: `code-reorg` — family fit with `docs-reorg-memory` / `docs-reorg-specs` (same analyze-and-propose posture, source-tree scope).
- **Report-only (user decision, post-review)**: the skill's job is to find problems and present suggestions — full stop. It does **not** apply changes, draft intakes, or decide how a fix is routed (micro change vs `/fab-new` vs ignore); that routing is the user's per-finding choice. An earlier draft borrowed `fab-dedupe`'s drafted-intake handoff; the review panel showed that coupling conflicts with the micro-change doctrine, drags in `_intake`/`_generation` consumer-enumeration updates, and rests on an unproven value loop (fab-dedupe's identical funnel has produced zero drafted intakes since shipping 2026-07-28). The user then cut the coupling entirely.
- **Hard boundary**: structure only — where files live and what they're called, with no behavioral change. Content/functionality judgment is out of scope; smelled content duplication emits a pointer to `/fab-dedupe` instead of being clustered here.

## Why

1. **Problem**: repositories accrue structural drift — junk drawers (`utils/`, `misc/`), singleton folders, naming inconsistency among siblings, files that co-change constantly but live far apart. No existing skill surfaces this for source trees: `docs-reorg-memory` and `docs-reorg-specs` cover only the two docs trees, and `fab-dedupe` covers content duplication, not placement/naming.
2. **Consequence of not fixing**: structural drift compounds silently, taxing every future reader (human or agent). Structural work does get prioritized in this repo when someone notices it (the cli-layering plan, the skill-prose consolidation wave) — what is missing is the *noticing*: a mechanical, evidence-gathering sweep that surfaces drift with data (co-change statistics, sibling-consistency counts, blast-radius measurements) that ad-hoc discussion doesn't collect. That evidence layer, not the conclusions, is what the skill adds over a plain conversation.
3. **Why this approach**: analyze → propose → present, entirely read-only. Rejected alternatives: (a) the skill applying moves/renames directly — renames can have blast radius (imports, doc cross-links, open branches) that deserves human routing; (b) the skill drafting intakes for accepted proposals (the fab-dedupe model) — rejected by user decision after review: it forces routing decisions the skill shouldn't own (a one-rename proposal is a micro change that must skip fab entirely; a cross-cutting move deserves a pipeline change), couples the skill to the `_intake`/`_generation` machinery, and the precedent handoff has never actually been exercised.

## What Changes

### New skill: `src/kit/skills/code-reorg.md`

A user-invocable, **fully read-only** skill `/code-reorg [<path>]` with three responsibilities:

1. **Detect** structural prediction failures in a scoped source tree.
2. **Propose** evidence-backed fixes (moves, renames, folder merges/splits), each with rationale, blast radius, and confidence — "do nothing" is a first-class outcome.
3. **Present** a ranked findings report. The report is the skill's terminal output — it modifies no files, drafts no intakes, and advances no stage.

It explicitly does **not**: judge code content or functionality, merge semantically duplicated code (points at `/fab-dedupe`), apply any change, or decide/execute how a finding is fixed. Each proposal MAY carry a *suggested next action* line as pure information — e.g. `micro: rename directly and commit` or a ready-to-paste `/fab-new <description>` one-liner — but acting on it is entirely the user's choice.

**Flow (five steps):**

1. **Scope** — resolve the target path from the argument, defaulting to all `source_paths` from `fab/project/config.yaml` swept as one combined scope. Echo the resolved scope and file count before analysis (a silently mis-resolved scope wastes the run). **Docs-tree carve-out**: a path inside `docs/memory/` or `docs/specs/` is refused with a pointer to `/docs-reorg-memory` / `/docs-reorg-specs` — those trees have FKF-aware sibling skills. The carve-out is **path-based, not content-based**: those two trees are the only exclusions, and everything else inside the scoped path is in scope regardless of file type — markdown in a source tree is just source, judged by the same placement/naming frames. Load the always-load layer; treat `fab/project/context.md`'s documented repo structure as the project's own convention frame.
2. **Gather signals** (mechanical, read-only):
   - Tree shape: depth, fan-out, singleton folders, oversized flat dirs, junk drawers (`utils/`, `helpers/`, `misc/` with unrelated contents)
   - Naming: casing / plural-singular / stutter inconsistencies among siblings (e.g., `pane/pane_send.go`)
   - Co-change coupling from `git log --name-only -M`, **with mandatory noise controls**: bound the history window (default: last 12 months), exclude bulk commits (default: any commit touching > 20 files — repo-wide sweeps manufacture spurious pairwise coupling), follow renames (`-M`), and before interpreting any pair, derive the *mandated-coupling carve-out* from the constitution and code-quality files (pairs whose co-change is required by project rules — e.g., a CLI-source ⇒ CLI-reference-doc rule — are expected coupling, not misplacement evidence) plus the *cross-layer whitelist* (tests, docs, and specs legitimately co-change with the source they cover; never propose colocating across layers against ecosystem convention). The skill file ships a worked co-change command — the pairwise aggregation is the one signal too fiddly to improvise per run.
   - Static import-direction: a file whose imports and importers concentrate in a different folder than its own is placement evidence — a deterministic complement to co-change, which only proxies this behaviorally
   - Note: reference density (what links/imports into a file) is computed lazily in step 4 for clustered proposals only — it is the blast-radius input, not a detection signal.
3. **Evaluate against frames**, in priority order: (a) the project's own stated conventions (`context.md`), (b) ecosystem convention (e.g., Go project layout), (c) internal sibling consistency. Every finding names the signal or frame it violates.
   - **Taste guard (load-bearing)**: a finding with no cited layer-2 signal and no named frame violation is dropped. "I would have organized it differently" does not qualify.
   - **Frame-(c) tightening**: a finding citing *only* sibling consistency additionally requires a quantified majority (e.g., "7 of 9 siblings use pattern X") — sibling consistency is otherwise the loophole that re-admits taste.
4. **Cluster into proposals** — group related findings into coherent units (one proposal = "rename these 4 for sibling consistency", not four proposals). Each proposal carries: the prediction failure it fixes, the concrete move/rename list, blast radius (references that break — imports, doc cross-links — **plus in-flight exposure**: open branches / active fab changes touching the affected files), and an SRAD-graded confidence. Folder merges/renames in package-scoped languages (Go package dirs: import paths change, identifier collisions possible) carry a mandatory elevated blast grade — such moves are never content-neutral.
5. **Present the ranked report** — highest confidence × lowest blast radius first — and stop. Content-duplication smells appear in a separate "for `/fab-dedupe`" section, not as proposals. Each proposal's suggested-next-action line is informational only. A clean tree yields a plain "no proposals — structure predicts well" close; that is a success, not a failure.

**Error contract** (enumerated in the skill's Error Handling table): path argument doesn't exist or lies outside the repo → error with the resolved attempt; `source_paths` missing/empty and no argument given → error naming the config key; shallow clone or insufficient git history → skip the co-change signal with an explicit note in the report (tree-shape and naming signals still run); path inside `docs/memory/`/`docs/specs/` → refusal with the sibling-skill pointer (step 1).

**Key properties table** (for the skill file): read-only — modifies no files, creates no changes, advances no stage; idempotent (same tree + same history ⇒ same findings); no `Next:` pipeline line — ends with the report (documented opt-out per `_preamble.md` § Next Steps Convention, like `/fab-discuss`).

**Frontmatter**: `helpers: [_srad]` — SRAD grades proposal confidence. No `_intake`/`_generation` (the skill drafts nothing).

### Integration points (New Skill Checklist, docs/specs/skills.md § New Skill Checklist)

- **Frontmatter `name` + `description`** (checklist item 1) — the one-liner must name the non-obvious behaviors: read-only findings report on source-tree structure (placement/naming/consolidation), suggestions only, applies nothing, docs trees excluded
- `docs/specs/skills.md` — new `## /code-reorg [<path>]` section with Flow skeleton, plus a `helpers:` row in § Skill Helpers for its `[_srad]` declaration
- `src/go/fab/cmd/fab/fabhelp.go` — add `code-reorg` to `skillToGroupMap` under the literal group `"Maintenance"` (the group holding `docs-reorg-*`), with test updates per the CLI ⇒ docs + tests rule; whether it also joins the hardcoded TYPICAL FLOW "Maintain docs:" line is decided at apply (it is a source-tree skill, so likely not)
- `docs/specs/glossary.md` — add the skill's row (the glossary does enumerate skills — verified: rows exist for `/fab-dedupe`, `/docs-reorg-memory`, `/docs-reorg-specs`)
- `README.md` — add the `/code-reorg` row to the hand-maintained command tables (they enumerate every user-invocable skill), and check the edit against the shll Toolkit Standards per the constitution's Toolkit Standards article
- Standard skill scaffolding: preamble-read line, Error Handling + Key Properties tables, documented `Next:`-line opt-out

## Affected Memory

- `pipeline/code-reorg.md`: (new) the skill's behavior — five-step flow, signal set + noise controls, taste guard, proposal schema, report-only contract, dedupe/docs-reorg boundaries
- `pipeline/dedupe.md`: (modify) note the code-reorg boundary (structure vs content duplication; code-reorg emits pointers to /fab-dedupe)
- `memory-docs/templates.md`: (modify) only if its reorg-family scope claims are made stale by a source-tree sibling existing — verify during apply; index files (`memory-docs/index.md` etc.) are regenerated via `fab memory-index` after frontmatter edits, never hand-edited

## Impact

- `src/kit/skills/code-reorg.md` — new file (the bulk of the change)
- `docs/specs/skills.md` — new section + § Skill Helpers row
- `docs/specs/glossary.md` — one row
- `README.md` — command-table row (+ shll standards check)
- `src/go/fab/cmd/fab/fabhelp.go` + its test — one map entry
- No `_intake.md`/`_generation.md` changes (the skill is not a Create-Intake consumer); no CLI command surface changes beyond the help grouping; no migrations; no template changes
- Change scale: one substantial new markdown skill + small doc/Go touches

## Open Questions

- Should signal gathering be configurable (a `reorg.detectors`-style config knob, like `consolidate.detectors`), or are built-in shell commands enough for v1? (Assumed: built-ins for v1 — see Assumptions #9.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skill source lives at `src/kit/skills/code-reorg.md`; `.claude/skills/` never edited directly | Constitution V + code-quality anti-pattern list determine this | S:90 R:90 A:100 D:100 |
| 2 | Confident | Skill named `code-reorg` | Discussed — user asked for a name, recommendation given (family fit with docs-reorg-*), user proceeded on it | S:80 R:70 A:75 D:70 |
| 3 | Certain | Report-only: the skill applies nothing, drafts nothing, routes nothing — findings + suggestions are the entire output | User decision, stated explicitly after the review cycle ("this skill is to find problems and present suggestions") | S:95 R:85 A:95 D:95 |
| 4 | Confident | Each proposal carries an informational suggested-next-action line (micro hint or ready-to-paste /fab-new text), never executed | Follows from #3; keeps findings actionable without owning routing | S:70 R:85 A:75 D:70 |
| 5 | Confident | Path argument defaults to config `source_paths`, swept as one combined scope per run; whole-repo runs out of scope by default | Discussed — scoped runs keep output actionable; combined scope matches how source_paths is used elsewhere | S:70 R:80 A:75 D:70 |
| 6 | Confident | Taste guard: uncited findings dropped; frame-(c)-only findings need a quantified sibling majority; "do nothing" is a valid outcome | Discussed + review panel tightening (design lens) | S:80 R:80 A:80 D:80 |
| 7 | Confident | Content-duplication smells route to /fab-dedupe as pointers; docs/memory + docs/specs paths refused with docs-reorg-* pointers | Discussed + review panel (design + adversarial lenses agreed the family boundary must be drawn) | S:80 R:75 A:80 D:75 |
| 8 | Confident | Co-change signal ships with noise controls: 12-month window, >20-file bulk-commit exclusion, `-M` rename following, constitution-derived mandated-coupling carve-out, cross-layer whitelist; skill file ships a worked command | Review panel must-fix (adversarial + design, independently) — evidence: mandated doc-mirror pairs hit ~81% co-change while colocation is prohibited | S:75 R:75 A:70 D:70 |
| 9 | Tentative | v1 uses built-in shell signal gathering; no config detector knob | Simpler v1; `consolidate.detectors` precedent makes a later knob easy to add | S:50 R:70 A:55 D:45 |
| 10 | Tentative | `helpers: [_srad]` only | Report-only design needs no other partial; exact list settled at apply | S:50 R:80 A:65 D:55 |
| 11 | Tentative | Default thresholds (bulk-commit cap 20 files, 12-month window, co-change fraction, junk-drawer criteria) are named defaults written into the skill file, final values settled at apply | Review panel flagged unpinned values; pinning exact numbers needs experimentation against the live tree | S:45 R:75 A:55 D:50 |
| 12 | Confident | fabhelp group is the literal `"Maintenance"`; glossary and README rows are definite additions | Verified by review panel against fabhelp.go:44, glossary.md, README.md command tables | S:80 R:85 A:85 D:80 |

12 assumptions (2 certain, 7 confident, 3 tentative, 0 unresolved).
