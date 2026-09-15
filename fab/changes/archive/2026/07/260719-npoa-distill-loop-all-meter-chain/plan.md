# Plan: Distill Loop-All + Narration-Meter Fix + Reorg→Distill Chain

**Change**: 260719-npoa-distill-loop-all-meter-chain
**Intake**: `intake.md`

## Requirements

### Meter: narration-density stops counting sanctioned citations

#### R1: Position-aware change-id counting in the narration-density meter
The `narration-density` meter (`internal/memoryindex` `topicBodyWarnings`) MUST count a body's narration markers as `countNarrationStems(body)` plus only the registry-gated change-id tokens that appear **outside** the two sanctioned-citation positions: (a) a parenthesized `(change-id)` citation, and (b) a change-id on an `*Introduced by*:` field line. Change-id tokens outside those positions (woven into prose) MUST still count. The stem list, threshold (`NarrationMarkerWarnThreshold = 5`), warning kind (`narration-density`), advisory (non-blocking) status, `Count` semantics, and the `--check --json` `warnings[]` shape are all unchanged.

- **GIVEN** a distilled topic file whose only change-id tokens are trailing `(change-id)` citations and `*Introduced by*:` field lines (and fewer than 5 narration stems)
- **WHEN** `fab memory-index --check` computes its body warnings
- **THEN** the file does NOT emit a `narration-density` warning (sanctioned citations no longer count toward the marker total)
- **AND** a file with ≥5 transition stems still emits the warning
- **AND** a mixed file counts stems plus only the change-id tokens outside the two allowed positions

#### R2: Stem list unchanged (no recall expansion)
The change MUST NOT expand `narrationStems`; the existing case-insensitive list (`no longer` / `previously` / `renamed` / `supersed`) is retained verbatim. The fix targets the false positive (sanctioned citations counting), not stem recall.

- **GIVEN** the existing stem list
- **WHEN** this change lands
- **THEN** `narrationStems` is byte-identical to before (only the change-id counting rule changes)

### Meter: tests conform to the new counting rule

#### R3: Tests pin the position-aware counting behavior
The change MUST ship tests in the same change (constitution test-alongside): a distilled fixture carrying only trailing `(id)` citations and `*Introduced by*:` lines does NOT flag; a fixture with ≥5 stems flags; a mixed fixture counts stems plus non-allowed-position ids only. Existing narration tests whose fixtures relied on parenthesized/allowed-position ids counting MUST be updated to conform to the new spec (never the reverse — Constitution VII).

- **GIVEN** the updated meter
- **WHEN** `go test ./internal/memoryindex/... ./cmd/fab/...` runs
- **THEN** all tests pass, including new fixtures that assert allowed-position ids do not count and non-allowed-position ids do

### Distill: no-arg default is a sequential all-domains loop

#### R4: No-arg `/docs-distill-memory` surveys once, then loops every flagged domain
`src/kit/skills/docs-distill-memory.md` no-arg behavior MUST survey once (the unchanged single `fab memory-index --check --json` call, same four-kind aggregation, same exclusion set, same older-binary grep fallback), then iterate EVERY flagged domain sequentially in `docs/memory/index.md` domain-table order, running the existing one-domain flow (full read → per-file report → per-domain approval → apply → regen) as the loop body per domain. The loop runs in the main session (no per-domain subagent dispatch — the approval prompt is interactive).

- **GIVEN** a no-arg `/docs-distill-memory` invocation with multiple flagged domains
- **WHEN** the skill runs
- **THEN** it surveys once, then processes each flagged domain in index.md order, one domain per approval unit
- **AND** the loop iterates the initial survey's flagged list (no re-survey between domains)

#### R5: Per-domain approval gate retained; bulk approval rejected
The per-domain approval prompt (apply all / cherry-pick / skip) MUST be retained as the loop body's gate. Bulk approval across all domains in one prompt MUST NOT be introduced — it would collapse the human safeguard on load-bearing memory files.

- **GIVEN** the all-domains loop
- **WHEN** each flagged domain is processed
- **THEN** the user is prompted per domain (apply all / cherry-pick / skip), never once for all domains

#### R6: Loop semantics — skip, no-rewrite, error-handling, terminal state
A **skipped** domain MUST stay untouched and move the loop on (reported in the terminal summary as skipped/remaining). A domain whose full read finds nothing MUST report "no rewrites proposed — already distilled" and continue. An exit-2 refuse-before-regen event within one domain follows the existing per-domain handling and MUST NOT silently swallow the remaining domains. The terminal state MUST be "all domains distilled" (every flagged domain processed) or a summary listing skipped/remaining domains.

- **GIVEN** a no-arg loop where the user skips domain A and domain B is already distilled
- **WHEN** the loop completes
- **THEN** A is untouched and listed as skipped, B reports already-distilled, and the run ends with an all-distilled-or-remaining summary

#### R7: Explicit `<domain>` override unchanged; multi-domain abort unchanged
An explicit `<domain>` argument MUST remain the targeted single-domain override — forces a full read, skips the survey, no loop. The multiple-explicit-domains abort in Error Handling MUST stay.

- **GIVEN** `/docs-distill-memory pipeline`
- **WHEN** the skill runs
- **THEN** it skips the survey, full-reads only `pipeline`, and does not loop
- **AND** `/docs-distill-memory a b` still aborts with the one-domain-per-run message

#### R8: Dynamic `Next:` semantics report surveyed truth, not re-invocation
The dynamic `Next:` line MUST report skipped/remaining domains (surveyed truth) rather than driving per-domain re-invocation — e.g. `Next: all domains distilled (survey heuristic) — /docs-reorg-memory or /fab-new`, or a line listing the skipped domains with flagged counts for a follow-up targeted run.

- **GIVEN** a completed no-arg loop with one skipped domain
- **WHEN** the skill emits its closing line
- **THEN** `Next:` lists the skipped domain with its flagged count (a follow-up targeted-run pointer), not a per-domain re-invocation instruction

#### R9: "One domain per run" language reframed to per-approval-unit
All "one domain per run" language in the skill (frontmatter `description:`, Purpose, Arguments, Key Properties, Output, Error Handling) MUST be reframed to "one domain per approval/apply unit, iterated within a single invocation" — the invocation processes every flagged domain, but each domain remains its own approval unit.

- **GIVEN** the reframed skill
- **WHEN** a reader consults any of those sections
- **THEN** none claims the invocation processes only one domain; each states the per-approval-unit framing

### Reorg: completion chains to distill

#### R10: `/docs-reorg-memory` emits a `Next: /docs-distill-memory` chain line
`src/kit/skills/docs-reorg-memory.md` MUST, at completion, reuse its existing single `fab memory-index --check --json` call's `warnings[]` output (no second survey call), aggregate flagged files with distill's survey rule (four kinds — `description-change-id`, `description-over-cap`, `description-length`, `narration-density`; dedupe by path; sub-domain rolls up to domain; re-apply the distillation exclusion set), and emit `Next: /docs-distill-memory (N files flagged across M domains)` when N ≥ 1, listed first. The normal completion output is emitted otherwise.

- **GIVEN** a reorg run whose `warnings[]` flags files in memory domains
- **WHEN** reorg completes
- **THEN** it emits `Next: /docs-distill-memory (N files flagged across M domains)` (listed first) computed from the reused single call, with N/M matching distill's aggregation rule

#### R11: Graceful degradation on reorg's older-binary fallback
On the older-binary fallback path (no `warnings[]` machine surface), the chain line MUST degrade gracefully — a plain pointer without counts (or the normal `Next:` line), alongside the existing upgrade warning.

- **GIVEN** a reorg run on a binary lacking the `warnings[]` machine surface
- **WHEN** reorg completes
- **THEN** the chain line is a plain `/docs-distill-memory` pointer without counts (or the normal completion output), never a fabricated count

### Docs & SPEC-mirror sweep

#### R12: Documented-semantics sweep for the meter change
The old counting rule (sanctioned citations count) MUST be updated everywhere it is restated: `src/kit/skills/_cli-fab.md` § fab memory-index, `docs/specs/skills/SPEC-_cli-fab.md`, and `docs/specs/fkf.md` § Present-truth debt meters. `src/kit/reference/fkf.md` is updated only if it restates the changed counting semantics (verified during apply). `docs/memory/pipeline/schemas.md` is a hydrate-stage update (NOT touched during apply).

- **GIVEN** the meter fix
- **WHEN** apply completes
- **THEN** every apply-scope doc surface restating "sanctioned citations count" reflects the new position-aware rule, and no apply edit touches `docs/memory/`

#### R13: SPEC-mirror class swept in the same change
Every `src/kit/skills/*.md` edit MUST carry its `docs/specs/skills/SPEC-*.md` mirror update in the same change — `SPEC-docs-distill-memory.md`, `SPEC-docs-reorg-memory.md`, `SPEC-_cli-fab.md` — plus the aggregate specs restating these behaviors: `docs/specs/skills.md` (distill + reorg entries) and `docs/specs/glossary.md` (distill "One domain per run").

- **GIVEN** the distill/reorg/_cli-fab skill edits
- **WHEN** apply completes
- **THEN** each skill's SPEC mirror and every aggregate spec restating the changed behavior is updated in the same change

### Non-Goals

- No umbrella command (`/docs-groom-memory`) — user-rejected.
- No merge of `/docs-distill-memory` and `/docs-reorg-memory` — user-rejected.
- No bulk approval across all domains in one prompt — user-rejected (collapses the per-domain human gate).
- No `narrationStems` list expansion (recall is out of scope; the fix targets the false positive only).
- No CLI signature change, no migration (no `fab memory-index` flag/JSON-shape change, no user-data restructuring).

### Design Decisions

#### Position-aware change-id counting (not dropping id-tokens entirely)
**Decision**: The narration-density meter counts change-id tokens only when they appear outside two sanctioned-citation positions — a parenthesized `(id)` citation and an `*Introduced by*:` field line — while still counting change-ids woven into prose.
**Why**: FKF §3.3 sanctions trailing `(change-id)` citations and `*Introduced by*:` provenance as content distillation should KEEP; counting them made fully-distilled files never clear the flag (the recorded caveat). Ids embedded in narration remain a genuine density signal, so they must keep counting.
**Rejected**: Dropping change-id occurrences from the count entirely — it would lose the density signal for ids narrated into prose, which distillation legitimately targets. Also rejected: expanding the stem list — out of scope; the bug is a false positive, not a recall gap.
*Introduced by*: 260719-npoa-distill-loop-all-meter-chain

#### No-arg distill loops all domains, one approval unit per domain
**Decision**: The no-arg default surveys once then iterates every flagged domain sequentially in the main session, keeping the per-domain approval gate as the loop body; explicit `<domain>` stays a single-domain override.
**Why**: Nobody re-invokes per domain, so the corpus never converged under the former per-domain-re-invocation model. "One domain per run" was only ever a property of the approval unit (a human approves per domain seeing per-file diffs), never of the invocation, so looping domains within one invocation loses no safety.
**Rejected**: Bulk approval across all domains in one prompt (collapses the per-domain human gate on load-bearing memory files); per-domain subagent dispatch (the approval prompt is interactive and must reach the user). This supersedes the 260718-ukpf DD "No-arg survey with auto-pick" Rejected (a) entry by user decision (hydrate rewrites that DD to present truth).
*Introduced by*: 260719-npoa-distill-loop-all-meter-chain

## Tasks

### Phase 1: Meter fix (Go prerequisite — lands before/with the skill-loop work)

- [x] T001 Add a position-aware change-id counter to `src/go/fab/internal/memoryindex/memoryindex.go`: count registry-gated change-id tokens that appear OUTSIDE (a) a parenthesized `(change-id)` citation and (b) an `*Introduced by*:` field line; keep `countChangeIDOccurrences` (still used by nothing else — verify) or replace it with the position-aware counter. Update `topicBodyWarnings` (~line 1266) so `markers := countNarrationStems(body) + <position-aware change-id count>`. Update the surrounding doc-comments (~lines 66–70, 112–116, 246, 1247–1269) to state the new rule (sanctioned citations no longer count; ids in allowed positions excluded). <!-- R1 R2 -->
- [x] T002 Update narration-density tests in `src/go/fab/internal/memoryindex/memoryindex_test.go`: fix `TestGather_NarrationDensity_Boundary` (its `body5`/`body4` fixtures rely on `(abcd)`/prose `abcd` counting) to conform to the new rule; add a distilled fixture (only trailing `(id)` + `*Introduced by*:` lines, <5 markers → NO flag), a ≥5-stem fixture (flags), and a mixed fixture (stems + non-allowed-position ids only). <!-- R3 -->
- [x] T003 Update `src/go/fab/cmd/fab/memory_index_test.go` `TestMemoryIndexCmd_CheckJSON_WarningsArrayPopulated` (~line 372): its `supersedes D (abcd) abcd` fixture would drop to 4 markers under the new rule — adjust the body so the narration warning still fires (e.g. add a stem or a non-allowed-position id) without relying on the parenthesized citation counting. <!-- R3 -->
- [x] T004 Run `go test ./internal/memoryindex/... ./cmd/fab/...` from `src/go/fab`; fix failures until green. <!-- R1 R2 R3 -->

### Phase 2: Documented-semantics sweep for the meter change

- [x] T005 Update `src/kit/skills/_cli-fab.md` § fab memory-index narration-marker bullet (~lines 813–816): replace "registry-gated change-id token occurrences (sanctioned citations count too — density is the distillation-debt signal)" with the position-aware rule (stems + change-id tokens outside allowed positions; trailing `(id)` citations and `*Introduced by*:` fields do not count). <!-- R12 -->
- [x] T006 Update `docs/specs/skills/SPEC-_cli-fab.md` (~line 36): the narration-marker parenthetical "(transition stems + registry-gated change-id occurrences, ≥5 — the distillation-debt meter)" → the position-aware phrasing. <!-- R12 R13 -->
- [x] T007 Update `docs/specs/fkf.md` § Present-truth debt meters (~lines 273–276): the narration-marker-density sentence ("plus registry-gated change-id token occurrences in the body …; sanctioned citations count too, because density, not violation, is the signal") → the position-aware rule. <!-- R12 -->
- [x] T008 Verify `src/kit/reference/fkf.md` (~line 105): it mentions "narration density" without the counting detail — confirm it does NOT restate the changed counting semantics; update only if it does. (Verified during intake: it does not — leave unchanged.) <!-- R12 -->

### Phase 3: Distill no-arg loop-all

- [x] T009 Rewrite `src/kit/skills/docs-distill-memory.md` Behavior Step 0 (survey mode) so the no-arg default surveys once (unchanged single call, four-kind aggregation, exclusion set, older-binary fallback) then iterates EVERY flagged domain sequentially in `docs/memory/index.md` domain-table order; the existing one-domain flow (Steps 1–5) is the loop body (full read → per-file report → per-domain approval → apply → regen). Loop runs in the main session (no per-domain subagent dispatch). Encode the skip/no-rewrite/exit-2/terminal-state semantics (R6) and the survey-once-no-re-survey rule. <!-- R4 R5 R6 -->
- [x] T010 Update `src/kit/skills/docs-distill-memory.md` Arguments, Pre-flight, and the explicit-`<domain>` override text so the explicit argument stays the single-domain override (full read, no survey, no loop) and the multiple-explicit-domains abort stays. <!-- R7 -->
- [x] T011 Update `src/kit/skills/docs-distill-memory.md` Output § Dynamic `Next:` line + the trailing `Next:` line (and its HTML comment) so the line reports skipped/remaining domains (surveyed truth) — all-distilled or a skipped-domains list with flagged counts — not per-domain re-invocation. <!-- R8 -->
- [x] T012 Reframe all "one domain per run" language in `src/kit/skills/docs-distill-memory.md` — frontmatter `description:`, Purpose, Arguments, Key Properties, Output, Error Handling — to "one domain per approval/apply unit, iterated within a single invocation". <!-- R9 -->

### Phase 4: Reorg → distill chain

- [x] T013 Add the completion chain to `src/kit/skills/docs-reorg-memory.md`: at completion, reuse the existing single `fab memory-index --check --json` call's `warnings[]` (no second call), aggregate flagged files with distill's survey rule (four kinds, dedupe by path, sub-domain roll-up, exclusion set re-applied), and emit `Next: /docs-distill-memory (N files flagged across M domains)` when N ≥ 1 (listed first). Note the `_preamble` § Next Steps Convention skill-file-wins carve-out (reorg has no prior `Next:` convention line). <!-- R10 -->
- [x] T014 Add graceful degradation for reorg's older-binary fallback path in `src/kit/skills/docs-reorg-memory.md`: omit the counts (plain `/docs-distill-memory` pointer or normal `Next:` line) alongside the existing upgrade warning. <!-- R11 -->

### Phase 5: SPEC-mirror + aggregate-spec sweep for the skill changes

- [x] T015 Update `docs/specs/skills/SPEC-docs-distill-memory.md` to mirror the no-arg loop-all behavior (Summary, Flow, Key properties): no-arg surveys once then loops every flagged domain sequentially in index.md order; per-domain approval retained; "one domain per approval/apply unit, iterated within a single invocation"; dynamic `Next:` reports skipped/remaining. <!-- R13 -->
- [x] T016 Update `docs/specs/skills/SPEC-docs-reorg-memory.md` to mirror the completion chain (`Next: /docs-distill-memory (N files flagged across M domains)` reusing the single call; older-binary graceful degradation) — Summary + Flow + Tools rows. <!-- R13 -->
- [x] T017 Update `docs/specs/skills.md` distill entry (~lines 760–778: no-arg survey/loop behavior + "one domain per run" property + dynamic `Next:`) and reorg entry (~lines 741–756: completion chain) to match the new behavior. <!-- R13 -->
- [x] T018 Update `docs/specs/glossary.md` (~line 58) `/docs-distill-memory` "One domain per run" → the per-approval-unit / loops-all-domains framing. <!-- R13 -->

## Execution Order

Phase 1 (Go meter fix + tests) is a prerequisite and MUST land before or with the Phase 3 skill loop-all work (otherwise the loop full-reads clean domains forever). Phase 2 (meter doc sweep) depends on Phase 1's decided rule. Phases 3–4 (skill changes) and Phase 5 (their SPEC mirrors) travel together — do the skill edit then its mirror. T008 is a verify-only task (expected: no change).

## Acceptance

### Functional Completeness

- [x] A-001 R1: The narration-density meter counts stems plus only change-id tokens outside the two sanctioned positions (parenthesized `(id)` citation; `*Introduced by*:` field line); ids in prose still count.
- [x] A-002 R2: `narrationStems` is unchanged (no list expansion).
- [x] A-003 R3: Tests pin the new counting rule (distilled-only → no flag; ≥5 stems → flag; mixed → stems + non-allowed-position ids), and updated existing tests conform to the spec.
- [x] A-004 R4: No-arg `/docs-distill-memory` surveys once then loops every flagged domain sequentially in index.md domain-table order (main-session loop, one-domain flow as body).
- [x] A-005 R5: The per-domain approval gate (apply all / cherry-pick / skip) is retained; no bulk-approval-across-all-domains prompt is introduced.
- [x] A-006 R6: Skipped domains stay untouched and are reported; a no-rewrite domain reports already-distilled and continues; exit-2 does not swallow remaining domains; terminal state is all-distilled or a skipped/remaining summary.
- [x] A-007 R7: Explicit `<domain>` stays the single-domain override (full read, no survey, no loop); the multiple-explicit-domains abort stays.
- [x] A-008 R8: The dynamic `Next:` line reports skipped/remaining domains (surveyed truth), not per-domain re-invocation.
- [x] A-009 R9: All "one domain per run" language is reframed to the per-approval-unit / single-invocation framing across the skill's `description:`, Purpose, Arguments, Key Properties, Output, Error Handling.
- [x] A-010 R10: `/docs-reorg-memory` emits `Next: /docs-distill-memory (N files flagged across M domains)` (listed first) at completion when N ≥ 1, computed from the reused single `--check --json` call with distill's aggregation rule.
- [x] A-011 R11: On reorg's older-binary fallback path the chain line degrades gracefully (no fabricated counts), alongside the upgrade warning.

### Behavioral Correctness

- [x] A-012 R1: Threshold (5), warning kind name, advisory status, `Count` semantics, and the `warnings[]` JSON shape are unchanged — only the marker-counting rule changed.
- [x] A-013 R4: The loop iterates the initial survey's flagged list (survey once — no re-survey between domains).

### Scenario Coverage

- [x] A-014 R6: A no-arg run where the user skips domain A and domain B is already distilled ends with A untouched/listed-skipped, B reported already-distilled, and an all-distilled-or-remaining terminal summary.

### Removal Verification

- [x] A-015 R12: No doc surface in apply scope still restates "sanctioned citations count"; `docs/memory/pipeline/schemas.md` is left for hydrate (not touched during apply).

### Edge Cases & Error Handling

- [x] A-016 R11: Reorg on a binary lacking the `warnings[]` surface never emits a fabricated `(N files flagged …)` count.

### Documentation Accuracy

- [x] A-017 R12: `src/kit/skills/_cli-fab.md`, `docs/specs/skills/SPEC-_cli-fab.md`, and `docs/specs/fkf.md` restate the position-aware rule; `src/kit/reference/fkf.md` verified (unchanged — does not restate the counting detail).
- [x] A-018 R13: Every `src/kit/skills/*.md` edit carries its `docs/specs/skills/SPEC-*.md` mirror update (distill, reorg, _cli-fab) plus `docs/specs/skills.md` + `docs/specs/glossary.md`.

### Cross References

- [x] A-019 R13: The distill/reorg bidirectional chain reads consistently across the skill files and their SPEC mirrors (distill points at reorg; reorg points at distill).

### Code Quality

- [x] A-020 Pattern consistency: the Go meter change follows existing `internal/memoryindex` patterns (token-scanning via `changeIDTokenSep`/`changeIDTokenID`, doc-comment style, hardcoded-const posture) — no new dependencies, no God functions.
- [x] A-021 No unnecessary duplication: the position-aware counter reuses the existing change-id token helpers rather than reimplementing tokenization; reorg reuses its single `--check --json` call (no second survey call).
- [x] A-022 Canonical source only: skill edits land in `src/kit/skills/*.md`, never `.claude/skills/`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | No-arg `/docs-distill-memory` surveys once then loops every flagged domain sequentially in index.md domain-table order (existing one-domain flow as the loop body); explicit `<domain>` stays the single-domain override | Intake decided by user; verbatim in the change description | S:95 R:85 A:95 D:95 |
| 2 | Certain | Per-domain approval gate retained; bulk approval across all domains rejected | User explicitly rejected bulk approval (collapses the human safeguard) | S:95 R:80 A:95 D:95 |
| 3 | Certain | No umbrella command, no skill merge — reorg→distill composition via `Next:` lines only | User rejected both | S:95 R:90 A:95 D:95 |
| 4 | Certain | The Go meter fix is a prerequisite ordered before/with the skill-loop work in this same change, shipping with tests | Change description states the ordering; constitution mandates test-alongside | S:90 R:80 A:90 D:90 |
| 5 | Confident | Marker decomposition: stem list unchanged; change-id tokens in allowed positions (parenthesized `(id)` citations, `*Introduced by*:` field lines) do not count; change-id tokens outside allowed positions still count | Description names the two allowed positions; "in allowed positions do NOT count" implies positional decomposition, not dropping id-tokens entirely; stem-list expansion out of scope | S:75 R:85 A:80 D:70 |
| 6 | Certain | Threshold (≥5), warning kind name, advisory status, `Count` semantics, and `--check --json` `warnings[]` shape unchanged — only the marker-counting rule changes | Description scopes the fix to counting; no CLI-surface change implied | S:85 R:85 A:90 D:90 |
| 7 | Certain | Documented-semantics sweep covers `_cli-fab.md` § fab memory-index, `SPEC-_cli-fab.md`, `docs/specs/fkf.md` § debt meters; `src/kit/reference/fkf.md` unchanged (verified — does not restate the counting detail); `pipeline/schemas.md` is a hydrate update, not apply | Grep-verified during intake and re-verified during apply | S:90 R:85 A:95 D:90 |
| 8 | Confident | Reorg's chain reuses its existing single `fab memory-index --check --json` call (no second survey), aggregates with distill's rule (four kinds, dedupe by path, sub-domain roll-up, exclusion set), and emits `Next: /docs-distill-memory (N files flagged across M domains)` when N ≥ 1 | Description prefers reuse; mirroring distill's aggregation keeps the two skills' counts consistent | S:75 R:85 A:80 D:65 |
| 9 | Confident | On reorg's older-binary fallback the chain line degrades gracefully — pointer without counts (or the normal `Next:` line), alongside the upgrade warning | Description covers only the machine-surface case; graceful degradation mirrors both skills' older-binary posture | S:60 R:90 A:75 D:60 |
| 10 | Confident | The no-arg loop runs in the main session sequentially (no per-domain subagent dispatch); the superseded-DD context-budget concern is accepted by user decision, and hydrate rewrites that DD | Per-domain approval prompts are interactive and must reach the user; the description specifies a sequential loop with approval gates and no dispatch | S:65 R:80 A:80 D:70 |
| 11 | Certain | The multiple-explicit-domains abort stays; skipped domains stay untouched; the loop iterates the initial survey's flagged list (no re-survey between domains); terminal output reports all-distilled or skipped/remaining; the dynamic `Next:` reports skipped/remaining instead of driving re-invocation | Description states these semantics directly | S:85 R:85 A:90 D:85 |
| 12 | Certain | Skill edits land in canonical `src/kit/skills/` only, with the full SPEC-mirror class swept (`SPEC-docs-distill-memory.md`, `SPEC-docs-reorg-memory.md`, `SPEC-_cli-fab.md`, `docs/specs/skills.md`, `docs/specs/glossary.md`) | Constitution Additional Constraints + code-quality § Sibling & Mirror Sweeps; description restates it | S:95 R:85 A:95 D:95 |

12 assumptions (8 certain, 4 confident, 0 tentative).

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant. (The one symbol it did obsolete, `internal/memoryindex.countChangeIDOccurrences`, was already deleted within this change — replaced by the position-aware `countNonSanctionedChangeIDs`; grep confirms zero remaining references.)
