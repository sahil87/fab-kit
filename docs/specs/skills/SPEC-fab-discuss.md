# fab-discuss
Read-only context priming for exploratory discussion — loads the always-load layer, shows an orientation summary, and signals readiness. No artifact generation, no stage advancement.
## Flow
```
User invokes /fab-discuss
├─ Read: always-load layer (per _preamble.md §1)
├─ Bash: fab resolve --folder --or-none; Read: .status.yaml if a change is active
├─ Bash: fab log command "fab-discuss"
└─ Output: orientation summary
```
### Tools used
Read (always-load files, `.status.yaml`), Bash (`fab resolve --folder --or-none`, `fab log command`).
### Sub-agents
None.
