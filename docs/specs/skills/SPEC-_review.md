# _review

## Contents

- Summary
- Flow

## Summary

Shared review dispatch logic invoked by `/fab-continue`, `/fab-ff`, `/fab-fff`, and `/fab-adopt` at the review stage. Defines the inward sub-agent (validates implementation against `plan.md`'s `## Requirements`, `## Tasks`, and `## Acceptance` with eight validation checks, including the parsimony pass and deletion-candidate prompt) and the outward sub-agent (Codex→Claude cascade with full repo access for holistic diff review). The dispatched sub-agents are run in parallel; their findings are merged into a single prioritized set with three severity tiers.

**Review Mode** (general `mode` parameter, added for `/fab-adopt`): the orchestrator MAY pass `mode` to select which sub-agents run — `full` (default — inward + outward) or `outward-only` (outward sub-agent only, for adopted changes that have no forward requirements for inward to validate). There is deliberately **no `inward-only`** value (no caller needs it — parsimony). `mode` gates exactly two steps: **Preconditions** (the inward `plan.md` checks — `## Tasks`/`## Acceptance` present, all tasks `[x]`) are checked **only** in `full` mode and skipped in `outward-only`; **Parallel Dispatch** dispatches only the sub-agent(s) the mode selects. **Findings Merge** and the deterministic pass/fail rule are identical for both modes — "any must-fix → fail", and zero findings passes (so an empty `outward-only` result, e.g. all `review_tools` disabled/unavailable, **passes** best-effort). Default is `full` when the param is omitted, so every existing caller is unaffected — the mode concept is purely additive.

**Nesting degradation** (260702-aetz / 3d): `review` is the one **nesting** stage (inward + outward reviewers + merge). On a harness WITH sub-agent support (native Agent-tool adapter, or a CLI harness offering sub-agents) those run as **parallel sub-agents**; on a harness WITHOUT sub-agent support (a CLI-dispatched review worker via `fab dispatch`) the worker runs them **sequentially inline in one context**. **Only the concurrency degrades — the outcome contract (same merged findings + pass/fail verdict) is identical.** The rule is fixed by `docs/specs/harness-adapters.md` § Nesting degradation; the canonical note lives in `_review.md` § Parallel Dispatch → Nesting degradation, and — because a cross-harness worker may never read fab's skill files beyond its prompt — the CLI-path review dispatch prompt also carries the degradation instruction (injected at the dispatch-seam sites `_preamble.md` § CLI-Adapter Dispatch / `fab-continue.md` Review Behavior / `_pipeline.md` Step 2).

This is an internal partial (`user-invocable: false`) — it is never invoked directly. Skills reference it via `helpers:` frontmatter (`/fab-ff`, `/fab-fff`, `/fab-adopt`) or a stage-conditional in-body read at review entry (`/fab-continue` — deliberately absent from its frontmatter list, per `_preamble.md` § Skill Helper Declaration).

The rework loop is NOT defined here — the file's trailing note points at the orchestrators: `fab-continue.md`'s Verdict section for manual rework, and `_pipeline.md` § Auto-Rework Loop for `/fab-ff`/`/fab-fff` (pointer corrected in 260611-szxd; it previously cited "fab-ff.md/fab-fff.md Step 3").

**Prose optimization** (260620-skop): a `## Contents` TOC added to `_review.md` (structural check, file >100 lines); no prose trimmed and no behavioral change (Flow unchanged). This SPEC also received a `## Contents` block under the same structural check.

## Flow

```
Orchestrator (fab-continue / fab-ff / fab-fff / fab-adopt) reads _review.md
│
├─ Review Mode: full (default — inward + outward) | outward-only (outward only)
│  (no inward-only value; default full → existing callers unaffected)
│
├─ Preconditions  [checked only in mode=full; skipped in outward-only]
│  Read: plan.md (## Tasks all [x], ## Acceptance present)
│
├─ Parallel Dispatch (Agent tool — dispatches only the sub-agent(s) the mode selects)
│  │  (nesting degradation, 260702-aetz: harness WITH sub-agent support ⇒
│  │   inward + outward run as parallel sub-agents; harness WITHOUT — a
│  │   CLI-dispatched review worker — ⇒ inward + outward + merge run
│  │   sequentially inline; only concurrency degrades, same merged
│  │   findings + verdict; CLI-path prompt carries this instruction)
│  │
│  ├─ Inward Sub-Agent   [full only]
│  │  Context: plan.md (## Requirements + ## Tasks + ## Acceptance),
│  │           touched source files, target memory files,
│  │           change_type (260612-w7dp — read from .status.yaml by
│  │           the dispatching orchestrator and passed in the prompt;
│  │           Steps 7–8 key their skip condition on it)
│  │  Validation Steps (8):
│  │    1. Tasks complete
│  │    2. Acceptance items
│  │    3. Run affected tests
│  │    4. Spot-check requirements (plan.md ## Requirements)
│  │    5. Memory drift check
│  │    6. Code quality check
│  │    7. Parsimony pass
│  │       (skipped for docs/chore/ci, or when
│  │        code-review.md `## Parsimony Pass`
│  │        `Enabled: false`)
│  │       Threshold: 100 net added lines
│  │       (advisory, hard-coded)
│  │       Categories (4): reuse-existing-utility,
│  │                        zero-call-sites,
│  │                        duplicated-logic, verbosity
│  │    8. Deletion-candidate prompt
│  │       (skipped under same conditions as 7)
│  │       Output: ## Deletion Candidates appended
│  │               to plan.md (replaces on rework)
│  │  Output: structured findings (must-fix /
│  │          should-fix / nice-to-have)
│  │
│  └─ Outward Sub-Agent   [both modes — the sole sub-agent in outward-only]
│     Context: full diff (git diff <base>...HEAD),
│              changed file paths, full repo access
│     Cascade: Codex → Claude (controlled by
│              `review_tools` in config.yaml)
│     Focus areas (5): interface contract violations,
│                      pattern inconsistencies,
│                      missing cross-references,
│                      behavioral regressions,
│                      structural issues
│     Output: structured findings (must-fix /
│             should-fix / nice-to-have)
│
└─ Findings Merge  [identical for both modes]
   1. Collect all findings (both sub-agents in full; outward only in outward-only)
   2. Deduplicate by file:line (keep higher severity) — no-op in outward-only (single source)
   3. Merge by severity into unified set
   4. Pass/fail (deterministic): any must-fix → review fails;
      no must-fix findings (including zero findings) → review passes
      (should-fix / nice-to-have are reported but never block;
       an empty outward-only result — no available external reviewer — passes best-effort)
```

### Validation Steps Inventory (Inward Sub-Agent)

The inward sub-agent runs all eight checks. Steps 7 and 8 (added in 260507-ogf2) share a hard-coded skip list and a single project-level toggle. The `change_type` the skip list keys on is a defined prompt input since 260612-w7dp: the dispatching orchestrator reads it from the change's `.status.yaml` (preflight does not emit it) and passes it in the sub-agent prompt.

| Step | Name | Skipped when | Severity surface |
|------|------|--------------|------------------|
| 1 | Tasks complete | (always runs) | must-fix on incomplete |
| 2 | Acceptance items | (always runs) | must-fix on unmet |
| 3 | Run affected tests | (always runs) | must-fix on failure |
| 4 | Spot-check requirements (`plan.md` `## Requirements`) | (always runs) | must-fix on mismatch |
| 5 | Memory drift check | (always runs) | warning only |
| 6 | Code quality check | (always runs; expanded when `code-quality.md` exists) | should-fix on inconsistency, plus per-anti-pattern |
| 7 | Parsimony pass | `change_type ∈ {docs, chore, ci}` OR `code-review.md` `## Parsimony Pass` `Enabled: false` | per-category mapping (see below) |
| 8 | Deletion-candidate prompt | same as Step 7 | informational only — never auto-deletes; surfaced for human reviewer |

#### Parsimony Pass Categories (Step 7)

| Category | Severity |
|----------|----------|
| `reuse-existing-utility` | Should-fix |
| `zero-call-sites` | Must-fix |
| `duplicated-logic` | Must-fix |
| `verbosity` | Nice-to-have |

Threshold (advisory, hard-coded in the prompt body): **100 net added lines**. Below threshold the pass still runs and MAY emit findings; above threshold the agent applies stricter scrutiny but no new severity tier is introduced. Findings MUST cite file paths and line ranges; abstract findings are prohibited.

#### Deletion Candidates (Step 8)

Output appended (or replaced on rework) as a top-level `## Deletion Candidates` section in `plan.md`, placed immediately below `## Notes` (or at end of file when `## Notes` is absent). The section heading is a stable parser contract. When the change type is in the skip list (`docs`, `chore`, `ci`), the section is omitted entirely (NOT written as "None"). Distinct from `## Acceptance > ### Removal Verification`, which covers *planned* removals declared in `plan.md` `## Requirements` (`### Deprecated Requirements`).

### Tools used

| Tool | Purpose |
|------|---------|
| Read | plan.md (## Requirements + ## Tasks + ## Acceptance), source files, memory files, code-review.md |
| Edit | plan.md (acceptance check-marks, ## Deletion Candidates section) |
| Bash | git diff (outward sub-agent base + diff resolution); test invocations (Step 3) |
| Agent | Inward sub-agent dispatch + Outward sub-agent dispatch (parallel) |

### Sub-agents

- **Inward Sub-Agent** (`subagent_type: "general-purpose"`) — validates implementation against `plan.md`'s `## Requirements`, `## Tasks`, and `## Acceptance`; emits structured three-tier findings.
- **Outward Sub-Agent** (`subagent_type: "general-purpose"`) — Codex→Claude cascade for holistic diff review; emits structured three-tier findings.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| Verdict transition | `fab status finish/fail <change> review` | Caller (orchestrator) — _review.md does not call directly |
