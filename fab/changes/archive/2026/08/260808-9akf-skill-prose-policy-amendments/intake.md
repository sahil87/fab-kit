# Intake: Skill Prose Policy Amendments (Phase 4)

**Change**: 260808-9akf-skill-prose-policy-amendments
**Created**: 2026-08-08

## Origin

One-shot `/fab-new` invocation executing **Phase 4 (policy + extraction)** of `fab/plans/sahil/26-08-08-skill-prose-consolidation.md`. Phases 1–3 are already merged to main (`260808-ip4q` drift fixes, `260808-mcxv` mechanical deletion, `260808-s2sz` structure restructures). The invocation specified:

> Phase 4 (policy amendments plus optional _git.md partial extraction): **4a** amend `internal-skill-optimize.md` — narrow the partial exemption (today's "never touches a `_*.md` partial" rule) to "partials are not trimmed as a side effect of optimizing a consumer skill; a dedicated partial-optimization pass is legitimate"; add a transition-narration bloat signal ("no longer" / "used to" / "supersedes" / change-id archaeology in an instruction file should be rewritten as present truth, keeping the don't-re-break force and dropping the history, per FKF §3.3 extended from docs/memory to skills); scope Rule 6 ("one output example max") to illustrative examples only, exempting output literals the skill's own logic greps or matches (git-pr's five checkmark lines, git-pr-review's three reply prefixes); optionally add a cross-file sibling-duplication signal. **4b** add the ownership convention to `_preamble.md` — "a skill file may state a rule it owns, or point at the file that owns it, never both" — the recurrence guard for the whole effort. **4c** decide whether to extract a `_git.md` partial from `git-pr.md` and `git-pr-review.md` (change resolution, branch-guard STOP message templated on skill name, FKF §5 never-hand-merge rule, status-commit procedure; parameterize the divergent push-failure policies — git-pr fails fast, git-pr-review deliberately soft-fails, do not flatten). Make the 4c call yourself using the plan's own guidance and record the decision plus rationale in the intake. Verify all line references against current HEAD. Update matching `docs/specs/skills/SPEC-*.md` mirrors and the constitution reference to shll standards if touched.

The 4c decision was delegated to the agent and is recorded below (§ What Changes › 4c) and in the Assumptions table.

**Mid-intake amendment (2026-08-08, user-directed)**: after intake creation the user redirected 4b's placement — the ownership convention is *kit-authoring* guidance whose only enforcement surface is this repo's `src/kit/skills/` corpus, so it does NOT belong in `_preamble.md` (deployed by `fab sync` into every installing repo and loaded on every skill execution). It belongs in `fab/project/code-quality.md` — this repo's project policy, loaded by every apply/review agent working here. `src/kit/skills/_preamble.md` is NOT touched by this change.

## Why

1. **The pain point**: Phases 1–3 removed ~13,000 words of drift-prone prose from `src/kit/skills/`, but the *policies that let the bloat accumulate* are unchanged. `internal-skill-optimize.md` still (a) exempts ALL `_*.md` partials from content optimization — even though ~45% of `_preamble.md` and ~25% of `_cli-fab.md` proved removable, (b) carries no signal for transition-narration bloat (the single largest deletion class in Phase 2: ~2,700 words of "no longer"/"used to"/change-id archaeology), and (c) states Rule 6 ("one output example max") unscoped, which would wrongly target output literals that other logic greps (git-pr's `✓` lines are grep targets in the protected set).
2. **The consequence of not fixing**: the next optimization pass either skips the partials entirely (re-accumulating the exact bloat Phase 3 just removed) or, worse, applies Rule 6 to protected output literals and breaks a grep contract. And without the ownership convention in `_preamble.md`, nothing stops the state-a-rule-AND-point-at-its-owner duplication pattern from re-growing — the drift mechanism the whole effort exists to kill (PR #539's reap step silently missing from three downstream restatements was the live demonstration).
3. **Why this approach**: policy amendments are the smallest diff with the largest leverage (the plan's own framing). They encode the effort's lessons in the files the relevant agents actually load — the optimizer skill (enforcement) and this repo's `fab/project/code-quality.md` (authoring policy, in the always-load layer for every apply/review agent here) — rather than in a plan document nobody re-reads. The plan originally targeted `_preamble.md` for the convention; the user redirected it to `code-quality.md` because `_preamble.md` ships to every installing repo, where the authoring convention has no enforcement surface (see Origin amendment note).

## What Changes

### 4a. Amend `src/kit/skills/internal-skill-optimize.md` (4 edits)

Verified against current HEAD (file is 116 lines; all Phase-1–3 shifts accounted for):

**4a-1. Narrow the partial exemption.** Today the exemption is absolute, stated at lines 12 ("**never** touches a `_*.md` partial (a partial is shared reference context, never an optimization target)"), 29, 35, and Constraints line 113. Replace the absolute framing with the two-sided rule:

> Partials are not trimmed **as a side effect of optimizing a consumer skill** (a consumer pass treats every `_*.md` file as read-only reference context). A **dedicated partial-optimization pass** — invoked explicitly with the partial's name (e.g., `/internal-skill-optimize _preamble`) — is legitimate and applies the same content signals.

All four statement sites must be updated consistently (the exemption is restated in Arguments, Pre-flight, Analysis, and Constraints — this change makes each site say the same narrowed rule or point at one owning statement, per the 4b convention).

**4a-2. Add a transition-narration bloat signal.** New row in the **Content signals** table (Analysis section, lines 44–54):

> | **Transition narration** | "no longer" / "used to" / "supersedes {id}" / "removed in {version}" / change-id archaeology woven into instruction prose. Rewrite as present truth: keep the don't-re-break-this force (the prohibition and its reason), drop the history. (FKF §3.3's present-truth rule, extended from docs/memory to skills — see `$(fab kit-path)/reference/fkf.md` §3.3.) |

**4a-3. Scope Rule 6 to illustrative examples.** Optimization Rule 6 (line 72: "**One output example max** — show the canonical happy-path format…") gains an exemption clause:

> Applies to **illustrative** examples only. Output literals that the skill's own logic (or a sibling skill's logic) greps or string-matches are contracts, not examples — never consolidate or reword them (e.g., `git-pr`'s five `✓ commit/push/pr/meta/status` lines, `git-pr-review`'s three `Fixed —`/`Deferred —`/`Skipped —` reply prefixes).

**4a-4. Add a cross-file sibling-duplication signal** (the optional item — included; see Assumptions #3). New Content-signals row + a batch-mode note:

> | **Sibling duplication** | The same rule, guard, or procedure stated in a sibling skill (twins like `fab-ff`↔`fab-fff`, or a consumer restating a partial-owned rule). Ownership rule: a file may state a rule it owns, or point at the file that owns it — never both. Keep the owner's statement, reduce the other to a pointer — or report the pair as an extraction candidate. |

(The signal states the ownership rule self-contained — it MUST NOT point at `fab/project/code-quality.md`, a project-local file absent in user repos; the deployed-path-leak lesson from Phase 1 item #8.)

Batch mode (Execution § Batch mode) currently reads files independently sorted by size; add one step: after per-file analysis, compare findings across files and report duplicated-rule clusters (report-only, like depth findings — extraction is a separate content-moving change).

**4a-5. Consequential**: the Contents/TOC of the file does not change (no `##` sections added or removed); Constraint 3 ("DO NOT merge skills or move content between skills (beyond referencing `_preamble.md`)") is **NOT widened** — that widening was contingent on 4c proceeding, which it does not (see 4c).

### 4b. Ownership convention in `fab/project/code-quality.md` (relocated from `_preamble.md` — user-directed)

Add a new bullet to `## Anti-Patterns (project-specific)` in `fab/project/code-quality.md`:

> - **Stating an owned rule AND pointing at its owner.** A skill file may **state a rule it owns**, or **point at the file that owns it** — never both. Restating an owned rule alongside (or instead of) a pointer is the drift mechanism the skill-prose consolidation effort exists to kill: the copy silently diverges when the owner changes (e.g., PR #539's `fab dispatch reap` step updated the `_preamble.md` canon but none of the three downstream restatements). Applies to every `src/kit/skills/*.md` edit.

Also add one cross-reference sentence at the end of `## Sibling & Mirror Sweeps` (the existing drift-prevention section): sweeps catch a diverged copy after the fact; the ownership convention prevents the copy existing at all.

**Placement rationale** (the amendment): `fab/project/code-quality.md` is in this repo's always-load layer, so every apply/review agent editing skill files here sees it, and review derives acceptance items from it — enforcement for free. `_preamble.md` would have shipped the convention to every installing repo, where no one authors kit skills. The constitution was considered and rejected as the heavier register (new normative rule → version bump ceremony) for what is architecture/style guidance. `code-quality.md` is a project file: **no SPEC mirror, no kit deployment, no memory hydration obligation.**

### 4c. `_git.md` partial extraction — DECISION: **skip** (record-only, no extraction)

**Decision**: do NOT extract `_git.md` (and consequently not the lower-ranked `_reorg.md` either). Phase 4 ships as a policy-only change.

**Rationale** (per the plan's own guidance, which explicitly allows "skip the partial-extraction work entirely if the allow-list churn is not worth it for a policy-only phase"):

1. **The wiring cost exceeds the ~60-line saving.** Extraction requires: a new `_git.md` partial, edits to both consumers, the closed `helpers:` allow-list edit in `_preamble.md` § Skill Helper Declaration (line 119 at HEAD), widening `internal-skill-optimize` Constraint 3, plus new/updated SPEC mirrors (`SPEC-_git.md` new, `SPEC-git-pr.md`, `SPEC-git-pr-review.md`, `SPEC-_preamble.md`, `SPEC-internal-skill-optimize.md`) and aggregate-spec sweeps.
2. **The shared blocks are more divergent than they look** (verified at HEAD): the change-resolution steps derive different variable sets (`git-pr` derives 4 variables incl. `{has_intake}`/`{change_type}`; `git-pr-review` captures only `{name}`); the detached-HEAD STOP messages differ ("Cannot ship from…" vs "Cannot process PR reviews from…"); the status-commit procedures differ in commit message, gating condition, AND the deliberately divergent push-failure policy (git-pr.md Step 4c fails fast; git-pr-review.md Step 6.5 deliberately soft-fails — "a completed review cycle must not be aborted by a transient push failure"). A partial parameterized on 4+ knobs (skill name, stage, commit message, push-failure policy) trades ~60 lines of visible duplication for comparable invisible indirection.
3. **The `helpers:` mechanism presupposes `_preamble` loading** ("After reading `_preamble` and before executing the skill body…"), and the git-* skills deliberately do NOT load `_preamble` today — they are lean, fully-autonomous skills. Wiring them into the helper model either pulls the whole always-load layer into every `/git-pr` invocation (scope creep) or invents a bespoke third loading pattern.
4. **The drift risk 4c targets is covered more cheaply** by this same change: the 4b ownership convention plus 4a-4's sibling-duplication signal make the git-pr↔git-pr-review pair a reportable finding on every future optimization pass.

**Consequences recorded** (per the plan's warning): because the git-* skills still do not load `_preamble`, their restatements of preamble-owned facts (branch naming, resolution grammar) remain **load-bearing and MUST NOT be deduped** — this change touches neither file. Extraction remains available as future standalone work if the pair drifts in practice.

### SPEC mirrors + constitution check

- `docs/specs/skills/SPEC-internal-skill-optimize.md` — (modify) mirror the 4a policy amendments.
- `docs/specs/skills/SPEC-_preamble.md` — NOT touched (4b relocated to `fab/project/code-quality.md`, a project file with no SPEC mirror).
- **Constitution / shll standards**: untouched. This change modifies no CLI surface, help output, README.md, or docs/site/ content, so the constitution's Toolkit Standards article triggers no standards check, and no constitution amendment is needed (the ownership convention is a kit-authoring convention, not a normative project constraint).
- Aggregate specs (`skills.md`, `glossary.md`, `architecture.md`): grep for "optimization target" / "one output example" phrasing during apply; update only if they restate the amended rules.

## Affected Memory

- *(none expected)* — `internal-skill-optimize` has no dedicated memory topic file and this change does not warrant creating one (the SPEC mirror carries the behavior); `fab/project/code-quality.md` is project configuration, not kit behavior, so no memory file documents its contents. `_shared/context-loading.md` is NOT affected — `_preamble.md` is no longer touched (4b relocated per the Origin amendment note).

## Impact

- **Files edited**: `src/kit/skills/internal-skill-optimize.md` (4 edits, ~15 lines net), `fab/project/code-quality.md` (one Anti-Patterns bullet + one Sibling-&-Mirror-Sweeps cross-reference sentence), `docs/specs/skills/SPEC-internal-skill-optimize.md`.
- **Files deliberately NOT touched**: `src/kit/skills/_preamble.md` + `docs/specs/skills/SPEC-_preamble.md` (4b relocated — user-directed), `src/kit/skills/git-pr.md`, `src/kit/skills/git-pr-review.md` (4c skipped; restatements load-bearing), the `helpers:` allow-list, Constraint 3's scope, constitution.
- **No Go code, no tests, no migrations** — prose-only change to two kit skill files plus mirrors.
- **Verification**: `fab sync` then spot-load deployed copies; grep the protected-set literals (`✓ commit`, `Fixed —`, etc.) before/after — byte-identical; confirm no `##` heading set changed in either edited file.

## Open Questions

*(none — the invocation resolved every decision point or delegated it with guidance; the delegated 4c decision is made and recorded above)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | **4c: skip the `_git.md` extraction**; ship Phase 4 policy-only | User confirmed the skip explicitly ("don't do 4c. Only 4a and 4b") after reviewing the cost/benefit: net saving ~15–30 lines (not ~60) once parameter plumbing is counted, per-invocation load increases, and 4b + 4a-4 cover the drift risk. Extraction remains available as future work if the pair drifts in practice | S:95 R:90 A:90 D:90 |
| 2 | Certain | **Skip `_reorg.md` too** | Plan explicitly ranks it below `_git.md` ("skip if the allow-list churn isn't wanted"); skipping the higher-ranked extraction entails skipping the lower | S:90 R:95 A:90 D:90 |
| 3 | Confident | **Include the optional 4a-4 sibling-duplication signal** (as a report-only content signal + batch-mode comparison step) | User marked it optional; including it is trivially removable, directly supported by the audit's cross-file findings, and it is the enforcement hook for the 4b convention | S:60 R:95 A:85 D:65 |
| 4 | Certain | **4b relocated to `fab/project/code-quality.md`** (Anti-Patterns bullet + Sibling-&-Mirror-Sweeps cross-reference); `_preamble.md` untouched | User-directed after intake creation: `_preamble.md` deploys to every installing repo where the authoring convention has no enforcement surface; `code-quality.md` is this repo's always-loaded review-enforced policy file. Constitution rejected as heavier register | S:95 R:90 A:90 D:90 |
| 5 | Certain | **Constitution + shll standards untouched** | Change touches no CLI surface/README/docs-site — the Toolkit Standards article's trigger conditions don't fire; verified against constitution § Toolkit Standards at HEAD | S:80 R:90 A:85 D:85 |
| 6 | Certain | **FKF §3.3 cited via `$(fab kit-path)/reference/fkf.md`**, not `docs/specs/fkf.md` | Deployed-skill leak lesson from Phase 1 item #8: dev-repo-only paths are absent in user repos; the kit-path reference ships with the kit | S:75 R:90 A:90 D:85 |
| 7 | Certain | **4a-1's narrowed exemption updated at all four statement sites** (lines 12, 29, 35, 113) consistently | The exemption is restated four times in the file; amending one site while leaving the absolute wording elsewhere would create intra-file contradiction — the exact defect class this effort fixes | S:85 R:90 A:95 D:90 |
| 8 | Certain | **`change_type: feat` kept as inferred** | Phase 4 adds new policy rules/signals and a new convention — additive behavior; siblings used `fix`/`refactor` matching their phases' nature | S:80 R:95 A:90 D:85 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
