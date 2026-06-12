# Intake: Skills Twins & Self-Duplication Refactor

**Change**: 260611-szxd-skills-twins-self-duplication-refactor
**Created**: 2026-06-12
**Status**: Draft

## Origin

One-shot invocation: `/fab-new szxd` (backlog ID). Resolved from `fab/backlog.md`:

> [szxd] 2026-06-11: Skills-review batch 4/4 — twins + self-duplication refactor. DEPENDS: do AFTER batch 3 (zc9m) — this extends the same helper model. GOAL: single-source duplicated skill logic; behavior identical unless explicitly flagged. ACTIONS: f031 make fab-draft.md a thin delta over fab-new — body becomes roughly (Read .claude/skills/fab-new/SKILL.md; execute its Steps 0-9 with these deltas: Step 9 tail = change NOT activated, user must /fab-switch; skip Steps 10-11; Output + Next per the Activation Preamble convention; drop the activation/git error rows). Steps 0-9 are currently byte-identical (~120 duplicated lines). Verifier guidance: do NOT move the shared steps into _generation. f007 extract the shared ff/fff pipeline bracket (pre-flight intake gate, context loading, behavior note, Steps 1-3 apply/review/hydrate, auto-rework loop + escalation rule, bail message) into a new _pipeline helper parameterized by driver name (fab-ff vs fab-fff) and terminal stage (hydrate vs review-pr); add _pipeline to the allowed helpers list; fab-ff/fab-fff shrink to arguments + step list + the fff-only ship/review-pr steps. Fold f071 into the extraction: state the rework-cycle choreography explicitly ONCE in the helper — per cycle, exactly which fab status commands fire (fail + reset pair repeats each cycle), re-dispatch apply via Apply Behavior subagent, then a FRESH review subagent. Also f019: add a review-failed dispatch row to fab-continue Step 1 that presents the rework menu directly — the ff/fff exhaustion message points users at /fab-continue for (manual rework options) that its dispatch table currently cannot reach. f032 fab-new Step 11 — compress the inlined 5-case git-branch logic into one condition/command/report table (~15 lines, evaluate in order, first match wins) + a keep-in-sync comment referencing git-branch.md Step 4; verifier confirmed the inline copy is correct for runtime token economy — do NOT delegate to /git-branch at runtime. f087 fab-archive.md — merge its two full document copies: mode detection + both argument lists once at the top; demote the second document (line ~117) to a Restore Mode section holding only its unique Behavior/Output/Error-Handling/Key-Properties content. f094 git-pr.md — resolve change context ONCE in a unified Step 0 ({name},{has_fab},{has_intake},{change_type}) and reference those variables in Steps 0b/1/1b/3c/4a; it currently re-resolves up to 6 times while warning against re-resolution. f098 git-pr-review.md — state the triage taxonomy once (merge Step 4 items 1+3 into one classify-and-assign list; Disposition Reference table is the single reply-format source; cut Rules to the 2 non-restated lines). f049 fab-operator.md — canonical spawn sequence stated once + a 3-row table mapping entry form -> initial command; currently restated 4-5x across Working-a-Change walkthroughs and autopilot steps. f116 fab-operator.md — extract the ~100-line status-frame spec out of tick step 1 into a Status Frame Format subsection; collapse the 4x-repeated render-path rationale (lines ~200,263,265) into one rule. f080 fab-setup.md migrations — delete the triplicated version read/parse/compare (pre-flight checks 1/2/4 at lines 334-337, Step 1 at 339-343, Semver Comparison at 460-462); run migrations-status once and branch on its local/engine fields. f077 fab-setup bootstrap — move fab sync to immediately after Phase 0 and delete steps 1c-1g/1i/1k that hand-duplicate sync scaffolding; explicit behavior-ORDER change with identical outcome via idempotency — flag it in the PR. DO NOT TOUCH: docs-reorg-memory vs docs-reorg-specs divergence — adversarially verified as justified (f197); parallel maintenance is sustainable there; at most port the 3 small pieces listed in f107 (Kind column, Link Impact note, no-dangling-link verify) specs-ward. CONSTRAINTS: SPEC mirror updates for every touched skill; the _pipeline helper + fab-draft delta change how agents read skills — after fab sync, dry-run /fab-draft and /fab-ff on a scratch change to verify the indirection holds; src/kit is canonical. REPORT: docs/specs/findings/skills-review-2026-06-11.md findings f031/f007/f071/f019/f032/f087/f094/f098/f049/f116/f080/f077/f197 (line numbers vs commit ae79e04c).

No prior discussion exists in this session. All decisions below are sourced from the backlog entry itself and the adversarially-verified findings report `docs/specs/findings/skills-review-2026-06-11.md` (per-finding verifier notes, line numbers vs commit ae79e04c). This is the final batch (4/4) of the skills-review remediation: batch 1 (9u91, PR #390), batch 2 (uliv, PR #391), batch 3 (zc9m, PR #392) are shipped but unmerged; this branch is main + a cherry-pick of the zc9m helper-model dependency (commit c6f80118).

## Why

1. **The pain point**: Skill logic is duplicated at two scales. *Twins*: `fab-draft.md` repeats `fab-new.md` Steps 0–9 byte-identically (~120 lines; 169 vs 256 lines total), and `fab-ff.md`/`fab-fff.md` share ~88% of their content (107 of 136 lines verbatim, most remaining diffs are driver-name token swaps). *Self-duplication*: single skills restate their own rules — `fab-archive.md` contains two full top-level documents, `git-pr.md` re-resolves the active change at up to six step sites (11 textual mentions) while telling the agent to reuse a single resolution, `git-pr-review.md` states its triage taxonomy three times, `fab-operator.md` restates the spawn sequence 4–5x and the render-path rationale 4x, `fab-setup.md` triplicates migration version handling and hand-duplicates `fab sync` scaffolding across seven bootstrap steps.

2. **The consequence of inaction**: Every logical edit must land 2+ times, and drift is no longer hypothetical: fab-ff says "Two gates" while fab-fff says "single intake confidence gate" (contradicting the constitution's sole-gate framing); fab-ff's post-bail `/fab-clarify` guidance is absent from fab-fff; fab-new's inline branch logic has dropped git-branch's STOP markers. Worse, two specified behaviors are currently unreachable or ambiguous: the ff/fff exhaustion message points users at `/fab-continue` for "manual rework options" its dispatch table has no row for (f019), and the rework-cycle choreography is under-specified enough that two conforming implementations leave different `.status.yaml` histories (f071 — `stage_metrics.review.iterations` feeds PR meta, so the divergence is observable).

3. **Why this approach**: Single-source each duplicated block using the opt-in helper model batch 3 (zc9m) introduced — `_pipeline` follows the `_generation`/`_review`/`_srad` precedent — plus a thin-delta skill for the fab-new/fab-draft twin and in-file consolidation for the self-duplicating skills. Behavior stays identical except one explicitly flagged ordering change (f077). Alternatives were adversarially rejected in the findings report: moving the shared Steps 0–9 into `_generation` (refuted — fab-continue/ff/fff also load it and would pay the context tax), delegating fab-new Step 11 to `/git-branch` at runtime (refuted — inline copy wins on runtime token economy), and merging the docs-reorg twins (f197 — divergence is justified; parallel maintenance is sustainable there).

## What Changes

All edits land in `src/kit/skills/` (canonical source); `.claude/skills/` is refreshed only via `fab sync`. Every touched skill gets its `docs/specs/skills/SPEC-*.md` mirror updated (constitution constraint). Finding IDs reference `docs/specs/findings/skills-review-2026-06-11.md`.

### 1. fab-draft becomes a thin delta over fab-new (f031)

`fab-draft.md`'s body is replaced with a delta instruction roughly: *"Read `.claude/skills/fab-new/SKILL.md`; execute its Steps 0–9 with these deltas: Step 9 tail = change NOT activated, user must `/fab-switch`; **skip Steps 10–11** (no activation, no git branch); Output + Next per the Activation Preamble convention (`_preamble.md` § Activation Preamble names /fab-draft); drop the activation/git error rows."* The skip-Steps-10–11 instruction must be explicit and prominent — the known risk of the delta form is an agent running activation by momentum. Precedent for cross-skill SKILL.md reads exists (fab-proceed reads other skill files), and `SPEC-fab-draft.md` already describes fab-draft as "identical to /fab-new through Step 9". Do NOT move the shared steps into `_generation` (verifier guidance: fab-continue/ff/fff also declare it).

### 2. New `_pipeline` helper: shared ff/fff bracket (f007) + explicit rework choreography (f071)

Create `src/kit/skills/_pipeline.md` containing the shared pipeline bracket extracted from fab-ff/fab-fff: pre-flight intake gate (`fab score --check-gate --stage intake`), context loading, behavior note, Steps 1–3 (apply → review → hydrate), the auto-rework loop + escalation rule, and the bail message. Parameterize by **driver name** (`fab-ff` vs `fab-fff`) and **terminal stage** (`hydrate` vs `review-pr`). Add `_pipeline` to `_preamble.md`'s allowed `helpers:` values (currently `_generation, _review, _cli-fab, _cli-external, _srad`). `fab-ff.md`/`fab-fff.md` shrink to arguments + step list + the fff-only ship/review-pr steps.

Fold f071 into the extraction — state the rework-cycle choreography explicitly ONCE in the helper: per cycle, exactly which `fab status` commands fire (the fail + reset pair repeats on **every** failed re-review, not just the first), re-dispatch apply via the `/fab-continue` Apply Behavior subagent, then dispatch a **fresh** review subagent. While extracting, resolve the documented drift in the bracket: unify gate terminology to the constitution's single-intake-gate framing, and keep fab-ff's post-bail `/fab-clarify` guidance in the shared text so fff regains it. `_review.md`'s stale pointer to the rework loop ("Step 3" — it is Step 2) should be corrected to point at the helper.

### 3. fab-continue gains a review-failed dispatch row (f019)

Add a row to `/fab-continue`'s Step 1 dispatch table for `progress.review == failed` that presents the existing rework menu (already defined in fab-continue) directly. This makes the ff/fff exhaustion guidance truthful — `_preamble`'s state table and the templates spec already treat review=failed as a resting state whose next action is the rework menu, but fab-continue's dispatch currently only handles `pending`/`active`/`ready`. Reword the ff/fff stop message (now in `_pipeline`) to describe what `/fab-continue` will actually do.

### 4. fab-new Step 11 compresses to a single table (f032)

Replace the inlined 5-case branch logic (~56 lines) with one condition/command/report table (~15 lines), annotated "evaluate in order, first match wins", plus a keep-in-sync comment referencing `git-branch.md` Step 4. The five cases (already-on-target / target-exists / on-main / local-only-branch rename / pushed-branch new) and their report strings are preserved verbatim — this is compression, not behavior change. Keep it inline: verifier confirmed the inline copy is correct for runtime token economy; do NOT delegate to `/git-branch` at runtime. `git-branch.md` itself is untouched.

### 5. fab-archive single-document merge (f087)

`fab-archive.md` currently holds two `#`-level documents (lines ~6 and ~117). Merge: mode detection + both argument lists stated once at the top; demote the second document to a `## Restore Mode` section holding only its unique Behavior/Output/Error-Handling/Key-Properties content. **Verifier caveat**: the restore-mode Pre-flight is NOT duplicated boilerplate — it uniquely waives the preflight/hydrate-guard (opposite of archive mode) and must be preserved as mode-specific content.

### 6. git-pr unified Step 0 resolution (f094)

Resolve change context ONCE in a unified Step 0 producing `{name}`, `{has_fab}`, `{has_intake}`, `{change_type}`; Steps 0b/1/1b/3c/4a reference those variables instead of re-running `fab change resolve`. Keep the Step 0b and Step 3c step names intact — `_cli-fab.md` and `prmeta.go` cite them by name.

### 7. git-pr-review triage taxonomy stated once (f098)

Merge Step 4 items 1 and 3 into one classify-and-assign list (keep the examples); the Disposition Reference table becomes the single reply-format source (drop reply formats from Step 5.5 item 1, preserving the 7-char-SHA + description detail); cut the Rules section to the two non-restated lines (fully autonomous; targeted changes only), keeping the general fail-fast line which has no other general statement.

### 8. fab-operator: spawn sequence once + Status Frame Format subsection (f049, f116)

- **f049**: Keep the canonical 6-step spawn sequence stated once (§6); replace the three Working-a-Change walkthroughs with a 3-row table mapping entry form → initial command (`/fab-switch <change> && /fab-proceed`, `/fab-new <escaped-text>`, `/fab-new <id>`) + "run the §6 spawn sequence". Autopilot steps 1–2 and Watches step 4 become one-line references. Preserve the variant-specific extras: shell-escaping note, idea-lookup pre-step, `--reuse`, and watch-enrollment extras.
- **f116**: Extract the ~74-line status-frame spec out of tick step 1 into a `Status Frame Format` subsection after Tick Behavior (step 1 ends "emit the status frame — see Status Frame Format"). Collapse the 4x-repeated render-path rationale into one rule: "Emit bare markdown (no code fence, no headings, no ANSI); channels: tables, emoji, bold, italic, code spans, plain URLs." Keep the runtime no-fence rule (agent-critical, distinct), the frame example, and the two column tables.

### 9. fab-setup migrations: version handling single-sourced to the binary (f080)

Delete the triplicated version read/parse/compare — pre-flight checks 1/2/4 (~lines 334–337), Step 1 (~339–343), and the Semver Comparison section (~460–462) — and also drop the corresponding Context Loading item. Run `fab migrations-status` once and branch on its returned `local`/`engine` fields (one-line rule). The binary already owns discovery and exits non-zero with remediation hints on missing version files.

### 10. fab-setup bootstrap: sync-first reorder (f077) — **flagged behavior-ORDER change**

Move `fab sync` (currently step 1j) to immediately after Phase 0 and delete steps 1c–1g, 1i, and 1k, which hand-duplicate sync's scaffolding (`scaffoldTreeWalk` copy-if-absent installs context/code-quality/code-review/index files; `scaffoldDirectories` creates `fab/changes/` + archive + `.gitkeep`; the `.gitignore` fragment merge covers `.fab-*`, subsuming 1k). Rewrite Bootstrap Output accordingly. **Verifier caveats**: `fab sync` requires `config.yaml` `fab_version` to exist, so sync must stay after Phase 0's interactive config creation (1a/1b); add a sync-failure guard; renumber step 1h's "step 1j" reference. Outcome is identical via idempotency, but this is the one explicit behavior-ORDER change in the batch — **it must be flagged in the PR description**.

### Out of Scope

- **docs-reorg twins (f197)**: DO NOT TOUCH the docs-reorg-memory vs docs-reorg-specs divergence — adversarially verified as justified; parallel maintenance is sustainable there.
- **f107 specs-ward port** (Kind column, Link Impact note, no-dangling-link verify into docs-reorg-specs): excluded. The backlog's "at most port" reads as a cap on the DO-NOT-TOUCH zone, not a directive; the ACTIONS list is the definitive work list. Trivially addable later if wanted.
- **Pre-Go-CLI staleness residue** (docs/specs naming/architecture/glossary/user-flow/change-types + srad autonomy-table residue, memory execution-skills/templates script names — recorded in the uliv plan's Non-Goals): left for a later docs sweep; theme mismatch with this duplication batch.

## Affected Memory

- `pipeline/planning-skills`: (modify) fab-draft thin-delta model over fab-new Steps 0–9; fab-new Step 11 single-table branch logic
- `pipeline/execution-skills`: (modify) `_pipeline` helper (shared ff/fff bracket + once-stated rework choreography), fab-continue review-failed dispatch row, git-pr unified Step 0 resolution, git-pr-review single-source triage, fab-archive single-document merge
- `_shared/context-loading`: (modify) `_pipeline` added to the opt-in helper allowlist
- `runtime/operator`: (modify) spawn sequence stated once + entry-form table; Status Frame Format subsection + collapsed render-path rule
- `distribution/setup`: (modify) bootstrap sync-first reorder (1c–1g/1i/1k deleted), migrations version handling delegated to `fab migrations-status`

## Impact

- **Skill sources** (12 files in `src/kit/skills/`): `fab-draft.md` (rewrite as delta), `_pipeline.md` (new), `fab-ff.md` + `fab-fff.md` (shrink to wrappers), `fab-continue.md` (dispatch row), `fab-new.md` (Step 11 table), `fab-archive.md` (merge), `git-pr.md` (Step 0), `git-pr-review.md` (taxonomy), `fab-operator.md` (f049 + f116), `fab-setup.md` (f080 + f077), `_preamble.md` (allowlist line), `_review.md` (stale rework-loop pointer).
- **SPEC mirrors** (`docs/specs/skills/`): one per touched skill + new `SPEC-_pipeline.md` (underscore helpers carry mirrors — `SPEC-_srad.md` precedent; only `_cli-fab`/`_cli-external` are excluded).
- **No Go code changes** — pure markdown/skill refactor; the `fab` CLI surface is unchanged.
- **Read-path risk**: the `_pipeline` helper and fab-draft delta change how agents *read* skills. Verification: after `fab sync`, dry-run `/fab-draft` and `/fab-ff` on a scratch change to confirm the indirection holds (draft creates without activating; ff resolves the helper and runs the bracket), then discard the scratch change.
- **Branch context**: diff overlaps unmerged PRs #390/#392 until they merge (this branch = main + zc9m cherry-pick); merge order remains 390 → 391 → 392 → this.

## Open Questions

*None — the backlog entry carries per-finding instructions, explicit verifier guidance, and an adversarially-verified findings report; no Unresolved decisions remain.*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope = the 12 enumerated ACTIONS findings (f031, f007 with f071 folded in, f019, f032, f087, f094, f098, f049, f116, f080, f077) | Backlog enumerates per-finding instructions from an adversarially-verified report | S:95 R:90 A:95 D:95 |
| 2 | Certain | fab-draft = thin delta over fab-new Steps 0–9; shared steps NOT moved into `_generation` | Explicit verifier guidance — fab-continue/ff/fff also load `_generation` and would pay the context tax | S:95 R:75 A:90 D:95 |
| 3 | Certain | New `_pipeline` helper parameterized by driver name (fab-ff/fab-fff) and terminal stage (hydrate/review-pr); added to `_preamble` allowed-helpers list | Backlog states the shape verbatim; `_generation`/`_review`/`_srad` precedent | S:90 R:70 A:90 D:90 |
| 4 | Certain | Rework choreography stated once in `_pipeline`: per cycle the fail+reset status pair repeats, apply re-dispatched via Apply Behavior subagent, then a FRESH review subagent | Backlog folds f071 into the extraction with exact wording | S:90 R:75 A:85 D:90 |
| 5 | Certain | fab-continue Step 1 gains a review-failed dispatch row presenting the existing rework menu directly | Backlog explicit; `_preamble` state table + templates spec already treat review=failed as a resting state | S:90 R:80 A:85 D:90 |
| 6 | Certain | fab-new Step 11 stays inline, compressed to a ~15-line first-match-wins table + keep-in-sync comment referencing git-branch.md Step 4 | Verifier confirmed inline wins on runtime token economy; no runtime /git-branch delegation | S:95 R:85 A:90 D:95 |
| 7 | Certain | docs-reorg-memory vs docs-reorg-specs divergence untouched | Backlog DO-NOT-TOUCH; f197 adversarially refuted a merge | S:95 R:90 A:95 D:95 |
| 8 | Certain | Behavior identical everywhere except the one flagged behavior-ORDER change (f077 sync-first bootstrap), which the PR description must call out | Backlog GOAL line + explicit f077 flag instruction | S:90 R:70 A:90 D:90 |
| 9 | Certain | SPEC mirror updated for every touched skill, including a new `SPEC-_pipeline.md` | Constitution constraint + backlog CONSTRAINTS; underscore-helper mirrors exist (SPEC-_srad.md et al.) | S:95 R:90 A:95 D:95 |
| 10 | Certain | All edits in `src/kit/skills/` (canonical); `.claude/skills/` refreshed only via `fab sync` | Constitution + project context.md state this explicitly | S:95 R:95 A:95 D:95 |
| 11 | Certain | fab-ff/fab-fff shrink to arguments + step list + fff-only ship/review-pr steps | Backlog explicit | S:90 R:75 A:90 D:90 |
| 12 | Certain | change_type = refactor | Title keyword "refactor"; hook word-boundary inference concurs; explicit set-change-type as backstop | S:90 R:95 A:95 D:95 |
| 13 | Confident | Post-sync verification = dry-run `/fab-draft` + `/fab-ff` on a scratch change, then discard the scratch change | Backlog mandates the dry-run; scratch-change mechanics (creation/cleanup) are interpretation | S:80 R:85 A:80 D:80 |
| 14 | Confident | Findings-report verifier caveats honored: f087 restore pre-flight preserved as mode-specific; f077 sync stays after Phase 0 + failure guard + 1h renumber; f080 also drops the Context Loading item; f098 keeps 7-char-SHA detail + fail-fast line; f094 keeps Step 0b/3c names for `_cli-fab`/prmeta cross-refs; f116 keeps no-fence rule + example + column tables; f049 preserves shell-escaping/idea-lookup/`--reuse`/watch-enrollment extras | High-confidence verifier notes in the report; same provenance as backlog but not restated in it | S:75 R:75 A:85 D:80 |
| 15 | Confident | Pre-Go-CLI staleness residue (uliv Non-Goals list) stays excluded | "Batch 4 or a later docs sweep" — ACTIONS omit it; theme mismatch (staleness vs duplication) | S:60 R:85 A:70 D:65 |
| 16 | Confident | f107 specs-ward port excluded | "At most port" caps the DO-NOT-TOUCH zone rather than directing work; ACTIONS list is the work list; trivially addable later | S:65 R:90 A:70 D:70 |

16 assumptions (12 certain, 4 confident, 0 tentative, 0 unresolved).
