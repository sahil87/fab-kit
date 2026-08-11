# Intake: Delete SPEC Mirror Tree and Retire the Mirror Rule

**Change**: 260811-rehi-delete-spec-mirror-tree
**Created**: 2026-08-11

## Origin

Promptless dispatch (`/fab-proceed` create-new, `{questioning-mode} = promptless-defer`) from a synthesized live-conversation description. This is the agreed follow-up to `260811-xy7a-condense-spec-skill-mirrors` (merged as PR #581, squash commit `cebbc156`).

> Delete the SPEC mirror tree and retire the mirror rule — fold the condensed Flow skeletons into `docs/specs/skills.md`, delete `docs/specs/skills/` entirely, remove the SPEC-mirror clause from the constitution (1.5.0 → 1.6.0), strip the enforcement/sweep machinery (code-review rule, code-quality anti-pattern + sweep class, skills.md checklist item 6, docs-reorg-specs reserved-path carve-outs), sweep live references repo-wide, and regenerate memory indexes. Historical records (`docs/specs/findings/`, dated `log.md` files) stay untouched.

All design decisions below were made in the live conversation (user-decided or user-agreed); this intake transfers them across the pipeline boundary verbatim.

## Why

1. **The pain point — the mirror tree's main output is work about itself.** Even after xy7a condensed the 36 mirrors to structural quick-reference (558 lines today), the only three Copilot review comments on PR #581 were mirror meta-consistency issues. Earlier, the June 2026 skills-review audit (`docs/specs/findings/skills-review-2026-06-12.md`) found a dozen confirmed drift bugs nobody had noticed — evidence the mirrors are not read operationally.
2. **The consequence of keeping them — the residual cost is the machinery, not the lines.** The mirrors carry: a constitution clause (a normative MUST rule), a code-review must-fix rule, a code-quality anti-pattern plus a Sibling & Mirror Sweeps class, a skills.md New Skill Checklist item, reserved-path carve-outs in `docs-reorg-specs`, and a standing header-drift exposure. Every structural skill change pays this tax; review rework in this repo has repeatedly traced to it.
3. **Why deletion is safe now.** xy7a's migrate-don't-drop audit already verified that load-bearing mirror content is covered by `docs/memory/` as present truth. What remains in the mirrors is the structural skeleton — and its human-curated home is `docs/specs/skills.md` (Constitution VI), where each skill already has a section.

**Alternatives rejected** (user-decided in conversation):

- **Keeping the condensed mirrors** — rejected on the evidence above (self-referential review load, unread drift).
- **Moving skeletons to `docs/memory/`** — rejected: specs are the human-curated design home; `skills.md` is the existing aggregate.
- **Dropping the skeletons entirely** — the user's default is fold-into-skills.md (graded assumption if reason to doubt emerges; none found for the user-invocable skills).

## What Changes

### 1. Fold the flow skeletons into `docs/specs/skills.md`

Each skill's **existing per-skill section** in `docs/specs/skills.md` gains its condensed Flow skeleton from the current post-xy7a mirror (title + short header + Flow diagram + Tools + Sub-agents shape). Include the Tools/Sub-agents one-liners **only where they add real information beyond the section's prose** — most sections already state tool usage; do not duplicate.

- Skeletons are the **current condensed versions** in `docs/specs/skills/SPEC-*.md`, not the old bloated flows.
- Coverage check performed at intake: all 27 user-invocable skills have `## /<skill>` sections in `skills.md` (verified against the 36 mirrors; the other 9 are partials — see below).
- **Partials** (`SPEC-_preamble.md`, `SPEC-_generation.md`, `SPEC-_srad.md`, `SPEC-_intake.md`, `SPEC-_review.md`, `SPEC-_pipeline.md`, `SPEC-_cli-fab.md`, `SPEC-_cli-external.md`, `SPEC-_cli-agents.md`) have **no per-skill sections** in `skills.md`; they are covered by `## Skill Helpers`. Fold a partial's Flow skeleton into that section only where it carries a real flow that adds information (e.g., `SPEC-_srad.md` has a 4-line flow; `SPEC-_cli-fab.md` has none — it is a pure reference and its skeleton is dropped without replacement). <!-- This is the one fold decision the conversation did not explicitly cover — graded Confident in ## Assumptions (row 7). -->
- In-file sweep: `skills.md:778` (the `/fab-operator` section's "Full spec" line) links to `skills/SPEC-fab-operator.md` — remove/reword that link as part of the fold.

### 2. Delete `docs/specs/skills/` entirely

- All 36 `SPEC-*.md` files (558 lines as of intake; no `index.md` exists in the directory).
- Remove the `skills/` row from `docs/specs/index.md` (currently the last table row: "Per-skill structural quick-reference mirrors — flow, tools, sub-agents").

### 3. Remove the SPEC-mirror rule from the constitution (`fab/project/constitution.md`)

- **Delete outright** the Additional Constraints clause: "Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file only when the change alters the skill's flow, tool usage, or sub-agent structure; prose-only skill edits do not trigger a mirror update".
- Add a **dated HTML-comment amendment annotation** per the file's existing convention (see the five existing `<!-- YYYY-MM-DD (change-id): ... -->` blocks), noting the clause removal, the mirror-tree deletion, and where the skeletons went.
- **Version bump 1.5.0 → 1.6.0** (removing a normative MUST rule — user-specified as a minor bump, consistent with the 1.3.0 → 1.4.0 precedent for adding one); **Last Amended: 2026-08-11**.

### 4. Strip the enforcement/sweep machinery

| File | Edit |
|------|------|
| `fab/project/code-review.md` | Delete the **SPEC-mirror sync** must-fix rule (first bullet under § Project-Specific Review Rules) |
| `fab/project/code-quality.md` | Delete the anti-pattern "**Shipping a structural skill change without its SPEC mirror**"; delete the mirror entry (first bullet) in § Sibling & Mirror Sweeps **and** the section's trailing blockquote ("On a CLI/command-signature change ... treat all affected skills' SPEC mirrors as the sweep class") — the other sweep classes (twin skills, aggregate specs, memory files) stay |
| `docs/specs/skills.md` | New Skill Checklist item 6: replace the "create `docs/specs/skills/SPEC-{name}.md`" instruction with "add the skill's Flow skeleton to its skills.md section" |
| `src/kit/skills/docs-reorg-specs.md` | Remove the reserved-path carve-outs for SPEC mirrors (line 29's "Reserved paths" note item and line 88's "never migrate reserved paths (`docs/specs/skills/SPEC-*.md`)" constraint) — the reserved-path concept for mirrors dies with the tree. This is a **canonical-source edit** (`.claude/skills/` deployed copies untouched; `fab sync` redeploys) |

### 5. Sweep live references repo-wide

Grep pattern: `docs/specs/skills/` and `SPEC-*.md`-style names. Repo-wide grep was **run at intake**; the complete live-reference inventory:

**Known sites (named in conversation):**

- `docs/memory/memory-docs/specs-index.md` — § Per-Skill SPEC Mirrors (lines ~41–45: tree description, naming convention) and the Design Decision "docs-reorg-specs exempts `docs/specs/skills/SPEC-*.md` from reorganization" (line ~57). Rewrite to present truth: the mirror tree no longer exists; skeletons live in `skills.md`.
- `docs/memory/pipeline/dedupe.md` — line 19's mirror sentence ("Its mirror is `docs/specs/skills/SPEC-fab-dedupe.md`, updated when ...").
- `docs/memory/pipeline/execution-skills.md` — line 67's trailing clause ("... mirrored into `docs/specs/skills/SPEC-git-pr.md`'s Flow tree").
- `fab/plans/sahil/skill-prose-consolidation.md` — execution constraint 1 (line ~38, the mirror-update-in-same-change constraint) and the line ~398 sweep-checklist item (`ls docs/specs/skills/SPEC-*.md` against every touched skill).

**Additional live sites found by the intake grep:**

- `docs/specs/fkf.md` — lines 204 ("The `docs/specs/skills/SPEC-*.md` mirrors stay constitution-pinned ...") and 229 ("... with the corresponding `SPEC-*.md` mirror updates per the constitution") — live present-truth claims about the now-dead rule.
- `docs/memory/_shared/context-loading.md` — line 93's parenthetical "`SPEC-_preamble.md` (partial SPECs keep the leading underscore)".
- `docs/memory/runtime/operator.md` — line 460's Design Decision sentence "... the three SPEC mirrors (`SPEC-_cli-external.md`, `SPEC-_preamble.md`, `SPEC-fab-operator.md`) were aligned in the same change" — trim the dead mirror clause (Confident assumption, row 9).

**Historical — left untouched (user carve-out):**

- `docs/specs/findings/*` (3 files), all dated `log.md` / `log.seed.md` files (freeze-on-write projections), `fab/backlog.md` completed `[x]` entries, `fab/changes/*` change artifacts, and `docs/specs/srad-scoring-rationale-v1-to-v2.md` (a design-rationale/history companion narrating what the v1→v2 change touched — same class as findings; Confident assumption, row 8).

**After memory edits:** regenerate memory indexes with `fab memory-index` (output taken wholesale).

### 6. Go code / tests check

**Verified at intake — none.** `grep -rn "docs/specs/skills/|SPEC-\*" --include="*.go" src/` and the same over `scripts/` return nothing; `docs/site/fkf.md` and `src/kit/reference/fkf.md` are also clean. The CLI⇒docs+tests rule is not triggered. Apply should re-verify with a final grep before finishing.

## Affected Memory

- `memory-docs/specs-index`: (modify) remove § Per-Skill SPEC Mirrors and the reserved-path-exemption Design Decision; record the retirement decision; describe the folded skeletons' home in `skills.md`
- `pipeline/dedupe`: (modify) drop the SPEC-fab-dedupe mirror sentence
- `pipeline/execution-skills`: (modify) drop the SPEC-git-pr Flow-tree mirror clause
- `_shared/context-loading`: (modify) drop the SPEC-_preamble parenthetical
- `runtime/operator`: (modify) trim the dead three-SPEC-mirrors clause in the `_cli-external` Design Decision entry

## Impact

- **Scope**: markdown-only; zero Go, zero scripts, zero tests. ~36 file deletions (558 lines) + ~14 file edits.
- **Files edited**: `docs/specs/skills.md` (largest edit — 27 skeleton folds + checklist + one link), `docs/specs/index.md`, `docs/specs/fkf.md`, `fab/project/constitution.md`, `fab/project/code-review.md`, `fab/project/code-quality.md`, `src/kit/skills/docs-reorg-specs.md`, 5 memory files (+ regenerated indexes/log via `fab memory-index`), `fab/plans/sahil/skill-prose-consolidation.md`.
- **Governance**: constitution version 1.5.0 → 1.6.0 (normative MUST rule removed); the code-review must-fix rule set shrinks by one; the code-quality sweep-class list shrinks by one.
- **Constraints honored**: `.claude/skills/` (deployed copies) untouched; `src/kit/skills/docs-reorg-specs.md` edit is a canonical-source edit; markdown-only artifacts (Constitution IV).
- **Risk**: low — all content deleted is either duplicated (memory coverage verified by xy7a's audit) or being relocated into `skills.md`; everything is git-recoverable.

## Open Questions

None — the synthesized conversation description resolves every decision point; the two judgment calls the conversation did not explicitly cover are graded (not deferred) in `## Assumptions` rows 7 and 9. No questions were deferred under the promptless-dispatch contract.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Delete `docs/specs/skills/` entirely (36 `SPEC-*.md`, 558 lines) and remove the `skills/` row from `docs/specs/index.md` | Discussed — user decided, with evidence (PR #581's only review comments were mirror meta-consistency; June 2026 audit found unread drift); xy7a's migrate-don't-drop audit verified memory coverage | S:95 R:70 A:90 D:95 |
| 2 | Certain | Fold each skill's condensed Flow skeleton into its existing `docs/specs/skills.md` section; Tools/Sub-agents one-liners only where they add information beyond the section's prose | Discussed — user chose fold-into-skills.md over moving to `docs/memory/` or dropping; `skills.md` is the human-curated aggregate (Constitution VI); all 27 user-invocable skills verified to have sections | S:90 R:80 A:85 D:85 |
| 3 | Certain | Constitution: delete the SPEC-mirror clause outright; dated HTML-comment amendment annotation; Last Amended 2026-08-11; version 1.5.0 → 1.6.0 | Discussed — user specified exactly; matches the file's amendment convention and the 1.3.0→1.4.0 minor-bump precedent for a normative-rule change | S:95 R:75 A:90 D:90 |
| 4 | Certain | Strip all four enforcement surfaces: code-review.md mirror rule, code-quality.md anti-pattern + mirror sweep-class entry + trailing mirror blockquote (other sweep classes stay), skills.md checklist item 6 replacement text, docs-reorg-specs.md carve-outs (canonical source) | Discussed — user enumerated each surface and the replacement text for item 6 | S:90 R:80 A:90 D:90 |
| 5 | Certain | Historical records untouched: `docs/specs/findings/`, dated `log.md`/`log.seed.md`, backlog `[x]` entries, `fab/changes/*` artifacts | Discussed — user carve-out; logs are freeze-on-write projections | S:90 R:90 A:95 D:90 |
| 6 | Certain | Sweep also covers three live sites beyond the conversation's known four: `docs/specs/fkf.md:204,229`, `docs/memory/_shared/context-loading.md:93`, `docs/specs/skills.md:778` (SPEC-fab-operator link) | Intake grep found them; they are live present-truth claims squarely under the user's "update live references" rule | S:70 R:85 A:85 D:80 |
| 7 | Confident | Partial mirrors (9 `SPEC-_*.md`) have no `skills.md` sections: fold a partial's flow skeleton into § Skill Helpers only where it adds real information; pure-reference partials (`_cli-fab`, `_cli-external`) lose their skeletons without replacement | Fold instruction addresses per-skill sections only; the user's "only where they add real information" principle extends naturally; § Skill Helpers already documents every partial | S:40 R:85 A:65 D:55 |
| 8 | Confident | `docs/specs/srad-scoring-rationale-v1-to-v2.md` mirror mentions (lines 267–268, 320) stay untouched as historical narration | The doc is a design-rationale/history companion describing what the v1→v2 change touched — same class as findings/ | S:55 R:90 A:75 D:70 |
| 9 | Confident | `docs/memory/runtime/operator.md:460`: trim the dead "three SPEC mirrors were aligned" clause during sweep rather than leaving it | DD narration straddles live/historical, but FKF present-truth removes dead-path claims from memory bodies; trivially reversible | S:35 R:90 A:55 D:45 |
| 10 | Certain | No Go/scripts/tests reference the mirror tree — CLI⇒docs+tests rule not triggered; apply re-verifies with a final grep | Verified empirically at intake (grep over `src/` `*.go` and `scripts/` returned nothing) | S:85 R:90 A:95 D:90 |
| 11 | Certain | Regenerate memory indexes via `fab memory-index` after memory edits, output taken wholesale; `.claude/skills/` untouched; markdown-only | Discussed — user-stated constraints; standing project conventions (Constitution IV/V) | S:90 R:90 A:95 D:90 |

11 assumptions (8 certain, 3 confident, 0 tentative, 0 unresolved).
