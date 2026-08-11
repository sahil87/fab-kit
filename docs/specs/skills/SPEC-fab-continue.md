# fab-continue
Advances the active change one stage per invocation through the six-stage pipeline (intake, apply, review, hydrate, ship, review-pr), with reset to a given stage. Intake runs in the main session; apply/review/hydrate are dispatched as sub-agents (this skill owns their status transitions); ship/review-pr delegate to `/git-pr` and `/git-pr-review`.
## Flow
```
User invokes /fab-continue [change-name] [stage]
├─ Read: _preamble.md; Bash: fab preflight
├─ [reset arg] Bash: fab status reset <change> <stage>
├─ [review-failed] reset apply + rework menu, stop; [review-pr-failed] re-run /git-pr-review
├─ Dispatch on current stage:
│  INTAKE (main session): Read templates/memory → Write intake.md (SRAD) → advance; finish intake
│  APPLY (dispatched): no plan.md → Write plan.md; per unchecked task: Edit/Write sources → Bash: tests → check off plan.md → finish apply
│  REVIEW (dispatched; worker reads _review.md, runs review inline): read diff/plan/source/memory → tests → unified findings
│    pass → finish review + set-acceptance / fail → fail review + reset apply (rework options)
│  HYDRATE (dispatched): Write/Edit docs/memory/** → set-summary → fab memory-index → finish hydrate
│  SHIP: delegate to /git-pr <change>
│  REVIEW-PR: delegate to /git-pr-review <change> (timeout → stage left active)
└─ Output: summary + Next: line
```
### Tools used
Read (preamble, templates, artifacts, source, memory), Write (`plan.md`, memory files), Edit (plan checkboxes, memory), Bash (`fab status` transitions, `fab preflight`, `fab memory-index`, tests), Agent (review sub-agent).
### Sub-agents
Single review sub-agent (reads `_review.md`, runs the review inline; returns one unified findings set).
