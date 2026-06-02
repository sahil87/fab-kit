# Quality Checklist: git-pr true-impact line count

**Change**: 260507-asvz-git-pr-true-impact-line-count
**Generated**: 2026-05-07
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Optional `true_impact_exclude` config field: `src/kit/scaffold/fab/project/config.yaml` contains a top-level `true_impact_exclude: [fab/, docs/]` entry (flow style).
- [x] CHK-002 Optional `true_impact_exclude` config field: this project's own `fab/project/config.yaml` contains a top-level `true_impact_exclude: [fab/, docs/]` entry (flow style).
- [x] CHK-003 Impact block computation: `src/kit/skills/git-pr.md` Step 3c includes a sub-step (3c-impact) that (a) reads `true_impact_exclude` via `yq`, (b) computes `BASE=$(git merge-base origin/main HEAD)`, (c) runs two `git diff --shortstat "$BASE...HEAD"` invocations — one with pathspec exclusions, one without.
- [x] CHK-004 Impact block format and placement: the rendered block is exactly two lines — `**Impact (excluding {COMMA_LIST})**: +A / −D` and `**git diff total**: +A_total / −D_total` — appended after the pipeline progress line with one blank-line separator, no `## Impact` heading.
- [x] CHK-005 Graceful degradation when fab context absent: `src/kit/skills/git-pr.md` 3c-impact step checks `{has_fab}` and skips the block when false.
- [x] CHK-006 New-project default in scaffold: scaffold edit (T001) is preserved through `fab sync` semantics — `src/kit/scaffold/fab/project/config.yaml` is the source-of-truth file.
- [x] CHK-007 Migration adds the field to existing configs: `src/kit/migrations/1.9.1-to-1.9.2.md` exists, follows the `{FROM}-to-{TO}.md` convention, and contains Summary / Pre-check / Changes / Verification sections per the established format.
- [x] CHK-008 Skill spec update: `docs/specs/skills/SPEC-git-pr.md` documents the new 3c-impact sub-step in the flow diagram and notes the two `git diff --shortstat` reads.

## Behavioral Correctness

- [x] CHK-009 Field absent → impact block omitted: when `fab/project/config.yaml` has no `true_impact_exclude` key, the PR body matches today's output byte-for-byte.
- [x] CHK-010 Field empty / null → impact block omitted: `true_impact_exclude: []` and `true_impact_exclude: null` produce the same output as "field absent".
- [x] CHK-011 Exclusion list rendered verbatim from config: the parenthetical in the first line reflects the actual config values (e.g., `(excluding vendor/)` for a single-entry list, `(excluding fab/, docs/, vendor/)` for three) — never hardcoded.
- [x] CHK-012 Three-dot range used for both passes: `git diff --shortstat "$BASE...HEAD"` (three dots), not `..` (two dots).
- [x] CHK-013 `--shortstat` clause-missing parser: when the output omits the `insertions(+)` or `deletions(-)` clause, the parsed value is `0` and renders as `+0` / `−0`.
- [x] CHK-014 True-impact pass with zero changes → block omitted: when every modified file falls inside an excluded path, the entire block is skipped (no misleading `+0 / −0`).
- [x] CHK-015 Migration idempotency: re-running `/fab-setup migrations` after the migration has been applied is a no-op (existing `true_impact_exclude` is detected and preserved, even when the user value differs from `[fab/, docs/]`).
- [x] CHK-016 Migration when `fab/project/config.yaml` is absent: the migration prints a skip message and does not error.

## Scenario Coverage

- [x] CHK-017 Field present with directory list — verifiable by manual run of `/git-pr` against this PR (since T002 populates the field on this branch).
- [x] CHK-018 Both passes return non-zero counts — verifiable by inspecting the PR body of this PR (which touches `fab/`, `docs/`, AND `src/kit/`).
- [x] CHK-019 Single-entry exclusion list — verifiable by temporarily editing `true_impact_exclude` to `[fab/]` and re-running `/git-pr` body assembly mentally / via shell.
- [x] CHK-020 Block placement scenario — confirm by inspection of this PR's actual rendered body: impact block is the last content, after the pipeline progress line.

## Edge Cases & Error Handling

- [x] CHK-021 One side of the pair is zero: when only `insertions` exists (no deletions clause), the rendered first line is `+X / −0` (and symmetrically for deletions-only).
- [x] CHK-022 No fab context (`fab/project/config.yaml` does not exist): `/git-pr` runs without error and emits no impact block.
- [x] CHK-023 Migration concurrent-edit safety: atomic write (temp + rename) prevents partial writes if the user has the config open in an editor.
- [x] CHK-024 Migration with non-`[fab/, docs/]` existing value: the migration detects the field's presence and skips, preserving the user's custom value (`[vendor/]`, etc.) — does NOT overwrite.

## Code Quality

- [x] CHK-025 Pattern consistency: skill edits in `src/kit/skills/git-pr.md` match the existing numbered-sub-step style (e.g., 3c sub-steps 1–5 in the current file).
- [x] CHK-026 No unnecessary duplication: existing `yq` and `git` invocations are reused where they already appear in `/git-pr`; the migration file structure mirrors `1.8.0-to-1.9.0.md` rather than reinventing one.
- [x] CHK-027 Documentation accuracy: `docs/specs/skills/SPEC-git-pr.md` flow diagram, `Tools used` table, and `Sub-agents` notes all reflect the new sub-step truthfully (no stale prose).
- [x] CHK-028 Cross-references: the spec and the skill text agree on field name (`true_impact_exclude`), placement ("after pipeline progress line"), and format (two lines, no heading).

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`
