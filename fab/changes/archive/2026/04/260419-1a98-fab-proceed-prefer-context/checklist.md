# Quality Checklist: /fab-proceed — Prefer Conversation Context Over Stale Intakes

**Change**: 260419-1a98-fab-proceed-prefer-context
**Generated**: 2026-04-19
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Detection Order: `src/kit/skills/fab-proceed.md` describes 5 steps in the order Active Change → Branch → Conversation Classification → Unactivated Intake Scan → Dispatch Decision. Steps 3 and 4 are explicitly called out as order-independent.
- [x] CHK-002 Conversation Classification: The substantive-vs-empty/thin criterion is stated verbatim and references the 4 inclusion triggers (technical requirements, design decisions, specific values, problem statements). No separate "thin but non-empty" tier is introduced.
- [x] CHK-003 Dispatch Table: The skill file contains the new 7-row dispatch table (Active Yes+match / Active Yes+nomatch / No+Substantive+None / No+Substantive+≥1+Relevant / No+Substantive+≥1+NotRelevant / No+Empty+≥1 / No+Empty+None). All 7 rows map to correct step sequences per spec §Dispatch Table.
- [x] CHK-004 Relevance Assessment: The skill file has a Relevance Assessment subsection that specifies (a) reading title + Origin + Why + What Changes per candidate, (b) clearly-relevant classification requires shared topic + overlapping terminology + consistent scope, (c) asymmetric-bias rule maps ambiguous to not clearly relevant, (d) date-descending tiebreak only among equally-relevant candidates.
- [x] CHK-005 Output Format — Bypass Note: The skill's Output section specifies the exact Note wording and that Notes appear BEFORE the `Created intake:` line, in date-descending order when multiple.
- [x] CHK-006 Zero-Prompt Preservation: The skill file explicitly states the skill does not prompt the user at any detection, relevance, or dispatch step.

## Behavioral Correctness

- [x] CHK-007 Step 3 + Step 4 ordering: The rewritten State Detection section makes clear that conversation classification is NOT gated on intake absence (unlike the current behavior where Step 4 only runs when Step 3 finds no intake). Text explicitly states both run when no active change exists.
- [x] CHK-008 Empty/thin → activate preserved: The new table's row 6 (No active + Empty/thin + ≥1 intake) resolves to `/fab-switch` → `/git-branch` → `/fab-fff` using the date-recency most-recent pick, preserving the resume-flow convenience that the current behavior provides.

## Removal Verification

- [x] CHK-009 Old 5-row dispatch table removed: The old table (with rows `Unactivated intake (no active change)` and `Conversation context (no intake)`) is no longer present in `src/kit/skills/fab-proceed.md`.
- [x] CHK-010 Old Step 3 "Unactivated Intake Check" language removed: The language "If no intake exists anywhere" (current gating of Step 4 on Step 3 emptiness) is removed; no remaining text implies Step 4 short-circuits when an intake is found.

## Scenario Coverage

- [x] CHK-011 Scenario "Substantive conversation with relevant draft" (spec §Dispatch Table): Manually trace the code path in the rewritten skill — substantive conversation + 1 relevant intake → relevance check classifies it clearly relevant → dispatch `/fab-switch` → `/git-branch` → `/fab-fff`. Confirm the skill text supports this path.
- [x] CHK-012 Scenario "Substantive conversation with unrelated draft" (spec §Dispatch Table): Manually trace — substantive conversation + 1 unrelated intake → relevance check classifies as not clearly relevant → dispatch `/fab-new` + emit bypass Note. Confirm the skill text supports this path.
- [x] CHK-013 Scenario "Empty conversation with existing draft" (spec §Dispatch Table): Manually trace — empty/thin + ≥1 intake → date-recency pick → dispatch `/fab-switch` → `/git-branch` → `/fab-fff`, no Note emitted. Confirm the skill text supports this path.
- [x] CHK-014 Scenario "Empty conversation with no draft" (spec §Dispatch Table): Manually trace — empty/thin + no intake → error with wording preserved from today (`Nothing to proceed with — start a discussion or run /fab-new (or /fab-draft) first.`). Confirm.
- [x] CHK-015 Scenario "Multiple clearly-relevant candidates" (spec §Relevance Assessment): Manually trace — two intakes both clearly relevant → date-descending tiebreak picks the higher `YYMMDD`; the non-selected candidate is left untouched with no bypass Note (activation path emits no Notes per Output Format).
- [x] CHK-016 Scenario "Multiple bypasses" (spec §Output Format): Manually trace — two unrelated intakes → two Note lines emitted in date-descending order, both BEFORE the `Created intake:` line.

## Edge Cases & Error Handling

- [x] CHK-017 Ambiguous relevance resolves without prompting: The text explicitly states the asymmetric-bias rule maps ambiguous candidates to not clearly relevant — no user prompt is issued.
- [x] CHK-018 Zero candidates after scan + substantive conversation: Skill dispatches `/fab-new` + emits no Note lines (there were no drafts to bypass).
- [x] CHK-019 Activation path emits no Note: When the skill activates an existing intake (relevant or empty/thin branches), the output contains no `Note: unactivated draft` lines.

## Code Quality

- [x] CHK-020 Pattern consistency: The rewritten skill file follows the same frontmatter, section ordering, and heading style as sibling skill files in `src/kit/skills/` (e.g., `fab-fff.md`, `fab-new.md`).
- [x] CHK-021 No unnecessary duplication: The substantive-vs-empty criterion is defined once (in Step 3 of State Detection) and referenced from the dispatch table rather than re-stated.

## Documentation Accuracy

- [x] CHK-022 SPEC-fab-proceed.md matches skill file: The Flow diagram, Dispatch Table, and Sub-agents section in `docs/specs/skills/SPEC-fab-proceed.md` describe the same logic as `src/kit/skills/fab-proceed.md`. No contradictions.
- [x] CHK-023 execution-skills.md matches skill file: The `**Pipeline orchestrator**:` paragraph in `docs/memory/fab-workflow/execution-skills.md` describes the new 5-step detection, 7-row dispatch, relevance check, asymmetric-bias rule, and bypass-note output. No stale 4-step or 5-row references remain.

## Cross-References

- [x] CHK-024 All `fab-proceed` references in `docs/memory/` point to consistent behavior: Grep `docs/memory/` for `fab-proceed` and confirm every hit describes the new logic (or is generic/unaffected).
- [x] CHK-025 All `fab-proceed` references in `docs/specs/` point to consistent behavior: Grep `docs/specs/` for `fab-proceed` and confirm every hit (including `user-flow.md`, `operator.md`, `skills.md` if they reference the dispatch table) either describes the new logic or is high-level and unaffected.

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`
