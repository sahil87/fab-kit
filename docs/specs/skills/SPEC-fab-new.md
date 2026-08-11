# fab-new
Creates a change from a natural-language description, Linear ticket, or backlog ID, then activates it and creates the git branch. Thin call-site over the shared `_intake` Create-Intake Procedure plus an activate-and-branch tail.
## Flow
```
User invokes /fab-new <description>
├─ Read: _preamble.md, .claude/skills/_intake/SKILL.md (+helpers)
├─ Create-Intake Procedure Steps 0–9 (interactive — see SPEC-_intake.md)
├─ Bash: fab change switch "{name}"
└─ Create Git Branch (same cases as git-branch): probe repo/branch/dirty/target-exists/rename-guard → checkout, checkout --track, checkout -b, or branch -m; dirty tree on create/rename → non-blocking note
```
### Tools used
Steps 0–9 per `SPEC-_intake.md`; own tail: Read (`_intake` skill + helpers), Bash (`fab change switch`, `fab resolve`, `git`).
### Sub-agents
None.
