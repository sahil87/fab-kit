# fab-clarify

## Summary

**Source organization:** Clarify owns the concise `[AUTO-MODE]` Skill Invocation Protocol and its machine-readable Auto Mode contract alongside the interactive flow.

Refines the intake artifact without advancing. Suggest Mode scans interactively
for gaps, `[NEEDS CLARIFICATION]` markers, and `<!-- assumed: ... -->` markers,
applies bulk confirmation before the zero-gap exit, and re-grades confirmed rows
by their recomputed composite. A first-line `[AUTO-MODE]` prefix selects
autonomous intake resolution and a machine-readable result. Both modes recompute
intake confidence.

**Helpers**: Declares `helpers: [_srad]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

## Flow

```
User invokes /fab-clarify [change-name]
  — OR —
[AUTO-MODE] invocation (prefix placement, detection, and transitivity are defined in this skill)
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
│
├─ Target is always intake.md (intake-only, 1.10.0). At apply or later, STOP (point to /fab-continue rework). Legacy `spec`/`plan`/`tasks` targets removed.
│
├─── SUGGEST MODE (interactive) ────────────────────────
│  │
│  ├─ Step 1: Read target artifact
│  │  └─ Read: fab/changes/{name}/intake.md
│  │
│  ├─ Step 1.5: Taxonomy Scan
│  │  └─ (agent reasoning — scan for gaps, markers)
│  │  └─ Present tentative assumption questions first
│  │  └─ (never stops on zero gaps — the early exit lives in
│  │     Step 2's not-triggered branch, AFTER the bulk-confirm
│  │     trigger is evaluated, 260612-c5tr)
│  │
│  ├─ Step 2: Bulk Confirm (if confident >= 3 AND confident >
│  │  │ tentative + unresolved; evaluated before any zero-gaps exit —
│  │  │ not-triggered + empty queue → "artifact looks solid" stop)
│  │  └─ Display Confident assumptions → user responds
│  │  └─ Edit: intake.md (S → 95, then recompute the composite
│  │     per _srad § SRAD Scoring and grade by its half-open
│  │     thresholds — not fiat-Certain; no weights/threshold numbers
│  │     restated in fab-clarify.md;
│  │     audit trail uses the same placement/append rules as Step 5)
│  │
│  ├─ Step 3-4: Ask Questions, Process Answers
│  │  └─ Edit: intake.md (resolve markers, update Assumptions;
│  │     re-grade the row by recomputed composite (S → 95) per
│  │     _srad § SRAD Scoring — not fiat-Certain, same as Step 2)
│  │
│  ├─ Step 5: Audit Trail
│  │  └─ Edit: intake.md (append ## Clarifications session)
│  │
│  ├─ Step 6: Coverage Summary
│  │
│  └─ Step 7: Recompute Confidence
│     └─ Bash: fab score --stage intake <change>     ◄── bookkeeping
│
├─── AUTO MODE (machine-readable) ──────────────────────
│  │
│  ├─ Read and scan intake.md without user interaction
│  ├─ Resolve contextual gaps; mark blocking user-input gaps
│  ├─ Return: {resolved, blocking, non_blocking, blocking_issues?}
│  └─ Recompute Confidence (non-advancing)
│     └─ Bash: fab score --stage intake <change>     ◄── bookkeeping
│
└─ Does NOT advance stage
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Preamble, artifacts, memory files |
| Edit | Update artifact in-place (markers, Assumptions table, Clarifications) |
| Bash | `fab preflight`, `fab score` |

### Sub-agents

None.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| Suggest Step 7 / Auto Mode step 4 | `fab score --stage intake <change>` | After intake.md edits (intake is the sole scoring source) |
