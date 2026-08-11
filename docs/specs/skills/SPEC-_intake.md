# _intake
Shared pre-boundary Create-Intake Procedure (Steps 0–9) for /fab-new, /fab-draft, /fab-dedupe, and /fab-proceed's create-new dispatch — the pre-boundary counterpart to `_pipeline.md`. Parameterized by `{questioning-mode}` (interactive | promptless-defer); consumers must have loaded `_generation` and `_srad`.
## Flow
```
├─ 0 Parse input: Linear ID → MCP fetch / backlog ID → read fab/backlog.md / natural language as-is
├─ 1 Generate slug (2–6 word kebab)
├─ 2 Gap analysis
├─ 3 Create change: collision pre-checks → [existing] route to resume, STOP / else fab change new [--change-id] (+ fab status add-issue if Linear)
├─ 4 Mine conversation context → Certain/Confident assumption rows
├─ 5 Generate intake.md → _generation.md § Intake Generation
├─ 6 Verify change type — [if wrong] fab status set-change-type
├─ 7 Confidence — fab score --stage intake <change>
├─ 8 SRAD question selection: interactive → ask via SRAD / promptless-defer → "Deferred" rows returned to the dispatcher
└─ 9 Advance — fab status advance <change> intake
```
### Tools used
Read `_generation.md`/`_srad.md`/templates/backlog; Write `intake.md`; Bash `fab change new`, `fab resolve`, `fab status *`, `fab score`; MCP (Linear) optional.
### Sub-agents
None.
