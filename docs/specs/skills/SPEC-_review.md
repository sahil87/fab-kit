# _review
Shared review logic run by the dispatched review worker (from /fab-continue, /fab-ff, /fab-fff, /fab-adopt) — the worker IS the single review agent, returning one unified three-tier findings set. A `mode` parameter (full | diff-only) selects whether plan-conformance steps run.
## Flow
```
Review worker reads _review.md at entry, runs the whole review inline
├─ Mode: full (default) / diff-only (holistic diff only)
├─ Preconditions [full]: plan.md tasks all [x], acceptance present
├─ Context: full diff (git diff <base>...HEAD), full repo access; full mode also plan.md + touched sources + memory
├─ Plan conformance [full]: acceptance items → run affected tests → spot-check requirements → memory drift → code quality → parsimony pass → deletion-candidate prompt
├─ Holistic-diff focus [both]: contract violations, pattern inconsistencies, missing cross-references, regressions, structural issues
└─ Verdict: one three-tier list (must/should/nice); any must-fix → fail, else pass
```
### Tools used
Read plan/sources/memory; Edit plan.md (check-marks, ## Deletion Candidates); Bash git diff + test invocations.
### Sub-agents
None — the worker IS the single review agent.
