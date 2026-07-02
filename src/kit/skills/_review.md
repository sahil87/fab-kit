---
name: _review
description: "Review behavior — inward sub-agent (plan requirements/acceptance) and outward sub-agent (Codex→Claude cascade with full repo access), dispatched in parallel during the review stage; a `mode` parameter (full | outward-only) selects which sub-agents run (full dispatches both in parallel; outward-only runs the outward sub-agent alone, used by fab-adopt)."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Shared Review Dispatch

> This file defines shared review logic used by `/fab-continue`, `/fab-ff`, and `/fab-fff`.
> Orchestrators reference this file by name rather than inlining review dispatch logic, ensuring
> review behavior is authoritative in one location — the same pattern as `_generation.md` for
> artifact generation procedures.
>
> **Orchestration** (stage guards, Verdict pass/fail transitions, rework options, rework loop)
> remains in each orchestrator's own file. This partial covers only the mechanics of dispatching
> the review sub-agents and merging their findings.

## Contents

- Review Mode
- Preconditions
- Inward Sub-Agent Dispatch
- Outward Sub-Agent Dispatch
- Parallel Dispatch
- Findings Merge

---

## Review Mode

The orchestrator MAY pass a **`mode`** parameter in the review dispatch. It selects which sub-agents run:

| `mode` | Sub-agents dispatched | Preconditions checked | Used by |
|--------|-----------------------|-----------------------|---------|
| `full` (default — param omitted) | inward + outward | yes (see Preconditions) | `/fab-continue`, `/fab-ff`, `/fab-fff` |
| `outward-only` | outward only | no (skipped — nothing for inward to validate) | `/fab-adopt` |

- **Default is `full`** when the param is omitted, so every existing caller is unaffected — the mode concept is purely additive.
- There is **no `inward-only` value** — no caller needs it today (parsimony). Do not add one speculatively.
- `mode` gates two steps below: **Preconditions** (checked only in `full`) and **Parallel Dispatch** (dispatches only the selected sub-agent(s)). **Findings Merge** and the pass/fail rule are identical for both modes — "any must-fix → fail" works with a single source, and an empty result set (zero findings) passes.

---

## Preconditions

> **Gated on `mode` (see Review Mode).** These preconditions are checked **only** in `full` mode — they validate the inward sub-agent's inputs. In `outward-only` mode they are **skipped** entirely (there is no inward sub-agent and nothing for it to validate; the outward sub-agent reads the diff directly, not `plan.md`).

- `plan.md` MUST exist with both `## Tasks` and `## Acceptance` sections populated. If `## Acceptance` is missing, STOP with "plan.md missing Acceptance section."
- All tasks under `## Tasks` MUST be `[x]`. If not: STOP with "{N} of {total} tasks are incomplete."

---

## Inward Sub-Agent Dispatch

The inward sub-agent validates implementation against the plan's `## Requirements`, `## Tasks`, and `## Acceptance`. It provides a fresh perspective — no shared context with the applying agent beyond the explicitly provided artifacts.

**Dispatch**: Via the Agent tool (`subagent_type: "general-purpose"`).

**Context provided to the sub-agent**: Standard subagent context files (per `_preamble.md` § Standard Subagent Context), plus change-specific files: `plan.md` (containing `## Requirements`, `## Tasks`, and `## Acceptance` sections), relevant source files (files touched by the change), and target memory file(s) from `docs/memory/`. The prompt MUST also carry the change's **`change_type`**: the dispatching orchestrator reads it from `fab/changes/{name}/.status.yaml` (e.g., `grep '^change_type:'` — `fab preflight` does not emit this field) and passes the value in the prompt; Steps 7–8 key their skip condition on it.

### Validation Steps

The inward sub-agent performs all of these checks:

1. **Tasks complete**: All `[x]` in `plan.md` `## Tasks`
2. **Acceptance items**: Inspect code/tests per item under `plan.md` `## Acceptance`. Mark `[x]` in place if met, `[x] **N/A**: {reason}` if N/A, leave `[ ]` with reason if not met
3. **Run affected tests**: Scoped to touched modules/files
4. **Spot-check requirements**: Verify key requirements and GIVEN/WHEN/THEN scenarios in `plan.md` `## Requirements`
5. **Memory drift check**: Compare implementation against referenced memory (warning only)
6. **Code quality check**: For each file modified during apply, verify:
   - Naming conventions consistent with surrounding code
   - Functions focused and appropriately sized
   - Error handling consistent with codebase style
   - Existing utilities reused where applicable
   - If `fab/project/code-quality.md` exists, check each applicable principle from `## Principles`
   - If `fab/project/code-quality.md` exists, check for violations listed in `## Anti-Patterns`
7. **Parsimony pass** (skipped when `change_type` — supplied in the prompt, see Context above — is `docs`, `chore`, or `ci`, or when `fab/project/code-review.md` `## Parsimony Pass` `Enabled: false`): Evaluate the apply-stage diff against the question *"Could the plan's `## Requirements` be satisfied with less code?"* Threshold for stricter scrutiny: **100 net added lines** (advisory, hard-coded — not project-configurable). Below threshold the pass still runs and MAY emit findings. Findings MUST cite specific file paths and line ranges; abstract findings (e.g., "the code could be smaller") MUST NOT be emitted. Each finding is classified into exactly one of these four categories with the mapped severity:

   | Category | Finding shape | Severity |
   |----------|---------------|----------|
   | `reuse-existing-utility` | Newly added code that duplicates a utility already present in the repo | Should-fix |
   | `zero-call-sites` | Newly added function/symbol/branch with zero call sites in the diff | Must-fix |
   | `duplicated-logic` | New code added alongside an existing implementation of the same logic | Must-fix |
   | `verbosity` | Redundant defensive checks, dead branches, or boilerplate that adds no behavior | Nice-to-have |

   Findings are merged into the inward sub-agent's structured output via the existing Findings Merge step.

8. **Deletion-candidate prompt** (skipped under the same conditions as Step 7): Answer *"What existing code (files, functions, branches, config) did this change make redundant or unused?"* Output as a structured list of candidates, each naming a specific symbol, file path, or block, with a one-line justification. The agent MAY answer the literal `None — this change adds new functionality without making existing code redundant` when truthful — the prompt's value is in *forcing the question*. The agent MUST NOT auto-delete; findings are surfaced for the human reviewer to act on.

   Append (or replace, on rework) the output as a new top-level `## Deletion Candidates` section in `plan.md`, placed immediately below the `## Notes` section (or at end of file when `## Notes` is absent). The section heading is a stable parser contract — do NOT alter the heading text. Format:

   ```markdown
   ## Deletion Candidates

   - `{file:line or symbol}` — {one-line justification}
   - `{file:line or symbol}` — {one-line justification}
   ```

   When the change type is in the skip list (`docs`, `chore`, `ci`), the section is omitted entirely from `plan.md` (NOT written as "None"). On rework cycles, an existing `## Deletion Candidates` section SHALL be replaced in place (not duplicated). The section is distinct from `## Acceptance > ### Removal Verification`: removal-verification covers *planned* removals declared in `plan.md` `## Requirements` (`### Deprecated Requirements`); deletion-candidates covers *discovered* opportunities the apply agent missed.

### Structured Output

The inward sub-agent SHALL return structured findings with a **three-tier priority scheme**:

- **Must-fix**: Requirements mismatches (vs. `plan.md` `## Requirements`), failing tests, acceptance violations
- **Should-fix**: Code quality issues, pattern inconsistencies
- **Nice-to-have**: Style suggestions, minor improvements

Each finding includes: severity tier, description, and file:line reference where applicable.

---

## Outward Sub-Agent Dispatch

The outward sub-agent performs a holistic diff review with full repository access. It is given the diff of all changed files and the list of changed file paths, and is permitted to read any file in the repo to explore context.

**Dispatch**: Via the Agent tool (`subagent_type: "general-purpose"`).

**Context provided to the sub-agent**:
- The diff of all changed files: compute the merge-base against the default branch (`git merge-base HEAD origin/main` or the resolved default), then use `git diff <base>...HEAD`
- The list of changed file paths: use the same resolved base with `git diff --name-only <base>...HEAD`
- Standard subagent context files (per `_preamble.md` § Standard Subagent Context)
- Full tool access (Read, Bash, Agent) — the sub-agent MAY read any file in the repo

**Cascade**: The outward sub-agent uses a **Codex → Claude cascade**, controlled by `review_tools` in `fab/project/config.yaml`:

```yaml
review_tools:
    codex: true    # first in cascade — set to false to skip
    claude: true   # fallback — set to false to skip
    copilot: true  # used by /git-pr-review only, not this cascade
```

When `review_tools` is absent, all tools default to `true`.

1. **Check config**: Read `review_tools.codex` — if `false`, skip Codex
2. **Attempt Codex**: `command -v codex` — if found and enabled, run Codex as the reviewer
3. **Check config**: Read `review_tools.claude` — if `false`, skip Claude
4. **If Codex unavailable/disabled or fails**, attempt Claude: `command -v claude` — if found and enabled, run Claude as the reviewer
5. If all enabled tools are unavailable or fail, return an empty findings set (graceful no-op — not an error condition). The review stage continues normally.

### Focus Areas

The outward sub-agent prompt instructs it to look for:

1. **Interface contract violations** — types, return values, API shape mismatches between changed code and callers/dependents
2. **Inconsistencies with documented patterns** — naming conventions, error handling style, or structural patterns described in memory files (`docs/memory/`) that the changed code violates
3. **Missing cross-references** — memory files or spec files that should reference the changed behavior but do not
4. **Behavioral regressions requiring full-repo context** — issues that the inward reviewer (scoped to changed files) would miss but are visible with full codebase access
5. **Structural issues** — duplication of existing utilities, abstraction violations, or architectural drift visible only in the broader codebase context

### Structured Output

The outward sub-agent returns findings in the same three-tier format as the inward sub-agent:

- **Must-fix**: Interface violations, regressions, or structural issues that must be resolved before ship
- **Should-fix**: Pattern inconsistencies or missing cross-references — addressed when clear and low-effort
- **Nice-to-have**: Minor improvements, optional refactors

Each finding includes: severity tier, description, and file:line reference where applicable.

---

## Parallel Dispatch

Dispatch the sub-agent(s) selected by `mode` (see Review Mode):

- **`full`** (default): both sub-agents (inward and outward) are dispatched **in parallel**. The orchestrator waits for both to return before proceeding to the Findings Merge step.
- **`outward-only`**: dispatch the outward sub-agent only. There is no inward sub-agent to wait for; proceed to Findings Merge once the outward sub-agent returns.

### Nesting degradation (harness without sub-agent support)

`review` is the **one nesting stage**: it spawns an inward reviewer + an outward reviewer + a merge. Concurrency depends on the running harness:

- **Harness WITH sub-agent support** (the native Agent-tool adapter, and any CLI harness that offers sub-agents): the inward + outward reviewers run as **parallel sub-agents** exactly as described above.
- **Harness WITHOUT sub-agent support** (a CLI-dispatched review worker on a harness that has no sub-agent primitive): the worker runs the inward reviewer, the outward reviewer, and the merge **sequentially inline in one context** instead of as parallel workers.

**Only the concurrency degrades — the outcome contract is identical**: the same merged findings + pass/fail verdict (Findings Merge below) are produced either way. This is fixed by `docs/specs/harness-adapters.md` § Nesting degradation (review is the nesting stage; degrade concurrency, never the outcome). Because a cross-harness worker may never read fab's skill files beyond the prompt it is handed, the CLI-path review dispatch prompt **carries this degradation instruction** in addition to this canonical note (the dispatch-seam sites — `_preamble.md` § CLI-Adapter Dispatch, `fab-continue.md` Review Behavior, `_pipeline.md` Step 2 — inject it on the CLI branch).

---

## Findings Merge

After the dispatched sub-agent(s) return, their findings are merged into a single prioritized set. In `outward-only` mode there is a single source (the outward sub-agent); the merge steps below operate on that one source, and the deduplication step (2) is a no-op. The pass/fail rule (4) is identical in both modes — including that **zero findings passes**, so an empty `outward-only` result (e.g. all `review_tools` disabled or unavailable) **passes** (best-effort; adoption must not hard-block when no external reviewer is available).

1. **Collect**: Gather all findings from the dispatched sub-agent(s) — both in `full` mode, the outward sub-agent only in `outward-only`
2. **Deduplicate**: If both sub-agents flag the same file:line issue, consolidate into a single finding (use the higher severity if they differ). (No-op in `outward-only` — a single source cannot collide with itself.)
3. **Merge by severity**: Combine into a unified must-fix / should-fix / nice-to-have list
4. **Pass/fail determination** (deterministic — no agent discretion):
   - If **any must-fix** finding exists (from either sub-agent) → review **fails**
   - **No must-fix findings (including zero findings) → review passes.** should-fix and nice-to-have findings are reported but never block.

The merged findings set is returned to the orchestrator for verdict and rework decisions.

> **Note**: The rework loop (bounded retry, escalation rule, pass/fail state transitions) is defined
> in the orchestrator (`fab-continue.md` Verdict section for manual rework; `_pipeline.md`
> § Auto-Rework Loop for `/fab-ff`/`/fab-fff`), not in this file. This file defines only the
> dispatch and merge mechanics.
