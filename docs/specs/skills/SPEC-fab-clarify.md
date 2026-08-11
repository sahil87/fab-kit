# fab-clarify
Refines the intake artifact without advancing. Suggest Mode scans interactively for gaps and markers; a first-line `[AUTO-MODE]` prefix selects autonomous resolution with a machine-readable result.
## Flow
```
User invokes /fab-clarify [change-name]  — or —  [AUTO-MODE] invocation
├─ Read: _preamble.md; Bash: fab preflight
├─ Target is intake.md; at apply or later → STOP (→ /fab-continue rework)
├─── SUGGEST MODE (interactive)
│  ├─ Read intake.md → taxonomy scan (gaps, markers)
│  ├─ Bulk-confirm confident items → Edit: intake.md
│  ├─ Ask questions, process answers → Edit: intake.md (append ## Clarifications)
│  └─ Coverage summary → Bash: fab score --stage intake
├─── AUTO MODE (machine-readable)
│  ├─ Scan + resolve contextual gaps; mark blocking user-input gaps
│  └─ Return {resolved, blocking, non_blocking, blocking_issues?} → Bash: fab score
└─ Does NOT advance stage
```
### Tools used
Read (preamble, artifacts, memory), Edit (intake.md in place), Bash (`fab preflight`, `fab score`).
### Sub-agents
None.
