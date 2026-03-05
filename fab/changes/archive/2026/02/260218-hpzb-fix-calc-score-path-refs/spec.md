# Spec: Fix calc-score.sh Short-Form Path References

**Change**: 260218-hpzb-fix-calc-score-path-refs
**Created**: 2026-02-18
**Affected memory**: None

## Skill Files: Path Reference Consistency

### Requirement: Full repo-root-relative paths for calc-score.sh

All backtick-enclosed references to `calc-score.sh` in skill markdown files SHALL use the full repo-root-relative path `fab/.kit/scripts/lib/calc-score.sh`. Short forms (`calc-score.sh`, `lib/calc-score.sh`) SHALL NOT appear in backtick-enclosed references.

This applies to both invocation-style references (where the LLM would construct a bash command) and descriptive references (where the script is named in context), since both appear in backticks and could be interpreted as executable paths.

The shell script itself (`fab/.kit/scripts/lib/calc-score.sh`) SHALL NOT be modified — it resolves paths internally via `BASH_SOURCE`.

#### Scenario: LLM invokes calc-score.sh from repo root
- **GIVEN** a skill file instructs the LLM to run `calc-score.sh`
- **WHEN** the LLM constructs a bash command from the backtick-enclosed path
- **THEN** the path `fab/.kit/scripts/lib/calc-score.sh` resolves correctly from the repo root

#### Scenario: All backtick references use full path
- **GIVEN** the skill files `fab/.kit/skills/fab-ff.md` and `fab/.kit/skills/_context.md`
- **WHEN** searching for backtick-enclosed references to `calc-score.sh`
- **THEN** every occurrence uses the full path `fab/.kit/scripts/lib/calc-score.sh`
- **AND** no occurrences of bare `calc-score.sh` or `lib/calc-score.sh` remain in backticks

### Requirement: Specific occurrences to fix

The following 5 short-form references SHALL be replaced with the full path:

| # | File | Line | Current | Fix |
|---|------|------|---------|-----|
| 1 | `fab/.kit/skills/fab-ff.md` | 14 | `` `calc-score.sh --check-gate` `` | `` `fab/.kit/scripts/lib/calc-score.sh --check-gate` `` |
| 2 | `fab/.kit/skills/fab-ff.md` | 28 | `` `lib/calc-score.sh --check-gate <change_dir>` `` | `` `fab/.kit/scripts/lib/calc-score.sh --check-gate <change_dir>` `` |
| 3 | `fab/.kit/skills/_context.md` | 151 | `` `calc-score.sh` `` | `` `fab/.kit/scripts/lib/calc-score.sh` `` |
| 4 | `fab/.kit/skills/_context.md` | 279 | `` `calc-score.sh --check-gate` `` | `` `fab/.kit/scripts/lib/calc-score.sh --check-gate` `` |
| 5 | `fab/.kit/skills/_context.md` | 283 | `` `lib/calc-score.sh` `` | `` `fab/.kit/scripts/lib/calc-score.sh` `` |

References already using the full path (no change needed):
- `_context.md:279` — first reference on that line
- `fab-clarify.md:95`
- `fab-continue.md:70`
- `_generation.md:50`

#### Scenario: No regressions in already-correct references
- **GIVEN** references that already use `fab/.kit/scripts/lib/calc-score.sh`
- **WHEN** the fix is applied
- **THEN** those references remain unchanged

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `fab/.kit/scripts/lib/calc-score.sh` as the canonical path | Confirmed from intake #1 — matches convention from stageman fix (260217-eywl) and existing correct references in skill files | S:95 R:95 A:95 D:95 |
| 2 | Certain | Only modify skill markdown files, not the shell script | Confirmed from intake #2 — script resolves paths internally via `BASH_SOURCE` | S:90 R:95 A:95 D:95 |
| 3 | Certain | Fix all 5 backtick-enclosed short-form references, not just the 3 invocation-style ones identified in intake | Descriptive references in backticks are equally ambiguous to an LLM; consistent full paths prevent any confusion. Expanded from intake's 3 to 5 occurrences during spec verification | S:85 R:95 A:90 D:90 |

3 assumptions (3 certain, 0 confident, 0 tentative, 0 unresolved).
