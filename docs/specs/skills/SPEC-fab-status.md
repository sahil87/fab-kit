# fab-status
Read-only status display: change name, branch, stage progress (n/6), plan counts, confidence, optional impact/warning lines, and a next-command suggestion.
## Flow
```
User invokes /fab-status [change-name]
├─ Bash: fab preflight; Read: kit VERSION + fab/.kit-migration-version; Bash: git branch --show-current
└─ Render: stage progress table, plan counts, confidence, impact line (⚠️+bold over threshold), refactor-growth warning
```
### Tools used
Bash (`fab preflight`, `git branch --show-current`), Read (VERSION, migration-version).
### Sub-agents
None.
