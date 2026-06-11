# Intake: Preamble Context Diet — Skills Review Batch 3/4

**Change**: 260611-zc9m-preamble-context-diet
**Created**: 2026-06-11
**Status**: Draft

## Origin

User invoked `/fab-new zc9m` (one-shot, backlog ID). Backlog entry `[zc9m]` in `fab/backlog.md` (2026-06-11):

> Skills-review batch 3/4 — _preamble.md context diet. CONTEXT: the preamble (32.3KB) is always-loaded by every skill; measured per-invocation context is 36.7KB-134.6KB and the preamble is 2-26x the body of the skill being run (finding f004 has the full ranked byte table). GOAL: cut what every invocation pays for, zero semantic loss. ACTIONS: f003 extract the SRAD Autonomy Framework + Worked Examples + Artifact Markers + Assumptions Summary block (_preamble.md lines 371-481, ~7.7KB = 24% of the preamble) into a new _srad helper; add _srad to the allowed helpers list (line 103); declare it in helpers: of the 6 planning skills (fab-new, fab-draft, fab-continue, fab-ff, fab-fff, fab-clarify — note fab-clarify currently has NO helpers: frontmatter, add it); leave a 3-line pointer in the preamble; update the internal-skill-optimize.md:33,48 pointers that say SRAD lives in _preamble; optionally compress Worked Examples 1-3 (lines 415-437) to the one-liner style of examples 2/3. f042 move the Confidence Scoring Formula/Schema/Template internals into _cli-fab fab score section; keep only Gate Threshold + Invocation in the preamble. f043 cut the Bulk Confirm subsection to one pointer sentence — fab-clarify.md:56-133 is the authority. f041 move the dormant [AUTO-MODE] Skill Invocation Protocol into fab-clarify.md (its sole referencer) or delete both dormant halves — deleting the fab-clarify Auto Mode is a behavior decision, ask the user. f040 move Operator Spawning Rules (preamble lines 152-173) into the _cli-external wt section; fab-operator section 6 remains the normative procedure. f001 make preamble section 1 descriptive, not exhaustive — add (unless the skill itself states otherwise in its Context Loading section); scope the Next:-line MUST (line 265) to pipeline-state skills or list exempt skills; reconcile the fab-switch contradiction (preamble:34 says it loads config.yaml, fab-switch.md:110/123 says config is not required). f117 trim fab-operator section 2 Context Loading to config/constitution/context only and add fab-operator to the preamble section 1 exception list — it loads code-quality/code-review/both doc indexes it never uses, against its own Context discipline principle. f122 fab-continue — move _generation/_review loading from frontmatter helpers to per-stage read instructions inside Apply Behavior / Review Behavior (saves 8.7-19.2KB when the current stage needs neither); IMPORTANT: do NOT apply this to fab-ff/fab-fff — the equivalent finding f074 was REFUTED because their rework loop has the orchestrator itself editing plan.md Requirements/Tasks/Acceptance, which genuinely needs _generation at orchestrator level. f046 fab-proceed.md:132-139 + fab-discuss.md context list — replace verbatim restated context-file lists with pointers to the preamble sections (the pattern _review.md:35 already uses). QUANTIFIED WIN: roughly 12KB off every planning-skill invocation and 5KB off the other ~14 skills, every single time. CONSTRAINTS: constitution.md:31 hard-codes _cli-fab.md by name — extending the helper model may warrant a dated governance note in the constitution; sync.go listSkills auto-deploys any new .md file (verified — no Go change needed for _srad); update SPEC mirrors including docs/specs/skills.md context-loading description; after the change re-run the wc -c measurement from finding f004 and record before/after in the PR. DEPENDS: do this BEFORE batch 4 (the twins refactor extends the same helper model). REPORT: /home/sahil/code/sahil87/fab-kit/docs/specs/findings/skills-review-2026-06-11.md findings f003/f004/f040/f041/f042/f043/f001/f117/f122/f046 (line numbers vs commit ae79e04c).

All ten findings were adversarially verified at high confidence in the findings report (`docs/specs/findings/skills-review-2026-06-11.md`). Line numbers throughout this intake are **vs commit ae79e04c** — re-locate by content, not line number, since batches 1 (PR #390) and 2 (PR #391) touch the same files.

## Why

1. **Pain point**: `_preamble.md` (32,260 bytes) is loaded by ~24 of 29 skills on every invocation. Finding f004 measured total per-invocation context at 36.7KB–134.6KB; the preamble alone is 2–26x the body of the skill being run. Concretely: `fab-discuss` has a 3KB body but pays 48.4KB per invocation; `fab-operator` tops out at 134.6KB and re-pays the full layer after every `/clear`. Roughly a third of the preamble (SRAD framework ~7.7KB, confidence-scoring internals ~61 lines, dormant [AUTO-MODE] protocol ~25 lines, operator spawning rules ~22 lines, bulk-confirm duplicate ~7 lines) serves only a small subset of skills — or no live skill at all.
2. **Consequence of not fixing**: every skill invocation in every fab-kit project pays this tax forever — slower loads, higher token cost, and a context window crowded with irrelevant framework text. Duplicated copies (bulk-confirm trigger, operator spawning rules, restated context lists) have already drifted once (`Step 1.5` references in memory docs, the fab-switch config contradiction) and will keep drifting. Batch 4 (`szxd`, the twins refactor) depends on the helper model extended here, so deferral blocks the rest of the series.
3. **Why this approach**: the helper mechanism (`helpers:` frontmatter, preamble § Skill Helper Declaration) already exists and `sync.go listSkills` auto-deploys any new `.md` — so moving consumer-specific content out of the always-load layer into opt-in helpers achieves the reduction with **zero semantic loss** (content moves, it doesn't disappear) and no Go changes. Alternatives rejected: deleting content outright (loses behavior — only the dormant [AUTO-MODE]/Auto-Mode pair is even a candidate, and that's surfaced to the user); shrinking via prose compression alone (saves far less, leaves the wrong-audience placement problem).

## What Changes

All edits target canonical sources in `src/kit/skills/` (never `.claude/skills/` deployed copies). Quantified win: ~12KB off every planning-skill invocation and ~5KB off the other ~14 skills.

### f003 — Extract SRAD into a new `_srad` helper

- Cut `_preamble.md` lines 371–481 (`## SRAD Autonomy Framework` through the end of `### Assumptions Summary Block` — scoring dimensions, aggregation formula, confidence grades, Critical Rule, skill-specific autonomy levels table, Worked Examples 1–3, Artifact Markers, Assumptions Summary block; ~7.7KB = 24% of the preamble) into a new `src/kit/skills/_srad.md` helper with standard internal-helper frontmatter (`user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`), mirroring `_generation.md`'s header style.
- Leave a ~3-line pointer in `_preamble.md` where the section was (what SRAD is, that planning skills load `_srad`, where it lives).
- Add `_srad` to the allowed `helpers:` values list (preamble line 103: currently `_generation`, `_review`, `_cli-fab`, `_cli-external`).
- Declare `_srad` in the `helpers:` frontmatter of the 6 planning skills: `fab-new`, `fab-draft`, `fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify`. Note: `fab-clarify` currently has NO `helpers:` key — add it.
- Update `internal-skill-optimize.md:33,48` pointers that say SRAD lives in `_preamble`.
- Compress Worked Examples 1–3 (lines 415–437) to the one-liner style of examples 2/3 (Example 1's full scoring table becomes a one-liner). <!-- assumed: backlog marks this "optionally" — doing it, consistent with the diet goal -->
- No Go change: `sync.go listSkills` auto-deploys any new `.md` (verified in f003).

### f042 — Move confidence-scoring internals to `_cli-fab`

- From `_preamble.md` § Confidence Scoring, move the `.status.yaml` schema block (lines ~490–499), the score formula (lines ~503–512), and the Template subsection (lines ~534–536) into `_cli-fab.md`'s `fab score` (extended) section.
- Keep in the preamble only **Gate Threshold** (flat 3.0, single intake gate, `--check-gate`) and **Invocation** (who scores, when) — agents never compute the score; `fab score` does (Go: `score.go`).

### f043 — Bulk Confirm becomes a one-sentence pointer

- Replace `_preamble.md` § Bulk Confirm (lines ~538–544) with one sentence: `/fab-clarify` offers a bulk-confirm flow for Confident assumptions — defined in `fab-clarify.md` (Step 2, Suggest Mode).
- `fab-clarify.md:56–133` is already the authoritative definition; the preamble copy duplicates its trigger (`confident >= 3 and confident > tentative + unresolved`), upgrade semantics (S → 95), and internal step numbering verbatim.

### f041 — Dormant [AUTO-MODE] protocol moves to fab-clarify.md

- `_preamble.md:310–334` defines the [AUTO-MODE] Skill Invocation Protocol; line 325 admits no skill uses it (auto-clarify removed in 1.10.0). Sole referencer: `fab-clarify`'s Auto Mode, itself "retained for future use".
- **Resolved (asked)**: move the protocol definition into `fab-clarify.md` (its sole referencer) and leave a 2-line pointer in the preamble. fab-clarify's Auto Mode is retained — zero behavior change. The alternative (deleting both dormant halves) was rejected by the user. <!-- clarified: user chose move-over-delete — preserves Auto Mode behavior, consistent with zero-semantic-loss goal -->
- Also: fix the [AUTO-MODE] mention in the live § Subagent Dispatch section (preamble line ~348), and update `glossary.md:113` + SPEC mirrors.

### f040 — Operator Spawning Rules move to `_cli-external`

- Move `_preamble.md` § Operator Spawning Rules (lines 152–173, ~22 always-load lines) into `_cli-external.md`'s `wt` section. Only `fab-operator` declares `helpers: [_cli-external]`, so only it pays.
- `fab-operator.md` §6 remains the normative step-by-step spawn procedure.
- While merging: reduce `_cli-external` to tool syntax plus ONE repo-targeting note (drop the duplicate `fab spawn-command --repo` rule at its tmux bullet, line ~107).

### f001 — Preamble contract becomes descriptive, not exhaustive

- § 1 Always Load: add "unless the skill's own Context Loading section says otherwise" so the ~10 self-exempting skills (docs-reorg-*, docs-hydrate-specs, git-branch, fab-archive, etc.) stop contradicting it.
- Scope the Next:-line MUST (line 265) to pipeline-state skills, or list the exempt skills (git-pr, git-pr-review, git-branch, fab-discuss, fab-operator currently violate it).
- Reconcile the fab-switch contradiction in favor of the skill file: preamble:34 says fab-switch "loads only config.yaml"; `fab-switch.md:110/123` says config is not required.

### f117 — Trim fab-operator's context loading

- `fab-operator.md` §2 currently loads all 7 always-load files; code-quality.md, code-review.md, and both doc indexes are used nowhere in its 646 lines, against its own §1 "Context discipline" principle (and re-paid after every `/clear`).
- Trim §2 to `config.yaml`, `constitution.md`, `context.md` only; add `fab-operator` to the preamble §1 exception list (alongside fab-setup, fab-status, docs-hydrate-memory). Deliberate behavior change, verifier-endorsed.

### f122 — fab-continue loads helpers per stage

- Remove `_generation`/`_review` from `fab-continue.md`'s frontmatter `helpers:`; add explicit read instructions at the point of use: read `_generation` at apply entry when `plan.md` needs generating (must also cover the rare intake-active regeneration path, fab-continue.md:50), read `_review` when entering Review Behavior. Saves 8.7–19.2KB on hydrate/ship/review-pr invocations and apply-resumes.
- Extend preamble § Skill Helper Declaration semantics to permit stage-conditional loading so the frontmatter contract stays honest.
- **Do NOT apply to fab-ff/fab-fff** — equivalent finding f074 was REFUTED: their auto-rework loop has the orchestrator itself editing plan.md Requirements/Tasks/Acceptance, which genuinely needs `_generation` at orchestrator level.

### f046 — Replace restated context lists with pointers

- `fab-proceed.md:132–139` (verbatim copy of the Standard Subagent Context 5-file list) → "include the standard subagent context files per `_preamble.md` § Standard Subagent Context" (pattern `_review.md:35` already uses).
- `fab-discuss.md:28–34` (verbatim copy of the 7-file always-load list) → "Load the always-load layer per `_preamble.md` §1", keeping only the do-not-run-preflight deltas.

### Governance, mirrors, measurement

- Constitution: add a dated explanatory comment (j6cs precedent, constitution.md:40–43 style) noting the helper-model extension (`_srad`, stage-conditional loading); constitution.md:31 hard-codes `_cli-fab.md` by name but is not violated. No new normative MUST rule.
- Update SPEC mirrors for every touched skill (`docs/specs/skills/SPEC-*.md`), the `docs/specs/skills.md` context-loading description, and check `docs/specs/srad.md` (carries the score formula) for consistency.
- After the change: re-run the `wc -c` measurement from finding f004 and record before/after totals in the PR description.

## Affected Memory

- `_shared/context-loading`: (modify) helper model gains `_srad` + stage-conditional loading; always-load contract becomes descriptive with exceptions (fab-operator added); Next:-line convention scoped
- `pipeline/planning-skills`: (modify) SRAD framework now lives in `_srad` helper, declared by the 6 planning skills
- `pipeline/clarify`: (modify) bulk-confirm authority consolidated to fab-clarify; [AUTO-MODE]/Auto-Mode outcome per user decision
- `pipeline/execution-skills`: (modify) fab-continue's per-stage helper loading
- `runtime/operator`: (modify) trimmed §2 context loading; spawning rules relocated to `_cli-external`

## Impact

- **Skill sources** (`src/kit/skills/`): `_preamble.md` (major), new `_srad.md`, `_cli-fab.md`, `_cli-external.md`, `fab-new.md`, `fab-draft.md`, `fab-continue.md`, `fab-ff.md`, `fab-fff.md`, `fab-clarify.md`, `fab-operator.md`, `fab-proceed.md`, `fab-discuss.md`, `fab-switch.md` (if reconcile touches it), `internal-skill-optimize.md`
- **Specs**: `docs/specs/skills.md`, `docs/specs/skills/SPEC-*.md` mirrors for touched skills, `docs/specs/glossary.md` ([AUTO-MODE] entry), `docs/specs/srad.md` (formula consistency)
- **Governance**: `fab/project/constitution.md` dated comment
- **Go**: none (`sync.go listSkills` auto-deploys `_srad.md` — verified)
- **Sequencing/conflict risk**: must land BEFORE batch 4 (`szxd`, extends the same helper model). Batches 1 (PR #390) and 2 (PR #391) are unmerged and touch the same skill files — expect rebases; findings line numbers are vs ae79e04c, so locate edits by content.
- **Validation**: `fab sync` then spot-check a planning skill resolves `_srad`; re-run f004 byte measurement.

## Open Questions

- None. The single open question (f041 move-vs-delete) was asked at intake and resolved — see Assumptions #6.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Extract SRAD block (preamble 371–481) into new `_srad` helper, declared by the 6 planning skills, 3-line pointer left behind | Backlog mandates exactly; f003 verifier confirmed consumers and line range | S:95 R:80 A:90 D:95 |
| 2 | Certain | No Go change for `_srad` deployment | f003 verifier: `sync.go listSkills` auto-deploys any new `.md`; helpers allowlist is prose-only | S:95 R:90 A:95 D:95 |
| 3 | Confident | Compress Worked Examples 1–3 to one-liner style | Backlog marks it "optionally"; consistent with diet goal; examples 2/3 are already one-liners; SRAD semantics preserved | S:75 R:90 A:85 D:70 |
| 4 | Certain | Move scoring formula/schema/template into `_cli-fab` § fab score; keep Gate Threshold + Invocation in preamble | Backlog mandates; f042 verifier confirmed agents never compute the score (Go owns it) | S:90 R:80 A:90 D:90 |
| 5 | Certain | Cut Bulk Confirm to a one-sentence pointer; fab-clarify.md Step 2 is sole authority | Backlog mandates; f043 verifier confirmed verbatim duplication | S:90 R:85 A:90 D:90 |
| 6 | Certain | f041: move [AUTO-MODE] protocol into fab-clarify.md (2-line pointer in preamble); Auto Mode retained | Asked — user chose move over delete; zero behavior change | S:95 R:85 A:90 D:90 |
| 7 | Certain | Move Operator Spawning Rules into `_cli-external` wt section; fab-operator §6 stays normative; drop the duplicate repo-targeting note | Backlog mandates; f040 verifier confirmed only fab-operator loads `_cli-external` | S:90 R:85 A:90 D:90 |
| 8 | Confident | f001 reconciliation direction: preamble §1 descriptive with skill-file override; Next:-line MUST scoped to pipeline-state skills; fab-switch's own file wins the config contradiction | Backlog offers "scope or list exempt" — scoping is simpler and self-maintaining; skill-file-wins matches the descriptive-contract direction | S:70 R:75 A:75 D:65 |
| 9 | Certain | Trim fab-operator §2 to config/constitution/context; add fab-operator to §1 exception list | Backlog mandates; f117 verifier endorsed the deliberate behavior change | S:90 R:80 A:85 D:90 |
| 10 | Certain | fab-continue: per-stage `_generation`/`_review` loading (covering intake-active regeneration path); do NOT touch fab-ff/fab-fff helpers | Backlog mandates with explicit f074-REFUTED guard | S:90 R:75 A:85 D:90 |
| 11 | Certain | Replace restated context lists in fab-proceed/fab-discuss with preamble pointers | Backlog mandates; `_review.md:35` proves the pattern | S:90 R:90 A:90 D:90 |
| 12 | Confident | Add a dated explanatory comment to the constitution for the helper-model extension; no new MUST rule | Backlog says "may warrant"; j6cs precedent shows the dated-comment form | S:65 R:85 A:80 D:70 |
| 13 | Certain | Update SPEC mirrors, docs/specs/skills.md, glossary, internal-skill-optimize pointers | Constitution constraint: skill changes MUST update SPEC-*.md mirrors | S:90 R:85 A:95 D:95 |
| 14 | Certain | Re-run f004 `wc -c` measurement; record before/after in the PR | Backlog mandates as acceptance evidence | S:95 R:95 A:95 D:95 |
| 15 | Confident | Build on current `zc9m` branch (main + cherry-picked uliv operator dependency); rebase as #390/#391 merge; land before batch 4 | Established batch workflow (#390 → rebase #391); findings line numbers vs ae79e04c handled by content-based edit location | S:70 R:70 A:75 D:75 |

15 assumptions (11 certain, 4 confident, 0 tentative, 0 unresolved).
