# _generation
Shared artifact generation procedures: Intake Generation (fab-new, fab-draft, fab-dedupe, fab-continue), Plan Generation (fab-continue, fab-ff, fab-fff at apply entry), and diff-based adoption variants (fab-adopt). Internal partial loaded via `helpers: [_generation]`.
## Flow
```
Consumer reads _generation.md (via helpers: declaration)
├─ Intake Generation: read intake template → fill every section substantively → append ## Assumptions per _srad → write intake.md
├─ Plan Generation: read plan template → ## Requirements (RFC-2119, stable R#) → Task + Acceptance entry per requirement (traceability R# → T# → test → A#) → ## Tasks / ## Acceptance / ## Assumptions → write plan.md
└─ Adoption variants (fab-adopt only, diff read once): Intake-from-Diff / Plan-from-Diff (headings only; no R#/T#/A# ceremony)
```
### Tools used
Read templates + memory; Write `intake.md` / `plan.md`.
### Sub-agents
None — procedures run inside the consuming skill's context.
