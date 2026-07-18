# Intake: Single Review Agent

**Change**: 260704-pag2-single-review-agent
**Created**: 2026-07-04

## Origin

Promptless dispatch from `/fab-proceed` (2026-07-04), carrying a synthesized description from a `/fab-discuss` conversation. All design decisions below were made by the user in that discussion; this intake was generated without further questions (`{questioning-mode} = promptless-defer`).

> Collapse the review stage's two parallel reviewer sub-agents (inward + outward) plus findings-merge step into a single review sub-agent whose prompt covers both checklists.
>
> Decisions made: (1) One dispatch per review — the single agent's prompt carries BOTH checklists from `_review.md`: the inward plan-conformance steps AND the outward holistic-diff focus areas. (2) Keep the Codex→Claude cascade (controlled by `fab/project/code-review.md` § Review Tools) as a step inside the single agent's procedure — reviewer diversity is preserved via the external-tool cascade, not via a second sub-agent. (3) No read-prohibition or phase-ordering on plan.md — change artifacts are committed on the branch and appear in the review diff anyway, so a "don't read plan.md first" instruction is flaky; instead the merged prompt carries a framing line: "conformance to plan.md is necessary but not sufficient; also judge the diff on its own merits against the repo." (4) Outcome contract unchanged — one unified three-tier findings list; the orchestrator keeps preconditions, the deterministic pass/fail rule, verdict transitions, and the rework loop; Findings Merge/dedup disappears as a distinct mechanic. (5) Model resolution simplifies — one `fab resolve-agent review --alias` call applies to the one agent; `_preamble.md`'s "Review resolves once" rule becomes moot and is rewritten/removed. (6) `/fab-adopt`'s `outward-only` review mode becomes a prompt variant of the same single dispatch that omits the plan-conformance steps (preconditions stay skipped in that mode, as today). (7) Delete the nesting-degradation machinery — `docs/specs/harness-adapters.md` § Nesting degradation is deleted and the CLI-branch degradation-instruction injection at the dispatch sites is removed.
>
> Alternatives rejected: keeping two parallel dispatches (parallelism was never the value; the anchor-free outward-reviewer independence rationale is already compromised because plan.md rides in the review diff; ~2× context-loading token cost at the review tier; the nesting caused the fyn5 and gvxd resumed-orchestrator-loses-background-children incidents; the harness-adapters spec already declares the sequential-inline single-context form outcome-identical). Phase-ordering inside the merged prompt (fresh diff review before opening plan.md) — rejected as flaky for the same plan-in-diff reason.
>
> Constraints: wall-clock may be slightly worse (accepted; review is not the pipeline bottleneck); checklist-fatigue risk mitigated by keeping the merged prompt lean; no measured per-reviewer findings attribution exists, so this change is design-reasoned, not data-driven; sweep the whole SPEC-mirror class up front; canonical sources under `src/kit/skills/` only; markdown-only change (verify no Go two-reviewer assumption); the `{stage}-result.yaml` review schema is unchanged.

## Why

1. **The two-reviewer split is not load-bearing for quality.** The independence rationale for the outward reviewer ("anchor-free holistic read") is already structurally compromised: `fab/changes/{name}/plan.md` and `intake.md` are committed on the change branch, so they ride inside the `git diff <base>...HEAD` handed to the outward reviewer — it sees the plan content anyway. And `docs/specs/harness-adapters.md` § Nesting degradation already concedes that a sequential-inline single-context form of the review produces an **identical outcome contract** ("degrade concurrency, never the outcome") — an admission that the parallel split buys concurrency, not quality. Reviewer diversity is preserved by the Codex→Claude external-tool cascade, which survives as a step inside the single agent.

2. **The nesting is operationally hazardous.** Review is the one nesting stage (sequencer → review block → two nested reviewer sub-agents + merge). In this project's history the nesting caused two real incidents (fyn5, gvxd): a resumed orchestrator loses its background reviewer children, stranding the review. A single dispatch with no nested children removes the failure class instead of working around it.

3. **Cost.** Two reviewer sub-agents each load the standard subagent context + the diff at the review tier (`claude-fable-5` / `xhigh` in this project) — roughly 2× context-loading for the anchor-independence benefit already shown to be compromised.

4. **If we don't do it**: every review pays the double context cost, the nesting-degradation machinery (spec section + three dispatch-site injection clauses) must be maintained forever for one stage, and the resumed-orchestrator incident class remains live.

## What Changes

### 1. `src/kit/skills/_review.md` — rewrite as a single-agent procedure

The file remains the single authority for review dispatch (same role as today), but its body changes from *two sub-agent dispatches + parallel dispatch + findings merge* to **one review agent's procedure**:

- **One dispatch per review.** The review stage dispatches ONE sub-agent. The dispatched worker runs the whole review inline — there is no nested Agent-tool dispatch inside the review block anymore (this is what makes native and CLI dispatch structurally identical; see § 4).
- **Merged checklist.** The single agent's prompt carries BOTH existing checklists:
  - the **plan-conformance steps** (today's inward Validation Steps 1–8): tasks-all-`[x]` verification, acceptance-item inspection with in-place checkbox mutation in `plan.md` (`[x]` / `[x] **N/A**: {reason}` / `[ ]` with reason), scoped test runs, requirements spot-check (GIVEN/WHEN/THEN), memory drift check (warning only), code-quality check, parsimony pass (with its existing four-category table, 100-net-added-lines advisory threshold, and `[docs, chore, ci]` + `## Parsimony Pass Enabled: false` skip conditions keyed on `change_type` supplied in the prompt), and the deletion-candidate prompt writing/replacing the `## Deletion Candidates` section in `plan.md` (stable heading contract unchanged);
  - the **holistic-diff focus areas** (today's outward list): interface contract violations, pattern inconsistencies vs `docs/memory/`, missing cross-references, behavioral regressions requiring full-repo context, structural issues. The agent receives the diff (`git diff <base>...HEAD` vs the default-branch merge-base) + changed-file list and has full repo read access, exactly as the outward sub-agent does today.
- **Framing line** (verbatim, carried in the merged prompt): *"conformance to plan.md is necessary but not sufficient; also judge the diff on its own merits against the repo."* No read-prohibition and no phase-ordering on `plan.md` — the agent may read everything in any order (user decision; plan.md rides in the diff regardless).
- **Codex→Claude cascade kept** as a step inside the single agent's procedure, controlled exactly as today by `fab/project/code-review.md` § Review Tools (absent section/entry = enabled; `- codex: false` disables; graceful empty-findings no-op when all tools unavailable/disabled — the `copilot` entry stays `/git-pr-review`-only).
- **Keep the prompt lean** (checklist-fatigue mitigation): the tasks-all-`[x]` step is already precondition-covered by the orchestrator, and mechanical steps stay compressed; do not pad the merged procedure with restated orchestration.
- **Findings Merge disappears as a distinct mechanic.** The single agent returns one unified three-tier findings list (must-fix / should-fix / nice-to-have, each with file:line where applicable). Deduplication across sources is moot (single source). The deterministic pass/fail rule is restated where it lives today (orchestrator-owned): any must-fix → fail; no must-fix (including zero findings) → pass.
- **`mode` parameter survives** with the same gating semantics: `full` (default) = merged checklist incl. plan-conformance steps + preconditions; the adoption mode = the same single dispatch with the plan-conformance steps omitted from the prompt and preconditions skipped (see § 5).
- **Preconditions unchanged** (`plan.md` exists with `## Tasks`/`## Acceptance` populated; all tasks `[x]`) and still checked only in `full` mode, by the orchestrator side as today.

### 2. Dispatch sites — one dispatch, one resolution

- **`src/kit/skills/_pipeline.md` Step 2**: currently resolves `fab resolve-agent review --alias` once and applies the profile to "both reviewer sub-agents (inward + outward) and the merge", with a CLI-branch clause about the worker degrading to sequential-inline. Rewrite: one resolution, one dispatched review block that runs the single merged procedure; delete the degradation clause. Verify the `/fab-ff` + `/fab-fff` twins restate nothing beyond the shared bracket (today they delegate to `_pipeline.md`; their SPECs do restate — see § 6).
- **`src/kit/skills/fab-continue.md` Review Behavior** (~lines 170–174): currently instructs the review block to execute `_review.md` end-to-end (Preconditions → Inward + Outward → Parallel Dispatch → Findings Merge), carries a "nested reviewers" resolution note (`fab resolve-agent review --alias` once for BOTH reviewers + merge — independent of the sequencer's resolution) and a "CLI-dispatched review worker (nesting degradation)" note. Rewrite: the review block executes the single merged procedure inline; the nested-resolution note and the degradation note are deleted (the only resolution left is the sequencer's, made when dispatching the review stage). The `change_type`-in-prompt contract stays (the parsimony/deletion-candidate skip conditions key on it).
- **`src/kit/skills/_preamble.md`**:
  - § Subagent Dispatch → Per-Stage Model Resolution: the **"Review resolves once"** paragraph (two reviewers + merge, merge-at-reviewer-tier tradeoff) becomes moot — rewrite/remove so review is unexceptional: one stage, one resolution, like every other stage.
  - § Standard Subagent Context "Nested dispatch" note: keep the rule (it is general), but re-anchor its example if it still cites the review sub-agent nesting as the canonical case.
  - § CLI-Adapter Dispatch: remove the CLI-branch degradation-instruction injection language (review-specific); the five-state machine, dispatch-prompt obligations, and block-contract carve-out are untouched.
- **`src/kit/skills/fab-adopt.md` Step 3**: still dispatches review in the adoption mode via whichever adapter `dispatch=` selects; wording updated from "outward-only / no inward sub-agent" to the prompt-variant framing (see § 5).

### 3. Outcome contract — explicitly unchanged

- The orchestrator keeps everything it owns today: preconditions, the deterministic pass/fail rule (any must-fix → fail; zero findings passes — including the empty adoption-mode result passing best-effort), verdict transitions (`fab status finish/fail review`), and the rework loop (budget in `code-review.md`, mechanics in `_pipeline.md` § Auto-Rework Loop / `fab-continue.md` Verdict).
- The `{stage}-result.yaml` review schema in `_preamble.md` § Dispatch-Prompt Obligations — the `status` vs `verdict` split and the `findings.must_fix/should_fix/nice_to_have` tiers — is byte-unchanged.
- The review tier (`agent.tiers.review`) and `fab resolve-agent` behavior are unchanged; only the number of consumers of the resolved profile drops to one.

### 4. Delete the nesting-degradation machinery

- **`docs/specs/harness-adapters.md` § Nesting degradation (the `review` stage)** (lines ~105–113): deleted. With a single reviewer, native Agent-tool dispatch and CLI `fab dispatch` are structurally identical — one worker runs the whole review inline — so there is nothing to degrade. Also update the § "Skill wiring is NOT part of the contract-defining change" sentence that lists "the nesting-degradation *implementation*" among 3d's deliverables (historical framing — rephrase minimally or annotate; the 3c/3d split narrative itself is history and stays), and the spec's index-table description in `docs/specs/index.md` (which currently lists "`review` nesting degradation" as part of the shared protocol).
- The CLI-branch degradation-instruction injections at the three dispatch sites (§ 2 above) are removed with their host paragraphs.
- **`docs/specs/stage-models.md`**: the "review stage resolves once … BOTH reviewer sub-agents (inward + outward) and the merge" paragraph (~lines 284–285), the § pointer that mentions "`review` nesting degradation" (~line 345), and the deferred-idea bullet "Role-granular keys (`review.inward`, `review.merge`)" (~line 421) all need rewording to the single-agent shape (the deferred-ideas bullet can simply drop the role examples or be marked obsolete).

### 5. `/fab-adopt` adoption review — prompt variant

`outward-only` mode becomes **a prompt variant of the same single dispatch**: the merged procedure minus the plan-conformance steps (no tasks/acceptance verification, no checkbox mutation, no parsimony/deletion-candidate steps — nothing for them to validate on a reverse-engineered thin plan), preconditions skipped, diff + focus areas + cascade retained. Zero findings still passes best-effort. The mode *value name* may be updated to match the new semantics (e.g. `diff-only`) since all callers are in-kit (`fab-adopt.md` is the only passer) — final naming decided at apply; semantics are fixed here. `src/kit/skills/_generation.md`'s Plan-from-Diff acceptance-stub text ("…outward review runs in this pipeline.") is reworded to match.

### 6. Documentation sweep (the whole mirror class, up front)

Per constitution + `code-quality.md` § Sibling & Mirror Sweeps, every touched skill's SPEC mirror updates in the same change, and aggregate restatements are swept by grepping `inward|outward|nesting|resolves once` repo-wide:

- **SPEC mirrors** (constitution-required): `docs/specs/skills/SPEC-_review.md`, `SPEC-_preamble.md` (lines ~11, ~116 restate "review resolves once for both reviewers + merge"), `SPEC-fab-continue.md`, `SPEC-_pipeline.md`, `SPEC-fab-adopt.md`, `SPEC-_generation.md` (if the stub-text edit lands), plus `SPEC-fab-ff.md` (lines ~18, ~27 restate "inward + outward … in parallel") and `SPEC-fab-fff.md` (verify — no literal inward/outward hits today, but sweep for restated review-dispatch facts).
- **Aggregate specs**: `docs/specs/skills.md` (lines ~50, ~356, ~446, ~453), `glossary.md` (outward-only mentions in the `/fab-adopt` and **Adopt** entries, lines ~52, ~116), `overview.md` (stage-table row 3 "inward sub-agent…", line ~70; `/fab-adopt` row line ~97), `user-flow.md` (lines ~38, ~64), `stage-models.md` and `harness-adapters.md` (§ 4 above), `docs/specs/index.md` (harness-adapters description).
- **Scaffold**: `src/kit/scaffold/fab/project/code-review.md` comment line ~61 ("the review-stage outward-reviewer Codex → Claude cascade" → "the review-stage Codex → Claude cascade"). No migration: § Review Tools reading semantics are unchanged (absent = enabled), and stale comments in existing projects' user-owned files are harmless.
- **NOT edited** (historical/generated artifacts): `src/kit/migrations/2.12.1-to-2.13.0.md` (shipped migration text), `docs/specs/findings/*.md` (dated review findings), `docs/specs/srad-scoring-rationale-v1-to-v2.md`, all `docs/memory/**/log.md` / `log.seed.md` (generated), and `fab/changes/**` artifacts of past changes.

### 7. Memory updates (hydrate)

See Affected Memory. The main rewrite is `pipeline/execution-skills.md` § Review Behavior (the two-sub-agent split, Review Mode table, per-sub-agent context/validation/focus-area subsections, and Findings Merge all restate the current shape; the historical decision-log entries lower in the file stay as history).

## Affected Memory

- `pipeline/execution-skills.md`: (modify) § Review Behavior — single-agent procedure replaces the inward/outward split, Review Mode table reworded to the prompt-variant semantics, Findings Merge subsection folded into the single-source outcome contract; historical decision entries retained as history
- `_shared/context-loading.md`: (modify) "Review resolves once" paragraph (~line 131) rewritten/removed; "Nested dispatch" example (~line 113) re-anchored if it still cites review nesting
- `_shared/configuration.md`: (modify) § Review Tools / § `review_tools` (retired) prose that names "the outward sub-agent" as the cascade's reader (~lines 171–200) — now the single review agent's cascade step; § Parsimony Pass prose naming "the inward sub-agent" (~line 199)

## Impact

- **Scope**: markdown only — kit skills (`src/kit/skills/`: `_review.md`, `_preamble.md`, `_pipeline.md`, `fab-continue.md`, `fab-adopt.md`, `_generation.md`), scaffold comment, SPEC mirrors + aggregate specs (`docs/specs/`), memory (`docs/memory/`). Deployed copies under `.claude/skills/` are never edited (regenerated by `fab sync`).
- **No Go change**: verified during intake — `grep -riE 'inward|outward|two reviewer|reviewer sub-agents|findings merge' src/go/` returns nothing; `fab dispatch` and `fab resolve-agent` are stage-generic and embed no two-reviewer assumption. No `_cli-fab.md` or Go-test obligations.
- **No config/migration surface**: `agent.tiers.review`, `providers:`, `code-review.md` § Review Tools semantics, and the `{stage}-result.yaml` schema are all unchanged; no user data restructures, so no `src/kit/migrations/` file.
- **Behavioral risk**: outcome contract is designed to be identical (same findings tiers, same pass/fail rule, same orchestrator ownership); the residual risks are checklist fatigue in one context (mitigated by a lean merged prompt) and slightly worse review wall-clock (accepted — review is not the pipeline bottleneck).
- **Runtime benefit**: removes the one nested-dispatch stage (the fyn5/gvxd resumed-orchestrator incident class) and ~halves review-stage context-loading cost at the `claude-fable-5`/`xhigh` review tier.

## Open Questions

None — the `/fab-discuss` synthesis resolved all consequential design decisions (see Origin and Assumptions). The two intake-level verifications it requested (Go two-reviewer assumptions; SPEC-fab-fff restatements) were performed or scheduled into the sweep above.

## Assumptions

<!-- STATE TRANSFER: This table is the sole continuity mechanism between the intake-stage
     agent and the apply-entry agent (which co-generates plan.md). -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | One dispatch per review; the dispatched worker runs the whole merged review inline — no nested Agent-tool dispatch inside the review block | Discussed — user decision 1 + decision 7's "one worker runs the whole review inline"; this is the premise that deletes the nesting machinery | S:95 R:70 A:90 D:90 |
| 2 | Certain | Codex→Claude cascade kept as a step inside the single agent, still controlled by `code-review.md` § Review Tools | Discussed — user decision 2 (diversity via external tools, not a second sub-agent) | S:95 R:85 A:90 D:90 |
| 3 | Certain | No read-prohibition/phase-ordering on plan.md; merged prompt carries the framing line "conformance to plan.md is necessary but not sufficient; also judge the diff on its own merits against the repo" | Discussed — user decision 3 with rationale (plan.md rides in the review diff; ordering instructions are flaky); phase-ordering explicitly rejected | S:95 R:90 A:90 D:90 |
| 4 | Certain | Outcome contract unchanged: unified three-tier findings, orchestrator-owned preconditions + deterministic pass/fail + verdict transitions + rework loop; Findings Merge dropped as a distinct mechanic; `{stage}-result.yaml` review schema byte-unchanged | Discussed — user decision 4 + constraints list | S:95 R:85 A:90 D:90 |
| 5 | Certain | Model resolution: the single `fab resolve-agent review --alias` at the sequencer's dispatch is the only review resolution; `_preamble.md` "Review resolves once" rewritten/removed (exact rewrite-vs-remove wording decided at apply) | Discussed — user decision 5; both wording options explicitly allowed | S:90 R:85 A:90 D:85 |
| 6 | Certain | `/fab-adopt` adoption review = same single dispatch with plan-conformance steps omitted from the prompt; preconditions stay skipped; zero findings still passes best-effort | Discussed — user decision 6 | S:95 R:85 A:90 D:90 |
| 7 | Certain | Delete `harness-adapters.md` § Nesting degradation + the CLI-branch degradation-instruction injections at `_preamble.md` § CLI-Adapter Dispatch, `fab-continue.md` Review Behavior, `_pipeline.md` Step 2 | Discussed — user decision 7; grounded against the live text at all four sites during intake | S:95 R:80 A:90 D:90 |
| 8 | Certain | No Go change (no `_cli-fab.md`/test obligations) and no migration (no user-data restructure; § Review Tools semantics unchanged) | Verified during intake — `src/go/` grep clean; config/tier/schema surfaces untouched | S:85 R:90 A:95 D:90 |
| 9 | Certain | Historical/generated artifacts are not swept: shipped migrations, dated findings docs, srad-rationale, generated `log.md`/`log.seed.md`, past-change artifacts | Project convention — memory logs are generated (`fab memory-index`), migrations/findings are shipped historical records | S:80 R:90 A:90 D:85 |
| 10 | Confident | `change_type` = `refactor` — restructuring of review dispatch with an explicitly unchanged outcome contract; skills/specs/memory are this project's source (Pure Prompt Play), so it is not `docs` | Keyword inference has no match for "collapse" (defaults `feat`); refactor definition "restructuring without behavior change" fits; set explicitly via `fab status set-change-type` | S:60 R:90 A:80 D:70 |
| 11 | Confident | `mode` parameter survives with fixed semantics (full vs plan-conformance-omitted); the value name `outward-only` is renamed to match the single-agent semantics (default proposal `diff-only`), all callers in-kit so the rename sweeps atomically — final name decided at apply, keeping `outward-only` acceptable | Semantics discussed (decision 6); naming not discussed — low-stakes, reversible, in-kit only | S:45 R:85 A:70 D:50 |
| 12 | Confident | Merged prompt stays lean: precondition-covered tasks-check not restated as agent work; mechanical steps compressed; both checklists carried without padding | Discussed constraint (checklist-fatigue mitigation "keep the merged prompt lean"); exact prompt text is apply's | S:70 R:85 A:80 D:70 |
| 13 | Confident | `_review.md` remains the single authority file (same name, same referenced-by-name pattern) rather than folding review into `fab-continue.md` | Existing project pattern (`_generation.md` precedent, memory: "review behavior authoritative in one location"); nothing in the discussion suggests dissolving the partial | S:65 R:80 A:85 D:75 |

13 assumptions (9 certain, 4 confident, 0 tentative, 0 unresolved).
