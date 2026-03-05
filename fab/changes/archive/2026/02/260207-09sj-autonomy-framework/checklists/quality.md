# Quality Checklist: Add SRAD Autonomy Framework to Planning Skills

**Change**: 260207-09sj-autonomy-framework
**Generated**: 2026-02-08
**Spec**: `spec.md`

## Functional Completeness

<!-- Every requirement in spec.md has working implementation -->
- [x] CHK-001 SRAD Scoring Table: `_context.md` contains formalized 4-dimension scoring table with High/Low descriptions
- [x] CHK-002 Confidence Grades: `_context.md` defines all 4 grades (Certain, Confident, Tentative, Unresolved) with markers and visibility rules
- [x] CHK-003 Worked Examples: `_context.md` includes 2-3 worked examples showing SRAD dimensions producing a confidence grade
- [x] CHK-004 Critical Rule: `_context.md` states that Unresolved+low-R+low-A decisions MUST always be asked
- [x] CHK-005 Assumed Marker Convention: `_context.md` documents `<!-- assumed: {description} -->` format and placement rules
- [x] CHK-006 fab-new SRAD Questions: `fab-new.md` uses SRAD scoring to select up to 3 Unresolved questions
- [x] CHK-007 fab-new Branch Auto-Create: `fab-new.md` auto-creates branch on main/master without prompting
- [x] CHK-008 fab-new Assumptions Output: `fab-new.md` output ends with Assumptions summary table
- [x] CHK-009 fab-new Assumptions in Artifact: `fab-new.md` persists `## Assumptions` section in `proposal.md`
- [x] CHK-010 fab-continue NEEDS CLARIFICATION Count: `fab-continue.md` output includes marker count
- [x] CHK-011 fab-continue Key Decisions Block: `fab-continue.md` output includes Key Decisions after plan generation
- [x] CHK-012 fab-continue Assumptions Summary: `fab-continue.md` output + artifact include Assumptions summary
- [x] CHK-013 fab-continue SRAD Questions: `fab-continue.md` applies SRAD for 1-3 questions per stage
- [x] CHK-014 fab-ff --auto Skips Questions: `fab-ff.md` specifies --auto has hard zero interruptions
- [x] CHK-015 fab-ff Plan-Skip Reasoning: `fab-ff.md` output includes one-sentence plan skip/generate rationale
- [x] CHK-016 fab-ff Cumulative Assumptions: `fab-ff.md` output ends with cumulative Assumptions summary across stages
- [x] CHK-017 fab-ff Per-Artifact Assumptions: `fab-ff.md` persists `## Assumptions` section in each generated artifact
- [x] CHK-018 fab-ff Auto-Guess Markers: `fab-ff.md` --auto mode marks all Unresolved decisions with `<!-- auto-guess: ... -->`
- [x] CHK-019 fab-clarify Assumed Markers (Suggest): `fab-clarify.md` taxonomy scan detects `<!-- assumed: ... -->` markers
- [x] CHK-020 fab-clarify Assumed Markers (Auto): `fab-clarify.md` auto mode resolves `<!-- assumed: ... -->` markers
- [x] CHK-021 fab-clarify Assumed Question Format: `fab-clarify.md` frames current assumption as recommendation with alternatives
- [x] CHK-022 fab-apply Soft Gate: `fab-apply.md` checks for auto-guess markers and prompts before proceeding

## Behavioral Correctness

<!-- Changed requirements behave as specified, not as before -->
- [x] CHK-023 fab-new branch prompt removed: On main/master, no branch creation prompt appears (was: offer to create)
- [x] CHK-024 fab-new question selection changed: Questions are SRAD-scored, not just ambiguity-based (was: up to 3 [BLOCKING] by gut feel)
- [x] CHK-025 fab-clarify scan expanded: Taxonomy scan finds `<!-- assumed: ... -->` in addition to existing `<!-- auto-guess: ... -->` and `[NEEDS CLARIFICATION]`

## Scenario Coverage

<!-- Key scenarios from spec.md have been exercised -->
- [x] CHK-026 Scenario: Clear description produces assumptions summary but no questions
- [x] CHK-027 Scenario: Ambiguous description triggers exactly the SRAD-identified unresolved questions
- [x] CHK-028 Scenario: Tentative assumption inserts `<!-- assumed: ... -->` marker inline
- [x] CHK-029 Scenario: fab-ff --auto makes auto-guesses with markers, listed in output
- [x] CHK-030 Scenario: fab-apply with auto-guesses warns and prompts y/n
- [x] CHK-031 Scenario: fab-apply without auto-guesses skips gate entirely
- [x] CHK-032 Scenario: fab-clarify presents assumed marker as recommended option with alternatives

## Edge Cases & Error Handling

<!-- Error states, boundary conditions, failure modes -->
- [x] CHK-033 fab-new with 0 assumptions: Assumptions summary section is omitted or shows "0 assumptions made"
- [x] CHK-034 fab-apply user answers "n" to soft gate: Implementation does not begin, clarify suggested
- [x] CHK-035 fab-ff cumulative with 0 assumptions across all stages: Summary says "No assumptions made"

## Documentation Accuracy

<!-- Extra category from config.yaml -->
- [x] CHK-036 planning-skills.md updated: Reflects SRAD framework, autonomy levels, new design decisions
- [x] CHK-037 clarify.md updated: Reflects `<!-- assumed: ... -->` scanning in both modes
- [x] CHK-038 context-loading.md updated: Reflects SRAD protocol in shared context

## Cross References

<!-- Extra category from config.yaml -->
- [x] CHK-039 _context.md SRAD section: Referenced correctly from all 4 skill files (fab-new, fab-continue, fab-ff, fab-clarify)
- [x] CHK-040 Assumptions Summary format: Consistent across fab-new, fab-continue, fab-ff output specs
- [x] CHK-041 Marker conventions: `<!-- assumed: ... -->` and `<!-- auto-guess: ... -->` usage consistent across all skill files

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
