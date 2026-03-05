# Proposal: Add SRAD Autonomy Framework to Planning Skills

**Change**: 260207-09sj-autonomy-framework
**Created**: 2026-02-07
**Status**: Draft

## Why

The planning skills (fab-new, fab-continue, fab-ff) currently make 38 out of 47 identified decision points as silent assumptions — the developer never sees what was decided or why. (The full decision-point inventory will be catalogued per-skill in `spec.md`, grouped by blast-radius tier.) <!-- clarified: decision-point inventory deferred to spec stage --> Some of these assumptions are high-blast-radius (scope boundaries, architectural choices, GIVEN/WHEN/THEN scenarios) and cascade through the entire pipeline unchecked. Meanwhile, the skills sometimes ask low-value questions (e.g., branch creation on main) that interrupt flow without adding safety. The pipeline needs a principled framework for deciding when to ask, when to assume, and when to surface assumptions visibly.

## What Changes

- **Add SRAD scoring framework** to `_context.md` as a shared decision-making protocol for all planning skills
- **Update fab-new** to: always ask top ~3 critical questions per SRAD scoring; surface remaining assumptions in output; suggest `/fab-clarify` for assumption review; stop asking about branch creation on main/master (auto-create instead)
- **Update fab-continue** to: surface [NEEDS CLARIFICATION] counts in output summary; add "Key Decisions" output block after plan generation; include Assumptions summary table in output; suggest `/fab-clarify` when tentative assumptions exist
- **Update fab-ff** to: clarify that `--auto` skips frontloaded questions entirely (hard zero interruptions); add brief plan-skip reasoning; include cumulative Assumptions summary table in output; add auto-guess markers for all unresolved items in `--auto` mode
- **Add `<!-- assumed: ... -->` marker convention** alongside existing `<!-- auto-guess: ... -->` for tentative assumptions that fab-clarify can scan and resolve
- **Update fab-clarify** to scan for `<!-- assumed: ... -->` markers in addition to `<!-- auto-guess: ... -->` markers

## The SRAD + Confidence Grades Framework

### SRAD Scoring

For each decision point, score on four dimensions:

| Dimension | High (safe to assume) | Low (consider asking) |
|-----------|----------------------|----------------------|
| **S — Signal Strength** | Detailed description, multiple sentences, clear intent | One-liner, vague phrase, ambiguous scope |
| **R — Reversibility** | Easily changed later via fab-clarify or stage reset | Cascades through multiple artifacts, expensive to undo |
| **A — Agent Competence** | Config, constitution, codebase give clear answer | Business priorities, user preferences, political context |
| **D — Disambiguation Type** | One obvious default interpretation | Multiple valid interpretations with different tradeoffs |

### Confidence Grades

Each decision produces an assumption graded on a 4-level scale:

| Grade | Meaning | Artifact Marker | Output Visibility |
|-------|---------|----------------|-------------------|
| **Certain** | Determined by config/constitution/template rules | None | None — not worth mentioning |
| **Confident** | Strong signal, one obvious interpretation | None | Noted in Assumptions summary |
| **Tentative** | Reasonable guess, multiple valid options | `<!-- assumed: {description} -->` | Noted in Assumptions summary, fab-clarify suggested |
| **Unresolved** | Cannot determine, incompatible interpretations | `<!-- auto-guess: ... -->` (fab-ff --auto) | Asked as question (fab-new/continue), batched upfront (fab-ff default), auto-guessed (fab-ff --auto) |

### Critical Rule

**Unresolved decisions with low Reversibility and low Agent Competence MUST always be asked** — even in fab-new and fab-continue. These are the top ~3 questions per skill invocation. The existence of `/fab-clarify` as an escape valve does NOT justify silently assuming high-blast-radius decisions. fab-clarify is for Tentative assumptions, not for Unresolved ones.

### Skill-Specific Autonomy Levels

| Aspect | fab-new (capture) | fab-continue (deliberate) | fab-ff default (speed) | fab-ff --auto (full trust) |
|--------|-------------------|---------------------------|------------------------|----------------------------|
| **Posture** | Assume confident+tentative, ask top ~3 unresolved | Surface tentative, ask top ~3 unresolved | Batch all unresolved upfront, then go | Assume everything, mark unresolved as auto-guess |
| **Interruption budget** | 0 for branch-on-main; max 3 for unresolved questions | 1-2 per stage | 0-1 batch at start | Hard zero |
| **Output** | Assumptions summary + "Run /fab-clarify to review" | Key Decisions block + Assumptions summary + [NEEDS CLARIFICATION] count | Cumulative Assumptions summary | Auto-guesses list + Assumptions summary |
| **Escape valve** | `/fab-clarify` | `/fab-clarify` | `/fab-clarify` | `/fab-clarify` |

### Assumptions Summary Block

Every skill invocation ends output with:

```
## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | OAuth2 over SAML | Config shows REST API stack |
| 2 | Tentative | Google + GitHub providers | Most common OSS combination |
| 3 | Tentative | Supplement existing auth | Description says "add", not "replace" |

3 assumptions made (1 confident, 2 tentative). Run /fab-clarify to review.
```

For fab-ff, this accumulates across all stages and is displayed once at the end.

## Affected Docs

### Modified Docs
- `fab-workflow/planning-skills`: Adding SRAD framework references and autonomy level definitions
- `fab-workflow/clarify`: Updating to include `<!-- assumed: ... -->` marker scanning
- `fab-workflow/context-loading`: Adding SRAD protocol to shared context conventions
- `fab/.kit/skills/_context.md`: Adding SRAD scoring table, confidence grades, and worked examples
- `fab/.kit/skills/fab-new.md`: Frontloaded questions, assumptions summary, branch-on-main auto-create
- `fab/.kit/skills/fab-continue.md`: Key Decisions block, assumptions summary, [NEEDS CLARIFICATION] count
- `fab/.kit/skills/fab-ff.md`: Batched unresolved questions, auto-guess markers, cumulative assumptions summary
- `fab/.kit/skills/fab-clarify.md`: Scanning for `<!-- assumed: ... -->` markers
<!-- clarified: skill files added to Affected Docs for spec-stage completeness -->

### New Docs
- None (framework is documented inline in _context.md and skill files)

### Removed Docs
- None

## Impact

- **Skills affected**: fab-new.md, fab-continue.md, fab-ff.md, fab-clarify.md, _context.md
- **Templates affected**: None (Assumptions summary is output formatting, not an artifact template)
- **Scripts affected**: None
- **Backward compatibility**: All changes are additive — existing artifacts remain valid. New `<!-- assumed: ... -->` markers are HTML comments, invisible to renderers.

## Open Questions

- [RESOLVED] Assumptions summary MUST be persisted as a trailing `## Assumptions` section in each generated artifact. This ensures `<!-- assumed: ... -->` markers are scannable by `fab-clarify` and assumptions survive beyond terminal output. <!-- clarified: assumptions persisted in artifact, not output-only -->
- [RESOLVED] `fab-apply` SHALL implement a soft gate: if any `<!-- auto-guess: ... -->` markers exist in the tasks artifact, print a warning with the count and prompt "continue? (y/n)" before proceeding. This balances the "full trust" contract with a lightweight safety check. <!-- clarified: soft gate on fab-apply when auto-guesses exist -->
- [RESOLVED] SRAD scoring SHALL be documented as a formalized table in `_context.md` accompanied by 2-3 worked examples demonstrating how the four dimensions interact to produce a confidence grade. The table provides consistent reference; the examples prevent mechanical application and show nuanced judgment. <!-- clarified: SRAD as formalized table with worked examples -->

## Clarifications

### Session 2026-02-07

- **Q**: Should the Assumptions summary be persisted in the generated artifact or remain output-only?
  **A**: Persist in artifact — append as a trailing `## Assumptions` section in each generated artifact.
- **Q**: Should `fab-ff --auto` have a gate blocking `/fab-apply` if auto-guesses exist?
  **A**: Soft gate — `fab-apply` warns with count and prompts "continue? (y/n)" when auto-guesses exist.
- **Q**: Should SRAD scoring be a formalized table or conceptual framework?
  **A**: Formalized table with 2-3 worked examples showing nuanced dimensional interaction.
- **Q**: Should the proposal include the full decision-point inventory (47 items)?
  **A**: Defer to spec stage — inventory will be catalogued per-skill in `spec.md`, grouped by blast-radius tier.
- **Q**: Should skill files be listed under Affected Docs?
  **A**: Yes — added all 5 skill files to Modified Docs for spec-stage completeness.
