# fab-adopt
Adopts a completed off-pipeline change (feature branch authored without fab, OPEN or not-yet-created PR) into the pipeline; MERGED PRs are out of scope. Only `apply` is marked skipped — intake, review, hydrate, ship, and review-pr genuinely run.
## Flow
```
User invokes /fab-adopt [<slug>]
├─ Guards (STOP before any mutation): detached/default branch, MERGED PR, already in pipeline, or empty diff vs base
├─ Intake (one main-session pass): fab change new + activate; reconstruct intake.md from the diff; human confirmation; write MINIMAL all-[x] plan.md
├─ Bash: fab status skip apply / reset review / set-summary
├─ Review (dispatched, mode: diff-only): pass → finish review / fail → auto-rework or hand back
├─ Hydrate (dispatched) → finish hydrate
├─ Ship: dispatch /git-pr {name} (retrofit existing PR or fresh PR)
└─ Land in review-pr → summary + Next: /git-pr-review
```
### Tools used
Bash (`git`, `gh pr view`, `fab change new`, `fab status`, `fab score`), Read (diff, PR body, templates), Write (`intake.md`, `plan.md`), Agent (review + hydrate, `/git-pr`).
### Sub-agents
`/fab-continue` Review (`mode: diff-only`), `/fab-continue` Hydrate, `/git-pr {name}` (ship).
