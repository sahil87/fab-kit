---
name: _review
description: "Review behavior — a single review sub-agent whose prompt carries both checklists: the plan-conformance steps (requirements/tasks/acceptance validation) and the holistic-diff focus areas (Codex→Claude cascade with full repo access). A `mode` parameter (full | diff-only) selects whether the plan-conformance steps are included (full) or omitted (diff-only, used by fab-adopt)."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Shared Review Dispatch

> This file defines shared review logic used by `/fab-continue`, `/fab-ff`, `/fab-fff`, and
> `/fab-adopt` (which consumes the `diff-only` mode). Orchestrators reference this file by name
> rather than inlining review logic, ensuring review behavior is authoritative in one location —
> the same pattern as `_generation.md` for artifact generation procedures.
>
> **The dispatched review block IS the single review agent.** The sequencer dispatches ONE
> review worker; that worker reads this file at entry and executes the merged checklists inline
> itself. There is no nested Agent-tool dispatch, no parallel dispatch, and no separate
> findings-merge step — the one worker runs both checklists and returns one unified findings list.
> (Where the sequencer performs that dispatch is described in `fab-continue.md` Normal Flow /
> `_pipeline.md` Step 2, not here.)
>
> **Orchestration** (stage guards, Verdict pass/fail transitions, rework options, rework loop)
> remains in each orchestrator's own file. This partial covers only what the review worker does
> and the shape of its findings.

## Contents

- Review Mode
- Preconditions
- Review Agent Dispatch
- Findings & Verdict

---

## Review Mode

The orchestrator MAY pass a **`mode`** parameter in the review dispatch. It selects whether the
plan-conformance steps are included:

| `mode` | Prompt carries | Preconditions checked | Used by |
|--------|----------------|-----------------------|---------|
| `full` (default — param omitted) | plan-conformance steps + holistic-diff focus areas | yes (see Preconditions) | `/fab-continue`, `/fab-ff`, `/fab-fff` |
| `diff-only` | holistic-diff focus areas only (plan-conformance steps omitted) | no (skipped — no forward plan to conform to) | `/fab-adopt` |

- **Default is `full`** when the param is omitted, so every existing caller is unaffected — the mode concept is purely additive.
- There is **no plan-conformance-only mode** — no caller needs it today (parsimony). Do not add one speculatively.
- `mode` gates two things below: **Preconditions** (checked only in `full`) and the **plan-conformance steps** inside Review Agent Dispatch (included only in `full`; omitted in `diff-only`). The **holistic-diff focus areas**, the **Codex→Claude cascade**, and the **Findings & Verdict** rule are identical for both modes — "any must-fix → fail", and an empty result set (zero findings) passes.

---

## Preconditions

> **Gated on `mode` (see Review Mode).** These preconditions are checked **only** in `full` mode — they validate the plan-conformance inputs. In `diff-only` mode they are **skipped** entirely (there is no forward plan for the agent to validate; the agent reads the diff directly, not `plan.md`).

- `plan.md` MUST exist with both `## Tasks` and `## Acceptance` sections populated. If `## Acceptance` is missing, STOP with "plan.md missing Acceptance section."
- All tasks under `## Tasks` MUST be `[x]`. If not: STOP with "{N} of {total} tasks are incomplete."

---

## Review Agent Dispatch

The review worker (the block the sequencer dispatched) runs the whole review inline: it reads this
file at entry and executes the merged checklists itself — validating the implementation against the
plan AND performing a holistic diff review with full repository access. There is no further
Agent-tool dispatch here; the worker IS the single review agent.

**Framing** (present in this file, which the worker reads): *"conformance to plan.md is necessary but not sufficient; also judge the diff on its own merits against the repo."* There is **no read-prohibition** and **no phase-ordering** on `plan.md` — the worker MAY read anything in any order (`plan.md` rides in the review diff regardless, so an ordering instruction would be flaky).

**Context the worker operates on**:
- Standard subagent context files (per `_preamble.md` § Standard Subagent Context)
- The diff of all changed files: compute the merge-base against the default branch (`git merge-base HEAD origin/main` or the resolved default), then use `git diff <base>...HEAD`
- The list of changed file paths: use the same resolved base with `git diff --name-only <base>...HEAD`
- Full tool access (Read, Edit, Write, Bash, per `_preamble.md` § Standard Subagent Context) — the worker MAY read any file in the repo, and MAY modify `plan.md` (marking acceptance checkmarks in the Plan-Conformance Steps below)
- **In `full` mode only**: `plan.md` (containing `## Requirements`, `## Tasks`, and `## Acceptance` sections), relevant source files (files touched by the change), target memory file(s) from `docs/memory/`, and the change's **`change_type`** — carried in the dispatch prompt by the sequencer (which reads it from `fab/changes/{name}/.status.yaml`, since `fab preflight` does not emit this field — see `fab-continue.md` Normal Flow / `_pipeline.md` Step 2); the parsimony/deletion-candidate steps key their skip condition on it.

Keep the two checklists lean (checklist-fatigue mitigation): the tasks-all-`[x]` check is covered by Preconditions (in `full` mode) — the worker does not re-verify it as a checklist step — and the mechanical steps below stay compressed; do not pad the merged procedure with restated orchestration.

### Plan-Conformance Steps (`full` mode only)

Included only in `full` mode (omitted entirely in `diff-only` — there is nothing for them to validate on a reverse-engineered thin plan). The worker validates the implementation against the plan's `## Requirements`, `## Tasks`, and `## Acceptance` (the tasks-all-`[x]` check is a Precondition, not repeated here):

1. **Acceptance items**: Inspect code/tests per item under `plan.md` `## Acceptance`. Mark `[x]` in place if met, `[x] **N/A**: {reason}` if N/A, leave `[ ]` with reason if not met
2. **Run affected tests**: Scoped to touched modules/files
3. **Spot-check requirements**: Verify key requirements and GIVEN/WHEN/THEN scenarios in `plan.md` `## Requirements`
4. **Memory drift check**: Compare implementation against referenced memory (warning only)
5. **Code quality check**: For each file modified during apply, verify:
   - Naming conventions consistent with surrounding code
   - Functions focused and appropriately sized
   - Error handling consistent with codebase style
   - Existing utilities reused where applicable
   - If `fab/project/code-quality.md` exists, check each applicable principle from `## Principles`
   - If `fab/project/code-quality.md` exists, check for violations listed in `## Anti-Patterns`
6. **Parsimony pass** (skipped when `change_type` — carried in the prompt, see Context above — is `docs`, `chore`, or `ci`, or when `fab/project/code-review.md` `## Parsimony Pass` `Enabled: false`): Evaluate the apply-stage diff against the question *"Could the plan's `## Requirements` be satisfied with less code?"* Threshold for stricter scrutiny: **100 net added lines** (advisory, hard-coded — not project-configurable). Below threshold the pass still runs and MAY emit findings. Findings MUST cite specific file paths and line ranges; abstract findings (e.g., "the code could be smaller") MUST NOT be emitted. Each finding is classified into exactly one of these four categories with the mapped severity:

   | Category | Finding shape | Severity |
   |----------|---------------|----------|
   | `reuse-existing-utility` | Newly added code that duplicates a utility already present in the repo | Should-fix |
   | `zero-call-sites` | Newly added function/symbol/branch with zero call sites in the diff | Must-fix |
   | `duplicated-logic` | New code added alongside an existing implementation of the same logic | Must-fix |
   | `verbosity` | Redundant defensive checks, dead branches, or boilerplate that adds no behavior | Nice-to-have |

   Parsimony findings feed the single unified findings list (see Findings & Verdict).

7. **Deletion-candidate prompt** (skipped under the same conditions as Step 6): Answer *"What existing code (files, functions, branches, config) did this change make redundant or unused?"* Output as a structured list of candidates, each naming a specific symbol, file path, or block, with a one-line justification. The worker MAY answer the literal `None — this change adds new functionality without making existing code redundant` when truthful — the prompt's value is in *forcing the question*. The worker MUST NOT auto-delete; findings are surfaced for the human reviewer to act on.

   Append (or replace, on rework) the output as a new top-level `## Deletion Candidates` section in `plan.md`, placed immediately below the `## Notes` section (or at end of file when `## Notes` is absent). The section heading is a stable parser contract — do NOT alter the heading text. Format:

   ```markdown
   ## Deletion Candidates

   - `{file:line or symbol}` — {one-line justification}
   - `{file:line or symbol}` — {one-line justification}
   ```

   When the change type is in the skip list (`docs`, `chore`, `ci`), the section is omitted entirely from `plan.md` (NOT written as "None"). On rework cycles, an existing `## Deletion Candidates` section SHALL be replaced in place (not duplicated). The section is distinct from `## Acceptance > ### Removal Verification`: removal-verification covers *planned* removals declared in `plan.md` `## Requirements` (`### Deprecated Requirements`); deletion-candidates covers *discovered* opportunities the apply agent missed.

### Holistic-Diff Focus Areas (both modes)

Beyond plan conformance, the agent judges the diff on its own merits against the repo (the framing line above). With full repository access and the diff + changed-file list in hand, it looks for:

1. **Interface contract violations** — types, return values, API shape mismatches between changed code and callers/dependents
2. **Inconsistencies with documented patterns** — naming conventions, error handling style, or structural patterns described in memory files (`docs/memory/`) that the changed code violates
3. **Missing cross-references** — memory files or spec files that should reference the changed behavior but do not
4. **Behavioral regressions requiring full-repo context** — issues visible only with full codebase access (not just the changed files)
5. **Structural issues** — duplication of existing utilities, abstraction violations, or architectural drift visible only in the broader codebase context

### Codex→Claude Cascade (both modes)

The holistic-diff review uses a **Codex → Claude cascade**, controlled by the `codex` and `claude` entries in `fab/project/code-review.md` § Review Tools:

The `## Review Tools` section (prose) lists each reviewer tool that is disabled. An **absent section — or an absent entry** — means the tool is **enabled**; a tool is disabled only when the section lists it as `false` (e.g. `- codex: false`). So an all-enabled setup needs nothing in this section. The `copilot` entry in the same section is read by `/git-pr-review` only, not this cascade.

1. **Check config**: Read the `codex` entry from `code-review.md` § Review Tools — if listed as `false`, skip Codex
2. **Attempt Codex**: `command -v codex` — if found and enabled, run Codex as the reviewer
3. **Check config**: Read the `claude` entry from `code-review.md` § Review Tools — if listed as `false`, skip Claude
4. **If Codex unavailable/disabled or fails**, attempt Claude: `command -v claude` — if found and enabled, run Claude as the reviewer
5. If all enabled tools are unavailable or fail, contribute an empty findings set from the cascade (graceful no-op — not an error condition). The review continues normally.

---

## Findings & Verdict

The single review agent returns **one unified prioritized findings set** with a **three-tier priority scheme**. There is a single source, so there is no cross-source deduplication or merge step — the agent consolidates everything it found (plan-conformance + holistic-diff + parsimony) into one list:

- **Must-fix**: Requirements mismatches (vs. `plan.md` `## Requirements`), failing tests, acceptance violations, interface violations, regressions, or structural issues that must be resolved before ship
- **Should-fix**: Code quality issues, pattern inconsistencies, missing cross-references — addressed when clear and low-effort
- **Nice-to-have**: Style suggestions, minor improvements, optional refactors

Each finding includes: severity tier, description, and file:line reference where applicable.

**Pass/fail rule** (deterministic — no agent discretion; owned by the orchestrator, restated here for the contract):

- If **any must-fix** finding exists → review **fails**
- **No must-fix findings (including zero findings) → review passes.** should-fix and nice-to-have findings are reported but never block.

Zero findings passes — so an empty `diff-only` result (e.g. all reviewer tools disabled via `code-review.md` § Review Tools, or unavailable) **passes** best-effort (adoption must not hard-block when no external reviewer is available).

The findings set is returned to the orchestrator for verdict and rework decisions.

> **Note**: The rework loop (bounded retry, escalation rule, pass/fail state transitions) is defined
> in the orchestrator (`fab-continue.md` Verdict section for manual rework; `_pipeline.md`
> § Auto-Rework Loop for `/fab-ff`/`/fab-fff`), not in this file. This file defines only the
> review procedure and the findings shape.
