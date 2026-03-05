# Intake: Fix calc-score.sh Short-Form Path References

**Change**: 260218-hpzb-fix-calc-score-path-refs
**Created**: 2026-02-18
**Status**: Draft

## Origin

> Fix calc-score.sh short-form path references in skill files — same pattern as the stageman path fix (260217-eywl). Replace `lib/calc-score.sh` with `fab/.kit/scripts/lib/calc-score.sh` in skill markdown files where the short form is used.

Follow-up from a review comment on the stageman path fix change (260217-eywl-fix-stageman-skill-path-refs):

> `lib/calc-score.sh` short-form references remain in a few skill files — same pattern, out of scope for this change. Consider a follow-up.

One-shot request. The prior change established the pattern; this applies it to the remaining script.

## Why

Skill markdown files instruct the LLM to invoke `lib/calc-score.sh`, which is relative to `fab/.kit/scripts/`. However, Claude executes bash commands from the **repo root**, where `lib/calc-score.sh` does not resolve. This is the exact same inconsistency that was fixed for `stageman.sh` in 260217-eywl.

Some references in `_context.md` and `fab-continue.md` already use the correct full path (`fab/.kit/scripts/lib/calc-score.sh`), but others still use short forms. The inconsistency means the LLM must silently infer the correct path — fragile and inconsistent with the convention established by the stageman fix.

## What Changes

### Path reference update across skill files

Replace short-form `calc-score.sh` references with the repo-root-relative `fab/.kit/scripts/lib/calc-score.sh`. Three occurrences across two files:

| File | Line | Current | Fix |
|------|------|---------|-----|
| `fab/.kit/skills/fab-ff.md` | 28 | `` `lib/calc-score.sh --check-gate <change_dir>` `` | `` `fab/.kit/scripts/lib/calc-score.sh --check-gate <change_dir>` `` |
| `fab/.kit/skills/_context.md` | 279 | `` `calc-score.sh --check-gate` `` | `` `fab/.kit/scripts/lib/calc-score.sh --check-gate` `` |
| `fab/.kit/skills/_context.md` | 283 | `` `lib/calc-score.sh` `` | `` `fab/.kit/scripts/lib/calc-score.sh` `` |

References already using the full path (no change needed):
- `_context.md:279` — first reference on that line is already correct
- `fab-clarify.md:95` — already correct
- `fab-continue.md:70` — already correct

### No script changes

The bash script (`calc-score.sh`) resolves paths correctly via `BASH_SOURCE` and does not need changes.

## Affected Memory

None — this is a mechanical path string replacement with no behavioral or spec-level change.

## Impact

- **Skill files**: 2 files modified (text-only path string replacements, 3 occurrences)
- **No behavioral change**: Same commands, correct paths
- **No script changes**: Shell script already resolves paths correctly internally

## Open Questions

None — this is a mechanical find-and-replace with a clear target pattern, following the exact precedent set by 260217-eywl.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `fab/.kit/scripts/lib/calc-score.sh` as the canonical path | Matches the convention established by the stageman fix (260217-eywl), `_context.md` existing correct references, and `settings.local.json` allowlist pattern | S:95 R:95 A:95 D:95 |
| 2 | Certain | Only modify skill markdown files, not the shell script | Script resolves paths internally via `BASH_SOURCE`; the issue is only in the instructions the LLM reads | S:90 R:95 A:95 D:95 |

2 assumptions (2 certain, 0 confident, 0 tentative, 0 unresolved).
