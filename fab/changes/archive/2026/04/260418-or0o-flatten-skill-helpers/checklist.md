# Quality Checklist: Flatten Skill Helper Include Tree

**Change**: 260418-or0o-flatten-skill-helpers
**Generated**: 2026-04-18
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Single universal helper: `src/kit/skills/_preamble.md` is the sole always-load helper; no skill inherits another helper without declaring it in frontmatter
- [ ] CHK-002 `_preamble` contains no external "also read" directives: grep for `Also read the \`_` in `_preamble.md` returns zero matches
- [ ] CHK-003 Naming conventions inlined: `src/kit/skills/_preamble.md` contains a `## Naming Conventions` subsection with content equivalent to the original `_naming.md` body
- [ ] CHK-004 Run-Kit reference inlined: `src/kit/skills/_preamble.md` contains a `## Run-Kit (rk) Reference` subsection with content equivalent to the original `_cli-rk.md` body, including silent-fail-when-rk-missing behavior
- [ ] CHK-005 Common fab commands inlined: `src/kit/skills/_preamble.md` contains a `## Common fab Commands` subsection documenting the top-6 runtime command families (`preflight`, `score`, `log command`, `change`, `resolve`, `status`) with purpose + canonical form
- [ ] CHK-006 `_cli-fab.md` compressed to ≤300 lines: `wc -l src/kit/skills/_cli-fab.md` reports ≤300
- [ ] CHK-007 `Skill Helper Declaration` convention documented: `_preamble.md` has a `## Skill Helper Declaration` subsection listing the 4 allowed values (`_generation`, `_review`, `_cli-fab`, `_cli-external`), stating `_naming`/`_cli-rk` are not allowed (inlined), and stating `_preamble` is implicit
- [ ] CHK-008 Every skill has correct `helpers:` per spec mapping: `fab-new`, `fab-draft` → `[_generation]`; `fab-continue`, `fab-ff`, `fab-fff` → `[_generation, _review]`; `fab-operator` → `[_cli-fab, _cli-external]`; all other 19 skills → empty or omitted
- [ ] CHK-009 Memory `context-loading.md` reflects new opt-in model: Always Load Layer describes single universal helper; Skill Helper Declaration subsection added; Always-Load `_cli-rk` design decision marked superseded
- [ ] CHK-010 Memory `kit-architecture.md` updated: directory tree omits `_naming.md`; mapping table lists `_preamble` as the sole always-load helper; `_cli-fab.md` moved to Selective
- [ ] CHK-011 Specs updated: `SPEC-preamble.md` documents the 4 new subsections; `SPEC-_cli-rk.md` deleted; `docs/specs/skills.md` documents `helpers:` frontmatter field
- [ ] CHK-012 Backlog item 84bh removed from `fab/backlog.md`

## Removal Verification

- [ ] CHK-013 `src/kit/skills/_naming.md` deleted
- [ ] CHK-014 `src/kit/skills/_cli-rk.md` deleted
- [ ] CHK-015 `docs/specs/skills/SPEC-_cli-rk.md` deleted
- [ ] CHK-016 No deployed `.claude/skills/_naming/` or `.claude/skills/_cli-rk/` directories after `fab sync` (stale directories removed)
- [ ] CHK-017 Grep for `_naming.md` across `src/kit/skills/*.md` returns zero matches (all inline references updated to `_preamble.md § Naming Conventions`)
- [ ] CHK-018 Grep for `_cli-rk.md` across `src/kit/skills/*.md` returns zero matches
- [ ] CHK-019 `_cli-fab.md` no longer duplicates the 5 common commands inlined in `_preamble.md` (canonical source is `_preamble`, `_cli-fab` cross-references it)
- [ ] CHK-020 No canonical fab flag documentation lost during `_cli-fab.md` compression (content either retained, cross-referenced from `_preamble`, or relocated to `docs/memory/fab-workflow/`)

## Scenario Coverage

- [ ] CHK-021 Scenario "Read-only skill loads only _preamble": verified by inspecting a read-only skill (e.g., `fab-discuss.md`) — its frontmatter has no `helpers:` field and its body does not reference other helpers
- [ ] CHK-022 Scenario "Planning skill loads declared helpers": verified by inspecting `fab-continue.md` — its frontmatter declares `helpers: [_generation, _review]`
- [ ] CHK-023 Scenario "Fanout directives removed": `_preamble.md` contains zero "Also read the `_X`" paragraphs
- [ ] CHK-024 Scenario "Inline references updated": `git-branch.md` and `git-pr.md` reference `_preamble.md § Naming Conventions` instead of `_naming.md`
- [ ] CHK-025 Scenario "Common commands accessible without _cli-fab": a skill with no `helpers:` list can issue the 5 common commands correctly based solely on the `_preamble` subsection
- [ ] CHK-026 Scenario "Operator skill still works": `fab-operator.md` continues to reference all needed commands and helpers after its frontmatter and body updates

## Edge Cases & Error Handling

- [ ] CHK-027 Unknown helper value in `helpers:` frontmatter: the convention-only enforcement means behavior is "agent tries to load `.claude/skills/{unknown}/SKILL.md` and gets a file-not-found". Verified by intent — not tested in this change since enforcement is a Non-Goal.
- [ ] CHK-028 A skill that genuinely needs `_cli-fab` but forgets to declare it: agent will call a fab command without knowing exhaustive flags — degrades to best-effort. Verify: post-change grep of each skill body for `fab ` calls matches the commands available via its declared helpers + `_preamble` commons.
- [ ] CHK-029 `fab sync` after inline helper deletion: regenerates `.claude/skills/` without `_naming/` and `_cli-rk/` directories, or with stale directories that must be manually removed (expected behavior per spec).

## Code Quality

- [ ] CHK-030 Pattern consistency: new `## Skill Helper Declaration` subsection in `_preamble.md` follows the same Markdown heading/prose style as existing subsections
- [ ] CHK-031 No unnecessary duplication: the 5 common commands inlined into `_preamble` do NOT also appear in `_cli-fab.md` (cross-reference, don't duplicate)
- [ ] CHK-032 Readability over cleverness: inlined content from `_naming.md` and `_cli-rk.md` appears verbatim (or only trivially adjusted for heading level) — no restructure that could introduce behavioral drift
- [ ] CHK-033 No god functions / oversized sections: `_preamble.md` remains navigable — each new subsection is clearly delimited with a heading

## Documentation Accuracy

- [ ] CHK-034 Memory changelog entries added to `context-loading.md` and `kit-architecture.md` citing this change folder
- [ ] CHK-035 Design decision "Flatten Helper Include Tree" added to `context-loading.md` with rationale (fanout unreliable, 15/24 skills don't use) and rejected alternatives (deeper split, do nothing, full inline)
- [ ] CHK-036 Superseded design decision "Always-Load `_cli-rk` Skill for rk Capabilities" marked as superseded with pointer to this change

## Cross References

- [ ] CHK-037 `_preamble.md` § Common fab Commands cross-references `_cli-fab.md` for exhaustive documentation
- [ ] CHK-038 `_cli-fab.md` (compressed) cross-references `_preamble.md § Common fab Commands` for the inlined 5
- [ ] CHK-039 `git-branch.md` and `git-pr.md` cross-references point to `_preamble.md § Naming Conventions` (not the now-deleted `_naming.md`)
- [ ] CHK-040 `fab-operator.md` Required Reading list references only existing files (no pointer to deleted `_naming.md`)
- [ ] CHK-041 `SPEC-preamble.md` cross-references `_preamble.md` as canonical source for each of the 4 new subsections

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
