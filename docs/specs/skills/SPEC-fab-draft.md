# fab-draft
Creates a change intake without activating it — a thin call-site over the shared `_intake` Create-Intake Procedure, stopping at intake `ready`. Run `/fab-switch {name}` to activate.
## Flow
```
User invokes /fab-draft <description>
├─ Read: _preamble.md, .claude/skills/_intake/SKILL.md (+helpers)
└─ Create-Intake Procedure Steps 0–9 (interactive — see SPEC-_intake.md); STOP after Step 9 (no activation, no branch)
```
### Tools used
Per the Create-Intake Procedure (see `SPEC-_intake.md`). No `fab change switch`, no git.
### Sub-agents
None.
