# _srad
SRAD autonomy framework helper — the decision framework planning skills apply when generating artifacts: four-dimension scoring, indicative confidence grades, the Critical Rule, per-skill autonomy levels, artifact markers, Assumptions Summary format. Internal partial loaded via planning skills' `helpers:` frontmatter.
## Flow
```
Planning skill declares helpers: [..., _srad] → reads _srad.md before its body
├─ SRAD scoring: 4 dimensions 0–100 → composite → grade (Certain/Confident/Tentative/Unresolved)
├─ Critical Rule: genuine unknowns MUST be asked (promptless dispatch → "Deferred" rows)
├─ Skill-specific autonomy levels; worked examples
└─ Artifact markers (<!-- assumed --> / <!-- clarified -->) + ## Assumptions block (Scores column required)
```
### Tools used
None — a convention document; `fab score` (Go) parses the Scores column it mandates.
### Sub-agents
None.
