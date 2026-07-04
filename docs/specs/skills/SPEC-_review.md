# _review

## Contents

- Summary
- Flow

## Summary

Shared review logic read by the dispatched review worker at the review stage (dispatched by `/fab-continue`, `/fab-ff`, `/fab-fff`, and `/fab-adopt`). **The dispatched review block IS the single review agent**: the worker reads this file at entry and executes the merged checklists inline itself — the plan-conformance steps (validates implementation against `plan.md`'s `## Requirements`, `## Tasks`, and `## Acceptance` with seven validation checks — the tasks-all-`[x]` check is a Precondition, not repeated — including the parsimony pass and deletion-candidate prompt) and the holistic-diff focus areas (Codex→Claude cascade with full repo access for a full-repo diff review). The one worker runs the whole review inline and returns one unified prioritized findings set with three severity tiers — there is no nested sub-agent dispatch, no parallel dispatch, and no separate findings-merge step. A verbatim framing line — *"conformance to plan.md is necessary but not sufficient; also judge the diff on its own merits against the repo."* — lives in this file (which the worker reads), with no read-prohibition or phase-ordering on `plan.md`.

**Review Mode** (general `mode` parameter, added for `/fab-adopt`): the orchestrator MAY pass `mode` to select whether the plan-conformance steps are included — `full` (default — plan-conformance steps + holistic-diff focus areas) or `diff-only` (holistic-diff focus areas only, for adopted changes that have no forward requirements to validate). There is deliberately **no plan-conformance-only** value (no caller needs it — parsimony). `mode` gates exactly two things: **Preconditions** (the `plan.md` checks — `## Tasks`/`## Acceptance` present, all tasks `[x]`) are checked **only** in `full` mode and skipped in `diff-only`; the **plan-conformance steps** are included in the prompt only in `full` mode. The **holistic-diff focus areas**, the **Codex→Claude cascade**, and the deterministic pass/fail rule are identical for both modes — "any must-fix → fail", and zero findings passes (so an empty `diff-only` result, e.g. all reviewer tools disabled via `code-review.md` § Review Tools, or unavailable, **passes** best-effort). Default is `full` when the param is omitted, so every existing caller is unaffected — the mode concept is purely additive.

**Single dispatch** (260704-pag2): review dispatches ONE sub-agent that runs the whole review inline — the merged prompt carries both checklists, so native Agent-tool dispatch and CLI `fab dispatch` are structurally identical (one worker, no nested children). This replaced the former two parallel reviewer sub-agents (inward + outward) plus a findings-merge step, and removed the review-stage nesting-degradation machinery entirely (reviewer diversity is preserved by the Codex→Claude external-tool cascade, kept as a step inside the single agent). The `{stage}-result.yaml` review schema (the `status` vs `verdict` split, the three findings tiers) is byte-unchanged.

This is an internal partial (`user-invocable: false`) — it is never invoked directly. Skills reference it via `helpers:` frontmatter (`/fab-ff`, `/fab-fff`, `/fab-adopt`) or a stage-conditional in-body read at review entry (`/fab-continue` — deliberately absent from its frontmatter list, per `_preamble.md` § Skill Helper Declaration).

The rework loop is NOT defined here — the file's trailing note points at the orchestrators: `fab-continue.md`'s Verdict section for manual rework, and `_pipeline.md` § Auto-Rework Loop for `/fab-ff`/`/fab-fff` (pointer corrected in 260611-szxd; it previously cited "fab-ff.md/fab-fff.md Step 3").

**Prose optimization** (260620-skop): a `## Contents` TOC added to `_review.md` (structural check, file >100 lines); no prose trimmed and no behavioral change (Flow unchanged). This SPEC also received a `## Contents` block under the same structural check.

## Flow

```
Dispatched review worker (dispatch owned by fab-continue / fab-ff / fab-fff / fab-adopt)
reads _review.md at entry and executes it inline — the worker IS the single review agent
│
├─ Review Mode: full (default — plan-conformance steps + holistic-diff focus areas)
│  | diff-only (holistic-diff focus areas only)
│  (no plan-conformance-only value; default full → existing callers unaffected)
│
├─ Preconditions  [checked only in mode=full; skipped in diff-only]
│  Read: plan.md (## Tasks all [x], ## Acceptance present)
│
├─ Review Agent Dispatch (the worker runs the whole review inline — no further Agent-tool dispatch)
│  │  Framing (verbatim in this file, which the worker reads): "conformance to plan.md is
│  │  necessary but not sufficient; also judge the diff on its own merits against the repo."
│  │  (no read-prohibition, no phase-ordering on plan.md)
│  │  Context: full diff (git diff <base>...HEAD), changed file paths, full repo access;
│  │           in full mode also plan.md (## Requirements + ## Tasks + ## Acceptance),
│  │           touched source files, target memory files, and
│  │           change_type (260612-w7dp — read from .status.yaml by the sequencer
│  │           and carried in the block dispatch prompt; the parsimony/deletion-candidate
│  │           steps key their skip condition on it)
│  │
│  ├─ Plan-Conformance Steps  [full only; tasks-all-[x] is a Precondition, not a step here]
│  │    1. Acceptance items
│  │    2. Run affected tests
│  │    3. Spot-check requirements (plan.md ## Requirements)
│  │    4. Memory drift check
│  │    5. Code quality check
│  │    6. Parsimony pass
│  │       (skipped for docs/chore/ci, or when
│  │        code-review.md `## Parsimony Pass`
│  │        `Enabled: false`)
│  │       Threshold: 100 net added lines (advisory, hard-coded)
│  │       Categories (4): reuse-existing-utility, zero-call-sites,
│  │                        duplicated-logic, verbosity
│  │    7. Deletion-candidate prompt
│  │       (skipped under same conditions as 6)
│  │       Output: ## Deletion Candidates appended to plan.md (replaces on rework)
│  │
│  ├─ Holistic-Diff Focus Areas  [both modes]
│  │    interface contract violations, pattern inconsistencies,
│  │    missing cross-references, behavioral regressions, structural issues
│  │
│  └─ Codex→Claude Cascade  [both modes]
│       (controlled by codex/claude entries in code-review.md § Review Tools;
│        graceful empty-findings no-op when all tools unavailable/disabled)
│
└─ Findings & Verdict  [one unified source — no merge/dedup step]
   The agent consolidates plan-conformance + holistic-diff + parsimony findings into
   one three-tier list (must-fix / should-fix / nice-to-have).
   Pass/fail (deterministic): any must-fix → review fails;
      no must-fix findings (including zero findings) → review passes
      (should-fix / nice-to-have are reported but never block;
       an empty diff-only result — no available external reviewer — passes best-effort)
```

### Validation Steps Inventory (Plan-Conformance Steps)

The plan-conformance steps run seven checks (in `full` mode only) — the tasks-all-`[x]` check is a Precondition (checked in `full` mode), not repeated as a step. Steps 6 and 7 (added in 260507-ogf2) share a hard-coded skip list and a single project-level toggle. The `change_type` the skip list keys on is a defined prompt input since 260612-w7dp: the **sequencer** reads it from the change's `.status.yaml` (preflight does not emit it) and carries it in the block dispatch prompt.

| Step | Name | Skipped when | Severity surface |
|------|------|--------------|------------------|
| 1 | Acceptance items | (always runs in full) | must-fix on unmet |
| 2 | Run affected tests | (always runs in full) | must-fix on failure |
| 3 | Spot-check requirements (`plan.md` `## Requirements`) | (always runs in full) | must-fix on mismatch |
| 4 | Memory drift check | (always runs in full) | warning only |
| 5 | Code quality check | (always runs in full; expanded when `code-quality.md` exists) | should-fix on inconsistency, plus per-anti-pattern |
| 6 | Parsimony pass | `change_type ∈ {docs, chore, ci}` OR `code-review.md` `## Parsimony Pass` `Enabled: false` | per-category mapping (see below) |
| 7 | Deletion-candidate prompt | same as Step 6 | informational only — never auto-deletes; surfaced for human reviewer |

#### Parsimony Pass Categories (Step 6)

| Category | Severity |
|----------|----------|
| `reuse-existing-utility` | Should-fix |
| `zero-call-sites` | Must-fix |
| `duplicated-logic` | Must-fix |
| `verbosity` | Nice-to-have |

Threshold (advisory, hard-coded in the prompt body): **100 net added lines**. Below threshold the pass still runs and MAY emit findings; above threshold the agent applies stricter scrutiny but no new severity tier is introduced. Findings MUST cite file paths and line ranges; abstract findings are prohibited.

#### Deletion Candidates (Step 7)

Output appended (or replaced on rework) as a top-level `## Deletion Candidates` section in `plan.md`, placed immediately below `## Notes` (or at end of file when `## Notes` is absent). The section heading is a stable parser contract. When the change type is in the skip list (`docs`, `chore`, `ci`), the section is omitted entirely (NOT written as "None"). Distinct from `## Acceptance > ### Removal Verification`, which covers *planned* removals declared in `plan.md` `## Requirements` (`### Deprecated Requirements`).

### Tools used

These are the tools the review worker uses while executing this file inline (the sequencer's Agent-tool / `fab dispatch` dispatch of the worker is described in `fab-continue.md` / `_pipeline.md`, not here):

| Tool | Purpose |
|------|---------|
| Read | plan.md (## Requirements + ## Tasks + ## Acceptance), source files, memory files, code-review.md |
| Edit | plan.md (acceptance check-marks, ## Deletion Candidates section) |
| Bash | git diff (base + diff resolution); test invocations (Step 2); the Codex→Claude cascade tools |

### Sub-agents

- **None dispatched by this file.** The dispatched review worker IS the single review agent — it reads `_review.md` at entry and runs the whole review inline (validating implementation against `plan.md`'s `## Requirements`, `## Tasks`, and `## Acceptance` in full mode AND performing a full-repo holistic diff review via the Codex→Claude cascade), emitting one unified structured three-tier findings set. It dispatches no further sub-agent. The dispatch of the worker itself is owned by the sequencer (`fab-continue.md` Normal Flow / `_pipeline.md` Step 2).

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| Verdict transition | `fab status finish/fail <change> review` | Caller (orchestrator) — _review.md does not call directly |
