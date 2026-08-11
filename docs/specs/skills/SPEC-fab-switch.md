# fab-switch
Switches the active change by creating the `.fab-status.yaml` symlink. Lists available changes with no argument; `--none` deactivates.
## Flow
```
User invokes /fab-switch [change-name] [--none]
├─ Read: _preamble.md (§1 always-load exception)
├─ No argument → Bash: fab change list → user selects; --none → fab change switch --none
└─ change-name → Bash: fab change switch "<name>" ([multi-match] ask; [no match] list) → Bash: fab log command
```
### Tools used
Bash (`fab change switch`, `fab change list`, `fab log command`).
### Sub-agents
None.
