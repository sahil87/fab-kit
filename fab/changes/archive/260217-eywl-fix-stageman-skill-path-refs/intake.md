# Intake: Fix Stageman Skill Path References

**Change**: 260217-eywl-fix-stageman-skill-path-refs
**Created**: 2026-02-17
**Status**: Draft

## Origin

> Fix stageman path references in skill files — all `lib/stageman.sh` references in skill markdown files should be `fab/.kit/scripts/lib/stageman.sh` (repo-root-relative), consistent with how preflight.sh is referenced in _context.md.

One-shot request. Investigation in the preceding conversation confirmed the inconsistency across all pipeline skill files.

## Why

All skill markdown files reference stageman as `lib/stageman.sh`, which is relative to `fab/.kit/scripts/`. However, Claude executes bash commands from the **repo root**, where `lib/stageman.sh` does not resolve. This creates a mismatch between the documented invocation path and the actual execution context.

The `_context.md` preamble already uses the correct repo-root-relative path for `preflight.sh` (`fab/.kit/scripts/lib/preflight.sh`), but stageman references were never updated to match. The `settings.local.json` allowlist also uses the full path (`Bash(fab/.kit/scripts/lib/stageman.sh:*)`), confirming repo-root-relative is the intended convention.

If left unfixed, the LLM must silently infer the correct path — which works most of the time but is fragile and inconsistent with the preflight convention.

## What Changes

### Path reference update across skill files

Replace all occurrences of `lib/stageman.sh` with `fab/.kit/scripts/lib/stageman.sh` in the following files:

| File | Occurrences |
|------|-------------|
| `fab/.kit/skills/fab-continue.md` | ~12 |
| `fab/.kit/skills/fab-ff.md` | ~12 |
| `fab/.kit/skills/fab-fff.md` | ~12 |
| `fab/.kit/skills/fab-clarify.md` | ~1 |
| `fab/.kit/skills/fab-status.md` | ~1 |
| `fab/.kit/skills/_generation.md` | ~3 |

Also update `lib/preflight.sh` references in skill files if any use the short form (verify — `_context.md` already uses the full path, but individual skills may vary).

### No script changes

The bash scripts themselves (`stageman.sh`, `preflight.sh`, `changeman.sh`) resolve paths correctly via `$0`/`BASH_SOURCE` and do not need changes.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update path convention documentation to reflect repo-root-relative stageman references

## Impact

- **Skill files**: 6 files modified (text-only path string replacements)
- **No behavioral change**: Same commands, correct paths
- **No script changes**: Shell scripts already resolve paths correctly internally

## Open Questions

None — this is a mechanical find-and-replace with a clear target pattern.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `fab/.kit/scripts/lib/stageman.sh` as the canonical path | Matches `_context.md` preflight convention and `settings.local.json` allowlist | S:95 R:95 A:95 D:95 |
| 2 | Certain | Only modify skill markdown files, not shell scripts | Scripts resolve paths internally via `$0`; the issue is only in the instructions the LLM reads | S:90 R:95 A:95 D:95 |
| 3 | Confident | Also check `lib/preflight.sh` references for consistency | `_context.md` uses the full path, but individual skills may have short-form refs | S:70 R:90 A:85 D:85 |

3 assumptions (2 certain, 1 confident, 0 tentative, 0 unresolved).
