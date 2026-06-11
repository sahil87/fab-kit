# _srad

## Summary

SRAD autonomy framework helper — the decision framework planning skills apply when generating artifacts: four-dimension scoring (Signal Strength, Reversibility, Agent Competence, Disambiguation Type) on a continuous 0–100 scale, weighted-mean aggregation to confidence grades (Certain/Confident/Tentative/Unresolved), the Critical Rule override (R < 25 AND A < 25 → Unresolved, must ask), skill-specific autonomy levels, worked examples, artifact markers (`<!-- assumed: ... -->` / `<!-- clarified: ... -->`), and the Assumptions Summary block format.

Extracted from `_preamble.md` § SRAD Autonomy Framework in 260611-zc9m (the preamble keeps a 3-line pointer). This is an internal partial (`user-invocable: false`) — never invoked directly. It is loaded via the frontmatter `helpers:` list of the six planning skills: `fab-new`, `fab-draft`, `fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify`. Non-planning skills do not load it.

## Flow

```
Planning skill declares helpers: [..., _srad]
│
├─ Read: .claude/skills/_srad/SKILL.md
│  (after _preamble, before the skill body —
│   per _preamble.md § Skill Helper Declaration)
│
├─ SRAD Scoring
│  (4 dimensions, 0–100; composite =
│   0.25*S + 0.30*R + 0.25*A + 0.20*D)
│
├─ Confidence Grades
│  (Certain 85–100 / Confident 60–84 /
│   Tentative 30–59 / Unresolved 0–29)
│
├─ Critical Rule
│  (low R + low A → always ask)
│
├─ Skill-Specific Autonomy Levels
│  (fab-new / fab-continue / fab-fff / fab-ff postures)
│
├─ Worked Examples (3, one-liner style)
│
├─ Artifact Markers
│  (<!-- assumed --> / <!-- clarified -->)
│
└─ Assumptions Summary Block
   (table format, Scores column required;
    intake: 4 grades; plan.md: no Unresolved)
```

### Tools used

None — `_srad.md` is a convention document consumed by planning skills, not an executor. `fab score` (Go) parses the Scores column it mandates.

### Sub-agents

None.
