# fab-clarify

## Summary

Refines the intake artifact without advancing. Two modes: Suggest (interactive, user-invoked) and Auto (autonomous — retained for future use; no orchestrator currently invokes it). Scans for gaps, `[NEEDS CLARIFICATION]` markers, and `<!-- assumed: ... -->` markers. Always recomputes the intake confidence. Hosts the `[AUTO-MODE]` Skill Invocation Protocol definition (moved from `_preamble.md` in 260611-zc9m; the preamble keeps a pointer). As of 260612-c5tr the bulk-confirm trigger is evaluated **before** the zero-gaps early exit (a below-gate, Confident-only intake no longer dead-ends at "artifact looks solid"), bulk-confirmed rows are re-graded by recomputed composite (S → 95) rather than labeled Certain by fiat, and both audit-trail writers share one placement/append rule.

**Helpers**: Declares `helpers: [_srad]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

**Prose optimization** (260620-skop): skill content trimmed to remove re-explanation of partial-owned concepts — the post-intake/missing-intake STOP messages now point to the canonical Error Handling table, the re-grade-by-composite rule is stated once (Step 2's Artifact Update) and referenced from Step 4, the shared audit-trail placement/append rule is stated once and referenced, and the dormant "retained for future use" statement is consolidated to § Skill Invocation Protocol → Currently Applicable; a `## Contents` TOC added to the skill. No behavioral change (Flow / Tools / Sub-agents unchanged).

## Flow

```
User invokes /fab-clarify [change-name]
  — OR —
[AUTO-MODE] invocation (defined in this skill's § Skill Invocation Protocol; no current orchestrator uses it)
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
│
├─ Target is always intake.md (intake-only, 1.10.0). At apply or later, STOP (point to /fab-continue rework). Legacy `spec`/`plan`/`tasks` targets removed.
│
├─── SUGGEST MODE (user invocation) ────────────────────
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
├─── AUTO MODE (retained for future use) ───────────────
│  │
│  ├─ Read intake.md
│  ├─ Autonomous gap resolution
│  │  └─ Edit: intake.md
│  ├─ Returns: {resolved, blocking, non_blocking}
│  └─ Step 4: Recompute Confidence (non-advancing)
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
| Suggest Step 7 / Auto Mode step 4 (always, both modes) | `fab score --stage intake <change>` | After intake.md edits (intake is the sole scoring source) |
