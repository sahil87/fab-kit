# Intake: Condense SPEC Skill Mirrors

**Change**: 260811-xy7a-condense-spec-skill-mirrors
**Created**: 2026-08-11

## Origin

Promptless dispatch via `/fab-proceed` (Create-Intake Procedure, `{questioning-mode} = promptless-defer`). The description below was synthesized from a live conversation in which the user made the key decisions explicitly; no questions were asked at intake — would-be questions are recorded as graded rows in `## Assumptions`.

> Condense the `docs/specs/skills/` SPEC mirrors to structural quick-reference and narrow the constitution's mirror rule. Strip every `SPEC-*.md` to title + at most a 2–3 sentence header + Flow diagram + Tools table + Sub-agents section (aggressive target: total tree at 10–20% of current size). Migrate load-bearing Summary content into `docs/memory/` as present-truth; deletion is the rule, migration the exception. Amend the constitution's mirror rule to trigger only on flow / tool-usage / sub-agent structure changes. Update the enforcement surfaces (`fab/project/code-review.md`, `fab/project/code-quality.md`) in the same change.

## Why

1. **The pain point — shadow memory.** The 36 `docs/specs/skills/SPEC-*.md` files (3,721 lines total, verified by `wc -l`) have drifted into a second, unofficial memory tree: their `## Summary` sections are accretions of dated change-keyed delta prose (e.g. `SPEC-fab-continue.md` is 214 lines dominated by "As of 260704-pag2…", "260718-wrct adds…" narration — the exact transition-narration style the FKF present-truth effort banned from `docs/memory/`). Skill behavior is already documented as current truth in `docs/memory/` (e.g. `pipeline/execution-skills.md`, `pipeline/planning-skills.md`), and the skill source itself is normative (Constitution I, Pure Prompt Play) — so the Summary prose is a third copy that must be manually kept in sync.

2. **The constitutional contradiction.** Constitution VI says specs are pre-implementation, human-curated design intent that "MUST NOT be auto-generated or overwritten by tooling" — yet the Additional Constraints mirror rule ("Changes to skill files … MUST update the corresponding `docs/specs/skills/SPEC-*.md` file") forces mechanical post-implementation updates on every skill edit, however trivial. The mirrors behave like generated artifacts while being governed as human-curated ones.

3. **The consequence of not fixing it.** The mirror-sweep obligation is the project's #1 documented rework cause (`fab/project/code-quality.md` § Sibling & Mirror Sweeps: "the single most common rework cause in this project"). Every skill-prose change pays a mirror tax, and the mirrors keep accreting narration that nobody reads as current truth.

4. **Why this approach over alternatives.**
   - *Full deletion of `docs/specs/skills/`* — rejected: the Flow / Tools / Sub-agents structural map is the one thing the aggregated memory files don't do well.
   - *Milder condensation (~30% of current size)* — rejected: user explicitly chose the more aggressive 10–20% target.

## What Changes

### 1. Strip all 36 `docs/specs/skills/SPEC-*.md` files to structural quick-reference

Each file is reduced to exactly:

- Title (`# {skill-name}`)
- At most a **2–3 sentence header** (what the skill does — no dated deltas, no change-id narration)
- `## Flow` diagram (the ASCII flow tree)
- Tools table (`### Tools used` or equivalent)
- `### Sub-agents` section

Everything else is deleted: the accreted `## Summary` dated-delta prose, `## Contents` TOCs, "Source organization" lines, and the `## Hooks` / bookkeeping-candidates sections some files carry (10 files carry Hooks/Bookkeeping sections, e.g. `SPEC-_preamble.md`, `SPEC-fab-ff.md` — grep `"## Hooks\|Bookkeeping"` to enumerate).

**Size target (aggressive, user-chosen):** total tree lands at **10–20% of current size** — ~370–740 lines from 3,721. Current outliers: `SPEC-_preamble.md` 410 lines, `SPEC-fab-continue.md` 214, `SPEC-fab-operator.md` 207, `SPEC-git-pr-review.md` 204. **Exemplar of the good shape:** `SPEC-fab-discuss.md` (36 lines: title, 2-sentence summary, Flow, Tools table, Sub-agents) — though even its "Source organization" line and `## Summary` heading ceremony can go if the header shape above supersedes them.

### 2. Migrate load-bearing Summary content — don't silently drop

Before deleting a `## Summary` section, check whether it carries content that is (a) load-bearing (a behavior/contract claim someone would need) and (b) not already in `docs/memory/`. Such content is merged into the relevant memory file as present-truth — FKF style, topic-keyed, no change-id narration (rationale goes into four-field `## Design Decisions` entries where applicable). Expectation from the audit: most content is already covered by memory (e.g. `pipeline/execution-skills.md`, `pipeline/planning-skills.md`, `runtime/operator.md`, `memory-docs/` domain); **migration is the exception, deletion the rule.** Regenerate memory indexes via `fab memory-index` after any memory write.

### 3. Amend the constitution's mirror rule (narrowing a normative MUST rule)

Current rule (`fab/project/constitution.md`, Additional Constraints):

> Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file

New rule: a mirror update is required **only when a change alters a skill's flow, tool usage, or sub-agent structure** — prose-only skill edits no longer trigger mirror updates.

Follow the constitution's amendment conventions (visible in its existing 4 amendment annotations): append a dated HTML-comment annotation (`<!-- 2026-08-11 (260811-xy7a): … -->`) explaining the amendment, update **Last Amended** to 2026-08-11, and bump the version (currently 1.4.0; this narrows/rewords a normative MUST rule — see Assumptions row 5 for the bump level).

### 4. Update the enforcement surfaces in the same change (explicit user requirement)

So the reviewer doesn't flag the stripped mirrors under the old rule:

- **`fab/project/code-review.md`** — the "SPEC-mirror sync" must-fix rule (Project-Specific Review Rules, first bullet) is rewritten to the narrowed trigger (flow / tool-usage / sub-agent structure changes only).
- **`fab/project/code-quality.md`** — § Sibling & Mirror Sweeps (the `src/kit/skills/*.md` ↔ `SPEC-*.md` class entry and the CLI-change blockquote "treat **all** of a skill's SPEC mirrors as the sweep class") and the anti-pattern entry "Shipping a skill change without its SPEC mirror" are rewritten to the narrowed rule.

### 5. Sweep class (same-change consistency updates)

Grounded by repo-wide grep; `docs/specs/glossary.md` and `docs/specs/architecture.md` verified clean (no mirror-rule restatements):

- `docs/specs/skills.md:126` — New Skill Checklist item 6 ("SPEC mirror file … The constitution requires updating a skill's SPEC on every skill edit") → narrowed rule + condensed shape.
- `docs/specs/index.md:36` — the `skills/` row description ("Per-skill flow diagrams — summary, tool usage, sub-agents, hooks, and bookkeeping candidates") → match the new shape (e.g. "flow, tools, sub-agents"). No reclassification of the row as generated-adjacent (out of scope — Assumptions row 6).
- `src/kit/skills/docs-reorg-specs.md:29,88` — the reserved-paths note restates "the constitution requires every skill edit to update its mirror" → narrowed wording. (Reserved-path protection itself is unchanged — mirrors keep their mechanical `SPEC-{source-filename}.md` names.)
- Memory files restating the mirror rule — see Affected Memory.

## Affected Memory

- `memory-docs/specs-index.md`: (modify) The primary home of mirror-rule and mirror-shape facts — "§ Per-Skill SPEC Mirrors" states the unconditional constitution rule and the "Summary + Flow + supporting tables" shape; the reserved-paths Design Decision restates the rule. Rewrite to the narrowed rule and the condensed shape (title + header + Flow + Tools + Sub-agents).
- `pipeline/dedupe.md`: (modify) Line 19 states "The constitution-required mirror is `docs/specs/skills/SPEC-fab-dedupe.md`" — reword to the narrowed rule.
- `pipeline/execution-skills.md`: (modify) Conditional migration target for any load-bearing execution-skill Summary content not already present; also mentions the `SPEC-git-pr.md` Flow mirror (that claim stays true — Flow survives the strip).
- `pipeline/planning-skills.md`: (modify) Conditional migration target for any load-bearing planning-skill Summary content not already present.

## Impact

- **Files stripped:** all 36 `docs/specs/skills/SPEC-*.md` (~3,000+ lines deleted; tree from 3,721 → ~370–740 lines).
- **Governance:** `fab/project/constitution.md` (rule narrowed, amendment annotation, version bump, Last Amended).
- **Enforcement surfaces:** `fab/project/code-review.md`, `fab/project/code-quality.md`.
- **Sweep:** `docs/specs/skills.md`, `docs/specs/index.md`, `src/kit/skills/docs-reorg-specs.md`.
- **Memory:** the four files listed in Affected Memory (+ `fab memory-index` regen of `docs/memory/**/index.md` / `log.md` if memory changes).
- **No Go code.** This is a docs/governance change; if no `.go` files change, no test updates are required (the code-review "Go changes ship tests" rule is untriggered). All artifacts remain CommonMark markdown (Constitution IV).
- **Risk shape:** large mechanical deletion — the main risk is dropping a load-bearing claim that exists nowhere else; mitigated by the migrate-don't-drop pass (What Changes § 2).

## Open Questions

None asked — promptless dispatch. Would-be questions were graded per SRAD; none landed Unresolved (see Assumptions).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Strip every `SPEC-*.md` to title + ≤2–3-sentence header + Flow + Tools table + Sub-agents; total tree at 10–20% of 3,721 lines (~370–740) | Discussed — user explicitly chose the aggressive target over ~30%; `SPEC-fab-discuss.md` named as exemplar | S:95 R:70 A:90 D:95 |
| 2 | Certain | Migrate-don't-drop: load-bearing Summary content not already in memory merges into memory as FKF present-truth; deletion is the default | Discussed — explicit user decision, including "migration the exception, deletion the rule" | S:90 R:80 A:85 D:85 |
| 3 | Certain | Constitution rule narrowed to "only when a change alters a skill's flow, tool usage, or sub-agent structure"; amendment follows the dated HTML-comment + Last-Amended + version-bump convention | Discussed — explicit user decision; convention visible in the constitution's 4 existing amendment annotations | S:95 R:75 A:90 D:90 |
| 4 | Certain | `code-review.md` + `code-quality.md` enforcement surfaces updated in the same change | Discussed — explicit user requirement (same-change coupling) | S:95 R:80 A:90 D:95 |
| 5 | Confident | Constitution version bump is **minor** (1.4.0 → 1.5.0) | Amendment precedent: new normative MUST rule → minor bump; cosmetic wording → none. Narrowing a normative MUST rule is a material semantic change, above cosmetic | S:70 R:85 A:70 D:60 |
| 6 | Confident | Reclassifying the `docs/specs/skills/` row in `docs/specs/index.md` as generated-adjacent reference is **out of scope**; only the row's shape description is updated | Floated in discussion but NOT explicitly agreed; user guidance was "out of scope or deferred"; trivially reversible as a follow-up | S:60 R:85 A:70 D:65 |
| 7 | Confident | Sections outside the agreed shape (`## Hooks`, bookkeeping-candidates, `## Contents` TOCs, "Source organization" lines) are deleted along with `## Summary` prose | The agreed shape enumerates exactly five elements; hooks are dead surface (fab registers/owns none per `docs/specs/hooks.md`); 10 files carry such sections | S:75 R:80 A:80 D:70 |
| 8 | Certain | Sweep class is exactly: `docs/specs/skills.md:126`, `docs/specs/index.md:36`, `src/kit/skills/docs-reorg-specs.md:29,88`, `memory-docs/specs-index.md`, `pipeline/dedupe.md:19` — glossary.md and architecture.md verified clean | Grounded by repo-wide grep for SPEC-mirror and specs/skills mentions at intake; historical Design-Decision entries (e.g. `runtime/operator.md`) record past changes and are not rewritten | S:85 R:75 A:90 D:85 |
| 9 | Confident | Editing `src/kit/skills/docs-reorg-specs.md`'s reserved-paths wording is a prose-only skill edit — under the amended rule it needs no mirror update beyond the strip this change already performs on `SPEC-docs-reorg-specs.md` | The amended rule applies to this change's own edits; the strip touches every mirror anyway, so the question is moot in practice | S:55 R:85 A:60 D:50 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).
