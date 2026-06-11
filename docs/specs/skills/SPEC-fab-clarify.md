# fab-clarify

## Summary

Refines the intake artifact without advancing. Two modes: Suggest (interactive, user-invoked) and Auto (autonomous — retained for future use; no orchestrator currently invokes it). Scans for gaps, `[NEEDS CLARIFICATION]` markers, and `<!-- assumed: ... -->` markers. Always recomputes the intake confidence. Hosts the `[AUTO-MODE]` Skill Invocation Protocol definition (moved from `_preamble.md` in 260611-zc9m; the preamble keeps a pointer).

**Helpers**: Declares `helpers: [_srad]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

## Flow

```
User invokes /fab-clarify [change-name] [target-artifact]
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
│  │  └─ Read: fab/changes/{name}/{artifact}.md
│  │
│  ├─ Step 1.5: Taxonomy Scan
│  │  └─ (agent reasoning — scan for gaps, markers)
│  │  └─ Present tentative assumption questions first
│  │
│  ├─ Step 2: Bulk Confirm (if confident >= 3, after tentative resolution)
│  │  └─ Display Confident assumptions → user responds
│  │  └─ Edit: {artifact}.md (upgrade grades in Assumptions table)
│  │
│  ├─ Step 3-4: Ask Questions, Process Answers
│  │  └─ Edit: {artifact}.md (resolve markers, update Assumptions)
│  │
│  ├─ Step 5: Audit Trail
│  │  └─ Edit: {artifact}.md (append ## Clarifications session)
│  │
│  ├─ Step 6: Coverage Summary
│  │
│  └─ Step 7: Recompute Confidence
│     └─ Bash: fab score <change>                    ◄── bookkeeping
│
├─── AUTO MODE (retained for future use) ───────────────
│  │
│  ├─ Read target artifact
│  ├─ Autonomous gap resolution
│  │  └─ Edit: {artifact}.md
│  └─ Returns: {resolved, blocking, non_blocking}
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
| 7 (always) | `fab score --stage intake <change>` | After intake.md edits (intake is the sole scoring source) |
