# fab-clarify

## Summary

Refines the current stage artifact without advancing. Two modes: Suggest (interactive, user-invoked) and Auto (autonomous, called by fab-ff/fab-fff). Scans for gaps, `[NEEDS CLARIFICATION]` markers, and `<!-- assumed: ... -->` markers. Optionally recomputes confidence.

## Flow

```
User invokes /fab-clarify [change-name] [target-artifact]
  — OR —
[AUTO-MODE] invoked by /fab-ff or /fab-fff
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
│
├─ Resolve target artifact (intake.md / spec.md / tasks.md)
│
├─── SUGGEST MODE (user invocation) ────────────────────
│  │
│  ├─ Step 1: Read target artifact
│  │  └─ Read: fab/changes/{name}/{artifact}.md
│  │
│  ├─ Step 1.5: Bulk Confirm (if confident >= 3)
│  │  └─ Display Confident assumptions → user responds
│  │  └─ Edit: {artifact}.md (upgrade grades in Assumptions table)
│  │
│  ├─ Step 2: Taxonomy Scan
│  │  └─ (agent reasoning — scan for gaps, markers)
│  │  └─ Build prioritized question queue (max 5)
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
├─── AUTO MODE (internal fab-ff call) ──────────────────
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
| 7 (Suggest only) | `fab score <change>` | After spec.md edits |
