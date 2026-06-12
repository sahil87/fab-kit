# _srad

## Summary

SRAD autonomy framework helper — the decision framework planning skills apply when generating artifacts: four-dimension scoring (Signal Strength, Reversibility, Agent Competence, Disambiguation Type) on a continuous 0–100 scale, weighted-mean aggregation to confidence grades via half-open thresholds (≥85 Certain / ≥60 Confident / ≥30 Tentative / else Unresolved — closed bands left continuous composites ungraded until 260612-c5tr), the Critical Rule override (R < 25 AND A < 25 → Unresolved, must ask — the single numeric definition), skill-specific autonomy levels covering all six declaring skills (4 columns + a fab-draft/fab-clarify covering note), worked examples, artifact markers (`<!-- assumed: ... -->` / `<!-- clarified: ... -->`), and the Assumptions Summary block format (omit-when-zero scoped to displayed output only — artifacts always carry `## Assumptions`).

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
│  (half-open: ≥85 Certain / ≥60 Confident /
│   ≥30 Tentative / else Unresolved)
│
├─ Critical Rule
│  (R < 25 AND A < 25 → always Unresolved, always ask —
│   the single numeric definition)
│  (promptless-dispatch carve-out, 260612-w7dp: under
│   /fab-proceed's defer-and-surface contract there is no
│   user to ask — would-be-asked Unresolved decisions become
│   "Deferred — promptless dispatch" rows surfaced by the
│   dispatcher; the intake gate backstops — fab score returns
│   0.0 whenever any Unresolved row exists. MUST-ask applies
│   unchanged wherever a user is reachable)
│
├─ Skill-Specific Autonomy Levels
│  (fab-new / fab-continue / fab-fff / fab-ff postures
│   + covering note: fab-draft = fab-new's column,
│   fab-clarify = the escape valve itself)
│
├─ Worked Examples (3, one-liner style — each
│  example's arithmetic reaches the grade it teaches)
│
├─ Artifact Markers
│  (<!-- assumed --> / <!-- clarified -->)
│
└─ Assumptions Summary Block
   (table format, Scores column required;
    intake: 4 grades; plan.md: no Unresolved;
    omit-when-zero = displayed output only — artifacts
    ALWAYS carry ## Assumptions, "0 assumptions." footer
    when empty)
```

### Tools used

None — `_srad.md` is a convention document consumed by planning skills, not an executor. `fab score` (Go) parses the Scores column it mandates.

### Sub-agents

None.
