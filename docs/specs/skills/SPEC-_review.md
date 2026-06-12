# _review

## Summary

Shared review dispatch logic invoked by `/fab-continue`, `/fab-ff`, and `/fab-fff` at the review stage. Defines the inward sub-agent (validates implementation against spec/plan with eight validation checks, including the parsimony pass and deletion-candidate prompt) and the outward sub-agent (Codex→Claude cascade with full repo access for holistic diff review). Both sub-agents are dispatched in parallel; their findings are merged into a single prioritized set with three severity tiers.

This is an internal partial (`user-invocable: false`) — it is never invoked directly. Skills reference it via `helpers: [_review]` frontmatter and the opening instruction in their review-stage step.

The rework loop is NOT defined here — the file's trailing note points at the orchestrators: `fab-continue.md`'s Verdict section for manual rework, and `_pipeline.md` § Auto-Rework Loop for `/fab-ff`/`/fab-fff` (pointer corrected in 260611-szxd; it previously cited "fab-ff.md/fab-fff.md Step 3").

## Flow

```
Orchestrator (fab-continue / fab-ff / fab-fff) reads _review.md
│
├─ Preconditions
│  Read: plan.md (## Tasks all [x], ## Acceptance present)
│
├─ Parallel Dispatch (Agent tool)
│  │
│  ├─ Inward Sub-Agent
│  │  Context: plan.md (## Requirements + ## Tasks + ## Acceptance),
│  │           touched source files, target memory files
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
│  └─ Outward Sub-Agent
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
└─ Findings Merge
   1. Collect all findings
   2. Deduplicate by file:line (keep higher severity)
   3. Merge by severity into unified set
   4. Pass/fail (deterministic): any must-fix → review fails;
      no must-fix findings (including zero findings) → review passes
      (should-fix / nice-to-have are reported but never block)
```

### Validation Steps Inventory (Inward Sub-Agent)

The inward sub-agent runs all eight checks. Steps 7 and 8 (added in 260507-ogf2) share a hard-coded skip list and a single project-level toggle.

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

- **Inward Sub-Agent** (`subagent_type: "general-purpose"`) — validates implementation against spec/plan; emits structured three-tier findings.
- **Outward Sub-Agent** (`subagent_type: "general-purpose"`) — Codex→Claude cascade for holistic diff review; emits structured three-tier findings.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| Verdict transition | `fab status finish/fail <change> review` | Caller (orchestrator) — _review.md does not call directly |
