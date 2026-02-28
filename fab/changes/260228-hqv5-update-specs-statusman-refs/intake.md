# Intake: Update Specs References from Stageman to Statusman

**Change**: 260228-hqv5-update-specs-statusman-refs
**Created**: 2026-02-28
**Status**: Draft

## Origin

> Update docs/specs/ references from stageman to statusman

Identified during Copilot review of PR #177 (260228-9fg2-refactor-kit-scripts). The refactor renamed `stageman.sh` → `statusman.sh` and updated all skill files, memory files, and scripts — but `docs/specs/` was intentionally excluded because specs are human-curated per constitution §VI. This follow-up change applies the rename to the spec files as a conscious human-initiated edit.

## Why

Seven spec files still reference `stageman.sh` (9 occurrences total). Developers reading the specs will encounter stale script names that don't match the actual codebase, causing confusion when they try to locate or invoke the referenced scripts. The rename to `statusman.sh` was completed in PR #177 — `stageman.sh` no longer exists.

## What Changes

### Text replacements in docs/specs/

Simple find-and-replace across 7 files — no structural or semantic changes:

| File | Line(s) | Current | Updated |
|------|---------|---------|---------|
| `architecture.md` | 40, 103 | `stageman.sh` | `statusman.sh` |
| `change-types.md` | 86 | `stageman.sh set-change-type` | `statusman.sh set-change-type` |
| `glossary.md` | 78 | `stageman.sh` in script listing | `statusman.sh` |
| `naming.md` | 56 | `stageman.sh get-issues` | `statusman.sh get-issues` |
| `skills.md` | 208 | `stageman.sh start intake` | `statusman.sh start intake` |
| `templates.md` | 52, 63 | `stageman.sh` | `statusman.sh` |
| `user-flow.md` | 260 | `stageman.sh` | `statusman.sh` |

Additionally, `architecture.md` line 40 has a directory tree showing `stageman.sh` — update to `statusman.sh`. Line 103 lists example scripts — update `lib/stageman.sh` to `lib/statusman.sh`. The `glossary.md` line 78 also lists internal scripts — update the enumeration to include `resolve.sh`, `logman.sh`, and `statusman.sh` (replacing `stageman.sh`), reflecting the new 5-script architecture from PR #177.

## Affected Memory

None — specs are independent of memory files. No behavioral changes.

## Impact

- `docs/specs/architecture.md` — directory tree + script examples table
- `docs/specs/change-types.md` — one reference
- `docs/specs/glossary.md` — script enumeration
- `docs/specs/naming.md` — one reference
- `docs/specs/skills.md` — one reference
- `docs/specs/templates.md` — two references
- `docs/specs/user-flow.md` — one reference

## Open Questions

None.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Pure text replacement: stageman → statusman | No semantic changes, no new content — only updating stale script names to match the codebase | S:95 R:95 A:95 D:95 |
| 2 | Certain | Update glossary script enumeration to reflect new architecture | The glossary lists internal scripts — should match the actual set (resolve.sh, statusman.sh, logman.sh, changeman.sh, calc-score.sh) | S:90 R:90 A:90 D:95 |

2 assumptions (2 certain, 0 confident, 0 tentative, 0 unresolved).
